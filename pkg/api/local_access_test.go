package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/localauth"
)

func doReq(t *testing.T, s *Server, method, path, host string, hdr map[string]string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Host = host
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
