package models

import "time"

// Status sesi rapat Project Manager.
const (
	PlanningQueued    = "queued"
	PlanningRunning   = "running"
	PlanningCompleted = "completed"
	PlanningFailed    = "failed"
)

// Pemicu rapat Project Manager.
const (
	PlanningTriggerManual      = "manual"        // tombol "Minta PM rapat" di studio
	PlanningTriggerRunComplete = "run_completed" // todo selesai dikerjakan agent orkestrator
	PlanningTriggerAgent       = "agent_request" // agent coding meminta saran lewat MCP (suggest_todos / vibeasik_pm_request_plan)
	PlanningTriggerSuggest     = "todo_suggest"  // tombol Saran AI di halaman Todo
)

// PlanningRequest adalah satu permintaan yang menunggu dibahas PM. Permintaan yang datang selama
// sesi masih antre digabung ke sesi yang sama sehingga PM membahas semuanya dalam satu rapat.
type PlanningRequest struct {
	Source    string    `json:"source"`              // salah satu PlanningTrigger*
	Requester string    `json:"requester,omitempty"` // mis. "Claude Code", "Antigravity · Akun 2", "developer"
	Note      string    `json:"note,omitempty"`      // pertanyaan / konteks dari peminta
	TodoID    uint      `json:"todo_id,omitempty"`   // todo yang baru selesai (pemicu run_completed)
	RunID     uint      `json:"run_id,omitempty"`
	At        time.Time `json:"at"`
}

// PlanningProposal adalah usulan todo dari seorang spesialis di rapat.
type PlanningProposal struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Rationale   string `json:"rationale,omitempty"`
}

// PlanningOpinion adalah pendapat satu peserta rapat (spesialis) beserta model LLM yang dipakainya.
type PlanningOpinion struct {
	Role        string             `json:"role"`
	RoleName    string             `json:"role_name"`
	Question    string             `json:"question,omitempty"` // pertanyaan PM untuk spesialis ini
	Model       string             `json:"model,omitempty"`
	Opinion     string             `json:"opinion,omitempty"`
	Concerns    []string           `json:"concerns,omitempty"`
	Proposals   []PlanningProposal `json:"proposals,omitempty"`
	NewFeatures *bool              `json:"new_features,omitempty"` // suara spesialis: perlu fitur baru?
	Error       string             `json:"error,omitempty"`
	DurationMs  int64              `json:"duration_ms"`
}

// PlanningContextLookup adalah data tambahan yang diminta PM dari sistem vibeasik (peta kode, todo).
type PlanningContextLookup struct {
	Tool   string `json:"tool"`
	Query  string `json:"query"`
	Result string `json:"result"`
}

// PlanningDecisionTodo adalah satu todo yang diputuskan PM, beserta nasibnya di database.
type PlanningDecisionTodo struct {
	Title       string   `json:"title"`
	Category    string   `json:"category"`
	Priority    string   `json:"priority"`
	ProposedBy  []string `json:"proposed_by,omitempty"`
	AssignHint  string   `json:"assign_hint,omitempty"` // saran PM: jenis agent/effort yang cocok
	TodoID      uint     `json:"todo_id,omitempty"`
	RedoOf      uint     `json:"redo_of,omitempty"` // todo selesai yang dikerjakan ulang karena berkasnya berubah lagi
	Outcome     string   `json:"outcome"`           // created, duplicate, assigned
	RunID       uint     `json:"run_id,omitempty"`
	AssignError string   `json:"assign_error,omitempty"`
}

// Jenis penolakan PM. Saran lama yang ditolak sebagai duplikat, sudah terpenuhi, atau tidak valid
// ditandai rejected otomatis; yang ditunda/bernilai rendah dibiarkan menunggu keputusan manusia.
const (
	RejectDuplicate = "duplicate"
	RejectFulfilled = "fulfilled"
	RejectInvalid   = "invalid" // false positive / tidak relevan
	RejectDeferred  = "deferred"
	RejectLowValue  = "low_value"
)

