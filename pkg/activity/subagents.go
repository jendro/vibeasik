package activity

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Sub agent Claude Code (tool Agent/Task) tampil sebagai anak buah agent yang menyuruhnya (mode divisi).
// Siklus hidupnya dibaca dari transkrip koordinator: tool_use Agent = sub agent mulai, tool_result dengan
// tool_use_id yang sama = sub agent melapor & ditutup. Transkrip sub agent sendiri
// (<proyek>/<session>/subagents/*.jsonl) hanya memperkaya aksi yang sedang dikerjakan.

// SubagentName adalah nama tampilan sub agent Claude Code di Kantor Agent.
const SubagentName = "Anak Buah Claude"

// DivisionPlanTool adalah tool MCP yang dipanggil koordinator untuk mengambil todo (mode divisi).
const DivisionPlanTool = "vibeasik_todo_dispatch_plan"

var subTodoRef = regexp.MustCompile(`(?m)^## Todo #(\d+)\b`)

type claudeSub struct {
	useID, key, task, action, model, role, agentID string
	todoID                                         uint
	background, done                               bool
	last                                           time.Time
}

// subagentTracker mencatat sub agent yang disuruh satu sesi Claude Code, urut waktu mulai.
type subagentTracker struct {
	parentKey string
	list      []*claudeSub
	byUse     map[string]*claudeSub
}

func newSubagentTracker(parentKey string) subagentTracker {
	return subagentTracker{parentKey: parentKey, byUse: map[string]*claudeSub{}}
}

// SubagentKey menurunkan key sub agent dari key koordinator dan prompt yang diberikan kepadanya, sehingga
// transkrip koordinator maupun transkrip sub agent itu sendiri memperbarui orang yang sama.
func SubagentKey(parentKey, prompt string) string {
	h := fnv.New64a()
	h.Write([]byte(strings.TrimSpace(prompt)))
	return fmt.Sprintf("%s:sub:%012x", parentKey, h.Sum64()&0xffffffffffff)
}

// start dipanggil untuk tool_use Agent/Task di transkrip koordinator.
func (t *subagentTracker) start(useID string, in map[string]interface{}, at time.Time, parentModel string) {
	str := func(k string) string { v, _ := in[k].(string); return strings.TrimSpace(v) }
	prompt := str("prompt")
	if useID == "" || prompt == "" || t.byUse[useID] != nil {
		return
	}
	sub := &claudeSub{useID: useID, key: SubagentKey(t.parentKey, prompt), model: util.FirstNonEmpty(str("model"), parentModel),
		role: util.FirstNonEmpty(str("subagent_type"), "general-purpose"), last: at, action: "Menerima tugas dari koordinator"}
	sub.task = util.FirstNonEmpty(strings.TrimPrefix(firstLine(prompt), "Tugas: "), str("description"))
	if m := subTodoRef.FindStringSubmatch(prompt); m != nil {
		if id, err := strconv.Atoi(m[1]); err == nil {
			sub.todoID = uint(id)
		}
	}
	sub.background, _ = in["run_in_background"].(bool)
	t.list = append(t.list, sub)
	t.byUse[useID] = sub
}

// result dipanggil untuk tool_result: sub agent foreground selesai melapor. Sub agent background baru
// diluncurkan; ia selesai saat notifikasi tugasnya datang.
func (t *subagentTracker) result(useID string, raw json.RawMessage, at time.Time) {
	sub := t.byUse[useID]
	if sub == nil || sub.done {
		return
	}
	var res struct {
		AgentID string `json:"agentId"`
	}
	_ = json.Unmarshal(raw, &res)
	if res.AgentID != "" {
		sub.agentID = res.AgentID
	}
	if sub.background {
		sub.action, sub.last = "Bekerja di latar belakang", at
		return
	}
	t.close(sub, at)
}

