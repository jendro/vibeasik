package models

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// Memori todo: ringkasan pendek (≤ TodoMemoryMax karakter) riwayat pengerjaan sebuah todo — berapa kali dicoba,
// nilai QA tiap percobaan, hasil percobaan terakhir, berkas yang disentuh, dan catatan QA terakhir. Dibangun
// tanpa LLM dari catatan penyelesaian di Rationale + EmployeeTask, jadi murah & selalu konsisten dengan datanya.
// Prompt agent memakai memori + laporan percobaan terakhir saja, bukan seluruh riwayat mentah yang terus
// membengkak; memori juga menjadi teks indeks pencarian todo serupa (internal/todorag).

const (
	TodoMemoryMax      = 900
	completionNoteMark = "Catatan penyelesaian ("
)

// CompletionNote adalah satu catatan penyelesaian yang ditambahkan update_todo ke Rationale.
type CompletionNote struct {
	Who  string
	When string
	Text string
}

// SplitRationale memisahkan analisis asli todo dari catatan penyelesaian yang ditambahkan tiap percobaan.
func SplitRationale(r string) (analysis string, notes []CompletionNote) {
	parts := strings.Split(r, completionNoteMark)
	analysis = strings.TrimSpace(parts[0])
	for _, p := range parts[1:] {
		head, text, ok := strings.Cut(p, "): ")
		if !ok {
			analysis = strings.TrimSpace(analysis + "\n" + completionNoteMark + p)
			continue
		}
		who, when, _ := strings.Cut(head, ", ")
		notes = append(notes, CompletionNote{Who: strings.TrimSpace(who), When: strings.TrimSpace(when), Text: strings.TrimSpace(text)})
	}
	return analysis, notes
}

// LatestCompletionNote: catatan penyelesaian percobaan terakhir ("" bila belum pernah dikerjakan).
func (t Todo) LatestCompletionNote() string {
	_, notes := SplitRationale(t.Rationale)
	if len(notes) == 0 {
		return ""
	}
	n := notes[len(notes)-1]
	return fmt.Sprintf("(%s, %s) %s", n.Who, n.When, n.Text)
}

// RationaleForPrompt: analisis asli (dipotong analysisMax) + catatan percobaan terakhir (dipotong noteMax).
// Pengganti clip(t.Rationale, n) yang selama ini justru mengambil catatan percobaan PERTAMA.
func (t Todo) RationaleForPrompt(analysisMax, noteMax int) string {
	analysis, notes := SplitRationale(t.Rationale)
	out := clipRunes(analysis, analysisMax)
	if len(notes) > 0 {
		n := notes[len(notes)-1]
		label := "Catatan penyelesaian percobaan terakhir"
		if len(notes) > 1 {
			label = fmt.Sprintf("Catatan penyelesaian percobaan terakhir (ke-%d dari %d)", len(notes), len(notes))
		}
		out = strings.TrimSpace(out + fmt.Sprintf("\n\n%s (%s, %s):\n%s", label, n.Who, n.When, clipRunes(n.Text, noteMax)))
	}
	return out
}

var notePathRe = regexp.MustCompile(`[A-Za-z0-9_./-]+\.(?:go|vue|ts|tsx|js|jsx|mjs|sql|py|php|dart|kt|java|rb|rs|css|scss|yaml|yml|json|md|html)\b`)

