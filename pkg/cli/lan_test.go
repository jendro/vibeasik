package cli

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/localauth"
)

func TestLanGate(t *testing.T) {
	var gotToken, gotHost, gotCookie string
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken, gotHost, gotCookie = r.Header.Get(localauth.Header), r.Host, r.Header.Get("Cookie")
		w.Write([]byte("cockpit"))
	}))
	defer engine.Close()
	eu, _ := url.Parse(engine.URL)
	g := newLanGate(eu, "rahasia")
	rotated := ""
	g.onRotate = func(p string) { rotated = p }
	srv := httptest.NewServer(g)
	defer srv.Close()
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	// tanpa sesi: ditolak, engine tidak tersentuh
	resp, _ := http.Get(srv.URL + "/api/x")
	if resp.StatusCode != 401 || gotToken != "" {
		t.Fatalf("tanpa PIN harus 401, dapat %d (token bocor ke engine: %q)", resp.StatusCode, gotToken)
	}

	// PIN benar: cookie sesi, lalu request diteruskan dengan token engine
	resp, _ = noRedirect.PostForm(srv.URL+"/__lan_pin", url.Values{"pin": {g.PIN()}})
	if resp.StatusCode != 303 || len(resp.Cookies()) != 1 {
		t.Fatalf("PIN benar harus 303 + cookie, dapat %d", resp.StatusCode)
	}
	req, _ := http.NewRequest("GET", srv.URL+"/", nil)
	req.AddCookie(resp.Cookies()[0])
	r2, _ := http.DefaultClient.Do(req)
	if r2.StatusCode != 200 || gotToken != "rahasia" || gotHost != eu.Host || gotCookie != "" {
		t.Fatalf("proxy salah: status=%d token=%q host=%q cookie=%q", r2.StatusCode, gotToken, gotHost, gotCookie)
	}

	// 5 kali salah mengganti PIN
	old := g.PIN()
	for i := 0; i < lanMaxAttempts; i++ {
		if _, ok := g.try("abcdef"); ok {
			t.Fatal("PIN salah diterima")
		}
	}
	if g.PIN() == old || rotated != g.PIN() {
		t.Fatal("PIN harus berganti setelah salah berulang")
	}
	if _, ok := g.try(old); ok {
		t.Fatal("PIN lama harus tidak berlaku")
	}
}

func TestLanAllowedPeer(t *testing.T) {
	for addr, want := range map[string]bool{"192.168.1.3:5000": true, "127.0.0.1:1": true, "10.0.0.2:9": true, "8.8.8.8:80": false, "bad": false} {
		if got := lanAllowedPeer(addr); got != want {
			t.Errorf("%s: %v, want %v", addr, got, want)
		}
	}
	if !strings.HasPrefix(newPIN(), "") || len(newPIN()) != 6 {
		t.Error("PIN harus 6 digit")
	}
}

func TestLanGateOriginAndRestore(t *testing.T) {
	var gotOrigin, gotReferer string
	hits := 0
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		gotOrigin, gotReferer = r.Header.Get("Origin"), r.Header.Get("Referer")
	}))
	defer engine.Close()
	eu, _ := url.Parse(engine.URL)
	state := t.TempDir() + "/lan-gate.json"
	g := newLanGate(eu, "rahasia")
	g.restore(state)
	srv := httptest.NewServer(g)
	defer srv.Close()
	session, ok := g.try(g.PIN())
	if !ok {
		t.Fatal("PIN benar ditolak")
	}
	post := func(base, origin string) int {
		req, _ := http.NewRequest("POST", base+"/api/agents/execute", nil)
		req.AddCookie(&http.Cookie{Name: lanCookie, Value: session})
		if origin != "" {
			req.Header.Set("Origin", origin)
			req.Header.Set("Referer", origin+"/office")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode
	}

	// Origin halaman gerbang sendiri: diteruskan sebagai origin engine, seperti Studio dari localhost
	if code := post(srv.URL, srv.URL); code != 200 || gotOrigin != engine.URL || gotReferer != "" {
		t.Fatalf("origin gerbang: status=%d origin=%q referer=%q", code, gotOrigin, gotReferer)
	}
	// website lain tidak boleh menumpang sesi
	before := hits
	if code := post(srv.URL, "http://jahat.example"); code != 403 || hits != before {
		t.Fatalf("origin asing harus 403 tanpa menyentuh engine, dapat %d", code)
	}
	// klien tanpa Origin (curl) tetap lolos tanpa Origin buatan
	if code := post(srv.URL, ""); code != 200 || gotOrigin != "" {
		t.Fatalf("tanpa origin: status=%d origin=%q", code, gotOrigin)
	}

	// gerbang dijalankan ulang: PIN dan sesi bertahan
	g2 := newLanGate(eu, "rahasia")
	g2.restore(state)
	srv2 := httptest.NewServer(g2)
	defer srv2.Close()
	if g2.PIN() != g.PIN() {
		t.Fatal("PIN harus bertahan setelah gerbang dijalankan ulang")
	}
	if code := post(srv2.URL, srv2.URL); code != 200 {
		t.Fatalf("sesi harus bertahan setelah gerbang dijalankan ulang, dapat %d", code)
	}
	if fi, err := os.Stat(state); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("berkas keadaan harus 0600: %v", err)
	}
}
