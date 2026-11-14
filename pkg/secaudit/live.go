package secaudit

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	defaultBurst     = 25
	maxBurst         = 60
	authBurst        = 12
	burstConcurrency = 5
	maxBodyRead      = 64 << 10
	probeOrigin      = "https://vibeasik-probe.invalid"
	probeUserAgent   = "vibeasik-security-test/1"
)

// sqlErrorRe = tanda galat basis data yang bocor ke respons (MySQL, PostgreSQL, SQLite, MSSQL, Oracle, ORM).
var sqlErrorRe = regexp.MustCompile(`(?i)(you have an error in your sql syntax|SQLSTATE\[|pq: |syntax error at or near|unterminated quoted string|sqlite3?\.|SQLite error|ORA-\d{5}|Microsoft OLE DB|Unclosed quotation mark|mysql_fetch|PDOException|QueryException|gorm\.io|psycopg2\.|SequelizeDatabaseError|PrismaClient)`)

// stackTraceRe = jejak stack / panic yang bocor ke respons.
var stackTraceRe = regexp.MustCompile(`(?i)(goroutine \d+ \[running\]|panic: |Traceback \(most recent call last\)|at [A-Za-z0-9_.]+ \([^)]*:\d+:\d+\)|Stack trace:|Whoops! There was an error)`)

// versionHeaderRe = header yang mengumumkan versi perangkat lunak server.
var versionHeaderRe = regexp.MustCompile(`\d+\.\d+`)

type prober struct {
	base      *url.URL
	client    *http.Client
	mu        sync.Mutex
	count     int
	onRequest func(method, path string, status int, latency time.Duration, err error)
}

func (p *prober) report(method, target string, r probeResp) {
	if p.onRequest == nil {
		return
	}
	path := target
	if u, err := url.Parse(target); err == nil {
		path = u.RequestURI()
		if p.base.Path != "" {
			path = strings.TrimPrefix(path, p.base.Path)
		}
	}
	p.onRequest(method, path, r.status, r.latency, r.err)
}

// ValidateBaseURL mengembalikan pesan galat bila URL server tidak bisa diuji (bukan http/https, tanpa host).
func ValidateBaseURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	if _, err := normalizeBaseURL(raw); err != nil {
		return err.Error()
	}
	return ""
}

// normalizeBaseURL menerima http(s)://host[:port][/prefix]; host saja dianggap http.
func normalizeBaseURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("base_url kosong")
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("base_url tidak valid: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("base_url harus http atau https")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("base_url tanpa host")
	}
	u.RawQuery, u.Fragment = "", ""
	u.Path = strings.TrimRight(u.Path, "/")
	return u, nil
}

func isLoopbackHost(u *url.URL) bool {
	h := u.Hostname()
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

func (p *prober) urlFor(path string, query url.Values) string {
	u := *p.base
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u.Path = p.base.Path + path
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u.String()
}

type probeResp struct {
	status  int
	headers http.Header
	body    string
	latency time.Duration
	err     error
}

func (p *prober) do(ctx context.Context, method, target string, body io.Reader, hdr map[string]string) probeResp {
	p.mu.Lock()
	p.count++
	p.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return probeResp{err: err}
	}
	req.Header.Set("User-Agent", probeUserAgent)
	req.Header.Set("Accept", "application/json, text/html;q=0.9, */*;q=0.5")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	start := time.Now()
	resp, err := p.client.Do(req)
	if err != nil {
		r := probeResp{err: err, latency: time.Since(start)}
		p.report(method, target, r)
		return r
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyRead))
	r := probeResp{status: resp.StatusCode, headers: resp.Header, body: string(data), latency: time.Since(start)}
	p.report(method, target, r)
	return r
}

