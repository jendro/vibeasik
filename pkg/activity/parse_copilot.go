package activity

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// ---------------------------------------------------------------------------
// Copilot CLI (juga agent Copilot CLI yang dibuka dari VS Code): ~/.copilot/session-state/<id>/events.jsonl

type copilotCLIParser struct{ sessionState }

func newCopilotCLIParser(path string) transcriptParser {
	id := filepath.Base(filepath.Dir(path))
	return &copilotCLIParser{sessionState{key: "copilot-cli:" + id, name: "Copilot CLI", client: "copilot-cli", transport: "transcript"}}
}

type copilotCLIData struct {
	Context struct {
		Cwd string `json:"cwd"`
	} `json:"context"`
	NewModel     string          `json:"newModel"`
	Model        string          `json:"model"`
	Content      string          `json:"content"`
	ToolName     string          `json:"toolName"`
	ToolTitle    string          `json:"toolTitle"`
	Arguments    json.RawMessage `json:"arguments"`
	ToolRequests []any           `json:"toolRequests"`
	Message      string          `json:"message"`
}

func (p *copilotCLIParser) feed(line []byte) {
	var e struct {
		Type      string         `json:"type"`
		Timestamp string         `json:"timestamp"`
		Data      copilotCLIData `json:"data"`
	}
	if json.Unmarshal(line, &e) != nil {
		return
	}
	p.at(e.Timestamp)
	if m := util.FirstNonEmpty(e.Data.NewModel, e.Data.Model); m != "" {
		p.model = m
	}
	p.handleEvent(e.Type, e.Data)
}

func (p *copilotCLIParser) handleEvent(eventType string, d copilotCLIData) {
	switch eventType {
	case "session.start", "session.resume":
		p.workPath(d.Context.Cwd)
		p.finished("Sesi dimulai")
	case "user.message":
		p.prompt(d.Content)
	case "assistant.turn_start", "permission.completed":
		p.status = models.WorkerStatusWorking
	case "tool.execution_start":
		p.handleToolExecution(d)
	case "assistant.message":
		p.handleAssistantMessage(d)
	case "permission.requested":
		p.finished(actionNeedsPermission)
	case "abort":
		p.finished("Dihentikan pengguna")
	case "session.error":
		p.finished("Error: " + firstLine(d.Message))
	case "session.shutdown":
		p.status, p.action = models.WorkerStatusLeft, "Sesi berakhir"
	}
}

func (p *copilotCLIParser) handleToolExecution(d copilotCLIData) {
	var args map[string]interface{}
	_ = json.Unmarshal(d.Arguments, &args)
	detail := ""
	for _, k := range []string{"description", "path", "command", "pattern", "query"} {
		if v, ok := args[k].(string); ok && v != "" {
			detail = v
			break
		}
	}
	p.todoCall(d.ToolName, args)
	p.status = models.WorkerStatusWorking
	p.action = util.FirstNonEmpty(d.ToolTitle, strings.TrimSpace(d.ToolName+" "+detail))
}

func (p *copilotCLIParser) handleAssistantMessage(d copilotCLIData) {
	if len(d.ToolRequests) == 0 && strings.TrimSpace(d.Content) != "" {
		p.done++
		p.finished(actionWaiting)
	}
}

// ---------------------------------------------------------------------------
// Copilot Chat di VS Code: <config>/Code/User/workspaceStorage/<ws>/chatSessions/<id>.jsonl
// Formatnya log patch: kind 0 = snapshot awal, kind 1 = set nilai di path k, kind 2 = tambah item di path k.

type copilotChatParser struct {
	sessionState
	latest int // indeks request terakhir
}

func newCopilotChatParser(path string) transcriptParser {
	id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	p := &copilotChatParser{sessionState: sessionState{key: "copilot-vscode:" + id, name: "Copilot Chat", client: "vscode-copilot-chat", transport: "transcript"}, latest: -1}
	p.path = vscodeWorkspacePath(filepath.Join(filepath.Dir(filepath.Dir(path)), "workspace.json"))
	return p
}

