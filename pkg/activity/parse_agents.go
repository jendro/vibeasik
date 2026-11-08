package activity

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
)

const (
	actionReadingPrompt   = "Membaca instruksi"
	actionThinking        = "Berpikir…"
	actionWaiting         = "Menunggu instruksi berikutnya"
	actionNeedsPermission = "Menunggu izin pengguna"
)

// sessionState adalah keadaan bersama yang diisi tiap parser.
type sessionState struct {
	key, name, client, transport string
	model, task, action, status  string
	path                         string
	done                         int
	last                         time.Time
	todoID                       uint
	todoStatus                   string
	coordinator                  bool // sesi memanggil rencana divisi: agent pengambil todo
}

func (s *sessionState) observation() Observation {
	return Observation{Key: s.key, Name: s.name, ClientName: s.client, Transport: s.transport, Model: s.model, Task: s.task,
		Action: s.action, Status: s.status, ProjectPath: s.path, TasksDone: s.done, LastEvent: s.last,
		TodoID: s.todoID, TodoStatus: s.todoStatus, Role: s.role()}
}

func (s *sessionState) role() string {
	if s.coordinator {
		return models.WorkerRoleCoordinator
	}
	return ""
}

// workPath mencatat path kerja, kecuali file internal milik agent (transkrip, cache, output langkah).
func (s *sessionState) workPath(path string) {
	if !strings.HasPrefix(path, "/") {
		return
	}
	for _, internal := range []string{"/.gemini/", "/.claude/", "/.copilot/", "/.config/Code/"} {
		if strings.Contains(path, internal) {
			return
		}
	}
	s.path = path
}

// todoCall mencatat panggilan tool MCP vibeasik update_todo yang terlihat di transkrip.
func (s *sessionState) todoCall(tool string, args map[string]interface{}) {
	if strings.HasSuffix(tool, DivisionPlanTool) {
		s.coordinator = true
	}
	if !strings.HasSuffix(tool, "update_todo") {
		return
	}
	if id := argUint(args, "todo_id"); id > 0 {
		s.todoID, s.todoStatus = id, argString(args, "status")
	}
}

func (s *sessionState) at(ts string) {
	if t, err := time.Parse(time.RFC3339Nano, ts); err == nil && t.After(s.last) {
		s.last = t
	}
}

func (s *sessionState) prompt(text string) {
	if t := firstLine(text); t != "" {
		s.task = t
	}
	s.status, s.action = models.WorkerStatusWorking, actionReadingPrompt
}

func (s *sessionState) finished(action string) {
	s.status, s.action = models.WorkerStatusIdle, action
}

// ---------------------------------------------------------------------------
// Antigravity (IDE & CLI): ~/.gemini/antigravity{,-cli}/brain/<percakapan>/.system_generated/logs/transcript.jsonl

type antigravityParser struct{ sessionState }

func newAntigravityParser(path, name, client string) transcriptParser {
	conv := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
	return &antigravityParser{sessionState{key: client + ":" + conv, name: name, client: client, transport: "transcript"}}
}

func (p *antigravityParser) feed(line []byte) {
	var e struct {
		Type      string `json:"type"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
		Content   string `json:"content"`
		ToolCalls []struct {
			Name string                     `json:"name"`
			Args map[string]json.RawMessage `json:"args"`
		} `json:"tool_calls"`
	}
	if json.Unmarshal(line, &e) != nil {
		return
	}
	p.at(e.CreatedAt)
	switch e.Type {
	case "USER_INPUT":
		p.prompt(between(e.Content, "<USER_REQUEST>", "</USER_REQUEST>"))
	case "PLANNER_RESPONSE":
		if len(e.ToolCalls) > 0 {
			for _, tc := range e.ToolCalls {
				for _, k := range []string{"Cwd", "TargetFile", "AbsolutePath", "SearchPath", "DirectoryPath"} {
					p.workPath(argText(tc.Args[k]))
				}
				if tc.Name == "call_mcp_tool" {
					var args map[string]interface{}
					_ = json.Unmarshal([]byte(argText(tc.Args["Arguments"])), &args)
					p.todoCall(argText(tc.Args["ToolName"]), args)
				}
			}
			tc := e.ToolCalls[len(e.ToolCalls)-1]
			action := util.FirstNonEmpty(argText(tc.Args["toolAction"]), argText(tc.Args["toolSummary"]))
			if action == "" && tc.Name == "call_mcp_tool" {
				action = "MCP: " + argText(tc.Args["ToolName"])
			}
			p.status, p.action = models.WorkerStatusWorking, util.FirstNonEmpty(action, tc.Name)
		} else if strings.TrimSpace(e.Content) != "" {
			p.done++
			p.finished(actionWaiting)
		} else {
			p.status, p.action = models.WorkerStatusWorking, actionThinking
		}
	case "GENERIC":
		if e.Status == "RUNNING" {
			p.status = models.WorkerStatusWorking
		}
	case "ERROR_MESSAGE":
		p.finished("Berhenti karena error")
	}
}

// argText membaca argumen tool Antigravity yang disimpan sebagai string JSON (kadang berlapis dua).
func argText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return ""
	}
	if strings.HasPrefix(s, `"`) {
		var inner string
		if json.Unmarshal([]byte(s), &inner) == nil {
			s = inner
		}
	}
	return strings.TrimSpace(s)
}

