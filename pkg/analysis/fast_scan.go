package analysis

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/util"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/scanner/adapters"
	"github.com/vibeasik/vibeasik/pkg/security"
)

// fastScanSystemPromptTmpl: prompt sistem fast scan; %s diisi skema rute (kosong bila rute sudah diekstrak statis).
const fastScanSystemPromptTmpl = `You are an elite fullstack software architect, security auditor, and devops expert.
Analyze the provided application component directory tree, environment files, database configs, and code snippets.
Return a STRICT JSON response containing the folder structure, run instructions, environment/database/authentication extraction and architecture. Keep every text field short: vibeasik post-processes the result.
All secret keys and passwords in the text have already been safely censored and aliased (e.g. [ALIASED_...]).
Respond with a JSON object in this exact schema:
{
  "folder_structure_summary": "Concise 2-3 paragraph explanation of folder architecture and module responsibilities.",
  "how_to_run": {
    "prerequisites": ["Runtime version or CLI required, e.g. Go 1.22+, Docker"],
    "install_command": "Command to install dependencies",
    "dev_command": "Command to run in development mode",
    "build_command": "Command to build for production",
    "test_command": "Command to run tests or linter",
    "steps": ["Step 1: ...", "Step 2: ..."]
  },
  "extracted_env": {
    "database_type": "Database engine used (e.g. PostgreSQL, MySQL, SQLite, MongoDB, Redis, None, etc.)",
    "orm_driver": "ORM or DB driver used (e.g. GORM, Prisma, Drizzle, TypeORM, SQLx, Fiber/sql, etc.)",
    "database_name": "Name of database (e.g. shop_db, myapp_db)",
    "auth_methods": ["Authentication methods used, e.g. JWT Bearer Token, OAuth2 Google, Cookie Session, Firebase Auth, API Key, Username/Password"],
    "login_endpoints": ["Detected login/auth endpoints, e.g. /api/auth/login, /api/login, /auth/google"],
    "third_party_services": [
      {"name": "e.g. Redis", "purpose": "Cache & Session Storage"},
      {"name": "e.g. Firebase", "purpose": "Push Notification & Admin Auth"}
    ],
    "environment_keys": [
      {
        "key": "ENV_VAR_NAME",
        "purpose": "What this variable is used for",
        "required": true,
        "is_secret": false,
        "safe_example": "Safe value or [ALIASED_...]"
      }
    ],
    "login_and_access_guide": "Clear guide on how to log in, what credentials or tokens are needed, and how user/admin access is authorized.",
    "dynamic_metadata": {
      "Key Name": "Dynamic value extracted (e.g. App Port, API Prefix, CORS Origin, Cache Driver, Docker Service, Token Expiry)"
    }
  },
%s  "architecture": {
    "framework": "e.g. Fiber v2, Gin, Echo, Chi, net/http, Nuxt 3, Flutter",
    "framework_type": "e.g. FastHTTP Asynchronous Engine, Minimalist Go Router",
    "plugins": ["Detected plugins, middlewares, or key libraries, e.g. JWT Auth, Firebase Admin SDK, GORM MySQL, Redis, bcrypt"],
    "design_pattern": "e.g. Headless MVC with E2EE Vault, Clean Architecture, 3-Tier Layered Architecture",
    "pattern_details": "Short explanation in Indonesian (max 3 sentences) of the architectural role of the main folders"
  },
  "detected_params": {
    "language": "e.g. go, typescript, dart, python",
    "framework": "e.g. fiber, nuxt, next, flutter, fastapi",
    "package_manager": "e.g. go, bun, npm, pnpm, pub, pip",
    "dev_command": "dev run command",
    "build_command": "build command",
    "port": 0
  }
}`

// fastScanTargetDir menentukan direktori target komponen (mungkin relatif/kosong).
func fastScanTargetDir(comp *models.Component, proj *models.Project) string {
	targetDir := comp.AbsolutePath
	if targetDir == "" {
		if proj != nil && proj.RootPath != "" && comp.RelativePath != "" {
			targetDir = filepath.Join(proj.RootPath, comp.RelativePath)
		} else if comp.RelativePath != "" {
			targetDir = comp.RelativePath
		}
	}
	return targetDir
}

// mergeDetectedArchitecture melengkapi arsitektur hasil LLM dengan deteksi statis.
func mergeDetectedArchitecture(arch *models.ArchitectureInfo, detected models.ArchitectureInfo) {
	if arch.Framework == "" {
		arch.Framework = detected.Framework
	}
	if arch.FrameworkType == "" {
		arch.FrameworkType = detected.FrameworkType
	}
	if arch.DesignPattern == "" {
		arch.DesignPattern = detected.DesignPattern
	}
	if arch.PatternDetails == "" {
		arch.PatternDetails = detected.PatternDetails
	}
	if len(arch.Plugins) == 0 {
		arch.Plugins = detected.Plugins
		return
	}
	// Merge any plugins detected statically that LLM missed
	for _, p := range detected.Plugins {
		arch.Plugins = addUnique(arch.Plugins, p)
	}
}

// mergeHeuristicEnv melengkapi hasil ekstraksi database/auth dengan heuristik statis.
func mergeHeuristicEnv(env *models.ExtractedEnvInfo, h models.ExtractedEnvInfo) {
	if env.DatabaseType == "" || env.DatabaseType == envDBNone {
		env.DatabaseType = h.DatabaseType
	}
	if env.ORMDriver == "" || env.ORMDriver == envORMNone {
		env.ORMDriver = h.ORMDriver
	}
	if env.DatabaseName == "" {
		env.DatabaseName = h.DatabaseName
	}
	if len(env.AuthMethods) == 0 && len(h.AuthMethods) > 0 {
		env.AuthMethods = h.AuthMethods
	}
	if len(env.LoginEndpoints) == 0 && len(h.LoginEndpoints) > 0 {
		env.LoginEndpoints = h.LoginEndpoints
	}
}

// applyParsedFastScan menyalin hasil parse JSON LLM ke result.
func applyParsedFastScan(result *models.FastScanResult, parsed *fastScanJSONPayload) {
	result.FolderStructureSummary = parsed.FolderStructureSummary
	result.HowToRun = parsed.HowToRun
	result.ExtractedEnv = parsed.ExtractedEnv
	result.DetectedParams = parsed.DetectedParams
	result.Routes = parsed.Routes
	result.Architecture = parsed.Architecture
	result.AiAnalysisMarkdown = parsed.AiAnalysisMarkdown

	// Ensure DynamicMetadata is non-nil
	if result.ExtractedEnv.DynamicMetadata == nil {
		result.ExtractedEnv.DynamicMetadata = make(map[string]string)
	}
}

// FastScanComponent performs an AI-powered quick scan on a project component
func FastScanComponent(db *gorm.DB, comp *models.Component, proj *models.Project) (*models.FastScanResult, error) {
	absDir, err := filepath.Abs(fastScanTargetDir(comp, proj))
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("directory does not exist on disk: %s", absDir)
	}

	// 1. Generate Clean Directory Tree (max depth 3); LLM cukup melihat 2 tingkat
	dirTree := generateCleanTree(absDir, 3)

	// 2. Rute diekstrak statis lebih dulu: bila adapter menemukannya, LLM tidak perlu membaca kode controller
	// maupun menulis ulang daftar rute (deskripsinya diisi EnrichComponentRoutes secara bertahap).
	staticRoutes := extractRoutes(absDir, comp.Role)
	needRoutes := len(staticRoutes) == 0

	// 3. Extract key manifest, config, database, auth, and env files
	fileSnippets := extractKeyFiles(absDir, needRoutes)

	// 4. Assemble scan prompt
	rawPrompt := fmt.Sprintf(`Component Name: %s
Role: %s (BE, FE, Mobile, etc.)
Path: %s
Registered Language: %s
Registered Framework: %s

DIRECTORY TREE:
%s

KEY FILES CONTENT (Config, Env, Database, Auth & Code):
%s
`, comp.Name, comp.Role, absDir, comp.Language, comp.Framework, generateCleanTree(absDir, 2), fileSnippets)

	// 5. Sanitize and censor all keys globally with deterministic safe aliases
	sanitizedContext, audit := security.GlobalSanitizer.Sanitize(rawPrompt)

	routesSchema, routesAsk := "", fmt.Sprintf("Routes (%d) were already extracted statically: do NOT list them.", len(staticRoutes))
	if needRoutes {
		routesSchema = fastScanRoutesSchema
		routesAsk = "List all routers, their HTTP methods, exact endpoint paths, handler functions, and short descriptions."
	}
	systemPrompt := fmt.Sprintf(fastScanSystemPromptTmpl, routesSchema)

	userPrompt := fmt.Sprintf(`Perform Fast Scan, Database Extraction, and API Route Inspection on this component:
%s

Ensure your response is valid JSON matching the requested structure. %s`, sanitizedContext, routesAsk)

	// 6. Query active LLM (Prefer Standard LLM for structured fast scan analysis)
	aiRawResponse := ""
	if db != nil {
		aiRawResponse, _ = queryActiveLLM(db, systemPrompt, userPrompt)
	}

	// 7. Parse JSON or generate robust fallback from heuristic
	result := &models.FastScanResult{
		ComponentID:   comp.ID,
		ComponentName: comp.Name,
		Role:          comp.Role,
		RelativePath:  comp.RelativePath,
		AbsolutePath:  absDir,
		DirectoryTree: dirTree,
		ScannedAt:     time.Now(),
		Routes:        make([]models.RouteInfo, 0),
		SecurityAudit: map[string]interface{}{
			"redacted_count": audit.RedactedCount,
			"records":        audit.Records,
			"message":        audit.Message,
			"safe_preview":   audit.SafePreview,
		},
	}

	if parsed := parseFastScanJSON(aiRawResponse); parsed != nil {
		applyParsedFastScan(result, parsed)
	} else {
		// Fallback heuristic if LLM returned free text or was offline
		fallbackHeuristic(result, absDir, comp, aiRawResponse)
	}

	// Guarantee architecture detection completeness (Framework, Plugins, Design Pattern)
	mergeDetectedArchitecture(&result.Architecture, detectArchitecture(absDir))

	// Guarantee complete route extraction & merging via adapter architecture
	if len(result.Routes) == 0 {
		result.Routes = staticRoutes
	} else if len(staticRoutes) > len(result.Routes) {
		result.Routes = mergeRoutes(result.Routes, staticRoutes)
	}

	// Guarantee database extraction completeness
	mergeHeuristicEnv(&result.ExtractedEnv, extractEnvHeuristic(absDir))

	// Generate clean rich Markdown report if raw response is empty, heuristic, or System One raw JSON
	if strings.TrimSpace(result.AiAnalysisMarkdown) == "" || strings.HasPrefix(result.AiAnalysisMarkdown, "System One Evaluation") {
		result.AiAnalysisMarkdown = generateDetailedMarkdownReport(result)
	}

	// 8. Save fast scan result in component database
	if db != nil && comp.ID > 0 {
		scanBytes, _ := json.Marshal(result)
		comp.FastScanData = string(scanBytes)
		db.Model(comp).Update("fast_scan_data", comp.FastScanData)
		_, _ = SyncComponentRoutes(db, comp, proj)
	}

	return result, nil
}

