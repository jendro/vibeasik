package guidelines

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// advice memuat bagian saran yang berbeda-beda; sisanya (status, kreator, sumber) seragam.
type advice struct {
	level, category, priority string
	ref, title, desc, why     string
}

// advisorTodo menyusun todo saran vibeasik-ai-advisor untuk proyek (comp nil = level proyek/global).
func advisorTodo(project *models.Project, comp *models.Component, target string, a advice) models.Todo {
	t := models.Todo{
		Level:          a.level,
		Category:       a.category,
		ProjectID:      &project.ID,
		ProjectName:    project.Name,
		TargetApps:     target,
		Title:          a.title,
		Description:    a.desc,
		Status:         models.TodoStatusSuggested,
		Priority:       a.priority,
		CreatorType:    models.CreatorAI,
		CreatorName:    "vibeasik-ai-advisor",
		IsAISuggestion: true,
		GuidelineRef:   a.ref,
		Rationale:      a.why,
		Source:         "ai_agent",
	}
	if comp != nil {
		t.ComponentID = &comp.ID
		t.ComponentName = comp.Name
	}
	return t
}

// fileMissing adalah predikat boolean murni yang memeriksa ketiadaan berkas (K5088).
// Kontrak predikat: os.IsNotExist(err) adalah jalur normal (bukan swallowed error).
// Galat selain "tidak ada" (mis. izin akses) dicatat sebagai [WARN] dan dianggap berkas ada agar tidak memicu saran keliru.
func fileMissing(path string) bool {
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("[WARN] fileMissing: os.Stat(%q): %v", path, err)
	}
	return os.IsNotExist(err)
}

// fileExists adalah predikat boolean murni yang memeriksa keberadaan berkas (K5088).
// Kontrak predikat: err == nil menyatakan berkas ada dan dapat diakses; os.IsNotExist(err)
// adalah jalur normal ketika berkas tidak ditemukan, bukan swallowed error.
// Error sistem selain "tidak ada" (mis. izin akses) dicatat sebagai [WARN] agar dapat diobservasi.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("[WARN] fileExists: os.Stat(%q): %v", path, err)
	}
	return err == nil
}

func hasRealRoot(p *models.Project) bool {
	return p.RootPath != "" && p.RootPath != "-"
}

// GenerateSuggestionsForProject inspects a project bundle and derives 3-tier Todo suggestions based on guidelines
func GenerateSuggestionsForProject(project *models.Project, allProjects []models.Project) []models.Todo {
	var suggestions []models.Todo
	if project == nil {
		return suggestions
	}

	var compNames []string
	var beComp, feComp *models.Component
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

	// 2. Level Project: hygiene .env.example (SEC-002), .gitignore (SEC-001), AI_CONTEXT.md (AI-002)
	if hasRealRoot(project) {
		suggestions = append(suggestions, projectHygieneSuggestions(project, allAppsLabel)...)
	}
	// 3. Security Cross-Component Check: CORS & Origin Whitelist (SEC-004)
	if beComp != nil && feComp != nil {
		suggestions = append(suggestions, corsSuggestion(project, beComp, feComp))
	}
	// NET-001 (Aturan 35/45): blok port & run_port dialokasikan engine otomatis (runner/slots.go), bukan todo.

	// 4. Level Component: Periksa setiap aplikasi teknis individual
	for i := range project.Components {
		suggestions = append(suggestions, componentSuggestions(project, &project.Components[i])...)
	}
	// 5. Level Project: Pemeriksaan DevOps & Deployment
	if hasRealRoot(project) {
		suggestions = append(suggestions, devopsSuggestions(project, allAppsLabel)...)
	}
	return suggestions
}

