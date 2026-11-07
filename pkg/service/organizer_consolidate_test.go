package service_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Kasus Silat Solo: FE & BE di folder terpisah, root proyek = folder BE (repo git).
func setupScatteredProject(t *testing.T) (base string, p models.Project) {
	t.Helper()
	base = t.TempDir()
	fe := filepath.Join(base, "Nuxt", "silatsoloFE")
	be := filepath.Join(base, "PHP", "silatsoloBE")
	for _, d := range []string{fe, be} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := exec.Command("git", "-C", d, "init", "-q").Run(); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.WriteFile(filepath.Join(fe, "package.json"), []byte(`{"name":"fe"}`), 0o644)
	_ = os.WriteFile(filepath.Join(be, "composer.json"), []byte(`{"name":"be"}`), 0o644)
	// Artefak Init vibeasik lama di root lama (folder BE).
	_ = os.MkdirAll(filepath.Join(be, ".vibeasik"), 0o755)
	_ = os.WriteFile(filepath.Join(be, ".vibeasik", "CODEMAP.md"), []byte("# old"), 0o644)

	p = models.Project{
		Name:     "silatsolo.com",
		Slug:     "silatsolocom",
		RootPath: be,
		Components: []models.Component{
			{Name: "SilatSoloFE", Role: models.RoleFrontendClient, AbsolutePath: fe, RelativePath: "Nuxt/silatsoloFE"},
			{Name: "SilatsoloBE", Role: models.RoleBackend, AbsolutePath: be, RelativePath: "PHP/silatsoloBE"},
		},
	}
	return base, p
}

