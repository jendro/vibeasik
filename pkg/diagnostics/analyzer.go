package diagnostics

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/security"
)

// ID aturan pemeriksaan tingkat proyek/komponen (struktur, bukan isi kode). ID tetap membuat temuan bisa dilacak
// per aturan di pembelajaran audit dan dipetakan ke dasar pemeriksaannya.
const (
	RulePortCollision       = "PROJ-PORT-COLLISION"
	RuleGitignoreMissing    = "PROJ-GITIGNORE-MISSING"
	RuleReadmeMissing       = "PROJ-README-MISSING"
	RuleComponentPathUnset  = "PROJ-COMPONENT-PATH-UNSET"
	RuleComponentDirMissing = "PROJ-COMPONENT-DIR-MISSING"
	RuleNoTests             = "PROJ-NO-TESTS"
	RuleFEBackendEnv        = "PROJ-FE-BACKEND-ENV"
	RuleGitUncommitted      = "GIT-UNCOMMITTED"
)

// componentRef: penanda komponen untuk kunci temuan tingkat komponen (path relatif, atau namanya bila kosong).
func componentRef(comp *models.Component) string {
	if comp.RelativePath != "" && comp.RelativePath != "." {
		return comp.RelativePath
	}
	return comp.Name
}

// IsAdvisoryFinding: temuan yang hanya potret keadaan sesaat atau perkiraan, bukan cacat kode yang bisa
// "diperbaiki" — working tree yang sedang berubah, port bawaan yang sama dengan proyek lain, dan estimasi biaya
// statis. Ditampilkan di laporan audit, tetapi tidak dijadikan todo otomatis.
func IsAdvisoryFinding(f models.Finding) bool {
	return f.Category == "git" || f.RuleID == RuleGitUncommitted || strings.HasPrefix(f.RuleID, "PORT-SHARED-") ||
		f.RuleID == rulePolynomialGrowth
}

// penaltyRule: aturan pengurangan skor berurutan; aturan pertama yang cocok menang.
// category kosong = semua kategori; tanpa titles & ruleIDs = cocok untuk seluruh kategori itu.
type penaltyRule struct {
	category string
	titles   []string // cocok bila Title memuat salah satunya
	ruleIDs  []string // cocok bila RuleID sama dengan salah satunya
	points   int
}

func (r penaltyRule) matches(f models.Finding) bool {
	if r.category != "" && f.Category != r.category {
		return false
	}
	if len(r.titles) == 0 && len(r.ruleIDs) == 0 {
		return true
	}
	for _, t := range r.titles {
		if strings.Contains(f.Title, t) {
			return true
		}
	}
	for _, id := range r.ruleIDs {
		if f.RuleID == id {
			return true
		}
	}
	return false
}

var penaltyRules = []penaltyRule{
	{category: "port", titles: []string{"Internal Port Collision"}, points: 20},
	{category: "port", titles: []string{"Cross-Project", "Shared Dev Port"}, points: 0},
	{titles: []string{"Missing Application Absolute Path"}, points: 30},
	{titles: []string{"Directory Missing on Disk", "Component Directory Missing"}, points: 20},
	{titles: []string{".env Not Excluded"}, ruleIDs: []string{"ENV-FILE-TRACKED", "ENV-NOT-IGNORED"}, points: 15},
	{titles: []string{"Missing .gitignore"}, points: 8},
	{titles: []string{"Missing .env.example"}, ruleIDs: []string{"ENV-EXAMPLE-MISSING"}, points: 5},
	{titles: []string{"Missing README"}, points: 4},
	{titles: []string{"No Automated Tests"}, points: 3},
	// Rumus biaya statis adalah batas atas kasus terburuk: bobotnya lebih ringan dari temuan yang pasti.
	{ruleIDs: []string{ruleNPlusOneChain}, points: 3},
	{ruleIDs: []string{rulePolynomialGrowth}, points: 1},
	{category: "git", points: 0},
	// Risiko profil = potret analisis (bukan pengukuran kode): ditindaklanjuti sebagai todo, tidak menurunkan skor.
	{category: CategoryRisk, points: 0},
}

