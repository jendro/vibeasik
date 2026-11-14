package routemap

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

// Bentuk eksekusi loop yang sebenarnya, supaya biaya (dan temuan N+1) mengikuti jumlah query nyata, bukan
// sekadar "ada query di dalam for":
//   - loop batch: maju per potongan (`start += chunk`) atau berulang sampai batch habis (keyset/kursor,
//     `for {}` / `for ctx.Err() == nil` yang berhenti saat batch kosong/kurang) — satu query mengurus banyak
//     baris, jadi dimodelkan sebagai satu putaran yang memproses seluruh data;
//   - `range` atas hasil query ber-Limit kecil: jumlah putaran tetap, bukan n;
//   - `return` di dalam loop: dijalankan paling banyak sekali (lihat costCtx.stmt).

// smallLimit: hasil query dengan Limit sampai angka ini dianggap koleksi berukuran tetap.
const smallLimit = 50

// batchExitRe: syarat berhenti loop batch — batch kosong/kurang dari ukurannya, tidak ada lagi, atau kursor habis.
var batchExitRe = regexp.MustCompile(`len\(|==\s*0\b|(?i)(more|done|empty|last|cursor|rowsaffected|exhausted)`)

// loopBound: jumlah putaran sebuah for. Loop batch → 1 putaran (bukan loop atas data, tidak tanpa batas).
// Syarat gabungan `cari() && i < 1000` dibatasi konstanta di salah satu sisinya.
func loopBound(x *ast.ForStmt) (k int64, data, unbounded bool) {
	k, data, unbounded = forBound(x)
	if be, ok := x.Cond.(*ast.BinaryExpr); ok && be.Op == token.LAND {
		for _, side := range []ast.Expr{be.X, be.Y} {
			if sk, sdata, _ := forBound(&ast.ForStmt{Init: x.Init, Cond: side}); !sdata {
				return sk, false, false
			}
		}
	}
	if data && (chunkStep(x) || batchExit(x.Body)) {
		return 1, false, false
	}
	return k, data, unbounded
}

// chunkStep: `for start := 0; start < len(xs); start += chunk` — maju per potongan, bukan per baris.
func chunkStep(x *ast.ForStmt) bool {
	as, ok := x.Post.(*ast.AssignStmt)
	if !ok || as.Tok != token.ADD_ASSIGN || len(as.Rhs) != 1 || x.Cond == nil {
		return false
	}
	if v, lit := intLit(as.Rhs[0]); lit && v <= 1 {
		return false
	}
	return true
}

// batchExit: badan loop memuat `if <batch habis> { break | return }` di tingkat teratasnya.
func batchExit(body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}
	for _, s := range body.List {
		ifs, ok := s.(*ast.IfStmt)
		if !ok || !exitsLoop(ifs.Body) {
			continue
		}
		if batchExitRe.MatchString(exprStr(ifs.Cond)) {
			return true
		}
	}
	return false
}

func exitsLoop(b *ast.BlockStmt) bool {
	for _, s := range b.List {
		switch x := s.(type) {
		case *ast.BranchStmt:
			if x.Tok == token.BREAK {
				return true
			}
		case *ast.ReturnStmt:
			return true
		}
	}
	return false
}

// rangeBoundIn: rangeBound ditambah hasil query ber-Limit kecil yang diisi di fungsi yang sama.
func rangeBoundIn(sc *fnScope, e ast.Expr) (k int64, data bool) {
	k, data = rangeBound(e)
	if !data || sc == nil {
		return k, data
	}
	if n, ok := sc.fixedSize(e); ok {
		return n, false
	}
	if id, ok := e.(*ast.Ident); ok {
		if lim, ok := sc.limits[id.Name]; ok && lim > 0 {
			return lim, false
		}
	}
	return k, data
}

// queryLimits: variabel yang diisi query ber-Limit literal kecil (`db.Limit(5).Find(&users)`) di badan fungsi.
// Variabel yang juga diisi query tanpa Limit tidak dicatat.
func queryLimits(body []ast.Stmt) map[string]int64 {
	out := map[string]int64{}
	seen := map[string]bool{}
	for _, s := range body {
		ast.Inspect(s, func(n ast.Node) bool {
			if _, ok := n.(*ast.FuncLit); ok {
				return false
			}
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := ce.Fun.(*ast.SelectorExpr)
			if !ok || !fillTerminals[sel.Sel.Name] || len(ce.Args) == 0 {
				return true
			}
			u, ok := ce.Args[0].(*ast.UnaryExpr)
			if !ok || u.Op != token.AND {
				return true
			}
			id, ok := u.X.(*ast.Ident)
			if !ok {
				return true
			}
			lim := chainLimit(sel.X)
			switch {
			case lim <= 0 || lim > smallLimit:
				out[id.Name] = -1
			case !seen[id.Name] || out[id.Name] > 0:
				out[id.Name] = max(out[id.Name], lim)
			}
			seen[id.Name] = true
			return true
		})
	}
	return out
}

