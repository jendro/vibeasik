package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNormalizeTags(t *testing.T) {
	got := NormalizeTags([]interface{}{" Rawat AI", "fitur", "fitur", ""})
	if got != "fitur,rawat-ai" {
		t.Fatalf("NormalizeTags = %q", got)
	}
	if NormalizeTags("belajar, Keamanan-Dasar") != "belajar,keamanan-dasar" {
		t.Fatal("CSV tidak dinormalkan")
	}
}

func TestFocusRank(t *testing.T) {
	cases := map[string]int{"fitur": 0, "rawat-ai,keamanan-dasar": 1, "": 4, "belajar": 5, "belajar,optimasi-query": 2}
	for tags, want := range cases {
		if got := FocusRank(tags); got != want {
			t.Errorf("FocusRank(%q) = %d, want %d", tags, got, want)
		}
	}
}

func TestWhereTagSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Todo{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&Todo{Title: "a", Tags: "fitur,rawat-ai"})
	db.Create(&Todo{Title: "b", Tags: "rawat-ai-lama"})
	var n int64
	WhereTag(db.Model(&Todo{}), "rawat-ai").Count(&n)
	if n != 1 {
		t.Fatalf("WhereTag rawat-ai = %d, want 1", n)
	}
}

// Todo gabungan lintas kategori ditemukan lewat kategori utama maupun kategori sub todonya.
func TestWhereCategoryMatchesGroupCategories(t *testing.T) {
	db := newCategoryTestDB(t)
	db.Create(&Todo{Title: "gabungan", Category: CategoryArchitecture, Categories: JoinCategories("Architecture", "security", "architecture")})
	db.Create(&Todo{Title: "biasa", Category: CategorySecurity})
	db.Create(&Todo{Title: "lain", Category: CategoryUIUX, Status: "pending"})
	count := func(cat string) int64 {
		var n int64
		WhereCategory(db.Model(&Todo{}), cat).Where("title <> ?", "x").Count(&n)
		return n
	}
	if n := count("security"); n != 2 {
		t.Fatalf("security should match the group and the plain todo, got %d", n)
	}
	if n := count("Architecture"); n != 1 {
		t.Fatalf("architecture should match the group only, got %d", n)
	}
	if got := JoinCategories("Architecture", "security", "architecture"); got != "architecture,security" {
		t.Fatalf("JoinCategories = %q", got)
	}
}

func newCategoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Todo{}); err != nil {
		t.Fatal(err)
	}
	return db
}
