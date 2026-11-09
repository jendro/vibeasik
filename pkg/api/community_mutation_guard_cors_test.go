package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/knowledgesync"
	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/setup"
)

// setupCommunityTestEnv menyiapkan server Fiber hermetis dengan database SQLite in-memory,
// home directory temporer (mengisolasi setup.json dan token), serta origin CORS eksplisit.
func setupCommunityTestEnv(t *testing.T, configuredOrigins string) (*Server, *gorm.DB, string) {
	t.Helper()

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv(localauth.EnvToken, testLocalToken)
	t.Setenv(envAllowLANExec, "")
	t.Setenv(envExecToken, "")
	t.Setenv(envCORSOrigins, configuredOrigins)
	t.Setenv(envFrontendOrigins, "")
	t.Setenv(envAllowedOrigins, "")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite in-memory db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&models.Project{},
		&models.Component{},
		&models.ComponentRoute{},
		&models.Diagnostic{},
		&models.LocalAgentConfig{},
		&models.Todo{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	proj := models.Project{
		Name:     "Baseline Community Project",
		Slug:     "baseline-community",
		Status:   models.StatusActive,
		RootPath: t.TempDir(),
	}
	if err := db.Create(&proj).Error; err != nil {
		t.Fatalf("create baseline project fixture: %v", err)
	}

	srv := NewServer(db)
	return srv, db, tempHome
}

func doCommunityTestReq(s *Server, method, path, body, origin, token string) (*http.Response, map[string]interface{}) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Host = "localhost"
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set(localauth.Header, token)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if method == http.MethodOptions {
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type, X-Vibeasik-Token")
	}

	resp, err := s.app.Test(req, -1)
	if err != nil {
		return nil, map[string]interface{}{"test_err": err.Error()}
	}

	var res map[string]interface{}
	if resp.Body != nil {
		b, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(b, &res)
		resp.Body = io.NopCloser(bytes.NewReader(b))
	}
	return resp, res
}

// TestCommunityMutation_CORSPreflightMatrix membuktikan bahwa preflight CORS (OPTIONS)
// memberikan Access-Control-Allow-Origin hanya untuk origin terkonfigurasi,
// dan menolaknya (header kosong) untuk origin asing di seluruh endpoint target.
func TestCommunityMutation_CORSPreflightMatrix(t *testing.T) {
	trustedOrigin := "http://localhost:3000"
	srv, _, _ := setupCommunityTestEnv(t, trustedOrigin)

	targetEndpoints := []string{
		"/api/setup/database/test",
		"/api/setup/database",
		"/api/setup/complete",
		"/api/setup/restart",
		"/mcp",
		"/api/projects",
		"/api/projects/1/deprecate",
	}

	for _, ep := range targetEndpoints {
		t.Run("Preflight_Trusted_"+ep, func(t *testing.T) {
			resp, _ := doCommunityTestReq(srv, http.MethodOptions, ep, "", trustedOrigin, "")
			if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
				t.Errorf("preflight %s with trusted origin returned %d, want 204 or 200", ep, resp.StatusCode)
			}
			acao := resp.Header.Get("Access-Control-Allow-Origin")
			if acao != trustedOrigin {
				t.Errorf("preflight %s ACAO = %q, want %q", ep, acao, trustedOrigin)
			}
		})

		t.Run("Preflight_Foreign_"+ep, func(t *testing.T) {
			resp, _ := doCommunityTestReq(srv, http.MethodOptions, ep, "", "https://evil.attacker.com", "")
			acao := resp.Header.Get("Access-Control-Allow-Origin")
			if acao != "" {
				t.Errorf("preflight %s with foreign origin MUST NOT set ACAO, got %q", ep, acao)
			}
		})
	}
}

