package secaudit

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	maxStaticFiles    = 4000
	maxStaticFileSize = 512 << 10
	maxLocations      = 8
	maxRouteRefs      = 200
)

var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".nuxt": true, ".output": true, "dist": true,
	"build": true, "bin": true, ".cache": true, "tmp": true, "coverage": true, ".next": true,
	"testdata": true, "__pycache__": true, ".venv": true, "venv": true, ".dart_tool": true,
}

var serverExt = map[string]bool{".go": true, ".py": true, ".php": true, ".ts": true, ".js": true, ".mjs": true}

var (
	// File TS/JS hanya dihitung sebagai server bila memakai kerangka server.
	jsServerRe = regexp.MustCompile(`(?i)(require\(['"]express['"]\)|from ['"](express|fastify|hono|koa|@nestjs/[a-z]+|h3|elysia)['"]|defineEventHandler\s*\(|createServer\s*\()`)

	routeDeclRe = regexp.MustCompile(`(?i)\.(Get|Post|Put|Patch|Delete|All)\s*\(\s*["'` + "`" + `](/[^"'` + "`" + `]*)`)
	mutationRe  = regexp.MustCompile(`(?i)\.(Post|Put|Patch|Delete)\s*\(\s*["'` + "`" + `]/`)
	authRouteRe = regexp.MustCompile(`(?i)\.(Post|Put|Patch)\s*\(\s*["'` + "`" + `]([^"'` + "`" + `]*(login|sign-?in|authenticate|register|sign-?up|forgot|reset-?password|/otp|/2fa|/token)[^"'` + "`" + `]*)["'` + "`" + `]`)
	rateLimitRe = regexp.MustCompile(`(?i)(limiter\.New|rate[\s_-]?limit|rate\.(New)?Limiter|express-rate-limit|rateLimit\s*\(|\bthrottle|tollbooth|x/time/rate|ThrottlerGuard|flask[\s_-]?limiter|django_ratelimit|slowapi|token[\s_-]?bucket|middleware/limiter|RateLimiter::)`)
	headersRe   = regexp.MustCompile(`(?i)(helmet\s*\(|middleware/helmet|helmet\.New|secure\.New|X-Frame-Options|Content-Security-Policy|X-Content-Type-Options|Strict-Transport-Security|SecurityMiddleware|talisman)`)
	hstsRe      = regexp.MustCompile(`(?i)(Strict-Transport-Security|HSTSMaxAge|hsts\s*:|SECURE_HSTS_SECONDS)`)
	cspRe       = regexp.MustCompile(`(?i)(Content-Security-Policy|ContentSecurityPolicy|contentSecurityPolicy)`)
	corsRe      = regexp.MustCompile(`(?i)(cors\.New|middleware/cors|require\(['"]cors['"]\)|from ['"]cors['"]|CORSMiddleware|AllowOrigins|Access-Control-Allow-Origin|CORS_ALLOWED_ORIGINS)`)
	corsStarRe  = regexp.MustCompile(`(?i)(AllowOrigins\s*:\s*["']\*["']|AllowOrigins\s*:\s*\[\]string\{\s*["']\*["']|origin\s*:\s*["']\*["']|allow_origins\s*=\s*\[\s*["']\*["']|Access-Control-Allow-Origin["']?\s*,\s*["']\*["']|CORS_ALLOW_ALL_ORIGINS\s*=\s*True)`)
	corsCredsRe = regexp.MustCompile(`(?i)(AllowCredentials\s*:\s*true|credentials\s*:\s*true|allow_credentials\s*=\s*True)`)
	sqlConcatRe = regexp.MustCompile(`(?i)((db|tx|conn)\.(Raw|Exec|Query|QueryRow|Where)\s*\(\s*fmt\.Sprintf|\.query\s*\(\s*['"` + "`" + `]\s*(SELECT|UPDATE|DELETE|INSERT)[^'"` + "`" + `]*['"` + "`" + `]\s*\+|\.query\s*\(\s*` + "`" + `[^` + "`" + `]*(SELECT|UPDATE|DELETE|INSERT)[^` + "`" + `]*\$\{|\$(db|pdo|conn|mysqli)->query\s*\(\s*["'][^"']*(SELECT|UPDATE|DELETE|INSERT)[^"']*["']\s*\.|\.execute\s*\(\s*f["'][^"']*(SELECT|UPDATE|DELETE|INSERT)[^"']*\{|\.execute\s*\(\s*["'][^"']*(SELECT|UPDATE|DELETE|INSERT)[^"']*%s[^"']*["']\s*%)`)
	sqlParamRe  = regexp.MustCompile(`(?i)(\.Where\(\s*"[^"]*\?|\$1\b|\.prepare\s*\(|PreparedStatement|bindParam|bindValue|\.Find\(|\.First\(|prisma\.|sqlx\.|sqlc|:\w+\s*\)|%s"\s*,\s*\(|execute\(\s*["'][^"']*%s["']\s*,)`)
	cmdInjectRe = regexp.MustCompile(`(?i)(exec\.Command\s*\(\s*"(sh|bash|cmd)"\s*,\s*"-c"|child_process\.(exec|execSync)\s*\(|\bos\.system\s*\(|subprocess\.[a-z]+\([^)]*shell\s*=\s*True|\bshell_exec\s*\(|\bpassthru\s*\()`)
	validatorRe = regexp.MustCompile(`(?i)(go-playground/validator|validator\.New|validate\.Struct|binding:"required|ozzo-validation|from ['"]zod['"]|z\.object\(|joi\.|class-validator|express-validator|pydantic|BaseModel\)|\$request->validate\(|Validator::make|FormRequest|yup\.object)`)
	bodyLimitRe = regexp.MustCompile(`(?i)(BodyLimit\s*:|MaxBytesReader|express\.json\(\s*\{[^}]*limit|bodyParser\.json\(\s*\{[^}]*limit|DATA_UPLOAD_MAX_MEMORY_SIZE|client_max_body_size|post_max_size|bodyLimit\s*:)`)
	authMwRe    = regexp.MustCompile(`(?i)(jwtware|jwt\.New|middleware/keyauth|keyauth\.New|basicauth\.New|passport\.authenticate|requireAuth|authMiddleware|AuthMiddleware|authRequired|RequireAuth|isAuthenticated|@UseGuards|login_required|Depends\(\s*get_current_user|auth:sanctum|->middleware\(\s*['"]auth|verifyToken|ensureAuth)`)
	errorLeakRe = regexp.MustCompile(`(?i)("error"\s*:\s*err\.Error\(\)|fiber\.Map\{\s*"error"\s*:\s*err\.Error\(\)|res\.(status\(\d+\)\.)?(json|send)\(\s*\{?\s*(error|message)\s*:\s*(err|error)\.(message|stack)|traceback\.format_exc\(\)|->getTraceAsString\(\))`)
	debugModeRe = regexp.MustCompile(`(?i)(DEBUG\s*=\s*True|APP_DEBUG\s*=\s*true|app\.run\([^)]*debug\s*=\s*True|EnablePrintRoutes\s*:\s*true|display_errors\s*=\s*On)`)
)

