package diagnostics

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// fileScan memuat klasifikasi berkas yang dipakai semua aturan per-baris InspectFile.
type fileScan struct {
	relPath, lang         string
	isUI, isAndroid       bool
	isServer, usesSQL     bool
	isControllerOrHandler bool
	hasRateLimiter        bool
	guardedGroups         map[string]bool
	hasBodyLimit          bool
	oneOffTool            bool // skrip sekali jalan (bench, seed, migrasi): aturan performa query tidak berlaku
	inLoop                bool
	loopStart             int // baris (1-based) loop terdalam yang sedang dibuka; 0 = di luar loop
	inRawString           bool
	inMarkup              bool // baris berada di markup komponen (.vue <template>, .svelte di luar <script>/<style>)
	lineNum               int
	trimmed               string
	// raw = baris asli, code = baris yang isi string & komentarnya dikosongkan (stripCode, panjang sama),
	// lead = jumlah spasi awal yang dipangkas dari trimmed. Aturan memakai matchCode/findCode agar pola kode tidak
	// terpicu oleh teks di dalam string, komentar, atau dokumentasi aturan itu sendiri.
	raw, code string
	lead      int
	lines     []string
	codeLines []string
	authAlias map[string]bool // variabel yang diisi middleware auth (a := s.requireUser)
	localFns  map[string]bool // nama alert/confirm/prompt yang dideklarasikan/diimpor di berkas ini
}

// codeAt: awal rentang [start,end) pada trimmed berada di kode, bukan di dalam string/komentar.
func (s *fileScan) codeAt(start, end int) bool {
	for p := s.lead + start; p < s.lead+end && p < len(s.code) && p < len(s.raw); p++ {
		if s.raw[p] == ' ' || s.raw[p] == '\t' {
			continue
		}
		return s.code[p] == s.raw[p]
	}
	return false
}

// matchCode: pola cocok pada baris ini dan kecocokannya berawal di kode.
func (s *fileScan) matchCode(re *regexp.Regexp) bool {
	for _, loc := range re.FindAllStringIndex(s.trimmed, -1) {
		if s.codeAt(loc[0], loc[1]) {
			return true
		}
	}
	return false
}

// findCode: submatch pertama yang berawal di kode beserta indeksnya (nil bila tidak ada).
func (s *fileScan) findCode(re *regexp.Regexp) ([]string, []int) {
	for _, loc := range re.FindAllStringSubmatchIndex(s.trimmed, -1) {
		if !s.codeAt(loc[0], loc[1]) {
			continue
		}
		out := make([]string, len(loc)/2)
		for i := range out {
			if loc[2*i] >= 0 {
				out[i] = s.trimmed[loc[2*i]:loc[2*i+1]]
			}
		}
		return out, loc
	}
	return nil, nil
}

// nextCodeLine: baris kode berikutnya (dipangkas) setelah baris saat ini; "" bila tidak ada.
func (s *fileScan) nextCodeLine() string {
	for i := s.lineNum; i < len(s.codeLines); i++ {
		if t := strings.TrimSpace(s.codeLines[i]); t != "" {
			return t
		}
	}
	return ""
}

// lineRule memeriksa satu baris; mengembalikan temuan bila aturan terpicu.
type lineRule func(s *fileScan) (models.Finding, bool)

// newLineFinding membangun temuan pada baris saat ini dengan field lokasi terisi.
func (s *fileScan) newFinding(category, severity, ruleID, title, desc, suggestion, theory, fix string) models.Finding {
	return models.Finding{
		Category:     category,
		Severity:     severity,
		RuleID:       ruleID,
		Title:        title,
		Description:  desc,
		Suggestion:   suggestion,
		TargetFile:   s.relPath,
		Line:         s.lineNum,
		Snippet:      s.trimmed,
		TheoryBasis:  theory,
		SuggestedFix: fix,
	}
}

func isJSLang(lang string) bool { return lang == "javascript" || lang == "typescript" }

// swallowedErrorLangs: bahasa yang mengenali pola catch kosong.
var swallowedErrorLangs = map[string]bool{
	"javascript": true, "typescript": true, "vue": true, "svelte": true,
	"kotlin": true, "java": true, "dart": true, "php": true,
}

// isSwallowedError: error dibuang tanpa penanganan. Go: `_ = err`, atau `_, err := f()` yang err-nya tidak pernah
// dibaca lagi (pola `_, err :=` sendiri justru MENANGKAP error; yang dibuang nilai pertamanya).
func isSwallowedError(s *fileScan) bool {
	switch {
	case s.lang == "go":
		if s.matchCode(ignoredErrGoRegex) {
			return true
		}
		return s.matchCode(capturedErrGoRegex) && !errReadAfter(s.codeLines, s.lineNum)
	case swallowedErrorLangs[s.lang]:
		return s.matchCode(emptyCatchRegex)
	case s.lang == "python":
		return s.matchCode(emptyExceptRegex)
	}
	return false
}

// errLookahead: jarak baris untuk mencari pembacaan err setelah `_, err := f()` (pernyataan multi-baris + if).
const errLookahead = 40

var (
	errWordRegex     = regexp.MustCompile(`\berr\b`)
	errReassignRegex = regexp.MustCompile(`^[\w\s,.*\[\]]*\berr\s*:?=[^=]`)
	goFuncStartRegex = regexp.MustCompile(`^func\b`)
)

// errReadAfter: err yang ditangkap pada baris from (1-based) dibaca lagi (if err, return err, log(err), …) sebelum
// fungsi berakhir atau err ditimpa penugasan baru.
func errReadAfter(codeLines []string, from int) bool {
	for i := from; i < len(codeLines) && i < from+errLookahead; i++ {
		t := strings.TrimSpace(codeLines[i])
		if t == "" {
			continue
		}
		if goFuncStartRegex.MatchString(t) {
			return false
		}
		if !errWordRegex.MatchString(t) {
			continue
		}
		// `x, err = g()` tanpa membaca err lama = nilai lama terbuang; `if err := g(); err != nil` adalah pembacaan baru.
		return !errReassignRegex.MatchString(t)
	}
	return false
}

// publicRouteMarkers: potongan path rute publik yang dikecualikan dari ROUTER-UNGUARDED-MUTATION.
var publicRouteMarkers = []string{"login", "register", "forgot", "callback", "webhook", "public", "scan", "health",
	"signup", "sign-up", "signin", "sign-in", "logout", "refresh", "verify", "otp", "reset-password", "/auth/"}

