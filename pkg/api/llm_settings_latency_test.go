package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/cloud"
	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
)

func setupLLMTestDB(s *Server) {
	_ = s.db.AutoMigrate(&models.LLMSettings{}, &models.LLMConfig{})
}

func TestLLMConfigs_LatencyAndCaching(t *testing.T) {
	s := newGuardTestServer(t)
	setupLLMTestDB(s)

	// Seed satu config
	cfg := models.LLMConfig{
		Provider:      models.ProviderSystemOne,
		Name:          "System One Default",
		URL:           "http://localhost:8750/v1/systemone",
		SelectedModel: "laya-multilingual",
		IsActive:      true,
	}
	if err := s.db.Create(&cfg).Error; err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	tok := map[string]string{localauth.Header: testLocalToken}

	// 1. First fetch -> DB fetch, check Cache-Control header
	resp1 := doReq(t, s, "GET", "/api/settings/llm/configs", "localhost", tok)
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp1.StatusCode)
	}
	cacheControl := resp1.Header.Get("Cache-Control")
	if cacheControl != "private, max-age=60" {
		t.Errorf("expected Cache-Control 'private, max-age=60', got %q", cacheControl)
	}

	var configs []models.LLMConfigDTO
	_ = json.NewDecoder(resp1.Body).Decode(&configs)
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}

	// 2. Second fetch -> Served from cachedConfigs (sub-millisecond)
	start := time.Now()
	resp2 := doReq(t, s, "GET", "/api/settings/llm/configs", "localhost", tok)
	elapsed := time.Since(start)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp2.StatusCode)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("expected cached read under 50ms, took %v", elapsed)
	}
}

func TestCreateLLMConfig_LatencyAndCacheControl(t *testing.T) {
	s := newGuardTestServer(t)
	setupLLMTestDB(s)
	tok := map[string]string{localauth.Header: testLocalToken}

	payload := map[string]interface{}{
		"provider":       models.ProviderStandardLLM,
		"name":           "OpenAI Secondary",
		"url":            "https://api.openai.com/v1",
		"key":            "sk-proj-test123456",
		"models":         []string{"gpt-4o", "gpt-4o-mini"},
		"selected_model": "gpt-4o-mini",
		"is_active":      false,
	}
	body, _ := json.Marshal(payload)

	start := time.Now()
	resp := doReqWithBody(t, s, "POST", "/api/settings/llm/configs", "localhost", string(body), tok)
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("expected Cache-Control 'no-store', got %q", cc)
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("create config took %v, exceeding 200ms SLO", elapsed)
	}

	// Verify GetAllConfigs cache was invalidated and now includes the new config
	respList := doReq(t, s, "GET", "/api/settings/llm/configs", "localhost", tok)
	var list []models.LLMConfigDTO
	_ = json.NewDecoder(respList.Body).Decode(&list)
	if len(list) < 1 {
		t.Errorf("expected list to have items after create")
	}
}

func TestRunCloudSecurityAudit_CachingAndLatency(t *testing.T) {
	s := newGuardTestServer(t)
	tok := map[string]string{localauth.Header: testLocalToken}

	// Seed cache directly to verify in-memory fast path and Cache-Control
	testBody := map[string]interface{}{
		"project_id": 49,
		"files":      []string{"main.go", "config.go"},
	}
	cacheKey := computeBodyHash(testBody)
	mockResult := &cloud.CloudAuditResponseDTO{
		SafetyScore: 95,
		Status:      "passed",
		Summary:     "Audit completed with zero critical issues",
	}
	setCachedCloudAudit(cacheKey, mockResult)

	bodyBytes, _ := json.Marshal(testBody)
	req := httptest.NewRequest("POST", "/api/vibeasik/security/cloud-audit", bytes.NewBuffer(bodyBytes))
	req.Host = "localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(localauth.Header, tok[localauth.Header])

	start := time.Now()
	resp, err := s.app.Test(req, -1)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from cached audit, got %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "private, max-age=60" {
		t.Errorf("expected Cache-Control 'private, max-age=60', got %q", cc)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("cached cloud audit took %v, expected < 50ms", elapsed)
	}

	var res cloud.CloudAuditResponseDTO
	_ = json.NewDecoder(resp.Body).Decode(&res)
	if res.SafetyScore != 95 || res.Status != "passed" {
		t.Errorf("unexpected audit response: %+v", res)
	}
}

