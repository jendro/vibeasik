package routemap

import "strings"

// Parser pernyataan PHP: menyusun pohon pernyataan (if/loop/switch/try/blok) dan mengumpulkan deklarasi
// fungsi & kelas. Ekspresi tidak diurai menjadi pohon: cukup potongan token yang nanti dipindai untuk
// mencari panggilan, closure, dan match — sama seperti penghitung Go yang hanya melihat CallExpr/FuncLit.

type phpStmtKind int

const (
	psExpr   phpStmtKind = iota // pernyataan biasa: 1 tick + panggilan di dalamnya
	psBlock                     // { … }
	psIf                        // Cond, Body, Else
	psLoop                      // Loop (for/foreach/while/do), Cond = kepala loop, Body
	psSwitch                    // Cond = subjek, Cases
	psTry                       // Body, Cases = catch, Else = finally
	psNop                       // deklarasi (namespace, use, function, class) — tanpa biaya
)

type phpStmt struct {
	Kind  phpStmtKind
	Line  int
	Expr  []phpTok // psExpr
	Cond  []phpTok
	Loop  string
	Body  []*phpStmt
	Else  []*phpStmt
	Cases [][]*phpStmt
	// CaseExprs: ekspresi `case X:` sejajar dengan Cases (switch)
	CaseExprs [][]phpTok
}

type phpParam struct {
	Type string // nama kelas pendek (tanpa namespace, tanpa ?), kosong = tanpa tipe
	Name string // $nama
}

type phpFunc struct {
	Key    string // "Kelas::method" atau "fungsi"
	Name   string
	Class  *phpClass
	Params []phpParam
	Body   []*phpStmt
	Line   int
	End    int
	Static bool
	file   *phpFile
}

type phpClass struct {
	Name       string // pendek
	FQN        string
	Kind       string // class, trait, interface, enum
	Extends    string // pendek
	Implements []string
	Traits     []string
	Methods    map[string]*phpFunc // nama method huruf kecil → fungsi
	PropTypes  map[string]string   // nama properti (tanpa $) → kelas pendek
	Line       int
	file       *phpFile
}

type phpFile struct {
	Rel       string
	Namespace string
	Stmts     []*phpStmt
	Funcs     []*phpFunc
	Classes   []*phpClass
	// Uses: alias pendek → nama lengkap (use App\Models\User as U)
	Uses map[string]string
}

type phpParser struct {
	t    []phpTok
	p    int
	file *phpFile
	cls  *phpClass
}

func parsePHPFile(rel, src string) *phpFile {
	f := &phpFile{Rel: rel, Uses: map[string]string{}}
	ps := &phpParser{t: lexPHP(src), file: f}
	for !ps.eof() { // `}` liar di tingkat berkas (templat campur HTML) tidak boleh menghentikan penguraian
		f.Stmts = append(f.Stmts, ps.stmts(func() bool { return false })...)
		ps.expect("}")
	}
	return f
}

// parsePHPTokens mengurai potongan token (badan closure) sebagai daftar pernyataan.
func parsePHPTokens(f *phpFile, cls *phpClass, toks []phpTok) []*phpStmt {
	ps := &phpParser{t: toks, file: f, cls: cls}
	return ps.stmts(func() bool { return false })
}

func (ps *phpParser) eof() bool { return ps.p >= len(ps.t) }

func (ps *phpParser) peek(k int) phpTok {
	if ps.p+k < len(ps.t) {
		return ps.t[ps.p+k]
	}
	return phpTok{K: ptOp, S: ""}
}

func (ps *phpParser) line() int {
	if ps.eof() {
		if len(ps.t) > 0 {
			return ps.t[len(ps.t)-1].Line
		}
		return 0
	}
	return ps.t[ps.p].Line
}

// stmts membaca pernyataan sampai EOF, `}` penutup, atau stop() (kata kunci sintaks alternatif: endif …).
func (ps *phpParser) stmts(stop func() bool) []*phpStmt {
	var out []*phpStmt
	for !ps.eof() && !ps.peek(0).is("}") && !stop() {
		before := ps.p
		if s := ps.stmt(); s != nil && s.Kind != psNop {
			out = append(out, s)
		}
		if ps.p == before { // pengaman: token yang tidak dikenali dilewati
			ps.p++
		}
	}
	return out
}