// staticFacts = hasil pindai sumber satu direktori.
type staticFacts struct {
	files       int
	routes      int
	mutations   int
	authRoutes  []Location
	rateLimit   []Location
	headers     []Location
	hsts        []Location
	csp         []Location
	cors        []Location
	corsStar    []Location
	corsCreds   []Location
	sqlConcat   []Location
	sqlParam    int
	cmdInject   []Location
	validator   []Location
	bodyLimit   []Location
	authMw      []Location
	errorLeak   []Location
	debugMode   []Location
	serverFiles int
	routeRefs   []RouteRef // rute yang terbaca dari sumber (cadangan bila rute komponen belum disinkron)
}

func addLoc(list []Location, file string, line int, text string) []Location {
	if len(list) >= maxLocations {
		return list
	}
	for _, l := range list {
		if l.File == file && l.Line == line {
			return list
		}
	}
	t := strings.TrimSpace(text)
	if len(t) > 160 {
		t = t[:160] + "…"
	}
	return append(list, Location{File: file, Line: line, Snippet: t})
}

// scanSource memindai berkas sumber server di dir (relatif terhadap base untuk laporan).
func scanSource(dir, base string) staticFacts {
	var f staticFacts
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] || (strings.HasPrefix(d.Name(), ".") && path != dir) {
				return filepath.SkipDir
			}
			return nil
		}
		if f.files >= maxStaticFiles {
			return filepath.SkipAll
		}
		name := d.Name()
		// Berkas env/konfigurasi dibaca hanya untuk mode debug (nilai tidak pernah dilaporkan).
		if name == ".env" || strings.HasPrefix(name, ".env.") || name == "settings.py" || name == "php.ini" {
			if strings.HasSuffix(name, ".example") {
				return nil
			}
			scanDebugOnly(&f, path, rel(base, path))
			return nil
		}
		ext := filepath.Ext(name)
		if !serverExt[ext] || strings.HasSuffix(name, "_test.go") || strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") || strings.HasSuffix(name, ".d.ts") {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil || info.Size() > maxStaticFileSize {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		content := string(data)
		if (ext == ".ts" || ext == ".js" || ext == ".mjs") && !jsServerRe.MatchString(content) {
			return nil
		}
		f.files++
		f.serverFiles++
		scanLines(&f, rel(base, path), content)
		return nil
	})
	return f
}

