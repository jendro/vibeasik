package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestDeleteTrafficSource_AtomicAndRollback(t *testing.T) {
	s := newGuardTestServer(t)

	// AutoMigrate traffic models
	if err := s.db.AutoMigrate(
		&models.TrafficSource{},
		&models.TrafficRouteStat{},
		&models.TrafficDimStat{},
		&models.TrafficRecentEvent{},
	); err != nil {
		t.Fatalf("migrate traffic models: %v", err)
	}

	// 1. Uji ID tidak valid
	for _, badID := range []string{"abc", "0", "-5"} {
		req := httptest.NewRequest("DELETE", "/api/traffic/sources/"+badID, nil)
		req.Host = "localhost"
		req.Header.Set(localauth.Header, testLocalToken)
		resp, err := s.app.Test(req)
		if err != nil {
			t.Fatalf("req failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest && resp.StatusCode != fiber.StatusNotFound {
			t.Errorf("bad id %s got status %d, want 400 or 404", badID, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}

	// 2. Uji source tidak ditemukan
	{
		req := httptest.NewRequest("DELETE", "/api/traffic/sources/9999", nil)
		req.Host = "localhost"
		req.Header.Set(localauth.Header, testLocalToken)
		resp, err := s.app.Test(req)
		if err != nil {
			t.Fatalf("req failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusNotFound {
			t.Errorf("non-existent source got status %d, want 404", resp.StatusCode)
		}
		_ = resp.Body.Close()
	}

	// 3. Uji penghapusan sukses atomik (parent dan semua child records terhapus)
	src := models.TrafficSource{

		Name:      "test-source",
		Mode:      models.TrafficModeDirect,
		Enabled:   true,
	}
	if err := s.db.Create(&src).Error; err != nil {
		t.Fatalf("create traffic source: %v", err)
	}

	rStat := models.TrafficRouteStat{SourceID: src.ID, Method: "GET", Route: "/api/test", Count: 10}
	dStat := models.TrafficDimStat{SourceID: src.ID, Dim: "browser", Value: "Chrome", Count: 10}
	event := models.TrafficRecentEvent{SourceID: src.ID, Method: "GET", Route: "/api/test", Status: 200}

	s.db.Create(&rStat)
	s.db.Create(&dStat)
	s.db.Create(&event)

	deleteReq := httptest.NewRequest("DELETE", fmt.Sprintf("/api/traffic/sources/%d", src.ID), nil)
	deleteReq.Host = "localhost"
	deleteReq.Header.Set(localauth.Header, testLocalToken)

	resp, err := s.app.Test(deleteReq)
	if err != nil {
		t.Fatalf("req failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("got status %d, want 200, body: %s", resp.StatusCode, string(raw))
	}

	var res map[string]any
	raw, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(raw, &res)
	if deleted, ok := res["deleted"].(bool); !ok || !deleted {
		t.Errorf("expected deleted=true, got %v", res)
	}

	// Pastikan data benar-benar terhapus dari seluruh tabel
	var count int64
	s.db.Model(&models.TrafficSource{}).Where("id = ?", src.ID).Count(&count)
	if count != 0 {
		t.Errorf("TrafficSource masih ada: count=%d", count)
	}
	s.db.Model(&models.TrafficRouteStat{}).Where("source_id = ?", src.ID).Count(&count)
	if count != 0 {
		t.Errorf("TrafficRouteStat masih ada: count=%d", count)
	}
	s.db.Model(&models.TrafficDimStat{}).Where("source_id = ?", src.ID).Count(&count)
	if count != 0 {
		t.Errorf("TrafficDimStat masih ada: count=%d", count)
	}
	s.db.Model(&models.TrafficRecentEvent{}).Where("source_id = ?", src.ID).Count(&count)
	if count != 0 {
		t.Errorf("TrafficRecentEvent masih ada: count=%d", count)
	}
}

func BenchmarkDeleteTrafficSource(b *testing.B) {
	s := newGuardTestServer(&testing.T{})
	_ = s.db.AutoMigrate(
		&models.TrafficSource{},
		&models.TrafficRouteStat{},
		&models.TrafficDimStat{},
		&models.TrafficRecentEvent{},
	)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		src := models.TrafficSource{
	
			Name:      fmt.Sprintf("bench-source-%d", i),
			Mode:      models.TrafficModeDirect,
			Enabled:   true,
		}
		s.db.Create(&src)
		s.db.Create(&models.TrafficRouteStat{SourceID: src.ID, Method: "GET", Route: "/api/test", Count: 1})
		s.db.Create(&models.TrafficDimStat{SourceID: src.ID, Dim: "browser", Value: "Chrome", Count: 1})
		s.db.Create(&models.TrafficRecentEvent{SourceID: src.ID, Method: "GET", Route: "/api/test", Status: 200})

		deleteReq := httptest.NewRequest("DELETE", fmt.Sprintf("/api/traffic/sources/%d", src.ID), nil)
		deleteReq.Host = "localhost"
		deleteReq.Header.Set(localauth.Header, testLocalToken)
		b.StartTimer()

		resp, err := s.app.Test(deleteReq, -1)
		if err != nil {
			b.Fatalf("req failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			b.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		_ = resp.Body.Close()
	}
}
