package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
	"github.com/vibeasik/vibeasik/pkg/security"
)

// SecurityService defines business logic for data protection and keyword sanitization
type SecurityService interface {
	GetStatus(ctx context.Context) (map[string]interface{}, error)
	GetKeywords(ctx context.Context) ([]models.CensoredKeyword, error)
	GetKeywordByID(ctx context.Context, id uint) (*models.CensoredKeyword, error)
	CreateKeyword(ctx context.Context, kw *models.CensoredKeyword) (*models.CensoredKeyword, error)
	UpdateKeyword(ctx context.Context, id uint, updates *models.CensoredKeyword) (*models.CensoredKeyword, error)
	DeleteKeyword(ctx context.Context, id uint) error
	TestCensor(ctx context.Context, payload string) (map[string]interface{}, error)
	ReloadSanitizer(ctx context.Context) error
}

type securityService struct {
	securityRepo repository.SecurityRepository
	db           *gorm.DB
}

// NewSecurityService instantiates a new SecurityService
func NewSecurityService(securityRepo repository.SecurityRepository, db *gorm.DB) SecurityService {
	return &securityService{
		securityRepo: securityRepo,
		db:           db,
	}
}

func (s *securityService) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	all, err := s.securityRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	activeCount := 0
	regexCount := 0
	for _, kw := range all {
		if kw.IsActive {
			activeCount++
		}
		if kw.IsRegex {
			regexCount++
		}
	}

	return map[string]interface{}{
		"active":          true,
		"censoring_mode":  "deterministic_alias",
		"total_keywords":  len(all),
		"active_keywords": activeCount,
		"regex_rules":     regexCount,
		"built_in_rules":  12,
		"description":     "Real-time heuristic and custom pattern masker before LLM dispatch.",
	}, nil
}

func (s *securityService) GetKeywords(ctx context.Context) ([]models.CensoredKeyword, error) {
	return s.securityRepo.GetAll(ctx)
}

func (s *securityService) GetKeywordByID(ctx context.Context, id uint) (*models.CensoredKeyword, error) {
	return s.securityRepo.GetByID(ctx, id)
}

func (s *securityService) CreateKeyword(ctx context.Context, kw *models.CensoredKeyword) (*models.CensoredKeyword, error) {
	if kw.Keyword == "" {
		return nil, fmt.Errorf("keyword is required")
	}

	if err := s.securityRepo.Create(ctx, kw); err != nil {
		return nil, err
	}

	_ = s.ReloadSanitizer(ctx)
	return kw, nil
}

func (s *securityService) UpdateKeyword(ctx context.Context, id uint, updates *models.CensoredKeyword) (*models.CensoredKeyword, error) {
	existing, err := s.securityRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if updates.Keyword != "" {
		existing.Keyword = updates.Keyword
	}
	if updates.Category != "" {
		existing.Category = updates.Category
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	existing.IsRegex = updates.IsRegex
	existing.IsActive = updates.IsActive

	if err := s.securityRepo.Update(ctx, existing); err != nil {
		return nil, err
	}

	_ = s.ReloadSanitizer(ctx)
	return existing, nil
}

func (s *securityService) DeleteKeyword(ctx context.Context, id uint) error {
	if err := s.securityRepo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.ReloadSanitizer(ctx)
	return nil
}

func (s *securityService) TestCensor(ctx context.Context, payload string) (map[string]interface{}, error) {
	sanitized, audit := security.GlobalSanitizer.Sanitize(payload)
	return map[string]interface{}{
		"original":  payload,
		"sanitized": sanitized,
		"audit":     audit,
	}, nil
}

func (s *securityService) ReloadSanitizer(ctx context.Context) error {
	database.SyncCensoredKeywords(s.db)
	return nil
}