// fastScanRoutesSchema: bagian skema rute, hanya diminta bila ekstraksi statis tidak menemukan rute.
const fastScanRoutesSchema = `  "routes": [
    {
      "method": "GET/POST/PUT/DELETE/PATCH",
      "path": "Full endpoint path e.g. /api/auth/register, /api/assets/:id",
      "handler": "Function or controller accessed, e.g. controllers.Register, controllers.GetAsset",
      "description": "Clear explanation in Indonesian of what this router does and its responsibility",
      "middleware": "Middleware applied, e.g. Public, Protected (JWT Bearer Token)"
    }
  ],
`

type fastScanJSONPayload struct {
	FolderStructureSummary string                  `json:"folder_structure_summary"`
	HowToRun               models.HowToRunInfo     `json:"how_to_run"`
	ExtractedEnv           models.ExtractedEnvInfo `json:"extracted_env"`
	DetectedParams         models.DetectedParams   `json:"detected_params"`
	Routes                 []models.RouteInfo      `json:"routes"`
	Architecture           models.ArchitectureInfo `json:"architecture"`
	AiAnalysisMarkdown     string                  `json:"ai_analysis_markdown"`
}

func parseFastScanJSON(raw string) *fastScanJSONPayload {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var res fastScanJSONPayload
	if err := json.Unmarshal([]byte(raw), &res); err == nil && res.FolderStructureSummary != "" {
		return &res
	}

	// Extract ```json ... ``` code fence
	reJSON := regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")
	matches := reJSON.FindStringSubmatch(raw)
	if len(matches) > 1 {
		if err := json.Unmarshal([]byte(matches[1]), &res); err == nil {
			if res.AiAnalysisMarkdown == "" {
				res.AiAnalysisMarkdown = raw
			}
			return &res
		}
	}

	// Look for first '{' to last '}'
	firstBrace := strings.Index(raw, "{")
	lastBrace := strings.LastIndex(raw, "}")
	if firstBrace >= 0 && lastBrace > firstBrace {
		if err := json.Unmarshal([]byte(raw[firstBrace:lastBrace+1]), &res); err == nil {
			if res.AiAnalysisMarkdown == "" {
				res.AiAnalysisMarkdown = raw
			}
			return &res
		}
	}

	return nil
}

// fallbackPlan: hasil deteksi stack berbasis berkas untuk fallbackHeuristic.
type fallbackPlan struct {
	lang       string
	pkgMgr     string
	devCmd     string
	buildCmd   string
	port       int
	prereqs    []string
	installCmd string
	testCmd    string
	steps      []string
}

func (p *fallbackPlan) defaultDev(cmd string) {
	if p.devCmd == "" {
		p.devCmd = cmd
	}
}

func (p *fallbackPlan) defaultBuild(cmd string) {
	if p.buildCmd == "" {
		p.buildCmd = cmd
	}
}

func (p *fallbackPlan) defaultPort(port int) {
	if p.port == 0 {
		p.port = port
	}
}

func composerMentions(dir, word string) bool {
	composer := filepath.Join(dir, "composer.json")
	return util.PathExists(composer) && strings.Contains(util.ReadFileString(composer), word)
}

func planGo(dir string, comp *models.Component, p *fallbackPlan) {
	p.lang = "go"
	p.pkgMgr = "go"
	p.defaultDev("go run main.go")
	p.defaultBuild("go build -o server main.go")
	p.testCmd = "go test ./..."
	p.installCmd = "go mod download"
	p.prereqs = []string{"Go 1.21+"}
	p.defaultPort(8080)
	p.steps = []string{
		"1. Pastikan Go terinstall (`go version`).",
		"2. Jalankan `go mod download` untuk mengunduh dependency.",
		"3. Jalankan `go run main.go` untuk mode development.",
		"4. Akses server di port :" + strconv.Itoa(p.port) + ".",
	}
}

func planFlutter(dir string, comp *models.Component, p *fallbackPlan) {
	p.lang = "dart"
	comp.Framework = "flutter"
	p.pkgMgr = "pub"
	p.defaultDev("flutter run")
	p.defaultBuild("flutter build apk")
	p.testCmd = "flutter test"
	p.installCmd = "flutter pub get"
	p.prereqs = []string{"Flutter SDK 3.x", "Android Studio / VS Code"}
	p.steps = []string{
		"1. Pastikan Flutter SDK terpasang (`flutter doctor`).",
		"2. Jalankan `flutter pub get` untuk mengunduh paket.",
		"3. Buka emulator atau sambungkan perangkat fisik.",
		"4. Jalankan `flutter run` untuk memulai aplikasi mobile.",
	}
}

// nodePackageManager memilih package manager dari berkas lock (bawaan bun).
func nodePackageManager(dir string) string {
	for _, l := range []struct{ file, mgr string }{
		{"pnpm-lock.yaml", "pnpm"}, {"yarn.lock", "yarn"}, {"package-lock.json", "npm"},
	} {
		if util.PathExists(filepath.Join(dir, l.file)) {
			return l.mgr
		}
	}
	return "bun"
}

func planNode(dir string, comp *models.Component, p *fallbackPlan) {
	p.pkgMgr = nodePackageManager(dir)
	if pathExistsAny(dir, "nuxt.config.ts", "nuxt.config.js") {
		comp.Framework = "nuxt"
		p.defaultPort(3000)
	}
	p.lang = "typescript"
	p.installCmd = fmt.Sprintf("%s install", p.pkgMgr)
	p.defaultDev(fmt.Sprintf("%s run dev", p.pkgMgr))
	p.defaultBuild(fmt.Sprintf("%s run build", p.pkgMgr))
	p.testCmd = fmt.Sprintf("%s test", p.pkgMgr)
	p.prereqs = []string{fmt.Sprintf("Node.js 20+ atau %s", p.pkgMgr)}
	p.steps = []string{
		fmt.Sprintf("1. Jalankan `%s install` untuk menginstal modul.", p.pkgMgr),
		fmt.Sprintf("2. Jalankan `%s` untuk menjalankan dev server.", p.devCmd),
		fmt.Sprintf("3. Buka browser di http://localhost:%d", p.port),
	}
}

func planCodeIgniter(dir string, comp *models.Component, p *fallbackPlan) {
	p.lang = "php"
	comp.Framework = "codeigniter"
	p.pkgMgr = "composer"
	p.defaultDev("php spark serve")
	p.testCmd = "php spark test"
	p.installCmd = "composer install"
	p.prereqs = []string{"PHP 8.2+", "Composer"}
	p.defaultPort(8080)
	p.steps = []string{
		"1. Pastikan PHP dan Composer terpasang (`php -v`, `composer -V`).",
		"2. Jalankan `composer install` untuk mengunduh dependency.",
		"3. Salin environment file: `cp env .env` dan sesuaikan konfigurasi database.",
		"4. Jalankan `php spark serve` untuk memulai server development lokal.",
		"5. Akses server di port :" + strconv.Itoa(p.port) + ".",
	}
}

