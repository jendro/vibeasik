package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/guidelines"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/standard"
)

func (s *Server) getTodos(c *fiber.Ctx) error {
	level := c.Query("level")
	projectIDStr := c.Query("project_id")
	componentIDStr := c.Query("component_id")
	status := c.Query("status")
	priority := c.Query("priority")
	creatorType := c.Query("creator_type")
	category := c.Query("category")
	isSuggestionStr := c.Query("is_ai_suggestion")
	source := c.Query("source")
	tag := c.Query("tag")

	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	query := s.db.Model(&models.Todo{})

	if level != "" {
		query = query.Where("level = ?", level)
	}

	if category != "" && category != "all" {
		query = query.Where("category = ?", category)
	}

	if source != "" && source != "all" {
		query = query.Where("source = ?", source)
	}

	if tag != "" && tag != "all" {
		query = models.WhereTag(query, tag)
	}

	if projectIDStr != "" {
		pid, err := strconv.Atoi(projectIDStr)
		if err == nil {
			query = query.Where("project_id = ?", pid)
		}
	}

	if componentIDStr != "" {
		cid, err := strconv.Atoi(componentIDStr)
		if err == nil {
			query = query.Where("component_id = ?", cid)
		}
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if priority != "" {
		query = query.Where("priority = ?", priority)
	}

	if creatorType != "" {
		query = query.Where("creator_type = ?", creatorType)
	}

	if isSuggestionStr != "" {
		isSuggestion := isSuggestionStr == "true" || isSuggestionStr == "1"
		query = query.Where("is_ai_suggestion = ?", isSuggestion)
	}

	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			query = query.Limit(limit)
		}
	}

	if offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			query = query.Offset(offset)
		}
	}

	var todos []models.Todo
	if err := query.Order("status ASC, created_at DESC").Find(&todos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(todos)
}

// getImpactDimensions mengembalikan definisi dimensi dampak, bobot per kategori, dan ambang tingkat
// sehingga UI tidak menyalin aturan skor dari engine.
func (s *Server) getImpactDimensions(c *fiber.Ctx) error {
	return c.JSON(models.ImpactReference())
}

func (s *Server) getProjectTodos(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")
	status := c.Query("status")
	priority := c.Query("priority")
	creatorType := c.Query("creator_type")
	category := c.Query("category")
	source := c.Query("source")

	query := s.db.Where("project_id = ?", id)
	if category != "" && category != "all" {
		query = query.Where("category = ?", category)
	}
	if source != "" && source != "all" {
		query = query.Where("source = ?", source)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if creatorType != "" {
		query = query.Where("creator_type = ?", creatorType)
	}

	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			query = query.Limit(limit)
		}
	}
	if offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			query = query.Offset(offset)
		}
	}

	var todos []models.Todo
	if err := query.Order("status ASC, created_at DESC").Find(&todos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(todos)
}

func (s *Server) getTodo(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid todo id"})
	}

	var todo models.Todo
	if err := s.db.First(&todo, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "todo not found"})
	}

	return c.JSON(todo)
}