func projectHygieneSuggestions(project *models.Project, label string) []models.Todo {
	var out []models.Todo
	root := project.RootPath
	if fileExists(filepath.Join(root, ".env")) && fileMissing(filepath.Join(root, ".env.example")) {
		out = append(out, advisorTodo(project, nil, label, advice{
			level: models.TodoLevelProject, category: models.CategorySecurity, priority: models.TodoPriorityHigh, ref: "SEC-002",
			title: fmt.Sprintf("Buat berkas template .env.example tersanitasi untuk %s", project.Name),
			desc:  "Dokumentasikan seluruh variabel lingkungan yang dibutuhkan proyek ke dalam .env.example dengan nilai dummy.",
			why:   "Patokan: [SEC-002] Berkas .env wajib memiliki pasangan .env.example tersanitasi tanpa kredensial asli.",
		}))
	}
	if fileMissing(filepath.Join(root, ".gitignore")) {
		out = append(out, advisorTodo(project, nil, label, advice{
			level: models.TodoLevelProject, category: models.CategorySecurity, priority: models.TodoPriorityCritical, ref: "SEC-001",
			title: fmt.Sprintf("Pasang .gitignore untuk mencegah kebocoran secret pada %s", project.Name),
			desc:  "Tambahkan .gitignore di root proyek dengan aturan mengabaikan .env, node_modules, log, dan kredensial lokal.",
			why:   "Patokan: [SEC-001] Repositori wajib memiliki .gitignore yang memfilter .env dan secrets agar tidak ter-commit ke remote git.",
		}))
	}
	if fileMissing(filepath.Join(root, "AI_CONTEXT.md")) {
		out = append(out, advisorTodo(project, nil, label, advice{
			level: models.TodoLevelProject, category: models.CategoryFeature, priority: models.TodoPriorityMedium, ref: "AI-002",
			title: fmt.Sprintf("Sediakan berkas AI_CONTEXT.md untuk panduan AI Agent pada %s", project.Name),
			desc:  "Dokumentasikan arsitektur bundle, konvensi kode, peran komponen, dan perintah run agar coding agent (Antigravity/Cursor) bekerja presisi.",
			why:   "Patokan: [AI-002] File AI_CONTEXT.md mempercepat konteks agent otonom hingga 80% dan mengeliminasi halusinasi refactoring.",
		}))
	}
	return out
}

func corsSuggestion(project *models.Project, be, fe *models.Component) models.Todo {
	return advisorTodo(project, nil, fmt.Sprintf("%s & %s [2 Apps]", be.Name, fe.Name), advice{
		level: models.TodoLevelProject, category: models.CategorySecurity, priority: models.TodoPriorityHigh, ref: "SEC-004",
		title: fmt.Sprintf("Konfigurasi CORS Whitelist & Proteksi Origin antara %s dan %s", be.Name, fe.Name),
		desc:  fmt.Sprintf("Pastikan CORS pada %s hanya mengizinkan origin lokal %s secara eksplisit dengan AllowCredentials: true, hindari penggunaan wildcard '*'.", be.Name, fe.Name),
		why:   "Patokan: [SEC-004] CORS wajib dibatasi hanya ke domain/origin resmi frontend yang terdaftar untuk mencegah cross-origin request forgery.",
	})
}

// componentSuggestions memeriksa satu aplikasi: health/keamanan backend, UI/UX frontend, dan keberadaan test.
func componentSuggestions(project *models.Project, comp *models.Component) []models.Todo {
	var out []models.Todo
	if comp.Role == models.RoleBackend {
		out = append(out, backendSuggestions(project, comp)...)
	}
	if strings.HasPrefix(comp.Role, "frontend") && (comp.Framework == "nuxt" || comp.Framework == "vue") {
		out = append(out, frontendSuggestions(project, comp)...)
	}
	if comp.Role == models.RoleBackend && comp.AbsolutePath != "" && !backendHasTests(comp.AbsolutePath) {
		out = append(out, advisorTodo(project, comp, singleAppLabel(comp), advice{
			level: models.TodoLevelComponent, category: models.CategoryArchitecture, priority: models.TodoPriorityCritical, ref: "TEST-001",
			title: fmt.Sprintf("Tambahkan unit test untuk business logic pada %s", comp.Name),
			desc:  "Buat file *_test.go untuk layer service dan utility dengan skenario happy path dan edge case. Target cakupan minimum 60%%.",
			why:   "Patokan: [TEST-001] Kode tanpa unit test adalah bom waktu — setiap refactor berpotensi merusak fitur tanpa mekanisme deteksi otomatis.",
		}))
	}
	if strings.HasPrefix(comp.Role, "frontend") && comp.AbsolutePath != "" && !frontendHasTests(comp.AbsolutePath) {
		out = append(out, advisorTodo(project, comp, singleAppLabel(comp), advice{
			level: models.TodoLevelComponent, category: models.CategoryUIUX, priority: models.TodoPriorityMedium, ref: "TEST-006",
			title: fmt.Sprintf("Tambahkan component test untuk komponen kritis pada %s", comp.Name),
			desc:  "Buat folder __tests__/ dan file *.test.ts menggunakan @vue/test-utils untuk memverifikasi rendering dan interaksi komponen kritis.",
			why:   "Patokan: [TEST-006] Komponen UI tanpa test rentan terhadap regresi visual dan fungsional saat dependensi diperbarui.",
		}))
	}
	return out
}

