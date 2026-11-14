package routemap

import (
	"go/ast"
	"go/token"
	"regexp"
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
func loopBound(x *ast.ForStmt) (k int64, data, unbounded bool) {
	k, data, unbounded = forBound(x)
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
func newFnScope(fi *fileInfo, key string, body []ast.Stmt) *fnScope {
	return &fnScope{fi: fi, key: key, closures: map[string]cst{}, limits: queryLimits(body), exit: newCst()}
}
