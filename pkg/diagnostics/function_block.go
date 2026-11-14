package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Blok fungsi sebagai bukti temuan: satu baris kode jarang cukup untuk memahami masalahnya, jadi todo dan penampil
// perubahan memakai fungsi utuh yang memuat baris itu (dilacak dengan pengukur fungsi yang sama dengan audit
// kompleksitas). Kode di luar fungsi (tingkat atas, template Vue) diberi potongan sekitar baris sebagai gantinya.

// FuncBlock adalah kode yang memuat satu baris temuan: fungsi utuh, atau potongan sekitar baris bila baris itu
// tidak berada di dalam fungsi bernama.
type FuncBlock struct {
	// Name: nama fungsi ("" = baris tidak berada di dalam fungsi bernama; Code berisi potongan sekitar baris).
	Name string `json:"name,omitempty"`
	// Start, End: rentang baris fungsi (1-based) — untuk potongan, rentang potongannya.
	Start int `json:"start"`
	End   int `json:"end"`
	// Line: baris temuan yang ditandai ▶ di Code.
	Line int `json:"line"`
	// Code: baris kode bernomor; baris temuan diawali ▶.
	Code string `json:"code"`
	// Cut: fungsi terlalu panjang sehingga hanya bagian di sekitar baris temuan yang ditulis.
	Cut bool `json:"cut,omitempty"`
}

// Label menyebut isi blok dalam satu frasa, mis. "fungsi `saveUser` (baris 10–42)".
func (b FuncBlock) Label() string {
	if b.Name == "" {
		return fmt.Sprintf("kode di sekitar baris %d (baris %d–%d)", b.Line, b.Start, b.End)
	}
	return fmt.Sprintf("fungsi `%s` (baris %d–%d)", b.Name, b.Start, b.End)
}

const (
	// funcBlockMaxLines: fungsi yang lebih panjang hanya ditulis sekitar baris temuannya.
	funcBlockMaxLines = 60
	// funcBlockAround: baris di atas & di bawah baris temuan untuk potongan.
	funcBlockAround = 8
	// funcBlockMaxFile: berkas yang lebih besar tidak dibaca.
	funcBlockMaxFile = 2 << 20
)

// FunctionBlockFile membaca berkas lalu mengembalikan blok yang memuat baris line (lihat FunctionBlock).
func FunctionBlockFile(path string, line int) (FuncBlock, bool) {
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() || fi.Size() > funcBlockMaxFile {
		return FuncBlock{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return FuncBlock{}, false
	}
	return FunctionBlock(path, string(data), line)
}

// FunctionBlock mengembalikan fungsi terdalam yang memuat baris line di berkas relPath, atau potongan sekitar
// baris itu bila bahasanya tidak dikenal atau baris berada di luar fungsi bernama. false bila baris di luar berkas.
func FunctionBlock(relPath, content string, line int) (FuncBlock, bool) {
	lines := strings.Split(content, "\n")
	if line <= 0 || line > len(lines) {
		return FuncBlock{}, false
	}
	if name, start, end, ok := EnclosingFunction(relPath, lines, line); ok {
		b := FuncBlock{Name: name, Start: start, End: end, Line: line}
		from, to := start, end
		if end-start+1 > funcBlockMaxLines {
			b.Cut = true
			from, to = max(start, line-funcBlockMaxLines/2), min(end, line+funcBlockMaxLines/2)
		}
		b.Code = numberLines(lines, from, to, line, start, end)
		return b, true
	}
	from, to := max(1, line-funcBlockAround), min(len(lines), line+funcBlockAround)
	return FuncBlock{Start: from, End: to, Line: line, Code: numberLines(lines, from, to, line, from, to)}, true
}

// EnclosingFunction: fungsi bernama terdalam yang memuat baris line (1-based) — nama dan rentang barisnya.
func EnclosingFunction(relPath string, lines []string, line int) (name string, start, end int, ok bool) {
	lang := auditFileExtensions[strings.ToLower(filepath.Ext(relPath))]
	src := lines
	if lang == "vue" || lang == "svelte" {
		src, lang = scriptOnly(lines)
	}
	profile, found := complexityProfiles[lang]
	if !found {
		return "", 0, 0, false
	}
	for _, f := range measureFunctions(src, lang, profile) {
		if f.line <= line && line <= f.end && (!ok || f.end-f.line < end-start) {
			name, start, end, ok = f.name, f.line, f.end, true
		}
	}
	return name, start, end, ok
}

var (
	reScriptOpen  = regexp.MustCompile(`(?i)^\s*<script\b[^>]*>`)
	reScriptClose = regexp.MustCompile(`(?i)</script>`)
	reScriptLang  = regexp.MustCompile(`(?i)\blang=["'](ts|tsx|typescript)["']`)
)

// scriptOnly: berkas komponen (Vue/Svelte) dengan semua baris di luar blok <script> dikosongkan, agar fungsinya
// bisa dilacak dengan profil JavaScript/TypeScript tanpa menggeser nomor baris.
func scriptOnly(lines []string) ([]string, string) {
	out := make([]string, len(lines))
	lang, in := "javascript", false
	for i, l := range lines {
		switch {
		case !in && reScriptOpen.MatchString(l):
			in = !reScriptClose.MatchString(l)
			if reScriptLang.MatchString(l) {
				lang = "typescript"
			}
		case in && reScriptClose.MatchString(l):
			in = false
		case in:
			out[i] = l
		}
	}
	return out, lang
}

// numberLines menulis baris from..to (1-based) dengan nomor baris; baris mark diawali ▶. Bila potongan tidak
// mencakup seluruh rentang lo..hi, baris "⋯" menandai bagian yang dilewati.
func numberLines(lines []string, from, to, mark, lo, hi int) string {
	width := len(fmt.Sprint(to))
	var b strings.Builder
	if from > lo {
		fmt.Fprintf(&b, "  %*s ⋯\n", width, "")
	}
	for i := from; i <= to; i++ {
		pin := "  "
		if i == mark {
			pin = "▶ "
		}
		fmt.Fprintf(&b, "%s%*d │ %s\n", pin, width, i, strings.TrimRight(lines[i-1], " \t\r"))
	}
	if to < hi {
		fmt.Fprintf(&b, "  %*s ⋯\n", width, "")
	}
	return strings.TrimRight(b.String(), "\n")
}
