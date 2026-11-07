package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/analysis"
	"github.com/vibeasik/vibeasik/pkg/diagnostics"
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
	updates := map[string]interface{}{}
	if (comp.Language == "" || comp.Language == "auto") && result.DetectedParams.Language != "" {
		comp.Language = result.DetectedParams.Language
		updates["language"] = comp.Language
	}
	if comp.Framework == "" && result.DetectedParams.Framework != "" {
		comp.Framework = result.DetectedParams.Framework
		updates["framework"] = comp.Framework
	}
	if comp.PackageManager == "" && result.DetectedParams.PackageManager != "" {
		comp.PackageManager = result.DetectedParams.PackageManager
		updates["package_manager"] = comp.PackageManager
	}
	if comp.DevCommand == "" && result.DetectedParams.DevCommand != "" {
		comp.DevCommand = result.DetectedParams.DevCommand
		updates["dev_command"] = comp.DevCommand
	}
	if comp.Port == 0 && result.DetectedParams.Port > 0 {
		comp.Port = result.DetectedParams.Port
		updates["port"] = comp.Port
	}
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

func (s *Server) getLatestProjectDiagnostic(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	var diag models.Diagnostic
	if err := s.db.Where("project_id = ? AND (component_id IS NULL OR component_id = 0)", id).Order("created_at DESC").First(&diag).Error; err != nil {
		// If no diagnostic exists yet, run one automatically!
		report, err := s.services.Diagnostics.DiagnoseProject(c.Context(), uint(id))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		diag = *report
	}

	var findings []models.Finding
	_ = json.Unmarshal([]byte(diag.FindingsJSON), &findings)

	// Sync & merge with latest DeepCheckReport if one exists
	var deepReport models.DeepCheckReport
	if err := s.db.Where("project_id = ?", id).Order("created_at desc").First(&deepReport).Error; err == nil && deepReport.FindingsJSON != "" {
		var deepFindings []models.Finding
		if err := json.Unmarshal([]byte(deepReport.FindingsJSON), &deepFindings); err == nil && len(deepFindings) > 0 {
			findings = models.MergeFindings(findings, deepFindings)
		}
	}

	originalCount := len(findings)
	findings = models.DeduplicateFindings(findings)
	if len(findings) != originalCount || (deepReport.ID != 0 && len(findings) > originalCount) {
		if cleanData, err := json.Marshal(findings); err == nil {
			diag.FindingsJSON = string(cleanData)
			diag.Score = diagnostics.CalculateHealthScore(findings)
			_ = s.db.Model(&diag).Updates(map[string]interface{}{
				"findings_json": diag.FindingsJSON,
				"score":         diag.Score,
			}).Error
		}
	}

	var optimizations []models.Optimization
	_ = json.Unmarshal([]byte(diag.Optimizations), &optimizations)

	// Calculate category stats for active (unresolved, unignored) findings
	categoryCounts := map[string]int{
		"deep":         0,
		"query":        0,
		"router":       0,
		"code_smell":   0,
		"architecture": 0,
		"security":     0,
		"port":         0,
		"other":        0,
	}
	severityCounts := map[string]int{
		"CRITICAL": 0,
		"WARNING":  0,
		"INFO":     0,
	}

	activeCount := 0
	ignoredCount := 0
	resolvedCount := 0

	for i := range findings {
		if findings[i].Status == models.FindingStatusIgnored {
			ignoredCount++
		} else if findings[i].Status == models.FindingStatusResolved {
			resolvedCount++
		} else {
			activeCount++
			cat := strings.ToLower(findings[i].Category)
			if _, ok := categoryCounts[cat]; ok {
				categoryCounts[cat]++
			} else {
				categoryCounts["other"]++
			}

			if findings[i].IsDeep || strings.HasPrefix(findings[i].RuleID, "DEEP-") || strings.HasPrefix(findings[i].Title, "[DEEP]") {
				categoryCounts["deep"]++
			}

			sev := strings.ToUpper(findings[i].Severity)
			if _, ok := severityCounts[sev]; ok {
				severityCounts[sev]++
			}
		}
	}

	// findings_json di dalam "diagnostic" menggandakan "findings" (belasan MB pada
	// project dengan riwayat temuan panjang); kosongkan kecuali diminta ?full=1.
	respDiag := diag
	if f := c.Query("full"); f != "1" && f != "true" {
		respDiag.FindingsJSON = ""
	}

	return c.JSON(fiber.Map{
		"diagnostic":              respDiag,
		"findings":                findings,
		"optimizations":           optimizations,
		"category_counts":         categoryCounts,
		"severity_counts":         severityCounts,
		"total_findings":          len(findings),
		"active_findings_count":   activeCount,
		"ignored_findings_count":  ignoredCount,
		"resolved_findings_count": resolvedCount,
		"score":                   diag.Score,
		"summary":                 diag.Summary,
	})
}

