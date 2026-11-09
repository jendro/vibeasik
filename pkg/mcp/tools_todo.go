package mcp

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/todorag"
	"github.com/vibeasik/vibeasik/pkg/util"
	"github.com/vibeasik/vibeasik/pkg/workspace"
)

// ---- list_todos ----

func (s *Server) toolListTodos(args map[string]interface{}) (*ToolCallResult, error) {
	query := s.db.Model(&models.Todo{})
	if lvl, ok := args["level"].(string); ok && lvl != "" {
		query = query.Where("level = ?", lvl)
	}
	resolvedPID, query := s.scopeTodoListToProject(query, args)
	if resolvedPID > 0 {
		query = query.Where("project_id = ?", resolvedPID)
	}
	if cid := argUint(args, "component_id"); cid > 0 {
		query = query.Where("component_id = ?", cid)
	}
	search := strings.TrimSpace(argRaw(args, "query"))
	query = filterTodoStatus(query, args, search)
	query = filterTodoAttributes(query, args)

	limit := intArg(args, "limit")
	if limit <= 0 {
		limit = listTodosDefaultLimit
	}
	limit = min(limit, listTodosMaxLimit)
	var total int64
	query.Session(&gorm.Session{}).Count(&total)

	var todos []models.Todo
	sims := map[uint]float64{}
	if search != "" {
		todos = s.similarTodos(query, resolvedPID, search, limit, sims)
		total = int64(len(todos))
	} else if err := query.Order("status ASC, created_at DESC").Limit(limit).Find(&todos).Error; err != nil {
		return nil, err
	}
	out := presentTodoList(args, todos, sims, search)
	return jsonResult(map[string]interface{}{"total": total, "returned": len(todos), "todos": out})
}

// argRaw membaca argumen string apa adanya (kosong bila bukan string).
func argRaw(args map[string]interface{}, key string) string {
	v, _ := args[key].(string)
	return v
}

// scopeTodoListToProject menentukan project_id: dari argumen, atau dikenali dari path workspace
// (sekaligus membatasi komponen aktif bila component_id tidak diberikan).
func (s *Server) scopeTodoListToProject(query *gorm.DB, args map[string]interface{}) (uint, *gorm.DB) {
	if pidVal, ok := args["project_id"]; ok && pidVal != nil {
		return argUint(args, "project_id"), query
	}
	targetPath, _ := args["path"].(string)
	match, err := workspace.IdentifyWorkspace(s.db, targetPath, localEngineURL)
	if err != nil || match.Project == nil {
		return 0, query
	}
	if match.CurrentComponent != nil && args["component_id"] == nil {
		query = query.Where("component_id = ? OR component_id IS NULL OR component_id = 0", match.CurrentComponent.ID)
	}
	return match.Project.ID, query
}

func filterTodoStatus(query *gorm.DB, args map[string]interface{}, search string) *gorm.DB {
	status, _ := args["status"].(string)
	includeClosed, _ := args["include_closed"].(bool)
	if status != "" {
		return query.Where("status = ?", status)
	}
	if search == "" && !includeClosed {
		// Default: hanya todo yang masih terbuka; riwayat selesai/ditolak dicari lewat query atau include_closed.
		return query.Where("status NOT IN ?", []string{models.TodoStatusCompleted, models.TodoStatusRejected})
	}
	return query
}

// activeFilter mengembalikan nilai filter string yang bermakna (tidak kosong dan bukan "all").
func activeFilter(args map[string]interface{}, key string) (string, bool) {
	v, ok := args[key].(string)
	v = strings.TrimSpace(v)
	if !ok || v == "" || strings.ToLower(v) == "all" {
		return "", false
	}
	return v, true
}

func filterTodoAttributes(query *gorm.DB, args map[string]interface{}) *gorm.DB {
	if creatorType, ok := args["creator_type"].(string); ok && creatorType != "" {
		query = query.Where("creator_type = ?", creatorType)
	}
	if isSugg, ok := args["is_ai_suggestion"].(bool); ok {
		query = query.Where("is_ai_suggestion = ?", isSugg)
	}
	if src, ok := activeFilter(args, "source"); ok {
		query = query.Where("source = ?", src)
	}
	if cat, ok := activeFilter(args, "category"); ok {
		query = models.WhereCategory(query, cat)
	}
	if tag, ok := args["tag"].(string); ok && tag != "all" {
		query = models.WhereTag(query, tag)
	}
	return query
}