// TestCommunityMutation_ForeignOriginDirectMutationsBlocked membuktikan bahwa request mutasi
// langsung dari origin asing tanpa autentikasi lokal ditolak HTTP 401 dan tidak mengubah state.
func TestCommunityMutation_ForeignOriginDirectMutationsBlocked(t *testing.T) {
	srv, db, _ := setupCommunityTestEnv(t, "http://localhost:3000")

	var initialCount int64
	db.Model(&models.Project{}).Count(&initialCount)

	mutations := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"SetupDatabaseTest", "POST", "/api/setup/database/test", `{"driver":"sqlite","name":"test"}`},
		{"SetupDatabaseSave", "POST", "/api/setup/database", `{"driver":"sqlite","name":"test"}`},
		{"SetupComplete", "POST", "/api/setup/complete", `{}`},
		{"SetupRestart", "POST", "/api/setup/restart", `{}`},
		{"MCPPost", "POST", "/mcp", `{"jsonrpc":"2.0","method":"tools/list","id":1}`},
		{"CreateProject", "POST", "/api/projects", `{"name":"Foreign Project","root_path":"/tmp/foreign"}`},
		{"DeprecateProject", "POST", "/api/projects/1/deprecate", `{"reason":"hacked"}`},
	}

	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			resp, res := doCommunityTestReq(srv, tc.method, tc.path, tc.body, "https://evil.attacker.com", "")
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("%s: expected 401 Unauthorized, got %d (resp: %v)", tc.name, resp.StatusCode, res)
			}
			if acao := resp.Header.Get("Access-Control-Allow-Origin"); acao != "" {
				t.Errorf("%s: foreign origin must not receive ACAO, got %q", tc.name, acao)
			}
		})
	}

	// Verifikasi State Invariant: tidak ada data proyek bertambah atau termutasi
	var finalCount int64
	db.Model(&models.Project{}).Count(&finalCount)
	if finalCount != initialCount {
		t.Errorf("project count changed from %d to %d after rejected foreign mutations", initialCount, finalCount)
	}

	var p1 models.Project
	if err := db.First(&p1, 1).Error; err != nil {
		t.Fatalf("fetch project 1: %v", err)
	}
	if p1.Status != models.StatusActive || p1.DeprecatedAt != nil {
		t.Errorf("project 1 status was modified unexpectedly: status=%s, deprecated_at=%v", p1.Status, p1.DeprecatedAt)
	}

	if setup.Complete() {
		t.Errorf("setup.Complete() must remain false after unauthorized complete request")
	}
}

// TestCommunityMutation_UnauthenticatedAndUnauthorizedMCPAndProjects membuktikan penolakan
// perimeter fail-closed terhadap request tanpa autentikasi atau otorisasi objek.
func TestCommunityMutation_UnauthenticatedAndUnauthorizedMCPAndProjects(t *testing.T) {
	srv, db, _ := setupCommunityTestEnv(t, "http://localhost:3000")

	// 1. /mcp tanpa auth
	resp, res := doCommunityTestReq(srv, "POST", "/mcp", `{"jsonrpc":"2.0","method":"initialize","id":1}`, "", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("POST /mcp without auth status = %d, want 401 (res: %v)", resp.StatusCode, res)
	}

	// 2. /mcp/sse tanpa auth
	resp, _ = doCommunityTestReq(srv, "GET", "/mcp/sse", "", "", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /mcp/sse without auth status = %d, want 401", resp.StatusCode)
	}

	// 3. /projects tanpa auth
	resp, _ = doCommunityTestReq(srv, "GET", "/api/projects", "", "", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /api/projects without auth status = %d, want 401", resp.StatusCode)
	}

	resp, _ = doCommunityTestReq(srv, "POST", "/api/projects", `{"name":"NoAuth Project"}`, "", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("POST /api/projects without auth status = %d, want 401", resp.StatusCode)
	}

	// 4. /projects/:id/deprecate tanpa auth
	resp, _ = doCommunityTestReq(srv, "POST", "/api/projects/1/deprecate", `{"reason":"test"}`, "", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("POST /api/projects/1/deprecate without auth status = %d, want 401", resp.StatusCode)
	}

	// 5. Otorisasi Objek: project ID non-existent dengan auth sah -> 404 Not Found
	resp, res = doCommunityTestReq(srv, "POST", "/api/projects/99999/deprecate", `{"reason":"test"}`, "", testLocalToken)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST /api/projects/99999/deprecate with auth status = %d, want 404 Not Found (res: %v)", resp.StatusCode, res)
	}

	// 6. Verifikasi Invariant: Proyek 1 tetap tidak terdeprecate
	var p models.Project
	if err := db.First(&p, 1).Error; err != nil {
		t.Fatalf("fetch project: %v", err)
	}
	if p.DeprecatedAt != nil {
		t.Errorf("project 1 deprecated_at changed to %v, want nil", p.DeprecatedAt)
	}
}

