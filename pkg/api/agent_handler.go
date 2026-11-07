package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/agents"
	"github.com/vibeasik/vibeasik/pkg/models"
)

func (s *Server) listLocalAgents(c *fiber.Ctx) error {
	reg := agents.GetRegistry()
	list := reg.DiscoverAll(s.db)
	return c.JSON(fiber.Map{
		"status": "ok",
		"agents": list,
	})
}

func (s *Server) executeLocalAgent(c *fiber.Ctx) error {
	var req models.AgentExecutionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Format request tidak valid: " + err.Error()})
	}

	if req.AgentKey == "" {
		return c.Status(400).JSON(fiber.Map{"error": "AgentKey wajib dipilih"})
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Prompt instruksi wajib diisi"})
	}

	reg := agents.GetRegistry()
	res, err := reg.Execute(c.Context(), req, s.db)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(res)
}

func (s *Server) createCustomAgent(c *fiber.Ctx) error {
	var body models.LocalAgentConfig
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Format request tidak valid: " + err.Error()})
	}

	body.Key = strings.ToLower(strings.TrimSpace(body.Key))
	if body.Key == "" || body.Command == "" || body.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Field 'key', 'name', dan 'command' wajib diisi"})
	}

	if body.Icon == "" {
		body.Icon = "i-lucide-terminal"
	}
	body.Enabled = true

	if err := s.db.Create(&body).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal menyimpan agent custom: " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"agent":   body,
		"message": "Agent custom plug-and-play berhasil ditambahkan!",
	})
}

func (s *Server) deleteCustomAgent(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Key agent wajib diisi"})
	}

	if err := s.db.Where("key = ?", key).Delete(&models.LocalAgentConfig{}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Agent custom berhasil dihapus",
	})
}
