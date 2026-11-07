package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/cloud"
	"github.com/vibeasik/vibeasik/pkg/diagnostics"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/service"
)

// Extended tools expose the remaining local-engine features (project lifecycle, components,
// routes, diagnostics, scale-up, security, queue, visual audit, organizer) so AI agents reach
// feature parity with the REST API used by the studio frontend.

func prop(typ, desc string) map[string]interface{} {
	return map[string]interface{}{"type": typ, "description": desc}
}

func idSchema(required ...string) ToolSchema {
	props := map[string]interface{}{}
	for _, k := range required {
		props[k] = prop("integer", "ID of the "+strings.TrimSuffix(k, "_id"))
	}
	return ToolSchema{Type: "object", Properties: props, Required: required}
}

func (s *Server) extendedToolsList() []Tool {
	return []Tool{
		// --- Project lifecycle & knowledge ---
		{
			Name:        "create_project",
			Description: "Registers a new project (bundle) in vibeasik, optionally with components and scaffolded directories. Use after scan_directory to persist discovered topology.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"name":               prop("string", "Project name"),
					"root_path":          prop("string", "Absolute root directory of the project"),
					"project_type":       prop("string", "bundle, multi_fe, monorepo, single_app (default: bundle)"),
					"description":        prop("string", "Optional project description"),
					"tags":               prop("string", "Optional comma-separated tags"),
					"create_directories": prop("boolean", "If true, creates root/component directories on disk"),
					"components": map[string]interface{}{
						"type":        "array",
						"description": "Optional components: [{name, role, relative_path, absolute_path, language, framework, port, dev_command, build_command}]",
						"items":       map[string]interface{}{"type": "object"},
					},
				},
				Required: []string{"name"},
			},
		},
		{
			Name:        "update_project_status",
			Description: "Changes a project's lifecycle status: deprecate (with reason), undeprecate, hold, done, or activate.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": prop("integer", "ID of the project"),
					"action":     prop("string", "deprecate, undeprecate, hold, done, activate"),
					"reason":     prop("string", "Reason (used for deprecate)"),
				},
				Required: []string{"project_id", "action"},
			},
		},
		{
			Name:        "scan_project_similarities",
			Description: "Detects duplicate or overlapping projects across the workspace (name, stack, and description similarity) to suggest merges.",
			InputSchema: ToolSchema{Type: "object", Properties: map[string]interface{}{}},
		},
		{
			Name:        "compare_projects",
			Description: "Compares two projects and returns a similarity score with overlapping components and stack.",
			InputSchema: idSchema("project_id_a", "project_id_b"),
		},

		// --- Components & routes ---
		{
			Name:        "get_component",
			Description: "Retrieves a single component (service/module) with its path, stack, port, commands and fast-scan data.",
			InputSchema: idSchema("component_id"),
		},
		{
			Name:        "add_component",
			Description: "Adds a component (backend, frontend_admin, frontend_client, frontend_mobile, worker, etc.) to an existing project.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id":       prop("integer", "ID of the project"),
					"name":             prop("string", "Component name"),
					"role":             prop("string", "backend, frontend_admin, frontend_client, frontend_mobile, worker, other"),
					"relative_path":    prop("string", "Path relative to project root"),
					"absolute_path":    prop("string", "Absolute path (overrides relative_path)"),
					"language":         prop("string", "go, typescript, python, dart, php, ..."),
					"framework":        prop("string", "fiber, nuxt, next, flutter, fastapi, ..."),
					"port":             prop("integer", "Dev server port"),
					"dev_command":      prop("string", "Command to run in development"),
					"build_command":    prop("string", "Command to build"),
					"create_directory": prop("boolean", "Create the directory on disk if missing"),
				},
				Required: []string{"project_id", "name"},
			},
		},
		{
			Name:        "diagnose_component",
			Description: "Runs the health audit on a single component and stores the diagnostic.",
			InputSchema: idSchema("component_id"),
		},
		{
			Name:        "fast_scan",
			Description: "Runs a fast scan (detected stack, ports, env, commands) on a component, or on all components of a project when project_id is given.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"component_id": prop("integer", "ID of the component"),
					"project_id":   prop("integer", "ID of the project (scans every component)"),
				},
			},
		},
		{
			Name:        "list_component_routes",
			Description: "Lists the API/page routes detected for a component, including method, path, domain and description.",
			InputSchema: idSchema("component_id"),
		},
		{
			Name:        "sync_component_routes",
			Description: "Re-detects routes from a component's source code. Set enrich=true to also enrich them (domain, function, description) with the LLM.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"component_id": prop("integer", "ID of the component"),
					"enrich":       prop("boolean", "Enrich routes with the LLM after syncing"),
				},
				Required: []string{"component_id"},
			},
		},

		// --- Diagnostics ---
		{
			Name:        "get_latest_diagnostic",
			Description: "Returns the latest diagnostic report (findings merged with deep check, optimizations, severity counts) for a project or a component. Runs one if none exists.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id":   prop("integer", "ID of the project"),
					"component_id": prop("integer", "ID of the component"),
				},
			},
		},
		{
			Name:        "set_finding_status",
			Description: "Marks a diagnostic finding as ignored, resolved, or open again in the project's latest diagnostic and recalculates the health score.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": prop("integer", "ID of the project"),
					"finding_id": prop("string", "Finding ID/key from get_latest_diagnostic"),
					"status":     prop("string", "ignored, resolved, open"),
				},
				Required: []string{"project_id", "finding_id", "status"},
			},
		},

		// --- Todos ---
		{
			Name:        "delete_todo",
			Description: "Permanently deletes a TODO. Prefer update_todo with status 'rejected' for AI suggestions that should be dismissed.",
			InputSchema: idSchema("todo_id"),
		},
		{
			Name:        "get_todo_impact_dimensions",
			Description: "Reference for rating a todo's impact before create_todo/update_todo: the 7 dimensions (0-5 each) with their meaning, the weight of each dimension per category, the score formula and the high/medium/low/minimal thresholds.",
			InputSchema: ToolSchema{Type: "object", Properties: map[string]interface{}{}},
		},

		// --- Scale-up architecture ---
		{
			Name:        "list_architecture_patterns",
			Description: "Lists the scale-up architecture patterns (layered, clean, hexagonal, MVPP, ...) with pros, cons, folder structure and code samples.",
			InputSchema: ToolSchema{Type: "object", Properties: map[string]interface{}{}},
		},
		{
			Name:        "evaluate_architecture",
			Description: "Evaluates the current architecture of a component (or all components of a project) and recommends a scale-up path.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"component_id": prop("integer", "ID of the component"),
					"project_id":   prop("integer", "ID of the project (evaluates every component)"),
				},
			},
		},

		// --- Security ---
		{
			Name:        "get_security_status",
			Description: "Returns the status of the secret sanitizer that masks credentials before any LLM dispatch, plus the configured censored keywords (values are not returned).",
			InputSchema: ToolSchema{Type: "object", Properties: map[string]interface{}{}},
		},
		{
			Name:        "test_secret_sanitizer",
			Description: "Runs a text payload through the secret sanitizer and returns the masked output with an audit of what was censored.",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]interface{}{"payload": prop("string", "Text to sanitize")},
				Required:   []string{"payload"},
			},
		},
		{
			Name:        "vibeasik_cloud_security_audit",
			Description: "Sends code snippets to the vibeasik.com cloud pre-flight security gate and returns issues found. Only send env KEY names, never values.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_name": prop("string", "Project name"),
					"language":     prop("string", "go, typescript, python, ..."),
					"framework":    prop("string", "fiber, nuxt, fastapi, ..."),
					"files_sample": map[string]interface{}{
						"type":        "object",
						"description": "Map of filename -> code snippet",
					},
					"env_keys": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Env variable names only (no values)",
					},
				},
				Required: []string{"project_name"},
			},
		},

		// --- Background queue ---

		// --- Visual audit ---

		// --- Workspace organizer ---
		{
			Name:        "plan_workspace_reorg",
			Description: "Generates a read-only plan to reorganize project folders into a tidy workspace structure. Does not move anything.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"target_directory": prop("string", "Target workspace root directory"),
					"strategy":         prop("string", "Organization strategy (e.g. by_project, by_stack)"),
					"project_ids": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "integer"},
						"description": "Optional subset of project IDs",
					},
				},
				Required: []string{"target_directory"},
			},
		},
		{
			Name:        "preview_consolidate_project",
			Description: "Previews consolidating a project's scattered components into one parent directory (read-only): planned moves, components sharing a folder, recovery of components already moved, warnings, and where the code map will live.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id":        prop("integer", "ID of the project"),
					"target_parent_dir": prop("string", "Optional target parent directory"),
				},
				Required: []string{"project_id"},
			},
		},

		{
			Name: "consolidate_project",
			Description: "Satukan Folder: MOVES a project's scattered component folders into one parent directory, updates their paths, rewrites .vibeasik.json and keeps vibeasik artifacts (.vibeasik/ code map) at the parent root. " +
				"Moves files on disk, so call preview_consolidate_project first, show the plan to the user, and pass confirm=true only after they agree.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id":        prop("integer", "ID of the project"),
					"target_parent_dir": prop("string", "Optional target parent directory (default from the preview)"),
					"confirm":           prop("boolean", "Must be true: the user approved the previewed moves"),
				},
				Required: []string{"project_id", "confirm"},
			},
		},

		// --- System ---
		{
			Name:        "browse_directories",
			Description: "Lists sub-directories of a local path (with optional name search) to help locate projects on disk.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"path":  prop("string", "Directory to browse (default: home)"),
					"query": prop("string", "Optional name filter"),
				},
			},
		},
	}
}

