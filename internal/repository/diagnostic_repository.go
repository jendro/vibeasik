package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/internal/models"
)

// DiagnosticRepository defines data access methods for Diagnostic models
type DiagnosticRepository interface {
	Create(ctx context.Context, diag *models.Diagnostic) error
	FindByProjectID(ctx context.Context, projectID uint) ([]models.Diagnostic, error)
	FindLatestByProjectID(ctx context.Context, projectID uint) (*models.Diagnostic, error)
	FindByComponentID(ctx context.Context, componentID uint) ([]models.Diagnostic, error)
	FindLatestByComponentID(ctx context.Context, componentID uint) (*models.Diagnostic, error)
}

type gormDiagnosticRepository struct {
	db *gorm.DB
}

// NewDiagnosticRepository instantiates a new DiagnosticRepository backed by GORM
func NewDiagnosticRepository(db *gorm.DB) DiagnosticRepository {
	return &gormDiagnosticRepository{db: db}
}

func (r *gormDiagnosticRepository) Create(ctx context.Context, diag *models.Diagnostic) error {
	return r.db.WithContext(ctx).Create(diag).Error
}

func (r *gormDiagnosticRepository) FindByProjectID(ctx context.Context, projectID uint) ([]models.Diagnostic, error) {
	var diags []models.Diagnostic
	if err := r.db.WithContext(ctx).Where("project_id = ? AND (component_id IS NULL OR component_id = 0)", projectID).Order("created_at desc").Find(&diags).Error; err != nil {
		return nil, err
	}
	return diags, nil
}

func (r *gormDiagnosticRepository) FindLatestByProjectID(ctx context.Context, projectID uint) (*models.Diagnostic, error) {
	var diag models.Diagnostic
	if err := r.db.WithContext(ctx).Where("project_id = ? AND (component_id IS NULL OR component_id = 0)", projectID).Order("created_at desc").First(&diag).Error; err != nil {
		return nil, err
	}
	return &diag, nil
}

func (r *gormDiagnosticRepository) FindByComponentID(ctx context.Context, componentID uint) ([]models.Diagnostic, error) {
	var diags []models.Diagnostic
	if err := r.db.WithContext(ctx).Where("component_id = ?", componentID).Order("created_at desc").Find(&diags).Error; err != nil {
		return nil, err
	}
	return diags, nil
}

func (r *gormDiagnosticRepository) FindLatestByComponentID(ctx context.Context, componentID uint) (*models.Diagnostic, error) {
	var diag models.Diagnostic
	if err := r.db.WithContext(ctx).Where("component_id = ?", componentID).Order("created_at desc").First(&diag).Error; err != nil {
		return nil, err
	}
	return &diag, nil
}
