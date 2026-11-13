package models

import (
	"fmt"
	"strings"
	"time"
)

// Diagnostic represents an automated code inspection and health report
type Diagnostic struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProjectID     uint      `gorm:"index;not null" json:"project_id"`
	ComponentID   *uint     `gorm:"index" json:"component_id,omitempty"`
	Summary       string    `gorm:"type:longtext" json:"summary"`
	Score         int       `json:"score"`
	FindingsJSON  string    `gorm:"type:longtext" json:"findings_json"`  // JSON array of Finding
	Optimizations string    `gorm:"type:longtext" json:"optimizations"`  // JSON array of suggestions
	StructureTree string    `gorm:"type:longtext" json:"structure_tree"` // JSON tree of project files
	CreatedAt     time.Time `json:"created_at"`
}

// Finding status constants
const (
	FindingStatusOpen     = "open"
	FindingStatusResolved = "resolved"
	FindingStatusIgnored  = "ignored"
)

// Finding represents an individual issue, warning, or best-practice finding
type Finding struct {
	ID           string `json:"id,omitempty"`
	Category     string `json:"category"` // query, router, code_smell, architecture, security, port, config
	Severity     string `json:"severity"` // CRITICAL, WARNING, INFO, OPTIMIZATION
	Title        string `json:"title"`
	Description  string `json:"description"`
	Suggestion   string `json:"suggestion"`
	TargetFile   string `json:"target_file,omitempty"`
	Line         int    `json:"line,omitempty"`
	Snippet      string `json:"snippet,omitempty"`
	RuleID       string `json:"rule_id,omitempty"`
	TheoryBasis  string `json:"theory_basis,omitempty"`
	SuggestedFix string `json:"suggested_fix,omitempty"`
	Status       string `json:"status,omitempty"`  // "open", "resolved", "ignored"
	IsDeep       bool   `json:"is_deep,omitempty"` // true if issue detected specifically by Deep Check
	IsLLM        bool   `json:"is_llm,omitempty"`  // true if issue reported by the queued LLM audit review

	// Vonis pembelajaran audit (models.AuditFeedback): false_positive = audit keliru, confirmed = temuan benar
	Verdict       string `json:"verdict,omitempty"`
	VerdictReason string `json:"verdict_reason,omitempty"`

	// Saved LLM resolution & action steps for AI agent execution and historical review
	ResolutionTitle string          `json:"resolution_title,omitempty"`
	Explanation     string          `json:"explanation,omitempty"`
	ActionSteps     []string        `json:"action_steps,omitempty"`
	CodeBefore      string          `json:"code_before,omitempty"`
	CodeAfter       string          `json:"code_after,omitempty"`
	TargetLines     string          `json:"target_lines,omitempty"`
	PlacementPlan   []PlacementStep `json:"placement_plan,omitempty"`
	TodoID          *uint           `json:"todo_id,omitempty"`
	ModelUsed       string          `json:"model_used,omitempty"`
	ResolvedAt      *time.Time      `json:"resolved_at,omitempty"`
}

