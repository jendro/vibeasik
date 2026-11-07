package activity

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func jsonLine(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Mode divisi: sesi Claude Code yang menyuruh sub agent tampil sebagai koordinator, setiap sub agent sebagai
// anak buahnya (memegang todo-nya), dan sub agent ditutup begitu hasilnya kembali ke koordinator.
func TestClaudeSubagentsBecomeDivision(t *testing.T) {
	d := setupDB(t)
	todo := models.Todo{Title: "Rapikan A"}
	d.Create(&todo)
	now := time.Now().UTC()
	ts := func(sec int) string { return now.Add(time.Duration(sec) * time.Second).Format(time.RFC3339Nano) }
	prompt := fmt.Sprintf("Tugas: Rapikan A\n\nKamu sub agent di divisi vibeasik.\n\n## Todo #%d\nKategori: ui_ux\n", todo.ID)

	parent := newClaudeParser("/h/.claude/projects/-home-dev-app/sess-1.jsonl")
	tf := &trackedFile{parser: parent}
	feed := func(p transcriptParser, lines ...string) {
		for _, l := range lines {
			p.feed([]byte(l))
		}
	}
	feed(parent,
		`{"type":"user","timestamp":"`+ts(0)+`","cwd":"/home/dev/app","message":{"content":"kerjakan todo vibeasik"}}`,
		`{"type":"assistant","timestamp":"`+ts(1)+`","message":{"id":"m1","model":"claude-opus-5-5","content":[{"type":"tool_use","id":"t0","name":"mcp__vibeasik__vibeasik_todo_dispatch_plan","input":{"project_id":1}}]}}`,
		jsonLine(t, map[string]interface{}{"type": "assistant", "timestamp": ts(2), "message": map[string]interface{}{"id": "m2", "content": []interface{}{
			map[string]interface{}{"type": "tool_use", "id": "toolu_1", "name": "Agent", "input": map[string]interface{}{"description": "Todo A", "prompt": prompt, "subagent_type": "general-purpose"}},
		}}}),
	)
	applyTracked(tf, time.Now())

	lead := worker(t, d, "report:claude-code:sess-1")
	if lead.Role != models.WorkerRoleCoordinator || lead.ParentKey != "" {
		t.Fatalf("session that dispatches todos must be the coordinator: %+v", lead)
	}
	subKey := SubagentKey("report:claude-code:sess-1", prompt)
	sub := worker(t, d, subKey)
	if sub.ParentKey != lead.Key || sub.Status != models.WorkerStatusWorking || sub.TodoID == nil || *sub.TodoID != todo.ID ||
		sub.Task != todo.Title || sub.Role != "Anak buah · general-purpose" || sub.Model != "claude-opus-5-5" {
		t.Fatalf("sub agent must work on its todo under the coordinator: %+v", sub)
	}

	// Transkrip sub agent sendiri memperkaya aksinya.
	file := newClaudeSubFileParser("/h/.claude/projects/-home-dev-app/sess-1/subagents/agent-abc.jsonl")
	ftf := &trackedFile{parser: file}
	feed(file,
		jsonLine(t, map[string]interface{}{"type": "user", "isSidechain": true, "timestamp": ts(3), "message": map[string]interface{}{"content": prompt}}),
		`{"type":"assistant","isSidechain":true,"timestamp":"`+ts(4)+`","message":{"id":"s1","content":[{"type":"tool_use","id":"x","name":"Edit","input":{"file_path":"/home/dev/app/a.vue"}}]}}`,
	)
	applyTracked(ftf, time.Now())
	if sub = worker(t, d, subKey); sub.Action != "Mengedit a.vue" || sub.Status != models.WorkerStatusWorking {
		t.Fatalf("sub agent transcript must update its action: %+v", sub)
	}

	// Hasil kembali ke koordinator → sub agent ditutup.
	feed(parent, `{"type":"user","timestamp":"`+ts(5)+`","toolUseResult":{"agentId":"abc","status":"completed"},"message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"HASIL: selesai"}]}}`)
	applyTracked(tf, time.Now())
	sub = worker(t, d, subKey)
	if sub.Status != models.WorkerStatusDone || sub.FinishedAt == nil || sub.TodoID != nil || sub.Task != todoDonePrefix+todo.Title || sub.TasksDone != 1 {
		t.Fatalf("sub agent must be closed after reporting: %+v", sub)
	}

	// Baris terakhir transkrip sub agent tidak boleh membuka kembali sub agent yang sudah ditutup,
	// dan transkrip sub agent tanpa pasangan di koordinator tidak membuat orang baru.
	feed(file, `{"type":"assistant","isSidechain":true,"timestamp":"`+ts(6)+`","message":{"id":"s2","stop_reason":"end_turn","content":[{"type":"text","text":"HASIL: selesai"}]}}`)
	applyTracked(ftf, time.Now())
	if sub = worker(t, d, subKey); sub.Status != models.WorkerStatusDone {
		t.Fatalf("closed sub agent revived: %+v", sub)
	}
	stray := newClaudeSubFileParser("/h/.claude/projects/-home-dev-app/sess-1/subagents/agent-zzz.jsonl")
	feed(stray, jsonLine(t, map[string]interface{}{"type": "user", "isSidechain": true, "timestamp": ts(7), "message": map[string]interface{}{"content": "prompt lain"}}))
	applyTracked(&trackedFile{parser: stray}, time.Now())
	var n int64
	d.Model(&models.AgentWorker{}).Where("parent_key = ?", lead.Key).Count(&n)
	if n != 1 {
		t.Fatalf("stray sub agent transcript created a worker (%d sub agents)", n)
	}
}

// Sub agent yang belum melapor ikut ditutup saat giliran koordinator berakhir atau dihentikan.
func TestSubagentsClosedWhenCoordinatorTurnEnds(t *testing.T) {
	p := newClaudeParser("/h/.claude/projects/-x/sess-2.jsonl").(*claudeParser)
	p.feed([]byte(`{"type":"assistant","timestamp":"2026-10-26T01:00:00Z","message":{"id":"m1","content":[{"type":"tool_use","id":"u1","name":"Task","input":{"description":"A","prompt":"Tugas: A"}}]}}`))
	if subs := p.subagents(); len(subs) != 1 || subs[0].Status != models.WorkerStatusWorking || subs[0].Task != "A" {
		t.Fatalf("started: %+v", subs)
	}
	p.feed([]byte(`{"type":"user","timestamp":"2026-10-26T01:00:09Z","message":{"content":[{"type":"text","text":"[Request interrupted by user]"}]}}`))
	if subs := p.subagents(); subs[0].Status != models.WorkerStatusDone {
		t.Fatalf("interrupted coordinator must close its sub agents: %+v", subs)
	}
}
