package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/internal/models"
)

// RouteRepository defines data access methods for ComponentRoute models
type RouteRepository interface {
	FindByComponentID(ctx context.Context, componentID uint) ([]models.ComponentRoute, error)
	FindByID(ctx context.Context, id uint) (*models.ComponentRoute, error)
	Create(ctx context.Context, route *models.ComponentRoute) error
	Update(ctx context.Context, route *models.ComponentRoute) error
	Delete(ctx context.Context, id uint) error
	DeleteByIDs(ctx context.Context, ids []uint) error
	DeleteByComponentID(ctx context.Context, componentID uint) error
	BulkCreate(ctx context.Context, routes []models.ComponentRoute) error
}

type gormRouteRepository struct {
	db *gorm.DB
}

// NewRouteRepository instantiates a new RouteRepository backed by GORM
func NewRouteRepository(db *gorm.DB) RouteRepository {
	return &gormRouteRepository{db: db}
}

func (r *gormRouteRepository) FindByComponentID(ctx context.Context, componentID uint) ([]models.ComponentRoute, error) {
	var routes []models.ComponentRoute
	if err := r.db.WithContext(ctx).Where("component_id = ?", componentID).Order("path asc").Find(&routes).Error; err != nil {
		return nil, err
	}
	return routes, nil
}

func (r *gormRouteRepository) FindByID(ctx context.Context, id uint) (*models.ComponentRoute, error) {
	var route models.ComponentRoute
	if err := r.db.WithContext(ctx).First(&route, id).Error; err != nil {
		return nil, err
	}
	return &route, nil
}

func (r *gormRouteRepository) Create(ctx context.Context, route *models.ComponentRoute) error {
	return r.db.WithContext(ctx).Create(route).Error
}

func (r *gormRouteRepository) Update(ctx context.Context, route *models.ComponentRoute) error {
	return r.db.WithContext(ctx).Save(route).Error
}

func (r *gormRouteRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.ComponentRoute{}, id).Error
}

func (r *gormRouteRepository) DeleteByIDs(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.ComponentRoute{}).Error
}

func (r *gormRouteRepository) DeleteByComponentID(ctx context.Context, componentID uint) error {
	return r.db.WithContext(ctx).Where("component_id = ?", componentID).Delete(&models.ComponentRoute{}).Error
}

func (r *gormRouteRepository) BulkCreate(ctx context.Context, routes []models.ComponentRoute) error {
	if len(routes) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&routes).Error
}