func TestSetActiveProvider_LatencyOptimization(t *testing.T) {
	s := newGuardTestServer(t)
	setupLLMTestDB(s)
	tok := map[string]string{localauth.Header: testLocalToken}

	// Ensure initial settings exist
	initSettings := models.LLMSettings{
		ActiveProvider: models.ProviderSystemOne,
		SystemOneURL:   "http://localhost:8750/v1/systemone",
	}
	_ = s.db.Create(&initSettings)

	body := `{"provider":"standard_llm"}`
	start := time.Now()
	resp := doReqWithBody(t, s, "POST", "/api/settings/llm/active-provider", "localhost", body, tok)
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("expected Cache-Control 'no-store', got %q", cc)
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("setActiveProvider took %v, exceeding 200ms SLO", elapsed)
	}
}

func TestSelectActiveModel_LatencyOptimization(t *testing.T) {
	s := newGuardTestServer(t)
	setupLLMTestDB(s)
	tok := map[string]string{localauth.Header: testLocalToken}

	initSettings := models.LLMSettings{
		ActiveProvider:      models.ProviderStandardLLM,
		StandardLLMSelected: "gpt-4o-mini",
	}
	_ = s.db.Create(&initSettings)

	// 1. Ubah model ke gpt-4o
	body := `{"model":"gpt-4o"}`
	start := time.Now()
	resp1 := doReqWithBody(t, s, "POST", "/api/settings/llm/select-model", "localhost", body, tok)
	elapsed1 := time.Since(start)

	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp1.StatusCode)
	}
	if cc := resp1.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("expected Cache-Control 'no-store', got %q", cc)
	}
	if elapsed1 > 200*time.Millisecond {
		t.Errorf("selectActiveModel update took %v, exceeding 200ms SLO", elapsed1)
	}

	// 2. Model sama -> Fast No-Op path
	start2 := time.Now()
	resp2 := doReqWithBody(t, s, "POST", "/api/settings/llm/select-model", "localhost", body, tok)
	elapsed2 := time.Since(start2)

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on no-op select, got %d", resp2.StatusCode)
	}
	if elapsed2 > 50*time.Millisecond {
		t.Errorf("no-op select took %v, expected fast path < 50ms", elapsed2)
	}
}

func TestSettingsEndpoints_LatencySLO_Benchmark(t *testing.T) {
	s := newGuardTestServer(t)
	setupLLMTestDB(s)
	tok := map[string]string{localauth.Header: testLocalToken}

	initSettings := models.LLMSettings{
		ActiveProvider:      models.ProviderSystemOne,
		SystemOneURL:        "http://localhost:8750/v1/systemone",
		StandardLLMSelected: "gpt-4o-mini",
	}
	_ = s.db.Create(&initSettings)

	iterations := 50
	latencies := make([]time.Duration, 0, iterations)

	// Benchmark GET /api/settings/llm/configs
	for i := 0; i < iterations; i++ {
		start := time.Now()
		resp := doReq(t, s, "GET", "/api/settings/llm/configs", "localhost", tok)
		elapsed := time.Since(start)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		_ = resp.Body.Close()
		latencies = append(latencies, elapsed)
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	p50 := latencies[iterations*50/100]
	p95 := latencies[iterations*95/100]
	p99 := latencies[iterations*99/100]

	t.Logf("📊 GET /api/settings/llm/configs Latency (N=%d): p50=%v, p95=%v, p99=%v", iterations, p50, p95, p99)
	if p95 > 200*time.Millisecond {
		t.Errorf("p95 latency %v melebihi 200ms SLO", p95)
	}
}
