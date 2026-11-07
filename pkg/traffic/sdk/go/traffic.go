// Package vibeasiktraffic mengirim statistik request (rute, status, lama proses, IP, negara, UA, perujuk)
// ke vibeasik Traffic Analyzer. Tidak pernah mengirim query string, header lain, maupun body request.
// Dipasang oleh vibeasik — konfigurasi lewat env:
//
//	VIBEASIK_TRAFFIC_URL        endpoint ingest (wajib; kosong = SDK nonaktif)
//	VIBEASIK_TRAFFIC_KEY        kunci ingest sumber (wajib)
//	VIBEASIK_TRAFFIC_FLUSH_SEC  interval kirim, detik (default 10; 1–2 = realtime)
//	VIBEASIK_TRAFFIC_INSTANCE   nama instance (default hostname)
//	VIBEASIK_TRAFFIC_HASH_IP    1 = IP disamarkan (SHA-256) sebelum dikirim
//	VIBEASIK_TRAFFIC_SKIP       prefix path yang tidak dicatat, pisah koma (default /health)
package vibeasiktraffic

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Event struct {
	TS       int64   `json:"ts"`
	Method   string  `json:"method"`
	Route    string  `json:"route,omitempty"`
	Path     string  `json:"path"`
	Status   int     `json:"status"`
	DurMS    float64 `json:"dur_ms"`
	BytesIn  int64   `json:"bytes_in"`
	BytesOut int64   `json:"bytes_out"`
	IP       string  `json:"ip"`
	Country  string  `json:"country,omitempty"`
	UA       string  `json:"ua,omitempty"`
	Referer  string  `json:"referer,omitempty"`
}

const maxBuffer = 20000 // event di memori; lebih dari ini dibuang (server tidak boleh bocor memori)

var (
	mu       sync.Mutex
	buf      []Event
	once     sync.Once
	endpoint = os.Getenv("VIBEASIK_TRAFFIC_URL")
	key      = os.Getenv("VIBEASIK_TRAFFIC_KEY")
	hashIP   = os.Getenv("VIBEASIK_TRAFFIC_HASH_IP") == "1"
	skip     = splitSkip(os.Getenv("VIBEASIK_TRAFFIC_SKIP"))
	instance = os.Getenv("VIBEASIK_TRAFFIC_INSTANCE")
	client   = &http.Client{Timeout: 10 * time.Second}
)

func splitSkip(v string) []string {
	if v == "" {
		v = "/health"
	}
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Enabled = env URL & kunci terisi.
func Enabled() bool { return endpoint != "" && key != "" }

func start() {
	if instance == "" {
		instance, _ = os.Hostname()
	}
	sec, _ := strconv.Atoi(os.Getenv("VIBEASIK_TRAFFIC_FLUSH_SEC"))
	if sec <= 0 {
		sec = 10
	}
	go func() {
		for range time.Tick(time.Duration(sec) * time.Second) {
			Flush()
		}
	}()
}

// Record mencatat satu request. Tidak pernah memblokir request (hanya append ke buffer).
func Record(ev Event) {
	if !Enabled() {
		return
	}
	for _, p := range skip {
		if strings.HasPrefix(ev.Path, p) {
			return
		}
	}
	once.Do(start)
	if hashIP && ev.IP != "" {
		sum := sha256.Sum256([]byte(ev.IP + key))
		ev.IP = "h:" + hex.EncodeToString(sum[:8])
	}
	mu.Lock()
	if len(buf) < maxBuffer {
		buf = append(buf, ev)
	}
	mu.Unlock()
}

// Flush mengirim buffer sekarang (panggil juga saat graceful shutdown).
func Flush() {
	mu.Lock()
	batch := buf
	buf = nil
	mu.Unlock()
	for len(batch) > 0 {
		n := min(len(batch), 5000)
		body, _ := json.Marshal(map[string]any{"instance": instance, "events": batch[:n]})
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Vibeasik-Key", key)
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode >= 500 || resp.StatusCode == 429 {
			if resp != nil {
				resp.Body.Close()
			}
			requeue(batch) // vibeasik sedang tidak bisa dihubungi: coba lagi pada flush berikutnya
			return
		}
		resp.Body.Close()
		batch = batch[n:]
	}
}

func requeue(batch []Event) {
	mu.Lock()
	defer mu.Unlock()
	if room := maxBuffer - len(buf); room > 0 {
		if len(batch) > room {
			batch = batch[len(batch)-room:]
		}
		buf = append(batch, buf...)
	}
}

// ClientIP: header proxy/CDN dulu, lalu alamat koneksi.
func ClientIP(h http.Header, remoteAddr string) string {
	for _, k := range []string{"CF-Connecting-IP", "True-Client-IP", "X-Real-IP"} {
		if v := strings.TrimSpace(h.Get(k)); v != "" {
			return v
		}
	}
	if v := h.Get("X-Forwarded-For"); v != "" {
		return strings.TrimSpace(strings.Split(v, ",")[0])
	}
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

func Country(h http.Header) string {
	for _, k := range []string{"CF-IPCountry", "X-Vercel-IP-Country", "CloudFront-Viewer-Country", "X-Country-Code"} {
		if v := h.Get(k); v != "" {
			return v
		}
	}
	return ""
}

func RefererHost(ref string) string {
	if u, err := url.Parse(ref); err == nil {
		return u.Host
	}
	return ""
}

type recorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *recorder) WriteHeader(code int) { r.status = code; r.ResponseWriter.WriteHeader(code) }
func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}
func (r *recorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Middleware untuk net/http, chi, gorilla/mux, gin (bungkus engine: http.ListenAndServe(addr, Middleware(r))), echo (echo.WrapMiddleware).
// Pola rute tidak diketahui di sini; vibeasik menormalkan path (/users/42 → /users/:id).
func Middleware(next http.Handler) http.Handler {
	if !Enabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &recorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		Record(Event{
			TS: start.UnixMilli(), Method: r.Method, Path: r.URL.Path, Status: status,
			DurMS:   float64(time.Since(start).Microseconds()) / 1000,
			BytesIn: max(r.ContentLength, 0), BytesOut: rec.bytes,
			IP: ClientIP(r.Header, r.RemoteAddr), Country: Country(r.Header), UA: r.UserAgent(), Referer: RefererHost(r.Referer()),
		})
	})
}