// severityPenalty: pengurangan skor menurut tingkat keparahan (default 1).
var severityPenalty = map[string]int{"CRITICAL": 8, "WARNING": 4, "INFO": 1}

// CalculateFindingPenalty returns the point deduction for a finding based on category, rule, and severity
func CalculateFindingPenalty(f models.Finding) int {
	for _, r := range penaltyRules {
		if r.matches(f) {
			return r.points
		}
	}
	if p, ok := severityPenalty[strings.ToUpper(f.Severity)]; ok {
		return p
	}
	return 1
}

// CalculateHealthScore computes the 0-100 codebase health score based ONLY on active findings.
// Resolved and Ignored findings do not reduce the health score.
func CalculateHealthScore(findings []models.Finding) int {
	score := 100
	for _, f := range findings {
		if f.Status == models.FindingStatusResolved || f.Status == models.FindingStatusIgnored {
			continue
		}
		score -= CalculateFindingPenalty(f)
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

// internalPortCollisionFindings mendeteksi dua komponen satu proyek yang memakai port sama.
func internalPortCollisionFindings(project *models.Project) []models.Finding {
	var findings []models.Finding
	portMap := make(map[int]string)
	for _, comp := range project.Components {
		port := effectivePort(comp)
		if port <= 0 {
			continue
		}
		existingComp, clash := portMap[port]
		if !clash {
			portMap[port] = comp.Name
			continue
		}
		findings = append(findings, models.Finding{
			Category:    "port",
			Severity:    "CRITICAL",
			RuleID:      RulePortCollision,
			TargetFile:  comp.Name,
			Line:        port,
			Title:       fmt.Sprintf("Internal Port Collision: %d", port),
			Description: fmt.Sprintf("Both '%s' and '%s' are assigned to port %d. Simultaneous execution will fail.", comp.Name, existingComp, port),
			Suggestion:  fmt.Sprintf("Reassign port for '%s' in dev_command or env file (e.g. port %d).", comp.Name, port+1),
		})
	}
	return findings
}

func hasProjectRoot(project *models.Project) bool {
	return project.RootPath != "" && project.RootPath != "-"
}

// rootFileFindings memeriksa berkas penting di root proyek (.gitignore, env, dev.sh/deploy.sh, README).
func rootFileFindings(project *models.Project) []models.Finding {
	var findings []models.Finding
	// .gitignore & README root hanya bermakna bila root proyek adalah repo git; root berupa folder kerja biasa
	// (kumpulan repo, folder dokumen) tidak punya indeks git untuk dijaga.
	isRepo := util.PathExists(filepath.Join(project.RootPath, ".git"))
	if isRepo && !util.FileExists(filepath.Join(project.RootPath, ".gitignore")) {
		findings = append(findings, models.Finding{
			Category:    "config",
			Severity:    "WARNING",
			RuleID:      RuleGitignoreMissing,
			Title:       "Missing .gitignore at Project Root",
			Description: "No root .gitignore file was found. Build artifacts and secrets could be accidentally tracked.",
			Suggestion:  "Add a comprehensive .gitignore covering node_modules, binaries, and .env files.",
			TargetFile:  ".gitignore",
		})
	}
	// Multi env (.env.local, .env.server*) ter-ignore & hanya .env.example yang di-commit
	findings = append(findings, inspectEnvHygiene(project.RootPath, "")...)
	// dev.sh & deploy.sh wajib (Aturan 45)
	findings = append(findings, inspectProjectTooling(project)...)

	if isRepo && !util.FileExists(filepath.Join(project.RootPath, "README.md")) {
		findings = append(findings, models.Finding{
			Category:    "documentation",
			Severity:    "INFO",
			RuleID:      RuleReadmeMissing,
			Title:       "Missing README.md Documentation",
			Description: "No root README.md found. AI agents and contributors need quick architectural instructions.",
			Suggestion:  "Create README.md describing the BE/FE bundle structure and run instructions.",
			TargetFile:  "README.md",
		})
	}
	return findings
}

// inspectProjectComponent memeriksa path, env, test, dan status git satu komponen proyek.
func inspectProjectComponent(project *models.Project, comp *models.Component) []models.Finding {
	var findings []models.Finding
	// Check component path exists
	if comp.AbsolutePath == "" {
		// Komponen tanpa path sama sekali: yang kurang adalah data komponennya, bukan direktori di disk.
		return append(findings, models.Finding{
			Category:    "architecture",
			Severity:    "WARNING",
			RuleID:      RuleComponentPathUnset,
			TargetFile:  comp.Name,
			Title:       fmt.Sprintf("Component Path Not Set: %s", comp.Name),
			Description: fmt.Sprintf("Component '%s' has neither an absolute nor a relative path, so its code cannot be inspected.", comp.Name),
			Suggestion:  "Set the component path (relative to the project root) or re-scan the project.",
		})
	}
	if _, err := os.Stat(comp.AbsolutePath); err != nil {
		return append(findings, models.Finding{
			Category:    "architecture",
			Severity:    "CRITICAL",
			RuleID:      RuleComponentDirMissing,
			TargetFile:  comp.Name,
			Title:       fmt.Sprintf("Component Directory Missing: %s", comp.Name),
			Description: fmt.Sprintf("The path '%s' does not exist on disk.", comp.AbsolutePath),
			Suggestion:  "Verify directory path or re-scan project.",
		})
	}

	// Env per komponen (root proyek sudah diperiksa di langkah 2)
	if comp.AbsolutePath != project.RootPath {
		findings = append(findings, inspectEnvHygiene(comp.AbsolutePath, comp.RelativePath)...)
	}

	// Check tests (komponen non-kode seperti dokumentasi/aset tidak diwajibkan punya test)
	if comp.Role != models.RoleOther && !hasTests(comp.AbsolutePath, comp.Language) {
		findings = append(findings, models.Finding{
			Category:    "quality",
			Severity:    "INFO",
			RuleID:      RuleNoTests,
			TargetFile:  componentRef(comp),
			Title:       fmt.Sprintf("No Automated Tests Detected in %s", comp.Name),
			Description: fmt.Sprintf("Component '%s' has no test suites detected.", comp.Name),
			Suggestion:  "Add unit or integration tests to ensure regression-free development.",
		})
	}

	// Git status inside component
	if getGitStatusShort(comp.AbsolutePath) != "" && (project.RootPath == "" || comp.AbsolutePath != project.RootPath) {
		findings = append(findings, models.Finding{
			Category:    "git",
			Severity:    "INFO",
			RuleID:      RuleGitUncommitted,
			Title:       fmt.Sprintf("Uncommitted Git Changes in %s", comp.Name),
			Description: fmt.Sprintf("Working tree in '%s' has modified files.", comp.Name),
			Suggestion:  "Review status with 'git status' and commit changes before running major AI refactors.",
			TargetFile:  comp.RelativePath,
		})
	}
	return findings
}

// envLacksAPITarget: ringkasan env tidak menyebut target API (API/URL/BACKEND).
func envLacksAPITarget(envSummary string) bool {
	return !strings.Contains(envSummary, "API") && !strings.Contains(envSummary, "URL") && !strings.Contains(envSummary, "BACKEND")
}

// bundleCoherenceFindings: FE yang env-nya tidak menunjuk backend bundle.
func bundleCoherenceFindings(backendComp *models.Component, feComponents []models.Component) []models.Finding {
	var findings []models.Finding
	if backendComp == nil || len(feComponents) == 0 {
		return findings
	}
	for _, fe := range feComponents {
		if fe.EnvSummary == "" || backendComp.Port <= 0 || !envLacksAPITarget(fe.EnvSummary) {
			continue
		}
		findings = append(findings, models.Finding{
			Category:    "architecture",
			Severity:    "INFO",
			RuleID:      RuleFEBackendEnv,
			TargetFile:  componentRef(&fe),
			Title:       fmt.Sprintf("Frontend '%s' May Lack Backend Endpoint Config", fe.Name),
			Description: fmt.Sprintf("Could not identify explicit API target env in '%s' pointing to backend port %d.", fe.Name, backendComp.Port),
			Suggestion:  fmt.Sprintf("Ensure %s has an environment variable e.g. NUXT_PUBLIC_API_URL=http://localhost:%d", fe.Name, backendComp.Port),
		})
	}
	return findings
}

// uncommittedRootFinding: working tree root proyek punya perubahan belum di-commit.
func uncommittedRootFinding(project *models.Project) []models.Finding {
	if !hasProjectRoot(project) || getGitStatusShort(project.RootPath) == "" {
		return nil
	}
	return []models.Finding{{
		Category:    "git",
		Severity:    "INFO",
		RuleID:      RuleGitUncommitted,
		Title:       "Uncommitted Git Changes Detected",
		Description: "Working tree has modified or uncommitted files.",
		Suggestion:  "Review status with 'git status' and commit changes before running major AI refactors.",
	}}
}

// buildDiagnostic menyusun laporan akhir: skor dari temuan aktif, JSON temuan/optimasi yang disanitasi.
func buildDiagnostic(projectID uint, findings []models.Finding, optimizations []models.Optimization, summaryFor func(score int) string, structureTree string) *models.Diagnostic {
	score := CalculateHealthScore(findings)

	// Rahasia hanya bisa ikut lewat potongan kode; path berkas, judul, dan ID tidak disamarkan (nama berkas/slug
	// panjang yang ikut tersamar membuat target temuan tidak bisa dibuka lagi).
	for i := range findings {
		findings[i].Snippet = security.SanitizeText(findings[i].Snippet)
		findings[i].SuggestedFix = security.SanitizeText(findings[i].SuggestedFix)
	}
	findingsBytes, _ := json.Marshal(findings)
	optimizationsBytes, _ := json.Marshal(optimizations)

	return &models.Diagnostic{
		ProjectID:     projectID,
		Summary:       summaryFor(score),
		Score:         score,
		FindingsJSON:  string(findingsBytes),
		Optimizations: security.SanitizeText(string(optimizationsBytes)),
		StructureTree: structureTree,
	}
}

// withResolvedComponentPaths mengembalikan salinan proyek yang komponennya punya path absolut: komponen yang hanya
// menyimpan path relatif (absolute_path kosong) dihitung dari root proyek. Tanpa ini seluruh pemeriksaan komponen
// membaca path kosong dan melaporkan direktori "hilang" padahal ada di disk.
func withResolvedComponentPaths(project *models.Project) *models.Project {
	if !hasProjectRoot(project) {
		return project
	}
	out := *project
	out.Components = append([]models.Component(nil), project.Components...)
	for i := range out.Components {
		if c := &out.Components[i]; c.AbsolutePath == "" && c.RelativePath != "" {
			c.AbsolutePath = resolveComponentDir(project.RootPath, c.RelativePath)
		}
	}
	return &out
}

// resolveComponentDir: direktori komponen dari path relatifnya. Sebagian proyek menyimpan path relatif yang
// diawali nama folder root ("toko/mobile" dengan root ".../toko"); kandidat yang benar-benar ada dipilih.
func resolveComponentDir(root, rel string) string {
	direct := filepath.Join(root, rel)
	if util.PathExists(direct) {
		return direct
	}
	relSlash := filepath.ToSlash(rel)
	if base := filepath.Base(root); strings.HasPrefix(relSlash, base+"/") {
		if alt := filepath.Join(filepath.Dir(root), rel); util.PathExists(alt) {
			return alt
		}
	}
	return direct
}

// fillFindingEvidence mengisi potongan kode (Snippet) temuan yang menunjuk berkas & baris tetapi belum membawa
// buktinya, dengan membaca baris itu dari disk. Temuan tanpa lokasi kode (berkas yang belum ada, tingkat proyek)
// dibiarkan: buktinya adalah keadaan proyek, bukan baris kode.
func fillFindingEvidence(project *models.Project, findings []models.Finding) {
	if !hasProjectRoot(project) {
		return
	}
	cache := map[string][]string{}
	for i := range findings {
		f := &findings[i]
		if f.Snippet != "" || f.TargetFile == "" || f.Line <= 0 {
			continue
		}
		ck := f.Component + "|" + f.TargetFile
		lines, ok := cache[ck]
		if !ok {
			for _, p := range evidencePaths(project, *f) {
				if data, err := os.ReadFile(p); err == nil && len(data) < 2<<20 {
					lines = strings.Split(string(data), "\n")
					break
				}
			}
			cache[ck] = lines
		}
		if f.Line <= len(lines) {
			if code := strings.TrimSpace(lines[f.Line-1]); len(code) > 220 {
				f.Snippet = code[:220] + "…"
			} else {
				f.Snippet = code
			}
		}
	}
}

// evidencePaths: berkas bukti sebuah temuan — folder komponen asalnya (bila tercatat) lalu akar proyek; tanpa
// tebakan lewat komponen lain agar potongan kode tidak diambil dari berkas yang salah.
func evidencePaths(project *models.Project, f models.Finding) []string {
	var out []string
	for _, c := range project.Components {
		if f.Component != "" && c.AbsolutePath != "" && project.ComponentKey(c) == f.Component {
			out = append(out, filepath.Join(c.AbsolutePath, filepath.FromSlash(f.TargetFile)))
		}
	}
	return append(out, filepath.Join(project.RootPath, filepath.FromSlash(f.TargetFile)))
}

// AnalyzeProject runs static diagnostics and returns an evaluation report
func AnalyzeProject(project *models.Project, allProjects []models.Project) (*models.Diagnostic, error) {
	project = withResolvedComponentPaths(project)
	findings := make([]models.Finding, 0)

	// 1. Port Conflict Detection (Internal & Cross-project). Port lintas proyek wajib unik (blok PORTS.md, Aturan 35/45).
	findings = append(findings, internalPortCollisionFindings(project)...)
	findings = append(findings, crossProjectPortFindings(project, allProjects)...)

	// 2. Check Essential Project Files at Root (if root_path is configured)
	if hasProjectRoot(project) {
		findings = append(findings, rootFileFindings(project)...)
	}

	// 3. Inspect Individual Components
	hasBackend := false
	var backendComp *models.Component
	feComponents := make([]models.Component, 0)
	for i := range project.Components {
		comp := &project.Components[i]
		if comp.Role == models.RoleBackend {
			hasBackend = true
			backendComp = comp
		} else if strings.HasPrefix(comp.Role, "frontend") {
			feComponents = append(feComponents, *comp)
		}
		// Temuan berkas komponen memakai path relatif terhadap folder komponen: catat komponennya agar berkas
		// dengan path relatif sama di komponen lain (app/layouts/default.vue) tidak tertukar.
		key := project.ComponentKey(*comp)
		for _, f := range inspectProjectComponent(project, comp) {
			if f.Component == "" {
				f.Component = key
			}
			findings = append(findings, f)
		}
	}

	// 4. Bundle Integration & Architecture Checks
	if hasBackend {
		findings = append(findings, bundleCoherenceFindings(backendComp, feComponents)...)
	}

	// FE & mobile wajib memuat data BE yang benar saat dev; Nuxt tanpa warning Duplicated imports
	findings = append(findings, inspectFrontendBackendLink(project, 0)...)
	for _, comp := range project.Components {
		findings = append(findings, inspectNuxtDuplicateAutoImports(comp)...)
	}

	// 5. Git Status Check (if root_path is configured)
	findings = append(findings, uncommittedRootFinding(project)...)

	// 6. Deep Static Code Inspection (Database Queries, Routers, and Application Code Smells)
	findings = append(findings, InspectProjectCode(project)...)

	// Risiko profil proyek yang belum ditindaklanjuti ikut diaudit (alur audit → todo)
	findings = append(findings, projectRiskFindings(project)...)

	// 7. Generate Optimizations & AI Prompts
	optimizations := generateOptimizations(project, findings, hasBackend, len(feComponents))
	findings = models.DeduplicateFindings(findings)
	// Setiap temuan yang menunjuk berkas & baris wajib membawa potongan kodenya sebagai bukti.
	fillFindingEvidence(project, findings)

	var structureTree string
	if hasProjectRoot(project) {
		structureTree = util.GenerateFolderTree(project.RootPath, 2)
	}
	return buildDiagnostic(project.ID, findings, optimizations, func(score int) string {
		return fmt.Sprintf("Analyzed %d components across %s architecture. Detected %d findings. Codebase health score: %d/100.",
			len(project.Components), project.ProjectType, len(findings), score)
	}, structureTree), nil
}

func generateOptimizations(project *models.Project, findings []models.Finding, hasBE bool, feCount int) []models.Optimization {
	opts := make([]models.Optimization, 0)

	// AI Readiness Optimization
	opts = append(opts, models.Optimization{
		Title:       "Model Context Protocol (MCP) Integration",
		Description: "Configure this project to expose its architecture, components, and dev commands directly to AI agents (Antigravity, Cursor, Claude).",
		Category:    "ai_readiness",
		Impact:      "HIGH",
		ActionSteps: []string{
			"Register project into ProjectManager MCP server",
			"Expose dev commands and port mapping in mcp.json",
			"Enable automated AI health audits before git commits",
		},
		PromptForAI: fmt.Sprintf(`Analyze the software bundle '%s' located at '%s'. It consists of %d components (%s). Recommend architectural improvements, identify potential concurrency or state synchronization issues between Backend and Frontend, and generate a task backlog for upcoming development.`, project.Name, project.RootPath, len(project.Components), project.ProjectType),
	})

	// Unified Docker Compose for Bundle
	if hasBE && feCount > 0 {
		opts = append(opts, models.Optimization{
			Title:       "Single-Command Local Orchestration",
			Description: "Create a root docker-compose.yml or Makefile to spin up all backend and frontend services concurrently with live-reload.",
			Category:    "devops",
			Impact:      "HIGH",
			ActionSteps: []string{
				"Add root docker-compose.dev.yml mapping all service ports",
				"Create Makefile or package.json script with concurrently/turbo",
				"Harmonize environment variables in single root .env",
			},
			PromptForAI: fmt.Sprintf(`Write a root docker-compose.dev.yml and Makefile for project '%s' that runs the backend and all %d frontend applications with hot reload enabled.`, project.Name, feCount),
		})
	}

	// Performance & Caching Optimization
	opts = append(opts, models.Optimization{
		Title:       "API Response Caching & Connection Pooling",
		Description: "Ensure database connection pools and HTTP response headers (ETags, Cache-Control) are configured for high-speed local dev and production.",
		Category:    "performance",
		Impact:      "MEDIUM",
		ActionSteps: []string{
			"Audit database connection pool settings (SetMaxOpenConns, SetMaxIdleConns)",
			"Add Gzip / Brotli compression middleware in Fiber/Express",
			"Verify CORS preflight caching",
		},
		PromptForAI: fmt.Sprintf(`Review the backend service of '%s' and optimize Fiber/Go database connection pooling, response compression, and CORS policy for local development and production.`, project.Name),
	})

	return opts
}

// testFileRegex: berkas test lintas ekosistem (Go, JS/TS, Dart, Python, PHP, Java/Kotlin).
var testFileRegex = regexp.MustCompile(`(_test\.(go|dart|py)|\.(test|spec)\.(ts|tsx|js|jsx|mjs|cjs)|Test\.(php|java|kt)|Tests?\.cs)$|^test_.*\.py$`)

// testDirNames: folder yang isinya test menurut konvensi umum.
var testDirNames = map[string]bool{"tests": true, "test": true, "__tests__": true, "spec": true, "e2e": true, "integration_test": true}

func hasTests(dir string, lang string) bool {
	var found bool
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || found {
			return filepath.SkipDir
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		name := info.Name()
		_, isSource := auditFileExtensions[strings.ToLower(filepath.Ext(name))]
		inTestDir := testDirNames[filepath.Base(filepath.Dir(path))] && (isSource || strings.HasSuffix(name, ".mjs"))
		if testFileRegex.MatchString(name) || strings.HasPrefix(name, "test_") || inTestDir {
			found = true
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

func getGitStatusShort(dir string) string {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// componentPathFindings memeriksa path komponen; bila ada, juga env, test, dan status git-nya.
func componentPathFindings(comp *models.Component) []models.Finding {
	if comp.AbsolutePath == "" {
		return []models.Finding{{
			Category:    "architecture",
			Severity:    "CRITICAL",
			Title:       "Missing Application Absolute Path",
			Description: fmt.Sprintf("Application '%s' does not have an absolute path configured.", comp.Name),
			Suggestion:  "Configure the exact directory path on disk.",
		}}
	}
	if _, err := os.Stat(comp.AbsolutePath); err != nil {
		return []models.Finding{{
			Category:    "architecture",
			Severity:    "CRITICAL",
			Title:       fmt.Sprintf("Directory Missing on Disk: %s", comp.Name),
			Description: fmt.Sprintf("Path '%s' does not exist.", comp.AbsolutePath),
			Suggestion:  "Create the directory or verify folder location.",
		}}
	}
	// Multi env (.env.local, .env.server*) ter-ignore & hanya .env.example yang di-commit
	findings := inspectEnvHygiene(comp.AbsolutePath, comp.RelativePath)

	// Automated tests
	if !hasTests(comp.AbsolutePath, comp.Language) {
		findings = append(findings, models.Finding{
			Category:    "quality",
			Severity:    "INFO",
			Title:       "No Automated Tests Detected",
			Description: fmt.Sprintf("Application '%s' has no test suites detected.", comp.Name),
			Suggestion:  "Add unit or integration tests to ensure regression-free development.",
		})
	}

	// Git status in component
	if getGitStatusShort(comp.AbsolutePath) != "" {
		findings = append(findings, models.Finding{
			Category:    "git",
			Severity:    "INFO",
			Title:       "Uncommitted Git Changes",
			Description: fmt.Sprintf("Working tree in '%s' has modified files.", comp.Name),
			Suggestion:  "Review and commit changes before running major AI refactors.",
		})
	}
	return findings
}

// componentPortFindings: port unik di dalam proyek dan lintas proyek (blok PORTS.md, Aturan 35/45).
func componentPortFindings(comp *models.Component, project *models.Project, allProjects []models.Project) []models.Finding {
	port := effectivePort(*comp)
	if port <= 0 {
		return nil
	}
	var findings []models.Finding
	for _, other := range project.Components {
		if other.ID != comp.ID && effectivePort(other) == port {
			findings = append(findings, models.Finding{
				Category:    "port",
				Severity:    "CRITICAL",
				RuleID:      RulePortCollision,
				TargetFile:  comp.Name,
				Line:        port,
				Title:       fmt.Sprintf("Internal Port Collision: %d", port),
				Description: fmt.Sprintf("Port %d is also assigned to '%s' in this project.", port, other.Name),
				Suggestion:  fmt.Sprintf("Reassign port for '%s' to avoid execution failure.", comp.Name),
			})
		}
	}
	single := *project
	single.Components = []models.Component{*comp}
	return append(findings, crossProjectPortFindings(&single, allProjects)...)
}

// frontendBackendTargetFinding: komponen frontend yang env-nya tidak menyebut target API backend proyek.
func frontendBackendTargetFinding(comp *models.Component, project *models.Project) []models.Finding {
	if !strings.HasPrefix(comp.Role, "frontend") {
		return nil
	}
	hasBE := false
	var bePort int
	for _, other := range project.Components {
		if other.Role == models.RoleBackend {
			hasBE = true
			bePort = other.Port
			break
		}
	}
	if !hasBE || bePort <= 0 || comp.EnvSummary == "" || !envLacksAPITarget(comp.EnvSummary) {
		return nil
	}
	return []models.Finding{{
		Category:    "architecture",
		Severity:    "INFO",
		Title:       "Missing Backend Endpoint Target in Environment",
		Description: fmt.Sprintf("Could not identify explicit API target env in '%s' pointing to backend port %d.", comp.Name, bePort),
		Suggestion:  fmt.Sprintf("Ensure %s has an environment variable e.g. API_URL=http://localhost:%d", comp.Name, bePort),
	}}
}

func componentOptimization(comp *models.Component) models.Optimization {
	return models.Optimization{
		Title:       fmt.Sprintf("AI & Agent Readiness for %s", comp.Name),
		Description: fmt.Sprintf("Configure developer commands and MCP instructions specifically for %s.", comp.Name),
		Category:    "ai_readiness",
		Impact:      "HIGH",
		ActionSteps: []string{
			fmt.Sprintf("Set working directory to '%s' in agent configuration", comp.AbsolutePath),
			fmt.Sprintf("Use dev command '%s' for testing changes", comp.DevCommand),
		},
		PromptForAI: fmt.Sprintf("Analyze the application '%s' (%s, %s/%s) located at '%s'. Review its code quality, security posture, and architecture. Suggest 3 high-impact refactoring steps.", comp.Name, comp.Role, comp.Language, comp.Framework, comp.AbsolutePath),
	}
}

// AnalyzeComponent runs focused diagnostics for an individual application component
func AnalyzeComponent(comp *models.Component, project *models.Project, allProjects []models.Project) (*models.Diagnostic, error) {
	findings := make([]models.Finding, 0)

	// 1-2. Path existence, env, tests, git
	findings = append(findings, componentPathFindings(comp)...)

	// 3. Port check
	findings = append(findings, componentPortFindings(comp, project, allProjects)...)
	findings = append(findings, inspectFrontendBackendLink(project, comp.ID)...)
	findings = append(findings, inspectNuxtDuplicateAutoImports(*comp)...)

	// 4. Synergy & Role specific checks
	findings = append(findings, frontendBackendTargetFinding(comp, project)...)

	// 5. Deep Static Code Inspection (Database Queries, Routers, and Application Code Smells)
	findings = append(findings, InspectComponentCode(comp)...)

	// Optimizations per component
	optimizations := []models.Optimization{componentOptimization(comp)}
	findings = models.DeduplicateFindings(findings)

	var structureTree string
	if comp.AbsolutePath != "" {
		structureTree = util.GenerateFolderTree(comp.AbsolutePath, 2)
	}
	return buildDiagnostic(project.ID, findings, optimizations, func(score int) string {
		return fmt.Sprintf("Application '%s' health score: %d/100 (%d findings). Role: %s, Stack: %s/%s.",
			comp.Name, score, len(findings), comp.Role, comp.Language, comp.Framework)
	}, structureTree), nil
}
