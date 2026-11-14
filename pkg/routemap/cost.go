package routemap

import (
	"go/ast"
	"go/token"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Model biaya proses ("tick") — hitungan statis, bukan pengukuran waktu:
//
//	1 tick            = satu pernyataan yang dieksekusi (assign, panggilan, return, cek syarat, i++ …)
//	if / switch       = 1 tick (cek syarat) + cabang TERMAHAL (kasus terburuk)
//	loop atas data    = (n + 1) cek syarat + n × isi loop        → menaikkan pangkat n
//	loop batas tetap  = (k + 1) cek syarat + k × isi loop        → hanya mengalikan koefisien
//	fungsi lokal      = rumus fungsi itu ikut dijumlahkan (dalam loop: dikalikan n)
//	pustaka luar      = 1 tick; sort = n·log n; callback iterator (Map/Filter/Walk …) = n × isi callback
//	query database    = dihitung terpisah sebagai Q(n); query di dalam loop → Q(n) = n + 1 (N+1 query)
//
// n = jumlah data yang diproses. Semua koleksi dianggap berukuran n (kasus terburuk seragam), sehingga
// rumusnya berbentuk T(n) = a·n² + b·n + c dan kelas pertumbuhannya (Big-O) adalah suku tertinggi.

// term = n^N · (log n)^L.
type term struct{ N, L int }

// poly = jumlah suku berkoefisien bulat.
type poly map[term]int64

const (
	maxCostDegree = 6
	maxCostCoef   = int64(1e15)
)

func satAdd(a, b int64) int64 {
	if s := a + b; s < maxCostCoef {
		return s
	}
	return maxCostCoef
}

func satMul(a, b int64) int64 {
	if a == 0 || b == 0 {
		return 0
	}
	if a > maxCostCoef/b {
		return maxCostCoef
	}
	return a * b
}

func (p poly) addTerm(t term, c int64) {
	if c == 0 {
		return
	}
	if t.N > maxCostDegree {
		t.N = maxCostDegree
	}
	p[t] = satAdd(p[t], c)
}

func (p poly) add(o poly) {
	for t, c := range o {
		p.addTerm(t, c)
	}
}

// times mengembalikan p × k·n^dn·(log n)^dl.
func (p poly) times(k int64, dn, dl int) poly {
	out := poly{}
	for t, c := range p {
		out.addTerm(term{t.N + dn, t.L + dl}, satMul(c, k))
	}
	return out
}

func (p poly) eval(n float64) float64 {
	lg := math.Max(1, math.Log2(n))
	sum := 0.0
	for t, c := range p {
		sum += float64(c) * math.Pow(n, float64(t.N)) * math.Pow(lg, float64(t.L))
	}
	return sum
}

func termLess(a, b term) bool {
	if a.N != b.N {
		return a.N < b.N
	}
	return a.L < b.L
}

// lead = suku tertinggi (penentu Big-O).
func (p poly) lead() (term, int64) {
	var best term
	var coef int64
	for t, c := range p {
		if c > 0 && (coef == 0 || termLess(best, t)) {
			best, coef = t, c
		}
	}
	return best, coef
}

func (p poly) sorted() []term {
	ts := make([]term, 0, len(p))
	for t, c := range p {
		if c > 0 {
			ts = append(ts, t)
		}
	}
	sort.Slice(ts, func(i, j int) bool { return termLess(ts[j], ts[i]) })
	return ts
}

var superDigits = []string{"⁰", "¹", "²", "³", "⁴", "⁵", "⁶", "⁷", "⁸", "⁹"}

func termString(t term) string {
	s := ""
	if t.N >= 1 {
		s = "n"
		if t.N >= 2 {
			s += superDigits[t.N%10]
		}
	}
	if t.L >= 1 {
		if s != "" {
			s += " "
		}
		s += "log n"
		if t.L >= 2 {
			s = strings.TrimSuffix(s, "log n") + "log" + superDigits[t.L%10] + " n"
		}
	}
	return s
}

func (p poly) String() string {
	var parts []string
	for _, t := range p.sorted() {
		c, ts := p[t], termString(t)
		switch {
		case ts == "":
			parts = append(parts, strconv.FormatInt(c, 10))
		case c == 1:
			parts = append(parts, ts)
		default:
			parts = append(parts, strconv.FormatInt(c, 10)+ts)
		}
	}
	if len(parts) == 0 {
		return "0"
	}
	return strings.Join(parts, " + ")
}

func bigO(p poly) (label, class string) {
	t, c := p.lead()
	if c == 0 || (t.N == 0 && t.L == 0) {
		return "O(1)", "constant"
	}
	label = "O(" + termString(t) + ")"
	switch {
	case t.N == 0:
		class = "log"
	case t.N == 1 && t.L == 0:
		class = "linear"
	case t.N == 1:
		class = "linearithmic"
	case t.N == 2 && t.L == 0:
		class = "quadratic"
	case t.N == 3 && t.L == 0:
		class = "cubic"
	default:
		class = "polynomial"
	}
	return label, class
}

// cst = biaya satu potong kode: tick, query DB, dan tanda bahaya.
type cst struct {
	ticks, queries poly
	recursive      bool
	unbounded      bool
	depth          int // kedalaman loop atas data
}

func newCst() cst { return cst{ticks: poly{}, queries: poly{}} }

func (c *cst) tick(n int64) { c.ticks.addTerm(term{}, n) }

func (c *cst) add(o cst) {
	c.ticks.add(o.ticks)
	c.queries.add(o.queries)
	c.recursive = c.recursive || o.recursive
	c.unbounded = c.unbounded || o.unbounded
	if o.depth > c.depth {
		c.depth = o.depth
	}
}

// scaled = biaya dikalikan k·n^dn·(log n)^dl (isi loop).
func (c cst) scaled(k int64, dn, dl int) cst {
	out := cst{ticks: c.ticks.times(k, dn, dl), queries: c.queries.times(k, dn, dl), recursive: c.recursive, unbounded: c.unbounded, depth: c.depth}
	if dn > 0 {
		out.depth += dn
	}
	return out
}

// heavier: cabang mana yang lebih mahal untuk n besar (kasus terburuk if/switch).
func heavier(a, b cst) cst {
	const probe = 1e6
	av, bv := a.ticks.eval(probe)+a.queries.eval(probe)*1e3, b.ticks.eval(probe)+b.queries.eval(probe)*1e3
	w, l := a, b
	if bv > av {
		w, l = b, a
	}
	w.recursive = w.recursive || l.recursive
	w.unbounded = w.unbounded || l.unbounded
	if l.depth > w.depth {
		w.depth = l.depth
	}
	return w
}

var (
	// Akar pemanggilan yang hampir pasti koneksi database (db, tx, s.db, h.DB, conn, pool …).
	dbRootRe = regexp.MustCompile(`(?i)^(?:g?db|tx|orm|conn|pool|sqlx?|pg|pgx|mongo|coll|collection|stmt|database|querier|queries|q)$|db$|DB$|Tx$|Conn$|Pool$`)
	// Method akhir yang benar-benar mengirim query (bukan penyusun query).
	queryTerminals = map[string]bool{
		"Find": true, "First": true, "Last": true, "Take": true, "Create": true, "CreateInBatches": true, "Save": true,
		"Update": true, "Updates": true, "UpdateColumn": true, "UpdateColumns": true, "Delete": true, "Count": true,
		"Scan": true, "Pluck": true, "Exec": true, "Row": true, "Rows": true, "FirstOrCreate": true,
		"Query": true, "QueryRow": true, "QueryContext": true, "QueryRowContext": true, "ExecContext": true,
		"QueryRowx": true, "Queryx": true, "NamedExec": true, "NamedQuery": true, "SelectContext": true, "GetContext": true,
		"FindOne": true, "InsertOne": true, "InsertMany": true, "UpdateOne": true, "UpdateMany": true, "DeleteOne": true,
		"DeleteMany": true, "Aggregate": true, "CountDocuments": true, "ReplaceOne": true, "FindOneAndUpdate": true,
	}
	// Penyusun query: rantai yang memuatnya dianggap rantai database walau akarnya tidak dikenali.
	queryBuilders = map[string]bool{
		"Where": true, "Model": true, "Table": true, "Preload": true, "Joins": true, "Raw": true, "Order": true,
		"Limit": true, "Offset": true, "Group": true, "Having": true, "Unscoped": true, "Scopes": true, "Clauses": true,
		"Omit": true, "Distinct": true, "WithContext": true, "Session": true, "Debug": true,
	}
	// Pustaka yang menjalankan callback sekali per elemen. `Do` sengaja tidak termasuk: singleflight.Do, sync.Once.Do,
	// dan retry.Do menjalankan callback-nya sekali, bukan per data.
	iterCallbackRe = regexp.MustCompile(`^(?:Map|Filter|ForEach|Each|Reduce|Walk|WalkDir|Range|IndexFunc|ContainsFunc|DeleteFunc|Any|All|Find|FilterMap|FlatMap|GroupBy|KeyBy|Reject|Uniq|UniqBy|Times)$`)
	sortFuncs      = map[string]bool{
		"sort.Slice": true, "sort.SliceStable": true, "sort.Sort": true, "sort.Stable": true, "sort.Strings": true,
		"sort.Ints": true, "sort.Float64s": true, "slices.Sort": true, "slices.SortFunc": true, "slices.SortStableFunc": true,
	}
)

// costCtx menghitung biaya fungsi lokal sebuah modul dengan memo; siklus (rekursi) dipotong & ditandai.
type costCtx struct {
	m    *module
	memo map[string]cst
	busy map[string]bool
	// inLoop[fungsi][terpanggil]: fungsi lokal yang dipanggil di dalam loop atas data — hanya merekalah yang
	// pantas disebut "membawa query" pada temuan N+1 (Hotspot.Via).
	inLoop map[string]map[string]bool
}

type fnScope struct {
	fi       *fileInfo
	key      string         // fungsi yang sedang dihitung ("" = badan handler inline)
	closures map[string]cst // closure lokal: fn := func(){…} → biayanya dipakai saat fn() dipanggil
	limits   map[string]int64
	// loops: kedalaman loop (semua jenis), dataLoops: kedalaman loop atas data. exit: biaya `return` di dalam
	// loop — dijalankan paling banyak sekali, jadi ditambahkan sekali setelah loop terluar, tidak dikalikan.
	loops, dataLoops int
	exit             cst
}

func (c *costCtx) funcCost(key string) cst {
	if v, ok := c.memo[key]; ok {
		return v
	}
	f := c.m.funcs[key]
	if f == nil || f.Decl == nil || f.Decl.Body == nil {
		return newCst()
	}
	if c.busy[key] {
		out := newCst()
		out.tick(1)
		out.recursive = true
		return out
	}
	c.busy[key] = true
	out := c.block(newFnScope(f.file, key, f.Decl.Body.List), f.Decl.Body.List)
	delete(c.busy, key)
	c.memo[key] = out
	return out
}

func (c *costCtx) block(sc *fnScope, list []ast.Stmt) cst {
	out := newCst()
	for _, s := range list {
		out.add(c.stmt(sc, s))
	}
	return out
}

func (c *costCtx) stmt(sc *fnScope, s ast.Stmt) cst {
	out := newCst()
	switch x := s.(type) {
	case nil, *ast.EmptyStmt:
	case *ast.BlockStmt:
		return c.block(sc, x.List)
	case *ast.LabeledStmt:
		return c.stmt(sc, x.Stmt)
	case *ast.IfStmt:
		out.add(c.stmt(sc, x.Init))
		out.tick(1)
		out.add(c.expr(sc, x.Cond))
		out.add(heavier(c.block(sc, x.Body.List), c.stmt(sc, x.Else)))
	case *ast.ForStmt:
		out.add(c.stmt(sc, x.Init))
		k, data, unbounded := loopBound(x)
		iter := c.loopBody(sc, data, func() cst {
			iter := newCst()
			iter.tick(1) // cek syarat
			iter.add(c.expr(sc, x.Cond))
			iter.add(c.block(sc, x.Body.List))
			iter.add(c.stmt(sc, x.Post))
			return iter
		})
		out.add(c.afterLoop(sc, loopCost(iter, k, data)))
		out.unbounded = out.unbounded || unbounded
	case *ast.RangeStmt:
		out.add(c.expr(sc, x.X))
		k, data := rangeBoundIn(sc, x.X)
		iter := c.loopBody(sc, data, func() cst {
			iter := newCst()
			iter.tick(1)
			iter.add(c.block(sc, x.Body.List))
			return iter
		})
		out.add(c.afterLoop(sc, loopCost(iter, k, data)))
	case *ast.ReturnStmt:
		out.tick(1)
		out.add(c.expr(sc, x))
		if sc.loops > 0 {
			sc.exit = heavier(sc.exit, out)
			return newCst()
		}
	case *ast.SwitchStmt:
		out.add(c.stmt(sc, x.Init))
		out.tick(1)
		out.add(c.expr(sc, x.Tag))
		out.add(c.clauses(sc, x.Body))
	case *ast.TypeSwitchStmt:
		out.add(c.stmt(sc, x.Init))
		out.tick(1)
		out.add(c.clauses(sc, x.Body))
	case *ast.SelectStmt:
		out.tick(1)
		out.add(c.clauses(sc, x.Body))
	case *ast.AssignStmt:
		out.tick(1)
		for i, r := range x.Rhs {
			if lit, ok := r.(*ast.FuncLit); ok && i < len(x.Lhs) {
				if id, ok := x.Lhs[i].(*ast.Ident); ok {
					sc.closures[id.Name] = c.closureBlock(sc, lit.Body.List)
					continue
				}
			}
			out.add(c.expr(sc, r))
		}
		for _, l := range x.Lhs {
			if _, ok := l.(*ast.Ident); !ok {
				out.add(c.expr(sc, l))
			}
		}
	default: // ExprStmt, GoStmt, DeferStmt, IncDecStmt, SendStmt, DeclStmt, BranchStmt
		out.tick(1)
		out.add(c.expr(sc, s))
	}
	return out
}

// loopBody menghitung isi satu putaran loop dengan kedalaman loop yang benar.
func (c *costCtx) loopBody(sc *fnScope, data bool, body func() cst) cst {
	sc.loops++
	if data {
		sc.dataLoops++
	}
	iter := body()
	sc.loops--
	if data {
		sc.dataLoops--
	}
	return iter
}

// afterLoop: setelah loop terluar selesai, biaya `return` di dalamnya ditambahkan sekali (tidak dikalikan).
func (c *costCtx) afterLoop(sc *fnScope, cost cst) cst {
	if sc.loops == 0 {
		cost.add(sc.exit)
		sc.exit = newCst()
	}
	return cost
}

// closureBlock: badan func literal dihitung terpisah dari loop di sekitarnya — `return` di dalamnya hanya
// mengakhiri closure itu, dan closure yang dipanggil per elemen dikalikan oleh pemanggilnya.
func (c *costCtx) closureBlock(sc *fnScope, list []ast.Stmt) cst {
	loops, data, exit := sc.loops, sc.dataLoops, sc.exit
	sc.loops, sc.dataLoops, sc.exit = 0, 0, newCst()
	out := c.block(sc, list)
	sc.loops, sc.dataLoops, sc.exit = loops, data, exit
	return out
}

// clauses: case termahal dari switch/select.
func (c *costCtx) clauses(sc *fnScope, body *ast.BlockStmt) cst {
	best := newCst()
	if body == nil {
		return best
	}
	for _, cl := range body.List {
		cur := newCst()
		switch x := cl.(type) {
		case *ast.CaseClause:
			for _, e := range x.List {
				cur.add(c.expr(sc, e))
			}
			cur.add(c.block(sc, x.Body))
		case *ast.CommClause:
			cur.add(c.stmt(sc, x.Comm))
			cur.add(c.block(sc, x.Body))
		}
		best = heavier(best, cur)
	}
	return best
}

// loopCost: k > 0 → loop batas tetap (k iterasi); data → loop atas data (n iterasi). Selalu +1 cek syarat penutup.
func loopCost(iter cst, k int64, data bool) cst {
	var out cst
	if data {
		out = iter.scaled(1, 1, 0)
	} else {
		out = iter.scaled(k, 0, 0)
	}
	out.tick(1)
	return out
}

func intLit(e ast.Expr) (int64, bool) {
	if p, ok := e.(*ast.ParenExpr); ok {
		return intLit(p.X)
	}
	b, ok := e.(*ast.BasicLit)
	if !ok || b.Kind != token.INT {
		return 0, false
	}
	v, err := strconv.ParseInt(strings.ReplaceAll(b.Value, "_", ""), 0, 64)
	return v, err == nil && v >= 0
}

// forBound: `for i := 0; i < 3; i++` → 3 iterasi tetap; selain itu loop atas data. `for {}` ditandai tanpa batas.
func forBound(x *ast.ForStmt) (k int64, data, unbounded bool) {
	if x.Cond == nil {
		return 0, true, true
	}
	be, ok := x.Cond.(*ast.BinaryExpr)
	if !ok {
		return 0, true, false
	}
	switch be.Op {
	case token.LSS, token.LEQ, token.GTR, token.GEQ, token.NEQ:
	default:
		return 0, true, false
	}
	if v, ok := intLit(be.Y); ok && v > 0 {
		if be.Op == token.LEQ {
			v++
		}
		return v, false, false
	}
	// hitung mundur: for i := 10; i > 0; i--
	if v, ok := intLit(be.Y); ok && v == 0 {
		if as, ok := x.Init.(*ast.AssignStmt); ok && len(as.Rhs) == 1 {
			if start, ok := intLit(as.Rhs[0]); ok && start > 0 {
				return start, false, false
			}
		}
	}
	return 0, true, false
}

// rangeBound: `range 10` atau `range []T{a, b}` → batas tetap; selain itu data.
func rangeBound(e ast.Expr) (k int64, data bool) {
	if v, ok := intLit(e); ok && v > 0 {
		return v, false
	}
	if cl, ok := e.(*ast.CompositeLit); ok && len(cl.Elts) > 0 {
		return int64(len(cl.Elts)), false
	}
	return 0, true
}

// expr menjumlahkan biaya panggilan di dalam satu ekspresi/pernyataan sederhana.
func (c *costCtx) expr(sc *fnScope, n ast.Node) cst {
	out := newCst()
	if n == nil {
		return out
	}
	if e, ok := n.(ast.Expr); ok && e == nil {
		return out
	}
	ast.Inspect(n, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			out.add(c.closureBlock(sc, x.Body.List))
			return false
		case *ast.CallExpr:
			return c.call(sc, x, &out)
		}
		return true
	})
	return out
}

