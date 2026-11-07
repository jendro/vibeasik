package models

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

// Dampak todo pada aplikasi. Kode aplikasi dirawat oleh AI agent, jadi kualitas kode dinilai dari sudut
// pandang agent (ai_context, maintainability); manusia cukup merasakan tampilan (ux) & kecepatan (speed).
const (
	ImpactLevelHigh    = "high"    // skor ≥ 70
	ImpactLevelMedium  = "medium"  // skor ≥ 45
	ImpactLevelLow     = "low"     // skor ≥ 20
	ImpactLevelMinimal = "minimal" // skor < 20
)

// Kunci dimensi dampak; urutannya sama dengan ImpactDims.
const (
	ImpactDimAIContext   = "ai_context"
	ImpactDimMaintain    = "maintainability"
	ImpactDimReliability = "reliability"
	ImpactDimSecurity    = "security"
	ImpactDimSpeed       = "speed"
	ImpactDimUX          = "ux"
	ImpactDimFeature     = "feature"
)

// ImpactDim menjelaskan satu dimensi dampak (skala 0–5, 0 = tidak berpengaruh).
type ImpactDim struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Audience string `json:"audience"` // ai_agent | app | human
	Hint     string `json:"hint"`
}

// ImpactDims adalah daftar dimensi dampak yang dipakai skor, prompt LLM, dan UI.
var ImpactDims = []ImpactDim{
	{ImpactDimAIContext, "Kejelasan untuk AI", "ai_agent", "Kode lebih mudah ditemukan & dipahami AI agent: file kecil, nama jelas, tanggung jawab tunggal, tipe/dokumentasi, hemat token konteks"},
	{ImpactDimMaintain, "Kemudahan Dirawat AI", "ai_agent", "Perubahan berikutnya oleh AI agent lebih aman: modular, coupling rendah, ada test, duplikasi berkurang"},
	{ImpactDimReliability, "Keandalan", "app", "Bug, crash, penanganan error, dan integritas data"},
	{ImpactDimSecurity, "Keamanan", "app", "Celah keamanan, kredensial, otorisasi, validasi input"},
	{ImpactDimSpeed, "Kecepatan Loading", "human", "Waktu muat & respons yang dirasakan pengguna"},
	{ImpactDimUX, "Tampilan & UX", "human", "Tampilan, kejelasan, dan kenyamanan pengguna"},
	{ImpactDimFeature, "Nilai Fitur", "human", "Kemampuan baru yang bisa dipakai pengguna"},
}

// impactWeights adalah bobot dimensi per kategori todo (urutan = ImpactDims). Tiap jenis task dinilai
// berbeda: refactor arsitektur diukur dari manfaatnya bagi AI agent, UI dari tampilan & kecepatan, dst.
var impactWeights = map[string][7]float64{
	CategoryArchitecture: {0.30, 0.30, 0.15, 0.05, 0.05, 0.05, 0.10},
	CategorySecurity:     {0.05, 0.05, 0.20, 0.50, 0.05, 0.05, 0.10},
	CategoryUIUX:         {0.10, 0.10, 0.05, 0.05, 0.25, 0.35, 0.10},
	CategoryPerformance:  {0.10, 0.10, 0.15, 0.05, 0.40, 0.10, 0.10},
	CategoryBackend:      {0.15, 0.20, 0.25, 0.15, 0.10, 0.05, 0.10},
	CategoryFeature:      {0.10, 0.10, 0.15, 0.05, 0.05, 0.15, 0.40},
	CategoryDevOps:       {0.20, 0.25, 0.25, 0.15, 0.10, 0.00, 0.05},
}

var defaultImpactWeights = [7]float64{1, 1, 1, 1, 1, 1, 1}

// ImpactWeightsFor mengembalikan bobot dimensi (urutan ImpactDims) untuk sebuah kategori.
func ImpactWeightsFor(category string) [7]float64 {
	if w, ok := impactWeights[category]; ok {
		return w
	}
	return defaultImpactWeights
}

