package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
)

// SettingsService defines business logic for LLM settings and multi-provider configurations
type SettingsService interface {
	GetSettings(ctx context.Context) (*models.LLMSettingsDTO, error)
	UpdateSettings(ctx context.Context, updates *models.LLMSettings) (*models.LLMSettingsDTO, error)
	SetActiveProvider(ctx context.Context, provider string) error
	SelectActiveModel(ctx context.Context, model string) error
	GetConfigs(ctx context.Context) ([]models.LLMConfigDTO, error)
	GetConfigByID(ctx context.Context, id uint) (*models.LLMConfigDTO, error)
	CreateConfig(ctx context.Context, cfg *models.LLMConfig, rawModels []string) (*models.LLMConfigDTO, error)
	UpdateConfig(ctx context.Context, id uint, updates *models.LLMConfig, rawModels []string) (*models.LLMConfigDTO, error)
	DeleteConfig(ctx context.Context, id uint) error
	ActivateConfig(ctx context.Context, id uint) (*models.LLMConfigDTO, error)
}

type settingsService struct {
	settingsRepo repository.SettingsRepository
}

// NewSettingsService instantiates a new SettingsService
func NewSettingsService(settingsRepo repository.SettingsRepository) SettingsService {
	return &settingsService{settingsRepo: settingsRepo}
}

func (s *settingsService) GetSettings(ctx context.Context) (*models.LLMSettingsDTO, error) {
	settings, err := s.settingsRepo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	dto := settings.ToDTO()
	return &dto, nil
}

func (s *settingsService) UpdateSettings(ctx context.Context, updates *models.LLMSettings) (*models.LLMSettingsDTO, error) {
	existing, err := s.settingsRepo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	if updates.ActiveProvider != "" {
		existing.ActiveProvider = updates.ActiveProvider
	}
	if updates.SystemOneURL != "" {
		existing.SystemOneURL = updates.SystemOneURL
	}
	if updates.SystemOneModel != "" {
		existing.SystemOneModel = updates.SystemOneModel
	}
	if updates.SystemOneKey != "" && !models.IsMaskedKey(updates.SystemOneKey) {
		existing.SystemOneKey = updates.SystemOneKey
	}
	if updates.StandardLLMURL != "" {
		existing.StandardLLMURL = updates.StandardLLMURL
	}
	if updates.StandardLLMKey != "" && !models.IsMaskedKey(updates.StandardLLMKey) {
		existing.StandardLLMKey = updates.StandardLLMKey
	}
	if updates.StandardLLMModels != "" {
		existing.StandardLLMModels = updates.StandardLLMModels
	}
	if updates.StandardLLMSelected != "" {
		existing.StandardLLMSelected = updates.StandardLLMSelected
	}

	if err := s.settingsRepo.UpdateSettings(ctx, existing); err != nil {
		return nil, err
	}
	dto := existing.ToDTO()
	return &dto, nil
}

func (s *settingsService) SetActiveProvider(ctx context.Context, provider string) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider != models.ProviderSystemOne && provider != models.ProviderStandardLLM {
		return fmt.Errorf("invalid provider. Must be %s or %s", models.ProviderSystemOne, models.ProviderStandardLLM)
	}

	settings, err := s.settingsRepo.GetSettings(ctx)
	if err != nil {
		return err
	}
	settings.ActiveProvider = provider

	// Pre-fetch active config and apply directly to settings before single update
	if activeCfg, err := s.settingsRepo.GetActiveConfig(ctx, provider); err == nil && activeCfg != nil {
		s.applyConfigToSettings(settings, activeCfg)
	}

	return s.settingsRepo.UpdateSettings(ctx, settings)
}

func (s *settingsService) SelectActiveModel(ctx context.Context, model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("model is required")
	}

	settings, err := s.settingsRepo.GetSettings(ctx)
	if err != nil {
		return err
	}

	activeCfg, _ := s.settingsRepo.GetActiveConfig(ctx, settings.ActiveProvider)
	if activeCfg != nil && activeCfg.SelectedModel != model {
		activeCfg.SelectedModel = model
		_ = s.settingsRepo.UpdateConfig(ctx, activeCfg)
	}

	var changed bool
	if settings.ActiveProvider == models.ProviderStandardLLM {
		if settings.StandardLLMSelected != model {
			settings.StandardLLMSelected = model
			changed = true
		}
	} else {
		if settings.SystemOneModel != model {
			settings.SystemOneModel = model
			changed = true
		}
	}

	if !changed && activeCfg != nil && activeCfg.SelectedModel == model {
		return nil
	}

	return s.settingsRepo.UpdateSettings(ctx, settings)
}

func (s *settingsService) GetConfigs(ctx context.Context) ([]models.LLMConfigDTO, error) {
	configs, err := s.settingsRepo.GetAllConfigs(ctx)
	if err != nil {
		return nil, err
	}
	dtos := make([]models.LLMConfigDTO, 0, len(configs))
	for _, c := range configs {
		dtos = append(dtos, c.ToDTO())
	}
	return dtos, nil
}