// PlacementStep adalah satu baris rencana penempatan logika hasil LLM: fungsi/tipe apa dipindah ke berkas & layer
// mana (sesuai peta layer proyek), atau dipakai ulang/disatukan dengan fungsi yang sudah ada (cegah fungsi kembar).
type PlacementStep struct {
	Symbol  string `json:"symbol"`
	Action  string `json:"action"` // pindah, pakai_ulang, satukan, tetap, baru
	Target  string `json:"target"` // berkas tujuan (relatif root proyek)
	Layer   string `json:"layer"`  // handler, service, repository, model, util
	Reuse   string `json:"reuse,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Warning string `json:"warning,omitempty"` // hasil cek vibeasik terhadap peta kode (folder baru, simbol tidak ada)
}

// CopyResolution menyalin hasil penyelesaian LLM dari src (temuan sama di laporan lain) ke f.
func (f *Finding) CopyResolution(src Finding) {
	f.Status = src.Status
	f.ResolutionTitle = src.ResolutionTitle
	f.Explanation = src.Explanation
	f.ActionSteps = src.ActionSteps
	f.CodeBefore = src.CodeBefore
	f.CodeAfter = src.CodeAfter
	f.TargetLines = src.TargetLines
	f.PlacementPlan = src.PlacementPlan
	f.ModelUsed = src.ModelUsed
	f.TheoryBasis = src.TheoryBasis
	f.TodoID = src.TodoID
	f.ResolvedAt = src.ResolvedAt
}

// GenerateFindingKey computes a stable unique identifier for a finding
func GenerateFindingKey(f Finding) string {
	if f.ID != "" {
		return f.ID
	}
	rule := f.RuleID
	if rule == "" {
		rule = f.Title
	}
	return fmt.Sprintf("%s_%s_%d", rule, f.TargetFile, f.Line)
}

// AreFindingsMatching checks whether two findings represent the same issue
func AreFindingsMatching(a, b Finding) bool {
	if a.ID != "" && b.ID != "" && a.ID == b.ID {
		return true
	}
	keyA := GenerateFindingKey(a)
	keyB := GenerateFindingKey(b)
	if keyA != "" && keyB != "" && keyA == keyB {
		return true
	}
	// Secondary fallback: Match on rule_id + target_file + line
	if a.RuleID != "" && b.RuleID != "" && a.RuleID == b.RuleID &&
		a.TargetFile == b.TargetFile && a.Line == b.Line {
		return true
	}
	// Fallback for title matching if rule_id is empty
	if a.Title != "" && b.Title != "" && a.Title == b.Title &&
		a.TargetFile == b.TargetFile && a.Line == b.Line {
		return true
	}
	return false
}

// DeduplicateFindings returns a slice of unique findings by their stable key.
// If duplicates exist, non-open statuses (e.g. ignored, resolved) take precedence.
func DeduplicateFindings(findings []Finding) []Finding {
	if len(findings) == 0 {
		return findings
	}
	unique := make([]Finding, 0, len(findings))

	for _, f := range findings {
		if f.ID == "" {
			f.ID = GenerateFindingKey(f)
		}
		if f.Status == "" {
			f.Status = FindingStatusOpen
		}

		foundIdx := -1
		for i, u := range unique {
			if AreFindingsMatching(u, f) {
				foundIdx = i
				break
			}
		}

		if foundIdx >= 0 {
			// If duplicate has an active resolution or ignored status, preserve that
			if unique[foundIdx].Status == FindingStatusOpen && f.Status != FindingStatusOpen && f.Status != "" {
				unique[foundIdx].Status = f.Status
			}
			continue
		}
		unique = append(unique, f)
	}
	return unique
}

// MergeFindings retains all existing findings without deleting any of them.
// For incoming findings, if an issue is already present in existing, it is NOT added again.
// If it is genuinely a new issue, it is appended.
func MergeFindings(existing []Finding, incoming []Finding) []Finding {
	result := make([]Finding, len(existing), len(existing)+len(incoming))
	copy(result, existing)

	for _, inc := range incoming {
		if inc.ID == "" {
			inc.ID = GenerateFindingKey(inc)
		}
		if inc.Status == "" {
			inc.Status = FindingStatusOpen
		}

		foundIdx := -1
		for i, curr := range result {
			if AreFindingsMatching(curr, inc) {
				foundIdx = i
				break
			}
		}

		if foundIdx >= 0 {
			// Already present in the table: do not add again!
			if result[foundIdx].Status == FindingStatusOpen && inc.Status != FindingStatusOpen && inc.Status != "" {
				result[foundIdx].Status = inc.Status
			}
			if result[foundIdx].TheoryBasis == "" && inc.TheoryBasis != "" {
				result[foundIdx].TheoryBasis = inc.TheoryBasis
			}
			if result[foundIdx].SuggestedFix == "" && inc.SuggestedFix != "" {
				result[foundIdx].SuggestedFix = inc.SuggestedFix
			}
			if result[foundIdx].Snippet == "" && inc.Snippet != "" {
				result[foundIdx].Snippet = inc.Snippet
			}
			if inc.IsDeep {
				result[foundIdx].IsDeep = true
			}
			if inc.IsLLM {
				result[foundIdx].IsLLM = true
			}
			if result[foundIdx].Verdict == "" && inc.Verdict != "" {
				result[foundIdx].Verdict, result[foundIdx].VerdictReason = inc.Verdict, inc.VerdictReason
			}
		} else {
			// Genuinely new finding: add it!
			result = append(result, inc)
		}
	}

	return DeduplicateFindings(result)
}

// Optimization represents an actionable suggestion from AI or heuristic analyzer
type Optimization struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"` // performance, devops, architecture, ai_readiness
	Impact      string   `json:"impact"`   // HIGH, MEDIUM, LOW
	ActionSteps []string `json:"action_steps"`
	PromptForAI string   `json:"prompt_for_ai"` // Copy-paste prompt ready for AI agents
}

