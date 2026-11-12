package routemap

import (
	"path"
	"regexp"
	"strconv"
	"strings"
)

// Biaya proses PHP (Laravel, CodeIgniter, Slim, PHP biasa) memakai model tick yang sama dengan Go (cost.go):
//
//	1 tick            = satu pernyataan (assign, panggilan, return, echo, cek syarat …)
//	if / switch / match / try = 1 tick cek syarat + cabang TERMAHAL
//	foreach / while   = loop atas data: (n + 1) cek syarat + n × isi; foreach [a, b] / for ($i < 3) = batas tetap
//	fungsi/method lokal = rumusnya ikut dijumlahkan (dalam loop dikalikan n)
//	pustaka luar      = 1 tick; sort/usort/sortBy = n·log n; array_map/->map/->each(callback) = n × isi callback
//	query database    = Q(n): Eloquent (User::find, ->get(), $model->save()), DB::, PDO/mysqli; di dalam loop → n + 1

type phpModule struct {
	Name, Dir, Framework string
	root                 string // root proyek (abs); phpFile.Rel relatif terhadapnya
	files                []*phpFile
	byRel                map[string]*phpFile
	funcs                map[string]*phpFunc    // fungsi global, huruf kecil
	classes              map[string][]*phpClass // nama pendek huruf kecil
	methodsByName        map[string][]*phpFunc
	implementers         map[string][]*phpClass // interface (pendek, huruf kecil) → kelas lokal yang mengimplementasikannya
	all                  []*phpFunc
	aliases              map[string]string // alias middleware Laravel → kelas pendek
}

func (m *phpModule) index() {
	m.byRel = map[string]*phpFile{}
	m.funcs = map[string]*phpFunc{}
	m.classes = map[string][]*phpClass{}
	m.methodsByName = map[string][]*phpFunc{}
	m.implementers = map[string][]*phpClass{}
	for _, f := range m.files {
		m.byRel[f.Rel] = f
		for _, fn := range f.Funcs {
			if _, dup := m.funcs[strings.ToLower(fn.Name)]; !dup {
				m.funcs[strings.ToLower(fn.Name)] = fn
			}
			m.all = append(m.all, fn)
		}
		for _, c := range f.Classes {
			m.classes[strings.ToLower(c.Name)] = append(m.classes[strings.ToLower(c.Name)], c)
			for _, i := range c.Implements {
				m.implementers[strings.ToLower(i)] = append(m.implementers[strings.ToLower(i)], c)
			}
			for ln, fn := range c.Methods {
				m.methodsByName[ln] = append(m.methodsByName[ln], fn)
				m.all = append(m.all, fn)
			}
		}
	}
}

// class mencari kelas lokal dari nama di kode (memperhitungkan `use … as …` berkas pemanggil).
func (m *phpModule) class(f *phpFile, name string) *phpClass {
	if name == "" {
		return nil
	}
	full := strings.TrimPrefix(name, "\\")
	if f != nil && !strings.Contains(full, "\\") {
		if u, ok := f.Uses[strings.ToLower(full)]; ok {
			full = u
		} else if f.Namespace != "" {
			full = f.Namespace + "\\" + full
		}
	}
	cands := m.classes[strings.ToLower(phpShort(full))]
	for _, c := range cands {
		if strings.EqualFold(c.FQN, full) {
			return c
		}
	}
	if len(cands) > 0 {
		return cands[0]
	}
	return nil
}

// fqn: nama lengkap kelas dari sudut pandang berkas f (untuk mengenali App\Models\… milik pustaka/vendor).
func phpFQN(f *phpFile, name string) string {
	name = strings.TrimPrefix(name, "\\")
	if f != nil && !strings.Contains(name, "\\") {
		if u, ok := f.Uses[strings.ToLower(name)]; ok {
			return u
		}
		if f.Namespace != "" {
			return f.Namespace + "\\" + name
		}
	}
	return name
}

// method mencari method di kelas, trait-nya, lalu induknya.
func (m *phpModule) method(c *phpClass, name string) *phpFunc {
	seen := map[*phpClass]bool{}
	for c != nil && !seen[c] {
		seen[c] = true
		if fn := c.Methods[strings.ToLower(name)]; fn != nil {
			return fn
		}
		for _, t := range c.Traits {
			if tc := m.class(c.file, t); tc != nil {
				if fn := tc.Methods[strings.ToLower(name)]; fn != nil {
					return fn
				}
			}
		}
		c = m.class(c.file, c.Extends)
	}
	return nil
}

// methods: method pada kelas; bila kelasnya interface/abstrak tanpa badan, method di kelas lokal yang
// mengimplementasikannya (injeksi lewat interface repository/service). Pemanggil memakai yang termahal.
func (m *phpModule) methods(c *phpClass, name string) []*phpFunc {
	if fn := m.method(c, name); fn != nil {
		return []*phpFunc{fn}
	}
	var out []*phpFunc
	impls := m.implementers[strings.ToLower(c.Name)]
	if len(impls) > 4 {
		return nil
	}
	for _, k := range impls {
		if fn := m.method(k, name); fn != nil {
			out = append(out, fn)
		}
	}
	return out
}

func (m *phpModule) propType(c *phpClass, prop string) string {
	seen := map[*phpClass]bool{}
	for c != nil && !seen[c] {
		seen[c] = true
		if t, ok := c.PropTypes[prop]; ok {
			return t
		}
		c = m.class(c.file, c.Extends)
	}
	return ""
}

