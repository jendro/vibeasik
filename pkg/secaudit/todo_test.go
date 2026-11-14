package secaudit

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestCreateTodosFromFailedChecks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "t.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Project{}, &models.Todo{}); err != nil {
		t.Fatal(err)
	}
	proj := models.Project{Name: "Demo", Slug: "demo", RootPath: "/x"}
	db.Create(&proj)
	rep := &Report{ProjectID: proj.ID, Target: "http://127.0.0.1:4101", Checks: []Check{
		{ID: "rate-limit-login", Category: CatRateLimit, Title: "Rate limit login", Status: StatusFail, Severity: SevWarning, Source: "live",
			Suggestion: "Pasang limiter di rute login", Locations: []Location{{File: "routes.go", Line: 12, Snippet: "app.Post(\"/login\", login)"}}},
		{ID: "hsts", Category: CatTransport, Title: "HSTS", Status: StatusWarn, Severity: SevInfo, Source: "live"},
		{ID: "sqli", Category: CatInjection, Title: "Query berparameter", Status: StatusPass, Severity: SevCritical},
		{ID: "cors", Category: CatCORS, Title: "CORS", Status: StatusSkipped, Severity: SevWarning},
	}}

	out, err := CreateTodos(db, rep)
	if err != nil || out.Created != 2 || out.Existed != 0 {
		t.Fatalf("hanya fail & warn yang jadi todo: %+v err=%v", out, err)
	}
	login := out.Todos[0]
	if login.Priority != models.TodoPriorityHigh || login.Category != models.CategorySecurity || login.AuditKey != "SECTEST|1|rate-limit-login" ||
		!strings.Contains(login.Description, "`routes.go:12`") || !strings.Contains(login.Description, "Pasang limiter") {
		t.Errorf("todo rate limit: %+v", login)
	}
	if out.Todos[1].Priority != models.TodoPriorityLow {
		t.Errorf("warn berseverity info = prioritas rendah: %s", out.Todos[1].Priority)
	}
	again, _ := CreateTodos(db, rep)
	if again.Created != 0 || again.Existed != 2 {
		t.Errorf("klik kedua tidak boleh menggandakan todo: %+v", again)
	}
}