func planLaravel(dir string, comp *models.Component, p *fallbackPlan) {
	p.lang = "php"
	comp.Framework = "laravel"
	p.pkgMgr = "composer"
	p.defaultDev("php artisan serve")
	p.testCmd = "php artisan test"
	p.installCmd = "composer install"
	p.prereqs = []string{"PHP 8.2+", "Composer"}
	p.defaultPort(8000)
	p.steps = []string{
		"1. Pastikan PHP dan Composer terpasang (`php -v`, `composer -V`).",
		"2. Jalankan `composer install` untuk mengunduh dependency.",
		"3. Jalankan `php artisan serve` untuk memulai server development lokal.",
		"4. Akses server di port :" + strconv.Itoa(p.port) + ".",
	}
}

// fallbackStacks: stack yang dikenali dari berkas di disk (urut, pertama cocok).
var fallbackStacks = []struct {
	match func(dir string) bool
	plan  func(dir string, comp *models.Component, p *fallbackPlan)
}{
	{func(d string) bool { return pathExistsAll(d, "go.mod") }, planGo},
	{func(d string) bool { return pathExistsAll(d, "pubspec.yaml") }, planFlutter},
	{func(d string) bool { return pathExistsAll(d, "package.json") }, planNode},
	{func(d string) bool {
		return pathExistsAny(d, "spark", "app/Config/Routes.php") || composerMentions(d, "codeigniter")
	}, planCodeIgniter},
	{func(d string) bool { return pathExistsAny(d, "artisan") || composerMentions(d, "laravel") }, planLaravel},
}

func fallbackHeuristic(result *models.FastScanResult, absDir string, comp *models.Component, rawAi string) {
	p := &fallbackPlan{
		lang:     comp.Language,
		pkgMgr:   comp.PackageManager,
		devCmd:   comp.DevCommand,
		buildCmd: comp.BuildCommand,
		port:     comp.Port,
	}

	// Detect by files on disk
	for _, st := range fallbackStacks {
		if st.match(absDir) {
			st.plan(absDir, comp, p)
			break
		}
	}

	result.FolderStructureSummary = fmt.Sprintf("Komponen %s berada di `%s`. Terdeteksi arsitektur berbasis %s dengan package manager %s.", comp.Name, absDir, p.lang, p.pkgMgr)
	result.HowToRun = models.HowToRunInfo{
		Prerequisites:  p.prereqs,
		InstallCommand: p.installCmd,
		DevCommand:     p.devCmd,
		BuildCommand:   p.buildCmd,
		TestCommand:    p.testCmd,
		Steps:          p.steps,
	}
	result.DetectedParams = models.DetectedParams{
		Language:       p.lang,
		Framework:      comp.Framework,
		PackageManager: p.pkgMgr,
		DevCommand:     p.devCmd,
		BuildCommand:   p.buildCmd,
		Port:           p.port,
	}

	// Extract environment, database, auth, and dynamic info heuristically
	result.ExtractedEnv = extractEnvHeuristic(absDir)
	result.Routes = extractRoutes(absDir, comp.Role)

	if strings.TrimSpace(rawAi) != "" && !strings.HasPrefix(rawAi, "System One Evaluation") {
		result.AiAnalysisMarkdown = rawAi
	} else {
		result.AiAnalysisMarkdown = generateDetailedMarkdownReport(result)
	}
}

const (
	envDBNone   = "None detected"
	envORMNone  = "Native / Standard"
	gormORMName = "GORM (Go Object Relational Mapping)"
)

// envScan: keadaan sementara selama ekstraksi environment/database satu direktori.
type envScan struct {
	info   *models.ExtractedEnvInfo
	dbType string
	orm    string
	dbName string
	dir    string
}

// envKeyRule mengklasifikasi nama variabel environment: cocok bila nama (huruf besar) sama dengan exact
// atau memuat salah satu keys; apply (opsional) memperbarui keadaan ekstraksi.
type envKeyRule struct {
	exact   string
	keys    []string
	purpose string
	apply   func(st *envScan, v string)
}

func (r envKeyRule) match(u string) bool {
	return (r.exact != "" && u == r.exact) || containsAny(u, r.keys...)
}

func envMeta(name string) func(st *envScan, v string) {
	return func(st *envScan, v string) { st.info.DynamicMetadata[name] = v }
}

func envService(name, purpose string) func(st *envScan, v string) {
	return func(st *envScan, v string) {
		st.info.ThirdPartyServices = append(st.info.ThirdPartyServices, models.DetectedService{Name: name, Purpose: purpose})
	}
}

var envKeyRules = []envKeyRule{
	{exact: "DB_DATABASE", keys: []string{"DATABASE.DEFAULT.DATABASE", "DB_NAME", "DATABASE_NAME"}, purpose: "Nama skema / database utama",
		apply: func(st *envScan, v string) {
			st.dbName = v
			st.info.DynamicMetadata["Database Name"] = v
		}},
	{exact: "DB_CONNECTION", keys: []string{"DATABASE.DEFAULT.DBDRIVER", "DB_DRIVER"}, purpose: "Driver koneksi database",
		apply: func(st *envScan, v string) {
			st.dbType = strings.ToUpper(v)
			if strings.EqualFold(v, "mysqli") || strings.EqualFold(v, "mysql") {
				st.dbType = "MySQL"
				st.orm = "CodeIgniter Model / MySQLi"
			} else if strings.EqualFold(v, "postgre") || strings.EqualFold(v, "postgres") {
				st.dbType = "PostgreSQL"
				st.orm = "CodeIgniter Model / Postgre"
			}
		}},
	{keys: []string{"DATABASE.DEFAULT.HOSTNAME", "DB_HOST", "DATABASE_HOST"}, purpose: "Host alamat server database", apply: envMeta("Database Host")},
	{keys: []string{"DATABASE.DEFAULT.PORT", "DB_PORT"}, purpose: "Port jaringan listening service database", apply: envMeta("Database Port")},
	{keys: []string{"APP.BASEURL", "APP_URL"}, purpose: "Base URL aplikasi", apply: envMeta("Base URL")},
	{keys: []string{"CI_ENVIRONMENT", "APP_ENV"}, purpose: "Environment runtime aplikasi (development/production)", apply: envMeta("Environment")},
	{keys: []string{"PORT"}, purpose: "Port jaringan listening service", apply: envMeta("Port")},
	{keys: []string{"JWT"}, purpose: "Kunci rahasia enkripsi JWT token autentikasi",
		apply: func(st *envScan, v string) {
			if !containsString(st.info.AuthMethods, "JWT Bearer Token") {
				st.info.AuthMethods = append(st.info.AuthMethods, "JWT Bearer Token")
			}
		}},
	{keys: []string{"REDIS"}, purpose: "Koneksi cache dan queue in-memory Redis", apply: envService("Redis", "In-Memory Caching & Session")},
	{keys: []string{"OPENAI", "AI_"}, purpose: "Kredensial API AI / LLM", apply: envService("OpenAI / AI Engine", "Integrasi Kecerdasan Buatan")},
	{keys: []string{"FIREBASE"}, purpose: "Kredensial layanan Firebase Admin SDK", apply: envService("Firebase", "Push Notification & Admin Auth")},
}

// findEnvFile mengembalikan path berkas .env pertama yang ada di dir ("" bila tak ada).
func findEnvFile(dir string) string {
	for _, ef := range []string{".env", ".env.example", ".env.server", ".env.local", ".env.development"} {
		p := filepath.Join(dir, ef)
		if util.PathExists(p) {
			return p
		}
	}
	return ""
}

// scanEnvFile membaca berkas env baris demi baris dan mengisi EnvironmentKeys serta keadaan ekstraksi.
func (st *envScan) scanEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			st.addEnvKey(strings.TrimSpace(parts[0]), strings.Trim(strings.TrimSpace(parts[1]), `"'`))
		}
	}
}

func (st *envScan) addEnvKey(key, rawVal string) {
	cleanVal := security.SanitizeText(rawVal)
	upperKey := strings.ToUpper(key)
	isSecret := containsAny(upperKey, "KEY", "SECRET", "PASS", "TOKEN")

	purpose := "Konfigurasi runtime lingkungan"
	for _, r := range envKeyRules {
		if r.match(upperKey) {
			purpose = r.purpose
			if r.apply != nil {
				r.apply(st, cleanVal)
			}
			break
		}
	}

	st.info.EnvironmentKeys = append(st.info.EnvironmentKeys, models.EnvKeyDescription{
		Key:         key,
		Purpose:     purpose,
		Required:    true,
		IsSecret:    isSecret,
		SafeExample: cleanVal,
	})
}

var dbSourceFiles = []string{
	"database/database.go",
	"database/db.go",
	"db/database.go",
	"db/db.go",
	"database.go",
	"db.go",
	"internal/database/db.go",
	"internal/database/database.go",
}

var dbSourceNameRe = regexp.MustCompile(`dbName\s*:=\s*os\.Getenv\("DB_NAME"\)\s*\n\s*if\s*dbName\s*==\s*""\s*\{\s*dbName\s*=\s*"([^"]+)"`)

// dbKeyword memetakan kata kunci isi sumber ke nama mesin database (urut, pertama cocok).
type dbKeyword struct {
	keys   []string
	dbType string
}

