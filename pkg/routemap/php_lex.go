package routemap

import "strings"

// Lexer PHP ringan untuk penghitung biaya proses: cukup untuk mengenali pernyataan, blok, panggilan, dan string
// literal. Komentar, atribut #[...], dan HTML di luar tag <?php dibuang; `?>` dibaca sebagai akhir pernyataan
// dan `<?=` sebagai `echo`.

type phpTokKind int

const (
	ptIdent phpTokKind = iota // nama, kata kunci, nama ber-namespace (App\Models\User)
	ptVar                     // $nama
	ptStr                     // string literal; S = isi (tanpa kutip), Interp = ada interpolasi
	ptNum
	ptOp // tanda baca / operator
)

type phpTok struct {
	K      phpTokKind
	S      string
	Line   int
	Interp bool
}

func (t phpTok) is(s string) bool { return t.K == ptOp && t.S == s }

// kw: kata kunci tanpa membedakan huruf besar/kecil (PHP tidak peka huruf pada kata kunci).
func (t phpTok) kw(s string) bool { return t.K == ptIdent && strings.EqualFold(t.S, s) }

var phpOps = []string{
	"<<=", ">>=", "**=", "...", "<=>", "===", "!==", "??=", "?->",
	"->", "=>", "::", "==", "!=", "<>", "<=", ">=", "&&", "||", "??", "++", "--", "+=", "-=", "*=", "/=", ".=",
	"%=", "&=", "|=", "^=", "<<", ">>", "**",
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '\\' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isIdentChar(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

func lexPHP(src string) []phpTok {
	var out []phpTok
	line := 1
	i, n := 0, len(src)
	emit := func(k phpTokKind, s string) { out = append(out, phpTok{K: k, S: s, Line: line}) }
	// HTML di luar <?php … ?> dilewati.
	skipHTML := func() {
		for i < n {
			j := strings.Index(src[i:], "<?")
			if j < 0 {
				line += strings.Count(src[i:], "\n")
				i = n
				return
			}
			line += strings.Count(src[i:i+j], "\n")
			i += j
			switch {
			case strings.HasPrefix(src[i:], "<?php"):
				i += 5
				return
			case strings.HasPrefix(src[i:], "<?="):
				i += 3
				emit(ptIdent, "echo")
				return
			default:
				i += 2
				return
			}
		}
	}
	skipHTML()
	for i < n {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r' || c == '\f' || c == '\v':
			i++
		case c == '?' && i+1 < n && src[i+1] == '>':
			emit(ptOp, ";")
			i += 2
			if i < n && src[i] == '\n' {
				line++
				i++
			}
			skipHTML()
		case c == '#' && i+1 < n && src[i+1] == '[':
			// atribut PHP 8: #[Route('/x')] — dibuang dengan menyeimbangkan kurung siku.
			depth := 0
			for i < n {
				switch src[i] {
				case '[':
					depth++
				case ']':
					depth--
				case '\n':
					line++
				}
				i++
				if depth == 0 {
					break
				}
			}
		case c == '#' || (c == '/' && i+1 < n && src[i+1] == '/'):
			for i < n && src[i] != '\n' {
				if src[i] == '?' && i+1 < n && src[i+1] == '>' {
					break
				}
				i++
			}
		case c == '/' && i+1 < n && src[i+1] == '*':
			j := strings.Index(src[i+2:], "*/")
			if j < 0 {
				j = n - i - 2
			}
			line += strings.Count(src[i:i+2+j], "\n")
			i += j + 4
		case c == '$' && i+1 < n && isIdentStart(src[i+1]) && src[i+1] != '\\':
			j := i + 1
			for j < n && isIdentChar(src[j]) && src[j] != '\\' {
				j++
			}
			emit(ptVar, src[i:j])
			i = j
		case c == '\'' || c == '"' || c == '`':
			start := line
			j := i + 1
			interp := false
			var b strings.Builder
			for j < n && src[j] != c {
				if src[j] == '\\' && j+1 < n {
					if c == '\'' && src[j+1] != '\'' && src[j+1] != '\\' {
						b.WriteByte('\\')
					}
					j++
				} else if c != '\'' && (src[j] == '$' || (src[j] == '{' && j+1 < n && src[j+1] == '$')) {
					interp = true
				}
				if src[j] == '\n' {
					line++
				}
				b.WriteByte(src[j])
				j++
			}
			out = append(out, phpTok{K: ptStr, S: b.String(), Line: start, Interp: interp})
			i = j + 1
		case c == '<' && strings.HasPrefix(src[i:], "<<<"):
			i = lexHeredoc(src, i, &line, &out)
		case c >= '0' && c <= '9' || (c == '.' && i+1 < n && src[i+1] >= '0' && src[i+1] <= '9'):
			j := i
			for j < n && (isIdentChar(src[j]) || src[j] == '.') && src[j] != '\\' {
				if src[j] == '.' && (j+1 >= n || src[j+1] < '0' || src[j+1] > '9') {
					break
				}
				j++
			}
			emit(ptNum, src[i:j])
			i = j
		case isIdentStart(c):
			j := i
			for j < n && isIdentChar(src[j]) {
				j++
			}
			emit(ptIdent, strings.TrimSuffix(src[i:j], "\\"))
			i = j
		default:
			op := string(c)
			for _, o := range phpOps {
				if strings.HasPrefix(src[i:], o) {
					op = o
					break
				}
			}
			emit(ptOp, op)
			i += len(op)
		}
	}
	return out
}

// lexHeredoc: <<<ID … ID / <<<'ID' … ID → satu token string.
func lexHeredoc(src string, i int, line *int, out *[]phpTok) int {
	n := len(src)
	j := i + 3
	for j < n && (src[j] == ' ' || src[j] == '\t') {
		j++
	}
	quote := byte(0)
	if j < n && (src[j] == '\'' || src[j] == '"') {
		quote = src[j]
		j++
	}
	k := j
	for k < n && isIdentChar(src[k]) && src[k] != '\\' {
		k++
	}
	id := src[j:k]
	if id == "" {
		*out = append(*out, phpTok{K: ptOp, S: "<<", Line: *line})
		return i + 2
	}
	if quote != 0 && k < n && src[k] == quote {
		k++
	}
	start := *line
	// cari baris yang (setelah spasi) diawali ID
	pos := k
	for pos < n {
		nl := strings.IndexByte(src[pos:], '\n')
		if nl < 0 {
			pos = n
			break
		}
		pos += nl + 1
		*line++
		t := strings.TrimLeft(src[pos:], " \t")
		if strings.HasPrefix(t, id) && (len(t) == len(id) || !isIdentChar(t[len(id)])) {
			body := src[k:pos]
			end := pos + (len(src[pos:]) - len(t)) + len(id)
			*out = append(*out, phpTok{K: ptStr, S: body, Line: start, Interp: quote != '\'' && strings.Contains(body, "$")})
			return end
		}
	}
	*out = append(*out, phpTok{K: ptStr, S: src[k:], Line: start})
	return n
}