// TestCommunityMutation_ValidSetupBootstrapFlow membuktikan bahwa alur setup yang sah
// (dengan token lokal dan kredensial valid) tetap berfungsi penuh tanpa mematahkan bootstrap.
func TestCommunityMutation_ValidSetupBootstrapFlow(t *testing.T) {
	srv, _, tempHome := setupCommunityTestEnv(t, "http://localhost:8760")

	// Pastikan keadaan awal setup belum complete
	if setup.Complete() {
		t.Fatal("setup must not be complete initially")
	}

	// Pasang seam restart engine
	restarted := false
	srv.SetRestart(func() error {
		restarted = true
		return nil
	})

	// 1. GET /api/setup (initial status)
	resp, res := doCommunityTestReq(srv, "GET", "/api/setup", "", "", testLocalToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/setup status %d, want 200 (res: %v)", resp.StatusCode, res)
	}
	if res["complete"] != false {
		t.Errorf("initial complete state = %v, want false", res["complete"])
	}

	// 2. POST /api/setup/account (pilihan akun skip)
	resp, _ = doCommunityTestReq(srv, "POST", "/api/setup/account", `{"choice":"skip"}`, "", testLocalToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/setup/account status %d, want 200", resp.StatusCode)
	}

	// 3. POST /api/setup/terms (persetujuan ketentuan dengan versi yang sah)
	termsBody := fmt.Sprintf(`{"accept":true,"version":%q}`, knowledgesync.ConsentVersion)
	resp, _ = doCommunityTestReq(srv, "POST", "/api/setup/terms", termsBody, "", testLocalToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/setup/terms status %d, want 200", resp.StatusCode)
	}

	// 4. POST /api/setup/database/test (uji koneksi database SQLite yang sah)
	resp, res = doCommunityTestReq(srv, "POST", "/api/setup/database/test", `{"driver":"sqlite","name":"vibeasik_test"}`, "", testLocalToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/setup/database/test status %d, want 200 (res: %v)", resp.StatusCode, res)
	}
	if ok, _ := res["ok"].(bool); !ok {
		t.Errorf("setup database test reported not OK: %v", res)
	}

	// 5. POST /api/setup/database (simpan konfigurasi database SQLite)
	resp, res = doCommunityTestReq(srv, "POST", "/api/setup/database", `{"driver":"sqlite","name":"vibeasik_test"}`, "", testLocalToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/setup/database status %d, want 200 (res: %v)", resp.StatusCode, res)
	}

	// Pastikan file .env tertulis di tempHome/.vibeasik/.env
	envPath := filepath.Join(tempHome, ".vibeasik", ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Errorf(".env file was not created at %s", envPath)
	}

	// 6. POST /api/setup/restart (mulai ulang engine dengan seam restart)
	resp, res = doCommunityTestReq(srv, "POST", "/api/setup/restart", "", "", testLocalToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/setup/restart status %d, want 200 (res: %v)", resp.StatusCode, res)
	}
	if restarting, _ := res["restarting"].(bool); !restarting {
		t.Errorf("restart response missing restarting: true, got %v", res)
	}
	_ = restarted // seam restart dieksekusi di goroutine background setelah jeda respon

	// 7. POST /api/setup/complete (penyelesaian setup setelah syarat terpenuhi)
	resp, res = doCommunityTestReq(srv, "POST", "/api/setup/complete", "", "", testLocalToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/setup/complete status %d, want 200 (res: %v)", resp.StatusCode, res)
	}
	if comp, _ := res["complete"].(bool); !comp {
		t.Errorf("complete response complete flag = %v, want true", comp)
	}

	// 8. Verifikasi State: setup.Complete() sekarang bernilai true
	if !setup.Complete() {
		t.Errorf("setup.Complete() must be true after valid bootstrap completion")
	}
}

