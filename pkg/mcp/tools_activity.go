package mcp

import (
	"fmt"

	"github.com/vibeasik/vibeasik/pkg/activity"
)

// Activity tool: agent mengumumkan pekerjaannya agar tampil sebagai karakter di Kantor Agent (/agent-office).
// Pencatatan sebenarnya terjadi di HandleRPCFrom (activity.ToolCalled) karena di sana identitas koneksi diketahui.

func (s *Server) activityToolsList() []Tool {
	return []Tool{
		{
			Name:        activity.ReportToolName,
			Description: "Announces what you (the AI coding agent) are working on right now so the user can watch it live in the vibeasik Agent Office. Call it when you start a task (status 'working' + task, optionally todo_id), when you finish it (status 'done'), when you wait for the user (status 'idle'), and when the session ends (status 'left'). Calling update_todo with status in_progress/completed is tracked automatically too.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_name":        prop("string", "Your agent name, e.g. 'Claude Code', 'Antigravity', 'Cursor' (defaults to the MCP client name)"),
					"task":              prop("string", "Short human-readable description of the current task"),
					"status":            prop("string", "working (default), idle, done, or left"),
					"action":            prop("string", "Current step, e.g. 'Editing components/Navbar.vue' or 'Running tests'"),
					"todo_id":           prop("integer", "ID of the vibeasik TODO being worked on (status 'done' counts it as finished)"),
					"project_id":        prop("integer", "ID of the vibeasik project being worked on"),
					"project_name":      prop("string", "Project name when it is not registered in vibeasik"),
					"model":             prop("string", "Model powering the agent, e.g. 'claude-opus-5-5'"),
					"session_id":        prop("string", "Stable session identifier, only needed when several sessions share one MCP connection"),
					"parent_session_id": prop("string", "Division mode: set by a sub agent to the session_id of the coordinator that assigned it (shown as its subordinate; status 'done' closes the sub agent)"),
				},
			},
		},
	}
}

// callActivityTool returns handled=false when name is not an activity tool.
func (s *Server) callActivityTool(name string, args map[string]interface{}) (*ToolCallResult, bool, error) {
	if name != activity.ReportToolName {
		return nil, false, nil
	}
	task, _ := args["task"].(string)
	status, _ := args["status"].(string)
	if status == "" {
		status = "working"
	}
	msg := fmt.Sprintf("Aktivitas tercatat (%s). Pantau di Studio: /agent-office", status)
	if task != "" {
		msg = fmt.Sprintf("Aktivitas tercatat (%s): %s. Pantau di Studio: /agent-office", status, task)
	}
	return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: msg}}}, true, nil
}
