package codemap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestLayerGuideMapsRealLayersAndTwins(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"backend/go.mod": "module example.com/be\n\ngo 1.22\n",
		"backend/handlers/admin_webinar.go": `package handlers

type webinarRepo struct{}

func (r *webinarRepo) ListWebinarRows() error { return nil }

func AdminListWebinars(c *fiber.Ctx) error { return nil }

func RegisterRoutes(r fiber.Router) {}
`,
		"backend/handlers/webinar.go":                "package handlers\n\nfunc ListWebinars(c *fiber.Ctx) error { return nil }\n\nfunc RegisterRoutes(r fiber.Router) {}\n",
		"backend/handlers/helpers.go":                "package handlers\n\nfunc paginate(q int) int { return q }\n",
		"backend/internal/domain/webinar/service.go": "package webinar\n\nfunc PublishWebinar() error { return nil }\n",
		"backend/repository/dlq.go":                  "package repository\n\nfunc ListDLQ() error { return nil }\n",
		"backend/models/models.go":                   "package models\n\ntype Webinar struct{}\n\ntype Category struct{}\n",
		"admin/app/composables/useWebinar.ts":        "export const useWebinar = () => 1\n",
		"backend/handlers/admin_webinar_test.go":     "package handlers\n\nfunc TestAdminListWebinars(t *testing.T) {}\n",
	})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Project{}, &models.Component{}, &models.ComponentRoute{}, &models.CodeMap{}, &models.CodeMapEntry{}); err != nil {
		t.Fatal(err)
	}
	p := models.Project{Name: "bimtek", Slug: "bimtek", RootPath: root, Components: []models.Component{
		{Name: "BE", Role: "backend", AbsolutePath: filepath.Join(root, "backend")},
		{Name: "Admin", Role: "frontend_admin", AbsolutePath: filepath.Join(root, "admin")},
	}}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}

	// path relatif komponen (seperti TargetFile temuan) harus dikenali
	g := BuildLayerGuide(db, p.ID, "handlers/admin_webinar.go", true)
	if g == nil {
		t.Fatal("guide expected")
	}
	if g.Target != "backend/handlers/admin_webinar.go" || g.Layer != LayerHandler {
		t.Fatalf("target/layer: %+v", g)
	}
	want := map[string]string{
		LayerHandler: "backend/handlers/", LayerService: "backend/internal/domain/webinar/",
		LayerRepository: "backend/repository/", LayerModel: "backend/models/", LayerUtil: "backend/handlers/",
	}
	for layer, dir := range want {
		if g.MainDir(layer) != dir {
			t.Fatalf("layer %s: want %s, got %+v", layer, dir, g.Layers[layer])
		}
	}
	if len(g.Inventory) != 4 || g.Inventory[0].Name != "webinarRepo" || g.Inventory[1].From != 5 || !strings.Contains(g.Inventory[1].Signature, "(r *webinarRepo)") {
		t.Fatalf("inventory with line ranges & signature expected: %+v", g.Inventory)
	}
	if len(g.Twins) != 1 || g.Twins[0].Name != "AdminListWebinars" || g.Twins[0].Others[0].Path != "backend/handlers/webinar.go" {
		t.Fatalf("AdminListWebinars ≈ ListWebinars expected (RegisterRoutes is generic): %+v", g.Twins)
	}
	reuse := map[string]bool{}
	for _, r := range g.Reuse {
		reuse[r.Path] = true
	}
	if !reuse["backend/internal/domain/webinar/service.go"] || !reuse["backend/handlers/helpers.go"] || !reuse["backend/models/models.go"] || reuse["backend/repository/dlq.go"] {
		t.Fatalf("reuse must list domain service, shared helpers & matching model only: %+v", g.Reuse)
	}
	prompt := g.Prompt()
	for _, s := range []string{"Peta Layer Proyek", "backend/internal/domain/webinar/", "Inventaris Fungsi", "Kandidat Fungsi Kembar", "[model] backend/models/models.go: Webinar"} {
		if !strings.Contains(prompt, s) {
			t.Fatalf("prompt missing %q:\n%s", s, prompt)
		}
	}
	if strings.Contains(prompt, "useWebinar") || strings.Contains(prompt, "Category") {
		t.Fatalf("other component / unrelated model leaked into prompt:\n%s", prompt)
	}
}

func TestTwinKeyIgnoresGenericNames(t *testing.T) {
	cases := map[string]string{
		"AdminListWebinars": "list webinars", "handleListWebinars": "list webinars", "list_webinars": "list webinars",
		"GetByID": "", "RegisterRoutes": "", "TableName": "", "New": "", "parseOfficialLimit": "limit official parse",
	}
	for in, want := range cases {
		if got := twinKey(in); got != want {
			t.Fatalf("twinKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLayerGuideHelpers(t *testing.T) {
	t.Run("isReuseLayer", func(t *testing.T) {
		valid := []string{LayerService, LayerRepository, LayerModel, LayerUtil}
		for _, l := range valid {
			if !isReuseLayer(l) {
				t.Fatalf("expected %s to be reuse layer", l)
			}
		}
		invalid := []string{LayerHandler, LayerMiddleware, "unknown", ""}
		for _, l := range invalid {
			if isReuseLayer(l) {
				t.Fatalf("expected %s to NOT be reuse layer", l)
			}
		}
	})

	t.Run("pathContainsWords", func(t *testing.T) {
		words := []string{"webinar", "admin"}
		if !pathContainsWords("backend/handlers/admin_webinar.go", words) {
			t.Fatal("expected match for admin_webinar.go")
		}
		if !pathContainsWords("BACKEND/HANDLERS/WEBINAR.GO", words) {
			t.Fatal("expected case-insensitive match")
		}
		if pathContainsWords("backend/handlers/user.go", words) {
			t.Fatal("did not expect match for user.go")
		}
		if pathContainsWords("", words) {
			t.Fatal("did not expect match for empty string")
		}
	})

	t.Run("buildLayerDirs", func(t *testing.T) {
		dirs := map[string]map[string]*LayerDir{
			LayerHandler: {
				"a/": &LayerDir{Dir: "a/", Files: 2, Lines: 100},
				"b/": &LayerDir{Dir: "b/", Files: 5, Lines: 200},
				"c/": &LayerDir{Dir: "c/", Files: 5, Lines: 150},
			},
		}
		res := buildLayerDirs(dirs)
		list := res[LayerHandler]
		if len(list) != 3 {
			t.Fatalf("expected 3 dirs, got %d", len(list))
		}
		if list[0].Dir != "b/" || list[1].Dir != "c/" || list[2].Dir != "a/" {
			t.Fatalf("unexpected sorting order: %+v", list)
		}
	})

	t.Run("newLayerGuide", func(t *testing.T) {
		d1 := &doc{entry: models.CodeMapEntry{Path: "backend/handlers/webinar.go", ComponentName: "BE", Lang: "go", Lines: 50}}
		g1 := newLayerGuide(d1)
		if g1.Scope != "BE" || g1.Target != "backend/handlers/webinar.go" {
			t.Fatalf("unexpected guide: %+v", g1)
		}

		d2 := &doc{entry: models.CodeMapEntry{Path: "backend/handlers/webinar.go", ComponentName: "", Lang: "go", Lines: 50}}
		g2 := newLayerGuide(d2)
		if g2.Scope != "backend" {
			t.Fatalf("expected topDir 'backend', got %s", g2.Scope)
		}
	})
}

