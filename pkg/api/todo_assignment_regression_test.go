package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
)

// helper postTodoReq mengirim HTTP request POST /api/todos ke Fiber app server hermetis.
func postTodoReq(t *testing.T, s *Server, body string, headers map[string]string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/todos", strings.NewReader(body))
	req.Host = "localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(localauth.Header, testLocalToken)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request POST /api/todos failed: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	var res map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &res)
	return resp.StatusCode, res
}

// 1. Skenario Payload Valid (201 Created): Penyimpanan Todo berhasil dan memicu assignment sekali.
func TestTodoStorageAndAssignment_ValidPayload(t *testing.T) {
	s := newGuardTestServer(t)

	var enqueueCount int32
	var enqueuedTodoID uint
	s.SetEnqueueAssignment(func(db *gorm.DB, todo *models.Todo) error {
		atomic.AddInt32(&enqueueCount, 1)
		enqueuedTodoID = todo.ID
		return nil
	})

	// Kasus 1: Payload minimal valid
	code, res := postTodoReq(t, s, `{"title":"Implementasi gateway pembayaran"}`, nil)
	if code != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %v", code, res)
	}

	todoID := uint(res["id"].(float64))
	if todoID == 0 {
		t.Fatal("expected non-zero ID for created todo")
	}
	if res["title"] != "Implementasi gateway pembayaran" {
		t.Errorf("title = %v, want 'Implementasi gateway pembayaran'", res["title"])
	}
	if res["status"] != models.TodoStatusPending {
		t.Errorf("status = %v, want pending", res["status"])
	}
	if res["priority"] != models.TodoPriorityMedium {
		t.Errorf("priority = %v, want medium", res["priority"])
	}
	if res["level"] != models.TodoLevelGlobal {
		t.Errorf("level = %v, want global", res["level"])
	}

	// Verifikasi di DB SQLite terisolasi
	var inDB models.Todo
	if err := s.db.First(&inDB, todoID).Error; err != nil {
		t.Fatalf("todo not found in db: %v", err)
	}
	if inDB.Title != "Implementasi gateway pembayaran" {
		t.Errorf("DB title = %q, want 'Implementasi gateway pembayaran'", inDB.Title)
	}

	// Verifikasi assignment enqueue dipanggil dengan ID yang tepat
	if count := atomic.LoadInt32(&enqueueCount); count != 1 {
		t.Errorf("enqueue count = %d, want 1", count)
	}
	if enqueuedTodoID != todoID {
		t.Errorf("enqueued todo ID = %d, want %d", enqueuedTodoID, todoID)
	}

	// Kasus 2: Payload dengan relasi ComponentID (resolusi entity otomatis)
	code2, res2 := postTodoReq(t, s, `{"title":"Optimasi query worker","component_id":1}`, nil)
	if code2 != fiber.StatusCreated {
		t.Fatalf("expected 201 Created for component todo, got %d: %v", code2, res2)
	}
	if res2["component_name"] != "api" {
		t.Errorf("component_name = %v, want 'api'", res2["component_name"])
	}
	if res2["project_name"] != "demo" {
		t.Errorf("project_name = %v, want 'demo'", res2["project_name"])
	}
	if count := atomic.LoadInt32(&enqueueCount); count != 2 {
		t.Errorf("enqueue count after 2nd todo = %d, want 2", count)
	}
}

