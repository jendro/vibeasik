package api

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type DeprecatePayload struct {
	Reason string `json:"reason"`
}

func (s *Server) deprecateProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	var payload DeprecatePayload
	_ = c.BodyParser(&payload)

	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		reason = "Ditandai sebagai deprecated oleh pengguna."
	}

	project, err := s.services.Projects.MarkDeprecated(c.Context(), uint(id), reason)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Proyek berhasil ditandai sebagai deprecated",
		"project": project,
	})
}

func (s *Server) undeprecateProject(c *fiber.Ctx) error {
	return s.activateProject(c)
}

func (s *Server) activateProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	project, err := s.services.Projects.ActivateProject(c.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Proyek berhasil diaktifkan kembali",
		"project": project,
	})
}

func (s *Server) holdProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	project, err := s.services.Projects.MarkHold(c.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Proyek berhasil ditandai sebagai hold",
		"project": project,
	})
}

func (s *Server) doneProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	project, err := s.services.Projects.MarkDone(c.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Proyek berhasil ditandai sebagai selesai (done)",
		"project": project,
	})
}
