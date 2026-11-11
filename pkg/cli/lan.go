package cli

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
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
	lanSessionTTL  = 30 * 24 * time.Hour // sama dengan cookie Studio dari `vibeasik open`
	lanStateFile   = "lan-gate.json"
)

var (
	lanListenPort int
	lanShowPIN    bool
)

// lanState PIN & sesi gerbang yang disimpan (0600) agar restart gerbang tidak memaksa masuk ulang.
type lanState struct {
	PIN      string               `json:"pin"`
	Sessions map[string]time.Time `json:"sessions"`
}

func lanStatePath() string { return filepath.Join(localauth.Dir(), lanStateFile) }

// lanGate gerbang LAN: proxy ke engine lokal yang hanya dibuka setelah PIN 6 digit benar.
// Token engine tidak pernah keluar dari laptop; proxy menyisipkannya sendiri ke setiap request.
type lanGate struct {
	mu       sync.Mutex
	pin      string
	failures int
	sessions map[string]time.Time
	token    string
	origin   string // origin engine; Origin browser LAN diterjemahkan ke sini
	state    string // berkas penyimpan PIN & sesi ("" = hanya di memori)
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
	g := &lanGate{pin: newPIN(), sessions: map[string]time.Time{}, token: token, origin: engine.Scheme + "://" + engine.Host}
	p := httputil.NewSingleHostReverseProxy(engine)
	dir := p.Director
	p.Director = func(r *http.Request) {
		dir(r)
		r.Host = engine.Host // engine menolak Host selain loopback (anti DNS-rebinding)
		r.Header.Set(localauth.Header, g.token)
		r.Header.Del("Cookie")
		r.Header.Del("Referer")
		// Sesi ber-PIN setara pemakai di laptop: Origin gerbang (sudah dicek sama-asal di ServeHTTP) diterjemahkan
		// ke origin engine supaya penjaga eksekusi memperlakukannya seperti Studio yang dibuka dari localhost.
		if r.Header.Get("Origin") != "" {
			r.Header.Set("Origin", g.origin)
		}
		r.Header.Del("X-Forwarded-For")
	}
	p.FlushInterval = -1 // SSE/WebSocket tanpa buffer
	g.proxy = p
	return g
}

// restore memuat PIN & sesi yang masih berlaku dari berkas; sesudahnya setiap perubahan disimpan ke sana.
func (g *lanGate) restore(path string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.state = path
	var st lanState
	if b, err := os.ReadFile(path); err == nil && json.Unmarshal(b, &st) == nil && len(st.PIN) == 6 {
		g.pin = st.PIN
		now := time.Now()
		for k, exp := range st.Sessions {
			if now.Before(exp) {
				g.sessions[k] = exp
			}
		}
	}
	g.saveLocked()
}

// saveLocked menulis keadaan gerbang; dipanggil dengan g.mu terkunci.
func (g *lanGate) saveLocked() {
	if g.state == "" {
		return
	}
	b, err := json.Marshal(lanState{PIN: g.pin, Sessions: g.sessions})
	if err != nil {
		return
	}
	tmp := g.state + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, g.state)
	}
}

// sameOrigin: Origin browser harus halaman gerbang ini sendiri; website lain tidak boleh menumpang sesi.
func sameOrigin(origin, host string) bool {
	u, err := url.Parse(origin)
	return err == nil && u.Host != "" && strings.EqualFold(u.Host, host)
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
		g.sessions[s] = time.Now().Add(lanSessionTTL)
		g.failures = 0
		g.saveLocked()
		return s, true
	}
	g.failures++
	if g.failures >= lanMaxAttempts {
		g.pin, g.failures = newPIN(), 0
		rotated = g.pin
		g.saveLocked()
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
		g.saveLocked()
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
			http.SetCookie(w, &http.Cookie{Name: lanCookie, Value: s, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: int(lanSessionTTL.Seconds())})
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
	if o := r.Header.Get("Origin"); o != "" && !sameOrigin(o, r.Host) {
		http.Error(w, "origin ditolak", http.StatusForbidden)
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
IP privat dan berhenti saat perintah dihentikan (Ctrl+C). Lalu lintas LAN memakai HTTP biasa: pakai hanya di jaringan tepercaya.

Setelah PIN benar, komputer itu setara dengan Studio yang dibuka lewat 'vibeasik open': semua aksi tersedia,
termasuk yang menjalankan perintah di laptop ini. Sesi berlaku 30 hari dan, bersama PIN, bertahan saat gerbang
dijalankan ulang. 'vibeasik lan --pin' menampilkan alamat dan PIN yang sedang berlaku.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if lanShowPIN {
			return printLanPIN(cmd)
		}
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
		g.restore(lanStatePath())
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

// printLanPIN menampilkan alamat & PIN gerbang yang sedang berlaku (dari berkas keadaan).
func printLanPIN(cmd *cobra.Command) error {
	var st lanState
	b, err := os.ReadFile(lanStatePath())
	if err != nil || json.Unmarshal(b, &st) != nil || st.PIN == "" {
		return fmt.Errorf("gerbang LAN belum pernah dijalankan — jalankan `vibeasik lan` dulu")
	}
	out := cmd.OutOrStdout()
	for _, u := range lanURLs(lanListenPort) {
		fmt.Fprintln(out, "  ", u)
	}
	fmt.Fprintf(out, "PIN: %s\n", st.PIN)
	return nil
}

func init() {
	lanCmd.Flags().BoolVar(&lanShowPIN, "pin", false, "Tampilkan alamat dan PIN yang sedang berlaku lalu keluar")
	lanCmd.Flags().IntVar(&lanListenPort, "listen", 8766, "Port gerbang LAN")
	rootCmd.AddCommand(lanCmd)
}
