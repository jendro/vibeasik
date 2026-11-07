package todorag

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/internal/models"
)

func testDB(t *testing.T) (*gorm.DB, uint) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+filepath.Join(t.TempDir(), "r.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Todo{}); err != nil {
		t.Fatal(err)
	}
	pid := uint(7)
	add := func(title, desc, target, status, memory string) {
		db.Create(&models.Todo{ProjectID: &pid, Title: title, Description: desc, TargetApps: target, Status: status, Memory: memory})
	}
	add("Turunkan latensi endpoint daftar webinar", "Endpoint GET daftar webinar 250 ms, tambah indeks & pagination.", "Bimtek (webinar.go:L10)", models.TodoStatusCompleted, "Percobaan: 1 · nilai QA: 82\nTerakhir: indeks komposit + keyset pagination, p95 90 ms")
	add("Enkripsi kolom PII peserta (NIK, no. HP)", "AES-256-GCM dan blind index untuk NIK.", "Bimtek BE", models.TodoStatusRejected, "")
	add("Terapkan composable useApi terpusat pada Web", "Satukan fetch ke useApi.ts.", "Bimtek Web [1 App]", models.TodoStatusPending, "")
	add("Perbaiki DEEP-FUNC-HIGH-COMPLEXITY: admin.go (L219)", "Fungsi terlalu kompleks.", "Bimtek (admin.go:L219)", models.TodoStatusPending, "")
	add("Halaman verifikasi sertifikat dengan QR", "Publik memindai QR untuk cek keaslian sertifikat.", "Bimtek Web", models.TodoStatusCompleted, "")
	return db, pid
}

func TestDuplicatesRespectTargetsAndFiles(t *testing.T) {
	db, pid := testDB(t)
	if d := Duplicates(db, models.Todo{ProjectID: &pid, Title: "Enkripsi field PII peserta (NIK, no. HP) at-rest", Description: "AES-256-GCM untuk NIK & blind index."}); len(d) == 0 || d[0].Status != models.TodoStatusRejected {
		t.Fatalf("todo PII yang pernah ditolak harus terdeteksi: %+v", d)
	}
	if d := Duplicates(db, models.Todo{ProjectID: &pid, Title: "Terapkan composable useApi terpusat pada Admin", Description: "Satukan fetch ke useApi.ts.", TargetApps: "Bimtek Admin [1 App]"}); len(d) != 0 {
		t.Fatalf("app lain (Admin vs Web) bukan duplikat: %+v", d)
	}
	if d := Duplicates(db, models.Todo{ProjectID: &pid, Title: "Perbaiki DEEP-FUNC-HIGH-COMPLEXITY: admin_user.go (L41)", Description: "Fungsi terlalu kompleks."}); len(d) != 0 {
		t.Fatalf("berkas lain bukan duplikat: %+v", d)
	}
	if d := Duplicates(db, models.Todo{ProjectID: &pid, Title: "Tambah tombol ekspor CSV peserta", Description: "Admin unduh CSV."}); len(d) != 0 {
		t.Fatalf("todo berbeda tidak boleh dianggap duplikat: %+v", d)
	}
}

func TestPromptSectionOnlyClosedRelatedTodos(t *testing.T) {
	db, pid := testDB(t)
	sec := PromptSection(db, models.Todo{ProjectID: &pid, Title: "Turunkan latensi endpoint detail webinar", Description: "GET detail webinar lambat, butuh indeks."}, 3)
	if !strings.Contains(sec, "Turunkan latensi endpoint daftar webinar") || !strings.Contains(sec, "keyset pagination") {
		t.Fatalf("todo latensi yang selesai beserta memorinya harus muncul:\n%s", sec)
	}
	if strings.Contains(sec, "useApi") || strings.Contains(sec, "DEEP-FUNC") {
		t.Fatalf("todo terbuka / tak relevan tidak boleh muncul:\n%s", sec)
	}
	// Indeks dibangun ulang saat todo proyek berubah.
	db.Create(&models.Todo{ProjectID: &pid, Title: "Cache katalog webinar publik", Description: "Latensi endpoint katalog webinar tinggi.", Status: models.TodoStatusCompleted})
	if hits := Similar(db, pid, "katalog webinar", "cache", 5, QuerySim); len(hits) == 0 || hits[0].Title != "Cache katalog webinar publik" {
		t.Fatalf("todo baru harus masuk indeks: %+v", hits)
	}
}
