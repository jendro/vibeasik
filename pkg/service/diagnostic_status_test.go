package service

import (
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestApplyKnownFindingStatusesCarriesResolved(t *testing.T) {
	fresh := []models.Finding{
		{RuleID: "SEC-001", TargetFile: "a.go", Line: 3, Severity: "HIGH"},
		{RuleID: "SEC-002", TargetFile: "b.go", Line: 9, Severity: "HIGH"},
	}
	known := []models.Finding{
		{RuleID: "SEC-001", TargetFile: "a.go", Line: 3, Status: models.FindingStatusResolved},
		{RuleID: "OLD-1", TargetFile: "z.go", Line: 1, Status: models.FindingStatusIgnored},
	}
	ApplyKnownFindingStatuses(fresh, known)
	if fresh[0].Status != models.FindingStatusResolved {
		t.Fatalf("status temuan dikenal tidak terbawa: %q", fresh[0].Status)
	}
	if fresh[1].Status == models.FindingStatusResolved || fresh[1].Status == models.FindingStatusIgnored {
		t.Fatalf("temuan baru ikut berubah status: %q", fresh[1].Status)
	}
	if len(fresh) != 2 {
		t.Fatalf("temuan lama tidak boleh ditambahkan, dapat %d", len(fresh))
	}
}
