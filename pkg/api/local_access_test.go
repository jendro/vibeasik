package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
)

func doReq(t *testing.T, s *Server, method, path, host string, hdr map[string]string) *http.Response {
	return doReqWithBody(t, s, method, path, host, "", hdr)
}

func doReqWithBody(t *testing.T, s *Server, method, path, host, body string, hdr map[string]string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Host = host
	if body != "" && (hdr == nil || hdr["Content-Type"] == "") {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func TestLocalAccessRequiresToken(t *testing.T) {
	s := newGuardTestServer(t)
	tok := map[string]string{localauth.Header: testLocalToken}

	cases := []struct {
		name, method, path, host string
		hdr                      map[string]string
		want                     int
	}{
		{"health tanpa token", "GET", "/health", "localhost", nil, 200},
		{"api tanpa token", "GET", "/api/projects", "localhost", nil, 401},
		{"mcp tanpa token", "POST", "/mcp", "127.0.0.1:8765", nil, 401},
		{"api token salah", "GET", "/api/projects", "localhost", map[string]string{localauth.Header: "salah"}, 401},
		{"api dengan header", "GET", "/api/projects", "localhost", tok, 200},
		{"api dengan bearer", "GET", "/api/projects", "127.0.0.1", map[string]string{"Authorization": "Bearer " + testLocalToken}, 200},
		{"api dengan cookie", "GET", "/api/projects", "localhost", map[string]string{"Cookie": localauth.Cookie + "=" + testLocalToken}, 200},
		{"dns rebinding", "GET", "/api/projects", "evil.example.com", tok, 403},
		{"ip lan tanpa mode lan", "GET", "/api/projects", "192.168.1.10:8765", tok, 403},
		{"traffic ingest berkunci sendiri", "POST", "/api/traffic/ingest", "localhost", nil, 401},
		{"halaman studio tanpa token", "GET", "/cockpit", "localhost", nil, 401},
	}
	for _, c := range cases {
		resp := doReq(t, s, c.method, c.path, c.host, c.hdr)
		if resp.StatusCode != c.want {
			t.Errorf("%s: %s %s host=%s → %d, want %d", c.name, c.method, c.path, c.host, resp.StatusCode, c.want)
		}
	}
	// ingest menolak karena kunci sumber kosong (handler sendiri), bukan karena token lokal
	resp := doReq(t, s, "POST", "/api/traffic/ingest", "localhost", nil)
	if resp.Header.Get("Content-Type") == "" {
		t.Error("ingest must answer from its own handler")
	}

	if resp := doReq(t, s, "GET", "/_nuxt/app.js", "localhost", nil); resp.StatusCode == http.StatusUnauthorized {
		t.Error("static Studio assets must load without token")
	}
	if resp := doReq(t, s, "GET", "/api/projects", "[::1]:8765", tok); resp.StatusCode != 200 {
		t.Errorf("IPv6 loopback host must be allowed, got %d", resp.StatusCode)
	}

	s.SetLAN(true)
	if resp := doReq(t, s, "GET", "/api/projects", "192.168.1.10:8765", tok); resp.StatusCode != 200 {
		t.Errorf("LAN mode must accept private IP host, got %d", resp.StatusCode)
	}
	if resp := doReq(t, s, "GET", "/api/projects", "evil.example.com", tok); resp.StatusCode != 403 {
		t.Errorf("LAN mode must still reject foreign host names, got %d", resp.StatusCode)
	}
}

func TestAuthHandshakeSetsCookie(t *testing.T) {
	s := newGuardTestServer(t)
	resp := doReq(t, s, "GET", "/auth?t="+testLocalToken+"&next=/todos", "localhost", nil)
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/todos" {
		t.Fatalf("handshake: %d → %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	ck := resp.Header.Get("Set-Cookie")
	for _, want := range []string{localauth.Cookie + "=" + testLocalToken, "HttpOnly", "SameSite=Strict"} {
		if !strings.Contains(ck, want) {
			t.Errorf("cookie %q missing %q", ck, want)
		}
	}
	// Open redirect ditolak
	resp = doReq(t, s, "GET", "/auth?t="+testLocalToken+"&next=//evil.example.com", "localhost", nil)
	if loc := resp.Header.Get("Location"); loc != "/cockpit" {
		t.Errorf("open redirect: got %q", loc)
	}
	// Token salah: tidak ada cookie
	resp = doReq(t, s, "GET", "/auth?t=salah", "localhost", nil)
	if resp.StatusCode != http.StatusUnauthorized || resp.Header.Get("Set-Cookie") != "" {
		t.Errorf("bad token handshake: %d %q", resp.StatusCode, resp.Header.Get("Set-Cookie"))
	}
}

func TestCORSOnlyStudioDevOrigin(t *testing.T) {
	s := newGuardTestServer(t)
	resp := doReq(t, s, "GET", "/health", "localhost", map[string]string{"Origin": "https://evil.example.com"})
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("foreign origin must not be allowed, got %q", got)
	}
	resp = doReq(t, s, "GET", "/health", "localhost", map[string]string{"Origin": "http://localhost:8760"})
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:8760" {
		t.Errorf("studio dev origin must be allowed, got %q", got)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("security headers missing")
	}
}

// TestLocalAccess_MutationFailClosedSecurityBoundary memverifikasi batas keamanan perimeter localAccess (K4453, K4213):
// Request mutasi yang dikirim dari remote IP non-loopback (192.168.1.50, 10.0.0.5, dsb.)
// tanpa auth token HARUS ditolak HTTP 403 Forbidden ("host_forbidden") sebelum mencapai handler.
func TestLocalAccess_MutationFailClosedSecurityBoundary(t *testing.T) {
	s := newGuardTestServer(t)

	mutationRoutes := []struct {
		method string
		path   string
		body   string
	}{
		// Diagnostics & Findings
		{"POST", "/api/projects/1/diagnostics/ignore-finding/finding-1", `{"finding_id":"finding-1"}`},
		{"POST", "/api/projects/1/diagnostics/delete-finding", `{"finding_id":"finding-1"}`},
		// Knowledge
		{"POST", "/api/knowledge/nodes", `{"title":"Aturan Keamanan","content":"Fail-closed boundary","kind":"rule","domain":"security"}`},
		{"POST", "/api/knowledge/recall", `{"query":"security test"}`},
		{"POST", "/api/knowledge/curate", `{}`},
		{"POST", "/api/knowledge/gaps/1/dismiss", `{}`},
		// Projects Lifecycle & Mutasi
		{"POST", "/api/projects", `{"name":"Project Luar","root_path":"/tmp/test"}`},
		{"PUT", "/api/projects/1", `{"name":"Project Update"}`},
		{"DELETE", "/api/projects/1", `{}`},
		{"POST", "/api/projects/1/undeprecate", `{}`},
		{"POST", "/api/projects/1/hold", `{}`},
		{"POST", "/api/projects/1/done", `{}`},
		{"POST", "/api/projects/1/activate", `{}`},
		{"POST", "/api/projects/1/merge", `{}`},
		{"POST", "/api/projects/1/summary", `{}`},
		{"POST", "/api/projects/1/code-map/build", `{}`},
		{"POST", "/api/projects/1/code-map/export", `{}`},
		{"POST", "/api/projects/1/link-workspace", `{"workspace_path":"/tmp"}`},
		{"POST", "/api/projects/1/init-vibeasik", `{}`},
		{"POST", "/api/projects/1/consolidate", `{}`},
		{"POST", "/api/projects/1/components", `{"name":"worker","role":"worker"}`},
		{"POST", "/api/projects/1/run", `{}`},
		{"POST", "/api/projects/1/stop", `{}`},
		{"POST", "/api/projects/1/ensure-unique-ports", `{}`},
		// Components & Routes
		{"PUT", "/api/components/1", `{"name":"api-updated"}`},
		{"DELETE", "/api/components/1", `{}`},
		{"POST", "/api/components/1/routes/sync", `{}`},
		{"PUT", "/api/components/1/routes/1", `{"path":"/new"}`},
		{"DELETE", "/api/components/1/routes/1", `{}`},
		{"POST", "/api/components/1/diagnose", `{}`},
		{"POST", "/api/components/1/ports", `{"port":3000}`},
		{"POST", "/api/components/1/run", `{}`},
		{"POST", "/api/components/1/stop", `{}`},
		{"POST", "/api/ports/ensure-unique", `{}`},
		// Security & Censor
		{"POST", "/api/security/test-censor", `{"text":"test"}`},
		{"POST", "/api/security/censored-keywords", `{"keyword":"secret"}`},
		{"PUT", "/api/security/censored-keywords/1", `{"keyword":"secret2"}`},
		{"DELETE", "/api/security/censored-keywords/1", `{}`},
		// LLM Settings
		{"PUT", "/api/settings/llm", `{"provider":"gemini"}`},
		{"POST", "/api/settings/llm/active-provider", `{"provider":"gemini"}`},
		{"POST", "/api/settings/llm/select-model", `{"model":"gemini-1.5-flash"}`},
		{"POST", "/api/settings/llm/configs", `{"name":"custom","api_key":"secret"}`},
		{"PUT", "/api/settings/llm/configs/1", `{"name":"custom2"}`},
		{"DELETE", "/api/settings/llm/configs/1", `{}`},
		{"POST", "/api/settings/llm/configs/1/activate", `{}`},
		{"PUT", "/api/settings/llm/configs/1/vision", `{"vision":true}`},
		{"PUT", "/api/settings/llm/configs/1/alias", `{"alias":"fast"}`},
		{"PUT", "/api/settings/workspace", `{"path":"/tmp"}`},
		// AI Templates & Cloud Audit
		{"POST", "/api/vibeasik/security/cloud-audit", `{}`},
		{"POST", "/api/vibeasik/ai-templates/inject", `{"template_id":"auth"}`},
		{"POST", "/api/vibeasik/deploy/generate-plan", `{}`},
		// Agents
		{"POST", "/api/agents/execute", `{"command":"ls"}`},
		{"POST", "/api/agents/custom", `{"name":"custom-bot"}`},
		{"DELETE", "/api/agents/custom/test-agent", `{}`},
		// Traffic
		{"POST", "/api/traffic/sources", `{"name":"source-1"}`},
		{"PUT", "/api/traffic/sources/1", `{"name":"source-updated"}`},
		{"DELETE", "/api/traffic/sources/1", `{}`},
		{"POST", "/api/traffic/sources/1/sync", `{}`},
		{"POST", "/api/traffic/sources/1/rotate-key", `{}`},
		{"POST", "/api/traffic/sources/1/sdk/install", `{}`},
		{"POST", "/api/traffic/sources/1/todo", `{"title":"Optimize"}`},
		// REA & Organizer & Todos
		{"POST", "/api/rea/analyze", `{}`},
		{"POST", "/api/organizer/plan", `{}`},
		{"POST", "/api/organizer/execute", `{}`},
		{"POST", "/api/todos", `{"title":"Todo Ilegal","project_id":1}`},
		{"DELETE", "/api/todos/1", `{}`},
		// MCP Endpoints
		{"POST", "/mcp", `{"jsonrpc":"2.0","method":"tools/list"}`},
		{"DELETE", "/mcp", `{}`},
	}

	remoteHosts := []string{
		"192.168.1.50",
		"192.168.1.50:8765",
		"10.0.0.5",
		"10.0.0.5:8765",
		"172.16.1.100:8765",
		"evil.attacker.com",
	}

	for _, tc := range mutationRoutes {
		// Uji 1: Localhost tanpa token auth -> HARUS 401 Unauthorized
		respLocal := doReqWithBody(t, s, tc.method, tc.path, "localhost", tc.body, nil)
		if respLocal.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s host=localhost (tanpa token) → %d, want 401 Unauthorized", tc.method, tc.path, respLocal.StatusCode)
		}
		_ = respLocal.Body.Close()

		for _, host := range remoteHosts {
			// Uji 2: Remote non-loopback tanpa auth token -> HARUS 403 Forbidden
			resp := doReqWithBody(t, s, tc.method, tc.path, host, tc.body, nil)
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("%s %s host=%s (tanpa token) → %d, want 403 Forbidden", tc.method, tc.path, host, resp.StatusCode)
			}
			var bodyMap map[string]interface{}
			_ = json.NewDecoder(resp.Body).Decode(&bodyMap)
			_ = resp.Body.Close()
			if bodyMap["error"] != "host_forbidden" {
				t.Errorf("%s %s host=%s: error message = %v, want host_forbidden", tc.method, tc.path, host, bodyMap["error"])
			}

			// Uji 3: Remote non-loopback dengan auth token (mode LAN nonaktif) -> TETAP 403 Forbidden
			respWithToken := doReqWithBody(t, s, tc.method, tc.path, host, tc.body, map[string]string{
				localauth.Header: testLocalToken,
			})
			if respWithToken.StatusCode != http.StatusForbidden {
				t.Errorf("%s %s host=%s (dengan token) → %d, want 403 Forbidden", tc.method, tc.path, host, respWithToken.StatusCode)
			}
			_ = respWithToken.Body.Close()
		}
	}
}

