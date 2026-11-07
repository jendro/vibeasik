package api

import (
	"github.com/gofiber/fiber/v2"
)

// getScaleUpPatterns returns all documented architecture patterns and folder blueprints
func (s *Server) getScaleUpPatterns(c *fiber.Ctx) error {
	patterns := s.services.ScaleUp.GetPatterns()
	return c.JSON(patterns)
}

// evaluateComponentArchitecture runs an architectural inspection on a single component
func (s *Server) evaluateComponentArchitecture(c *fiber.Ctx) error {
	componentID, err := c.ParamsInt("componentId")
	if err != nil || componentID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID komponen tidak valid"})
	}

	eval, err := s.services.ScaleUp.EvaluateComponent(c.Context(), uint(componentID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(eval)
}

// evaluateProjectArchitecture evaluates all code components in a project
func (s *Server) evaluateProjectArchitecture(c *fiber.Ctx) error {
	projectID, err := c.ParamsInt("projectId")
	if err != nil || projectID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID proyek tidak valid"})
	}

	evals, err := s.services.ScaleUp.EvaluateProject(c.Context(), uint(projectID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(evals)
}