var phpModelBases = map[string]bool{"model": true, "authenticatable": true, "pivot": true, "morphpivot": true}

// isModel: kelas Eloquent (turunan Model) atau kelas di namespace …\Models\.
func (m *phpModule) isModel(f *phpFile, name string) bool {
	if name == "" {
		return false
	}
	if phpModelBases[strings.ToLower(phpShort(name))] && m.class(f, name) == nil { // Illuminate\…\Model / CodeIgniter\Model
		return true
	}
	if strings.Contains(strings.ToLower(phpFQN(f, name)), "\\models\\") {
		return true
	}
	c := m.class(f, name)
	for i := 0; c != nil && i < 8; i++ {
		if phpModelBases[strings.ToLower(c.Extends)] || strings.Contains(strings.ToLower(c.FQN), "\\models\\") {
			return true
		}
		c = m.class(c.file, c.Extends)
	}
	return false
}

func phpNodeKey(fn *phpFunc) string { return fn.file.Rel + "#" + fn.Key }

// ---------- penghitung ----------

type phpCostCtx struct {
	m        *phpModule
	memo     map[*phpFunc]cst
	busy     map[*phpFunc]bool
	fileMemo map[*phpFile]cst
	fileBusy map[*phpFile]bool
	edges    map[string]map[string]bool
	cur      []string
}

func newPHPCostCtx(m *phpModule) *phpCostCtx {
	return &phpCostCtx{m: m, memo: map[*phpFunc]cst{}, busy: map[*phpFunc]bool{}, fileMemo: map[*phpFile]cst{},
		fileBusy: map[*phpFile]bool{}, edges: map[string]map[string]bool{}}
}

type phpScope struct {
	file     *phpFile
	cls      *phpClass
	vars     map[string]string // $var → kelas pendek
	closures map[string]cst
}

func (c *phpCostCtx) edge(to string) {
	if len(c.cur) == 0 {
		return
	}
	from := c.cur[len(c.cur)-1]
	if c.edges[from] == nil {
		c.edges[from] = map[string]bool{}
	}
	c.edges[from][to] = true
}

func (c *phpCostCtx) funcScope(fn *phpFunc) *phpScope {
	sc := &phpScope{file: fn.file, cls: fn.Class, vars: map[string]string{}, closures: map[string]cst{}}
	for _, p := range fn.Params {
		if p.Type != "" {
			sc.vars[p.Name] = p.Type
		}
	}
	return sc
}

func (c *phpCostCtx) funcCost(fn *phpFunc) cst {
	if v, ok := c.memo[fn]; ok {
		return v
	}
	if c.busy[fn] {
		out := newCst()
		out.tick(1)
		out.recursive = true
		return out
	}
	c.busy[fn] = true
	c.cur = append(c.cur, phpNodeKey(fn))
	out := c.block(c.funcScope(fn), fn.Body)
	c.cur = c.cur[:len(c.cur)-1]
	delete(c.busy, fn)
	c.memo[fn] = out
	return out
}

// fileCost: pernyataan tingkat atas sebuah berkas (halaman PHP biasa / berkas yang di-include).
func (c *phpCostCtx) fileCost(f *phpFile) cst {
	if v, ok := c.fileMemo[f]; ok {
		return v
	}
	if c.fileBusy[f] {
		out := newCst()
		out.tick(1)
		out.recursive = true
		return out
	}
	c.fileBusy[f] = true
	c.cur = append(c.cur, "file:"+f.Rel)
	out := c.block(&phpScope{file: f, vars: map[string]string{}, closures: map[string]cst{}}, f.Stmts)
	c.cur = c.cur[:len(c.cur)-1]
	delete(c.fileBusy, f)
	c.fileMemo[f] = out
	return out
}

func (c *phpCostCtx) block(sc *phpScope, list []*phpStmt) cst {
	out := newCst()
	for _, s := range list {
		out.add(c.stmt(sc, s))
	}
	return out
}

func (c *phpCostCtx) stmt(sc *phpScope, s *phpStmt) cst {
	out := newCst()
	switch s.Kind {
	case psBlock:
		return c.block(sc, s.Body)
	case psIf:
		out.tick(1)
		out.add(c.expr(sc, s.Cond))
		out.add(heavier(c.block(sc, s.Body), c.block(sc, s.Else)))
	case psSwitch:
		out.tick(1)
		out.add(c.expr(sc, s.Cond))
		best := newCst()
		for i, cs := range s.Cases {
			cur := newCst()
			if i < len(s.CaseExprs) {
				cur.add(c.expr(sc, s.CaseExprs[i]))
			}
			cur.add(c.block(sc, cs))
			best = heavier(best, cur)
		}
		out.add(best)
	case psTry:
		out.add(c.block(sc, s.Body))
		best := newCst()
		for _, cs := range s.Cases {
			best = heavier(best, c.block(sc, cs))
		}
		out.add(best)
		out.add(c.block(sc, s.Else))
	case psLoop:
		return c.loop(sc, s)
	case psExpr:
		if len(s.Expr) == 0 {
			return out
		}
		out.tick(1)
		out.add(c.exprStmt(sc, s.Expr))
	}
	return out
}

