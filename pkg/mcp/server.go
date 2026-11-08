package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/activity"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
	"github.com/vibeasik/vibeasik/pkg/runner"
	"github.com/vibeasik/vibeasik/pkg/service"
	"github.com/vibeasik/vibeasik/pkg/workspace"
)

// Server coordinates MCP protocol requests
type Server struct {
	db        *gorm.DB
	services  *service.Services
	runner    *runner.Service
	daemonURL string // diisi pada mode stdio bila local engine sedang berjalan
}

// NewServer returns an initialized MCP server
func NewServer(db *gorm.DB) *Server {
	return &Server{db: db, services: service.NewServices(repository.NewRepositories(db), db), runner: runner.NewService(db)}
}

// HandleRPC processes a single JSON-RPC MCP request and returns the response
func (s *Server) HandleRPC(req JSONRPCRequest) JSONRPCResponse {
	return s.HandleRPCFrom(req, activity.Caller{})
}

// HandleRPCFrom is HandleRPC with the caller's connection identity, so the request is
// shown as activity of that agent in the Agent Office. An empty Caller.Key disables tracking.
func (s *Server) HandleRPCFrom(req JSONRPCRequest, caller activity.Caller) JSONRPCResponse {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		var params struct {
			ClientInfo struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"clientInfo"`
		}
		_ = json.Unmarshal(req.Params, &params)
		activity.ClientConnected(caller, params.ClientInfo.Name, params.ClientInfo.Version)
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools":     map[string]interface{}{},
				"resources": map[string]interface{}{},
				"prompts":   map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "vibeasik",
				"version": "1.0.0",
			},
			"instructions": workspace.KnowledgeFirst + " " + workspace.TodoWorkflow,
		}

	case "notifications/initialized":
		return resp

	case "tools/list":
		resp.Result = map[string]interface{}{
			"tools": s.getToolsList(),
		}

	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &RPCError{Code: -32602, Message: "Invalid params: " + err.Error()}
			return resp
		}
		activity.ToolCalled(caller, params.Name, params.Arguments)
		result, err := s.callTool(params.Name, params.Arguments)
		activity.ToolFinished(caller, params.Name, err)
		if err != nil {
			resp.Result = ToolCallResult{
				IsError: true,
				Content: []ToolContent{{Type: "text", Text: "Error: " + err.Error()}},
			}
		} else {
			resp.Result = result
		}

	case "resources/list":
		var projects []models.Project
		s.db.Find(&projects)
		resources := make([]map[string]interface{}, 0)
		for _, p := range projects {
			resources = append(resources, map[string]interface{}{
				"uri":         fmt.Sprintf("project://%d/overview", p.ID),
				"name":        p.Name + " Overview",
				"description": fmt.Sprintf("Architecture overview for %s (%s)", p.Name, p.ProjectType),
				"mimeType":    "application/json",
			})
		}
		resp.Result = map[string]interface{}{"resources": resources}

	case "resources/read":
		var params struct {
			URI string `json:"uri"`
		}
		_ = json.Unmarshal(req.Params, &params)
		resp.Result = map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"uri":      params.URI,
					"mimeType": "text/plain",
					"text":     s.readResource(params.URI),
				},
			},
		}

	case "prompts/list":
		resp.Result = map[string]interface{}{
			"prompts": []map[string]interface{}{
				{
					"name":        "audit_project_bundle",
					"description": "Thoroughly review a project bundle consisting of Backend and multiple Frontends.",
					"arguments": []map[string]interface{}{
						{"name": "project_id", "description": "ID of the project to audit", "required": true},
					},
				},
				{
					"name":        "plan_cross_stack_feature",
					"description": "Plan a feature that requires concurrent changes in Go Backend and Nuxt/Flutter Frontends.",
					"arguments": []map[string]interface{}{
						{"name": "project_id", "description": "Project ID", "required": true},
						{"name": "feature_description", "description": "Description of the feature to implement", "required": true},
					},
				},
			},
		}

	default:
		resp.Error = &RPCError{Code: -32601, Message: fmt.Sprintf("Method '%s' not found", req.Method)}
	}

	return resp
}

func (s *Server) getToolsList() []Tool {
	return filterProfile(s.allTools(), os.Getenv(profileEnv))
}

func (s *Server) allTools() []Tool {
	tools := append(append(s.coreToolsList(), s.extendedToolsList()...), s.runnerToolsList()...)
	tools = append(tools, s.codeMapToolsList()...)
	tools = append(tools, s.pagingToolsList()...)
	tools = append(tools, s.knowledgeToolsList()...)
	tools = append(tools, s.activityToolsList()...)
	return append(tools, moduleTools()...)
}

func (s *Server) coreToolsList() []Tool {
	return []Tool{
		{
			Name:        "list_projects",
			Description: "Lists all software projects managed locally (compact: id, name, slug, type, root path, summary, components). Use get_project_detail or detail: true for full records.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"detail": map[string]interface{}{"type": "boolean", "description": "Return full project records incl. component scan data (large)"},
				},
			},
		},
		{
			Name:        "get_project_detail",
			Description: "Retrieves complete architecture detail of a project, including all Backend and Frontend components, ports, and dev commands.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{"type": "integer", "description": "ID of the project"},
					"slug":       map[string]interface{}{"type": "string", "description": "Slug of the project"},
				},
			},
		},
		{
			Name:        "scan_directory",
			Description: "Scans any local directory on the machine to automatically discover Go, Nuxt, Vue, React, Flutter, Python stacks, ports, and bundle topologies.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"path": map[string]interface{}{"type": "string", "description": "Local absolute path to scan"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "run_diagnostics",
			Description: "Executes an automated code health audit on a managed project. Checks port conflicts, git hygiene, security vulnerabilities, and architecture cohesion.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{"type": "integer", "description": "ID of the project to diagnose"},
				},
				Required: []string{"project_id"},
			},
		},
		{
			Name:        "get_ai_recommendations",
			Description: "Generates tailored architectural, DevOps, and performance recommendations with ready-to-run prompts for AI agents.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{"type": "integer", "description": "ID of the project"},
				},
				Required: []string{"project_id"},
			},
		},
		{
			Name:        "vibeasik_tutorials",
			Description: "Fetches interactive educational guides from vibeasik.com covering web architecture, database, security, and AI agent integration.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"category": map[string]interface{}{"type": "string", "description": "Category filter: dasar-pemrograman, database, security, deployment, ai-agent"},
				},
			},
		},
		{
			Name:        "vibeasik_search_domain",
			Description: "Searches domain availability, pricing (IDR), and auto-DNS configuration for vibecoders.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Domain name to search, e.g. 'myvibeapp'"},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "vibeasik_deploy_plan",
			Description: "Generates production deployment blueprints (Dockerfile, docker-compose.yml, Caddyfile with Auto-SSL) for a project.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_name": map[string]interface{}{"type": "string", "description": "Name of the project"},
					"language":     map[string]interface{}{"type": "string", "description": "Language: go, typescript, python"},
					"framework":    map[string]interface{}{"type": "string", "description": "Framework: fiber, nuxt, fastapi"},
					"port":         map[string]interface{}{"type": "integer", "description": "Port of the service"},
					"domain":       map[string]interface{}{"type": "string", "description": "Target domain (e.g. app.myvibe.com)"},
				},
				Required: []string{"project_name", "language", "framework"},
			},
		},
		{
			Name:        "vibeasik_list_local_agents",
			Description: "Discovers and lists all installed local terminal AI agents on developer's machine (agy, claude, aider, custom).",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
			},
		},
		{
			Name:        "vibeasik_run_local_agent",
			Description: "Executes a task prompt using an installed local terminal AI agent (agy, claude, aider) inside a target project folder.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_key":    map[string]interface{}{"type": "string", "description": "Agent key: agy, claude, aider, or custom key"},
					"prompt":       map[string]interface{}{"type": "string", "description": "Instruction/task for the agent to perform"},
					"project_path": map[string]interface{}{"type": "string", "description": "Target folder directory path"},
				},
				Required: []string{"agent_key", "prompt"},
			},
		},
		{
			Name:        "vibeasik_ai_templates",
			Description: "Retrieves official AI templates (System One reasoning client, MCP server boilerplate) for Go, TypeScript, Python.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"type": map[string]interface{}{"type": "string", "description": "Optional template type: system-one or mcp-server"},
				},
			},
		},
		{
			Name:        "vibeasik_inject_ai_template",
			Description: "Injects an AI boilerplate template into a target local project directory.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"template_id": map[string]interface{}{"type": "string", "description": "Template ID: tmpl-system-one-go or tmpl-mcp-server-go"},
					"target_path": map[string]interface{}{"type": "string", "description": "Target folder directory path"},
				},
				Required: []string{"template_id", "target_path"},
			},
		},
		{
			Name:        "vibeasik_check_ports",
			Description: "Inspects a list of network ports on 127.0.0.1 to detect active (in use) vs free ports.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"ports": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "integer"},
						"description": "List of TCP port numbers to check (e.g. [3000, 8765, 8770])",
					},
				},
				Required: []string{"ports"},
			},
		},
		{
			Name:        "get_current_project",
			Description: "Identifies the current project and component from the workspace path or .vibeasik.json file. Returns project architecture, connected components, and active TODO tasks.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Optional workspace directory path. If omitted, checks current working directory or .vibeasik.json.",
					},
				},
			},
		},
		{
			Name:        "link_project_workspace",
			Description: "Initializes a project for AI agents ('Init vibeasik'): writes the .vibeasik.json identifier into the project root and every component (project ID, TODO & code map instructions), builds the code map and writes .vibeasik/CODEMAP.md + codemap.jsonl (kept up to date automatically while vibeasik serve runs) and adds .vibeasik/ to .gitignore. With component_id only that component's .vibeasik.json is written.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id":   map[string]interface{}{"type": "integer", "description": "ID of the project to link"},
					"component_id": map[string]interface{}{"type": "integer", "description": "Optional component ID to link"},
				},
				Required: []string{"project_id"},
			},
		},
		{
			Name:        "list_todos",
			Description: "Lists TODOs across 3-tier hierarchy: global (Platform), project (Bundle), component (Module/Service). Compact by default (memory = short work history instead of full rationale; detail: true for full records), open todos only unless status/include_closed is given, newest first up to limit (default 50). To check whether something was already done or rejected, pass query (similarity search over all statuses). Supports filters for category, level, project_id, component_id, status, creator_type, and is_ai_suggestion. If project_id is omitted, auto-resolves from path or .vibeasik.json. To work todos, use vibeasik_todo_dispatch_plan (default division mode: you coordinate, one fresh sub agent and one commit per todo). Whoever works a todo: set it in_progress, re-check in the actual code that the problem still exists (line numbers shift; it may already be fixed), treat code examples as references to adapt rather than paste, and cover every location the todo lists.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"path":             map[string]interface{}{"type": "string", "description": "Optional directory path. If project_id is omitted, auto-resolves project_id from this path or .vibeasik.json in current directory."},
					"category":         map[string]interface{}{"type": "string", "description": "Filter by category: security, ui_ux, feature, architecture, performance, backend, devops, general"},
					"level":            map[string]interface{}{"type": "string", "description": "Filter by level: global, project, component"},
					"project_id":       map[string]interface{}{"type": "integer", "description": "Optional project ID filter"},
					"component_id":     map[string]interface{}{"type": "integer", "description": "Optional component ID filter"},
					"status":           map[string]interface{}{"type": "string", "description": "Status: pending, in_progress, completed, suggested, rejected"},
					"creator_type":     map[string]interface{}{"type": "string", "description": "Creator type: human or ai_agent"},
					"source":           map[string]interface{}{"type": "string", "description": "Source/origin: autonomous_executive, diagnostic_audit, pm_agent, ai_agent, manual"},
					"is_ai_suggestion": map[string]interface{}{"type": "boolean", "description": "Filter for AI-suggested todos only"},
					"tag":              map[string]interface{}{"type": "string", "description": "Filter by one focus tag: fitur, keamanan-dasar, optimasi-query, rawat-ai, belajar"},
					"query":            map[string]interface{}{"type": "string", "description": "Similarity search (title/description/memory) over all statuses, best matches first"},
					"include_closed":   map[string]interface{}{"type": "boolean", "description": "Include completed & rejected todos when status is not given"},
					"detail":           map[string]interface{}{"type": "boolean", "description": "Return full todo records (large)"},
					"limit":            map[string]interface{}{"type": "integer", "description": "Max todos returned (default 50, max 200)"},
				},
			},
		},
		{
			Name:        "create_todo",
			Description: "Creates a new actionable TODO or planning item in the 3-tier hierarchy (global, project, component) with creator tagging (human vs ai_agent). Near-duplicates of an existing todo in the project are refused (the existing one is returned) unless force: true; weaker matches are listed as a warning.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"title":              map[string]interface{}{"type": "string", "description": "Title of the task or TODO"},
					"level":              map[string]interface{}{"type": "string", "description": "Level in hierarchy: global (Platform), project (Bundle), component (Module/Service)"},
					"category":           map[string]interface{}{"type": "string", "description": "Category: feature, security, ui_ux, architecture, performance, backend, devops, general (default: general)"},
					"parent_id":          map[string]interface{}{"type": "integer", "description": "Optional parent TODO ID for cascading hierarchy"},
					"project_id":         map[string]interface{}{"type": "integer", "description": "Optional project ID to link with"},
					"component_id":       map[string]interface{}{"type": "integer", "description": "Optional component ID to link with"},
					"description":        map[string]interface{}{"type": "string", "description": "Detailed description or context of the task"},
					"priority":           map[string]interface{}{"type": "string", "description": "Priority: low, medium, high, critical (default: medium)"},
					"creator_type":       map[string]interface{}{"type": "string", "description": "Creator: human or ai_agent (default: ai_agent when called by agent)"},
					"creator_name":       map[string]interface{}{"type": "string", "description": "Name of the creator or agent (e.g. Antigravity, Claude Code)"},
					"is_ai_suggestion":   map[string]interface{}{"type": "boolean", "description": "True if this is an AI recommendation awaiting human approval"},
					"guideline_ref":      map[string]interface{}{"type": "string", "description": "Guideline rubric ID (e.g. PLAN-001, ARCH-001, SEC-001)"},
					"tags":               map[string]interface{}{"type": "string", "description": "Comma-separated focus tags: fitur, keamanan-dasar, optimasi-query, rawat-ai (main focus) or belajar (other topics kept for learning)"},
					"rationale":          map[string]interface{}{"type": "string", "description": "Technical rationale explaining why this todo is recommended"},
					"completed_by_agent": map[string]interface{}{"type": "string", "description": "Optional coding agent name if task is created already completed (e.g. Antigravity, Claude Code, Aider, Cursor)"},
					"completed_by_model": map[string]interface{}{"type": "string", "description": "Optional AI model name if task is created already completed (e.g. gemini-3.8-flash, claude-3.7-sonnet)"},
					"completed_at":       map[string]interface{}{"type": "string", "description": "Optional ISO8601 completion timestamp if task is already completed"},
					"complexity":         complexityArgSchema,
					"impact":             impactArgSchema,
					"force":              map[string]interface{}{"type": "boolean", "description": "Create even when a near-duplicate todo exists"},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "update_todo",
			Description: "Updates an existing TODO task status (pending, in_progress, completed, suggested, rejected), title, priority, complexity/impact rating, or records task completion details (completed_at timestamp, completed_by_agent, completed_by_model, completion_note). Only mark 'completed' after re-reviewing your diff against the todo's done criteria and running the relevant build/tests; pass completion_note with what changed, files touched and how it was verified (or the evidence that it was already fulfilled).",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"todo_id":            map[string]interface{}{"type": "integer", "description": "ID of the TODO to update"},
					"status":             map[string]interface{}{"type": "string", "description": "New status: pending, in_progress, completed, suggested, rejected"},
					"title":              map[string]interface{}{"type": "string", "description": "Updated title"},
					"category":           map[string]interface{}{"type": "string", "description": "Updated category: feature, security, ui_ux, architecture, performance, backend, devops, general"},
					"priority":           map[string]interface{}{"type": "string", "description": "Updated priority: low, medium, high, critical"},
					"tags":               map[string]interface{}{"type": "string", "description": "Comma-separated focus tags: fitur, keamanan-dasar, optimasi-query, rawat-ai (main focus) or belajar (other topics kept for learning)"},
					"completed_by_agent": map[string]interface{}{"type": "string", "description": "Name/identifier of the AI coding agent that worked on and completed the task (e.g. 'Antigravity', 'Claude Code', 'Aider', 'Cursor')"},
					"completed_by_model": map[string]interface{}{"type": "string", "description": "Name/identifier of the AI model used by the agent (e.g. 'gemini-3.8-flash', 'claude-3.7-sonnet', 'gpt-4o')"},
					"completed_at":       map[string]interface{}{"type": "string", "description": "Optional ISO8601 completion timestamp (e.g. '2026-10-19T10:00:00Z'). Automatically defaults to current time if status is completed and omitted."},
					"completion_note":    map[string]interface{}{"type": "string", "description": "Verification summary appended to the todo rationale: what changed, files touched, how it was verified, or evidence it was already fulfilled"},
					"complexity":         complexityArgSchema,
					"impact":             impactArgSchema,
				},
				Required: []string{"todo_id"},
			},
		},
		{
			Name:        "list_guidelines",
			Description: "Lists all standard architecture, security, performance, and planning rubrics/guidelines used by AI agents to audit projects and give suggestions.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"category": map[string]interface{}{"type": "string", "description": "Optional category filter: perencanaan_hierarki, arsitektur_dan_desain, backend_dan_api, frontend_dan_antarmuka, keamanan_dan_hygiene, performa_dan_jaringan, ai_dan_mcp_integration"},
				},
			},
		},
		{
			Name:        "suggest_todos",
			Description: "Inspects project topology against vibeasik guidelines (instant rule-based suggestions) and, for a project, hands the deeper planning to vibeasik's Project Manager agent: it holds a meeting with specialist agents on high-intelligence LLMs and decides the next todos in the background. Prefer vibeasik_pm_request_plan with a note when you just finished a todo.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{"type": "integer", "description": "Optional project ID. If omitted, generates suggestions for all projects and global workspace."},
					"save_to_db": map[string]interface{}{"type": "boolean", "description": "If true, automatically saves suggestions to DB with status 'suggested'."},
				},
			},
		},
		{
			Name:        "vibeasik_rea_status",
			Description: "Checks if REA (Reverse Engineer Anything) toolkit is installed on this machine, returns version and capabilities. REA enables AI agents to inspect binaries, Electron apps, .NET assemblies, Android APKs, websites, and SQLite databases without source code.",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
			},
		},
		{
			Name:        "vibeasik_rea_analyze",
			Description: "Runs REA reverse engineering analysis on a target path (JavaScript/Electron app directory, ASAR file, binary, or APK). Returns structured evidence including modules, imports, IPC boundaries, and relationships.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"target_path":   map[string]interface{}{"type": "string", "description": "Absolute path to the target application directory, binary file, or ASAR archive to analyze"},
					"analysis_type": map[string]interface{}{"type": "string", "description": "Type of analysis: javascript (default), native, android, website, sqlite"},
				},
				Required: []string{"target_path"},
			},
		},
	}
}

// GetToolsList returns public tool definition list
func (s *Server) GetToolsList() []Tool {
	return s.getToolsList()
}

func (s *Server) readResource(uri string) string {
	return fmt.Sprintf("MCP Resource for %s\nvibeasik.com System is active.", uri)
}

// RunStdioLoop reads MCP requests from stdin and responds on stdout (for AI agent CLI integration)
func (s *Server) RunStdioLoop(in io.Reader, out io.Writer) error {
	// Satu proses stdio = satu sesi agent (Claude Code, Antigravity, ...) di Kantor Agent.
	caller := activity.Caller{Key: fmt.Sprintf("mcp-stdio:%d", os.Getpid()), Transport: "stdio"}
	defer activity.ClientDisconnected(caller)
	reader := bufio.NewReader(in)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		line = []byte(strings.TrimSpace(string(line)))
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		// JSON-RPC Notification: must not receive a response
		if req.ID == nil || strings.HasPrefix(req.Method, "notifications/") {
			s.HandleRPCFrom(req, caller)
			continue
		}

		resp := s.HandleRPCFrom(req, caller)
		outBytes, _ := json.Marshal(resp)
		out.Write(outBytes)
		out.Write([]byte("\n"))
	}
}