// BuildTodoMemory menyusun memori todo dari catatan penyelesaian dan nilai QA tiap percobaan (urut lama → baru).
func BuildTodoMemory(t Todo, scores []int, lastQA string) string {
	_, notes := SplitRationale(t.Rationale)
	if len(notes) == 0 && len(scores) == 0 {
		return ""
	}
	var b strings.Builder
	n := max(len(notes), len(scores))
	fmt.Fprintf(&b, "Percobaan: %d", n)
	if len(notes) > 0 {
		first, last := notes[0].When, notes[len(notes)-1].When
		if first == last {
			fmt.Fprintf(&b, " (%s)", last)
		} else {
			fmt.Fprintf(&b, " (%s – %s)", first, last)
		}
	}
	if len(scores) > 0 {
		s := make([]string, len(scores))
		for i, v := range scores {
			s[i] = strconv.Itoa(v)
		}
		b.WriteString(" · nilai QA: " + strings.Join(s, "→"))
	}
	b.WriteString("\n")
	if len(notes) > 0 {
		last := notes[len(notes)-1]
		fmt.Fprintf(&b, "Terakhir (%s, %s): %s\n", last.Who, last.When, clipRunes(noteOutcome(last.Text), 260))
		if files := noteFiles(notes, 6); len(files) > 0 {
			b.WriteString("Berkas: " + strings.Join(files, ", ") + "\n")
		}
	}
	if q := strings.Join(strings.Fields(lastQA), " "); q != "" {
		b.WriteString("QA terakhir: " + clipRunes(q, 300) + "\n")
	}
	return clipRunes(strings.TrimSpace(b.String()), TodoMemoryMax)
}

// noteOutcome: baris HASIL bila ada, selain itu kalimat isi pertama (judul markdown dilewati).
func noteOutcome(text string) string {
	first := ""
	for _, line := range strings.Split(text, "\n") {
		l := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#*-> "))
		if l == "" {
			continue
		}
		if up := strings.ToUpper(l); strings.HasPrefix(up, "HASIL") {
			if _, v, ok := strings.Cut(l, ":"); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
		if first == "" && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			first = l
		}
	}
	if first == "" {
		first = strings.TrimSpace(text)
	}
	return strings.Join(strings.Fields(first), " ")
}

// noteFiles: berkas yang disebut catatan penyelesaian, catatan terbaru didahulukan.
func noteFiles(notes []CompletionNote, n int) []string {
	seen := map[string]bool{}
	var out []string
	for i := len(notes) - 1; i >= 0 && len(out) < n; i-- {
		for _, f := range notePathRe.FindAllString(notes[i].Text, -1) {
			f = strings.Trim(f, "./")
			if f == "" || seen[f] || !strings.ContainsAny(f, "/_") && !strings.Contains(f, ".") {
				continue
			}
			seen[f] = true
			out = append(out, f)
			if len(out) >= n {
				break
			}
		}
	}
	return out
}

// RefreshTodoMemory menghitung ulang memori satu todo dan menyimpannya bila berubah.
func RefreshTodoMemory(db *gorm.DB, todoID uint) string {
	var t Todo
	if db.Select("id", "rationale", "memory").First(&t, todoID).Error != nil {
		return ""
	}
	mem := todoMemoryFromDB(db, t)
	if mem != t.Memory {
		db.Model(&Todo{}).Where("id = ?", todoID).Update("memory", mem)
	}
	return mem
}

// BackfillTodoMemory mengisi memori todo lama yang pernah dikerjakan tetapi belum punya memori. updated_at
// tidak disentuh agar urutan "terakhir diubah" todo tetap apa adanya.
func BackfillTodoMemory(db *gorm.DB) int {
	var rows []Todo
	db.Select("id", "rationale", "memory").Where("(memory = '' OR memory IS NULL)").
		Where("rationale LIKE ? OR id IN (?)", "%"+completionNoteMark+"%",
			db.Model(&EmployeeTask{}).Select("todo_id").Where("score IS NOT NULL")).Find(&rows)
	n := 0
	for _, t := range rows {
		if mem := todoMemoryFromDB(db, t); mem != "" {
			db.Model(&Todo{}).Where("id = ?", t.ID).UpdateColumn("memory", mem)
			n++
		}
	}
	return n
}

func todoMemoryFromDB(db *gorm.DB, t Todo) string {
	var tasks []EmployeeTask
	db.Select("id", "score", "qa_comment").Where("todo_id = ? AND score IS NOT NULL", t.ID).Order("id ASC").Find(&tasks)
	scores := make([]int, 0, len(tasks))
	lastQA := ""
	for _, k := range tasks {
		scores = append(scores, *k.Score)
		lastQA = k.QAComment
	}
	return BuildTodoMemory(t, scores, lastQA)
}

func clipRunes(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if n <= 0 || len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
