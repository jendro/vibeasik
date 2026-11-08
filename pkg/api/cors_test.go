package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateOrigin(t *testing.T) {
	cases := []struct {
		input  string
		want   string
		wantOK bool
	}{
		{"http://localhost:3000", "http://localhost:3000", true},
		{"https://app.example.com", "https://app.example.com", true},
		{"https://app.example.com:8443", "https://app.example.com:8443", true},
		{"  https://app.example.com/  ", "https://app.example.com", true},
		{"https://app.example.com/path?foo=bar#frag", "https://app.example.com", true},
		{"*", "", false},
		{"", "", false},
		{"   ", "", false},
		{"ftp://example.com", "", false},
		{"javascript:alert(1)", "", false},
		{"localhost:3000", "", false}, // missing scheme
	}

	for _, c := range cases {
		got, ok := validateOrigin(c.input)
		if ok != c.wantOK || got != c.want {
			t.Errorf("validateOrigin(%q) = (%q, %v), want (%q, %v)", c.input, got, ok, c.want, c.wantOK)
		}
	}
}

func TestCorsOrigins_FallbackAndEnvResolution(t *testing.T) {
	// 1. Tanpa env: fallback ke defaultCORSOrigins
	t.Setenv(envCORSOrigins, "")
	t.Setenv(envFrontendOrigins, "")
	t.Setenv(envAllowedOrigins, "")
	if got := corsOrigins(); got != defaultCORSOrigins {
		t.Errorf("corsOrigins() default fallback = %q, want %q", got, defaultCORSOrigins)
	}

	// 2. FRONTEND_ORIGINS disetel
	t.Setenv(envFrontendOrigins, "http://localhost:3000, https://app.example.com/")
	if got := corsOrigins(); got != "http://localhost:3000,https://app.example.com" {
		t.Errorf("corsOrigins() with FRONTEND_ORIGINS = %q", got)
	}

	// 3. ALLOWED_ORIGINS disetel saat FRONTEND_ORIGINS kosong
	t.Setenv(envFrontendOrigins, "")
	t.Setenv(envAllowedOrigins, "https://dashboard.example.com:9000")
	if got := corsOrigins(); got != "https://dashboard.example.com:9000" {
		t.Errorf("corsOrigins() with ALLOWED_ORIGINS = %q", got)
	}

	// 4. Wildcard di env ditolak keras (K36) -> allowlist kosong
	t.Setenv(envFrontendOrigins, "*")
	t.Setenv(envAllowedOrigins, "")
	if got := corsOrigins(); got != "" {
		t.Errorf("corsOrigins() with wildcard must return empty, got %q", got)
	}
}

func TestCORSMatrix_Integration(t *testing.T) {
	// Uji matriks origin x method pada router Fiber nyata (K4213, K4336, K4942)
	t.Setenv(envFrontendOrigins, "http://localhost:3000,https://app.example.com")
	s := newGuardTestServer(t)

	tests := []struct {
		name       string
		method     string
		origin     string
		wantACAO   string
		wantStatus int
	}{
		{
			name:       "Origin terdaftar (localhost) pada GET",
			method:     "GET",
			origin:     "http://localhost:3000",
			wantACAO:   "http://localhost:3000",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Origin terdaftar (production) pada preflight OPTIONS",
			method:     "OPTIONS",
			origin:     "https://app.example.com",
			wantACAO:   "https://app.example.com",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "Origin asing (evil.example.com) ditolak pada GET",
			method:     "GET",
			origin:     "https://evil.example.com",
			wantACAO:   "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Origin asing (evil.example.com) ditolak pada preflight OPTIONS",
			method:     "OPTIONS",
			origin:     "https://evil.example.com",
			wantACAO:   "",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "Request tanpa header Origin tetap lolos",
			method:     "GET",
			origin:     "",
			wantACAO:   "",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			req.Host = "localhost"
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.method == "OPTIONS" {
				req.Header.Set("Access-Control-Request-Method", "GET")
			}
			resp, err := s.app.Test(req, -1)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			gotACAO := resp.Header.Get("Access-Control-Allow-Origin")
			if gotACAO != tt.wantACAO {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", gotACAO, tt.wantACAO)
			}
			// Pastikan header Vary selalu menyertakan Origin (K2308)
			gotVary := resp.Header.Get("Vary")
			if !strings.Contains(gotVary, "Origin") {
				t.Errorf("Vary = %q, expected to contain 'Origin'", gotVary)
			}
		})
	}
}
