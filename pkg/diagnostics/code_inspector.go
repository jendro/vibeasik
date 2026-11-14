package diagnostics

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/util"
)

// Ignored directory names for code auditing
var ignoredDirNames = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".nuxt":        true,
	".output":      true,
	"dist":         true,
	"build":        true,
	"bin":          true,
	".cache":       true,
	"tmp":          true,
	"ephemeral":    true, // Flutter ios/linux/macos/windows/flutter/ephemeral: hasil generator flutter tool
	".vscode":      true,
	".idea":        true,
	"coverage":     true,
	"_nuxt":        true, // bundel hasil build Nuxt yang ikut tersimpan (situs statis)
	"testdata":     true, // data uji (konvensi Go): bukan kode aplikasi
	"__pycache__":  true,
}

// gitIgnoredPaths: path (relatif dir, tanpa garis miring akhir) yang diabaikan git di bawah dir — hasil build,
// unggahan pengguna, arsip lama. Yang tidak dilacak repo bukan kode proyek, jadi tidak diaudit. Kosong bila dir
// bukan bagian repo git atau git tidak tersedia.
func gitIgnoredPaths(dir string) map[string]bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "ls-files", "--others", "--ignored", "--exclude-standard", "--directory")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	ignored := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSuffix(strings.TrimSpace(line), "/"); line != "" {
			ignored[filepath.FromSlash(line)] = true
		}
	}
	return ignored
}

// Allowed source code extensions
var auditFileExtensions = map[string]string{
	".go":     "go",
	".ts":     "typescript",
	".tsx":    "typescript",
	".js":     "javascript",
	".jsx":    "javascript",
	".mjs":    "javascript",
	".vue":    "vue",
	".svelte": "svelte",
	".dart":   "dart",
	".php":    "php",
	".py":     "python",
	".java":   "java",
	".kt":     "kotlin",
	".xml":    "xml", // hanya layout Android (res/layout*), lihat isAndroidLayoutXML
}

