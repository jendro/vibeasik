package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/vibeasik/vibeasik/pkg/agents"
	"github.com/vibeasik/vibeasik/pkg/diagnostics"
	"github.com/vibeasik/vibeasik/pkg/guidelines"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/scanner"
	"github.com/vibeasik/vibeasik/pkg/standard"
	"github.com/vibeasik/vibeasik/pkg/workspace"
)

// localEngineURL adalah alamat engine harian yang ditanamkan ke identifier workspace.
const localEngineURL = "http://127.0.0.1:8765"

func (s *Server) toolListProjects(args map[string]interface{}) (*ToolCallResult, error) {
	var projects []models.Project
	s.db.Preload("Components").Find(&projects)
	var out interface{} = projects
	if detail, _ := args["detail"].(bool); !detail {
		compact := make([]compactProject, len(projects))
		for i, p := range projects {
			compact[i] = toCompactProject(p)
		}
		out = compact
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return textResult(string(data)), nil
}

func (s *Server) toolGetProjectDetail(args map[string]interface{}) (*ToolCallResult, error) {
	var p models.Project
	if pid, ok := args["project_id"]; ok {
		s.db.Preload("Components").First(&p, pid)
	} else if slug, ok := args["slug"]; ok {
		s.db.Preload("Components").Where("slug = ?", slug).First(&p)
	}
	if p.ID == 0 {
		return nil, fmt.Errorf("project not found")
	}
	return prettyJSONText(p), nil
}

func (s *Server) toolScanDirectory(args map[string]interface{}) (*ToolCallResult, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("path is required")
	}
	scanRes, err := scanner.ScanDirectory(path)
	if err != nil {
		return nil, err
	}
	return prettyJSONText(scanRes), nil
}

// loadProjectWithAll memuat proyek beserta komponennya dan seluruh proyek lain (bahan analisis).
func (s *Server) loadProjectWithAll(pid uint) (models.Project, []models.Project) {
	var p models.Project
	s.db.Preload("Components").First(&p, pid)
	if p.ID == 0 {
		return p, nil
	}
	var all []models.Project
	s.db.Preload("Components").Find(&all)
	return p, all
}

func (s *Server) toolRunDiagnostics(args map[string]interface{}) (*ToolCallResult, error) {
	pid := argUint(args, "project_id")
	p, allProjects := s.loadProjectWithAll(pid)
	if p.ID == 0 {
		return nil, fmt.Errorf("project with ID %d not found", pid)
	}
	diag, err := diagnostics.AnalyzeProject(&p, allProjects)
	if err != nil {
		return nil, err
	}
	s.db.Create(diag)
	p.HealthScore = diag.Score
	s.db.Save(&p)
	return prettyJSONText(diag), nil
}

func (s *Server) toolAIRecommendations(args map[string]interface{}) (*ToolCallResult, error) {
	p, allProjects := s.loadProjectWithAll(argUint(args, "project_id"))
	if p.ID == 0 {
		return nil, fmt.Errorf("project not found")
	}
	diag, err := diagnostics.AnalyzeProject(&p, allProjects)
	if err != nil {
		return nil, err
	}
	return textResult(fmt.Sprintf("=== AI Recommendations for %s ===\n%s", p.Name, diag.Optimizations)), nil
}

func (s *Server) toolListLocalAgents(args map[string]interface{}) (*ToolCallResult, error) {
	return prettyJSONText(agents.GetRegistry().DiscoverAll(s.db)), nil
}

func (s *Server) toolCurrentProject(args map[string]interface{}) (*ToolCallResult, error) {
	targetPath, _ := args["path"].(string)
	result, err := workspace.IdentifyWorkspace(s.db, targetPath, localEngineURL)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	return textResult(string(data)), nil
}

func (s *Server) toolLinkProjectWorkspace(args map[string]interface{}) (*ToolCallResult, error) {
	pid := argUint(args, "project_id")
	if pid == 0 {
		return nil, fmt.Errorf("valid project_id is required")
	}
	cid := argUint(args, "component_id")
	// Tanpa component_id = "Init vibeasik": tautan AI + peta kode di repo yang diperbarui otomatis.
	if cid == 0 {
		res, err := workspace.InitProject(s.db, pid, localEngineURL)
		if err != nil {
			return nil, err
		}
		return jsonResult(res)
	}
	createdFiles, idf, err := workspace.LinkProjectWorkspaces(s.db, pid, cid, localEngineURL)
	if err != nil {
		return nil, err
	}
	return prettyJSONText(map[string]interface{}{
		"message":       "Workspace linked successfully with AI Agent identifier (.vibeasik.json)",
		"project_id":    pid,
		"files_created": createdFiles,
		"identifier":    idf,
	}), nil
}

