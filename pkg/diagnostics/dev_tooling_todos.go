package diagnostics

import (
	"fmt"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// defaultRuleGuideline memetakan RuleID audit Aturan 45 ke patokan guidelines/ (GuidelineRef todo).
var defaultRuleGuideline = map[string]string{
	"DEV-SCRIPT-MISSING": "ARCH-002", "DEV-SCRIPT-NOT-EXECUTABLE": "ARCH-002", "DEV-SCRIPT-INCOMPLETE": "ARCH-002",
	"MOBILE-DEV-SCRIPT-MISSING": "ARCH-002", "MOBILE-DEV-SCRIPT-NOT-EXECUTABLE": "ARCH-002", "MOBILE-DEV-SCRIPT-INCOMPLETE": "ARCH-002",
	"DEPLOY-SCRIPT-MISSING": "DEV-007", "DEPLOY-ENV-NOT-RESTORED": "DEV-007", "DEPLOY-ENV-NOT-SELECTABLE": "DEV-007",
	"FE-API-PORT-MISMATCH": "NET-010", "MOBILE-API-LOCALHOST": "NET-010",
	"FE-DUPLICATE-IMPORT": "FE-015",
}

// standardEnvRules: temuan env Aturan 32 yang harus dibereskan agent (berkas yang masuk git: .gitignore & template).
var standardEnvRules = map[string]bool{
	"ENV-FILE-TRACKED": true, "ENV-NOT-IGNORED": true, "ENV-IGNORE-INCOMPLETE": true,
	"ENV-EXAMPLE-MISSING": true, "ENV-EXAMPLE-IGNORED": true, "ENV-EXAMPLE-DRIFT": true,
}

// StandardFindings = kekurangan standar berkas proyek (Aturan 45 & 49): dev.sh, dev-mobile.sh, deploy.sh, .gitignore env & .env.example.
// Dikerjakan agent sebagai satu todo gabungan; .env.local & .env.server dibuat engine (EnsureEnvFiles).
func StandardFindings(project *models.Project) []models.Finding {
	if project == nil {
		return nil
	}
	findings := inspectProjectTooling(project)
	seen := map[string]bool{}
	add := func(dir, rel string) {
		if dir == "" || dir == "-" || seen[dir] {
			return
		}
		seen[dir] = true
		for _, f := range inspectEnvHygiene(dir, rel) {
			if standardEnvRules[f.RuleID] {
				findings = append(findings, f)
			}
		}
	}
	add(project.RootPath, "")
	for _, c := range project.Components {
		add(c.AbsolutePath, c.RelativePath)
	}
	return findings
}

// CodeTodos = todo kode default proyek (URL API FE/mobile ke BE, duplicate import), status pending agar langsung
// ikut "Suruh Agent Otomatis". Judul memuat berkas sasaran agar stabil dan dedup judul+proyek+komponen tidak
// menelan temuan berkas lain. Port lintas proyek tidak di sini: engine mengalokasikan blok slot sendiri.
func CodeTodos(project *models.Project) []models.Todo {
	if project == nil {
		return nil
	}
	var todos []models.Todo
	for i := range project.Components {
		comp := &project.Components[i]
		for _, f := range inspectFrontendBackendLink(project, comp.ID) {
			todos = append(todos, TodoFromFinding(project, comp, f))
		}
		for _, f := range inspectNuxtDuplicateAutoImports(*comp) {
			todos = append(todos, TodoFromFinding(project, comp, f))
		}
		if !strings.HasPrefix(comp.Role, "frontend") || comp.AbsolutePath == "" {
			continue
		}
		for _, af := range ListAuditableFiles(comp.AbsolutePath, comp.RelativePath, 1200) {
			if af.Lang != "typescript" && af.Lang != "javascript" && af.Lang != "vue" && af.Lang != "svelte" {
				continue
			}
			// satu todo per berkas: semua import ganda di berkas itu dikerjakan sekaligus
			if fs := inspectDuplicateImports(af.RelPath, strings.Split(readSmallFile(af.AbsPath), "\n")); len(fs) > 0 {
				f := fs[0]
				f.Title = "Gabungkan duplicate import"
				var lines []string
				for _, d := range fs {
					lines = append(lines, fmt.Sprintf("- L%d %s", d.Line, d.Snippet))
				}
				f.Description = fmt.Sprintf("%d import ganda di %s:\n%s", len(fs), af.RelPath, strings.Join(lines, "\n"))
				todos = append(todos, TodoFromFinding(project, comp, f))
			}
		}
	}
	return todos
}

// TodoFromFinding mengubah satu temuan default proyek menjadi todo pending (comp nil = level proyek).
func TodoFromFinding(project *models.Project, comp *models.Component, f models.Finding) models.Todo {
	title := f.Title
	if f.TargetFile != "" && !strings.Contains(title, f.TargetFile) {
		title += " (" + f.TargetFile + ")"
	}
	desc := f.Description + "\n\nLangkah: " + f.Suggestion
	if f.TargetFile != "" {
		desc += "\n\nFile Target: `" + f.TargetFile + "`"
	}
	if f.SuggestedFix != "" {
		desc += "\n\nContoh:\n```\n" + f.SuggestedFix + "\n```"
	}
	prio := models.TodoPriorityMedium
	if f.Severity == "CRITICAL" {
		prio = models.TodoPriorityHigh
	} else if f.Severity == "INFO" {
		prio = models.TodoPriorityLow
	}
	ref := defaultRuleGuideline[f.RuleID]
	t := models.Todo{
		Level:        models.TodoLevelProject,
		Category:     models.CategoryDevOps,
		ProjectID:    &project.ID,
		ProjectName:  project.Name,
		TargetApps:   fmt.Sprintf("Bundle %s", project.Name),
		Title:        title,
		Description:  desc,
		Status:       models.TodoStatusPending,
		Priority:     prio,
		CreatorType:  models.CreatorAI,
		CreatorName:  "vibeasik-standard",
		GuidelineRef: ref,
		Rationale:    fmt.Sprintf("Patokan: [%s] default proyek vibeasik (Aturan 45), temuan audit %s.", ref, f.RuleID),
		Source:       "vibeasik_standard",
	}
	if comp != nil {
		t.Level, t.ComponentID, t.ComponentName, t.TargetApps = models.TodoLevelComponent, &comp.ID, comp.Name, comp.Name
		if f.Category == "frontend" || f.Category == "mobile" {
			t.Category = models.CategoryUIUX
		}
	}
	return t
}
