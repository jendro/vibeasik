package standard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/runner"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Project{}, &models.Component{}, &models.Todo{}); err != nil {
		t.Fatal(err)
	}
	orig := runner.RegistryPath
	runner.RegistryPath = func() string { return "" }
	t.Cleanup(func() { runner.RegistryPath = orig; models.ProjectChanged = nil })
	return db
}

func openTodos(db *gorm.DB, title string) []models.Todo {
	var ts []models.Todo
	db.Where("title = ? AND status = ?", title, models.TodoStatusPending).Find(&ts)
	return ts
}

func TestEnsureLifecycle(t *testing.T) {
	db := setupDB(t)
	root := t.TempDir()
	be := filepath.Join(root, "backend")
	os.MkdirAll(be, 0o755)
	os.WriteFile(filepath.Join(be, ".env"), []byte("PORT=8080\nDB_PASSWORD=rahasia\n"), 0o644)

	p := models.Project{Name: "Toko", Slug: "toko", RootPath: root, Status: models.StatusActive,
		Components: []models.Component{{Name: "api", Role: models.RoleBackend, RelativePath: "backend", AbsolutePath: be, Port: 8080}}}
	db.Create(&p)

	res, err := Ensure(db, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	var comp models.Component
	db.First(&comp, p.Components[0].ID)
	if comp.RunPort != 4105 || res.PortsChanged != 1 {
		t.Fatalf("backend must get slot 0 offset +5 (4105), got %d", comp.RunPort)
	}
	// Di luar git → engine membuat env lokal & server; port .env ikut blok
	local, _ := os.ReadFile(filepath.Join(be, ".env.local"))
	server, _ := os.ReadFile(filepath.Join(be, ".env.server"))
	if !strings.Contains(string(local), "PORT=4105") || !strings.Contains(string(server), "DB_PASSWORD=\n") {
		t.Fatalf("env files wrong:\nlocal=%s\nserver=%s", local, server)
	}
	title := StandardTitle("Toko")
	todos := openTodos(db, title)
	if len(todos) != 1 || todos[0].Priority != models.TodoPriorityHigh || !strings.Contains(todos[0].Description, "PORT=4105") ||
		!strings.Contains(todos[0].Description, "`dev.sh`") || !strings.Contains(todos[0].Description, "`deploy.sh`") {
		t.Fatalf("standard todo wrong: %+v", todos)
	}

	// Idempoten: tidak ada todo ganda
	if _, err := Ensure(db, p.ID); err != nil {
		t.Fatal(err)
	}
	if n := len(openTodos(db, title)); n != 1 {
		t.Fatalf("want 1 open standard todo, got %d", n)
	}

	// Standar lengkap → todo gabungan pending ditutup otomatis
	devsh := "#!/usr/bin/env bash\ntrap 'kill 0' EXIT\n(cd backend && PORT=4105 go run .) &\nwait\n"
	deploysh := "#!/usr/bin/env bash\nT=${1:-server}\ntrap 'cp backend/.env.local backend/.env' EXIT\ncp \"backend/.env.$T\" backend/.env\n"
	os.WriteFile(filepath.Join(root, "dev.sh"), []byte(devsh), 0o755)
	os.WriteFile(filepath.Join(root, "deploy.sh"), []byte(deploysh), 0o755)
	os.WriteFile(filepath.Join(be, ".env.example"), []byte("PORT=\nDB_PASSWORD=\n"), 0o644)
	res, _ = Ensure(db, p.ID)
	if len(res.Closed) != 1 || len(openTodos(db, title)) != 0 {
		t.Fatalf("standard todo must be closed when complete: %+v", res)
	}
}

func TestUpsertRespectsRejectedAndRecentDone(t *testing.T) {
	db := setupDB(t)
	pid := uint(9)
	base := models.Todo{ProjectID: &pid, Title: "X", Status: models.TodoStatusPending}

	rej := base
	rej.Status = models.TodoStatusRejected
	db.Create(&rej)
	if got, _ := upsert(db, base); got != nil {
		t.Fatal("rejected todo must not be recreated")
	}
	db.Delete(&rej)

	now := time.Now()
	done := base
	done.Status, done.CompletedAt = models.TodoStatusCompleted, &now
	db.Create(&done)
	if got, _ := upsert(db, base); got != nil {
		t.Fatal("recently completed todo must not be recreated (agent result may still be merging)")
	}
	withNew := base
	withNew.Description = "1. **dev-mobile.sh Belum Ada di Root Proyek** — ..."
	got, _ := upsert(db, withNew)
	if got == nil {
		t.Fatal("a new standard item must not be held back by a recent completion that did not cover it")
	}
	db.Delete(got)
	db.Model(&done).Update("description", withNew.Description)
	if got, _ := upsert(db, withNew); got != nil {
		t.Fatal("recent completion covering the same items must still hold the todo back")
	}
	old := now.Add(-48 * time.Hour)
	db.Model(&done).Update("completed_at", &old)
	if got, _ := upsert(db, base); got == nil || got.Status != models.TodoStatusPending {
		t.Fatal("still-missing standard after an old completion must create a new pending todo")
	}
}

func TestHookSchedulesEnsure(t *testing.T) {
	db := setupDB(t)
	debounce = 10 * time.Millisecond
	models.ProjectChanged = func(id uint) { Schedule(db, id) }
	p := models.Project{Name: "Baru", Slug: "baru", Status: models.StatusActive,
		Components: []models.Component{{Name: "web", Role: models.RoleFrontendClient}}}
	db.Create(&p)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var c models.Component
		db.First(&c, p.Components[0].ID)
		if c.RunPort == 4100 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("creating a project must allocate its port block automatically")
}
