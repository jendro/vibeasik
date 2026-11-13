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

func TestReconcileStaticFindings(t *testing.T) {
	todo := uint(7)
	prev := []Finding{
		// masih ada di baris yang sama, sudah jadi todo
		{RuleID: "QUERY-SQLI", TargetFile: "a.go", Line: 10, Snippet: "db.Raw(q)", Status: FindingStatusResolved, TodoID: &todo, ResolutionTitle: "Bind"},
		// kodenya bergeser 5 baris, pernah divonis keliru
		{RuleID: "SMELL-X", TargetFile: "a.go", Line: 20, Snippet: "_ = err", Status: FindingStatusIgnored, Verdict: "false_positive", VerdictReason: "sengaja"},
		// terbuka dan tidak terdeteksi lagi → basi
		{RuleID: "ROUTER-Y", TargetFile: "b.go", Line: 5, Snippet: "app.Post(\"/x\", h)", Status: FindingStatusOpen},
		// selesai dan kodenya sudah diperbaiki → riwayat dipertahankan
		{RuleID: "ROUTER-Y", TargetFile: "b.go", Line: 9, Snippet: "app.Post(\"/y\", h)", Status: FindingStatusResolved, TodoID: &todo},
		// bukan hasil audit statis → tidak dinilai ulang
		{RuleID: "DEEP-ROUTE-X", TargetFile: "h.go", IsDeep: true, Status: FindingStatusOpen},
		{RuleID: "LLM-REVIEW-GO", TargetFile: "h.go", Line: 3, IsLLM: true, Status: FindingStatusOpen},
		{RuleID: "RUNTIME-PAGE-BLANK", TargetFile: "/", Status: FindingStatusOpen},
	}
	fresh := []Finding{
		{RuleID: "QUERY-SQLI", TargetFile: "a.go", Line: 10, Snippet: "db.Raw(q)"},
		{RuleID: "SMELL-X", TargetFile: "a.go", Line: 25, Snippet: "_ = err"},
		{RuleID: "NEW-RULE", TargetFile: "c.go", Line: 1, Snippet: "baru"},
	}
	got := map[string]Finding{}
	for _, f := range ReconcileStaticFindings(prev, fresh) {
		got[f.ID] = f
	}
	if len(got) != 7 {
		t.Fatalf("harus 7 temuan (1 basi dibuang), dapat %d: %+v", len(got), got)
	}
	if f := got["QUERY-SQLI_a.go_10"]; f.Status != FindingStatusResolved || f.TodoID == nil || *f.TodoID != 7 || f.ResolutionTitle != "Bind" {
		t.Errorf("temuan yang masih ada harus mempertahankan status & todo: %+v", f)
	}
	if f := got["SMELL-X_a.go_25"]; f.Status != FindingStatusIgnored || f.Verdict != "false_positive" || f.Line != 25 {
		t.Errorf("temuan yang bergeser baris harus pindah ke baris baru dengan vonisnya: %+v", f)
	}
	if _, ok := got["SMELL-X_a.go_20"]; ok {
		t.Error("lokasi lama temuan yang bergeser tidak boleh tersisa")
	}
	if _, ok := got["ROUTER-Y_b.go_5"]; ok {
		t.Error("temuan statis terbuka yang tidak terdeteksi lagi harus dibuang")
	}
	if f := got["NEW-RULE_c.go_1"]; f.Status != FindingStatusOpen {
		t.Errorf("temuan baru harus terbuka: %+v", f)
	}
	kept := 0
	for _, f := range got {
		if f.IsDeep || f.IsLLM || f.RuleID == "RUNTIME-PAGE-BLANK" || (f.RuleID == "ROUTER-Y" && f.Line == 9) {
			kept++
		}
	}
	if kept != 4 {
		t.Errorf("temuan non-statis dan riwayat yang selesai harus dipertahankan, dapat %d", kept)
	}
}
