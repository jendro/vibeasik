package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/service"
)

// runProjectSecurityTest: POST /api/projects/:id/security-test — audit router/sumber backend proyek dan
// probe server berjalan (port lokal komponen atau base_url live) untuk kontrol keamanan standar.
func (s *Server) runProjectSecurityTest(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	var req service.SecurityTestRequest
	if msg := bindJSON(c, &req); msg != "" {
		return badPayload(c, msg)
	}
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	if msg := firstProblem(
		maxLen("base_url", req.BaseURL, maxURL),
		burstProblem(req.Burst),
	); msg != "" {
		return badPayload(c, msg)
	}
	req.ProjectID = uint(id)
	rep, err := s.services.SecTest.Run(c.UserContext(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rep)
}

// getProjectSecurityTest: GET /api/projects/:id/security-test — hasil uji terakhir sejak engine menyala.
func (s *Server) getProjectSecurityTest(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	rep, ok := s.services.SecTest.Last(uint(id))
	if !ok {
		return c.JSON(fiber.Map{"report": nil})
	}
	return c.JSON(fiber.Map{"report": rep})
}

func burstProblem(n int) string {
	if n < 0 || n > 60 {
		return "burst harus 0–60"
	}
	return ""
}
