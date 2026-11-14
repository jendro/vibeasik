package insight

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestResolveProjectRef(t *testing.T) {
	db := testDB(t)
	db.Create(&models.Project{Name: "Bimtek APSI", Slug: "bimtek-apsi", RootPath: "/a"})
	db.Create(&models.Project{Name: "Toko Kopi", Slug: "toko-kopi", RootPath: "/b"})
	db.Create(&models.Project{Name: "Toko Buku", Slug: "toko-buku", RootPath: "/c"})

	for _, ref := range []string{"bimtekapsi", "Bimtek APSI", "bimtek-apsi", "bimtek"} {
		if id, err := ResolveProjectRef(db, ref); err != nil || id != 1 {
			t.Errorf("%q → %d, %v", ref, id, err)
		}
	}
	if id, _ := ResolveProjectRef(db, "3"); id != 3 {
		t.Errorf("ID angka harus dipakai apa adanya, dapat %d", id)
	}
	if _, err := ResolveProjectRef(db, "toko"); err == nil || !strings.Contains(err.Error(), "beberapa proyek") {
		t.Errorf("nama ambigu harus ditolak: %v", err)
	}
	if _, err := ResolveProjectRef(db, "tidakada"); err == nil {
		t.Error("nama tak dikenal harus galat")
	}
}

func TestBuildFindingsTodoPrompt(t *testing.T) {
	db := testDB(t)
	proj := models.Project{Name: "Demo", Slug: "demo", RootPath: "/x"}
	db.Create(&proj)
	fs, _ := json.Marshal([]models.Finding{
		{ID: "1", RuleID: "COST-POLYNOMIAL-GROWTH", Severity: "CRITICAL", Status: "open"},
		{ID: "2", RuleID: "QUERY-NPLUS1-CHAIN", Severity: "WARNING", Status: "open"},
		{ID: "3", RuleID: "COST-POLYNOMIAL-GROWTH", Severity: "WARNING", Status: models.FindingStatusResolved},
		{ID: "4", RuleID: "RUNTIME-PAGE-SLOW", Severity: "WARNING", Status: "open"},
		{ID: "5", RuleID: "GIT-UNCOMMITTED", Severity: "INFO", Status: "open"},
	})
	db.Create(&models.Diagnostic{ProjectID: proj.ID, FindingsJSON: string(fs), CreatedAt: time.Now()})

	cost, err := BuildFindingsTodoPrompt(db, proj.ID, "cost")
	if err != nil {
		t.Fatal(err)
	}
	if cost.Open != 2 || cost.Critical != 1 || cost.Advisory != 1 || cost.Scope.Page != "/process-cost" {
		t.Errorf("biaya proses: %+v", cost)
	}
	for _, want := range []string{`rule_prefix: "COST-,QUERY-NPLUS1"`, "project ID 1", "vibeasik_audit_triage_submit", "get_latest_diagnostic", "1 temuan bersifat perkiraan", "halaman Biaya Proses"} {
		if !strings.Contains(cost.Prompt, want) {
			t.Errorf("prompt harus memuat %q:\n%s", want, cost.Prompt)
		}
	}
	if all, _ := BuildFindingsTodoPrompt(db, proj.ID, ""); all.Open != 4 || all.Scope.Key != FindingScopeAudit || strings.Contains(all.Prompt, "rule_prefix") {
		t.Errorf("audit = semua temuan tanpa saringan: %+v", all)
	}
	if rt, _ := BuildFindingsTodoPrompt(db, proj.ID, "runtime"); rt.Open != 1 {
		t.Errorf("uji runtime: %+v", rt)
	}
}

func TestCostSourceLinesAndBatchTodos(t *testing.T) {
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
	app.Get("/b", b)
}

func load(id int) int {
	var x int
	db.First(&x, id)
	return x
}

func b(c *fiber.Ctx) error {
	var ids []int
	db.Find(&ids)
	for _, id := range ids {
		load(id)
	}
	return nil
}
`)
	proj := models.Project{Name: "demo", Slug: "demo", RootPath: root}
	db.Create(&proj)

	win, err := CostSourceLines(db, proj.ID, "main.go", 15, 19)
	if err != nil || win.From != 15 || len(win.Lines) != 5 || win.Lines[0] != "func load(id int) int {" || win.Total < 25 {
		t.Fatalf("potongan berkas: %+v err=%v", win, err)
	}
	for _, bad := range []string{"../etc/passwd", "/etc/passwd", "nope.go"} {
		if _, err := CostSourceLines(db, proj.ID, bad, 1, 2); !errors.Is(err, ErrSourceOutside) {
			t.Errorf("%q harus ditolak: %v", bad, err)
		}
	}

	page, err := ProcessCost(db, proj.ID, CostQuery{}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var callee *string
	for _, r := range page.Routes {
		for _, e := range r.Evidence {
			if e.Callee != nil {
				name := e.Callee.Name
				callee = &name
				if e.Callee.File != "main.go" || e.Callee.Line != 15 {
					t.Errorf("fungsi terpanggil harus menunjuk deklarasinya: %+v", e.Callee)
				}
			}
		}
	}
	if callee == nil || *callee != "load" {
		t.Fatalf("bukti pemanggilan di dalam loop harus menunjuk fungsi load")
	}

	// Todo gabungan lintas berkas (kunci "*go", mis. validasi false positive) tidak boleh menelan sumber biaya.
	wild := models.Todo{ProjectID: &proj.ID, Title: "Validasi false positive N+1", Status: models.TodoStatusInProgress, AuditKey: "QUERY-NPLUS1|*go"}
	db.Create(&wild)

	batch, err := CreateAllCostTodos(db, proj.ID)
	if err != nil || batch.Created != 1 || batch.Merged != 0 || batch.Files != 1 || len(batch.Todos) != 1 || batch.Todos[0].ID == wild.ID {
		t.Fatalf("jadikan semua todo: %+v err=%v", batch, err)
	}
	again, _ := CreateAllCostTodos(db, proj.ID)
	if again.Created != 0 || again.Merged != 0 {
		t.Errorf("klik kedua tidak boleh membuat todo baru: %+v", again)
	}
}
