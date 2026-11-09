package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func (s *Server) generateReorgPlan(c *fiber.Ctx) error {
	var req models.ReorgPlanRequest
	if msg := firstProblem(bindJSON(c, &req), absPath("target_directory", req.TargetDirectory),
		maxLen("strategy", req.Strategy, maxName)); msg != "" {
		return badPayload(c, msg)
	}

	plan, err := s.services.Organizer.GenerateReorgPlan(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(plan)
}

func (s *Server) executeReorgMoves(c *fiber.Ctx) error {
	var payload models.ExecuteMovePayload
	if msg := bindJSONRequired(c, &payload); msg != "" {
		return badPayload(c, "payload tidak valid: "+msg)
	}
	if len(payload.Moves) > 1000 {
		return badPayload(c, "field moves maksimal 1000 item")
	}

	if len(payload.Moves) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tidak ada folder yang dipilih untuk dipindahkan"})
	}

	result, err := s.services.Organizer.ExecuteMoves(c.Context(), payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (s *Server) previewConsolidateProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	targetParentDir := c.Query("target_parent_dir")

	preview, err := s.services.Organizer.PreviewConsolidate(c.Context(), uint(id), targetParentDir)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(preview)
}

func (s *Server) consolidateProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	var payload models.ConsolidatePayload
	if msg := firstProblem(bindJSON(c, &payload), absPath("target_parent_dir", payload.TargetParentDir)); msg != "" {
		return badPayload(c, msg)
	}

	apiBase := c.BaseURL()

	result, err := s.services.Organizer.Consolidate(c.Context(), uint(id), payload.TargetParentDir, apiBase)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func (s *Server) getWorkspaceSettings(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "private, max-age=60")
	ws, err := s.services.Organizer.GetWorkspaceSettings(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(ws)
}

func (s *Server) updateWorkspaceSettings(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-cache")
	var payload struct {
		DefaultProjectsRoot string `json:"default_projects_root"`
	}
	if msg := firstProblem(bindJSONRequired(c, &payload), absPath("default_projects_root", payload.DefaultProjectsRoot)); msg != "" {
		return badPayload(c, "payload tidak valid: "+msg)
	}

	ws, err := s.services.Organizer.UpdateWorkspaceSettings(c.Context(), payload.DefaultProjectsRoot)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(ws)
}
