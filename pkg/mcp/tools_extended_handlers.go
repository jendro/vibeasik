package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/vibeasik/vibeasik/pkg/cloud"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/service"
)

// extendedToolHandlers memetakan tool berbasis service layer ke penanganannya.
var extendedToolHandlers = map[string]toolHandler{
	"create_project":                (*Server).toolCreateProject,
	"update_project_status":         (*Server).toolUpdateProjectStatus,
	"scan_project_similarities":     (*Server).toolScanSimilarities,
	"compare_projects":              (*Server).toolCompareProjects,
	"get_component":                 (*Server).toolGetComponent,
	"add_component":                 (*Server).toolAddComponent,
	"diagnose_component":            (*Server).toolDiagnoseComponent,
	"fast_scan":                     (*Server).toolFastScan,
	"list_component_routes":         (*Server).toolListComponentRoutes,
	"sync_component_routes":         (*Server).toolSyncComponentRoutes,
	"get_latest_diagnostic":         (*Server).toolLatestDiagnostic,
	"set_finding_status":            (*Server).toolSetFindingStatus,
	"get_todo_impact_dimensions":    (*Server).toolImpactDimensions,
	"delete_todo":                   (*Server).toolDeleteTodo,
	"list_architecture_patterns":    (*Server).toolArchitecturePatterns,
	"evaluate_architecture":         (*Server).toolEvaluateArchitecture,
	"get_security_status":           (*Server).toolSecurityStatus,
	"test_secret_sanitizer":         (*Server).toolSecretSanitizer,
	"vibeasik_cloud_security_audit": (*Server).toolCloudSecurityAudit,
	"vibeasik_security_test":        (*Server).toolSecurityTest,
	"plan_workspace_reorg":          (*Server).toolPlanReorg,
	"preview_consolidate_project":   (*Server).toolPreviewConsolidate,
	"consolidate_project":           (*Server).toolConsolidateProject,
	"browse_directories":            (*Server).toolBrowseDirectories,
}

// projectStatusActions memetakan action update_project_status ke operasi ProjectService.
var projectStatusActions = map[string]func(ctx context.Context, p service.ProjectService, pid uint, reason string) (*models.Project, error){
	"deprecate": func(ctx context.Context, p service.ProjectService, pid uint, reason string) (*models.Project, error) {
		return p.MarkDeprecated(ctx, pid, reason)
	},
	"undeprecate": func(ctx context.Context, p service.ProjectService, pid uint, _ string) (*models.Project, error) {
		return p.UnmarkDeprecated(ctx, pid)
	},
	"hold": func(ctx context.Context, p service.ProjectService, pid uint, _ string) (*models.Project, error) {
		return p.MarkHold(ctx, pid)
	},
	"done": func(ctx context.Context, p service.ProjectService, pid uint, _ string) (*models.Project, error) {
		return p.MarkDone(ctx, pid)
	},
	"activate": func(ctx context.Context, p service.ProjectService, pid uint, _ string) (*models.Project, error) {
		return p.ActivateProject(ctx, pid)
	},
}

func (s *Server) toolCreateProject(args map[string]interface{}) (*ToolCallResult, error) {
	var input service.CreateProjectInput
	if err := decodeArgs(args, &input); err != nil {
		return nil, fmt.Errorf("invalid project payload: %w", err)
	}
	p, err := s.services.Projects.Create(context.Background(), input)
	if err != nil {
		return nil, err
	}
	registry.ProjectCreated(s.db, p)
	return jsonResult(p)
}

func (s *Server) toolUpdateProjectStatus(args map[string]interface{}) (*ToolCallResult, error) {
	pid := argUint(args, "project_id")
	if pid == 0 {
		return nil, fmt.Errorf("valid project_id is required")
	}
	action, ok := projectStatusActions[argString(args, "action")]
	if !ok {
		return nil, fmt.Errorf("action must be one of: deprecate, undeprecate, hold, done, activate")
	}
	p, err := action(context.Background(), s.services.Projects, pid, argString(args, "reason"))
	if err != nil {
		return nil, err
	}
	return jsonResult(p)
}

func (s *Server) toolScanSimilarities(args map[string]interface{}) (*ToolCallResult, error) {
	r, err := s.services.Similarity.ScanSimilarities(context.Background())
	if err != nil {
		return nil, err
	}
	return jsonResult(r)
}