// 2. Skenario Payload Tidak Valid (400 Bad Request): Validasi ketat di boundary handler.
func TestTodoStorageAndAssignment_InvalidPayload(t *testing.T) {
	s := newGuardTestServer(t)

	var enqueueCount int32
	s.SetEnqueueAssignment(func(db *gorm.DB, todo *models.Todo) error {
		atomic.AddInt32(&enqueueCount, 1)
		return nil
	})

	longTitle := strings.Repeat("A", maxLabel+1)

	testCases := []struct {
		name    string
		payload string
		wantSub string
	}{
		{"Empty JSON object", `{}`, "title is required"},
		{"Empty title string", `{"title":""}`, "title is required"},
		{"Whitespace only title", `{"title":"   \t  \n"}`, "title is required"},
		{"Malformed JSON syntax", `{"title":"broken`, "tidak valid"},
		{"Title exceeds length limit", fmt.Sprintf(`{"title":"%s"}`, longTitle), "maksimal"},
		{"Invalid status value", `{"title":"valid","status":"invalid_state"}`, "status harus salah satu dari"},
		{"Invalid priority value", `{"title":"valid","priority":"extreme"}`, "priority harus salah satu dari"},
		{"Invalid level value", `{"title":"valid","level":"universe"}`, "level harus salah satu dari"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code, res := postTodoReq(t, s, tc.payload, nil)
			if code != fiber.StatusBadRequest {
				t.Fatalf("status = %d, want 400 Bad Request for %q (res: %v)", code, tc.name, res)
			}
			errMsg, _ := res["error"].(string)
			if !strings.Contains(strings.ToLower(errMsg), strings.ToLower(tc.wantSub)) {
				t.Errorf("error msg = %q, want containing %q", errMsg, tc.wantSub)
			}
		})
	}

	// Verifikasi tidak ada data yang masuk ke database
	var count int64
	s.db.Model(&models.Todo{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 todos in db after invalid payloads, got %d", count)
	}

	// Verifikasi assignment enqueue sama sekali tidak terpanggil
	if c := atomic.LoadInt32(&enqueueCount); c != 0 {
		t.Errorf("enqueue count = %d, want 0 for invalid payloads", c)
	}
}

// 3. Skenario Kegagalan Database / Rollback: Transaksi tidak meninggalkan data parsial.
func TestTodoStorageAndAssignment_DatabaseFailureRollback(t *testing.T) {
	s := newGuardTestServer(t)

	var enqueueCount int32
	s.SetEnqueueAssignment(func(db *gorm.DB, todo *models.Todo) error {
		atomic.AddInt32(&enqueueCount, 1)
		return nil
	})

	// Injeksi kegagalan DB melalui callback GORM pada tabel todos
	errInjected := errors.New("simulasi kegagalan I/O database")
	cb := s.db.Callback()
	_ = cb.Create().Before("gorm:create").Register("test_fail_create", func(tx *gorm.DB) {
		if tx.Statement.Table == "todos" || (tx.Statement.Schema != nil && tx.Statement.Schema.Table == "todos") {
			_ = tx.AddError(errInjected)
		}
	})

	code, res := postTodoReq(t, s, `{"title":"Todo yang harus rollback"}`, nil)
	if code != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 Internal Server Error, res: %v", code, res)
	}

	// Kontrak error terstruktur
	if res["code"] != "DB_SAVE_FAILED" {
		t.Errorf("code = %v, want 'DB_SAVE_FAILED'", res["code"])
	}
	if res["stage"] != "database" {
		t.Errorf("stage = %v, want 'database'", res["stage"])
	}

	// Hapus callback injeksi agar query verifikasi dapat berjalan normal
	_ = cb.Create().Remove("test_fail_create")

	// Verifikasi di DB SQLite: tidak ada data tersimpan (transaksi rollback)
	var count int64
	s.db.Model(&models.Todo{}).Where("title = ?", "Todo yang harus rollback").Count(&count)
	if count != 0 {
		t.Errorf("expected 0 todos in DB after DB failure rollback, got %d", count)
	}

	// Assignment tidak boleh pernah dipanggil bila DB Create gagal
	if c := atomic.LoadInt32(&enqueueCount); c != 0 {
		t.Errorf("enqueue count = %d, want 0 when DB create fails", c)
	}
}