// RouteSimulation represents a real-time and synthetic execution analysis of an individual route
type RouteSimulation struct {
	Method                 string  `json:"method"`
	Path                   string  `json:"path"`
	Handler                string  `json:"handler"`
	Component              string  `json:"component"`
	SimulatedLatencyMs     float64 `json:"simulated_latency_ms"`
	RealLatencyMs          float64 `json:"real_latency_ms,omitempty"`
	LatencyStatus          string  `json:"latency_status"` // "FAST", "NORMAL", "SLOW", "CRITICAL"
	HasFormValidation      bool    `json:"has_form_validation"`
	HasRateLimiter         bool    `json:"has_rate_limiter"`
	QueryVulnerabilityRisk string  `json:"query_vulnerability_risk"` // "LOW", "MEDIUM", "HIGH"
	Details                string  `json:"details,omitempty"`
}

// DeepCheckReport represents an in-depth audit report with execution time simulation, form validation, and query vulnerability checks
type DeepCheckReport struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	ProjectID             uint      `gorm:"index;not null" json:"project_id"`
	Score                 int       `json:"score"`
	Summary               string    `gorm:"type:longtext" json:"summary"`
	FindingsJSON          string    `gorm:"type:longtext" json:"findings_json"`
	RouteSimulationsJSON  string    `gorm:"type:longtext" json:"route_simulations_json"`
	GitCommitHash         string    `gorm:"size:64" json:"git_commit_hash"`
	ProjectChecksum       string    `gorm:"size:64" json:"project_checksum"`
	TotalRoutesChecked    int       `json:"total_routes_checked"`
	TotalFunctionsChecked int       `json:"total_functions_checked"`
	TotalFindings         int       `json:"total_findings"`
	CreatedAt             time.Time `json:"created_at"`
}

// externalFindingPrefixes: awalan rule temuan yang TIDAK dihasilkan audit statis (Deep Check, uji runtime, review
// LLM/agent, ronda pegawai). Temuan seperti ini tidak ikut dinilai ulang saat audit statis dijalankan.
var externalFindingPrefixes = []string{"DEEP-", "RUNTIME-", "LLM-REVIEW-", "PROBE-", "PEGANGAN-"}

// IsStaticAuditFinding: temuan berasal dari audit statis (dipindai ulang setiap audit proyek).
func IsStaticAuditFinding(f Finding) bool {
	if f.IsDeep || f.IsLLM {
		return false
	}
	for _, p := range externalFindingPrefixes {
		if strings.HasPrefix(f.RuleID, p) {
			return false
		}
	}
	return true
}

func isOpenFinding(f Finding) bool { return f.Status == "" || f.Status == FindingStatusOpen }

