package activity

import (
	"net/url"
	"path/filepath"
	"strings"
	"unicode"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

var llmProviders = []struct{ needle, name string }{
	{"api.openai.com", "OpenAI"},
	{"anthropic", "Anthropic"},
	{"openrouter", "OpenRouter"},
	{"generativelanguage", "Gemini"},
	{"googleapis", "Google AI"},
	{"groq", "Groq"},
	{"deepseek", "DeepSeek"},
	{"mistral", "Mistral"},
	{"together", "Together AI"},
	{"x.ai", "xAI"},
	{":11434", "Ollama"},
	{":1234", "LM Studio"},
	{":8750", "System One"},
	{"systemone", "System One"},
}

// LLMProviderName menebak nama penyedia LLM dari endpoint.
func LLMProviderName(endpoint string) string {
	l := strings.ToLower(endpoint)
	for _, p := range llmProviders {
		if strings.Contains(l, p.needle) {
			return p.name
		}
	}
	if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
		return u.Hostname()
	}
	return "LLM"
}

var rolePrefixes = []string{"anda adalah ", "kamu adalah ", "you are an ", "you are a ", "you are "}

// LLMRole mengambil peran dari kalimat pembuka system prompt,
// mis. "Anda adalah Principal Software Architect, ..." -> "Principal Software Architect".
func LLMRole(systemPrompt string) string {
	line := firstLine(systemPrompt)
	lower := strings.ToLower(line)
	for _, p := range rolePrefixes {
		if strings.HasPrefix(lower, p) {
			line = line[len(p):]
			break
		}
	}
	for _, sep := range []string{",", ".", " pada ", " untuk ", " for ", " who ", " yang "} {
		if i := strings.Index(line, sep); i > 0 {
			line = line[:i]
		}
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	r := []rune(line)
	r[0] = unicode.ToUpper(r[0])
	return truncate(string(r), 60)
}

// BeginLLM mencatat satu request LLM keluar sebagai "kurir" yang datang ke kantor.
// Kembaliannya wajib dipanggil saat request selesai (aman dipanggil walau Init belum dijalankan).
func BeginLLM(endpoint, model, systemPrompt string) func(err error) {
	d := getDB()
	if d == nil {
		return func(error) {}
	}
	mu.RLock()
	ctxFn := llmContext
	mu.RUnlock()

	task, project := "", ""
	if ctxFn != nil {
		task, project = ctxFn()
	}
	if task == "" {
		task = "Menjawab permintaan analisis"
	}
	// Kantor Agent hanya menampilkan nama model (alias bila diatur), bukan alamat endpoint tujuan request.
	label := truncate(models.ResolveModelAlias(d, endpoint, model), maxGenericLength)
	if label == "" {
		label = "LLM"
	}
	key := newKey("llm")
	upsertWorker(d, key, models.AgentWorker{
		Kind:        models.WorkerKindLLMCall,
		Name:        label,
		Model:       label,
		Role:        LLMRole(systemPrompt),
		Transport:   "http",
		Status:      models.WorkerStatusWorking,
		Action:      "Menunggu jawaban model",
		Task:        task,
		ProjectName: project,
	}, nil)
	return func(err error) {
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		finish(d, key, err != nil, msg)
	}
}

// BeginLocalAgent mencatat CLI agent (agy/claude/aider/custom) yang dijalankan vibeasik.
func BeginLocalAgent(agentKey, prompt, projectPath string) func(success bool, errMsg string) {
	return BeginLocalAgentTodo(agentKey, prompt, projectPath, nil)
}

// BeginLocalAgentTodo = BeginLocalAgent yang menautkan todo yang dikerjakan (run orkestrator), sehingga
// Kantor Agent bisa menampilkan pekerja ini sebagai pegawai pemilik todo.
func BeginLocalAgentTodo(agentKey, prompt, projectPath string, todoID *uint) func(success bool, errMsg string) {
	d := getDB()
	if d == nil {
		return func(bool, string) {}
	}
	project, path := "", ""
	if projectPath != "" && projectPath != "." {
		project, path = filepath.Base(projectPath), truncate(projectPath, maxPathLength)
	}
	key := newKey("local")
	upsertWorker(d, key, models.AgentWorker{
		Kind:        models.WorkerKindLocalAgent,
		Name:        FriendlyAgentName(agentKey),
		ClientName:  agentKey,
		Transport:   "cli",
		Status:      models.WorkerStatusWorking,
		Action:      "Menjalankan CLI " + agentKey,
		Task:        firstLine(prompt),
		TodoID:      todoID,
		ProjectName: project,
		ProjectPath: path,
	}, nil)
	return func(success bool, errMsg string) {
		finish(d, key, !success, errMsg)
	}
}

// PlannerWorker adalah satu peserta rapat perencanaan (Project Manager atau spesialis) di Kantor Agent.
// Nil-safe: bila pencatatan belum aktif, semua method menjadi no-op.
type PlannerWorker struct {
	d   *gorm.DB
	key string
}

// BeginPlanner mencatat agent perencana vibeasik yang mulai bekerja. roleKey dipakai studio untuk
// memilih penampilan karakter (pm, business, design, architect, security, qa).
func BeginPlanner(roleKey, name, model, task string, projectID uint, projectName string) *PlannerWorker {
	d := getDB()
	if d == nil {
		return nil
	}
	key := newKey("planner")
	var pid *uint
	if projectID > 0 {
		pid = &projectID
	}
	upsertWorker(d, key, models.AgentWorker{
		Kind:        models.WorkerKindPlanner,
		Name:        name,
		ClientName:  roleKey,
		Model:       truncate(model, maxGenericLength),
		Role:        name,
		Transport:   "engine",
		Status:      models.WorkerStatusWorking,
		Action:      "Membaca konteks",
		Task:        task,
		ProjectID:   pid,
		ProjectName: projectName,
	}, nil)
	return &PlannerWorker{d: d, key: key}
}

// Step memperbarui aksi yang sedang dikerjakan peserta rapat.
func (p *PlannerWorker) Step(action string) {
	if p == nil {
		return
	}
	upsertWorker(p.d, p.key, models.AgentWorker{}, func(w *models.AgentWorker) { w.Action = action })
}

// SetModel mencatat model LLM yang akhirnya menjawab (bisa berbeda dari rencana bila ada fallback).
func (p *PlannerWorker) SetModel(model string) {
	if p == nil || model == "" {
		return
	}
	upsertWorker(p.d, p.key, models.AgentWorker{}, func(w *models.AgentWorker) { w.Model = truncate(model, maxGenericLength) })
}

// Done menutup peserta rapat; err != nil membuatnya tampil bermasalah di kantor.
func (p *PlannerWorker) Done(err error) {
	if p == nil {
		return
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	} else {
		p.d.Model(&models.AgentWorker{}).Where("worker_key = ?", p.key).UpdateColumn("tasks_done", gorm.Expr("tasks_done + 1"))
	}
	finish(p.d, p.key, err != nil, msg)
}