// Regex patterns for Query Auditing
var (
	// Loop detection
	loopKeywordRegex = regexp.MustCompile(`(?m)^\s*(for\b|while\b|foreach\b|\.map\(|\.forEach\()`)

	// DB query detection
	// Hanya pemanggilan yang MEMBACA dari database: penyusun kondisi (db.Where(...) tanpa terminal, Model::where,
	// $this->db->where, objects.filter yang lazy) belum menyentuh database, dan penulisan per baris (Create/Update/
	// Delete) bukan pola N+1 SELECT yang diselesaikan dengan eager loading.
	dbQueryCallRegex = regexp.MustCompile(`(?i)(\b\w*db\w*\b(?:\.\w+\([^;]*\))*\.(Find|First|Last|Take|Scan|Count|Pluck|Query|QueryRow|QueryContext|QueryRowContext|Rows?)\(|repo\.(Find|Get|List|Query)|prisma\.[a-zA-Z]+\.(find|create|update|delete)|sequelize\.|Model::(find|all)\(|\$this->db->(query|get)\(|session\.(query|execute)|objects\.(get|all)\()`)

	// SQL Injection string concat
	sqlConcatRegex = regexp.MustCompile(`(?i)(db\.(Raw|Exec|Where)\s*\(\s*fmt\.Sprintf|\.query\s*\(\s*['"` + "`" + `]\s*SELECT.*WHERE.*['"` + "`" + `]\s*\+|\.query\s*\(\s*` + "`" + `.*SELECT.*WHERE.*\$\{.*` + "`" + `|\$db->query\s*\(\s*['"].*SELECT.*['"]\s*\.|\.execute\s*\(\s*f["'].*SELECT.*\{)`)

	// SELECT * pattern
	selectStarRegex = regexp.MustCompile(`(?i)(['"` + "`" + `]SELECT\s+\*\s+FROM\s+[a-zA-Z0-9_]+['"` + "`" + `]|\.Select\(\s*["']\*["']\s*\))`)

	// Heavy Wildcard LIKE pattern
	wildcardLikeRegex = regexp.MustCompile(`(?i)(LIKE|ILIKE)\s*['"` + "`" + `]%[^'"\s]+%?['"` + "`" + `]`)

	// Unclosed rows
	dbQueryGoRegex  = regexp.MustCompile(`(?m)([a-zA-Z0-9_]+)\s*,\s*err\s*:?=\s*[a-zA-Z0-9_.]*db\.Query\(`)
	deferCloseRegex = regexp.MustCompile(`defer\s+[a-zA-Z0-9_]+\.Close\(\)`)

	// Transaction without rollback defer
	dbBeginGoRegex  = regexp.MustCompile(`(?m)([a-zA-Z0-9_]+)\s*:?=\s*[a-zA-Z0-9_.]*db\.Begin\(`)
	deferTxRollback = regexp.MustCompile(`defer.*Rollback\(\)`)

	// Router smells
	routeGroupRegex    = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*:?=\s*([A-Za-z_][A-Za-z0-9_]*)\.Group\s*\(`)
	routeUseRegex      = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\.Use\s*\(`)
	routeReceiverRegex = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\.(?i:Post|Put|Delete|Patch)\s*\(`)
	routeMutationRegex = regexp.MustCompile(`(?i)\.(Post|Put|Delete|Patch)\s*\(\s*["']([^"']+)["']\s*,\s*([a-zA-Z0-9_.]+)`)
	wildcardCorsRegex  = regexp.MustCompile(`(?i)(AllowOrigins\s*:\s*["']\*["']|AllowOrigins\s*:\s*\[\]string\{["']\*["']\}|origin\s*:\s*["']\*["'])`)
	// stubHandlerRegex mendeteksi pola handler yang belum tersedia atau masih berupa placeholder.
	// Pemisahan token mencegah pemeriksa mendeteksi literal regex ini sebagai stub.
	stubHandlerRegex = regexp.MustCompile(
		`(?i)(` +
			`//\s*TODO:?\s*(?:implement|fix|handle)\b` +
			`|c\.SendString\(\s*["']TODO["']\s*\)` +
			`|\bpanic\(\s*["']not implemented["']\s*\)` +
			`|\breturn\b[^\n]{0,80}\bnot` + ` implemented\b` +
			`|\bhttp\.StatusNot` + `Implemented\b` +
			`|(?:^|[^\w.])TODO\(` + // Kotlin TODO(); bukan context.TODO() milik Go
			`|\bthrow\s+(?:new\s+)?(?:Not` + `ImplementedError|UnsupportedOperationException|Not` + `ImplementedException)\s*[\(;]` +
			`)`,
	)
	// `pass` sebagai stub ditangani ruleStubHandler (hanya bila satu-satunya isi def di berkas handler).
	stubPythonRegex = regexp.MustCompile(`(?i)(raise\s+NotImplementedError|#\s*TODO:?\s*(implement|fix|handle))`)

	// Code smells
	// capturedErrGoRegex: `_, err := f()` MENANGKAP error (nilai pertama yang dibuang); baru menjadi temuan bila err
	// tidak pernah dibaca sesudahnya (isSwallowedError).
	capturedErrGoRegex = regexp.MustCompile(`(?m)^\s*_\s*,\s*err\s*:?=`)
	ignoredErrGoRegex  = regexp.MustCompile(`(?m)^\s*_\s*=\s*err\b`)
	emptyCatchRegex    = regexp.MustCompile(`(?m)catch\s*\([^)]*\)\s*\{\s*\}`)
	emptyExceptRegex   = regexp.MustCompile(`(?m)except\s*(Exception)?\s*:\s*pass\b`)
	// Hanya paket net/http (http.Get), bukan method .Get milik klien sendiri (s.http.Get(ctx, url)).
	noTimeoutClientRegex = regexp.MustCompile(`(&http\.Client\{\s*\}|(?:^|[^\w.])http\.(Get|Post|Head)\s*\()`)
	hardcodedSecretRegex = regexp.MustCompile(`(?i)(jwt[_-]?secret|api[_-]?key|db[_-]?password|auth[_-]?token)\s*[:=]\s*["']([a-zA-Z0-9_\-+=/]{12,})["']`)

	// Security Critical: Command Injection / RCE
	commandInjectionRegex = regexp.MustCompile(`(?i)(exec\.Command\s*\(|child_process\.(exec|execSync)\s*\(|\beval\s*\(|\bos\.system\s*\(|\bsubprocess\.(Popen|call|run)\s*\([^)]*shell\s*=\s*True|\bpassthru\s*\(|\bshell_exec\s*\()`)

	// Security Critical: Path Traversal
	pathTraversalRegex = regexp.MustCompile(`(?i)(os\.ReadFile|ioutil\.ReadFile|fs\.readFileSync|open)\s*\([^)]*(\+|concat|\$\{)[^)]*(req\.|c\.Params|c\.Query|params\.|query\.)`)

	// Security: Dangerous HTML Injection / XSS in Frontend
	unsafeHtmlRegex = regexp.MustCompile(`(?i)(v-html\s*=|dangerouslySetInnerHTML\s*=|(?:\.|\b)innerHTML\s*=)`)

	// Security: deklarasi rute autentikasi (login/register/reset) yang rawan brute-force.
	// Dibatasi ke method submisi (POST/PUT/PATCH) agar halaman GET login tidak ikut terpicu.
	authRouteRegex = regexp.MustCompile(`(?i)\.(Post|Put|Patch)\s*\(\s*["']([^"']*(?:login|sign-?in|log-?in|authenticate|register|sign-?up|forgot|reset-?password|/otp|/2fa|/mfa)[^"']*)["']`)
	// rateLimiterRegex: kehadiran middleware/pustaka rate limiting lintas framework (Go, Node, Python).
	rateLimiterRegex = regexp.MustCompile(`(?i)(limiter\.New|rate[\s_-]?limit|rate\.(?:New)?Limiter|express-rate-limit|rateLimit\s*\(|\bthrottle|\w*limiter\w*\s*\(|tollbooth|x/time/rate|ThrottlerGuard|flask[\s_-]?limiter|django_ratelimit|slidingwindow|leaky[\s_-]?bucket|token[\s_-]?bucket|middleware/limiter)`)

	// Performance / Heavy Process: Blocking Synchronous I/O in Handlers
	syncIoRegex = regexp.MustCompile(`(?i)\bfs\.(readFileSync|writeFileSync|appendFileSync)\s*\(`)

	// Performance / Resource: Unbounded Read Body (OOM Hazard)
	unboundedBodyReadRegex = regexp.MustCompile(`(?i)(io\.ReadAll|ioutil\.ReadAll)\s*\([^)]*Body\)`)

	// Smelly Design / UX: Native Browser Dialog Violation (Zero Alert Policy).
	// Hanya panggilan global / window.*; method objek lain (inquirer.prompt, dialog.confirm) diabaikan.
	nativeDialogRegex = regexp.MustCompile(`(?:^|[^\w.$])(?:window\.)?(alert|confirm|prompt)\s*\(|\bwindow\.(alert|confirm|prompt)\s*\(`)
)

// InspectProjectCode analyzes all source files within project components for query, router, and code smells
func InspectProjectCode(project *models.Project) []models.Finding {
	if project == nil {
		return nil
	}

	var allFindings []models.Finding

	// Collect unique component paths and project root
	visitedPaths := make(map[string]bool)

	for _, comp := range project.Components {
		targetDir := comp.AbsolutePath
		if targetDir == "" && project.RootPath != "" && comp.RelativePath != "" {
			targetDir = filepath.Join(project.RootPath, comp.RelativePath)
		}
		if targetDir == "" || visitedPaths[targetDir] {
			continue
		}
		visitedPaths[targetDir] = true

		if fi, err := os.Stat(targetDir); err == nil && fi.IsDir() {
			findings := InspectDirectory(targetDir, comp.RelativePath)
			allFindings = append(allFindings, findings...)
		}
	}

	// Also inspect root files if not covered
	if project.RootPath != "" && project.RootPath != "-" && !visitedPaths[project.RootPath] {
		// Only check top-level and direct non-component subdirs
		findings := inspectRootLevelCode(project.RootPath)
		allFindings = append(allFindings, findings...)
	}

	return allFindings
}

// InspectComponentCode audits code specifically within one component's directory
func InspectComponentCode(comp *models.Component) []models.Finding {
	if comp == nil || comp.AbsolutePath == "" {
		return nil
	}
	fi, err := os.Stat(comp.AbsolutePath)
	if err != nil || !fi.IsDir() {
		return nil
	}
	return InspectDirectory(comp.AbsolutePath, comp.RelativePath)
}

// InspectDirectory walks a directory and inspects code files up to a reasonable depth
func InspectDirectory(dir string, baseRelPath string) []models.Finding {
	var findings []models.Finding
	maxFiles := 1200
	scannedFiles := 0
	hasGo, hasPHP := false, false
	var inspected []string // berkas sumber yang diaudit, dibaca ulang hanya untuk deteksi object storage
	gitIgnored := gitIgnoredPaths(dir)

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if rel, relErr := filepath.Rel(dir, path); relErr == nil && gitIgnored[rel] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			name := info.Name()
			if ignoredDirNames[name] || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if scannedFiles >= maxFiles {
			return filepath.SkipDir
		}

		ext := strings.ToLower(filepath.Ext(path))
		relPath, _ := filepath.Rel(dir, path)
		if baseRelPath != "" {
			relPath = filepath.Join(baseRelPath, relPath)
		}

		// Konfigurasi paket Flutter (lint) diperiksa dari pubspec.yaml-nya
		if info.Name() == "pubspec.yaml" {
			findings = append(findings, inspectFlutterProject(path, relPath)...)
			return nil
		}

		// Aset media (gambar/video) diperiksa ukurannya saja, tanpa membaca isi
		if _, isAsset := auditAssetExtensions[ext]; isAsset {
			if f := InspectAssetFile(relPath, info.Size()); f != nil {
				findings = append(findings, *f)
			}
			return nil
		}

		lang, ok := auditFileExtensions[ext]
		if !ok || (lang == "xml" && !isAndroidLayoutXML(relPath)) {
			return nil
		}

		// Skip test files, mock files, minified bundles, dan keluaran generator kode
		if isTestOrGeneratedFile(info.Name()) {
			return nil
		}

		hasGo = hasGo || lang == "go"
		hasPHP = hasPHP || lang == "php"
		inspected = append(inspected, path)
		fileFindings := InspectFile(path, relPath, lang)
		findings = append(findings, fileFindings...)
		scannedFiles++

		return nil
	})

	findings = dropStorageWhenObjectStore(findings, inspected)
	findings = refineRouterFindings(findings, dir, baseRelPath, hasGo)
	if hasPHP {
		findings = append(findings, phpCostFindings(dir, baseRelPath, findings)...)
	}
	return findings
}

