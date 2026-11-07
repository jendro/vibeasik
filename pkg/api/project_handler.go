package api

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/service"
	"github.com/vibeasik/vibeasik/pkg/workspace"
)

func (s *Server) getProjects(c *fiber.Ctx) error {
	projectType := c.Query("type")
	status := c.Query("status")

	projects, err := s.services.Projects.GetAll(c.Context(), projectType, status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(projects)
}

type CreateProjectPayload struct {
	models.Project
	CreateDirectories bool `json:"create_directories"`
}

func (s *Server) createProject(c *fiber.Ctx) error {
	var payload CreateProjectPayload
	if p := bindJSONRequired(c, &payload); p != "" {
		return badPayload(c, p)
	}
	p := payload.Project
	if msg := firstProblem(needText("name", p.Name, 255), maxLen("slug", p.Slug, 255),
		maxLen("root_path", p.RootPath, 1024), maxLen("short_description", p.ShortDescription, maxText),
		maxLen("description", p.Description, maxText)); msg != "" {
		return badPayload(c, msg)
	}

	devCommands := make([]string, 0, len(p.Components))
	for _, comp := range p.Components {
		devCommands = append(devCommands, comp.DevCommand)
	}
	if denied, resp := guardDevCommand(c, "", devCommands...); denied {
		return resp
	}

	if p.Slug == "" {
		p.Slug = service.Slugify(p.Name)
	}

	if p.RootPath == "" {
		if len(p.Components) > 0 && p.Components[0].AbsolutePath != "" {
			p.RootPath = p.Components[0].AbsolutePath
		} else {
			p.RootPath = "-"
		}
	}

	if payload.CreateDirectories && p.RootPath != "" && p.RootPath != "-" {
		_ = os.MkdirAll(p.RootPath, 0755)
	}

	for i := range p.Components {
		comp := &p.Components[i]
		if comp.RelativePath != "" && comp.AbsolutePath == "" {
			if p.RootPath != "" && p.RootPath != "-" {
				comp.AbsolutePath = filepath.Join(p.RootPath, comp.RelativePath)
			} else {
				comp.AbsolutePath = comp.RelativePath
			}
		}
		if comp.AbsolutePath != "" && comp.RelativePath == "" {
			comp.RelativePath = filepath.Base(comp.AbsolutePath)
		}
		if comp.Role == "" {
			comp.Role = models.RoleOther
		}
		if comp.Name == "" {
			if comp.RelativePath != "" {
				comp.Name = comp.RelativePath
			} else {
				comp.Name = "App " + comp.Role
			}
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
		if payload.CreateDirectories && comp.AbsolutePath != "" {
			_ = os.MkdirAll(comp.AbsolutePath, 0755)
		}
	}

	if err := s.repos.Projects.Create(c.Context(), &p); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Modul Pro: dewan eksekutif & pipeline latar (fast scan LLM → ringkasan → rute).
	registry.ProjectCreated(s.db, &p)

	return c.Status(fiber.StatusCreated).JSON(p)
}

func (s *Server) getProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	p, err := s.services.Projects.GetByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}
	return c.JSON(p)
}

func (s *Server) updateProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	var rawMap map[string]interface{}
	_ = c.BodyParser(&rawMap)

	var updateData models.Project
	if msg := bindJSONRequired(c, &updateData); msg != "" {
		return badPayload(c, msg)
	}
	if msg := firstProblem(maxLen("name", updateData.Name, 255), maxLen("slug", updateData.Slug, 255),
		maxLen("root_path", updateData.RootPath, 1024), maxLen("description", updateData.Description, maxText)); msg != "" {
		return badPayload(c, msg)
	}

	updated, err := s.services.Projects.Update(c.Context(), uint(id), &updateData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if autoVal, ok := rawMap["autonomous"].(bool); ok {
		_ = s.repos.Projects.UpdateFields(c.Context(), uint(id), map[string]interface{}{"autonomous": autoVal})
		updated.Autonomous = autoVal
	}

	registry.ProjectChanged(updated.ID, updated.RootPath)
	return c.JSON(updated)
}