func (s *Server) toolCompareProjects(args map[string]interface{}) (*ToolCallResult, error) {
	r, err := s.services.Similarity.CompareTwoProjects(context.Background(), argUint(args, "project_id_a"), argUint(args, "project_id_b"))
	if err != nil {
		return nil, err
	}
	return jsonResult(r)
}

func (s *Server) toolGetComponent(args map[string]interface{}) (*ToolCallResult, error) {
	comp, err := s.services.Components.GetByID(context.Background(), argUint(args, "component_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(comp)
}

func (s *Server) toolAddComponent(args map[string]interface{}) (*ToolCallResult, error) {
	var input service.AddComponentInput
	if err := decodeArgs(args, &input); err != nil {
		return nil, fmt.Errorf("invalid component payload: %w", err)
	}
	comp, err := s.services.Components.AddToProject(context.Background(), argUint(args, "project_id"), input)
	if err != nil {
		return nil, err
	}
	return jsonResult(comp)
}

func (s *Server) toolDiagnoseComponent(args map[string]interface{}) (*ToolCallResult, error) {
	diag, err := s.services.Diagnostics.DiagnoseComponent(context.Background(), argUint(args, "component_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(diag)
}

// byComponentOrProject menjalankan fnComp bila component_id ada, jika tidak fnProj untuk project_id.
func byComponentOrProject(args map[string]interface{}, fnComp, fnProj func(id uint) (interface{}, error)) (*ToolCallResult, error) {
	var r interface{}
	var err error
	if cid := argUint(args, "component_id"); cid > 0 {
		r, err = fnComp(cid)
	} else if pid := argUint(args, "project_id"); pid > 0 {
		r, err = fnProj(pid)
	} else {
		return nil, fmt.Errorf("component_id or project_id is required")
	}
	if err != nil {
		return nil, err
	}
	return jsonResult(r)
}

func (s *Server) toolFastScan(args map[string]interface{}) (*ToolCallResult, error) {
	ctx := context.Background()
	return byComponentOrProject(args,
		func(id uint) (interface{}, error) { return s.services.Diagnostics.FastScanComponent(ctx, id) },
		func(id uint) (interface{}, error) { return s.services.Diagnostics.FastScanProject(ctx, id) })
}

func (s *Server) toolEvaluateArchitecture(args map[string]interface{}) (*ToolCallResult, error) {
	ctx := context.Background()
	return byComponentOrProject(args,
		func(id uint) (interface{}, error) { return s.services.ScaleUp.EvaluateComponent(ctx, id) },
		func(id uint) (interface{}, error) { return s.services.ScaleUp.EvaluateProject(ctx, id) })
}

func (s *Server) toolListComponentRoutes(args map[string]interface{}) (*ToolCallResult, error) {
	routes, err := s.services.Routes.GetRoutes(context.Background(), argUint(args, "component_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(routes)
}

func (s *Server) toolSyncComponentRoutes(args map[string]interface{}) (*ToolCallResult, error) {
	syncRes, err := s.services.Routes.SyncRoutes(context.Background(), argUint(args, "component_id"))
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{"sync": syncRes}
	if enrich, _ := args["enrich"].(bool); enrich {
		out["enrich"] = registry.UpgradeRequired("routes.enrich").Error()
	}
	return jsonResult(out)
}

func (s *Server) toolLatestDiagnostic(args map[string]interface{}) (*ToolCallResult, error) {
	return s.latestDiagnostic(context.Background(), argUint(args, "project_id"), argUint(args, "component_id"))
}

func (s *Server) toolSetFindingStatus(args map[string]interface{}) (*ToolCallResult, error) {
	return s.setFindingStatus(argUint(args, "project_id"), argString(args, "finding_id"), argString(args, "status"))
}

func (s *Server) toolImpactDimensions(args map[string]interface{}) (*ToolCallResult, error) {
	return jsonResult(models.ImpactReference())
}

func (s *Server) toolDeleteTodo(args map[string]interface{}) (*ToolCallResult, error) {
	tid := argUint(args, "todo_id")
	if tid == 0 {
		return nil, fmt.Errorf("valid todo_id is required")
	}
	if os.Getenv(envOrchestratorRunID) != "" {
		return nil, fmt.Errorf("agent yang dijalankan orkestrator vibeasik tidak boleh menghapus todo; status todo diatur otomatis oleh orkestrator")
	}
	deleted, err := models.DeleteTodo(s.db, tid)
	if err != nil {
		return nil, err
	}
	if deleted == 0 {
		return nil, fmt.Errorf("todo with ID %d not found", tid)
	}
	return jsonResult(map[string]interface{}{"status": "deleted", "id": tid})
}

func (s *Server) toolArchitecturePatterns(args map[string]interface{}) (*ToolCallResult, error) {
	return jsonResult(s.services.ScaleUp.GetPatterns())
}

func (s *Server) toolSecurityStatus(args map[string]interface{}) (*ToolCallResult, error) {
	ctx := context.Background()
	status, err := s.services.Security.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	keywords, err := s.services.Security.GetKeywords(ctx)
	if err != nil {
		return nil, err
	}
	// Only expose keyword metadata: the keyword itself may be a secret.
	kwMeta := make([]map[string]interface{}, 0, len(keywords))
	for _, kw := range keywords {
		kwMeta = append(kwMeta, map[string]interface{}{"id": kw.ID, "category": kw.Category, "is_active": kw.IsActive, "is_regex": kw.IsRegex})
	}
	status["keywords"] = kwMeta
	return jsonResult(status)
}

func (s *Server) toolSecretSanitizer(args map[string]interface{}) (*ToolCallResult, error) {
	out, err := s.services.Security.TestCensor(context.Background(), argString(args, "payload"))
	if err != nil {
		return nil, err
	}
	return jsonResult(out)
}

func (s *Server) toolCloudSecurityAudit(args map[string]interface{}) (*ToolCallResult, error) {
	r, err := cloud.NewClient().AuditProject(args)
	if err != nil {
		return nil, err
	}
	return jsonResult(r)
}

func (s *Server) toolSecurityTest(args map[string]interface{}) (*ToolCallResult, error) {
	req := service.SecurityTestRequest{
		ProjectID:   argUint(args, "project_id"),
		ComponentID: argUint(args, "component_id"),
		BaseURL:     argString(args, "base_url"),
		Burst:       int(argUint(args, "burst")),
	}
	if v, ok := args["live"].(bool); ok {
		req.Live = &v
	}
	req.SkipBurst, _ = args["skip_burst"].(bool)
	req.StartServer, _ = args["start_server"].(bool)
	if req.Burst > 60 {
		req.Burst = 60
	}
	rep, err := s.services.SecTest.Run(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return jsonResult(rep)
}

func (s *Server) toolPlanReorg(args map[string]interface{}) (*ToolCallResult, error) {
	var req models.ReorgPlanRequest
	if err := decodeArgs(args, &req); err != nil {
		return nil, err
	}
	plan, err := s.services.Organizer.GenerateReorgPlan(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return jsonResult(plan)
}

func (s *Server) toolPreviewConsolidate(args map[string]interface{}) (*ToolCallResult, error) {
	preview, err := s.services.Organizer.PreviewConsolidate(context.Background(), argUint(args, "project_id"), argString(args, "target_parent_dir"))
	if err != nil {
		return nil, err
	}
	return jsonResult(preview)
}

func (s *Server) toolConsolidateProject(args map[string]interface{}) (*ToolCallResult, error) {
	if confirm, _ := args["confirm"].(bool); !confirm {
		return nil, fmt.Errorf("consolidate_project memindahkan folder: tampilkan preview_consolidate_project ke pengguna lalu panggil lagi dengan confirm=true")
	}
	result, err := s.services.Organizer.Consolidate(context.Background(), argUint(args, "project_id"), argString(args, "target_parent_dir"), localEngineURL)
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (s *Server) toolBrowseDirectories(args map[string]interface{}) (*ToolCallResult, error) {
	r, err := s.services.System.BrowseDirectories(context.Background(), argString(args, "path"), argString(args, "query"))
	if err != nil {
		return nil, err
	}
	return jsonResult(r)
}

// callExtendedTool returns handled=false when the tool name is not an extended tool.
func (s *Server) callExtendedTool(name string, args map[string]interface{}) (*ToolCallResult, bool, error) {
	h, ok := extendedToolHandlers[name]
	if !ok {
		return nil, false, nil
	}
	res, err := h(s, args)
	return res, true, err
}