var (
	// authWordRegex: nama middleware/pembungkus yang memeriksa identitas atau hak akses (tidak peka huruf besar).
	authWordRegex = regexp.MustCompile(`(?i)(auth|jwt|protect|guard|require|role|perm|session|token|secure|localaccess|execaccess|verif|signed|signature|apikey|api_key|admin|bearer|rbac|acl)`)
	// httpClientRecvRegex: penerima panggilan .post/.put/.delete yang merupakan klien HTTP, bukan router.
	httpClientRecvRegex = regexp.MustCompile(`(?i)(^|[.$_])(axios|https?|\w*client|request|superagent|got|ky|fetch|dio|\$http|\$api|\$axios)$`)
	routeRecvChainRegex = regexp.MustCompile(`([A-Za-z_$][\w$]*(?:\.[A-Za-z_$][\w$]*)*)$`)
	// clientCallPrefixRegex: hasil panggilan ditunggu/ditampung/dikembalikan → panggilan klien, bukan pendaftaran rute.
	clientCallPrefixRegex = regexp.MustCompile(`(\bawait\s|\breturn\s|[^=!<>]=\s*[^=]*$)`)
	authAliasRegex        = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*:?=\s*([^=].*)$`)
)

// isRouteRegistration: baris ini mendaftarkan rute (bukan panggilan klien HTTP / method lain bernama Post/Delete).
// prefix = teks sebelum ".Post(".
func isRouteRegistration(prefix, routePath string) bool {
	if routePath != "" && !strings.HasPrefix(routePath, "/") && !strings.HasPrefix(routePath, ":") {
		return false // "user:profile", "https://…": bukan pola path rute
	}
	if clientCallPrefixRegex.MatchString(prefix) {
		return false
	}
	recv := routeRecvChainRegex.FindString(strings.TrimSpace(prefix))
	for _, seg := range strings.Split(recv, ".") {
		if httpClientRecvRegex.MatchString(seg) {
			return false
		}
	}
	return true
}

// splitCallArgs memecah argumen tingkat atas dari teks setelah "(" pembuka; complete=false bila pemanggilan tidak
// ditutup di baris ini (argumen berlanjut ke baris berikut).
func splitCallArgs(rest string) (args []string, complete bool) {
	depth, start := 0, 0
	var quote byte
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		if quote != 0 {
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'', '`':
			quote = c
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth == 0 {
				return append(args, strings.TrimSpace(rest[start:i])), true
			}
			depth--
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(rest[start:i]))
				start = i + 1
			}
		}
	}
	return append(args, strings.TrimSpace(rest[start:])), false
}

// routeArgsGuarded: salah satu middleware (argumen sebelum handler terakhir) atau pembungkus handler memeriksa
// auth. Nama handler sendiri tidak dihitung (createToken bukan middleware).
func routeArgsGuarded(args []string, alias map[string]bool) bool {
	if len(args) == 0 {
		return false
	}
	for _, a := range args[:len(args)-1] {
		if authWordRegex.MatchString(a) || alias[a] {
			return true
		}
	}
	last := args[len(args)-1]
	if i := strings.Index(last, "("); i > 0 && !strings.HasPrefix(last, "func") {
		return authWordRegex.MatchString(last[:i]) // requireAuth(h.Create)
	}
	return false
}

func isPublicRoutePath(routePath string) bool {
	for _, m := range publicRouteMarkers {
		if strings.Contains(routePath, m) {
			return true
		}
	}
	return false
}

var fakeSecretMarkers = []string{"xxx", "CHANGE_ME", "dummy", "test"}

func isFakeSecret(val string) bool {
	for _, m := range fakeSecretMarkers {
		if strings.Contains(val, m) {
			return true
		}
	}
	return false
}

var (
	goShellExecRegex  = regexp.MustCompile(`exec\.Command(?:Context)?\s*\(\s*(?:[\w.]+\s*,\s*)?"(?:sh|bash|zsh|cmd|cmd\.exe|powershell)"\s*,\s*"(?:-c|/C|/c|-Command)"\s*,\s*(.+)$`)
	phpVarRegex       = regexp.MustCompile(`\$[A-Za-z_{]`)
	evalMethodRegex   = regexp.MustCompile(`\.\s*eval\s*\($`)
	goStringLitRegex  = regexp.MustCompile("^(\"[^\"]*\"|`[^`]*`)$")
	fStringOpenRegex  = regexp.MustCompile(`\bf["']`)
	templateExprRegex = regexp.MustCompile(`\$\{`)
)

// dynamicCommandArgs: argumen pemanggilan perintah (teks setelah "(" pembuka) memuat nilai yang dirakit saat
// jalan. Hanya argumen yang diperiksa, bukan seluruh baris (setiap baris PHP memuat "$").
func dynamicCommandArgs(lang, rest string) bool {
	args, _ := splitCallArgs(rest)
	joined := strings.Join(args, ",")
	switch lang {
	case "php":
		return phpVarRegex.MatchString(joined)
	case "python":
		return strings.Contains(joined, "+") || strings.Contains(joined, "%") || strings.Contains(joined, ".format(") || fStringOpenRegex.MatchString(joined)
	}
	return strings.Contains(joined, "+") || templateExprRegex.MatchString(joined) || strings.Contains(joined, "fmt.Sprintf")
}

func isStubLine(lang, line string) bool {
	return stubHandlerRegex.MatchString(line) || (lang == "python" && stubPythonRegex.MatchString(line))
}

func ruleNPlusOne(s *fileScan) (models.Finding, bool) {
	if !s.inLoop || !s.usesSQL || s.oneOffTool || !s.matchCode(dbQueryCallRegex) {
		return models.Finding{}, false
	}
	// Query yang langsung diikuti return/break berjalan paling banyak sekali (pencarian sampai ketemu), bukan N kali.
	if next := s.nextCodeLine(); strings.HasPrefix(next, "return") || strings.HasPrefix(next, "break") {
		return models.Finding{}, false
	}
	// Loop yang dipotong konstanta (`if i >= 3 { break }`) menjalankan query paling banyak K kali; query ke tabel
	// yang berbeda tiap putaran (`.Table(r.Name)`) tidak bisa digabung dengan WHERE IN.
	if s.loopCappedByBreak() || s.matchCode(dynamicTableRegex) {
		return models.Finding{}, false
	}
	return s.newFinding("query", "CRITICAL", "QUERY-NPLUS1",
		"N+1 Query Anti-Pattern: Pemanggilan Database di Dalam Loop",
		fmt.Sprintf("Pemanggilan kueri database terdeteksi di dalam perulangan (baris %d). Ini memicu latensi kumulatif O(N) dan saturasi connection pool.", s.lineNum),
		"Gunakan eager loading (GORM Preload, SQL JOIN, atau WHERE IN batching) di luar perulangan sebelum iterasi dilakukan.",
		"Martin Fowler PoEAA (Patterns of Enterprise Application Architecture): N+1 Selects Anti-Pattern. Menyebabkan bottleneck I/O jaringan database.",
		"// Refactor ke Eager Loading (Contoh GORM):\n// SEBELUM:\n// for _, user := range users { db.Find(&orders, \"user_id = ?\", user.ID) }\n// SESUDAH:\n// db.Preload(\"Orders\").Find(&users)"), true
}