func inspectRootLevelCode(rootPath string) []models.Finding {
	var findings []models.Finding
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return findings
	}

	var inspected []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if lang, ok := auditFileExtensions[ext]; ok && lang != "xml" {
			fullPath := filepath.Join(rootPath, entry.Name())
			inspected = append(inspected, fullPath)
			findings = append(findings, InspectFile(fullPath, entry.Name(), lang)...)
		}
	}
	return dropStorageWhenObjectStore(findings, inspected)
}

// isServerLang menandai bahasa yang lazim dipakai untuk kode server/backend.
func isServerLang(lang string) bool {
	switch lang {
	case "go", "python", "php", "java", "kotlin", "typescript", "javascript":
		return true
	}
	return false
}

// isUISourceFile menandai berkas antarmuka web: komponen .vue/.svelte/.tsx/.jsx atau TS/JS di folder
// sumber frontend (components/, pages/, composables/, ...). Nuxt server/ & api/ tidak termasuk.
func isUISourceFile(relPath string, lang string) bool {
	if lang == "vue" || lang == "svelte" {
		return true
	}
	lower := strings.ToLower(filepath.ToSlash(relPath))
	if strings.HasSuffix(lower, ".tsx") || strings.HasSuffix(lower, ".jsx") {
		return !strings.Contains(lower, "server/")
	}
	if lang != "typescript" && lang != "javascript" {
		return false
	}
	// Konfigurasi framework frontend (nuxt/vite/next/svelte/astro/tailwind/app.config) = kode web
	base := filepath.Base(lower)
	for _, prefix := range []string{"nuxt.config.", "vite.config.", "next.config.", "svelte.config.", "astro.config.", "tailwind.config.", "app.config.", "vue.config."} {
		if strings.HasPrefix(base, prefix) {
			return true
		}
	}
	return isFrontendSourcePath(lower)
}

