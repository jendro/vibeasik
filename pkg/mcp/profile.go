package mcp

import "strings"

// Profil tool MCP: daftar tool (tools/list) dikirim utuh ke model di awal setiap sesi agent — ±120 tool ≈ 95 KB.
// Agent yang hanya mengerjakan todo cukup memakai profil "todo" (VIBEASIK_MCP_PROFILE=todo di konfigurasi MCP
// agent) sehingga konteks awalnya jauh lebih ringan. Profil hanya menyaring tools/list; tools/call tetap
// menerima semua tool agar alur yang menyebut tool di luar profil tidak putus.
const profileEnv = "VIBEASIK_MCP_PROFILE"

// todoCoreTools tool profil "todo" yang disediakan build Community.
var todoCoreTools = []string{
	"get_current_project", "link_project_workspace", "list_todos", "create_todo", "update_todo", "delete_todo",
	"get_latest_diagnostic", "vibeasik_knowledge_recall", "vibeasik_code_map_search", "vibeasik_code_map_file",
	"vibeasik_code_map_overview", "vibeasik_code_map_routes", "vibeasik_code_map_cost", "vibeasik_paging_plan", "vibeasik_check_ports",
	"vibeasik_run_component", "vibeasik_run_status", "vibeasik_report_activity",
}

// todoModuleTools tool profil "todo" yang disediakan modul Pro lewat registry; tidak tampil bila modulnya tidak ada.
var todoModuleTools = []string{
	"vibeasik_todo_dispatch_plan", "vibeasik_dispatch_agent", "vibeasik_agent_runs", "vibeasik_qa_gate",
	"vibeasik_staff_brief", "record_audit_feedback", "get_project_base_knowledge", "vibeasik_design_guidance",
}

var profiles = map[string][]string{
	"todo": append(append([]string{}, todoCoreTools...), todoModuleTools...),
}

// filterProfile menyaring tool menurut profil; profil kosong/"full"/tak dikenal = semua tool.
func filterProfile(tools []Tool, profile string) []Tool {
	names, ok := profiles[strings.ToLower(strings.TrimSpace(profile))]
	if !ok {
		return tools
	}
	keep := make(map[string]bool, len(names))
	for _, n := range names {
		keep[n] = true
	}
	out := make([]Tool, 0, len(names))
	for _, t := range tools {
		if keep[t.Name] {
			out = append(out, t)
		}
	}
	return out
}