func (s *Server) toolListGuidelines(args map[string]interface{}) (*ToolCallResult, error) {
	categories, err := guidelines.LoadAllGuidelines()
	if err != nil {
		return nil, err
	}
	if catFilter, ok := args["category"].(string); ok && catFilter != "" {
		var filtered []guidelines.GuidelineCategory
		for _, cat := range categories {
			if cat.Category == catFilter {
				filtered = append(filtered, cat)
			}
		}
		return prettyJSONText(filtered), nil
	}
	return prettyJSONText(categories), nil
}

func (s *Server) toolSuggestTodos(args map[string]interface{}) (*ToolCallResult, error) {
	pid := argUint(args, "project_id")
	saveToDB := true
	if val, ok := args["save_to_db"].(bool); ok {
		saveToDB = val
	}

	var allProjects []models.Project
	s.db.Preload("Components").Preload("Components.Routes").Find(&allProjects)

	suggestions, targetProject := s.collectSuggestions(pid, allProjects)
	// Saran cerdas (rapat Project Manager / LLM) disediakan modul Pro.
	suggestions = append(suggestions, registry.SuggestTodos(s.db, targetProject, allProjects)...)

	finalSuggestions := s.reconcileSuggestions(suggestions, saveToDB)
	if targetProject == nil {
		return prettyJSONText(finalSuggestions), nil
	}
	return jsonResult(map[string]interface{}{"guideline_suggestions": finalSuggestions})
}

// collectSuggestions menghimpun saran aturan untuk satu proyek (pid > 0) atau seluruh workspace.
func (s *Server) collectSuggestions(pid uint, allProjects []models.Project) ([]models.Todo, *models.Project) {
	var suggestions []models.Todo
	if pid == 0 {
		suggestions = append(suggestions, guidelines.GenerateGlobalSuggestions(allProjects)...)
		for i := range allProjects {
			suggestions = append(suggestions, s.projectSuggestions(&allProjects[i], allProjects)...)
		}
		return suggestions, nil
	}
	for i := range allProjects {
		if allProjects[i].ID == pid {
			return s.projectSuggestions(&allProjects[i], allProjects), &allProjects[i]
		}
	}
	return nil, nil
}

func (s *Server) projectSuggestions(p *models.Project, allProjects []models.Project) []models.Todo {
	out := append([]models.Todo(nil), guidelines.GenerateSuggestionsForProject(p, allProjects)...)
	// Standar vibeasik (Aturan 45): port, env & todo pending dipastikan langsung, bukan saran
	if r, _ := standard.Ensure(s.db, p.ID); r != nil {
		out = append(out, r.Todos...)
	}
	return out
}

// reconcileSuggestions menyimpan saran baru (bila saveToDB) dan melengkapi todo yang sudah ada.
func (s *Server) reconcileSuggestions(suggestions []models.Todo, saveToDB bool) []models.Todo {
	final := make([]models.Todo, 0)
	todoIdx := models.NewTodoIndex(s.db, suggestions)
	for _, sugg := range suggestions {
		existing, found := todoIdx.Find(sugg)
		if !found {
			if saveToDB && s.db.Create(&sugg).Error == nil {
				todoIdx.Add(sugg)
			}
			final = append(final, sugg)
			continue
		}
		s.backfillExisting(&existing, sugg)
		final = append(final, existing)
	}
	return final
}

// backfillExisting mengisi kategori generik dan target app yang kosong pada todo yang sudah ada.
func (s *Server) backfillExisting(existing *models.Todo, sugg models.Todo) {
	updates := map[string]interface{}{}
	if (existing.Category == "" || existing.Category == models.CategoryGeneral) && sugg.Category != "" {
		existing.Category = sugg.Category
		updates["category"] = sugg.Category
	}
	if existing.TargetApps == "" && sugg.TargetApps != "" {
		existing.TargetApps = sugg.TargetApps
		updates["target_apps"] = sugg.TargetApps
	}
	if len(updates) > 0 {
		s.db.Model(existing).Updates(updates)
	}
}