// fillTerminals: metode pengisi hasil query ke variabel tujuan (argumen pertama &dest).
var fillTerminals = map[string]bool{"Find": true, "Scan": true}

// chainLimit: angka Limit(lit) pada rantai penyusun query (0 = tidak ada / bukan literal).
func chainLimit(e ast.Expr) int64 {
	for {
		ce, ok := e.(*ast.CallExpr)
		if !ok {
			return 0
		}
		sel, ok := ce.Fun.(*ast.SelectorExpr)
		if !ok {
			return 0
		}
		if sel.Sel.Name == "Limit" && len(ce.Args) == 1 {
			if v, ok := intLit(ce.Args[0]); ok {
				return v
			}
			return 0
		}
		e = sel.X
	}
}

// newFnScope menyiapkan cakupan biaya satu badan fungsi/handler.
func newFnScope(m *module, fi *fileInfo, key string, body []ast.Stmt) *fnScope {
	return &fnScope{m: m, fi: fi, key: key, closures: map[string]cst{}, limits: queryLimits(body), fixedVars: fixedLocals(body), exit: newCst()}
}

// textSplitFuncs: pemecah teks (strings.Split, regexp FindAll…) — banyak putarannya mengikuti panjang teks itu
// (daftar host di env, deklarasi CSS), bukan jumlah baris data. Dulu loop seperti ini dihitung sebagai loop atas
// data, sehingga fungsi string biasa (AssetURL) membuat setiap pemanggilnya di dalam loop tampak O(n²).
var textSplitFuncs = map[string]bool{
	"Split": true, "SplitN": true, "SplitAfter": true, "Fields": true, "FieldsFunc": true,
	"FindAllString": true, "FindAllStringSubmatch": true, "FindAllStringIndex": true, "FindAllStringSubmatchIndex": true,
}

func textSplit(e ast.Expr) bool {
	ce, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok || !textSplitFuncs[sel.Sel.Name] {
		return false
	}
	if strings.HasPrefix(sel.Sel.Name, "Find") {
		return true // metode *regexp.Regexp
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && (pkg.Name == "strings" || pkg.Name == "bytes")
}

// fixedCollection: range atas variabel tingkat paket yang diisi literal (`var AvailableTemplates = []T{…}`,
// `var meta = map[string]X{…}`) — ukurannya tetap sebanyak elemen literal itu, bukan n. Mengenali nama dari paket
// yang sama maupun paket lokal lain (`certificates.AvailableTemplates`).
func (m *module) fixedCollection(fi *fileInfo, e ast.Expr) (int64, bool) {
	if m == nil || fi == nil {
		return 0, false
	}
	var key string
	switch x := e.(type) {
	case *ast.Ident:
		if x.Obj != nil && !topLevelVar(fi, x.Obj) { // variabel lokal/parameter fungsi
			return 0, false
		}
		key = fi.Pkg + "." + x.Name
	case *ast.SelectorExpr:
		pkg, ok := x.X.(*ast.Ident)
		if !ok || pkg.Obj != nil {
			return 0, false
		}
		path, ok := fi.Imports[pkg.Name]
		if !ok {
			return 0, false
		}
		key = path + "." + x.Sel.Name
	default:
		return 0, false
	}
	m.fixedOnce.Do(m.indexFixedCollections)
	n, ok := m.fixed[key]
	return n, ok
}

func (m *module) indexFixedCollections() {
	m.fixed = map[string]int64{}
	reassigned := map[string]bool{}
	for _, f := range m.files {
		for _, d := range f.AST.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) != len(vs.Values) {
					continue
				}
				for i, name := range vs.Names {
					if cl, ok := vs.Values[i].(*ast.CompositeLit); ok && len(cl.Elts) > 0 {
						m.fixed[f.Pkg+"."+name.Name] = int64(len(cl.Elts))
					}
				}
			}
		}
		// Variabel paket yang diisi ulang (append/assign) di fungsi bukan koleksi tetap.
		ast.Inspect(f.AST, func(n ast.Node) bool {
			as, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for _, l := range as.Lhs {
				if id, ok := l.(*ast.Ident); ok && (id.Obj == nil || topLevelVar(f, id.Obj)) {
					reassigned[f.Pkg+"."+id.Name] = true
				}
			}
			return true
		})
	}
	for k := range reassigned {
		delete(m.fixed, k)
	}
}

