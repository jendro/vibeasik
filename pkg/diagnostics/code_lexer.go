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
// codeStripper memproses satu baris karakter demi karakter; status komentar/string multi-baris
// dibawa antar baris.
type codeStripper struct {
	syn     lexSyntax
	lang    string
	inBlock bool   // di dalam /* */
	inMulti string // pembatas string multi-baris yang masih terbuka: "`", "'''", `"""`
	line    string
	b       []byte
	i       int
}

func (c *codeStripper) blank(from, to int) {
	for j := from; j < to; j++ {
		c.b[j] = ' '
	}
}

// stepBlock: di dalam komentar blok.
func (c *codeStripper) stepBlock() {
	if strings.HasPrefix(c.line[c.i:], "*/") {
		c.b[c.i], c.b[c.i+1] = ' ', ' '
		c.i += 2
		c.inBlock = false
		return
	}
	c.b[c.i] = ' '
	c.i++
}

// stepMulti: di dalam string multi-baris.
func (c *codeStripper) stepMulti() {
	if strings.HasPrefix(c.line[c.i:], c.inMulti) && (c.inMulti != "`" || c.i == 0 || c.line[c.i-1] != '\\') {
		c.i += len(c.inMulti)
		c.inMulti = ""
		return
	}
	c.b[c.i] = ' '
	c.i++
}

// startsLineComment: sisa baris diawali penanda komentar baris (PHP 8 attribute #[...] bukan komentar).
func (c *codeStripper) startsLineComment(rest string) bool {
	for _, lc := range c.syn.lineComments {
		if strings.HasPrefix(rest, lc) && !(lc == "#" && c.lang == "php" && strings.HasPrefix(rest, "#[")) {
			return true
		}
	}
	return false
}

// skipQuoted mengosongkan isi string satu baris; tanda kutip pembuka/penutup dipertahankan.
func (c *codeStripper) skipQuoted() {
	q := c.b[c.i]
	c.i++
	for c.i < len(c.b) && c.b[c.i] != q {
		if c.b[c.i] == '\\' && c.i+1 < len(c.b) {
			c.b[c.i], c.b[c.i+1] = ' ', ' '
			c.i += 2
			continue
		}
		c.b[c.i] = ' '
		c.i++
	}
	c.i++ // tanda kutip penutup (string satu baris yang tidak ditutup berakhir di ujung baris)
}

// stepCode: di luar komentar/string; mengembalikan true bila sisa baris adalah komentar baris.
func (c *codeStripper) stepCode() (lineDone bool) {
	rest := c.line[c.i:]
	switch {
	case c.syn.blockComment && strings.HasPrefix(rest, "/*"):
		c.b[c.i], c.b[c.i+1] = ' ', ' '
		c.i += 2
		c.inBlock = true
	case c.startsLineComment(rest):
		c.blank(c.i, len(c.b))
		return true
	case c.syn.tripleQuote && (strings.HasPrefix(rest, `"""`) || strings.HasPrefix(rest, "'''")):
		c.inMulti = rest[:3]
		c.i += 3
	case c.syn.backtick && c.b[c.i] == '`':
		c.inMulti = "`"
		c.i++
	case c.b[c.i] == '"' || c.b[c.i] == '\'':
		c.skipQuoted()
	default:
		c.i++
	}
	return false
}

// stripCode mengosongkan isi komentar dan literal string (diganti spasi, panjang baris tetap)
// agar pencocokan pola kode (percabangan, kurung kurawal, setState, ...) tidak tertipu teks
// di dalam string atau komentar. Tanda kutip pembuka/penutup dipertahankan.
func stripCode(lines []string, lang string) []string {
	c := &codeStripper{syn: lexSyntaxFor(lang), lang: lang}
	out := make([]string, len(lines))
	for li, line := range lines {
		c.line, c.b, c.i = line, []byte(line), 0
		for c.i < len(c.b) {
			switch {
			case c.inBlock:
				c.stepBlock()
			case c.inMulti != "":
				c.stepMulti()
			default:
				if c.stepCode() {
					c.i = len(c.b)
				}
			}
		}
		out[li] = string(c.b)
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