func (s *Server) getLatestComponentDiagnostic(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}

	var diag models.Diagnostic
	if err := s.db.Where("component_id = ?", id).Order("created_at DESC").First(&diag).Error; err != nil {
		report, err := s.services.Diagnostics.DiagnoseComponent(c.Context(), uint(id))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		diag = *report
	}

	var findings []models.Finding
	_ = json.Unmarshal([]byte(diag.FindingsJSON), &findings)
	findings = models.DeduplicateFindings(findings)

	var optimizations []models.Optimization
	_ = json.Unmarshal([]byte(diag.Optimizations), &optimizations)

	return c.JSON(fiber.Map{
		"diagnostic":     diag,
		"findings":       findings,
		"optimizations":  optimizations,
		"total_findings": len(findings),
		"score":          diag.Score,
		"summary":        diag.Summary,
	})
}

type FindingActionPayload struct {
	FindingID  string          `json:"finding_id"`
	RuleID     string          `json:"rule_id"`
	Title      string          `json:"title"`
	TargetFile string          `json:"target_file"`
	Line       int             `json:"line"`
	Status     string          `json:"status"`
	Finding    *models.Finding `json:"finding,omitempty"`
}

// isMatchingFinding: f adalah temuan target (aturan pencocokan tunggal di models.AreFindingsMatching) atau
// kuncinya sama dengan kunci dari URL/payload.
func isMatchingFinding(f models.Finding, target models.Finding, targetKey string) bool {
	return (targetKey != "" && models.GenerateFindingKey(f) == targetKey) || models.AreFindingsMatching(f, target)
}

func syncFindingStatusInDeepCheck(db *gorm.DB, projectID uint, target models.Finding, newStatus string) {
	var deepReport models.DeepCheckReport
	if err := db.Where("project_id = ?", projectID).Order("created_at desc").First(&deepReport).Error; err != nil || deepReport.FindingsJSON == "" {
		return
	}

	var deepFindings []models.Finding
	if err := json.Unmarshal([]byte(deepReport.FindingsJSON), &deepFindings); err != nil {
		return
	}

	targetKey := target.ID
	if targetKey == "" {
		targetKey = models.GenerateFindingKey(target)
	}

	updated := false
	for i := range deepFindings {
		if isMatchingFinding(deepFindings[i], target, targetKey) {
			deepFindings[i].Status = newStatus
			if deepFindings[i].ID == "" {
				deepFindings[i].ID = targetKey
			}
			updated = true
			break
		}
	}

	if updated {
		newScore := diagnostics.CalculateHealthScore(deepFindings)
		deepReport.Score = newScore
		if data, err := json.Marshal(deepFindings); err == nil {
			deepReport.FindingsJSON = string(data)
			_ = db.Model(&deepReport).Updates(map[string]interface{}{
				"findings_json": deepReport.FindingsJSON,
				"score":         newScore,
			}).Error
			_ = db.Model(&models.Project{}).Where("id = ?", projectID).Update("last_deep_check_score", newScore).Error
		}
	}
}

func deleteFindingInDeepCheck(db *gorm.DB, projectID uint, target models.Finding, targetKey string) bool {
	var deepReport models.DeepCheckReport
	if err := db.Where("project_id = ?", projectID).Order("created_at desc").First(&deepReport).Error; err != nil || deepReport.FindingsJSON == "" {
		return false
	}

	var deepFindings []models.Finding
	if err := json.Unmarshal([]byte(deepReport.FindingsJSON), &deepFindings); err != nil {
		return false
	}

	filtered := make([]models.Finding, 0, len(deepFindings))
	deleted := false
	for _, f := range deepFindings {
		if !deleted && isMatchingFinding(f, target, targetKey) {
			deleted = true
			continue
		}
		filtered = append(filtered, f)
	}

	if deleted {
		newScore := diagnostics.CalculateHealthScore(filtered)
		deepReport.Score = newScore
		deepReport.TotalFindings = len(filtered)
		if data, err := json.Marshal(filtered); err == nil {
			deepReport.FindingsJSON = string(data)
			_ = db.Model(&deepReport).Updates(map[string]interface{}{
				"findings_json":  deepReport.FindingsJSON,
				"score":          newScore,
				"total_findings": len(filtered),
			}).Error
			_ = db.Model(&models.Project{}).Where("id = ?", projectID).Update("last_deep_check_score", newScore).Error
		}
	}
	return deleted
}

