package secaudit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Uji serangan (DAST): setiap endpoint GET yang aman (dan rute login) dikirimi payload yang biasa dipakai
// peretas — SQL/NoSQL injection, XSS, path traversal, command injection, SSTI, open redirect, CRLF, token JWT
// palsu, akses tanpa login, berkas sensitif — lalu responsnya dinilai: galat basis data, isi /etc/passwd,
// hasil perintah, payload terpantul di HTML, jeda waktu, atau login yang tembus. Rute mutasi (POST/PUT/PATCH)
// hanya diserang bila AttackMutations dinyalakan; DELETE tidak pernah dikirim.

// Kategori serangan (Attack.Category).
const (
	AtkSQLi       = "sqli"
	AtkNoSQLi     = "nosqli"
	AtkXSS        = "xss"
	AtkTraversal  = "traversal"
	AtkCmdi       = "cmdi"
	AtkSSTI       = "ssti"
	AtkRedirect   = "redirect"
	AtkCRLF       = "crlf"
	AtkAuthBypass = "auth_bypass"
	AtkJWT        = "jwt"
	AtkAnon       = "anon_access"
	AtkExposure   = "exposure"
	AtkHostHeader = "host_header"
	AtkTrace      = "trace"
	AtkBodyLimit  = "body_limit"
)

// Vonis satu serangan.
const (
	VerdictVulnerable = "vulnerable" // terbukti tembus
	VerdictSuspect    = "suspect"    // tanda mencurigakan (mis. 5xx), perlu dicek manual
	VerdictSafe       = "safe"       // ditangani dengan benar
	VerdictBlocked    = "blocked"    // ditahan rate limit / WAF sehingga tidak bisa dinilai
	VerdictError      = "error"      // request gagal
)

// Attack = satu serangan yang dikirim beserta hasilnya.
type Attack struct {
	Category  string `json:"category"`
	Route     string `json:"route"` // pola rute yang diserang, mis. /api/items/:id
	Method    string `json:"method"`
	Path      string `json:"path"`
	Payload   string `json:"payload"`
	Status    int    `json:"status,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
	Verdict   string `json:"verdict"`
	Note      string `json:"note,omitempty"`
}

const (
	maxAttackTargets  = 60
	maxTimeTargets    = 6
	attackConcurrency = 4
	maxAttacksStored  = 2500
	sleepSeconds      = 3
	probeMarker       = "vbsk7331"
	sstiExpr          = "7*7191"
	sstiResult        = "50337"
)

var (
	passwdRe   = regexp.MustCompile(`root:[x*]?:0:0:`)
	winIniRe   = regexp.MustCompile(`(?i)\[(fonts|extensions)\]`)
	cmdOutRe   = regexp.MustCompile(`uid=\d+\([a-z0-9_-]+\) gid=\d+`)
	tokenRe    = regexp.MustCompile(`(?i)"(access_?token|token|jwt|id_token|session)"\s*:\s*"[^"]{10,}`)
	mongoErrRe = regexp.MustCompile(`(?i)(MongoError|MongoServerError|CastError|\$ne|BSON|E11000|unknown operator)`)
	adminish   = regexp.MustCompile(`(?i)/(admin|users?|accounts?|me|profile|settings|dashboard|internal|private|payments?|orders?|reports?)(/|$)`)
)

// injection payload per kategori untuk parameter query & segmen path.
var injectionPayloads = []struct {
	cat     string
	payload string
}{
	{AtkSQLi, `'`},
	{AtkSQLi, `1' OR '1'='1' -- `},
	{AtkSQLi, `1 UNION SELECT NULL,NULL,NULL--`},
	{AtkXSS, `"><svg/onload=alert('` + probeMarker + `')>`},
	{AtkTraversal, `../../../../../../../../etc/passwd`},
	{AtkTraversal, `....//....//....//....//etc/passwd`},
	{AtkCmdi, `;id`},
	{AtkCmdi, `$(id)`},
	{AtkSSTI, `{{` + sstiExpr + `}}${` + sstiExpr + `}<%= ` + sstiExpr + ` %>`},
}

// Rute yang menolak anonim (401/403) biasanya menolak sebelum handler; cukup payload ringkas untuk menguji
// lapisan autentikasinya, sehingga kuota rate limit dipakai untuk rute yang benar-benar memproses input.
var protectedPayloads = []struct {
	cat     string
	payload string
}{
	{AtkSQLi, `'`},
	{AtkTraversal, `../../../../../../../../etc/passwd`},
}

var timePayloads = []string{
	`1' AND SLEEP(` + strconv.Itoa(sleepSeconds) + `)-- `,
	`1' AND (SELECT 1 FROM pg_sleep(` + strconv.Itoa(sleepSeconds) + `))::text='1`,
	`1;SELECT pg_sleep(` + strconv.Itoa(sleepSeconds) + `);--`,
	`1);WAITFOR DELAY '0:0:` + strconv.Itoa(sleepSeconds) + `'--`,
}

