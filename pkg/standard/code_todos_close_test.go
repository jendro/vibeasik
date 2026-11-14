package standard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Todo kode standar yang masalahnya sudah tidak terdeteksi ditutup; yang masih terdeteksi tetap; todo yang sedang
// dilebur ke todo gabungan tidak dibuat salinannya.
func TestEnsureClosesGoneCodeTodosAndSkipsMerged(t *testing.T) {
	db := setupDB(t)
	root := t.TempDir()
	fe := filepath.Join(root, "frontend")
	os.MkdirAll(filepath.Join(fe, "app"), 0o755)
	os.WriteFile(filepath.Join(fe, "nuxt.config.ts"), []byte("export default defineNuxtConfig({})\n"), 0o644)
	// Masih bermasalah: dua import dari modul yang sama.
	os.WriteFile(filepath.Join(fe, "app", "a.ts"), []byte("import { a } from 'x'\nimport { b } from 'x'\n"), 0o644)
	// Bukan import ganda: namespace + named tidak bisa digabung.
	os.WriteFile(filepath.Join(fe, "app", "b.ts"), []byte("import * as vue from 'vue'\nimport { h } from 'vue'\n"), 0o644)

	p := models.Project{Name: "Toko", Slug: "toko", RootPath: root, Status: models.StatusActive,
		Components: []models.Component{{Name: "web", Role: "frontend_client", RelativePath: "frontend", AbsolutePath: fe, Framework: "nuxt"}}}
	db.Create(&p)
	comp := p.Components[0].ID
	pid := p.ID
	gone := models.Todo{ProjectID: &pid, ComponentID: &comp, Title: "Duplicated Auto-Import Nuxt: 'useToast' (app/composables/useApi.ts)",
		Status: models.TodoStatusPending, Source: "vibeasik_standard"}
	db.Create(&gone)

	if _, err := Ensure(db, p.ID); err != nil {
		t.Fatal(err)
	}
	db.First(&gone, gone.ID)
	if gone.Status != models.TodoStatusCompleted {
		t.Fatalf("todo yang masalahnya hilang harus ditutup, status %q", gone.Status)
	}
	var open []models.Todo
	db.Where("project_id = ? AND source = ? AND status = ?", p.ID, "vibeasik_standard", models.TodoStatusPending).Find(&open)
	var dupA, dupB int
	for _, t := range open {
		if t.Title == "Gabungkan duplicate import (frontend/app/a.ts)" {
			dupA++
		}
		if t.Title == "Gabungkan duplicate import (frontend/app/b.ts)" {
			dupB++
		}
	}
	if dupA != 1 || dupB != 0 {
		t.Fatalf("a.ts harus tetap 1 todo (ada %d), b.ts tanpa todo (ada %d); todo: %+v", dupA, dupB, open)
	}

	// Dilebur ke todo gabungan → Ensure berikutnya tidak membuat salinan baru.
	for _, o := range open {
		if o.Title == "Gabungkan duplicate import (frontend/app/a.ts)" {
			db.Model(&models.Todo{}).Where("id = ?", o.ID).Update("status", models.TodoStatusMerged)
		}
	}
	if _, err := Ensure(db, p.ID); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Model(&models.Todo{}).Where("title = ? AND status = ?", "Gabungkan duplicate import (frontend/app/a.ts)", models.TodoStatusPending).Count(&n)
	if n != 0 {
		t.Fatalf("todo yang sudah dilebur dibuat ulang (%d)", n)
	}
}
