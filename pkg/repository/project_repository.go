package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// ProjectRepository defines data access methods for Project models
type ProjectRepository interface {
	FindAll(ctx context.Context, projectType string, status string) ([]models.Project, error)
	FindByID(ctx context.Context, id uint) (*models.Project, error)
	FindBySlug(ctx context.Context, slug string) (*models.Project, error)
	Create(ctx context.Context, project *models.Project) error
	Update(ctx context.Context, project *models.Project) error
	Delete(ctx context.Context, id uint) error
	UpdateHealthScore(ctx context.Context, id uint, score int) error
	UpdateFields(ctx context.Context, id uint, updates map[string]interface{}) error
}

type gormProjectRepository struct {
	db *gorm.DB
}

// NewProjectRepository instantiates a new ProjectRepository backed by GORM
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &gormProjectRepository{db: db}
}

func (r *gormProjectRepository) FindAll(ctx context.Context, projectType string, status string) ([]models.Project, error) {
	var projects []models.Project
	query := r.db.WithContext(ctx).Preload("Components").Order("updated_at desc")

	if projectType != "" {
		query = query.Where("project_type = ?", projectType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (r *gormProjectRepository) FindByID(ctx context.Context, id uint) (*models.Project, error) {
	var project models.Project
	if err := r.db.WithContext(ctx).Preload("Components").First(&project, id).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *gormProjectRepository) FindBySlug(ctx context.Context, slug string) (*models.Project, error) {
	var project models.Project
	if err := r.db.WithContext(ctx).Preload("Components").Where("slug = ?", slug).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *gormProjectRepository) Create(ctx context.Context, project *models.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *gormProjectRepository) Update(ctx context.Context, project *models.Project) error {
	return r.db.WithContext(ctx).Save(project).Error
}

func (r *gormProjectRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Project{}, id).Error
}

func (r *gormProjectRepository) UpdateFields(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Project{}).Where("id = ?", id).Updates(updates).Error
}

func (r *gormProjectRepository) UpdateHealthScore(ctx context.Context, id uint, score int) error {
	return r.db.WithContext(ctx).Model(&models.Project{}).Where("id = ?", id).Update("health_score", score).Error
}