// Parameter query yang umum dibaca handler (daftar, cari, urut, filter).
var commonParams = []string{"id", "q", "search", "keyword", "name", "sort", "order", "filter", "category", "type", "status", "file", "path"}

var redirectParams = []string{"redirect", "url", "next", "return", "returnUrl", "redirect_uri", "continue", "dest"}

// Berkas yang tidak boleh terbuka di server web.
var sensitiveFiles = []struct {
	path string
	re   *regexp.Regexp
	what string
}{
	{"/.env", regexp.MustCompile(`(?m)^(APP_KEY|DB_PASS(WORD)?|DATABASE_URL|SECRET|JWT_SECRET|AWS_SECRET)[A-Z_]*=`), "berkas .env (rahasia)"},
	{"/.git/config", regexp.MustCompile(`\[core\]`), "konfigurasi git"},
	{"/.git/HEAD", regexp.MustCompile(`^ref: refs/`), "repositori git"},
	{"/config.json", regexp.MustCompile(`(?i)"(password|secret|api_?key)"\s*:`), "config.json berisi rahasia"},
	{"/backup.sql", regexp.MustCompile(`(?i)(CREATE TABLE|INSERT INTO)`), "dump basis data"},
	{"/dump.sql", regexp.MustCompile(`(?i)(CREATE TABLE|INSERT INTO)`), "dump basis data"},
	{"/phpinfo.php", regexp.MustCompile(`(?i)phpinfo\(\)|PHP Version`), "phpinfo"},
	{"/server-status", regexp.MustCompile(`(?i)Apache Server Status`), "status Apache"},
	{"/actuator/env", regexp.MustCompile(`(?i)"propertySources"`), "Spring actuator"},
	{"/debug/pprof/", regexp.MustCompile(`(?i)Types of profiles available`), "Go pprof"},
	{"/debug/vars", regexp.MustCompile(`"memstats"`), "Go expvar"},
	{"/.DS_Store", regexp.MustCompile("Bud1"), ".DS_Store"},
	{"/wp-config.php.bak", regexp.MustCompile(`DB_PASSWORD`), "cadangan wp-config"},
	{"/storage/logs/laravel.log", regexp.MustCompile(`(?i)(local\.ERROR|Stack trace)`), "log Laravel"},
}

type attackTarget struct {
	path     string // pola rute, mis. /api/items/:id
	concrete string // /api/items/1
	params   bool
	base     probeResp
}

type attacker struct {
	p        *prober
	authHdr  map[string]string
	loopback bool
	log      func(level, msg string)
	deadline time.Time // batas waktu fase serangan; sesudahnya 429 tidak lagi ditunggu
	mu       sync.Mutex
	results  []Attack
	blocked  int
	paused   time.Time // semua serangan menunggu sampai waktu ini (rate limit server tercapai)
	waits    int
}

const (
	defaultAttackBudget = 8 * time.Minute
	maxRetryAfter       = 65 * time.Second
)

func (a *attacker) note(level, format string, args ...any) {
	if a.log != nil {
		a.log(level, fmt.Sprintf(format, args...))
	}
}

// waitPause menahan request selama server sedang membatasi (429), agar kuota tidak terbuang.
func (a *attacker) waitPause(ctx context.Context) {
	a.mu.Lock()
	until := a.paused
	a.mu.Unlock()
	if d := time.Until(until); d > 0 {
		select {
		case <-ctx.Done():
		case <-time.After(d):
		}
	}
}

func (a *attacker) add(at Attack) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if at.Verdict == VerdictBlocked {
		a.blocked++
	}
	if len(a.results) < maxAttacksStored {
		a.results = append(a.results, at)
	}
}

// send mengirim satu request. Jawaban 429 menghentikan sementara SEMUA serangan sesuai Retry-After (kuota
// server habis), lalu request diulang — sampai batas waktu fase serangan habis; sesudahnya dicatat "ditahan".
func (a *attacker) send(ctx context.Context, method, target string, body string, hdr map[string]string) probeResp {
	h := map[string]string{}
	for k, v := range a.authHdr {
		h[k] = v
	}
	for k, v := range hdr {
		h[k] = v
	}
	var r probeResp
	for attempt := 0; attempt < 4; attempt++ {
		a.waitPause(ctx)
		var rdr io.Reader
		if body != "" {
			rdr = strings.NewReader(body)
		}
		r = a.p.do(ctx, method, target, rdr, h)
		if r.status != http.StatusTooManyRequests || ctx.Err() != nil || time.Now().After(a.deadline) {
			return r
		}
		wait := 2 * time.Second
		if s, err := strconv.Atoi(strings.TrimSpace(r.headers.Get("Retry-After"))); err == nil && s > 0 {
			wait = time.Duration(s)*time.Second + 300*time.Millisecond
		}
		wait = min(wait, maxRetryAfter)
		if until := time.Now().Add(wait); until.After(a.deadline) {
			return r
		}
		a.mu.Lock()
		if time.Now().Add(wait).After(a.paused) {
			if time.Until(a.paused) <= 0 {
				a.waits++
				a.note("warn", "Rate limit server tercapai — serangan dijeda %d dtk lalu dilanjutkan", int(wait.Seconds()))
			}
			a.paused = time.Now().Add(wait)
		}
		a.mu.Unlock()
	}
	return r
}