var (
	// capBreakRegex: `if i >= 3 {` / `if (i > 10) break;` / `if i == 5: break` — batas konstanta di dalam loop.
	capBreakRegex = regexp.MustCompile(`^(?:\}\s*)?(?:else\s+)?if\s*\(?\s*[A-Za-z_$][\w$]*\s*(?:>=|>|==)\s*\d+\s*\)?\s*[:{]?\s*(break\b)?`)
	// dynamicTableRegex: nama tabel dari variabel/field (Go `.Table(r.Name)`, Laravel `DB::table($t)`).
	dynamicTableRegex = regexp.MustCompile(`(?:\.Table|::table)\(\s*[$A-Za-z_][\w.$>\[\]-]*\s*[,)]`)
)

// loopCappedByBreak: loop terdalam dipotong konstanta sebelum baris ini (`if i >= K { break }`), jadi isinya
// berjalan paling banyak K kali.
func (s *fileScan) loopCappedByBreak() bool {
	if s.loopStart <= 0 {
		return false
	}
	for i := s.loopStart; i < s.lineNum-1 && i < len(s.codeLines); i++ {
		m := capBreakRegex.FindStringSubmatch(strings.TrimSpace(s.codeLines[i]))
		if m == nil {
			continue
		}
		if m[1] != "" {
			return true
		}
		for j := i + 1; j < len(s.codeLines); j++ {
			if t := strings.TrimSpace(s.codeLines[j]); t != "" {
				if strings.HasPrefix(t, "break") {
					return true
				}
				break
			}
		}
	}
	return false
}

func ruleSQLInjection(s *fileScan) (models.Finding, bool) {
	if !s.usesSQL || !s.matchCode(sqlConcatRegex) || isSafeSprintfSQL(s.trimmed) {
		return models.Finding{}, false
	}
	return s.newFinding("query", "CRITICAL", "QUERY-SQLI",
		"Potensi Kerentanan SQL Injection (String Concatenation Query)",
		fmt.Sprintf("Query SQL dibentuk menggunakan penggabungan string langsung atau formatting dinamis pada baris %d.", s.lineNum),
		"Gunakan parameterized query (placeholder '?' atau prepared statement) agar input pengguna diperlakukan sebagai literal nilai, bukan kode SQL.",
		"OWASP Top 10 A03:2021-Injection & CWE-89 (Improper Neutralization of Special Elements in SQL Command).",
		"// Gunakan parameterized query dengan placeholder dan kirim nilai sebagai argumen terpisah.\n// Contoh aman: db.Raw(\"SELECT * FROM users WHERE id = ?\", id)"), true
}

func ruleSelectStar(s *fileScan) (models.Finding, bool) {
	if !s.usesSQL || !s.matchCode(selectStarRegex) || strings.Contains(s.trimmed, "COUNT(*)") {
		return models.Finding{}, false
	}
	return s.newFinding("query", "WARNING", "QUERY-SELECT-STAR",
		"SELECT * Anti-Pattern: Pengambilan Kolom Tanpa Batas (Projection Smell)",
		fmt.Sprintf("Penggunaan 'SELECT *' terdeteksi pada baris %d. Mengambil semua kolom meningkatkan beban memori, transfer data jaringan, dan mematikan optimasi covering index.", s.lineNum),
		"Sebutkan hanya kolom yang dibutuhkan secara eksplisit (misal: 'SELECT id, name, email FROM ...') dan tambahkan klausa batas (LIMIT/Pagination).",
		"C.J. Date Relational Model Principles & Baron Schwartz High Performance MySQL Indexing Theory.",
		"// Nyatakan kolom secara spesifik:\n// SEBELUM: db.Raw(\"SELECT * FROM orders WHERE status = ?\", status)\n// SESUDAH: db.Select(\"id, order_number, total, status\").Where(\"status = ?\", status).Limit(50)"), true
}

// ruleTrackGuardedGroups memperbarui peta group terjaga; tidak menghasilkan temuan.
func ruleTrackGuardedGroups(s *fileScan) (models.Finding, bool) {
	if !s.codeAt(0, len(s.trimmed)) {
		return models.Finding{}, false
	}
	if m := routeGroupRegex.FindStringSubmatch(s.trimmed); m != nil {
		s.guardedGroups[m[1]] = s.guardedGroups[m[2]] || hasAuthMiddleware(s.trimmed)
	} else if m := routeUseRegex.FindStringSubmatch(s.trimmed); m != nil && hasAuthMiddleware(s.trimmed) {
		s.guardedGroups[m[1]] = true
	} else if m := authAliasRegex.FindStringSubmatch(s.trimmed); m != nil && !strings.Contains(m[2], "(") && authWordRegex.MatchString(m[2]) {
		s.authAlias[m[1]] = true // a := s.requireUser → `a` dipakai sebagai middleware di rute berikutnya
	}
	return models.Finding{}, false
}

// ruleUnguardedMutation menandai PENDAFTARAN rute pengubah data yang rantai middleware-nya, sejauh terlihat di
// berkas ini, tidak memuat pemeriksaan auth. Yang tidak bisa dipastikan dari baris itu (argumen berlanjut ke baris
// berikut) tidak dilaporkan; untuk Go hasilnya masih disaring peta rute (refineRouterFindings).
func ruleUnguardedMutation(s *fileScan) (models.Finding, bool) {
	if !s.isServer {
		return models.Finding{}, false
	}
	matches, loc := s.findCode(routeMutationRegex)
	if len(matches) < 3 || routeOnGuardedGroup(s.trimmed, s.guardedGroups) {
		return models.Finding{}, false
	}
	httpMethod := strings.ToUpper(matches[1])
	routePath := matches[2]
	if !isRouteRegistration(s.trimmed[:loc[0]], routePath) || isPublicRoutePath(strings.ToLower(routePath)) {
		return models.Finding{}, false
	}
	args, complete := splitCallArgs(s.trimmed[loc[6]:])
	if !complete || routeArgsGuarded(args, s.authAlias) {
		return models.Finding{}, false
	}
	return s.newFinding("router", "WARNING", "ROUTER-UNGUARDED-MUTATION",
		fmt.Sprintf("Rute Mutasi Data (%s %s) Tanpa Proteksi Middleware", httpMethod, routePath),
		fmt.Sprintf("Rute pengubah status '%s %s' pada baris %d tidak memiliki middleware otorisasi/autentikasi eksplisit.", httpMethod, routePath, s.lineNum),
		"Pasang middleware autentikasi (JWT Bearer Token / Session Guard / Role Checker) pada rantai handler rute.",
		"OWASP API Security Top 10 API1:2023 (Broken Object Level Authorization - BOLA) & API2:2023 (Broken Authentication).",
		fmt.Sprintf("// Pasang middleware autentikasi sebelum handler:\n// SESUDAH: api.%s(\"%s\", authMiddleware, handler)", strings.Title(strings.ToLower(httpMethod)), routePath)), true
}

