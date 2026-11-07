package models

import "testing"

func TestImpactFinalizeWeightsByCategory(t *testing.T) {
	// Refactor untuk AI agent: skor tinggi pada kategori arsitektur, jauh lebih rendah bila dinilai sebagai UI.
	refactor := TodoImpact{ImpactAIContext: 5, ImpactMaintain: 4, ImpactReliability: 2, ImpactSource: ComplexitySourceLLM}
	asArch, asUI := refactor, refactor
	asArch.Finalize(CategoryArchitecture)
	asUI.Finalize(CategoryUIUX)
	if asArch.ImpactLevel != ImpactLevelHigh || asArch.ImpactScore <= asUI.ImpactScore {
		t.Fatalf("architecture refactor should score high and above the UI weighting: arch=%+v ui=%+v", asArch, asUI)
	}
	// Todo yang fokus (satu dimensi kritis) tidak boleh tenggelam oleh rata-rata.
	sec := TodoImpact{ImpactSecurity: 5, ImpactReliability: 3, ImpactSource: ComplexitySourceLLM}
	sec.Finalize(CategorySecurity)
	if sec.ImpactScore < 70 {
		t.Fatalf("critical security fix should be high impact, got %+v", sec)
	}
	if sec.ImpactSummary == "" || sec.ImpactSummary[:11] != "Meningkatka" {
		t.Fatalf("default summary should explain what improves: %q", sec.ImpactSummary)
	}
	clamp := TodoImpact{ImpactUX: 9, ImpactSpeed: -3, ImpactSource: ComplexitySourceManual}
	clamp.Finalize(CategoryUIUX)
	if clamp.ImpactUX != 5 || clamp.ImpactSpeed != 0 || clamp.ImpactScore > 100 {
		t.Fatalf("dimensions must clamp to 0-5: %+v", clamp)
	}
}

func TestEstimateImpactByTaskType(t *testing.T) {
	ui := EstimateImpact(&Todo{Title: "Ganti warna tombol simpan", Category: CategoryUIUX, Priority: TodoPriorityLow})
	split := EstimateImpact(&Todo{Title: "Refactor & pecah god component TodoPage.vue jadi subkomponen + composable", Category: CategoryArchitecture, Priority: TodoPriorityHigh})
	perf := EstimateImpact(&Todo{Title: "Tambah cache & index untuk query N+1 daftar proyek (loading lambat)", Category: CategoryPerformance, Priority: TodoPriorityHigh})
	if ui.ImpactScore >= split.ImpactScore {
		t.Fatalf("trivial UI tweak should impact less than a refactor: ui=%+v split=%+v", ui, split)
	}
	if split.ImpactAIContext < 3 || split.ImpactMaintain < 3 {
		t.Fatalf("refactor should help AI agents most: %+v", split)
	}
	if perf.ImpactSpeed < 4 {
		t.Fatalf("perf todo should raise speed: %+v", perf)
	}
	for _, i := range []TodoImpact{ui, split, perf} {
		if !i.HasImpact() || i.ImpactLevel == "" {
			t.Fatalf("estimate must be finalized: %+v", i)
		}
	}
}

func TestApplyLLMRatingsAndBeforeCreate(t *testing.T) {
	td := Todo{Title: "x", Category: CategorySecurity}
	td.ApplyLLMRatings(nil, &LLMImpact{Security: 4, Summary: "Menutup celah IDOR"}, ComplexitySourceLLM)
	_ = td.BeforeCreate(nil)
	if td.ImpactSource != ComplexitySourceLLM || td.ImpactSecurity != 4 || td.ImpactSummary != "Menutup celah IDOR" {
		t.Fatalf("LLM impact must be kept: %+v", td.TodoImpact)
	}
	if td.ComplexitySource != ComplexitySourceHeuristic {
		t.Fatalf("missing complexity must fall back to heuristic: %+v", td.TodoComplexity)
	}
	zero := Todo{Title: "Tambah fitur export CSV laporan", Category: CategoryFeature}
	zero.ApplyLLMRatings(nil, &LLMImpact{}, ComplexitySourceLLM)
	_ = zero.BeforeCreate(nil)
	if zero.ImpactSource != ComplexitySourceHeuristic || zero.ImpactFeature < 3 {
		t.Fatalf("all-zero LLM impact counts as unrated: %+v", zero.TodoImpact)
	}
}

func TestClassifyTodoImpactAndExtractTarget(t *testing.T) {
	// 1. Tampilan (UI/UX)
	uiTodo := Todo{Title: "Perbaiki jarak atas sidebar kanan", Category: CategoryUIUX, AuditKey: "UI-MARGIN|frontend/components/Sidebar.vue"}
	target := ExtractFileTarget(uiTodo)
	if target != "frontend/components/Sidebar.vue" {
		t.Fatalf("expected frontend/components/Sidebar.vue, got %s", target)
	}
	resUI := ClassifyTodoImpact(uiTodo, target)
	if resUI.ImpactType != ImpactTypeUIUX || resUI.Rank != 2 || resUI.ActionStrategy != StrategyPriorityDirect {
		t.Fatalf("expected UI/UX priority direct, got %+v", resUI)
	}

	// 2. Performa Nyata
	perfTodo := Todo{Title: "Hilangkan N+1 Query pada User Profile", Category: CategoryBackend, AuditKey: "QUERY-NPLUS1|internal/user/service.go"}
	resPerf := ClassifyTodoImpact(perfTodo, ExtractFileTarget(perfTodo))
	if resPerf.ImpactType != ImpactTypePerformance || resPerf.Rank != 2 || resPerf.ActionStrategy != StrategyPriorityDirect {
		t.Fatalf("expected Performance priority direct, got %+v", resPerf)
	}

	// 3. Kode Saja (Micro / low impact)
	microTodo := Todo{Title: "Optimalkan Loading Gambar Avatar (lazy load)", Category: CategoryGeneral, AuditKey: "FE-IMG-NO-LAZY|frontend/components/Avatar.vue"}
	resMicro := ClassifyTodoImpact(microTodo, ExtractFileTarget(microTodo))
	if resMicro.ImpactType != ImpactTypeCodeOnly || resMicro.Rank != 0 || resMicro.ActionStrategy != StrategyBatchAndDefer {
		t.Fatalf("expected CodeOnly batch & defer, got %+v", resMicro)
	}
}
