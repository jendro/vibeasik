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
// (GEMINI.md, AGENTS.md, CLAUDE.md bila ada, .clinerules untuk Cline) yang ditulis "Init vibeasik".

// KnowledgeFirst: versi satu paragraf (MCP instructions & .vibeasik.json).
const KnowledgeFirst = "PENGETAHUAN VIBEASIK DIDAHULUKAN (wajib): untuk proyek yang dikelola vibeasik, sumber kebenaran keputusan teknis & desain adalah pengetahuan vibeasik — bukan pengetahuan bawaan model, skill/rules desain milik agent, atau selera sendiri. " +
	"(a) Sebelum merancang, menulis, atau me-review kode: panggil 'vibeasik_knowledge_recall' {query: tugas + berkas target} dan ikuti butirnya (label K<id>). " +
	"(b) Tugas tampilan (UI/UX, warna, layout, tipografi, komponen, layar mobile): panggil 'vibeasik_design_guidance' {task, project_id} PERTAMA; 'MODE GALERI MURNI' = hanya angka & pola galeri. Skill desain agent dan default Tailwind/Bootstrap/Material tidak dipakai bila bertentangan. " +
	"(c) Kedudukan: Galeri Desain (kesimpulan & pola teramati) > patokan & pengetahuan vibeasik, termasuk pengetahuan PEGAWAI vibeasik yang ditugaskan pada todo > teori desain vibeasik ('vibeasik_design_theory') > pengetahuan umum model. Bila bertentangan, ikuti vibeasik dan sebutkan bedanya. " +
	"(c2) Todo vibeasik dikerjakan berpasangan dengan PEGAWAI yang ditugaskan (pair programming): kamu driver, pegawai navigator & mentor. Ambil brief-nya ('update_todo' in_progress mengembalikannya, atau 'vibeasik_staff_brief' {todo_id}) SEBELUM menyentuh kode, lalu konsultasikan lewat 'vibeasik_staff_consult' {todo_id, phase}: 'plan' sebelum edit pertama, 'stuck' saat buntu, 'review' sebelum melapor. Standar & pengetahuannya di atas kebiasaan model sendiri. " +
	"(d) Bila vibeasik belum punya jawabannya: boleh memakai pengetahuan sendiri, tandai '(pengetahuan umum)' di laporan dan laporkan celahnya lewat 'vibeasik_knowledge_gap'. Butir yang terbukti keliru → 'vibeasik_knowledge_feedback' {node_id, helpful: false}; pelajaran baru yang berguna lintas proyek → 'vibeasik_knowledge_learn'. " +
	"(e) Laporan akhir / completion_note memuat baris 'PEGAWAI: <nama — jabatan>' dan 'KONSULTASI: plan #id · review #id' (untuk todo), 'PENGETAHUAN: K<id>, …' (butir yang dipakai) dan, untuk tampilan, 'DESAIN: <referensi #id> · <prinsip> · <penyimpangan beralasan>'."

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
	b.WriteString("3. **Urutan kedudukan**: bagian **PATOKAN UTAMA** di hasil `vibeasik_design_guidance` (mis. gaya admin untuk tugas admin/dashboard) > Galeri Desain (kesimpulan & pola teramati) > patokan & pengetahuan vibeasik, termasuk pengetahuan **pegawai vibeasik** yang ditugaskan pada todo > teori desain vibeasik (`vibeasik_design_theory`) > pengetahuan umum model. Bila bertentangan, ikuti vibeasik dan sebutkan bedanya di laporan.\n")
	b.WriteString("4. **Bila vibeasik belum punya jawabannya**: boleh memakai pengetahuan sendiri, tandai `(pengetahuan umum)` di laporan, dan laporkan celahnya lewat `vibeasik_knowledge_gap` `{topic}` agar vibeasik mempelajarinya.\n")
	b.WriteString("5. **Butir vibeasik keliru**: jangan diam-diam diabaikan — `vibeasik_knowledge_feedback` `{node_id, helpful: false}`. Pelajaran baru yang berguna lintas proyek → `vibeasik_knowledge_learn`.\n")
	b.WriteString("6. **Laporan akhir / `completion_note`** memuat baris `PEGAWAI: <nama — jabatan>` dan `KONSULTASI: plan #id · review #id` (untuk todo), `PENGETAHUAN: K<id>, …` (butir yang dipakai) dan untuk tampilan `DESAIN: <referensi #id> · <prinsip yang diikuti> · <penyimpangan beralasan>`.\n")
	fmt.Fprintf(&b, "7. **Todo**: ambil lewat `list_todos` `{project_id: %d}` dan ikuti alur kerja todo vibeasik (instruksi MCP server & `.vibeasik.json`). Perintah \"kerjakan todo di vibeasik\" = panggil `vibeasik_todo_dispatch_plan` `{project_id: %d, coordinator: \"<agent key-mu: claude/agy/cline/…>\"}` lalu ikuti `coordinator_steps`; Claude Code & Antigravity menjalankan satu sub agent baru per todo di aplikasinya dengan `subagent_launch` (sub agent membaca `subagent_prompt_file` berisi brief pegawai); agent tanpa sub agent (Cline, …) otomatis memakai mode worker: tiap todo dikirim lewat `vibeasik_dispatch_agent` ke proses agent baru yang bekerja sebagai pegawai di kolom `employee`.\n", projectID, projectID)
	b.WriteString("8. **Todo dikerjakan berpasangan dengan pegawai vibeasik (pair programming)**: kamu *driver*, pegawai yang ditugaskan *navigator & mentor*. Sebelum menyentuh kode, set todo `in_progress` lewat `update_todo` (hasilnya memuat brief pegawai) atau panggil `vibeasik_staff_brief` `{todo_id}`. Konsultasi wajib lewat `vibeasik_staff_consult` `{todo_id, phase}`: `plan` sebelum edit pertama, `stuck` saat buntu atau hendak menyerah, `review` (repo, branch, base) sebelum melapor — konsultasi yang dilewati memotong nilai QA. Pengetahuan pegawai berkedudukan di atas pengetahuan umum model; bila kamu tidak setuju, ikuti pegawai dan tulis alasannya di laporan. Tulis bagian `PELAJARAN:` di `completion_note`; QA menilainya.\n\n")
	b.WriteString("Bila MCP `vibeasik` tidak tersedia, berhenti dan minta pengguna menyalakan `vibeasik serve` / mendaftarkan MCP-nya — jangan lanjut hanya dengan pengetahuan sendiri.\n")
	return b.String()
}