func TestConsolidateUpdatesDBAndRoot(t *testing.T) {
	services, db := setupTestServices(t)
	_ = db.AutoMigrate(&models.WorkspaceSettings{}, &models.CodeMap{}, &models.CodeMapEntry{})
	ctx := context.Background()

	base, p := setupScatteredProject(t)
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "projects")

	res, err := services.Organizer.Consolidate(ctx, p.ID, target, "http://localhost:8765")
	if err != nil {
		t.Fatalf("consolidate: %v", err)
	}
	root := filepath.Join(target, "silatsolocom")
	if res.ProjectRoot != root {
		t.Fatalf("root = %s", res.ProjectRoot)
	}

	var got models.Project
	if err := db.Preload("Components").First(&got, p.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.RootPath != root || !got.IsUnified {
		t.Fatalf("project not updated: root=%s unified=%v", got.RootPath, got.IsUnified)
	}
	want := map[string]string{"SilatSoloFE": filepath.Join(root, "frontend"), "SilatsoloBE": filepath.Join(root, "backend")}
	for _, c := range got.Components {
		if c.AbsolutePath != want[c.Name] {
			t.Fatalf("component %s path = %s, want %s", c.Name, c.AbsolutePath, want[c.Name])
		}
	}

	// Komponen vibeasik (tautan AI & peta kode) berada di root folder induk.
	for _, f := range []string{".vibeasik.json", ".vibeasik/CODEMAP.md"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Fatalf("missing %s in unified root: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "backend", ".vibeasik")); err == nil {
		t.Fatalf("stale .vibeasik/ still inside backend component")
	}

	// Menjalankan ulang setelah tersatukan tidak boleh dianggap konflik.
	pv, err := services.Organizer.PreviewConsolidate(ctx, p.ID, target)
	if err != nil {
		t.Fatal(err)
	}
	if pv.HasConflict {
		t.Fatalf("re-preview reports conflict: %+v", pv.Items)
	}
}

// Folder sudah dipindah tetapi DB masih menyimpan path lama (sisa penyatuan yang gagal di tengah jalan):
// preview harus mengenalinya sebagai "sudah di tempat", bukan konflik.
func TestConsolidateRecoversHalfDoneMove(t *testing.T) {
	services, db := setupTestServices(t)
	_ = db.AutoMigrate(&models.WorkspaceSettings{}, &models.CodeMap{}, &models.CodeMapEntry{})
	ctx := context.Background()

	base, p := setupScatteredProject(t)
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "projects")
	root := filepath.Join(target, "silatsolocom")
	_ = os.MkdirAll(root, 0o755)
	if err := os.Rename(p.Components[0].AbsolutePath, filepath.Join(root, "frontend")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(p.Components[1].AbsolutePath, filepath.Join(root, "backend")); err != nil {
		t.Fatal(err)
	}

	pv, err := services.Organizer.PreviewConsolidate(ctx, p.ID, target)
	if err != nil {
		t.Fatal(err)
	}
	if pv.HasConflict || !pv.CanConsolidate {
		t.Fatalf("half-done move reported as conflict: %+v", pv.Items)
	}
	if _, err := services.Organizer.Consolidate(ctx, p.ID, target, "http://localhost:8765"); err != nil {
		t.Fatalf("consolidate: %v", err)
	}
	var got models.Project
	_ = db.Preload("Components").First(&got, p.ID).Error
	if got.RootPath != root || !got.IsUnified {
		t.Fatalf("project not updated: root=%s unified=%v", got.RootPath, got.IsUnified)
	}
	for _, c := range got.Components {
		if filepath.Dir(c.AbsolutePath) != root {
			t.Fatalf("component %s path = %s", c.Name, c.AbsolutePath)
		}
	}
}

// Kasus asli Silat Solo: dua pasang komponen ganda menunjuk folder yang sama. Dulu folder dipindah oleh komponen
// pertama lalu komponen kembarannya gagal ("sumber tidak ditemukan") sehingga penyatuan berhenti di tengah jalan.
func TestConsolidateDuplicateComponentsMoveOnce(t *testing.T) {
	services, db := setupTestServices(t)
	_ = db.AutoMigrate(&models.WorkspaceSettings{}, &models.CodeMap{}, &models.CodeMapEntry{})
	ctx := context.Background()

	base, p := setupScatteredProject(t)
	fe, be := p.Components[0].AbsolutePath, p.Components[1].AbsolutePath
	p.Components = append(p.Components,
		models.Component{Name: "silatsoloFE", Role: models.RoleFrontendClient, AbsolutePath: fe},
		models.Component{Name: "silatsoloBE", Role: models.RoleBackend, AbsolutePath: be},
	)
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "projects")

	pv, err := services.Organizer.PreviewConsolidate(ctx, p.ID, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(pv.Items) != 2 {
		t.Fatalf("want 2 move items for 2 folders, got %d", len(pv.Items))
	}
	if _, err := services.Organizer.Consolidate(ctx, p.ID, target, "http://localhost:8765"); err != nil {
		t.Fatalf("consolidate: %v", err)
	}
	root := filepath.Join(target, "silatsolocom")
	var comps []models.Component
	db.Where("project_id = ?", p.ID).Find(&comps)
	for _, c := range comps {
		want := filepath.Join(root, "backend")
		if c.Role == models.RoleFrontendClient {
			want = filepath.Join(root, "frontend")
		}
		if c.AbsolutePath != want {
			t.Fatalf("component %d %s path = %s, want %s", c.ID, c.Name, c.AbsolutePath, want)
		}
	}
}

// Komponen di dalam folder komponen lain ikut pindah bersama induknya, tidak dipindahkan dua kali.
func TestConsolidateNestedComponentMovesWithParent(t *testing.T) {
	services, db := setupTestServices(t)
	_ = db.AutoMigrate(&models.WorkspaceSettings{}, &models.CodeMap{}, &models.CodeMapEntry{})
	ctx := context.Background()

	base, p := setupScatteredProject(t)
	be := p.Components[1].AbsolutePath
	docs := filepath.Join(be, "docs")
	_ = os.MkdirAll(docs, 0o755)
	p.Components = append(p.Components, models.Component{Name: "Docs", Role: models.RoleDocs, AbsolutePath: docs})
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "projects")
	if _, err := services.Organizer.Consolidate(ctx, p.ID, target, "http://localhost:8765"); err != nil {
		t.Fatalf("consolidate: %v", err)
	}
	var c models.Component
	db.Where("project_id = ? AND name = ?", p.ID, "Docs").First(&c)
	if want := filepath.Join(target, "silatsolocom", "backend", "docs"); c.AbsolutePath != want || !isDir(want) {
		t.Fatalf("nested component path = %s, want %s", c.AbsolutePath, want)
	}
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}
