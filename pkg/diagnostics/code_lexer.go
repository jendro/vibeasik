package diagnostics

import "strings"

// lexSyntax mendefinisikan sintaks komentar & string satu bahasa untuk stripCode
type lexSyntax struct {
	lineComments []string // mis. "//", "#"
	blockComment bool     // /* ... */
	backtick     bool     // `...` (JS/TS template, Go raw string), boleh multi-baris
	tripleQuote  bool     // '''...''' / """...""" (Python, Dart, Kotlin), boleh multi-baris
}

func lexSyntaxFor(lang string) lexSyntax {
	switch lang {
	case "python":
		return lexSyntax{lineComments: []string{"#"}, tripleQuote: true}
	case "php":
		return lexSyntax{lineComments: []string{"//", "#"}, blockComment: true}
	case "go", "typescript", "javascript", "vue", "svelte":
		return lexSyntax{lineComments: []string{"//"}, blockComment: true, backtick: true}
	case "dart", "kotlin":
		return lexSyntax{lineComments: []string{"//"}, blockComment: true, tripleQuote: true}
	default:
		return lexSyntax{lineComments: []string{"//"}, blockComment: true}
	}
}

// stripCode mengosongkan isi komentar dan literal string (diganti spasi, panjang baris tetap)
// agar pencocokan pola kode (percabangan, kurung kurawal, setState, ...) tidak tertipu teks
// di dalam string atau komentar. Tanda kutip pembuka/penutup dipertahankan.
func stripCode(lines []string, lang string) []string {
	syn := lexSyntaxFor(lang)
	out := make([]string, len(lines))
	inBlock := false // di dalam /* */
	inMulti := ""    // pembatas string multi-baris yang masih terbuka: "`", "'''", `"""`

	for li, line := range lines {
		b := []byte(line)
		i := 0
		for i < len(b) {
			if inBlock {
				if strings.HasPrefix(line[i:], "*/") {
					b[i], b[i+1] = ' ', ' '
					i += 2
					inBlock = false
					continue
				}
				b[i] = ' '
				i++
				continue
			}
			if inMulti != "" {
				if strings.HasPrefix(line[i:], inMulti) && (inMulti != "`" || i == 0 || line[i-1] != '\\') {
					i += len(inMulti)
					inMulti = ""
					continue
				}
				b[i] = ' '
				i++
				continue
			}

			rest := line[i:]
			if syn.blockComment && strings.HasPrefix(rest, "/*") {
				b[i], b[i+1] = ' ', ' '
				i += 2
				inBlock = true
				continue
			}
			isLineComment := false
			for _, lc := range syn.lineComments {
				// PHP 8 attribute #[...] bukan komentar
				if strings.HasPrefix(rest, lc) && !(lc == "#" && lang == "php" && strings.HasPrefix(rest, "#[")) {
					isLineComment = true
					break
				}
			}
			if isLineComment {
				for j := i; j < len(b); j++ {
					b[j] = ' '
				}
				break
			}
			if syn.tripleQuote && (strings.HasPrefix(rest, `"""`) || strings.HasPrefix(rest, "'''")) {
				inMulti = rest[:3]
				i += 3
				continue
			}
			if syn.backtick && b[i] == '`' {
				inMulti = "`"
				i++
				continue
			}
			if b[i] == '"' || b[i] == '\'' {
				q := b[i]
				i++
				for i < len(b) && b[i] != q {
					if b[i] == '\\' && i+1 < len(b) {
						b[i], b[i+1] = ' ', ' '
						i += 2
						continue
					}
					b[i] = ' '
					i++
				}
				i++ // tanda kutip penutup (string satu baris yang tidak ditutup berakhir di ujung baris)
				continue
			}
			i++
		}
		out[li] = string(b)
	}
	return out
}

// bracketDelta menghitung perubahan kedalaman kurung kurawal dan kurung biasa/siku pada satu baris
// yang sudah dibersihkan stripCode. maxBrace = kedalaman kurawal tertinggi relatif di baris tsb.
func bracketDelta(stripped string) (brace int, paren int, maxBrace int) {
	for i := 0; i < len(stripped); i++ {
		switch stripped[i] {
		case '{':
			brace++
			if brace > maxBrace {
				maxBrace = brace
			}
		case '}':
			brace--
		case '(', '[':
			paren++
		case ')', ']':
			paren--
		}
	}
	return brace, paren, maxBrace
}