// agentRuleFiles: berkas yang selalu dibaca agent coding. GEMINI.md (Antigravity/Gemini CLI) & AGENTS.md (Codex,
// Cursor, Antigravity, dll.) dibuat bila belum ada; CLAUDE.md hanya diperbarui bila proyek sudah memakainya.
// .clinerules (Cline) boleh berupa berkas atau folder: berkas lama diberi blok, selain itu blok ditulis ke
// .clinerules/vibeasik.md agar aturan Cline milik pengguna di folder itu tidak tersentuh.
var agentRuleFiles = []struct {
	name   string
	create bool
}{{"GEMINI.md", true}, {"AGENTS.md", true}, {"CLAUDE.md", false}, {clineRules, true}}

const clineRules = ".clinerules"

// ruleFilePath: lokasi berkas aturan; untuk Cline, folder .clinerules (atau belum ada) → .clinerules/vibeasik.md.
func ruleFilePath(root, name string) (string, error) {
	path := filepath.Join(root, name)
	if name != clineRules {
		return path, nil
	}
	if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
		return path, nil
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("gagal membuat %s: %w", path, err)
	}
	return filepath.Join(path, "vibeasik.md"), nil
}

// WriteAgentRules menulis/memperbarui blok protokol di berkas konteks agent pada root proyek. Aman diulang.
func WriteAgentRules(root string, projectID uint, projectName string) ([]string, error) {
	if root == "" {
		return nil, nil
	}
	block := rulesBegin + "\n" + AgentRulesMarkdown(projectID, projectName) + rulesEnd + "\n"
	var written []string
	for _, f := range agentRuleFiles {
		path, err := ruleFilePath(root, f.name)
		if err != nil {
			return written, err
		}
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
