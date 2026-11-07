package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// SecurityRepository defines data access methods for CensoredKeyword models
type SecurityRepository interface {
	GetAll(ctx context.Context) ([]models.CensoredKeyword, error)
	GetActive(ctx context.Context) ([]models.CensoredKeyword, error)
	GetByID(ctx context.Context, id uint) (*models.CensoredKeyword, error)
	Create(ctx context.Context, kw *models.CensoredKeyword) error
	Update(ctx context.Context, kw *models.CensoredKeyword) error
	Delete(ctx context.Context, id uint) error
}

type gormSecurityRepository struct {
	db *gorm.DB
}

// NewSecurityRepository instantiates a new SecurityRepository backed by GORM
func NewSecurityRepository(db *gorm.DB) SecurityRepository {
	return &gormSecurityRepository{db: db}
}

func (r *gormSecurityRepository) GetAll(ctx context.Context) ([]models.CensoredKeyword, error) {
	var list []models.CensoredKeyword
	if err := r.db.WithContext(ctx).Order("id asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *gormSecurityRepository) GetActive(ctx context.Context) ([]models.CensoredKeyword, error) {
	var list []models.CensoredKeyword
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("id asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *gormSecurityRepository) GetByID(ctx context.Context, id uint) (*models.CensoredKeyword, error) {
	var kw models.CensoredKeyword
	if err := r.db.WithContext(ctx).First(&kw, id).Error; err != nil {
		return nil, err
	}
	return &kw, nil
}

func (r *gormSecurityRepository) Create(ctx context.Context, kw *models.CensoredKeyword) error {
	return r.db.WithContext(ctx).Create(kw).Error
}

func (r *gormSecurityRepository) Update(ctx context.Context, kw *models.CensoredKeyword) error {
	return r.db.WithContext(ctx).Save(kw).Error
}

func (r *gormSecurityRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.CensoredKeyword{}, id).Error
}