// isAndroidSourceFile menandai Kotlin/Java aplikasi Android (bukan server JVM seperti Spring/Ktor).
func isAndroidSourceFile(relPath string, content string) bool {
	lower := strings.ToLower(filepath.ToSlash(relPath))
	if strings.Contains(content, "import android.") || strings.Contains(content, "import androidx.") {
		return true
	}
	// Berkas Android tanpa import platform (mis. util) tetap dikenali dari struktur modul app/
	return strings.Contains(lower, "/src/main/") && (strings.Contains(lower, "app/src/") || strings.Contains(lower, "android/"))
}

// InspectFile performs in-depth AST & regex audits on a single source file
// hasAuthMiddleware: baris memuat middleware autentikasi/otorisasi (heuristik nama).
func hasAuthMiddleware(line string) bool {
	for _, k := range []string{"Auth", "Jwt", "JWT", "Protect", "Guard", "Require", "Role", "Perm", "Session", "Token", "Secure", "localAccess", "LocalAccess", "ExecAccess"} {
		if strings.Contains(line, k) {
			return true
		}
	}
	return false
}

// routeOnGuardedGroup: rute didaftarkan pada group yang middleware auth-nya dipasang di level group.
func routeOnGuardedGroup(line string, guarded map[string]bool) bool {
	m := routeReceiverRegex.FindStringSubmatch(line)
	return m != nil && guarded[m[1]]
}

func InspectFile(fullPath string, relPath string, lang string) []models.Finding {
	findings := make([]models.Finding, 0)

	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return findings
	}

	content := string(contentBytes)
	if isGeneratedContent(content) {
		return findings
	}
	lines := strings.Split(content, "\n")

	// Layout XML Android hanya diperiksa aturan tampilan (bukan aturan kode/server)
	if lang == "xml" {
		return append(findings, inspectUIPerformance(content, relPath, lang, lines)...)
	}

	// Klasifikasi bahasa & peran berkas agar setiap aturan hanya berlaku pada ekosistem yang tepat:
	// aturan server (rute, CORS, SQL, path traversal) tidak dijalankan pada komponen UI web,
	// aplikasi Android, maupun Flutter; aturan UI web tidak dijalankan pada kode server.
	scan := newFileScan(relPath, lang, content)
	// Aturan Flutter hanya untuk Dart milik aplikasi Flutter, bukan Dart murni (server/CLI)
	isFlutter := lang == "dart" && isFlutterSource(fullPath, content)

	// Smell 1: God Controller / Fat File (>400 baris KODE di berkas handler/controller; baris kosong, komentar,
	// dan berkas tipe/data tidak dihitung)
	if len(lines) > godControllerLines && scan.isControllerOrHandler && !isTypeOrDataFile(relPath) {
		if code := countCodeLines(lines, lang); code > godControllerLines && godControllerApplies(relPath, lang, lines, code) {
			findings = append(findings, godControllerFinding(relPath, lines))
		}
	}

	findings = append(findings, scanCodeLines(scan, lines)...)
	return append(findings, inspectFileLevel(scan, isFlutter, content, lines)...)
}

