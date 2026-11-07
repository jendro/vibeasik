package models

import "time"

// Peran eksekutif per proyek.
const (
	ExecCEO          = "ceo"
	ExecCTO          = "cto"
	ExecProductOwner = "product_owner"
	ExecCMO          = "cmo" // riset & intelijen pasar: kondisi pasar, kompetitor, ramalan pasar
)

const (
	ExecActive   = "active"
	ExecInactive = "inactive"
)

// ProjectExecutive: satu eksekutif terikat ke satu proyek (setiap proyek punya 1 CEO, 1 CTO, 1 PO, 1 CMO).
type ProjectExecutive struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ProjectID   uint           `gorm:"uniqueIndex:idx_exec_proj_role;not null" json:"project_id"`
	ProjectName string         `gorm:"size:255" json:"project_name"`
	Role        string         `gorm:"uniqueIndex:idx_exec_proj_role;size:32;not null" json:"role"` // ceo, cto, product_owner, cmo
	EmployeeID  uint           `gorm:"index" json:"employee_id"`                                    // ID pegawai di models.Employee
	Name        string         `gorm:"size:128" json:"name"`
	Title       string         `gorm:"size:128" json:"title"`  // "Chief Executive Officer", dll.
	Brief       string         `gorm:"type:text" json:"brief"` // tanggung jawab khusus proyek ini
	Personality string         `gorm:"type:text" json:"personality"`
	Motto       string         `gorm:"size:255" json:"motto"`
	Avatar      EmployeeAvatar `gorm:"type:text;serializer:json" json:"avatar"`
	Skills      map[string]int `gorm:"type:text;serializer:json" json:"skills"`
	Status      string         `gorm:"size:32;default:'active'" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// Jenis pengetahuan eksekutif.
const (
	ExecKnowledgeVision       = "product_vision"     // visi & tujuan produk
	ExecKnowledgeTechDecision = "technical_decision" // keputusan arsitektur/teknis
	ExecKnowledgeMarket       = "market"             // pasar, pengguna, kompetitor
	ExecKnowledgeUserFeedback = "user_feedback"      // masukan pengguna
	ExecKnowledgeLesson       = "lesson"             // pelajaran dari pengembangan
	ExecKnowledgeReality      = "app_reality"        // fakta aplikasi nyata: fitur yang sudah dibangun, kondisi kode
	ExecKnowledgeRoadmap      = "roadmap"            // perencanaan ke depan: milestone, urutan prioritas, target rilis
	ExecKnowledgeForecast     = "forecast"           // ramalan pasar: arah tren, peluang & ancaman ke depan
)

// ExecKnowledgeKinds: semua jenis pengetahuan eksekutif yang sah (urutan = urutan tampilan di prompt).
var ExecKnowledgeKinds = []string{
	ExecKnowledgeReality, ExecKnowledgeVision, ExecKnowledgeRoadmap, ExecKnowledgeTechDecision,
	ExecKnowledgeMarket, ExecKnowledgeForecast, ExecKnowledgeUserFeedback, ExecKnowledgeLesson,
}

// ValidExecKnowledgeKind memeriksa apakah kind termasuk jenis pengetahuan eksekutif yang sah.
func ValidExecKnowledgeKind(kind string) bool {
	for _, k := range ExecKnowledgeKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// Sumber pengetahuan.
const (
	ExecSourceSeed       = "seed"            // bawaan saat dibuat
	ExecSourceTodo       = "todo_completed"  // dari todo yang selesai
	ExecSourceMeeting    = "meeting"         // dari rapat PM/eksekutif
	ExecSourceProfile    = "profile"         // dari profil proyek
	ExecSourceManual     = "manual"          // dimasukkan manual
	ExecSourceChat       = "chat"            // dari obrolan interaktif dengan developer
	ExecSourceAutonomous = "autonomous"      // dari musyawarah otonom dewan
	ExecSourceMarket     = "market_research" // dari riset pasar web (sinyal nyata HN/GitHub/Reddit)
	ExecSourceQA         = "qa_review"       // fakta proyek yang dipetik QA dari hasil kerja pegawai
)

const (
	ExecKnowledgeActive   = "active"
	ExecKnowledgeArchived = "archived"
)

// ExecutiveKnowledge: pengetahuan produk yang dikumpulkan eksekutif dari todo, review, rapat.
type ExecutiveKnowledge struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ExecutiveID     uint      `gorm:"index;not null" json:"executive_id"`
	ProjectID       uint      `gorm:"index;not null" json:"project_id"`
	Topic           string    `gorm:"size:191" json:"topic"`
	Content         string    `gorm:"type:text" json:"content"`
	Kind            string    `gorm:"size:32" json:"kind"`   // lihat ExecKnowledgeKinds
	Source          string    `gorm:"size:32" json:"source"` // seed, todo_completed, meeting, profile, manual
	Weight          int       `json:"weight"`
	SourceTodoID    *uint     `json:"source_todo_id,omitempty"`
	SourceSessionID *uint     `json:"source_session_id,omitempty"`
	Status          string    `gorm:"size:32;index;default:'active'" json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