// call menambahkan biaya satu panggilan; false = argumennya sudah dihitung di sini.
func (c *costCtx) call(sc *fnScope, ce *ast.CallExpr, out *cst) bool {
	if id, ok := ce.Fun.(*ast.Ident); ok {
		if cl, ok := sc.closures[id.Name]; ok {
			out.add(cl)
			return true
		}
	}
	if fs := c.m.resolveFuncs(sc.fi, ce.Fun); len(fs) > 0 {
		best := newCst()
		for _, f := range fs {
			if f != nil {
				best = heavier(best, c.funcCost(f.Key))
				c.noteLoopCall(sc, f.Key)
			}
		}
		out.add(best)
		return true
	}
	name := exprStr(ce.Fun)
	sel, _ := ce.Fun.(*ast.SelectorExpr)
	switch {
	case sortFuncs[name]:
		cmp := newCst()
		cmp.tick(1)
		c.callbackArgs(sc, ce, &cmp, out)
		out.add(cmp.scaled(1, 1, 1))
		return false
	case sel != nil && iterCallbackRe.MatchString(sel.Sel.Name) && hasFuncLit(ce):
		body := newCst()
		c.callbackArgs(sc, ce, &body, out)
		out.add(body.scaled(1, 1, 0))
		out.add(c.expr(sc, sel.X))
		return false
	case sel != nil && isQueryCall(sel, len(ce.Args)):
		out.queries.addTerm(term{}, 1)
	}
	return true
}

