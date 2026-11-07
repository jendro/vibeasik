package api

import (
	"github.com/gofiber/fiber/v2"
)

func (s *Server) scanSimilarities(c *fiber.Ctx) error {
	result, err := s.services.Similarity.ScanSimilarities(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (s *Server) compareProjects(c *fiber.Ctx) error {
	idA, err := c.ParamsInt("idA")
	if err != nil || idA <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "idA tidak valid"})
	}
	idB, err := c.ParamsInt("idB")
	if err != nil || idB <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "idB tidak valid"})
	}

	match, err := s.services.Similarity.CompareTwoProjects(c.Context(), uint(idA), uint(idB))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(match)
}
