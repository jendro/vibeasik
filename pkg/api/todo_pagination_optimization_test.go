package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
)

// TestTodoPaginationAndIndexOptimization memverifikasi:
// 1. Skema SQLite memiliki composite index idx_todos_status_created (status, created_at)
// 2. Default limit = 50 jika query tanpa parameter ?limit
// 3. Batas plafon maxTodoLimit = 200 jika klien meminta > 200
// 4. Offset pagination bekerja dengan benar
// 5. Latensi GET /api/todos konsisten berada di bawah 200ms SLO
func TestTodoPaginationAndIndexOptimization(t *testing.T) {
	s := newGuardTestServer(t)

	// 1. Verifikasi composite index idx_todos_status_created pada SQLite schema
	type indexRow struct {
		Seq    int    `gorm:"column:seq"`
		Name   string `gorm:"column:name"`
		Unique int    `gorm:"column:unique"`
		Origin string `gorm:"column:origin"`
	}
	var indexes []indexRow
	if err := s.db.Raw("PRAGMA index_list('todos')").Scan(&indexes).Error; err != nil {
		t.Fatalf("gagal membaca PRAGMA index_list: %v", err)
	}
	foundCompositeIndex := false
	for _, idx := range indexes {
		if idx.Name == "idx_todos_status_created" {
			foundCompositeIndex = true
			break
		}
	}
	if !foundCompositeIndex {
		t.Errorf("indeks 'idx_todos_status_created' tidak ditemukan di tabel todos. Indeks terdaftar: %+v", indexes)
	}

	// Periksa kolom index info
	type indexColRow struct {
		Seqno int    `gorm:"column:seqno"`
		Cid   int    `gorm:"column:cid"`
		Name  string `gorm:"column:name"`
	}
	var cols []indexColRow
	if err := s.db.Raw("PRAGMA index_info('idx_todos_status_created')").Scan(&cols).Error; err != nil {
		t.Fatalf("gagal membaca PRAGMA index_info: %v", err)
	}
	if len(cols) != 2 || cols[0].Name != "status" || cols[1].Name != "created_at" {
		t.Errorf("kolom idx_todos_status_created tidak sesuai: %+v (want: [status, created_at])", cols)
	}

	// 2. Siapkan dataset: 60 todos (35 pending, 25 completed) untuk proyek 1
	projID := uint(1)
	baseTime := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	for i := 1; i <= 60; i++ {
		status := models.TodoStatusPending
		if i > 35 {
			status = models.TodoStatusCompleted
		}
		item := models.Todo{
			ProjectID:   &projID,
			ProjectName: "demo",
			Title:       fmt.Sprintf("Todo item %03d", i),
			Status:      status,
			Level:       models.TodoLevelProject,
			CreatedAt:   baseTime.Add(time.Duration(i) * time.Minute),
			UpdatedAt:   baseTime.Add(time.Duration(i) * time.Minute),
		}
		if err := s.db.Create(&item).Error; err != nil {
			t.Fatalf("gagal membuat todo %d: %v", i, err)
		}
	}

	fetchTodos := func(path string) (int, []models.Todo, time.Duration) {
		req := httptest.NewRequest("GET", path, nil)
		req.Host = "localhost"
		req.Header.Set(localauth.Header, testLocalToken)
		start := time.Now()
		resp, err := s.app.Test(req, -1)
		duration := time.Since(start)
		if err != nil {
			t.Fatalf("request %s failed: %v", path, err)
		}
		var list []models.Todo
		_ = json.NewDecoder(resp.Body).Decode(&list)
		return resp.StatusCode, list, duration
	}

	// 3. Uji Default Limit: GET /api/todos tanpa query parameter
	// Harus dibatasi tepat defaultTodoLimit = 50 meskipun ada 60 baris di database
	code, items, latency := fetchTodos("/api/todos")
	if code != 200 {
		t.Fatalf("GET /api/todos status = %d, want 200", code)
	}
	if len(items) != defaultTodoLimit {
		t.Errorf("GET /api/todos tanpa limit: got %d items, want defaultTodoLimit = %d", len(items), defaultTodoLimit)
	}
	if latency > 200*time.Millisecond {
		t.Errorf("GET /api/todos latency = %v melebihi SLO 200ms", latency)
	}

	// Verifikasi urutan: todo pending (terbuka) harus mendahului completed
	if len(items) > 0 && items[0].Status != models.TodoStatusPending {
		t.Errorf("item pertama harus berstatus pending karena todoOpenFirstOrder, got %s", items[0].Status)
	}

	// 4. Uji Custom Limit: GET /api/todos?limit=10
	code, items10, _ := fetchTodos("/api/todos?limit=10")
	if code != 200 || len(items10) != 10 {
		t.Errorf("GET /api/todos?limit=10: status=%d, count=%d (want 10)", code, len(items10))
	}

	// 5. Uji Clamping Max Limit: GET /api/todos?limit=500
	// Diuji batas fungsi todoQueryLimit:
	// Karena hanya ada 60 item di db, pastikan semua 60 item kembali dan tidak melebihi maxTodoLimit (200)
	code, itemsClamped, _ := fetchTodos("/api/todos?limit=500")
	if code != 200 || len(itemsClamped) != 60 {
		t.Errorf("GET /api/todos?limit=500: status=%d, count=%d (want 60)", code, len(itemsClamped))
	}

	// 6. Uji Offset Pagination: GET /api/todos?limit=5&offset=10
	code, itemsOffset, _ := fetchTodos("/api/todos?limit=5&offset=10")
	if code != 200 || len(itemsOffset) != 5 {
		t.Fatalf("GET /api/todos?limit=5&offset=10: status=%d, count=%d (want 5)", code, len(itemsOffset))
	}
	// Pastikan item yang diambil cocok dengan elemen indeks ke 10 s/d 14 dari items default
	for i := 0; i < 5; i++ {
		if itemsOffset[i].ID != items[10+i].ID {
			t.Errorf("offset mismatch pada indeks %d: got ID %d, want ID %d", i, itemsOffset[i].ID, items[10+i].ID)
		}
	}

	// 7. Uji Endpoint Berlingkup Proyek: GET /api/projects/1/todos
	code, projItems, projLatency := fetchTodos("/api/projects/1/todos")
	if code != 200 {
		t.Fatalf("GET /api/projects/1/todos status = %d, want 200", code)
	}
	if len(projItems) != defaultTodoLimit {
		t.Errorf("GET /api/projects/1/todos: got %d items, want default limit %d", len(projItems), defaultTodoLimit)
	}
	if projLatency > 200*time.Millisecond {
		t.Errorf("GET /api/projects/1/todos latency = %v melebihi SLO 200ms", projLatency)
	}
}

func BenchmarkGetTodosPagination(b *testing.B) {
	s := newGuardTestServer(&testing.T{})
	projID := uint(1)
	baseTime := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	for i := 1; i <= 60; i++ {
		_ = s.db.Create(&models.Todo{
			ProjectID:   &projID,
			ProjectName: "demo",
			Title:       fmt.Sprintf("Benchmark todo %d", i),
			Status:      models.TodoStatusPending,
			CreatedAt:   baseTime.Add(time.Duration(i) * time.Minute),
		})
	}

	req := httptest.NewRequest("GET", "/api/todos?limit=50", nil)
	req.Host = "localhost"
	req.Header.Set(localauth.Header, testLocalToken)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resp, err := s.app.Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		if resp.StatusCode != 200 {
			b.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	}
}