func (s *Server) deleteProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	if err := s.services.Projects.Delete(c.Context(), uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	registry.ProjectChanged(uint(id), "")
	return c.JSON(fiber.Map{"message": "project deleted successfully"})
}

func (s *Server) generateProjectSummary(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	fullDesc, err := registry.ProjectSummary(s.db, uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var p models.Project
	if err := s.db.Preload("Components").First(&p, uint(id)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":           "project summary generated successfully",
		"short_description": p.ShortDescription,
		"description":       fullDesc,
		"project":           p,
	})
}

type LinkWorkspacePayload struct {
	ComponentID uint `json:"component_id"`
}

func (s *Server) linkProjectWorkspace(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	var payload LinkWorkspacePayload
	_ = c.BodyParser(&payload)

	apiBase := c.BaseURL()

	createdFiles, idf, err := workspace.LinkProjectWorkspaces(s.db, uint(id), payload.ComponentID, apiBase)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":       "Workspace linked successfully with AI Agent identifier (.vibeasik.json)",
		"project_id":    id,
		"files_created": createdFiles,
		"identifier":    idf,
	})
}

// initProject = "Init vibeasik": tautan AI (.vibeasik.json) + peta kode di repo yang diperbarui otomatis.
func (s *Server) initProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	res, err := workspace.InitProject(s.db, uint(id), c.BaseURL())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

func (s *Server) identifyProject(c *fiber.Ctx) error {
	targetPath := c.Query("path")
	apiBase := c.BaseURL()

	result, err := workspace.IdentifyWorkspace(s.db, targetPath, apiBase)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

type MergeProjectPayload struct {
	TargetProjectID uint `json:"target_project_id"`
}

func (s *Server) mergeProject(c *fiber.Ctx) error {
	sourceID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	var payload MergeProjectPayload
	if msg := bindJSONRequired(c, &payload); msg != "" || payload.TargetProjectID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": firstProblem(msg, "target_project_id is required")})
	}

	if uint(sourceID) == payload.TargetProjectID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot merge project into itself"})
	}

	var source models.Project
	if err := s.db.Preload("Components").First(&source, uint(sourceID)).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "source project not found"})
	}

	var target models.Project
	if err := s.db.Preload("Components").First(&target, payload.TargetProjectID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "target project not found"})
	}

	// 1. Move all components from source project to target project
	if err := s.db.Model(&models.Component{}).Where("project_id = ?", uint(sourceID)).Update("project_id", payload.TargetProjectID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to transfer components: " + err.Error()})
	}

	// 2. Transfer todos from source project to target project
	_ = s.db.Model(&models.Todo{}).Where("project_id = ?", uint(sourceID)).Update("project_id", payload.TargetProjectID)

	// 3. Release source project from monitoring (delete project record from database, files on disk stay untouched)
	if err := s.db.Delete(&models.Project{}, uint(sourceID)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to release source project: " + err.Error()})
	}

	// 4. Re-evaluate target project topology based on updated components
	var targetComps []models.Component
	s.db.Where("project_id = ?", payload.TargetProjectID).Find(&targetComps)
	hasBackend := false
	feCount := 0
	for _, comp := range targetComps {
		if comp.Role == models.RoleBackend {
			hasBackend = true
		}
		if comp.Role == models.RoleFrontendAdmin || comp.Role == models.RoleFrontendClient || comp.Role == models.RoleFrontendMobile {
			feCount++
		}
	}
	if hasBackend && feCount > 1 {
		target.ProjectType = models.TypeMultiFE
	} else if hasBackend && feCount >= 1 {
		target.ProjectType = models.TypeBundle
	}
	_ = s.db.Model(&models.Project{}).Where("id = ?", payload.TargetProjectID).Update("project_type", target.ProjectType)

	// 5. Regenerate AI summary for target project based on all combined applications
	fullDesc, _ := registry.ProjectSummary(s.db, payload.TargetProjectID)

	// 6. Fetch final updated target project with its newly merged components
	var finalTarget models.Project
	_ = s.db.Preload("Components").First(&finalTarget, payload.TargetProjectID)

	return c.JSON(fiber.Map{
		"message":        "Proyek berhasil digabungkan",
		"target_project": finalTarget,
		"description":    fullDesc,
	})
}