// rawURL menyusun URL dengan path & query mentah (payload tidak di-encode ulang).
func (a *attacker) rawURL(path, rawQuery string) string {
	if path == "" || path[0] != '/' {
		path = "/" + path
	}
	s := a.p.base.Scheme + "://" + a.p.base.Host + a.p.base.EscapedPath() + path
	if rawQuery != "" {
		s += "?" + rawQuery
	}
	return s
}

func shortPath(target string) string {
	if u, err := url.Parse(target); err == nil {
		return u.RequestURI()
	}
	return target
}

func isHTML(h http.Header) bool {
	return strings.Contains(strings.ToLower(h.Get("Content-Type")), "text/html")
}

func isJSON(h http.Header) bool {
	return strings.Contains(strings.ToLower(h.Get("Content-Type")), "json")
}

// judge menilai respons payload injeksi terhadap baseline rutenya.
func judge(cat, payload string, r, base probeResp) (verdict, note string) {
	if r.err != nil {
		return VerdictError, r.err.Error()
	}
	if r.status == http.StatusTooManyRequests {
		return VerdictBlocked, "ditahan rate limit"
	}
	if m := sqlErrorRe.FindString(r.body); m != "" && !sqlErrorRe.MatchString(base.body) {
		if cat == AtkSQLi || cat == AtkNoSQLi {
			return VerdictVulnerable, fmt.Sprintf("galat basis data bocor: %q", m)
		}
		return VerdictSuspect, fmt.Sprintf("galat basis data bocor: %q", m)
	}
	switch cat {
	case AtkXSS:
		if strings.Contains(r.body, payload) && isHTML(r.headers) {
			return VerdictVulnerable, "payload terpantul utuh di HTML (reflected XSS)"
		}
		if strings.Contains(r.body, payload) {
			return VerdictSafe, "terpantul di JSON — aman selama frontend meng-escape"
		}
	case AtkTraversal:
		if passwdRe.MatchString(r.body) || winIniRe.MatchString(r.body) {
			return VerdictVulnerable, "isi berkas sistem terbaca (path traversal)"
		}
	case AtkCmdi:
		if m := cmdOutRe.FindString(r.body); m != "" {
			return VerdictVulnerable, fmt.Sprintf("hasil perintah shell terlihat: %q", m)
		}
	case AtkSSTI:
		if strings.Contains(r.body, sstiResult) && !strings.Contains(base.body, sstiResult) {
			return VerdictVulnerable, "ekspresi template dievaluasi server (" + sstiExpr + " = " + sstiResult + ")"
		}
	case AtkNoSQLi:
		if m := mongoErrRe.FindString(r.body); m != "" && !mongoErrRe.MatchString(base.body) {
			return VerdictSuspect, fmt.Sprintf("operator NoSQL memicu galat: %q", m)
		}
	}
	if m := stackTraceRe.FindString(r.body); m != "" && !stackTraceRe.MatchString(base.body) {
		return VerdictSuspect, fmt.Sprintf("stack trace bocor: %q", m)
	}
	if r.status >= 500 && base.status < 500 {
		return VerdictSuspect, fmt.Sprintf("HTTP %d (baseline %d) — input berbahaya tidak ditangani", r.status, base.status)
	}
	return VerdictSafe, ""
}

// pickTargets memilih rute GET yang aman diserang; rute berparameter dan rute data lebih dulu.
func pickTargets(routes []RouteRef) []attackTarget {
	seen := map[string]bool{}
	var out []attackTarget
	for _, r := range routes {
		m := strings.ToUpper(r.Method)
		if m != http.MethodGet || r.Path == "" || strings.Contains(r.Path, "*") || unsafeLivePath.MatchString(r.Path) || seen[r.Path] {
			continue
		}
		seen[r.Path] = true
		out = append(out, attackTarget{path: r.Path, concrete: concrete(r.Path), params: paramSegRe.MatchString(r.Path)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].params != out[j].params {
			return out[i].params
		}
		return strings.HasPrefix(out[i].path, "/api") && !strings.HasPrefix(out[j].path, "/api")
	})
	if len(out) > maxAttackTargets {
		out = out[:maxAttackTargets]
	}
	if len(out) == 0 {
		out = []attackTarget{{path: "/", concrete: "/"}}
	}
	return out
}