// TestLocalAccess_MutationNegativeAuthorizationAndStateImmutability membuktikan negative authorization dan state immutability (K2296, K4453, K35):
// Ketika request mutasi ditolak oleh perimeter keamanan (karena remote IP ilegal, token hilang, atau token salah),
// handler sama sekali TIDAK dipanggil dan state database (data temuan diagnostik & tabel knowledge) terbukti TIDAK berubah.
func TestLocalAccess_MutationNegativeAuthorizationAndStateImmutability(t *testing.T) {
	s := newGuardTestServer(t)

	// Fixture data diagnostik dengan 1 temuan awal berstatus "open"
	findingKey := "SEC-004-TEST"
	finding := models.Finding{
		ID:         findingKey,
		RuleID:     "SEC-004",
		Title:      "Broken Object Level Authorization",
		TargetFile: "pkg/api/server.go",
		Line:       237,
		Status:     models.FindingStatusOpen,
	}
	findingsJSON, err := json.Marshal([]models.Finding{finding})
	if err != nil {
		t.Fatalf("marshal finding: %v", err)
	}
	initialFindingsStr := string(findingsJSON)
	initialScore := 80

	diag := models.Diagnostic{
		ProjectID:    1,
		Score:        initialScore,
		FindingsJSON: initialFindingsStr,
	}
	if err := s.db.Create(&diag).Error; err != nil {
		t.Fatalf("create diagnostic fixture: %v", err)
	}

	// Verifikasi state awal di SQLite
	var diagBefore models.Diagnostic
	if err := s.db.Where("project_id = ?", 1).First(&diagBefore).Error; err != nil {
		t.Fatalf("query diagnostic: %v", err)
	}
	if diagBefore.Score != initialScore || diagBefore.FindingsJSON != initialFindingsStr {
		t.Fatalf("state awal tidak cocok: score=%d, findings=%s", diagBefore.Score, diagBefore.FindingsJSON)
	}

	var kCountInitial int64
	s.db.Model(&models.KnowledgeNode{}).Count(&kCountInitial)
	if kCountInitial != 0 {
		t.Fatalf("expected 0 knowledge nodes initially, got %d", kCountInitial)
	}

	// 1. Serangan mutasi dari remote IP non-loopback tanpa auth token
	resp := doReqWithBody(t, s, "POST", "/api/projects/1/diagnostics/ignore-finding/"+findingKey, "192.168.1.50:8765", `{"finding_id":"`+findingKey+`"}`, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("remote non-loopback ignore-finding: got %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Assert Immutability 1: State diagnostic & finding tidak berubah setelah penolakan 403
	var diagAfterRemote models.Diagnostic
	if err := s.db.Where("project_id = ?", 1).First(&diagAfterRemote).Error; err != nil {
		t.Fatal(err)
	}
	if diagAfterRemote.Score != initialScore {
		t.Errorf("state score berubah pasca 403: got %d, want %d", diagAfterRemote.Score, initialScore)
	}
	if diagAfterRemote.FindingsJSON != initialFindingsStr {
		t.Errorf("state findings_json berubah pasca 403: got %s, want %s", diagAfterRemote.FindingsJSON, initialFindingsStr)
	}

	// 2. Request dari loopback namun tanpa token atau token salah (401 Unauthorized)
	unauthCases := []struct {
		name string
		hdr  map[string]string
	}{
		{"tanpa token", nil},
		{"token salah", map[string]string{localauth.Header: "token-palsu-123"}},
		{"bearer salah", map[string]string{"Authorization": "Bearer token-palsu-123"}},
	}
	for _, uc := range unauthCases {
		resp = doReqWithBody(t, s, "POST", "/api/projects/1/diagnostics/ignore-finding/"+findingKey, "localhost", `{"finding_id":"`+findingKey+`"}`, uc.hdr)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("unauthorized ignore-finding (%s): got %d, want 401", uc.name, resp.StatusCode)
		}
		_ = resp.Body.Close()

		// Assert Immutability 2: State tetap tidak tersentuh pasca 401
		var diagAfterUnauth models.Diagnostic
		if err := s.db.Where("project_id = ?", 1).First(&diagAfterUnauth).Error; err != nil {
			t.Fatal(err)
		}
		if diagAfterUnauth.Score != initialScore || diagAfterUnauth.FindingsJSON != initialFindingsStr {
			t.Errorf("state temuan termutasi pada request unauthorized (%s)!", uc.name)
		}
	}

	// 3. Serangan injeksi knowledge node dari remote IP non-loopback
	maliciousNodePayload := `{"title":"Injected Malicious Rule","content":"Remote injection payload","kind":"rule","domain":"security"}`
	resp = doReqWithBody(t, s, "POST", "/api/knowledge/nodes", "10.0.0.5", maliciousNodePayload, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("remote knowledge node injection: got %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Assert Immutability 3: Jumlah knowledge node tetap 0
	var kCountAfterRemote int64
	s.db.Model(&models.KnowledgeNode{}).Count(&kCountAfterRemote)
	if kCountAfterRemote != 0 {
		t.Errorf("knowledge node bocor ke DB pada 403! count=%d", kCountAfterRemote)
	}

	// 4. Request injeksi knowledge node dari loopback tanpa token
	resp = doReqWithBody(t, s, "POST", "/api/knowledge/nodes", "127.0.0.1", maliciousNodePayload, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthorized knowledge node creation: got %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	var kCountAfterUnauth int64
	s.db.Model(&models.KnowledgeNode{}).Count(&kCountAfterUnauth)
	if kCountAfterUnauth != 0 {
		t.Errorf("knowledge node bocor ke DB pada 401! count=%d", kCountAfterUnauth)
	}

	// 5. Remote call ke /api/knowledge/curate
	resp = doReqWithBody(t, s, "POST", "/api/knowledge/curate", "192.168.1.50", `{}`, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("remote knowledge curate: got %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 6. Negative Authorization IDOR / Boundary kepemilikan resource (K35, K2296):
	// Request mutasi dengan token valid tetapi ID proyek fiktif/tidak ada (999999)
	resp = doReqWithBody(t, s, "POST", "/api/projects/999999/diagnostics/ignore-finding/"+findingKey, "localhost", `{"finding_id":"`+findingKey+`"}`, map[string]string{
		localauth.Header: testLocalToken,
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("mutasi finding pada project_id fiktif: got %d, want 404 Not Found", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 7. Negative Authorization: Mutasi project fiktif via PUT /api/projects/999999
	resp = doReqWithBody(t, s, "PUT", "/api/projects/999999", "localhost", `{"name":"Hacked Project"}`, map[string]string{
		localauth.Header: testLocalToken,
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("mutasi project pada project_id fiktif: got %d, want 404 Not Found", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 8. Negative Authorization: Mutasi component fiktif via DELETE /api/components/999999
	resp = doReqWithBody(t, s, "DELETE", "/api/components/999999", "localhost", `{}`, map[string]string{
		localauth.Header: testLocalToken,
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("hapus component pada component_id fiktif: got %d, want 404 Not Found", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 9. Negative Authorization: Hapus todo fiktif via DELETE /api/todos/999999
	resp = doReqWithBody(t, s, "DELETE", "/api/todos/999999", "localhost", `{}`, map[string]string{
		localauth.Header: testLocalToken,
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("hapus todo pada todo_id fiktif: got %d, want 404 Not Found", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Verifikasi State Immutability: entitas demo project #1 dan component #1 tetap utuh
	var projExisting models.Project
	if err := s.db.First(&projExisting, 1).Error; err != nil || projExisting.Name != "demo" {
		t.Fatalf("project #1 termutasi secara tidak sah! got: %+v, err: %v", projExisting, err)
	}
	var compExisting models.Component
	if err := s.db.First(&compExisting, 1).Error; err != nil || compExisting.Name != "api" {
		t.Fatalf("component #1 termutasi secara tidak sah! got: %+v, err: %v", compExisting, err)
	}
}

// TestLocalAccess_AuthorizedMutations memverifikasi bahwa akses mutasi lokal yang sah (loopback + token lokal)
// berhasil diproses dan benar-benar memutasi state secara terotorisasi (K62, K4213).
func TestLocalAccess_AuthorizedMutations(t *testing.T) {
	s := newGuardTestServer(t)

	// Fixture data diagnostik
	findingKey := "SEC-004-VALID"
	finding := models.Finding{
		ID:         findingKey,
		RuleID:     "SEC-004",
		Title:      "Broken Object Level Authorization",
		TargetFile: "pkg/api/server.go",
		Line:       237,
		Status:     models.FindingStatusOpen,
	}
	findingsJSON, err := json.Marshal([]models.Finding{finding})
	if err != nil {
		t.Fatalf("marshal finding: %v", err)
	}
	diag := models.Diagnostic{
		ProjectID:    1,
		Score:        80,
		FindingsJSON: string(findingsJSON),
	}
	if err := s.db.Create(&diag).Error; err != nil {
		t.Fatalf("create diagnostic fixture: %v", err)
	}

	// 1. Mutasi sah: POST ignore-finding dari 127.0.0.1 dengan header token lokal
	resp := doReqWithBody(t, s, "POST", "/api/projects/1/diagnostics/ignore-finding/"+findingKey, "127.0.0.1:8765",
		`{"finding_id":"`+findingKey+`"}`,
		map[string]string{localauth.Header: testLocalToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authorized ignore-finding: got %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Assert: status temuan kini berhasil berubah menjadi "ignored"
	var diagUpdated models.Diagnostic
	if err := s.db.Where("project_id = ?", 1).First(&diagUpdated).Error; err != nil {
		t.Fatal(err)
	}
	var updatedFindings []models.Finding
	if err := json.Unmarshal([]byte(diagUpdated.FindingsJSON), &updatedFindings); err != nil {
		t.Fatal(err)
	}
	if len(updatedFindings) != 1 || updatedFindings[0].Status != models.FindingStatusIgnored {
		t.Errorf("status finding tidak termutasi ke ignored: %+v", updatedFindings)
	}

	// 2. Mutasi sah via IPv6 loopback [::1]:8765 (toggle kembali status finding)
	resp = doReqWithBody(t, s, "POST", "/api/projects/1/diagnostics/ignore-finding/"+findingKey, "[::1]:8765",
		`{"finding_id":"`+findingKey+`"}`,
		map[string]string{localauth.Header: testLocalToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authorized ignore-finding via IPv6: got %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 3. Mutasi sah: tambah knowledge node dari localhost
	validNodePayload := `{"title":"Patokan Validasi Boundary","content":"Boundary fail-closed wajib diuji integrasi","kind":"rule","domain":"security"}`
	resp = doReqWithBody(t, s, "POST", "/api/knowledge/nodes", "localhost", validNodePayload,
		map[string]string{localauth.Header: testLocalToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authorized knowledge node creation: got %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	var kCount int64
	s.db.Model(&models.KnowledgeNode{}).Count(&kCount)
	if kCount != 1 {
		t.Errorf("knowledge node count tidak bertambah: got %d, want 1", kCount)
	}

	// 4. Mutasi sah dengan Bearer token di Authorization header
	resp = doReqWithBody(t, s, "POST", "/api/knowledge/curate", "127.0.0.1", `{}`,
		map[string]string{"Authorization": "Bearer " + testLocalToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authorized knowledge curate via Bearer: got %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 5. Mutasi sah dengan Cookie Studio
	resp = doReqWithBody(t, s, "POST", "/api/knowledge/curate", "localhost", `{}`,
		map[string]string{"Cookie": localauth.Cookie + "=" + testLocalToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authorized knowledge curate via Cookie: got %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// TestLocalAccess_ForeignBrowserOriginRejected memverifikasi mitigasi CSRF dan DNS-rebinding pada rute mutasi:
// Request dari origin browser publik asing (mis. http://attacker.com) tidak diizinkan oleh CORS allowlist
// dan endpoint eksekusi menolak foreign origin dengan status 403 Forbidden (exec_origin_forbidden).
func TestLocalAccess_ForeignBrowserOriginRejected(t *testing.T) {
	s := newGuardTestServer(t)

	foreignOrigin := "http://attacker.com"
	mutationPath := "/api/projects/1/diagnostics/ignore-finding/finding-1"

	// 1. Preflight OPTIONS dari origin asing: CORS tidak menyertakan origin asing di Allow-Origin
	preflightReq := doReq(t, s, "OPTIONS", mutationPath, "localhost", map[string]string{
		"Origin":                         foreignOrigin,
		"Access-Control-Request-Method":  "POST",
		"Access-Control-Request-Headers": "X-Vibeasik-Token, Content-Type",
	})
	if got := preflightReq.Header.Get("Access-Control-Allow-Origin"); got == foreignOrigin {
		t.Errorf("preflight CORS origin asing tidak boleh disetujui: got %q", got)
	}
	_ = preflightReq.Body.Close()

	// 2. Direct POST dengan Origin asing: CORS tidak mengizinkan origin
	postResp := doReqWithBody(t, s, "POST", mutationPath, "localhost", `{"finding_id":"finding-1"}`, map[string]string{
		"Origin":         foreignOrigin,
		localauth.Header: testLocalToken,
	})
	if got := postResp.Header.Get("Access-Control-Allow-Origin"); got == foreignOrigin {
		t.Errorf("CORS allow origin tidak boleh memantulkan origin penyerang: got %q", got)
	}
	_ = postResp.Body.Close()

	// 3. Endpoint eksekusi shell (/api/components/:id/run) diproteksi ganda oleh exec guard:
	// Meskipun membawa token lokal (dan mode LAN aktif), browser origin asing tetap ditolak 403 (exec_origin_forbidden)
	t.Setenv(envAllowLANExec, "1")
	execResp := doReqWithBody(t, s, "POST", "/api/components/1/run", "localhost", `{}`, map[string]string{
		"Origin":         foreignOrigin,
		localauth.Header: testLocalToken,
	})
	if execResp.StatusCode != http.StatusForbidden {
		t.Errorf("exec guard with foreign origin: got %d, want 403", execResp.StatusCode)
	}
	var execBody map[string]interface{}
	_ = json.NewDecoder(execResp.Body).Decode(&execBody)
	_ = execResp.Body.Close()
	if execBody["code"] != "exec_origin_forbidden" {
		t.Errorf("exec guard error code: got %v, want exec_origin_forbidden", execBody["code"])
	}

	// 4. Host DNS-rebinding (evil-rebind.attacker.test): ditolak langsung oleh perimeter s.localAccess
	rebindResp := doReqWithBody(t, s, "POST", mutationPath, "evil-rebind.attacker.test", `{}`, map[string]string{
		localauth.Header: testLocalToken,
	})
	if rebindResp.StatusCode != http.StatusForbidden {
		t.Errorf("dns rebinding host: got %d, want 403", rebindResp.StatusCode)
	}
	var rebindBody map[string]interface{}
	_ = json.NewDecoder(rebindResp.Body).Decode(&rebindBody)
	_ = rebindResp.Body.Close()
	if rebindBody["error"] != "host_forbidden" {
		t.Errorf("dns rebinding error code: got %v, want host_forbidden", rebindBody["error"])
	}
}