// ImpactReference menjelaskan cara skor dampak dihitung (dimensi, bobot per kategori, ambang level)
// untuk UI dan agent yang menilai todo sendiri.
func ImpactReference() map[string]interface{} {
	cats := []string{CategoryFeature, CategorySecurity, CategoryUIUX, CategoryArchitecture,
		CategoryPerformance, CategoryBackend, CategoryDevOps, CategoryGeneral}
	weights := make(map[string][7]float64, len(cats))
	for _, cat := range cats {
		weights[cat] = ImpactWeightsFor(cat)
	}
	return map[string]interface{}{
		"dimensions": ImpactDims,
		"weights":    weights,
		"levels":     map[string]int{ImpactLevelHigh: 70, ImpactLevelMedium: 45, ImpactLevelLow: 20, ImpactLevelMinimal: 0},
		"formula":    "skor = 100 × (0.6 × rata-rata berbobot + 0.4 × puncak berbobot), dimensi 0–5",
	}
}

// TodoImpact adalah penilaian dampak multi-dimensi sebuah todo (skala 0–5 per dimensi).
type TodoImpact struct {
	ImpactAIContext   int    `gorm:"default:0" json:"impact_ai_context"`
	ImpactMaintain    int    `gorm:"column:impact_maintainability;default:0" json:"impact_maintainability"`
	ImpactReliability int    `gorm:"default:0" json:"impact_reliability"`
	ImpactSecurity    int    `gorm:"default:0" json:"impact_security"`
	ImpactSpeed       int    `gorm:"default:0" json:"impact_speed"`
	ImpactUX          int    `gorm:"column:impact_ux;default:0" json:"impact_ux"`
	ImpactFeature     int    `gorm:"default:0" json:"impact_feature"`
	ImpactScore       int    `gorm:"default:0;index" json:"impact_score"` // skor final 0–100, berbobot kategori
	ImpactLevel       string `gorm:"size:16" json:"impact_level"`
	ImpactSummary     string `gorm:"type:text" json:"impact_summary,omitempty"` // apa yang meningkat setelah todo selesai
	ImpactSource      string `gorm:"size:16" json:"impact_source,omitempty"`    // heuristic | llm | manual
}

// HasImpact melaporkan apakah dampak sudah dinilai (0 adalah nilai sah, jadi dilihat dari sumbernya).
func (i TodoImpact) HasImpact() bool { return i.ImpactSource != "" }

func (i *TodoImpact) dims() [7]*int {
	return [7]*int{&i.ImpactAIContext, &i.ImpactMaintain, &i.ImpactReliability, &i.ImpactSecurity, &i.ImpactSpeed, &i.ImpactUX, &i.ImpactFeature}
}

// Values mengembalikan nilai dimensi dengan urutan ImpactDims.
func (i TodoImpact) Values() [7]int {
	return [7]int{i.ImpactAIContext, i.ImpactMaintain, i.ImpactReliability, i.ImpactSecurity, i.ImpactSpeed, i.ImpactUX, i.ImpactFeature}
}

func clampImpact(v int) int {
	return max(0, min(5, v))
}

// Finalize membatasi dimensi ke 0–5 lalu menghitung skor final 0–100 untuk kategori todo:
// 60% rata-rata berbobot + 40% puncak berbobot, agar todo yang fokus (mis. satu celah keamanan
// kritis) tetap bernilai tinggi walau dimensi lain nol.
func (i *TodoImpact) Finalize(category string) {
	w := ImpactWeightsFor(category)
	var sumW, sum, wMax, peak float64
	for _, x := range w {
		wMax = math.Max(wMax, x)
	}
	for k, p := range i.dims() {
		*p = clampImpact(*p)
		v := float64(*p) / 5
		sum += w[k] * v
		sumW += w[k]
		peak = math.Max(peak, v*w[k]/wMax)
	}
	i.ImpactScore = int(math.Round(100 * (0.6*sum/sumW + 0.4*peak)))
	i.ImpactLevel = ImpactLevelFor(i.ImpactScore)
	if strings.TrimSpace(i.ImpactSummary) == "" {
		i.ImpactSummary = i.DefaultSummary()
	}
}