func ruleWildcardCORS(s *fileScan) (models.Finding, bool) {
	if !s.isServer || !s.matchCode(wildcardCorsRegex) {
		return models.Finding{}, false
	}

	return s.newFinding("router", "WARNING", "ROUTER-WILDCARD-CORS",
		"Kebijakan CORS Mengizinkan Semua Origin",
		fmt.Sprintf("Konfigurasi CORS pada baris %d mengizinkan semua origin (*). Pada API yang menggunakan cookie, token, atau kredensial pengguna, konfigurasi ini dapat memperluas permukaan CSRF dan kebocoran data. Origin production harus dibatasi ke allowlist frontend resmi.", s.lineNum),
		"Simpan allowlist origin pada environment variable, validasi setiap origin, dan gunakan hanya origin yang terdaftar. Contoh: FRONTEND_ORIGINS=http://localhost:3000,https://app.example.com; jangan gunakan wildcard untuk API berkredensial.",
		"W3C Cross-Origin Resource Sharing Specification, RFC 6454 (The Web Origin Concept), dan OWASP API Security API8:2023 Security Misconfiguration.",
		"// Contoh konfigurasi berbasis allowlist environment:\n// FRONTEND_ORIGINS=http://localhost:3000,https://app.example.com\n// Set AllowOrigins dari origin yang tervalidasi; jangan gunakan AllowOrigins: \"*\" pada production."), true
}

func ruleStubHandler(s *fileScan) (models.Finding, bool) {
	if !s.isServer {
		return models.Finding{}, false
	}
	if s.lang == "python" && s.trimmed == "pass" {
		// `pass` hanya stub bila menjadi satu-satunya isi sebuah def di berkas handler; `class X(Exception): pass`
		// dan `except …: pass` adalah Python biasa.
		if !s.isControllerOrHandler || !isPythonDefBody(s.codeLines, s.lineNum) {
			return models.Finding{}, false
		}
		return stubHandlerFinding(s.relPath, s.lineNum, s.trimmed), true
	}
	if !s.matchCode(stubHandlerRegex) && !(s.lang == "python" && s.matchCode(stubPythonRegex)) {
		return models.Finding{}, false
	}
	return stubHandlerFinding(s.relPath, s.lineNum, s.trimmed), true
}

// isPythonDefBody: baris lineNum (1-based) adalah satu-satunya pernyataan di badan sebuah `def`.
func isPythonDefBody(codeLines []string, lineNum int) bool {
	indent := func(l string) int { return len(l) - len(strings.TrimLeft(l, " \t")) }
	cur := indent(codeLines[lineNum-1])
	prev := -1
	for i := lineNum - 2; i >= 0; i-- {
		if strings.TrimSpace(codeLines[i]) != "" {
			prev = i
			break
		}
	}
	if prev < 0 || indent(codeLines[prev]) >= cur {
		return false
	}
	head := strings.TrimSpace(codeLines[prev])
	if !strings.HasPrefix(head, "def ") && !strings.HasPrefix(head, "async def ") {
		return false
	}
	for i := lineNum; i < len(codeLines); i++ {
		if strings.TrimSpace(codeLines[i]) != "" {
			return indent(codeLines[i]) < cur
		}
	}
	return true
}

var (
	routePathLitRegex = regexp.MustCompile(`(?i)\.(Get|Post|Put|Delete|Patch)\s*\(\s*["'](/[^"']*)["']`)
	camelSplitRegex   = regexp.MustCompile(`[A-Z]?[a-z0-9]+|[A-Z]+`)
)

var routeVerbs = map[string]bool{"get": true, "fetch": true, "delete": true, "remove": true, "create": true, "add": true, "update": true, "edit": true}

// verbSegment mengembalikan segmen path yang KATA PERTAMANYA kata kerja CRUD dan diikuti kata lain
// (/getUser, /create-order). Segmen yang hanya memuat kata kerja sebagai potongan kata lain (/edition, /credits)
// atau parameter (:id, {id}) tidak dihitung.
func verbSegment(routePath string) string {
	for _, seg := range strings.Split(routePath, "/") {
		if seg == "" || strings.HasPrefix(seg, ":") || strings.HasPrefix(seg, "{") || strings.HasPrefix(seg, "*") {
			continue
		}
		words := camelSplitRegex.FindAllString(seg, -1)
		if len(words) >= 2 && routeVerbs[strings.ToLower(words[0])] {
			return seg
		}
	}
	return ""
}

func ruleVerbInPath(s *fileScan) (models.Finding, bool) {
	if !s.isServer {
		return models.Finding{}, false
	}
	m, loc := s.findCode(routePathLitRegex)
	if m == nil || verbSegment(m[2]) == "" || !isRouteRegistration(s.trimmed[:loc[0]], m[2]) {
		return models.Finding{}, false
	}
	return s.newFinding("router", "INFO", "ROUTER-VERB-IN-PATH",
		"Pelanggaran RESTful URI: Menggunakan Kata Kerja di Path Rute",
		fmt.Sprintf("URL rute pada baris %d mengandung kata kerja (get/create/delete). Dalam paradigma REST, URI adalah kata benda (noun/resource).", s.lineNum),
		"Ganti kata kerja pada URI dengan kata benda plural dan gunakan HTTP method yang sesuai (GET, POST, PUT, DELETE).",
		"Roy Fielding Architectural Styles and the Design of Network-based Software Architectures (REST Chapter 5).",
		"// Ubah pola rute ke standar REST:\n// SEBELUM: app.Get(\"/getUser/:id\", ...)\n// SESUDAH: app.Get(\"/users/:id\", ...)"), true
}

func ruleSwallowedError(s *fileScan) (models.Finding, bool) {
	if !isSwallowedError(s) {
		return models.Finding{}, false
	}
	return s.newFinding("code_smell", "WARNING", "SMELL-SWALLOWED-ERROR",
		"Swallowed Error Smell: Kesalahan / Error Diabaikan Tanpa Penanganan",
		fmt.Sprintf("Error return diabaikan atau ditangkap tanpa log/handling pada baris %d. Ini memicu silent failure dan inkonsistensi state data.", s.lineNum),
		"Periksa error return secara eksplisit. Log error dengan context yang relevan atau teruskan error ke error-handling layer.",
		"Jim Shore's Fail-Fast Principle & Google SRE Observability Engineering.",
		"// Tangani error secara eksplisit:\n// SESUDAH:\nif err != nil {\n    log.Printf(\"[ERROR] failed operation: %v\", err)\n    return err\n}"), true
}

func ruleMissingTimeout(s *fileScan) (models.Finding, bool) {
	if s.lang != "go" || !s.matchCode(noTimeoutClientRegex) {
		return models.Finding{}, false
	}
	return s.newFinding("code_smell", "WARNING", "SMELL-MISSING-TIMEOUT",
		"HTTP Client Tanpa Batas Waktu (Infinite Timeout Hazard)",
		fmt.Sprintf("Inisialisasi HTTP client default tanpa field Timeout terdeteksi pada baris %d. Permintaan dapat menggantung selamanya jika server tujuan macet.", s.lineNum),
		"Atur timeout eksplisit pada struct http.Client (contoh: Timeout: 10 * time.Second) atau sertakan context.WithTimeout.",
		"Michael Nygard 'Release It!': Design and Deploy Production-Ready Software (Circuit Breakers & Timeouts).",
		"// Gunakan client dengan timeout terkonfigurasi:\nclient := &http.Client{\n    Timeout: 10 * time.Second,\n}"), true
}