func updateFindingStatusInLatestDiag(db *gorm.DB, projectID uint, target models.Finding, newStatus string) (int, error) {
	// 1. Sync in DeepCheckReport if present
	syncFindingStatusInDeepCheck(db, projectID, target, newStatus)

	// 2. Sync in Diagnostic record
	var diag models.Diagnostic
	if err := db.Where("project_id = ?", projectID).Order("created_at DESC").First(&diag).Error; err != nil {
		return 0, err
	}

	var findings []models.Finding
	if err := json.Unmarshal([]byte(diag.FindingsJSON), &findings); err != nil {
		return diag.Score, err
	}

	findings = models.DeduplicateFindings(findings)

	updated := false
	targetKey := target.ID
	if targetKey == "" {
		targetKey = models.GenerateFindingKey(target)
	}

	for i := range findings {
		if isMatchingFinding(findings[i], target, targetKey) {
			findings[i].Status = newStatus
			if findings[i].ID == "" {
				findings[i].ID = targetKey
			}
			updated = true
			break
		}
	}

	// If not found in diag (e.g. was only in deep report), append it with newStatus
	if !updated {
		targetFinding := target
		targetFinding.Status = newStatus
		if targetFinding.ID == "" {
			targetFinding.ID = targetKey
		}
		findings = append(findings, targetFinding)
		updated = true
	}

	if updated {
		diag.Score = diagnostics.CalculateHealthScore(findings)
		data, err := json.Marshal(findings)
		if err == nil {
			diag.FindingsJSON = string(data)
			_ = db.Model(&diag).Updates(map[string]interface{}{
				"findings_json": diag.FindingsJSON,
				"score":         diag.Score,
			}).Error
			_ = db.Model(&models.Project{}).Where("id = ?", projectID).Update("health_score", diag.Score).Error
			return diag.Score, nil
		}
	}
	return diag.Score, nil
}

func (s *Server) deleteDiagnosticFinding(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	var payload FindingActionPayload
	_ = c.BodyParser(&payload)

	findingKey := c.Params("findingKey")
	if findingKey == "" {
		findingKey = payload.FindingID
	}

	var target models.Finding
	if payload.Finding != nil {
		target = *payload.Finding
	} else {
		target = models.Finding{
			ID:         findingKey,
			RuleID:     payload.RuleID,
			Title:      payload.Title,
			TargetFile: payload.TargetFile,
			Line:       payload.Line,
		}
	}

	targetKey := findingKey
	if targetKey == "" {
		targetKey = target.ID
	}
	if targetKey == "" {
		targetKey = models.GenerateFindingKey(target)
	}

	// Also delete from DeepCheckReport if present
	deletedFromDeep := deleteFindingInDeepCheck(s.db, uint(id), target, targetKey)

	var diag models.Diagnostic
	if err := s.db.Where("project_id = ?", id).Order("created_at DESC").First(&diag).Error; err != nil {
		if deletedFromDeep {
			return c.JSON(fiber.Map{
				"success":        true,
				"message":        "Temuan Deep Check berhasil dihapus",
				"findings":       []models.Finding{},
				"total_findings": 0,
				"score":          100,
			})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no diagnostic found for project"})
	}

	var findings []models.Finding
	if err := json.Unmarshal([]byte(diag.FindingsJSON), &findings); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to parse findings"})
	}

	findings = models.DeduplicateFindings(findings)

	filtered := make([]models.Finding, 0, len(findings))
	deleted := false

	for _, f := range findings {
		if !deleted && isMatchingFinding(f, target, targetKey) {
			deleted = true
			continue
		}
		filtered = append(filtered, f)
	}

	if !deleted && !deletedFromDeep {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "finding not found in diagnostic report"})
	}

	// Recalculate health score accurately
	diag.Score = diagnostics.CalculateHealthScore(filtered)

	data, err := json.Marshal(filtered)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to encode findings"})
	}
	diag.FindingsJSON = string(data)

	if err := s.db.Model(&diag).Updates(map[string]interface{}{
		"findings_json": diag.FindingsJSON,
		"score":         diag.Score,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save updated diagnostic"})
	}

	// Update project's cached health score
	_ = s.repos.Projects.UpdateHealthScore(c.Context(), uint(id), diag.Score)

	return c.JSON(fiber.Map{
		"success":        true,
		"message":        "Temuan audit berhasil dihapus",
		"findings":       filtered,
		"total_findings": len(filtered),
		"score":          diag.Score,
	})
}

