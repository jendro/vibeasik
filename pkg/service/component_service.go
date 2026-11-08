package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
)

// AddComponentInput defines inputs when attaching a component to a project
type AddComponentInput struct {
	models.Component
	CreateDirectory bool `json:"create_directory"`
}

// ComponentService defines business logic for project components
type ComponentService interface {
	GetByID(ctx context.Context, id uint) (*models.Component, error)
	GetByProjectID(ctx context.Context, projectID uint) ([]models.Component, error)
	AddToProject(ctx context.Context, projectID uint, input AddComponentInput) (*models.Component, error)
	Update(ctx context.Context, id uint, updates *models.Component) (*models.Component, error)
	Delete(ctx context.Context, id uint) error
	CreateDir(ctx context.Context, id uint) (string, error)
	UpdateFastScan(ctx context.Context, id uint, fastScanJSON string) error
	ApplyScanParams(ctx context.Context, id uint, params models.DetectedParams) (*models.Component, error)
}

type componentService struct {
	componentRepo repository.ComponentRepository
	projectRepo   repository.ProjectRepository
}

// NewComponentService instantiates a new ComponentService
func NewComponentService(compRepo repository.ComponentRepository, projRepo repository.ProjectRepository) ComponentService {
	return &componentService{
		componentRepo: compRepo,
		projectRepo:   projRepo,
	}
}

func (s *componentService) GetByID(ctx context.Context, id uint) (*models.Component, error) {
	return s.componentRepo.FindByID(ctx, id)
}

func (s *componentService) GetByProjectID(ctx context.Context, projectID uint) ([]models.Component, error) {
	return s.componentRepo.FindByProjectID(ctx, projectID)
}

func (s *componentService) AddToProject(ctx context.Context, projectID uint, input AddComponentInput) (*models.Component, error) {
	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	comp := input.Component
	comp.ProjectID = projectID

	if comp.Name == "" {
		return nil, fmt.Errorf("component name is required")
	}

	if comp.AbsolutePath == "" && proj.RootPath != "" && comp.RelativePath != "" {
		comp.AbsolutePath = filepath.Join(proj.RootPath, comp.RelativePath)
	}

	if input.CreateDirectory && comp.AbsolutePath != "" {
		if err := os.MkdirAll(comp.AbsolutePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	if err := s.componentRepo.Create(ctx, &comp); err != nil {
		return nil, err
	}
	return &comp, nil
}

func (s *componentService) Update(ctx context.Context, id uint, updates *models.Component) (*models.Component, error) {
	existing, err := s.componentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	mergeComponentUpdates(existing, updates)

	if err := s.componentRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *componentService) Delete(ctx context.Context, id uint) error {
	return s.componentRepo.Delete(ctx, id)
}

func (s *componentService) CreateDir(ctx context.Context, id uint) (string, error) {
	comp, err := s.componentRepo.FindByID(ctx, id)
	if err != nil {
		return "", err
	}

	targetPath := comp.AbsolutePath
	if targetPath == "" {
		proj, err := s.projectRepo.FindByID(ctx, comp.ProjectID)
		if err != nil {
			return "", fmt.Errorf("parent project not found: %w", err)
		}
		if proj.RootPath != "" && comp.RelativePath != "" {
			targetPath = filepath.Join(proj.RootPath, comp.RelativePath)
		}
	}

	if targetPath == "" {
		return "", fmt.Errorf("no path specified for component")
	}

	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if comp.AbsolutePath != targetPath {
		comp.AbsolutePath = targetPath
		_ = s.componentRepo.Update(ctx, comp)
	}

	return targetPath, nil
}

func (s *componentService) UpdateFastScan(ctx context.Context, id uint, fastScanJSON string) error {
	return s.componentRepo.UpdateFastScanData(ctx, id, fastScanJSON)
}

func (s *componentService) ApplyScanParams(ctx context.Context, id uint, params models.DetectedParams) (*models.Component, error) {
	updates := make(map[string]interface{})
	if params.Language != "" {
		updates["language"] = params.Language
	}
	if params.Framework != "" {
		updates["framework"] = params.Framework
	}
	if params.PackageManager != "" {
		updates["package_manager"] = params.PackageManager
	}
	if params.DevCommand != "" {
		updates["dev_command"] = params.DevCommand
	}
	if params.BuildCommand != "" {
		updates["build_command"] = params.BuildCommand
	}
	if params.Port > 0 {
		updates["port"] = params.Port
	}

	if len(updates) > 0 {
		if err := s.componentRepo.UpdateFields(ctx, id, updates); err != nil {
			return nil, err
		}
	}

	return s.componentRepo.FindByID(ctx, id)
}

// mergeComponentUpdates menyalin hanya field yang terisi (non-zero) dari updates ke existing.
func mergeComponentUpdates(existing, updates *models.Component) {
	for _, f := range []struct {
		dst *string
		src string
	}{
		{&existing.Name, updates.Name},
		{&existing.Role, updates.Role},
		{&existing.RelativePath, updates.RelativePath},
		{&existing.AbsolutePath, updates.AbsolutePath},
		{&existing.Language, updates.Language},
		{&existing.Framework, updates.Framework},
		{&existing.PackageManager, updates.PackageManager},
		{&existing.DevCommand, updates.DevCommand},
		{&existing.BuildCommand, updates.BuildCommand},
		{&existing.HealthURL, updates.HealthURL},
		{&existing.EnvFile, updates.EnvFile},
		{&existing.EnvSummary, updates.EnvSummary},
	} {
		if f.src != "" {
			*f.dst = f.src
		}
	}
	if updates.Port != 0 {
		existing.Port = updates.Port
	}
}
