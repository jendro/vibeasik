package models

import "time"

// LocalAgentConfig represents a user-defined plug-and-play CLI agent
type LocalAgentConfig struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Key          string    `gorm:"uniqueIndex;size:80;not null" json:"key"` // e.g. "custom-script"
	Name         string    `gorm:"size:120;not null" json:"name"`
	Command      string    `gorm:"size:255;not null" json:"command"` // binary name or executable path
	ArgsTemplate string    `gorm:"type:text" json:"args_template"`   // e.g. "-p {{prompt}}"
	Description  string    `gorm:"type:text" json:"description"`
	Icon         string    `gorm:"size:60" json:"icon"` // icon identifier, e.g. "i-lucide-bot"
	Enabled      bool      `gorm:"default:true" json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AgentStatusDTO describes the live status of an agent (built-in or custom)
type AgentStatusDTO struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	Command      string   `json:"command"`
	Detected     bool     `json:"detected"`
	Path         string   `json:"path"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
	IsBuiltin    bool     `json:"is_builtin"`
}

// AgentExecutionRequest specifies a task prompt to run against a target project
type AgentExecutionRequest struct {
	AgentKey    string `json:"agent_key"`
	Prompt      string `json:"prompt"`
	ProjectPath string `json:"project_path"` // absolute or relative directory to run the agent in
	TimeoutSec  int    `json:"timeout_sec"`  // optional timeout, default 60s
}

// AgentExecutionResult contains the execution output and timing
type AgentExecutionResult struct {
	AgentKey string `json:"agent_key"`
	Success  bool   `json:"success"`
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
	Duration string `json:"duration"`
	Message  string `json:"message"`
}