// callbackArgs: func literal → cb (dikalikan pemanggil), argumen lain → out (sekali).
func (c *costCtx) callbackArgs(sc *fnScope, ce *ast.CallExpr, cb, out *cst) {
	for _, a := range ce.Args {
		if lit, ok := a.(*ast.FuncLit); ok {
			cb.add(c.closureBlock(sc, lit.Body.List))
			continue
		}
		out.add(c.expr(sc, a))
	}
}

// noteLoopCall mencatat bahwa fungsi sc.key memanggil callee di dalam loop atas data.
func (c *costCtx) noteLoopCall(sc *fnScope, callee string) {
	if sc.dataLoops == 0 || sc.key == "" || callee == sc.key {
		return
	}
	if c.inLoop[sc.key] == nil {
		c.inLoop[sc.key] = map[string]bool{}
	}
	c.inLoop[sc.key][callee] = true
}

func hasFuncLit(ce *ast.CallExpr) bool {
	for _, a := range ce.Args {
		if _, ok := a.(*ast.FuncLit); ok {
			return true
		}
	}
	return false
}

// isQueryCall: method akhir pengirim query pada rantai yang berakar di koneksi DB atau memuat penyusun query.
func isQueryCall(sel *ast.SelectorExpr, args int) bool {
	name := sel.Sel.Name
	direct := false // dipanggil langsung pada akar (bukan hasil rantai)
	if _, chained := sel.X.(*ast.CallExpr); !chained {
		direct = true
	}
	// sqlx: db.Select(&dest, query) / db.Get(&dest, query) — beda dari penyusun gorm Select("kolom").
	if (name == "Select" || name == "Get") && direct && args >= 2 {
		return dbRootRe.MatchString(chainRoot(sel.X))
	}
	if !queryTerminals[name] {
		return false
	}
	builder := false
	e := sel.X
	for {
		switch x := e.(type) {
		case *ast.CallExpr:
			if s, ok := x.Fun.(*ast.SelectorExpr); ok {
				if queryBuilders[s.Sel.Name] {
					builder = true
				}
				e = s.X
				continue
			}
		case *ast.ParenExpr:
			e = x.X
			continue
		}
		break
	}
	return builder || dbRootRe.MatchString(chainRoot(e))
}

