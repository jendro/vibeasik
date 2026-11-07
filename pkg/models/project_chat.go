package models

import "time"

// ProjectChatMessage represents a persistent conversation exchange between developer and AI Architect
type ProjectChatMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"index;not null" json:"project_id"`
	Role      string    `gorm:"size:32;not null" json:"role"` // "user", "assistant", "system"
	Content   string    `gorm:"type:text;not null" json:"content"`
	TodosJSON string    `gorm:"type:text" json:"todos_json,omitempty"` // JSON serialized array of LLMTodoRecommendation
	ModelName string    `gorm:"size:128" json:"model_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ProjectChatRequest is the incoming payload from the developer
type ProjectChatRequest struct {
	Message string `json:"message"`
	Model   string `json:"model,omitempty"` // Optional model override
}

// ProjectChatResponse is returned by the AI planner to the client
type ProjectChatResponse struct {
	Reply     string                  `json:"reply"`    // Conversational answer & architectural review
	Analysis  string                  `json:"analysis"` // Structured architectural impact analysis
	Todos     []LLMTodoRecommendation `json:"todos"`    // Concrete, structured actionable TODO items
	Model     string                  `json:"model"`
	MessageID uint                    `json:"message_id"`
}
