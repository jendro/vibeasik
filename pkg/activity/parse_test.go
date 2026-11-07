package activity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func feedAll(p transcriptParser, lines ...string) Observation {
	for _, l := range lines {
		p.feed([]byte(l))
	}
	return p.observation()
}

func TestAntigravityParser(t *testing.T) {
	p := newAntigravityParser("/h/.gemini/antigravity/brain/conv-1/.system_generated/logs/transcript.jsonl", "Antigravity", "antigravity")
	o := feedAll(p,
		`{"step_index":0,"type":"USER_INPUT","status":"DONE","created_at":"2026-10-24T03:48:26Z","content":"<USER_REQUEST>\nkerjakan todo yang ada di vibeasik\n</USER_REQUEST>\n<ADDITIONAL_METADATA>x</ADDITIONAL_METADATA>"}`,
		`{"step_index":2,"type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-10-24T03:48:27Z","tool_calls":[{"name":"run_command","args":{"CommandLine":"\"go test ./...\"","Cwd":"\"/home/dev/vibeasik\"","toolAction":"\"Running go tests\"","toolSummary":"\"Run tests\""}}]}`,
	)
	if o.Key != "antigravity:conv-1" || o.Task != "kerjakan todo yang ada di vibeasik" || o.Action != "Running go tests" || o.Status != "working" || o.ProjectPath != "/home/dev/vibeasik" {
		t.Fatalf("working: %+v", o)
	}
	o = feedAll(p, `{"step_index":3,"type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-10-24T03:50:00Z"}`)
	if o.Action != actionThinking || o.Status != "working" {
		t.Fatalf("thinking: %+v", o)
	}
	o = feedAll(p, `{"step_index":4,"type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-10-24T03:51:00Z","content":"Semua todo selesai."}`)
	if o.Status != "idle" || o.TasksDone != 1 || !o.LastEvent.Equal(time.Date(2026, 10, 24, 3, 51, 0, 0, time.UTC)) {
		t.Fatalf("final answer: %+v", o)
	}
}

func TestClaudeParser(t *testing.T) {
	p := newClaudeParser("/h/.claude/projects/-home-dev-app/d2e811d9-aaaa.jsonl")
	o := feedAll(p,
		`{"type":"user","timestamp":"2026-10-24T03:00:00Z","cwd":"/home/dev/app","message":{"role":"user","content":[{"type":"text","text":"<system-reminder>ctx</system-reminder>"},{"type":"text","text":"Buat halaman login\ndetail"}]}}`,
		`{"type":"assistant","timestamp":"2026-10-24T03:00:05Z","message":{"id":"m1","model":"claude-opus-5-5","stop_reason":"tool_use","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/home/dev/app/Login.vue"}}]}}`,
		`{"type":"assistant","isSidechain":true,"timestamp":"2026-10-24T03:00:06Z","message":{"id":"s1","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/x/sub.go"}}]}}`,
	)
	if o.Key != "report:claude-code:d2e811d9-aaaa" || o.Task != "Buat halaman login" || o.Action != "Mengedit Login.vue" || o.Model != "claude-opus-5-5" || o.Status != "working" {
		t.Fatalf("working: %+v", o)
	}
	o = feedAll(p,
		`{"type":"assistant","timestamp":"2026-10-24T03:01:00Z","message":{"id":"m2","stop_reason":"end_turn","content":[{"type":"thinking","thinking":"..."}]}}`,
		`{"type":"assistant","timestamp":"2026-10-24T03:01:01Z","message":{"id":"m2","stop_reason":"end_turn","content":[{"type":"text","text":"Selesai"}]}}`,
	)
	if o.Status != "idle" || o.TasksDone != 1 || o.Action != actionWaiting {
		t.Fatalf("end_turn counted once: %+v", o)
	}
}

