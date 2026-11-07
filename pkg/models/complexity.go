package models

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// Tingkat kerumitan todo; dipakai orkestrator untuk memilih agent, model, dan effort.
const (
	ComplexityQuick   = "quick"
	ComplexityNormal  = "normal"
	ComplexityComplex = "complex"
)

// Sumber penilaian kerumitan
const (
	ComplexitySourceHeuristic = "heuristic"
	ComplexitySourceLLM       = "llm"
	ComplexitySourceManual    = "manual"
)

// TodoComplexity adalah penilaian kerumitan multi-dimensi (skala 1–5 per dimensi).
type TodoComplexity struct {
	ComplexityLogic  int     `gorm:"default:0" json:"complexity_logic"`  // kerumitan logika/algoritma/state
	ComplexityCoding int     `gorm:"default:0" json:"complexity_coding"` // volume & kesulitan menulis kode
	ComplexityScope  int     `gorm:"default:0" json:"complexity_scope"`  // luas cakupan: file, komponen, aplikasi
	ComplexityRisk   int     `gorm:"default:0" json:"complexity_risk"`   // risiko regresi, keamanan, data
	ComplexityScore  float64 `gorm:"default:0" json:"complexity_score"`  // gabungan berbobot 1.0–5.0
	ComplexityLevel  string  `gorm:"size:16;index" json:"complexity_level"`
	ComplexityNote   string  `gorm:"type:text" json:"complexity_note,omitempty"`
	ComplexitySource string  `gorm:"size:16" json:"complexity_source,omitempty"`
}

// HasComplexity melaporkan apakah keempat dimensi sudah dinilai.
func (c TodoComplexity) HasComplexity() bool {
	return c.ComplexityLogic > 0 && c.ComplexityCoding > 0 && c.ComplexityScope > 0 && c.ComplexityRisk > 0
}

func clampDim(v int) int {
	if v < 1 {
		return 1
	}
	if v > 5 {
		return 5
	}
	return v
}

// Finalize membatasi dimensi ke 1–5 lalu menghitung skor gabungan & tingkat.
// Logika diberi bobot terbesar karena paling menentukan apakah model cepat sanggup mengerjakan.
func (c *TodoComplexity) Finalize() {
	c.ComplexityLogic, c.ComplexityCoding = clampDim(c.ComplexityLogic), clampDim(c.ComplexityCoding)
	c.ComplexityScope, c.ComplexityRisk = clampDim(c.ComplexityScope), clampDim(c.ComplexityRisk)
	score := 0.35*float64(c.ComplexityLogic) + 0.25*float64(c.ComplexityCoding) + 0.2*float64(c.ComplexityScope) + 0.2*float64(c.ComplexityRisk)
	c.ComplexityScore = math.Round(score*10) / 10
	c.ComplexityLevel = ComplexityLevelFor(c.ComplexityScore)
}

// ComplexityLevelFor memetakan skor gabungan ke quick (≤2.2), normal (≤3.4), complex.
func ComplexityLevelFor(score float64) string {
	switch {
	case score <= 2.2:
		return ComplexityQuick
	case score <= 3.4:
		return ComplexityNormal
	}
	return ComplexityComplex
}

var (
	logicWords = []string{"algoritma", "algorithm", "konkuren", "concurren", "race condition", "deadlock", "goroutine", "transaksi", "transaction",
		"state", "sinkron", "sync", "cache invalidation", "rekursi", "recursive", "siklomatis", "cyclomatic", "kompleksitas", "complexity",
		"auth", "oauth", "jwt", "permission", "otorisasi", "n+1", "query", "index", "retry", "queue", "antrean", "workflow", "state machine"}
	codingWords = []string{"refactor", "refaktor", "dekomposisi", "decompos", "god component", "god object", "monolit", "migrasi", "migration",
		"rewrite", "tulis ulang", "seluruh", "semua halaman", "all pages", "modul baru", "fitur baru", "new feature", "endpoint baru", "split", "pecah"}
	riskWords = []string{"keamanan", "security", "injection", "xss", "csrf", "secret", "credential", "token", "password", "payment", "pembayaran",
		"database", "schema", "skema", "migrasi", "migration", "produksi", "production", "hapus data", "delete", "auth", "privasi", "pii"}
	trivialWords = []string{"typo", "padding", "margin", "warna", "color", "label", "teks", "copywriting", "rename", "ganti nama", "ikon", "icon",
		"komentar", "comment", "readme", "lint", "format", "spasi"}
	filePattern = regexp.MustCompile(`[\w./-]+\.(go|vue|ts|tsx|js|jsx|py|dart|php|rb|java|kt|swift|rs|css|scss|sql)\b`)
)

// keywordRe mencocokkan kata kunci di awal kata (imbuhan Indonesia tetap cocok); kata pendek harus utuh
// agar "state" tidak cocok dengan "statement".
func keywordRe(words []string) *regexp.Regexp {
	parts := make([]string, 0, len(words))
	for _, w := range words {
		q := regexp.QuoteMeta(w)
		if len(w) <= 5 {
			q += `\b`
		}
		parts = append(parts, q)
	}
	return regexp.MustCompile(`(?i)\b(?:` + strings.Join(parts, "|") + `)`)
}

