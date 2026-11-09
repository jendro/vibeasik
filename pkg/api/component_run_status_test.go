package api

import (
	"encoding/json"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/runner"
)

// TestGetComponentRunStatus_OptimizedLatencyAndQuery menguji bahwa endpoint HTTP
// GET /api/components/:id/run mengeksekusi tepat 1 query DB dan merespons dalam < 50ms.
func TestGetComponentRunStatus_OptimizedLatencyAndQuery(t *testing.T) {
	s := newGuardTestServer(t)

	var queryCount int32
	cb := s.db.Callback()
	_ = cb.Query().After("gorm:query").Register("test_count_get_status_query", func(db *gorm.DB) {
		atomic.AddInt32(&queryCount, 1)
	})

	req := httptest.NewRequest("GET", "/api/components/1/run", nil)
	req.Host = "localhost"
	req.Header.Set(localauth.Header, testLocalToken)

	atomic.StoreInt32(&queryCount, 0)
	start := time.Now()
	resp, err := s.app.Test(req, -1)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("GET /api/components/:id/run too slow: %v (want < 50ms)", elapsed)
	}

	var status runner.Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if status.ComponentID != 1 || status.Name != "api" {
		t.Errorf("unexpected status body: %+v", status)
	}

	// Hanya tepat 1 query (loadComponentByID), bukan 2 (tanpa siblings)
	if count := atomic.LoadInt32(&queryCount); count != 1 {
		t.Errorf("query count = %d, want tepat 1 (tanpa siblings query)", count)
	}
}
