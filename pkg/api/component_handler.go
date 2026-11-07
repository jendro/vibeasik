package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
)

type AddComponentPayload struct {
	models.Component
	CreateFolder bool `json:"create_folder"`
}

func (s *Server) addComponent(c *fiber.Ctx) error {
	projectID, err := c.ParamsInt("id")
	if err != nil || projectID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	proj, err := s.repos.Projects.FindByID(c.Context(), uint(projectID))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var payload AddComponentPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	comp := payload.Component
	comp.ProjectID = uint(projectID)
	if denied, resp := guardDevCommand(c, "", comp.DevCommand); denied {
		return resp
	}

	if comp.RelativePath != "" && comp.AbsolutePath == "" {
		comp.AbsolutePath = filepath.Join(proj.RootPath, comp.RelativePath)
	} else if comp.AbsolutePath != "" && comp.RelativePath == "" {
		rel, err := filepath.Rel(proj.RootPath, comp.AbsolutePath)
		if err == nil && !strings.HasPrefix(rel, "..") {
			comp.RelativePath = rel
		} else {
			comp.RelativePath = filepath.Base(comp.AbsolutePath)
		}
	}

	if comp.Name == "" {
		if comp.RelativePath != "" {
			comp.Name = comp.RelativePath
		} else {
			comp.Name = "Folder " + comp.Role
		}
	}

	if comp.Role == "" {
		comp.Role = models.RoleOther
	}

	if comp.Language == "" {
		if comp.Role == models.RoleBackend {
			comp.Language = "go"
		} else if strings.Contains(comp.Role, "mobile") {
			comp.Language = "dart"
		} else if strings.Contains(comp.Role, "front") {
			comp.Language = "typescript"
		} else {
			comp.Language = "auto"
		}
	}

	if payload.CreateFolder && comp.AbsolutePath != "" {
		_ = os.MkdirAll(comp.AbsolutePath, 0755)
	}

	if err := s.repos.Components.Create(c.Context(), &comp); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	registry.ProjectChanged(comp.ProjectID, proj.RootPath)
	return c.Status(fiber.StatusCreated).JSON(comp)
}

func (s *Server) getComponent(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	comp, err := s.services.Components.GetByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "component not found"})
	}
	return c.JSON(comp)
}

func (s *Server) updateComponent(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	comp, err := s.repos.Components.FindByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "component not found"})
	}

	var updateData models.Component
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if denied, resp := guardDevCommand(c, comp.DevCommand, updateData.DevCommand); denied {
		return resp
	}

	comp.Name = updateData.Name
	comp.Role = updateData.Role
	comp.RelativePath = updateData.RelativePath
	comp.AbsolutePath = updateData.AbsolutePath
	if comp.AbsolutePath == "" && comp.RelativePath != "" {
		if proj, err := s.repos.Projects.FindByID(c.Context(), comp.ProjectID); err == nil {
			comp.AbsolutePath = filepath.Join(proj.RootPath, comp.RelativePath)
		}
	}
	comp.Language = updateData.Language
	comp.Framework = updateData.Framework
	comp.PackageManager = updateData.PackageManager
	comp.DevCommand = updateData.DevCommand
	comp.BuildCommand = updateData.BuildCommand
	comp.Port = updateData.Port
	comp.HealthURL = updateData.HealthURL
	comp.EnvFile = updateData.EnvFile
	comp.EnvSummary = updateData.EnvSummary

	if err := s.repos.Components.Update(c.Context(), comp); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if proj, err := s.repos.Projects.FindByID(c.Context(), comp.ProjectID); err == nil && proj != nil {
		registry.ProjectChanged(comp.ProjectID, proj.RootPath)
	} else {
		registry.ProjectChanged(comp.ProjectID, "")
	}

	return c.JSON(comp)
}

func (s *Server) deleteComponent(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	comp, _ := s.repos.Components.FindByID(c.Context(), uint(id))
	if err := s.services.Components.Delete(c.Context(), uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if comp != nil {
		registry.ProjectChanged(comp.ProjectID, "")
	}
	return c.JSON(fiber.Map{"message": "component deleted"})
}

func (s *Server) createComponentDir(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	path, err := s.services.Components.CreateDir(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("failed to create directory: %v", err)})
	}

	return c.JSON(fiber.Map{"message": "directory created successfully", "path": path})
}
