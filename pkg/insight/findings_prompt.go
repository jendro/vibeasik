package insight

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/diagnostics"
	"github.com/vibeasik/vibeasik/pkg/models"
)

// Sumber temuan per halaman pemeriksaan. Ketiganya ada di himpunan temuan audit yang sama
// (models.CurrentProjectFindings); yang membedakan hanya awalan rule-nya.
const (
	FindingScopeAudit   = "audit"
	FindingScopeCost    = "cost"
	FindingScopeRuntime = "runtime"
)

// FindingScope: satu sumber temuan dan halaman Studio tempat temuannya ditinjau.
type FindingScope struct {
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	Page         string   `json:"page"`
	RulePrefixes []string `json:"rule_prefixes,omitempty"` // kosong = semua temuan
}

// FindingScopes: urutan sama dengan menu "Periksa & tingkatkan".
var FindingScopes = []FindingScope{
	{Key: FindingScopeAudit, Label: "audit", Page: "/audit-report"},
	{Key: FindingScopeCost, Label: "biaya proses (Big-O & N+1)", Page: "/process-cost", RulePrefixes: []string{"COST-", "QUERY-NPLUS1"}},
	{Key: FindingScopeRuntime, Label: "uji runtime", Page: "/runtime-test", RulePrefixes: []string{"RUNTIME-"}},
}

// FindingScopeFor mengembalikan sumber temuan; kunci kosong/tak dikenal = audit (semua temuan).
func FindingScopeFor(key string) FindingScope {
	k := strings.ToLower(strings.TrimSpace(key))
	for _, s := range FindingScopes {
		if s.Key == k {
			return s
		}
	}
	return FindingScopes[0]
}

// Match: temuan ini termasuk sumber s.
func (s FindingScope) Match(f models.Finding) bool {
	if len(s.RulePrefixes) == 0 {
		return true
	}
	rule := strings.ToUpper(f.RuleID)
	for _, p := range s.RulePrefixes {
		if strings.HasPrefix(rule, p) {
			return true
		}
	}
	return false
}

// FindingsTodoPrompt: prompt siap salin untuk agent coding pengguna — menjadikan temuan satu sumber todo lewat MCP
// vibeasik, dengan agent sendiri yang membaca kode (tanpa kuota LLM luar). Teksnya sama untuk tombol "Salin prompt
// agent" di Studio dan tool MCP vibeasik_findings_todo_prompt.
type FindingsTodoPrompt struct {
	ProjectID   uint         `json:"project_id"`
	ProjectName string       `json:"project_name"`
	Scope       FindingScope `json:"scope"`
	Open        int          `json:"open"`     // temuan terbuka (belum jadi todo, tidak diabaikan)
	Critical    int          `json:"critical"` // di antaranya kritis
	// Advisory: temuan terbuka berupa perkiraan / keadaan sesaat (estimasi O(n²) statis, git belum di-commit,
	// port bawaan) — tidak ikut triase agent; tetap bisa dijadikan todo satu per satu.
	Advisory int    `json:"advisory"`
	Prompt   string `json:"prompt"`
}

// BuildFindingsTodoPrompt menyusun prompt untuk proyek & sumber temuan. Hanya membaca.
func BuildFindingsTodoPrompt(db *gorm.DB, projectID uint, scopeKey string) (*FindingsTodoPrompt, error) {
	var project models.Project
	if err := db.Select("id", "name").First(&project, projectID).Error; err != nil {
		return nil, fmt.Errorf("proyek %d tidak ditemukan", projectID)
	}
	scope := FindingScopeFor(scopeKey)
	findings, _, err := models.CurrentProjectFindings(db, projectID)
	if err != nil {
		return nil, err
	}
	out := &FindingsTodoPrompt{ProjectID: project.ID, ProjectName: project.Name, Scope: scope}
	for _, f := range findings {
		if scope.Match(f) && models.IsOpenFinding(f) {
			out.Open++
			if strings.EqualFold(f.Severity, "CRITICAL") {
				out.Critical++
			}
			if diagnostics.IsAdvisoryFinding(f) {
				out.Advisory++
			}
		}
	}
	out.Prompt = findingsTodoPromptText(project, scope, out.Advisory)
	return out, nil
}