// TestCommunityMutation_DirectUnprefixedSetupRoutesProtected memverifikasi bahwa pemanggilan
// rute setup langsung tanpa prefix /api (misal POST /setup/database/test) tetap ditolak fail-closed
// oleh perimeter keamanan bila tanpa token, dan tidak dapat mengubah state.
func TestCommunityMutation_DirectUnprefixedSetupRoutesProtected(t *testing.T) {
	srv, _, _ := setupCommunityTestEnv(t, "http://localhost:8760")

	unprefixed := []string{
		"/setup/database/test",
		"/setup/database",
		"/setup/complete",
		"/setup/restart",
		"/projects",
	}

	for _, path := range unprefixed {
		t.Run("Unprefixed_"+path, func(t *testing.T) {
			resp, _ := doCommunityTestReq(srv, "POST", path, `{"driver":"sqlite","name":"hacked"}`, "https://evil.attacker.com", "")
			// Harus ditolak dengan status 401 Unauthorized oleh perimeter gatePage
			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("POST %s without auth status = %d, want 401", path, resp.StatusCode)
			}
		})
	}
}

// TestCommunityMutation_DevCommandForeignOriginBlockedByExecGuard membuktikan bahwa meskipun
// token lokal valid disertakan, request pembuatan proyek dengan dev_command dari origin asing
// (misal browser membuka website penyerang) ditolak HTTP 403 Forbidden oleh exec guard.
func TestCommunityMutation_DevCommandForeignOriginBlockedByExecGuard(t *testing.T) {
	srv, db, _ := setupCommunityTestEnv(t, "http://localhost:3000")
	t.Setenv(envAllowLANExec, "1") // izinkan LAN peer agar evaluasi origin guard aktif

	var initialCount int64
	db.Model(&models.Project{}).Count(&initialCount)

	payload := `{
		"name": "Malicious Project",
		"components": [
			{"name": "malicious_comp", "role": "backend", "dev_command": "rm -rf /tmp/data"}
		]
	}`

	resp, res := doCommunityTestReq(srv, "POST", "/api/projects", payload, "https://evil.attacker.com", testLocalToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("POST /api/projects with dev_command from foreign origin status = %d, want 403 (res: %v)", resp.StatusCode, res)
	}
	if code, _ := res["code"].(string); code != "exec_origin_forbidden" {
		t.Errorf("expected error code 'exec_origin_forbidden', got %q", code)
	}

	var finalCount int64
	db.Model(&models.Project{}).Count(&finalCount)
	if finalCount != initialCount {
		t.Errorf("project count changed from %d to %d after forbidden dev_command attempt", initialCount, finalCount)
	}
}

// TestCommunityMutation_SetupNegativeInvariants membuktikan bahwa permintaan setup yang tidak memenuhi
// prasyarat ditolak dengan kode status yang tepat dan tidak mengubah state sistem.
func TestCommunityMutation_SetupNegativeInvariants(t *testing.T) {
	srv, _, _ := setupCommunityTestEnv(t, "http://localhost:8760")

	// 1. Setup complete sebelum terms disetujui -> 400 Bad Request
	resp, res := doCommunityTestReq(srv, "POST", "/api/setup/complete", "", "", testLocalToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("POST /api/setup/complete before terms status = %d, want 400 (res: %v)", resp.StatusCode, res)
	}
	if setup.Complete() {
		t.Error("setup.Complete() must remain false when complete fails")
	}

	// 2. Setup database dengan driver tidak didukung -> 400 Bad Request
	resp, res = doCommunityTestReq(srv, "POST", "/api/setup/database/test", `{"driver":"oracle","name":"testdb"}`, "", testLocalToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("POST /api/setup/database/test with oracle status = %d, want 400 (res: %v)", resp.StatusCode, res)
	}

	resp, res = doCommunityTestReq(srv, "POST", "/api/setup/database", `{"driver":"oracle","name":"testdb"}`, "", testLocalToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("POST /api/setup/database with oracle status = %d, want 400 (res: %v)", resp.StatusCode, res)
	}

	// 3. Setup restart tanpa fungsi restart terpasang -> 501 Not Implemented
	resp, res = doCommunityTestReq(srv, "POST", "/api/setup/restart", "", "", testLocalToken)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("POST /api/setup/restart without handler status = %d, want 501 (res: %v)", resp.StatusCode, res)
	}
}