type chatRequest struct {
	ModelID    string `json:"modelId"`
	Timestamp  int64  `json:"timestamp"`
	ModelState *struct {
		Value       int   `json:"value"`
		CompletedAt int64 `json:"completedAt"`
	} `json:"modelState"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
	Response []json.RawMessage `json:"response"`
}

func (p *copilotChatParser) feed(line []byte) {
	var e struct {
		Kind int             `json:"kind"`
		K    []any           `json:"k"`
		V    json.RawMessage `json:"v"`
	}
	if json.Unmarshal(line, &e) != nil {
		return
	}
	key := copilotPathKey(e.K)
	p.handleKind(e.Kind, key, e.V)
}

func copilotPathKey(k []any) string {
	path := make([]string, len(k))
	for i, item := range k {
		if s, ok := item.(string); ok {
			path[i] = s
		} else {
			path[i] = "#"
		}
	}
	return strings.Join(path, "/")
}

func (p *copilotChatParser) handleKind(kind int, key string, v json.RawMessage) {
	switch {
	case kind == 0:
		p.handleSnapshot(v)
	case kind == 2 && key == "requests":
		p.handleAppendRequests(v)
	case kind == 2 && key == "requests/#/response":
		p.handleAppendResponse(v)
	case kind == 1 && key == "requests/#/modelState":
		p.handleModelState(v)
	}
}

func (p *copilotChatParser) handleSnapshot(v json.RawMessage) {
	var snap struct {
		Requests []chatRequest `json:"requests"`
	}
	if json.Unmarshal(v, &snap) == nil {
		p.addRequests(snap.Requests)
	}
}

func (p *copilotChatParser) handleAppendRequests(v json.RawMessage) {
	var reqs []chatRequest
	if json.Unmarshal(v, &reqs) == nil {
		p.addRequests(reqs)
	}
}

func (p *copilotChatParser) handleAppendResponse(v json.RawMessage) {
	var parts []json.RawMessage
	if json.Unmarshal(v, &parts) == nil {
		p.responseParts(parts)
	}
}

func (p *copilotChatParser) handleModelState(v json.RawMessage) {
	var st struct {
		Value       int   `json:"value"`
		CompletedAt int64 `json:"completedAt"`
	}
	if json.Unmarshal(v, &st) == nil && st.CompletedAt > 0 {
		p.complete(st.Value, st.CompletedAt)
	}
}

func (p *copilotChatParser) addRequests(reqs []chatRequest) {
	for _, r := range reqs {
		p.latest++
		p.prompt(r.Message.Text)
		if r.ModelID != "" {
			p.model = strings.TrimPrefix(r.ModelID, "copilot/")
		}
		if r.Timestamp > 0 {
			p.atMillis(r.Timestamp)
		}
		p.responseParts(r.Response)
		if r.ModelState != nil && r.ModelState.CompletedAt > 0 {
			p.complete(r.ModelState.Value, r.ModelState.CompletedAt)
		}
	}
}

func (p *copilotChatParser) responseParts(parts []json.RawMessage) {
	for _, raw := range parts {
		var part struct {
			Kind              string          `json:"kind"`
			InvocationMessage json.RawMessage `json:"invocationMessage"`
		}
		if json.Unmarshal(raw, &part) != nil || part.Kind != "toolInvocationSerialized" {
			continue
		}
		var msg struct {
			Value string `json:"value"`
		}
		text := ""
		if json.Unmarshal(part.InvocationMessage, &msg) == nil {
			text = msg.Value
		} else {
			_ = json.Unmarshal(part.InvocationMessage, &text)
		}
		if text = cleanChatMarkdown(text); text != "" {
			p.status, p.action = models.WorkerStatusWorking, text
		}
	}
}

func (p *copilotChatParser) complete(value int, completedAt int64) {
	p.atMillis(completedAt)
	if value == 1 {
		p.done++
		p.finished(actionWaiting)
		return
	}
	p.finished("Dibatalkan / gagal")
}

func (p *copilotChatParser) atMillis(ms int64) {
	if t := time.UnixMilli(ms); t.After(p.last) {
		p.last = t
	}
}

var (
	chatFileLink = regexp.MustCompile(`\[[^\]]*\]\(file://([^)#]+)(#[^)]*)?\)`)
	chatLink     = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
)

// cleanChatMarkdown mengubah "Reading [](file:///a/b.go#1-2), lines 1 to 2" menjadi "Reading b.go, lines 1 to 2".
func cleanChatMarkdown(s string) string {
	s = chatFileLink.ReplaceAllStringFunc(s, func(m string) string {
		sub := chatFileLink.FindStringSubmatch(m)
		p, _ := url.PathUnescape(sub[1])
		return filepath.Base(p)
	})
	s = chatLink.ReplaceAllString(s, "$1")
	return strings.TrimSpace(strings.ReplaceAll(s, "`", ""))
}

// vscodeWorkspacePath membaca folder/workspace yang dibuka VS Code untuk sebuah workspaceStorage.
func vscodeWorkspacePath(file string) string {
	raw, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	var ws struct {
		Folder    string `json:"folder"`
		Workspace string `json:"workspace"`
	}
	if json.Unmarshal(raw, &ws) != nil {
		return ""
	}
	u, err := url.Parse(util.FirstNonEmpty(ws.Folder, ws.Workspace))
	if err != nil || u.Scheme != "file" {
		return ""
	}
	if ws.Folder == "" {
		return filepath.Dir(u.Path) // file .code-workspace → folder tempatnya
	}
	return u.Path
}
