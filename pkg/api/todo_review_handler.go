package api

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/guidelines"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/standard"
)

// rescoreTodo menghitung ulang skor kerumitan/dampak bila dimensinya atau kategori diubah lewat update.
// Dimensi yang diisi tangan tanpa sumber eksplisit dicatat sebagai penilaian manual.
func rescoreTodo(todo *models.Todo, updates map[string]interface{}) bool {
	touched := false
	for k := range updates {
		if updateTouchesRescore(todo, updates, k) {
			touched = true
		}
	}
	if touched {
		todo.Rescore()
	}
	return touched
}

func updateTouchesRescore(todo *models.Todo, updates map[string]interface{}, k string) bool {
	if strings.HasPrefix(k, "complexity_") && k != "complexity_source" {
		if _, ok := updates["complexity_source"]; !ok && k != "complexity_note" {
			todo.ComplexitySource = models.ComplexitySourceManual
		}
		return true
	}
	if strings.HasPrefix(k, "impact_") && k != "impact_source" {
		if _, ok := updates["impact_source"]; !ok && k != "impact_summary" {
			todo.ImpactSource = models.ComplexitySourceManual
		}
		return true
	}
	return k == "category"
}

func (s *Server) approveTodo(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid todo id"})
	}

	targetStatus := models.TodoStatusPending
	var body struct {
		Status string `json:"status"`
	}
	if msg := firstProblem(bindJSON(c, &body), oneOf("status", body.Status, todoStatuses...)); msg != "" {
		return badPayload(c, msg)
	}

	var todo models.Todo
	if err := s.db.First(&todo, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "todo not found"})
	}
	if body.Status != "" {
		targetStatus = body.Status
	}

	applyTodoApprovalState(&todo, targetStatus)

	if err := s.db.Save(&todo).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "approved", "todo": todo})
}

func applyTodoApprovalState(todo *models.Todo, targetStatus string) {
	todo.Status = targetStatus
	if targetStatus == models.TodoStatusCompleted {
		if todo.CompletedAt == nil {
			now := time.Now()
			todo.CompletedAt = &now
		}
		if todo.CompletedByAgent == "" {
			todo.CompletedByAgent = "developer"
		}
		return
	}
	if targetStatus == models.TodoStatusPending || targetStatus == models.TodoStatusInProgress {
		todo.CompletedAt = nil
		todo.CompletedByAgent = ""
		todo.CompletedByModel = ""
	}
}

func (s *Server) rejectTodo(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid todo id"})
	}

	var todo models.Todo
	if err := s.db.First(&todo, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "todo not found"})
	}

	todo.Status = models.TodoStatusRejected
	if err := s.db.Save(&todo).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "rejected", "todo": todo})
}

// getGuidelines returns all grouped guidelines from the guidelines directory
func (s *Server) getGuidelines(c *fiber.Ctx) error {
	categories, err := guidelines.LoadAllGuidelines()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(categories)
}

func findProjectByID(projects []models.Project, id uint) *models.Project {
	for i := range projects {
		if projects[i].ID == id {
			return &projects[i]
		}
	}
	return nil
}

func (s *Server) collectProjectSuggestions(targetProject *models.Project, allProjects []models.Project) []models.Todo {
	var suggestions []models.Todo
	suggestions = append(suggestions, guidelines.GenerateSuggestionsForProject(targetProject, allProjects)...)
	// Standar vibeasik (Aturan 45): port, env & todo pending dipastikan langsung, bukan saran
	if r, _ := standard.Ensure(s.db, targetProject.ID); r != nil {
		suggestions = append(suggestions, r.Todos...)
	}
	return suggestions
}

func (s *Server) collectAllSuggestions(allProjects []models.Project) []models.Todo {
	var suggestions []models.Todo
	suggestions = append(suggestions, guidelines.GenerateGlobalSuggestions(allProjects)...)
	for i := range allProjects {
		suggestions = append(suggestions, s.collectProjectSuggestions(&allProjects[i], allProjects)...)
	}
	return suggestions
}

func (s *Server) syncExistingSuggestion(existing *models.Todo, sugg models.Todo) {
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

func (s *Server) processSuggestions(suggestions []models.Todo, saveToDB bool) ([]models.Todo, int) {
	finalSuggestions := make([]models.Todo, 0, len(suggestions))
	createdCount := 0
	todoIdx := models.NewTodoIndex(s.db, suggestions)
	for _, sugg := range suggestions {
		existing, found := todoIdx.Find(sugg)
		if !found {
			if saveToDB {
				if errCreate := s.db.Create(&sugg).Error; errCreate == nil {
					createdCount++
					todoIdx.Add(sugg)
				}
			}
			finalSuggestions = append(finalSuggestions, sugg)
		} else {
			s.syncExistingSuggestion(&existing, sugg)
			finalSuggestions = append(finalSuggestions, existing)
		}
	}
	return finalSuggestions, createdCount
}

// suggestTodos analyzes projects against guidelines and generates actionable suggestions
func (s *Server) suggestTodos(c *fiber.Ctx) error {
	var body struct {
		ProjectID uint  `json:"project_id"`
		SaveToDB  *bool `json:"save_to_db"`
	}
	if msg := bindJSON(c, &body); msg != "" {
		return badPayload(c, msg)
	}

	saveToDB := true
	if body.SaveToDB != nil {
		saveToDB = *body.SaveToDB
	}

	var allProjects []models.Project
	s.db.Preload("Components").Preload("Components.Routes").Find(&allProjects)

	var suggestions []models.Todo
	var targetProject *models.Project

	if body.ProjectID > 0 {
		targetProject = findProjectByID(allProjects, body.ProjectID)
		if targetProject != nil {
			suggestions = append(suggestions, s.collectProjectSuggestions(targetProject, allProjects)...)
		}
	} else {
		suggestions = append(suggestions, s.collectAllSuggestions(allProjects)...)
	}

	// Saran cerdas (rapat Project Manager / LLM) disediakan modul Pro; Community hanya saran rule.
	var pmSession *models.PlanningSession
	pmJob, pmError := "", ""
	suggestions = append(suggestions, registry.SuggestTodos(s.db, targetProject, allProjects)...)

	finalSuggestions, createdCount := s.processSuggestions(suggestions, saveToDB)

	return c.JSON(fiber.Map{
		"count":       len(finalSuggestions),
		"new_count":   createdCount,
		"saved_to_db": saveToDB,
		"suggestions": finalSuggestions,
		"pm_session":  pmSession,
		"pm_job_id":   pmJob,
		"pm_error":    pmError,
	})
}

var (
	todoStatuses = []string{models.TodoStatusPending, models.TodoStatusInProgress, models.TodoStatusCompleted,
		models.TodoStatusSuggested, models.TodoStatusRejected, models.TodoStatusMerged, models.TodoStatusSplit}
	todoPriorities = []string{models.TodoPriorityLow, models.TodoPriorityMedium, models.TodoPriorityHigh, models.TodoPriorityCritical}
)
