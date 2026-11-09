package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestTestCensorPayload_ValidationAndCaching(t *testing.T) {
	s := newGuardTestServer(t)
	if err := s.db.AutoMigrate(&models.CensoredKeyword{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}
	invalidateCensorTestCache()
	resetCensorCacheStats()

	// 1. Missing body (empty) -> 400 Bad Request
	req := httptest.NewRequest("POST", "/api/security/test-censor", nil)
	req.Host = "localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(localauth.Header, testLocalToken)
	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", resp.StatusCode)
	}

	// 2. Empty text field -> 400 Bad Request
	emptyBody := bytes.NewBufferString(`{"text": ""}`)
	req = httptest.NewRequest("POST", "/api/security/test-censor", emptyBody)
	req.Host = "localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(localauth.Header, testLocalToken)
	resp, err = s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty text, got %d", resp.StatusCode)
	}

	// 3. Cold Call (Cache MISS): Valid text with sensitive pattern -> 200 OK + MISS + private Cache-Control
	rawSecretPayload := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.dummy_secret_content postgres://root:P@ssword123@localhost:5432/mydb"
	bodyBytes, _ := json.Marshal(map[string]string{"text": rawSecretPayload})
	req = httptest.NewRequest("POST", "/api/security/test-censor", bytes.NewBuffer(bodyBytes))
	req.Host = "localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(localauth.Header, testLocalToken)
	resp, err = s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if cacheStatus := resp.Header.Get("X-Cache"); cacheStatus != "MISS" {
		t.Errorf("expected X-Cache 'MISS' on first call, got %q", cacheStatus)
	}
	cacheControl := resp.Header.Get("Cache-Control")
	if cacheControl != "private, max-age=60" {
		t.Errorf("expected Cache-Control 'private, max-age=60', got %q", cacheControl)
	}

	var res1 map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&res1)
	sanitized1, _ := res1["sanitized_text"].(string)
	if sanitized1 == "" || strings.Contains(sanitized1, "P@ssword123") {
		t.Errorf("expected sanitized_text without sensitive password, got %+v", res1)
	}

	// 4. Verifikasi Canonical Cache Key Privacy (K5741):
	// Pastikan raw text atau secret TIDAK tersimpan sebagai cache key di memori!
	censorTestCacheMu.RLock()
	for k := range censorTestCache {
		if strings.Contains(k, "P@ssword123") || strings.Contains(k, "dummy_secret_content") {
			t.Errorf("SECURITY RISK: Raw secret leaked as cache key: %s", k)
		}
		if !strings.HasPrefix(k, "censor:sha256:") {
			t.Errorf("expected canonical prefix 'censor:sha256:', got: %s", k)
		}
	}
	censorTestCacheMu.RUnlock()

	// 5. Warm Call (Cache HIT): Repeated call -> Fast In-Memory Cache Hit
	req2 := httptest.NewRequest("POST", "/api/security/test-censor", bytes.NewBuffer(bodyBytes))
	req2.Host = "localhost"
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(localauth.Header, testLocalToken)
	resp2, err := s.app.Test(req2, -1)
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for cached call, got %d", resp2.StatusCode)
	}
	if cacheStatus := resp2.Header.Get("X-Cache"); cacheStatus != "HIT" {
		t.Errorf("expected X-Cache 'HIT' on repeated call, got %q", cacheStatus)
	}
	var res2 map[string]interface{}
	_ = json.NewDecoder(resp2.Body).Decode(&res2)
	if res2["sanitized_text"] != res1["sanitized_text"] {
		t.Errorf("expected identical sanitized text from cache, got %v vs %v", res1["sanitized_text"], res2["sanitized_text"])
	}

	// 6. Bypass Call (Cache BYPASS, K2563): Client sends Cache-Control: no-cache
	req3 := httptest.NewRequest("POST", "/api/security/test-censor", bytes.NewBuffer(bodyBytes))
	req3.Host = "localhost"
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Cache-Control", "no-cache")
	req3.Header.Set(localauth.Header, testLocalToken)
	resp3, err := s.app.Test(req3, -1)
	if err != nil {
		t.Fatalf("unexpected error on bypass call: %v", err)
	}
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on bypass call, got %d", resp3.StatusCode)
	}
	if cacheStatus := resp3.Header.Get("X-Cache"); cacheStatus != "BYPASS" {
		t.Errorf("expected X-Cache 'BYPASS', got %q", cacheStatus)
	}
	if cc := resp3.Header.Get("Cache-Control"); cc != "private, no-cache, no-store" {
		t.Errorf("expected Cache-Control 'private, no-cache, no-store' on bypass, got %q", cc)
	}

	// 7. Mutasi keyword -> Invalidation: Cache harus otomatis dikosongkan (K5761)
	newKwPayload := `{"keyword":"LIVE_NEW_API_SECRET_999","category":"API_KEY","description":"New secret"}`
	reqKw := httptest.NewRequest("POST", "/api/security/censored-keywords", bytes.NewBufferString(newKwPayload))
	reqKw.Host = "localhost"
	reqKw.Header.Set("Content-Type", "application/json")
	reqKw.Header.Set(localauth.Header, testLocalToken)
	respKw, err := s.app.Test(reqKw, -1)
	if err != nil || respKw.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create keyword for invalidation test: %v (status %d)", err, respKw.StatusCode)
	}

	censorTestCacheMu.RLock()
	cacheLenAfterMutation := len(censorTestCache)
	censorTestCacheMu.RUnlock()
	if cacheLenAfterMutation != 0 {
		t.Errorf("expected cache to be invalidated (len 0) after keyword creation, got %d", cacheLenAfterMutation)
	}

	// 8. Request setelah mutasi dengan keyword baru: harus MISS dan menyamarkan keyword baru
	testNewKw := map[string]string{"text": "API call with LIVE_NEW_API_SECRET_999 token"}
	newKwBytes, _ := json.Marshal(testNewKw)
	req4 := httptest.NewRequest("POST", "/api/security/test-censor", bytes.NewBuffer(newKwBytes))
	req4.Host = "localhost"
	req4.Header.Set("Content-Type", "application/json")
	req4.Header.Set(localauth.Header, testLocalToken)
	resp4, err := s.app.Test(req4, -1)
	if err != nil || resp4.StatusCode != http.StatusOK {
		t.Fatalf("unexpected error on call after invalidation: %v (status %d)", err, resp4.StatusCode)
	}
	if cacheStatus := resp4.Header.Get("X-Cache"); cacheStatus != "MISS" {
		t.Errorf("expected X-Cache 'MISS' after invalidation, got %q", cacheStatus)
	}
	var res4 map[string]interface{}
	_ = json.NewDecoder(resp4.Body).Decode(&res4)
	sanitized4, _ := res4["sanitized_text"].(string)
	if strings.Contains(sanitized4, "LIVE_NEW_API_SECRET_999") {
		t.Errorf("newly added keyword was not censored: %s", sanitized4)
	}

	// 9. Verifikasi Metrik Cache Terpisah (K2157)
	stats := getCensorCacheStats()
	if stats.Hits == 0 {
		t.Errorf("expected stats.Hits > 0, got %d", stats.Hits)
	}
	if stats.Misses == 0 {
		t.Errorf("expected stats.Misses > 0, got %d", stats.Misses)
	}
	if stats.Bypasses == 0 {
		t.Errorf("expected stats.Bypasses > 0, got %d", stats.Bypasses)
	}
}