// ImpactLevelFor memetakan skor 0–100 ke high (≥70), medium (≥45), low (≥20), minimal.
func ImpactLevelFor(score int) string {
	switch {
	case score >= 70:
		return ImpactLevelHigh
	case score >= 45:
		return ImpactLevelMedium
	case score >= 20:
		return ImpactLevelLow
	}
	return ImpactLevelMinimal
}

// TopImpacts mengembalikan label dimensi bernilai ≥ minValue, terbesar dulu.
func (i TodoImpact) TopImpacts(minValue int) []string {
	type kv struct {
		label string
		v     int
	}
	var out []kv
	for k, v := range i.Values() {
		if v >= minValue && v > 0 {
			out = append(out, kv{ImpactDims[k].Label, v})
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].v > out[b].v })
	labels := make([]string, len(out))
	for k, x := range out {
		labels[k] = fmt.Sprintf("%s (%d)", x.label, x.v)
	}
	return labels
}

// DefaultSummary menyusun ringkasan "meningkatkan apa" dari dimensi terbesar.
func (i TodoImpact) DefaultSummary() string {
	top := i.TopImpacts(2)
	if len(top) == 0 {
		top = i.TopImpacts(1)
	}
	if len(top) == 0 {
		return "Dampak kecil pada aplikasi"
	}
	if len(top) > 3 {
		top = top[:3]
	}
	return "Meningkatkan " + strings.Join(top, ", ")
}

// LLMImpact adalah penilaian dampak 0–5 yang dikembalikan LLM (atau agent lewat MCP) untuk sebuah todo.
type LLMImpact struct {
	AIContext       int    `json:"ai_context"`
	Maintainability int    `json:"maintainability"`
	Reliability     int    `json:"reliability"`
	Security        int    `json:"security"`
	Speed           int    `json:"speed"`
	UX              int    `json:"ux"`
	Feature         int    `json:"feature"`
	Summary         string `json:"summary"`
}

// ToTodoImpact mengubah penilaian LLM menjadi TodoImpact; ok=false bila semua dimensi nol (dianggap tidak dinilai).
func (l *LLMImpact) ToTodoImpact(source string) (TodoImpact, bool) {
	if l == nil {
		return TodoImpact{}, false
	}
	i := TodoImpact{ImpactAIContext: l.AIContext, ImpactMaintain: l.Maintainability, ImpactReliability: l.Reliability,
		ImpactSecurity: l.Security, ImpactSpeed: l.Speed, ImpactUX: l.UX, ImpactFeature: l.Feature,
		ImpactSummary: strings.TrimSpace(l.Summary), ImpactSource: source}
	for _, v := range i.Values() {
		if v > 0 {
			return i, true
		}
	}
	return TodoImpact{}, false
}

var (
	aiContextRe = keywordRe([]string{"refactor", "refaktor", "pecah", "split", "dekomposisi", "decompos", "monolit", "god component", "god object",
		"penamaan", "naming", "rename", "ganti nama", "dokumentasi", "documentation", "docs", "readme", "jsdoc", "docstring", "komentar",
		"tipe", "typing", "typescript", "interface", "struktur folder", "folder structure", "dead code", "kode mati", "unused", "magic number",
		"konstanta", "single responsibility", "srp", "peta kode", "code map", "codemap"})
	maintainRe = keywordRe([]string{"test", "unit test", "pengujian", "coupling", "decoupl", "dependency injection", "abstraksi", "abstraction",
		"reuse", "duplikasi", "duplicat", "dry", "modular", "modul", "lint", "standarisasi", "standardiz", "kontrak", "contract", "openapi",
		"konfigurasi", "config", "ci", "pipeline", "composable", "helper", "service layer", "repository"})
	reliabilityRe = keywordRe([]string{"bug", "error", "crash", "panic", "null", "nil", "undefined", "exception", "handle", "penanganan",
		"retry", "timeout", "race condition", "deadlock", "leak", "bocor", "transaksi", "transaction", "rollback", "validasi", "validation",
		"integritas", "integrity", "konsisten", "consisten", "fix", "perbaiki", "gagal", "fail", "hilang", "unclosed", "defer"})
	securityRe = keywordRe([]string{"keamanan", "security", "injection", "xss", "csrf", "secret", "credential", "kredensial", "token", "password",
		"auth", "jwt", "oauth", "cors", "rate limit", "sanitasi", "sanitiz", "hash", "bcrypt", "argon", "enkripsi", "encrypt", "privasi",
		"pii", "otorisasi", "authoriz", "permission", "owasp", "bola", ".env"})
	speedRe = keywordRe([]string{"performa", "performance", "cepat", "lambat", "slow", "loading", "lazy", "cache", "caching", "index",
		"n+1", "query", "bundle", "optimasi", "optimiz", "latency", "latensi", "pagination", "paginasi", "debounce", "memo", "prefetch",
		"compress", "gzip", "webp", "render", "ssr", "overfetch", "select *", "waterfall"})
	uxRe = keywordRe([]string{"tampilan", "ui", "ux", "layout", "responsif", "responsive", "mobile", "warna", "color", "ikon", "icon", "toast",
		"skeleton", "empty state", "aksesibilitas", "accessibility", "a11y", "font", "tipografi", "animasi", "modal", "form", "tombol",
		"button", "dark mode", "copywriting", "feedback", "tap target", "spinner"})
	featureRe = keywordRe([]string{"fitur", "feature", "tambah", "tambahkan", "baru", "new", "integrasi", "integration", "webhook", "export",
		"ekspor", "import", "impor", "notifikasi", "notification", "dashboard", "laporan", "report", "otomasi", "automation", "realtime"})
)