// PlanningRejected adalah usulan yang ditolak PM beserta alasannya.
type PlanningRejected struct {
	Title  string `json:"title"`
	Reason string `json:"reason"`
	Kind   string `json:"kind,omitempty"`
	TodoID uint   `json:"todo_id,omitempty"` // saran lama yang dimaksud penolakan ini
	// Applied: saran lama itu sudah ditandai rejected oleh PM; Note menjelaskan bila tidak diterapkan.
	Applied bool   `json:"applied"`
	Note    string `json:"note,omitempty"`
}

// PlanningSession adalah satu rapat Project Manager: PM menganalisa konteks, mengundang spesialis
// (bisnis, desain, arsitektur, keamanan, QA), lalu memutuskan todo apa yang dikerjakan dan apakah
// perlu fitur baru. Seluruh jejak rapat disimpan agar keputusan bisa ditelusuri.
type PlanningSession struct {
	ID              uint                    `gorm:"primaryKey" json:"id"`
	ProjectID       uint                    `gorm:"index" json:"project_id"`
	ProjectName     string                  `gorm:"size:255" json:"project_name"`
	Status          string                  `gorm:"size:32;index" json:"status"`
	Trigger         string                  `gorm:"size:32" json:"trigger"`
	Requests        []PlanningRequest       `gorm:"type:text;serializer:json" json:"requests"`
	Step            string                  `gorm:"size:255" json:"step,omitempty"` // langkah rapat yang sedang berjalan
	PMModel         string                  `gorm:"size:128" json:"pm_model,omitempty"`
	Situation       string                  `gorm:"type:text" json:"situation,omitempty"` // analisa awal PM
	AddFeatures     bool                    `json:"add_features"`
	FeatureDecision string                  `gorm:"type:text" json:"feature_decision,omitempty"` // alasan perlu/tidaknya fitur baru
	Summary         string                  `gorm:"type:text" json:"summary,omitempty"`
	Lookups         []PlanningContextLookup `gorm:"type:text;serializer:json" json:"lookups"`
	Opinions        []PlanningOpinion       `gorm:"type:text;serializer:json" json:"opinions"`
	Decisions       []PlanningDecisionTodo  `gorm:"type:text;serializer:json" json:"decisions"`
	Rejected        []PlanningRejected      `gorm:"type:text;serializer:json" json:"rejected"`
	TodosCreated    int                     `json:"todos_created"`
	TodosAssigned   int                     `json:"todos_assigned"`
	TodosRejected   int                     `json:"todos_rejected"` // saran lama yang ditandai rejected oleh PM
	Error           string                  `gorm:"type:text" json:"error,omitempty"`
	StartedAt       *time.Time              `json:"started_at,omitempty"`
	FinishedAt      *time.Time              `json:"finished_at,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

// PlannerSettings mengatur agent Project Manager (satu baris).
type PlannerSettings struct {
	ID      uint `gorm:"primaryKey" json:"id"`
	Enabled bool `gorm:"default:true" json:"enabled"`
	// PMModel: "auto" = model terpintar yang terdaftar di Pengaturan LLM, atau "<config_id>:<model>" pilihan manual
	PMModel string `gorm:"size:255;default:'auto'" json:"pm_model"`
	// MinIntelligence: skor kecerdasan minimal (0–100) model yang boleh dipakai rapat perencanaan
	MinIntelligence int `gorm:"default:85" json:"min_intelligence"`
	// Specialists: peran spesialis yang boleh diundang PM (CSV, kosong = semua)
	Specialists string `gorm:"size:255" json:"specialists"`
	// MaxSpecialists: jumlah spesialis maksimal per rapat
	MaxSpecialists int `gorm:"default:3" json:"max_specialists"`
	// MaxTodos: jumlah todo baru maksimal per rapat
	MaxTodos int `gorm:"default:5" json:"max_todos"`
	// ReviewAfterRun: PM meninjau hasil setiap todo yang selesai dikerjakan agent orkestrator
	ReviewAfterRun bool `gorm:"default:true" json:"review_after_run"`
	// AutoAssign: todo keputusan PM langsung berstatus pending dan dikirim ke agent coding lewat orkestrator
	AutoAssign bool      `gorm:"default:false" json:"auto_assign"`
	UpdatedAt  time.Time `json:"updated_at"`
}
