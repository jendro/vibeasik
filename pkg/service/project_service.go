package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
)

// Slugify generates a clean URL-friendly slug from a string
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteRune('-')
		}
	}
	res := b.String()
	for strings.Contains(res, "--") {
		res = strings.ReplaceAll(res, "--", "-")
	}
	res = strings.Trim(res, "-")
	if res == "" {
		res = fmt.Sprintf("proj-%d", time.Now().Unix())
	}
	return res
}

// CreateProjectInput holds data for creating a new project
type CreateProjectInput struct {
	models.Project
	CreateDirectories bool `json:"create_directories"`
}

// ProjectService defines business logic for software projects
type ProjectService interface {
	GetAll(ctx context.Context, projectType, status string) ([]models.Project, error)
	GetByID(ctx context.Context, id uint) (*models.Project, error)
	GetBySlug(ctx context.Context, slug string) (*models.Project, error)
	Create(ctx context.Context, input CreateProjectInput) (*models.Project, error)
	Update(ctx context.Context, id uint, updates *models.Project) (*models.Project, error)
	Delete(ctx context.Context, id uint) error
	MarkDeprecated(ctx context.Context, id uint, reason string) (*models.Project, error)
	UnmarkDeprecated(ctx context.Context, id uint) (*models.Project, error)
	MarkHold(ctx context.Context, id uint) (*models.Project, error)
	MarkDone(ctx context.Context, id uint) (*models.Project, error)
	ActivateProject(ctx context.Context, id uint) (*models.Project, error)
}

type projectService struct {
	projectRepo repository.ProjectRepository
}

// NewProjectService instantiates a new ProjectService
func NewProjectService(projectRepo repository.ProjectRepository) ProjectService {
	return &projectService{projectRepo: projectRepo}
}

func (s *projectService) GetAll(ctx context.Context, projectType, status string) ([]models.Project, error) {
	return s.projectRepo.FindAll(ctx, projectType, status)
}

func (s *projectService) GetByID(ctx context.Context, id uint) (*models.Project, error) {
	return s.projectRepo.FindByID(ctx, id)
}

func (s *projectService) GetBySlug(ctx context.Context, slug string) (*models.Project, error) {
	return s.projectRepo.FindBySlug(ctx, slug)
}

func (s *projectService) Create(ctx context.Context, input CreateProjectInput) (*models.Project, error) {
	p := input.Project
	if p.Name == "" {
		return nil, fmt.Errorf("project name is required")
	}
	if p.Slug == "" {
		p.Slug = Slugify(p.Name)
	}

	// Verify slug uniqueness
	if existing, _ := s.projectRepo.FindBySlug(ctx, p.Slug); existing != nil {
		p.Slug = fmt.Sprintf("%s-%d", p.Slug, time.Now().Unix())
	}

	if p.Status == "" {
		p.Status = "active"
	}
	if p.ProjectType == "" {
		p.ProjectType = models.TypeBundle
	}
	p.HealthScore = 100

	// Scaffold directory structure on disk if requested
	if input.CreateDirectories && p.RootPath != "" {
		_ = os.MkdirAll(p.RootPath, 0755)

		for i := range p.Components {
			c := &p.Components[i]
			targetPath := c.AbsolutePath
			if targetPath == "" && c.RelativePath != "" {
				targetPath = filepath.Join(p.RootPath, c.RelativePath)
				c.AbsolutePath = targetPath
			}
			if targetPath != "" {
				_ = os.MkdirAll(targetPath, 0755)
			}
		}
	}

	if err := s.projectRepo.Create(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *projectService) Update(ctx context.Context, id uint, updates *models.Project) (*models.Project, error) {
	existing, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.ShortDescription != "" {
		existing.ShortDescription = updates.ShortDescription
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.RootPath != "" {
		existing.RootPath = updates.RootPath
	}
	if updates.ProjectType != "" {
		existing.ProjectType = updates.ProjectType
	}
	if updates.Status != "" {
		existing.Status = updates.Status
	}
	if updates.GitBranch != "" {
		existing.GitBranch = updates.GitBranch
	}
	if updates.GitRemote != "" {
		existing.GitRemote = updates.GitRemote
	}
	if updates.Tags != "" {
		existing.Tags = updates.Tags
	}
	if updates.DeprecatedReason != "" {
		existing.DeprecatedReason = updates.DeprecatedReason
	}
	if updates.DeprecatedAt != nil {
		existing.DeprecatedAt = updates.DeprecatedAt
	}

	if err := s.projectRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *projectService) Delete(ctx context.Context, id uint) error {
	return s.projectRepo.Delete(ctx, id)
}

func (s *projectService) MarkDeprecated(ctx context.Context, id uint, reason string) (*models.Project, error) {
	existing, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	existing.Status = models.StatusDeprecated
	existing.DeprecatedAt = &now
	existing.DeprecatedReason = reason

	if err := s.projectRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *projectService) UnmarkDeprecated(ctx context.Context, id uint) (*models.Project, error) {
	return s.ActivateProject(ctx, id)
}

func (s *projectService) MarkHold(ctx context.Context, id uint) (*models.Project, error) {
	existing, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Status = models.StatusHold

	if err := s.projectRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *projectService) MarkDone(ctx context.Context, id uint) (*models.Project, error) {
	existing, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Status = models.StatusDone

	if err := s.projectRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *projectService) ActivateProject(ctx context.Context, id uint) (*models.Project, error) {
	existing, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Status = models.StatusActive
	existing.DeprecatedAt = nil
	existing.DeprecatedReason = ""

	if err := s.projectRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}