func ruleHardcodedSecret(s *fileScan) (models.Finding, bool) {
	loc := hardcodedSecretRegex.FindStringSubmatchIndex(s.trimmed)
	if loc == nil {
		return models.Finding{}, false
	}
	// Nama kunci harus berada di kode (atau kunci berkutip `"api_key": "…"`), bukan di tengah teks/atribut lain
	// (`:active-api-key="activeApiKey"` adalah prop Vue).
	keyInCode := s.codeAt(loc[0], loc[1]) || (loc[0] > 0 && strings.ContainsRune(`"'`, rune(s.trimmed[loc[0]-1])) && s.codeAt(loc[0]-1, loc[0]))
	if !keyInCode || (loc[0] > 0 && (s.trimmed[loc[0]-1] == '-' || isWordByte(s.trimmed[loc[0]-1]))) {
		return models.Finding{}, false
	}
	matches := []string{"", s.trimmed[loc[2]:loc[3]], s.trimmed[loc[4]:loc[5]]}
	if isFakeSecret(matches[2]) || isIdentifierLike(matches[2]) {
		return models.Finding{}, false
	}
	return s.newFinding("security", "CRITICAL", "SMELL-HARDCODED-SECRET",
		"Kredensial / Secret Tertulis Langsung (Hardcoded Secret)",
		fmt.Sprintf("Kunci rahasia atau kredensial '%s' terdeteksi langsung di dalam kode sumber pada baris %d.", matches[1], s.lineNum),
		"Pindahkan nilai secret ke file .env dan panggil via environment variables (os.Getenv atau process.env).",
		"The Twelve-Factor App (Factor III: Config - Store config in the environment) & OWASP Secrets Management Cheat Sheet.",
		fmt.Sprintf("// Panggil secret dari environment:\nsecret := os.Getenv(\"%s\")", strings.ToUpper(matches[1]))), true
}

func ruleCommandInjection(s *fileScan) (models.Finding, bool) {
	if s.isAndroid || s.lang == "dart" || !commandInjectionCandidate(s) {
		return models.Finding{}, false
	}
	return s.newFinding("security", "CRITICAL", "SEC-COMMAND-INJECTION",
		"Celah Keamanan Kritis: Potensi Command Injection / RCE",
		fmt.Sprintf("Pemanggilan eksekusi shell / proses sistem terdeteksi dengan penggabungan parameter dinamis pada baris %d. Penyerang dapat menyusupkan perintah arbitrary untuk merusak sistem host.", s.lineNum),
		"Gunakan whitelist argumen tetap, pisahkan executable dari daftar argumen (tanpa shell invocation 'sh -c'), atau gunakan library API native.",
		"CWE-78: Improper Neutralization of Special Elements used in an OS Command ('OS Command Injection') & OWASP Top 10 A03:2021.",
		"Gunakan exec.Command(executable, args...) tanpa shell wrapper; validasi nilai dinamis dengan allowlist sebelum memasukkannya sebagai argumen terpisah. Contoh: exec.Command(\"ping\", \"-c\", \"1\", validatedHost)"), true
}

// commandInjectionCandidate: pemanggilan perintah sistem yang argumennya dirakit dinamis. Go exec.Command tanpa
// shell meneruskan argumen apa adanya (tidak ditafsir shell), jadi hanya `sh -c <dinamis>` yang berisiko; eval
// hanya berlaku untuk bahasa yang punya eval global (bukan method .eval() milik objek).
func commandInjectionCandidate(s *fileScan) bool {
	if s.lang == "go" {
		m, _ := s.findCode(goShellExecRegex)
		if m == nil {
			return false
		}
		args, _ := splitCallArgs(m[1])
		return len(args) > 0 && !goStringLitRegex.MatchString(args[0])
	}
	for _, loc := range commandInjectionRegex.FindAllStringIndex(s.trimmed, -1) {
		if !s.codeAt(loc[0], loc[1]) || strings.HasPrefix(s.trimmed[loc[0]:], "exec.Command") {
			continue
		}
		if evalMethodRegex.MatchString(s.trimmed[:loc[1]]) {
			continue // method .eval() milik objek, bukan eval global
		}
		open := strings.Index(s.trimmed[loc[0]:], "(")
		if open >= 0 && dynamicCommandArgs(s.lang, s.trimmed[loc[0]+open+1:]) {
			return true
		}
	}
	return false
}

func isWordByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isIdentifierLike: nilai hanya huruf (camelCase/kata) — nama variabel atau placeholder, bukan rahasia acak.
func isIdentifierLike(v string) bool {
	for _, r := range v {
		if !unicode.IsLetter(r) && r != '_' {
			return false
		}
	}
	return true
}

