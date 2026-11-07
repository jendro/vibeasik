package diagnostics

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
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
	dbQueryCallRegex = regexp.MustCompile(`(?i)(db\.(Find|Where|First|Last|Query|QueryRow|Exec|Raw|Take)|repo\.(Find|Get|List|Query)|prisma\.[a-zA-Z]+\.(find|create|update|delete)|sequelize\.|Model::(find|where|all)|\$this->db->(query|get|where)|session\.(query|execute)|objects\.(filter|get|all))`)

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
	stubHandlerRegex   = regexp.MustCompile(`(?i)(//\s*TODO:?\s*(implement|fix|handle)|c\.SendString\(["']TODO["']\)|return.*status.*not implemented|\bpanic\(["']not implemented["']\)|throw\s+(new\s+)?(NotImplementedError|UnsupportedOperationException|NotImplementedException)|\bTODO\(\))`)
	stubPythonRegex    = regexp.MustCompile(`(?i)(^pass$|raise\s+NotImplementedError|#\s*TODO:?\s*(implement|fix|handle))`)
	routerVerbRegex    = regexp.MustCompile(`(?i)\.(Get|Post|Put|Delete)\s*\(\s*["']\/[a-zA-Z0-9_/-]*(get|fetch|delete|remove|create|add|update|edit)[A-Z_][a-zA-Z0-9_/-]*["']`)

	// Code smells
	swallowedErrorGoRegex = regexp.MustCompile(`(?m)^\s*_\s*,\s*err\s*:?=`)
	ignoredErrGoRegex     = regexp.MustCompile(`(?m)^\s*_\s*=\s*err\b`)
	emptyCatchRegex       = regexp.MustCompile(`(?m)catch\s*\([^)]*\)\s*\{\s*\}`)
	emptyExceptRegex      = regexp.MustCompile(`(?m)except\s*(Exception)?\s*:\s*pass\b`)
	noTimeoutClientRegex  = regexp.MustCompile(`(&http\.Client\{\s*\}|http\.(Get|Post|Head)\s*\()`)
	hardcodedSecretRegex  = regexp.MustCompile(`(?i)(jwt[_-]?secret|api[_-]?key|db[_-]?password|auth[_-]?token)\s*[:=]\s*["']([a-zA-Z0-9_\-+=/]{12,})["']`)

	// Security Critical: Command Injection / RCE
	commandInjectionRegex = regexp.MustCompile(`(?i)(exec\.Command\s*\(|child_process\.(exec|execSync)\s*\(|\beval\s*\(|\bos\.system\s*\(|\bsubprocess\.(Popen|call|run)\s*\([^)]*shell\s*=\s*True|\bpassthru\s*\(|\bshell_exec\s*\()`)

	// Security Critical: Path Traversal
	pathTraversalRegex = regexp.MustCompile(`(?i)(os\.ReadFile|ioutil\.ReadFile|fs\.readFileSync|open)\s*\([^)]*(\+|concat|\$\{)[^)]*(req\.|c\.Params|c\.Query|params\.|query\.)`)

	// Security: Dangerous HTML Injection / XSS in Frontend
	unsafeHtmlRegex = regexp.MustCompile(`(?i)(v-html\s*=|dangerouslySetInnerHTML\s*=|(?:\.|\b)innerHTML\s*=)`)

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
	hasGo := false
	var inspected []string // berkas sumber yang diaudit, dibaca ulang hanya untuk deteksi object storage

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
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
		inspected = append(inspected, path)
		fileFindings := InspectFile(path, relPath, lang)
		findings = append(findings, fileFindings...)
		scannedFiles++

		return nil
	})

	findings = dropStorageWhenObjectStore(findings, inspected)
	return refineRouterFindings(findings, dir, baseRelPath, hasGo)
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
	for _, k := range []string{"Auth", "Jwt", "JWT", "Protect", "Guard", "Require", "Role", "Perm", "Session", "Token", "Secure"} {
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
	lineCount := len(lines)

	// Layout XML Android hanya diperiksa aturan tampilan (bukan aturan kode/server)
	if lang == "xml" {
		return append(findings, inspectUIPerformance(content, relPath, lang, lines)...)
	}

	// Klasifikasi bahasa & peran berkas agar setiap aturan hanya berlaku pada ekosistem yang tepat:
	// aturan server (rute, CORS, SQL, path traversal) tidak dijalankan pada komponen UI web,
	// aplikasi Android, maupun Flutter; aturan UI web tidak dijalankan pada kode server.
	isUI := isUISourceFile(relPath, lang)
	isAndroid := (lang == "kotlin" || lang == "java") && isAndroidSourceFile(relPath, content)
	isServer := isServerLang(lang) && !isUI && !isAndroid
	// Aturan Flutter hanya untuk Dart milik aplikasi Flutter, bukan Dart murni (server/CLI)
	isFlutter := lang == "dart" && isFlutterSource(fullPath, content)
	usesSQL := isServer || lang == "dart" || isAndroid // sqflite / Room rawQuery juga memakai SQL

	// Smell 1: God Controller / Fat File (>400 LOC in handler/controller/service)
	isControllerOrHandler := isServer && (strings.Contains(relPath, "handler") ||
		strings.Contains(relPath, "controller") ||
		strings.Contains(relPath, "router") ||
		strings.Contains(relPath, "api"))

	if lineCount > 400 && isControllerOrHandler {
		findings = append(findings, models.Finding{
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
		})
	}

	// Line-by-line inspection with context tracking
	inLoop := false
	loopDepth := 0
	// guardedGroups: variabel router group yang sudah dijaga middleware auth (Group(..., Auth) / .Use(Auth));
	// group turunan dari group terjaga ikut terjaga, jadi rute di dalamnya bukan ROUTER-UNGUARDED-MUTATION.
	guardedGroups := map[string]bool{}

	for idx, line := range lines {
		lineNum := idx + 1
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") ||
			((lang == "python" || lang == "php") && strings.HasPrefix(trimmed, "#")) {
			if stubHandlerRegex.MatchString(trimmed) || (lang == "python" && stubPythonRegex.MatchString(trimmed)) {
				findings = append(findings, stubHandlerFinding(relPath, lineNum, trimmed))
			}
			continue
		}

		// Track loop blocks
		if loopKeywordRegex.MatchString(trimmed) {
			inLoop = true
			loopDepth++
		}

		if inLoop {
			if strings.Contains(trimmed, "{") {
				loopDepth += strings.Count(trimmed, "{") - 1
			}
			if strings.Contains(trimmed, "}") {
				loopDepth -= strings.Count(trimmed, "}")
				if loopDepth <= 0 {
					inLoop = false
					loopDepth = 0
				}
			}

			// Rule DB-001: N+1 Query in Loop
			if usesSQL && dbQueryCallRegex.MatchString(trimmed) {
				findings = append(findings, models.Finding{
					Category:     "query",
					Severity:     "CRITICAL",
					RuleID:       "QUERY-NPLUS1",
					Title:        "N+1 Query Anti-Pattern: Pemanggilan Database di Dalam Loop",
					Description:  fmt.Sprintf("Pemanggilan kueri database terdeteksi di dalam perulangan (baris %d). Ini memicu latensi kumulatif O(N) dan saturasi connection pool.", lineNum),
					Suggestion:   "Gunakan eager loading (GORM Preload, SQL JOIN, atau WHERE IN batching) di luar perulangan sebelum iterasi dilakukan.",
					TargetFile:   relPath,
					Line:         lineNum,
					Snippet:      trimmed,
					TheoryBasis:  "Martin Fowler PoEAA (Patterns of Enterprise Application Architecture): N+1 Selects Anti-Pattern. Menyebabkan bottleneck I/O jaringan database.",
					SuggestedFix: "// Refactor ke Eager Loading (Contoh GORM):\n// SEBELUM:\n// for _, user := range users { db.Find(&orders, \"user_id = ?\", user.ID) }\n// SESUDAH:\n// db.Preload(\"Orders\").Find(&users)",
				})
			}
		}

		// Rule DB-002: SQL Injection / Unsafe Concatenation
		if usesSQL && sqlConcatRegex.MatchString(trimmed) {
			findings = append(findings, models.Finding{
				Category:     "query",
				Severity:     "CRITICAL",
				RuleID:       "QUERY-SQLI",
				Title:        "Potensi Kerentanan SQL Injection (String Concatenation Query)",
				Description:  fmt.Sprintf("Query SQL dibentuk menggunakan penggabungan string langsung atau formatting dinamis pada baris %d.", lineNum),
				Suggestion:   "Gunakan parameterized query (placeholder '?' atau prepared statement) agar input pengguna diperlakukan sebagai literal nilai, bukan kode SQL.",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "OWASP Top 10 A03:2021-Injection & CWE-89 (Improper Neutralization of Special Elements in SQL Command).",
				SuggestedFix: "// Ganti penggabungan string dengan parameterized query:\n// SEBELUM: db.Raw(fmt.Sprintf(\"SELECT * FROM users WHERE id = %s\", id))\n// SESUDAH: db.Raw(\"SELECT * FROM users WHERE id = ?\", id)",
			})
		}

		// Rule DB-003: SELECT * Anti-Pattern
		if usesSQL && selectStarRegex.MatchString(trimmed) && !strings.Contains(trimmed, "COUNT(*)") {
			findings = append(findings, models.Finding{
				Category:     "query",
				Severity:     "WARNING",
				RuleID:       "QUERY-SELECT-STAR",
				Title:        "SELECT * Anti-Pattern: Pengambilan Kolom Tanpa Batas (Projection Smell)",
				Description:  fmt.Sprintf("Penggunaan 'SELECT *' terdeteksi pada baris %d. Mengambil semua kolom meningkatkan beban memori, transfer data jaringan, dan mematikan optimasi covering index.", lineNum),
				Suggestion:   "Sebutkan hanya kolom yang dibutuhkan secara eksplisit (misal: 'SELECT id, name, email FROM ...') dan tambahkan klausa batas (LIMIT/Pagination).",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "C.J. Date Relational Model Principles & Baron Schwartz High Performance MySQL Indexing Theory.",
				SuggestedFix: "// Nyatakan kolom secara spesifik:\n// SEBELUM: db.Raw(\"SELECT * FROM orders WHERE status = ?\", status)\n// SESUDAH: db.Select(\"id, order_number, total, status\").Where(\"status = ?\", status).Limit(50)",
			})
		}

		if m := routeGroupRegex.FindStringSubmatch(trimmed); m != nil {
			guardedGroups[m[1]] = guardedGroups[m[2]] || hasAuthMiddleware(trimmed)
		} else if m := routeUseRegex.FindStringSubmatch(trimmed); m != nil && hasAuthMiddleware(trimmed) {
			guardedGroups[m[1]] = true
		}

		// Rule ROUTE-001: Unguarded State-Altering Route (POST, PUT, DELETE without Auth)
		if matches := routeMutationRegex.FindStringSubmatch(trimmed); isServer && len(matches) >= 3 && !routeOnGuardedGroup(trimmed, guardedGroups) {
			httpMethod := strings.ToUpper(matches[1])
			routePath := matches[2]
			// Exclude common public auth endpoints
			isPublicAuth := strings.Contains(routePath, "login") ||
				strings.Contains(routePath, "register") ||
				strings.Contains(routePath, "forgot") ||
				strings.Contains(routePath, "callback") ||
				strings.Contains(routePath, "webhook") ||
				strings.Contains(routePath, "public") ||
				strings.Contains(routePath, "scan") ||
				strings.Contains(routePath, "health")

			hasAuthGuard := hasAuthMiddleware(trimmed)

			if !isPublicAuth && !hasAuthGuard {
				findings = append(findings, models.Finding{
					Category:     "router",
					Severity:     "WARNING",
					RuleID:       "ROUTER-UNGUARDED-MUTATION",
					Title:        fmt.Sprintf("Rute Mutasi Data (%s %s) Tanpa Proteksi Middleware", httpMethod, routePath),
					Description:  fmt.Sprintf("Rute pengubah status '%s %s' pada baris %d tidak memiliki middleware otorisasi/autentikasi eksplisit.", httpMethod, routePath, lineNum),
					Suggestion:   "Pasang middleware autentikasi (JWT Bearer Token / Session Guard / Role Checker) pada rantai handler rute.",
					TargetFile:   relPath,
					Line:         lineNum,
					Snippet:      trimmed,
					TheoryBasis:  "OWASP API Security Top 10 API1:2023 (Broken Object Level Authorization - BOLA) & API2:2023 (Broken Authentication).",
					SuggestedFix: fmt.Sprintf("// Pasang middleware autentikasi sebelum handler:\n// SESUDAH: api.%s(\"%s\", authMiddleware, handler)", strings.Title(strings.ToLower(httpMethod)), routePath),
				})
			}
		}

		// Rule ROUTE-002: Wildcard CORS
		if isServer && wildcardCorsRegex.MatchString(trimmed) {
			findings = append(findings, models.Finding{
				Category:     "router",
				Severity:     "WARNING",
				RuleID:       "ROUTER-WILDCARD-CORS",
				Title:        "Kebijakan CORS Terlalu Permisif (AllowOrigins: '*')",
				Description:  fmt.Sprintf("Konfigurasi CORS pada baris %d mengizinkan seluruh domain (*). Ini berisiko CSRF dan kebocoran data jika API menangani kredensial pengguna.", lineNum),
				Suggestion:   "Tentukan origin domain frontend secara eksplisit (misal: 'http://localhost:3000' atau domain resmi production).",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "W3C Cross-Origin Resource Sharing Specification & RFC 6454 (The Web Origin Concept).",
				SuggestedFix: "// Batasi AllowOrigins ke domain terpercaya:\n// SESUDAH: cors.New(cors.Config{ AllowOrigins: \"http://localhost:3000, https://myapp.com\" })",
			})
		}

		// Rule ROUTE-003: Stub / Dangling Handler
		if stubHandlerRegex.MatchString(trimmed) || (lang == "python" && stubPythonRegex.MatchString(trimmed)) {
			findings = append(findings, stubHandlerFinding(relPath, lineNum, trimmed))
		}

		// Rule ROUTE-004: REST Verb in Path
		if isServer && routerVerbRegex.MatchString(trimmed) {
			findings = append(findings, models.Finding{
				Category:     "router",
				Severity:     "INFO",
				RuleID:       "ROUTER-VERB-IN-PATH",
				Title:        "Pelanggaran RESTful URI: Menggunakan Kata Kerja di Path Rute",
				Description:  fmt.Sprintf("URL rute pada baris %d mengandung kata kerja (get/create/delete). Dalam paradigma REST, URI adalah kata benda (noun/resource).", lineNum),
				Suggestion:   "Ganti kata kerja pada URI dengan kata benda plural dan gunakan HTTP method yang sesuai (GET, POST, PUT, DELETE).",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "Roy Fielding Architectural Styles and the Design of Network-based Software Architectures (REST Chapter 5).",
				SuggestedFix: "// Ubah pola rute ke standar REST:\n// SEBELUM: app.Get(\"/getUser/:id\", ...)\n// SESUDAH: app.Get(\"/users/:id\", ...)",
			})
		}

		// Rule SMELL-002: Swallowed Error
		if (lang == "go" && (swallowedErrorGoRegex.MatchString(trimmed) || ignoredErrGoRegex.MatchString(trimmed))) ||
			((lang == "javascript" || lang == "typescript" || lang == "vue" || lang == "svelte" || lang == "kotlin" || lang == "java" || lang == "dart" || lang == "php") && emptyCatchRegex.MatchString(trimmed)) ||
			(lang == "python" && emptyExceptRegex.MatchString(trimmed)) {
			findings = append(findings, models.Finding{
				Category:     "code_smell",
				Severity:     "WARNING",
				RuleID:       "SMELL-SWALLOWED-ERROR",
				Title:        "Swallowed Error Smell: Kesalahan / Error Diabaikan Tanpa Penanganan",
				Description:  fmt.Sprintf("Error return diabaikan atau ditangkap tanpa log/handling pada baris %d. Ini memicu silent failure dan inkonsistensi state data.", lineNum),
				Suggestion:   "Periksa error return secara eksplisit. Log error dengan context yang relevan atau teruskan error ke error-handling layer.",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "Jim Shore's Fail-Fast Principle & Google SRE Observability Engineering.",
				SuggestedFix: "// Tangani error secara eksplisit:\n// SESUDAH:\nif err != nil {\n    log.Printf(\"[ERROR] failed operation: %v\", err)\n    return err\n}",
			})
		}

		// Rule SMELL-003: Missing HTTP Client Timeout
		if lang == "go" && noTimeoutClientRegex.MatchString(trimmed) {
			findings = append(findings, models.Finding{
				Category:     "code_smell",
				Severity:     "WARNING",
				RuleID:       "SMELL-MISSING-TIMEOUT",
				Title:        "HTTP Client Tanpa Batas Waktu (Infinite Timeout Hazard)",
				Description:  fmt.Sprintf("Inisialisasi HTTP client default tanpa field Timeout terdeteksi pada baris %d. Permintaan dapat menggantung selamanya jika server tujuan macet.", lineNum),
				Suggestion:   "Atur timeout eksplisit pada struct http.Client (contoh: Timeout: 10 * time.Second) atau sertakan context.WithTimeout.",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "Michael Nygard 'Release It!': Design and Deploy Production-Ready Software (Circuit Breakers & Timeouts).",
				SuggestedFix: "// Gunakan client dengan timeout terkonfigurasi:\nclient := &http.Client{\n    Timeout: 10 * time.Second,\n}",
			})
		}

		// Rule SMELL-004: Hardcoded Secrets
		if matches := hardcodedSecretRegex.FindStringSubmatch(trimmed); len(matches) >= 3 {
			val := matches[2]
			if !strings.Contains(val, "xxx") && !strings.Contains(val, "CHANGE_ME") && !strings.Contains(val, "dummy") && !strings.Contains(val, "test") {
				findings = append(findings, models.Finding{
					Category:     "security",
					Severity:     "CRITICAL",
					RuleID:       "SMELL-HARDCODED-SECRET",
					Title:        "Kredensial / Secret Tertulis Langsung (Hardcoded Secret)",
					Description:  fmt.Sprintf("Kunci rahasia atau kredensial '%s' terdeteksi langsung di dalam kode sumber pada baris %d.", matches[1], lineNum),
					Suggestion:   "Pindahkan nilai secret ke file .env dan panggil via environment variables (os.Getenv atau process.env).",
					TargetFile:   relPath,
					Line:         lineNum,
					Snippet:      trimmed,
					TheoryBasis:  "The Twelve-Factor App (Factor III: Config - Store config in the environment) & OWASP Secrets Management Cheat Sheet.",
					SuggestedFix: fmt.Sprintf("// Panggil secret dari environment:\nsecret := os.Getenv(\"%s\")", strings.ToUpper(matches[1])),
				})
			}
		}

		// Rule SEC-001: Command Injection / Remote Code Execution (RCE)
		if !isAndroid && lang != "dart" && commandInjectionRegex.MatchString(trimmed) && (strings.Contains(trimmed, "+") || strings.Contains(trimmed, "$") || strings.Contains(trimmed, "fmt.Sprintf")) {
			findings = append(findings, models.Finding{
				Category:     "security",
				Severity:     "CRITICAL",
				RuleID:       "SEC-COMMAND-INJECTION",
				Title:        "Celah Keamanan Kritis: Potensi Command Injection / RCE",
				Description:  fmt.Sprintf("Pemanggilan eksekusi shell / proses sistem terdeteksi dengan penggabungan parameter dinamis pada baris %d. Penyerang dapat menyusupkan perintah arbitrary untuk merusak sistem host.", lineNum),
				Suggestion:   "Gunakan whitelist argumen tetap, pisahkan executable dari daftar argumen (tanpa shell invocation 'sh -c'), atau gunakan library API native.",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "CWE-78: Improper Neutralization of Special Elements used in an OS Command ('OS Command Injection') & OWASP Top 10 A03:2021.",
				SuggestedFix: "// Lewatkan argumen sebagai array slice terpisah tanpa pemanggilan shell:\n// SEBELUM: exec.Command(\"sh\", \"-c\", \"ping \" + host)\n// SESUDAH: exec.Command(\"ping\", \"-c\", \"1\", validatedHost)",
			})
		}

		// Rule SEC-002: Path Traversal / Arbitrary File Access
		if isServer && pathTraversalRegex.MatchString(trimmed) {
			findings = append(findings, models.Finding{
				Category:     "security",
				Severity:     "CRITICAL",
				RuleID:       "SEC-PATH-TRAVERSAL",
				Title:        "Celah Keamanan: Potensi Path Traversal / Arbitrary File Access",
				Description:  fmt.Sprintf("Operasi I/O file membaca parameter request secara langsung pada baris %d tanpa verifikasi batasan folder dasar. Karakter '../' dapat disalahgunakan untuk membaca file sistem rahasia.", lineNum),
				Suggestion:   "Terapkan filepath.Clean(), bersihkan karakter '../', dan pastikan jalur berada di dalam direktori penyimpanan yang aman (safe root directory check).",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "CWE-22: Improper Limitation of a Pathname to a Restricted Directory ('Path Traversal').",
				SuggestedFix: "// Bersihkan path dan validasi batas folder:\ncleanPath := filepath.Clean(filepath.Join(baseDir, requestedPath))\nif !strings.HasPrefix(cleanPath, baseDir) { return errors.New(\"akses ilegal\") }",
			})
		}

		// Rule SEC-003: Dangerous HTML Injection / XSS in Frontend
		if (isUI || ((lang == "typescript" || lang == "javascript") && !isServer)) && unsafeHtmlRegex.MatchString(trimmed) && !strings.Contains(trimmed, "DOMPurify") && !strings.Contains(trimmed, "sanitize") {
			findings = append(findings, models.Finding{
				Category:     "security",
				Severity:     "CRITICAL",
				RuleID:       "SEC-XSS-UNSAFE-HTML",
				Title:        "Celah Keamanan: Dangerous Raw HTML Rendering (XSS Hazard)",
				Description:  fmt.Sprintf("Penyisipan HTML mentah tanpa sanitizer (v-html / innerHTML) pada baris %d berisiko memicu Cross-Site Scripting (XSS) jika memuat input pengguna.", lineNum),
				Suggestion:   "Gunakan text binding biasa ({{ text }}) atau lewatkan konten melalui DOMPurify.sanitize() sebelum dirender ke DOM.",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "OWASP Top 10 A03:2021-Injection (Cross-Site Scripting XSS) & CWE-79.",
				SuggestedFix: "// Bersihkan HTML sebelum rendering:\nimport DOMPurify from 'dompurify'\nconst safeContent = computed(() => DOMPurify.sanitize(rawHtml))",
			})
		}

		// Rule PERF-001: Heavy Query - Leading Wildcard LIKE Full Table Scan
		if usesSQL && wildcardLikeRegex.MatchString(trimmed) {
			findings = append(findings, models.Finding{
				Category:     "query",
				Severity:     "WARNING",
				RuleID:       "PERF-WILDCARD-LIKE",
				Title:        "Kueri Berat: Wildcard Prefix LIKE '%...' (Full Table Scan Hazard)",
				Description:  fmt.Sprintf("Pencarian database menggunakan LIKE '%%...' pada baris %d mematikan indeks B-Tree dan memaksa database memindai seluruh baris tabel secara sekuensial.", lineNum),
				Suggestion:   "Hapus wildcard di awal (gunakan LIKE 'prefix%%') atau beralih ke Full-Text Search (FTS) index untuk pencarian teks bebas skala besar.",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "Relational Database B-Tree Index Mechanics & SQL Query Execution Plan Optimization.",
				SuggestedFix: "// Gunakan indexing berbasis prefix atau FTS:\n// SEBELUM: db.Where(\"name LIKE ?\", \"%\" + query + \"%\")\n// SESUDAH: db.Where(\"name LIKE ?\", query + \"%\") // atau gunakan FTS5",
			})
		}

		// Rule PERF-002: Heavy Process - Blocking Synchronous I/O in Handlers
		if (lang == "javascript" || lang == "typescript") && syncIoRegex.MatchString(trimmed) && isControllerOrHandler {
			findings = append(findings, models.Finding{
				Category:     "code_smell",
				Severity:     "CRITICAL",
				RuleID:       "PERF-SYNC-IO-BLOCKING",
				Title:        "Proses Berat: Operasi File Sinkron Memblokir Event Loop Server",
				Description:  fmt.Sprintf("Pemanggilan fs.readFileSync / writeFileSync pada baris %d di dalam handler HTTP membekukan NodeJS event loop dan menahan semua request koneksi pengguna lain.", lineNum),
				Suggestion:   "Ganti dengan operasi file asinkron (fs.promises.readFile atau fs.readFile non-blocking) dengan async/await.",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "Node.js Event Loop Architecture & Concurrency Model (Reactive Non-blocking I/O).",
				SuggestedFix: "// Gunakan async I/O:\nconst data = await fs.promises.readFile(filePath, 'utf-8')",
			})
		}

		// Rule PERF-003: Heavy Process - Unbounded Memory Buffering (OOM Hazard)
		if lang == "go" && unboundedBodyReadRegex.MatchString(trimmed) && isControllerOrHandler {
			findings = append(findings, models.Finding{
				Category:     "code_smell",
				Severity:     "WARNING",
				RuleID:       "PERF-UNBOUNDED-BODY-READ",
				Title:        "Proses Berat: Pembacaan Request Body Tanpa Batas Ukuran (OOM Risk)",
				Description:  fmt.Sprintf("Pembacaan seluruh request body menggunakan io.ReadAll pada baris %d tanpa pembatas ukuran dapat menguras memori RAM hingga server crash (Out-Of-Memory).", lineNum),
				Suggestion:   "Bungkus body dengan http.MaxBytesReader(w, r.Body, maxBytes) atau gunakan middleware BodyLimit.",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "Resource Allocation Management & Resilience Engineering (Buffer Over-allocation Mitigation).",
				SuggestedFix: "// Batasi ukuran pembacaan stream:\nlimitedReader := io.LimitReader(r.Body, 2*1024*1024) // Maksimal 2MB\ndata, err := io.ReadAll(limitedReader)",
			})
		}

		// Rule DESIGN-001: Smelly Design - Native Browser Alert / Confirm Violation
		if isUI && nativeDialogRegex.MatchString(trimmed) {
			findings = append(findings, models.Finding{
				Category:     "code_smell",
				Severity:     "WARNING",
				RuleID:       "DESIGN-NATIVE-DIALOG",
				Title:        "Smelly Design: Penggunaan Dialog Bawaan Browser (Zero Alert Policy)",
				Description:  fmt.Sprintf("Terdeteksi pemanggilan alert(), confirm(), atau prompt() pada baris %d. Dialog bawaan browser bersifat synchronous blocking dan merusak tampilan profesional aplikasi modern.", lineNum),
				Suggestion:   "Ganti alert() dengan toast notification elegan (useToast()) dan ganti confirm() dengan modal dialog kustom.",
				TargetFile:   relPath,
				Line:         lineNum,
				Snippet:      trimmed,
				TheoryBasis:  "Vibeasik UX Guidelines & Modern Non-blocking Modal Dialog Design Systems.",
				SuggestedFix: "// Gunakan useToast() dari Nuxt UI:\nconst toast = useToast()\ntoast.add({ title: 'Berhasil', description: 'Operasi selesai', color: 'success' })",
			})
		}
	}

	// Best practice Flutter: lifecycle controller, build(), async gap, penyimpanan aman, logging
	if isFlutter {
		findings = append(findings, inspectFlutterBestPractices(content, relPath, lines)...)
	}

	// Lokasi simpan/muat berkas harus tetap setelah aplikasi di-build & diunggah ke server
	if isServer {
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
	if isUI {
		findings = append(findings, inspectFrontendSmells(content, relPath, lines)...)
	}

	// Tampilan & performa UI (FE, Android, Flutter): ukuran berkas, a11y, layout, rendering.
	// Dart non-Flutter tidak punya widget sehingga aturan tampilan dilewati.
	if lang != "dart" || isFlutter {
		findings = append(findings, inspectUIPerformance(content, relPath, lang, lines)...)
	}

	return findings
}

// inspectFrontendSmells audits frontend components for lifecycle API reloads, variable caching, and overfetching
func inspectFrontendSmells(content string, relPath string, lines []string) []models.Finding {
	var findings []models.Finding

	isTabComponent := strings.Contains(relPath, "Tab.vue") ||
		strings.Contains(relPath, "/tabs/") ||
		(strings.Contains(relPath, "detail/") && strings.HasSuffix(relPath, ".vue"))

	// 1. Check for Overfetching / Unused Heavy API Data
	// Example: Calling getProjectBaseKnowledge in a chat/ide tab or a tab that doesn't actually need it
	if strings.Contains(content, "getProjectBaseKnowledge") {
		isBaseKnowledgeTab := strings.Contains(relPath, "BaseKnowledgeTab") || strings.Contains(relPath, "base-knowledge")
		if !isBaseKnowledgeTab && isTabComponent {
			for idx, line := range lines {
				if strings.Contains(line, "getProjectBaseKnowledge") {
					findings = append(findings, models.Finding{
						Category:     "frontend",
						Severity:     "WARNING",
						RuleID:       "FE-OVERFETCHING-UNUSED-DATA",
						Title:        "Overfetching API: Pemanggilan Data Base Knowledge pada Komponen yang Tidak Membutuhkannya",
						Description:  fmt.Sprintf("Komponen '%s' pada baris %d memanggil 'getProjectBaseKnowledge()'. Endpoint ini menyusun pohon direktori, rute, dan env yang berat. Komponen ini sebaiknya tidak memanggil data tersebut atau menggunakan data yang sudah ada di props.", filepath.Base(relPath), idx+1),
						Suggestion:   "Hapus pemanggilan 'getProjectBaseKnowledge()' dari komponen ini. Gunakan computed property dari props.project atau cached data.",
						TargetFile:   relPath,
						Line:         idx + 1,
						Snippet:      strings.TrimSpace(line),
						TheoryBasis:  "REST/GraphQL Resource Optimization & Network Payload Minimization - Mencegah dead data fetching.",
						SuggestedFix: "// Hindari memanggil endpoint agregasi berat:\nconst count = computed(() => props.project?.components?.length || 0)",
					})
					break
				}
			}
		}
	}

	// 2. Check for Uncached Tab Reload on Mount (FE-API-RELOAD-UNCACHED)
	// If it's a tab component with onMounted calling API directly without checking cachedData or data.value
	if isTabComponent && strings.Contains(content, "onMounted") {
		hasApiCallInMount := strings.Contains(content, "loadBaseKnowledge(") ||
			strings.Contains(content, "loadInitialData(") ||
			strings.Contains(content, "fetchData(") ||
			strings.Contains(content, "loadData(") ||
			strings.Contains(content, "getProject(") ||
			strings.Contains(content, "$fetch(")

		hasCacheCheck := strings.Contains(content, "cachedData") ||
			strings.Contains(content, "cachedBaseKnowledge") ||
			strings.Contains(content, "cache") ||
			strings.Contains(content, "if (!data.value)") ||
			strings.Contains(content, "if (!props.") ||
			strings.Contains(content, "v-model:cached")

		if hasApiCallInMount && !hasCacheCheck {
			for idx, line := range lines {
				if strings.Contains(line, "onMounted") {
					findings = append(findings, models.Finding{
						Category:     "frontend",
						Severity:     "WARNING",
						RuleID:       "FE-API-RELOAD-UNCACHED",
						Title:        "Pemanggilan API Berulang Tanpa Caching Variabel (Tab Switch Reload Smell)",
						Description:  fmt.Sprintf("Komponen sub-tab '%s' pada baris %d memanggil pemuatan data API pada hook onMounted() tanpa mekanisme caching variabel/state. Saat pengguna berpindah tab, data akan selalu di-reload ulang dari backend.", filepath.Base(relPath), idx+1),
						Suggestion:   "Simpan data pada state/ref di level halaman induk atau composable, lalu berikan cachedData sebagai prop ke sub-tab agar tidak reload saat tab berganti.",
						TargetFile:   relPath,
						Line:         idx + 1,
						Snippet:      strings.TrimSpace(line),
						TheoryBasis:  "Single Page Application (SPA) State Management & Component Lifecycle Optimization.",
						SuggestedFix: "// Simpan data di ref induk dan berikan sebagai prop:\nconst props = defineProps<{ cachedData?: MyData }>()\nonMounted(() => {\n  if (!props.cachedData) loadData()\n})",
					})
					break
				}
			}
		}
	}

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
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
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
		lang, ok := auditFileExtensions[strings.ToLower(filepath.Ext(path))]
		if !ok || (lang == "xml" && !isAndroidLayoutXML(relPath)) {
			return nil
		}
		if isTestOrGeneratedFile(info.Name()) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		if isGeneratedContent(content) {
			return nil
		}
		isUI := isUISourceFile(relPath, lang)
		isAndroid := (lang == "kotlin" || lang == "java") && isAndroidSourceFile(relPath, content)
		out = append(out, AuditableFile{
			AbsPath:   path,
			RelPath:   relPath,
			Lang:      lang,
			Lines:     strings.Count(content, "\n") + 1,
			IsUI:      isUI,
			IsAndroid: isAndroid,
			IsServer:  isServerLang(lang) && !isUI && !isAndroid,
			IsFlutter: lang == "dart" && isFlutterSource(path, content),
		})
		return nil
	})
	return out
}
