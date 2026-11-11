package diagnostics

import (
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestProfileRiskFindingsSkipsFollowedUp(t *testing.T) {
	id := uint(7)
	risks := []models.ProfileRisk{
		{Title: "Token tersimpan polos", Severity: "high", Detail: "Lihat pkg/auth/token.go baris 40."},
		{Title: "Tanpa test", Severity: "medium"},
		{Title: "Sudah jadi todo", Severity: "high", TodoID: &id, Status: models.RiskStatusPlanned},
		{Title: "Diterima", Severity: "low", Status: models.RiskStatusAccepted},
	}
	got := ProfileRiskFindings(risks)
	if len(got) != 2 {
		t.Fatalf("want 2 findings, got %d", len(got))
	}
	if got[0].Severity != "WARNING" || got[0].TargetFile != "pkg/auth/token.go" || got[0].Category != CategoryRisk {
		t.Fatalf("unexpected first finding: %+v", got[0])
	}
	if got[1].Severity != "INFO" || got[1].TargetFile != "" {
		t.Fatalf("unexpected second finding: %+v", got[1])
	}
	if got[0].ID == got[1].ID {
		t.Fatal("risk findings must have distinct keys")
	}
	if len(models.DeduplicateFindings(got)) != 2 {
		t.Fatal("distinct risks must not be merged by deduplication")
	}
	if CalculateHealthScore(got) != 100 {
		t.Fatal("profile risks must not lower the health score")
	}
}

func TestRiskTargetFile(t *testing.T) {
	cases := map[string]string{
		"Handler di pkg/api/login_handler.go tanpa batas": "pkg/api/login_handler.go",
		"Halaman pages/index.vue berat":                   "pages/index.vue",
		"Belum ada dokumentasi":                           "",
	}
	for in, want := range cases {
		if got := RiskTargetFile(models.ProfileRisk{Detail: in}); got != want {
			t.Errorf("%q: want %q, got %q", in, want, got)
		}
	}
}
