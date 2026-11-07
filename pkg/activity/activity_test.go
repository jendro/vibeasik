package activity

import (
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := d.AutoMigrate(&models.AgentWorker{}, &models.Todo{}, &models.Project{}, &models.Component{}, &models.LLMConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	Init(d)
	t.Cleanup(func() { Init(nil) })
	return d
}

func worker(t *testing.T, d *gorm.DB, key string) models.AgentWorker {
	t.Helper()
	var w models.AgentWorker
	if err := d.Where("worker_key = ?", key).First(&w).Error; err != nil {
		t.Fatalf("worker %s not found: %v", key, err)
	}
	return w
}

func TestMCPAgentLifecycle(t *testing.T) {
	d := setupDB(t)
	project := models.Project{Name: "Toko Online", RootPath: "/home/dev/toko"}
	d.Create(&project)
	todo := models.Todo{Title: "Tambah halaman checkout", ProjectID: &project.ID, ProjectName: project.Name}
	d.Create(&todo)

	c := Caller{Key: "mcp-stdio:42", Transport: "stdio"}
	ClientConnected(c, "claude-code", "2.1.0")
	if w := worker(t, d, c.Key); w.Name != "Claude Code" || w.Status != models.WorkerStatusIdle {
		t.Fatalf("after initialize: name=%q status=%q", w.Name, w.Status)
	}

	ToolCalled(c, "update_todo", map[string]interface{}{"todo_id": float64(todo.ID), "status": "in_progress"})
	w := worker(t, d, c.Key)
	if w.Status != models.WorkerStatusWorking || w.Task != todo.Title || w.TodoID == nil || *w.TodoID != todo.ID {
		t.Fatalf("in_progress not linked: %+v", w)
	}
	if w.ProjectName != "Toko Online" || w.ToolCalls != 1 {
		t.Fatalf("project/toolcalls: %+v", w)
	}

	ToolCalled(c, "update_todo", map[string]interface{}{"todo_id": float64(todo.ID), "status": "completed"})
	ToolFinished(c, "update_todo", errors.New("boom"))
	w = worker(t, d, c.Key)
	if w.TasksDone != 1 || w.TodoID != nil || w.LastError == "" {
		t.Fatalf("completed not applied: %+v", w)
	}

	ClientDisconnected(c)
	if w := worker(t, d, c.Key); w.Status != models.WorkerStatusLeft || w.FinishedAt == nil {
		t.Fatalf("disconnect: %+v", w)
	}
}

func TestReportActivityWithoutCaller(t *testing.T) {
	d := setupDB(t)
	project := models.Project{Name: "vibeasik", RootPath: "/home/dev/vibeasik"}
	d.Create(&project)

	d.Create(&models.Component{ProjectID: project.ID, Name: "FE", AbsolutePath: "/home/dev/vibeasik/studio-frontend"})
	w := ReportActivity(Caller{}, Report{AgentName: "claude-code", SessionID: "abc", Task: "Refactor navbar", ProjectPath: "/home/dev/vibeasik/studio-frontend/app"})
	if w == nil || w.Name != "Claude Code" || w.ProjectID == nil || *w.ProjectID != project.ID {
		t.Fatalf("report: %+v", w)
	}
	again := ReportActivity(Caller{}, Report{AgentName: "claude-code", SessionID: "abc", Status: "done"})
	if again.ID != w.ID || again.TasksDone != 1 || again.Status != models.WorkerStatusIdle || again.Task != "Refactor navbar" {
		t.Fatalf("same session must update same worker: %+v", again)
	}
	// Root proyek yang terlalu umum (mis. ~/Documents) tidak boleh menautkan repo lain di dalamnya.
	d.Create(&models.Project{Name: "Dokumen", RootPath: "/home/dev"})
	for _, p := range []string{"/home/dev/lain", "/home/dev/lain/web"} {
		if got := projectIDForPath(p); got != 0 {
			t.Fatalf("broad root must not match %s, got %d", p, got)
		}
	}
	if got := projectIDForPath("/home/dev/vibeasik"); got != project.ID {
		t.Fatalf("exact root must match, got %d", got)
	}
	if ReportActivity(Caller{}, Report{}) != nil {
		t.Fatal("report without name/session must be ignored")
	}
}

func TestReportToolUsesCallerIdentity(t *testing.T) {
	d := setupDB(t)
	c := Caller{Key: "mcp-http:s1", Transport: "http"}
	ToolCalled(c, ReportToolName, map[string]interface{}{"agent_name": "Antigravity", "task": "Menulis test", "model": "gemini-3"})
	w := worker(t, d, c.Key)
	if w.Name != "Antigravity" || w.Task != "Menulis test" || w.Model != "gemini-3" || w.Transport != "http" {
		t.Fatalf("report tool: %+v", w)
	}
}

func TestLLMCallShowsModelAlias(t *testing.T) {
	d := setupDB(t)
	cfg := models.LLMConfig{Provider: models.ProviderStandardLLM, Name: "Proxy", URL: "https://proxy.example.com/v1"}
	cfg.SetModelAlias("vendor/claude-x-20260101", "Claude X")
	d.Create(&cfg)

	BeginLLM("https://proxy.example.com/v1/chat/completions", "vendor/claude-x-20260101", "")(nil)
	BeginLLM("https://other.example.com/v1", "gpt-5", "")(nil)
	var ws []models.AgentWorker
	d.Where("kind = ?", models.WorkerKindLLMCall).Order("id").Find(&ws)
	if len(ws) != 2 || ws[0].Name != "Claude X" || ws[0].Model != "Claude X" || ws[0].ClientName != "" || ws[1].Name != "gpt-5" {
		t.Fatalf("llm worker must show model alias without endpoint: %+v", ws)
	}
}

func TestLLMCallVisibility(t *testing.T) {
	d := setupDB(t)
	SetLLMContextProvider(func() (string, string) { return "Fast Scan BE", "Toko" })
	t.Cleanup(func() { SetLLMContextProvider(nil) })

	done := BeginLLM("https://api.openai.com/v1", "gpt-5", "Anda adalah Principal Software Architect, Systems Integrator.\nDetail")
	var w models.AgentWorker
	d.Where("kind = ?", models.WorkerKindLLMCall).First(&w)
	if w.Name != "gpt-5" || w.ClientName != "" || w.Role != "Principal Software Architect" || w.Task != "Fast Scan BE" || w.ProjectName != "Toko" {
		t.Fatalf("llm worker: %+v", w)
	}
	now := time.Now()
	if visible, _ := VisibleWorkers(now); len(visible) != 1 {
		t.Fatalf("working llm must be visible, got %d", len(visible))
	}
	done(nil)
	if visible, _ := VisibleWorkers(now.Add(FinishedLinger / 2)); len(visible) != 1 || visible[0].Status != models.WorkerStatusDone {
		t.Fatalf("finished llm must linger: %+v", visible)
	}
	if visible, _ := VisibleWorkers(now.Add(FinishedLinger + 5*time.Second)); len(visible) != 0 {
		t.Fatalf("finished llm must leave after linger, got %d", len(visible))
	}

	failed := BeginLLM("https://api.openai.com/v1", "gpt-5", "Detail")
	failed(errors.New("rate limit"))
	if visible, _ := VisibleWorkers(now.Add(FinishedLinger + 5*time.Second)); len(visible) != 1 || visible[0].Status != models.WorkerStatusError {
		t.Fatalf("failed llm must stay for inspection: %+v", visible)
	}
	if visible, _ := VisibleWorkers(now.Add(ErrorLinger + 5*time.Second)); len(visible) != 0 {
		t.Fatalf("failed llm must leave after error linger, got %d", len(visible))
	}
}

func TestNamesAndRoles(t *testing.T) {
	cases := map[string]string{"claude-code": "Claude Code", "agy": "Antigravity", "Cursor IDE": "Cursor", "my-bot": "My Bot", "": "Agent MCP", "claude-ai": "Claude Desktop"}
	for in, want := range cases {
		if got := FriendlyAgentName(in); got != want {
			t.Errorf("FriendlyAgentName(%q)=%q want %q", in, got, want)
		}
	}
	roles := map[string]string{
		"You are an elite fullstack software architect, security auditor.": "Elite fullstack software architect",
		"Anda adalah Senior Enterprise Software Architect & API Designer.": "Senior Enterprise Software Architect & API Designer",
		"": "",
	}
	for in, want := range roles {
		if got := LLMRole(in); got != want {
			t.Errorf("LLMRole(%q)=%q want %q", in, got, want)
		}
	}
	if LLMProviderName("http://localhost:11434/v1") != "Ollama" || LLMProviderName("https://llm.example.com/v1") != "llm.example.com" {
		t.Error("LLMProviderName mapping wrong")
	}
}

func TestNoopWithoutInit(t *testing.T) {
	Init(nil)
	BeginLLM("x", "y", "z")(nil)
	BeginLocalAgent("agy", "p", "")(true, "")
	ToolCalled(Caller{Key: "k"}, "list_projects", nil)
	if ReportActivity(Caller{}, Report{AgentName: "a"}) != nil {
		t.Fatal("expected nil without init")
	}
	if w, err := VisibleWorkers(time.Now()); err != nil || len(w) != 0 {
		t.Fatal("expected empty list without init")
	}
}

func TestFinishedCodingAgentGoesHome(t *testing.T) {
	setupDB(t)
	now := time.Now()
	obs := Observation{Key: "report:claude-code:s1", Name: "Claude Code", Status: models.WorkerStatusIdle, Action: actionWaiting, LastEvent: now}
	ApplyObservation(obs, 1, false, false)
	if visible, _ := VisibleWorkers(now.Add(IdleHomeAfter / 2)); len(visible) != 1 {
		t.Fatalf("finished agent must linger briefly, got %d", len(visible))
	}
	if visible, _ := VisibleWorkers(now.Add(IdleHomeAfter + 5*time.Second)); len(visible) != 0 {
		t.Fatalf("finished agent must go home, got %d", len(visible))
	}

	// Koordinator yang anak buahnya masih bekerja tetap di kantor.
	ApplyObservation(Observation{Key: "report:claude-code:s1:sub", Name: "Anak Buah", ParentKey: obs.Key, Status: models.WorkerStatusWorking, LastEvent: now}, 0, false, false)
	if visible, _ := VisibleWorkers(now.Add(IdleHomeAfter + 5*time.Second)); len(visible) != 2 {
		t.Fatalf("supervising coordinator must stay, got %d", len(visible))
	}

	// Chat baru di sesi yang sama: datang lagi.
	later := now.Add(time.Minute)
	obs.Status, obs.Action, obs.LastEvent = models.WorkerStatusWorking, actionReadingPrompt, later
	ApplyObservation(obs, 0, false, false)
	if visible, _ := VisibleWorkers(later.Add(5 * time.Second)); len(visible) != 2 {
		t.Fatalf("new chat must bring the agent back, got %d", len(visible))
	}
}