var dbSourceKeywords = []dbKeyword{
	{[]string{"mysql", "driver/mysql"}, "MySQL"},
	{[]string{"postgres", "lib/pq", "pgx"}, "PostgreSQL"},
	{[]string{"sqlite"}, "SQLite"},
}

// scanDBSource mendeteksi database dari berkas sumber database pertama yang ada.
func (st *envScan) scanDBSource() {
	for _, dbf := range dbSourceFiles {
		p := filepath.Join(st.dir, dbf)
		if !util.PathExists(p) {
			continue
		}
		if content, err := os.ReadFile(p); err == nil {
			st.applyDBSource(string(content))
		}
		return
	}
}

func (st *envScan) applyDBSource(content string) {
	for _, k := range dbSourceKeywords {
		if containsAny(content, k.keys...) {
			st.dbType = k.dbType
			if strings.Contains(content, "gorm") {
				st.orm = gormORMName
			}
			break
		}
	}
	if st.dbName == "" {
		if m := dbSourceNameRe.FindStringSubmatch(content); len(m) > 1 {
			st.dbName = m[1]
		}
	}
}

var goModDBKeywords = []dbKeyword{
	{[]string{"gorm.io/driver/mysql", "go-sql-driver/mysql"}, "MySQL"},
	{[]string{"gorm.io/driver/postgres", "lib/pq", "pgx"}, "PostgreSQL"},
	{[]string{"sqlite"}, "SQLite"},
}

var prismaProviders = []dbKeyword{
	{[]string{`provider = "postgresql"`}, "PostgreSQL"},
	{[]string{`provider = "mysql"`}, "MySQL"},
	{[]string{`provider = "sqlite"`}, "SQLite"},
}

// scanManifestDB mendeteksi database/ORM dari go.mod atau skema Prisma.
func (st *envScan) scanManifestDB() {
	if util.PathExists(filepath.Join(st.dir, "go.mod")) {
		modContent, _ := os.ReadFile(filepath.Join(st.dir, "go.mod"))
		sMod := string(modContent)
		if strings.Contains(sMod, "gorm.io/gorm") {
			st.orm = gormORMName
		}
		for _, k := range goModDBKeywords {
			if containsAny(sMod, k.keys...) && st.dbType == envDBNone {
				st.dbType = k.dbType
			}
		}
	} else if util.PathExists(filepath.Join(st.dir, "prisma/schema.prisma")) {
		st.orm = "Prisma ORM"
		prismaContent, _ := os.ReadFile(filepath.Join(st.dir, "prisma/schema.prisma"))
		for _, k := range prismaProviders {
			if containsAny(string(prismaContent), k.keys...) {
				st.dbType = k.dbType
				break
			}
		}
	}
}

// detectAuth mengisi metode autentikasi bawaan bila env belum menyebutkannya.
func (st *envScan) detectAuth() {
	if len(st.info.AuthMethods) != 0 {
		return
	}
	if util.PathExists(filepath.Join(st.dir, "server/api/auth.ts")) || util.PathExists(filepath.Join(st.dir, "server/api/auth/[...].ts")) {
		st.info.AuthMethods = append(st.info.AuthMethods, "Nuxt Auth / OAuth")
		st.info.LoginEndpoints = append(st.info.LoginEndpoints, "/api/auth/login")
	} else {
		st.info.AuthMethods = append(st.info.AuthMethods, "JWT Bearer Token")
	}
}

// isFrontendDir: direktori berisi penanda proyek frontend (Node/Nuxt/Next/Flutter).
func isFrontendDir(dir string) bool {
	for _, f := range []string{"package.json", "nuxt.config.ts", "nuxt.config.js", "next.config.js", "pubspec.yaml"} {
		if util.PathExists(filepath.Join(dir, f)) {
			return true
		}
	}
	return false
}

// applyFrontendDefaults mengisi nilai bawaan komponen frontend yang tidak punya database.
func (st *envScan) applyFrontendDefaults() {
	if st.dbType != envDBNone && st.dbType != "" {
		return
	}
	st.dbType = "Client Cache & Remote REST API"
	st.orm = "Typed HTTP Composables ($fetch / useApi)"
	st.dbName = "Client Reactive State"
	st.info.AuthMethods = []string{"Client Bearer Token Storage", "Route Navigation Guard Middleware", "Reactive Session State"}
	st.info.LoginEndpoints = []string{"/login (Form UI)", "/auth/login"}
	st.info.LoginAndAccessGuide = "Aplikasi frontend mengelola token otentikasi di client storage/cookies dan menerapkan Route Navigation Guards sebelum mengakses halaman terlindungi."
}

// frontendCap: kemampuan frontend yang dideteksi dari dependency package.json; meta (opsional) = entri DynamicMetadata.
type frontendCap struct {
	deps     []string
	label    string
	metaKey  string
	metaText string
}

var frontendCaps = []frontendCap{
	{[]string{"pinia", "@pinia/nuxt"}, "State Management (Pinia)", "State Management", "Pinia Reactive Store"},
	{[]string{"tailwindcss", "@nuxtjs/tailwindcss"}, "Responsive Layout (Tailwind CSS)", "Styling Engine", "Tailwind CSS Utility-First"},
	{[]string{"@nuxt/ui"}, "Nuxt UI Design System", "UI Framework", "Nuxt UI Component Suite"},
	{[]string{"zod"}, "Schema Validation (Zod)", "", ""},
	{[]string{"chart.js", "vue-chartjs"}, "Interactive Charts & Data Visualization", "", ""},
	{[]string{"@vueuse/core"}, "Vue Composition Utilities (@vueuse)", "", ""},
}

func (p packageJSON) hasDep(name string) bool {
	if _, ok := p.Dependencies[name]; ok {
		return true
	}
	_, ok := p.DevDependencies[name]
	return ok
}

func (p packageJSON) hasAnyDep(names ...string) bool {
	for _, n := range names {
		if p.hasDep(n) {
			return true
		}
	}
	return false
}

// scanFrontendCapabilities mengisi DynamicMetadata kemampuan frontend dari package.json.
func (st *envScan) scanFrontendCapabilities() {
	pkgPath := filepath.Join(st.dir, "package.json")
	if !util.PathExists(pkgPath) {
		return
	}
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return
	}
	var pkg packageJSON
	if json.Unmarshal(data, &pkg) != nil {
		return
	}
	caps := []string{}
	for _, c := range frontendCaps {
		if !pkg.hasAnyDep(c.deps...) {
			continue
		}
		caps = append(caps, c.label)
		if c.metaKey != "" {
			st.info.DynamicMetadata[c.metaKey] = c.metaText
		}
	}
	caps = append(caps, "Route Navigation Guards")
	caps = append(caps, "Double Security Censored Keys Vault")

	st.info.DynamicMetadata["Frontend Capabilities"] = strings.Join(caps, ", ")
	if pkg.PackageManager != "" {
		st.info.DynamicMetadata["Package Manager"] = pkg.PackageManager
	}
}

func extractEnvHeuristic(dir string) models.ExtractedEnvInfo {
	envInfo := models.ExtractedEnvInfo{
		AuthMethods:        make([]string, 0),
		LoginEndpoints:     make([]string, 0),
		ThirdPartyServices: make([]models.DetectedService, 0),
		EnvironmentKeys:    make([]models.EnvKeyDescription, 0),
		DynamicMetadata:    make(map[string]string),
	}
	st := &envScan{info: &envInfo, dbType: envDBNone, orm: envORMNone, dir: dir}

	if p := findEnvFile(dir); p != "" {
		st.scanEnvFile(p)
	}
	st.scanDBSource()
	st.scanManifestDB()
	st.detectAuth()

	isFrontend := isFrontendDir(dir)
	if isFrontend {
		st.applyFrontendDefaults()
		st.scanFrontendCapabilities()
	}

	if len(envInfo.LoginEndpoints) == 0 {
		envInfo.LoginEndpoints = []string{"/api/auth/login", "/api/auth/register", "/api/auth/google"}
	}

	envInfo.DatabaseType = st.dbType
	envInfo.ORMDriver = st.orm
	envInfo.DatabaseName = st.dbName
	if envInfo.LoginAndAccessGuide == "" {
		envInfo.LoginAndAccessGuide = fmt.Sprintf("Aplikasi menggunakan autentikasi %s. Endpoint login dan registrasi publik berada di %s.", strings.Join(envInfo.AuthMethods, ", "), strings.Join(envInfo.LoginEndpoints, ", "))
	}
	envInfo.DynamicMetadata["Arsitektur Database"] = fmt.Sprintf("%s (%s)", st.dbType, st.orm)
	if st.dbName != "" {
		envInfo.DynamicMetadata["Database Name"] = st.dbName
	}

	return envInfo
}

func containsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func generateCleanTree(rootPath string, maxDepth int) string {
	var sb strings.Builder
	baseName := filepath.Base(rootPath)
	sb.WriteString(baseName + "/\n")

	walkTree(rootPath, "", 1, maxDepth, &sb)
	return sb.String()
}