func (s *settingsService) GetConfigByID(ctx context.Context, id uint) (*models.LLMConfigDTO, error) {
	cfg, err := s.settingsRepo.GetConfigByID(ctx, id)
	if err != nil {
		return nil, err
	}
	dto := cfg.ToDTO()
	return &dto, nil
}

func (s *settingsService) CreateConfig(ctx context.Context, cfg *models.LLMConfig, rawModels []string) (*models.LLMConfigDTO, error) {
	if cfg.Provider == "" || cfg.Name == "" || cfg.URL == "" {
		return nil, fmt.Errorf("provider, name, and url are required")
	}

	if len(rawModels) > 0 {
		b, _ := json.Marshal(rawModels)
		cfg.Models = string(b)
		if cfg.SelectedModel == "" {
			cfg.SelectedModel = rawModels[0]
		}
	}

	if cfg.IsActive {
		// Set active will deactivate others
		if err := s.settingsRepo.CreateConfig(ctx, cfg); err != nil {
			return nil, err
		}
		activated, err := s.settingsRepo.SetActiveConfig(ctx, cfg.ID)
		if err != nil {
			return nil, err
		}
		s.syncConfigToSettings(ctx, activated)
		dto := activated.ToDTO()
		return &dto, nil
	}

	if err := s.settingsRepo.CreateConfig(ctx, cfg); err != nil {
		return nil, err
	}
	dto := cfg.ToDTO()
	return &dto, nil
}

func (s *settingsService) UpdateConfig(ctx context.Context, id uint, updates *models.LLMConfig, rawModels []string) (*models.LLMConfigDTO, error) {
	existing, err := s.settingsRepo.GetConfigByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.URL != "" {
		existing.URL = updates.URL
	}
	if updates.Key != "" && !models.IsMaskedKey(updates.Key) {
		existing.Key = updates.Key
	}
	if updates.Provider != "" {
		existing.Provider = updates.Provider
	}
	if len(rawModels) > 0 {
		b, _ := json.Marshal(rawModels)
		existing.Models = string(b)
	}
	if updates.SelectedModel != "" {
		existing.SelectedModel = updates.SelectedModel
	}
	if updates.AliasJSON != "" {
		existing.AliasJSON = updates.AliasJSON
	}

	if updates.IsActive && !existing.IsActive {
		if err := s.settingsRepo.UpdateConfig(ctx, existing); err != nil {
			return nil, err
		}
		activated, err := s.settingsRepo.SetActiveConfig(ctx, existing.ID)
		if err != nil {
			return nil, err
		}
		s.syncConfigToSettings(ctx, activated)
		dto := activated.ToDTO()
		return &dto, nil
	}

	if err := s.settingsRepo.UpdateConfig(ctx, existing); err != nil {
		return nil, err
	}
	if existing.IsActive {
		s.syncConfigToSettings(ctx, existing)
	}

	dto := existing.ToDTO()
	return &dto, nil
}

func (s *settingsService) DeleteConfig(ctx context.Context, id uint) error {
	return s.settingsRepo.DeleteConfig(ctx, id)
}

func (s *settingsService) ActivateConfig(ctx context.Context, id uint) (*models.LLMConfigDTO, error) {
	activated, err := s.settingsRepo.SetActiveConfig(ctx, id)
	if err != nil {
		return nil, err
	}
	s.syncConfigToSettings(ctx, activated)
	dto := activated.ToDTO()
	return &dto, nil
}

func (s *settingsService) applyConfigToSettings(settings *models.LLMSettings, cfg *models.LLMConfig) {
	if cfg == nil || settings == nil {
		return
	}

	if cfg.Provider == models.ProviderSystemOne {
		settings.SystemOneURL = cfg.URL
		if cfg.Key != "" {
			settings.SystemOneKey = cfg.Key
		}
		if cfg.SelectedModel != "" {
			settings.SystemOneModel = cfg.SelectedModel
		}
		settings.ActiveProvider = models.ProviderSystemOne
	} else {
		settings.StandardLLMURL = cfg.URL
		if cfg.Key != "" {
			settings.StandardLLMKey = cfg.Key
		}
		if cfg.Models != "" {
			settings.StandardLLMModels = cfg.Models
		}
		if cfg.SelectedModel != "" {
			settings.StandardLLMSelected = cfg.SelectedModel
		}
		settings.ActiveProvider = models.ProviderStandardLLM
	}
}

func (s *settingsService) syncConfigToSettings(ctx context.Context, cfg *models.LLMConfig) {
	if cfg == nil {
		return
	}
	settings, err := s.settingsRepo.GetSettings(ctx)
	if err != nil {
		return
	}

	s.applyConfigToSettings(settings, cfg)
	_ = s.settingsRepo.UpdateSettings(ctx, settings)
}
