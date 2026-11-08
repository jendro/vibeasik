package diagnostics

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// fileScan memuat klasifikasi berkas yang dipakai semua aturan per-baris InspectFile.
type fileScan struct {
	relPath, lang         string
	isUI, isAndroid       bool
	isServer, usesSQL     bool
	isControllerOrHandler bool
	guardedGroups         map[string]bool
	inLoop                bool
	lineNum               int
	trimmed               string
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

func isSwallowedError(lang, line string) bool {
	switch {
	case lang == "go":
		return swallowedErrorGoRegex.MatchString(line) || ignoredErrGoRegex.MatchString(line)
	case swallowedErrorLangs[lang]:
		return emptyCatchRegex.MatchString(line)
	case lang == "python":
		return emptyExceptRegex.MatchString(line)
	}
	return false
}

// publicRouteMarkers: potongan path rute publik yang dikecualikan dari ROUTER-UNGUARDED-MUTATION.
var publicRouteMarkers = []string{"login", "register", "forgot", "callback", "webhook", "public", "scan", "health"}

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

func hasDynamicConcat(line string) bool {
	return strings.Contains(line, "+") || strings.Contains(line, "$") || strings.Contains(line, "fmt.Sprintf")
}

func isStubLine(lang, line string) bool {
	return stubHandlerRegex.MatchString(line) || (lang == "python" && stubPythonRegex.MatchString(line))
}

func ruleNPlusOne(s *fileScan) (models.Finding, bool) {
	if !s.inLoop || !s.usesSQL || !dbQueryCallRegex.MatchString(s.trimmed) {
		return models.Finding{}, false
	}
	return s.newFinding("query", "CRITICAL", "QUERY-NPLUS1",
		"N+1 Query Anti-Pattern: Pemanggilan Database di Dalam Loop",
		fmt.Sprintf("Pemanggilan kueri database terdeteksi di dalam perulangan (baris %d). Ini memicu latensi kumulatif O(N) dan saturasi connection pool.", s.lineNum),
		"Gunakan eager loading (GORM Preload, SQL JOIN, atau WHERE IN batching) di luar perulangan sebelum iterasi dilakukan.",
		"Martin Fowler PoEAA (Patterns of Enterprise Application Architecture): N+1 Selects Anti-Pattern. Menyebabkan bottleneck I/O jaringan database.",
		"// Refactor ke Eager Loading (Contoh GORM):\n// SEBELUM:\n// for _, user := range users { db.Find(&orders, \"user_id = ?\", user.ID) }\n// SESUDAH:\n// db.Preload(\"Orders\").Find(&users)"), true
}

func ruleSQLInjection(s *fileScan) (models.Finding, bool) {
	if !s.usesSQL || !sqlConcatRegex.MatchString(s.trimmed) {
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
	if !s.usesSQL || !selectStarRegex.MatchString(s.trimmed) || strings.Contains(s.trimmed, "COUNT(*)") {
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
	if m := routeGroupRegex.FindStringSubmatch(s.trimmed); m != nil {
		s.guardedGroups[m[1]] = s.guardedGroups[m[2]] || hasAuthMiddleware(s.trimmed)
	} else if m := routeUseRegex.FindStringSubmatch(s.trimmed); m != nil && hasAuthMiddleware(s.trimmed) {
		s.guardedGroups[m[1]] = true
	}
	return models.Finding{}, false
}

func ruleUnguardedMutation(s *fileScan) (models.Finding, bool) {
	matches := routeMutationRegex.FindStringSubmatch(s.trimmed)
	if !s.isServer || len(matches) < 3 || routeOnGuardedGroup(s.trimmed, s.guardedGroups) {
		return models.Finding{}, false
	}
	httpMethod := strings.ToUpper(matches[1])
	routePath := matches[2]
	if isPublicRoutePath(routePath) || hasAuthMiddleware(s.trimmed) {
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
	if !s.isServer || !wildcardCorsRegex.MatchString(s.trimmed) {
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
	if !isStubLine(s.lang, s.trimmed) {
		return models.Finding{}, false
	}
	return stubHandlerFinding(s.relPath, s.lineNum, s.trimmed), true
}

func ruleVerbInPath(s *fileScan) (models.Finding, bool) {
	if !s.isServer || !routerVerbRegex.MatchString(s.trimmed) {
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
	if !isSwallowedError(s.lang, s.trimmed) {
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
	if s.lang != "go" || !noTimeoutClientRegex.MatchString(s.trimmed) {
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
	matches := hardcodedSecretRegex.FindStringSubmatch(s.trimmed)
	if len(matches) < 3 || isFakeSecret(matches[2]) {
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
	if s.isAndroid || s.lang == "dart" || !commandInjectionRegex.MatchString(s.trimmed) || !hasDynamicConcat(s.trimmed) {
		return models.Finding{}, false
	}
	return s.newFinding("security", "CRITICAL", "SEC-COMMAND-INJECTION",
		"Celah Keamanan Kritis: Potensi Command Injection / RCE",
		fmt.Sprintf("Pemanggilan eksekusi shell / proses sistem terdeteksi dengan penggabungan parameter dinamis pada baris %d. Penyerang dapat menyusupkan perintah arbitrary untuk merusak sistem host.", s.lineNum),
		"Gunakan whitelist argumen tetap, pisahkan executable dari daftar argumen (tanpa shell invocation 'sh -c'), atau gunakan library API native.",
		"CWE-78: Improper Neutralization of Special Elements used in an OS Command ('OS Command Injection') & OWASP Top 10 A03:2021.",
		"Gunakan exec.Command(executable, args...) tanpa shell wrapper; validasi nilai dinamis dengan allowlist sebelum memasukkannya sebagai argumen terpisah. Contoh: exec.Command(\"ping\", \"-c\", \"1\", validatedHost)"), true
}

func rulePathTraversal(s *fileScan) (models.Finding, bool) {
	if !s.isServer || !pathTraversalRegex.MatchString(s.trimmed) {
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
	if !webSide || !unsafeHtmlRegex.MatchString(s.trimmed) ||
		strings.Contains(s.trimmed, "DOMPurify") || strings.Contains(s.trimmed, "sanitize") {
		return models.Finding{}, false
	}
	return s.newFinding("security", "CRITICAL", "SEC-XSS-UNSAFE-HTML",
		"Celah Keamanan: Dangerous Raw HTML Rendering (XSS Hazard)",
		fmt.Sprintf("Penyisipan HTML mentah tanpa sanitizer (v-html / innerHTML) pada baris %d berisiko memicu Cross-Site Scripting (XSS) jika memuat input pengguna.", s.lineNum),
		"Gunakan text binding biasa ({{ text }}) atau lewatkan konten melalui DOMPurify.sanitize() sebelum dirender ke DOM.",
		"OWASP Top 10 A03:2021-Injection (Cross-Site Scripting XSS) & CWE-79.",
		"// Bersihkan HTML sebelum rendering:\nimport DOMPurify from 'dompurify'\nconst safeContent = computed(() => DOMPurify.sanitize(rawHtml))"), true
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
	if !isJSLang(s.lang) || !syncIoRegex.MatchString(s.trimmed) || !s.isControllerOrHandler {
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
	if s.lang != "go" || !unboundedBodyReadRegex.MatchString(s.trimmed) || !s.isControllerOrHandler {
		return models.Finding{}, false
	}
	return s.newFinding("code_smell", "WARNING", "PERF-UNBOUNDED-BODY-READ",
		"Proses Berat: Pembacaan Request Body Tanpa Batas Ukuran (OOM Risk)",
		fmt.Sprintf("Pembacaan seluruh request body menggunakan io.ReadAll pada baris %d tanpa pembatas ukuran dapat menguras memori RAM hingga server crash (Out-Of-Memory).", s.lineNum),
		"Bungkus body dengan http.MaxBytesReader(w, r.Body, maxBytes) atau gunakan middleware BodyLimit.",
		"Resource Allocation Management & Resilience Engineering (Buffer Over-allocation Mitigation).",
		"// Batasi ukuran pembacaan stream:\nlimitedReader := io.LimitReader(r.Body, 2*1024*1024) // Maksimal 2MB\ndata, err := io.ReadAll(limitedReader)"), true
}

func ruleNativeDialog(s *fileScan) (models.Finding, bool) {
	if !s.isUI || !nativeDialogRegex.MatchString(s.trimmed) {
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
	ruleWildcardLike, ruleSyncIO, ruleUnboundedBody, ruleNativeDialog,
}

// loopTracker melacak kedalaman blok loop berdasarkan kurung kurawal baris demi baris.
type loopTracker struct {
	inLoop bool
	depth  int
}

// update menerima baris terpangkas dan mengembalikan apakah baris ini berada dalam loop.
func (l *loopTracker) update(trimmed string) bool {
	if loopKeywordRegex.MatchString(trimmed) {
		l.inLoop = true
		l.depth++
	}
	if !l.inLoop {
		return false
	}
	if strings.Contains(trimmed, "{") {
		l.depth += strings.Count(trimmed, "{") - 1
	}
	if strings.Contains(trimmed, "}") {
		l.depth -= strings.Count(trimmed, "}")
		if l.depth <= 0 {
			l.inLoop = false
			l.depth = 0
		}
	}
	return l.inLoop
}

// isCommentLine: baris kosong atau komentar (// /* * dan # untuk python/php).
func isCommentLine(lang, trimmed string) bool {
	if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
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

// scanCodeLines menjalankan seluruh aturan per-baris pada isi berkas.
func scanCodeLines(s *fileScan, lines []string) []models.Finding {
	var findings []models.Finding
	var loops loopTracker
	for idx, line := range lines {
		s.lineNum = idx + 1
		s.trimmed = strings.TrimSpace(line)
		if isCommentLine(s.lang, s.trimmed) {
			if isStubLine(s.lang, s.trimmed) {
				findings = append(findings, stubHandlerFinding(s.relPath, s.lineNum, s.trimmed))
			}
			continue
		}
		s.inLoop = loops.update(s.trimmed)
		for _, rule := range codeLineRules {
			if f, ok := rule(s); ok {
				findings = append(findings, f)
			}
		}
	}
	return findings
}