var (
	sprintfFormatRegex = regexp.MustCompile("fmt\\.Sprintf\\s*\\(\\s*(\"[^\"]*\"|`[^`]*`)")
	sqlDDLRegex        = regexp.MustCompile(`(?i)\b(CREATE|ALTER|DROP)\s+(DATABASE|SCHEMA|TABLE|INDEX|USER)\b`)
	sprintfVerbRegex   = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z]`)
)

// isSafeSprintfSQL: fmt.Sprintf pada query yang tidak bisa menjadi celah injeksi nilai — DDL (identifier memang
// tidak bisa di-bind sebagai parameter) atau format yang seluruh verb-nya numerik (%d).
func isSafeSprintfSQL(line string) bool {
	m := sprintfFormatRegex.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	if sqlDDLRegex.MatchString(m[1]) {
		return true
	}
	verbs := sprintfVerbRegex.FindAllString(m[1], -1)
	if len(verbs) == 0 {
		return false
	}
	for _, v := range verbs {
		if c := v[len(v)-1]; c != 'd' && c != 'f' {
			return false
		}
	}
	return true
}

func rulePathTraversal(s *fileScan) (models.Finding, bool) {
	if !s.isServer || !s.matchCode(pathTraversalRegex) {
		return models.Finding{}, false
	}
	return s.newFinding("security", "CRITICAL", "SEC-PATH-TRAVERSAL",
		"Celah Keamanan: Potensi Path Traversal / Arbitrary File Access",
		fmt.Sprintf("Operasi I/O file membaca parameter request secara langsung pada baris %d tanpa verifikasi batasan folder dasar. Karakter '../' dapat disalahgunakan untuk membaca file sistem rahasia.", s.lineNum),
		"Terapkan filepath.Clean(), bersihkan karakter '../', dan pastikan jalur berada di dalam direktori penyimpanan yang aman (safe root directory check).",
		"CWE-22: Improper Limitation of a Pathname to a Restricted Directory ('Path Traversal').",
		"// Bersihkan path dan validasi batas folder:\ncleanPath := filepath.Clean(filepath.Join(baseDir, requestedPath))\nif !strings.HasPrefix(cleanPath, baseDir) { return errors.New(\"akses ilegal\") }"), true
}

func ruleUnsafeHTML(s *fileScan) (models.Finding, bool) {
	webSide := s.isUI || (isJSLang(s.lang) && !s.isServer)
	if !webSide || !s.matchCode(unsafeHtmlRegex) || sanitizerRegex.MatchString(s.trimmed) {
		return models.Finding{}, false
	}
	// Nilai yang dirender bisa sudah disanitasi di tempat ia dibuat (computed/fungsi di berkas yang sama).
	if m := unsafeHtmlValueRegex.FindStringSubmatch(s.trimmed); m != nil && identSanitizedInFile(s.lines, m[1]) {
		return models.Finding{}, false
	}
	return s.newFinding("security", "CRITICAL", "SEC-XSS-UNSAFE-HTML",
		"Celah Keamanan: Dangerous Raw HTML Rendering (XSS Hazard)",
		fmt.Sprintf("Penyisipan HTML mentah tanpa sanitizer (v-html / innerHTML) pada baris %d berisiko memicu Cross-Site Scripting (XSS) jika memuat input pengguna.", s.lineNum),
		"Gunakan text binding biasa ({{ text }}) atau lewatkan konten melalui DOMPurify.sanitize() sebelum dirender ke DOM.",
		"OWASP Top 10 A03:2021-Injection (Cross-Site Scripting XSS) & CWE-79.",
		"// Bersihkan HTML sebelum rendering:\nimport DOMPurify from 'dompurify'\nconst safeContent = computed(() => DOMPurify.sanitize(rawHtml))"), true
}

var (
	sanitizerRegex       = regexp.MustCompile(`(?i)(DOMPurify|sanitiz|escapeHtml|escape_html|xss\(|purify)`)
	unsafeHtmlValueRegex = regexp.MustCompile(`(?:v-html|innerHTML|__html)\s*[=:]\s*["'{]*\s*([A-Za-z_$][\w$]*)`)
)

// identSanitizedInFile: identifier yang dirender sebagai HTML didefinisikan di berkas ini lewat sanitizer
// (mis. `const safeHtml = computed(() => DOMPurify.sanitize(md.render(x)))`), diperiksa sampai 8 baris sesudah
// deklarasinya.
func identSanitizedInFile(lines []string, ident string) bool {
	decl := regexp.MustCompile(`(?:\b(?:const|let|var|function)\s+` + regexp.QuoteMeta(ident) + `\b|\b` + regexp.QuoteMeta(ident) + `\s*(?:=[^=]|\())`)
	for i, l := range lines {
		if !decl.MatchString(l) || strings.Contains(l, "v-html") {
			continue
		}
		for j := i; j < len(lines) && j <= i+8; j++ {
			if sanitizerRegex.MatchString(lines[j]) {
				return true
			}
		}
	}
	return false
}

func ruleAuthRateLimit(s *fileScan) (models.Finding, bool) {
	// hasRateLimiter dihitung sekali per berkas: bila limiter ada di mana pun di berkas ini,
	// rute autentikasi dianggap sudah terlindungi dan aturan tidak dipicu (menekan false positive).
	if !s.isServer || s.hasRateLimiter {
		return models.Finding{}, false
	}
	matches, loc := s.findCode(authRouteRegex)
	if len(matches) < 3 || !isRouteRegistration(s.trimmed[:loc[0]], matches[2]) {
		return models.Finding{}, false
	}
	httpMethod := strings.ToUpper(matches[1])
	routePath := matches[2]
	return s.newFinding("security", "WARNING", "SEC-RATE-LIMIT-MISSING",
		fmt.Sprintf("Endpoint Autentikasi (%s %s) Tanpa Proteksi Rate Limiting", httpMethod, routePath),
		fmt.Sprintf("Rute autentikasi '%s %s' pada baris %d dideklarasikan tanpa middleware rate limiting yang terdeteksi di berkas ini. Tanpa pembatas laju per-IP, endpoint login menjadi sasaran brute-force password dan credential stuffing, dan dapat dibanjiri request untuk menguras kapasitas server (Denial of Service). Catatan: bila limiter dipasang pada berkas router/middleware global terpisah, telusuri rantai middleware efektif lebih dahulu sebelum menindaklanjuti.", httpMethod, routePath, s.lineNum),
		"Pasang rate limiter per-IP address: maksimal 5 request/menit untuk endpoint autentikasi/login dan 100 request/menit untuk API umum; kembalikan HTTP 429 Too Many Requests dengan header Retry-After.",
		"OWASP Top 10 A07:2021-Identification and Authentication Failures, OWASP API Security API4:2023 (Unrestricted Resource Consumption) & CWE-307 (Improper Restriction of Excessive Authentication Attempts).",
		"// Pasang rate limiter ketat pada grup autentikasi (contoh Fiber):\n// auth := app.Group(\"/auth\", limiter.New(limiter.Config{Max: 5, Expiration: time.Minute}))\n// auth.Post(\"/login\", handler)"), true
}

func ruleWildcardLike(s *fileScan) (models.Finding, bool) {
	if !s.usesSQL || !wildcardLikeRegex.MatchString(s.trimmed) {
		return models.Finding{}, false
	}
	return s.newFinding("query", "WARNING", "PERF-WILDCARD-LIKE",
		"Kueri Berat: Leading-Wildcard LIKE",
		fmt.Sprintf("Pola LIKE dengan wildcard di awal pada baris %d dapat menonaktifkan pemakaian indeks B-Tree dan menyebabkan pemindaian tabel.", s.lineNum),
		"Hapus wildcard di awal (gunakan LIKE 'prefix%%') atau beralih ke Full-Text Search (FTS) index untuk pencarian teks bebas skala besar.",
		"Relational Database B-Tree Index Mechanics & SQL Query Execution Plan Optimization.",
		"// Gunakan indexing berbasis prefix atau FTS:\n// SEBELUM: db.Where(\"name LIKE ?\", \"%\" + query + \"%\")\n// SESUDAH: db.Where(\"name LIKE ?\", query + \"%\") // atau gunakan FTS5"), true
}

func ruleSyncIO(s *fileScan) (models.Finding, bool) {
	if !isJSLang(s.lang) || !s.matchCode(syncIoRegex) || !s.isControllerOrHandler {
		return models.Finding{}, false
	}
	return s.newFinding("code_smell", "CRITICAL", "PERF-SYNC-IO-BLOCKING",
		"Proses Berat: Operasi File Sinkron Memblokir Event Loop Server",
		fmt.Sprintf("Pemanggilan fs.readFileSync / writeFileSync pada baris %d di dalam handler HTTP membekukan NodeJS event loop dan menahan semua request koneksi pengguna lain.", s.lineNum),
		"Ganti dengan operasi file asinkron (fs.promises.readFile atau fs.readFile non-blocking) dengan async/await.",
		"Node.js Event Loop Architecture & Concurrency Model (Reactive Non-blocking I/O).",
		"// Gunakan async I/O:\nconst data = await fs.promises.readFile(filePath, 'utf-8')"), true
}

func ruleUnboundedBody(s *fileScan) (models.Finding, bool) {
	// Body yang sudah dibatasi di berkas ini (MaxBytesReader/LimitReader/BodyLimit) dan body RESPONS klien HTTP
	// (resp.Body) bukan pembacaan request tanpa batas.
	if s.lang != "go" || !s.isControllerOrHandler || s.hasBodyLimit || !s.matchCode(unboundedBodyReadRegex) || clientRespBodyRegex.MatchString(s.trimmed) {
		return models.Finding{}, false
	}
	return s.newFinding("code_smell", "WARNING", "PERF-UNBOUNDED-BODY-READ",
		"Proses Berat: Pembacaan Request Body Tanpa Batas Ukuran (OOM Risk)",
		fmt.Sprintf("Pembacaan seluruh request body menggunakan io.ReadAll pada baris %d tanpa pembatas ukuran dapat menguras memori RAM hingga server crash (Out-Of-Memory).", s.lineNum),
		"Bungkus body dengan http.MaxBytesReader(w, r.Body, maxBytes) atau gunakan middleware BodyLimit.",
		"Resource Allocation Management & Resilience Engineering (Buffer Over-allocation Mitigation).",
		"// Batasi ukuran pembacaan stream:\nlimitedReader := io.LimitReader(r.Body, 2*1024*1024) // Maksimal 2MB\ndata, err := io.ReadAll(limitedReader)"), true
}

var (
	clientRespBodyRegex = regexp.MustCompile(`(?i)\b(resp|res|response|httpResp)\w*\.Body\)`)
	// eventAttrOpenRegex: teks sebelum kecocokan berada di dalam nilai atribut handler (@click="…", onClick={…}).
	eventAttrOpenRegex = regexp.MustCompile(`(?:@[\w.:-]+|v-on:[\w.:-]+|\bon[A-Z]\w*|\bon[a-z]+)\s*=\s*(?:"[^"]*|'[^']*|\{[^}]*)$`)
	localDialogRegex   = regexp.MustCompile(`(?:\bfunction\s+|\b(?:const|let|var)\s+|[{,]\s*)(alert|confirm|prompt)\b\s*(?:[=(,}:]|$)`)
)

// localDialogNames: alert/confirm/prompt yang dideklarasikan, diimpor, atau di-destructure di berkas ini
// (mis. `const { confirm } = useConfirm()`), sehingga pemanggilannya bukan dialog bawaan browser.
func localDialogNames(codeLines []string) map[string]bool {
	out := map[string]bool{}
	for _, l := range codeLines {
		for _, m := range localDialogRegex.FindAllStringSubmatch(l, -1) {
			out[m[1]] = true
		}
	}
	return out
}

func ruleNativeDialog(s *fileScan) (models.Finding, bool) {
	if !s.isUI {
		return models.Finding{}, false
	}
	if s.localFns == nil {
		s.localFns = localDialogNames(s.codeLines)
	}
	hit := false
	for _, loc := range nativeDialogRegex.FindAllStringSubmatchIndex(s.trimmed, -1) {
		name := ""
		for _, g := range []int{2, 4} {
			if g+1 < len(loc) && loc[g] >= 0 {
				name = s.trimmed[loc[g]:loc[g+1]]
			}
		}
		explicitWindow := strings.Contains(s.trimmed[loc[0]:loc[1]], "window.")
		if s.localFns[name] && !explicitWindow {
			continue
		}
		// Di kode (<script>) atau di dalam nilai atribut handler template; teks/komentar/string lain dilewati.
		inHandler := eventAttrOpenRegex.MatchString(s.trimmed[:loc[0]+1])
		if inHandler || (!s.inMarkup && s.codeAt(loc[0], loc[1])) {
			hit = true
			break
		}
	}
	if !hit {
		return models.Finding{}, false
	}
	return s.newFinding("code_smell", "WARNING", "DESIGN-NATIVE-DIALOG",
		"Smelly Design: Penggunaan Dialog Bawaan Browser (Zero Alert Policy)",
		fmt.Sprintf("Terdeteksi pemanggilan alert(), confirm(), atau prompt() pada baris %d. Dialog bawaan browser bersifat synchronous blocking dan merusak tampilan profesional aplikasi modern.", s.lineNum),
		"Ganti alert() dengan toast notification elegan (useToast()) dan ganti confirm() dengan modal dialog kustom.",
		"Vibeasik UX Guidelines & Modern Non-blocking Modal Dialog Design Systems.",
		"// Gunakan useToast() dari Nuxt UI:\nconst toast = useToast()\ntoast.add({ title: 'Berhasil', description: 'Operasi selesai', color: 'success' })"), true
}

// codeLineRules dijalankan berurutan pada tiap baris kode (bukan komentar); urutan menentukan urutan temuan.
var codeLineRules = []lineRule{
	ruleNPlusOne, ruleSQLInjection, ruleSelectStar,
	ruleTrackGuardedGroups, ruleUnguardedMutation, ruleWildcardCORS,
	ruleStubHandler, ruleVerbInPath, ruleSwallowedError, ruleMissingTimeout,
	ruleHardcodedSecret, ruleCommandInjection, rulePathTraversal, ruleUnsafeHTML,
	ruleAuthRateLimit, ruleWildcardLike, ruleSyncIO, ruleUnboundedBody, ruleNativeDialog,
}

// loopTracker melacak apakah sebuah baris berada di dalam badan loop. Bahasa berkurawal: tumpukan kedalaman
// kurawal saat loop dibuka (dihitung pada baris yang string/komentarnya sudah dikosongkan). Python: indentasi.
// Loop atas daftar literal (`for _, x := range []string{…}`, `for x in ["a", "b"]`) berukuran tetap, bukan loop
// data, sehingga tidak dihitung.
type loopTracker struct {
	lang   string
	depth  int
	starts []loopFrame
	line   int // baris (1-based) yang sedang diproses, diisi pemanggil sebelum update
}

// innermost: baris pembuka loop terdalam yang masih terbuka (0 = di luar loop).
func (l *loopTracker) innermost() int {
	if n := len(l.starts); n > 0 {
		return l.starts[n-1].start
	}
	return 0
}

type loopFrame struct {
	start  int  // baris (1-based) pembuka loop
	depth  int  // kedalaman kurawal (atau indentasi Python) saat loop dibuka
	opened bool // kurawal pembuka badan loop sudah terlihat
	idle   int  // baris kode yang lewat tanpa kurawal pembuka (loop satu pernyataan tanpa kurawal)
}

var (
	literalLoopRegex = regexp.MustCompile(`(range\s+\[[^\]]*\][^{]*\{|\bin\s+[\[(]|\bof\s+\[|range\s+\d+\b)`)
	phpEndLoopRegex  = regexp.MustCompile(`^\s*end(for|foreach|while)\b`)
)

// update menerima baris asli dan versi kodenya; mengembalikan apakah baris ini berada dalam loop data.
func (l *loopTracker) update(raw, code string) bool {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return len(l.starts) > 0
	}
	if l.lang == "python" {
		return l.updateIndent(raw, code)
	}
	if phpEndLoopRegex.MatchString(code) && len(l.starts) > 0 {
		l.starts = l.starts[:len(l.starts)-1]
		return len(l.starts) > 0
	}
	isLoop := loopKeywordRegex.MatchString(code) && !literalLoopRegex.MatchString(code)
	if isLoop {
		l.starts = append(l.starts, loopFrame{start: l.line, depth: l.depth})
	}
	in := len(l.starts) > 0
	for i := 0; i < len(code); i++ {
		switch code[i] {
		case '{':
			l.depth++
			if n := len(l.starts); n > 0 && !l.starts[n-1].opened && l.depth > l.starts[n-1].depth {
				l.starts[n-1].opened = true
			}
		case '}':
			l.depth--
			for n := len(l.starts); n > 0 && l.starts[n-1].opened && l.depth <= l.starts[n-1].depth; n = len(l.starts) {
				l.starts = l.starts[:n-1]
			}
		}
	}
	// Loop tanpa kurawal (satu pernyataan, atau sintaks `foreach (...):`) ditutup setelah dua baris kode.
	if n := len(l.starts); n > 0 && !l.starts[n-1].opened && !isLoop {
		l.starts[n-1].idle++
		if l.starts[n-1].idle >= 2 && !strings.HasSuffix(trimmed, ":") {
			l.starts = l.starts[:n-1]
		}
	}
	return in
}