func findingsTodoPromptText(p models.Project, s FindingScope, advisory int) string {
	filter := fmt.Sprintf("project_id: %d", p.ID)
	if len(s.RulePrefixes) > 0 {
		filter += fmt.Sprintf(`, rule_prefix: "%s"`, strings.Join(s.RulePrefixes, ","))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Jadikan todo temuan %s proyek \"%s\" (project ID %d) lewat MCP vibeasik.\n", s.Label, p.Name, p.ID)
	b.WriteString("Kerjakan sendiri dengan membaca kodenya: jangan memakai LLM luar, jangan menjalankan scan/audit ulang, dan jangan mengubah kode aplikasi — tugas ini hanya memeriksa temuan lalu membuat todo.\n\n")
	fmt.Fprintf(&b, "1. Panggil `vibeasik_audit_triage_brief` {%s}. Hasilnya: kelompok temuan terbuka per rule + berkas, apa yang harus dibuktikan (must_verify), rekam jejak false positive rule itu, dan todo terbuka yang akan digabung.\n", filter)
	b.WriteString("2. Untuk tiap kelompok, baca berkas di baris yang disebut beserta pemanggil/rutenya. Putuskan `confirmed` bila masalahnya nyata, `false_positive` bila tidak.\n")
	b.WriteString("3. Kirim keputusan lewat `vibeasik_audit_triage_submit` {project_id, agent: \"<nama agent & model>\", decisions}. `confirmed` wajib membawa resolution (todo_title, target_file, target_lines, code_before disalin persis dari berkas, code_after, action_steps); `false_positive` membawa reason + evidence (kutipan baris dari berkas).\n")
	b.WriteString("4. Ulangi langkah 1–3 dengan offset berikutnya sampai brief kosong. Todo di stale_todos (halaman pertama) yang temuannya sudah hilang ditutup dengan `update_todo` status \"rejected\".\n")
	b.WriteString("5. Laporkan: jumlah todo dibuat / digabung, false positive beserta alasannya, dan kelompok yang dilewati.\n\n")
	if advisory > 0 {
		fmt.Fprintf(&b, "Catatan: %d temuan bersifat perkiraan atau keadaan sesaat (mis. estimasi O(n²) statis, perubahan git belum di-commit) dan sengaja tidak ikut triase. ", advisory)
		if s.Key == FindingScopeCost {
			b.WriteString("Untuk biaya proses, tombol \"Jadikan semua todo\" di halaman Biaya Proses membuat todonya dari hitungan kode tanpa LLM; ")
		}
		b.WriteString("bila tetap perlu, periksa kodenya lalu buat todo dengan `create_todo`.\n\n")
	}
	fmt.Fprintf(&b, "Bila tool `vibeasik_audit_triage_*` tidak tersedia: ambil temuan dengan `get_latest_diagnostic` {project_id: %d}", p.ID)
	if len(s.RulePrefixes) > 0 {
		fmt.Fprintf(&b, " (hanya rule berawalan %s)", strings.Join(s.RulePrefixes, ", "))
	}
	b.WriteString(", periksa kodenya, buat satu todo per temuan yang terbukti dengan `create_todo` (judul, berkas:baris, langkah perbaikan), dan tandai temuan yang keliru dengan `set_finding_status` status \"ignored\".\n")
	return b.String()
}

// ResolveProjectRef mencari proyek dari ID atau nama/slug yang ditulis bebas ("bimtekapsi", "Bimtek APSI").
func ResolveProjectRef(db *gorm.DB, ref string) (uint, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0, fmt.Errorf("sebutkan project_id atau nama proyek")
	}
	if id, err := strconv.ParseUint(ref, 10, 64); err == nil && id > 0 {
		return uint(id), nil
	}
	var projects []models.Project
	if err := db.Select("id", "name", "slug").Find(&projects).Error; err != nil {
		return 0, err
	}
	want := compactName(ref)
	var hits []models.Project
	for _, p := range projects {
		if compactName(p.Name) == want || compactName(p.Slug) == want {
			return p.ID, nil
		}
		if strings.Contains(compactName(p.Name), want) || strings.Contains(compactName(p.Slug), want) {
			hits = append(hits, p)
		}
	}
	switch len(hits) {
	case 1:
		return hits[0].ID, nil
	case 0:
		return 0, fmt.Errorf("proyek %q tidak ditemukan; lihat list_projects", ref)
	}
	names := make([]string, 0, len(hits))
	for _, h := range hits {
		names = append(names, fmt.Sprintf("%s (#%d)", h.Name, h.ID))
	}
	return 0, fmt.Errorf("nama %q cocok dengan beberapa proyek: %s — sebutkan project_id", ref, strings.Join(names, ", "))
}

// compactName: huruf kecil tanpa spasi/tanda baca, agar "Bimtek APSI", "bimtek-apsi" dan "bimtekapsi" sama.
func compactName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