// newFileScan mengklasifikasikan berkas (UI web / Android / server / SQL) untuk aturan per-baris.
func newFileScan(relPath, lang, content string) *fileScan {
	isUI := isUISourceFile(relPath, lang)
	isAndroid := (lang == "kotlin" || lang == "java") && isAndroidSourceFile(relPath, content)
	isServer := isServerLang(lang) && !isUI && !isAndroid
	return &fileScan{
		relPath:               relPath,
		lang:                  lang,
		isUI:                  isUI,
		isAndroid:             isAndroid,
		isServer:              isServer,
		usesSQL:               isServer || lang == "dart" || isAndroid, // sqflite / Room rawQuery juga memakai SQL
		hasRateLimiter:        isServer && rateLimiterRegex.MatchString(content),
		hasBodyLimit:          isServer && bodyLimitRegex.MatchString(content),
		oneOffTool:            isOneOffToolPath(relPath),
		guardedGroups:         map[string]bool{},
		authAlias:             map[string]bool{},
		isControllerOrHandler: isServer && isHandlerPath(relPath),
	}
}

const godControllerLines = 400

const (
	// godControllerHandlers: berkas Go dengan handler HTTP sebanyak ini memegang banyak use case sekaligus.
	godControllerHandlers = 8
	// godControllerHardLines: di atas ukuran ini berkas handler Go tetap dilaporkan walau handlernya sedikit.
	godControllerHardLines = 1200
)

var (
	goHandlerFuncRegex = regexp.MustCompile(`^func\s+(?:\([^)]*\)\s*)?[A-Z]\w*\([^)]*(?:\*fiber\.Ctx|http\.ResponseWriter|\*gin\.Context|echo\.Context)`)
	routeTableRegex    = regexp.MustCompile(`(?i)(^|/)(routes?|router)(/|\.[a-z]+$|_[a-z]+\.[a-z]+$)`)
)

// godControllerApplies: ukuran saja tidak membuktikan pelanggaran SRP. Berkas tabel rute dilewati; di Go,
// berkas yang sudah dipecah per fitur (sedikit handler, banyak fungsi bantu kecil) tidak dilaporkan — yang
// dilaporkan berkas dengan banyak handler (banyak use case) atau yang sangat besar. Bahasa lain memakai batas baris.
func godControllerApplies(relPath, lang string, lines []string, code int) bool {
	if routeTableRegex.MatchString(filepath.ToSlash(relPath)) {
		return false
	}
	if lang != "go" {
		return true
	}
	return code > godControllerHardLines || countGoHandlers(lines) >= godControllerHandlers
}

func countGoHandlers(lines []string) int {
	n := 0
	for _, l := range lines {
		if goHandlerFuncRegex.MatchString(l) {
			n++
		}
	}
	return n
}

var (
	bodyLimitRegex = regexp.MustCompile(`(MaxBytesReader|LimitReader|BodyLimit|LimitRequestBody)`)
	// handlerDirRegex / handlerFileRegex: berkas lapisan HTTP dikenali dari SEGMEN path, bukan sembarang substring
	// ("api" di types/api.ts, api-docs.ts, api_service.dart bukan handler).
	handlerDirRegex  = regexp.MustCompile(`(^|/)(handlers?|controllers?|routers?|routes?|api|apis|endpoints?|http)(/|$)`)
	handlerFileRegex = regexp.MustCompile(`(^|[_.-])(handlers?|controllers?)([_.-]|$)|controller\.[a-z]+$|handler\.[a-z]+$`)
	typeOrDataRegex  = regexp.MustCompile(`(^|/)(types?|typings|data|fixtures|mocks?|docs?|schemas?|generated|constants?)(/|$)|[_.-](docs|types|raw|schema|data|constants)\.[a-z]+$`)
	oneOffToolRegex  = regexp.MustCompile(`(^|/)(cmd/bench[^/]*|cmd/[^/]*(?:check|fix|backfill|cleanup|repair|reindex|maint)[^/]*|bench(marks?)?|scripts?|seeds?|seeders?|migrations?|fixtures|tools)(/|$)|(^|[_/-])(seed|seeder|migrate|migration|bench)[_.-]`)
)

// isHandlerPath: berkas berada di lapisan HTTP (handler/controller/router) menurut folder atau nama berkasnya.
func isHandlerPath(relPath string) bool {
	lower := strings.ToLower(filepath.ToSlash(relPath))
	dir, base := filepath.ToSlash(filepath.Dir(lower)), filepath.Base(lower)
	// Django: views.py / urls.py adalah lapisan HTTP-nya.
	return handlerDirRegex.MatchString(dir) || handlerFileRegex.MatchString(base) || base == "views.py" || base == "urls.py"
}

