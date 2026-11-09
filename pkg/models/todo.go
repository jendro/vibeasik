package models

import (
	"time"
)

// Todo level constants (Hierarki: Global/Platform, Project, Component)
const (
	TodoLevelGlobal    = "global"    // Global: Kebijakan makro platform, standarisasi arsitektur, multi-project
	TodoLevelProject   = "project"   // Project: Bundle proyek, integrasi sistem, release milestone
	TodoLevelComponent = "component" // Component: Aplikasi/servis spesifik (BE, FE, Mobile, Worker)
)

// Todo status constants
const (
	TodoStatusPending    = "pending"
	TodoStatusInProgress = "in_progress"
	TodoStatusCompleted  = "completed"
	TodoStatusSuggested  = "suggested" // Saran AI menunggu konfirmasi/persetujuan manusia
	TodoStatusRejected   = "rejected"  // Saran AI ditolak oleh manusia
)

// Todo priority constants
const (
	TodoPriorityLow      = "low"
	TodoPriorityMedium   = "medium"
	TodoPriorityHigh     = "high"
	TodoPriorityCritical = "critical"
)

// Todo creator type constants
const (
	CreatorHuman = "human"
	CreatorAI    = "ai_agent"
)

// Todo category constants
const (
	CategoryFeature      = "feature"
	CategorySecurity     = "security"
	CategoryUIUX         = "ui_ux"
	CategoryArchitecture = "architecture"
	CategoryPerformance  = "performance"
	CategoryBackend      = "backend"
	CategoryDevOps       = "devops"
	CategoryGeneral      = "general"
)

