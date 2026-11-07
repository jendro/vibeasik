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

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/scanner/adapters"
	"github.com/vibeasik/vibeasik/pkg/security"
)

// FastScanComponent performs an AI-powered quick scan on a project component
func FastScanComponent(db *gorm.DB, comp *models.Component, proj *models.Project) (*models.FastScanResult, error) {
	targetDir := comp.AbsolutePath
	if targetDir == "" {
		if proj != nil && proj.RootPath != "" && comp.RelativePath != "" {
			targetDir = filepath.Join(proj.RootPath, comp.RelativePath)
		} else if comp.RelativePath != "" {
			targetDir = comp.RelativePath
		}
	}

	absDir, err := filepath.Abs(targetDir)
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

	systemPrompt := `You are an elite fullstack software architect, security auditor, and devops expert.
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
	routesSchema, routesAsk := "", fmt.Sprintf("Routes (%d) were already extracted statically: do NOT list them.", len(staticRoutes))
	if needRoutes {
		routesSchema = fastScanRoutesSchema
		routesAsk = "List all routers, their HTTP methods, exact endpoint paths, handler functions, and short descriptions."
	}
	systemPrompt = fmt.Sprintf(systemPrompt, routesSchema)

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

	parsed := parseFastScanJSON(aiRawResponse)
	if parsed != nil {
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
	} else {
		// Fallback heuristic if LLM returned free text or was offline
		fallbackHeuristic(result, absDir, comp, aiRawResponse)
	}

	// Guarantee architecture detection completeness (Framework, Plugins, Design Pattern)
	detectedArch := detectArchitecture(absDir)
	if result.Architecture.Framework == "" {
		result.Architecture.Framework = detectedArch.Framework
	}
	if result.Architecture.FrameworkType == "" {
		result.Architecture.FrameworkType = detectedArch.FrameworkType
	}
	if result.Architecture.DesignPattern == "" {
		result.Architecture.DesignPattern = detectedArch.DesignPattern
	}
	if result.Architecture.PatternDetails == "" {
		result.Architecture.PatternDetails = detectedArch.PatternDetails
	}
	if len(result.Architecture.Plugins) == 0 {
		result.Architecture.Plugins = detectedArch.Plugins
	} else {
		// Merge any plugins detected statically that LLM missed
		for _, p := range detectedArch.Plugins {
			if !containsString(result.Architecture.Plugins, p) {
				result.Architecture.Plugins = append(result.Architecture.Plugins, p)
			}
		}
	}

	// Guarantee complete route extraction & merging via adapter architecture
	if len(result.Routes) == 0 {
		result.Routes = staticRoutes
	} else if len(staticRoutes) > len(result.Routes) {
		result.Routes = mergeRoutes(result.Routes, staticRoutes)
	}

	// Guarantee database extraction completeness
	heuristicEnv := extractEnvHeuristic(absDir)
	if result.ExtractedEnv.DatabaseType == "" || result.ExtractedEnv.DatabaseType == "None detected" {
		result.ExtractedEnv.DatabaseType = heuristicEnv.DatabaseType
	}
	if result.ExtractedEnv.ORMDriver == "" || result.ExtractedEnv.ORMDriver == "Native / Standard" {
		result.ExtractedEnv.ORMDriver = heuristicEnv.ORMDriver
	}
	if result.ExtractedEnv.DatabaseName == "" {
		result.ExtractedEnv.DatabaseName = heuristicEnv.DatabaseName
	}
	if len(result.ExtractedEnv.AuthMethods) == 0 && len(heuristicEnv.AuthMethods) > 0 {
		result.ExtractedEnv.AuthMethods = heuristicEnv.AuthMethods
	}
	if len(result.ExtractedEnv.LoginEndpoints) == 0 && len(heuristicEnv.LoginEndpoints) > 0 {
		result.ExtractedEnv.LoginEndpoints = heuristicEnv.LoginEndpoints
	}

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

func fallbackHeuristic(result *models.FastScanResult, absDir string, comp *models.Component, rawAi string) {
	lang := comp.Language
	pkgMgr := comp.PackageManager
	devCmd := comp.DevCommand
	buildCmd := comp.BuildCommand
	port := comp.Port

	var prereqs []string
	var installCmd, testCmd string
	var steps []string

	// Detect by files on disk
	if fileExists(filepath.Join(absDir, "go.mod")) {
		lang = "go"
		pkgMgr = "go"
		if devCmd == "" {
			devCmd = "go run main.go"
		}
		if buildCmd == "" {
			buildCmd = "go build -o server main.go"
		}
		testCmd = "go test ./..."
		installCmd = "go mod download"
		prereqs = []string{"Go 1.21+"}
		if port == 0 {
			port = 8080
		}
		steps = []string{
			"1. Pastikan Go terinstall (`go version`).",
			"2. Jalankan `go mod download` untuk mengunduh dependency.",
			"3. Jalankan `go run main.go` untuk mode development.",
			"4. Akses server di port :" + strconv.Itoa(port) + ".",
		}
	} else if fileExists(filepath.Join(absDir, "pubspec.yaml")) {
		lang = "dart"
		comp.Framework = "flutter"
		pkgMgr = "pub"
		if devCmd == "" {
			devCmd = "flutter run"
		}
		if buildCmd == "" {
			buildCmd = "flutter build apk"
		}
		testCmd = "flutter test"
		installCmd = "flutter pub get"
		prereqs = []string{"Flutter SDK 3.x", "Android Studio / VS Code"}
		steps = []string{
			"1. Pastikan Flutter SDK terpasang (`flutter doctor`).",
			"2. Jalankan `flutter pub get` untuk mengunduh paket.",
			"3. Buka emulator atau sambungkan perangkat fisik.",
			"4. Jalankan `flutter run` untuk memulai aplikasi mobile.",
		}
	} else if fileExists(filepath.Join(absDir, "package.json")) {
		pkgMgr = "bun"
		if fileExists(filepath.Join(absDir, "pnpm-lock.yaml")) {
			pkgMgr = "pnpm"
		} else if fileExists(filepath.Join(absDir, "yarn.lock")) {
			pkgMgr = "yarn"
		} else if fileExists(filepath.Join(absDir, "package-lock.json")) {
			pkgMgr = "npm"
		}

		if fileExists(filepath.Join(absDir, "nuxt.config.ts")) || fileExists(filepath.Join(absDir, "nuxt.config.js")) {
			comp.Framework = "nuxt"
			if port == 0 {
				port = 3000
			}
		}

		lang = "typescript"
		installCmd = fmt.Sprintf("%s install", pkgMgr)
		if devCmd == "" {
			devCmd = fmt.Sprintf("%s run dev", pkgMgr)
		}
		if buildCmd == "" {
			buildCmd = fmt.Sprintf("%s run build", pkgMgr)
		}
		testCmd = fmt.Sprintf("%s test", pkgMgr)
		prereqs = []string{fmt.Sprintf("Node.js 20+ atau %s", pkgMgr)}
		steps = []string{
			fmt.Sprintf("1. Jalankan `%s install` untuk menginstal modul.", pkgMgr),
			fmt.Sprintf("2. Jalankan `%s` untuk menjalankan dev server.", devCmd),
			fmt.Sprintf("3. Buka browser di http://localhost:%d", port),
		}
	} else if fileExists(filepath.Join(absDir, "spark")) || fileExists(filepath.Join(absDir, "app/Config/Routes.php")) || (fileExists(filepath.Join(absDir, "composer.json")) && strings.Contains(readFileString(filepath.Join(absDir, "composer.json")), "codeigniter")) {
		lang = "php"
		comp.Framework = "codeigniter"
		pkgMgr = "composer"
		if devCmd == "" {
			devCmd = "php spark serve"
		}
		testCmd = "php spark test"
		installCmd = "composer install"
		prereqs = []string{"PHP 8.2+", "Composer"}
		if port == 0 {
			port = 8080
		}
		steps = []string{
			"1. Pastikan PHP dan Composer terpasang (`php -v`, `composer -V`).",
			"2. Jalankan `composer install` untuk mengunduh dependency.",
			"3. Salin environment file: `cp env .env` dan sesuaikan konfigurasi database.",
			"4. Jalankan `php spark serve` untuk memulai server development lokal.",
			"5. Akses server di port :" + strconv.Itoa(port) + ".",
		}
	} else if fileExists(filepath.Join(absDir, "artisan")) || (fileExists(filepath.Join(absDir, "composer.json")) && strings.Contains(readFileString(filepath.Join(absDir, "composer.json")), "laravel")) {
		lang = "php"
		comp.Framework = "laravel"
		pkgMgr = "composer"
		if devCmd == "" {
			devCmd = "php artisan serve"
		}
		testCmd = "php artisan test"
		installCmd = "composer install"
		prereqs = []string{"PHP 8.2+", "Composer"}
		if port == 0 {
			port = 8000
		}
		steps = []string{
			"1. Pastikan PHP dan Composer terpasang (`php -v`, `composer -V`).",
			"2. Jalankan `composer install` untuk mengunduh dependency.",
			"3. Jalankan `php artisan serve` untuk memulai server development lokal.",
			"4. Akses server di port :" + strconv.Itoa(port) + ".",
		}
	}

	result.FolderStructureSummary = fmt.Sprintf("Komponen %s berada di `%s`. Terdeteksi arsitektur berbasis %s dengan package manager %s.", comp.Name, absDir, lang, pkgMgr)
	result.HowToRun = models.HowToRunInfo{
		Prerequisites:  prereqs,
		InstallCommand: installCmd,
		DevCommand:     devCmd,
		BuildCommand:   buildCmd,
		TestCommand:    testCmd,
		Steps:          steps,
	}
	result.DetectedParams = models.DetectedParams{
		Language:       lang,
		Framework:      comp.Framework,
		PackageManager: pkgMgr,
		DevCommand:     devCmd,
		BuildCommand:   buildCmd,
		Port:           port,
	}

	// Extract environment, database, auth, and dynamic info heuristically
	extractedEnv := extractEnvHeuristic(absDir)
	result.ExtractedEnv = extractedEnv
	result.Routes = extractRoutes(absDir, comp.Role)

	if strings.TrimSpace(rawAi) != "" && !strings.HasPrefix(rawAi, "System One Evaluation") {
		result.AiAnalysisMarkdown = rawAi
	} else {
		result.AiAnalysisMarkdown = generateDetailedMarkdownReport(result)
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

	// Check .env, .env.example, .env.server, etc.
	envFiles := []string{".env", ".env.example", ".env.server", ".env.local", ".env.development"}
	foundEnvPath := ""
	for _, ef := range envFiles {
		p := filepath.Join(dir, ef)
		if fileExists(p) {
			foundEnvPath = p
			break
		}
	}

	dbType := "None detected"
	ormDriver := "Native / Standard"
	dbName := ""

	if foundEnvPath != "" {
		f, err := os.Open(foundEnvPath)
		if err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					rawVal := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
					cleanVal := security.SanitizeText(rawVal)
					isSecret := strings.Contains(strings.ToUpper(key), "KEY") ||
						strings.Contains(strings.ToUpper(key), "SECRET") ||
						strings.Contains(strings.ToUpper(key), "PASS") ||
						strings.Contains(strings.ToUpper(key), "TOKEN")

					purpose := "Konfigurasi runtime lingkungan"
					upperKey := strings.ToUpper(key)

					if strings.Contains(upperKey, "DATABASE.DEFAULT.DATABASE") || upperKey == "DB_DATABASE" || strings.Contains(upperKey, "DB_NAME") || strings.Contains(upperKey, "DATABASE_NAME") {
						purpose = "Nama skema / database utama"
						dbName = cleanVal
						envInfo.DynamicMetadata["Database Name"] = cleanVal
					} else if strings.Contains(upperKey, "DATABASE.DEFAULT.DBDRIVER") || upperKey == "DB_CONNECTION" || strings.Contains(upperKey, "DB_DRIVER") {
						purpose = "Driver koneksi database"
						dbType = strings.ToUpper(cleanVal)
						if strings.EqualFold(cleanVal, "mysqli") || strings.EqualFold(cleanVal, "mysql") {
							dbType = "MySQL"
							ormDriver = "CodeIgniter Model / MySQLi"
						} else if strings.EqualFold(cleanVal, "postgre") || strings.EqualFold(cleanVal, "postgres") {
							dbType = "PostgreSQL"
							ormDriver = "CodeIgniter Model / Postgre"
						}
					} else if strings.Contains(upperKey, "DATABASE.DEFAULT.HOSTNAME") || strings.Contains(upperKey, "DB_HOST") || strings.Contains(upperKey, "DATABASE_HOST") {
						purpose = "Host alamat server database"
						envInfo.DynamicMetadata["Database Host"] = cleanVal
					} else if strings.Contains(upperKey, "DATABASE.DEFAULT.PORT") || strings.Contains(upperKey, "DB_PORT") {
						purpose = "Port jaringan listening service database"
						envInfo.DynamicMetadata["Database Port"] = cleanVal
					} else if strings.Contains(upperKey, "APP.BASEURL") || strings.Contains(upperKey, "APP_URL") {
						purpose = "Base URL aplikasi"
						envInfo.DynamicMetadata["Base URL"] = cleanVal
					} else if strings.Contains(upperKey, "CI_ENVIRONMENT") || strings.Contains(upperKey, "APP_ENV") {
						purpose = "Environment runtime aplikasi (development/production)"
						envInfo.DynamicMetadata["Environment"] = cleanVal
					} else if strings.Contains(upperKey, "PORT") {
						purpose = "Port jaringan listening service"
						envInfo.DynamicMetadata["Port"] = cleanVal
					} else if strings.Contains(upperKey, "JWT") {
						purpose = "Kunci rahasia enkripsi JWT token autentikasi"
						if !containsString(envInfo.AuthMethods, "JWT Bearer Token") {
							envInfo.AuthMethods = append(envInfo.AuthMethods, "JWT Bearer Token")
						}
					} else if strings.Contains(upperKey, "REDIS") {
						purpose = "Koneksi cache dan queue in-memory Redis"
						envInfo.ThirdPartyServices = append(envInfo.ThirdPartyServices, models.DetectedService{Name: "Redis", Purpose: "In-Memory Caching & Session"})
					} else if strings.Contains(upperKey, "OPENAI") || strings.Contains(upperKey, "AI_") {
						purpose = "Kredensial API AI / LLM"
						envInfo.ThirdPartyServices = append(envInfo.ThirdPartyServices, models.DetectedService{Name: "OpenAI / AI Engine", Purpose: "Integrasi Kecerdasan Buatan"})
					} else if strings.Contains(upperKey, "FIREBASE") {
						purpose = "Kredensial layanan Firebase Admin SDK"
						envInfo.ThirdPartyServices = append(envInfo.ThirdPartyServices, models.DetectedService{Name: "Firebase", Purpose: "Push Notification & Admin Auth"})
					}

					envInfo.EnvironmentKeys = append(envInfo.EnvironmentKeys, models.EnvKeyDescription{
						Key:         key,
						Purpose:     purpose,
						Required:    true,
						IsSecret:    isSecret,
						SafeExample: cleanVal,
					})
				}
			}
		}
	}

	// Detect DB from database source code files (e.g. database/database.go, internal/database/db.go, etc.)
	dbFiles := []string{
		"database/database.go",
		"database/db.go",
		"db/database.go",
		"db/db.go",
		"database.go",
		"db.go",
		"internal/database/db.go",
		"internal/database/database.go",
	}
	for _, dbf := range dbFiles {
		p := filepath.Join(dir, dbf)
		if fileExists(p) {
			content, err := os.ReadFile(p)
			if err == nil {
				sContent := string(content)
				if strings.Contains(sContent, "mysql") || strings.Contains(sContent, "driver/mysql") {
					dbType = "MySQL"
					if strings.Contains(sContent, "gorm") {
						ormDriver = "GORM (Go Object Relational Mapping)"
					}
				} else if strings.Contains(sContent, "postgres") || strings.Contains(sContent, "lib/pq") || strings.Contains(sContent, "pgx") {
					dbType = "PostgreSQL"
					if strings.Contains(sContent, "gorm") {
						ormDriver = "GORM (Go Object Relational Mapping)"
					}
				} else if strings.Contains(sContent, "sqlite") {
					dbType = "SQLite"
					if strings.Contains(sContent, "gorm") {
						ormDriver = "GORM (Go Object Relational Mapping)"
					}
				}

				if dbName == "" {
					reDB := regexp.MustCompile(`dbName\s*:=\s*os\.Getenv\("DB_NAME"\)\s*\n\s*if\s*dbName\s*==\s*""\s*\{\s*dbName\s*=\s*"([^"]+)"`)
					m := reDB.FindStringSubmatch(sContent)
					if len(m) > 1 {
						dbName = m[1]
					}
				}
			}
			break
		}
	}

	// Detect DB from manifest
	if fileExists(filepath.Join(dir, "go.mod")) {
		modContent, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
		sMod := string(modContent)
		if strings.Contains(sMod, "gorm.io/gorm") {
			ormDriver = "GORM (Go Object Relational Mapping)"
		}
		if (strings.Contains(sMod, "gorm.io/driver/mysql") || strings.Contains(sMod, "go-sql-driver/mysql")) && dbType == "None detected" {
			dbType = "MySQL"
		}
		if (strings.Contains(sMod, "gorm.io/driver/postgres") || strings.Contains(sMod, "lib/pq") || strings.Contains(sMod, "pgx")) && dbType == "None detected" {
			dbType = "PostgreSQL"
		}
		if strings.Contains(sMod, "sqlite") && dbType == "None detected" {
			dbType = "SQLite"
		}
	} else if fileExists(filepath.Join(dir, "prisma/schema.prisma")) {
		ormDriver = "Prisma ORM"
		prismaContent, _ := os.ReadFile(filepath.Join(dir, "prisma/schema.prisma"))
		if strings.Contains(string(prismaContent), `provider = "postgresql"`) {
			dbType = "PostgreSQL"
		} else if strings.Contains(string(prismaContent), `provider = "mysql"`) {
			dbType = "MySQL"
		} else if strings.Contains(string(prismaContent), `provider = "sqlite"`) {
			dbType = "SQLite"
		}
	}

	// Detect Auth methods
	if len(envInfo.AuthMethods) == 0 {
		if fileExists(filepath.Join(dir, "server/api/auth.ts")) || fileExists(filepath.Join(dir, "server/api/auth/[...].ts")) {
			envInfo.AuthMethods = append(envInfo.AuthMethods, "Nuxt Auth / OAuth")
			envInfo.LoginEndpoints = append(envInfo.LoginEndpoints, "/api/auth/login")
		} else {
			envInfo.AuthMethods = append(envInfo.AuthMethods, "JWT Bearer Token")
		}
	}

	// Detect if this is a Frontend component
	isFrontend := fileExists(filepath.Join(dir, "package.json")) ||
		fileExists(filepath.Join(dir, "nuxt.config.ts")) ||
		fileExists(filepath.Join(dir, "nuxt.config.js")) ||
		fileExists(filepath.Join(dir, "next.config.js")) ||
		fileExists(filepath.Join(dir, "pubspec.yaml"))

	if isFrontend && (dbType == "None detected" || dbType == "") {
		dbType = "Client Cache & Remote REST API"
		ormDriver = "Typed HTTP Composables ($fetch / useApi)"
		dbName = "Client Reactive State"
		envInfo.AuthMethods = []string{"Client Bearer Token Storage", "Route Navigation Guard Middleware", "Reactive Session State"}
		envInfo.LoginEndpoints = []string{"/login (Form UI)", "/auth/login"}
		envInfo.LoginAndAccessGuide = "Aplikasi frontend mengelola token otentikasi di client storage/cookies dan menerapkan Route Navigation Guards sebelum mengakses halaman terlindungi."
	}

	if isFrontend {
		pkgPath := filepath.Join(dir, "package.json")
		if fileExists(pkgPath) {
			data, err := os.ReadFile(pkgPath)
			if err == nil {
				var pkg packageJSON
				if json.Unmarshal(data, &pkg) == nil {
					hasDepLocal := func(name string) bool {
						if pkg.Dependencies != nil {
							if _, ok := pkg.Dependencies[name]; ok {
								return true
							}
						}
						if pkg.DevDependencies != nil {
							if _, ok := pkg.DevDependencies[name]; ok {
								return true
							}
						}
						return false
					}

					caps := []string{}
					if hasDepLocal("pinia") || hasDepLocal("@pinia/nuxt") {
						caps = append(caps, "State Management (Pinia)")
						envInfo.DynamicMetadata["State Management"] = "Pinia Reactive Store"
					}
					if hasDepLocal("tailwindcss") || hasDepLocal("@nuxtjs/tailwindcss") {
						caps = append(caps, "Responsive Layout (Tailwind CSS)")
						envInfo.DynamicMetadata["Styling Engine"] = "Tailwind CSS Utility-First"
					}
					if hasDepLocal("@nuxt/ui") {
						caps = append(caps, "Nuxt UI Design System")
						envInfo.DynamicMetadata["UI Framework"] = "Nuxt UI Component Suite"
					}
					if hasDepLocal("zod") {
						caps = append(caps, "Schema Validation (Zod)")
					}
					if hasDepLocal("chart.js") || hasDepLocal("vue-chartjs") {
						caps = append(caps, "Interactive Charts & Data Visualization")
					}
					if hasDepLocal("@vueuse/core") {
						caps = append(caps, "Vue Composition Utilities (@vueuse)")
					}
					caps = append(caps, "Route Navigation Guards")
					caps = append(caps, "Double Security Censored Keys Vault")

					envInfo.DynamicMetadata["Frontend Capabilities"] = strings.Join(caps, ", ")
					if pkg.PackageManager != "" {
						envInfo.DynamicMetadata["Package Manager"] = pkg.PackageManager
					}
				}
			}
		}
	}

	if len(envInfo.LoginEndpoints) == 0 {
		envInfo.LoginEndpoints = []string{"/api/auth/login", "/api/auth/register", "/api/auth/google"}
	}

	envInfo.DatabaseType = dbType
	envInfo.ORMDriver = ormDriver
	envInfo.DatabaseName = dbName
	if envInfo.LoginAndAccessGuide == "" {
		envInfo.LoginAndAccessGuide = fmt.Sprintf("Aplikasi menggunakan autentikasi %s. Endpoint login dan registrasi publik berada di %s.", strings.Join(envInfo.AuthMethods, ", "), strings.Join(envInfo.LoginEndpoints, ", "))
	}
	envInfo.DynamicMetadata["Arsitektur Database"] = fmt.Sprintf("%s (%s)", dbType, ormDriver)
	if dbName != "" {
		envInfo.DynamicMetadata["Database Name"] = dbName
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

// extractKeyFiles: cuplikan berkas manifest/konfigurasi. withCode = sertakan kode rute/controller (hanya bila
// rute tidak bisa diekstrak statis); tanpa itu cukup 60 baris pertama per berkas.
func extractKeyFiles(dir string, withCode bool) string {
	var sb strings.Builder
	targets := []string{
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

	for _, t := range targets {
		p := filepath.Join(dir, t)
		if fileExists(p) {
			maxLines := 60
			switch {
			case withCode && (strings.HasSuffix(t, "main.go") || strings.Contains(t, "routes") || strings.Contains(t, "Routes") || strings.Contains(t, "useApi")):
				maxLines = 400
			case strings.Contains(t, "database"):
				maxLines = 150
			}
			content, err := readSnippet(p, maxLines)
			if err == nil && content != "" {
				sb.WriteString(fmt.Sprintf("\n--- FILE: %s ---\n%s\n", t, content))
			}
		}
	}

	if !withCode {
		return sb.String()
	}
	// Also extract controller signatures or composables if available
	ctrlDirs := []string{"controllers", "routes", "handlers", "api", "composables", "app/composables", "server/api", "app/Controllers", "app/Controllers/Api", "application/controllers"}
	for _, cd := range ctrlDirs {
		p := filepath.Join(dir, cd)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			entries, _ := os.ReadDir(p)
			count := 0
			for _, e := range entries {
				if e.IsDir() || strings.HasSuffix(e.Name(), "_test.go") || strings.HasSuffix(e.Name(), ".test.ts") {
					continue
				}
				name := e.Name()
				if !strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".js") && !strings.HasSuffix(name, ".php") {
					continue
				}
				count++
				if count > 8 {
					break
				}
				cPath := filepath.Join(p, name)
				cContent, err := readSnippet(cPath, 150)
				if err == nil && cContent != "" {
					sb.WriteString(fmt.Sprintf("\n--- FILE: %s/%s ---\n%s\n", cd, name, cContent))
				}
			}
		}
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

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func readFileString(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return string(b)
}

// detectArchitecture analyzes manifests, go.mod, directory hierarchy, and key files to identify framework, plugins, and design patterns
func detectArchitecture(dir string) models.ArchitectureInfo {
	var arch models.ArchitectureInfo
	arch.Plugins = make([]string, 0)

	// 1. Check Go projects
	goModPath := filepath.Join(dir, "go.mod")
	if fileExists(goModPath) {
		modBytes, err := os.ReadFile(goModPath)
		if err == nil {
			sMod := string(modBytes)

			// Framework Detection
			if strings.Contains(sMod, "github.com/gofiber/fiber/v2") {
				arch.Framework = "Fiber v2"
				arch.FrameworkType = "FastHTTP Asynchronous High-Performance Web Engine"
			} else if strings.Contains(sMod, "github.com/gofiber/fiber/v3") {
				arch.Framework = "Fiber v3"
				arch.FrameworkType = "FastHTTP Asynchronous High-Performance Web Engine"
			} else if strings.Contains(sMod, "github.com/gin-gonic/gin") {
				arch.Framework = "Gin"
				arch.FrameworkType = "High-Performance HTTP Web Framework (HttpRouter Engine)"
			} else if strings.Contains(sMod, "github.com/labstack/echo/v4") || strings.Contains(sMod, "github.com/labstack/echo") {
				arch.Framework = "Echo v4"
				arch.FrameworkType = "Minimalist High-Performance Web Framework"
			} else if strings.Contains(sMod, "github.com/go-chi/chi/v5") || strings.Contains(sMod, "github.com/go-chi/chi") {
				arch.Framework = "Chi Router"
				arch.FrameworkType = "Lightweight, Idiomatic & Composable net/http Router"
			} else if strings.Contains(sMod, "github.com/gorilla/mux") {
				arch.Framework = "Gorilla Mux"
				arch.FrameworkType = "Standard-Compliant net/http URL Router & Dispatcher"
			} else if strings.Contains(sMod, "github.com/zeromicro/go-zero") {
				arch.Framework = "Go-Zero"
				arch.FrameworkType = "Microservice Web & RPC Framework with Built-in Tooling"
			} else if strings.Contains(sMod, "github.com/kataras/iris") {
				arch.Framework = "Iris"
				arch.FrameworkType = "Expressive High-Performance Web Framework"
			} else if strings.Contains(sMod, "github.com/beego/beego") {
				arch.Framework = "Beego"
				arch.FrameworkType = "Full-Stack MVC Framework for Go"
			} else {
				arch.Framework = "Standard net/http"
				arch.FrameworkType = "Go Standard Library HTTP Multiplexer (Go 1.22+ Enhanced)"
			}

			// Plugins & Libraries Detection
			addPlugin := func(name string) {
				for _, p := range arch.Plugins {
					if p == name {
						return
					}
				}
				arch.Plugins = append(arch.Plugins, name)
			}

			if strings.Contains(sMod, "github.com/gofiber/contrib/jwt") || strings.Contains(sMod, "golang-jwt/jwt") || strings.Contains(sMod, "dgrijalva/jwt-go") {
				addPlugin("JWT Authentication (golang-jwt/jwt)")
			}
			if strings.Contains(sMod, "firebase.google.com/go/v4") {
				addPlugin("Firebase Admin SDK (FCM & Push Notifications)")
			}
			if strings.Contains(sMod, "gorm.io/gorm") {
				addPlugin("GORM v2 (Object-Relational Mapping)")
			}
			if strings.Contains(sMod, "gorm.io/driver/mysql") || strings.Contains(sMod, "go-sql-driver/mysql") {
				addPlugin("MySQL Database Driver")
			}
			if strings.Contains(sMod, "gorm.io/driver/postgres") || strings.Contains(sMod, "jackc/pgx") || strings.Contains(sMod, "lib/pq") {
				addPlugin("PostgreSQL Database Driver")
			}
			if strings.Contains(sMod, "gorm.io/driver/sqlite") || strings.Contains(sMod, "mattn/go-sqlite3") {
				addPlugin("SQLite Driver (Offline / Fallback Database)")
			}
			if strings.Contains(sMod, "github.com/jmoiron/sqlx") {
				addPlugin("SQLx (Database Extensions)")
			}
			if strings.Contains(sMod, "go.mongodb.org/mongo-driver") {
				addPlugin("MongoDB Official Driver")
			}
			if strings.Contains(sMod, "github.com/redis/go-redis") || strings.Contains(sMod, "go-redis/redis") {
				addPlugin("Redis Client (Cache & Session Storage)")
			}
			if strings.Contains(sMod, "golang.org/x/crypto") {
				addPlugin("bcrypt / Argon2 Cryptography (golang.org/x/crypto)")
			}
			if strings.Contains(sMod, "github.com/google/uuid") || strings.Contains(sMod, "gofrs/uuid") {
				addPlugin("Google UUID Generator")
			}
			if strings.Contains(sMod, "github.com/joho/godotenv") {
				addPlugin("Godotenv (.env Configuration Loader)")
			}
			if strings.Contains(sMod, "github.com/spf13/viper") {
				addPlugin("Viper (Configuration Management)")
			}
			if strings.Contains(sMod, "github.com/go-playground/validator") {
				addPlugin("Go-Playground Validator")
			}
			if strings.Contains(sMod, "github.com/klauspost/compress") || strings.Contains(sMod, "fasthttp") {
				addPlugin("Klauspost Compress (LevelBestSpeed HTTP Compression)")
			}
			if strings.Contains(sMod, "github.com/gorilla/websocket") {
				addPlugin("Gorilla WebSocket (Bi-directional Streaming)")
			}
			if strings.Contains(sMod, "go.uber.org/zap") {
				addPlugin("Uber Zap (Structured Fast Logger)")
			} else if strings.Contains(sMod, "github.com/sirupsen/logrus") {
				addPlugin("Logrus Structured Logger")
			} else if strings.Contains(sMod, "github.com/rs/zerolog") {
				addPlugin("Zerolog JSON Logger")
			}
			if strings.Contains(sMod, "github.com/swaggo/swag") {
				addPlugin("Swagger / OpenAPI Specification")
			}
			if strings.Contains(sMod, "github.com/prometheus/client_golang") {
				addPlugin("Prometheus Metrics Client")
			}
		}

		// Also check built-in middlewares in main.go or routes
		mainPath := filepath.Join(dir, "main.go")
		if mainBytes, err := os.ReadFile(mainPath); err == nil {
			sMain := string(mainBytes)
			if strings.Contains(sMain, "cors.New") || strings.Contains(sMain, "middleware.CORS") {
				arch.Plugins = append(arch.Plugins, "CORS Security Middleware")
			}
			if strings.Contains(sMain, "logger.New") {
				arch.Plugins = append(arch.Plugins, "Fiber Logger Middleware")
			}
			if strings.Contains(sMain, "recover.New") {
				arch.Plugins = append(arch.Plugins, "Panic Recovery Middleware")
			}
		}

		// Design Pattern Detection
		hasCleanArch := fileExists(filepath.Join(dir, "domain")) &&
			(fileExists(filepath.Join(dir, "usecase")) || fileExists(filepath.Join(dir, "usecases"))) &&
			(fileExists(filepath.Join(dir, "repository")) || fileExists(filepath.Join(dir, "repositories")))

		hasMVC := fileExists(filepath.Join(dir, "controllers")) && fileExists(filepath.Join(dir, "models"))
		has3Tier := fileExists(filepath.Join(dir, "handlers")) && fileExists(filepath.Join(dir, "services"))
		hasStdLayout := fileExists(filepath.Join(dir, "cmd")) && fileExists(filepath.Join(dir, "internal"))

		// Check for E2EE or Zero-Knowledge vault
		hasE2EE := fileExists(filepath.Join(dir, "MEKANISME_ENKRIPSI.md")) ||
			fileExists(filepath.Join(dir, "KEAMANAN_DAN_PEMULIHAN_AKUN.md"))

		if !hasE2EE {
			ctrlPath := filepath.Join(dir, "controllers")
			if cEntries, err := os.ReadDir(ctrlPath); err == nil {
				for _, ce := range cEntries {
					if strings.Contains(ce.Name(), "key") || strings.Contains(ce.Name(), "asset") {
						data, _ := os.ReadFile(filepath.Join(ctrlPath, ce.Name()))
						if strings.Contains(string(data), "encrypted_") || strings.Contains(string(data), "family_key") || strings.Contains(string(data), "AES") {
							hasE2EE = true
							break
						}
					}
				}
			}
		}

		if hasCleanArch {
			arch.DesignPattern = "Clean Architecture (Hexagonal / Ports & Adapters)"
			arch.PatternDetails = "Memisahkan software menjadi 4 lapisan independen: Domain (entitas bisnis), Use Cases (aturan aplikasi spesifik), Repositories (abstraksi database), dan Delivery / HTTP Handlers. Sangat modular, minim coupling, dan memudahkan testing otomatis secara terisolasi."
		} else if hasMVC {
			if hasE2EE {
				arch.DesignPattern = "Headless MVC with Zero-Knowledge E2EE Vault"
				arch.PatternDetails = "Kombinasi Headless Model-Controller dengan Zero-Knowledge End-to-End Encryption (E2EE) Vault. Server bertindak sebagai perantara sinkronisasi dan penyimpanan payload cipher (.enc), di mana seluruh kunci privat dan enkripsi data dikelola eksklusif pada sisi klien. Model mengorganisir entitas database & delta-sync, Controller menangani orkestrasi HTTP dan validasi, serta Middleware mengamankan rute via JWT."
			} else {
				arch.DesignPattern = "MVC (Model-View-Controller / Model-Controller Pattern)"
				arch.PatternDetails = "Pemisahan tanggung jawab modular: Models menangani skema database dan persistensi ORM, Controllers mengeksekusi logika bisnis dan formatting respons HTTP, Routes mengelola pemetaan URL, serta Middleware menangani otentikasi lintas endpoint."
			}
		} else if has3Tier {
			arch.DesignPattern = "3-Tier Layered Architecture (Handler - Service - Repository)"
			arch.PatternDetails = "Arsitektur berlapis berorientasi service: Handlers mengelola serialisasi request/response HTTP, Services mengisolasi proses bisnis utama, dan Repositories bertanggung jawab atas query database serta transaksi data."
		} else if hasStdLayout {
			arch.DesignPattern = "Standard Go Project Layout (cmd/internal/pkg)"
			arch.PatternDetails = "Mengikuti arsitektur idiomatik Standard Go: cmd/ berisi file entrypoint aplikasi, internal/ memuat seluruh kode privat yang terlindungi dari import luar, dan pkg/ memuat pustaka yang dapat digunakan kembali."
		} else {
			arch.DesignPattern = "Modular Flat / Package-Oriented Architecture"
			arch.PatternDetails = "Struktur paket terorganisir per domain fungsional dengan dependensi langsung pada standar pustaka Go untuk throughput maksimal."
		}

		return arch
	}

	// 2. Check Node / Nuxt / Next / Vue / React / Svelte / Angular projects
	pkgJsonPath := filepath.Join(dir, "package.json")
	if fileExists(pkgJsonPath) {
		return detectNodeAndFrontendArchitecture(dir, pkgJsonPath)
	}

	// 3. Check Flutter / Dart projects
	pubspecPath := filepath.Join(dir, "pubspec.yaml")
	if fileExists(pubspecPath) {
		return detectFlutterArchitecture(dir, pubspecPath)
	}

	// 4. Check PHP projects (CodeIgniter, Laravel, etc.)
	composerPath := filepath.Join(dir, "composer.json")
	composerContent := ""
	if fileExists(composerPath) {
		composerContent = readFileString(composerPath)
	}

	if fileExists(filepath.Join(dir, "spark")) || fileExists(filepath.Join(dir, "app/Config/Routes.php")) || strings.Contains(composerContent, "codeigniter4") {
		arch.Framework = "CodeIgniter 4"
		arch.FrameworkType = "Lightweight High-Performance PHP MVC Framework"
		arch.DesignPattern = "MVC (Model-View-Controller)"
		arch.PatternDetails = "Struktur MVC modular: Controllers menangani orkestrasi HTTP dan validasi, Models mengorganisir entitas database dan migrasi via Spark CLI, serta Filters mengamankan rute via JWT dan Middleware."
		return arch
	}

	if fileExists(filepath.Join(dir, "application/config/routes.php")) || fileExists(filepath.Join(dir, "system/core/CodeIgniter.php")) {
		arch.Framework = "CodeIgniter 3"
		arch.FrameworkType = "Classic PHP MVC Framework"
		arch.DesignPattern = "MVC (Model-View-Controller)"
		arch.PatternDetails = "Arsitektur MVC klasik CodeIgniter: Controllers mengeksekusi request HTTP, Models menangani persistensi Active Record, dan Views merender antarmuka."
		return arch
	}

	if fileExists(filepath.Join(dir, "artisan")) || strings.Contains(composerContent, "laravel") {
		arch.Framework = "Laravel"
		arch.FrameworkType = "Fullstack PHP Framework (Eloquent ORM & Artisan CLI)"
		arch.DesignPattern = "MVC (Model-View-Controller)"
		arch.PatternDetails = "Arsitektur MVC modern Laravel: Eloquent Models mengelola relasi database, Controllers mengelola alur bisnis, Routes memetakan endpoint REST, serta Middleware menangani autentikasi & otorisasi."
		return arch
	}

	// Fallback generic
	arch.Framework = "Custom / Standard"
	arch.FrameworkType = "Modular Application"
	arch.DesignPattern = "Component-Based Architecture"
	arch.PatternDetails = "Aplikasi modular mandiri."
	return arch
}

type packageJSON struct {
	Name            string            `json:"name"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	PackageManager  string            `json:"packageManager"`
}

func detectNodeAndFrontendArchitecture(dir, pkgPath string) models.ArchitectureInfo {
	var arch models.ArchitectureInfo
	arch.Plugins = make([]string, 0)

	addPlugin := func(name string) {
		for _, p := range arch.Plugins {
			if p == name {
				return
			}
		}
		arch.Plugins = append(arch.Plugins, name)
	}

	data, err := os.ReadFile(pkgPath)
	var pkg packageJSON
	if err == nil {
		_ = json.Unmarshal(data, &pkg)
	}

	hasDep := func(name string) bool {
		if pkg.Dependencies != nil {
			if _, ok := pkg.Dependencies[name]; ok {
				return true
			}
		}
		if pkg.DevDependencies != nil {
			if _, ok := pkg.DevDependencies[name]; ok {
				return true
			}
		}
		return false
	}

	isNuxt := hasDep("nuxt") || fileExists(filepath.Join(dir, "nuxt.config.ts")) || fileExists(filepath.Join(dir, "nuxt.config.js"))
	isNext := hasDep("next") || fileExists(filepath.Join(dir, "next.config.js")) || fileExists(filepath.Join(dir, "next.config.mjs")) || fileExists(filepath.Join(dir, "next.config.ts"))
	isSvelte := hasDep("@sveltejs/kit") || hasDep("svelte")
	isVue := hasDep("vue")
	isReact := hasDep("react")
	isAngular := hasDep("@angular/core")

	if isNuxt {
		nuxtVer := "Nuxt 3"
		if v, ok := pkg.Dependencies["nuxt"]; ok && strings.Contains(v, "4.") {
			nuxtVer = "Nuxt 4"
		} else if v, ok := pkg.DevDependencies["nuxt"]; ok && strings.Contains(v, "4.") {
			nuxtVer = "Nuxt 4"
		}
		arch.Framework = nuxtVer + " (Vue 3.5+)"
		arch.FrameworkType = "Vue 3 Universal Fullstack SSR & Nitro Web Engine"
	} else if isNext {
		arch.Framework = "Next.js (React 18/19)"
		arch.FrameworkType = "React Fullstack SSR & React Server Components (Turbopack Engine)"
	} else if isSvelte {
		arch.Framework = "SvelteKit"
		arch.FrameworkType = "Compiling Reactive Web Framework (Vite Bundler)"
	} else if isVue {
		arch.Framework = "Vue 3 SPA"
		arch.FrameworkType = "Declarative Component-Based SPA (Vite Bundler)"
	} else if isReact {
		arch.Framework = "React SPA"
		arch.FrameworkType = "Component-Based Reactive SPA (Vite/Webpack Bundler)"
	} else if isAngular {
		arch.Framework = "Angular"
		arch.FrameworkType = "Enterprise TypeScript Web Platform"
	} else {
		arch.Framework = "Node.js Web App"
		arch.FrameworkType = "JavaScript / TypeScript Application"
	}

	// Detect UI Toolkits & Design Systems
	if hasDep("@nuxt/ui") {
		addPlugin("Nuxt UI (Design System & Headless Components)")
	}
	if hasDep("tailwindcss") || hasDep("@nuxtjs/tailwindcss") {
		addPlugin("Tailwind CSS (Utility-First Design Engine)")
	}
	if hasDep("unocss") {
		addPlugin("UnoCSS (Instant Atomic CSS Engine)")
	}
	if hasDep("primevue") {
		addPlugin("PrimeVue Component Suite")
	}
	if hasDep("vuetify") {
		addPlugin("Vuetify Material Design 3 Framework")
	}
	if hasDep("radix-vue") || hasDep("shadcn-vue") || hasDep("@radix-ui/react-dialog") {
		addPlugin("Radix / Shadcn UI Primitives")
	}
	if hasDep("element-plus") {
		addPlugin("Element Plus Desktop UI Kit")
	}
	if hasDep("ant-design-vue") || hasDep("antd") {
		addPlugin("Ant Design Enterprise Kit")
	}
	if hasDep("daisyui") {
		addPlugin("DaisyUI Semantic Components")
	}
	if hasDep("@headlessui/vue") || hasDep("@headlessui/react") {
		addPlugin("Headless UI Accessible Components")
	}

	// Detect Vector Iconography
	if hasDep("@iconify-json/lucide") || hasDep("lucide-vue-next") || hasDep("lucide-react") {
		addPlugin("Lucide Icons (Vector Iconography)")
	}
	if hasDep("@iconify-json/simple-icons") {
		addPlugin("Simple Icons (Brand & Tech Vectors)")
	}
	if hasDep("@heroicons/vue") || hasDep("@heroicons/react") {
		addPlugin("Heroicons SVG Icons")
	}

	// Detect State Management
	if hasDep("pinia") || hasDep("@pinia/nuxt") {
		addPlugin("Pinia (Reactive Central Store & Persistence)")
	}
	if hasDep("vuex") {
		addPlugin("Vuex Centralized Store")
	}
	if hasDep("@reduxjs/toolkit") || hasDep("redux") {
		addPlugin("Redux Toolkit State Container")
	}
	if hasDep("zustand") {
		addPlugin("Zustand Lightweight State Manager")
	}

	// Detect HTTP & Communication
	if hasDep("ofetch") {
		addPlugin("OFetch (Resilient HTTP Client)")
	}
	if hasDep("axios") {
		addPlugin("Axios HTTP Client with Interceptors")
	}
	if hasDep("@tanstack/vue-query") || hasDep("@tanstack/react-query") {
		addPlugin("TanStack Query (Async Cache & Sync)")
	}
	if hasDep("@trpc/client") {
		addPlugin("tRPC TypeSafe API Client")
	}
	if hasDep("socket.io-client") {
		addPlugin("Socket.IO Real-Time WebSocket Client")
	}

	// Detect Validation & Forms
	if hasDep("zod") {
		addPlugin("Zod (TypeScript-First Schema Validation)")
	}
	if hasDep("vee-validate") {
		addPlugin("VeeValidate Form Validation")
	}
	if hasDep("@formkit/vue") {
		addPlugin("FormKit Form Builder")
	}

	// Detect Visualization & Utilities
	if hasDep("chart.js") || hasDep("vue-chartjs") {
		addPlugin("Chart.js (Interactive Data Visualization)")
	}
	if hasDep("echarts") || hasDep("vue-echarts") {
		addPlugin("Apache ECharts Visualizations")
	}
	if hasDep("apexcharts") || hasDep("vue3-apexcharts") {
		addPlugin("ApexCharts Modern SVG Charts")
	}
	if hasDep("@vueuse/core") {
		addPlugin("VueUse (Composition API Utilities)")
	}
	if hasDep("date-fns") || hasDep("dayjs") {
		addPlugin("Dayjs / Date-fns Timezone Formatter")
	}

	// Detect Tooling & Quality
	if hasDep("typescript") || hasDep("vue-tsc") {
		addPlugin("TypeScript Strict Typing Engine")
	}
	if hasDep("eslint") || hasDep("@nuxt/eslint") {
		addPlugin("ESLint Code Quality & Hygiene")
	}
	if hasDep("vitest") {
		addPlugin("Vitest Unit Testing Framework")
	}
	if hasDep("@playwright/test") {
		addPlugin("Playwright End-to-End Automation")
	}

	// Design Pattern Detection based on directory hierarchy
	hasPages := fileExists(filepath.Join(dir, "pages")) || fileExists(filepath.Join(dir, "app/pages"))
	hasComposables := fileExists(filepath.Join(dir, "composables")) || fileExists(filepath.Join(dir, "app/composables"))
	hasComponents := fileExists(filepath.Join(dir, "components")) || fileExists(filepath.Join(dir, "app/components"))
	hasServerApi := fileExists(filepath.Join(dir, "server/api")) || fileExists(filepath.Join(dir, "server/routes"))
	hasMiddleware := fileExists(filepath.Join(dir, "middleware")) || fileExists(filepath.Join(dir, "app/middleware"))

	if isNuxt && hasPages && hasComposables {
		arch.DesignPattern = "Vue 3 Composition API with Feature-Driven Architecture & File-Based Routing"
		arch.PatternDetails = "Arsitektur frontend modern berbasis konvensi direktori: app/pages/ untuk routing file deklaratif, app/composables/ untuk pemanggilan API terisolasi (useApi) dan reusable state, app/components/ untuk antarmuka atomik & modular (Nuxt UI/Tailwind), serta middleware untuk proteksi rute navigasi klien."
	} else if isNext {
		arch.DesignPattern = "Next.js App Router Architecture (React Server Components)"
		arch.PatternDetails = "Arsitektur modular React berbasis Server Components, Client Hooks, dan Server Actions dengan rute direktori app/."
	} else if hasComposables || hasComponents {
		arch.DesignPattern = "Component-Driven Reactive SPA Architecture"
		arch.PatternDetails = "Pemisahan lapisan presentasi deklaratif dengan manajemen state reaktif, isolasi komponen visual, dan pemanggilan API terstruktur."
	} else {
		arch.DesignPattern = "Modular Frontend Web Architecture"
		arch.PatternDetails = "Struktur paket terorganisir per modul fungsional antarmuka."
	}

	if hasServerApi {
		arch.PatternDetails += " Dilengkapi Nitro Serverless Engine (server/api) untuk endpoint API fullstack hybrid."
	}
	if hasMiddleware {
		arch.PatternDetails += " Memanfaatkan Route Navigation Guards untuk otorisasi akses sebelum pergantian halaman."
	}

	return arch
}

func detectFlutterArchitecture(dir, pubspecPath string) models.ArchitectureInfo {
	var arch models.ArchitectureInfo
	arch.Framework = "Flutter (Dart SDK)"
	arch.FrameworkType = "Cross-Platform Declarative Mobile/Web UI Toolkit"
	arch.Plugins = []string{"Flutter SDK", "Dart SDK", "Material Design 3"}

	content, err := os.ReadFile(pubspecPath)
	if err == nil {
		s := string(content)
		if strings.Contains(s, "flutter_bloc") || strings.Contains(s, "bloc:") {
			arch.Plugins = append(arch.Plugins, "BLoC Pattern State Management")
			arch.DesignPattern = "BLoC (Business Logic Component) Architecture"
			arch.PatternDetails = "Pemisahan tegas antara UI Presentasi, Business Logic (BLoC Events/States), dan Data Repositories."
		} else if strings.Contains(s, "flutter_riverpod") || strings.Contains(s, "riverpod:") {
			arch.Plugins = append(arch.Plugins, "Riverpod Reactive State Management")
			arch.DesignPattern = "Riverpod Unidirectional Data Flow Architecture"
			arch.PatternDetails = "State management reaktif compile-time safe dengan dependency injection deklaratif."
		} else if strings.Contains(s, "provider:") {
			arch.Plugins = append(arch.Plugins, "Provider State Management")
			arch.DesignPattern = "Provider & ChangeNotifier Architecture"
			arch.PatternDetails = "Pengelolaan state deklaratif berbasis inherited widgets dan model notifikasi perubahan."
		} else if strings.Contains(s, "get:") {
			arch.Plugins = append(arch.Plugins, "GetX Ecosystem")
			arch.DesignPattern = "GetX Reactive Micro-Framework"
			arch.PatternDetails = "Navigasi tanpa konteks, dependency injection ringan, dan reactive state management."
		} else {
			arch.DesignPattern = "Clean Architecture for Flutter (lib/domain/data/presentation)"
			arch.PatternDetails = "Pemisahan UI deklaratif dengan state management dan repository layer pada direktori lib/."
		}

		if strings.Contains(s, "dio:") {
			arch.Plugins = append(arch.Plugins, "Dio HTTP Networking Client")
		}
		if strings.Contains(s, "go_router:") {
			arch.Plugins = append(arch.Plugins, "GoRouter Declarative Routing")
		}
		if strings.Contains(s, "hive:") || strings.Contains(s, "isar:") {
			arch.Plugins = append(arch.Plugins, "Hive / Isar Fast Local NoSQL Database")
		}
		if strings.Contains(s, "shared_preferences:") {
			arch.Plugins = append(arch.Plugins, "SharedPreferences Key-Value Storage")
		}
	}

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