var (
	logicRe   = keywordRe(logicWords)
	codingRe  = keywordRe(codingWords)
	riskRe    = keywordRe(riskWords)
	trivialRe = keywordRe(trivialWords)
	codeBlock = regexp.MustCompile("(?s)```.*?```")
	// Bagian instruksi baku yang ditambahkan resolver audit tidak menggambarkan tugasnya.
	boilerplate = regexp.MustCompile(`(?s)###[^\n]*Instruksi Eksekusi untuk AI Agent.*$`)
	multiApps   = regexp.MustCompile(`([2-9]|\d{2,})\s*Apps`)
)

// countHits menghitung kata kunci berbeda yang muncul.
func countHits(text string, re *regexp.Regexp) int {
	seen := map[string]bool{}
	for _, m := range re.FindAllString(text, -1) {
		seen[strings.ToLower(m)] = true
	}
	return len(seen)
}

// EstimateComplexity menilai kerumitan todo secara heuristik (tanpa LLM) dari teks, kategori, prioritas & cakupan.
func EstimateComplexity(t *Todo) TodoComplexity {
	desc := boilerplate.ReplaceAllString(t.Description, "")
	codeLines := 0
	for _, blk := range codeBlock.FindAllString(desc, -1) {
		codeLines += strings.Count(blk, "\n") - 1
	}
	prose := codeBlock.ReplaceAllString(desc, "")
	text := strings.ToLower(t.Title + "\n" + prose + "\n" + t.Rationale)
	files := map[string]bool{}
	for _, f := range filePattern.FindAllString(strings.ToLower(t.Title+"\n"+desc+"\n"+t.TargetApps), -1) {
		files[f] = true
	}
	n := len(prose)

	logic := 2 + min(countHits(text, logicRe)/2, 2)
	switch t.Category {
	case CategoryArchitecture, CategorySecurity, CategoryBackend, CategoryPerformance:
		logic++
	case CategoryUIUX:
		logic--
	}

	coding := 1
	switch {
	case n > 2500:
		coding = 3
	case n > 1200:
		coding = 2
	}
	switch {
	case codeLines > 80:
		coding += 2
	case codeLines > 30:
		coding++
	}
	coding += min(countHits(text, codingRe)/2, 1)
	if len(files) >= 4 {
		coding++
	}

	scope := 2
	switch t.Level {
	case TodoLevelGlobal:
		scope = 4
	case TodoLevelProject:
		scope = 3
	}
	if len(files) >= 3 || multiApps.MatchString(t.TargetApps) {
		scope++
	}
	if len(files) == 1 && t.Level != TodoLevelGlobal {
		scope--
	}

	risk := 1 + min(countHits(text, riskRe)/2, 2)
	switch t.Priority {
	case TodoPriorityCritical:
		risk += 2
	case TodoPriorityHigh:
		risk++
	}
	if t.Category == CategorySecurity {
		risk++
	}

	if countHits(text, trivialRe) > 0 && n < 900 {
		logic, coding = logic-1, coding-1
	}

	c := TodoComplexity{ComplexityLogic: logic, ComplexityCoding: coding, ComplexityScope: scope, ComplexityRisk: risk, ComplexitySource: ComplexitySourceHeuristic}
	c.Finalize()
	c.ComplexityNote = fmt.Sprintf("Taksiran otomatis: %d file disebut, %d baris contoh kode, kategori %s, prioritas %s",
		len(files), codeLines, firstNonBlank(t.Category, "umum"), firstNonBlank(t.Priority, "-"))
	return c
}

func firstNonBlank(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// BeforeCreate mengisi kerumitan & dampak heuristik untuk setiap todo baru yang belum dinilai (audit, MCP, manual, saran AI).
func (t *Todo) BeforeCreate(tx *gorm.DB) error {
	applyRejectGuard(tx, t)
	if t.HasComplexity() {
		t.TodoComplexity.Finalize()
		if t.ComplexitySource == "" {
			t.ComplexitySource = ComplexitySourceManual
		}
	} else {
		t.TodoComplexity = EstimateComplexity(t)
	}
	if t.HasImpact() {
		t.TodoImpact.Finalize(t.Category)
	} else {
		t.TodoImpact = EstimateImpact(t)
	}
	return nil
}

// ApplyLLMRatings memasang penilaian kerumitan & dampak dari LLM/agent bila lengkap; sisanya diisi heuristik oleh BeforeCreate.
func (t *Todo) ApplyLLMRatings(c *LLMComplexity, i *LLMImpact, source string) {
	if c != nil && c.Logic > 0 && c.Coding > 0 && c.Scope > 0 && c.Risk > 0 {
		t.TodoComplexity = TodoComplexity{ComplexityLogic: c.Logic, ComplexityCoding: c.Coding, ComplexityScope: c.Scope,
			ComplexityRisk: c.Risk, ComplexityNote: c.Note, ComplexitySource: source}
	}
	if imp, ok := i.ToTodoImpact(source); ok {
		t.TodoImpact = imp
	}
}

// Rescore menghitung ulang skor kerumitan & dampak setelah dimensinya diubah (REST/MCP update).
func (t *Todo) Rescore() {
	if t.HasComplexity() {
		t.TodoComplexity.Finalize()
	}
	if t.HasImpact() {
		t.TodoImpact.Finalize(t.Category)
	}
}
