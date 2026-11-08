package activity

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/util"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

const todoDonePrefix = "Selesai: "

// ReportToolName adalah tool MCP yang dipakai agent untuk mengumumkan pekerjaannya sendiri.
const ReportToolName = "vibeasik_report_activity"

// Report adalah laporan aktivitas eksplisit dari agent (tool MCP, REST, atau hook CLI).
type Report struct {
	AgentName   string `json:"agent_name"`
	SessionID   string `json:"session_id"`
	Model       string `json:"model"`
	Status      string `json:"status"` // working | idle | done | left
	Action      string `json:"action"`
	Task        string `json:"task"`
	TodoID      uint   `json:"todo_id"`
	ProjectID   uint   `json:"project_id"`
	ProjectName string `json:"project_name"`
	ProjectPath string `json:"project_path"` // direktori kerja agent; dicocokkan ke root_path proyek vibeasik
	Transport   string `json:"transport"`
	// ParentSessionID: diisi sub agent (mode divisi) dengan session_id koordinator yang menyuruhnya.
	ParentSessionID string `json:"parent_session_id"`
}

var knownAgents = []struct{ needle, name string }{
	{"claude-code", "Claude Code"},
	{"claude code", "Claude Code"},
	{"claude-ai", "Claude Desktop"},
	{"claude desktop", "Claude Desktop"},
	{"antigravity", "Antigravity"},
	{"agy", "Antigravity"},
	{"cursor", "Cursor"},
	{"windsurf", "Windsurf"},
	{"cline", "Cline"},
	{"aider", "Aider"},
	{"codex", "Codex"},
	{"gemini", "Gemini CLI"},
	{"copilot-cli", "Copilot CLI"},
	{"copilot cli", "Copilot CLI"},
	{"copilot", "Copilot Chat"},
	{"visual studio code", "Copilot Chat"}, // klien MCP VS Code = agent Copilot Chat
	{"rea", "Rea"},
	{"claude", "Claude Code"},
}

// FriendlyAgentName memetakan nama klien mentah (clientInfo MCP / key adapter) ke nama tampilan.
func FriendlyAgentName(raw string) string {
	l := strings.ToLower(strings.TrimSpace(raw))
	if l == "" {
		return "Agent MCP"
	}
	for _, k := range knownAgents {
		if l == k.needle || (len(k.needle) > 3 && strings.Contains(l, k.needle)) {
			return k.name
		}
	}
	words := strings.FieldsFunc(raw, func(r rune) bool { return r == '-' || r == '_' || r == ' ' })
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return truncate(strings.Join(words, " "), maxGenericLength)
}

// ClientConnected dipanggil saat MCP `initialize`: agent masuk kantor.
func ClientConnected(c Caller, clientName, version string) {
	d := getDB()
	if d == nil || c.Key == "" {
		return
	}
	name := FriendlyAgentName(clientName)
	upsertWorker(d, c.Key, models.AgentWorker{Kind: models.WorkerKindCodingAgent}, func(w *models.AgentWorker) {
		w.Name = name
		w.ClientName = truncate(strings.TrimSpace(clientName+" "+version), maxGenericLength)
		w.Transport = c.Transport
		w.Status = models.WorkerStatusIdle
		w.Action = "Terhubung ke vibeasik MCP"
	})
}

// ClientDisconnected menandai sesi MCP berakhir (agent pulang). Tidak membuat baris baru.
func ClientDisconnected(c Caller) {
	d := getDB()
	if d == nil || c.Key == "" {
		return
	}
	now := time.Now()
	d.Model(&models.AgentWorker{}).Where("worker_key = ? AND status <> ?", c.Key, models.WorkerStatusLeft).Updates(map[string]interface{}{
		"status":       models.WorkerStatusLeft,
		"finished_at":  now,
		"last_seen_at": now,
	})
}

// ToolCalled dipanggil sebelum tool MCP dijalankan.
func ToolCalled(c Caller, tool string, args map[string]interface{}) {
	d := getDB()
	if d == nil {
		return
	}
	if tool == ReportToolName {
		ReportActivity(c, reportFromArgs(args))
		return
	}
	if c.Key == "" {
		return
	}
	upsertWorker(d, c.Key, models.AgentWorker{Kind: models.WorkerKindCodingAgent, Name: "Agent MCP", Transport: c.Transport}, func(w *models.AgentWorker) {
		w.Status = models.WorkerStatusWorking
		w.Action = tool
		w.ToolCalls++
		if pid := argUint(args, "project_id"); pid > 0 && (w.ProjectID == nil || *w.ProjectID != pid) {
			setProject(w, pid)
		}
		if tool == "update_todo" {
			applyTodoUpdate(w, argUint(args, "todo_id"), argString(args, "status"))
		}
		if tool == DivisionPlanTool {
			w.Role = models.WorkerRoleCoordinator
		}
	})
	if tool == "update_todo" {
		linkTodoToObservedSibling(d, c.Key, argUint(args, "todo_id"), argString(args, "status"))
	}
}