func (ps *phpParser) expect(op string) {
	if ps.peek(0).is(op) {
		ps.p++
	}
}

// balanced mengambil isi kurung yang dimulai di posisi sekarang (`(`, `[`, `{`) tanpa kurung luarnya.
func (ps *phpParser) balanced() []phpTok {
	if ps.eof() {
		return nil
	}
	open := ps.t[ps.p].S
	close := map[string]string{"(": ")", "[": "]", "{": "}"}[open]
	if close == "" {
		return nil
	}
	start := ps.p + 1
	depth := 0
	for ps.p < len(ps.t) {
		t := ps.t[ps.p]
		if t.K == ptOp {
			switch t.S {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
		}
		ps.p++
		if depth == 0 {
			return ps.t[start : ps.p-1]
		}
	}
	return ps.t[start:]
}

// body: badan if/loop — blok, satu pernyataan, atau sintaks alternatif `: … endX;`.
func (ps *phpParser) body(ends ...string) []*phpStmt {
	if ps.peek(0).is(":") {
		ps.p++
		list := ps.stmts(func() bool {
			for _, e := range ends {
				if ps.peek(0).kw(e) {
					return true
				}
			}
			return false
		})
		return list
	}
	if ps.peek(0).is("{") {
		ps.p++
		list := ps.stmts(func() bool { return false })
		ps.expect("}")
		return list
	}
	if s := ps.stmt(); s != nil && s.Kind != psNop {
		return []*phpStmt{s}
	}
	return nil
}

func (ps *phpParser) endKeyword(kw string) {
	if ps.peek(0).kw(kw) {
		ps.p++
		ps.expect(";")
	}
}

var phpModifiers = map[string]bool{"abstract": true, "final": true, "readonly": true}

func (ps *phpParser) stmt() *phpStmt {
	t := ps.peek(0)
	line := t.Line
	switch {
	case t.is(";"):
		ps.p++
		return nil
	case t.is("{"):
		ps.p++
		list := ps.stmts(func() bool { return false })
		ps.expect("}")
		return &phpStmt{Kind: psBlock, Line: line, Body: list}
	case t.K != ptIdent:
		return ps.exprStmt()
	}
	switch strings.ToLower(t.S) {
	case "namespace":
		ps.p++
		var name []string
		for !ps.eof() && ps.peek(0).K == ptIdent {
			name = append(name, ps.peek(0).S)
			ps.p++
		}
		ps.file.Namespace = strings.TrimPrefix(strings.Join(name, ""), "\\")
		if ps.peek(0).is("{") {
			ps.p++
			list := ps.stmts(func() bool { return false })
			ps.expect("}")
			return &phpStmt{Kind: psBlock, Line: line, Body: list}
		}
		ps.expect(";")
		return &phpStmt{Kind: psNop}
	case "use":
		ps.p++
		ps.useDecl()
		return &phpStmt{Kind: psNop}
	case "declare":
		ps.p++
		ps.balanced()
		ps.expect(";")
		return &phpStmt{Kind: psNop}
	case "if":
		return ps.ifStmt()
	case "foreach", "for", "while":
		kw := strings.ToLower(t.S)
		ps.p++
		cond := ps.balanced()
		body := ps.body("end" + kw)
		ps.endKeyword("end" + kw)
		return &phpStmt{Kind: psLoop, Line: line, Loop: kw, Cond: cond, Body: body}
	case "do":
		ps.p++
		body := ps.body()
		var cond []phpTok
		if ps.peek(0).kw("while") {
			ps.p++
			cond = ps.balanced()
		}
		ps.expect(";")
		return &phpStmt{Kind: psLoop, Line: line, Loop: "do", Cond: cond, Body: body}
	case "switch":
		ps.p++
		subj := ps.balanced()
		return ps.switchBody(line, subj)
	case "try":
		ps.p++
		s := &phpStmt{Kind: psTry, Line: line, Body: ps.body()}
		for ps.peek(0).kw("catch") {
			ps.p++
			ps.balanced()
			s.Cases = append(s.Cases, ps.body())
		}
		if ps.peek(0).kw("finally") {
			ps.p++
			s.Else = ps.body()
		}
		return s
	case "function":
		// function nama(…) — deklarasi; function (…) / function &(…) — closure dalam ekspresi.
		nx := ps.peek(1)
		if nx.is("&") {
			nx = ps.peek(2)
		}
		if nx.K == ptIdent {
			ps.p++
			ps.expect("&")
			ps.funcDecl(nil, false)
			return &phpStmt{Kind: psNop}
		}
		return ps.exprStmt()
	case "abstract", "final", "readonly", "class", "interface", "trait", "enum":
		j := 0
		for phpModifiers[strings.ToLower(ps.peek(j).S)] && ps.peek(j).K == ptIdent {
			j++
		}
		kw := ps.peek(j)
		if kw.K == ptIdent && ps.peek(j+1).K == ptIdent {
			switch strings.ToLower(kw.S) {
			case "class", "interface", "trait", "enum":
				ps.p += j
				ps.classDecl()
				return &phpStmt{Kind: psNop}
			}
		}
		return ps.exprStmt()
	}
	return ps.exprStmt()
}

func (ps *phpParser) useDecl() {
	// use A\B\C; use A\B\{C, D as E}; use function A\b;
	var prefix []string
	var cur []string
	alias := ""
	flush := func() {
		if len(cur) == 0 {
			return
		}
		full := strings.TrimPrefix(strings.Join(append(append([]string{}, prefix...), cur...), ""), "\\")
		short := alias
		if short == "" {
			short = phpShort(full)
		}
		ps.file.Uses[strings.ToLower(short)] = full
		cur, alias = nil, ""
	}
	for !ps.eof() {
		t := ps.peek(0)
		switch {
		case t.is(";"):
			flush()
			ps.p++
			return
		case t.is(","):
			flush()
		case t.is("{"):
			prefix = cur
			cur = nil
		case t.is("}"):
			flush()
		case t.kw("as"):
			if nx := ps.peek(1); nx.K == ptIdent {
				alias = nx.S
				ps.p++
			}
		case t.K == ptIdent && (t.kw("function") || t.kw("const")) && len(cur) == 0:
		case t.K == ptIdent:
			cur = append(cur, t.S)
		}
		ps.p++
	}
	flush()
}

func (ps *phpParser) ifStmt() *phpStmt {
	line := ps.line()
	ps.p++ // if / elseif
	cond := ps.balanced()
	s := &phpStmt{Kind: psIf, Line: line, Cond: cond}
	alt := ps.peek(0).is(":")
	s.Body = ps.body("elseif", "else", "endif")
	switch {
	case ps.peek(0).kw("elseif"):
		s.Else = []*phpStmt{ps.ifStmt()}
		return s
	case ps.peek(0).kw("else") && ps.peek(1).kw("if") && !alt:
		ps.p++
		s.Else = []*phpStmt{ps.ifStmt()}
		return s
	case ps.peek(0).kw("else"):
		ps.p++
		s.Else = ps.body("endif")
	}
	if alt {
		ps.endKeyword("endif")
	}
	return s
}

func (ps *phpParser) switchBody(line int, subj []phpTok) *phpStmt {
	s := &phpStmt{Kind: psSwitch, Line: line, Cond: subj}
	alt := false
	switch {
	case ps.peek(0).is("{"):
		ps.p++
	case ps.peek(0).is(":"):
		ps.p++
		alt = true
	default:
		return s
	}
	isCase := func() bool { return ps.peek(0).kw("case") || ps.peek(0).kw("default") || ps.peek(0).kw("endswitch") }
	for !ps.eof() && !ps.peek(0).is("}") && !ps.peek(0).kw("endswitch") {
		if !ps.peek(0).kw("case") && !ps.peek(0).kw("default") {
			ps.p++
			continue
		}
		ps.p++
		var ce []phpTok
		depth := 0
		for !ps.eof() {
			t := ps.peek(0)
			if t.K == ptOp {
				switch t.S {
				case "(", "[", "{":
					depth++
				case ")", "]", "}":
					depth--
				}
				if depth == 0 && (t.S == ":" || t.S == ";") {
					break
				}
			}
			ce = append(ce, t)
			ps.p++
		}
		ps.p++ // ':'
		s.CaseExprs = append(s.CaseExprs, ce)
		s.Cases = append(s.Cases, ps.stmts(isCase))
	}
	if alt {
		ps.endKeyword("endswitch")
	} else {
		ps.expect("}")
	}
	return s
}

// exprStmt: token sampai `;` pada kedalaman 0 (closure & match di dalamnya ikut).
func (ps *phpParser) exprStmt() *phpStmt {
	line := ps.line()
	start := ps.p
	depth := 0
	for !ps.eof() {
		t := ps.t[ps.p]
		if t.K == ptOp {
			switch t.S {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				if depth == 0 {
					return &phpStmt{Kind: psExpr, Line: line, Expr: ps.t[start:ps.p]}
				}
				depth--
			case ";":
				if depth == 0 {
					s := &phpStmt{Kind: psExpr, Line: line, Expr: ps.t[start:ps.p]}
					ps.p++
					return s
				}
			}
		}
		ps.p++
	}
	return &phpStmt{Kind: psExpr, Line: line, Expr: ps.t[start:]}
}

// params mengurai daftar parameter `(Type $a, ?Other $b = null, private Svc $svc)`.
func (ps *phpParser) params() (out []phpParam, promoted map[string]string) {
	toks := ps.balanced()
	promoted = map[string]string{}
	for _, part := range splitTop(toks, ",") {
		var typ, name string
		promote := false
		for _, t := range part {
			if t.is("=") {
				break
			}
			switch {
			case t.K == ptVar:
				name = t.S
			case t.K == ptIdent && name == "":
				switch strings.ToLower(t.S) {
				case "public", "private", "protected":
					promote = true
				case "readonly", "null", "array", "string", "int", "float", "bool", "mixed", "callable", "iterable", "object", "self", "static":
				default:
					typ = phpShort(t.S)
				}
			}
		}
		if name == "" {
			continue
		}
		out = append(out, phpParam{Type: typ, Name: name})
		if promote && typ != "" {
			promoted[strings.TrimPrefix(name, "$")] = typ
		}
	}
	return out, promoted
}

// funcDecl: posisi di nama fungsi.
func (ps *phpParser) funcDecl(cls *phpClass, static bool) *phpFunc {
	nameTok := ps.peek(0)
	ps.p++
	fn := &phpFunc{Name: nameTok.S, Class: cls, Line: nameTok.Line, Static: static, file: ps.file}
	var promoted map[string]string
	if ps.peek(0).is("(") {
		fn.Params, promoted = ps.params()
	}
	if cls != nil {
		for k, v := range promoted {
			cls.PropTypes[k] = v
		}
	}
	// tipe kembalian
	for !ps.eof() && !ps.peek(0).is("{") && !ps.peek(0).is(";") {
		ps.p++
	}
	if ps.peek(0).is(";") { // abstract / interface
		fn.End = ps.peek(0).Line
		ps.p++
		return nil
	}
	saved := ps.cls
	ps.cls = cls
	ps.p++
	fn.Body = ps.stmts(func() bool { return false })
	fn.End = ps.line()
	ps.expect("}")
	ps.cls = saved
	if cls != nil && strings.EqualFold(fn.Name, "__construct") {
		// $this->repo = $repo; dengan parameter bertipe → tipe properti (injeksi lewat konstruktor)
		types := map[string]string{}
		for _, p := range fn.Params {
			if p.Type != "" {
				types[p.Name] = p.Type
			}
		}
		for _, st := range fn.Body {
			e := st.Expr
			if st.Kind == psExpr && len(e) == 5 && e[0].S == "$this" && e[1].is("->") && e[2].K == ptIdent && e[3].is("=") && e[4].K == ptVar {
				if typ := types[e[4].S]; typ != "" {
					cls.PropTypes[e[2].S] = typ
				}
			}
		}
	}
	if cls != nil {
		fn.Key = cls.Name + "::" + fn.Name
		cls.Methods[strings.ToLower(fn.Name)] = fn
	} else {
		fn.Key = fn.Name
		ps.file.Funcs = append(ps.file.Funcs, fn)
	}
	return fn
}

func (ps *phpParser) classDecl() {
	kind := strings.ToLower(ps.peek(0).S)
	ps.p++
	nameTok := ps.peek(0)
	ps.p++
	c := &phpClass{Name: nameTok.S, Kind: kind, Methods: map[string]*phpFunc{}, PropTypes: map[string]string{}, Line: nameTok.Line, file: ps.file}
	c.FQN = nameTok.S
	if ps.file.Namespace != "" {
		c.FQN = ps.file.Namespace + "\\" + nameTok.S
	}
	for !ps.eof() && !ps.peek(0).is("{") {
		if ps.peek(0).kw("extends") && ps.peek(1).K == ptIdent {
			c.Extends = phpShort(ps.peek(1).S)
			ps.p++
		}
		if ps.peek(0).kw("implements") {
			for ps.p++; !ps.eof() && !ps.peek(0).is("{"); ps.p++ {
				if ps.peek(0).K == ptIdent {
					c.Implements = append(c.Implements, phpShort(ps.peek(0).S))
				}
			}
			break
		}
		ps.p++
	}
	ps.p++ // {
	ps.file.Classes = append(ps.file.Classes, c)
	for !ps.eof() && !ps.peek(0).is("}") {
		t := ps.peek(0)
		switch {
		case t.kw("use"):
			ps.p++
			for !ps.eof() && !ps.peek(0).is(";") && !ps.peek(0).is("{") {
				if ps.peek(0).K == ptIdent {
					c.Traits = append(c.Traits, phpShort(ps.peek(0).S))
				}
				ps.p++
			}
			if ps.peek(0).is("{") {
				ps.balanced()
			} else {
				ps.p++
			}
		case t.kw("function"):
			ps.p++
			ps.expect("&")
			static := false
			for k := ps.p - 2; k >= 0 && k >= ps.p-5; k-- {
				if ps.t[k].kw("static") {
					static = true
				}
				if ps.t[k].is(";") || ps.t[k].is("}") || ps.t[k].is("{") {
					break
				}
			}
			ps.funcDecl(c, static)
		case t.K == ptVar:
			// properti bertipe: private UserService $svc;
			if p := ps.p - 1; p >= 0 && ps.t[p].K == ptIdent {
				if typ := phpShort(ps.t[p].S); !isPHPScalar(typ) {
					c.PropTypes[strings.TrimPrefix(t.S, "$")] = typ
				}
			}
			ps.skipTo(";")
		case t.kw("const") || t.kw("case"):
			ps.skipTo(";")
		default:
			ps.p++
		}
	}
	ps.expect("}")
}

// skipTo melompati token sampai op (pada kedalaman 0) dan melewatinya.
func (ps *phpParser) skipTo(op string) {
	depth := 0
	for !ps.eof() {
		t := ps.t[ps.p]
		if t.K == ptOp {
			switch t.S {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				if depth == 0 {
					return
				}
				depth--
			}
			if depth == 0 && t.S == op {
				ps.p++
				return
			}
		}
		ps.p++
	}
}

func isPHPScalar(s string) bool {
	switch strings.ToLower(s) {
	case "", "public", "private", "protected", "static", "readonly", "var", "array", "string", "int", "float", "bool",
		"mixed", "callable", "iterable", "object", "null", "self", "true", "false":
		return true
	}
	return false
}

// phpShort: App\Models\User → User.
func phpShort(s string) string {
	s = strings.TrimPrefix(s, "?")
	if i := strings.LastIndex(s, "\\"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// splitTop memecah token pada pemisah di kedalaman 0.
func splitTop(toks []phpTok, sep string) [][]phpTok {
	var out [][]phpTok
	depth, start := 0, 0
	for i, t := range toks {
		if t.K != ptOp {
			continue
		}
		switch t.S {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		default:
			if depth == 0 && t.S == sep {
				out = append(out, toks[start:i])
				start = i + 1
			}
		}
	}
	if start < len(toks) {
		out = append(out, toks[start:])
	}
	return out
}
