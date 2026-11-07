package codectx

import (
	"fmt"
	"os"
	"strings"
)

// Batas ukuran berkas yang dibaca untuk konteks (berkas hasil build/raksasa dilewati).
const maxSourceBytes = 1 << 20

// sources menyimpan isi berkas yang sudah dibaca selama satu penyusunan konteks.
type sources map[string][]string

func (s sources) lines(abs string) []string {
	if l, ok := s[abs]; ok {
		return l
	}
	var out []string
	if fi, err := os.Stat(abs); err == nil && fi.Size() <= maxSourceBytes {
		if data, err := os.ReadFile(abs); err == nil {
			out = strings.Split(string(data), "\n")
		}
	}
	s[abs] = out
	return out
}

// braceLangs memakai kurung kurawal untuk blok fungsi.
var braceLangs = map[string]bool{
	"go": true, "php": true, "typescript": true, "javascript": true, "vue": true, "svelte": true, "java": true,
	"kotlin": true, "dart": true, "csharp": true, "rust": true, "swift": true,
}

// blockEnd mencari baris penutup fungsi yang dimulai di baris start (1-based): kurung kurawal seimbang untuk
// bahasa berkurung (string & komentar dilewati), indentasi untuk Python, "end" untuk Ruby. limit = batas
// baris (simbol berikutnya) bila struktur tidak bisa dibaca.
func blockEnd(lines []string, start int, lang string, limit int) int {
	if start < 1 || start > len(lines) {
		return start
	}
	if limit <= 0 || limit > len(lines) {
		limit = len(lines)
	}
	switch {
	case lang == "python":
		return indentEnd(lines, start, limit)
	case lang == "ruby":
		return rubyEnd(lines, start, limit)
	case braceLangs[lang]:
		if end := braceEnd(lines, start, limit); end > 0 {
			return end
		}
	}
	return limit
}

func braceEnd(lines []string, start, limit int) int {
	depth, opened := 0, false
	inBlock := false // /* ... */
	var quote rune   // string terbuka; hanya backtick (Go/JS) yang boleh melewati akhir baris
	for i := start - 1; i < limit; i++ {
		l := lines[i]
		for j := 0; j < len(l); j++ {
			c := l[j]
			if inBlock {
				if c == '*' && j+1 < len(l) && l[j+1] == '/' {
					inBlock = false
					j++
				}
				continue
			}
			if quote != 0 {
				if c == '\\' {
					j++
				} else if rune(c) == quote {
					quote = 0
				}
				continue
			}
			switch c {
			case '"', '\'', '`':
				quote = rune(c)
			case '/':
				if j+1 < len(l) && l[j+1] == '/' {
					j = len(l)
				} else if j+1 < len(l) && l[j+1] == '*' {
					inBlock = true
					j++
				}
			case '#':
				if j == 0 || strings.TrimSpace(l[:j]) == "" { // komentar PHP/shell di awal baris
					j = len(l)
				}
			case '{':
				depth++
				opened = true
			case '}':
				depth--
				if opened && depth <= 0 {
					return i + 1
				}
			}
		}
		if quote != '`' {
			quote = 0
		}
	}
	return 0
}

func indentOf(l string) int {
	n := 0
	for _, c := range l {
		switch c {
		case ' ':
			n++
		case '\t':
			n += 4
		default:
			return n
		}
	}
	return n
}

func indentEnd(lines []string, start, limit int) int {
	base := indentOf(lines[start-1])
	end := start
	for i := start; i < limit; i++ {
		l := lines[i]
		if strings.TrimSpace(l) == "" {
			continue
		}
		if indentOf(l) <= base {
			break
		}
		end = i + 1
	}
	return end
}

func rubyEnd(lines []string, start, limit int) int {
	base := indentOf(lines[start-1])
	for i := start; i < limit; i++ {
		if strings.TrimSpace(lines[i]) == "end" && indentOf(lines[i]) == base {
			return i + 1
		}
	}
	return limit
}

// numbered merender baris [from, to] bernomor; mark = baris yang ditandai ">" (0 = tidak ada). Bila lebih dari
// max baris, bagian tengah diringkas agar awal & akhir fungsi tetap terlihat (dan baris mark ikut).
func numbered(lines []string, from, to, mark, max int) string {
	if from < 1 {
		from = 1
	}
	if to > len(lines) {
		to = len(lines)
	}
	if to < from {
		return ""
	}
	keep := func(n int) bool { return true }
	if max > 0 && to-from+1 > max {
		head := max * 2 / 3
		tail := max - head
		keep = func(n int) bool {
			return n < from+head || n > to-tail || (mark > 0 && n >= mark-3 && n <= mark+3)
		}
	}
	var b strings.Builder
	skipped := 0
	for n := from; n <= to; n++ {
		if !keep(n) {
			skipped++
			continue
		}
		if skipped > 0 {
			fmt.Fprintf(&b, "      … (%d baris)\n", skipped)
			skipped = 0
		}
		prefix := "  "
		if n == mark {
			prefix = "> "
		}
		fmt.Fprintf(&b, "%s%4d | %s\n", prefix, n, lines[n-1])
	}
	if skipped > 0 {
		fmt.Fprintf(&b, "      … (%d baris)\n", skipped)
	}
	return b.String()
}