// EstimateImpact menaksir dampak todo secara heuristik (tanpa LLM) dari kategori, prioritas, dan kata kunci.
func EstimateImpact(t *Todo) TodoImpact {
	desc := codeBlock.ReplaceAllString(boilerplate.ReplaceAllString(t.Description, ""), "")
	text := strings.ToLower(t.Title + "\n" + desc + "\n" + t.Rationale)

	var d [7]int
	switch t.Category {
	case CategoryArchitecture:
		d[0], d[1] = 2, 2
	case CategorySecurity:
		d[3] = 3
	case CategoryUIUX:
		d[5] = 3
	case CategoryPerformance:
		d[4] = 3
	case CategoryBackend:
		d[1], d[2] = 1, 2
	case CategoryFeature:
		d[6] = 3
	case CategoryDevOps:
		d[1], d[2] = 2, 1
	}
	ref := strings.ToUpper(t.GuidelineRef)
	switch {
	case strings.HasPrefix(ref, "SEC"):
		d[3] = max(d[3], 3)
	case strings.HasPrefix(ref, "PERF"), strings.HasPrefix(ref, "NET"):
		d[4] = max(d[4], 2)
	case strings.HasPrefix(ref, "ARCH"), strings.HasPrefix(ref, "DEEP-FUNC"), strings.HasPrefix(ref, "SMELL"):
		d[0], d[1] = max(d[0], 2), max(d[1], 2)
	case strings.HasPrefix(ref, "FE"), strings.HasPrefix(ref, "UI"):
		d[5] = max(d[5], 2)
	}
	for k, re := range []*regexp.Regexp{aiContextRe, maintainRe, reliabilityRe, securityRe, speedRe, uxRe, featureRe} {
		seen := map[string]bool{}
		for _, m := range re.FindAllString(text, -1) {
			seen[strings.ToLower(m)] = true
		}
		d[k] += min((len(seen)+1)/2, 2)
	}

	// Prioritas menaikkan dimensi utama (yang terbesar setelah penilaian di atas).
	bump := 0
	switch t.Priority {
	case TodoPriorityCritical:
		bump = 2
	case TodoPriorityHigh:
		bump = 1
	}
	if bump > 0 {
		top := 0
		w := ImpactWeightsFor(t.Category)
		for k := range d {
			if float64(d[k])*w[k] > float64(d[top])*w[top] {
				top = k
			}
		}
		d[top] += bump
	}
	if countHits(text, trivialRe) > 0 && len(desc) < 900 {
		for k := range d {
			d[k] = min(d[k], 2)
		}
	}

	i := TodoImpact{ImpactAIContext: d[0], ImpactMaintain: d[1], ImpactReliability: d[2], ImpactSecurity: d[3],
		ImpactSpeed: d[4], ImpactUX: d[5], ImpactFeature: d[6], ImpactSource: ComplexitySourceHeuristic}
	i.Finalize(t.Category)
	return i
}