func argUint(args map[string]interface{}, key string) uint {
	switch v := args[key].(type) {
	case float64:
		return uint(v)
	case int:
		return uint(v)
	case string:
		parsed, _ := strconv.Atoi(v)
		return uint(parsed)
	}
	return 0
}

// argUints membaca array angka (JSON) atau CSV "1,2".
func argUints(args map[string]interface{}, key string) []uint {
	var out []uint
	switch v := args[key].(type) {
	case []interface{}:
		for _, x := range v {
			if n := argUint(map[string]interface{}{"v": x}, "v"); n > 0 {
				out = append(out, n)
			}
		}
	case string:
		for _, part := range strings.Split(v, ",") {
			if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && n > 0 {
				out = append(out, uint(n))
			}
		}
	}
	return out
}

func argString(args map[string]interface{}, key string) string {
	v, _ := args[key].(string)
	return strings.TrimSpace(v)
}

func jsonResult(v interface{}) (*ToolCallResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: string(data)}}}, nil
}

// decodeArgs maps loosely-typed tool arguments onto a struct via its json tags.
func decodeArgs(src interface{}, dst interface{}) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// callExtendedTool returns handled=false when the tool name is not an extended tool.
func (s *Server) callExtendedTool(name string, args map[string]interface{}) (*ToolCallResult, bool, error) {
	ctx := context.Background()
	svc := s.services

	switch name {
	case "create_project":
		var input service.CreateProjectInput
		if err := decodeArgs(args, &input); err != nil {
			return nil, true, fmt.Errorf("invalid project payload: %w", err)
		}
		p, err := svc.Projects.Create(ctx, input)
		if err != nil {
			return nil, true, err
		}
		registry.ProjectCreated(s.db, p)
		res, err := jsonResult(p)
		return res, true, err

	case "update_project_status":
		pid := argUint(args, "project_id")
		if pid == 0 {
			return nil, true, fmt.Errorf("valid project_id is required")
		}
		var p *models.Project
		var err error
		switch argString(args, "action") {
		case "deprecate":
			p, err = svc.Projects.MarkDeprecated(ctx, pid, argString(args, "reason"))
		case "undeprecate":
			p, err = svc.Projects.UnmarkDeprecated(ctx, pid)
		case "hold":
			p, err = svc.Projects.MarkHold(ctx, pid)
		case "done":
			p, err = svc.Projects.MarkDone(ctx, pid)
		case "activate":
			p, err = svc.Projects.ActivateProject(ctx, pid)
		default:
			return nil, true, fmt.Errorf("action must be one of: deprecate, undeprecate, hold, done, activate")
		}
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(p)
		return res, true, err

	case "scan_project_similarities":
		r, err := svc.Similarity.ScanSimilarities(ctx)
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(r)
		return res, true, err

	case "compare_projects":
		r, err := svc.Similarity.CompareTwoProjects(ctx, argUint(args, "project_id_a"), argUint(args, "project_id_b"))
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(r)
		return res, true, err

	case "get_component":
		comp, err := svc.Components.GetByID(ctx, argUint(args, "component_id"))
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(comp)
		return res, true, err

	case "add_component":
		var input service.AddComponentInput
		if err := decodeArgs(args, &input); err != nil {
			return nil, true, fmt.Errorf("invalid component payload: %w", err)
		}
		comp, err := svc.Components.AddToProject(ctx, argUint(args, "project_id"), input)
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(comp)
		return res, true, err

	case "diagnose_component":
		diag, err := svc.Diagnostics.DiagnoseComponent(ctx, argUint(args, "component_id"))
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(diag)
		return res, true, err

	case "fast_scan":
		var r interface{}
		var err error
		if cid := argUint(args, "component_id"); cid > 0 {
			r, err = svc.Diagnostics.FastScanComponent(ctx, cid)
		} else if pid := argUint(args, "project_id"); pid > 0 {
			r, err = svc.Diagnostics.FastScanProject(ctx, pid)
		} else {
			return nil, true, fmt.Errorf("component_id or project_id is required")
		}
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(r)
		return res, true, err

	case "list_component_routes":
		routes, err := svc.Routes.GetRoutes(ctx, argUint(args, "component_id"))
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(routes)
		return res, true, err

	case "sync_component_routes":
		cid := argUint(args, "component_id")
		syncRes, err := svc.Routes.SyncRoutes(ctx, cid)
		if err != nil {
			return nil, true, err
		}
		out := map[string]interface{}{"sync": syncRes}
		if enrich, _ := args["enrich"].(bool); enrich {
			out["enrich"] = registry.UpgradeRequired("routes.enrich").Error()
		}
		res, err := jsonResult(out)
		return res, true, err

	case "get_latest_diagnostic":
		res, err := s.latestDiagnostic(ctx, argUint(args, "project_id"), argUint(args, "component_id"))
		return res, true, err

	case "set_finding_status":
		res, err := s.setFindingStatus(argUint(args, "project_id"), argString(args, "finding_id"), argString(args, "status"))
		return res, true, err

	case "get_todo_impact_dimensions":
		res, err := jsonResult(models.ImpactReference())
		return res, true, err

	case "delete_todo":
		tid := argUint(args, "todo_id")
		if tid == 0 {
			return nil, true, fmt.Errorf("valid todo_id is required")
		}
		if os.Getenv(envOrchestratorRunID) != "" {
			return nil, true, fmt.Errorf("agent yang dijalankan orkestrator vibeasik tidak boleh menghapus todo; status todo diatur otomatis oleh orkestrator")
		}
		deleted, err := models.DeleteTodo(s.db, tid)
		if err != nil {
			return nil, true, err
		}
		if deleted == 0 {
			return nil, true, fmt.Errorf("todo with ID %d not found", tid)
		}
		res, err := jsonResult(map[string]interface{}{"status": "deleted", "id": tid})
		return res, true, err

	case "list_architecture_patterns":
		res, err := jsonResult(svc.ScaleUp.GetPatterns())
		return res, true, err

	case "evaluate_architecture":
		var r interface{}
		var err error
		if cid := argUint(args, "component_id"); cid > 0 {
			r, err = svc.ScaleUp.EvaluateComponent(ctx, cid)
		} else if pid := argUint(args, "project_id"); pid > 0 {
			r, err = svc.ScaleUp.EvaluateProject(ctx, pid)
		} else {
			return nil, true, fmt.Errorf("component_id or project_id is required")
		}
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(r)
		return res, true, err

	case "get_security_status":
		status, err := svc.Security.GetStatus(ctx)
		if err != nil {
			return nil, true, err
		}
		keywords, err := svc.Security.GetKeywords(ctx)
		if err != nil {
			return nil, true, err
		}
		// Only expose keyword metadata: the keyword itself may be a secret.
		kwMeta := make([]map[string]interface{}, 0, len(keywords))
		for _, kw := range keywords {
			kwMeta = append(kwMeta, map[string]interface{}{"id": kw.ID, "category": kw.Category, "is_active": kw.IsActive, "is_regex": kw.IsRegex})
		}
		status["keywords"] = kwMeta
		res, err := jsonResult(status)
		return res, true, err

	case "test_secret_sanitizer":
		out, err := svc.Security.TestCensor(ctx, argString(args, "payload"))
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(out)
		return res, true, err

	case "vibeasik_cloud_security_audit":
		r, err := cloud.NewClient().AuditProject(args)
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(r)
		return res, true, err

	case "plan_workspace_reorg":
		var req models.ReorgPlanRequest
		if err := decodeArgs(args, &req); err != nil {
			return nil, true, err
		}
		plan, err := svc.Organizer.GenerateReorgPlan(ctx, req)
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(plan)
		return res, true, err

	case "preview_consolidate_project":
		preview, err := svc.Organizer.PreviewConsolidate(ctx, argUint(args, "project_id"), argString(args, "target_parent_dir"))
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(preview)
		return res, true, err

	case "consolidate_project":
		if confirm, _ := args["confirm"].(bool); !confirm {
			return nil, true, fmt.Errorf("consolidate_project memindahkan folder: tampilkan preview_consolidate_project ke pengguna lalu panggil lagi dengan confirm=true")
		}
		result, err := svc.Organizer.Consolidate(ctx, argUint(args, "project_id"), argString(args, "target_parent_dir"), "http://127.0.0.1:8765")
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(result)
		return res, true, err

	case "browse_directories":
		r, err := svc.System.BrowseDirectories(ctx, argString(args, "path"), argString(args, "query"))
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(r)
		return res, true, err
	}

	return nil, false, nil
}