func singleAppLabel(comp *models.Component) string {
	return fmt.Sprintf("%s [1 App]", comp.Name)
}

// hasHealthProbe: komponen punya HealthURL atau rute /health atau /ping.
func hasHealthProbe(comp *models.Component) bool {
	if comp.HealthURL != "" {
		return true
	}
	for _, r := range comp.Routes {
		if strings.Contains(r.Path, "/health") || strings.Contains(r.Path, "/ping") {
			return true
		}
	}
	return false
}

func backendSuggestions(project *models.Project, comp *models.Component) []models.Todo {
	label := singleAppLabel(comp)
	var out []models.Todo
	if !hasHealthProbe(comp) {
		out = append(out, advisorTodo(project, comp, label, advice{
			level: models.TodoLevelComponent, category: models.CategoryBackend, priority: models.TodoPriorityHigh, ref: "API-001",
			title: fmt.Sprintf("Tambahkan endpoint GET /health pada %s", comp.Name),
			desc:  fmt.Sprintf("Pasang endpoint health probe bebas autentikasi pada port :%d yang mengembalikan JSON {'status': 'ok'}.", comp.Port),
			why:   "Patokan: [API-001] Health probe wajib tersedia agar Vibeasik dan reverse proxy (Caddy/Docker) dapat memantau status liveness servis.",
		}))
	}
	out = append(out,
		advisorTodo(project, comp, label, advice{
			level: models.TodoLevelComponent, category: models.CategorySecurity, priority: models.TodoPriorityHigh, ref: "SEC-005",
			title: fmt.Sprintf("Pasang Middleware Rate Limiting & Anti Brute-Force pada %s", comp.Name),
			desc:  fmt.Sprintf("Terapkan rate limiting pada %s (maksimal 100 req/menit untuk API umum, dan 5 req/menit untuk endpoint autentikasi/login).", comp.Name),
			why:   "Patokan: [SEC-005] Proteksi DoS dan brute-force wajib diterapkan di tingkat gateway backend.",
		}),
		advisorTodo(project, comp, label, advice{
			level: models.TodoLevelComponent, category: models.CategorySecurity, priority: models.TodoPriorityMedium, ref: "SEC-006",
			title: fmt.Sprintf("Terapkan HTTP Security Headers (CSP, HSTS, X-Frame-Options) pada %s", comp.Name),
			desc:  "Konfigurasikan security headers (X-Content-Type-Options: nosniff, X-Frame-Options: DENY, Content-Security-Policy) untuk mencegah serangan clickjacking dan MIME-sniffing.",
			why:   "Patokan: [SEC-006] Standar OWASP mengharuskan header keamanan dasar aktif di setiap respons HTTP.",
		}))
	return out
}

func frontendSuggestions(project *models.Project, comp *models.Component) []models.Todo {
	label := singleAppLabel(comp)
	var out []models.Todo
	if comp.AbsolutePath != "" && fileMissing(filepath.Join(comp.AbsolutePath, "composables", "useApi.ts")) {
		out = append(out, advisorTodo(project, comp, label, advice{
			level: models.TodoLevelComponent, category: models.CategoryUIUX, priority: models.TodoPriorityMedium, ref: "FE-001",
			title: fmt.Sprintf("Terapkan composable useApi.ts terpusat pada %s", comp.Name),
			desc:  "Buat composable useApi.ts untuk mengonsentrasikan pemanggilan API backend dengan typed $fetch dan penanganan token otomatis.",
			why:   "Patokan: [FE-001] Frontend wajib menggunakan wrapper API client terpusat untuk menjamin konsistensi header token dan endpoint dinamis.",
		}))
	}
	out = append(out,
		advisorTodo(project, comp, label, advice{
			level: models.TodoLevelComponent, category: models.CategoryUIUX, priority: models.TodoPriorityMedium, ref: "FE-002",
			title: fmt.Sprintf("Audit Desain Responsif Mobile & Antislop UI pada %s", comp.Name),
			desc:  "Verifikasi breakpoint seluler (sm, md, lg), tap target minimal 44px, serta skema warna low-fatigue yang nyaman di mata pengguna.",
			why:   "Patokan: [FE-002] Layout antarmuka harus reflow dengan bersih dari ukuran layar ponsel 375px hingga layar lebar desktop tanpa horizontal overflow.",
		}),
		advisorTodo(project, comp, label, advice{
			level: models.TodoLevelComponent, category: models.CategoryUIUX, priority: models.TodoPriorityMedium, ref: "FE-003",
			title: fmt.Sprintf("Sediakan Empty State & Notifikasi Toast pada %s", comp.Name),
			desc:  "Lengkapi tabel dan daftar dengan visual state saat data belum ada, serta berikan toast alert saat request gagal atau offline.",
			why:   "Patokan: [FE-003] Antarmuka tidak boleh membiarkan pengguna bingung dengan tampilan kosong tanpa aksi jelas.",
		}))
	return out
}

