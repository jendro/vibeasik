package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
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
