package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Protokol "pengetahuan vibeasik dulu" (Aturan 47): agent coding luar (Antigravity, Claude Code, Cursor, Codex, …)
// membawa pengetahuan, skill, dan selera desainnya sendiri. Protokol ini memerintahkan mereka mengambil pengetahuan
// dari vibeasik lebih dulu dan menjadikannya sumber kebenaran. Dikirim lewat tiga jalur agar tidak bergantung pada
// satu agent: `instructions` MCP initialize, `.vibeasik.json`, dan blok aturan di berkas yang selalu dibaca agent
// (GEMINI.md, AGENTS.md, CLAUDE.md bila ada) yang ditulis "Init vibeasik".

// KnowledgeFirst: versi satu paragraf (MCP instructions & .vibeasik.json).
const KnowledgeFirst = "PENGETAHUAN VIBEASIK DIDAHULUKAN (wajib): untuk proyek yang dikelola vibeasik, sumber kebenaran keputusan teknis & desain adalah pengetahuan vibeasik — bukan pengetahuan bawaan model, skill/rules desain milik agent, atau selera sendiri. " +
	"(a) Sebelum merancang, menulis, atau me-review kode: panggil 'vibeasik_knowledge_recall' {query: tugas + berkas target} dan ikuti butirnya (label K<id>). " +
	"(b) Tugas tampilan (UI/UX, warna, layout, tipografi, komponen, layar mobile): panggil 'vibeasik_design_guidance' {task, project_id} PERTAMA; 'MODE GALERI MURNI' = hanya angka & pola galeri. Skill desain agent dan default Tailwind/Bootstrap/Material tidak dipakai bila bertentangan. " +
	"(c) Kedudukan: Galeri Desain (kesimpulan & pola teramati) > patokan & pengetahuan vibeasik > teori desain vibeasik ('vibeasik_design_theory') > pengetahuan umum model. Bila bertentangan, ikuti vibeasik dan sebutkan bedanya. " +
	"(d) Bila vibeasik belum punya jawabannya: boleh memakai pengetahuan sendiri, tandai '(pengetahuan umum)' di laporan dan laporkan celahnya lewat 'vibeasik_knowledge_gap'. Butir yang terbukti keliru → 'vibeasik_knowledge_feedback' {node_id, helpful: false}; pelajaran baru yang berguna lintas proyek → 'vibeasik_knowledge_learn'. " +
	"(e) Laporan akhir / completion_note memuat baris 'PENGETAHUAN: K<id>, …' (butir yang dipakai) dan, untuk tampilan, 'DESAIN: <referensi #id> · <prinsip> · <penyimpangan beralasan>'."

// Penanda blok terkelola: isi di antaranya ditulis ulang setiap Init; teks pengguna di luar blok tidak disentuh.
const (
	rulesBegin = "<!-- vibeasik:knowledge-first:begin (dikelola vibeasik — jangan diedit, ditulis ulang saat Init vibeasik) -->"
	rulesEnd   = "<!-- vibeasik:knowledge-first:end -->"
)

