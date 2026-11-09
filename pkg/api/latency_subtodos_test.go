package api

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/cloud"
	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
)

// Sub todo #3512: s.ensureUniquePorts (POST /api/ports/ensure-unique)
func TestSubTodo3512_EnsureUniquePorts_LatencyAndHeaders(t *testing.T) {
	s := newGuardTestServer(t)
	_ = s.db.AutoMigrate(&models.Project{}, &models.Component{})

	tok := map[string]string{localauth.Header: testLocalToken}
	start := time.Now()
	resp := doReqWithBody(t, s, "POST", "/api/ports/ensure-unique", "localhost", "{}", tok)
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control 'no-cache', got %q", cc)
	}

	if elapsed > 200*time.Millisecond {
		t.Errorf("s.ensureUniquePorts took %v, exceeding 200ms SLO", elapsed)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := res["updated_count"]; !ok {
		t.Errorf("missing updated_count in response")
	}
}

// Sub todo #3516: s.getWorkspaceSettings (GET /api/settings/workspace)
func TestSubTodo3516_GetWorkspaceSettings_LatencyAndCaching(t *testing.T) {
	s := newGuardTestServer(t)
	_ = s.db.AutoMigrate(&models.WorkspaceSettings{})

	tok := map[string]string{localauth.Header: testLocalToken}

	// 1. First fetch -> populates in-memory cache
	resp1 := doReq(t, s, "GET", "/api/settings/workspace", "localhost", tok)
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp1.StatusCode)
	}
	if cc := resp1.Header.Get("Cache-Control"); cc != "private, max-age=60" {
		t.Errorf("expected Cache-Control 'private, max-age=60', got %q", cc)
	}

	// 2. Second fetch -> fast path in-memory cache (<50ms)
	start := time.Now()
	resp2 := doReq(t, s, "GET", "/api/settings/workspace", "localhost", tok)
	elapsed := time.Since(start)

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp2.StatusCode)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("cached getWorkspaceSettings took %v, expected < 50ms", elapsed)
	}

	var ws models.WorkspaceSettings
	if err := json.NewDecoder(resp2.Body).Decode(&ws); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if ws.DefaultProjectsRoot == "" {
		t.Errorf("expected non-empty default_projects_root")
	}
}

// Sub todo #3517: s.updateWorkspaceSettings (PUT /api/settings/workspace)
func TestSubTodo3517_UpdateWorkspaceSettings_LatencyAndInvalidation(t *testing.T) {
	s := newGuardTestServer(t)
	_ = s.db.AutoMigrate(&models.WorkspaceSettings{})

	tok := map[string]string{localauth.Header: testLocalToken}
	tmpDir := os.TempDir()

	body := `{"default_projects_root":"` + tmpDir + `"}`
	start := time.Now()
	resp := doReqWithBody(t, s, "PUT", "/api/settings/workspace", "localhost", body, tok)
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control 'no-cache', got %q", cc)
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("updateWorkspaceSettings took %v, exceeding 200ms SLO", elapsed)
	}

	// Verify subsequent GET reflects the updated root immediately from cache
	respGet := doReq(t, s, "GET", "/api/settings/workspace", "localhost", tok)
	var ws models.WorkspaceSettings
	_ = json.NewDecoder(respGet.Body).Decode(&ws)
	if ws.DefaultProjectsRoot != tmpDir {
		t.Errorf("expected default_projects_root %q, got %q", tmpDir, ws.DefaultProjectsRoot)
	}
}

// Sub todo #3518: s.generateVibeDeployPlan (POST /api/vibeasik/deploy/generate-plan)
func TestSubTodo3518_GenerateVibeDeployPlan_LatencyAndCaching(t *testing.T) {
	s := newGuardTestServer(t)
	tok := map[string]string{localauth.Header: testLocalToken}

	testPayload := map[string]interface{}{
		"project_id": 49,
		"target":     "docker-compose",
	}
	cacheKey := computeBodyHash(testPayload)
	mockPlan := &cloud.DeployPlanResponseDTO{
		TargetPlatform: "docker",
		Architecture:   "monolith",
		Notes:          "Test deploy plan",
	}
	setCachedDeployPlan(cacheKey, mockPlan)

	bodyBytes, _ := json.Marshal(testPayload)
	start := time.Now()
	resp := doReqWithBody(t, s, "POST", "/api/vibeasik/deploy/generate-plan", "localhost", string(bodyBytes), tok)
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "private, max-age=60" {
		t.Errorf("expected Cache-Control 'private, max-age=60', got %q", cc)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("cached generateVibeDeployPlan took %v, expected < 50ms", elapsed)
	}

	var plan cloud.DeployPlanResponseDTO
	if err := json.NewDecoder(resp.Body).Decode(&plan); err != nil {
		t.Fatalf("decode plan response: %v", err)
	}
	if plan.TargetPlatform != "docker" || plan.Architecture != "monolith" {
		t.Errorf("unexpected deploy plan: %+v", plan)
	}
}

// Sub todo #3520: s.getVibeAITemplates (GET /api/vibeasik/ai-templates)
func TestSubTodo3520_GetVibeAITemplates_LatencyAndCaching(t *testing.T) {
	s := newGuardTestServer(t)
	tok := map[string]string{localauth.Header: testLocalToken}

	mockTemplates := []cloud.AITemplateDTO{
		{
			ID:          "assistant-1",
			Name:        "Code Assistant",
			Description: "Assists with code generation",
		},
	}
	setCachedAITemplates(mockTemplates)

	start := time.Now()
	resp := doReq(t, s, "GET", "/api/vibeasik/ai-templates", "localhost", tok)
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "public, max-age=300" {
		t.Errorf("expected Cache-Control 'public, max-age=300', got %q", cc)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("cached getVibeAITemplates took %v, expected < 50ms", elapsed)
	}

	var res struct {
		Templates []cloud.AITemplateDTO `json:"templates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode templates response: %v", err)
	}
	if len(res.Templates) != 1 || res.Templates[0].ID != "assistant-1" {
		t.Errorf("unexpected templates response: %+v", res)
	}
}
