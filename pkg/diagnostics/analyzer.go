package diagnostics

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/security"
)

// CalculateFindingPenalty returns the point deduction for a finding based on category, rule, and severity
func CalculateFindingPenalty(f models.Finding) int {
	if f.Category == "port" && strings.Contains(f.Title, "Internal Port Collision") {
		return 20
	}
	if f.Category == "port" && (strings.Contains(f.Title, "Cross-Project") || strings.Contains(f.Title, "Shared Dev Port")) {
		return 0
	}
	if strings.Contains(f.Title, "Missing Application Absolute Path") {
		return 30
	}
	if strings.Contains(f.Title, "Directory Missing on Disk") || strings.Contains(f.Title, "Component Directory Missing") {
		return 20
	}
	if strings.Contains(f.Title, ".env Not Excluded") || f.RuleID == "ENV-FILE-TRACKED" || f.RuleID == "ENV-NOT-IGNORED" {
		return 15
	}
	if strings.Contains(f.Title, "Missing .gitignore") {
		return 8
	}
	if strings.Contains(f.Title, "Missing .env.example") || f.RuleID == "ENV-EXAMPLE-MISSING" {
		return 5
	}
	if strings.Contains(f.Title, "Missing README") {
		return 4
	}
	if strings.Contains(f.Title, "No Automated Tests") {
		return 3
	}
	if f.Category == "git" {
		return 0
	}
	switch strings.ToUpper(f.Severity) {
	case "CRITICAL":
		return 8
	case "WARNING":
		return 4
	case "INFO":
		return 1
	default:
		return 1
	}
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

// AnalyzeProject runs static diagnostics and returns an evaluation report
func AnalyzeProject(project *models.Project, allProjects []models.Project) (*models.Diagnostic, error) {
	findings := make([]models.Finding, 0)
	optimizations := make([]models.Optimization, 0)
	score := 100

	// 1. Port Conflict Detection (Internal & Cross-project). Port lintas proyek wajib unik (blok PORTS.md, Aturan 35/45).
	portMap := make(map[int]string)
	for _, comp := range project.Components {
		port := effectivePort(comp)
		if port <= 0 {
			continue
		}
		if existingComp, clash := portMap[port]; clash {
			findings = append(findings, models.Finding{
				Category:    "port",
				Severity:    "CRITICAL",
				Title:       fmt.Sprintf("Internal Port Collision: %d", port),
				Description: fmt.Sprintf("Both '%s' and '%s' are assigned to port %d. Simultaneous execution will fail.", comp.Name, existingComp, port),
				Suggestion:  fmt.Sprintf("Reassign port for '%s' in dev_command or env file (e.g. port %d).", comp.Name, port+1),
			})
			score -= 20
		} else {
			portMap[port] = comp.Name
		}
	}
	findings = append(findings, crossProjectPortFindings(project, allProjects)...)

	// 2. Check Essential Project Files at Root (if root_path is configured)
	if project.RootPath != "" && project.RootPath != "-" {
		rootGitignore := filepath.Join(project.RootPath, ".gitignore")
		if !util.FileExists(rootGitignore) {
			findings = append(findings, models.Finding{
				Category:    "config",
				Severity:    "WARNING",
				Title:       "Missing .gitignore at Project Root",
				Description: "No root .gitignore file was found. Build artifacts and secrets could be accidentally tracked.",
				Suggestion:  "Add a comprehensive .gitignore covering node_modules, binaries, and .env files.",
				TargetFile:  ".gitignore",
			})
			score -= 8
		}
		// Multi env (.env.local, .env.server*) ter-ignore & hanya .env.example yang di-commit
		findings = append(findings, inspectEnvHygiene(project.RootPath, "")...)
		// dev.sh & deploy.sh wajib (Aturan 45)
		findings = append(findings, inspectProjectTooling(project)...)

		rootReadme := filepath.Join(project.RootPath, "README.md")
		if !util.FileExists(rootReadme) {
			findings = append(findings, models.Finding{
				Category:    "documentation",
				Severity:    "INFO",
				Title:       "Missing README.md Documentation",
				Description: "No root README.md found. AI agents and contributors need quick architectural instructions.",
				Suggestion:  "Create README.md describing the BE/FE bundle structure and run instructions.",
				TargetFile:  "README.md",
			})
			score -= 4
		}
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

		// Check component path exists
		if _, err := os.Stat(comp.AbsolutePath); err != nil {
			findings = append(findings, models.Finding{
				Category:    "architecture",
				Severity:    "CRITICAL",
				Title:       fmt.Sprintf("Component Directory Missing: %s", comp.Name),
				Description: fmt.Sprintf("The path '%s' does not exist on disk.", comp.AbsolutePath),
				Suggestion:  "Verify directory path or re-scan project.",
			})
			score -= 20
			continue
		}

		// Env per komponen (root proyek sudah diperiksa di langkah 2)
		if comp.AbsolutePath != project.RootPath {
			findings = append(findings, inspectEnvHygiene(comp.AbsolutePath, comp.RelativePath)...)
		}

		// Check tests
		if !hasTests(comp.AbsolutePath, comp.Language) {
			findings = append(findings, models.Finding{
				Category:    "quality",
				Severity:    "INFO",
				Title:       fmt.Sprintf("No Automated Tests Detected in %s", comp.Name),
				Description: fmt.Sprintf("Component '%s' has no test suites detected.", comp.Name),
				Suggestion:  "Add unit or integration tests to ensure regression-free development.",
			})
			score -= 3
		}

		// Git status inside component
		compGitStatus := getGitStatusShort(comp.AbsolutePath)
		if compGitStatus != "" && (project.RootPath == "" || comp.AbsolutePath != project.RootPath) {
			findings = append(findings, models.Finding{
				Category:    "git",
				Severity:    "INFO",
				Title:       fmt.Sprintf("Uncommitted Git Changes in %s", comp.Name),
				Description: fmt.Sprintf("Working tree in '%s' has modified files.", comp.Name),
				Suggestion:  "Review status with 'git status' and commit changes before running major AI refactors.",
				TargetFile:  comp.RelativePath,
			})
		}
	}

	// 4. Bundle Integration & Architecture Checks
	if hasBackend && len(feComponents) > 0 {
		// Multi-FE or Bundle coherence
		for _, fe := range feComponents {
			if fe.EnvSummary != "" && backendComp != nil && backendComp.Port > 0 {
				if !strings.Contains(fe.EnvSummary, "API") && !strings.Contains(fe.EnvSummary, "URL") && !strings.Contains(fe.EnvSummary, "BACKEND") {
					findings = append(findings, models.Finding{
						Category:    "architecture",
						Severity:    "INFO",
						Title:       fmt.Sprintf("Frontend '%s' May Lack Backend Endpoint Config", fe.Name),
						Description: fmt.Sprintf("Could not identify explicit API target env in '%s' pointing to backend port %d.", fe.Name, backendComp.Port),
						Suggestion:  fmt.Sprintf("Ensure %s has an environment variable e.g. NUXT_PUBLIC_API_URL=http://localhost:%d", fe.Name, backendComp.Port),
					})
				}
			}
		}
	}

	// FE & mobile wajib memuat data BE yang benar saat dev; Nuxt tanpa warning Duplicated imports
	findings = append(findings, inspectFrontendBackendLink(project, 0)...)
	for _, comp := range project.Components {
		findings = append(findings, inspectNuxtDuplicateAutoImports(comp)...)
	}

	// 5. Git Status Check (if root_path is configured)
	if project.RootPath != "" && project.RootPath != "-" {
		gitStatusOutput := getGitStatusShort(project.RootPath)
		if gitStatusOutput != "" {
			findings = append(findings, models.Finding{
				Category:    "git",
				Severity:    "INFO",
				Title:       "Uncommitted Git Changes Detected",
				Description: "Working tree has modified or uncommitted files.",
				Suggestion:  "Review status with 'git status' and commit changes before running major AI refactors.",
			})
		}
	}

	// 6. Deep Static Code Inspection (Database Queries, Routers, and Application Code Smells)
	codeFindings := InspectProjectCode(project)
	for _, cf := range codeFindings {
		findings = append(findings, cf)
		if cf.Severity == "CRITICAL" {
			score -= 8
		} else if cf.Severity == "WARNING" {
			score -= 4
		} else {
			score -= 1
		}
	}

	// 7. Generate Optimizations & AI Prompts
	optimizations = generateOptimizations(project, findings, hasBackend, len(feComponents))

	findings = models.DeduplicateFindings(findings)
	score = CalculateHealthScore(findings)

	findingsBytes, _ := json.Marshal(findings)
	optimizationsBytes, _ := json.Marshal(optimizations)

	safeFindings := security.SanitizeText(string(findingsBytes))
	safeOptimizations := security.SanitizeText(string(optimizationsBytes))

	summary := fmt.Sprintf("Analyzed %d components across %s architecture. Detected %d findings. Codebase health score: %d/100.",
		len(project.Components), project.ProjectType, len(findings), score)

	var structureTree string
	if project.RootPath != "" && project.RootPath != "-" {
		structureTree = generateFolderTree(project.RootPath, 2)
	}

	return &models.Diagnostic{
		ProjectID:     project.ID,
		Summary:       summary,
		Score:         score,
		FindingsJSON:  safeFindings,
		Optimizations: safeOptimizations,
		StructureTree: structureTree,
	}, nil
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
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".test.ts") || strings.HasSuffix(name, ".spec.ts") || strings.HasSuffix(name, ".test.js") || strings.HasSuffix(name, "test.dart") || strings.HasPrefix(name, "test_") {
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

func generateFolderTree(root string, maxDepth int) string {
	var sb strings.Builder
	var walk func(path string, depth int, prefix string)

	walk = func(current string, depth int, prefix string) {
		if depth > maxDepth {
			return
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			return
		}
		for i, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "dist" {
				continue
			}
			isLast := i == len(entries)-1
			connector := "├── "
			if isLast {
				connector = "└── "
			}

			if entry.IsDir() {
				sb.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, connector, name))
				newPrefix := prefix + "│   "
				if isLast {
					newPrefix = prefix + "    "
				}
				walk(filepath.Join(current, name), depth+1, newPrefix)
			} else {
				sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, name))
			}
		}
	}

	sb.WriteString(fmt.Sprintf("%s/\n", filepath.Base(root)))
	walk(root, 1, "")
	return sb.String()
}