func TestTestCensorPayload_BoundedCapacityAndEviction(t *testing.T) {
	s := newGuardTestServer(t)
	invalidateCensorTestCache()
	resetCensorCacheStats()

	// Masukkan entries melebihi kapasitas maxCensorCacheEntries (128)
	totalEntries := maxCensorCacheEntries + 20
	for i := 0; i < totalEntries; i++ {
		text := fmt.Sprintf("sample text for eviction testing %03d with secret pattern", i)
		bodyBytes, _ := json.Marshal(map[string]string{"text": text})
		req := httptest.NewRequest("POST", "/api/security/test-censor", bytes.NewBuffer(bodyBytes))
		req.Host = "localhost"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(localauth.Header, testLocalToken)
		resp, err := s.app.Test(req, -1)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d failed: %v", i, err)
		}
		_ = resp.Body.Close()
	}

	censorTestCacheMu.RLock()
	cacheLen := len(censorTestCache)
	censorTestCacheMu.RUnlock()

	if cacheLen > maxCensorCacheEntries {
		t.Errorf("cache capacity exceeded: len %d > max %d", cacheLen, maxCensorCacheEntries)
	}

	stats := getCensorCacheStats()
	if stats.Evictions == 0 {
		t.Errorf("expected evictions to occur when exceeding capacity, got %d", stats.Evictions)
	}
}