// Columns mengembalikan kolom DB dampak untuk Updates (map agar nilai 0 ikut tersimpan).
func (i TodoImpact) Columns() map[string]interface{} {
	return map[string]interface{}{
		"impact_ai_context": i.ImpactAIContext, "impact_maintainability": i.ImpactMaintain, "impact_reliability": i.ImpactReliability,
		"impact_security": i.ImpactSecurity, "impact_speed": i.ImpactSpeed, "impact_ux": i.ImpactUX, "impact_feature": i.ImpactFeature,
		"impact_score": i.ImpactScore, "impact_level": i.ImpactLevel, "impact_summary": i.ImpactSummary, "impact_source": i.ImpactSource,
	}
}

// Columns mengembalikan kolom DB kerumitan untuk Updates.
func (c TodoComplexity) Columns() map[string]interface{} {
	return map[string]interface{}{
		"complexity_logic": c.ComplexityLogic, "complexity_coding": c.ComplexityCoding, "complexity_scope": c.ComplexityScope,
		"complexity_risk": c.ComplexityRisk, "complexity_score": c.ComplexityScore, "complexity_level": c.ComplexityLevel,
		"complexity_note": c.ComplexityNote, "complexity_source": c.ComplexitySource,
	}
}

// Konstanta 3 Pilar Dampak Todo & Strategi Efisiensi Agent
const (
	ImpactTypeUIUX        = "ui_ux"
	ImpactTypePerformance = "performance"
	ImpactTypeCodeOnly    = "code_only"

	ImpactLabelUIUX        = "Tampilan (UI/UX)"
	ImpactLabelPerformance = "Performa Nyata"
	ImpactLabelCodeOnly    = "Kode Saja (Dampak Kecil)"

	StrategyPriorityDirect = "Prioritas Langsung"
	StrategyBatchAndDefer  = "Dikerjakan Belakangan & Digabung"
)

// ClassificationResult memuat taksonomi dampak todo
type ClassificationResult struct {
	ImpactType      string `json:"impact_type"`  // ui_ux | performance | code_only
	ImpactLabel     string `json:"impact_label"` // Tampilan (UI/UX) | Performa Nyata | Kode Saja (Dampak Kecil)
	ImpactTier      string `json:"impact_tier"`  // high | low
	ImpactRationale string `json:"impact_rationale"`
	ActionStrategy  string `json:"action_strategy"` // Prioritas Langsung | Dikerjakan Belakangan & Digabung
	Rank            int    `json:"rank"`            // 2 = High (UI/UX, Performance), 1 = Standard, 0 = Code-only (deferred)
}

var reFileInText = regexp.MustCompile(`(?i)([a-zA-Z0-9_\-\./]+\.(?:vue|ts|js|dart|go|jsx|tsx|css|json))`)

// ExtractFileTarget mengekstrak berkas target dari audit key, judul, atau deskripsi todo.
func ExtractFileTarget(t Todo) string {
	if idx := strings.Index(t.AuditKey, "|"); idx != -1 {
		return strings.TrimSpace(t.AuditKey[idx+1:])
	}
	if m := reFileInText.FindStringSubmatch(t.Title); len(m) > 1 {
		return m[1]
	}
	if m := reFileInText.FindStringSubmatch(t.Description); len(m) > 1 {
		return m[1]
	}
	return ""
}

