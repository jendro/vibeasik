package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestRebaseCodeMapPathsRewritesMovedPrefix(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.CodeMap{}, &models.CodeMapEntry{}); err != nil {
		t.Fatal(err)
	}
	oldRoot, newRoot := filepath.FromSlash("/data/old/toko"), filepath.FromSlash("/data/new/toko")
	sibling := filepath.FromSlash("/data/old/toko-lama/main.go")
	db.Create(&models.CodeMap{ProjectID: 1, RootPath: oldRoot})
	db.Create(&models.CodeMap{ProjectID: 2, RootPath: oldRoot})
	entries := []models.CodeMapEntry{
		{ProjectID: 1, Path: "backend/main.go", AbsPath: filepath.Join(oldRoot, "backend", "main.go")},
		{ProjectID: 1, Path: "README.md", AbsPath: filepath.Join(oldRoot, "README.md")},
		{ProjectID: 1, Path: "x/main.go", AbsPath: sibling},
		{ProjectID: 2, Path: "README.md", AbsPath: filepath.Join(oldRoot, "README.md")},
	}
	db.Create(&entries)

	s := &organizerService{db: db}
	if err := s.rebaseCodeMapPaths(context.Background(), 1, []pathMove{{from: oldRoot, to: newRoot}}); err != nil {
		t.Fatal(err)
	}

	var maps []models.CodeMap
	db.Order("project_id").Find(&maps)
	if maps[0].RootPath != newRoot || maps[1].RootPath != oldRoot {
		t.Fatalf("code map roots = %s, %s", maps[0].RootPath, maps[1].RootPath)
	}
	var got []models.CodeMapEntry
	db.Order("id").Find(&got)
	want := []string{
		filepath.Join(newRoot, "backend", "main.go"),
		filepath.Join(newRoot, "README.md"),
		sibling, // folder bersebelahan dengan awalan nama yang sama tidak ikut berubah
		filepath.Join(oldRoot, "README.md"),
	}
	for i, e := range got {
		if e.AbsPath != want[i] {
			t.Fatalf("entry %d abs_path = %s, want %s", i, e.AbsPath, want[i])
		}
		if e.Path != entries[i].Path {
			t.Fatalf("entry %d path changed to %s", i, e.Path)
		}
	}
}
