package service_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-C", dir, "-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}, args...)
	if out, err := exec.Command("git", full...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// Proyek yang sudah berada di satu folder: repo git di root, komponen di subfolder, berkas lain di root.
func setupSingleRootProject(t *testing.T) (base string, p models.Project) {
	t.Helper()
	base = t.TempDir()
	root := filepath.Join(base, "old", "toko")
	writeFiles(t, root, map[string]string{
		"README.md":             "# toko",
		"dev.sh":                "#!/bin/sh\n",
		".env.example":          "PORT=1\n",
		"docs/catatan.md":       "catatan",
		"backend/main.go":       "package main\n",
		"frontend/package.json": `{"name":"fe"}`,
	})
	gitRun(t, root, "init", "-q")
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-q", "-m", "init")

	p = models.Project{
		Name:     "Toko",
		Slug:     "toko",
		RootPath: root,
		Components: []models.Component{
			{Name: "api", Role: models.RoleBackend, AbsolutePath: filepath.Join(root, "backend"), RelativePath: "backend"},
			{Name: "web", Role: models.RoleFrontendClient, AbsolutePath: filepath.Join(root, "frontend"), RelativePath: "frontend"},
		},
	}
	return base, p
}

func TestPreviewConsolidateSingleRootIsOneItem(t *testing.T) {
	services, db := setupTestServices(t)
	_ = db.AutoMigrate(&models.WorkspaceSettings{})
	ctx := context.Background()

	base, p := setupSingleRootProject(t)
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "projects")

	pv, err := services.Organizer.PreviewConsolidate(ctx, p.ID, target)
	if err != nil {
		t.Fatal(err)
	}
	if !pv.WholeRoot || len(pv.Items) != 1 {
		t.Fatalf("want one whole-root item, got whole_root=%v items=%+v", pv.WholeRoot, pv.Items)
	}
	item := pv.Items[0]
	if item.CurrentPath != p.RootPath || item.TargetPath != filepath.Join(target, "toko") || item.Status != "pending" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if !item.IsGitRepo || pv.HasConflict || !pv.CanConsolidate {
		t.Fatalf("unexpected flags: item=%+v conflict=%v", item, pv.HasConflict)
	}
	// Preview tidak menyentuh disk.
	if !isDir(p.RootPath) || isDir(item.TargetPath) {
		t.Fatalf("preview moved files")
	}
}