func (s *Server) latestDiagnostic(ctx context.Context, pid, cid uint) (*ToolCallResult, error) {
	var diag models.Diagnostic
	switch {
	case cid > 0:
		if err := s.db.Where("component_id = ?", cid).Order("created_at DESC").First(&diag).Error; err != nil {
			report, err := s.services.Diagnostics.DiagnoseComponent(ctx, cid)
			if err != nil {
				return nil, err
			}
			diag = *report
		}
	case pid > 0:
		if err := s.db.Where("project_id = ? AND (component_id IS NULL OR component_id = 0)", pid).Order("created_at DESC").First(&diag).Error; err != nil {
			report, err := s.services.Diagnostics.DiagnoseProject(ctx, pid)
			if err != nil {
				return nil, err
			}
			diag = *report
		}
	default:
		return nil, fmt.Errorf("project_id or component_id is required")
	}

	var findings []models.Finding
	_ = json.Unmarshal([]byte(diag.FindingsJSON), &findings)
	if cid == 0 {
		var deep models.DeepCheckReport
		if err := s.db.Where("project_id = ?", pid).Order("created_at desc").First(&deep).Error; err == nil && deep.FindingsJSON != "" {
			var deepFindings []models.Finding
			if json.Unmarshal([]byte(deep.FindingsJSON), &deepFindings) == nil {
				findings = models.MergeFindings(findings, deepFindings)
			}
		}
	}
	findings = models.DeduplicateFindings(findings)
	for i := range findings {
		if findings[i].ID == "" {
			findings[i].ID = models.GenerateFindingKey(findings[i])
		}
	}

	var optimizations []models.Optimization
	_ = json.Unmarshal([]byte(diag.Optimizations), &optimizations)

	severity := map[string]int{}
	active := 0
	for _, f := range findings {
		if f.Status == models.FindingStatusIgnored || f.Status == models.FindingStatusResolved {
			continue
		}
		active++
		severity[strings.ToUpper(f.Severity)]++
	}

	return jsonResult(map[string]interface{}{
		"diagnostic_id":         diag.ID,
		"score":                 diag.Score,
		"summary":               diag.Summary,
		"created_at":            diag.CreatedAt,
		"findings":              findings,
		"optimizations":         optimizations,
		"total_findings":        len(findings),
		"active_findings_count": active,
		"severity_counts":       severity,
	})
}