// isTypeOrDataFile: berkas deklarasi tipe atau data statis (panjangnya bukan tanda terlalu banyak tanggung jawab).
func isTypeOrDataFile(relPath string) bool {
	return typeOrDataRegex.MatchString(strings.ToLower(filepath.ToSlash(relPath)))
}

// isOneOffToolPath: skrip sekali jalan (benchmark, seed, migrasi, alat bantu); query di dalam loopnya disengaja.
func isOneOffToolPath(relPath string) bool {
	return oneOffToolRegex.MatchString(strings.ToLower(filepath.ToSlash(relPath)))
}

// countCodeLines: jumlah baris yang memuat kode setelah komentar dikosongkan.
func countCodeLines(lines []string, lang string) int {
	n := 0
	for _, l := range stripCode(lines, lang) {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}

// inspectFileLevel menjalankan pemeriksaan tingkat berkas (bukan per-baris) sesuai peran berkas.
func inspectFileLevel(scan *fileScan, isFlutter bool, content string, lines []string) []models.Finding {
	var findings []models.Finding
	relPath, lang := scan.relPath, scan.lang

	// Best practice Flutter: lifecycle controller, build(), async gap, penyimpanan aman, logging
	if isFlutter {
		findings = append(findings, inspectFlutterBestPractices(content, relPath, lines)...)
	}

	// Lokasi simpan/muat berkas harus tetap setelah aplikasi di-build & diunggah ke server
	if scan.isServer {
		findings = append(findings, inspectStoragePaths(content, relPath, lang, lines)...)
	}

	// Go-specific block analysis for Resource Leaks (Cursor Close & Tx Rollback)
	if lang == "go" {
		findings = append(findings, inspectGoResourceLeaks(content, relPath)...)
	}

	if lang == "typescript" || lang == "javascript" || lang == "vue" || lang == "svelte" {
		findings = append(findings, inspectDuplicateImports(relPath, lines)...)
	}

	// Frontend-specific inspection for Tab API Caching & Overfetching
	if scan.isUI {
		findings = append(findings, inspectFrontendSmells(content, relPath, lines)...)
	}

	// Tampilan & performa UI (FE, Android, Flutter): ukuran berkas, a11y, layout, rendering.
	// Dart non-Flutter tidak punya widget sehingga aturan tampilan dilewati.
	if lang != "dart" || isFlutter {
		findings = append(findings, inspectUIPerformance(content, relPath, lang, lines)...)
	}
	return findings
}

// firstLineContaining mengembalikan indeks baris pertama yang memuat needle (-1 bila tidak ada).
func firstLineContaining(lines []string, needle string) int {
	for idx, line := range lines {
		if strings.Contains(line, needle) {
			return idx
		}
	}
	return -1
}

// overfetchingFinding: komponen tab memanggil getProjectBaseKnowledge tanpa perlu.
func overfetchingFinding(content, relPath string, lines []string, isTabComponent bool) []models.Finding {
	if !strings.Contains(content, "getProjectBaseKnowledge") || !isTabComponent {
		return nil
	}
	if strings.Contains(relPath, "BaseKnowledgeTab") || strings.Contains(relPath, "base-knowledge") {
		return nil
	}
	idx := firstLineContaining(lines, "getProjectBaseKnowledge")
	if idx < 0 {
		return nil
	}
	return []models.Finding{{
		Category:     "frontend",
		Severity:     "WARNING",
		RuleID:       "FE-OVERFETCHING-UNUSED-DATA",
		Title:        "Overfetching API: Pemanggilan Data Base Knowledge pada Komponen yang Tidak Membutuhkannya",
		Description:  fmt.Sprintf("Komponen '%s' pada baris %d memanggil 'getProjectBaseKnowledge()'. Endpoint ini menyusun pohon direktori, rute, dan env yang berat. Komponen ini sebaiknya tidak memanggil data tersebut atau menggunakan data yang sudah ada di props.", filepath.Base(relPath), idx+1),
		Suggestion:   "Hapus pemanggilan 'getProjectBaseKnowledge()' dari komponen ini. Gunakan computed property dari props.project atau cached data.",
		TargetFile:   relPath,
		Line:         idx + 1,
		Snippet:      strings.TrimSpace(lines[idx]),
		TheoryBasis:  "REST/GraphQL Resource Optimization & Network Payload Minimization - Mencegah dead data fetching.",
		SuggestedFix: "// Hindari memanggil endpoint agregasi berat:\nconst count = computed(() => props.project?.components?.length || 0)",
	}}
}

// uncachedReloadFinding: tab memuat API pada onMounted tanpa pengecekan cache.
func uncachedReloadFinding(content, relPath string, lines []string, isTabComponent bool) []models.Finding {
	if !isTabComponent || !strings.Contains(content, "onMounted") {
		return nil
	}
	hasApiCallInMount := util.ContainsAny(content, "loadBaseKnowledge(", "loadInitialData(", "fetchData(", "loadData(", "getProject(", "$fetch(")
	hasCacheCheck := util.ContainsAny(content, "cachedData", "cachedBaseKnowledge", "cache", "if (!data.value)", "if (!props.", "v-model:cached")
	if !hasApiCallInMount || hasCacheCheck {
		return nil
	}
	idx := firstLineContaining(lines, "onMounted")
	if idx < 0 {
		return nil
	}
	return []models.Finding{{
		Category:     "frontend",
		Severity:     "WARNING",
		RuleID:       "FE-API-RELOAD-UNCACHED",
		Title:        "Pemanggilan API Berulang Tanpa Caching Variabel (Tab Switch Reload Smell)",
		Description:  fmt.Sprintf("Komponen sub-tab '%s' pada baris %d memanggil pemuatan data API pada hook onMounted() tanpa mekanisme caching variabel/state. Saat pengguna berpindah tab, data akan selalu di-reload ulang dari backend.", filepath.Base(relPath), idx+1),
		Suggestion:   "Simpan data pada state/ref di level halaman induk atau composable, lalu berikan cachedData sebagai prop ke sub-tab agar tidak reload saat tab berganti.",
		TargetFile:   relPath,
		Line:         idx + 1,
		Snippet:      strings.TrimSpace(lines[idx]),
		TheoryBasis:  "Single Page Application (SPA) State Management & Component Lifecycle Optimization.",
		SuggestedFix: "// Simpan data di ref induk dan berikan sebagai prop:\nconst props = defineProps<{ cachedData?: MyData }>()\nonMounted(() => {\n  if (!props.cachedData) loadData()\n})",
	}}
}

// inspectFrontendSmells audits frontend components for lifecycle API reloads, variable caching, and overfetching
func inspectFrontendSmells(content string, relPath string, lines []string) []models.Finding {
	isTabComponent := strings.Contains(relPath, "Tab.vue") ||
		strings.Contains(relPath, "/tabs/") ||
		(strings.Contains(relPath, "detail/") && strings.HasSuffix(relPath, ".vue"))

	var findings []models.Finding
	findings = append(findings, overfetchingFinding(content, relPath, lines, isTabComponent)...)
	findings = append(findings, uncachedReloadFinding(content, relPath, lines, isTabComponent)...)
	return findings
}

// inspectGoResourceLeaks looks for unclosed db.Query results or transaction leaks
func inspectGoResourceLeaks(content string, relPath string) []models.Finding {
	var findings []models.Finding
	lines := strings.Split(content, "\n")

	// Check db.Query without defer rows.Close()
	queryMatches := dbQueryGoRegex.FindAllStringSubmatchIndex(content, -1)
	if len(queryMatches) > 0 && !deferCloseRegex.MatchString(content) {
		firstIdx := queryMatches[0][0]
		lineNum := countLinesUntil(content, firstIdx)
		snippet := ""
		if lineNum-1 < len(lines) {
			snippet = strings.TrimSpace(lines[lineNum-1])
		}

		findings = append(findings, models.Finding{
			Category:     "query",
			Severity:     "CRITICAL",
			RuleID:       "QUERY-RESOURCE-LEAK",
			Title:        "Database Connection Leak: sql.Rows Tidak Ditutup (Missing defer Close)",
			Description:  fmt.Sprintf("Pemanggilan db.Query pada baris %d tidak diikuti dengan 'defer rows.Close()'. Ini menyebabkan koneksi database menggantung dan kolapsnya connection pool.", lineNum),
			Suggestion:   "Segera tambahkan 'defer rows.Close()' tepat setelah pemeriksaan 'if err != nil'.",
			TargetFile:   relPath,
			Line:         lineNum,
			Snippet:      snippet,
			TheoryBasis:  "RAII (Resource Acquisition Is Initialization) Pattern & Database Connection Pool Lifecycle Management.",
			SuggestedFix: "rows, err := db.Query(...)\nif err != nil {\n    return err\n}\ndefer rows.Close() // Pastikan cursor ditutup dan koneksi dikembalikan ke pool",
		})
	}

	// Check db.Begin without defer tx.Rollback()
	txMatches := dbBeginGoRegex.FindAllStringSubmatchIndex(content, -1)
	if len(txMatches) > 0 && !deferTxRollback.MatchString(content) {
		firstIdx := txMatches[0][0]
		lineNum := countLinesUntil(content, firstIdx)
		snippet := ""
		if lineNum-1 < len(lines) {
			snippet = strings.TrimSpace(lines[lineNum-1])
		}

		findings = append(findings, models.Finding{
			Category:     "query",
			Severity:     "WARNING",
			RuleID:       "QUERY-MISSING-TX-ROLLBACK",
			Title:        "Dangling Database Transaction: Transaksi Tanpa Guard Rollback",
			Description:  fmt.Sprintf("Transaksi database dibuka pada baris %d tanpa pengaman rollback otomatis jika terjadi panic atau error mendadak.", lineNum),
			Suggestion:   "Pasang 'defer tx.Rollback()' setelah memulai transaksi. Jika transaksi berhasil dicommit, rollback akan diabaikan secara aman.",
			TargetFile:   relPath,
			Line:         lineNum,
			Snippet:      snippet,
			TheoryBasis:  "ACID Principles (Atomicity & Isolation) - Mencegah deadlock dan kebocoran row-level locking.",
			SuggestedFix: "tx := db.Begin()\ndefer tx.Rollback() // Aman: jika tx.Commit() berhasil, Rollback() akan menjadi no-op",
		})
	}

	return findings
}

func countLinesUntil(s string, byteOffset int) int {
	if byteOffset > len(s) {
		byteOffset = len(s)
	}
	scanner := bufio.NewScanner(strings.NewReader(s[:byteOffset]))
	count := 0
	for scanner.Scan() {
		count++
	}
	if count == 0 {
		return 1
	}
	return count
}

func stubHandlerFinding(relPath string, lineNum int, snippet string) models.Finding {
	return models.Finding{
		Category:     "router",
		Severity:     "INFO",
		RuleID:       "ROUTER-STUB-HANDLER",
		Title:        "Handler Rute Belum Terimplementasi (Dangling / Stub Endpoint)",
		Description:  fmt.Sprintf("Terdeteksi penanda handler yang belum selesai atau bernilai placeholder pada baris %d.", lineNum),
		Suggestion:   "Selesaikan logika bisnis handler atau kembalikan HTTP status 501 (Not Implemented) dengan dokumentasi error yang jelas.",
		TargetFile:   relPath,
		Line:         lineNum,
		Snippet:      snippet,
		TheoryBasis:  "Richardson Maturity Model (Level 2) & Robustness Principle (Postel's Law).",
		SuggestedFix: "// Kembalikan status HTTP 501 jika endpoint belum selesai:\nreturn c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{\"error\": \"Endpoint dalam tahap pengembangan\"})",
	}
}

// AuditableFile adalah berkas sumber yang lolos filter audit beserta bahasanya
type AuditableFile struct {
	AbsPath string
	RelPath string
	Lang    string
	Lines   int
	// IsUI / IsAndroid / IsServer mengikuti klasifikasi yang sama dengan InspectFile
	IsUI      bool
	IsAndroid bool
	IsServer  bool
	IsFlutter bool
}

// ListAuditableFiles mengembalikan berkas kode yang akan diaudit pada sebuah direktori komponen,
// memakai filter folder, ekstensi, dan pengecualian test yang sama dengan InspectDirectory.
func ListAuditableFiles(dir string, baseRelPath string, maxFiles int) []AuditableFile {
	var out []AuditableFile
	gitIgnored := gitIgnoredPaths(dir)
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if rel, relErr := filepath.Rel(dir, path); relErr == nil && gitIgnored[rel] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			if ignoredDirNames[info.Name()] || (strings.HasPrefix(info.Name(), ".") && path != dir) {
				return filepath.SkipDir
			}
			return nil
		}
		if maxFiles > 0 && len(out) >= maxFiles {
			return filepath.SkipAll
		}
		relPath, _ := filepath.Rel(dir, path)
		if baseRelPath != "" {
			relPath = filepath.Join(baseRelPath, relPath)
		}
		if f, ok := auditableFileAt(path, relPath, info.Name()); ok {
			out = append(out, f)
		}
		return nil
	})
	return out
}

// auditableFileAt membaca satu berkas dan mengklasifikasikannya; ok=false bila tidak diaudit.
func auditableFileAt(path, relPath, name string) (AuditableFile, bool) {
	lang, ok := auditFileExtensions[strings.ToLower(filepath.Ext(path))]
	if !ok || (lang == "xml" && !isAndroidLayoutXML(relPath)) || isTestOrGeneratedFile(name) {
		return AuditableFile{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return AuditableFile{}, false
	}
	content := string(data)
	if isGeneratedContent(content) {
		return AuditableFile{}, false
	}
	isUI := isUISourceFile(relPath, lang)
	isAndroid := (lang == "kotlin" || lang == "java") && isAndroidSourceFile(relPath, content)
	return AuditableFile{
		AbsPath:   path,
		RelPath:   relPath,
		Lang:      lang,
		Lines:     strings.Count(content, "\n") + 1,
		IsUI:      isUI,
		IsAndroid: isAndroid,
		IsServer:  isServerLang(lang) && !isUI && !isAndroid,
		IsFlutter: lang == "dart" && isFlutterSource(path, content),
	}, true
}