func (c *phpCostCtx) loop(sc *phpScope, s *phpStmt) cst {
	out := newCst()
	iter := newCst()
	iter.tick(1) // cek syarat
	var k int64
	data, unbounded := true, false
	switch s.Loop {
	case "foreach":
		subj := s.Cond
		for i, t := range s.Cond {
			if t.kw("as") {
				subj = s.Cond[:i]
				break
			}
		}
		out.add(c.expr(sc, subj))
		if v, ok := phpFixedCount(subj); ok {
			k, data = v, false
		}
		iter.add(c.block(sc, s.Body))
	case "for":
		parts := splitTop(s.Cond, ";")
		for len(parts) < 3 {
			parts = append(parts, nil)
		}
		if len(parts[0]) > 0 {
			out.tick(1)
			out.add(c.expr(sc, parts[0]))
		}
		iter.add(c.expr(sc, parts[1]))
		iter.add(c.block(sc, s.Body))
		if len(parts[2]) > 0 {
			iter.tick(1)
			iter.add(c.expr(sc, parts[2]))
		}
		k, data, unbounded = phpForBound(parts[0], parts[1])
	default: // while, do
		iter.add(c.expr(sc, s.Cond))
		iter.add(c.block(sc, s.Body))
		k, data, unbounded = phpForBound(nil, s.Cond)
	}
	out.add(loopCost(iter, k, data))
	out.unbounded = out.unbounded || unbounded
	return out
}

// phpFixedCount: foreach atas literal [a, b, c] / array(a, b) / range(1, 10) → jumlah tetap.
func phpFixedCount(toks []phpTok) (int64, bool) {
	if len(toks) >= 2 && toks[0].is("[") && toks[len(toks)-1].is("]") && matchingClose(toks, 0) == len(toks)-1 {
		if n := len(splitTop(toks[1:len(toks)-1], ",")); n > 0 {
			return int64(n), true
		}
	}
	if len(toks) >= 3 && toks[0].kw("array") && toks[1].is("(") && matchingClose(toks, 1) == len(toks)-1 {
		if n := len(splitTop(toks[2:len(toks)-1], ",")); n > 0 {
			return int64(n), true
		}
	}
	if len(toks) >= 3 && toks[0].kw("range") && toks[1].is("(") && matchingClose(toks, 1) == len(toks)-1 {
		args := splitTop(toks[2:len(toks)-1], ",")
		if len(args) == 2 && len(args[0]) == 1 && len(args[1]) == 1 {
			a, ea := strconv.ParseInt(args[0][0].S, 10, 64)
			b, eb := strconv.ParseInt(args[1][0].S, 10, 64)
			if ea == nil && eb == nil && args[0][0].K == ptNum && args[1][0].K == ptNum {
				if b < a {
					a, b = b, a
				}
				return b - a + 1, true
			}
		}
	}
	return 0, false
}