// ClassifyTodoImpact mengelompokkan todo ke dalam 3 pilar: Tampilan, Performa Nyata, atau Kode Saja (Dampak Mikro).
func ClassifyTodoImpact(t Todo, fileTarget string) ClassificationResult {
	titleLower := strings.ToLower(t.Title)
	auditLower := strings.ToLower(t.AuditKey)

	isMicro := strings.Contains(titleLower, "ikon sidebar") ||
		strings.Contains(titleLower, "confirmation button") ||
		strings.Contains(titleLower, "tombol login google") ||
		strings.Contains(titleLower, "alt gambar") ||
		strings.Contains(titleLower, "standarisasi alt") ||
		strings.Contains(titleLower, "preview gambar form") ||
		strings.Contains(titleLower, "preview logo") ||
		(strings.Contains(auditLower, "fe-img-no-lazy") &&
			!strings.Contains(titleLower, "lcp") &&
			!strings.Contains(titleLower, "hero") &&
			!strings.Contains(titleLower, "fold") &&
			!strings.Contains(titleLower, "sidebar warga")) ||
		strings.Contains(auditLower, "fe-img-missing-alt") ||
		strings.Contains(auditLower, "fe-small-font") ||
		strings.Contains(titleLower, "normalisasi working tree") ||
		strings.Contains(titleLower, "audit dan commit") ||
		strings.Contains(titleLower, "regresi test") ||
		strings.Contains(titleLower, "smoke test") ||
		strings.Contains(titleLower, "tipe openapi") ||
		(t.ImpactScore > 0 && t.ImpactScore < 30)

	isPerf := t.Category == CategoryPerformance ||
		strings.Contains(auditLower, "query-nplus1") ||
		strings.Contains(auditLower, "mem-leak") ||
		strings.Contains(auditLower, "release-not-optimized") ||
		strings.Contains(auditLower, "slow-query") ||
		strings.Contains(titleLower, "memory leak") ||
		strings.Contains(titleLower, "n+1") ||
		strings.Contains(titleLower, "r8") ||
		strings.Contains(titleLower, "lcp") ||
		strings.Contains(titleLower, "cegah memory leak") ||
		strings.Contains(titleLower, "eliminasi n+1") ||
		strings.Contains(titleLower, "batch lookup") ||
		strings.Contains(titleLower, "cls pada logo") ||
		(t.ImpactSpeed >= 3 && t.ImpactScore >= 45 && !isMicro)

	isUI := t.Category == CategoryUIUX ||
		strings.Contains(auditLower, "clickable-non-interactive") ||
		strings.Contains(auditLower, "native-dialog") ||
		strings.Contains(auditLower, "contrast") ||
		strings.Contains(auditLower, "accessibility") ||
		strings.Contains(titleLower, "margin") ||
		strings.Contains(titleLower, "padding") ||
		strings.Contains(titleLower, "jarak atas") ||
		strings.Contains(titleLower, "sidebar kanan aktif") ||
		strings.Contains(titleLower, "modal") ||
		strings.Contains(titleLower, "toast") ||
		strings.Contains(titleLower, "dropdown") ||
		strings.Contains(titleLower, "aksesibilitas keyboard") ||
		(t.ImpactUX >= 3 && t.ImpactUX > t.ImpactSpeed && !isMicro)

	if isUI && !isMicro {
		return ClassificationResult{
			ImpactType:      ImpactTypeUIUX,
			ImpactLabel:     ImpactLabelUIUX,
			ImpactTier:      "high",
			ImpactRationale: "Perubahan visual langsung atau interaksi layar (tata letak, margin, modal, dialog, atau aksesibilitas pengguna).",
			ActionStrategy:  StrategyPriorityDirect,
			Rank:            2,
		}
	}
	if isPerf {
		return ClassificationResult{
			ImpactType:      ImpactTypePerformance,
			ImpactLabel:     ImpactLabelPerformance,
			ImpactTier:      "high",
			ImpactRationale: "Optimasi performa nyata (Core Web Vitals LCP/CLS hero, eliminasi N+1 database, r8 build, atau pencegahan memory leak).",
			ActionStrategy:  StrategyPriorityDirect,
			Rank:            2,
		}
	}

	return ClassificationResult{
		ImpactType:      ImpactTypeCodeOnly,
		ImpactLabel:     ImpactLabelCodeOnly,
		ImpactTier:      "low",
		ImpactRationale: "Perubahan atribut kode internal atau tes duplikatif tanpa perubahan visual dan efek performa imperseptibel. Disarankan dikerjakan belakangan dan digabungkan.",
		ActionStrategy:  StrategyBatchAndDefer,
		Rank:            0,
	}
}