func (l *loopTracker) updateIndent(raw, code string) bool {
	indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
	for n := len(l.starts); n > 0 && indent <= l.starts[n-1].depth; n = len(l.starts) {
		l.starts = l.starts[:n-1]
	}
	in := len(l.starts) > 0
	if loopKeywordRegex.MatchString(code) && !literalLoopRegex.MatchString(code) {
		l.starts = append(l.starts, loopFrame{start: l.line, depth: indent, opened: true})
	}
	return in
}

// isCommentLine: baris kosong atau komentar (// /* * <!-- dan # untuk python/php).
func isCommentLine(lang, trimmed string) bool {
	if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "<!--") {
		return true
	}
	return (lang == "python" || lang == "php") && strings.HasPrefix(trimmed, "#")
}

// godControllerFinding: berkas handler/controller > 400 baris.
func godControllerFinding(relPath string, lines []string) models.Finding {
	lineCount := len(lines)
	return models.Finding{
		Category:     "code_smell",
		Severity:     "WARNING",
		RuleID:       "SMELL-GOD-CONTROLLER",
		Title:        fmt.Sprintf("God Controller Smell: File Terlalu Besar (%d Baris)", lineCount),
		Description:  fmt.Sprintf("Berkas handler '%s' memiliki %d baris kode. Ukuran yang membengkak mengindikasikan pelanggaran Single Responsibility Principle (SRP).", filepath.Base(relPath), lineCount),
		Suggestion:   "Pecah handler menjadi beberapa modul controller yang lebih kecil berdasarkan domain entitas bisnis (misal: user_handler, order_handler).",
		TargetFile:   relPath,
		Line:         1,
		Snippet:      lines[0],
		TheoryBasis:  "Robert C. Martin (Uncle Bob) SOLID - Single Responsibility Principle (SRP) & Martin Fowler 'Large Class' Code Smell.",
		SuggestedFix: "// Pisahkan tanggung jawab handler ke controller terpisah:\n// 1. Ekstrak action handlers ke sub-handler modular\n// 2. Gunakan dependency injection atau service layer terpisah",
	}
}

