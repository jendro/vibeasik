package insight

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Salinan sistem lama di bawah root proyek (legacy/ yang sudah digantikan) bukan komponen terdaftar, jadi tidak
// boleh ikut dihitung biaya prosesnya — dulu N+1 di Laravel lama menjadi todo baru.
func TestProcessCostOnlyCountsRegisteredComponents(t *testing.T) {
	db := testDB(t)
	root := t.TempDir()
	write := func(rel, src string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	app := func(mod string) string {
		return `package main

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	app := fiber.New()
	app.Get("/` + mod + `", h)
}

func h(c *fiber.Ctx) error {
	var ids []int
	db.Find(&ids)
	for _, id := range ids {
		db.First(&id, id)
	}
	return nil
}
`
	}
	write("backend/go.mod", "module backend\n\ngo 1.22\n")
	write("backend/main.go", app("new"))
	write("legacy/api/go.mod", "module legacyapi\n\ngo 1.22\n")
	write("legacy/api/main.go", app("old"))

	proj := models.Project{Name: "P", RootPath: root}
	db.Create(&proj)
	db.Create(&models.Component{ProjectID: proj.ID, Name: "BE", Role: "backend", AbsolutePath: filepath.Join(root, "backend")})

	page, err := ProcessCost(db, proj.ID, CostQuery{}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range page.Routes {
		if r.Path == "/old" {
			t.Fatalf("rute dari legacy/ ikut dihitung: %+v", r)
		}
	}
	found := false
	for _, r := range page.Routes {
		found = found || r.Path == "/new"
	}
	if !found {
		t.Fatalf("rute komponen terdaftar hilang: %+v", page.Routes)
	}
}
