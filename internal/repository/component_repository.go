package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/internal/models"
)

// ComponentRepository defines data access methods for Component models
type ComponentRepository interface {
	FindByID(ctx context.Context, id uint) (*models.Component, error)
	FindByProjectID(ctx context.Context, projectID uint) ([]models.Component, error)
	Create(ctx context.Context, comp *models.Component) error
	Update(ctx context.Context, comp *models.Component) error
	Delete(ctx context.Context, id uint) error
	UpdateFastScanData(ctx context.Context, id uint, data string) error
	UpdateFields(ctx context.Context, id uint, updates map[string]interface{}) error
	UpdateHealthScore(ctx context.Context, id uint, score int) error
}

type gormComponentRepository struct {
	db *gorm.DB
}

// NewComponentRepository instantiates a new ComponentRepository backed by GORM
func NewComponentRepository(db *gorm.DB) ComponentRepository {
	return &gormComponentRepository{db: db}
}

func (r *gormComponentRepository) FindByID(ctx context.Context, id uint) (*models.Component, error) {
	var comp models.Component
	if err := r.db.WithContext(ctx).First(&comp, id).Error; err != nil {
		return nil, err
	}
	return &comp, nil
}

func (r *gormComponentRepository) FindByProjectID(ctx context.Context, projectID uint) ([]models.Component, error) {
	var comps []models.Component
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&comps).Error; err != nil {
		return nil, err
	}
	return comps, nil
}

func (r *gormComponentRepository) Create(ctx context.Context, comp *models.Component) error {
	return r.db.WithContext(ctx).Create(comp).Error
}

func (r *gormComponentRepository) Update(ctx context.Context, comp *models.Component) error {
	return r.db.WithContext(ctx).Save(comp).Error
}

func (r *gormComponentRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Component{}, id).Error
}

func (r *gormComponentRepository) UpdateFastScanData(ctx context.Context, id uint, data string) error {
	return r.db.WithContext(ctx).Model(&models.Component{}).Where("id = ?", id).Update("fast_scan_data", data).Error
}

func (r *gormComponentRepository) UpdateFields(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Component{}).Where("id = ?", id).Updates(updates).Error
}

func (r *gormComponentRepository) UpdateHealthScore(ctx context.Context, id uint, score int) error {
	return r.db.WithContext(ctx).Model(&models.Component{}).Where("id = ?", id).Update("health_score", score).Error
}
