package mcp

import (
	"encoding/json"
	"fmt"
)

// toolHandler mengeksekusi satu tool MCP inti; args sudah berupa JSON longgar dari klien.
type toolHandler func(s *Server, args map[string]interface{}) (*ToolCallResult, error)

// coreToolHandlers memetakan nama tool inti ke penanganannya (dispatch O(1)). Tool yang
// dimiliki berkas tools_*.go lain (extended, runner, activity, codemap, paging, knowledge)
// dan modul terdaftar dicoba lebih dulu oleh callTool.
var coreToolHandlers = map[string]toolHandler{
	"list_projects":               (*Server).toolListProjects,
	"get_project_detail":          (*Server).toolGetProjectDetail,
	"scan_directory":              (*Server).toolScanDirectory,
	"run_diagnostics":             (*Server).toolRunDiagnostics,
	"get_ai_recommendations":      (*Server).toolAIRecommendations,
	"vibeasik_tutorials":          (*Server).toolTutorials,
	"vibeasik_search_domain":      (*Server).toolSearchDomain,
	"vibeasik_deploy_plan":        (*Server).toolDeployPlan,
	"vibeasik_list_local_agents":  (*Server).toolListLocalAgents,
	"vibeasik_run_local_agent":    (*Server).toolRunLocalAgent,
	"vibeasik_ai_templates":       (*Server).toolAITemplates,
	"vibeasik_inject_ai_template": (*Server).toolInjectAITemplate,
	"vibeasik_check_ports":        (*Server).toolCheckPorts,
	"vibeasik_rea_status":         (*Server).toolREAStatus,
	"vibeasik_rea_analyze":        (*Server).toolREAAnalyze,
	"get_current_project":         (*Server).toolCurrentProject,
	"link_project_workspace":      (*Server).toolLinkProjectWorkspace,
	"list_todos":                  (*Server).toolListTodos,
	"create_todo":                 (*Server).toolCreateTodo,
	"update_todo":                 (*Server).toolUpdateTodo,
	"list_guidelines":             (*Server).toolListGuidelines,
	"suggest_todos":               (*Server).toolSuggestTodos,
}

// textResult membungkus satu teks sebagai hasil tool.
func textResult(text string) *ToolCallResult {
	return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: text}}}
}

// prettyJSONText meniru json.MarshalIndent yang kegagalannya diabaikan (teks kosong).
func prettyJSONText(v interface{}) *ToolCallResult {
	data, _ := json.MarshalIndent(v, "", "  ")
	return textResult(string(data))
}

func (s *Server) callTool(name string, args map[string]interface{}) (*ToolCallResult, error) {
	if res, handled, err := s.callExtendedTool(name, args); handled {
		return res, err
	}
	if res, handled, err := s.callRunnerTool(name, args); handled {
		return res, err
	}
	if res, handled, err := s.callActivityTool(name, args); handled {
		return res, err
	}
	if res, handled, err := s.callCodeMapTool(name, args); handled {
		return res, err
	}
	if res, handled, err := s.callFindingsTool(name, args); handled {
		return res, err
	}
	if res, handled, err := s.callPagingTool(name, args); handled {
		return res, err
	}
	if res, handled, err := s.callKnowledgeTool(name, args); handled {
		return res, err
	}
	if res, handled, err := s.callModuleTool(name, args); handled {
		return res, err
	}
	if h, ok := coreToolHandlers[name]; ok {
		return h(s, args)
	}
	return nil, fmt.Errorf("unknown tool: %s", name)
}
