package insight

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Project{}, &models.Component{}, &models.Todo{}, &models.Diagnostic{}, &models.ProcessCostSnapshot{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestBuildOwnerRecap(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, 3, 10, 15, 0, 0, 0, time.UTC)
	proj := models.Project{Name: "demo", HealthScore: 82}
	db.Create(&proj)
	pid := proj.ID
	at := func(daysAgo int) time.Time { return now.AddDate(0, 0, -daysAgo) }
	ptr := func(t time.Time) *time.Time { return &t }
	score := 90
	todos := []models.Todo{
		{ProjectID: &pid, Title: "Perbaiki crash saat login", Status: "completed", Category: "backend", CreatedAt: at(5), CompletedAt: ptr(at(2)), Score: &score},
		{ProjectID: &pid, Title: "Hilangkan N+1 query daftar pesanan", Status: "completed", Category: "performance", CreatedAt: at(4), CompletedAt: ptr(at(1))},
		{ProjectID: &pid, Title: "Halaman laporan", Status: "pending", Category: "feature", Priority: "high", CreatedAt: at(3)},
		{ProjectID: &pid, Title: "Tambah indeks tabel log", Status: "in_progress", Category: "backend", CreatedAt: at(40)},
		{ProjectID: &pid, Title: "Saran ditolak", Status: "rejected", Category: "general", CreatedAt: at(1)},
		{ProjectID: &pid, Title: "Selesai lama", Status: "completed", Category: "feature", CreatedAt: at(90), CompletedAt: ptr(at(80))},
	}
	for i := range todos {
		if err := db.Create(&todos[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	resolved := at(1)
	findings, _ := json.Marshal([]models.Finding{
		{Title: "a", Severity: "CRITICAL", Status: "open"},
		{Title: "b", Severity: "WARNING", Status: "resolved", ResolvedAt: &resolved},
		{Title: "c", Severity: "INFO", Status: "ignored"},
	})
	db.Create(&models.Diagnostic{ProjectID: pid, Score: 60, CreatedAt: at(10), FindingsJSON: "[]"})
	db.Create(&models.Diagnostic{ProjectID: pid, Score: 82, CreatedAt: at(1), FindingsJSON: string(findings)})

	r, err := BuildOwnerRecap(db, pid, 30, now)
	if err != nil {
		t.Fatal(err)
	}
	if r.Todos.Done != 3 || r.Todos.Open != 2 || r.Todos.Total != 5 || r.Todos.Percent != 60 {
		t.Errorf("todo: %+v", r.Todos)
	}
	if r.Todos.DoneInPeriod != 2 || r.Todos.InProgress != 1 || r.Todos.OpenHighPriority != 1 || r.Todos.AvgScore != 90 {
		t.Errorf("todo periode: %+v", r.Todos)
	}
	if r.Bugs.Done != 1 || r.Bugs.DoneInPeriod != 1 || len(r.Bugs.Latest) != 1 {
		t.Errorf("bug: %+v", r.Bugs)
	}
	if r.Algorithm.Done != 1 || r.Algorithm.Open != 1 {
		t.Errorf("algoritma: %+v", r.Algorithm)
	}
	if r.Findings.Open != 1 || r.Findings.OpenCritical != 1 || r.Findings.Resolved != 1 || r.Findings.Ignored != 1 || r.Findings.ResolvedInPeriod != 1 {
		t.Errorf("temuan: %+v", r.Findings)
	}
	if len(r.Health) != 2 || r.Health[0].Score != 60 || r.Health[1].Score != 82 {
		t.Errorf("kesehatan: %+v", r.Health)
	}
	if len(r.Timeline) != 30 || r.Timeline[29].Open != 2 {
		t.Fatalf("timeline: %d titik, terbuka akhir %d", len(r.Timeline), r.Timeline[29].Open)
	}
	// 5 hari lalu dibuat 1 todo: sebelum itu hanya todo lama (dibuat 40 hari lalu) yang terbuka.
	if got := r.Timeline[29-6].Open; got != 1 {
		t.Errorf("terbuka 6 hari lalu = %d, mau 1", got)
	}
}

func TestProcessCostPagingAndSnapshot(t *testing.T) {
	db := testDB(t)
	root := t.TempDir()
	write := func(name, src string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module demo\n\ngo 1.22\n")
	write("main.go", `package main

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	app := fiber.New()
	app.Get("/a", a)
	app.Get("/b", b)
}

func a(c *fiber.Ctx) error { return c.SendString("ok") }

func b(c *fiber.Ctx) error {
	var ids []int
	db.Find(&ids)
	for _, id := range ids {
		db.First(&id, id)
	}
	return nil
}
`)
	proj := models.Project{Name: "demo", RootPath: root}
	db.Create(&proj)
	now := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)

	page, err := ProcessCost(db, proj.ID, CostQuery{Limit: 1}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !page.Summary.Supported || page.Summary.Routes != 2 || page.Total != 2 || len(page.Routes) != 1 {
		t.Fatalf("halaman: %+v total=%d rute=%d", page.Summary, page.Total, len(page.Routes))
	}
	if page.Routes[0].Path != "/b" || !page.Routes[0].NPlusOne || page.Summary.NPlusOneRoutes != 1 {
		t.Errorf("rute termahal: %+v", page.Routes[0])
	}
	if page.Summary.RouteClasses["constant"] != 1 || page.Summary.RouteClasses["linear"] != 1 {
		t.Errorf("kelas: %+v", page.Summary.RouteClasses)
	}
	// Panggilan kedua di hari yang sama memperbarui potret, tidak menambah baris.
	if _, err := ProcessCost(db, proj.ID, CostQuery{Kind: "funcs", NPlusOne: true}, now); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Model(&models.ProcessCostSnapshot{}).Count(&n)
	if n != 1 || len(page.History) != 1 || page.History[0].NPlusOne != 1 {
		t.Errorf("potret: %d baris, history %+v", n, page.History)
	}
}