// notification menutup sub agent background yang dilaporkan selesai lewat <task-notification>:
// dicocokkan dengan agentId-nya, atau sub agent background tertua yang masih terbuka.
func (t *subagentTracker) notification(text string, at time.Time) {
	var oldest *claudeSub
	for _, sub := range t.list {
		if !sub.background || sub.done {
			continue
		}
		if sub.agentID != "" && strings.Contains(text, sub.agentID) {
			t.close(sub, at)
			return
		}
		if oldest == nil {
			oldest = sub
		}
	}
	if oldest != nil {
		t.close(oldest, at)
	}
}

// closeForeground: giliran koordinator berakhir (atau dihentikan), jadi sub agent foreground pasti sudah berhenti.
func (t *subagentTracker) closeForeground(at time.Time) {
	for _, sub := range t.list {
		if !sub.background && !sub.done {
			t.close(sub, at)
		}
	}
}

func (t *subagentTracker) close(sub *claudeSub, at time.Time) {
	sub.done, sub.last = true, at
	sub.action = "Melapor ke koordinator"
}

// observations mengembalikan keadaan setiap sub agent untuk ditulis sebagai pekerja anak buah.
func (t *subagentTracker) observations(projectPath string) []Observation {
	out := make([]Observation, 0, len(t.list))
	for _, sub := range t.list {
		o := Observation{Key: sub.key, ParentKey: t.parentKey, Name: SubagentName, ClientName: "claude-code-subagent",
			Transport: "transcript", Model: sub.model, Role: models.WorkerRoleSubagent + " · " + sub.role, Task: sub.task,
			Action: sub.action, Status: models.WorkerStatusWorking, ProjectPath: projectPath, LastEvent: sub.last, TodoID: sub.todoID,
			TodoStatus: models.TodoStatusInProgress}
		if sub.done {
			o.Status = models.WorkerStatusDone
			if sub.todoID > 0 {
				o.TodoStatus = models.TodoStatusCompleted
			} else {
				o.TasksDone = 1
			}
		}
		out = append(out, o)
	}
	return out
}

// ---------------------------------------------------------------------------
// Transkrip sub agent: ~/.claude/projects/<proyek>/<session>/subagents/*.jsonl

type claudeSubFileParser struct {
	*claudeParser
	session string
	prompt  string
}

func newClaudeSubFileParser(path string) transcriptParser {
	session := filepath.Base(filepath.Dir(filepath.Dir(path)))
	p := newClaudeParser(filepath.Join(filepath.Dir(path), session+".jsonl")).(*claudeParser)
	p.sidechain = true
	return &claudeSubFileParser{claudeParser: p, session: session}
}

func (p *claudeSubFileParser) feed(line []byte) {
	if p.prompt == "" {
		var e struct {
			Type    string `json:"type"`
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &e) == nil && e.Type == "user" {
			p.prompt = claudeUserText(e.Message.Content)
		}
	}
	p.claudeParser.feed(line)
}

// observation: hanya aksi & model; siklus hidup dan label tugas milik transkrip koordinator. Pekerja
// yang belum dibuat koordinator (prompt tidak cocok) atau sudah ditutup tidak disentuh.
func (p *claudeSubFileParser) observation() Observation {
	o := p.claudeParser.observation()
	if p.prompt == "" {
		return Observation{}
	}
	parent := "report:claude-code:" + slug(p.session)
	return Observation{Key: SubagentKey(parent, p.prompt), ParentKey: parent, Name: SubagentName, Model: o.Model,
		Action: o.Action, Status: models.WorkerStatusWorking, LastEvent: o.LastEvent, OnlyExisting: true}
}

// claudeUserText mengambil teks pesan user Claude (string atau blok text pertama).
func claudeUserText(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return strings.TrimSpace(text)
	}
	var blocks []claudeBlock
	_ = json.Unmarshal(raw, &blocks)
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			return strings.TrimSpace(b.Text)
		}
	}
	return ""
}