// linkTodoToObservedSibling meneruskan update_todo dari koneksi MCP ke sesi agent yang sama yang
// terpantau dari transkripnya (karakter itulah yang tampil), yaitu sesi bernama sama yang paling baru aktif.
func linkTodoToObservedSibling(d *gorm.DB, mcpKey string, todoID uint, status string) {
	var conn models.AgentWorker
	if todoID == 0 || d.Select("name").Where("worker_key = ?", mcpKey).First(&conn).Error != nil {
		return
	}
	var sib models.AgentWorker
	err := d.Where("name = ? AND worker_key NOT LIKE ? AND parent_key = '' AND last_seen_at >= ?", conn.Name, "mcp-%", time.Now().Add(-IdleAfter*3)).
		Order("last_seen_at DESC").First(&sib).Error
	if err != nil {
		return
	}
	upsertWorker(d, sib.Key, sib, func(w *models.AgentWorker) {
		w.LastSeenAt = sib.LastSeenAt // waktu aktivitas tetap milik transkrip
		applyTodoUpdate(w, todoID, status)
	})
}

// ToolFinished dipanggil setelah tool MCP selesai; hanya mencatat error terakhir.
func ToolFinished(c Caller, tool string, err error) {
	d := getDB()
	if d == nil || c.Key == "" || tool == ReportToolName {
		return
	}
	updates := map[string]interface{}{"last_seen_at": time.Now(), "last_error": ""}
	if err != nil {
		updates["last_error"] = truncate(fmt.Sprintf("%s: %v", tool, err), maxTaskLength)
	}
	d.Model(&models.AgentWorker{}).Where("worker_key = ?", c.Key).Updates(updates)
}

// ReportActivity menerapkan laporan eksplisit agent. Tanpa Caller (REST/hook), key diturunkan
// dari session_id atau nama agent sehingga laporan berikutnya memperbarui orang yang sama.
func ReportActivity(c Caller, r Report) *models.AgentWorker {
	d := getDB()
	if d == nil {
		return nil
	}
	name := strings.TrimSpace(r.AgentName)
	key, parentKey := c.Key, ""
	if parent := strings.TrimSpace(r.ParentSessionID); parent != "" {
		// Sub agent berbagi koneksi MCP dengan koordinatornya, jadi key-nya selalu dari session_id sendiri.
		parentKey = "report:" + slug(name) + ":" + slug(parent)
		key = parentKey + ":sub:" + slug(util.FirstNonEmpty(r.SessionID, r.Task, "sub"))
	} else if key == "" {
		id := strings.TrimSpace(r.SessionID)
		if id == "" {
			id = name
		}
		if id == "" {
			return nil
		}
		key = "report:" + slug(name) + ":" + slug(id)
	}
	transport := c.Transport
	if transport == "" {
		transport = truncate(r.Transport, 30)
	}
	return upsertWorker(d, key, models.AgentWorker{Kind: models.WorkerKindCodingAgent, Name: FriendlyAgentName(name), Transport: transport}, func(w *models.AgentWorker) {
		if name != "" {
			w.Name = FriendlyAgentName(name)
			if w.ClientName == "" {
				w.ClientName = truncate(name, maxGenericLength)
			}
		}
		if r.Model != "" {
			w.Model = truncate(r.Model, maxGenericLength)
		}
		if r.Task != "" {
			w.Task = r.Task
		}
		if parentKey != "" {
			w.ParentKey, w.Role = parentKey, models.WorkerRoleSubagent
		}
		if r.Action != "" {
			w.Action = r.Action
		}
		if r.ProjectPath != "" {
			w.ProjectPath = truncate(r.ProjectPath, maxPathLength)
		}
		if r.ProjectID == 0 && r.ProjectPath != "" {
			r.ProjectID = projectIDForPath(r.ProjectPath)
		}
		if r.ProjectID > 0 {
			setProject(w, r.ProjectID)
		} else if r.ProjectName != "" {
			w.ProjectID = nil // proyek tidak terdaftar di vibeasik: jangan pertahankan tautan lama
			w.ProjectName = truncate(r.ProjectName, maxGenericLength)
		}
		status := strings.ToLower(strings.TrimSpace(r.Status))
		if r.TodoID > 0 {
			todoStatus := models.TodoStatusInProgress
			if status == models.WorkerStatusDone {
				todoStatus = models.TodoStatusCompleted
			}
			applyTodoUpdate(w, r.TodoID, todoStatus)
		}
		switch status {
		case models.WorkerStatusIdle:
			w.Status = models.WorkerStatusIdle
		case models.WorkerStatusDone:
			// "done" = satu pekerjaan selesai, agent tetap di kantor dan siap task berikutnya;
			// sub agent justru ditutup setelah melapor (satu sub agent = satu todo).
			if r.TodoID == 0 {
				w.TasksDone++
			}
			w.Status = models.WorkerStatusIdle
			if parentKey != "" {
				now := time.Now()
				w.Status, w.FinishedAt = models.WorkerStatusDone, &now
			}
		case models.WorkerStatusLeft:
			now := time.Now()
			w.Status = models.WorkerStatusLeft
			w.FinishedAt = &now
		default:
			w.Status = models.WorkerStatusWorking
		}
	})
}