func (s *Server) setFindingStatus(pid uint, findingID, status string) (*ToolCallResult, error) {
	if pid == 0 || findingID == "" {
		return nil, fmt.Errorf("project_id and finding_id are required")
	}
	if status != models.FindingStatusIgnored && status != models.FindingStatusResolved && status != models.FindingStatusOpen {
		return nil, fmt.Errorf("status must be one of: ignored, resolved, open")
	}

	var diag models.Diagnostic
	if err := s.db.Where("project_id = ?", pid).Order("created_at DESC").First(&diag).Error; err != nil {
		return nil, fmt.Errorf("no diagnostic found for project %d", pid)
	}
	var findings []models.Finding
	if err := json.Unmarshal([]byte(diag.FindingsJSON), &findings); err != nil {
		return nil, fmt.Errorf("failed to parse findings: %w", err)
	}
	findings = models.DeduplicateFindings(findings)

	found := false
	for i := range findings {
		if findings[i].ID == findingID || models.GenerateFindingKey(findings[i]) == findingID {
			findings[i].Status = status
			if findings[i].ID == "" {
				findings[i].ID = models.GenerateFindingKey(findings[i])
			}
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("finding %s not found in latest diagnostic", findingID)
	}

	diag.Score = diagnostics.CalculateHealthScore(findings)
	data, err := json.Marshal(findings)
	if err != nil {
		return nil, err
	}
	if err := s.db.Model(&diag).Updates(map[string]interface{}{"findings_json": string(data), "score": diag.Score}).Error; err != nil {
		return nil, err
	}
	s.db.Model(&models.Project{}).Where("id = ?", pid).Update("health_score", diag.Score)

	return jsonResult(map[string]interface{}{"finding_id": findingID, "status": status, "score": diag.Score})
}