// ReconcileStaticFindings menyatukan temuan audit sebelumnya dengan hasil audit statis terbaru:
//
//   - temuan statis yang muncul lagi mempertahankan status, vonis, dan todo-nya; bila kodenya hanya bergeser baris
//     (rule, berkas, dan potongan kode sama) temuan lama DIPINDAH ke baris baru, bukan dihitung sebagai temuan baru;
//   - temuan statis TERBUKA yang tidak muncul lagi dibuang: kodenya sudah berubah atau aturannya sudah diperbaiki,
//     jadi tidak boleh lagi menjadi dasar todo;
//   - temuan statis yang sudah diselesaikan/diabaikan dan temuan dari pemeriksaan lain (Deep Check, runtime,
//     review) dipertahankan apa adanya.
func ReconcileStaticFindings(prev, fresh []Finding) []Finding {
	seen := make([]bool, len(prev))
	type moveKey struct{ rule, file, snippet string }
	byMove := map[moveKey][]int{}
	for i, p := range prev {
		if h := AuditSnippetHash(p.Snippet); h != "" && IsStaticAuditFinding(p) {
			k := moveKey{p.RuleID, p.TargetFile, h}
			byMove[k] = append(byMove[k], i)
		}
	}
	matchPrev := func(f Finding) int {
		for i, p := range prev {
			if !seen[i] && AreFindingsMatching(p, f) {
				return i
			}
		}
		return -1
	}

	out := make([]Finding, 0, len(prev)+len(fresh))
	var unmatched []Finding
	for _, f := range fresh {
		if f.Status == "" {
			f.Status = FindingStatusOpen
		}
		i := matchPrev(f)
		if i < 0 {
			unmatched = append(unmatched, f)
			continue
		}
		seen[i] = true
		out = append(out, carryFindingState(prev[i], f))
	}
	// Sisa temuan baru: cari temuan lama yang sama tetapi barisnya bergeser.
	for _, f := range unmatched {
		moved := -1
		if h := AuditSnippetHash(f.Snippet); h != "" {
			for _, i := range byMove[moveKey{f.RuleID, f.TargetFile, h}] {
				if !seen[i] {
					moved = i
					break
				}
			}
		}
		if moved < 0 {
			if f.ID == "" {
				f.ID = GenerateFindingKey(f)
			}
			out = append(out, f)
			continue
		}
		seen[moved] = true
		out = append(out, carryFindingState(prev[moved], f))
	}
	for i, p := range prev {
		if seen[i] {
			continue
		}
		if IsStaticAuditFinding(p) && isOpenFinding(p) {
			continue // basi: tidak terdeteksi lagi
		}
		if p.ID == "" {
			p.ID = GenerateFindingKey(p)
		}
		out = append(out, p)
	}
	return out
}

// carryFindingState: temuan terbaru (lokasi, potongan kode, deskripsi) dengan status, vonis, dan hasil penyelesaian
// milik temuan lama yang sama.
func carryFindingState(old, cur Finding) Finding {
	cur.ID = ""
	cur.ID = GenerateFindingKey(cur)
	if !isOpenFinding(old) {
		cur.Status = old.Status
	}
	cur.Verdict, cur.VerdictReason = old.Verdict, old.VerdictReason
	cur.CopyResolution(Finding{
		Status: cur.Status, ResolutionTitle: old.ResolutionTitle, Explanation: old.Explanation, ActionSteps: old.ActionSteps,
		CodeBefore: old.CodeBefore, CodeAfter: old.CodeAfter, TargetLines: old.TargetLines, PlacementPlan: old.PlacementPlan,
		ModelUsed: old.ModelUsed, TheoryBasis: firstNonEmptyString(old.TheoryBasis, cur.TheoryBasis), TodoID: old.TodoID, ResolvedAt: old.ResolvedAt,
	})
	if cur.Status == "" {
		cur.Status = FindingStatusOpen
	}
	return cur
}

func firstNonEmptyString(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