// dirHasFileSuffix: ada berkas (bukan direktori) di dir dengan salah satu akhiran.
func dirHasFileSuffix(dir string, suffixes ...string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[WARN] dirHasFileSuffix: os.ReadDir(%q): %v", dir, err)
		}
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		for _, sfx := range suffixes {
			if strings.HasSuffix(e.Name(), sfx) {
				return true
			}
		}
	}
	return false
}

func backendHasTests(path string) bool {
	return dirHasFileSuffix(filepath.Join(path, "internal"), "_test.go") || dirHasFileSuffix(path, "_test.go")
}

func frontendHasTests(path string) bool {
	return fileExists(filepath.Join(path, "__tests__")) || dirHasFileSuffix(path, ".test.ts", ".spec.ts")
}

// hasDockerfileIn: Dockerfile (atau dockerfile) ada langsung di dir.
func hasDockerfileIn(dir string) bool {
	for _, name := range []string{"Dockerfile", "dockerfile"} {
		if fileExists(filepath.Join(dir, name)) {
			return true
		}
	}
	return false
}

func projectHasDockerfile(project *models.Project) bool {
	if hasDockerfileIn(project.RootPath) {
		return true
	}
	for _, comp := range project.Components {
		if comp.AbsolutePath != "" && hasDockerfileIn(comp.AbsolutePath) {
			return true
		}
	}
	return false
}

func devopsSuggestions(project *models.Project, label string) []models.Todo {
	var out []models.Todo
	hasDockerfile := projectHasDockerfile(project)
	if !hasDockerfile && len(project.Components) > 0 {
		out = append(out, advisorTodo(project, nil, label, advice{
			level: models.TodoLevelProject, category: models.CategoryArchitecture, priority: models.TodoPriorityHigh, ref: "DEV-001",
			title: fmt.Sprintf("Sediakan Dockerfile multi-stage untuk deployment %s", project.Name),
			desc:  "Buat Dockerfile dengan 2 stage: builder (kompilasi) dan runner (image minimal alpine/distroless) untuk production deployment.",
			why:   "Patokan: [DEV-001] Dockerfile multi-stage memisahkan compiler dari image production, mengecilkan ukuran image dan memperkecil attack surface.",
		}))
	}
	if hasDockerfile && fileMissing(filepath.Join(project.RootPath, ".dockerignore")) {
		out = append(out, advisorTodo(project, nil, label, advice{
			level: models.TodoLevelProject, category: models.CategorySecurity, priority: models.TodoPriorityHigh, ref: "DEV-002",
			title: fmt.Sprintf("Buat .dockerignore untuk mencegah kebocoran file ke Docker image pada %s", project.Name),
			desc:  "Tambahkan .dockerignore dengan entri: .git, node_modules, .env, dist/, bin/, *.exe, .vscode/ agar tidak masuk ke Docker build context.",
			why:   "Patokan: [DEV-002] Tanpa .dockerignore, .git dan node_modules masuk ke Docker context, memperlambat build dan berpotensi menyertakan secrets.",
		}))
	}
	if fileMissing(filepath.Join(project.RootPath, "CHANGELOG.md")) {
		out = append(out, advisorTodo(project, nil, label, advice{
			level: models.TodoLevelProject, category: models.CategoryFeature, priority: models.TodoPriorityMedium, ref: "DEV-004",
			title: fmt.Sprintf("Sediakan CHANGELOG.md untuk dokumentasi riwayat perubahan %s", project.Name),
			desc:  "Buat CHANGELOG.md menggunakan format Keep a Changelog untuk mendokumentasikan fitur baru, bugfix, dan breaking changes per versi rilis.",
			why:   "Patokan: [DEV-004] CHANGELOG memudahkan rollback target dan membantu pengguna mengetahui perubahan tanpa harus membaca git log.",
		}))
	}
	return out
}
