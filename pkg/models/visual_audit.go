package models

import "time"

// VisualCaptureRequest represents parameters to capture a screenshot of a running project component
type VisualCaptureRequest struct {
	ProjectID     uint   `json:"project_id"`
	ComponentID   uint   `json:"component_id"`
	ComponentName string `json:"component_name,omitempty"`
	Port          int    `json:"port"`
	Route         string `json:"route"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	CustomURL     string `json:"custom_url,omitempty"`
}

// VisualCaptureResult represents the result of a page screenshot capture
type VisualCaptureResult struct {
	ProjectID     uint      `json:"project_id"`
	ComponentID   uint      `json:"component_id"`
	ComponentName string    `json:"component_name"`
	TargetURL     string    `json:"target_url"`
	Port          int       `json:"port"`
	Route         string    `json:"route"`
	CapturedAt    time.Time `json:"captured_at"`
	ImagePath     string    `json:"image_path"`
	Base64Image   string    `json:"base64_image,omitempty"`
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	SizeKB        float64   `json:"size_kb"`
}

// VisualIssue represents an issue detected in visual or data audit
type VisualIssue struct {
	Category    string `json:"category"` // "data", "visual", "layout", "accessibility"
	Severity    string `json:"severity"` // "error", "warning", "info"
	Title       string `json:"title"`
	Description string `json:"description"`
	TargetArea  string `json:"target_area,omitempty"` // e.g., "Header", "User Table", "Footer"
}

// VisualAuditRequest represents the request to analyze a captured screenshot with LLM
type VisualAuditRequest struct {
	ProjectID   uint   `json:"project_id"`
	ComponentID uint   `json:"component_id"`
	Port        int    `json:"port"`
	Route       string `json:"route"`
	CustomURL   string `json:"custom_url,omitempty"`
	Base64Image string `json:"base64_image,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
}

// VisualAuditResponse represents the multimodal AI review of the component's UI and data
type VisualAuditResponse struct {
	ProjectID        uint          `json:"project_id"`
	ComponentID      uint          `json:"component_id"`
	ComponentName    string        `json:"component_name"`
	Route            string        `json:"route"`
	TargetURL        string        `json:"target_url"`
	VisualScore      int           `json:"visual_score"`       // 0 - 100
	DataHygieneScore int           `json:"data_hygiene_score"` // 0 - 100
	Summary          string        `json:"summary"`
	DataIssues       []VisualIssue `json:"data_issues"`
	VisualIssues     []VisualIssue `json:"visual_issues"`
	SuggestedFixes   []string      `json:"suggested_fixes"`
	AIAgentPrompt    string        `json:"ai_agent_prompt"`
	ScreenshotPath   string        `json:"screenshot_path"`
	AuditedAt        time.Time     `json:"audited_at"`
}
