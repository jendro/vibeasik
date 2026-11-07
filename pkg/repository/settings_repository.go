package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// SettingsRepository defines data access methods for LLMSettings and LLMConfig models
type SettingsRepository interface {
	GetSettings(ctx context.Context) (*models.LLMSettings, error)
	UpdateSettings(ctx context.Context, settings *models.LLMSettings) error
	GetAllConfigs(ctx context.Context) ([]models.LLMConfig, error)
	GetConfigByID(ctx context.Context, id uint) (*models.LLMConfig, error)
	GetActiveConfig(ctx context.Context, provider string) (*models.LLMConfig, error)
	CreateConfig(ctx context.Context, cfg *models.LLMConfig) error
	UpdateConfig(ctx context.Context, cfg *models.LLMConfig) error
	DeleteConfig(ctx context.Context, id uint) error
	SetActiveConfig(ctx context.Context, id uint) (*models.LLMConfig, error)
}

type gormSettingsRepository struct {
	db *gorm.DB
}

// NewSettingsRepository instantiates a new SettingsRepository backed by GORM
func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &gormSettingsRepository{db: db}
}

func (r *gormSettingsRepository) GetSettings(ctx context.Context) (*models.LLMSettings, error) {
	var settings models.LLMSettings
	if err := r.db.WithContext(ctx).First(&settings).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

func (r *gormSettingsRepository) UpdateSettings(ctx context.Context, settings *models.LLMSettings) error {
	return r.db.WithContext(ctx).Save(settings).Error
}

func (r *gormSettingsRepository) GetAllConfigs(ctx context.Context) ([]models.LLMConfig, error) {
	var list []models.LLMConfig
	if err := r.db.WithContext(ctx).Order("id asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *gormSettingsRepository) GetConfigByID(ctx context.Context, id uint) (*models.LLMConfig, error) {
	var cfg models.LLMConfig
	if err := r.db.WithContext(ctx).First(&cfg, id).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *gormSettingsRepository) GetActiveConfig(ctx context.Context, provider string) (*models.LLMConfig, error) {
	var cfg models.LLMConfig
	query := r.db.WithContext(ctx).Where("is_active = ?", true)
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if err := query.First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *gormSettingsRepository) CreateConfig(ctx context.Context, cfg *models.LLMConfig) error {
	return r.db.WithContext(ctx).Create(cfg).Error
}

func (r *gormSettingsRepository) UpdateConfig(ctx context.Context, cfg *models.LLMConfig) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}

func (r *gormSettingsRepository) DeleteConfig(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.LLMConfig{}, id).Error
}

func (r *gormSettingsRepository) SetActiveConfig(ctx context.Context, id uint) (*models.LLMConfig, error) {
	var target models.LLMConfig
	if err := r.db.WithContext(ctx).First(&target, id).Error; err != nil {
		return nil, fmt.Errorf("config not found: %w", err)
	}

	// Deactivate other configs with the same provider
	if err := r.db.WithContext(ctx).Model(&models.LLMConfig{}).Where("provider = ?", target.Provider).Update("is_active", false).Error; err != nil {
		return nil, fmt.Errorf("failed to deactivate other configs: %w", err)
	}

	// Activate target config
	target.IsActive = true
	if err := r.db.WithContext(ctx).Save(&target).Error; err != nil {
		return nil, fmt.Errorf("failed to activate config: %w", err)
	}

	return &target, nil
}