// 4. Skenario Kegagalan Enqueue Assignment: Error terstruktur tanpa HTTP 500 tak terjelaskan, data di-rollback.
func TestTodoStorageAndAssignment_EnqueueFailureRollbackAndContract(t *testing.T) {
	s := newGuardTestServer(t)

	// Kasus 4A: Enqueue assignment gagal dengan error terstruktur
	s.SetEnqueueAssignment(func(db *gorm.DB, todo *models.Todo) error {
		return errors.New("antrean assignment penuh / worker timeout")
	})

	code, res := postTodoReq(t, s, `{"title":"Todo gagal enqueue"}`, nil)

	// Kontrak eksplisit: 503 Service Unavailable, BUKAN 500 tak terjelaskan
	if code != fiber.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable, got %d: %v", code, res)
	}

	if res["code"] != "ASSIGNMENT_ENQUEUE_FAILED" {
		t.Errorf("code = %v, want 'ASSIGNMENT_ENQUEUE_FAILED'", res["code"])
	}
	if res["status"] != "error" {
		t.Errorf("status = %v, want 'error'", res["status"])
	}
	if res["stage"] != "assignment" {
		t.Errorf("stage = %v, want 'assignment'", res["stage"])
	}
	if !strings.Contains(fmt.Sprint(res["detail"]), "antrean assignment penuh") {
		t.Errorf("detail = %v, want containing 'antrean assignment penuh'", res["detail"])
	}

	// Verifikasi rollback: record todo TIDAK BOLEH tersimpan di database dalam kondisi unassigned parsial
	var count int64
	s.db.Model(&models.Todo{}).Where("title = ?", "Todo gagal enqueue").Count(&count)
	if count != 0 {
		t.Fatalf("KONTRAK DILANGGAR: Todo tersimpan di DB padahal enqueue gagal (count = %d, want 0)", count)
	}

	// Kasus 4B: Enqueue assignment mengalami panic (fault tolerance)
	// Server tidak boleh panic/crash; transaksi harus tetap di-rollback dan mengembalikan 503 terstruktur
	s.SetEnqueueAssignment(func(db *gorm.DB, todo *models.Todo) error {
		panic("fatal unexpected panic di worker pool")
	})

	codePanic, resPanic := postTodoReq(t, s, `{"title":"Todo memicu panic di queue"}`, nil)
	if codePanic != fiber.StatusServiceUnavailable {
		t.Fatalf("expected 503 on enqueue panic, got %d: %v", codePanic, resPanic)
	}
	if resPanic["code"] != "ASSIGNMENT_ENQUEUE_FAILED" {
		t.Errorf("panic code = %v, want 'ASSIGNMENT_ENQUEUE_FAILED'", resPanic["code"])
	}

	s.db.Model(&models.Todo{}).Where("title = ?", "Todo memicu panic di queue").Count(&count)
	if count != 0 {
		t.Fatalf("KONTRAK DILANGGAR: Todo tersimpan di DB setelah panic (count = %d, want 0)", count)
	}

	// Kasus 4C: Enqueue via registry hook global (registry.H.EnqueueTodoAssignment)
	s.SetEnqueueAssignment(nil) // reset local handler seam
	registry.H.EnqueueTodoAssignment = func(db *gorm.DB, todo *models.Todo) error {
		return errors.New("registry hook queue error")
	}
	t.Cleanup(func() {
		registry.H.EnqueueTodoAssignment = nil
	})

	codeHook, resHook := postTodoReq(t, s, `{"title":"Todo registry hook error"}`, nil)
	if codeHook != fiber.StatusServiceUnavailable {
		t.Fatalf("expected 503 on registry hook error, got %d: %v", codeHook, resHook)
	}
	if resHook["code"] != "ASSIGNMENT_ENQUEUE_FAILED" {
		t.Errorf("hook code = %v, want 'ASSIGNMENT_ENQUEUE_FAILED'", resHook["code"])
	}

	s.db.Model(&models.Todo{}).Where("title = ?", "Todo registry hook error").Count(&count)
	if count != 0 {
		t.Fatalf("KONTRAK DILANGGAR: Todo tersimpan di DB setelah registry hook error (count = %d, want 0)", count)
	}
}

