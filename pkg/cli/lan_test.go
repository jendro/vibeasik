package cli

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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
