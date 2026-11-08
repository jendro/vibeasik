package api

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/diagnostics"
	"github.com/vibeasik/vibeasik/pkg/models"
)

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
