package models

import (
	"testing"
)

func TestAreFindingsMatching(t *testing.T) {
	f1 := Finding{
		ID:         "deep_1",
		RuleID:     "DEEP-ROUTE-LATENCY-CRITICAL",
		TargetFile: "handler.go",
		Line:       10,
	}
	f2 := Finding{
		ID:         "deep_1",
		RuleID:     "DEEP-ROUTE-LATENCY-CRITICAL",
		TargetFile: "handler.go",
		Line:       10,
	}
	if !AreFindingsMatching(f1, f2) {
		t.Errorf("expected findings with same ID to match")
	}

	f3 := Finding{
		RuleID:     "SQL-RAW-QUERY-INJECTION",
		TargetFile: "repo.go",
		Line:       45,
	}
	f4 := Finding{
		RuleID:     "SQL-RAW-QUERY-INJECTION",
		TargetFile: "repo.go",
		Line:       45,
	}
	if !AreFindingsMatching(f3, f4) {
		t.Errorf("expected findings with same RuleID, TargetFile, Line to match")
	}

	f5 := Finding{
		Title:      "Missing README",
		TargetFile: "README.md",
		Line:       0,
	}
	f6 := Finding{
		Title:      "Missing README",
		TargetFile: "README.md",
		Line:       0,
	}
	if !AreFindingsMatching(f5, f6) {
		t.Errorf("expected findings with same Title, TargetFile, Line to match")
	}

	fDiffLine := Finding{
		RuleID:     "SQL-RAW-QUERY-INJECTION",
		TargetFile: "repo.go",
		Line:       100,
	}
	if AreFindingsMatching(f3, fDiffLine) {
		t.Errorf("expected findings at different lines not to match")
	}
}

func TestMergeFindings_PreservesExistingAndAvoidsDuplicates(t *testing.T) {
	existing := []Finding{
		{
			ID:         "deep_latency_1",
			RuleID:     "DEEP-ROUTE-LATENCY-CRITICAL",
			Title:      "[DEEP] Latency Critical",
			TargetFile: "router.go",
			Line:       20,
			Status:     FindingStatusResolved,
			IsDeep:     true,
		},
		{
			RuleID:     "PORT-COLLISION",
			Title:      "Port Collision",
			TargetFile: "main.go",
			Line:       0,
			Status:     FindingStatusOpen,
		},
	}

	incoming := []Finding{
		// Duplicate of existing[1]
		{
			RuleID:       "PORT-COLLISION",
			Title:        "Port Collision",
			TargetFile:   "main.go",
			Line:         0,
			Status:       FindingStatusOpen,
			SuggestedFix: "// new suggestion",
		},
		// Duplicate of existing[0] (deep check issue re-detected)
		{
			ID:         "deep_latency_1",
			RuleID:     "DEEP-ROUTE-LATENCY-CRITICAL",
			Title:      "[DEEP] Latency Critical",
			TargetFile: "router.go",
			Line:       20,
			Status:     FindingStatusOpen,
			IsDeep:     true,
		},
		// Brand new finding
		{
			RuleID:     "SECURITY-HARDCODED-SECRET",
			Title:      "Hardcoded Secret",
			TargetFile: "config.go",
			Line:       12,
			Status:     FindingStatusOpen,
		},
	}

	merged := MergeFindings(existing, incoming)

	// Expected length: 2 existing + 1 genuinely new = 3 findings!
	if len(merged) != 3 {
		t.Fatalf("expected 3 findings after merge, got %d", len(merged))
	}

	// Verify existing resolved finding preserved its resolved status and was not duplicated
	if merged[0].ID != "deep_latency_1" || merged[0].Status != FindingStatusResolved {
		t.Errorf("expected deep_latency_1 to remain resolved, got status=%s", merged[0].Status)
	}

	// Verify port collision finding was not duplicated and got enriched
	if merged[1].RuleID != "PORT-COLLISION" {
		t.Errorf("expected second finding to be PORT-COLLISION")
	}
	if merged[1].SuggestedFix != "// new suggestion" {
		t.Errorf("expected PORT-COLLISION to be enriched with suggested fix")
	}

	// Verify brand new finding was added
	if merged[2].RuleID != "SECURITY-HARDCODED-SECRET" {
		t.Errorf("expected third finding to be SECURITY-HARDCODED-SECRET")
	}
}