// chainRoot: nama pengenal terakhir akar rantai (s.db → "db", h.repo.DB → "DB", tx → "tx").
func chainRoot(e ast.Expr) string {
	for {
		switch x := e.(type) {
		case *ast.CallExpr:
			e = x.Fun
			if s, ok := e.(*ast.SelectorExpr); ok {
				e = s.X
			}
			continue
		case *ast.ParenExpr:
			e = x.X
			continue
		case *ast.StarExpr:
			e = x.X
			continue
		case *ast.SelectorExpr:
			return x.Sel.Name
		case *ast.Ident:
			return x.Name
		}
		return ""
	}
}

// ---------- hasil untuk API / laporan ----------

// GrowthPoint = jumlah tick & query pada ukuran data n.
type GrowthPoint struct {
	N       int64   `json:"n"`
	Ticks   float64 `json:"ticks"`
	Queries float64 `json:"queries"`
}

// Cost = rumus biaya satu fungsi atau rute.
type Cost struct {
	Formula      string        `json:"formula"`       // T(n), mis. "3n² + 12n + 25"
	BigO         string        `json:"big_o"`         // O(n²)
	Class        string        `json:"class"`         // constant, log, linear, linearithmic, quadratic, cubic, polynomial
	Degree       int           `json:"degree"`        // pangkat n tertinggi
	Ticks1       int64         `json:"ticks_1"`       // T(1): jumlah tick untuk 1 data
	QueryFormula string        `json:"query_formula"` // Q(n), mis. "n + 1"
	QueryBigO    string        `json:"query_big_o"`
	Queries1     int64         `json:"queries_1"`
	NPlusOne     bool          `json:"n_plus_one"`          // query di dalam loop atas data
	Recursive    bool          `json:"recursive,omitempty"` // ada rekursi: rumus adalah batas bawah
	Unbounded    bool          `json:"unbounded,omitempty"` // ada loop tanpa syarat berhenti statis (for {})
	LoopDepth    int           `json:"loop_depth"`          // kedalaman loop atas data
	Growth       []GrowthPoint `json:"growth"`
	// Evidence: baris kode yang menjadi dasar rumus (loop atas data, query, pemanggilan berbiaya) di EvidenceFile.
	// Kosong untuk biaya konstan tanpa query: tidak ada baris yang menaikkan pertumbuhan.
	Evidence     []CostEvidence `json:"evidence,omitempty"`
	EvidenceFile string         `json:"evidence_file,omitempty"`
	rank         float64
	qDegree      int // pangkat n tertinggi pada Q(n)
}

