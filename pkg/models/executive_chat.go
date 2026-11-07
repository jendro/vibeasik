package models

import "time"

// ExecutiveChatMessage menyimpan percakapan Ruang Klien: pengguna (klien) dengan dewan eksekutif proyek
// (CEO, CTO, PO, CMO) yang menjadi penanggung jawab proyek.
type ExecutiveChatMessage struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ProjectID    uint      `gorm:"index;not null" json:"project_id"`
	Role         string    `gorm:"size:32;not null" json:"role"`   // "user", "assistant"
	Target       string    `gorm:"size:32;not null" json:"target"` // "all", "ceo", "cto", "product_owner"
	Kind         string    `gorm:"size:16" json:"kind,omitempty"`  // jenis permintaan klien: bug, question, feature, change, other
	SpeakerRole  string    `gorm:"size:32" json:"speaker_role"`    // "user", "all", "ceo", "cto", "product_owner"
	SpeakerName  string    `gorm:"size:128" json:"speaker_name"`
	SpeakerTitle string    `gorm:"size:128" json:"speaker_title"`
	AvatarJSON   string    `gorm:"type:text" json:"avatar_json,omitempty"`
	Content      string    `gorm:"type:text;not null" json:"content"`
	TodosJSON    string    `gorm:"type:text" json:"todos_json,omitempty"`    // Serialized []LLMTodoRecommendation
	SavedTodoIDs string    `gorm:"size:512" json:"saved_todo_ids,omitempty"` // CSV id todo yang dibuat dari balasan ini
	ModelName    string    `gorm:"size:128" json:"model_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ExecutiveChatRequest payload dari frontend saat mengirim pesan ke eksekutif
type ExecutiveChatRequest struct {
	Message       string `json:"message"`
	Target        string `json:"target,omitempty"`          // "all", "ceo", "cto", "product_owner"
	Kind          string `json:"kind,omitempty"`            // jenis permintaan klien: bug, question, feature, change, other
	Model         string `json:"model,omitempty"`           // Model pilihan pengguna (misal gpt-4o, claude-3-5-sonnet, deepseek-chat)
	AutoSaveTodos bool   `json:"auto_save_todos,omitempty"` // Bila true, todo rekomendasi langsung disimpan ke DB
}

// ExecutiveChatResponse balasan dari eksekutif
type ExecutiveChatResponse struct {
	Reply        string                  `json:"reply"`
	Analysis     string                  `json:"analysis"`
	SpeakerRole  string                  `json:"speaker_role"` // "all", "ceo", "cto", "product_owner"
	SpeakerName  string                  `json:"speaker_name"`
	SpeakerTitle string                  `json:"speaker_title"`
	Avatar       *EmployeeAvatar         `json:"avatar,omitempty"`
	Todos        []LLMTodoRecommendation `json:"todos,omitempty"`
	BoardNotes   []ExecutiveKnowledge    `json:"board_notes,omitempty"`
	SavedTodos   []Todo                  `json:"saved_todos,omitempty"`
	Model        string                  `json:"model"`
	MessageID    uint                    `json:"message_id"`
}
