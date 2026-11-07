package guidelines

import (
	"io/fs"

	"encoding/json"
	"fmt"
	vibeasik "github.com/vibeasik/vibeasik"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibeasik/vibeasik/internal/models"
)

// GuidelineItem represents a single actionable rule/rubric
type GuidelineItem struct {
	ID              string `json:"id"`
	Level           string `json:"level"` // global, project, component
	Title           string `json:"title"`
	Rule            string `json:"rule"`
	Rationale       string `json:"rationale"`
	InspectionCheck string `json:"inspection_check"`
	SuggestedAction string `json:"suggested_action"`
	Severity        string `json:"severity"` // critical, high, medium, low
}

// GuidelineCategory represents a grouped collection of guidelines (e.g. 01_perencanaan, 02_arsitektur)
type GuidelineCategory struct {
	Category    string          `json:"category"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	FilePath    string          `json:"file_path,omitempty"`
	Guidelines  []GuidelineItem `json:"guidelines"`
}

// FindGuidelinesDir locates the guidelines folder dynamically
func FindGuidelinesDir() string {
	candidates := []string{
		"guidelines",
		"../guidelines",
		"../../guidelines",
	}

	// Override pengguna: rule kustom di ~/.vibeasik/guidelines menggantikan rule bawaan.
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".vibeasik", "guidelines"))
	}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "guidelines"),
			filepath.Join(cwd, "..", "guidelines"),
		)
	}

	for _, c := range candidates {
		if stat, err := os.Stat(c); err == nil && stat.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}

	return ""
}

// LoadAllGuidelines reads and parses all JSON files from the guidelines folder
func LoadAllGuidelines() ([]GuidelineCategory, error) {
	dir := FindGuidelinesDir()
	if dir == "" {
		return loadEmbeddedGuidelines()
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read guidelines dir: %w", err)
	}

	var categories []GuidelineCategory
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			fullPath := filepath.Join(dir, f.Name())
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}

			var cat GuidelineCategory
			if err := json.Unmarshal(content, &cat); err == nil {
				cat.FilePath = fullPath
				categories = append(categories, cat)
			}
		}
	}

	return categories, nil
}

// loadEmbeddedGuidelines memuat rule bawaan yang disematkan di binary (vibeasik.Guidelines).
func loadEmbeddedGuidelines() ([]GuidelineCategory, error) {
	files, err := fs.ReadDir(vibeasik.Guidelines, "guidelines")
	if err != nil {
		return nil, fmt.Errorf("embedded guidelines unavailable: %w", err)
	}
	var categories []GuidelineCategory
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		content, err := fs.ReadFile(vibeasik.Guidelines, "guidelines/"+f.Name())
		if err != nil {
			continue
		}
		var cat GuidelineCategory
		if err := json.Unmarshal(content, &cat); err == nil {
			cat.FilePath = "embedded:guidelines/" + f.Name()
			categories = append(categories, cat)
		}
	}
	return categories, nil
}

// GenerateSuggestionsForProject inspects a project bundle and derives 3-tier Todo suggestions based on guidelines
func GenerateSuggestionsForProject(project *models.Project, allProjects []models.Project) []models.Todo {
	var suggestions []models.Todo

	if project == nil {
		return suggestions
	}

	var compNames []string
	var beComp *models.Component
	var feComp *models.Component

	for i := range project.Components {
		c := &project.Components[i]
		compNames = append(compNames, c.Name)
		if c.Role == models.RoleBackend && beComp == nil {
			beComp = c
		}
		if strings.HasPrefix(c.Role, "frontend") && feComp == nil {
			feComp = c
		}
	}

	allAppsLabel := fmt.Sprintf("Seluruh Aplikasi (Bundle %s)", project.Name)
	if len(compNames) > 0 {
		allAppsLabel = fmt.Sprintf("Seluruh Aplikasi (%d Apps: %s)", len(compNames), strings.Join(compNames, ", "))
	}

	// 1. ARCH-002 (dev.sh) & DEV-007 (deploy.sh): saran todo dari audit Aturan 45, diagnostics.DefaultProjectTodos
	//    dipanggil POST /api/todos/suggest (paket diagnostics mengimpor guidelines, jadi tidak di sini).

	// 2. Level Project: Periksa hygiene .env.example (SEC-002) - Category: Security
	if project.RootPath != "" && project.RootPath != "-" {
		envPath := filepath.Join(project.RootPath, ".env")
		examplePath := filepath.Join(project.RootPath, ".env.example")
		if _, err := os.Stat(envPath); err == nil {
			if _, err := os.Stat(examplePath); os.IsNotExist(err) {
				suggestions = append(suggestions, models.Todo{
					Level:          models.TodoLevelProject,
					Category:       models.CategorySecurity,
					ProjectID:      &project.ID,
					ProjectName:    project.Name,
					TargetApps:     allAppsLabel,
					Title:          fmt.Sprintf("Buat berkas template .env.example tersanitasi untuk %s", project.Name),
					Description:    "Dokumentasikan seluruh variabel lingkungan yang dibutuhkan proyek ke dalam .env.example dengan nilai dummy.",
					Status:         models.TodoStatusSuggested,
					Priority:       models.TodoPriorityHigh,
					CreatorType:    models.CreatorAI,
					CreatorName:    "vibeasik-ai-advisor",
					IsAISuggestion: true,
					GuidelineRef:   "SEC-002",
					Rationale:      "Patokan: [SEC-002] Berkas .env wajib memiliki pasangan .env.example tersanitasi tanpa kredensial asli.",
					Source:         "ai_agent",
				})
			}
		}

		// Security Check: .gitignore .env protection
		gitignorePath := filepath.Join(project.RootPath, ".gitignore")
		if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
			suggestions = append(suggestions, models.Todo{
				Level:          models.TodoLevelProject,
				Category:       models.CategorySecurity,
				ProjectID:      &project.ID,
				ProjectName:    project.Name,
				TargetApps:     allAppsLabel,
				Title:          fmt.Sprintf("Pasang .gitignore untuk mencegah kebocoran secret pada %s", project.Name),
				Description:    "Tambahkan .gitignore di root proyek dengan aturan mengabaikan .env, node_modules, log, dan kredensial lokal.",
				Status:         models.TodoStatusSuggested,
				Priority:       models.TodoPriorityCritical,
				CreatorType:    models.CreatorAI,
				CreatorName:    "vibeasik-ai-advisor",
				IsAISuggestion: true,
				GuidelineRef:   "SEC-001",
				Rationale:      "Patokan: [SEC-001] Repositori wajib memiliki .gitignore yang memfilter .env dan secrets agar tidak ter-commit ke remote git.",
				Source:         "ai_agent",
			})
		}

		// Feature Check: AI Context File (AI-002)
		aiContextPath := filepath.Join(project.RootPath, "AI_CONTEXT.md")
		if _, err := os.Stat(aiContextPath); os.IsNotExist(err) {
			suggestions = append(suggestions, models.Todo{
				Level:          models.TodoLevelProject,
				Category:       models.CategoryFeature,
				ProjectID:      &project.ID,
				ProjectName:    project.Name,
				TargetApps:     allAppsLabel,
				Title:          fmt.Sprintf("Sediakan berkas AI_CONTEXT.md untuk panduan AI Agent pada %s", project.Name),
				Description:    "Dokumentasikan arsitektur bundle, konvensi kode, peran komponen, dan perintah run agar coding agent (Antigravity/Cursor) bekerja presisi.",
				Status:         models.TodoStatusSuggested,
				Priority:       models.TodoPriorityMedium,
				CreatorType:    models.CreatorAI,
				CreatorName:    "vibeasik-ai-advisor",
				IsAISuggestion: true,
				GuidelineRef:   "AI-002",
				Rationale:      "Patokan: [AI-002] File AI_CONTEXT.md mempercepat konteks agent otonom hingga 80% dan mengeliminasi halusinasi refactoring.",
				Source:         "ai_agent",
			})
		}
	}

	// 3. Security Cross-Component Check: CORS & Origin Whitelist (SEC-004)
	if beComp != nil && feComp != nil {
		suggestions = append(suggestions, models.Todo{
			Level:          models.TodoLevelProject,
			Category:       models.CategorySecurity,
			ProjectID:      &project.ID,
			ProjectName:    project.Name,
			TargetApps:     fmt.Sprintf("%s & %s [2 Apps]", beComp.Name, feComp.Name),
			Title:          fmt.Sprintf("Konfigurasi CORS Whitelist & Proteksi Origin antara %s dan %s", beComp.Name, feComp.Name),
			Description:    fmt.Sprintf("Pastikan CORS pada %s hanya mengizinkan origin lokal %s secara eksplisit dengan AllowCredentials: true, hindari penggunaan wildcard '*'.", beComp.Name, feComp.Name),
			Status:         models.TodoStatusSuggested,
			Priority:       models.TodoPriorityHigh,
			CreatorType:    models.CreatorAI,
			CreatorName:    "vibeasik-ai-advisor",
			IsAISuggestion: true,
			GuidelineRef:   "SEC-004",
			Rationale:      "Patokan: [SEC-004] CORS wajib dibatasi hanya ke domain/origin resmi frontend yang terdaftar untuk mencegah cross-origin request forgery.",
			Source:         "ai_agent",
		})
	}

	// NET-001 (Aturan 35/45): blok port & run_port dialokasikan engine otomatis (runner/slots.go), bukan todo.

	// 4. Level Component: Periksa setiap aplikasi teknis individual
	for i := range project.Components {
		comp := &project.Components[i]
		singleAppLabel := fmt.Sprintf("%s [1 App]", comp.Name)

		// Pemeriksaan API-001: Health check probe - Category: Backend
		if comp.Role == models.RoleBackend {
			hasHealth := false
			if comp.HealthURL != "" {
				hasHealth = true
			}
			for _, r := range comp.Routes {
				if strings.Contains(r.Path, "/health") || strings.Contains(r.Path, "/ping") {
					hasHealth = true
					break
				}
			}

			if !hasHealth {
				suggestions = append(suggestions, models.Todo{
					Level:          models.TodoLevelComponent,
					Category:       models.CategoryBackend,
					ProjectID:      &project.ID,
					ProjectName:    project.Name,
					ComponentID:    &comp.ID,
					ComponentName:  comp.Name,
					TargetApps:     singleAppLabel,
					Title:          fmt.Sprintf("Tambahkan endpoint GET /health pada %s", comp.Name),
					Description:    fmt.Sprintf("Pasang endpoint health probe bebas autentikasi pada port :%d yang mengembalikan JSON {'status': 'ok'}.", comp.Port),
					Status:         models.TodoStatusSuggested,
					Priority:       models.TodoPriorityHigh,
					CreatorType:    models.CreatorAI,
					CreatorName:    "vibeasik-ai-advisor",
					IsAISuggestion: true,
					GuidelineRef:   "API-001",
					Rationale:      "Patokan: [API-001] Health probe wajib tersedia agar Vibeasik dan reverse proxy (Caddy/Docker) dapat memantau status liveness servis.",
					Source:         "ai_agent",
				})
			}

			// Security Check: Rate Limiting (SEC-005) - Category: Security
			suggestions = append(suggestions, models.Todo{
				Level:          models.TodoLevelComponent,
				Category:       models.CategorySecurity,
				ProjectID:      &project.ID,
				ProjectName:    project.Name,
				ComponentID:    &comp.ID,
				ComponentName:  comp.Name,
				TargetApps:     singleAppLabel,
				Title:          fmt.Sprintf("Pasang Middleware Rate Limiting & Anti Brute-Force pada %s", comp.Name),
				Description:    fmt.Sprintf("Terapkan rate limiting pada %s (maksimal 100 req/menit untuk API umum, dan 5 req/menit untuk endpoint autentikasi/login).", comp.Name),
				Status:         models.TodoStatusSuggested,
				Priority:       models.TodoPriorityHigh,
				CreatorType:    models.CreatorAI,
				CreatorName:    "vibeasik-ai-advisor",
				IsAISuggestion: true,
				GuidelineRef:   "SEC-005",
				Rationale:      "Patokan: [SEC-005] Proteksi DoS dan brute-force wajib diterapkan di tingkat gateway backend.",
				Source:         "ai_agent",
			})

			// Security Check: HTTP Security Headers (SEC-006) - Category: Security
			suggestions = append(suggestions, models.Todo{
				Level:          models.TodoLevelComponent,
				Category:       models.CategorySecurity,
				ProjectID:      &project.ID,
				ProjectName:    project.Name,
				ComponentID:    &comp.ID,
				ComponentName:  comp.Name,
				TargetApps:     singleAppLabel,
				Title:          fmt.Sprintf("Terapkan HTTP Security Headers (CSP, HSTS, X-Frame-Options) pada %s", comp.Name),
				Description:    "Konfigurasikan security headers (X-Content-Type-Options: nosniff, X-Frame-Options: DENY, Content-Security-Policy) untuk mencegah serangan clickjacking dan MIME-sniffing.",
				Status:         models.TodoStatusSuggested,
				Priority:       models.TodoPriorityMedium,
				CreatorType:    models.CreatorAI,
				CreatorName:    "vibeasik-ai-advisor",
				IsAISuggestion: true,
				GuidelineRef:   "SEC-006",
				Rationale:      "Patokan: [SEC-006] Standar OWASP mengharuskan header keamanan dasar aktif di setiap respons HTTP.",
				Source:         "ai_agent",
			})
		}

		// Pemeriksaan Frontend: UI/UX Composable & Design Hygiene - Category: UI/UX
		if strings.HasPrefix(comp.Role, "frontend") && (comp.Framework == "nuxt" || comp.Framework == "vue") {
			if comp.AbsolutePath != "" {
				compPath := filepath.Join(comp.AbsolutePath, "composables", "useApi.ts")
				if _, err := os.Stat(compPath); os.IsNotExist(err) {
					suggestions = append(suggestions, models.Todo{
						Level:          models.TodoLevelComponent,
						Category:       models.CategoryUIUX,
						ProjectID:      &project.ID,
						ProjectName:    project.Name,
						ComponentID:    &comp.ID,
						ComponentName:  comp.Name,
						TargetApps:     singleAppLabel,
						Title:          fmt.Sprintf("Terapkan composable useApi.ts terpusat pada %s", comp.Name),
						Description:    "Buat composable useApi.ts untuk mengonsentrasikan pemanggilan API backend dengan typed $fetch dan penanganan token otomatis.",
						Status:         models.TodoStatusSuggested,
						Priority:       models.TodoPriorityMedium,
						CreatorType:    models.CreatorAI,
						CreatorName:    "vibeasik-ai-advisor",
						IsAISuggestion: true,
						GuidelineRef:   "FE-001",
						Rationale:      "Patokan: [FE-001] Frontend wajib menggunakan wrapper API client terpusat untuk menjamin konsistensi header token dan endpoint dinamis.",
						Source:         "ai_agent",
					})
				}
			}

			// UI/UX Check: Responsive Mobile & Antislop (FE-002)
			suggestions = append(suggestions, models.Todo{
				Level:          models.TodoLevelComponent,
				Category:       models.CategoryUIUX,
				ProjectID:      &project.ID,
				ProjectName:    project.Name,
				ComponentID:    &comp.ID,
				ComponentName:  comp.Name,
				TargetApps:     singleAppLabel,
				Title:          fmt.Sprintf("Audit Desain Responsif Mobile & Antislop UI pada %s", comp.Name),
				Description:    "Verifikasi breakpoint seluler (sm, md, lg), tap target minimal 44px, serta skema warna low-fatigue yang nyaman di mata pengguna.",
				Status:         models.TodoStatusSuggested,
				Priority:       models.TodoPriorityMedium,
				CreatorType:    models.CreatorAI,
				CreatorName:    "vibeasik-ai-advisor",
				IsAISuggestion: true,
				GuidelineRef:   "FE-002",
				Rationale:      "Patokan: [FE-002] Layout antarmuka harus reflow dengan bersih dari ukuran layar ponsel 375px hingga layar lebar desktop tanpa horizontal overflow.",
				Source:         "ai_agent",
			})

			// UI/UX Check: Empty States & Error Toast Alerts (FE-003)
			suggestions = append(suggestions, models.Todo{
				Level:          models.TodoLevelComponent,
				Category:       models.CategoryUIUX,
				ProjectID:      &project.ID,
				ProjectName:    project.Name,
				ComponentID:    &comp.ID,
				ComponentName:  comp.Name,
				TargetApps:     singleAppLabel,
				Title:          fmt.Sprintf("Sediakan Empty State & Notifikasi Toast pada %s", comp.Name),
				Description:    "Lengkapi tabel dan daftar dengan visual state saat data belum ada, serta berikan toast alert saat request gagal atau offline.",
				Status:         models.TodoStatusSuggested,
				Priority:       models.TodoPriorityMedium,
				CreatorType:    models.CreatorAI,
				CreatorName:    "vibeasik-ai-advisor",
				IsAISuggestion: true,
				GuidelineRef:   "FE-003",
				Rationale:      "Patokan: [FE-003] Antarmuka tidak boleh membiarkan pengguna bingung dengan tampilan kosong tanpa aksi jelas.",
				Source:         "ai_agent",
			})
		}

		// Pemeriksaan TEST-001: Keberadaan file test untuk komponen backend - Category: Architecture
		if comp.Role == models.RoleBackend && comp.AbsolutePath != "" {
			hasTestFiles := false
			testDirs := []string{
				filepath.Join(comp.AbsolutePath, "internal"),
				comp.AbsolutePath,
			}
			for _, tdir := range testDirs {
				if entries, err := os.ReadDir(tdir); err == nil {
					for _, e := range entries {
						if !e.IsDir() && strings.HasSuffix(e.Name(), "_test.go") {
							hasTestFiles = true
							break
						}
					}
				}
				if hasTestFiles {
					break
				}
			}

			if !hasTestFiles {
				suggestions = append(suggestions, models.Todo{
					Level:          models.TodoLevelComponent,
					Category:       models.CategoryArchitecture,
					ProjectID:      &project.ID,
					ProjectName:    project.Name,
					ComponentID:    &comp.ID,
					ComponentName:  comp.Name,
					TargetApps:     singleAppLabel,
					Title:          fmt.Sprintf("Tambahkan unit test untuk business logic pada %s", comp.Name),
					Description:    "Buat file *_test.go untuk layer service dan utility dengan skenario happy path dan edge case. Target cakupan minimum 60%%.",
					Status:         models.TodoStatusSuggested,
					Priority:       models.TodoPriorityCritical,
					CreatorType:    models.CreatorAI,
					CreatorName:    "vibeasik-ai-advisor",
					IsAISuggestion: true,
					GuidelineRef:   "TEST-001",
					Rationale:      "Patokan: [TEST-001] Kode tanpa unit test adalah bom waktu — setiap refactor berpotensi merusak fitur tanpa mekanisme deteksi otomatis.",
					Source:         "ai_agent",
				})
			}
		}

		// Pemeriksaan TEST-001 untuk frontend: Keberadaan folder __tests__ atau file *.test.ts
		if strings.HasPrefix(comp.Role, "frontend") && comp.AbsolutePath != "" {
			hasTestFiles := false
			testsDir := filepath.Join(comp.AbsolutePath, "__tests__")
			if _, err := os.Stat(testsDir); err == nil {
				hasTestFiles = true
			}
			if !hasTestFiles {
				if entries, err := os.ReadDir(comp.AbsolutePath); err == nil {
					for _, e := range entries {
						if !e.IsDir() && (strings.HasSuffix(e.Name(), ".test.ts") || strings.HasSuffix(e.Name(), ".spec.ts")) {
							hasTestFiles = true
							break
						}
					}
				}
			}

			if !hasTestFiles {
				suggestions = append(suggestions, models.Todo{
					Level:          models.TodoLevelComponent,
					Category:       models.CategoryUIUX,
					ProjectID:      &project.ID,
					ProjectName:    project.Name,
					ComponentID:    &comp.ID,
					ComponentName:  comp.Name,
					TargetApps:     singleAppLabel,
					Title:          fmt.Sprintf("Tambahkan component test untuk komponen kritis pada %s", comp.Name),
					Description:    "Buat folder __tests__/ dan file *.test.ts menggunakan @vue/test-utils untuk memverifikasi rendering dan interaksi komponen kritis.",
					Status:         models.TodoStatusSuggested,
					Priority:       models.TodoPriorityMedium,
					CreatorType:    models.CreatorAI,
					CreatorName:    "vibeasik-ai-advisor",
					IsAISuggestion: true,
					GuidelineRef:   "TEST-006",
					Rationale:      "Patokan: [TEST-006] Komponen UI tanpa test rentan terhadap regresi visual dan fungsional saat dependensi diperbarui.",
					Source:         "ai_agent",
				})
			}
		}
	}

	// 5. Level Project: Pemeriksaan DevOps & Deployment
	if project.RootPath != "" && project.RootPath != "-" {
		// DEV-001: Keberadaan Dockerfile - Category: Architecture
		hasDockerfile := false
		for _, name := range []string{"Dockerfile", "dockerfile"} {
			if _, err := os.Stat(filepath.Join(project.RootPath, name)); err == nil {
				hasDockerfile = true
				break
			}
		}
		// Also check component paths for Dockerfile
		if !hasDockerfile {
			for _, comp := range project.Components {
				if comp.AbsolutePath != "" {
					for _, name := range []string{"Dockerfile", "dockerfile"} {
						if _, err := os.Stat(filepath.Join(comp.AbsolutePath, name)); err == nil {
							hasDockerfile = true
							break
						}
					}
				}
				if hasDockerfile {
					break
				}
			}
		}

		if !hasDockerfile && len(project.Components) > 0 {
			suggestions = append(suggestions, models.Todo{
				Level:          models.TodoLevelProject,
				Category:       models.CategoryArchitecture,
				ProjectID:      &project.ID,
				ProjectName:    project.Name,
				TargetApps:     allAppsLabel,
				Title:          fmt.Sprintf("Sediakan Dockerfile multi-stage untuk deployment %s", project.Name),
				Description:    "Buat Dockerfile dengan 2 stage: builder (kompilasi) dan runner (image minimal alpine/distroless) untuk production deployment.",
				Status:         models.TodoStatusSuggested,
				Priority:       models.TodoPriorityHigh,
				CreatorType:    models.CreatorAI,
				CreatorName:    "vibeasik-ai-advisor",
				IsAISuggestion: true,
				GuidelineRef:   "DEV-001",
				Rationale:      "Patokan: [DEV-001] Dockerfile multi-stage memisahkan compiler dari image production, mengecilkan ukuran image dan memperkecil attack surface.",
				Source:         "ai_agent",
			})
		}

		// DEV-002: Keberadaan .dockerignore jika Dockerfile ada - Category: Security
		if hasDockerfile {
			dockerignorePath := filepath.Join(project.RootPath, ".dockerignore")
			if _, err := os.Stat(dockerignorePath); os.IsNotExist(err) {
				suggestions = append(suggestions, models.Todo{
					Level:          models.TodoLevelProject,
					Category:       models.CategorySecurity,
					ProjectID:      &project.ID,
					ProjectName:    project.Name,
					TargetApps:     allAppsLabel,
					Title:          fmt.Sprintf("Buat .dockerignore untuk mencegah kebocoran file ke Docker image pada %s", project.Name),
					Description:    "Tambahkan .dockerignore dengan entri: .git, node_modules, .env, dist/, bin/, *.exe, .vscode/ agar tidak masuk ke Docker build context.",
					Status:         models.TodoStatusSuggested,
					Priority:       models.TodoPriorityHigh,
					CreatorType:    models.CreatorAI,
					CreatorName:    "vibeasik-ai-advisor",
					IsAISuggestion: true,
					GuidelineRef:   "DEV-002",
					Rationale:      "Patokan: [DEV-002] Tanpa .dockerignore, .git dan node_modules masuk ke Docker context, memperlambat build dan berpotensi menyertakan secrets.",
					Source:         "ai_agent",
				})
			}
		}

		// DEV-004: Keberadaan CHANGELOG.md - Category: Feature
		changelogPath := filepath.Join(project.RootPath, "CHANGELOG.md")
		if _, err := os.Stat(changelogPath); os.IsNotExist(err) {
			suggestions = append(suggestions, models.Todo{
				Level:          models.TodoLevelProject,
				Category:       models.CategoryFeature,
				ProjectID:      &project.ID,
				ProjectName:    project.Name,
				TargetApps:     allAppsLabel,
				Title:          fmt.Sprintf("Sediakan CHANGELOG.md untuk dokumentasi riwayat perubahan %s", project.Name),
				Description:    "Buat CHANGELOG.md menggunakan format Keep a Changelog untuk mendokumentasikan fitur baru, bugfix, dan breaking changes per versi rilis.",
				Status:         models.TodoStatusSuggested,
				Priority:       models.TodoPriorityMedium,
				CreatorType:    models.CreatorAI,
				CreatorName:    "vibeasik-ai-advisor",
				IsAISuggestion: true,
				GuidelineRef:   "DEV-004",
				Rationale:      "Patokan: [DEV-004] CHANGELOG memudahkan rollback target dan membantu pengguna mengetahui perubahan tanpa harus membaca git log.",
				Source:         "ai_agent",
			})
		}
	}

	return suggestions
}

// GenerateGlobalSuggestions generates workspace/national level todo suggestions
func GenerateGlobalSuggestions(allProjects []models.Project) []models.Todo {
	var suggestions []models.Todo

	// PLAN-001: Global MCP Observability across workspace - Category: Feature
	suggestions = append(suggestions, models.Todo{
		Level:          models.TodoLevelGlobal,
		Category:       models.CategoryFeature,
		TargetApps:     "Global Platform (Semua Proyek)",
		Title:          "Standarisasi Model Context Protocol (MCP) di Seluruh Proyek Lokal",
		Description:    "Pastikan seluruh proyek lokal mendaftarkan berkas .vscode/mcp.json agar AI coding agent dapat langsung menginspeksi topologi kode.",
		Status:         models.TodoStatusSuggested,
		Priority:       models.TodoPriorityHigh,
		CreatorType:    models.CreatorAI,
		CreatorName:    "vibeasik-ai-advisor",
		IsAISuggestion: true,
		GuidelineRef:   "AI-001",
		Rationale:      "Patokan: [AI-001] Integrasi MCP menyeluruh meningkatkan akurasi konteks AI agent dan mengeliminasi tebak-tebakan rute pada level perencanaan workspace.",
		Source:         "ai_agent",
	})

	// SEC-001: Zero-Credential Leak Audit - Category: Security
	suggestions = append(suggestions, models.Todo{
		Level:          models.TodoLevelGlobal,
		Category:       models.CategorySecurity,
		TargetApps:     "Global Platform (Semua Proyek)",
		Title:          "Audit Rutin Sanitasi Kredensial & Secrets Lintas-Proyek",
		Description:    "Jalankan scanner keamanan berkala untuk memastikan tidak ada database password atau AI API key yang tertrack ke dalam git repositori.",
		Status:         models.TodoStatusSuggested,
		Priority:       models.TodoPriorityHigh,
		CreatorType:    models.CreatorAI,
		CreatorName:    "vibeasik-ai-advisor",
		IsAISuggestion: true,
		GuidelineRef:   "SEC-001",
		Rationale:      "Patokan: [SEC-001] Menjaga integritas keamanan seluruh workspace dari kebocoran token rahasia secara otomatis.",
		Source:         "ai_agent",
	})

	// NET-002: Local Subdomain / Traefik Reverse Proxy - Category: Performance
	suggestions = append(suggestions, models.Todo{
		Level:          models.TodoLevelGlobal,
		Category:       models.CategoryPerformance,
		TargetApps:     "Global Network (Reverse Proxy Traefik/Caddy)",
		Title:          "Setup Reverse Proxy Lokal untuk DNS Subdomain Proyek (*.localhost)",
		Description:    "Konfigurasikan Traefik atau Caddy lokal untuk memetakan domain lokal (misal: app.localhost, api.localhost) ke port internal masing-masing servis.",
		Status:         models.TodoStatusSuggested,
		Priority:       models.TodoPriorityMedium,
		CreatorType:    models.CreatorAI,
		CreatorName:    "vibeasik-ai-advisor",
		IsAISuggestion: true,
		GuidelineRef:   "NET-002",
		Rationale:      "Patokan: [NET-002] Menghilangkan konflik nomor port statis secara permanen di tingkat jaringan lokal dan menyederhanakan akses multi-bundle.",
		Source:         "ai_agent",
	})

	return suggestions
}