var growthSizes = []int64{1, 10, 100, 1000, 10000}

func (c cst) export() Cost {
	out := Cost{Formula: c.ticks.String(), QueryFormula: c.queries.String(), Recursive: c.recursive, Unbounded: c.unbounded, LoopDepth: c.depth}
	out.BigO, out.Class = bigO(c.ticks)
	out.QueryBigO, _ = bigO(c.queries)
	lead, _ := c.ticks.lead()
	out.Degree = lead.N
	out.Ticks1 = int64(c.ticks.eval(1))
	out.Queries1 = int64(c.queries.eval(1))
	if qt, qc := c.queries.lead(); qc > 0 && qt.N >= 1 {
		out.NPlusOne = true
		out.qDegree = qt.N
	}
	for _, n := range growthSizes {
		out.Growth = append(out.Growth, GrowthPoint{N: n, Ticks: c.ticks.eval(float64(n)), Queries: c.queries.eval(float64(n))})
	}
	out.rank = c.ticks.eval(1e4) + c.queries.eval(1e4)*1e3
	return out
}

// FuncCost = biaya satu fungsi lokal.
type FuncCost struct {
	Module string `json:"module"`
	Lang   string `json:"lang"` // go, php
	Name   string `json:"name"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	End    int    `json:"end"`
	Routes int    `json:"routes"` // jumlah rute yang menjangkau fungsi ini (lewat handler atau middleware lokalnya)
	Cost
}

// RouteCost = biaya satu rute: handler beserta semua fungsi lokal yang dijangkaunya + middleware lokal di rantainya.
type RouteCost struct {
	Module      string `json:"module"`
	Lang        string `json:"lang"`                // go, php
	Framework   string `json:"framework,omitempty"` // fiber, gin, Laravel, CodeIgniter 4, PHP …
	Method      string `json:"method"`
	Path        string `json:"path"`
	Handler     string `json:"handler"`
	HandlerDecl string `json:"handler_decl,omitempty"`
	Auth        bool   `json:"auth"`
	Middlewares int    `json:"middlewares"` // middleware lokal yang ikut dihitung
	Cost
}

// CostReport = seluruh rute & fungsi, urut dari yang paling mahal saat data tumbuh.
type CostReport struct {
	Routes []RouteCost `json:"routes"`
	Funcs  []FuncCost  `json:"funcs"`
}

// needsEvidence: biaya yang tumbuh bersama data atau menyentuh database — harus bisa ditunjuk barisnya.
func (c Cost) needsEvidence() bool {
	return c.Degree > 0 || c.Queries1 > 0 || c.qDegree > 0 || c.Recursive || c.Unbounded
}

// computeCosts dijalankan sekali per modul setelah rute & call graph selesai disusun.
func (g *Graph) computeCosts(m *module) {
	c := &costCtx{m: m, memo: map[string]cst{}, busy: map[string]bool{}, inLoop: map[string]map[string]bool{}}
	src := newSrcCache(m.base)
	byPos := map[string]string{}
	for k, f := range m.funcs {
		byPos[f.Pos] = k
	}
	g.funcCosts = make(map[string]Cost, len(m.funcs))
	for k := range m.funcs {
		g.funcCosts[k] = c.funcCost(k).export()
	}
	g.loopCallees = c.inLoop
	// Bukti dikumpulkan setelah semua biaya fungsi termemo, agar biaya fungsi terpanggil sudah final.
	for k, f := range m.funcs {
		fc := g.funcCosts[k]
		if !fc.needsEvidence() || f.Decl == nil || f.Decl.Body == nil || f.file == nil {
			continue
		}
		fc.Evidence, fc.EvidenceFile = c.evidenceFor(f.file, f.Decl.Body.List, src), f.file.Rel
		g.funcCosts[k] = fc
	}
	g.routeCosts = make(map[*Route]RouteCost, len(g.Routes))
	for _, rt := range g.Routes {
		total := newCst()
		switch {
		case rt.handlerKey != "":
			total.add(c.funcCost(rt.handlerKey))
		case rt.body != nil && rt.bodyFile != nil:
			total.add(c.block(newFnScope(rt.bodyFile, "", rt.body.List), rt.body.List))
		default:
			total.tick(1)
		}
		mws := 0
		for _, mw := range rt.Chain {
			if k, ok := byPos[mw.Decl]; ok && mw.Decl != "" {
				total.add(c.funcCost(k))
				mws++
			}
		}
		cost := total.export()
		// Bukti rute = bukti handlernya (fungsi bernama atau badan handler inline).
		if cost.needsEvidence() {
			switch {
			case rt.handlerKey != "":
				hc := g.funcCosts[rt.handlerKey]
				cost.Evidence, cost.EvidenceFile = hc.Evidence, hc.EvidenceFile
			case rt.body != nil && rt.bodyFile != nil:
				cost.Evidence, cost.EvidenceFile = c.evidenceFor(rt.bodyFile, rt.body.List, src), rt.bodyFile.Rel
			}
		}
		g.routeCosts[rt] = RouteCost{Module: g.Module, Lang: "go", Framework: g.Framework, Method: rt.Method, Path: rt.Path, Handler: rt.Handler,
			HandlerDecl: rt.HandlerDecl, Auth: rt.Auth, Middlewares: mws, Cost: cost}
	}
}

// Costs menyusun laporan biaya semua modul hasil Analyze.
func Costs(graphs []*Graph) *CostReport {
	rep := &CostReport{Routes: []RouteCost{}, Funcs: []FuncCost{}}
	for _, g := range graphs {
		for _, rt := range g.Routes {
			if rc, ok := g.routeCosts[rt]; ok {
				rep.Routes = append(rep.Routes, rc)
			}
		}
		reach := map[string]int{}
		for _, keys := range g.routeReach() {
			for k := range keys {
				reach[k]++
			}
		}
		for k, fc := range g.funcCosts {
			ref, ok := g.ref(k)
			if !ok {
				continue
			}
			rep.Funcs = append(rep.Funcs, FuncCost{Module: g.Module, Lang: "go", Name: ref.Name, File: ref.File, Line: ref.Line, End: ref.End, Routes: reach[k], Cost: fc})
		}
	}
	finishCosts(rep)
	return rep
}

// finishCosts mengurutkan rute & fungsi dari yang paling mahal saat data tumbuh dan membuang rute ganda.
func finishCosts(rep *CostReport) {
	sort.SliceStable(rep.Routes, func(i, j int) bool {
		a, b := rep.Routes[i], rep.Routes[j]
		if a.rank != b.rank {
			return a.rank > b.rank
		}
		return a.Path+a.Method < b.Path+b.Method
	})
	// Rute yang terdaftar lebih dari sekali (dua entry main, alias router) cukup tampil sekali: yang termahal.
	seen := map[string]bool{}
	uniq := rep.Routes[:0]
	for _, r := range rep.Routes {
		k := r.Module + "|" + r.Method + "|" + r.Path
		if !seen[k] {
			seen[k] = true
			uniq = append(uniq, r)
		}
	}
	rep.Routes = uniq
	sort.SliceStable(rep.Funcs, func(i, j int) bool {
		a, b := rep.Funcs[i], rep.Funcs[j]
		if a.rank != b.rank {
			return a.rank > b.rank
		}
		return a.File+a.Name < b.File+b.Name
	})
}

// reachableFrom: fungsi lokal yang dijangkau dari key (termasuk key sendiri).
func (g *Graph) reachableFrom(key string) map[string]bool {
	seen := map[string]bool{key: true}
	stack := []string{key}
	for len(stack) > 0 {
		k := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, c := range g.callees[k] {
			if !seen[c] {
				seen[c] = true
				stack = append(stack, c)
			}
		}
	}
	return seen
}

// routeReach: untuk tiap rute, fungsi lokal yang dijangkau handler dan middleware lokal di rantainya.
func (g *Graph) routeReach() map[*Route]map[string]bool {
	byPos := make(map[string]string, len(g.funcs))
	for k, f := range g.funcs {
		byPos[f.Pos] = k
	}
	out := make(map[*Route]map[string]bool, len(g.Routes))
	for _, rt := range g.Routes {
		seen := map[string]bool{}
		if rt.handlerKey != "" {
			seen = g.reachableFrom(rt.handlerKey)
		}
		for _, mw := range rt.Chain {
			if k, ok := byPos[mw.Decl]; ok && mw.Decl != "" && !seen[k] {
				for r := range g.reachableFrom(k) {
					seen[r] = true
				}
			}
		}
		out[rt] = seen
	}
	return out
}

// Hotspot = fungsi tempat biaya sebuah rute BERASAL: pangkatnya lebih tinggi daripada semua fungsi yang
// dipanggilnya, jadi loop yang menaikkan pangkat ada di badan fungsi ini. Satu sumber dilaporkan sekali walau
// dijangkau banyak rute, supaya laporan menunjuk tempat perbaikan, bukan daftar rute yang terdampak.
type Hotspot struct {
	Kind       string   `json:"kind"` // nplus1 = query dikalikan loop data; growth = tick O(n²) ke atas
	Func       FuncCost `json:"func"`
	Via        string   `json:"via,omitempty"` // fungsi terpanggil yang membawa query/loop; kosong = di badan fungsi ini sendiri
	Routes     []string `json:"routes"`        // contoh rute terdampak (maks. 3)
	RouteCount int      `json:"route_count"`
	// AllRoutes: semua rute terdampak ("METHOD path"), untuk menautkan baris rute ke sumber biayanya.
	AllRoutes []string `json:"-"`
}

const hotspotRouteSamples = 3

// Hotspots mencari sumber N+1 query dan pertumbuhan O(n²) ke atas pada fungsi yang dijangkau rute, urut dari
// yang termahal. Rute yang tidak terjangkau dari main (kode mati) tidak dihitung.
func Hotspots(graphs []*Graph) []Hotspot {
	var out []Hotspot
	for _, g := range graphs {
		routesOf := map[string][]string{}
		for rt, keys := range g.routeReach() {
			if rt.Unreachable {
				continue
			}
			label := rt.Method + " " + rt.Path
			for k := range keys {
				routesOf[k] = append(routesOf[k], label)
			}
		}
		for k, fc := range g.funcCosts {
			routes := uniqueSorted(routesOf[k])
			if len(routes) == 0 {
				continue
			}
			ref, ok := g.ref(k)
			if !ok {
				continue
			}
			maxQ, maxT, viaQ, viaT := 0, 0, "", ""
			var bestQ, bestT float64
			for _, c := range g.callees[k] {
				if c == k {
					continue
				}
				cc, ok := g.funcCosts[c]
				if !ok {
					continue
				}
				if cc.qDegree > maxQ {
					maxQ = cc.qDegree
				}
				if cc.Degree > maxT {
					maxT = cc.Degree
				}
				name := shortName(g, c)
				// Hanya terpanggil di dalam loop data yang membawa query ke N+1; yang dipanggil sekali di luar loop tidak.
				if q := float64(cc.Queries1); q > bestQ && g.loopCallees[k][c] {
					bestQ, viaQ = q, name
				}
				if cc.Degree >= 1 && cc.rank > bestT {
					bestT, viaT = cc.rank, name
				}
			}
			fn := FuncCost{Module: g.Module, Lang: "go", Name: ref.Name, File: ref.File, Line: ref.Line, End: ref.End, Routes: len(routes), Cost: fc}
			sample := routes
			if len(sample) > hotspotRouteSamples {
				sample = sample[:hotspotRouteSamples]
			}
			if fc.qDegree >= 1 && fc.qDegree > maxQ {
				out = append(out, Hotspot{Kind: "nplus1", Func: fn, Via: viaQ, Routes: sample, RouteCount: len(routes), AllRoutes: routes})
			}
			if fc.Degree >= 2 && fc.Degree > maxT {
				out = append(out, Hotspot{Kind: "growth", Func: fn, Via: viaT, Routes: sample, RouteCount: len(routes), AllRoutes: routes})
			}
		}
	}
	sortHotspots(out)
	return out
}

// sortHotspots: termahal dulu; seri diurutkan jenis + berkas + nama agar hasilnya stabil.
func sortHotspots(out []Hotspot) {
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Func.rank != b.Func.rank {
			return a.Func.rank > b.Func.rank
		}
		return a.Kind+a.Func.File+a.Func.Name < b.Kind+b.Func.File+b.Func.Name
	})
}

func shortName(g *Graph, key string) string {
	if r, ok := g.ref(key); ok {
		return r.Name
	}
	return key
}

func uniqueSorted(in []string) []string {
	sort.Strings(in)
	out := in[:0]
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}