func walkTree(currentPath, prefix string, currentDepth, maxDepth int, sb *strings.Builder) {
	if currentDepth > maxDepth {
		return
	}

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return
	}

	var validEntries []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if shouldIgnoreName(name) {
			continue
		}
		validEntries = append(validEntries, e)
	}

	for i, e := range validEntries {
		isLast := i == len(validEntries)-1
		connector := "├── "
		nextPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			nextPrefix = prefix + "    "
		}

		if e.IsDir() {
			sb.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, connector, e.Name()))
			walkTree(filepath.Join(currentPath, e.Name()), nextPrefix, currentDepth+1, maxDepth, sb)
		} else {
			sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, e.Name()))
		}
	}
}

func shouldIgnoreName(name string) bool {
	if strings.HasPrefix(name, ".") && name != ".env.example" && name != ".env" && name != ".env.server" {
		return true
	}
	ignored := map[string]bool{
		"node_modules": true,
		"vendor":       true,
		"dist":         true,
		"build":        true,
		".nuxt":        true,
		".next":        true,
		".output":      true,
		".dart_tool":   true,
		"coverage":     true,
		"tmp":          true,
		"__pycache__":  true,
		".turbo":       true,
		".git":         true,
		"Pods":         true,
	}
	return ignored[name]
}

// keyFileTargets: berkas manifest/konfigurasi yang dicuplik (urut tampil).
var keyFileTargets = []string{
	".env.example",
	".env",
	".env.server",
	".env.local",
	".env.development",
	"docker-compose.yml",
	"docker-compose.yaml",
	"Dockerfile",
	"package.json",
	"go.mod",
	"pubspec.yaml",
	"Makefile",
	"database/database.go",
	"database/db.go",
	"db/database.go",
	"db/db.go",
	"database.go",
	"db.go",
	"internal/database/db.go",
	"internal/database/database.go",
	"internal/config/config.go",
	"config/config.go",
	"utils/config.go",
	"prisma/schema.prisma",
	"drizzle.config.ts",
	"nuxt.config.ts",
	"nuxt.config.js",
	"next.config.js",
	"next.config.mjs",
	"next.config.ts",
	"vite.config.ts",
	"vite.config.js",
	"tailwind.config.ts",
	"tailwind.config.js",
	"tsconfig.json",
	"spark",
	"app/Config/Routes.php",
	"app/Config/Database.php",
	"app/Config/App.php",
	"app/Config/Filters.php",
	"application/config/routes.php",
	"application/config/database.php",
	"composer.json",
	"routes/api.php",
	"routes/web.php",
	"main.go",
	"app.vue",
	"app/app.vue",
	"app/composables/useApi.ts",
	"composables/useApi.ts",
	"lib/main.dart",
	"README.md",
}

// keyFileCtrlDirs: direktori controller/composable yang dicuplik bila kode rute dibutuhkan.
var keyFileCtrlDirs = []string{"controllers", "routes", "handlers", "api", "composables", "app/composables", "server/api", "app/Controllers", "app/Controllers/Api", "application/controllers"}

// keyFileMaxLines: batas baris cuplikan per berkas.
func keyFileMaxLines(t string, withCode bool) int {
	switch {
	case withCode && (strings.HasSuffix(t, "main.go") || containsAny(t, "routes", "Routes", "useApi")):
		return 400
	case strings.Contains(t, "database"):
		return 150
	}
	return 60
}

// isSnippetSource: berkas kode yang layak dicuplik dari direktori controller.
func isSnippetSource(name string) bool {
	if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".test.ts") {
		return false
	}
	for _, ext := range []string{".go", ".ts", ".js", ".php"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// appendControllerSnippets menambahkan cuplikan (maks 8 berkas per direktori) dari direktori controller.
func appendControllerSnippets(sb *strings.Builder, dir string) {
	for _, cd := range keyFileCtrlDirs {
		p := filepath.Join(dir, cd)
		if info, err := os.Stat(p); err != nil || !info.IsDir() {
			continue
		}
		entries, _ := os.ReadDir(p)
		count := 0
		for _, e := range entries {
			if e.IsDir() || !isSnippetSource(e.Name()) {
				continue
			}
			count++
			if count > 8 {
				break
			}
			cContent, err := readSnippet(filepath.Join(p, e.Name()), 150)
			if err == nil && cContent != "" {
				sb.WriteString(fmt.Sprintf("\n--- FILE: %s/%s ---\n%s\n", cd, e.Name(), cContent))
			}
		}
	}
}

// extractKeyFiles: cuplikan berkas manifest/konfigurasi. withCode = sertakan kode rute/controller (hanya bila
// rute tidak bisa diekstrak statis); tanpa itu cukup 60 baris pertama per berkas.
func extractKeyFiles(dir string, withCode bool) string {
	var sb strings.Builder
	for _, t := range keyFileTargets {
		p := filepath.Join(dir, t)
		if !util.PathExists(p) {
			continue
		}
		content, err := readSnippet(p, keyFileMaxLines(t, withCode))
		if err == nil && content != "" {
			sb.WriteString(fmt.Sprintf("\n--- FILE: %s ---\n%s\n", t, content))
		}
	}

	if withCode {
		// Also extract controller signatures or composables if available
		appendControllerSnippets(&sb, dir)
	}
	return sb.String()
}

func readSnippet(path string, maxLines int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > maxLines {
		return strings.Join(lines[:maxLines], "\n") + "\n... (truncated)", nil
	}
	return string(data), nil
}

// pathExistsAll / pathExistsAny: pemeriksaan keberadaan beberapa path relatif terhadap dir.
func pathExistsAll(dir string, names ...string) bool {
	for _, n := range names {
		if !util.PathExists(filepath.Join(dir, n)) {
			return false
		}
	}
	return true
}

func pathExistsAny(dir string, names ...string) bool {
	for _, n := range names {
		if util.PathExists(filepath.Join(dir, n)) {
			return true
		}
	}
	return false
}

// archFramework: kerangka kerja terdeteksi bila teks memuat salah satu keys (urut, pertama cocok).
type archFramework struct {
	keys      []string
	framework string
	fwType    string
}

var goFrameworks = []archFramework{
	{[]string{"github.com/gofiber/fiber/v2"}, "Fiber v2", "FastHTTP Asynchronous High-Performance Web Engine"},
	{[]string{"github.com/gofiber/fiber/v3"}, "Fiber v3", "FastHTTP Asynchronous High-Performance Web Engine"},
	{[]string{"github.com/gin-gonic/gin"}, "Gin", "High-Performance HTTP Web Framework (HttpRouter Engine)"},
	{[]string{"github.com/labstack/echo/v4", "github.com/labstack/echo"}, "Echo v4", "Minimalist High-Performance Web Framework"},
	{[]string{"github.com/go-chi/chi/v5", "github.com/go-chi/chi"}, "Chi Router", "Lightweight, Idiomatic & Composable net/http Router"},
	{[]string{"github.com/gorilla/mux"}, "Gorilla Mux", "Standard-Compliant net/http URL Router & Dispatcher"},
	{[]string{"github.com/zeromicro/go-zero"}, "Go-Zero", "Microservice Web & RPC Framework with Built-in Tooling"},
	{[]string{"github.com/kataras/iris"}, "Iris", "Expressive High-Performance Web Framework"},
	{[]string{"github.com/beego/beego"}, "Beego", "Full-Stack MVC Framework for Go"},
}

// archPlugin: plugin terdeteksi bila teks memuat salah satu keys. group tak kosong = rantai else-if
// (hanya anggota pertama yang cocok dalam grup yang dipakai).
type archPlugin struct {
	keys  []string
	name  string
	group string
}

var goPlugins = []archPlugin{
	{[]string{"github.com/gofiber/contrib/jwt", "golang-jwt/jwt", "dgrijalva/jwt-go"}, "JWT Authentication (golang-jwt/jwt)", ""},
	{[]string{"firebase.google.com/go/v4"}, "Firebase Admin SDK (FCM & Push Notifications)", ""},
	{[]string{"gorm.io/gorm"}, "GORM v2 (Object-Relational Mapping)", ""},
	{[]string{"gorm.io/driver/mysql", "go-sql-driver/mysql"}, "MySQL Database Driver", ""},
	{[]string{"gorm.io/driver/postgres", "jackc/pgx", "lib/pq"}, "PostgreSQL Database Driver", ""},
	{[]string{"gorm.io/driver/sqlite", "mattn/go-sqlite3"}, "SQLite Driver (Offline / Fallback Database)", ""},
	{[]string{"github.com/jmoiron/sqlx"}, "SQLx (Database Extensions)", ""},
	{[]string{"go.mongodb.org/mongo-driver"}, "MongoDB Official Driver", ""},
	{[]string{"github.com/redis/go-redis", "go-redis/redis"}, "Redis Client (Cache & Session Storage)", ""},
	{[]string{"golang.org/x/crypto"}, "bcrypt / Argon2 Cryptography (golang.org/x/crypto)", ""},
	{[]string{"github.com/google/uuid", "gofrs/uuid"}, "Google UUID Generator", ""},
	{[]string{"github.com/joho/godotenv"}, "Godotenv (.env Configuration Loader)", ""},
	{[]string{"github.com/spf13/viper"}, "Viper (Configuration Management)", ""},
	{[]string{"github.com/go-playground/validator"}, "Go-Playground Validator", ""},
	{[]string{"github.com/klauspost/compress", "fasthttp"}, "Klauspost Compress (LevelBestSpeed HTTP Compression)", ""},
	{[]string{"github.com/gorilla/websocket"}, "Gorilla WebSocket (Bi-directional Streaming)", ""},
	{[]string{"go.uber.org/zap"}, "Uber Zap (Structured Fast Logger)", "logger"},
	{[]string{"github.com/sirupsen/logrus"}, "Logrus Structured Logger", "logger"},
	{[]string{"github.com/rs/zerolog"}, "Zerolog JSON Logger", "logger"},
	{[]string{"github.com/swaggo/swag"}, "Swagger / OpenAPI Specification", ""},
	{[]string{"github.com/prometheus/client_golang"}, "Prometheus Metrics Client", ""},
}

// goMainMiddlewares: middleware bawaan yang terlihat di main.go.
var goMainMiddlewares = []archPlugin{
	{[]string{"cors.New", "middleware.CORS"}, "CORS Security Middleware", ""},
	{[]string{"logger.New"}, "Fiber Logger Middleware", ""},
	{[]string{"recover.New"}, "Panic Recovery Middleware", ""},
}

// addUnique menambahkan val ke slice bila belum ada.
func addUnique(slice []string, val string) []string {
	if containsString(slice, val) {
		return slice
	}
	return append(slice, val)
}

// matchPlugins mengembalikan nama plugin (urut tabel) yang cocok dengan teks, menghormati grup else-if.
func matchPlugins(text string, table []archPlugin) []string {
	var out []string
	seenGroup := map[string]bool{}
	for _, p := range table {
		if p.group != "" && seenGroup[p.group] {
			continue
		}
		if containsAny(text, p.keys...) {
			out = append(out, p.name)
			if p.group != "" {
				seenGroup[p.group] = true
			}
		}
	}
	return out
}

// goHasE2EE: proyek memakai vault E2EE/Zero-Knowledge (dokumen penanda atau controller key/asset terenkripsi).
func goHasE2EE(dir string) bool {
	if pathExistsAny(dir, "MEKANISME_ENKRIPSI.md", "KEAMANAN_DAN_PEMULIHAN_AKUN.md") {
		return true
	}
	ctrlPath := filepath.Join(dir, "controllers")
	cEntries, err := os.ReadDir(ctrlPath)
	if err != nil {
		return false
	}
	for _, ce := range cEntries {
		if !containsAny(ce.Name(), "key", "asset") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(ctrlPath, ce.Name()))
		if containsAny(string(data), "encrypted_", "family_key", "AES") {
			return true
		}
	}
	return false
}

