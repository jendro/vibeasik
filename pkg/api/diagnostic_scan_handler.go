package api

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/analysis"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/scanner"
	"github.com/vibeasik/vibeasik/pkg/scanner/adapters"
)

func (s *Server) listAdapters(c *fiber.Ctx) error {
	return c.JSON(adapters.GetRegisteredAdaptersMetadata())
}

func (s *Server) scanPath(c *fiber.Ctx) error {
	var body struct {
		Path string `json:"path"`
	}
	if err := c.BodyParser(&body); err != nil || body.Path == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "path is required"})
	}

	result, err := scanner.ScanDirectory(body.Path)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func (s *Server) groupScanComponents(c *fiber.Ctx) error {
	var body models.GroupScanRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body: " + err.Error()})
	}

	existingProjects, err := s.services.Projects.GetAll(c.Context(), "", "")
	if err != nil {
		existingProjects = []models.Project{}
	}

	result, err := analysis.GroupScanComponents(s.db, body.RootPath, body.Components, existingProjects)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func (s *Server) diagnoseProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	report, err := s.services.Diagnostics.DiagnoseProject(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(report)
}

func (s *Server) diagnoseComponent(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	report, err := s.services.Diagnostics.DiagnoseComponent(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(report)
}

func applyDetectedString(val *string, detected string, key string, updates map[string]interface{}, allowAuto bool) {
	if (*val == "" || (allowAuto && *val == "auto")) && detected != "" {
		*val = detected
		updates[key] = *val
	}
}

func applyDetectedPort(val *int, detected int, key string, updates map[string]interface{}) {
	if *val == 0 && detected > 0 {
		*val = detected
		updates[key] = *val
	}
}

func buildDetectedParamUpdates(comp *models.Component, detected models.DetectedParams) map[string]interface{} {
	updates := map[string]interface{}{}
	applyDetectedString(&comp.Language, detected.Language, "language", updates, true)
	applyDetectedString(&comp.Framework, detected.Framework, "framework", updates, false)
	applyDetectedString(&comp.PackageManager, detected.PackageManager, "package_manager", updates, false)
	applyDetectedString(&comp.DevCommand, detected.DevCommand, "dev_command", updates, false)
	applyDetectedPort(&comp.Port, detected.Port, "port", updates)
	return updates
}

func (s *Server) fastScanComponent(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	comp, err := s.repos.Components.FindByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "component not found"})
	}

	proj, err := s.repos.Projects.FindByID(c.Context(), comp.ProjectID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	result, err := analysis.FastScanComponent(s.db, comp, proj)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Auto-sync routes to DB so pages and requests are persisted immediately
	_, _ = analysis.SyncComponentRoutes(s.db, comp, proj)

	// Auto-apply detected params to component in DB if empty or default
	updates := buildDetectedParamUpdates(comp, result.DetectedParams)
	if len(updates) > 0 {
		_ = s.repos.Components.UpdateFields(c.Context(), comp.ID, updates)
	}

	return c.JSON(result)
}

func (s *Server) updateComponentFastScan(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	comp, err := s.repos.Components.FindByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "component not found"})
	}

	var payload models.FastScanResult
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	payload.ScannedAt = time.Now()
	dataBytes, _ := json.Marshal(payload)
	comp.FastScanData = string(dataBytes)

	if err := s.repos.Components.UpdateFastScanData(c.Context(), comp.ID, comp.FastScanData); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	registry.ProjectChanged(comp.ProjectID, "")
	return c.JSON(payload)
}

type ApplyScanParamsPayload struct {
	DevCommand     string `json:"dev_command"`
	BuildCommand   string `json:"build_command"`
	Port           int    `json:"port"`
	Language       string `json:"language"`
	Framework      string `json:"framework"`
	PackageManager string `json:"package_manager"`
}

func (s *Server) applyFastScanParams(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	var payload ApplyScanParamsPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	current := ""
	if comp, err := s.repos.Components.FindByID(c.Context(), uint(id)); err == nil && comp != nil {
		current = comp.DevCommand
	}
	if denied, resp := guardDevCommand(c, current, payload.DevCommand); denied {
		return resp
	}

	updated, err := s.services.Components.ApplyScanParams(c.Context(), uint(id), models.DetectedParams{
		DevCommand:     payload.DevCommand,
		BuildCommand:   payload.BuildCommand,
		Port:           payload.Port,
		Language:       payload.Language,
		Framework:      payload.Framework,
		PackageManager: payload.PackageManager,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	registry.ProjectChanged(updated.ProjectID, "")
	return c.JSON(updated)
}

func (s *Server) fastScanProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	proj, err := s.repos.Projects.FindByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var results []*models.FastScanResult
	for i := range proj.Components {
		comp := &proj.Components[i]
		res, err := analysis.FastScanComponent(s.db, comp, proj)
		if err == nil && res != nil {
			results = append(results, res)
		}
	}

	return c.JSON(fiber.Map{
		"project_id":   proj.ID,
		"project_name": proj.Name,
		"components":   results,
		"scanned_at":   time.Now(),
	})
}
