package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/knowledgesync"
	"github.com/vibeasik/vibeasik/pkg/setup"
)

func (s *Server) completeSetup(c *fiber.Ctx) error {
	if err := setup.Finish(knowledgesync.ConsentVersion); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(s.viewWithRestart())
}
