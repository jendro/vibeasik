package models

import (
	"strings"
	"testing"
)

const sampleRationale = "Landasan: SLO 200 ms.\n\n" +
	"Catatan penyelesaian (Antigravity, 1 Nov 2026 04:05): HASIL: latensi turun ke 6.31 ms (ns/op).\nBERKAS: backend/handlers/dup.go\n\n" +
	"Catatan penyelesaian (agent, 2 Nov 2026 21:27): ### Ringkasan\nHASIL: p95 HTTP 98 ms, EXPLAIN memakai indeks.\nBERKAS DIUBAH:\n1. backend/handlers/dup.go\n2. backend/migrations/031_idx.sql"

func TestSplitRationaleAndLatestNote(t *testing.T) {
	analysis, notes := SplitRationale(sampleRationale)
	if analysis != "Landasan: SLO 200 ms." || len(notes) != 2 {
		t.Fatalf("analysis=%q notes=%d", analysis, len(notes))
	}
	if notes[1].Who != "agent" || notes[1].When != "2 Nov 2026 21:27" {
		t.Fatalf("note = %+v", notes[1])
	}
	td := Todo{Rationale: sampleRationale}
	if latest := td.LatestCompletionNote(); !strings.Contains(latest, "p95 HTTP 98 ms") || strings.Contains(latest, "6.31") {
		t.Fatalf("catatan terakhir salah: %q", latest)
	}
	// Pemotongan untuk prompt QA harus memuat percobaan TERAKHIR, bukan yang pertama.
	p := td.RationaleForPrompt(100, 500)
	if !strings.Contains(p, "p95 HTTP 98 ms") || strings.Contains(p, "6.31 ms") || !strings.Contains(p, "ke-2 dari 2") {
		t.Fatalf("RationaleForPrompt = %q", p)
	}
}

func TestBuildTodoMemory(t *testing.T) {
	m := BuildTodoMemory(Todo{Rationale: sampleRationale}, []int{20, 61}, "Belum ada  test integrasi\nHTTP.")
	for _, want := range []string{"Percobaan: 2", "20→61", "p95 HTTP 98 ms", "backend/migrations/031_idx.sql", "backend/handlers/dup.go", "QA terakhir: Belum ada test integrasi HTTP."} {
		if !strings.Contains(m, want) {
			t.Errorf("memori tanpa %q:\n%s", want, m)
		}
	}
	if strings.Count(m, "backend/handlers/dup.go") != 1 {
		t.Errorf("berkas ganda: %s", m)
	}
	if BuildTodoMemory(Todo{Rationale: "analisis saja"}, nil, "") != "" {
		t.Error("todo yang belum dikerjakan tidak punya memori")
	}
	long := BuildTodoMemory(Todo{Rationale: "x\n\nCatatan penyelesaian (a, b): " + strings.Repeat("kata ", 2000)}, nil, strings.Repeat("q ", 2000))
	if len([]rune(long)) > TodoMemoryMax+1 {
		t.Errorf("memori melebihi batas: %d", len([]rune(long)))
	}
}