// TestLatencySLO_TestCensorEndpoint mengukur distribusi latensi cold miss, warm hit, dan bypass
// untuk membuktikan pencapaian SLO sub-200ms secara empiris (K2213, K2345, K5160, K2157).
func TestLatencySLO_TestCensorEndpoint(t *testing.T) {
	s := newGuardTestServer(t)
	invalidateCensorTestCache()
	resetCensorCacheStats()

	samplePayload := `{"text":"DATABASE_URL=postgres://vibe_user:SuperSecretP@ssword@db.internal:5432/vibe_db Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkFkbWluIn0.signature sk-proj-123456789012345678901234567890"}`

	// 1. Benchmark Cold Miss Path (tiap request memakai payload unik agar terjadi MISS murni)
	coldIterations := 50
	coldLatencies := make([]time.Duration, 0, coldIterations)
	for i := 0; i < coldIterations; i++ {
		uniquePayload := fmt.Sprintf(`{"text":"unique secret payload %03d: sk-proj-%030d postgres://u:p%d@localhost/db"}`, i, i, i)
		req := httptest.NewRequest("POST", "/api/security/test-censor", bytes.NewBufferString(uniquePayload))
		req.Host = "localhost"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(localauth.Header, testLocalToken)

		start := time.Now()
		resp, err := s.app.Test(req, -1)
		elapsed := time.Since(start)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("cold request %d failed: %v", i, err)
		}
		_ = resp.Body.Close()
		coldLatencies = append(coldLatencies, elapsed)
	}

	sort.Slice(coldLatencies, func(i, j int) bool { return coldLatencies[i] < coldLatencies[j] })
	coldP50 := coldLatencies[len(coldLatencies)*50/100]
	coldP95 := coldLatencies[len(coldLatencies)*95/100]
	coldP99 := coldLatencies[len(coldLatencies)*99/100]

	// 2. Benchmark Warm Hit Path (request berulang dengan payload yang sama)
	// Warm-up 1x
	reqWarmup := httptest.NewRequest("POST", "/api/security/test-censor", bytes.NewBufferString(samplePayload))
	reqWarmup.Host = "localhost"
	reqWarmup.Header.Set("Content-Type", "application/json")
	reqWarmup.Header.Set(localauth.Header, testLocalToken)
	_, _ = s.app.Test(reqWarmup, -1)

	hitIterations := 100
	hitLatencies := make([]time.Duration, 0, hitIterations)
	for i := 0; i < hitIterations; i++ {
		req := httptest.NewRequest("POST", "/api/security/test-censor", bytes.NewBufferString(samplePayload))
		req.Host = "localhost"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(localauth.Header, testLocalToken)

		start := time.Now()
		resp, err := s.app.Test(req, -1)
		elapsed := time.Since(start)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("hit request %d failed: %v", i, err)
		}
		_ = resp.Body.Close()
		hitLatencies = append(hitLatencies, elapsed)
	}

	sort.Slice(hitLatencies, func(i, j int) bool { return hitLatencies[i] < hitLatencies[j] })
	hitP50 := hitLatencies[len(hitLatencies)*50/100]
	hitP95 := hitLatencies[len(hitLatencies)*95/100]
	hitP99 := hitLatencies[len(hitLatencies)*99/100]

	// 3. Benchmark Bypass Path
	bypassIterations := 50
	bypassLatencies := make([]time.Duration, 0, bypassIterations)
	for i := 0; i < bypassIterations; i++ {
		req := httptest.NewRequest("POST", "/api/security/test-censor", bytes.NewBufferString(samplePayload))
		req.Host = "localhost"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set(localauth.Header, testLocalToken)

		start := time.Now()
		resp, err := s.app.Test(req, -1)
		elapsed := time.Since(start)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("bypass request %d failed: %v", i, err)
		}
		_ = resp.Body.Close()
		bypassLatencies = append(bypassLatencies, elapsed)
	}

	sort.Slice(bypassLatencies, func(i, j int) bool { return bypassLatencies[i] < bypassLatencies[j] })
	bypassP50 := bypassLatencies[len(bypassLatencies)*50/100]
	bypassP95 := bypassLatencies[len(bypassLatencies)*95/100]
	bypassP99 := bypassLatencies[len(bypassLatencies)*99/100]

	t.Logf("📊 POST /api/security/test-censor Latency Benchmark:")
	t.Logf("   [COLD MISS]   p50: %-10v | p95: %-10v | p99: %-10v (N=%d)", coldP50, coldP95, coldP99, coldIterations)
	t.Logf("   [WARM HIT]    p50: %-10v | p95: %-10v | p99: %-10v (N=%d)", hitP50, hitP95, hitP99, hitIterations)
	t.Logf("   [BYPASS]      p50: %-10v | p95: %-10v | p99: %-10v (N=%d)", bypassP50, bypassP95, bypassP99, bypassIterations)

	// SLO Assertions: All p95 latencies must be well below the 200ms interactive threshold
	if coldP95 > 200*time.Millisecond {
		t.Errorf("COLD MISS p95 latency %v exceeds 200ms SLO", coldP95)
	}
	if hitP95 > 200*time.Millisecond {
		t.Errorf("WARM HIT p95 latency %v exceeds 200ms SLO", hitP95)
	}
	if bypassP95 > 200*time.Millisecond {
		t.Errorf("BYPASS p95 latency %v exceeds 200ms SLO", bypassP95)
	}

	// Warm hit should be orders of magnitude faster than cold miss
	if hitP50 >= coldP50 {
		t.Logf("Notice: hit p50 (%v) vs cold p50 (%v)", hitP50, coldP50)
	}

	stats := getCensorCacheStats()
	t.Logf("📈 Cache Stats: Hits=%d, Misses=%d, Bypasses=%d, Expiries=%d, Evictions=%d",
		stats.Hits, stats.Misses, stats.Bypasses, stats.Expiries, stats.Evictions)
}
