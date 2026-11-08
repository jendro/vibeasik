package api

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/diagnostics"
	"github.com/vibeasik/vibeasik/pkg/models"
)

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

func resolveFindingTarget(c *fiber.Ctx) (FindingActionPayload, models.Finding, string) {
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
	return payload, target, targetKey
}

func (s *Server) loadDiagnosticFindings(projectID int) (*models.Diagnostic, []models.Finding, error) {
	var diag models.Diagnostic
	if err := s.db.Where("project_id = ?", projectID).Order("created_at DESC").First(&diag).Error; err != nil {
		return nil, nil, err
	}
	var findings []models.Finding
	if err := json.Unmarshal([]byte(diag.FindingsJSON), &findings); err != nil {
		return &diag, nil, err
	}
	return &diag, models.DeduplicateFindings(findings), nil
}

func (s *Server) saveUpdatedDiagnostic(ctx context.Context, diag *models.Diagnostic, findings []models.Finding, projectID uint) error {
	diag.Score = diagnostics.CalculateHealthScore(findings)
	data, err := json.Marshal(findings)
	if err != nil {
		return err
	}
	diag.FindingsJSON = string(data)
	if err := s.db.Model(diag).Updates(map[string]interface{}{
		"findings_json": diag.FindingsJSON,
		"score":         diag.Score,
	}).Error; err != nil {
		return err
	}
	_ = s.repos.Projects.UpdateHealthScore(ctx, projectID, diag.Score)
	return nil
}

func filterDeletedFinding(findings []models.Finding, target models.Finding, targetKey string) ([]models.Finding, bool) {
	filtered := make([]models.Finding, 0, len(findings))
	deleted := false
	for _, f := range findings {
		if !deleted && isMatchingFinding(f, target, targetKey) {
			deleted = true
			continue
		}
		filtered = append(filtered, f)
	}
	return filtered, deleted
}

func updateFindingStatus(findings []models.Finding, target models.Finding, targetKey string, requestedStatus string) (string, bool) {
	for i := range findings {
		if isMatchingFinding(findings[i], target, targetKey) {
			if requestedStatus != "" {
				findings[i].Status = requestedStatus
			} else if findings[i].Status == models.FindingStatusIgnored {
				findings[i].Status = models.FindingStatusOpen
			} else {
				findings[i].Status = models.FindingStatusIgnored
			}
			if findings[i].ID == "" {
				findings[i].ID = models.GenerateFindingKey(findings[i])
			}
			return findings[i].Status, true
		}
	}
	return "", false
}

func (s *Server) deleteDiagnosticFinding(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}

	_, target, targetKey := resolveFindingTarget(c)
	deletedFromDeep := deleteFindingInDeepCheck(s.db, uint(id), target, targetKey)

	diag, findings, err := s.loadDiagnosticFindings(id)
	if err != nil {
		if deletedFromDeep {
			return c.JSON(fiber.Map{
				"success":        true,
				"message":        "Temuan Deep Check berhasil dihapus",
				"findings":       []models.Finding{},
				"total_findings": 0,
				"score":          100,
			})
		}
		if diag == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no diagnostic found for project"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to parse findings"})
	}

	filtered, deleted := filterDeletedFinding(findings, target, targetKey)
	if !deleted && !deletedFromDeep {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "finding not found in diagnostic report"})
	}

	if err := s.saveUpdatedDiagnostic(c.Context(), diag, filtered, uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save updated diagnostic"})
	}

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

	payload, target, targetKey := resolveFindingTarget(c)

	diag, findings, err := s.loadDiagnosticFindings(id)
	if err != nil {
		if diag == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no diagnostic found for project"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to parse findings"})
	}

	updatedStatus, found := updateFindingStatus(findings, target, targetKey, payload.Status)
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "finding not found in diagnostic report"})
	}

	if err := s.saveUpdatedDiagnostic(c.Context(), diag, findings, uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save updated diagnostic"})
	}

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
