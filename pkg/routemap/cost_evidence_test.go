package routemap

import (
	"strings"
	"testing"
)

func evidenceAt(ev []CostEvidence, kind, codePart string) (CostEvidence, bool) {
	for _, e := range ev {
		if e.Kind == kind && strings.Contains(e.Code, codePart) {
			return e, true
		}
	}
	return CostEvidence{}, false
}

// Setiap rumus yang tumbuh atau menyentuh database harus bisa ditunjuk barisnya: loop atas data, query, dan
// pemanggilan berbiaya — dengan kedalaman loop yang sama dengan pangkat pada rumus.
func TestGoCostEvidencePointsAtLoopsAndQueries(t *testing.T) {
	root := writeTree(t, map[string]string{"go.mod": "module demo\n\ngo 1.22\n", "main.go": costFixture})
	rep := Costs(Analyze(root))

	pairs := routeCost(t, rep, "/pairs")
	if pairs.EvidenceFile != "main.go" {
		t.Fatalf("/pairs: berkas bukti = %q, mau main.go", pairs.EvidenceFile)
	}
	outer, ok1 := evidenceAt(pairs.Evidence, "loop", "for i := range us")
	inner, ok2 := evidenceAt(pairs.Evidence, "loop", "for j := range us")
	if !ok1 || !ok2 || outer.Depth != 1 || inner.Depth != 2 || !strings.Contains(inner.Note, "n²") || outer.Line >= inner.Line {
		t.Fatalf("/pairs harus menunjuk kedua loop bersarang (tingkat 1 lalu 2): %+v", pairs.Evidence)
	}
	if q, ok := evidenceAt(pairs.Evidence, "query", "db.Find(&us)"); !ok || q.Depth != 0 {
		t.Errorf("/pairs: query di luar loop harus tercatat tanpa kedalaman: %+v", pairs.Evidence)
	}

	nplus := routeCost(t, rep, "/nplus")
	q, ok := evidenceAt(nplus.Evidence, "query", "Find(&us[i].Orders)")
	if !ok || q.Depth != 1 || !strings.Contains(q.Note, "N+1") {
		t.Fatalf("/nplus harus menunjuk query di dalam loop sebagai N+1: %+v", nplus.Evidence)
	}
	if _, ok := evidenceAt(nplus.Evidence, "loop", "for i := range us"); !ok {
		t.Errorf("/nplus: loop pembungkus query harus ikut jadi bukti: %+v", nplus.Evidence)
	}

	if s, ok := evidenceAt(routeCost(t, rep, "/sorted").Evidence, "call", "sort.Slice"); !ok || !strings.Contains(s.Note, "n log n") {
		t.Errorf("/sorted harus menunjuk pengurutan: %+v", routeCost(t, rep, "/sorted").Evidence)
	}
	// Loop batas tetap tidak menaikkan pertumbuhan: tidak ada bukti, dan rumusnya memang konstan.
	if ev := routeCost(t, rep, "/fixed").Evidence; len(ev) != 0 {
		t.Errorf("/fixed (O(1) tanpa query) tidak boleh punya bukti pertumbuhan: %+v", ev)
	}
	if one := routeCost(t, rep, "/one"); len(one.Evidence) != 1 || one.Evidence[0].Kind != "query" {
		t.Errorf("/one harus menunjuk satu query-nya: %+v", one.Evidence)
	}
	for _, e := range pairs.Evidence {
		if e.Line <= 0 || e.Code == "" || e.Note == "" {
			t.Errorf("bukti tanpa baris/kode/penjelasan: %+v", e)
		}
	}
}

// Biaya yang datang dari fungsi terpanggil ditunjuk di baris pemanggilannya, dengan nama fungsi sumbernya.
func TestGoCostEvidenceNamesCostlyCallee(t *testing.T) {
	src := `package main

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Item struct{ ID uint }

var db *gorm.DB

func main() {
	app := fiber.New()
	app.Get("/items", listItems)
}

func listItems(c *fiber.Ctx) error {
	var items []Item
	db.Find(&items)
	for i := range items {
		items[i] = loadOne(items[i].ID)
	}
	return c.JSON(items)
}

func loadOne(id uint) Item {
	var it Item
	db.First(&it, id)
	return it
}
`
	root := writeTree(t, map[string]string{"go.mod": "module demo\n\ngo 1.22\n", "main.go": src})
	rc := routeCost(t, Costs(Analyze(root)), "/items")
	e, ok := evidenceAt(rc.Evidence, "query", "loadOne(items[i].ID)")
	if !rc.NPlusOne || !ok || e.Depth != 1 || !strings.Contains(e.Note, "loadOne") || !strings.Contains(e.Note, "N+1") {
		t.Fatalf("pemanggilan loadOne di dalam loop harus jadi bukti N+1 dengan nama fungsinya: %+v", rc.Evidence)
	}
}

func TestPHPCostEvidencePointsAtLoopsAndQueries(t *testing.T) {
	rep, _ := PHPCosts(writeTree(t, laravelFixture))
	pairs := phpRouteCost(t, rep, "GET", "/admin/pairs")
	loops := 0
	for _, e := range pairs.Evidence {
		if e.Kind == "loop" {
			loops++
		}
		if e.Line <= 0 || e.Code == "" {
			t.Errorf("bukti PHP tanpa baris/kode: %+v", e)
		}
	}
	if pairs.EvidenceFile == "" || loops < 2 {
		t.Fatalf("/admin/pairs (O(n²)) harus menunjuk dua loop bersarang di %q: %+v", pairs.EvidenceFile, pairs.Evidence)
	}
	nplus := phpRouteCost(t, rep, "GET", "/nplus")
	found := false
	for _, e := range nplus.Evidence {
		found = found || (e.Kind == "query" && e.Depth >= 1 && strings.Contains(e.Note, "N+1"))
	}
	if !found {
		t.Fatalf("/nplus harus menunjuk query di dalam loop: %+v", nplus.Evidence)
	}
}