// withPayloadInPath mengganti parameter rute pertama dengan payload (ter-escape sebagai segmen path).
func withPayloadInPath(pattern, payload string) string {
	done := false
	return paramSegRe.ReplaceAllStringFunc(pattern, func(string) string {
		if done {
			return "1"
		}
		done = true
		return url.PathEscape(payload)
	})
}

func payloadQuery(payload string) string {
	v := url.Values{}
	for _, k := range commonParams {
		v.Set(k, payload)
	}
	return v.Encode()
}

// runAttacks menjalankan seluruh uji serangan dan mengembalikan daftar serangan beserta ringkasan per kontrol.
func runAttacks(ctx context.Context, p *prober, opts LiveOptions) ([]Attack, []Check, []string) {
	budget := opts.AttackBudget
	if budget <= 0 {
		budget = defaultAttackBudget
	}
	a := &attacker{p: p, authHdr: authHeaders(opts.AuthHeader), loopback: isLoopbackHost(p.base), log: opts.OnLog, deadline: time.Now().Add(budget)}
	targets := pickTargets(opts.Routes)

	// Baseline: jawaban normal tiap rute (dengan login bila header otorisasi diisi).
	protected := 0
	for i := range targets {
		targets[i].base = a.send(ctx, http.MethodGet, a.rawURL(targets[i].concrete, ""), "", nil)
		if s := targets[i].base.status; s == http.StatusUnauthorized || s == http.StatusForbidden {
			protected++
		}
	}
	var notes []string
	a.note("info", "%d endpoint GET jadi sasaran (%d meminta login)", len(targets), protected)
	if protected > 0 && opts.AuthHeader == "" {
		notes = append(notes, fmt.Sprintf("%d dari %d endpoint GET meminta login (401/403): injeksi pada endpoint itu hanya menguji lapisan autentikasi. Isi header otorisasi (token akun uji) di checklist agar isi endpoint ikut diserang.", protected, len(targets)))
	}

	// 1. Injeksi lewat query & segmen path ke semua target (paralel terbatas).
	type job struct {
		t      *attackTarget
		cat    string
		pl     string
		target string
	}
	var jobs []job
	for i := range targets {
		t := &targets[i]
		set := injectionPayloads
		if t.base.status == http.StatusUnauthorized || t.base.status == http.StatusForbidden {
			set = protectedPayloads
		}
		for _, ip := range set {
			jobs = append(jobs, job{t, ip.cat, ip.payload, a.rawURL(t.concrete, payloadQuery(ip.payload))})
			if t.params {
				jobs = append(jobs, job{t, ip.cat, ip.payload, a.rawURL(withPayloadInPath(t.path, ip.payload), "")})
			}
		}
		if len(set) == len(protectedPayloads) {
			continue
		}
		// NoSQL: operator di nama parameter (id[$ne]=…), CRLF di nilai mentah, open redirect di parameter tujuan.
		jobs = append(jobs,
			job{t, AtkNoSQLi, "id[$ne]=1&q[$regex]=.*", a.rawURL(t.concrete, "id[$ne]=1&q[$regex]=.*&search[$gt]=")},
			job{t, AtkCRLF, "%0d%0aSet-Cookie:" + probeMarker + "=1", a.rawURL(t.concrete, "q=x%0d%0aSet-Cookie:"+probeMarker+"=1&lang=x%0d%0aSet-Cookie:"+probeMarker+"=1")},
		)
		rv := url.Values{}
		for _, k := range redirectParams {
			rv.Set(k, "https://vibeasik-probe.invalid/x")
		}
		jobs = append(jobs, job{t, AtkRedirect, "https://vibeasik-probe.invalid/x", a.rawURL(t.concrete, rv.Encode())})
	}
	sem := make(chan struct{}, attackConcurrency)
	var wg sync.WaitGroup
	for _, j := range jobs {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(j job) {
			defer wg.Done()
			defer func() { <-sem }()
			r := a.send(ctx, http.MethodGet, j.target, "", nil)
			at := Attack{Category: j.cat, Route: j.t.path, Method: http.MethodGet, Path: shortPath(j.target), Payload: j.pl, Status: r.status, LatencyMs: r.latency.Milliseconds()}
			switch j.cat {
			case AtkCRLF:
				at.Verdict = VerdictSafe
				for _, c := range r.headers.Values("Set-Cookie") {
					if strings.Contains(c, probeMarker) {
						at.Verdict, at.Note = VerdictVulnerable, "header baru tersisip lewat CRLF (response splitting)"
					}
				}
				if r.err != nil {
					at.Verdict, at.Note = VerdictError, r.err.Error()
				}
			case AtkRedirect:
				at.Verdict = VerdictSafe
				if loc := r.headers.Get("Location"); r.status >= 300 && r.status < 400 && strings.HasPrefix(loc, "https://vibeasik-probe.invalid") {
					at.Verdict, at.Note = VerdictVulnerable, "redirect ke domain asing: "+loc
				}
				if r.err != nil {
					at.Verdict, at.Note = VerdictError, r.err.Error()
				}
			default:
				at.Verdict, at.Note = judge(j.cat, j.pl, r, j.t.base)
			}
			a.add(at)
		}(j)
	}
	wg.Wait()

	// 2. SQL injection berbasis waktu: jeda ≥ sleepSeconds dibanding baseline, dikonfirmasi sekali lagi.
	n := 0
	for i := range targets {
		t := &targets[i]
		if n >= maxTimeTargets || ctx.Err() != nil || t.base.err != nil || t.base.status == http.StatusUnauthorized || t.base.status == http.StatusForbidden {
			continue
		}
		n++
		for _, pl := range timePayloads {
			target := a.rawURL(t.concrete, payloadQuery(pl))
			if t.params {
				target = a.rawURL(withPayloadInPath(t.path, pl), "")
			}
			r := a.send(ctx, http.MethodGet, target, "", nil)
			at := Attack{Category: AtkSQLi, Route: t.path, Method: http.MethodGet, Path: shortPath(target), Payload: pl, Status: r.status, LatencyMs: r.latency.Milliseconds(), Verdict: VerdictSafe}
			delay := r.latency - t.base.latency
			if r.err == nil && delay >= time.Duration(sleepSeconds)*time.Second-300*time.Millisecond {
				again := a.send(ctx, http.MethodGet, target, "", nil)
				if again.err == nil && again.latency-t.base.latency >= time.Duration(sleepSeconds)*time.Second-300*time.Millisecond {
					at.Verdict, at.Note = VerdictVulnerable, fmt.Sprintf("respons tertunda %.1f dtk dua kali (time-based blind SQLi)", again.latency.Seconds())
				} else {
					at.Verdict, at.Note = VerdictSuspect, fmt.Sprintf("sekali tertunda %.1f dtk, tidak terulang", r.latency.Seconds())
				}
			} else if r.err == nil {
				at.Verdict, at.Note = judge(AtkSQLi, pl, r, t.base)
			} else {
				at.Verdict, at.Note = VerdictError, r.err.Error()
			}
			a.add(at)
		}
	}

	// 3. Rute login: bypass dengan SQLi / operator NoSQL, dan batas ukuran body.
	for i, ap := range opts.AuthPaths {
		if i >= 2 || ctx.Err() != nil {
			break
		}
		a.loginAttacks(ctx, ap)
	}

	// 4. Akses tanpa login & token JWT palsu ke rute yang terlihat privat.
	a.accessAttacks(ctx, targets, opts.AuthHeader != "")

	// 5. Berkas sensitif, TRACE, dan Host header.
	a.serverAttacks(ctx)

	if a.blocked > 0 {
		notes = append(notes, fmt.Sprintf("%d serangan tetap ditahan rate limit (429) setelah batas waktu uji habis, sehingga tidak bisa dinilai. Rate limit melindungi; untuk uji menyeluruh longgarkan limit di server lokal lalu ulangi.", a.blocked))
	}
	if a.waits > 0 {
		notes = append(notes, fmt.Sprintf("Serangan dijeda %d kali karena rate limit server; semua request diulang setelah jeda.", a.waits))
	}
	return a.results, attackChecks(a.results, len(targets)), notes
}

