package models

import "testing"

func TestEstimateComplexityLevels(t *testing.T) {
	quick := EstimateComplexity(&Todo{Title: "Perbaiki typo label tombol simpan", Category: CategoryUIUX, Priority: TodoPriorityLow, Level: TodoLevelComponent})
	if quick.ComplexityLevel != ComplexityQuick {
		t.Fatalf("typo fix should be quick, got %+v", quick)
	}
	heavy := EstimateComplexity(&Todo{
		Title:       "Refactor alur auth JWT & migrasi skema database user",
		Description: "Pecah handler auth.go, user_repository.go, session.go, middleware.go; tambah transaksi dan retry, hindari race condition token refresh. " + string(make([]byte, 1600)),
		Category:    CategorySecurity, Priority: TodoPriorityCritical, Level: TodoLevelProject, TargetApps: "Seluruh Aplikasi (2 Apps)",
	})
	if heavy.ComplexityLevel != ComplexityComplex || heavy.ComplexityRisk < 4 || heavy.ComplexityLogic < 3 {
		t.Fatalf("auth+migration refactor should be complex & risky, got %+v", heavy)
	}
	for _, d := range []int{heavy.ComplexityLogic, heavy.ComplexityCoding, heavy.ComplexityScope, heavy.ComplexityRisk} {
		if d < 1 || d > 5 {
			t.Fatalf("dimension out of 1-5: %+v", heavy)
		}
	}
}

func TestBeforeCreateKeepsProvidedComplexity(t *testing.T) {
	td := Todo{Title: "x", TodoComplexity: TodoComplexity{ComplexityLogic: 5, ComplexityCoding: 5, ComplexityScope: 4, ComplexityRisk: 9, ComplexitySource: ComplexitySourceLLM}}
	if err := td.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if td.ComplexityRisk != 5 || td.ComplexityLevel != ComplexityComplex || td.ComplexitySource != ComplexitySourceLLM {
		t.Fatalf("LLM rating must be kept and clamped: %+v", td.TodoComplexity)
	}
	empty := Todo{Title: "Ganti warna tombol", Category: CategoryUIUX}
	_ = empty.BeforeCreate(nil)
	if !empty.HasComplexity() || empty.ComplexitySource != ComplexitySourceHeuristic {
		t.Fatalf("unrated todo must get a heuristic rating: %+v", empty.TodoComplexity)
	}
}