func (s *Server) createTodo(c *fiber.Ctx) error {
	var payload models.Todo
	if msg := bindJSONRequired(c, &payload); msg != "" {
		return badPayload(c, msg)
	}

	payload.Title = strings.TrimSpace(payload.Title)
	if payload.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}
	if msg := firstProblem(maxLen("title", payload.Title, maxLabel), maxLen("description", payload.Description, maxText),
		oneOf("level", payload.Level, models.TodoLevelGlobal, models.TodoLevelProject, models.TodoLevelComponent),
		oneOf("status", payload.Status, todoStatuses...),
		oneOf("priority", payload.Priority, todoPriorities...)); msg != "" {
		return badPayload(c, msg)
	}

	if payload.Level == "" {
		if payload.ComponentID != nil && *payload.ComponentID > 0 {
			payload.Level = models.TodoLevelComponent
		} else if payload.ProjectID != nil && *payload.ProjectID > 0 {
			payload.Level = models.TodoLevelProject
		} else {
			payload.Level = models.TodoLevelGlobal
		}
	}

	if payload.Status == "" {
		if payload.IsAISuggestion {
			payload.Status = models.TodoStatusSuggested
		} else {
			payload.Status = models.TodoStatusPending
		}
	}
	if payload.Priority == "" {
		payload.Priority = models.TodoPriorityMedium
	}
	if payload.CreatorType == "" {
		payload.CreatorType = models.CreatorHuman
	}
	if payload.CreatorName == "" {
		payload.CreatorName = "developer"
	}
	if payload.Category == "" {
		payload.Category = models.CategoryGeneral
	}

	// Resolve project name
	if payload.ProjectID != nil && *payload.ProjectID > 0 {
		var p models.Project
		if err := s.db.Select("name").First(&p, *payload.ProjectID).Error; err == nil {
			payload.ProjectName = p.Name
		}
	}

	// Resolve component name
	if payload.ComponentID != nil && *payload.ComponentID > 0 {
		var comp models.Component
		if err := s.db.Select("name, project_id").First(&comp, *payload.ComponentID).Error; err == nil {
			payload.ComponentName = comp.Name
			if payload.ProjectID == nil || *payload.ProjectID == 0 {
				payload.ProjectID = &comp.ProjectID
				var p models.Project
				if err := s.db.Select("name").First(&p, comp.ProjectID).Error; err == nil {
					payload.ProjectName = p.Name
				}
			}
		}
	}

	if payload.TargetApps == "" {
		if payload.ComponentName != "" {
			payload.TargetApps = fmt.Sprintf("%s [1 App]", payload.ComponentName)
		} else if payload.ProjectName != "" {
			payload.TargetApps = fmt.Sprintf("Seluruh Aplikasi (Bundle %s)", payload.ProjectName)
		} else {
			payload.TargetApps = "Global Platform"
		}
	}

	if payload.Status == models.TodoStatusCompleted && payload.CompletedAt == nil {
		now := time.Now()
		payload.CompletedAt = &now
	}

	if err := s.db.Create(&payload).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(payload)
}

func (s *Server) updateTodo(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid todo id"})
	}

	var updates map[string]interface{}
	if msg := bindJSONRequired(c, &updates); msg != "" {
		return badPayload(c, msg)
	}
	if msg := validateTodoUpdates(updates); msg != "" {
		return badPayload(c, msg)
	}

	var todo models.Todo
	if err := s.db.First(&todo, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "todo not found"})
	}

	delete(updates, "id")
	if raw, ok := updates["tags"]; ok {
		updates["tags"] = models.NormalizeTags(raw)
	}

	newStatus, hasStatus := updates["status"].(string)
	if hasStatus && newStatus != "" {
		if newStatus == models.TodoStatusCompleted {
			if _, hasCompAt := updates["completed_at"]; !hasCompAt || updates["completed_at"] == nil {
				now := time.Now()
				updates["completed_at"] = &now
			}
			if _, hasAgent := updates["completed_by_agent"]; !hasAgent && todo.CompletedByAgent == "" {
				updates["completed_by_agent"] = "developer"
			}
		} else if newStatus == models.TodoStatusPending || newStatus == models.TodoStatusInProgress {
			updates["completed_at"] = nil
			updates["completed_by_agent"] = ""
			updates["completed_by_model"] = ""
		}
	}

	if err := s.db.Model(&todo).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	s.db.First(&todo, id)
	if rescoreTodo(&todo, updates) {
		s.db.Model(&todo).Updates(todo.TodoComplexity.Columns())
		s.db.Model(&todo).Updates(todo.TodoImpact.Columns())
	}

	if newStatus == models.TodoStatusCompleted {
		// Modul Pro: penugasan pegawai, penilaian QA, dan pengetahuan eksekutif.
		registry.TodoCompleted(s.db, todo.ID, todo.CompletedByAgent)
		s.db.First(&todo, id)
	}

	return c.JSON(todo)
}