// similarTodos mengurutkan todo menurut kemiripan dengan kueri di semua status (RAG: sudah pernah
// dikerjakan / ditolak?) dan mengisi sims dengan skor tiap todo.
func (s *Server) similarTodos(query *gorm.DB, pid uint, search string, limit int, sims map[uint]float64) []models.Todo {
	hits := todorag.Similar(s.db, pid, search, search, limit, todorag.QuerySim)
	ids := make([]uint, 0, len(hits))
	for _, h := range hits {
		ids = append(ids, h.ID)
		sims[h.ID] = h.Sim
	}
	var found []models.Todo
	if len(ids) > 0 {
		query.Where("id IN ?", ids).Find(&found)
	}
	byID := map[uint]models.Todo{}
	for _, t := range found {
		byID[t.ID] = t
	}
	var todos []models.Todo
	for _, id := range ids {
		if t, ok := byID[id]; ok {
			todos = append(todos, t)
		}
	}
	return todos
}

// presentTodoList mengembalikan todo utuh (detail: true) atau bentuk ringkas.
func presentTodoList(args map[string]interface{}, todos []models.Todo, sims map[uint]float64, search string) interface{} {
	if detail, _ := args["detail"].(bool); detail {
		return todos
	}
	compact := make([]compactTodo, len(todos))
	for i, t := range todos {
		compact[i] = toCompactTodo(t)
		compact[i].Similarity = sims[t.ID]
		if search == "" {
			compact[i].Memory = util.OneLine(t.Memory, listMemoryMax) // daftar biasa: cukup ringkasan; kueri: memori utuh
		}
	}
	return compact
}

// ---- create_todo ----

func (s *Server) toolCreateTodo(args map[string]interface{}) (*ToolCallResult, error) {
	title := argString(args, "title")
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	todo := newTodoFromArgs(args, title)
	s.attachTodoProject(&todo, args)
	s.attachTodoComponent(&todo, args)
	if parentID := argUint(args, "parent_id"); parentID > 0 {
		todo.ParentID = &parentID
	}
	todo.Level = todoLevelFor(&todo, argRaw(args, "level"))
	todo.ApplyLLMRatings(complexityArg(args), impactArg(args), models.ComplexitySourceManual)

	dups := todorag.Duplicates(s.db, todo)
	if force, _ := args["force"].(bool); !force && len(dups) > 0 && dups[0].Sim >= todorag.BlockSim && todo.Status != models.TodoStatusCompleted {
		return textResult("Todo TIDAK dibuat: kemungkinan besar duplikat. " + duplicateNotice(dups) +
			"Bila todo existing masih terbuka, kerjakan/perbarui itu. Bila memang pekerjaan berbeda, panggil create_todo lagi dengan force: true."), nil
	}
	if err := s.db.Create(&todo).Error; err != nil {
		return nil, err
	}
	res := prettyJSONText(todo)
	if len(dups) > 0 {
		res.Content = append(res.Content, ToolContent{Type: "text", Text: duplicateNotice(dups)})
	}
	return res, nil
}

// strOr mengembalikan argumen string, atau def bila kosong/bukan string.
func strOr(args map[string]interface{}, key, def string) string {
	if v := argRaw(args, key); v != "" {
		return v
	}
	return def
}

