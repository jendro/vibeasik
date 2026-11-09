package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
)

// TestCreateTodoEntityResolutionAndQueryCount menguji pengurangan query dan resolusi relasi
// komponen-proyek via LEFT JOIN tunggal serta verifikasi K3634 (ID returning).
func TestCreateTodoEntityResolutionAndQueryCount(t *testing.T) {
	s := newGuardTestServer(t)

	// Intercept query DB untuk menghitung jumlah SELECT dan INSERT
	var selectCount int32
	var insertCount int32

	cb := s.db.Callback()
	_ = cb.Query().After("gorm:query").Register("test_count_select", func(db *gorm.DB) {
		atomic.AddInt32(&selectCount, 1)
	})
	_ = cb.Row().After("gorm:row").Register("test_count_row", func(db *gorm.DB) {
		atomic.AddInt32(&selectCount, 1)
	})
	_ = cb.Create().After("gorm:create").Register("test_count_insert", func(db *gorm.DB) {
		atomic.AddInt32(&insertCount, 1)
	})

	resetCounts := func() {
		atomic.StoreInt32(&selectCount, 0)
		atomic.StoreInt32(&insertCount, 0)
	}

	postTodo := func(body string) (int, models.Todo) {
		req := httptest.NewRequest("POST", "/api/todos", strings.NewReader(body))
		req.Host = "localhost"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(localauth.Header, testLocalToken)
		resp, err := s.app.Test(req, -1)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		var created models.Todo
		_ = json.NewDecoder(resp.Body).Decode(&created)
		return resp.StatusCode, created
	}

	// Kasus 1: Hanya ComponentID (Jalur kritis yang sebelumnya memicu 3 SELECT + 1 INSERT)
	// Sekarang harus diselesaikan dengan tepat 1 SELECT (LEFT JOIN components -> projects) + 1 INSERT.
	resetCounts()
	code, todo1 := postTodo(`{"title":"Todo komponen api","component_id":1}`)
	if code != 201 {
		t.Fatalf("Kasus 1: status = %d, want 201", code)
	}
	if todo1.ID == 0 {
		t.Fatal("K3634: ID hasil INSERT harus otomatis terisi tanpa query SELECT terpisah")
	}
	if todo1.ComponentName != "api" {
		t.Errorf("Kasus 1: ComponentName = %q, want 'api'", todo1.ComponentName)
	}
	if todo1.ProjectID == nil || *todo1.ProjectID != 1 {
		t.Errorf("Kasus 1: ProjectID = %v, want 1", todo1.ProjectID)
	}
	if todo1.ProjectName != "demo" {
		t.Errorf("Kasus 1: ProjectName = %q, want 'demo'", todo1.ProjectName)
	}
	if todo1.TargetApps != "api [1 App]" {
		t.Errorf("Kasus 1: TargetApps = %q, want 'api [1 App]'", todo1.TargetApps)
	}
	if sCount := atomic.LoadInt32(&selectCount); sCount != 1 {
		t.Errorf("Kasus 1: SELECT count = %d, want tepat 1 (LEFT JOIN)", sCount)
	}
	if iCount := atomic.LoadInt32(&insertCount); iCount != 1 {
		t.Errorf("Kasus 1: INSERT count = %d, want 1", iCount)
	}

	// Kasus 2: Hanya ProjectID (lookup tunggal ke projects)
	// Tepat 1 SELECT + 1 INSERT
	resetCounts()
	code, todo2 := postTodo(`{"title":"Todo proyek demo","project_id":1}`)
	if code != 201 {
		t.Fatalf("Kasus 2: status = %d, want 201", code)
	}
	if todo2.ID == 0 {
		t.Fatal("K3634: ID harus terisi")
	}
	if todo2.ProjectName != "demo" {
		t.Errorf("Kasus 2: ProjectName = %q, want 'demo'", todo2.ProjectName)
	}
	if todo2.ComponentName != "" {
		t.Errorf("Kasus 2: ComponentName harus kosong, got %q", todo2.ComponentName)
	}
	if todo2.TargetApps != "Seluruh Aplikasi (Bundle demo)" {
		t.Errorf("Kasus 2: TargetApps = %q, want 'Seluruh Aplikasi (Bundle demo)'", todo2.TargetApps)
	}
	if sCount := atomic.LoadInt32(&selectCount); sCount != 1 {
		t.Errorf("Kasus 2: SELECT count = %d, want 1", sCount)
	}

	// Kasus 3: Keduanya (ComponentID dan ProjectID sama-sama terisi)
	// Guard clause refs.ProjectName == "" mencegah query ganda; tepat 1 SELECT + 1 INSERT
	resetCounts()
	code, todo3 := postTodo(`{"title":"Todo keduanya","component_id":1,"project_id":1}`)
	if code != 201 {
		t.Fatalf("Kasus 3: status = %d, want 201", code)
	}
	if todo3.ComponentName != "api" || todo3.ProjectName != "demo" {
		t.Errorf("Kasus 3: data entity salah: comp=%q, prj=%q", todo3.ComponentName, todo3.ProjectName)
	}
	if sCount := atomic.LoadInt32(&selectCount); sCount != 1 {
		t.Errorf("Kasus 3: SELECT count = %d, want 1", sCount)
	}

	// Kasus 4: Tanpa ComponentID dan tanpa ProjectID (Global Todo)
	// Tepat 0 SELECT + 1 INSERT
	resetCounts()
	code, todo4 := postTodo(`{"title":"Todo global platform"}`)
	if code != 201 {
		t.Fatalf("Kasus 4: status = %d, want 201", code)
	}
	if todo4.Level != models.TodoLevelGlobal {
		t.Errorf("Kasus 4: Level = %q, want 'global'", todo4.Level)
	}
	if todo4.TargetApps != "Global Platform" {
		t.Errorf("Kasus 4: TargetApps = %q, want 'Global Platform'", todo4.TargetApps)
	}
	if sCount := atomic.LoadInt32(&selectCount); sCount != 0 {
		t.Errorf("Kasus 4: SELECT count = %d, want 0 (tanpa query)", sCount)
	}
	if iCount := atomic.LoadInt32(&insertCount); iCount != 1 {
		t.Errorf("Kasus 4: INSERT count = %d, want 1", iCount)
	}
}

func BenchmarkCreateTodoOptimization(b *testing.B) {
	s := newGuardTestServer(&testing.T{})
	payload := []byte(`{"title":"Benchmark todo","component_id":1}`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/todos", bytes.NewReader(payload))
		req.Host = "localhost"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(localauth.Header, testLocalToken)
		resp, err := s.app.Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		if resp.StatusCode != 201 {
			b.Fatalf("expected 201, got %d", resp.StatusCode)
		}
	}
}
