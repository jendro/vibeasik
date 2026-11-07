package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/analysis"
	"github.com/vibeasik/vibeasik/pkg/models"
)

func (s *Server) getComponentRoutes(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	comp, err := s.repos.Components.FindByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "component not found"})
	}

	var routes []models.ComponentRoute
	s.db.Where("component_id = ?", comp.ID).Order("domain asc, path asc").Find(&routes)

	// If empty, auto-sync from code or FastScanData
	if len(routes) == 0 {
		proj, _ := s.repos.Projects.FindByID(c.Context(), comp.ProjectID)
		syncRes, err := analysis.SyncComponentRoutes(s.db, comp, proj)
		if err == nil && len(syncRes.Routes) > 0 {
			return c.JSON(syncRes.Routes)
		}
	}

	return c.JSON(routes)
}

func (s *Server) syncComponentRoutes(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	result, err := s.services.Routes.SyncRoutes(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func (s *Server) updateComponentRoute(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	routeId, err := c.ParamsInt("routeId")
	if err != nil || routeId <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid route id"})
	}

	var body struct {
		Description    string `json:"description"`
		Domain         string `json:"domain"`
		FunctionGroup  string `json:"function_group"`
		Tags           string `json:"tags"`
		TargetRequests string `json:"target_requests"`
	}
	if msg := firstProblem(bindJSONRequired(c, &body), maxLen("description", body.Description, maxText),
		maxLen("domain", body.Domain, maxName), maxLen("function_group", body.FunctionGroup, maxName),
		maxLen("tags", body.Tags, maxLabel), maxLen("target_requests", body.TargetRequests, maxName)); msg != "" {
		return badPayload(c, msg)
	}

	var r models.ComponentRoute
	if err := s.db.Where("id = ? AND component_id = ?", routeId, id).First(&r).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "route not found"})
	}

	if body.Description != "" {
		r.Description = body.Description
	}
	if body.Domain != "" {
		r.Domain = body.Domain
	}
	if body.FunctionGroup != "" {
		r.FunctionGroup = body.FunctionGroup
	}
	if body.Tags != "" {
		r.Tags = body.Tags
	}
	if body.TargetRequests != "" {
		r.TargetRequests = body.TargetRequests
	}

	if err := s.repos.Routes.Update(c.Context(), &r); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(r)
}

func (s *Server) deleteComponentRoute(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	routeId, err := c.ParamsInt("routeId")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid route id"})
	}

	if err := s.db.Where("id = ? AND component_id = ?", routeId, id).Delete(&models.ComponentRoute{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "route deleted"})
}
