package cli

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/localauth"
)

const (
	lanCookie      = "va_lan"
	lanMaxAttempts = 5
)

var lanListenPort int

// lanGate gerbang LAN: proxy ke engine lokal yang hanya dibuka setelah PIN 6 digit benar.
// Token engine tidak pernah keluar dari laptop; proxy menyisipkannya sendiri ke setiap request.
type lanGate struct {
	mu       sync.Mutex
	pin      string
	failures int
	sessions map[string]time.Time
	token    string
	proxy    *httputil.ReverseProxy
	onRotate func(pin string)
}

func newPIN() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func newLanGate(engine *url.URL, token string) *lanGate {
	g := &lanGate{pin: newPIN(), sessions: map[string]time.Time{}, token: token}
	p := httputil.NewSingleHostReverseProxy(engine)
	dir := p.Director
	p.Director = func(r *http.Request) {
		dir(r)
		r.Host = engine.Host // engine menolak Host selain loopback (anti DNS-rebinding)
		r.Header.Set(localauth.Header, g.token)
		r.Header.Del("Cookie")
		r.Header.Del("X-Forwarded-For")
	}
	p.FlushInterval = -1 // SSE/WebSocket tanpa buffer
	g.proxy = p
	return g
}

func (g *lanGate) PIN() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.pin
}

// try memeriksa PIN; salah 5 kali berturut-turut mengganti PIN (brute force 6 digit tidak mungkin).
func (g *lanGate) try(pin string) (session string, ok bool) {
	g.mu.Lock()
	var rotated string
	defer func() {
		g.mu.Unlock()
		if rotated != "" && g.onRotate != nil {
			g.onRotate(rotated)
		}
	}()
	if subtle.ConstantTimeCompare([]byte(pin), []byte(g.pin)) == 1 {
		b := make([]byte, 24)
		_, _ = rand.Read(b)
		s := hex.EncodeToString(b)
		g.sessions[s] = time.Now().Add(12 * time.Hour)
		g.failures = 0
		return s, true
	}
	g.failures++
	if g.failures >= lanMaxAttempts {
		g.pin, g.failures = newPIN(), 0
		rotated = g.pin
	}
	return "", false
}

func (g *lanGate) authed(r *http.Request) bool {
	c, err := r.Cookie(lanCookie)
	if err != nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	exp, ok := g.sessions[c.Value]
	if ok && time.Now().After(exp) {
		delete(g.sessions, c.Value)
		return false
	}
	return ok
}

func lanAllowedPeer(remote string) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

const lanPage = `<!doctype html><html lang="id"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>vibeasik — masuk</title>
<style>body{font:16px system-ui;background:#0f1115;color:#e6e6e6;display:grid;place-items:center;height:100vh;margin:0}
form{display:grid;gap:12px;width:min(320px,90vw)}input,button{font:inherit;padding:12px;border-radius:8px;border:1px solid #444;background:#1a1d23;color:inherit}
input{font-size:28px;text-align:center;letter-spacing:.3em}button{background:#3b82f6;border-color:#3b82f6;cursor:pointer}p{margin:0;color:#aaa}.e{color:#f87171}</style>
<form method="post" action="/__lan_pin"><p>Masukkan PIN 6 digit yang tampil di terminal laptop.</p>%s
<input name="pin" inputmode="numeric" pattern="[0-9]{6}" maxlength="6" autocomplete="off" autofocus required><button>Buka vibeasik</button></form></html>`

func (g *lanGate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !lanAllowedPeer(r.RemoteAddr) {
		http.Error(w, "hanya jaringan lokal", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost && r.URL.Path == "/__lan_pin" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if s, ok := g.try(strings.TrimSpace(r.PostFormValue("pin"))); ok {
			http.SetCookie(w, &http.Cookie{Name: lanCookie, Value: s, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 12 * 3600})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		time.Sleep(time.Second)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, lanPage, `<p class="e">PIN salah.</p>`)
		return
	}
	if !g.authed(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, lanPage, "")
		return
	}
	g.proxy.ServeHTTP(w, r)
}

// lanURLs alamat http://<ip-lan>:port untuk dibuka dari komputer lain.
func lanURLs(port int) []string {
	var out []string
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok {
			if ip := n.IP.To4(); ip != nil && ip.IsPrivate() {
				out = append(out, fmt.Sprintf("http://%s:%d", ip, port))
			}
		}
	}
	return out
}

var lanCmd = &cobra.Command{
	Use:   "lan",
	Short: "Bagikan Studio ke komputer lain di LAN (dilindungi PIN 6 digit)",
	Long: `Membuka gerbang di jaringan lokal ke engine yang berjalan di laptop ini. Dari komputer lain cukup buka
alamat yang tampil lalu ketik PIN 6 digit — tanpa menyalin token. Engine tetap bind 127.0.0.1 dan token tidak
pernah meninggalkan laptop. PIN diganti otomatis setelah 5 kali salah. Gerbang hanya menerima koneksi dari
IP privat dan berhenti saat perintah dihentikan (Ctrl+C). Lalu lintas LAN memakai HTTP biasa: pakai hanya di jaringan tepercaya.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := localauth.Token()
		if err != nil {
			return err
		}
		enginePort := config.GetConfig().Port
		engine := &url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", enginePort)}
		hc := http.Client{Timeout: 3 * time.Second}
		resp, err := hc.Get(engine.String() + "/health")
		if err != nil {
			return fmt.Errorf("engine belum berjalan di %s — jalankan `vibeasik serve` dulu", engine.Host)
		}
		resp.Body.Close()

		g := newLanGate(engine, token)
		ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", lanListenPort))
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		g.onRotate = func(pin string) { fmt.Fprintf(out, "\nPIN salah 5 kali — PIN baru: %s\n", pin) }
		fmt.Fprintln(out, "Studio dibagikan ke LAN. Di komputer lain buka salah satu alamat:")
		for _, u := range lanURLs(lanListenPort) {
			fmt.Fprintln(out, "  ", u)
		}
		fmt.Fprintf(out, "PIN: %s   (Ctrl+C untuk berhenti)\n", g.PIN())
		return (&http.Server{Handler: g, ReadHeaderTimeout: 10 * time.Second}).Serve(ln)
	},
}

func init() {
	lanCmd.Flags().IntVar(&lanListenPort, "listen", 8766, "Port gerbang LAN")
	rootCmd.AddCommand(lanCmd)
}
