package models

import "time"

// Jenis pekerja yang divisualisasikan di halaman Kantor Agent.
const (
	WorkerKindCodingAgent = "coding_agent" // Agent coding eksternal (Claude Code, Antigravity, Cursor) via MCP / hook
	WorkerKindLocalAgent  = "local_agent"  // CLI agent yang dijalankan vibeasik (agy, claude, aider, custom)
	WorkerKindLLMCall     = "llm_call"     // Satu request ke LLM luar (OpenAI-compatible / System One)
	WorkerKindPlanner     = "planner"      // Agent perencana vibeasik: Project Manager & spesialis rapat (bisnis, desain, …)
)

// Status pekerja.
const (
	WorkerStatusWorking = "working"
	WorkerStatusIdle    = "idle"
	WorkerStatusDone    = "done"
	WorkerStatusError   = "error"
	WorkerStatusLeft    = "left" // Sesi agent berakhir; karakter keluar kantor
)

// Peran agent coding dalam mode divisi (kolom Role): koordinator mengambil todo dan menyuruh sub agent.
const (
	WorkerRoleCoordinator = "Pengambil todo"
	WorkerRoleSubagent    = "Anak buah"
)

// AgentWorker adalah satu "orang" di Kantor Agent: siapa, sedang apa, dan task apa.
// Disimpan di DB (bukan hanya memori) karena proses MCP stdio (`vibeasik mcp`) berjalan
// terpisah dari engine, tetapi memakai database yang sama.
type AgentWorker struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Key        string `gorm:"column:worker_key;size:191;uniqueIndex" json:"key"`
	Kind       string `gorm:"size:32;index" json:"kind"`
	Name       string `gorm:"size:128" json:"name"`        // Nama tampilan, mis. "Claude Code"
	ClientName string `gorm:"size:128" json:"client_name"` // Nama mentah dari clientInfo MCP / adapter key
	Model      string `gorm:"size:128" json:"model,omitempty"`
	Role       string `gorm:"size:191" json:"role,omitempty"` // Peran LLM dari system prompt, mis. "Principal Software Architect"
	// ParentKey = key agent koordinator (pengambil todo) bila pekerja ini sub agent suruhannya;
	// Kantor Agent menampilkan keduanya sebagai satu divisi.
	ParentKey   string     `gorm:"column:parent_key;size:191;index" json:"parent_key,omitempty"`
	Transport   string     `gorm:"size:32" json:"transport,omitempty"`
	Status      string     `gorm:"size:32;index" json:"status"`
	Action      string     `gorm:"size:191" json:"action,omitempty"` // Aksi terakhir (nama tool MCP / langkah)
	Task        string     `gorm:"size:512" json:"task,omitempty"`   // Pekerjaan yang sedang dikerjakan
	TodoID      *uint      `json:"todo_id,omitempty"`
	ProjectID   *uint      `json:"project_id,omitempty"`
	ProjectName string     `gorm:"size:255" json:"project_name,omitempty"`
	ProjectPath string     `gorm:"size:1024" json:"project_path,omitempty"` // Direktori kerja agent (untuk rekap commit)
	ToolCalls   int        `json:"tool_calls"`
	TasksDone   int        `json:"tasks_done"`
	LastError   string     `gorm:"size:512" json:"last_error,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	LastSeenAt  time.Time  `gorm:"index" json:"last_seen_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}