// AgentRulesMarkdown: protokol dalam bentuk aturan Markdown untuk berkas konteks agent.
func AgentRulesMarkdown(projectID uint, projectName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Pengetahuan vibeasik didahulukan (WAJIB)\n\n")
	fmt.Fprintf(&b, "Proyek **%s** dikelola vibeasik (Project ID **%d**, MCP server `vibeasik`). Sumber kebenaran untuk keputusan teknis & desain adalah pengetahuan vibeasik — **bukan** pengetahuan bawaan model, skill/rules desain milik agent, atau selera sendiri.\n\n", projectName, projectID)
	b.WriteString("1. **Sebelum merancang, menulis, atau me-review kode**: panggil `vibeasik_knowledge_recall` `{query: \"<tugas> <berkas target>\"}` dan ikuti butirnya (label `K<id>`). Cari lokasi kode lewat `vibeasik_code_map_search`.\n")
	fmt.Fprintf(&b, "2. **Tugas tampilan** (UI/UX, warna, layout, tipografi, komponen, layar mobile): panggil `vibeasik_design_guidance` `{task: \"<tugas>\", project_id: %d}` **PERTAMA**. Bila hasilnya \"MODE GALERI MURNI\", hanya angka & pola dari galeri yang dipakai. Skill desain agent (mis. frontend-design, antislop-ui) dan default Tailwind/Bootstrap/Material **tidak dipakai** bila bertentangan dengan galeri.\n", projectID)
	b.WriteString("3. **Urutan kedudukan**: Galeri Desain (kesimpulan & pola teramati) > patokan & pengetahuan vibeasik > teori desain vibeasik (`vibeasik_design_theory`) > pengetahuan umum model. Bila bertentangan, ikuti vibeasik dan sebutkan bedanya di laporan.\n")
	b.WriteString("4. **Bila vibeasik belum punya jawabannya**: boleh memakai pengetahuan sendiri, tandai `(pengetahuan umum)` di laporan, dan laporkan celahnya lewat `vibeasik_knowledge_gap` `{topic}` agar vibeasik mempelajarinya.\n")
	b.WriteString("5. **Butir vibeasik keliru**: jangan diam-diam diabaikan — `vibeasik_knowledge_feedback` `{node_id, helpful: false}`. Pelajaran baru yang berguna lintas proyek → `vibeasik_knowledge_learn`.\n")
	b.WriteString("6. **Laporan akhir / `completion_note`** memuat baris `PENGETAHUAN: K<id>, …` (butir yang dipakai) dan untuk tampilan `DESAIN: <referensi #id> · <prinsip yang diikuti> · <penyimpangan beralasan>`.\n")
	fmt.Fprintf(&b, "7. **Todo**: ambil lewat `list_todos` `{project_id: %d}` dan ikuti alur kerja todo vibeasik (instruksi MCP server & `.vibeasik.json`).\n\n", projectID)
	b.WriteString("Bila MCP `vibeasik` tidak tersedia, berhenti dan minta pengguna menyalakan `vibeasik serve` / mendaftarkan MCP-nya — jangan lanjut hanya dengan pengetahuan sendiri.\n")
	return b.String()
}

// agentRuleFiles: berkas yang selalu dibaca agent coding. GEMINI.md (Antigravity/Gemini CLI) & AGENTS.md (Codex,
// Cursor, Antigravity, dll.) dibuat bila belum ada; CLAUDE.md hanya diperbarui bila proyek sudah memakainya.
var agentRuleFiles = []struct {
	name   string
	create bool
}{{"GEMINI.md", true}, {"AGENTS.md", true}, {"CLAUDE.md", false}}

// WriteAgentRules menulis/memperbarui blok protokol di berkas konteks agent pada root proyek. Aman diulang.
func WriteAgentRules(root string, projectID uint, projectName string) ([]string, error) {
	if root == "" {
		return nil, nil
	}
	block := rulesBegin + "\n" + AgentRulesMarkdown(projectID, projectName) + rulesEnd + "\n"
	var written []string
	for _, f := range agentRuleFiles {
		path := filepath.Join(root, f.name)
		old, err := os.ReadFile(path)
		if err != nil && !(os.IsNotExist(err) && f.create) {
			continue
		}
		next := upsertBlock(string(old), block)
		if next == string(old) {
			continue
		}
		if err := os.WriteFile(path, []byte(next), 0644); err != nil {
			return written, fmt.Errorf("gagal menulis %s: %w", path, err)
		}
		written = append(written, path)
	}
	return written, nil
}

// upsertBlock mengganti blok terkelola bila ada; selain itu menaruhnya di awal berkas (aturan di atas lebih dulu
// dibaca dan tidak tenggelam di bawah dokumentasi panjang).
func upsertBlock(content, block string) string {
	if i := strings.Index(content, rulesBegin); i >= 0 {
		if j := strings.Index(content[i:], rulesEnd); j >= 0 {
			end := i + j + len(rulesEnd)
			if end < len(content) && content[end] == '\n' {
				end++
			}
			return content[:i] + block + content[end:]
		}
	}
	if strings.TrimSpace(content) == "" {
		return block
	}
	return block + "\n" + content
}
