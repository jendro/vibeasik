package api

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/paging"
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
	// Bentuk array tetap; ?limit=&offset= opsional, total di header X-Total-Count.
	return c.JSON(paging.Apply(c, projects))
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
	if denied, resp := validateCreateProject(c, &p); denied {
		return resp
	}

	provisionProjectDefaults(&p, payload.CreateDirectories)

	if err := s.repos.Projects.Create(c.Context(), &p); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Modul Pro: dewan eksekutif & pipeline latar (fast scan LLM → ringkasan → rute).
	registry.ProjectCreated(s.db, &p)

	return c.Status(fiber.StatusCreated).JSON(p)
}

func validateCreateProject(c *fiber.Ctx, p *models.Project) (bool, error) {
	if msg := firstProblem(
		needText("name", p.Name, 255),
		maxLen("slug", p.Slug, 255),
		maxLen("root_path", p.RootPath, 1024),
		maxLen("short_description", p.ShortDescription, maxText),
		maxLen("description", p.Description, maxText),
	); msg != "" {
		return true, badPayload(c, msg)
	}

	devCommands := make([]string, 0, len(p.Components))
	for _, comp := range p.Components {
		devCommands = append(devCommands, comp.DevCommand)
	}
	if denied, resp := guardDevCommand(c, "", devCommands...); denied {
		return true, resp
	}
	return false, nil
}

func provisionProjectDefaults(p *models.Project, createDirectories bool) {
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
	if createDirectories && p.RootPath != "" && p.RootPath != "-" {
		_ = os.MkdirAll(p.RootPath, 0755)
	}
	for i := range p.Components {
		provisionComponentDefaults(&p.Components[i], p.RootPath, createDirectories)
	}
}

func provisionComponentDefaults(comp *models.Component, rootPath string, createDirectories bool) {
	if comp.RelativePath != "" && comp.AbsolutePath == "" {
		if rootPath != "" && rootPath != "-" {
			comp.AbsolutePath = filepath.Join(rootPath, comp.RelativePath)
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
		comp.Language = defaultComponentLanguage(comp.Role)
	}
	if createDirectories && comp.AbsolutePath != "" {
		_ = os.MkdirAll(comp.AbsolutePath, 0755)
	}
}

func defaultComponentLanguage(role string) string {
	if role == models.RoleBackend {
		return "go"
	}
	if strings.Contains(role, "mobile") {
		return "dart"
	}
	if strings.Contains(role, "front") {
		return "typescript"
	}
	return "auto"
}

func (s *Server) getProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
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
	if err != nil || id <= 0 {
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
	if err != nil || id <= 0 {
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
	if err != nil || id <= 0 {
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
	if err != nil || id <= 0 {
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
	if err != nil || id <= 0 {
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

func isFrontendRole(role string) bool {
	return role == models.RoleFrontendAdmin || role == models.RoleFrontendClient || role == models.RoleFrontendMobile
}

func determineMergedProjectType(comps []models.Component) string {
	hasBackend := false
	feCount := 0
	for _, comp := range comps {
		if comp.Role == models.RoleBackend {
			hasBackend = true
		}
		if isFrontendRole(comp.Role) {
			feCount++
		}
	}
	if hasBackend && feCount > 1 {
		return models.TypeMultiFE
	}
	if hasBackend && feCount >= 1 {
		return models.TypeBundle
	}
	return ""
}

func (s *Server) transferMergeData(sourceID, targetID uint) error {
	// 1. Move all components from source project to target project
	if err := s.db.Model(&models.Component{}).Where("project_id = ?", sourceID).Update("project_id", targetID).Error; err != nil {
		return err
	}
	// 2. Transfer todos from source project to target project
	_ = s.db.Model(&models.Todo{}).Where("project_id = ?", sourceID).Update("project_id", targetID)
	// 3. Release source project from monitoring
	return s.db.Delete(&models.Project{}, sourceID).Error
}

func (s *Server) updateMergedProjectTopology(targetID uint) {
	var targetComps []models.Component
	s.db.Where("project_id = ?", targetID).Find(&targetComps)
	if newType := determineMergedProjectType(targetComps); newType != "" {
		_ = s.db.Model(&models.Project{}).Where("id = ?", targetID).Update("project_type", newType)
	}
}

func (s *Server) mergeProject(c *fiber.Ctx) error {
	sourceID, err := c.ParamsInt("id")
	if err != nil || sourceID <= 0 {
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

	if err := s.transferMergeData(uint(sourceID), payload.TargetProjectID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to merge project data: " + err.Error()})
	}

	s.updateMergedProjectTopology(payload.TargetProjectID)

	// Regenerate AI summary for target project based on all combined applications
	fullDesc, _ := registry.ProjectSummary(s.db, payload.TargetProjectID)

	// Fetch final updated target project with its newly merged components
	var finalTarget models.Project
	_ = s.db.Preload("Components").First(&finalTarget, payload.TargetProjectID)

	return c.JSON(fiber.Map{
		"message":        "Proyek berhasil digabungkan",
		"target_project": finalTarget,
		"description":    fullDesc,
	})
}