// applyTodoUpdate menautkan todo ke pekerja: in_progress = sedang dikerjakan, completed = selesai.
func applyTodoUpdate(w *models.AgentWorker, todoID uint, status string) {
	d := getDB()
	if d == nil || todoID == 0 {
		return
	}
	var todo models.Todo
	if err := d.Select("id", "title", "project_id", "project_name").First(&todo, todoID).Error; err != nil {
		return
	}
	switch status {
	case models.TodoStatusCompleted:
		if w.Task != todoDonePrefix+todo.Title { // sumber kedua (MCP + transkrip) tidak menghitung ulang
			w.TasksDone++
		}
		w.Task = todoDonePrefix + todo.Title
		if w.TodoID != nil && *w.TodoID == todoID {
			w.TodoID = nil
		}
	case models.TodoStatusRejected, models.TodoStatusPending:
		if w.TodoID != nil && *w.TodoID == todoID {
			w.TodoID = nil
		}
	default:
		id := todo.ID
		w.TodoID = &id
		w.Task = todo.Title
	}
	if todo.ProjectID != nil {
		w.ProjectID = todo.ProjectID
		w.ProjectName = todo.ProjectName
	}
}

func setProject(w *models.AgentWorker, pid uint) {
	d := getDB()
	if d == nil {
		return
	}
	var p models.Project
	if err := d.Select("id", "name").First(&p, pid).Error; err == nil {
		w.ProjectID = &pid
		w.ProjectName = p.Name
	}
}

// projectIDForPath mencocokkan direktori kerja agent ke proyek vibeasik: lewat path absolut komponen
// (path itu sendiri atau di dalamnya), atau root proyek yang persis sama. Root proyek yang lebih umum
// (mis. ~/Documents) tidak dicocokkan agar repo lain di dalamnya tidak salah tertaut.
func projectIDForPath(path string) uint {
	d := getDB()
	if d == nil {
		return 0
	}
	clean := filepath.Clean(path)
	within := func(root string) bool {
		root = filepath.Clean(root)
		return clean == root || strings.HasPrefix(clean, root+string(filepath.Separator))
	}

	var comps []models.Component
	d.Select("id", "project_id", "absolute_path").Where("absolute_path <> ''").Find(&comps)
	best, bestLen := uint(0), 0
	for _, c := range comps {
		if within(c.AbsolutePath) && len(c.AbsolutePath) > bestLen {
			best, bestLen = c.ProjectID, len(c.AbsolutePath)
		}
	}
	if best > 0 {
		return best
	}

	var projects []models.Project
	d.Select("id", "root_path").Where("root_path <> ''").Find(&projects)
	for _, p := range projects {
		if filepath.Clean(p.RootPath) == clean {
			return p.ID
		}
	}
	return 0
}

func reportFromArgs(args map[string]interface{}) Report {
	return Report{
		AgentName:       argString(args, "agent_name"),
		SessionID:       argString(args, "session_id"),
		Model:           argString(args, "model"),
		Status:          argString(args, "status"),
		Action:          argString(args, "action"),
		Task:            argString(args, "task"),
		TodoID:          argUint(args, "todo_id"),
		ParentSessionID: argString(args, "parent_session_id"),
		ProjectID:       argUint(args, "project_id"),
		ProjectName:     argString(args, "project_name"),
	}
}

func argString(args map[string]interface{}, key string) string {
	if v, ok := args[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func argUint(args map[string]interface{}, key string) uint {
	switch v := args[key].(type) {
	case float64:
		if v > 0 {
			return uint(v)
		}
	case int:
		if v > 0 {
			return uint(v)
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			return uint(n)
		}
	}
	return 0
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	return truncate(strings.Trim(slugRe.ReplaceAllString(strings.ToLower(s), "-"), "-"), 60)
}