// topLevelVar: objek hasil resolusi parser adalah var tingkat paket di berkas ini (bukan var lokal fungsi).
func topLevelVar(fi *fileInfo, obj *ast.Object) bool {
	vs, ok := obj.Decl.(*ast.ValueSpec)
	if !ok || obj.Kind != ast.Var {
		return false
	}
	for _, d := range fi.AST.Decls {
		if gd, ok := d.(*ast.GenDecl); ok {
			for _, spec := range gd.Specs {
				if spec == vs {
					return true
				}
			}
		}
	}
	return false
}

// fixedSize: ekspresi koleksi berukuran tetap — potongan teks, variabel lokal berisi literal/potongan teks, atau
// variabel paket berisi literal.
func (sc *fnScope) fixedSize(e ast.Expr) (int64, bool) {
	if textSplit(e) {
		return 1, true
	}
	if id, ok := e.(*ast.Ident); ok {
		if n, ok := sc.fixedVars[id.Name]; ok {
			return n, true
		}
	}
	return sc.m.fixedCollection(sc.fi, e)
}

// loopBoundIn: loopBound ditambah `for i := 0; i < len(vals); i++` atas koleksi berukuran tetap.
func loopBoundIn(sc *fnScope, x *ast.ForStmt) (k int64, data, unbounded bool) {
	k, data, unbounded = loopBound(x)
	if !data || sc == nil {
		return k, data, unbounded
	}
	be, ok := x.Cond.(*ast.BinaryExpr)
	if !ok || (be.Op != token.LSS && be.Op != token.LEQ) {
		return k, data, unbounded
	}
	ce, ok := be.Y.(*ast.CallExpr)
	if !ok || len(ce.Args) != 1 {
		return k, data, unbounded
	}
	if fn, ok := ce.Fun.(*ast.Ident); !ok || fn.Name != "len" {
		return k, data, unbounded
	}
	if n, ok := sc.fixedSize(ce.Args[0]); ok {
		return n, false, false
	}
	return k, data, unbounded
}

// fixedLocals: variabel lokal yang hanya diisi literal tak kosong (`vals := []int{1000, 900}`) atau potongan teks
// (`hosts := strings.Split(os.Getenv("X"), ",")`). Variabel yang juga diisi nilai lain (append, hasil query) tidak
// dicatat.
func fixedLocals(body []ast.Stmt) map[string]int64 {
	out := map[string]int64{}
	other := map[string]bool{}
	note := func(name string, v ast.Expr) {
		switch {
		case textSplit(v):
			out[name] = max(out[name], 1)
		default:
			if cl, ok := v.(*ast.CompositeLit); ok && len(cl.Elts) > 0 {
				out[name] = max(out[name], int64(len(cl.Elts)))
				return
			}
			other[name] = true
		}
	}
	for _, s := range body {
		ast.Inspect(s, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncLit:
				return false
			case *ast.AssignStmt:
				for i, l := range x.Lhs {
					id, ok := l.(*ast.Ident)
					if !ok {
						continue
					}
					if len(x.Rhs) == len(x.Lhs) && (x.Tok == token.DEFINE || x.Tok == token.ASSIGN) {
						note(id.Name, x.Rhs[i])
					} else {
						other[id.Name] = true
					}
				}
			case *ast.ValueSpec:
				for i, name := range x.Names {
					if i < len(x.Values) {
						note(name.Name, x.Values[i])
					}
				}
			case *ast.CallExpr: // append(&xs, …) / Find(&xs): diisi dari luar literal
				for _, a := range x.Args {
					if u, ok := a.(*ast.UnaryExpr); ok && u.Op == token.AND {
						if id, ok := u.X.(*ast.Ident); ok {
							other[id.Name] = true
						}
					}
				}
			}
			return true
		})
	}
	for k := range other {
		delete(out, k)
	}
	return out
}