// 5. Skenario Pengulangan Request (Idempotency / Deduplikasi): Request ulang tidak menggandakan assignment.
func TestTodoStorageAndAssignment_IdempotencyAndDeduplication(t *testing.T) {
	s := newGuardTestServer(t)

	var enqueueCount int32
	s.SetEnqueueAssignment(func(db *gorm.DB, todo *models.Todo) error {
		atomic.AddInt32(&enqueueCount, 1)
		return nil
	})

	// Subtest A: Deduplikasi melalui header Idempotency-Key (API-011)
	t.Run("Header Idempotency-Key", func(t *testing.T) {
		headers := map[string]string{"Idempotency-Key": "idemp-req-alpha-001"}
		body := `{"title":"Deploy service ke cluster k8s","priority":"high"}`

		// Request 1: Pembuatan pertama -> 201 Created
		code1, res1 := postTodoReq(t, s, body, headers)
		if code1 != fiber.StatusCreated {
			t.Fatalf("Req 1: status = %d, want 201 Created (res: %v)", code1, res1)
		}
		firstID := uint(res1["id"].(float64))
		if firstID == 0 {
			t.Fatal("firstID is 0")
		}
		if atomic.LoadInt32(&enqueueCount) != 1 {
			t.Fatalf("enqueueCount = %d, want 1 after Req 1", atomic.LoadInt32(&enqueueCount))
		}

		// Request 2 (Pengulangan request yang sama): Idempotent return -> 200 OK
		code2, res2 := postTodoReq(t, s, body, headers)
		if code2 != fiber.StatusOK && code2 != fiber.StatusCreated {
			t.Fatalf("Req 2: status = %d, want 200 OK or 201 (res: %v)", code2, res2)
		}
		secondID := uint(res2["id"].(float64))
		if secondID != firstID {
			t.Errorf("secondID = %d, want same ID as first %d", secondID, firstID)
		}

		// Verifikasi krusial: assignment TIDAK digandakan!
		if c := atomic.LoadInt32(&enqueueCount); c != 1 {
			t.Errorf("KONTRAK DILANGGAR: enqueueCount = %d, want tetap 1 (tidak boleh ganda)", c)
		}

		// Verifikasi krusial: baris di database TIDAK digandakan!
		var rowCount int64
		s.db.Model(&models.Todo{}).Where("idempotency_key = ?", "idemp-req-alpha-001").Count(&rowCount)
		if rowCount != 1 {
			t.Errorf("rowCount in DB = %d, want 1 (tidak boleh ganda)", rowCount)
		}
	})

	// Subtest B: Deduplikasi melalui body JSON field idempotency_key
	t.Run("Payload Field idempotency_key", func(t *testing.T) {
		body := `{"title":"Migrasi skema database","idempotency_key":"idemp-body-beta-002"}`

		// Request 1 -> 201 Created
		code1, res1 := postTodoReq(t, s, body, nil)
		if code1 != fiber.StatusCreated {
			t.Fatalf("Req 1: status = %d, want 201", code1)
		}
		todoID := uint(res1["id"].(float64))

		curEnqueue := atomic.LoadInt32(&enqueueCount)

		// Request 2 -> 200 OK
		code2, res2 := postTodoReq(t, s, body, nil)
		if code2 != fiber.StatusOK && code2 != fiber.StatusCreated {
			t.Fatalf("Req 2: status = %d, want 200", code2)
		}
		if uint(res2["id"].(float64)) != todoID {
			t.Errorf("got id %v, want %d", res2["id"], todoID)
		}

		// Assignment tidak bertambah
		if atomic.LoadInt32(&enqueueCount) != curEnqueue {
			t.Errorf("enqueue count increased: got %d, want %d", atomic.LoadInt32(&enqueueCount), curEnqueue)
		}
	})

	// Subtest C: Deduplikasi melalui header X-Idempotency-Key
	t.Run("Header X-Idempotency-Key", func(t *testing.T) {
		headers := map[string]string{"X-Idempotency-Key": "x-idemp-gamma-003"}
		body := `{"title":"Verifikasi sertifikat TLS"}`

		code1, _ := postTodoReq(t, s, body, headers)
		if code1 != fiber.StatusCreated {
			t.Fatalf("Req 1: status = %d, want 201", code1)
		}
		curEnqueue := atomic.LoadInt32(&enqueueCount)

		code2, _ := postTodoReq(t, s, body, headers)
		if code2 != fiber.StatusOK && code2 != fiber.StatusCreated {
			t.Fatalf("Req 2: status = %d, want 200", code2)
		}
		if atomic.LoadInt32(&enqueueCount) != curEnqueue {
			t.Errorf("enqueue count increased: got %d, want %d", atomic.LoadInt32(&enqueueCount), curEnqueue)
		}
	})

	// Subtest D: Deduplikasi audit_key
	t.Run("AuditKey Deduplication", func(t *testing.T) {
		body := `{"title":"Audit Temuan SEC-001","audit_key":"SEC-001|pkg/auth.go"}`

		code1, res1 := postTodoReq(t, s, body, nil)
		if code1 != fiber.StatusCreated {
			t.Fatalf("Req 1: status = %d, want 201", code1)
		}
		curEnqueue := atomic.LoadInt32(&enqueueCount)

		code2, res2 := postTodoReq(t, s, body, nil)
		if code2 != fiber.StatusOK && code2 != fiber.StatusCreated {
			t.Fatalf("Req 2: status = %d, want 200", code2)
		}
		if uint(res2["id"].(float64)) != uint(res1["id"].(float64)) {
			t.Errorf("got id %v, want %v", res2["id"], res1["id"])
		}
		if atomic.LoadInt32(&enqueueCount) != curEnqueue {
			t.Errorf("enqueue count increased for audit_key: got %d, want %d", atomic.LoadInt32(&enqueueCount), curEnqueue)
		}
	})
}