// goDesignPattern menentukan pola desain proyek Go dari hierarki direktori.
func goDesignPattern(dir string) (string, string) {
	hasCleanArch := pathExistsAll(dir, "domain") && pathExistsAny(dir, "usecase", "usecases") && pathExistsAny(dir, "repository", "repositories")
	hasMVC := pathExistsAll(dir, "controllers", "models")
	has3Tier := pathExistsAll(dir, "handlers", "services")
	hasStdLayout := pathExistsAll(dir, "cmd", "internal")
	hasE2EE := goHasE2EE(dir)

	switch {
	case hasCleanArch:
		return "Clean Architecture (Hexagonal / Ports & Adapters)",
			"Memisahkan software menjadi 4 lapisan independen: Domain (entitas bisnis), Use Cases (aturan aplikasi spesifik), Repositories (abstraksi database), dan Delivery / HTTP Handlers. Sangat modular, minim coupling, dan memudahkan testing otomatis secara terisolasi."
	case hasMVC && hasE2EE:
		return "Headless MVC with Zero-Knowledge E2EE Vault",
			"Kombinasi Headless Model-Controller dengan Zero-Knowledge End-to-End Encryption (E2EE) Vault. Server bertindak sebagai perantara sinkronisasi dan penyimpanan payload cipher (.enc), di mana seluruh kunci privat dan enkripsi data dikelola eksklusif pada sisi klien. Model mengorganisir entitas database & delta-sync, Controller menangani orkestrasi HTTP dan validasi, serta Middleware mengamankan rute via JWT."
	case hasMVC:
		return "MVC (Model-View-Controller / Model-Controller Pattern)",
			"Pemisahan tanggung jawab modular: Models menangani skema database dan persistensi ORM, Controllers mengeksekusi logika bisnis dan formatting respons HTTP, Routes mengelola pemetaan URL, serta Middleware menangani otentikasi lintas endpoint."
	case has3Tier:
		return "3-Tier Layered Architecture (Handler - Service - Repository)",
			"Arsitektur berlapis berorientasi service: Handlers mengelola serialisasi request/response HTTP, Services mengisolasi proses bisnis utama, dan Repositories bertanggung jawab atas query database serta transaksi data."
	case hasStdLayout:
		return "Standard Go Project Layout (cmd/internal/pkg)",
			"Mengikuti arsitektur idiomatik Standard Go: cmd/ berisi file entrypoint aplikasi, internal/ memuat seluruh kode privat yang terlindungi dari import luar, dan pkg/ memuat pustaka yang dapat digunakan kembali."
	}
	return "Modular Flat / Package-Oriented Architecture",
		"Struktur paket terorganisir per domain fungsional dengan dependensi langsung pada standar pustaka Go untuk throughput maksimal."
}

// detectGoArchitecture: framework, plugin, dan pola desain proyek Go (dir berisi go.mod).
func detectGoArchitecture(dir, goModPath string, arch models.ArchitectureInfo) models.ArchitectureInfo {
	if modBytes, err := os.ReadFile(goModPath); err == nil {
		sMod := string(modBytes)
		arch.Framework = "Standard net/http"
		arch.FrameworkType = "Go Standard Library HTTP Multiplexer (Go 1.22+ Enhanced)"
		for _, f := range goFrameworks {
			if containsAny(sMod, f.keys...) {
				arch.Framework, arch.FrameworkType = f.framework, f.fwType
				break
			}
		}
		for _, name := range matchPlugins(sMod, goPlugins) {
			arch.Plugins = addUnique(arch.Plugins, name)
		}
	}

	// Also check built-in middlewares in main.go or routes
	if mainBytes, err := os.ReadFile(filepath.Join(dir, "main.go")); err == nil {
		arch.Plugins = append(arch.Plugins, matchPlugins(string(mainBytes), goMainMiddlewares)...)
	}

	arch.DesignPattern, arch.PatternDetails = goDesignPattern(dir)
	return arch
}

// phpArchRule: kerangka PHP yang dikenali dari berkas penanda atau isi composer.json.
type phpArchRule struct {
	paths    []string
	composer []string
	arch     models.ArchitectureInfo
}

var phpArchRules = []phpArchRule{
	{[]string{"spark", "app/Config/Routes.php"}, []string{"codeigniter4"}, models.ArchitectureInfo{
		Framework: "CodeIgniter 4", FrameworkType: "Lightweight High-Performance PHP MVC Framework",
		DesignPattern:  "MVC (Model-View-Controller)",
		PatternDetails: "Struktur MVC modular: Controllers menangani orkestrasi HTTP dan validasi, Models mengorganisir entitas database dan migrasi via Spark CLI, serta Filters mengamankan rute via JWT dan Middleware."}},
	{[]string{"application/config/routes.php", "system/core/CodeIgniter.php"}, nil, models.ArchitectureInfo{
		Framework: "CodeIgniter 3", FrameworkType: "Classic PHP MVC Framework",
		DesignPattern:  "MVC (Model-View-Controller)",
		PatternDetails: "Arsitektur MVC klasik CodeIgniter: Controllers mengeksekusi request HTTP, Models menangani persistensi Active Record, dan Views merender antarmuka."}},
	{[]string{"artisan"}, []string{"laravel"}, models.ArchitectureInfo{
		Framework: "Laravel", FrameworkType: "Fullstack PHP Framework (Eloquent ORM & Artisan CLI)",
		DesignPattern:  "MVC (Model-View-Controller)",
		PatternDetails: "Arsitektur MVC modern Laravel: Eloquent Models mengelola relasi database, Controllers mengelola alur bisnis, Routes memetakan endpoint REST, serta Middleware menangani autentikasi & otorisasi."}},
}

// detectPHPArchitecture mengenali CodeIgniter 4/3, Laravel, atau mengembalikan arsitektur generik.
func detectPHPArchitecture(dir string, arch models.ArchitectureInfo) models.ArchitectureInfo {
	composerContent := ""
	if composerPath := filepath.Join(dir, "composer.json"); util.PathExists(composerPath) {
		composerContent = util.ReadFileString(composerPath)
	}
	for _, r := range phpArchRules {
		if pathExistsAny(dir, r.paths...) || containsAny(composerContent, r.composer...) {
			found := r.arch
			found.Plugins = arch.Plugins
			return found
		}
	}
	arch.Framework = "Custom / Standard"
	arch.FrameworkType = "Modular Application"
	arch.DesignPattern = "Component-Based Architecture"
	arch.PatternDetails = "Aplikasi modular mandiri."
	return arch
}