// probeLive menguji server berjalan. Hanya request biasa (GET, OPTIONS, POST body kosong ke rute login)
// yang dikirim; tidak ada data yang diubah. Hasilnya bukti nyata untuk tiap kontrol.
func probeLive(ctx context.Context, opts LiveOptions) (*LiveResult, []Check) {
	res := &LiveResult{BaseURL: opts.BaseURL}
	base, err := normalizeBaseURL(opts.BaseURL)
	if err != nil {
		res.Error = err.Error()
		return res, []Check{liveSkipped(err.Error())}
	}
	res.BaseURL = base.String()
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	p := &prober{base: base, client: &http.Client{
		Timeout: timeout,
		// Redirect tidak diikuti: header keamanan dinilai pada respons yang benar-benar dikirim server.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}, onRequest: opts.OnRequest}
	// Koneksi keep-alive ke server sasaran ditutup begitu uji selesai.
	defer p.client.CloseIdleConnections()
	paths := opts.Paths
	if len(paths) == 0 {
		paths = []string{"/"}
	}

	// 1. Jangkauan: tiap rute terpilih diminta sekali; yang menjawab menjadi sasaran uji berikutnya.
	type answered struct {
		path string
		resp probeResp
	}
	var ok []answered
	var lastErr error
	for _, path := range paths {
		r := p.do(ctx, http.MethodGet, p.urlFor(path, nil), nil, nil)
		if r.err != nil {
			lastErr = r.err
			continue
		}
		ok = append(ok, answered{path, r})
	}
	if len(ok) == 0 {
		res.Error = lastErr.Error()
		res.RequestsRun = p.count
		return res, []Check{liveSkipped("Server tidak bisa dihubungi: " + lastErr.Error())}
	}
	first := ok[0].resp
	target := ok[0].path
	res.Reachable = true
	res.StatusCode = first.status
	res.LatencyMs = first.latency.Milliseconds()
	res.Headers = pickHeaders(first.headers)

	// 2. Kontrol per rute: tiap rute dinilai sendiri, laporan memuat hasil terburuk beserta rutenya.
	var headers, disclosure, cors, boundary []Check
	for _, a := range ok {
		res.Paths = append(res.Paths, a.path)
		headers = append(headers, liveHeaderCheck(a.resp.headers))
		disclosure = append(disclosure, liveDisclosureCheck(a.resp.headers))
		cors = append(cors, liveCORSCheck(ctx, p, a.path))
		boundary = append(boundary, liveErrorLeakCheck(ctx, p, a.path))
	}
	var checks []Check
	checks = append(checks, worstOf(headers, res.Paths))
	checks = append(checks, liveTransportCheck(base, first.headers))
	checks = append(checks, worstOf(disclosure, res.Paths))
	checks = append(checks, worstOf(cors, res.Paths))
	checks = append(checks, worstOf(boundary, res.Paths))
	if opts.SkipBurst {
		checks = append(checks, Check{ID: "LIVE-RATE-LIMIT", Category: CatRateLimit, Title: "Rate limit terbukti di server", Status: StatusSkipped,
			Severity: SevInfo, Source: "live", Evidence: []string{"Uji burst dilewati atas permintaan."}})
	} else {
		burst := opts.Burst
		if burst <= 0 {
			burst = defaultBurst
		}
		if burst > maxBurst {
			burst = maxBurst
		}
		checks = append(checks, liveRateLimitCheck(ctx, p, http.MethodGet, target, burst, "LIVE-RATE-LIMIT", "Rate limit terbukti di server (DDoS)"))
		for i, ap := range opts.AuthPaths {
			if i >= 2 {
				break
			}
			c := liveRateLimitCheck(ctx, p, http.MethodPost, ap, authBurst, "LIVE-AUTH-LIMIT", "Rate limit rute login "+ap+" (brute-force)")
			checks = append(checks, c)
		}
	}
	res.RequestsRun = p.count
	return res, checks
}

// worstOf memilih hasil terburuk dari pemeriksaan yang sama di beberapa rute dan menyebut rutenya.
func worstOf(per []Check, paths []string) Check {
	best := 0
	for i := range per {
		if rank(per[i]) < rank(per[best]) {
			best = i
		}
	}
	c := per[best]
	if len(per) > 1 {
		if c.Status == StatusPass {
			c.Evidence = append([]string{fmt.Sprintf("Semua %d rute aman: %s", len(paths), strings.Join(paths, ", "))}, c.Evidence...)
		} else {
			c.Evidence = append([]string{"Rute " + paths[best] + ":"}, c.Evidence...)
		}
	}
	return c
}

func liveSkipped(reason string) Check {
	return Check{ID: "LIVE-REACH", Category: CatTransport, Title: "Server berjalan bisa diuji", Status: StatusSkipped,
		Severity: SevInfo, Source: "live", Evidence: []string{reason},
		Suggestion: "Jalankan komponen (vibeasik run) atau isi URL server live, lalu ulangi uji."}
}

var reportedHeaders = []string{"Content-Security-Policy", "Strict-Transport-Security", "X-Content-Type-Options", "X-Frame-Options",
	"Referrer-Policy", "Permissions-Policy", "Cross-Origin-Opener-Policy", "Access-Control-Allow-Origin", "Server", "X-Powered-By"}

func pickHeaders(h http.Header) map[string]string {
	out := map[string]string{}
	for _, k := range reportedHeaders {
		if v := h.Get(k); v != "" {
			if len(v) > 200 {
				v = v[:200] + "…"
			}
			out[k] = v
		}
	}
	return out
}

func liveHeaderCheck(h http.Header) Check {
	c := Check{ID: "LIVE-HEADERS", Category: CatHeaders, Title: "Header keamanan di respons nyata", Source: "live"}
	var missing, ok []string
	if strings.EqualFold(h.Get("X-Content-Type-Options"), "nosniff") {
		ok = append(ok, "X-Content-Type-Options: nosniff")
	} else {
		missing = append(missing, "X-Content-Type-Options: nosniff")
	}
	csp := h.Get("Content-Security-Policy")
	if h.Get("X-Frame-Options") != "" || strings.Contains(strings.ToLower(csp), "frame-ancestors") {
		ok = append(ok, "perlindungan clickjacking (X-Frame-Options / frame-ancestors)")
	} else {
		missing = append(missing, "X-Frame-Options atau CSP frame-ancestors")
	}
	if csp != "" {
		ok = append(ok, "Content-Security-Policy")
	} else {
		missing = append(missing, "Content-Security-Policy")
	}
	if h.Get("Referrer-Policy") != "" {
		ok = append(ok, "Referrer-Policy")
	} else {
		missing = append(missing, "Referrer-Policy")
	}
	for _, s := range ok {
		c.Evidence = append(c.Evidence, "Ada: "+s)
	}
	for _, s := range missing {
		c.Evidence = append(c.Evidence, "Tidak ada: "+s)
	}
	switch {
	case len(missing) == 0:
		c.Status, c.Severity = StatusPass, SevInfo
	case len(missing) <= 2 && len(ok) >= 2:
		c.Status, c.Severity = StatusWarn, SevWarning
	default:
		c.Status, c.Severity = StatusFail, SevWarning
	}
	c.Suggestion = "Tambahkan header yang belum ada lewat middleware helmet atau konfigurasi reverse proxy (nginx add_header)."
	return c
}

func liveTransportCheck(base *url.URL, h http.Header) Check {
	c := Check{ID: "LIVE-TRANSPORT", Category: CatTransport, Title: "HTTPS & HSTS", Source: "live"}
	switch {
	case base.Scheme == "https" && h.Get("Strict-Transport-Security") != "":
		c.Status, c.Severity = StatusPass, SevInfo
		c.Evidence = []string{"HTTPS aktif dengan HSTS: " + h.Get("Strict-Transport-Security")}
	case base.Scheme == "https":
		c.Status, c.Severity = StatusWarn, SevWarning
		c.Evidence = []string{"HTTPS aktif tetapi tanpa Strict-Transport-Security."}
		c.Suggestion = "Tambahkan Strict-Transport-Security: max-age=31536000; includeSubDomains."
	case isLoopbackHost(base):
		c.Status, c.Severity = StatusSkipped, SevInfo
		c.Evidence = []string{"Server lokal (loopback) — HTTPS dinilai pada server live."}
	default:
		c.Status, c.Severity = StatusFail, SevCritical
		c.Evidence = []string{"Server non-lokal dilayani lewat HTTP biasa — kredensial & cookie terkirim tanpa enkripsi."}
		c.Suggestion = "Pasang TLS (Let's Encrypt / Cloudflare) dan alihkan HTTP ke HTTPS."
	}
	return c
}

func liveDisclosureCheck(h http.Header) Check {
	c := Check{ID: "LIVE-DISCLOSURE", Category: CatErrorLeaks, Title: "Versi server tidak diumumkan", Source: "live"}
	var leaks []string
	for _, k := range []string{"Server", "X-Powered-By", "X-AspNet-Version", "X-Runtime"} {
		v := h.Get(k)
		if v == "" {
			continue
		}
		if k == "X-Powered-By" || versionHeaderRe.MatchString(v) {
			leaks = append(leaks, k+": "+v)
		}
	}
	if len(leaks) == 0 {
		c.Status, c.Severity = StatusPass, SevInfo
		c.Evidence = []string{"Tidak ada header yang membuka versi perangkat lunak."}
		return c
	}
	c.Status, c.Severity = StatusWarn, SevInfo
	c.Evidence = leaks
	c.Suggestion = "Sembunyikan versi (nginx server_tokens off, hapus X-Powered-By, Fiber ServerHeader kosong)."
	return c
}

func liveCORSCheck(ctx context.Context, p *prober, path string) Check {
	c := Check{ID: "LIVE-CORS", Category: CatCORS, Title: "CORS menolak asal asing", Source: "live"}
	r := p.do(ctx, http.MethodGet, p.urlFor(path, nil), nil, map[string]string{"Origin": probeOrigin})
	if r.err != nil {
		c.Status, c.Severity = StatusSkipped, SevInfo
		c.Evidence = []string{"Request CORS gagal: " + r.err.Error()}
		return c
	}
	acao := r.headers.Get("Access-Control-Allow-Origin")
	creds := strings.EqualFold(r.headers.Get("Access-Control-Allow-Credentials"), "true")
	switch {
	case acao == probeOrigin && creds:
		c.Status, c.Severity = StatusFail, SevCritical
		c.Evidence = []string{"Asal asing dipantulkan dengan Allow-Credentials: true — situs mana pun bisa membaca data atas nama pengguna yang login."}
	case acao == probeOrigin:
		c.Status, c.Severity = StatusWarn, SevWarning
		c.Evidence = []string{"Asal asing dipantulkan di Access-Control-Allow-Origin."}
	case acao == "*" && creds:
		c.Status, c.Severity = StatusFail, SevCritical
		c.Evidence = []string{"Access-Control-Allow-Origin: * bersama credentials."}
	case acao == "*":
		c.Status, c.Severity = StatusWarn, SevInfo
		c.Evidence = []string{"Access-Control-Allow-Origin: * (aman hanya untuk API publik tanpa cookie/sesi)."}
	default:
		c.Status, c.Severity = StatusPass, SevInfo
		c.Evidence = []string{"Asal asing tidak diizinkan (tidak ada Access-Control-Allow-Origin untuk " + probeOrigin + ")."}
	}
	if c.Status != StatusPass {
		c.Suggestion = "Batasi AllowOrigins ke domain frontend yang sah; jangan memantulkan Origin apa adanya."
	}
	return c
}

// liveErrorLeakCheck mengirim parameter berisi tanda kutip dan nilai panjang ke rute GET, lalu memastikan
// server menjawab terkendali (bukan 5xx) dan tidak membocorkan galat SQL / stack trace.
func liveErrorLeakCheck(ctx context.Context, p *prober, path string) Check {
	c := Check{ID: "LIVE-INPUT-BOUNDARY", Category: CatInjection, Title: "Input tak lazim dijawab terkendali (SQL error tidak bocor)", Source: "live"}
	cases := []url.Values{
		{"id": {"1'"}, "q": {"test'\""}},
		{"id": {"-1"}, "sort": {"name;"}, "page": {"abc"}},
		{"q": {strings.Repeat("a", 2048)}},
	}
	var leaks []string
	var fivexx int
	for _, q := range cases {
		r := p.do(ctx, http.MethodGet, p.urlFor(path, q), nil, nil)
		if r.err != nil {
			continue
		}
		if r.status >= 500 {
			fivexx++
		}
		if m := sqlErrorRe.FindString(r.body); m != "" {
			leaks = append(leaks, fmt.Sprintf("HTTP %d memuat galat basis data: %q", r.status, m))
		}
		if m := stackTraceRe.FindString(r.body); m != "" {
			leaks = append(leaks, fmt.Sprintf("HTTP %d memuat stack trace: %q", r.status, m))
		}
	}
	switch {
	case len(leaks) > 0:
		c.Status, c.Severity = StatusFail, SevCritical
		c.Evidence = leaks
		c.Suggestion = "Galat basis data terlihat dari luar — tanda query tidak berparameter atau galat mentah dikirim ke klien. Pakai placeholder dan kirim pesan umum."
	case fivexx > 0:
		c.Status, c.Severity = StatusWarn, SevWarning
		c.Evidence = []string{fmt.Sprintf("%d dari %d input tak lazim dijawab 5xx — input belum divalidasi sebelum diproses.", fivexx, len(cases))}
		c.Suggestion = "Validasi query/param di awal handler dan balas 400 untuk input tidak sah."
	default:
		c.Status, c.Severity = StatusPass, SevInfo
		c.Evidence = []string{fmt.Sprintf("%d input tak lazim (kutip, angka negatif, string 2 KB) dijawab tanpa 5xx dan tanpa galat SQL.", len(cases))}
	}
	return c
}

// liveRateLimitCheck mengirim burst kecil; rate limit terbukti bila ada jawaban 429.
func liveRateLimitCheck(ctx context.Context, p *prober, method, path string, n int, id, title string) Check {
	c := Check{ID: id, Category: CatRateLimit, Title: title, Source: "live", GuidelineID: "SEC-007"}
	type outcome struct {
		status     int
		retryAfter string
		limitHdr   bool
		err        bool
	}
	results := make([]outcome, n)
	sem := make(chan struct{}, burstConcurrency)
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < n; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			var body io.Reader
			hdr := map[string]string{}
			if method == http.MethodPost {
				body = strings.NewReader("{}")
				hdr["Content-Type"] = "application/json"
			}
			r := p.do(ctx, method, p.urlFor(path, nil), body, hdr)
			if r.err != nil {
				results[i] = outcome{err: true}
				return
			}
			results[i] = outcome{
				status:     r.status,
				retryAfter: r.headers.Get("Retry-After"),
				limitHdr:   r.headers.Get("X-RateLimit-Limit") != "" || r.headers.Get("RateLimit-Limit") != "" || r.headers.Get("X-Ratelimit-Limit") != "",
			}
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	var got429, errs, limitHdr int
	retryAfter := ""
	statuses := map[int]int{}
	for _, o := range results {
		if o.err {
			errs++
			continue
		}
		statuses[o.status]++
		if o.status == http.StatusTooManyRequests {
			got429++
			if retryAfter == "" {
				retryAfter = o.retryAfter
			}
		}
		if o.limitHdr {
			limitHdr++
		}
	}
	c.Evidence = []string{fmt.Sprintf("%d request %s %s dalam %d ms → %s", n, method, path, elapsed.Milliseconds(), formatStatuses(statuses, errs))}
	switch {
	case got429 > 0 && retryAfter != "":
		c.Status, c.Severity = StatusPass, SevInfo
		c.Evidence = append(c.Evidence, fmt.Sprintf("%d dijawab 429 dengan Retry-After: %s", got429, retryAfter))
	case got429 > 0:
		c.Status, c.Severity = StatusWarn, SevInfo
		c.Evidence = append(c.Evidence, fmt.Sprintf("%d dijawab 429, tetapi tanpa header Retry-After.", got429))
		c.Suggestion = "Sertakan Retry-After pada jawaban 429 agar klien tahu kapan boleh mencoba lagi."
	case limitHdr > 0:
		c.Status, c.Severity = StatusDeclared, SevInfo
		c.Evidence = append(c.Evidence, "Header RateLimit-* dikirim; batasnya lebih tinggi dari jumlah burst uji.")
	case errs == n:
		c.Status, c.Severity = StatusSkipped, SevInfo
		c.Evidence = append(c.Evidence, "Semua request gagal; rate limit tidak bisa dinilai.")
	default:
		c.Status, c.Severity = StatusFail, SevWarning
		if id == "LIVE-AUTH-LIMIT" {
			c.Severity = SevCritical
		}
		c.Evidence = append(c.Evidence, "Tidak ada jawaban 429 — rate limit tidak teramati pada burst ini.")
		c.Suggestion = "Pasang limiter per IP (API ±100/menit, login ±5/menit) di aplikasi atau reverse proxy, balas 429 + Retry-After."
	}
	return c
}

func formatStatuses(m map[int]int, errs int) string {
	var parts []string
	for _, code := range []int{200, 201, 204, 301, 302, 304, 400, 401, 403, 404, 405, 422, 429, 500, 502, 503} {
		if v := m[code]; v > 0 {
			parts = append(parts, fmt.Sprintf("%d×%d", v, code))
			delete(m, code)
		}
	}
	for code, v := range m {
		parts = append(parts, fmt.Sprintf("%d×%d", v, code))
	}
	if errs > 0 {
		parts = append(parts, fmt.Sprintf("%d×gagal", errs))
	}
	if len(parts) == 0 {
		return "tanpa jawaban"
	}
	return strings.Join(parts, ", ")
}