func between(s, open, close string) string {
	if i := strings.Index(s, open); i >= 0 {
		s = s[i+len(open):]
		if j := strings.Index(s, close); j >= 0 {
			s = s[:j]
		}
	}
	return s
}

// ---------------------------------------------------------------------------
// Claude Code: ~/.claude/projects/<proyek>/<session>.jsonl (termasuk sesi aplikasi desktop)

type claudeParser struct {
	sessionState
	endedMessages map[string]bool
	subs          subagentTracker
	sidechain     bool // transkrip sub agent: semua barisnya isSidechain
}

type claudeBlock struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id"`
	ToolUseID string                 `json:"tool_use_id"`
	Text      string                 `json:"text"`
	Name      string                 `json:"name"`
	Input     map[string]interface{} `json:"input"`
}

func newClaudeParser(path string) transcriptParser {
	id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	// Key sama dengan laporan `vibeasik hook claude-code` sehingga hook & transkrip memperbarui orang yang sama.
	key := "report:claude-code:" + slug(id)
	return &claudeParser{
		sessionState:  sessionState{key: key, name: "Claude Code", client: "claude-code", transport: "transcript"},
		endedMessages: map[string]bool{},
		subs:          newSubagentTracker(key),
	}
}

func (p *claudeParser) feed(line []byte) {
	var e struct {
		Type          string          `json:"type"`
		Timestamp     string          `json:"timestamp"`
		Cwd           string          `json:"cwd"`
		IsSidechain   bool            `json:"isSidechain"`
		IsMeta        bool            `json:"isMeta"`
		ToolUseResult json.RawMessage `json:"toolUseResult"`
		Message       struct {
			ID         string          `json:"id"`
			Model      string          `json:"model"`
			StopReason string          `json:"stop_reason"`
			Content    json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &e) != nil || e.IsSidechain != p.sidechain || e.IsMeta || (e.Type != "user" && e.Type != "assistant") {
		return
	}
	p.at(e.Timestamp)
	p.workPath(e.Cwd)
	var blocks []claudeBlock
	var text string
	if json.Unmarshal(e.Message.Content, &text) != nil {
		_ = json.Unmarshal(e.Message.Content, &blocks)
	}

	if e.Type == "user" {
		if text != "" {
			blocks = append(blocks, claudeBlock{Type: "text", Text: text})
		}
		for _, b := range blocks {
			t := strings.TrimSpace(b.Text)
			switch {
			case b.Type == "tool_result":
				p.status = models.WorkerStatusWorking
				p.subs.result(b.ToolUseID, e.ToolUseResult, p.last)
			case b.Type == "text" && strings.HasPrefix(t, "<task-notification"):
				p.subs.notification(t, p.last)
			case b.Type != "text" || t == "" || strings.HasPrefix(t, "<"):
			case strings.HasPrefix(t, "[Request interrupted"):
				p.finished("Dihentikan pengguna")
				p.subs.closeForeground(p.last)
			default:
				p.prompt(t)
			}
		}
		return
	}

	if e.Message.Model != "" && !strings.HasPrefix(e.Message.Model, "<") {
		p.model = e.Message.Model
	}
	usedTool := false
	for _, b := range blocks {
		if b.Type != "tool_use" {
			continue
		}
		usedTool = true
		p.todoCall(b.Name, b.Input)
		if b.Name == "Agent" || b.Name == "Task" {
			p.subs.start(b.ID, b.Input, p.last, p.model)
			p.coordinator = true
		}
		p.status, p.action = models.WorkerStatusWorking, ClaudeToolAction(b.Name, b.Input)
		if task := ClaudeActiveTask(b.Name, b.Input); task != "" {
			p.task = task
		}
	}
	switch {
	case e.Message.StopReason == "end_turn" && !usedTool:
		if !p.endedMessages[e.Message.ID] {
			p.endedMessages[e.Message.ID] = true
			p.done++
		}
		p.finished(actionWaiting)
		p.subs.closeForeground(p.last)
	case !usedTool:
		p.status, p.action = models.WorkerStatusWorking, actionThinking
	}
}

// subagents: anak buah yang disuruh sesi ini (lihat subagents.go).
func (p *claudeParser) subagents() []Observation { return p.subs.observations(p.path) }
