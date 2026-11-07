package mcp

import (
	"strings"

	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/service"
)

// ToolRunsCommands reports whether a tools/call would execute a shell command or store a dev_command the
// runner later executes. The HTTP transport uses it to apply the same loopback-only policy as the REST
// exec routes; stdio is local by definition and skips it.
func ToolRunsCommands(name string, args map[string]interface{}) bool {
	switch name {
	case "vibeasik_run_component", "vibeasik_run_local_agent", "vibeasik_runtime_simulation", "vibeasik_dispatch_agent",
		"vibeasik_dispatch_agent_batch", "vibeasik_retry_agent_run":
		return true
	case "vibeasik_staff_agenda_add":
		// Agenda CLI dikerjakan agent coding di laptop (read-only, tetap menjalankan proses).
		executor, _ := args["executor"].(string)
		kind, _ := args["kind"].(string)
		return strings.EqualFold(strings.TrimSpace(executor), "cli") || (strings.TrimSpace(executor) == "" && strings.EqualFold(strings.TrimSpace(kind), "investigate"))
	case "vibeasik_pm_request_plan", "suggest_todos":
		// Dengan auto-assign, keputusan rapat PM langsung dijalankan agent coding.
		return registry.AutoAssignOn()
	case "add_component":
		// Decode like the tool does: json matches keys case-insensitively, so a raw map lookup is not enough.
		var input service.AddComponentInput
		return decodeArgs(args, &input) != nil || strings.TrimSpace(input.DevCommand) != ""
	case "create_project":
		var input service.CreateProjectInput
		if decodeArgs(args, &input) != nil {
			return true
		}
		for _, comp := range input.Components {
			if strings.TrimSpace(comp.DevCommand) != "" {
				return true
			}
		}
	}
	return false
}