func matchingClose(toks []phpTok, open int) int {
	depth := 0
	for i := open; i < len(toks); i++ {
		if toks[i].K != ptOp {
			continue
		}
		switch toks[i].S {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// phpForBound: `$i < 3` → 3 iterasi tetap; `$i = 10; $i > 0` → 10; syarat kosong / while (true) → tanpa batas.
func phpForBound(init, cond []phpTok) (k int64, data, unbounded bool) {
	if len(cond) == 0 {
		return 0, true, true
	}
	if len(cond) == 1 && (cond[0].kw("true") || (cond[0].K == ptNum && cond[0].S != "0")) {
		return 0, true, true
	}
	if len(splitTop(cond, ",")) > 1 {
		return 0, true, false
	}
	op := -1
	for i, t := range cond {
		if t.K == ptOp {
			switch t.S {
			case "<", "<=", ">", ">=", "!=", "!==", "<>":
				op = i
			case "&&", "||", "and", "or":
				return 0, true, false
			}
		}
	}
	if op < 0 || op+2 != len(cond) || cond[op+1].K != ptNum {
		return 0, true, false
	}
	v, err := strconv.ParseInt(strings.ReplaceAll(cond[op+1].S, "_", ""), 0, 64)
	if err != nil || v < 0 {
		return 0, true, false
	}
	if v > 0 {
		if cond[op].S == "<=" {
			v++
		}
		return v, false, false
	}
	// hitung mundur: $i = 10; $i > 0
	if len(init) == 3 && init[0].K == ptVar && init[1].is("=") && init[2].K == ptNum {
		if start, err := strconv.ParseInt(init[2].S, 0, 64); err == nil && start > 0 {
			return start, false, false
		}
	}
	return 0, true, false
}

// exprStmt menangani pola pernyataan khusus (closure ke variabel, new, include) lalu biaya ekspresinya.
func (c *phpCostCtx) exprStmt(sc *phpScope, toks []phpTok) cst {
	if len(toks) >= 3 && toks[0].K == ptVar && toks[1].is("=") {
		rhs := toks[2:]
		if isClosureStart(rhs, 0) {
			body, _ := c.closure(sc, rhs, 0)
			sc.closures[toks[0].S] = body
			return newCst()
		}
		if rhs[0].kw("new") && len(rhs) > 1 && rhs[1].K == ptIdent {
			sc.vars[toks[0].S] = phpShort(rhs[1].S)
		} else if len(rhs) > 2 && rhs[0].K == ptIdent && rhs[1].is("::") && c.m.isModel(sc.file, rhs[0].S) {
			sc.vars[toks[0].S] = phpShort(rhs[0].S)
		}
		// $builder = $db->table('x')->where(…); → $builder->get() nanti tetap dikenali sebagai query
		if rhs[0].K == ptVar || rhs[0].K == ptIdent {
			if ch, next := parsePHPChain(rhs, 0); ch != nil && next == len(rhs) && len(ch.Segs) > 0 {
				cost, db := c.chainCostDB(sc, ch)
				if db {
					sc.vars[toks[0].S] = phpBuilderVar
				}
				return cost
			}
		}
	}
	if len(toks) > 0 && toks[0].K == ptIdent {
		switch strings.ToLower(toks[0].S) {
		case "require", "require_once", "include", "include_once":
			out := newCst()
			if f := c.resolveInclude(sc.file, toks[1:]); f != nil {
				c.edge("file:" + f.Rel)
				out.add(c.fileCost(f))
			}
			return out
		}
	}
	return c.expr(sc, toks)
}

// resolveInclude: 'x.php', __DIR__ . '/x.php', dirname(__FILE__) . '/x.php' → berkas lokal.
func (c *phpCostCtx) resolveInclude(from *phpFile, toks []phpTok) *phpFile {
	var b strings.Builder
	for _, t := range toks {
		if t.K == ptStr {
			if t.Interp {
				return nil
			}
			b.WriteString(t.S)
		}
		if t.K == ptVar {
			return nil
		}
	}
	p := b.String()
	if p == "" {
		return nil
	}
	dir := path.Dir(from.Rel)
	for _, cand := range []string{path.Join(dir, p), path.Clean(strings.TrimPrefix(p, "/"))} {
		if f := c.m.byRel[cand]; f != nil {
			return f
		}
	}
	return nil
}

func isClosureStart(toks []phpTok, i int) bool {
	if i < len(toks) && toks[i].kw("static") {
		i++
	}
	if i >= len(toks) {
		return false
	}
	if toks[i].kw("function") || toks[i].kw("fn") {
		j := i + 1
		if j < len(toks) && toks[j].is("&") {
			j++
		}
		return j < len(toks) && toks[j].is("(")
	}
	return false
}

// closure menghitung badan closure yang dimulai di toks[i]; mengembalikan biaya dan indeks setelah closure.
func (c *phpCostCtx) closure(sc *phpScope, toks []phpTok, i int) (cst, int) {
	if toks[i].kw("static") {
		i++
	}
	arrow := toks[i].kw("fn")
	i++
	if i < len(toks) && toks[i].is("&") {
		i++
	}
	end := matchingClose(toks, i)
	if end < 0 {
		return newCst(), len(toks)
	}
	// parameter closure bertipe ikut dikenal (function (User $u) …)
	for _, part := range splitTop(toks[i+1:end], ",") {
		var typ string
		for _, t := range part {
			if t.K == ptIdent && !isPHPScalar(t.S) {
				typ = phpShort(t.S)
			}
			if t.K == ptVar && typ != "" {
				sc.vars[t.S] = typ
				break
			}
		}
	}
	i = end + 1
	if arrow {
		for i < len(toks) && !toks[i].is("=>") {
			i++
		}
		i++
		start := i
		depth := 0
		for i < len(toks) {
			t := toks[i]
			if t.K == ptOp {
				switch t.S {
				case "(", "[", "{":
					depth++
				case ")", "]", "}":
					if depth == 0 {
						goto done
					}
					depth--
				case ",", ";":
					if depth == 0 {
						goto done
					}
				}
			}
			i++
		}
	done:
		body := newCst()
		if start < i {
			body.tick(1) // return implisit
			body.add(c.expr(sc, toks[start:i]))
		}
		return body, i
	}
	if i < len(toks) && toks[i].kw("use") && i+1 < len(toks) && toks[i+1].is("(") {
		i = matchingClose(toks, i+1) + 1
	}
	for i < len(toks) && !toks[i].is("{") {
		i++
	}
	if i >= len(toks) {
		return newCst(), i
	}
	close := matchingClose(toks, i)
	if close < 0 {
		close = len(toks)
	}
	stmts := parsePHPTokens(sc.file, sc.cls, toks[i+1:close])
	return c.block(sc, stmts), close + 1
}

// ---------- ekspresi: rantai panggilan ----------

type phpSeg struct {
	Op   string // "->", "::", "call", "index"
	Name string
	Args [][]phpTok
	Line int
}

type phpChain struct {
	Root     phpTok
	RootKind string // var, name, new
	Segs     []phpSeg
	NewArgs  [][]phpTok
}

var phpExprKeywords = map[string]bool{
	"return": true, "echo": true, "print": true, "throw": true, "yield": true, "new": true, "clone": true, "instanceof": true,
	"and": true, "or": true, "xor": true, "not": true, "global": true, "unset": true, "isset": true,
	"empty": true, "array": true, "list": true, "match": true, "fn": true, "function": true, "use": true, "as": true,
	"true": true, "false": true, "null": true, "break": true, "continue": true, "else": true, "default": true, "case": true,
	"goto": true, "exit": true, "die": true, "from": true, "readonly": true, "const": true, "eval": true,
}

// parseChain membaca satu rantai dari toks[i]: $a->b()->c, Kelas::m(), fungsi(), new Kelas(…)->m().
func parsePHPChain(toks []phpTok, i int) (*phpChain, int) {
	ch := &phpChain{Root: toks[i]}
	switch {
	case toks[i].kw("new"):
		if i+1 >= len(toks) || toks[i+1].K != ptIdent {
			return nil, i + 1
		}
		ch.Root, ch.RootKind = toks[i+1], "new"
		i += 2
		if i < len(toks) && toks[i].is("(") {
			end := matchingClose(toks, i)
			if end < 0 {
				return ch, len(toks)
			}
			ch.NewArgs = splitTop(toks[i+1:end], ",")
			i = end + 1
		}
	case toks[i].K == ptVar:
		ch.RootKind = "var"
		i++
	default:
		ch.RootKind = "name"
		i++
	}
	for i < len(toks) {
		t := toks[i]
		switch {
		case (t.is("->") || t.is("?->") || t.is("::")) && i+1 < len(toks) && (toks[i+1].K == ptIdent || toks[i+1].K == ptVar):
			op := "->"
			if t.is("::") {
				op = "::"
			}
			ch.Segs = append(ch.Segs, phpSeg{Op: op, Name: toks[i+1].S, Line: toks[i+1].Line})
			i += 2
		case t.is("("):
			end := matchingClose(toks, i)
			if end < 0 {
				end = len(toks)
			}
			var args [][]phpTok
			if end > i+1 {
				args = splitTop(toks[i+1:min(end, len(toks))], ",")
			}
			ch.Segs = append(ch.Segs, phpSeg{Op: "call", Args: args, Line: t.Line})
			i = end + 1
		case t.is("["):
			end := matchingClose(toks, i)
			if end < 0 {
				end = len(toks)
			}
			ch.Segs = append(ch.Segs, phpSeg{Op: "index", Args: [][]phpTok{toks[i+1 : min(end, len(toks))]}, Line: t.Line})
			i = end + 1
		default:
			return ch, i
		}
	}
	return ch, i
}

// expr menjumlahkan biaya panggilan dalam potongan ekspresi.
func (c *phpCostCtx) expr(sc *phpScope, toks []phpTok) cst {
	out := newCst()
	for i := 0; i < len(toks); {
		t := toks[i]
		switch {
		case isClosureStart(toks, i):
			body, next := c.closure(sc, toks, i)
			out.add(body)
			i = next
		case t.kw("match") && i+1 < len(toks) && toks[i+1].is("("):
			next := c.match(sc, toks, i, &out)
			i = next
		case t.K == ptVar || t.kw("new") || (t.K == ptIdent && !phpExprKeywords[strings.ToLower(t.S)]):
			// `->x` setelah `)` (mis. (new A)->b()) dilewati: rantai dimulai di akar yang dikenal saja
			ch, next := parsePHPChain(toks, i)
			if ch != nil {
				out.add(c.chainCost(sc, ch))
			}
			if next <= i {
				next = i + 1
			}
			i = next
		default:
			i++
		}
	}
	return out
}

// match (x) { a, b => r1, default => r2 } — subjek + lengan termahal.
func (c *phpCostCtx) match(sc *phpScope, toks []phpTok, i int, out *cst) int {
	end := matchingClose(toks, i+1)
	if end < 0 {
		return len(toks)
	}
	out.add(c.expr(sc, toks[i+2:end]))
	j := end + 1
	if j >= len(toks) || !toks[j].is("{") {
		return j
	}
	close := matchingClose(toks, j)
	if close < 0 {
		close = len(toks)
	}
	best := newCst()
	for _, arm := range splitTop(toks[j+1:min(close, len(toks))], ",") {
		best = heavier(best, c.expr(sc, arm))
	}
	out.add(best)
	return close + 1
}

var (
	phpSortFuncs = map[string]bool{
		"sort": true, "rsort": true, "usort": true, "uasort": true, "uksort": true, "asort": true, "arsort": true, "ksort": true,
		"krsort": true, "natsort": true, "natcasesort": true, "array_multisort": true,
	}
	phpIterFuncs = map[string]bool{
		"array_map": true, "array_filter": true, "array_walk": true, "array_walk_recursive": true, "array_reduce": true,
		"preg_replace_callback": true, "iterator_apply": true, "array_find": true, "array_any": true, "array_all": true,
		"array_find_key": true,
	}
	// method koleksi Laravel yang menjalankan callback sekali per elemen
	phpCollIter = map[string]bool{
		"each": true, "map": true, "filter": true, "reject": true, "flatmap": true, "transform": true, "mapwithkeys": true,
		"groupby": true, "keyby": true, "every": true, "contains": true, "first": true, "last": true, "sum": true, "reduce": true,
		"partition": true, "unique": true, "search": true, "takewhile": true, "takeuntil": true, "skipwhile": true,
		"skipuntil": true, "maptogroups": true, "eachspread": true, "mapspread": true, "mapinto": true, "countby": true,
		"min": true, "max": true, "avg": true, "where": false, "chunk": true, "chunkbyid": true, "lazy": true, "pipe": false,
		"tap": false, "when": false, "unless": false, "transaction": false, "sole": true, "doesntcontain": true, "some": true,
		"firstwhere": false, "foreach": true,
	}
	phpCollSort = map[string]bool{"sort": true, "sortby": true, "sortbydesc": true, "sortdesc": true, "sortkeys": true, "sortkeysdesc": true, "sortkeysusing": true}
	// pemicu query Eloquent / query builder (method akhir)
	phpQueryTerminals = map[string]bool{
		"get": true, "first": true, "firstorfail": true, "find": true, "findorfail": true, "findmany": true, "findornew": true,
		"all": true, "paginate": true, "simplepaginate": true, "cursorpaginate": true, "count": true, "sum": true, "avg": true,
		"average": true, "max": true, "min": true, "exists": true, "doesntexist": true, "pluck": true, "value": true,
		"create": true, "insert": true, "insertgetid": true, "insertorignore": true, "update": true, "delete": true,
		"save": true, "destroy": true, "upsert": true, "updateorcreate": true, "firstorcreate": true, "firstornew": true,
		"updateorinsert": true, "increment": true, "decrement": true, "chunk": true, "chunkbyid": true, "each": true,
		"cursor": true, "lazy": true, "lazybyid": true, "sole": true, "firstwhere": true, "forcedelete": true, "restore": true,
		"truncate": true, "createmany": true, "savemany": true, "attach": true, "detach": true, "sync": true, "toggle": true,
		"syncwithoutdetaching": true, "updateexistingpivot": true, "implode": true, "tosql": false, "dd": false,
		// CodeIgniter 4 (Model & Query Builder)
		"findall": true, "findcolumn": true, "insertbatch": true, "updatebatch": true, "getresult": true,
		"getresultarray": true, "getrow": true, "getrowarray": true, "countallresults": true, "countall": true,
		"getwhere": true, "replace": true, "emptytable": true,
	}
	phpQueryBuilders = map[string]bool{
		"where": true, "orwhere": true, "wherein": true, "wherenotin": true, "wherenull": true, "wherenotnull": true,
		"wherebetween": true, "wherehas": true, "wheredoesnthave": true, "wheredate": true, "whereraw": true, "wherekey": true,
		"wherecolumn": true, "whereexists": true, "wherelike": true, "orderby": true, "orderbydesc": true, "latest": true,
		"oldest": true, "with": true, "withcount": true, "withsum": true, "withtrashed": true, "onlytrashed": true, "has": true,
		"doesnthave": true, "join": true, "leftjoin": true, "rightjoin": true, "select": true, "selectraw": true,
		"addselect": true, "groupby": true, "having": true, "limit": true, "take": true, "skip": true, "offset": true,
		"query": true, "table": true, "distinct": true, "inrandomorder": true, "lockforupdate": true, "sharedlock": true,
		"withoutglobalscopes": true, "newquery": true, "on": true, "connection": true, "from": true, "fromsub": true,
		"orderbyraw": true, "whereyear": true, "wheremonth": true, "wheretime": true, "orwherehas": true, "scopes": true,
		"forpage": true, "unless": false,
	}
	phpDBFacadeDirect = map[string]bool{
		"select": true, "selectone": true, "insert": true, "update": true, "delete": true, "statement": true,
		"unprepared": true, "affectingstatement": true, "scalar": true, "cursor": true,
	}
	phpDBFuncs = map[string]bool{
		"mysqli_query": true, "mysqli_real_query": true, "mysqli_multi_query": true, "mysqli_stmt_execute": true,
		"mysqli_execute": true, "mysqli_execute_query": true, "pg_query": true, "pg_query_params": true, "pg_execute": true,
		"pg_send_query": true, "mysql_query": true, "odbc_exec": true, "sqlsrv_query": true, "oci_execute": true,
	}
	// $pdo->query(), $db->exec(), $stmt->execute(), $wpdb->get_results() …
	phpDBVarRe   = regexp.MustCompile(`(?i)^\$?(pdo|db|dbh|conn|connection|mysqli|link|database|wpdb|mysql|pg)$`)
	phpStmtVarRe = regexp.MustCompile(`(?i)^\$?(stmt|sth|statement|query|prepared|ps)$`)
	phpPDOCalls  = map[string]bool{"query": true, "exec": true, "execute": true, "get_results": true, "get_row": true,
		"get_var": true, "get_col": true, "insert": true, "update": true, "delete": true, "replace": true, "execute_query": true,
		"real_query": true, "multi_query": true}
	// tulis/muat ulang langsung pada instance model: $user->save(), $post->load('x') …
	phpInstanceQuery = map[string]bool{"save": true, "delete": true, "update": true, "refresh": true, "load": true,
		"loadmissing": true, "loadcount": true, "push": true, "forcedelete": true, "restore": true, "increment": true,
		"decrement": true, "touch": true, "fresh": true, "saveorfail": true, "deleteorfail": true, "updatequietly": true,
		"savequietly": true}
	// relasi tanpa argumen lalu pemicu: $user->posts()->get(), $order->items()->count()
	phpRelTerminals = map[string]bool{"get": true, "first": true, "count": true, "exists": true, "paginate": true,
		"pluck": true, "sum": true, "find": true, "findorfail": true, "firstorfail": true, "create": true, "save": true,
		"delete": true, "update": true, "attach": true, "detach": true, "sync": true, "toggle": true, "createmany": true,
		"savemany": true, "updateorcreate": true, "firstorcreate": true}
	// nama method yang terlalu umum untuk ditebak pemiliknya bila tipe penerima tidak diketahui
	phpGenericMethods = map[string]bool{"handle": true, "get": true, "set": true, "find": true, "first": true, "save": true,
		"update": true, "delete": true, "create": true, "all": true, "make": true, "run": true, "execute": true, "render": true,
		"toarray": true, "boot": true, "register": true, "__construct": true, "__invoke": true, "map": true, "filter": true,
		"each": true, "count": true, "has": true, "put": true, "push": true, "query": true, "where": true, "with": true,
		"load": true, "send": true, "call": true, "process": true, "index": true, "show": true, "store": true, "edit": true,
		"destroy": true, "add": true, "remove": true, "list": true, "init": true, "validate": true, "rules": true,
		"authorize": true, "build": true, "fill": true, "json": true, "view": true, "response": true, "redirect": true}
	phpCollVars  = map[string]bool{"$collection": true, "$items": true, "$arr": true, "$array": true, "$data": true, "$list": true, "$rows": true}
	phpNonDBVars = map[string]bool{"$this": true, "$request": true, "$req": true, "$cache": true, "$session": true,
		"$response": true, "$collection": true, "$items": true, "$arr": true, "$array": true, "$data": true}
)

func phpHasClosureArg(args [][]phpTok) bool {
	for _, a := range args {
		if isClosureStart(stripNamedArg(a), 0) {
			return true
		}
	}
	return false
}

// stripNamedArg: `callback: fn ($x) => …` → `fn ($x) => …`; `...$xs` → `$xs`.
func stripNamedArg(a []phpTok) []phpTok {
	if len(a) >= 2 && a[0].K == ptIdent && a[1].is(":") {
		return a[2:]
	}
	if len(a) >= 1 && a[0].is("...") {
		return a[1:]
	}
	return a
}

// args menghitung argumen: closure → cb (dikalikan pemanggil), selainnya → out (sekali).
func (c *phpCostCtx) args(sc *phpScope, args [][]phpTok, cb, out *cst) {
	for _, a := range args {
		a = stripNamedArg(a)
		if cb != nil && isClosureStart(a, 0) {
			body, next := c.closure(sc, a, 0)
			cb.add(body)
			if next < len(a) {
				out.add(c.expr(sc, a[next:]))
			}
			continue
		}
		out.add(c.expr(sc, a))
	}
}

// phpBuilderVar: penanda tipe variabel yang menyimpan query builder yang belum dieksekusi.
const phpBuilderVar = "@builder"

func (c *phpCostCtx) chainCost(sc *phpScope, ch *phpChain) cst {
	out, _ := c.chainCostDB(sc, ch)
	return out
}

// chainCostDB: biaya satu rantai panggilan, kiri ke kanan; db = rantai berakhir sebagai query builder
// yang belum dieksekusi.
func (c *phpCostCtx) chainCostDB(sc *phpScope, ch *phpChain) (cst, bool) {
	out := newCst()
	for _, a := range ch.NewArgs {
		out.add(c.expr(sc, a))
	}
	recv := "" // kelas penerima yang diketahui (untuk ->method berikutnya)
	if ch.RootKind == "new" {
		recv = phpShort(ch.Root.S)
		if cls := c.m.class(sc.file, ch.Root.S); cls != nil {
			if ctor := c.m.method(cls, "__construct"); ctor != nil {
				c.edge(phpNodeKey(ctor))
				out.add(c.funcCost(ctor))
			}
		}
	}
	rootVar := ""
	if ch.RootKind == "var" {
		rootVar = ch.Root.S
		if rootVar == "$this" && sc.cls != nil {
			recv = sc.cls.Name
		} else {
			recv = sc.vars[rootVar]
		}
	}
	dbish := false
	if ch.RootKind == "var" && (phpDBVarRe.MatchString(rootVar) || recv == phpBuilderVar || (rootVar != "$this" && c.m.isModel(sc.file, recv))) {
		dbish = true
	}
	if recv == phpBuilderVar {
		recv = ""
	}
	calls := 0
	relCall := false // ada panggilan ->relasi() tanpa argumen sebelumnya
	for j, seg := range ch.Segs {
		switch seg.Op {
		case "index":
			for _, a := range seg.Args {
				out.add(c.expr(sc, a))
			}
			recv = ""
			continue
		case "->", "::":
			if seg.Op == "->" && j+1 < len(ch.Segs) && ch.Segs[j+1].Op == "call" {
				continue
			}
			if seg.Op == "->" {
				// akses properti: $this->repo → tipe properti
				if c2 := c.m.class(sc.file, recv); c2 != nil {
					recv = c.m.propType(c2, seg.Name)
				} else if recv == "" && j == 0 && rootVar == "$this" && sc.cls != nil {
					recv = c.m.propType(sc.cls, seg.Name)
				} else {
					recv = ""
				}
				if phpDBVarRe.MatchString(seg.Name) || c.m.isModel(sc.file, recv) {
					dbish = true // $this->db->…, $this->model->where()->paginate()
				}
			}
			continue
		}
		// panggilan
		calls++
		name, op := "", ""
		if j > 0 && (ch.Segs[j-1].Op == "->" || ch.Segs[j-1].Op == "::") {
			name, op = ch.Segs[j-1].Name, ch.Segs[j-1].Op
		} else if j == 0 && ch.RootKind == "name" {
			name, op = ch.Root.S, "func"
		} else if j == 0 && ch.RootKind == "var" {
			if cl, ok := sc.closures[rootVar]; ok {
				out.add(cl)
				c.args(sc, seg.Args, nil, &out)
				continue
			}
		}
		lname := strings.ToLower(phpShort(name))
		staticRoot := op == "::" && j == 1 && ch.RootKind == "name"

		// 1) fungsi/method lokal (beberapa kandidat → yang termahal, seperti resolveFuncs Go)
		if fns := c.resolve(sc, ch, j, op, name, recv); len(fns) > 0 {
			best := newCst()
			for _, f := range fns {
				c.edge(phpNodeKey(f))
				best = heavier(best, c.funcCost(f))
			}
			fn := fns[0]
			out.add(best)
			c.args(sc, seg.Args, nil, &out)
			recv = ""
			if staticRoot && c.m.isModel(sc.file, ch.Root.S) {
				dbish = true // scope lokal model: User::active()->get()
			}
			if len(seg.Args) == 0 && fn.Class != nil && c.m.isModel(fn.Class.file, fn.Class.Name) {
				relCall = true // relasi lokal model: $user->posts()->get()
			}
			continue
		}
		hasCB := phpHasClosureArg(seg.Args)
		// 2) pengurutan: n·log n × pembanding
		if (op == "func" && phpSortFuncs[lname]) || (op == "->" && phpCollSort[lname]) {
			cmp := newCst()
			cmp.tick(1)
			c.args(sc, seg.Args, &cmp, &out)
			out.add(cmp.scaled(1, 1, 1))
			continue
		}
		// 3) callback per elemen
		if hasCB && ((op == "func" && phpIterFuncs[lname]) || (op != "func" && phpCollIter[lname])) {
			body := newCst()
			c.args(sc, seg.Args, &body, &out)
			out.add(body.scaled(1, 1, 0))
			if dbish && phpQueryTerminals[lname] {
				out.queries.addTerm(term{}, 1)
				dbish = false
			}
			continue
		}
		c.args(sc, seg.Args, nil, &out)
		// 4) query database
		switch {
		case op == "func" && phpDBFuncs[lname]:
			out.queries.addTerm(term{}, 1)
		case staticRoot && strings.EqualFold(phpShort(ch.Root.S), "DB"):
			if phpDBFacadeDirect[lname] {
				out.queries.addTerm(term{}, 1)
			} else {
				dbish = true
			}
		case staticRoot && c.m.isModel(sc.file, ch.Root.S):
			if phpQueryTerminals[lname] {
				out.queries.addTerm(term{}, 1)
			} else if phpQueryBuilders[lname] {
				dbish = true
			}
		case op == "->" && j == 1 && recv != "" && c.m.isModel(sc.file, recv) && phpQueryTerminals[lname]:
			out.queries.addTerm(term{}, 1) // $model->findAll() / $userModel->find($id)
		case op == "->" && dbish && phpQueryTerminals[lname]:
			out.queries.addTerm(term{}, 1)
			dbish, relCall = false, false
		case op == "->" && (phpDBVarRe.MatchString(rootVar) || phpStmtVarRe.MatchString(rootVar)) && calls == 1 && phpPDOCalls[lname]:
			out.queries.addTerm(term{}, 1)
		case op == "->" && relCall && phpRelTerminals[lname]:
			out.queries.addTerm(term{}, 1)
			relCall = false
		case op == "->" && j == 1 && ch.RootKind == "var" && !phpNonDBVars[strings.ToLower(rootVar)] && phpInstanceQuery[lname] &&
			(recv == "" || c.m.isModel(sc.file, recv)) && !phpStmtVarRe.MatchString(rootVar):
			out.queries.addTerm(term{}, 1)
		case op == "->" && phpQueryBuilders[lname] && !phpCollVars[strings.ToLower(rootVar)]:
			dbish = true
		case op == "->" && len(seg.Args) == 0 && j == 1 && ch.RootKind == "var" && !phpNonDBVars[strings.ToLower(rootVar)]:
			relCall = true
		}
		recv = ""
	}
	return out, dbish
}

// resolve mencari fungsi/method lokal untuk panggilan ke-j dalam rantai.
func (c *phpCostCtx) resolve(sc *phpScope, ch *phpChain, j int, op, name, recv string) []*phpFunc {
	switch op {
	case "func":
		if fn := c.m.funcs[strings.ToLower(phpShort(name))]; fn != nil {
			return []*phpFunc{fn}
		}
	case "::":
		if j != 1 || ch.RootKind != "name" {
			return nil
		}
		var cls *phpClass
		switch strings.ToLower(ch.Root.S) {
		case "self", "static":
			cls = sc.cls
		case "parent":
			if sc.cls != nil {
				cls = c.m.class(sc.cls.file, sc.cls.Extends)
			}
		default:
			cls = c.m.class(sc.file, ch.Root.S)
		}
		if cls != nil {
			return c.m.methods(cls, name)
		}
		return nil
	case "->":
		if cls := c.m.class(sc.file, recv); cls != nil {
			return c.m.methods(cls, name)
		}
		if recv != "" { // tipe diketahui tetapi bukan kelas lokal (pustaka luar)
			return nil
		}
		ln := strings.ToLower(name)
		if phpGenericMethods[ln] || phpQueryTerminals[ln] || phpQueryBuilders[ln] || phpCollIter[ln] {
			return nil
		}
		if cands := c.m.methodsByName[ln]; len(cands) == 1 {
			return cands
		}
	}
	return nil
}