func TestCopilotCLIParser(t *testing.T) {
	p := newCopilotCLIParser("/h/.copilot/session-state/sess-9/events.jsonl")
	o := feedAll(p,
		`{"type":"session.start","data":{"context":{"cwd":"/home/dev/tenan"}},"timestamp":"2026-10-24T01:00:00Z"}`,
		`{"type":"user.message","data":{"content":"perbaiki bug jurnal"},"timestamp":"2026-10-24T01:00:01Z"}`,
		`{"type":"tool.execution_start","data":{"toolName":"bash","toolTitle":"Running tests","model":"claude-sonnet-5"},"timestamp":"2026-10-24T01:00:02Z"}`,
	)
	if o.Key != "copilot-cli:sess-9" || o.Task != "perbaiki bug jurnal" || o.Action != "Running tests" || o.Model != "claude-sonnet-5" || o.ProjectPath != "/home/dev/tenan" {
		t.Fatalf("working: %+v", o)
	}
	o = feedAll(p,
		`{"type":"assistant.message","data":{"content":"Sudah diperbaiki","toolRequests":[]},"timestamp":"2026-10-24T01:01:00Z"}`,
		`{"type":"session.shutdown","data":{},"timestamp":"2026-10-24T01:02:00Z"}`,
	)
	if o.TasksDone != 1 || o.Status != "left" {
		t.Fatalf("done+shutdown: %+v", o)
	}
}

func TestCopilotChatParser(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "ws1", "chatSessions"), 0o755)
	os.WriteFile(filepath.Join(dir, "ws1", "workspace.json"), []byte(`{"folder":"file:///home/dev/erp"}`), 0o644)
	p := newCopilotChatParser(filepath.Join(dir, "ws1", "chatSessions", "chat-1.jsonl"))
	o := feedAll(p,
		`{"kind":0,"v":{"sessionId":"chat-1","requests":[]}}`,
		`{"kind":2,"k":["requests"],"v":[{"timestamp":1792800000000,"modelId":"copilot/gpt-5","message":{"text":"fix purchase order"},"response":[]}]}`,
		`{"kind":2,"k":["requests",0,"response"],"v":[{"kind":"toolInvocationSerialized","invocationMessage":{"value":"Reading [](file:///home/dev/erp/po_handler.go#120-120), lines 120 to 280"}}]}`,
	)
	if o.Key != "copilot-vscode:chat-1" || o.Task != "fix purchase order" || o.Action != "Reading po_handler.go, lines 120 to 280" || o.Model != "gpt-5" || o.ProjectPath != "/home/dev/erp" || o.Status != "working" {
		t.Fatalf("working: %+v", o)
	}
	o = feedAll(p, `{"kind":1,"k":["requests",0,"modelState"],"v":{"value":1,"completedAt":1792800060000}}`)
	if o.Status != "idle" || o.TasksDone != 1 {
		t.Fatalf("completed: %+v", o)
	}
}

func TestScanSourcesReadsIncrementally(t *testing.T) {
	d := setupDB(t)
	dir := t.TempDir()
	logDir := filepath.Join(dir, "brain", "conv-7", ".system_generated", "logs")
	os.MkdirAll(logDir, 0o755)
	file := filepath.Join(logDir, "transcript.jsonl")
	now := time.Now().UTC()
	line := func(step int, body string) string {
		return `{"step_index":` + string(rune('0'+step)) + `,"created_at":"` + now.Format(time.RFC3339) + `",` + body + "}\n"
	}
	os.WriteFile(file, []byte(line(0, `"type":"USER_INPUT","content":"<USER_REQUEST>tulis test</USER_REQUEST>"`)), 0o644)

	watchMu.Lock()
	sources = []transcriptSource{{Agent: "Antigravity", Dir: filepath.Join(dir, "brain"),
		files: func() []string {
			m, _ := filepath.Glob(filepath.Join(dir, "brain", "*", ".system_generated", "logs", "transcript.jsonl"))
			return m
		},
		parser: func(p string) transcriptParser { return newAntigravityParser(p, "Antigravity", "antigravity") }}}
	tracked = map[string]*trackedFile{}
	watchMu.Unlock()
	t.Cleanup(func() { watchMu.Lock(); sources, tracked = nil, map[string]*trackedFile{}; watchMu.Unlock() })

	scanSources(time.Now())
	w := worker(t, d, "antigravity:conv-7")
	if w.Task != "tulis test" || w.Status != models.WorkerStatusWorking {
		t.Fatalf("first scan: %+v", w)
	}

	// Baris yang belum lengkap tidak boleh diproses sampai agent selesai menulisnya.
	f, _ := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0o644)
	full := line(1, `"type":"PLANNER_RESPONSE","content":"Selesai"`)
	f.WriteString(full[:20])
	f.Close()
	scanSources(time.Now().Add(time.Second))
	if w := worker(t, d, "antigravity:conv-7"); w.Status != models.WorkerStatusWorking {
		t.Fatalf("partial line must be ignored: %+v", w)
	}
	f, _ = os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString(full[20:])
	f.Close()
	scanSources(time.Now().Add(2 * time.Second))
	w = worker(t, d, "antigravity:conv-7")
	if w.Status != models.WorkerStatusIdle || w.TasksDone != 1 {
		t.Fatalf("completed line: %+v", w)
	}
	if st := WatchedSources(); len(st) != 1 || !st[0].Found || st[0].ActiveSessions != 1 {
		t.Fatalf("sources: %+v", st)
	}
}

