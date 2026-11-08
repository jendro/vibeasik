package api

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/diagnostics"
	"github.com/vibeasik/vibeasik/pkg/models"
)

type findingStats struct {
	categoryCounts map[string]int
	severityCounts map[string]int
	activeCount    int
	ignoredCount   int
	resolvedCount  int
}

func (s *Server) getOrCreateProjectDiagnostic(c *fiber.Ctx, id int) (models.Diagnostic, error) {
	var diag models.Diagnostic
	if err := s.db.Where("project_id = ? AND (component_id IS NULL OR component_id = 0)", id).Order("created_at DESC").First(&diag).Error; err != nil {
		report, err := s.services.Diagnostics.DiagnoseProject(c.Context(), uint(id))
		if err != nil {
			return diag, err
		}
		diag = *report
	}
	return diag, nil
}

func (s *Server) syncProjectDeepCheckFindings(diag *models.Diagnostic, id int) []models.Finding {
	var findings []models.Finding
	_ = json.Unmarshal([]byte(diag.FindingsJSON), &findings)

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
			_ = s.db.Model(diag).Updates(map[string]interface{}{
				"findings_json": diag.FindingsJSON,
				"score":         diag.Score,
			}).Error
		}
	}
	return findings
}

func computeFindingStats(findings []models.Finding) findingStats {
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

	activeCount, ignoredCount, resolvedCount := 0, 0, 0
	for i := range findings {
		switch findings[i].Status {
		case models.FindingStatusIgnored:
			ignoredCount++
		case models.FindingStatusResolved:
			resolvedCount++
		default:
			activeCount++
			tallyActiveFinding(&findings[i], categoryCounts, severityCounts)
		}
	}

	return findingStats{
		categoryCounts: categoryCounts,
		severityCounts: severityCounts,
		activeCount:    activeCount,
		ignoredCount:   ignoredCount,
		resolvedCount:  resolvedCount,
	}
}

func tallyActiveFinding(f *models.Finding, categoryCounts, severityCounts map[string]int) {
	cat := strings.ToLower(f.Category)
	if _, ok := categoryCounts[cat]; ok {
		categoryCounts[cat]++
	} else {
		categoryCounts["other"]++
	}

	if f.IsDeep || strings.HasPrefix(f.RuleID, "DEEP-") || strings.HasPrefix(f.Title, "[DEEP]") {
		categoryCounts["deep"]++
	}

	sev := strings.ToUpper(f.Severity)
	if _, ok := severityCounts[sev]; ok {
		severityCounts[sev]++
	}
}

func (s *Server) getLatestProjectDiagnostic(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	diag, err := s.getOrCreateProjectDiagnostic(c, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	findings := s.syncProjectDeepCheckFindings(&diag, id)

	var optimizations []models.Optimization
	_ = json.Unmarshal([]byte(diag.Optimizations), &optimizations)

	stats := computeFindingStats(findings)

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
		"category_counts":         stats.categoryCounts,
		"severity_counts":         stats.severityCounts,
		"total_findings":          len(findings),
		"active_findings_count":   stats.activeCount,
		"ignored_findings_count":  stats.ignoredCount,
		"resolved_findings_count": stats.resolvedCount,
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
