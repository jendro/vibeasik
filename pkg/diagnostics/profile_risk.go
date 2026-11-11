package diagnostics

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// CategoryRisk: temuan audit yang berasal dari risiko profil proyek (disusun saat analisis profil).
const CategoryRisk = "risk"

// riskFilePattern mengenali path berkas kode di teks risiko (mis. "pkg/api/auth.go", "pages/index.vue").
var riskFilePattern = regexp.MustCompile(`[\w.-]+(?:/[\w.-]+)*\.(?:go|vue|ts|tsx|js|jsx|mjs|py|dart|php|rb|java|kt|swift|rs|sql|cs)\b`)

// RiskTargetFile mengembalikan berkas kode pertama yang disebut risiko, atau "" bila risikonya belum konkret.
func RiskTargetFile(r models.ProfileRisk) string {
	return riskFilePattern.FindString(r.Title + " " + r.Detail)
}

// RiskFindingID: kunci stabil temuan risiko (dari judulnya) agar status audit terbawa antar audit.
func RiskFindingID(title string) string {
	return "PROFILE-RISK_" + strings.ToLower(strings.Join(strings.Fields(title), "-"))
}

// riskNeedsAudit: risiko yang belum punya tindak lanjut (belum jadi todo, belum ditangani/diterima).
func riskNeedsAudit(r models.ProfileRisk) bool {
	if strings.TrimSpace(r.Title) == "" || r.TodoID != nil {
		return false
	}
	return r.Status == "" || r.Status == models.RiskStatusOpen
}

// ProfileRiskFindings mengubah risiko profil yang belum ditindaklanjuti menjadi temuan audit, supaya ditangani
// lewat alur audit → todo yang sama dengan temuan lain.
func ProfileRiskFindings(risks []models.ProfileRisk) []models.Finding {
	var out []models.Finding
	for _, r := range risks {
		if !riskNeedsAudit(r) {
			continue
		}
		sev := "INFO"
		if r.Severity == "high" {
			sev = "WARNING"
		}
		out = append(out, models.Finding{
			ID:          RiskFindingID(r.Title),
			Category:    CategoryRisk,
			Severity:    sev,
			Title:       "Risiko proyek: " + strings.TrimSpace(r.Title),
			Description: r.Detail,
			Suggestion:  "Selesaikan temuan ini agar dibuatkan todo, atau abaikan bila risikonya diterima.",
			TargetFile:  RiskTargetFile(r),
		})
	}
	return out
}

// projectRiskFindings membaca risiko dari profil proyek yang tersimpan.
func projectRiskFindings(project *models.Project) []models.Finding {
	if project == nil || project.ProfileData == "" {
		return nil
	}
	var p models.ProjectProfile
	if json.Unmarshal([]byte(project.ProfileData), &p) != nil {
		return nil
	}
	return ProfileRiskFindings(p.Risks)
}
