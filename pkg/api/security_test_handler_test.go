package api

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
)

func secTestRequest(t *testing.T, s *Server, method, path, body, token string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = "localhost"
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set(localauth.Header, token)
	}
	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return decodeResp(t, resp)
}

// Jalur penolakan: tanpa token 401, payload tidak sah 400, dan tidak ada hasil uji yang tersimpan.
func TestSecurityTestRejects(t *testing.T) {
	s := newGuardTestServer(t)
	if code, _ := secTestRequest(t, s, "POST", "/api/projects/1/security-test", `{}`, ""); code != 401 {
		t.Errorf("tanpa token → %d, ingin 401", code)
	}
	for _, tc := range []struct{ path, body, want string }{
		{"/api/projects/1/security-test", `{"base_url":"file:///etc/passwd"}`, "http atau https"},
		{"/api/projects/1/security-test", `{"base_url":"gopher://x"}`, "http atau https"},
		{"/api/projects/1/security-test", `{"burst":500}`, "burst"},
		{"/api/projects/1/security-test", `{"base_url":"` + strings.Repeat("a", 3000) + `"}`, "maksimal"},
		{"/api/projects/1/security-test", `{"burst":`, "tidak valid"},
		{"/api/projects/0/security-test", `{}`, "invalid project id"},
		{"/api/projects/999/security-test", `{}`, "tidak ditemukan"},
		{"/api/projects/1/security-test", `{"component_id":999}`, "bukan milik"},
	} {
		code, out := secTestRequest(t, s, "POST", tc.path, tc.body, testLocalToken)
		msg, _ := out["error"].(string)
		if code != 400 || !strings.Contains(msg, tc.want) {
			t.Errorf("%s %s → %d %q, ingin 400 memuat %q", tc.path, tc.body[:min(len(tc.body), 40)], code, msg, tc.want)
		}
	}
	if code, out := secTestRequest(t, s, "GET", "/api/projects/1/security-test", ``, testLocalToken); code != 200 || out["report"] != nil {
		t.Errorf("penolakan tidak boleh menyimpan hasil: %d %v", code, out)
	}
}

// Jalur sah: router dibaca dari sumber komponen, server dipilih dari port yang benar-benar menjawab
// (run_port mati → port yang dideklarasikan), dan hasil terakhir bisa dibaca lagi lewat GET.
func TestSecurityTestRunsAgainstAnsweringPort(t *testing.T) {
	s := newGuardTestServer(t)
	live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer live.Close()
	_, portStr, _ := net.SplitHostPort(strings.TrimPrefix(live.URL, "http://"))
	declared, _ := strconv.Atoi(portStr)
	dead, _ := net.Listen("tcp", "127.0.0.1:0")
	deadPort := dead.Addr().(*net.TCPAddr).Port
	dead.Close()

	var comp models.Component
	s.db.Where("project_id = ?", 1).First(&comp)
	src := `package main
func routes(app *fiber.App) {
	app.Get("/api/items", list)
	app.Post("/api/login", login)
}`
	if err := os.WriteFile(filepath.Join(comp.AbsolutePath, "server.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	s.db.Model(&comp).Updates(map[string]interface{}{"port": declared, "run_port": deadPort})

	code, out := secTestRequest(t, s, "POST", "/api/projects/1/security-test", `{"burst":5}`, testLocalToken)
	if code != 200 {
		t.Fatalf("POST → %d %v", code, out)
	}
	lv, _ := out["live"].(map[string]interface{})
	if lv == nil || lv["reachable"] != true || lv["base_url"] != live.URL {
		t.Fatalf("server yang dipilih keliru: %v (ingin %s)", lv, live.URL)
	}
	if out["files_read"].(float64) != 1 || out["routes_seen"].(float64) != 2 {
		t.Errorf("router tidak terbaca: files=%v routes=%v", out["files_read"], out["routes_seen"])
	}
	// Rute GET dari sumber dipakai sebagai sasaran burst, bukan "/".
	found := false
	for _, c := range out["checks"].([]interface{}) {
		m := c.(map[string]interface{})
		if m["id"] == "LIVE-RATE-LIMIT" {
			ev, _ := m["evidence"].([]interface{})
			found = len(ev) > 0 && strings.Contains(ev[0].(string), "GET /api/items")
		}
	}
	if !found {
		t.Errorf("burst seharusnya ke GET /api/items")
	}
	code, last := secTestRequest(t, s, "GET", "/api/projects/1/security-test", ``, testLocalToken)
	rep, _ := last["report"].(map[string]interface{})
	if code != 200 || rep == nil || rep["score"] != out["score"] {
		t.Errorf("GET hasil terakhir: %d %v", code, last)
	}
}