// Todo represents an actionable task item linked in a 3-tier hierarchy
type Todo struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Level            string     `gorm:"size:64;default:'project';index" json:"level"`    // global, project, component
	Category         string     `gorm:"size:64;default:'general';index" json:"category"` // feature, security, ui_ux, architecture, performance, backend, devops, general
	Categories       string     `gorm:"size:256" json:"categories,omitempty"`           // CSV semua kategori (todo gabungan lintas kategori); Category = kategori utama
	ParentID         *uint      `gorm:"index" json:"parent_id,omitempty"`                // Cascading parent todo
	SplitFromID      *uint      `gorm:"index" json:"split_from_id,omitempty"`            // Induk yang dipecah per aplikasi (todo_split.go); sub todo ini satu aplikasi saja
	ProjectID        *uint      `gorm:"index" json:"project_id,omitempty"`
	ProjectName      string     `gorm:"size:255" json:"project_name,omitempty"`
	ComponentID      *uint      `gorm:"index" json:"component_id,omitempty"`
	ComponentName    string     `gorm:"size:255" json:"component_name,omitempty"`
	TargetApps       string     `gorm:"size:512" json:"target_apps,omitempty"` // e.g. "Backend API (BE) [1 App]" or "Seluruh Aplikasi (2 Apps)"
	Title            string     `gorm:"size:512;not null" json:"title"`
	Description      string     `gorm:"type:text" json:"description"`
	Status           string     `gorm:"size:64;default:'pending';index;index:idx_todos_status_created,priority:1" json:"status"` // pending, in_progress, completed, suggested, rejected
	Priority         string     `gorm:"size:64;default:'medium'" json:"priority"`      // low, medium, high, critical
	CreatorType      string     `gorm:"size:64;default:'human'" json:"creator_type"`   // human, ai_agent
	CreatorName      string     `gorm:"size:255;default:'developer'" json:"creator_name"`
	IsAISuggestion   bool       `gorm:"default:false;index" json:"is_ai_suggestion"`
	GuidelineRef     string     `gorm:"size:128" json:"guideline_ref,omitempty"`   // e.g. PLAN-001, ARCH-001, SEC-001
	Tags             string     `gorm:"size:512;index" json:"tags,omitempty"`      // CSV tag fokus (todo_tags.go): fitur, keamanan-dasar, optimasi-query, rawat-ai, belajar
	Rationale        string     `gorm:"type:text" json:"rationale,omitempty"`      // Penjelasan/analisis mengapa saran ini dibuat
	Source           string     `gorm:"size:64;default:'manual'" json:"source"`    // manual, ai_agent, scanner, code_comment
	AuditKey         string     `gorm:"size:600;index" json:"audit_key,omitempty"` // todo audit: "<rule>|<berkas>" — temuan dengan kunci sama digabung ke todo ini
	DueDate          string     `gorm:"size:64" json:"due_date,omitempty"`
	CompletedAt      *time.Time `gorm:"index" json:"completed_at,omitempty"`
	CompletedByAgent string     `gorm:"size:128" json:"completed_by_agent,omitempty"` // Coding agent that completed the task (e.g. Antigravity, Claude Code, Aider, Cursor)
	CompletedByModel string     `gorm:"size:128" json:"completed_by_model,omitempty"` // AI model used by the agent (e.g. gemini-3.8-flash, claude-3.7-sonnet)
	EmployeeID       *uint      `gorm:"index" json:"employee_id,omitempty"`           // Pegawai vibeasik yang ditugaskan (persona agent coding yang mengerjakannya)
	EmployeeName     string     `gorm:"size:128" json:"employee_name,omitempty"`
	EmployeeLeaderID *uint      `json:"employee_leader_id,omitempty"`                                 // Leader divisi yang menimbang & menugaskan pegawai
	EmployeeReason   string     `gorm:"size:600" json:"employee_reason,omitempty"`                    // Alasan leader memilih pegawai ini
	ReworkCount      int        `gorm:"default:0" json:"rework_count,omitempty"`                      // Berapa kali todo dibuka lagi karena nilai QA rendah
	Memory           string     `gorm:"type:text" json:"memory,omitempty"`                            // Ringkasan riwayat pengerjaan (todo_memory.go), dipakai prompt & pencarian todo serupa
	ReworkNote       string     `gorm:"type:text" json:"rework_note,omitempty"`                       // Arahan leader untuk pengerjaan ulang terakhir (ikut di brief pegawai)
	Score            *int       `json:"score,omitempty"`                                              // Nilai review QA (0–100)
	HighScoreSkills  []string   `gorm:"type:text;serializer:json" json:"high_score_skills,omitempty"` // Skill penyebab nilai tinggi
	HighScoreNotes   string     `gorm:"type:text" json:"high_score_notes,omitempty"`                  // Catatan detail kenapa task ini sukses/bernilai tinggi
	SuccessFactors   []string   `gorm:"type:text;serializer:json" json:"success_factors,omitempty"`   // Butir-butir faktor penentu keberhasilan
	InheritedFrom    string     `gorm:"size:128" json:"inherited_from,omitempty"`                     // Nama eks pegawai yang ilmunya/standarnya diwarisi
	TodoComplexity              // Kerumitan multi-dimensi (logika, koding, cakupan, risiko) untuk orkestrator agent
	TodoImpact                  // Dampak pada aplikasi (7 dimensi 0–5 → skor final 0–100 berbobot kategori)
	CreatedAt        time.Time  `gorm:"index:idx_todos_status_created,priority:2" json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// LLMTodoRecommendation represents structured AI recommendations for actionable planning & todos
type LLMTodoRecommendation struct {
	Category       string         `json:"category"`
	Level          string         `json:"level"`
	TargetApps     string         `json:"target_apps"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Priority       string         `json:"priority"`
	GuidelineRef   string         `json:"guideline_ref"`
	Rationale      string         `json:"rationale"`
	ActionDecision string         `json:"action_decision,omitempty"` // "do" (lakukan), "defer" (tunda), "skip" (tidak perlu)
	DecisionReason string         `json:"decision_reason,omitempty"` // Alasan pertimbangan dewan
	Complexity     *LLMComplexity `json:"complexity,omitempty"`
	Impact         *LLMImpact     `json:"impact,omitempty"`
}

// LLMComplexity adalah penilaian kerumitan 1–5 yang dikembalikan LLM untuk sebuah todo.
type LLMComplexity struct {
	Logic  int    `json:"logic"`
	Coding int    `json:"coding"`
	Scope  int    `json:"scope"`
	Risk   int    `json:"risk"`
	Note   string `json:"note"`
}
