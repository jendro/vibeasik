package api

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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

// waitSecSession mem-poll GET sampai sesi uji tidak lagi berjalan.
func waitSecSession(t *testing.T, s *Server) map[string]interface{} {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		_, out := secTestRequest(t, s, "GET", "/api/projects/1/security-test", ``, testLocalToken)
		sess, _ := out["session"].(map[string]interface{})
		if sess != nil && sess["status"] != "running" {
			return out
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("sesi uji keamanan tidak selesai")
	return nil
}

func secStep(sess map[string]interface{}, key string) map[string]interface{} {
	for _, st := range sess["steps"].([]interface{}) {
		m := st.(map[string]interface{})
		if m["key"] == key {
			return m
		}
	}
	return nil
}

// Uji di latar: server yang belum berjalan dinyalakan lewat dev command, diuji, lalu dimatikan lagi.
func TestSecurityTestSessionStartsAndStopsServer(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 tidak tersedia untuk server uji")
	}
	s := newGuardTestServer(t)
	t.Setenv(envAllowLANExec, "1") // app.Test tidak datang dari loopback
	var comp models.Component
	s.db.Where("project_id = ?", 1).First(&comp)
	_ = os.WriteFile(filepath.Join(comp.AbsolutePath, "server.go"), []byte("package main\nfunc r(app *fiber.App) { app.Get(\"/\", h) }\n"), 0o644)
	s.db.Model(&comp).Updates(map[string]interface{}{"dev_command": `python3 -m http.server "$PORT" --bind 127.0.0.1`, "port": 0, "run_port": 0})

	code, out := secTestRequest(t, s, "POST", "/api/projects/1/security-test/start", `{"start_server":true,"skip_burst":true}`, testLocalToken)
	if code != 200 || out["session"] == nil {
		t.Fatalf("start → %d %v", code, out)
	}
	if ev, ok := out["session"].(map[string]interface{})["events"].([]interface{}); !ok || ev == nil {
		t.Errorf("events harus array (bukan null) agar Studio tidak galat: %v", out["session"])
	}
	if code, out := secTestRequest(t, s, "POST", "/api/projects/1/security-test/start", `{"start_server":true}`, testLocalToken); code != 409 && code != 200 {
		t.Errorf("start kedua saat berjalan → %d %v, ingin 409", code, out)
	}
	res := waitSecSession(t, s)
	sess := res["session"].(map[string]interface{})
	if sess["status"] != "completed" {
		t.Fatalf("status sesi %v: %v", sess["status"], sess["events"])
	}
	if st := secStep(sess, "server"); st["status"] != "done" {
		t.Fatalf("server tidak dinyalakan: %v · %v", st, sess["events"])
	}
	if st := secStep(sess, "live"); st["status"] != "done" {
		t.Errorf("server tidak diuji: %v", st)
	}
	if st := secStep(sess, "stop"); st["status"] != "done" {
		t.Errorf("server tidak dimatikan: %v", st)
	}
	srv, _ := sess["server"].(map[string]interface{})
	if srv == nil || srv["source"] != "launched" || srv["state"] != "stopped" {
		t.Errorf("info server keliru: %v", srv)
	}
	rep, _ := res["report"].(map[string]interface{})
	if lv, _ := rep["live"].(map[string]interface{}); lv == nil || lv["reachable"] != true {
		t.Errorf("laporan tanpa bukti server: %v", rep["live"])
	}
}

// Dev command yang langsung keluar: langkah server gagal dengan alasan, kode tetap dibaca, sesi tetap selesai.
func TestSecurityTestSessionServerExits(t *testing.T) {
	s := newGuardTestServer(t)
	t.Setenv(envAllowLANExec, "1")
	var comp models.Component
	s.db.Where("project_id = ?", 1).First(&comp)
	s.db.Model(&comp).Updates(map[string]interface{}{"dev_command": "echo boom; exit 3", "port": 0, "run_port": 0})
	if code, out := secTestRequest(t, s, "POST", "/api/projects/1/security-test/start", `{"start_server":true}`, testLocalToken); code != 200 {
		t.Fatalf("start → %d %v", code, out)
	}
	sess := waitSecSession(t, s)["session"].(map[string]interface{})
	if sess["status"] != "completed" {
		t.Fatalf("status %v", sess["status"])
	}
	if st := secStep(sess, "server"); st["status"] != "failed" || !strings.Contains(st["detail"].(string), "keluar") {
		t.Errorf("langkah server: %v", st)
	}
	if st := secStep(sess, "live"); st["status"] != "skipped" {
		t.Errorf("tanpa server, probe harus dilewati: %v", st)
	}
	srv, _ := sess["server"].(map[string]interface{})
	logs, _ := srv["logs"].([]interface{})
	joined := fmt.Sprint(logs...)
	if !strings.Contains(joined, "boom") {
		t.Errorf("log server tidak terbawa: %v", logs)
	}
}