func TestMergeDuplicateMCP(t *testing.T) {
	ws := mergeDuplicateMCP([]models.AgentWorker{
		{Key: "antigravity:c1", Name: "Antigravity"},
		{Key: "mcp-stdio:10", Name: "Antigravity", ToolCalls: 4},
		{Key: "mcp-http:x", Name: "Cline", ToolCalls: 0},
		{Key: "mcp-http:y", Name: "Cline 2", ToolCalls: 2},
	})
	var keys []string
	for _, w := range ws {
		keys = append(keys, w.Key)
	}
	if strings.Join(keys, ",") != "antigravity:c1,mcp-http:y" {
		t.Fatalf("merged: %v", keys)
	}
}

func TestCleanChatMarkdown(t *testing.T) {
	got := cleanChatMarkdown("Searching for regex `CreateJournal` in [](file:///a/b%20c.go#1) and [docs](https://x.y)")
	if got != "Searching for regex CreateJournal in b c.go and docs" {
		t.Fatalf("got %q", got)
	}
}

func TestMCPTodoLinksToObservedAgent(t *testing.T) {
	d := setupDB(t)
	todo := models.Todo{Title: "Perbaiki scanner"}
	d.Create(&todo)
	ApplyObservation(Observation{Key: "antigravity:c1", Name: "Antigravity", Task: "kerjakan todo", Status: "working", LastEvent: time.Now()}, 0, false, true)

	c := Caller{Key: "mcp-stdio:77", Transport: "stdio"}
	ClientConnected(c, "antigravity", "2.13")
	ToolCalled(c, "update_todo", map[string]interface{}{"todo_id": float64(todo.ID), "status": "in_progress"})

	w := worker(t, d, "antigravity:c1")
	if w.TodoID == nil || *w.TodoID != todo.ID || w.Task != todo.Title {
		t.Fatalf("todo must be linked to the observed Antigravity session: %+v", w)
	}
	visible, _ := VisibleWorkers(time.Now())
	if len(visible) != 1 || visible[0].Key != "antigravity:c1" {
		t.Fatalf("MCP connection must merge into the observed agent, got %+v", visible)
	}
}

func TestTranscriptTodoCallLinksTodo(t *testing.T) {
	d := setupDB(t)
	todo := models.Todo{Title: "Rapikan halaman todo"}
	d.Create(&todo)
	p := newAntigravityParser("/h/.gemini/antigravity/brain/c9/.system_generated/logs/transcript.jsonl", "Antigravity", "antigravity")
	call := func(status string) string {
		return `{"type":"PLANNER_RESPONSE","created_at":"2026-10-24T04:00:00Z","tool_calls":[{"name":"call_mcp_tool","args":{"Arguments":"{\"todo_id\":` +
			fmt.Sprint(todo.ID) + `,\"status\":\"` + status + `\"}","ServerName":"\"vibeasik\"","ToolName":"\"update_todo\"","toolAction":"\"Updating TODO\""}}]}`
	}
	tf := &trackedFile{parser: p}
	p.feed([]byte(`{"type":"USER_INPUT","created_at":"2026-10-24T03:59:00Z","content":"<USER_REQUEST>kerjakan todo</USER_REQUEST>"}`))
	p.feed([]byte(call("in_progress")))
	applyTracked(tf, time.Now())
	w := worker(t, d, "antigravity:c9")
	if w.TodoID == nil || *w.TodoID != todo.ID || w.Task != todo.Title {
		t.Fatalf("in_progress from transcript: %+v", w)
	}
	p.feed([]byte(call("completed")))
	applyTracked(tf, time.Now())
	// Laporan kedua dari koneksi MCP untuk todo yang sama tidak boleh menambah hitungan.
	ApplyObservation(Observation{Key: "antigravity:c9", Name: "Antigravity", Status: "working", TodoID: todo.ID, TodoStatus: "completed", LastEvent: time.Now()}, 0, false, false)
	w = worker(t, d, "antigravity:c9")
	if w.TodoID != nil || w.TasksDone != 1 {
		t.Fatalf("completed once: %+v", w)
	}
}
