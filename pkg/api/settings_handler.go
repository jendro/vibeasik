package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func (s *Server) getLLMSettings(c *fiber.Ctx) error {
	dto, err := s.services.Settings.GetSettings(c.Context())
	if err != nil {
		// Provide default settings if empty
		defaultModels := "[\"gpt-4o-mini\", \"gpt-4o\", \"deepseek-chat\", \"claude-3-5-sonnet\", \"qwen-2.5-coder\"]"
		settings := models.LLMSettings{
			ActiveProvider:      models.ProviderSystemOne,
			SystemOneURL:        "http://localhost:8750/v1/systemone",
			SystemOneModel:      "convaiinnovations/laya-multilingual",
			SystemOneKey:        "",
			StandardLLMURL:      "https://api.openai.com/v1",
			StandardLLMKey:      "",
			StandardLLMModels:   defaultModels,
			StandardLLMSelected: "gpt-4o-mini",
		}
		_ = s.repos.Settings.UpdateSettings(c.Context(), &settings)
		d := settings.ToDTO()
		return c.JSON(d)
	}
	return c.JSON(dto)
}

func (s *Server) updateLLMSettings(c *fiber.Ctx) error {
	var dto models.LLMSettingsDTO
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	var settings models.LLMSettings
	settings.FromDTO(dto)

	updated, err := s.services.Settings.UpdateSettings(c.Context(), &settings)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(updated)
}

func (s *Server) setActiveProvider(c *fiber.Ctx) error {
	var body struct {
		Provider string `json:"provider"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := s.services.Settings.SetActiveProvider(c.Context(), body.Provider); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":         "active provider updated",
		"active_provider": body.Provider,
	})
}

func (s *Server) selectActiveModel(c *fiber.Ctx) error {
	var body struct {
		Model string `json:"model"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Model) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "model is required"})
	}

	selected := strings.TrimSpace(body.Model)
	if err := s.services.Settings.SelectActiveModel(c.Context(), selected); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":        "active model updated successfully",
		"selected_model": selected,
	})
}

func (s *Server) getLLMConfigs(c *fiber.Ctx) error {
	configs, err := s.services.Settings.GetConfigs(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if provider := c.Query("provider"); provider != "" {
		filtered := make([]models.LLMConfigDTO, 0)
		for _, cfg := range configs {
			if cfg.Provider == provider {
				filtered = append(filtered, cfg)
			}
		}
		return c.JSON(filtered)
	}

	return c.JSON(configs)
}

func (s *Server) getLLMConfig(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid config id"})
	}

	cfg, err := s.services.Settings.GetConfigByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "configuration not found"})
	}
	return c.JSON(cfg)
}

type UpsertLLMConfigPayload struct {
	Provider      string   `json:"provider"`
	Name          string   `json:"name"`
	URL           string   `json:"url"`
	Key           string   `json:"key"`
	Models        []string `json:"models"`
	SelectedModel string   `json:"selected_model"`
	IsActive      bool     `json:"is_active"`
}

func (s *Server) createLLMConfig(c *fiber.Ctx) error {
	var payload UpsertLLMConfigPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	payload.Name = strings.TrimSpace(payload.Name)
	payload.URL = strings.TrimSpace(payload.URL)
	if payload.Name == "" || payload.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name and url are required"})
	}
	if payload.Provider != models.ProviderSystemOne && payload.Provider != models.ProviderStandardLLM {
		payload.Provider = models.ProviderStandardLLM
	}

	cfg := models.LLMConfig{
		Provider:      payload.Provider,
		Name:          payload.Name,
		URL:           payload.URL,
		Key:           payload.Key,
		SelectedModel: payload.SelectedModel,
		IsActive:      payload.IsActive,
	}

	dto, err := s.services.Settings.CreateConfig(c.Context(), &cfg, payload.Models)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(dto)
}

func (s *Server) updateLLMConfig(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid config id"})
	}

	var payload UpsertLLMConfigPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	cfgUpdates := models.LLMConfig{
		Provider:      payload.Provider,
		Name:          payload.Name,
		URL:           payload.URL,
		Key:           payload.Key,
		SelectedModel: payload.SelectedModel,
		IsActive:      payload.IsActive,
	}

	dto, err := s.services.Settings.UpdateConfig(c.Context(), uint(id), &cfgUpdates, payload.Models)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(dto)
}

func (s *Server) deleteLLMConfig(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid config id"})
	}

	cfg, err := s.repos.Settings.GetConfigByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "configuration not found"})
	}

	wasActive := cfg.IsActive
	provider := cfg.Provider

	if err := s.services.Settings.DeleteConfig(c.Context(), uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if wasActive {
		var next models.LLMConfig
		if err := s.db.Where("provider = ?", provider).Order("id asc").First(&next).Error; err == nil {
			_, _ = s.services.Settings.ActivateConfig(c.Context(), next.ID)
		}
	}

	return c.JSON(fiber.Map{"message": "configuration deleted successfully"})
}

func (s *Server) activateLLMConfig(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid config id"})
	}

	dto, err := s.services.Settings.ActivateConfig(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "configuration not found"})
	}

	return c.JSON(fiber.Map{
		"message": "configuration activated successfully",
		"config":  dto,
	})
}

// setLLMModelVision: tandai satu model konfigurasi menerima gambar (vision) atau tidak.
func (s *Server) setLLMModelVision(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid config id"})
	}
	var body struct {
		Model  string `json:"model"`
		Vision bool   `json:"vision"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Model) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "model wajib diisi"})
	}
	var cfg models.LLMConfig
	if err := s.db.First(&cfg, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "configuration not found"})
	}
	cfg.SetModelVision(strings.TrimSpace(body.Model), body.Vision)
	if err := s.db.Model(&cfg).Update("vision_json", cfg.VisionJSON).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cfg.ToDTO())
}

// setLLMModelAlias: beri alias tampilan untuk satu model konfigurasi (alias kosong = hapus).
func (s *Server) setLLMModelAlias(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid config id"})
	}
	var body struct {
		Model string `json:"model"`
		Alias string `json:"alias"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Model) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "model wajib diisi"})
	}
	var cfg models.LLMConfig
	if err := s.db.First(&cfg, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "configuration not found"})
	}
	cfg.SetModelAlias(strings.TrimSpace(body.Model), body.Alias)
	if err := s.db.Model(&cfg).Update("alias_json", cfg.AliasJSON).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cfg.ToDTO())
}
