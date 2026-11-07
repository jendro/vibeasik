package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/analysis"
	"github.com/vibeasik/vibeasik/pkg/codectx"
	"github.com/vibeasik/vibeasik/pkg/diagnostics"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/repository"
	"github.com/vibeasik/vibeasik/pkg/standard"
)

// ProjectFastScanResponse encapsulates the result of fast-scanning an entire project
type ProjectFastScanResponse struct {
	ProjectID      uint                     `json:"project_id"`
	ProjectName    string                   `json:"project_name"`
	Components     []*models.FastScanResult `json:"components"`
	ProjectSummary string                   `json:"project_summary,omitempty"`
	ScannedAt      time.Time                `json:"scanned_at"`
}

// DiagnosticService defines business logic for diagnostics, scans, and AI reviews
type DiagnosticService interface {
	DiagnoseProject(ctx context.Context, projectID uint) (*models.Diagnostic, error)
	DiagnoseComponent(ctx context.Context, componentID uint) (*models.Diagnostic, error)
	FastScanComponent(ctx context.Context, componentID uint) (*models.FastScanResult, error)
	FastScanProject(ctx context.Context, projectID uint) (*ProjectFastScanResponse, error)
}

type diagnosticService struct {
	diagRepo      repository.DiagnosticRepository
	projectRepo   repository.ProjectRepository
	componentRepo repository.ComponentRepository
	db            *gorm.DB
}

// NewDiagnosticService instantiates a new DiagnosticService
func NewDiagnosticService(diagRepo repository.DiagnosticRepository, projRepo repository.ProjectRepository, compRepo repository.ComponentRepository, db *gorm.DB) DiagnosticService {
	return &diagnosticService{
		diagRepo:      diagRepo,
		projectRepo:   projRepo,
		componentRepo: compRepo,
		db:            db,
	}
}

func (s *diagnosticService) DiagnoseProject(ctx context.Context, projectID uint) (*models.Diagnostic, error) {
	// Standar vibeasik dulu (blok port, env, todo pending Aturan 45) agar audit menilai kondisi terbaru
	if s.db != nil {
		_, _ = standard.Ensure(s.db, projectID)
	}
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	allProjects, err := s.projectRepo.FindAll(ctx, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to load projects: %w", err)
	}

	// 1. Audit each individual component/application one by one!
	for i := range project.Components {
		comp := &project.Components[i]
		compReport, err := diagnostics.AnalyzeComponent(comp, project, allProjects)
		if err == nil && compReport != nil {
			compID := comp.ID
			compReport.ComponentID = &compID
			_ = s.diagRepo.Create(ctx, compReport)
			_ = s.componentRepo.UpdateHealthScore(ctx, comp.ID, compReport.Score)
			comp.HealthScore = compReport.Score
		}
	}

	report, err := diagnostics.AnalyzeProject(project, allProjects)
	if err != nil {
		return nil, fmt.Errorf("diagnostics failed: %w", err)
	}

	var freshFindings []models.Finding
	if report.FindingsJSON != "" {
		_ = json.Unmarshal([]byte(report.FindingsJSON), &freshFindings)
	}
	// Pengecekan fungsi global lintas berkas: fungsi kembar & fungsi privat yang tidak dipakai (codectx)
	freshFindings = append(freshFindings, codectx.GlobalFunctionFindings(s.db, project)...)

	// Load existing findings from latest project diagnostic so we do NOT delete already found issues
	prevDiag, _ := s.diagRepo.FindLatestByProjectID(ctx, projectID)
	var prevFindings []models.Finding
	if prevDiag != nil && prevDiag.FindingsJSON != "" {
		_ = json.Unmarshal([]byte(prevDiag.FindingsJSON), &prevFindings)
	}

	// Merge findings: keep all existing findings, and do not duplicate if already in table
	mergedFindings := models.MergeFindings(prevFindings, freshFindings)
	// Temuan yang pernah divonis audit keliru (pembelajaran audit) langsung diredam
	registry.FilterFindings(s.db, projectID, mergedFindings)

	// Accurately compute health score based ONLY on active findings
	activeCount := 0
	ignoredCount := 0
	resolvedCount := 0
	for _, f := range mergedFindings {
		if f.Status == models.FindingStatusIgnored {
			ignoredCount++
		} else if f.Status == models.FindingStatusResolved {
			resolvedCount++
		} else {
			activeCount++
		}
	}

	report.Score = diagnostics.CalculateHealthScore(mergedFindings)
	report.Summary = fmt.Sprintf("Analyzed %d components across %s architecture. Total %d findings (%d aktif, %d diabaikan, %d diselesaikan). Codebase health score: %d/100.",
		len(project.Components), project.ProjectType, len(mergedFindings), activeCount, ignoredCount, resolvedCount, report.Score)

	if updatedData, err := json.Marshal(mergedFindings); err == nil {
		report.FindingsJSON = string(updatedData)
	}

	if err := s.diagRepo.Create(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	_ = s.projectRepo.UpdateHealthScore(ctx, projectID, report.Score)
	return report, nil
}

func (s *diagnosticService) DiagnoseComponent(ctx context.Context, componentID uint) (*models.Diagnostic, error) {
	comp, err := s.componentRepo.FindByID(ctx, componentID)
	if err != nil {
		return nil, fmt.Errorf("component not found: %w", err)
	}

	proj, err := s.projectRepo.FindByID(ctx, comp.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("parent project not found: %w", err)
	}

	allProjects, _ := s.projectRepo.FindAll(ctx, "", "")

	report, err := diagnostics.AnalyzeComponent(comp, proj, allProjects)
	if err != nil {
		return nil, fmt.Errorf("component diagnosis failed: %w", err)
	}

	compID := comp.ID
	report.ComponentID = &compID

	_ = s.diagRepo.Create(ctx, report)
	_ = s.componentRepo.UpdateHealthScore(ctx, comp.ID, report.Score)
	return report, nil
}

func (s *diagnosticService) FastScanComponent(ctx context.Context, componentID uint) (*models.FastScanResult, error) {
	comp, err := s.componentRepo.FindByID(ctx, componentID)
	if err != nil {
		return nil, fmt.Errorf("component not found: %w", err)
	}

	proj, err := s.projectRepo.FindByID(ctx, comp.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("parent project not found: %w", err)
	}

	result, err := analysis.FastScanComponent(s.db, comp, proj)
	if err != nil {
		return nil, err
	}

	// Persist fast scan data
	dataJSON, _ := json.Marshal(result)
	_ = s.componentRepo.UpdateFastScanData(ctx, comp.ID, string(dataJSON))

	return result, nil
}

func (s *diagnosticService) FastScanProject(ctx context.Context, projectID uint) (*ProjectFastScanResponse, error) {
	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	results := make([]*models.FastScanResult, 0, len(proj.Components))
	for i := range proj.Components {
		comp := &proj.Components[i]
		res, err := analysis.FastScanComponent(s.db, comp, proj)
		if err == nil && res != nil {
			results = append(results, res)
			dataJSON, _ := json.Marshal(res)
			_ = s.componentRepo.UpdateFastScanData(ctx, comp.ID, string(dataJSON))
		}
	}

	summary, _ := registry.ProjectSummary(s.db, projectID)

	return &ProjectFastScanResponse{
		ProjectID:      projectID,
		ProjectName:    proj.Name,
		Components:     results,
		ProjectSummary: summary,
		ScannedAt:      time.Now(),
	}, nil
}
