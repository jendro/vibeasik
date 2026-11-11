package diagnostics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

const costFindingSrc = `package main

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

type Item struct{ ID uint }

func main() {
	app := fiber.New()
	app.Get("/items", listItems)
	app.Get("/inline", inline)
}

func listItems(c *fiber.Ctx) error { return c.JSON(enrich()) }

func enrich() []Item {
	var items []Item
	db.Find(&items)
	for i := range items {
		items[i] = loadOne(items[i].ID)
		for j := range items {
			_ = j
		}
	}
	return items
}

func loadOne(id uint) Item {
	var it Item
	db.First(&it, id)
	return it
}

func inline(c *fiber.Ctx) error {
	var items []Item
	db.Find(&items)
	for i := range items {
		db.First(&items[i], items[i].ID)
	}
	return c.JSON(items)
}
`

func TestCostFindingsReportSourceFunction(t *testing.T) {
	dir := t.TempDir()
	for name, src := range map[string]string{"go.mod": "module demo\n\ngo 1.22\n", "main.go": costFindingSrc} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Aturan per baris sudah menangkap query yang tertulis langsung di dalam loop fungsi inline.
	lineRule := []models.Finding{{RuleID: "QUERY-NPLUS1", TargetFile: "be/main.go", Line: 41}}
	got := refineRouterFindings(lineRule, dir, "be", true)

	byRule := map[string][]models.Finding{}
	for _, f := range got {
		byRule[f.RuleID] = append(byRule[f.RuleID], f)
	}
	chain := byRule[ruleNPlusOneChain]
	if len(chain) != 1 {
		t.Fatalf("temuan N+1 berantai: %d, mau 1 (fungsi inline sudah dilaporkan aturan per baris): %+v", len(chain), chain)
	}
	f := chain[0]
	if f.TargetFile != "be/main.go" || f.Line != 20 || f.Severity != "WARNING" || f.Category != "query" {
		t.Errorf("lokasi/keparahan: %+v", f)
	}
	for _, want := range []string{"main.enrich", "main.loadOne", "Q(n) = n + 1", "GET /items"} {
		if !strings.Contains(f.Description, want) {
			t.Errorf("deskripsi tidak memuat %q: %s", want, f.Description)
		}
	}
	growth := byRule[rulePolynomialGrowth]
	if len(growth) != 1 || growth[0].Severity != "INFO" || !strings.Contains(growth[0].Title, "O(n²)") {
		t.Fatalf("temuan pertumbuhan: %+v", growth)
	}
	if CalculateFindingPenalty(f) != 3 || CalculateFindingPenalty(growth[0]) != 1 {
		t.Errorf("bobot skor: N+1 berantai %d (mau 3), pertumbuhan %d (mau 1)", CalculateFindingPenalty(f), CalculateFindingPenalty(growth[0]))
	}
}
