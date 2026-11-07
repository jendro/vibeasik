package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestTodoIndexFind(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Skip("sqlite tidak tersedia:", err)
	}
	if err := db.AutoMigrate(&Todo{}); err != nil {
		t.Skip(err)
	}
	p1, p2 := uint(1), uint(2)
	db.Create(&Todo{Title: "A", ProjectID: &p1})
	db.Create(&Todo{Title: "A", ProjectID: &p2})
	idx := NewTodoIndex(db, []Todo{{Title: "A", ProjectID: &p2}, {Title: "B"}})
	got, ok := idx.Find(Todo{Title: "A", ProjectID: &p2})
	if !ok || got.ProjectID == nil || *got.ProjectID != 2 {
		t.Fatalf("salah cocok: %+v %v", got, ok)
	}
	if _, ok := idx.Find(Todo{Title: "B"}); ok {
		t.Fatal("B tidak boleh ada")
	}
	if _, ok := idx.Find(Todo{Title: "A"}); !ok {
		t.Fatal("A tanpa project harus cocok")
	}
	idx.Add(Todo{Title: "B"})
	if _, ok := idx.Find(Todo{Title: "B"}); !ok {
		t.Fatal("B setelah Add harus cocok")
	}
}