// detectArchitecture analyzes manifests, go.mod, directory hierarchy, and key files to identify framework, plugins, and design patterns
func detectArchitecture(dir string) models.ArchitectureInfo {
	var arch models.ArchitectureInfo
	arch.Plugins = make([]string, 0)

	// 1. Check Go projects
	if goModPath := filepath.Join(dir, "go.mod"); util.PathExists(goModPath) {
		return detectGoArchitecture(dir, goModPath, arch)
	}

	// 2. Check Node / Nuxt / Next / Vue / React / Svelte / Angular projects
	if pkgJsonPath := filepath.Join(dir, "package.json"); util.PathExists(pkgJsonPath) {
		return detectNodeAndFrontendArchitecture(dir, pkgJsonPath)
	}

	// 3. Check Flutter / Dart projects
	if pubspecPath := filepath.Join(dir, "pubspec.yaml"); util.PathExists(pubspecPath) {
		return detectFlutterArchitecture(dir, pubspecPath)
	}

	// 4. Check PHP projects (CodeIgniter, Laravel, etc.)
	return detectPHPArchitecture(dir, arch)
}

type packageJSON struct {
	Name            string            `json:"name"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	PackageManager  string            `json:"packageManager"`
}

// nodePlugins: plugin frontend terdeteksi bila salah satu dependency ada (urut = urutan tampil).
var nodePlugins = []struct {
	deps []string
	name string
}{
	{[]string{"@nuxt/ui"}, "Nuxt UI (Design System & Headless Components)"},
	{[]string{"tailwindcss", "@nuxtjs/tailwindcss"}, "Tailwind CSS (Utility-First Design Engine)"},
	{[]string{"unocss"}, "UnoCSS (Instant Atomic CSS Engine)"},
	{[]string{"primevue"}, "PrimeVue Component Suite"},
	{[]string{"vuetify"}, "Vuetify Material Design 3 Framework"},
	{[]string{"radix-vue", "shadcn-vue", "@radix-ui/react-dialog"}, "Radix / Shadcn UI Primitives"},
	{[]string{"element-plus"}, "Element Plus Desktop UI Kit"},
	{[]string{"ant-design-vue", "antd"}, "Ant Design Enterprise Kit"},
	{[]string{"daisyui"}, "DaisyUI Semantic Components"},
	{[]string{"@headlessui/vue", "@headlessui/react"}, "Headless UI Accessible Components"},
	{[]string{"@iconify-json/lucide", "lucide-vue-next", "lucide-react"}, "Lucide Icons (Vector Iconography)"},
	{[]string{"@iconify-json/simple-icons"}, "Simple Icons (Brand & Tech Vectors)"},
	{[]string{"@heroicons/vue", "@heroicons/react"}, "Heroicons SVG Icons"},
	{[]string{"pinia", "@pinia/nuxt"}, "Pinia (Reactive Central Store & Persistence)"},
	{[]string{"vuex"}, "Vuex Centralized Store"},
	{[]string{"@reduxjs/toolkit", "redux"}, "Redux Toolkit State Container"},
	{[]string{"zustand"}, "Zustand Lightweight State Manager"},
	{[]string{"ofetch"}, "OFetch (Resilient HTTP Client)"},
	{[]string{"axios"}, "Axios HTTP Client with Interceptors"},
	{[]string{"@tanstack/vue-query", "@tanstack/react-query"}, "TanStack Query (Async Cache & Sync)"},
	{[]string{"@trpc/client"}, "tRPC TypeSafe API Client"},
	{[]string{"socket.io-client"}, "Socket.IO Real-Time WebSocket Client"},
	{[]string{"zod"}, "Zod (TypeScript-First Schema Validation)"},
	{[]string{"vee-validate"}, "VeeValidate Form Validation"},
	{[]string{"@formkit/vue"}, "FormKit Form Builder"},
	{[]string{"chart.js", "vue-chartjs"}, "Chart.js (Interactive Data Visualization)"},
	{[]string{"echarts", "vue-echarts"}, "Apache ECharts Visualizations"},
	{[]string{"apexcharts", "vue3-apexcharts"}, "ApexCharts Modern SVG Charts"},
	{[]string{"@vueuse/core"}, "VueUse (Composition API Utilities)"},
	{[]string{"date-fns", "dayjs"}, "Dayjs / Date-fns Timezone Formatter"},
	{[]string{"typescript", "vue-tsc"}, "TypeScript Strict Typing Engine"},
	{[]string{"eslint", "@nuxt/eslint"}, "ESLint Code Quality & Hygiene"},
	{[]string{"vitest"}, "Vitest Unit Testing Framework"},
	{[]string{"@playwright/test"}, "Playwright End-to-End Automation"},
}

// nodeFrameworkName menentukan framework frontend (nama, tipe) dari flag deteksi, urut prioritas.
func nodeFrameworkName(pkg packageJSON, dir string) (fw, fwType string, isNuxt, isNext bool) {
	isNuxt = pkg.hasDep("nuxt") || pathExistsAny(dir, "nuxt.config.ts", "nuxt.config.js")
	isNext = pkg.hasDep("next") || pathExistsAny(dir, "next.config.js", "next.config.mjs", "next.config.ts")
	switch {
	case isNuxt:
		nuxtVer := "Nuxt 3"
		if v, ok := pkg.Dependencies["nuxt"]; ok && strings.Contains(v, "4.") {
			nuxtVer = "Nuxt 4"
		} else if v, ok := pkg.DevDependencies["nuxt"]; ok && strings.Contains(v, "4.") {
			nuxtVer = "Nuxt 4"
		}
		return nuxtVer + " (Vue 3.5+)", "Vue 3 Universal Fullstack SSR & Nitro Web Engine", isNuxt, isNext
	case isNext:
		return "Next.js (React 18/19)", "React Fullstack SSR & React Server Components (Turbopack Engine)", isNuxt, isNext
	case pkg.hasAnyDep("@sveltejs/kit", "svelte"):
		return "SvelteKit", "Compiling Reactive Web Framework (Vite Bundler)", isNuxt, isNext
	case pkg.hasDep("vue"):
		return "Vue 3 SPA", "Declarative Component-Based SPA (Vite Bundler)", isNuxt, isNext
	case pkg.hasDep("react"):
		return "React SPA", "Component-Based Reactive SPA (Vite/Webpack Bundler)", isNuxt, isNext
	case pkg.hasDep("@angular/core"):
		return "Angular", "Enterprise TypeScript Web Platform", isNuxt, isNext
	}
	return "Node.js Web App", "JavaScript / TypeScript Application", isNuxt, isNext
}

// nodeDesignPattern menentukan pola desain frontend dari hierarki direktori.
func nodeDesignPattern(dir string, isNuxt, isNext bool) (string, string) {
	hasPages := pathExistsAny(dir, "pages", "app/pages")
	hasComposables := pathExistsAny(dir, "composables", "app/composables")
	hasComponents := pathExistsAny(dir, "components", "app/components")
	hasServerApi := pathExistsAny(dir, "server/api", "server/routes")
	hasMiddleware := pathExistsAny(dir, "middleware", "app/middleware")

	var pattern, details string
	switch {
	case isNuxt && hasPages && hasComposables:
		pattern = "Vue 3 Composition API with Feature-Driven Architecture & File-Based Routing"
		details = "Arsitektur frontend modern berbasis konvensi direktori: app/pages/ untuk routing file deklaratif, app/composables/ untuk pemanggilan API terisolasi (useApi) dan reusable state, app/components/ untuk antarmuka atomik & modular (Nuxt UI/Tailwind), serta middleware untuk proteksi rute navigasi klien."
	case isNext:
		pattern = "Next.js App Router Architecture (React Server Components)"
		details = "Arsitektur modular React berbasis Server Components, Client Hooks, dan Server Actions dengan rute direktori app/."
	case hasComposables || hasComponents:
		pattern = "Component-Driven Reactive SPA Architecture"
		details = "Pemisahan lapisan presentasi deklaratif dengan manajemen state reaktif, isolasi komponen visual, dan pemanggilan API terstruktur."
	default:
		pattern = "Modular Frontend Web Architecture"
		details = "Struktur paket terorganisir per modul fungsional antarmuka."
	}
	if hasServerApi {
		details += " Dilengkapi Nitro Serverless Engine (server/api) untuk endpoint API fullstack hybrid."
	}
	if hasMiddleware {
		details += " Memanfaatkan Route Navigation Guards untuk otorisasi akses sebelum pergantian halaman."
	}
	return pattern, details
}

func detectNodeAndFrontendArchitecture(dir, pkgPath string) models.ArchitectureInfo {
	var arch models.ArchitectureInfo
	arch.Plugins = make([]string, 0)

	data, err := os.ReadFile(pkgPath)
	var pkg packageJSON
	if err == nil {
		_ = json.Unmarshal(data, &pkg)
	}

	var isNuxt, isNext bool
	arch.Framework, arch.FrameworkType, isNuxt, isNext = nodeFrameworkName(pkg, dir)

	for _, p := range nodePlugins {
		if pkg.hasAnyDep(p.deps...) {
			arch.Plugins = addUnique(arch.Plugins, p.name)
		}
	}

	arch.DesignPattern, arch.PatternDetails = nodeDesignPattern(dir, isNuxt, isNext)
	return arch
}

// flutterStateRule: state management Flutter dari isi pubspec (urut, pertama cocok); plugin kosong = tanpa plugin.
type flutterStateRule struct {
	keys    []string
	plugin  string
	pattern string
	details string
}

var flutterStateRules = []flutterStateRule{
	{[]string{"flutter_bloc", "bloc:"}, "BLoC Pattern State Management", "BLoC (Business Logic Component) Architecture", "Pemisahan tegas antara UI Presentasi, Business Logic (BLoC Events/States), dan Data Repositories."},
	{[]string{"flutter_riverpod", "riverpod:"}, "Riverpod Reactive State Management", "Riverpod Unidirectional Data Flow Architecture", "State management reaktif compile-time safe dengan dependency injection deklaratif."},
	{[]string{"provider:"}, "Provider State Management", "Provider & ChangeNotifier Architecture", "Pengelolaan state deklaratif berbasis inherited widgets dan model notifikasi perubahan."},
	{[]string{"get:"}, "GetX Ecosystem", "GetX Reactive Micro-Framework", "Navigasi tanpa konteks, dependency injection ringan, dan reactive state management."},
}

var flutterExtraPlugins = []archPlugin{
	{[]string{"dio:"}, "Dio HTTP Networking Client", ""},
	{[]string{"go_router:"}, "GoRouter Declarative Routing", ""},
	{[]string{"hive:", "isar:"}, "Hive / Isar Fast Local NoSQL Database", ""},
	{[]string{"shared_preferences:"}, "SharedPreferences Key-Value Storage", ""},
}

func detectFlutterArchitecture(dir, pubspecPath string) models.ArchitectureInfo {
	var arch models.ArchitectureInfo
	arch.Framework = "Flutter (Dart SDK)"
	arch.FrameworkType = "Cross-Platform Declarative Mobile/Web UI Toolkit"
	arch.Plugins = []string{"Flutter SDK", "Dart SDK", "Material Design 3"}

	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		return arch
	}
	s := string(content)
	arch.DesignPattern = "Clean Architecture for Flutter (lib/domain/data/presentation)"
	arch.PatternDetails = "Pemisahan UI deklaratif dengan state management dan repository layer pada direktori lib/."
	for _, r := range flutterStateRules {
		if containsAny(s, r.keys...) {
			arch.Plugins = append(arch.Plugins, r.plugin)
			arch.DesignPattern, arch.PatternDetails = r.pattern, r.details
			break
		}
	}
	arch.Plugins = append(arch.Plugins, matchPlugins(s, flutterExtraPlugins)...)
	return arch
}

// extractRoutes parses route definitions using the extensible adapter registry
func extractRoutes(dir string, role ...string) []models.RouteInfo {
	compRole := ""
	if len(role) > 0 {
		compRole = role[0]
	}
	return adapters.ScanRoutes(dir, compRole)
}

func mergeRoutes(llmRoutes, staticRoutes []models.RouteInfo) []models.RouteInfo {
	staticMap := make(map[string]models.RouteInfo)
	for _, sr := range staticRoutes {
		key := fmt.Sprintf("%s:%s", sr.Method, sr.Path)
		staticMap[key] = sr
	}

	merged := make([]models.RouteInfo, len(llmRoutes))
	existing := make(map[string]bool)
	for i, r := range llmRoutes {
		key := fmt.Sprintf("%s:%s", r.Method, r.Path)
		existing[key] = true
		if sr, ok := staticMap[key]; ok {
			if len(r.TargetRequests) == 0 && len(sr.TargetRequests) > 0 {
				r.TargetRequests = sr.TargetRequests
			}
			if r.Handler == "" && sr.Handler != "" {
				r.Handler = sr.Handler
			}
		}
		merged[i] = r
	}
	for _, sr := range staticRoutes {
		key := fmt.Sprintf("%s:%s", sr.Method, sr.Path)
		if !existing[key] {
			merged = append(merged, sr)
			existing[key] = true
		}
	}
	return merged
}

func generateDetailedMarkdownReport(res *models.FastScanResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Analisis Arsitektur & Spesifikasi Komponen: %s\n\n", res.ComponentName))
	sb.WriteString(fmt.Sprintf("- **Role**: `%s`\n", res.Role))
	sb.WriteString(fmt.Sprintf("- **Path**: `%s`\n", res.AbsolutePath))
	sb.WriteString(fmt.Sprintf("- **Bahasa & Framework**: `%s` / `%s`\n", res.DetectedParams.Language, res.DetectedParams.Framework))
	sb.WriteString(fmt.Sprintf("- **Listening Port**: `:%d`\n\n", res.DetectedParams.Port))

	sb.WriteString("### 1. Arsitektur, Framework & Desain Pattern\n\n")
	sb.WriteString(fmt.Sprintf("- **Framework**: **%s** (%s)\n", res.Architecture.Framework, res.Architecture.FrameworkType))
	sb.WriteString(fmt.Sprintf("- **Desain Pattern**: **%s**\n", res.Architecture.DesignPattern))
	if res.Architecture.PatternDetails != "" {
		sb.WriteString(fmt.Sprintf("- **Detail Pola Arsitektur**: %s\n", res.Architecture.PatternDetails))
	}
	if len(res.Architecture.Plugins) > 0 {
		sb.WriteString(fmt.Sprintf("- **Plugins, Libraries & Tooling**: %s\n", strings.Join(res.Architecture.Plugins, ", ")))
	}
	if caps, ok := res.ExtractedEnv.DynamicMetadata["Frontend Capabilities"]; ok && caps != "" {
		sb.WriteString(fmt.Sprintf("- **Kemampuan Utama (Capabilities)**: %s\n", caps))
	}
	sb.WriteString("\n")

	sb.WriteString("### 2. Database & Persistence Layer\n\n")
	sb.WriteString(fmt.Sprintf("- **Database Engine**: **%s**\n", res.ExtractedEnv.DatabaseType))
	sb.WriteString(fmt.Sprintf("- **ORM / Driver**: `%s`\n", res.ExtractedEnv.ORMDriver))
	sb.WriteString(fmt.Sprintf("- **Nama Database**: `%s`\n", res.ExtractedEnv.DatabaseName))
	if host, ok := res.ExtractedEnv.DynamicMetadata["Database Host"]; ok && host != "" {
		sb.WriteString(fmt.Sprintf("- **Host Koneksi**: `%s`\n", host))
	}
	sb.WriteString("\n")

	sb.WriteString("### 3. Mekanisme Autentikasi & Keamanan\n\n")
	sb.WriteString(fmt.Sprintf("- **Metode Autentikasi**: %s\n", strings.Join(res.ExtractedEnv.AuthMethods, ", ")))
	sb.WriteString(fmt.Sprintf("- **Endpoint Login**: `%s`\n", strings.Join(res.ExtractedEnv.LoginEndpoints, "`, `")))
	sb.WriteString(fmt.Sprintf("- **Petunjuk Akses**: %s\n\n", res.ExtractedEnv.LoginAndAccessGuide))

	sb.WriteString(fmt.Sprintf("### 4. Daftar Lengkap Router & Halaman (%d Rute / Endpoint Terdeteksi)\n\n", len(res.Routes)))
	sb.WriteString("| Method | Endpoint / URL Path | Target Fungsi (Handler) | Keamanan | Penjelasan Fungsi |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")
	for _, r := range res.Routes {
		sb.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | %s | %s |\n", r.Method, r.Path, r.Handler, r.Middleware, r.Description))
	}
	sb.WriteString("\n")

	sb.WriteString("### 5. Panduan Eksekusi Lokal\n\n")
	sb.WriteString(fmt.Sprintf("- **Dev Command**: `%s`\n", res.HowToRun.DevCommand))
	sb.WriteString(fmt.Sprintf("- **Build Command**: `%s`\n", res.HowToRun.BuildCommand))
	if res.HowToRun.InstallCommand != "" {
		sb.WriteString(fmt.Sprintf("- **Install Dependencies**: `%s`\n", res.HowToRun.InstallCommand))
	}
	sb.WriteString("\n")

	sb.WriteString("### 6. Audit Sensor Kunci & Double Security Shield\n\n")
	redactedCount, _ := res.SecurityAudit["redacted_count"].(int)
	msg, _ := res.SecurityAudit["message"].(string)
	if redactedCount > 0 {
		sb.WriteString(fmt.Sprintf("- **Status Proteksi**: 🛡️ **%d Kredensial/Kunci Berhasil Disensor** secara otomatis.\n", redactedCount))
		sb.WriteString(fmt.Sprintf("- **Audit Detail**: %s\n", msg))
	} else {
		sb.WriteString("- **Status Proteksi**: 🛡️ **Zero Credential Leaks** — Payload 100% aman dan bersih.\n")
	}
	sb.WriteString("- **Kebijakan Sensor**: Double Sanitization aktif. Semua kunci disensor dengan safe aliasing deterministik sebelum dikirim ke AI atau kemanapun.\n\n")

	return sb.String()
}
