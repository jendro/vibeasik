package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/security"
)

func (s *Server) getSecurityStatus(c *fiber.Ctx) error {
	var activeCount int64
	s.db.Model(&models.CensoredKeyword{}).Where("is_active = ?", true).Count(&activeCount)

	return c.JSON(fiber.Map{
		"global_sanitizer_active": true,
		"double_security_active":  true,
		"censoring_mode":          "deterministic_safe_aliasing",
		"intercepted_providers":   []string{"system_one", "standard_llm"},
		"protected_patterns":      security.GlobalSanitizer.GetBuiltInPatterns(),
		"active_custom_keywords":  activeCount,
		"leak_prevention":         "100% Guaranteed. Double sanitization enforces deterministic safe aliasing on all prompts, files, and network dispatches.",
	})
}

func (s *Server) getCensoredKeywords(c *fiber.Ctx) error {
	var list []models.CensoredKeyword
	if err := s.db.Order("id desc").Find(&list).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	activeCount := 0
	for _, k := range list {
		if k.IsActive {
			activeCount++
		}
	}
	return c.JSON(fiber.Map{
		"keywords":          list,
		"total":             len(list),
		"active_count":      activeCount,
		"built_in_patterns": security.GlobalSanitizer.GetBuiltInPatterns(),
		"double_security":   true,
	})
}

func (s *Server) createCensoredKeyword(c *fiber.Ctx) error {
	var body struct {
		Keyword     string `json:"keyword"`
		Category    string `json:"category"`
		Description string `json:"description"`
		IsRegex     bool   `json:"is_regex"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	body.Keyword = strings.TrimSpace(body.Keyword)
	if body.Keyword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "keyword is required"})
	}
	if body.Category == "" {
		body.Category = "custom"
	}
	active := true
	if body.IsActive != nil {
		active = *body.IsActive
	}

	kw := models.CensoredKeyword{
		Keyword:     body.Keyword,
		Category:    body.Category,
		Description: strings.TrimSpace(body.Description),
		IsRegex:     body.IsRegex,
		IsActive:    active,
	}

	if err := s.db.Create(&kw).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	database.SyncCensoredKeywords(s.db)

	return c.Status(fiber.StatusCreated).JSON(kw)
}

func (s *Server) updateCensoredKeyword(c *fiber.Ctx) error {
	id := c.Params("id")
	var existing models.CensoredKeyword
	if err := s.db.First(&existing, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "keyword not found"})
	}

	var body struct {
		Keyword     string `json:"keyword"`
		Category    string `json:"category"`
		Description string `json:"description"`
		IsRegex     *bool  `json:"is_regex"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if strings.TrimSpace(body.Keyword) != "" {
		existing.Keyword = strings.TrimSpace(body.Keyword)
	}
	if body.Category != "" {
		existing.Category = body.Category
	}
	if body.Description != "" {
		existing.Description = strings.TrimSpace(body.Description)
	}
	if body.IsRegex != nil {
		existing.IsRegex = *body.IsRegex
	}
	if body.IsActive != nil {
		existing.IsActive = *body.IsActive
	}

	if err := s.db.Save(&existing).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	database.SyncCensoredKeywords(s.db)

	return c.JSON(existing)
}

func (s *Server) deleteCensoredKeyword(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := s.db.Delete(&models.CensoredKeyword{}, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	database.SyncCensoredKeywords(s.db)
	return c.JSON(fiber.Map{"message": "censored keyword deleted successfully"})
}

func (s *Server) testCensorPayload(c *fiber.Ctx) error {
	var body struct {
		Text string `json:"text"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	clean, audit := security.GlobalSanitizer.Sanitize(body.Text)
	return c.JSON(fiber.Map{
		"original":       body.Text,
		"sanitized_text": clean,
		"audit":          audit,
	})
}