// newTodoFromArgs mengisi kolom todo yang tidak butuh akses database.
func newTodoFromArgs(args map[string]interface{}, title string) models.Todo {
	isAISugg, _ := args["is_ai_suggestion"].(bool)
	status := models.TodoStatusPending
	if isAISugg {
		status = models.TodoStatusSuggested
	}
	todo := models.Todo{
		Title:          title,
		Description:    argRaw(args, "description"),
		Status:         status,
		Priority:       strOr(args, "priority", models.TodoPriorityMedium),
		CreatorType:    strOr(args, "creator_type", models.CreatorAI),
		CreatorName:    strOr(args, "creator_name", "ai_agent"),
		IsAISuggestion: isAISugg,
		GuidelineRef:   argRaw(args, "guideline_ref"),
		Rationale:      argRaw(args, "rationale"),
		Source:         strOr(args, "source", "ai_agent"),
		Category:       models.CategoryGeneral,
	}
	if cat := argString(args, "category"); cat != "" {
		todo.Category = strings.ToLower(cat)
	}
	if raw, ok := args["tags"]; ok {
		todo.Tags = models.NormalizeTags(raw)
	}
	applyCompletionArgs(&todo, args)
	return todo
}

// applyCompletionArgs mencatat siapa/kapan todo diselesaikan bila klien menyebutkannya.
func applyCompletionArgs(todo *models.Todo, args map[string]interface{}) {
	if v := argRaw(args, "completed_by_agent"); v != "" {
		todo.CompletedByAgent = v
	}
	if v := argRaw(args, "completed_by_model"); v != "" {
		todo.CompletedByModel = v
	}
	if at := argRaw(args, "completed_at"); at != "" {
		if parsedTime, err := time.Parse(time.RFC3339, at); err == nil {
			todo.CompletedAt = &parsedTime
		}
	} else if todo.Status == models.TodoStatusCompleted {
		now := time.Now()
		todo.CompletedAt = &now
	}
}

func (s *Server) attachTodoProject(todo *models.Todo, args map[string]interface{}) {
	pid := argUint(args, "project_id")
	if pid == 0 {
		return
	}
	todo.ProjectID = &pid
	var p models.Project
	if err := s.db.Select("name").First(&p, pid).Error; err == nil {
		todo.ProjectName = p.Name
	}
}

func (s *Server) attachTodoComponent(todo *models.Todo, args map[string]interface{}) {
	cid := argUint(args, "component_id")
	if cid == 0 {
		return
	}
	todo.ComponentID = &cid
	var comp models.Component
	if err := s.db.Select("name, project_id").First(&comp, cid).Error; err != nil {
		return
	}
	todo.ComponentName = comp.Name
	if todo.ProjectID != nil && *todo.ProjectID != 0 {
		return
	}
	todo.ProjectID = &comp.ProjectID
	var p models.Project
	if err := s.db.Select("name").First(&p, comp.ProjectID).Error; err == nil {
		todo.ProjectName = p.Name
	}
}

// todoLevelFor memakai level yang diminta, atau menurunkannya dari komponen/proyek todo.
func todoLevelFor(todo *models.Todo, level string) string {
	switch {
	case level != "":
		return level
	case todo.ComponentID != nil && *todo.ComponentID > 0:
		return models.TodoLevelComponent
	case todo.ProjectID != nil && *todo.ProjectID > 0:
		return models.TodoLevelProject
	}
	return models.TodoLevelGlobal
}

// ---- update_todo ----

func (s *Server) toolUpdateTodo(args map[string]interface{}) (*ToolCallResult, error) {
	tid := argUint(args, "todo_id")
	if tid == 0 {
		return nil, fmt.Errorf("valid todo_id is required")
	}
	var todo models.Todo
	if err := s.db.First(&todo, tid).Error; err != nil {
		return nil, fmt.Errorf("todo with ID %d not found", tid)
	}

	newStatus, hasStatus := args["status"].(string)
	updates := buildTodoUpdates(args, &todo, hasStatus, newStatus)
	if len(updates) > 0 {
		if err := s.db.Model(&todo).Updates(updates).Error; err != nil {
			return nil, err
		}
		s.db.First(&todo, tid)
		if updates["rationale"] != nil {
			todo.Memory = models.RefreshTodoMemory(s.db, todo.ID)
		}
	}
	s.rateUpdatedTodo(&todo, args, updates)

	res := prettyJSONText(todo)
	// Selesai tanpa pernah ditandai in_progress (agent langsung menutup todo) → tetap dicatat atas nama
	// pegawai agar ikut dinilai QA dan pengetahuannya diperbarui.
	agent := firstNonEmptyStr(args["completed_by_agent"], todo.CompletedByAgent, "agent coding")
	if hasStatus && newStatus == models.TodoStatusCompleted {
		registry.TodoCompleted(s.db, tid, agent)
	}
	// Mulai dikerjakan → modul Pro dapat menugaskan todo ke pegawai vibeasik atau memecahnya per aplikasi.
	if hasStatus && newStatus == models.TodoStatusInProgress {
		if note, _ := registry.TodoStarted(s.db, tid, agent); note != "" {
			res.Content = append(res.Content, ToolContent{Type: "text", Text: note})
		}
	}
	return res, nil
}