// AnalyzeComponent runs focused diagnostics for an individual application component
func AnalyzeComponent(comp *models.Component, project *models.Project, allProjects []models.Project) (*models.Diagnostic, error) {
	findings := make([]models.Finding, 0)
	optimizations := make([]models.Optimization, 0)
	score := 100

	// 1. Path existence
	if comp.AbsolutePath == "" {
		findings = append(findings, models.Finding{
			Category:    "architecture",
			Severity:    "CRITICAL",
			Title:       "Missing Application Absolute Path",
			Description: fmt.Sprintf("Application '%s' does not have an absolute path configured.", comp.Name),
			Suggestion:  "Configure the exact directory path on disk.",
		})
		score -= 30
	} else if _, err := os.Stat(comp.AbsolutePath); err != nil {
		findings = append(findings, models.Finding{
			Category:    "architecture",
			Severity:    "CRITICAL",
			Title:       fmt.Sprintf("Directory Missing on Disk: %s", comp.Name),
			Description: fmt.Sprintf("Path '%s' does not exist.", comp.AbsolutePath),
			Suggestion:  "Create the directory or verify folder location.",
		})
		score -= 30
	} else {
		// 2. Multi env (.env.local, .env.server*) ter-ignore & hanya .env.example yang di-commit
		findings = append(findings, inspectEnvHygiene(comp.AbsolutePath, comp.RelativePath)...)

		// Automated tests
		if !hasTests(comp.AbsolutePath, comp.Language) {
			findings = append(findings, models.Finding{
				Category:    "quality",
				Severity:    "INFO",
				Title:       "No Automated Tests Detected",
				Description: fmt.Sprintf("Application '%s' has no test suites detected.", comp.Name),
				Suggestion:  "Add unit or integration tests to ensure regression-free development.",
			})
			score -= 5
		}

		// Git status in component
		gitStatus := getGitStatusShort(comp.AbsolutePath)
		if gitStatus != "" {
			findings = append(findings, models.Finding{
				Category:    "git",
				Severity:    "INFO",
				Title:       "Uncommitted Git Changes",
				Description: fmt.Sprintf("Working tree in '%s' has modified files.", comp.Name),
				Suggestion:  "Review and commit changes before running major AI refactors.",
			})
		}
	}

	// 3. Port check: unik di dalam proyek dan lintas proyek (blok PORTS.md, Aturan 35/45)
	if port := effectivePort(*comp); port > 0 {
		for _, other := range project.Components {
			if other.ID != comp.ID && effectivePort(other) == port {
				findings = append(findings, models.Finding{
					Category:    "port",
					Severity:    "CRITICAL",
					Title:       fmt.Sprintf("Internal Port Collision: %d", port),
					Description: fmt.Sprintf("Port %d is also assigned to '%s' in this project.", port, other.Name),
					Suggestion:  fmt.Sprintf("Reassign port for '%s' to avoid execution failure.", comp.Name),
				})
				score -= 20
			}
		}
		single := *project
		single.Components = []models.Component{*comp}
		findings = append(findings, crossProjectPortFindings(&single, allProjects)...)
	}
	findings = append(findings, inspectFrontendBackendLink(project, comp.ID)...)
	findings = append(findings, inspectNuxtDuplicateAutoImports(*comp)...)

	// 4. Synergy & Role specific checks
	if strings.HasPrefix(comp.Role, "frontend") {
		hasBE := false
		var bePort int
		for _, other := range project.Components {
			if other.Role == models.RoleBackend {
				hasBE = true
				bePort = other.Port
				break
			}
		}
		if hasBE && bePort > 0 && comp.EnvSummary != "" {
			if !strings.Contains(comp.EnvSummary, "API") && !strings.Contains(comp.EnvSummary, "URL") && !strings.Contains(comp.EnvSummary, "BACKEND") {
				findings = append(findings, models.Finding{
					Category:    "architecture",
					Severity:    "INFO",
					Title:       "Missing Backend Endpoint Target in Environment",
					Description: fmt.Sprintf("Could not identify explicit API target env in '%s' pointing to backend port %d.", comp.Name, bePort),
					Suggestion:  fmt.Sprintf("Ensure %s has an environment variable e.g. API_URL=http://localhost:%d", comp.Name, bePort),
				})
			}
		}
	}

	// 5. Deep Static Code Inspection (Database Queries, Routers, and Application Code Smells)
	compFindings := InspectComponentCode(comp)
	for _, cf := range compFindings {
		findings = append(findings, cf)
		if cf.Severity == "CRITICAL" {
			score -= 8
		} else if cf.Severity == "WARNING" {
			score -= 4
		} else {
			score -= 1
		}
	}

	// Optimizations per component
	optimizations = append(optimizations, models.Optimization{
		Title:       fmt.Sprintf("AI & Agent Readiness for %s", comp.Name),
		Description: fmt.Sprintf("Configure developer commands and MCP instructions specifically for %s.", comp.Name),
		Category:    "ai_readiness",
		Impact:      "HIGH",
		ActionSteps: []string{
			fmt.Sprintf("Set working directory to '%s' in agent configuration", comp.AbsolutePath),
			fmt.Sprintf("Use dev command '%s' for testing changes", comp.DevCommand),
		},
		PromptForAI: fmt.Sprintf("Analyze the application '%s' (%s, %s/%s) located at '%s'. Review its code quality, security posture, and architecture. Suggest 3 high-impact refactoring steps.", comp.Name, comp.Role, comp.Language, comp.Framework, comp.AbsolutePath),
	})

	findings = models.DeduplicateFindings(findings)
	score = CalculateHealthScore(findings)

	findingsBytes, _ := json.Marshal(findings)
	optimizationsBytes, _ := json.Marshal(optimizations)

	safeFindings := security.SanitizeText(string(findingsBytes))
	safeOptimizations := security.SanitizeText(string(optimizationsBytes))

	summary := fmt.Sprintf("Application '%s' health score: %d/100 (%d findings). Role: %s, Stack: %s/%s.",
		comp.Name, score, len(findings), comp.Role, comp.Language, comp.Framework)

	var structureTree string
	if comp.AbsolutePath != "" {
		structureTree = generateFolderTree(comp.AbsolutePath, 2)
	}

	return &models.Diagnostic{
		ProjectID:     project.ID,
		Summary:       summary,
		Score:         score,
		FindingsJSON:  safeFindings,
		Optimizations: safeOptimizations,
		StructureTree: structureTree,
	}, nil
}