func authHeaders(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(strings.ToLower(raw), "cookie:") {
		return map[string]string{"Cookie": strings.TrimSpace(raw[len("cookie:"):])}
	}
	if strings.HasPrefix(strings.ToLower(raw), "authorization:") {
		raw = strings.TrimSpace(raw[len("authorization:"):])
	}
	if !strings.Contains(raw, " ") {
		raw = "Bearer " + raw
	}
	return map[string]string{"Authorization": raw}
}

var loginFields = []string{"email", "username", "identifier", "login", "user", "phone"}

func loginBody(v any) string {
	m := map[string]any{"password": v}
	for _, f := range loginFields {
		m[f] = v
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func (a *attacker) loginAttacks(ctx context.Context, path string) {
	target := a.p.urlFor(path, nil)
	cases := []struct {
		cat, label, body string
	}{
		{AtkAuthBypass, `' OR '1'='1' -- `, loginBody(`admin' OR '1'='1' -- `)},
		{AtkAuthBypass, `" OR ""="`, loginBody(`" OR ""="`)},
		{AtkNoSQLi, `{"$ne": null}`, loginBody(map[string]any{"$ne": nil})},
		{AtkNoSQLi, `{"$gt": ""}`, loginBody(map[string]any{"$gt": ""})},
	}
	for _, c := range cases {
		r := a.send(ctx, http.MethodPost, target, c.body, map[string]string{"Content-Type": "application/json"})
		at := Attack{Category: c.cat, Route: path, Method: http.MethodPost, Path: path, Payload: c.label, Status: r.status, LatencyMs: r.latency.Milliseconds()}
		switch {
		case r.err != nil:
			at.Verdict, at.Note = VerdictError, r.err.Error()
		case r.status == http.StatusTooManyRequests:
			at.Verdict, at.Note = VerdictBlocked, "ditahan rate limit"
		case r.status >= 200 && r.status < 300 && (tokenRe.MatchString(r.body) || len(r.headers.Values("Set-Cookie")) > 0):
			at.Verdict, at.Note = VerdictVulnerable, "login tembus tanpa kredensial (token/sesi diberikan)"
		case sqlErrorRe.MatchString(r.body):
			at.Verdict, at.Note = VerdictVulnerable, fmt.Sprintf("galat basis data bocor: %q", sqlErrorRe.FindString(r.body))
		case mongoErrRe.MatchString(r.body):
			at.Verdict, at.Note = VerdictSuspect, fmt.Sprintf("operator NoSQL memicu galat: %q", mongoErrRe.FindString(r.body))
		case r.status >= 500:
			at.Verdict, at.Note = VerdictSuspect, fmt.Sprintf("HTTP %d — input login tidak ditangani", r.status)
		default:
			at.Verdict = VerdictSafe
		}
		a.add(at)
	}

	// Body raksasa: server sebaiknya menolak (413) sebelum membaca semuanya. Hanya ke server lokal.
	if !a.loopback || ctx.Err() != nil {
		return
	}
	big := `{"email":"` + strings.Repeat("A", 11<<20) + `","password":"x"}`
	r := a.send(ctx, http.MethodPost, target, big, map[string]string{"Content-Type": "application/json"})
	at := Attack{Category: AtkBodyLimit, Route: path, Method: http.MethodPost, Path: path, Payload: "body JSON 11 MB", Status: r.status, LatencyMs: r.latency.Milliseconds()}
	switch {
	case r.err != nil || r.status == http.StatusRequestEntityTooLarge:
		at.Verdict, at.Note = VerdictSafe, "body besar ditolak"
	case r.status == http.StatusTooManyRequests:
		at.Verdict, at.Note = VerdictBlocked, "ditahan rate limit"
	default:
		at.Verdict, at.Note = VerdictSuspect, fmt.Sprintf("body 11 MB dibaca server (HTTP %d) — batas ukuran body belum dipasang", r.status)
	}
	a.add(at)
}

// forgeJWT membuat token tanpa tanda tangan (alg none) dan token HS256 bersandi lemah "secret".
func forgeJWT(alg string) string {
	enc := base64.RawURLEncoding
	head := enc.EncodeToString([]byte(`{"alg":"` + alg + `","typ":"JWT"}`))
	now := time.Now().Unix()
	claims := enc.EncodeToString([]byte(fmt.Sprintf(`{"sub":"1","id":1,"user_id":1,"role":"admin","is_admin":true,"exp":%d,"iat":%d}`, now+3600, now)))
	if alg == "none" {
		return head + "." + claims + "."
	}
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(head + "." + claims))
	return head + "." + claims + "." + enc.EncodeToString(mac.Sum(nil))
}

func (a *attacker) accessAttacks(ctx context.Context, targets []attackTarget, haveAuth bool) {
	jwtTried := 0
	for i := range targets {
		t := &targets[i]
		if ctx.Err() != nil || !adminish.MatchString(t.path) || strings.Contains(t.path, "/auth/") {
			continue
		}
		// Tanpa header login sama sekali.
		r := a.p.do(ctx, http.MethodGet, a.rawURL(t.concrete, ""), nil, nil)
		at := Attack{Category: AtkAnon, Route: t.path, Method: http.MethodGet, Path: t.concrete, Payload: "tanpa token", Status: r.status, LatencyMs: r.latency.Milliseconds(), Verdict: VerdictSafe}
		switch {
		case r.err != nil:
			at.Verdict, at.Note = VerdictError, r.err.Error()
		case r.status == http.StatusTooManyRequests:
			at.Verdict = VerdictBlocked
		case r.status >= 200 && r.status < 300 && isJSON(r.headers) && len(strings.TrimSpace(r.body)) > 2:
			at.Verdict, at.Note = VerdictVulnerable, "data rute privat terbuka tanpa login"
		}
		a.add(at)

		// Token palsu ke rute yang menolak anonim.
		if (r.status == http.StatusUnauthorized || r.status == http.StatusForbidden) && jwtTried < 4 {
			jwtTried++
			for _, alg := range []string{"none", "HS256"} {
				fr := a.p.do(ctx, http.MethodGet, a.rawURL(t.concrete, ""), nil, map[string]string{"Authorization": "Bearer " + forgeJWT(alg)})
				label := "JWT alg=none"
				if alg == "HS256" {
					label = `JWT HS256 kunci "secret"`
				}
				ft := Attack{Category: AtkJWT, Route: t.path, Method: http.MethodGet, Path: t.concrete, Payload: label, Status: fr.status, LatencyMs: fr.latency.Milliseconds(), Verdict: VerdictSafe}
				if fr.err != nil {
					ft.Verdict, ft.Note = VerdictError, fr.err.Error()
				} else if fr.status >= 200 && fr.status < 300 {
					ft.Verdict, ft.Note = VerdictVulnerable, "token palsu diterima — tanda tangan JWT tidak diverifikasi / kunci lemah"
				}
				a.add(ft)
			}
		}
	}
	_ = haveAuth
}

func (a *attacker) serverAttacks(ctx context.Context) {
	for _, f := range sensitiveFiles {
		if ctx.Err() != nil {
			return
		}
		r := a.p.do(ctx, http.MethodGet, a.rawURL(f.path, ""), nil, nil)
		at := Attack{Category: AtkExposure, Route: f.path, Method: http.MethodGet, Path: f.path, Payload: f.what, Status: r.status, LatencyMs: r.latency.Milliseconds(), Verdict: VerdictSafe}
		if r.err != nil {
			at.Verdict, at.Note = VerdictError, r.err.Error()
		} else if r.status == http.StatusOK && f.re.MatchString(r.body) {
			at.Verdict, at.Note = VerdictVulnerable, f.what+" bisa diunduh siapa saja"
		}
		a.add(at)
	}
	// TRACE memantulkan request (cross-site tracing).
	r := a.p.do(ctx, "TRACE", a.rawURL("/", ""), nil, map[string]string{"X-Vbsk-Probe": probeMarker})
	at := Attack{Category: AtkTrace, Route: "/", Method: "TRACE", Path: "/", Payload: "method TRACE", Status: r.status, LatencyMs: r.latency.Milliseconds(), Verdict: VerdictSafe}
	if r.err == nil && r.status == http.StatusOK && strings.Contains(r.body, probeMarker) {
		at.Verdict, at.Note = VerdictSuspect, "TRACE aktif dan memantulkan header request"
	}
	a.add(at)
	// Host header palsu terpantul di tautan / redirect (password reset poisoning, cache poisoning).
	hr := a.p.do(ctx, http.MethodGet, a.rawURL("/", ""), nil, map[string]string{"X-Forwarded-Host": "vibeasik-probe.invalid"})
	ht := Attack{Category: AtkHostHeader, Route: "/", Method: http.MethodGet, Path: "/", Payload: "X-Forwarded-Host: vibeasik-probe.invalid", Status: hr.status, LatencyMs: hr.latency.Milliseconds(), Verdict: VerdictSafe}
	if hr.err == nil && (strings.Contains(hr.headers.Get("Location"), "vibeasik-probe.invalid") || strings.Contains(hr.body, "//vibeasik-probe.invalid")) {
		ht.Verdict, ht.Note = VerdictVulnerable, "host palsu dipakai untuk membuat tautan/redirect"
	}
	a.add(ht)
}

// attackMeta = judul, kategori kontrol, keparahan, dan saran per kategori serangan.
var attackMeta = map[string]struct {
	id, cat, title, sev, fix string
}{
	AtkSQLi:       {"LIVE-SQLI", CatInjection, "SQL injection pada endpoint", SevCritical, "Pakai query berparameter/ORM untuk setiap input; jangan merangkai string SQL. Kirim pesan galat umum, bukan galat basis data."},
	AtkNoSQLi:     {"LIVE-NOSQLI", CatInjection, "NoSQL injection (operator $ne/$gt/$regex)", SevCritical, "Tolak objek/operator pada field yang seharusnya string; validasi tipe input sebelum query."},
	AtkXSS:        {"LIVE-XSS", CatXSS, "Reflected XSS", SevCritical, "Escape keluaran HTML (template auto-escape), validasi input, dan pasang Content-Security-Policy."},
	AtkTraversal:  {"LIVE-TRAVERSAL", CatTraversal, "Path traversal (baca berkas sistem)", SevCritical, "Jangan memakai input sebagai path berkas; gunakan filepath.Clean + cek prefix folder yang diizinkan, atau ID berkas dari basis data."},
	AtkCmdi:       {"LIVE-CMDI", CatInjection, "Command injection", SevCritical, "Jangan menjalankan shell dengan input pengguna; panggil program dengan argumen terpisah dan daftar nilai yang diizinkan."},
	AtkSSTI:       {"LIVE-SSTI", CatInjection, "Server-side template injection", SevCritical, "Jangan menyusun template dari input pengguna; kirim input sebagai data ke template yang tetap."},
	AtkRedirect:   {"LIVE-OPEN-REDIRECT", CatInput, "Open redirect", SevWarning, "Batasi tujuan redirect ke path relatif atau daftar domain yang diizinkan."},
	AtkCRLF:       {"LIVE-CRLF", CatHeaders, "CRLF / header injection", SevWarning, "Buang \\r\\n dari nilai yang dimasukkan ke header respons."},
	AtkAuthBypass: {"LIVE-LOGIN-BYPASS", CatAuth, "Login bypass dengan SQL injection", SevCritical, "Cek kredensial dengan query berparameter dan bandingkan hash password di kode, bukan di SQL."},
	AtkJWT:        {"LIVE-JWT-FORGERY", CatAuth, "Token JWT palsu (alg none / kunci lemah)", SevCritical, "Verifikasi tanda tangan dengan algoritma tetap (tolak alg none) dan kunci rahasia acak yang panjang dari env."},
	AtkAnon:       {"LIVE-ANON-ACCESS", CatAuth, "Rute privat bisa dibuka tanpa login", SevCritical, "Pasang middleware autentikasi (dan cek peran) pada grup rute admin/akun."},
	AtkExposure:   {"LIVE-EXPOSURE", CatTraversal, "Berkas sensitif terbuka (.env, .git, dump SQL)", SevCritical, "Jangan sajikan folder proyek sebagai static; blokir dotfile, dan pindahkan rahasia ke luar webroot."},
	AtkHostHeader: {"LIVE-HOST-HEADER", CatHeaders, "Host header injection", SevWarning, "Bangun URL absolut dari konfigurasi (PUBLIC_BASE_URL), bukan dari header Host/X-Forwarded-Host."},
	AtkTrace:      {"LIVE-TRACE", CatHeaders, "Method TRACE dimatikan", SevWarning, "Matikan method TRACE di server/proxy."},
	AtkBodyLimit:  {"LIVE-BODY-LIMIT", CatInput, "Batas ukuran body (DoS memori)", SevWarning, "Pasang batas body (mis. 1–4 MB) di server/proxy sehingga request raksasa ditolak 413."},
}

var attackOrder = []string{AtkSQLi, AtkNoSQLi, AtkAuthBypass, AtkXSS, AtkTraversal, AtkCmdi, AtkSSTI, AtkJWT, AtkAnon, AtkExposure, AtkRedirect, AtkCRLF, AtkHostHeader, AtkTrace, AtkBodyLimit}

// attackChecks merangkum serangan menjadi satu kontrol per kategori.
func attackChecks(results []Attack, targets int) []Check {
	by := map[string][]Attack{}
	for _, at := range results {
		by[at.Category] = append(by[at.Category], at)
	}
	var out []Check
	for _, cat := range attackOrder {
		meta := attackMeta[cat]
		list := by[cat]
		c := Check{ID: meta.id, Category: meta.cat, Title: meta.title, Source: "live", Suggestion: meta.fix}
		if len(list) == 0 {
			c.Status, c.Severity = StatusSkipped, SevInfo
			c.Evidence = []string{"Tidak ada sasaran yang cocok untuk serangan ini."}
			c.Suggestion = ""
			out = append(out, c)
			continue
		}
		var vuln, sus, blocked, safe int
		var ev []string
		paths := map[string]bool{}
		for _, at := range list {
			paths[at.Method+" "+at.Route] = true
			switch at.Verdict {
			case VerdictVulnerable:
				vuln++
				if len(ev) < 6 {
					ev = append(ev, fmt.Sprintf("%s %s → HTTP %d · %s", at.Method, at.Path, at.Status, at.Note))
				}
			case VerdictSuspect:
				sus++
			case VerdictBlocked:
				blocked++
			case VerdictSafe:
				safe++
			}
		}
		if vuln == 0 && sus > 0 {
			for _, at := range list {
				if at.Verdict == VerdictSuspect && len(ev) < 6 {
					ev = append(ev, fmt.Sprintf("%s %s → HTTP %d · %s", at.Method, at.Path, at.Status, at.Note))
				}
			}
		}
		summary := fmt.Sprintf("%d serangan ke %d sasaran: %d tembus, %d mencurigakan, %d aman, %d ditahan rate limit.", len(list), len(paths), vuln, sus, safe, blocked)
		switch {
		case vuln > 0:
			c.Status, c.Severity = StatusFail, meta.sev
		case sus > 0:
			c.Status, c.Severity = StatusWarn, SevWarning
		case safe == 0 || blocked > safe:
			c.Status, c.Severity = StatusSkipped, SevInfo
			summary += " Sebagian besar tidak bisa dinilai — ulangi dengan rate limit dilonggarkan."
		default:
			c.Status, c.Severity = StatusPass, SevInfo
			c.Suggestion = ""
		}
		c.Evidence = append([]string{summary}, ev...)
		out = append(out, c)
	}
	_ = targets
	return out
}
