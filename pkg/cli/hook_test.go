package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHookEventToReport(t *testing.T) {
	cases := []struct {
		raw, status, action, task string
	}{
		{`{"hook_event_name":"UserPromptSubmit","session_id":"s1","cwd":"/w/app","prompt":"Buat halaman login"}`, "working", "Membaca instruksi", "Buat halaman login"},
		{`{"hook_event_name":"PreToolUse","tool_name":"Edit","tool_input":{"file_path":"/w/app/src/Login.vue"}}`, "working", "Mengedit Login.vue", ""},
		{`{"hook_event_name":"PreToolUse","tool_name":"TodoWrite","tool_input":{"todos":[{"content":"A","status":"completed"},{"content":"Tulis test","activeForm":"Menulis test","status":"in_progress"}]}}`, "working", "Memperbarui daftar tugas", "Menulis test"},
		{`{"hook_event_name":"PreToolUse","tool_name":"mcp__vibeasik__list_todos","tool_input":{}}`, "working", "MCP: list_todos", ""},
		{`{"hook_event_name":"Stop"}`, "idle", "Menunggu instruksi berikutnya", ""},
		{`{"hook_event_name":"SessionEnd"}`, "left", "Sesi berakhir", ""},
	}
	for _, c := range cases {
		r, ok := hookEventToReport("claude-code", []byte(c.raw))
		if !ok {
			t.Fatalf("not mapped: %s", c.raw)
		}
		task, _ := r["task"].(string)
		if r["status"] != c.status || r["action"] != c.action || task != c.task {
			t.Errorf("%s -> %v", c.raw, r)
		}
	}
	if _, ok := hookEventToReport("claude-code", []byte(`{"hook_event_name":"SubagentStop"}`)); ok {
		t.Error("unknown events must be ignored")
	}
	if _, ok := hookEventToReport("claude-code", []byte(`not json`)); ok {
		t.Error("invalid json must be ignored")
	}
}

func TestInstallClaudeHooksIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(`{"model":"opus","hooks":{"Stop":[{"hooks":[{"type":"command","command":"say done"}]}]}}`), 0o644)

	added, err := installClaudeHooks(path, "vibeasik hook claude-code")
	if err != nil || added != len(hookEvents) {
		t.Fatalf("first install added=%d err=%v", added, err)
	}
	if again, _ := installClaudeHooks(path, "vibeasik hook claude-code"); again != 0 {
		t.Fatalf("second install must add nothing, added %d", again)
	}
	var s map[string]interface{}
	raw, _ := os.ReadFile(path)
	json.Unmarshal(raw, &s)
	stop := s["hooks"].(map[string]interface{})["Stop"].([]interface{})
	if s["model"] != "opus" || len(stop) != 2 {
		t.Fatalf("existing settings must be kept: %s", raw)
	}
}