// rawStringTracker melacak apakah baris berada dalam multiline raw string literal (backticks).
type rawStringTracker struct {
	inRaw bool
}

func (r *rawStringTracker) update(trimmed string) bool {
	ticks := strings.Count(trimmed, "`")
	if !r.inRaw {
		if ticks%2 != 0 {
			r.inRaw = true
		}
		return false
	}
	if ticks%2 != 0 {
		r.inRaw = false
	}
	return true
}

// markupTracker menandai baris markup pada berkas komponen (.vue/.svelte): semua baris di luar blok
// <script> dan <style>.
type markupTracker struct {
	lang  string
	block bool
}

func (m *markupTracker) update(line string) bool {
	if m.lang != "vue" && m.lang != "svelte" {
		return false
	}
	t := strings.TrimSpace(line)
	if strings.HasPrefix(t, "<script") || strings.HasPrefix(t, "<style") {
		m.block = !strings.Contains(t, "</script>") && !strings.Contains(t, "</style>")
		return false
	}
	if strings.HasPrefix(t, "</script>") || strings.HasPrefix(t, "</style>") {
		m.block = false
		return false
	}
	return !m.block
}

// blankHTMLComments mengosongkan isi komentar HTML (<!-- … -->, boleh multi-baris) pada baris kode berkas
// komponen (.vue/.svelte) agar teks komentar template tidak dicocokkan sebagai kode.
func blankHTMLComments(codeLines, rawLines []string) {
	in := false
	for li, raw := range rawLines {
		b := []byte(codeLines[li])
		for i := 0; i < len(raw) && i < len(b); i++ {
			switch {
			case !in && strings.HasPrefix(raw[i:], "<!--"):
				in = true
				b[i] = ' '
			case in && strings.HasPrefix(raw[i:], "-->"):
				in = false
				for j := i; j < i+3 && j < len(b); j++ {
					b[j] = ' '
				}
				i += 2
			case in:
				b[i] = ' '
			}
		}
		codeLines[li] = string(b)
	}
}

// scanCodeLines menjalankan seluruh aturan per-baris pada isi berkas.
func scanCodeLines(s *fileScan, lines []string) []models.Finding {
	var findings []models.Finding
	s.lines, s.codeLines = lines, stripCode(lines, s.lang)
	if s.lang == "vue" || s.lang == "svelte" {
		blankHTMLComments(s.codeLines, lines)
	}
	if s.authAlias == nil {
		s.authAlias = map[string]bool{}
	}
	loops := loopTracker{lang: s.lang}
	var rawStrings rawStringTracker
	markup := markupTracker{lang: s.lang}
	for idx, line := range lines {
		s.lineNum = idx + 1
		s.inMarkup = markup.update(line)
		s.raw, s.code = line, s.codeLines[idx]
		s.trimmed = strings.TrimSpace(line)
		s.lead = len(line) - len(strings.TrimLeftFunc(line, unicode.IsSpace))
		s.inRawString = rawStrings.update(s.trimmed)
		if isCommentLine(s.lang, s.trimmed) {
			// Penanda "TODO: implement" hanya berarti stub endpoint di berkas handler.
			if s.isControllerOrHandler && !s.inRawString && strings.TrimSpace(s.code) == "" && isStubLine(s.lang, s.trimmed) {
				findings = append(findings, stubHandlerFinding(s.relPath, s.lineNum, s.trimmed))
			}
			continue
		}
		loops.line = s.lineNum
		s.inLoop = loops.update(line, s.code)
		s.loopStart = loops.innermost()
		for _, rule := range codeLineRules {
			if f, ok := rule(s); ok {
				findings = append(findings, f)
			}
		}
	}
	return findings
}
