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

// Sub todo #3521: s.injectVibeAITemplate (POST /api/vibeasik/ai-templates/inject)
func TestSubTodo3521_InjectVibeAITemplate_LatencyAndCaching(t *testing.T) {
	s := newGuardTestServer(t)
	tok := map[string]string{localauth.Header: testLocalToken}
	tmpDir, err := os.MkdirTemp("", "vibeasik_test_inject_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockTemplates := []cloud.AITemplateDTO{
		{
			ID:          "fast-assistant",
			Name:        "Fast Assistant",
			Description: "Assists with fast injection",
			Files: []cloud.DeployFileDTO{
				{
					Filename: "ai_helper.go",
					Content:  "package main\n\n// injected helper",
				},
			},
		},
	}
	setCachedAITemplates(mockTemplates)

	payload := map[string]interface{}{
		"component_id": 1,
		"template_id":  "fast-assistant",
		"target_path":  tmpDir,
	}
	bodyBytes, _ := json.Marshal(payload)

	start := time.Now()
	resp := doReqWithBody(t, s, "POST", "/api/vibeasik/ai-templates/inject", "localhost", string(bodyBytes), tok)
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control 'no-cache', got %q", cc)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("cached injectVibeAITemplate took %v, expected < 50ms", elapsed)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode inject response: %v", err)
	}
	if res["success"] != true || res["template_name"] != "Fast Assistant" {
		t.Errorf("unexpected response: %+v", res)
	}
	if _, err := os.Stat(tmpDir + "/ai_helper.go"); os.IsNotExist(err) {
		t.Errorf("expected injected file ai_helper.go to exist in %s", tmpDir)
	}
}

// Sub todo #3522: s.listTrafficSources (GET /api/traffic/sources)
func TestSubTodo3522_ListTrafficSources_LatencyAndBatching(t *testing.T) {
	s := newGuardTestServer(t)
	_ = s.db.AutoMigrate(&models.Project{}, &models.Component{}, &models.TrafficSource{})

	p := models.Project{Name: "Traffic Test Project", Slug: "traffic-test-project"}
	s.db.Create(&p)
	comp := models.Component{ProjectID: p.ID, Name: "Traffic Test Comp", Role: "backend"}
	s.db.Create(&comp)

	// Create multiple sources to verify N+1 batch query behavior
	for i := 0; i < 5; i++ {
		src := models.TrafficSource{
			Name:        "Source Batch",
			ProjectID:   &p.ID,
			ComponentID: &comp.ID,
			Mode:        models.TrafficModeDirect,
			Enabled:     true,
		}
		s.db.Create(&src)
	}

	tok := map[string]string{localauth.Header: testLocalToken}
	start := time.Now()
	resp := doReq(t, s, "GET", "/api/traffic/sources", "localhost", tok)
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "private, max-age=60" {
		t.Errorf("expected Cache-Control 'private, max-age=60', got %q", cc)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("listTrafficSources took %v, expected < 50ms", elapsed)
	}

	var list []trafficSourceView
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(list) < 5 {
		t.Fatalf("expected at least 5 traffic sources, got %d", len(list))
	}
	if list[0].ProjectName != "Traffic Test Project" || list[0].ComponentName != "Traffic Test Comp" {
		t.Errorf("batch resolved relations mismatch: ProjectName=%q, ComponentName=%q", list[0].ProjectName, list[0].ComponentName)
	}
}

// Sub todo #3523: s.createTrafficSource (POST /api/traffic/sources)
func TestSubTodo3523_CreateTrafficSource_LatencyAndHeaders(t *testing.T) {
	s := newGuardTestServer(t)
	_ = s.db.AutoMigrate(&models.Project{}, &models.Component{}, &models.TrafficSource{})

	p := models.Project{Name: "Source Project", Slug: "source-project"}
	s.db.Create(&p)
	comp := models.Component{ProjectID: p.ID, Name: "Source Component", Role: "backend"}
	s.db.Create(&comp)

	tok := map[string]string{localauth.Header: testLocalToken}
	payload := map[string]interface{}{
		"component_id": comp.ID,
		"mode":         models.TrafficModeDirect,
	}
	bodyBytes, _ := json.Marshal(payload)

	start := time.Now()
	resp := doReqWithBody(t, s, "POST", "/api/traffic/sources", "localhost", string(bodyBytes), tok)
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control 'no-cache', got %q", cc)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("createTrafficSource took %v, expected < 50ms", elapsed)
	}

	var created models.TrafficSource
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created response: %v", err)
	}
	if created.Name != "Source Component" || created.ProjectID == nil || *created.ProjectID != p.ID {
		t.Errorf("unexpected auto-resolved source fields: name=%q, project_id=%v", created.Name, created.ProjectID)
	}
}

// Sub todo #3524: s.runReaAnalysis (POST /api/rea/analyze)
func TestSubTodo3524_RunReaAnalysis_LatencyAndCaching(t *testing.T) {
	s := newGuardTestServer(t)
	tok := map[string]string{localauth.Header: testLocalToken}

	cacheKey := "javascript:/home/test/app"
	mockResult := map[string]interface{}{
		"analyzed": true,
		"score":    95,
	}
	setCachedReaAnalysis(cacheKey, map[string]interface{}{
		"success":       true,
		"analysis_type": "javascript",
		"target_path":   "/home/test/app",
		"duration":      "15ms",
		"result":        mockResult,
	})

	payload := map[string]interface{}{
		"target_path":   "/home/test/app",
		"analysis_type": "javascript",
	}
	bodyBytes, _ := json.Marshal(payload)

	start := time.Now()
	resp := doReqWithBody(t, s, "POST", "/api/rea/analyze", "localhost", string(bodyBytes), tok)
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "private, max-age=60" {
		t.Errorf("expected Cache-Control 'private, max-age=60', got %q", cc)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("cached runReaAnalysis took %v, expected < 50ms", elapsed)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode rea response: %v", err)
	}
	if res["success"] != true || res["duration"] != "15ms" {
		t.Errorf("unexpected rea cached result: %+v", res)
	}
}

