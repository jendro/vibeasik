package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
)

func (s *Server) getTodos(c *fiber.Ctx) error {
	query := buildTodoFilterQuery(s.db.Model(&models.Todo{}), c)

	var todos []models.Todo
	// Todo terbuka didahulukan: hasil selalu dibatasi (applyTodoPagination), jadi urutan status alfabetis
	// membuat batas terisi todo 'completed' dan todo pending/in_progress hilang dari halaman todos.
	if err := query.Order(todoOpenFirstOrder).Find(&todos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(todos)
}

// todoOpenFirstOrder: todo yang masih terbuka di atas, todo selesai/ditolak di bawah, terbaru lebih dulu.
const todoOpenFirstOrder = "CASE WHEN status IN ('completed', 'rejected') THEN 1 ELSE 0 END, status ASC, created_at DESC, id DESC"

func buildTodoFilterQuery(query *gorm.DB, c *fiber.Ctx) *gorm.DB {
	query = applyTodoStringFilters(query, c)
	query = applyTodoEntityFilters(query, c)
	return applyTodoPagination(query, c)
}

func applyTodoStringFilters(query *gorm.DB, c *fiber.Ctx) *gorm.DB {
	if level := c.Query("level"); level != "" {
		query = query.Where("level = ?", level)
	}
	if category := c.Query("category"); category != "" && category != "all" {
		query = models.WhereCategory(query, category)
	}
	if source := c.Query("source"); source != "" && source != "all" {
		query = query.Where("source = ?", source)
	}
	if tag := c.Query("tag"); tag != "" && tag != "all" {
		query = models.WhereTag(query, tag)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if priority := c.Query("priority"); priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if creatorType := c.Query("creator_type"); creatorType != "" {
		query = query.Where("creator_type = ?", creatorType)
	}
	return query
}

func applyTodoEntityFilters(query *gorm.DB, c *fiber.Ctx) *gorm.DB {
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if pid, err := strconv.Atoi(projectIDStr); err == nil {
			query = query.Where("project_id = ?", pid)
		}
	}
	if componentIDStr := c.Query("component_id"); componentIDStr != "" {
		if cid, err := strconv.Atoi(componentIDStr); err == nil {
			query = query.Where("component_id = ?", cid)
		}
	}
	if isSuggestionStr := c.Query("is_ai_suggestion"); isSuggestionStr != "" {
		isSuggestion := isSuggestionStr == "true" || isSuggestionStr == "1"
		query = query.Where("is_ai_suggestion = ?", isSuggestion)
	}
	return query
}

const (
	// defaultTodoLimit: jumlah baris default GET /api/todos agar query tidak
	// pernah menjadi unbounded result set (High Performance MySQL, Ch.6).
	defaultTodoLimit = 50
	// maxTodoLimit: plafon baris per request supaya klien tidak menarik
	// seluruh tabel dalam satu panggilan.
	maxTodoLimit = 200
)

// todoQueryLimit menerjemahkan ?limit menjadi jumlah baris yang SELALU terbatas:
// kosong/tidak valid -> defaultTodoLimit, melebihi plafon -> maxTodoLimit.
func todoQueryLimit(c *fiber.Ctx) int {
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit <= 0 {
		return defaultTodoLimit
	}
	if limit > maxTodoLimit {
		return maxTodoLimit
	}
	return limit
}

func applyTodoPagination(query *gorm.DB, c *fiber.Ctx) *gorm.DB {
	query = query.Limit(todoQueryLimit(c))
	if offset, err := strconv.Atoi(c.Query("offset")); err == nil && offset > 0 {
		query = query.Offset(offset)
	}
	return query
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

	query := s.db.Where("project_id = ?", id)
	query = applyTodoStringFilters(query, c)
	query = applyTodoPagination(query, c)

	var todos []models.Todo
	if err := query.Order(todoOpenFirstOrder).Find(&todos).Error; err != nil {
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
	if denied, resp := validateCreateTodo(c, &payload); denied {
		return resp
	}

	applyTodoDefaults(&payload)
	s.resolveTodoEntities(c, &payload)
	finalizeTodoPayload(&payload)

	if err := s.db.WithContext(c.Context()).Create(&payload).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(payload)
}

func validateCreateTodo(c *fiber.Ctx, payload *models.Todo) (bool, error) {
	payload.Title = strings.TrimSpace(payload.Title)
	if payload.Title == "" {
		return true, c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}
	if msg := firstProblem(
		maxLen("title", payload.Title, maxLabel),
		maxLen("description", payload.Description, maxText),
		oneOf("level", payload.Level, models.TodoLevelGlobal, models.TodoLevelProject, models.TodoLevelComponent),
		oneOf("status", payload.Status, todoStatuses...),
		oneOf("priority", payload.Priority, todoPriorities...),
	); msg != "" {
		return true, badPayload(c, msg)
	}
	return false, nil
}

func applyTodoDefaults(payload *models.Todo) {
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
}

// todoEntityRefs menampung nama entity yang sudah didenormalisasi untuk payload todo.
type todoEntityRefs struct {
	ComponentName string
	ProjectID     uint
	ProjectName   string
}

// resolveTodoEntities mengisi ProjectID/ProjectName/ComponentName dengan maksimal dua
// round-trip: satu JOIN komponen→proyek, satu lookup proyek bila belum terisi.
func (s *Server) resolveTodoEntities(c *fiber.Ctx, payload *models.Todo) {
	refs := s.lookupTodoEntityRefs(c, payload)
	payload.ComponentName = refs.ComponentName
	payload.ProjectName = refs.ProjectName

	if payload.ProjectID != nil && *payload.ProjectID > 0 {
		return
	}
	if refs.ProjectID == 0 {
		return
	}
	id := refs.ProjectID
	payload.ProjectID = &id
}

func (s *Server) lookupTodoEntityRefs(c *fiber.Ctx, payload *models.Todo) todoEntityRefs {
	var refs todoEntityRefs
	db := s.db.WithContext(c.Context())

	if payload.ComponentID != nil && *payload.ComponentID > 0 {
		db.Table("components").
			Select("components.name AS component_name, components.project_id, projects.name AS project_name").
			Joins("LEFT JOIN projects ON projects.id = components.project_id").
			Where("components.id = ?", *payload.ComponentID).
			Limit(1).
			Scan(&refs)
	}

	needProject := payload.ProjectID != nil && *payload.ProjectID > 0 && refs.ProjectName == ""
	if needProject {
		db.Model(&models.Project{}).Select("name").Where("id = ?", *payload.ProjectID).Limit(1).Pluck("name", &refs.ProjectName)
	}

	return refs
}

func finalizeTodoPayload(payload *models.Todo) {
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
}

func adjustTodoStatusFields(updates map[string]interface{}, newStatus string, existingAgent string) {
	switch newStatus {
	case models.TodoStatusCompleted:
		if updates["completed_at"] == nil {
			now := time.Now()
			updates["completed_at"] = &now
		}
		if _, hasAgent := updates["completed_by_agent"]; !hasAgent && existingAgent == "" {
			updates["completed_by_agent"] = "developer"
		}
	case models.TodoStatusPending, models.TodoStatusInProgress:
		updates["completed_at"] = nil
		updates["completed_by_agent"] = ""
		updates["completed_by_model"] = ""
	}
}

func normalizeTodoUpdatePayload(updates map[string]interface{}, todo *models.Todo) string {
	delete(updates, "id")
	if raw, ok := updates["tags"]; ok {
		updates["tags"] = models.NormalizeTags(raw)
	}

	newStatus, hasStatus := updates["status"].(string)
	if hasStatus && newStatus != "" {
		adjustTodoStatusFields(updates, newStatus, todo.CompletedByAgent)
		return newStatus
	}
	return ""
}

func (s *Server) applyTodoPostUpdates(todo *models.Todo, id int, newStatus string, updates map[string]interface{}) {
	s.db.First(todo, id)
	if rescoreTodo(todo, updates) {
		s.db.Model(todo).Updates(todo.TodoComplexity.Columns())
		s.db.Model(todo).Updates(todo.TodoImpact.Columns())
	}

	if newStatus == models.TodoStatusCompleted {
		// Modul Pro: penugasan pegawai, penilaian QA, dan pengetahuan eksekutif.
		registry.TodoCompleted(s.db, todo.ID, todo.CompletedByAgent)
		s.db.First(todo, id)
	}
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

	newStatus := normalizeTodoUpdatePayload(updates, &todo)

	if err := s.db.Model(&todo).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	s.applyTodoPostUpdates(&todo, id, newStatus, updates)

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