func (s *Server) ignoreDiagnosticFinding(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	var payload FindingActionPayload
	_ = c.BodyParser(&payload)

	findingKey := c.Params("findingKey")
	if findingKey == "" {
		findingKey = payload.FindingID
	}

	var target models.Finding
	if payload.Finding != nil {
		target = *payload.Finding
	} else {
		target = models.Finding{
			ID:         findingKey,
			RuleID:     payload.RuleID,
			Title:      payload.Title,
			TargetFile: payload.TargetFile,
			Line:       payload.Line,
		}
	}

	var diag models.Diagnostic
	if err := s.db.Where("project_id = ?", id).Order("created_at DESC").First(&diag).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no diagnostic found for project"})
	}

	var findings []models.Finding
	if err := json.Unmarshal([]byte(diag.FindingsJSON), &findings); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to parse findings"})
	}

	findings = models.DeduplicateFindings(findings)

	targetKey := findingKey
	if targetKey == "" {
		targetKey = target.ID
	}
	if targetKey == "" {
		targetKey = models.GenerateFindingKey(target)
	}

	found := false
	var updatedStatus string
	for i := range findings {
		if isMatchingFinding(findings[i], target, targetKey) {
			found = true
			if payload.Status != "" {
				findings[i].Status = payload.Status
			} else if findings[i].Status == models.FindingStatusIgnored {
				findings[i].Status = models.FindingStatusOpen
			} else {
				findings[i].Status = models.FindingStatusIgnored
			}
			if findings[i].ID == "" {
				findings[i].ID = models.GenerateFindingKey(findings[i])
			}
			updatedStatus = findings[i].Status
			break
		}
	}

	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "finding not found in diagnostic report"})
	}

	// Recalculate health score accurately based on active findings
	diag.Score = diagnostics.CalculateHealthScore(findings)

	data, err := json.Marshal(findings)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to encode findings"})
	}
	diag.FindingsJSON = string(data)

	if err := s.db.Model(&diag).Updates(map[string]interface{}{
		"findings_json": diag.FindingsJSON,
		"score":         diag.Score,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save updated diagnostic"})
	}

	_ = s.repos.Projects.UpdateHealthScore(c.Context(), uint(id), diag.Score)

	msg := "Temuan audit berhasil diabaikan"
	if updatedStatus == models.FindingStatusOpen {
		msg = "Status temuan audit dikembalikan ke aktif (open)"
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"message":  msg,
		"status":   updatedStatus,
		"findings": findings,
		"score":    diag.Score,
	})
}

const (
	defaultDiagnosticsLimit = 100
	maxDiagnosticsLimit     = 500
)

// listProjectDiagnostics mengembalikan riwayat diagnostik project secara berpaginasi
// (terbaru dulu). Query: limit (default 100, maks 500), offset, component_id,
// full=1 untuk menyertakan findings_json/optimizations/structure_tree (default: ringkas).
func (s *Server) listProjectDiagnostics(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	limit := c.QueryInt("limit", defaultDiagnosticsLimit)
	if limit <= 0 {
		limit = defaultDiagnosticsLimit
	}
	if limit > maxDiagnosticsLimit {
		limit = maxDiagnosticsLimit
	}
	offset := c.QueryInt("offset", 0)
	if offset < 0 {
		offset = 0
	}

	q := s.db.Model(&models.Diagnostic{}).Where("project_id = ?", id)
	if cid := c.QueryInt("component_id", 0); cid > 0 {
		q = q.Where("component_id = ?", cid)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if f := c.Query("full"); f != "1" && f != "true" {
		q = q.Select("id", "project_id", "component_id", "summary", "score", "created_at")
	}
	items := []models.Diagnostic{}
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"items": items, "total": total, "limit": limit, "offset": offset})
}