// rateUpdatedTodo menerapkan penilaian kompleksitas/dampak dari klien (atau penilaian ulang
// bila kategori berubah) dan menyimpan kolomnya.
func (s *Server) rateUpdatedTodo(todo *models.Todo, args, updates map[string]interface{}) {
	c, i := complexityArg(args), impactArg(args)
	if c == nil && i == nil && updates["category"] == nil {
		return
	}
	todo.ApplyLLMRatings(c, i, models.ComplexitySourceManual)
	todo.Rescore()
	s.db.Model(todo).Updates(todo.TodoComplexity.Columns())
	s.db.Model(todo).Updates(todo.TodoImpact.Columns())
}

// buildTodoUpdates menyusun kolom yang diubah dari argumen update_todo.
func buildTodoUpdates(args map[string]interface{}, todo *models.Todo, hasStatus bool, newStatus string) map[string]interface{} {
	updates := make(map[string]interface{})
	if hasStatus && newStatus != "" {
		updates["status"] = newStatus
		applyStatusTransition(updates, args, todo, newStatus)
	}
	if agent := argRaw(args, "completed_by_agent"); agent != "" {
		updates["completed_by_agent"] = agent
	} else if hasStatus && newStatus == models.TodoStatusCompleted && todo.CompletedByAgent == "" {
		updates["completed_by_agent"] = "ai_agent"
	}
	if model := argRaw(args, "completed_by_model"); model != "" {
		updates["completed_by_model"] = model
	}
	if at := argRaw(args, "completed_at"); at != "" && updates["completed_at"] == nil {
		if parsedTime, err := time.Parse(time.RFC3339, at); err == nil {
			updates["completed_at"] = &parsedTime
		}
	}
	applyFieldUpdates(updates, args)
	if note := argString(args, "completion_note"); note != "" {
		who := firstNonEmptyStr(args["completed_by_agent"], todo.CompletedByAgent, "agent")
		entry := fmt.Sprintf("Catatan penyelesaian (%s, %s): %s", who, time.Now().Format("2 Jan 2006 15:04"), note)
		updates["rationale"] = strings.TrimSpace(todo.Rationale + "\n\n" + entry)
	}
	return updates
}

// applyStatusTransition mengatur completed_at dan penyelesai sesuai status baru.
func applyStatusTransition(updates, args map[string]interface{}, todo *models.Todo, newStatus string) {
	switch newStatus {
	case models.TodoStatusCompleted:
		if at := argRaw(args, "completed_at"); at != "" {
			if parsedTime, err := time.Parse(time.RFC3339, at); err == nil {
				updates["completed_at"] = &parsedTime
			} else {
				now := time.Now()
				updates["completed_at"] = &now
			}
		} else if todo.CompletedAt == nil {
			now := time.Now()
			updates["completed_at"] = &now
		}
	case models.TodoStatusPending, models.TodoStatusInProgress:
		updates["completed_at"] = nil
		updates["completed_by_agent"] = ""
		updates["completed_by_model"] = ""
	}
}

// applyFieldUpdates menyalin judul, kategori, tag, dan prioritas yang diberikan klien.
func applyFieldUpdates(updates, args map[string]interface{}) {
	if title := argString(args, "title"); title != "" {
		updates["title"] = title
	}
	if cat := argString(args, "category"); cat != "" {
		updates["category"] = strings.ToLower(cat)
	}
	if raw, ok := args["tags"]; ok {
		updates["tags"] = models.NormalizeTags(raw)
	}
	if priority := argRaw(args, "priority"); priority != "" {
		updates["priority"] = priority
	}
}