func TestConsolidateSingleRootMovesWholeFolder(t *testing.T) {
	services, db := setupTestServices(t)
	_ = db.AutoMigrate(&models.WorkspaceSettings{}, &models.CodeMap{}, &models.CodeMapEntry{})
	ctx := context.Background()

	base, p := setupSingleRootProject(t)
	oldRoot := p.RootPath
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	// Worktree tertaut di luar folder proyek menyimpan path absolut repo utama.
	linked := filepath.Join(base, "worktrees", "todo-1")
	gitRun(t, oldRoot, "worktree", "add", "-q", "-b", "todo-1", linked)

	target := filepath.Join(base, "projects")
	res, err := services.Organizer.Consolidate(ctx, p.ID, target, "http://localhost:8765")
	if err != nil {
		t.Fatalf("consolidate: %v", err)
	}
	root := filepath.Join(target, "toko")
	if res.ProjectRoot != root || res.MovedCount != 1 || len(res.Items) != 1 {
		t.Fatalf("result: root=%s moved=%d items=%d", res.ProjectRoot, res.MovedCount, len(res.Items))
	}

	// Berkas di root proyek ikut pindah dan lokasi lama tidak menyisakan apa pun.
	for _, f := range []string{".git", "README.md", "dev.sh", ".env.example", "docs/catatan.md", "backend/main.go", "frontend/package.json"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Fatalf("missing %s in moved root: %v", f, err)
		}
	}
	if _, err := os.Stat(oldRoot); !os.IsNotExist(err) {
		t.Fatalf("old root still exists: %v", err)
	}

	var got models.Project
	if err := db.Preload("Components").First(&got, p.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.RootPath != root || !got.IsUnified {
		t.Fatalf("project not updated: root=%s unified=%v", got.RootPath, got.IsUnified)
	}
	want := map[string]string{"api": "backend", "web": "frontend"}
	for _, c := range got.Components {
		if c.AbsolutePath != filepath.Join(root, want[c.Name]) || c.RelativePath != want[c.Name] {
			t.Fatalf("component %s: abs=%s rel=%s", c.Name, c.AbsolutePath, c.RelativePath)
		}
	}

	// Repo utama tetap satu dan worktree tertaut kembali mengenalinya.
	if out, err := exec.Command("git", "-C", linked, "status", "--porcelain").CombinedOutput(); err != nil {
		t.Fatalf("linked worktree broken after move: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", root, "log", "--oneline").CombinedOutput(); err != nil || len(out) == 0 {
		t.Fatalf("moved repo lost its history: %v\n%s", err, out)
	}

	// Menjalankan ulang: sudah di tempat, tetap satu item, bukan konflik.
	pv, err := services.Organizer.PreviewConsolidate(ctx, p.ID, target)
	if err != nil {
		t.Fatal(err)
	}
	if pv.HasConflict || len(pv.Items) != 1 || pv.Items[0].Status != "already_in_place" {
		t.Fatalf("re-preview: %+v", pv.Items)
	}
}

// Folder proyek sudah dipindah dengan tangan tetapi database masih menyimpan path lama.
func TestConsolidateSingleRootRecoversManualMove(t *testing.T) {
	services, db := setupTestServices(t)
	_ = db.AutoMigrate(&models.WorkspaceSettings{}, &models.CodeMap{}, &models.CodeMapEntry{})
	ctx := context.Background()

	base, p := setupSingleRootProject(t)
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "projects")
	root := filepath.Join(target, "toko")
	mkdirs(t, target)
	if err := os.Rename(p.RootPath, root); err != nil {
		t.Fatal(err)
	}

	res, err := services.Organizer.Consolidate(ctx, p.ID, target, "http://localhost:8765")
	if err != nil {
		t.Fatalf("consolidate: %v", err)
	}
	if res.MovedCount != 0 {
		t.Fatalf("nothing should move, got %d", res.MovedCount)
	}
	var comps []models.Component
	db.Where("project_id = ?", p.ID).Find(&comps)
	for _, c := range comps {
		if filepath.Dir(c.AbsolutePath) != root || !isDir(c.AbsolutePath) {
			t.Fatalf("component %s path = %s", c.Name, c.AbsolutePath)
		}
	}
}

// Komponen yang terpencar, atau root yang hanya folder induk umum, tetap dipindahkan per komponen.
func TestPreviewConsolidateScatteredStaysPerComponent(t *testing.T) {
	services, db := setupTestServices(t)
	_ = db.AutoMigrate(&models.WorkspaceSettings{})
	ctx := context.Background()

	perComponent := func(t *testing.T, p models.Project, target string) {
		t.Helper()
		pv, err := services.Organizer.PreviewConsolidate(ctx, p.ID, target)
		if err != nil {
			t.Fatal(err)
		}
		if pv.WholeRoot || len(pv.Items) != 2 {
			t.Fatalf("want 2 per-component items, got whole_root=%v items=%+v", pv.WholeRoot, pv.Items)
		}
		want := map[string]string{
			models.RoleFrontendClient: filepath.Join(target, p.Slug, "frontend"),
			models.RoleBackend:        filepath.Join(target, p.Slug, "backend"),
		}
		for _, item := range pv.Items {
			if item.TargetPath != want[item.Role] || item.ComponentID == 0 || item.Status != "pending" {
				t.Fatalf("unexpected item: %+v", item)
			}
		}
	}

	t.Run("component outside root", func(t *testing.T) {
		base, p := setupScatteredProject(t)
		if err := db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
		perComponent(t, p, filepath.Join(base, "projects"))
	})

	t.Run("root is a plain parent folder", func(t *testing.T) {
		base, p := setupScatteredProject(t)
		p.Name, p.Slug, p.RootPath = "Induk", "induk", base
		if err := db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
		perComponent(t, p, filepath.Join(filepath.Dir(base), filepath.Base(base)+"-projects"))
	})

	t.Run("root also holds another project", func(t *testing.T) {
		base, p := setupSingleRootProject(t)
		p.Name, p.Slug = "Bersama", "bersama"
		if err := db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
		other := models.Project{Name: "Lain", Slug: "lain", RootPath: filepath.Join(p.RootPath, "docs")}
		if err := db.Create(&other).Error; err != nil {
			t.Fatal(err)
		}
		perComponent(t, p, filepath.Join(base, "projects"))
	})
}
