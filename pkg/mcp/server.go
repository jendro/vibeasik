package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/activity"
	"github.com/vibeasik/vibeasik/pkg/agents"
	"github.com/vibeasik/vibeasik/pkg/cloud"
	"github.com/vibeasik/vibeasik/pkg/diagnostics"
	"github.com/vibeasik/vibeasik/pkg/guidelines"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/repository"
	"github.com/vibeasik/vibeasik/pkg/runner"
	"github.com/vibeasik/vibeasik/pkg/scanner"
	"github.com/vibeasik/vibeasik/pkg/service"
	"github.com/vibeasik/vibeasik/pkg/standard"
	"github.com/vibeasik/vibeasik/pkg/todorag"
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
	if res, handled, err := s.callPagingTool(name, args); handled {
		return res, err
	}
	if res, handled, err := s.callKnowledgeTool(name, args); handled {
		return res, err
	}
	if res, handled, err := s.callModuleTool(name, args); handled {
		return res, err
	}

	switch name {
	case "list_projects":
		var projects []models.Project
		s.db.Preload("Components").Find(&projects)
		var out interface{} = projects
		if detail, _ := args["detail"].(bool); !detail {
			compact := make([]compactProject, len(projects))
			for i, p := range projects {
				compact[i] = toCompactProject(p)
			}
			out = compact
		}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return nil, err
		}
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "get_project_detail":
		var p models.Project
		if pid, ok := args["project_id"]; ok {
			s.db.Preload("Components").Preload("Diagnostics").First(&p, pid)
		} else if slug, ok := args["slug"]; ok {
			s.db.Preload("Components").Preload("Diagnostics").Where("slug = ?", slug).First(&p)
		}
		if p.ID == 0 {
			return nil, fmt.Errorf("project not found")
		}
		data, _ := json.MarshalIndent(p, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "scan_directory":
		path, ok := args["path"].(string)
		if !ok || path == "" {
			return nil, fmt.Errorf("path is required")
		}
		scanRes, err := scanner.ScanDirectory(path)
		if err != nil {
			return nil, err
		}
		data, _ := json.MarshalIndent(scanRes, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "run_diagnostics":
		var pid uint
		switch v := args["project_id"].(type) {
		case float64:
			pid = uint(v)
		case string:
			parsed, _ := strconv.Atoi(v)
			pid = uint(parsed)
		}
		var p models.Project
		s.db.Preload("Components").First(&p, pid)
		if p.ID == 0 {
			return nil, fmt.Errorf("project with ID %d not found", pid)
		}

		var allProjects []models.Project
		s.db.Preload("Components").Find(&allProjects)

		diag, err := diagnostics.AnalyzeProject(&p, allProjects)
		if err != nil {
			return nil, err
		}
		s.db.Create(diag)
		p.HealthScore = diag.Score
		s.db.Save(&p)

		data, _ := json.MarshalIndent(diag, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "get_ai_recommendations":
		var pid uint
		switch v := args["project_id"].(type) {
		case float64:
			pid = uint(v)
		case string:
			parsed, _ := strconv.Atoi(v)
			pid = uint(parsed)
		}
		var p models.Project
		s.db.Preload("Components").First(&p, pid)
		if p.ID == 0 {
			return nil, fmt.Errorf("project not found")
		}
		var allProjects []models.Project
		s.db.Preload("Components").Find(&allProjects)

		diag, err := diagnostics.AnalyzeProject(&p, allProjects)
		if err != nil {
			return nil, err
		}
		return &ToolCallResult{
			Content: []ToolContent{
				{Type: "text", Text: fmt.Sprintf("=== AI Recommendations for %s ===\n%s", p.Name, diag.Optimizations)},
			},
		}, nil

	case "vibeasik_tutorials":
		category, _ := args["category"].(string)
		cli := cloud.NewClient()
		tuts, err := cli.GetTutorials(category)
		if err != nil {
			return nil, err
		}
		data, _ := json.MarshalIndent(tuts, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "vibeasik_search_domain":
		query, ok := args["query"].(string)
		if !ok || query == "" {
			return nil, fmt.Errorf("query is required")
		}
		cli := cloud.NewClient()
		domains, err := cli.SearchDomains(query)
		if err != nil {
			return nil, err
		}
		data, _ := json.MarshalIndent(domains, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "vibeasik_deploy_plan":
		cli := cloud.NewClient()
		plan, err := cli.GenerateDeployPlan(args)
		if err != nil {
			return nil, err
		}
		data, _ := json.MarshalIndent(plan, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "vibeasik_list_local_agents":
		reg := agents.GetRegistry()
		list := reg.DiscoverAll(s.db)
		data, _ := json.MarshalIndent(list, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "vibeasik_run_local_agent":
		agentKey, _ := args["agent_key"].(string)
		prompt, _ := args["prompt"].(string)
		projectPath, _ := args["project_path"].(string)

		req := models.AgentExecutionRequest{
			AgentKey:    agentKey,
			Prompt:      prompt,
			ProjectPath: projectPath,
			TimeoutSec:  120,
		}

		reg := agents.GetRegistry()
		res, err := reg.Execute(context.Background(), req, s.db)
		if err != nil {
			return nil, err
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "vibeasik_ai_templates":
		cli := cloud.NewClient()
		tmpls, err := cli.GetAITemplates()
		if err != nil {
			return nil, err
		}
		data, _ := json.MarshalIndent(tmpls, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "vibeasik_inject_ai_template":
		tmplID, _ := args["template_id"].(string)
		targetPath, _ := args["target_path"].(string)
		if tmplID == "" || targetPath == "" {
			return nil, fmt.Errorf("template_id and target_path are required")
		}
		cli := cloud.NewClient()
		tmpls, err := cli.GetAITemplates()
		if err != nil {
			return nil, err
		}
		var matched *cloud.AITemplateDTO
		for _, t := range tmpls {
			if t.ID == tmplID {
				matched = &t
				break
			}
		}
		if matched == nil {
			return nil, fmt.Errorf("template %s not found", tmplID)
		}
		var createdFiles []string
		for _, f := range matched.Files {
			fullPath := filepath.Join(targetPath, filepath.FromSlash(f.Filename))
			_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
			if err := os.WriteFile(fullPath, []byte(f.Content), 0644); err != nil {
				return nil, err
			}
			createdFiles = append(createdFiles, f.Filename)
		}
		return &ToolCallResult{
			Content: []ToolContent{
				{Type: "text", Text: fmt.Sprintf("Template %s injected successfully into %s. Created files: %s", matched.Name, targetPath, strings.Join(createdFiles, ", "))},
			},
		}, nil

	case "vibeasik_check_ports":
		portsRaw, ok := args["ports"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("ports array required")
		}
		results := make(map[string]bool)
		for _, p := range portsRaw {
			var port int
			switch v := p.(type) {
			case float64:
				port = int(v)
			case int:
				port = v
			}
			if port > 0 {
				conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 150*time.Millisecond)
				if err == nil {
					conn.Close()
					results[strconv.Itoa(port)] = true // in use
				} else {
					results[strconv.Itoa(port)] = false // free
				}
			}
		}
		data, _ := json.MarshalIndent(results, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "vibeasik_rea_status":
		reg := agents.GetRegistry()
		adapter, ok := reg.Get("rea")
		if !ok {
			return &ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: `{"installed": false, "message": "REA adapter not registered"}`}},
			}, nil
		}
		detected, path, version := adapter.Detect()
		status := map[string]interface{}{
			"installed":    detected,
			"path":         path,
			"version":      version,
			"capabilities": adapter.Capabilities(),
			"install_cmd":  "npm install --global rea-agents",
			"setup_cmd":    "npx rea-agents setup",
			"description":  adapter.Description(),
		}
		if !detected {
			status["message"] = "REA belum terpasang. Jalankan: npm install --global rea-agents && npx rea-agents setup"
		}
		data, _ := json.MarshalIndent(status, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "vibeasik_rea_analyze":
		targetPath, _ := args["target_path"].(string)
		if targetPath == "" {
			return nil, fmt.Errorf("target_path is required")
		}
		analysisType, _ := args["analysis_type"].(string)
		if analysisType == "" {
			analysisType = "javascript"
		}

		// Build REA CLI command based on analysis type
		var cmdName string
		var cmdArgs []string

		// Try direct rea binary first, fallback to npx
		reaPath, err := exec.LookPath("rea")
		if err != nil {
			reaPath, err = exec.LookPath("npx")
			if err != nil {
				return nil, fmt.Errorf("REA belum terpasang. Jalankan: npm install --global rea-agents")
			}
			cmdName = reaPath
			cmdArgs = append(cmdArgs, "-y", "rea-agents@latest")
		} else {
			cmdName = reaPath
		}

		switch analysisType {
		case "javascript", "electron":
			cmdArgs = append(cmdArgs, "analyze-javascript-application", targetPath, "--json")
		case "native", "binary":
			cmdArgs = append(cmdArgs, "analyze", targetPath, "--json")
		case "android", "apk":
			cmdArgs = append(cmdArgs, "analyze", targetPath, "--json")
		case "sqlite", "database":
			cmdArgs = append(cmdArgs, "analyze", targetPath, "--json")
		default:
			cmdArgs = append(cmdArgs, "analyze-javascript-application", targetPath, "--json")
		}

		execCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		cmd := exec.CommandContext(execCtx, cmdName, cmdArgs...)
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		outputText := string(out)

		if err != nil && outputText == "" {
			outputText = fmt.Sprintf("REA analysis error: %s", err.Error())
		}

		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: outputText}},
		}, nil

	case "get_current_project":
		targetPath, _ := args["path"].(string)
		result, err := workspace.IdentifyWorkspace(s.db, targetPath, "http://127.0.0.1:8765")
		if err != nil {
			return nil, err
		}
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, err
		}
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "link_project_workspace":
		var pid uint
		switch v := args["project_id"].(type) {
		case float64:
			pid = uint(v)
		case int:
			pid = uint(v)
		case string:
			parsed, _ := strconv.Atoi(v)
			pid = uint(parsed)
		}
		if pid == 0 {
			return nil, fmt.Errorf("valid project_id is required")
		}

		var cid uint
		if cidVal, ok := args["component_id"]; ok && cidVal != nil {
			switch v := cidVal.(type) {
			case float64:
				cid = uint(v)
			case int:
				cid = uint(v)
			case string:
				parsed, _ := strconv.Atoi(v)
				cid = uint(parsed)
			}
		}

		// Tanpa component_id = "Init vibeasik": tautan AI + peta kode di repo yang diperbarui otomatis.
		if cid == 0 {
			res, err := workspace.InitProject(s.db, pid, "http://127.0.0.1:8765")
			if err != nil {
				return nil, err
			}
			return jsonResult(res)
		}
		createdFiles, idf, err := workspace.LinkProjectWorkspaces(s.db, pid, cid, "http://127.0.0.1:8765")
		if err != nil {
			return nil, err
		}
		res := map[string]interface{}{
			"message":       "Workspace linked successfully with AI Agent identifier (.vibeasik.json)",
			"project_id":    pid,
			"files_created": createdFiles,
			"identifier":    idf,
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "list_todos":
		query := s.db.Model(&models.Todo{})
		if lvl, ok := args["level"].(string); ok && lvl != "" {
			query = query.Where("level = ?", lvl)
		}
		var resolvedPID uint
		if pidVal, ok := args["project_id"]; ok && pidVal != nil {
			switch v := pidVal.(type) {
			case float64:
				resolvedPID = uint(v)
			case int:
				resolvedPID = uint(v)
			case string:
				parsed, _ := strconv.Atoi(v)
				resolvedPID = uint(parsed)
			}
		} else {
			targetPath, _ := args["path"].(string)
			if match, err := workspace.IdentifyWorkspace(s.db, targetPath, "http://127.0.0.1:8765"); err == nil && match.Project != nil {
				resolvedPID = match.Project.ID
				if match.CurrentComponent != nil && args["component_id"] == nil {
					query = query.Where("component_id = ? OR component_id IS NULL OR component_id = 0", match.CurrentComponent.ID)
				}
			}
		}
		if resolvedPID > 0 {
			query = query.Where("project_id = ?", resolvedPID)
		}
		if cidVal, ok := args["component_id"]; ok && cidVal != nil {
			var cid uint
			switch v := cidVal.(type) {
			case float64:
				cid = uint(v)
			case int:
				cid = uint(v)
			case string:
				parsed, _ := strconv.Atoi(v)
				cid = uint(parsed)
			}
			if cid > 0 {
				query = query.Where("component_id = ?", cid)
			}
		}
		search, _ := args["query"].(string)
		search = strings.TrimSpace(search)
		status, _ := args["status"].(string)
		includeClosed, _ := args["include_closed"].(bool)
		if status != "" {
			query = query.Where("status = ?", status)
		} else if search == "" && !includeClosed {
			// Default: hanya todo yang masih terbuka; riwayat selesai/ditolak dicari lewat query atau include_closed.
			query = query.Where("status NOT IN ?", []string{models.TodoStatusCompleted, models.TodoStatusRejected})
		}
		if creatorType, ok := args["creator_type"].(string); ok && creatorType != "" {
			query = query.Where("creator_type = ?", creatorType)
		}
		if isSugg, ok := args["is_ai_suggestion"].(bool); ok {
			query = query.Where("is_ai_suggestion = ?", isSugg)
		}
		if src, ok := args["source"].(string); ok && strings.TrimSpace(src) != "" && strings.ToLower(strings.TrimSpace(src)) != "all" {
			query = query.Where("source = ?", strings.TrimSpace(src))
		}
		if cat, ok := args["category"].(string); ok && strings.TrimSpace(cat) != "" && strings.ToLower(strings.TrimSpace(cat)) != "all" {
			query = query.Where("LOWER(category) = ?", strings.ToLower(strings.TrimSpace(cat)))
		}
		if tag, ok := args["tag"].(string); ok && tag != "all" {
			query = models.WhereTag(query, tag)
		}

		limit := intArg(args, "limit")
		if limit <= 0 {
			limit = listTodosDefaultLimit
		}
		limit = min(limit, listTodosMaxLimit)
		var total int64
		query.Session(&gorm.Session{}).Count(&total)

		var todos []models.Todo
		sims := map[uint]float64{}
		if search != "" {
			// RAG: urutkan menurut kemiripan dengan kueri di semua status (sudah pernah dikerjakan / ditolak?).
			hits := todorag.Similar(s.db, resolvedPID, search, search, limit, todorag.QuerySim)
			ids := make([]uint, 0, len(hits))
			for _, h := range hits {
				ids = append(ids, h.ID)
				sims[h.ID] = h.Sim
			}
			var found []models.Todo
			if len(ids) > 0 {
				query.Where("id IN ?", ids).Find(&found)
			}
			byID := map[uint]models.Todo{}
			for _, t := range found {
				byID[t.ID] = t
			}
			for _, id := range ids {
				if t, ok := byID[id]; ok {
					todos = append(todos, t)
				}
			}
			total = int64(len(todos))
		} else if err := query.Order("status ASC, created_at DESC").Limit(limit).Find(&todos).Error; err != nil {
			return nil, err
		}
		var out interface{} = todos
		if detail, _ := args["detail"].(bool); !detail {
			compact := make([]compactTodo, len(todos))
			for i, t := range todos {
				compact[i] = toCompactTodo(t)
				compact[i].Similarity = sims[t.ID]
				if search == "" {
					compact[i].Memory = oneLine(t.Memory, listMemoryMax) // daftar biasa: cukup ringkasan; kueri: memori utuh
				}
			}
			out = compact
		}
		data, err := json.MarshalIndent(map[string]interface{}{"total": total, "returned": len(todos), "todos": out}, "", "  ")
		if err != nil {
			return nil, err
		}
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "create_todo":
		title, _ := args["title"].(string)
		title = strings.TrimSpace(title)
		if title == "" {
			return nil, fmt.Errorf("title is required")
		}
		level, _ := args["level"].(string)
		desc, _ := args["description"].(string)
		prio, _ := args["priority"].(string)
		if prio == "" {
			prio = models.TodoPriorityMedium
		}
		creatorType, _ := args["creator_type"].(string)
		if creatorType == "" {
			creatorType = models.CreatorAI
		}
		creatorName, _ := args["creator_name"].(string)
		if creatorName == "" {
			creatorName = "ai_agent"
		}
		isAISugg, _ := args["is_ai_suggestion"].(bool)
		guidelineRef, _ := args["guideline_ref"].(string)
		rationale, _ := args["rationale"].(string)
		source, _ := args["source"].(string)
		if source == "" {
			source = "ai_agent"
		}

		status := models.TodoStatusPending
		if isAISugg {
			status = models.TodoStatusSuggested
		}

		todo := models.Todo{
			Title:          title,
			Description:    desc,
			Status:         status,
			Priority:       prio,
			CreatorType:    creatorType,
			CreatorName:    creatorName,
			IsAISuggestion: isAISugg,
			GuidelineRef:   guidelineRef,
			Rationale:      rationale,
			Source:         source,
		}

		if cat, ok := args["category"].(string); ok && strings.TrimSpace(cat) != "" {
			todo.Category = strings.ToLower(strings.TrimSpace(cat))
		} else {
			todo.Category = models.CategoryGeneral
		}
		if raw, ok := args["tags"]; ok {
			todo.Tags = models.NormalizeTags(raw)
		}

		if compAgent, ok := args["completed_by_agent"].(string); ok && compAgent != "" {
			todo.CompletedByAgent = compAgent
		}
		if compModel, ok := args["completed_by_model"].(string); ok && compModel != "" {
			todo.CompletedByModel = compModel
		}
		if compAtStr, ok := args["completed_at"].(string); ok && compAtStr != "" {
			if parsedTime, err := time.Parse(time.RFC3339, compAtStr); err == nil {
				todo.CompletedAt = &parsedTime
			}
		} else if status == models.TodoStatusCompleted {
			now := time.Now()
			todo.CompletedAt = &now
		}

		if pidVal, ok := args["project_id"]; ok && pidVal != nil {
			var pid uint
			switch v := pidVal.(type) {
			case float64:
				pid = uint(v)
			case int:
				pid = uint(v)
			case string:
				parsed, _ := strconv.Atoi(v)
				pid = uint(parsed)
			}
			if pid > 0 {
				todo.ProjectID = &pid
				var p models.Project
				if err := s.db.Select("name").First(&p, pid).Error; err == nil {
					todo.ProjectName = p.Name
				}
			}
		}

		if cidVal, ok := args["component_id"]; ok && cidVal != nil {
			var cid uint
			switch v := cidVal.(type) {
			case float64:
				cid = uint(v)
			case int:
				cid = uint(v)
			case string:
				parsed, _ := strconv.Atoi(v)
				cid = uint(parsed)
			}
			if cid > 0 {
				todo.ComponentID = &cid
				var comp models.Component
				if err := s.db.Select("name, project_id").First(&comp, cid).Error; err == nil {
					todo.ComponentName = comp.Name
					if todo.ProjectID == nil || *todo.ProjectID == 0 {
						todo.ProjectID = &comp.ProjectID
						var p models.Project
						if err := s.db.Select("name").First(&p, comp.ProjectID).Error; err == nil {
							todo.ProjectName = p.Name
						}
					}
				}
			}
		}

		if parentVal, ok := args["parent_id"]; ok && parentVal != nil {
			var parentID uint
			switch v := parentVal.(type) {
			case float64:
				parentID = uint(v)
			case int:
				parentID = uint(v)
			case string:
				parsed, _ := strconv.Atoi(v)
				parentID = uint(parsed)
			}
			if parentID > 0 {
				todo.ParentID = &parentID
			}
		}

		if level == "" {
			if todo.ComponentID != nil && *todo.ComponentID > 0 {
				level = models.TodoLevelComponent
			} else if todo.ProjectID != nil && *todo.ProjectID > 0 {
				level = models.TodoLevelProject
			} else {
				level = models.TodoLevelGlobal
			}
		}
		todo.Level = level
		todo.ApplyLLMRatings(complexityArg(args), impactArg(args), models.ComplexitySourceManual)

		dups := todorag.Duplicates(s.db, todo)
		if force, _ := args["force"].(bool); !force && len(dups) > 0 && dups[0].Sim >= todorag.BlockSim && todo.Status != models.TodoStatusCompleted {
			return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: "Todo TIDAK dibuat: kemungkinan besar duplikat. " + duplicateNotice(dups) +
				"Bila todo existing masih terbuka, kerjakan/perbarui itu. Bila memang pekerjaan berbeda, panggil create_todo lagi dengan force: true."}}}, nil
		}
		if err := s.db.Create(&todo).Error; err != nil {
			return nil, err
		}
		data, _ := json.MarshalIndent(todo, "", "  ")
		content := []ToolContent{{Type: "text", Text: string(data)}}
		if len(dups) > 0 {
			content = append(content, ToolContent{Type: "text", Text: duplicateNotice(dups)})
		}
		return &ToolCallResult{Content: content}, nil

	case "update_todo":
		var tid uint
		switch v := args["todo_id"].(type) {
		case float64:
			tid = uint(v)
		case int:
			tid = uint(v)
		case string:
			parsed, _ := strconv.Atoi(v)
			tid = uint(parsed)
		}
		if tid == 0 {
			return nil, fmt.Errorf("valid todo_id is required")
		}

		var todo models.Todo
		if err := s.db.First(&todo, tid).Error; err != nil {
			return nil, fmt.Errorf("todo with ID %d not found", tid)
		}

		updates := make(map[string]interface{})
		newStatus, hasStatus := args["status"].(string)
		if hasStatus && newStatus != "" {
			updates["status"] = newStatus
			if newStatus == models.TodoStatusCompleted {
				if compAtStr, ok := args["completed_at"].(string); ok && compAtStr != "" {
					if parsedTime, err := time.Parse(time.RFC3339, compAtStr); err == nil {
						updates["completed_at"] = &parsedTime
					} else {
						now := time.Now()
						updates["completed_at"] = &now
					}
				} else if todo.CompletedAt == nil {
					now := time.Now()
					updates["completed_at"] = &now
				}
			} else if newStatus == models.TodoStatusPending || newStatus == models.TodoStatusInProgress {
				updates["completed_at"] = nil
				updates["completed_by_agent"] = ""
				updates["completed_by_model"] = ""
			}
		}

		if agent, ok := args["completed_by_agent"].(string); ok && agent != "" {
			updates["completed_by_agent"] = agent
		} else if hasStatus && newStatus == models.TodoStatusCompleted && todo.CompletedByAgent == "" {
			updates["completed_by_agent"] = "ai_agent"
		}

		if model, ok := args["completed_by_model"].(string); ok && model != "" {
			updates["completed_by_model"] = model
		}

		if compAtStr, ok := args["completed_at"].(string); ok && compAtStr != "" && updates["completed_at"] == nil {
			if parsedTime, err := time.Parse(time.RFC3339, compAtStr); err == nil {
				updates["completed_at"] = &parsedTime
			}
		}

		if title, ok := args["title"].(string); ok && strings.TrimSpace(title) != "" {
			updates["title"] = strings.TrimSpace(title)
		}
		if cat, ok := args["category"].(string); ok && strings.TrimSpace(cat) != "" {
			updates["category"] = strings.ToLower(strings.TrimSpace(cat))
		}
		if raw, ok := args["tags"]; ok {
			updates["tags"] = models.NormalizeTags(raw)
		}
		if priority, ok := args["priority"].(string); ok && priority != "" {
			updates["priority"] = priority
		}
		if note, ok := args["completion_note"].(string); ok && strings.TrimSpace(note) != "" {
			who := firstNonEmptyStr(args["completed_by_agent"], todo.CompletedByAgent, "agent")
			entry := fmt.Sprintf("Catatan penyelesaian (%s, %s): %s", who, time.Now().Format("2 Jan 2006 15:04"), strings.TrimSpace(note))
			updates["rationale"] = strings.TrimSpace(todo.Rationale + "\n\n" + entry)
		}

		if len(updates) > 0 {
			if err := s.db.Model(&todo).Updates(updates).Error; err != nil {
				return nil, err
			}
			s.db.First(&todo, tid)
			if updates["rationale"] != nil {
				todo.Memory = models.RefreshTodoMemory(s.db, todo.ID)
			}
		}
		if c, i := complexityArg(args), impactArg(args); c != nil || i != nil || updates["category"] != nil {
			todo.ApplyLLMRatings(c, i, models.ComplexitySourceManual)
			todo.Rescore()
			s.db.Model(&todo).Updates(todo.TodoComplexity.Columns())
			s.db.Model(&todo).Updates(todo.TodoImpact.Columns())
		}

		data, _ := json.MarshalIndent(todo, "", "  ")
		content := []ToolContent{{Type: "text", Text: string(data)}}
		// Selesai tanpa pernah ditandai in_progress (agent langsung menutup todo) → tetap dicatat atas nama
		// pegawai agar ikut dinilai QA dan pengetahuannya diperbarui.
		agent := firstNonEmptyStr(args["completed_by_agent"], todo.CompletedByAgent, "agent coding")
		if hasStatus && newStatus == models.TodoStatusCompleted {
			registry.TodoCompleted(s.db, tid, agent)
		}
		// Mulai dikerjakan → modul Pro dapat menugaskan todo ke pegawai vibeasik atau memecahnya per aplikasi.
		if hasStatus && newStatus == models.TodoStatusInProgress {
			if note, stop := registry.TodoStarted(s.db, tid, agent); note != "" || stop {
				if note != "" {
					content = append(content, ToolContent{Type: "text", Text: note})
				}
				if stop {
					return &ToolCallResult{Content: content}, nil
				}
			}
		}
		return &ToolCallResult{Content: content}, nil

	case "list_guidelines":
		categories, err := guidelines.LoadAllGuidelines()
		if err != nil {
			return nil, err
		}
		if catFilter, ok := args["category"].(string); ok && catFilter != "" {
			var filtered []guidelines.GuidelineCategory
			for _, cat := range categories {
				if cat.Category == catFilter {
					filtered = append(filtered, cat)
				}
			}
			data, _ := json.MarshalIndent(filtered, "", "  ")
			return &ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: string(data)}},
			}, nil
		}
		data, _ := json.MarshalIndent(categories, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "suggest_todos":
		var pid uint
		if pidVal, ok := args["project_id"]; ok && pidVal != nil {
			switch v := pidVal.(type) {
			case float64:
				pid = uint(v)
			case int:
				pid = uint(v)
			case string:
				parsed, _ := strconv.Atoi(v)
				pid = uint(parsed)
			}
		}

		saveToDB := true
		if val, ok := args["save_to_db"].(bool); ok {
			saveToDB = val
		}

		var allProjects []models.Project
		s.db.Preload("Components").Preload("Components.Routes").Find(&allProjects)

		var suggestions []models.Todo
		var targetProject *models.Project
		if pid > 0 {
			for i := range allProjects {
				if allProjects[i].ID == pid {
					targetProject = &allProjects[i]
					break
				}
			}
			if targetProject != nil {
				suggestions = append(suggestions, guidelines.GenerateSuggestionsForProject(targetProject, allProjects)...)
				// Standar vibeasik (Aturan 45): port, env & todo pending dipastikan langsung, bukan saran
				if r, _ := standard.Ensure(s.db, targetProject.ID); r != nil {
					suggestions = append(suggestions, r.Todos...)
				}
			}
		} else {
			suggestions = append(suggestions, guidelines.GenerateGlobalSuggestions(allProjects)...)
			for i := range allProjects {
				suggestions = append(suggestions, guidelines.GenerateSuggestionsForProject(&allProjects[i], allProjects)...)
				// Standar vibeasik (Aturan 45): port, env & todo pending dipastikan langsung, bukan saran
				if r, _ := standard.Ensure(s.db, allProjects[i].ID); r != nil {
					suggestions = append(suggestions, r.Todos...)
				}
			}
		}

		// Saran cerdas (rapat Project Manager / LLM) disediakan modul Pro.
		suggestions = append(suggestions, registry.SuggestTodos(s.db, targetProject, allProjects)...)

		finalSuggestions := make([]models.Todo, 0)
		for _, sugg := range suggestions {
			var existing models.Todo
			q := s.db.Where("title = ?", sugg.Title)
			if sugg.ProjectID != nil && *sugg.ProjectID > 0 {
				q = q.Where("project_id = ?", *sugg.ProjectID)
			}
			if sugg.ComponentID != nil && *sugg.ComponentID > 0 {
				q = q.Where("component_id = ?", *sugg.ComponentID)
			}

			if err := q.First(&existing).Error; err != nil {
				if saveToDB {
					s.db.Create(&sugg)
				}
				finalSuggestions = append(finalSuggestions, sugg)
			} else {
				updates := map[string]interface{}{}
				if (existing.Category == "" || existing.Category == models.CategoryGeneral) && sugg.Category != "" {
					existing.Category = sugg.Category
					updates["category"] = sugg.Category
				}
				if existing.TargetApps == "" && sugg.TargetApps != "" {
					existing.TargetApps = sugg.TargetApps
					updates["target_apps"] = sugg.TargetApps
				}
				if len(updates) > 0 {
					s.db.Model(&existing).Updates(updates)
				}
				finalSuggestions = append(finalSuggestions, existing)
			}
		}

		if targetProject == nil {
			data, _ := json.MarshalIndent(finalSuggestions, "", "  ")
			return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: string(data)}}}, nil
		}
		return jsonResult(map[string]interface{}{"guideline_suggestions": finalSuggestions})

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
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