// rescoreTodo menghitung ulang skor kerumitan/dampak bila dimensinya atau kategori diubah lewat update.
// Dimensi yang diisi tangan tanpa sumber eksplisit dicatat sebagai penilaian manual.
func rescoreTodo(todo *models.Todo, updates map[string]interface{}) bool {
	touched := false
	for k := range updates {
		switch {
		case strings.HasPrefix(k, "complexity_") && k != "complexity_source":
			if _, ok := updates["complexity_source"]; !ok && k != "complexity_note" {
				todo.ComplexitySource = models.ComplexitySourceManual
			}
			touched = true
		case strings.HasPrefix(k, "impact_") && k != "impact_source":
			if _, ok := updates["impact_source"]; !ok && k != "impact_summary" {
				todo.ImpactSource = models.ComplexitySourceManual
			}
			touched = true
		case k == "category":
			touched = true
		}
	}
	if touched {
		todo.Rescore()
	}
	return touched
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

	todo.Status = targetStatus
	if targetStatus == models.TodoStatusCompleted && todo.CompletedAt == nil {
		now := time.Now()
		todo.CompletedAt = &now
		if todo.CompletedByAgent == "" {
			todo.CompletedByAgent = "developer"
		}
	} else if targetStatus == models.TodoStatusPending || targetStatus == models.TodoStatusInProgress {
		todo.CompletedAt = nil
		todo.CompletedByAgent = ""
		todo.CompletedByModel = ""
	}

	if err := s.db.Save(&todo).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "approved", "todo": todo})
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

func (s *Server) deleteTodo(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid todo id"})
	}

	if _, err := models.DeleteTodo(s.db, uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "deleted", "id": id})
}

// getGuidelines returns all grouped guidelines from the guidelines directory
func (s *Server) getGuidelines(c *fiber.Ctx) error {
	categories, err := guidelines.LoadAllGuidelines()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(categories)
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
		for i := range allProjects {
			if allProjects[i].ID == body.ProjectID {
				targetProject = &allProjects[i]
				break
			}
		}
		if targetProject != nil {
			suggestions = append(suggestions, guidelines.GenerateSuggestionsForProject(targetProject, allProjects)...)
			// Standar vibeasik (Aturan 45): port, env & todo pending dipastikan langsung, bukan saran
			if r, _ := standard.Ensure(s.db, targetProject.ID); r != nil {
				suggestions = append(suggestions, r.Todos...)
			}
		}
	} else {
		// Global suggestions
		suggestions = append(suggestions, guidelines.GenerateGlobalSuggestions(allProjects)...)
		// Suggestions across all projects
		for i := range allProjects {
			suggestions = append(suggestions, guidelines.GenerateSuggestionsForProject(&allProjects[i], allProjects)...)
			// Standar vibeasik (Aturan 45): port, env & todo pending dipastikan langsung, bukan saran
			if r, _ := standard.Ensure(s.db, allProjects[i].ID); r != nil {
				suggestions = append(suggestions, r.Todos...)
			}
		}
	}

	// Saran cerdas (rapat Project Manager / LLM) disediakan modul Pro; Community hanya saran rule.
	var pmSession *models.PlanningSession
	pmJob, pmError := "", ""
	suggestions = append(suggestions, registry.SuggestTodos(s.db, targetProject, allProjects)...)

	// Filter out suggestions that already exist in DB with same title & project_id
	finalSuggestions := make([]models.Todo, 0)
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
			// Update category and target_apps if existing row was empty
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
				s.db.Model(&existing).Updates(updates)
			}
			finalSuggestions = append(finalSuggestions, existing)
		}
	}

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

// validateTodoUpdates memeriksa bentuk field umum pada PUT /todos/:id (kunci lain diteruskan apa adanya).
func validateTodoUpdates(u map[string]interface{}) string {
	str := func(k string) (string, string) {
		v, ok := u[k]
		if !ok || v == nil {
			return "", ""
		}
		sv, isStr := v.(string)
		if !isStr {
			return "", "field " + k + " harus berupa teks"
		}
		return sv, ""
	}
	for _, k := range []string{"title", "description", "status", "priority", "level"} {
		if _, m := str(k); m != "" {
			return m
		}
	}
	title, _ := str("title")
	desc, _ := str("description")
	status, _ := str("status")
	priority, _ := str("priority")
	level, _ := str("level")
	if _, has := u["title"]; has && u["title"] != nil && strings.TrimSpace(title) == "" {
		return "field title tidak boleh kosong"
	}
	return firstProblem(maxLen("title", title, maxLabel), maxLen("description", desc, maxText),
		oneOf("status", status, todoStatuses...), oneOf("priority", priority, todoPriorities...),
		oneOf("level", level, models.TodoLevelGlobal, models.TodoLevelProject, models.TodoLevelComponent))
}