func rel(base, path string) string {
	if r, err := filepath.Rel(base, path); err == nil {
		return filepath.ToSlash(r)
	}
	return filepath.ToSlash(path)
}

func scanDebugOnly(f *staticFacts, path, relPath string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	sc := bufio.NewScanner(file)
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
		if debugModeRe.MatchString(line) {
			// Hanya nama kunci yang dicatat, nilainya tidak.
			key, _, _ := strings.Cut(strings.TrimSpace(line), "=")
			f.debugMode = addLoc(f.debugMode, relPath, n, strings.TrimSpace(key)+"=…")
		}
	}
}

// Kata kunci murah (huruf kecil) per pola: regex hanya dijalankan bila salah satunya ada di baris.
// Tanpa saringan ini ±17 regex case-insensitive per baris membuat repo ratusan berkas butuh puluhan detik.
var (
	kwRoute     = []string{".get(", ".post(", ".put(", ".patch(", ".delete(", ".all("}
	kwMutation  = []string{".post(", ".put(", ".patch(", ".delete("}
	kwRateLimit = []string{"limit", "throttl", "slowapi", "tollbooth", "bucket", "x/time/rate", "rate."}
	kwHeaders   = []string{"helmet", "x-frame", "content-security", "x-content-type", "strict-transport", "secure.new", "securitymiddleware", "talisman"}
	kwHSTS      = []string{"strict-transport", "hsts"}
	kwCSP       = []string{"content-security", "contentsecuritypolicy"}
	kwCORS      = []string{"cors", "alloworigins", "access-control-allow-origin"}
	kwCORSStar  = []string{"*"}
	kwCreds     = []string{"credentials"}
	kwSQL       = []string{"sprintf", "select", "update", "delete", "insert"}
	kwSQLParam  = []string{"where(", "$1", "prepare", "bind", ".find(", ".first(", "prisma.", "sqlx", "sqlc", "execute("}
	kwCmd       = []string{"exec.command", "child_process", "os.system", "subprocess", "shell_exec", "passthru"}
	kwValidator = []string{"valid", "binding:", "zod", "z.object", "joi.", "pydantic", "basemodel", "formrequest", "yup."}
	kwBodyLimit = []string{"bodylimit", "maxbytesreader", "limit", "data_upload_max", "client_max_body", "post_max_size"}
	kwAuth      = []string{"auth", "jwt", "passport", "guard", "login_required", "verifytoken", "current_user"}
	kwErrLeak   = []string{"err.error()", "error.message", "err.message", "stack", "traceback", "gettraceasstring"}
	kwDebug     = []string{"debug", "enableprintroutes", "display_errors"}
)

