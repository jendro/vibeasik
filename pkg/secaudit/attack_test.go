package secaudit

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// vulnerableServer meniru kesalahan yang biasa dieksploitasi peretas.
func vulnerableServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/items/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/items/")
		if strings.Contains(id, "'") {
			w.WriteHeader(500)
			_, _ = io.WriteString(w, `{"error":"You have an error in your SQL syntax near '`+id+`'"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":1}`)
	})
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<p>Hasil untuk "+r.URL.Query().Get("q")+"</p>")
	})
	mux.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Query().Get("file"), "etc/passwd") {
			_, _ = io.WriteString(w, "root:x:0:0:root:/root:/bin/bash\n")
			return
		}
		_, _ = io.WriteString(w, "ok")
	})
	mux.HandleFunc("/go", func(w http.ResponseWriter, r *http.Request) {
		if u := r.URL.Query().Get("redirect"); u != "" {
			http.Redirect(w, r, u, http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, "ok")
	})
	mux.HandleFunc("/api/admin/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"email":"a@b.c"}]`)
	})
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if strings.Contains(string(b), "OR '1'='1") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"token":"eyJhbGciOiJIUzI1NiJ9.eyJ9.x"}`)
			return
		}
		w.WriteHeader(401)
	})
	mux.HandleFunc("/.env", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "APP_ENV=prod\nDB_PASSWORD=rahasia\n")
	})
	return httptest.NewServer(mux)
}

// safeServer menolak/menangani input berbahaya dengan benar.
func safeServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/admin"):
			w.WriteHeader(401)
		case r.URL.Path == "/api/auth/login":
			w.WriteHeader(401)
		case r.URL.Path == "/api/items" || strings.HasPrefix(r.URL.Path, "/api/items/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"items":[]}`)
		default:
			w.WriteHeader(404)
		}
	})
	return httptest.NewServer(mux)
}

var attackRoutes = []RouteRef{
	{Method: "GET", Path: "/api/items/:id"},
	{Method: "GET", Path: "/api/items"},
	{Method: "GET", Path: "/search"},
	{Method: "GET", Path: "/files"},
	{Method: "GET", Path: "/go"},
	{Method: "GET", Path: "/api/admin/users"},
	{Method: "POST", Path: "/api/auth/login"},
	{Method: "DELETE", Path: "/api/items/:id"},
}

func checkByID(checks []Check, id string) Check {
	for _, c := range checks {
		if c.ID == id {
			return c
		}
	}
	return Check{}
}

func TestAttacksFindCommonVulnerabilities(t *testing.T) {
	srv := vulnerableServer()
	defer srv.Close()
	var deletes int
	_, checks := probeLive(context.Background(), LiveOptions{BaseURL: srv.URL, Routes: attackRoutes, AuthPaths: []string{"/api/auth/login"},
		Paths: []string{"/api/items"}, SkipBurst: true, Timeout: 5 * time.Second,
		OnRequest: func(method, _ string, _ int, _ time.Duration, _ error) {
			if method == http.MethodDelete {
				deletes++
			}
		}})
	for _, id := range []string{"LIVE-SQLI", "LIVE-XSS", "LIVE-TRAVERSAL", "LIVE-OPEN-REDIRECT", "LIVE-ANON-ACCESS", "LIVE-LOGIN-BYPASS", "LIVE-EXPOSURE"} {
		c := checkByID(checks, id)
		if c.Status != StatusFail {
			t.Errorf("%s: status %q, ingin fail · %v", id, c.Status, c.Evidence)
		}
	}
	if deletes > 0 {
		t.Errorf("uji serangan tidak boleh mengirim DELETE (%d)", deletes)
	}
}

func TestAttacksPassOnSafeServer(t *testing.T) {
	srv := safeServer()
	defer srv.Close()
	live, checks := probeLive(context.Background(), LiveOptions{BaseURL: srv.URL, Routes: attackRoutes, AuthPaths: []string{"/api/auth/login"},
		Paths: []string{"/api/items"}, SkipBurst: true, Timeout: 5 * time.Second})
	for _, c := range checks {
		if c.Source == "live" && (c.Status == StatusFail) && strings.HasPrefix(c.ID, "LIVE-") && c.ID != "LIVE-HEADERS" {
			t.Errorf("server aman dinilai rentan: %s %v", c.ID, c.Evidence)
		}
	}
	if len(live.Attacks) < 50 {
		t.Errorf("serangan terlalu sedikit: %d", len(live.Attacks))
	}
	if c := checkByID(checks, "LIVE-SQLI"); c.Status != StatusPass {
		t.Errorf("LIVE-SQLI: %s %v", c.Status, c.Evidence)
	}
	if c := checkByID(checks, "LIVE-JWT-FORGERY"); c.Status != StatusPass {
		t.Errorf("JWT palsu harus diuji ke rute admin yang menolak anonim: %s %v", c.Status, c.Evidence)
	}
}

// limitedServer: 30 request per jendela 1 detik per kunci; kunci = X-Forwarded-For bila trustXFF.
func limitedServer(trustXFF bool) *httptest.Server {
	var mu sync.Mutex
	counts := map[string]int{}
	window := time.Now()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := "ip"
		if trustXFF && r.Header.Get("X-Forwarded-For") != "" {
			key = r.Header.Get("X-Forwarded-For")
		}
		mu.Lock()
		if time.Since(window) > time.Second {
			counts, window = map[string]int{}, time.Now()
		}
		counts[key]++
		over := counts[key] > 30
		mu.Unlock()
		if over {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(429)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"items":[]}`)
	}))
}

// Rate limit server tidak boleh menyembunyikan hasil: serangan dijeda sesuai Retry-After lalu diulang.
func TestAttacksPauseOnRateLimit(t *testing.T) {
	srv := limitedServer(false)
	defer srv.Close()
	var logs []string
	live, checks := probeLive(context.Background(), LiveOptions{BaseURL: srv.URL, Routes: attackRoutes[:3], Paths: []string{"/api/items"},
		SkipBurst: true, Timeout: 5 * time.Second, AttackBudget: time.Minute, OnLog: func(_, m string) { logs = append(logs, m) }})
	blocked := 0
	for _, a := range live.Attacks {
		if a.Verdict == VerdictBlocked {
			blocked++
		}
	}
	if blocked > 0 {
		t.Errorf("%d serangan masih ditahan meski ada jeda", blocked)
	}
	if c := checkByID(checks, "LIVE-SQLI"); c.Status != StatusPass {
		t.Errorf("LIVE-SQLI: %s %v", c.Status, c.Evidence)
	}
	if !strings.Contains(strings.Join(logs, "\n"), "dijeda") {
		t.Errorf("jeda rate limit harus tercatat di aktivitas: %v", logs)
	}
}

func TestRateLimitBypassWithForgedIP(t *testing.T) {
	for _, trust := range []bool{true, false} {
		srv := limitedServer(trust)
		_, checks := probeLive(context.Background(), LiveOptions{BaseURL: srv.URL, Paths: []string{"/api/items"}, SkipAttacks: true, Burst: 40, Timeout: 5 * time.Second})
		c := checkByID(checks, "LIVE-RATE-LIMIT-BYPASS")
		want := StatusPass
		if trust {
			want = StatusFail
		}
		if c.Status != want {
			t.Errorf("trustXFF=%v: %s, ingin %s · %v", trust, c.Status, want, c.Evidence)
		}
		srv.Close()
	}
}