func hasAny(lower string, kws []string) bool {
	for _, k := range kws {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

func scanLines(f *staticFacts, relPath, content string) {
	for i, line := range strings.Split(content, "\n") {
		n := i + 1
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "#") || strings.HasPrefix(trim, "*") {
			continue
		}
		lower := strings.ToLower(line)
		match := func(kws []string, re *regexp.Regexp) bool { return hasAny(lower, kws) && re.MatchString(line) }
		if hasAny(lower, kwRoute) {
			if m := routeDeclRe.FindStringSubmatch(line); m != nil {
				f.routes++
				if len(f.routeRefs) < maxRouteRefs {
					f.routeRefs = append(f.routeRefs, RouteRef{Method: strings.ToUpper(m[1]), Path: m[2]})
				}
			}
			if match(kwMutation, mutationRe) {
				f.mutations++
			}
			if authRouteRe.MatchString(line) {
				f.authRoutes = addLoc(f.authRoutes, relPath, n, line)
			}
		}
		if match(kwRateLimit, rateLimitRe) {
			f.rateLimit = addLoc(f.rateLimit, relPath, n, line)
		}
		if match(kwHeaders, headersRe) {
			f.headers = addLoc(f.headers, relPath, n, line)
		}
		if match(kwHSTS, hstsRe) {
			f.hsts = addLoc(f.hsts, relPath, n, line)
		}
		if match(kwCSP, cspRe) {
			f.csp = addLoc(f.csp, relPath, n, line)
		}
		if match(kwCORS, corsRe) {
			f.cors = addLoc(f.cors, relPath, n, line)
		}
		if match(kwCORSStar, corsStarRe) {
			f.corsStar = addLoc(f.corsStar, relPath, n, line)
		}
		if match(kwCreds, corsCredsRe) {
			f.corsCreds = addLoc(f.corsCreds, relPath, n, line)
		}
		if match(kwSQL, sqlConcatRe) {
			f.sqlConcat = addLoc(f.sqlConcat, relPath, n, line)
		} else if match(kwSQLParam, sqlParamRe) {
			f.sqlParam++
		}
		if match(kwCmd, cmdInjectRe) {
			f.cmdInject = addLoc(f.cmdInject, relPath, n, line)
		}
		if match(kwValidator, validatorRe) {
			f.validator = addLoc(f.validator, relPath, n, line)
		}
		if match(kwBodyLimit, bodyLimitRe) {
			f.bodyLimit = addLoc(f.bodyLimit, relPath, n, line)
		}
		if match(kwAuth, authMwRe) {
			f.authMw = addLoc(f.authMw, relPath, n, line)
		}
		if match(kwErrLeak, errorLeakRe) {
			f.errorLeak = addLoc(f.errorLeak, relPath, n, line)
		}
		if match(kwDebug, debugModeRe) {
			f.debugMode = addLoc(f.debugMode, relPath, n, line)
		}
	}
}

// corsComboWindow: "*" dan credentials dianggap satu konfigurasi CORS bila berada di berkas yang sama
// dan paling jauh sekian baris; pasangan lintas berkas/jauh biasanya teks penjelasan, bukan konfigurasi.
const corsComboWindow = 15

func corsCombo(star, creds []Location) []Location {
	var out []Location
	for _, s := range star {
		for _, c := range creds {
			d := s.Line - c.Line
			if s.File == c.File && d <= corsComboWindow && d >= -corsComboWindow {
				out = addLoc(out, s.File, s.Line, s.Snippet)
				out = addLoc(out, c.File, c.Line, c.Snippet)
			}
		}
	}
	return out
}

// staticChecks mengubah fakta sumber menjadi daftar pemeriksaan berbukti kode.
func staticChecks(f staticFacts) []Check {
	var out []Check
	if f.serverFiles == 0 {
		return []Check{{
			ID: "STATIC-NO-SERVER", Category: CatAuth, Title: "Tidak ada kode server yang bisa diperiksa",
			Status: StatusSkipped, Severity: SevInfo, Source: "static",
			Evidence:   []string{"Tidak ditemukan berkas router/backend (Go, Node server, Python, PHP)."},
			Suggestion: "Uji keamanan router berlaku untuk komponen backend. Untuk frontend, jalankan probe live ke URL-nya.",
		}}
	}

	// Rate limit (DDoS & brute force) — SEC-007.
	rl := Check{ID: "RATE-LIMIT", Category: CatRateLimit, Title: "Rate limiting (DDoS & brute-force)", Source: "static", GuidelineID: "SEC-007"}
	switch {
	case len(f.rateLimit) > 0:
		rl.Status, rl.Severity = StatusDeclared, SevInfo
		rl.Evidence = []string{"Middleware rate limit ditemukan di kode."}
		rl.Locations = f.rateLimit
		rl.Suggestion = "Buktikan dengan probe live: burst request harus dijawab 429 + Retry-After."
	case len(f.authRoutes) > 0:
		rl.Status, rl.Severity = StatusFail, SevCritical
		rl.Evidence = []string{"Ada rute login/registrasi tanpa rate limiter — rawan brute-force password."}
		rl.Locations = f.authRoutes
		rl.Suggestion = "Pasang limiter per IP: ±100 req/menit untuk API umum dan ±5 req/menit untuk rute autentikasi, balas 429 dengan Retry-After."
	default:
		rl.Status, rl.Severity = StatusFail, SevWarning
		rl.Evidence = []string{"Tidak ada middleware rate limit — satu klien bisa menghabiskan kapasitas server."}
		rl.Suggestion = "Pasang limiter global per IP pada router utama (mis. fiber limiter.New, express-rate-limit, slowapi), atau di reverse proxy (nginx limit_req, Cloudflare)."
	}
	out = append(out, rl)

	// Header keamanan.
	hd := Check{ID: "SEC-HEADERS", Category: CatHeaders, Title: "Header keamanan respons", Source: "static"}
	if len(f.headers) > 0 {
		hd.Status, hd.Severity = StatusDeclared, SevInfo
		hd.Evidence = []string{"Header keamanan / helmet dipasang di kode."}
		if len(f.csp) == 0 {
			hd.Status, hd.Severity = StatusWarn, SevWarning
			hd.Evidence = append(hd.Evidence, "Content-Security-Policy belum terlihat.")
		}
		hd.Locations = f.headers
	} else {
		hd.Status, hd.Severity = StatusFail, SevWarning
		hd.Evidence = []string{"Tidak ada X-Content-Type-Options, X-Frame-Options, atau CSP di kode."}
	}
	hd.Suggestion = "Pasang middleware helmet (fiber/helmet, helmet untuk Express, secure middleware) atau set header di reverse proxy."
	out = append(out, hd)

	// CORS.
	cr := Check{ID: "CORS", Category: CatCORS, Title: "CORS asal terbatas", Source: "static"}
	switch {
	case len(corsCombo(f.corsStar, f.corsCreds)) > 0:
		cr.Status, cr.Severity = StatusFail, SevCritical
		cr.Evidence = []string{"CORS mengizinkan semua asal (*) sekaligus credentials — situs lain bisa memakai sesi pengguna."}
		cr.Locations = corsCombo(f.corsStar, f.corsCreds)
	case len(f.corsStar) > 0:
		cr.Status, cr.Severity = StatusWarn, SevWarning
		cr.Evidence = []string{"CORS mengizinkan semua asal (*)."}
		cr.Locations = f.corsStar
	case len(f.cors) > 0:
		cr.Status, cr.Severity = StatusDeclared, SevInfo
		cr.Evidence = []string{"CORS dikonfigurasi dengan daftar asal."}
		cr.Locations = f.cors
	default:
		cr.Status, cr.Severity = StatusSkipped, SevInfo
		cr.Evidence = []string{"Tidak ada konfigurasi CORS (browser memblokir lintas asal secara bawaan)."}
	}
	cr.Suggestion = "Isi AllowOrigins dengan domain frontend yang sah saja; jangan pakai * bersama credentials."
	out = append(out, cr)

	// SQL injection.
	sq := Check{ID: "SQL-INJECTION", Category: CatInjection, Title: "Query berparameter (SQL injection)", Source: "static", GuidelineID: "audit:go:4"}
	if len(f.sqlConcat) > 0 {
		sq.Status, sq.Severity = StatusFail, SevCritical
		sq.Evidence = []string{"Query SQL disusun dengan penggabungan string / Sprintf — input pengguna bisa mengubah perintah SQL."}
		sq.Locations = f.sqlConcat
		sq.Suggestion = "Ganti dengan placeholder (? / $1 / :nama) atau query builder ORM; jangan menyisipkan nilai ke string SQL."
	} else {
		sq.Status, sq.Severity = StatusPass, SevInfo
		if f.sqlParam > 0 {
			sq.Evidence = []string{"Tidak ada penggabungan string SQL; query memakai parameter/ORM."}
		} else {
			sq.Evidence = []string{"Tidak ada pola penggabungan string SQL."}
		}
	}
	out = append(out, sq)

	// Command injection.
	ci := Check{ID: "CMD-INJECTION", Category: CatInjection, Title: "Eksekusi perintah shell", Source: "static"}
	if len(f.cmdInject) > 0 {
		ci.Status, ci.Severity = StatusWarn, SevWarning
		ci.Evidence = []string{"Ada pemanggilan shell; pastikan argumennya tidak berasal dari input pengguna."}
		ci.Locations = f.cmdInject
		ci.Suggestion = "Panggil program langsung dengan argumen terpisah (tanpa sh -c / shell=True) dan validasi daftar nilai yang diizinkan."
	} else {
		ci.Status, ci.Severity = StatusPass, SevInfo
		ci.Evidence = []string{"Tidak ada eksekusi shell dari kode server."}
	}
	out = append(out, ci)

	// Validasi input & batas body.
	iv := Check{ID: "INPUT-VALIDATION", Category: CatInput, Title: "Validasi input & batas ukuran body", Source: "static"}
	switch {
	case len(f.validator) > 0 && len(f.bodyLimit) > 0:
		iv.Status, iv.Severity = StatusDeclared, SevInfo
		iv.Evidence = []string{"Pustaka validasi dan batas body ditemukan."}
	case len(f.validator) > 0:
		iv.Status, iv.Severity = StatusWarn, SevInfo
		iv.Evidence = []string{"Validasi ada, tetapi batas ukuran body tidak diatur eksplisit."}
	case f.mutations > 0:
		iv.Status, iv.Severity = StatusWarn, SevWarning
		iv.Evidence = []string{"Ada rute mutasi tanpa pustaka validasi yang terlihat."}
	default:
		iv.Status, iv.Severity = StatusSkipped, SevInfo
		iv.Evidence = []string{"Tidak ada rute mutasi."}
	}
	iv.Locations = append(f.validator, f.bodyLimit...)
	if len(iv.Locations) > maxLocations {
		iv.Locations = iv.Locations[:maxLocations]
	}
	iv.Suggestion = "Validasi setiap body/query (validator, zod, pydantic, FormRequest) dan batasi ukuran body (BodyLimit / express.json({limit}))."
	out = append(out, iv)

	// Autentikasi rute mutasi.
	au := Check{ID: "AUTH-MUTATION", Category: CatAuth, Title: "Autentikasi rute mutasi", Source: "static"}
	switch {
	case f.mutations == 0:
		au.Status, au.Severity = StatusSkipped, SevInfo
		au.Evidence = []string{"Tidak ada rute POST/PUT/PATCH/DELETE."}
	case len(f.authMw) > 0:
		au.Status, au.Severity = StatusDeclared, SevInfo
		au.Evidence = []string{"Middleware autentikasi ditemukan; pastikan menutup semua rute mutasi (global, group, atau per rute)."}
		au.Locations = f.authMw
	default:
		au.Status, au.Severity = StatusFail, SevCritical
		au.Evidence = []string{"Ada rute mutasi tetapi tidak terlihat middleware autentikasi."}
	}
	au.Suggestion = "Petakan middleware global → group → rute; pasang auth sekali di group mutasi, lalu uji tanpa token harus 401."
	out = append(out, au)

	// Bocoran galat & mode debug.
	el := Check{ID: "ERROR-LEAK", Category: CatErrorLeaks, Title: "Galat internal tidak bocor ke klien", Source: "static"}
	switch {
	case len(f.debugMode) > 0:
		el.Status, el.Severity = StatusFail, SevWarning
		el.Evidence = []string{"Mode debug aktif di konfigurasi — stack trace dan detail server bisa tampil ke pengunjung."}
		el.Locations = f.debugMode
	case len(f.errorLeak) > 0:
		el.Status, el.Severity = StatusWarn, SevInfo
		el.Evidence = []string{"Pesan galat mentah (err.Error()/stack) dikirim ke klien; bisa membuka nama tabel/query."}
		el.Locations = f.errorLeak
	default:
		el.Status, el.Severity = StatusPass, SevInfo
		el.Evidence = []string{"Tidak ada galat mentah yang dikirim ke klien."}
	}
	el.Suggestion = "Kirim pesan umum ke klien dan catat detail galat di log server; matikan debug di env server."
	out = append(out, el)

	return out
}
