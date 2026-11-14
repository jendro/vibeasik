package routemap

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CostEvidence = satu baris kode yang menjadi dasar rumus biaya sebuah fungsi/rute: loop atas data yang menaikkan
// pangkat n, query database, atau pemanggilan kode yang biayanya ikut tumbuh. Bukti dikumpulkan dengan penghitung
// yang sama dengan rumusnya (forBound/rangeBound, isQueryCall, biaya fungsi terpanggil), sehingga setiap suku
// T(n) dan Q(n) bisa ditunjuk barisnya.
type CostEvidence struct {
	Kind  string `json:"kind"`  // loop | query | call
	Line  int    `json:"line"`  // baris di berkas (lihat Cost.EvidenceFile)
	Code  string `json:"code"`  // isi baris itu (dipangkas)
	Depth int    `json:"depth"` // jumlah loop atas data yang melingkupi baris ini (untuk loop: termasuk dirinya)
	Note  string `json:"note"`  // kenapa baris ini menaikkan biaya
	// Callee: fungsi lokal terpanggil yang membawa biayanya (pemanggilan atau query "lewat `fungsi`"), agar
	// pembaca bisa melompat ke fungsi itu. Kosong untuk loop, query langsung, dan pemanggilan PHP.
	Callee *CostCallee `json:"callee,omitempty"`
}

// CostCallee = fungsi terpanggil yang ditunjuk sebuah bukti biaya.
type CostCallee struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
}

const (
	maxCostEvidence     = 12
	maxEvidenceCodeLen  = 200
	evidenceKindLoop    = "loop"
	evidenceKindQuery   = "query"
	evidenceKindCall    = "call"
	evidenceMaxCallName = 2
)

// srcCache membaca baris sumber sekali per berkas.
type srcCache struct {
	root  string
	lines map[string][]string
}

func newSrcCache(root string) *srcCache { return &srcCache{root: root, lines: map[string][]string{}} }

func (s *srcCache) line(rel string, n int) string {
	ls, ok := s.lines[rel]
	if !ok {
		if data, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(rel))); err == nil {
			ls = strings.Split(string(data), "\n")
		}
		s.lines[rel] = ls
	}
	if n < 1 || n > len(ls) {
		return ""
	}
	code := strings.TrimSpace(ls[n-1])
	if len(code) > maxEvidenceCodeLen {
		code = code[:maxEvidenceCodeLen] + "…"
	}
	return code
}

func powerLabel(d int) string {
	switch d {
	case 1:
		return "n"
	case 2:
		return "n²"
	case 3:
		return "n³"
	}
	return fmt.Sprintf("n^%d", d)
}

func loopNote(depth int, unbounded bool) string {
	switch {
	case unbounded:
		return "loop tanpa syarat berhenti: dihitung sebagai loop atas data"
	case depth <= 1:
		return "loop atas data: isinya dijalankan n kali"
	}
	return fmt.Sprintf("loop bersarang tingkat %d: isinya dijalankan %s kali", depth, powerLabel(depth))
}

func costDegree(c cst) int {
	lead, _ := c.ticks.lead()
	return lead.N
}

func hasQueries(c cst) bool {
	_, coef := c.queries.lead()
	return coef > 0
}

// costNote menjelaskan biaya sebuah pernyataan/pemanggilan; ok=false bila tidak menaikkan pertumbuhan.
func costNote(cost cst, depth int, names []string) (kind, note string, ok bool) {
	q, deg := hasQueries(cost), costDegree(cost)
	if !q && deg == 0 && !cost.recursive {
		return "", "", false
	}
	who := "kode"
	if len(names) > 0 {
		who = "`" + strings.Join(names, "`, `") + "`"
	}
	label, _ := bigO(cost.ticks)
	via := ""
	if len(names) > 0 {
		via = " lewat " + who
	}
	steps := ""
	if deg > 0 {
		steps = ", langkah " + label
	}
	switch {
	case q && depth > 0:
		kind = evidenceKindQuery
		note = fmt.Sprintf("query di dalam loop data%s: dijalankan %s kali (N+1), Q = %s per putaran%s", via, powerLabel(depth), cost.queries.String(), steps)
	case q:
		kind = evidenceKindQuery
		note = fmt.Sprintf("menjalankan query%s: Q = %s%s", via, cost.queries.String(), steps)
	case cost.recursive && deg == 0:
		kind = evidenceKindCall
		note = "pemanggilan rekursif: rumus adalah batas bawah"
	default:
		kind = evidenceKindCall
		note = fmt.Sprintf("memanggil %s berbiaya %s", who, label)
		if depth > 0 {
			note += fmt.Sprintf(", di dalam loop data → dikalikan %s", powerLabel(depth))
		}
	}
	return kind, note, true
}

// finishEvidence membuang baris ganda, membatasi jumlah (yang paling menentukan pertumbuhan didahulukan), lalu
// mengurutkan menurut baris.
func finishEvidence(in []CostEvidence) []CostEvidence {
	seen := map[string]bool{}
	out := in[:0:0]
	for _, e := range in {
		k := fmt.Sprintf("%s|%d", e.Kind, e.Line)
		if e.Line <= 0 || e.Code == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, e)
	}
	weight := func(e CostEvidence) int {
		w := e.Depth * 10
		switch e.Kind {
		case evidenceKindQuery:
			w += 5
		case evidenceKindLoop:
			w += 3
		default:
			w++
		}
		return w
	}
	if len(out) > maxCostEvidence {
		sort.SliceStable(out, func(i, j int) bool { return weight(out[i]) > weight(out[j]) })
		out = out[:maxCostEvidence]
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out
}

// ---------- Go ----------

type goEvidence struct {
	c   *costCtx
	sc  *fnScope
	src *srcCache
	out []CostEvidence
}

// evidenceFor mengumpulkan bukti biaya dari badan sebuah fungsi/handler Go.
func (c *costCtx) evidenceFor(fi *fileInfo, body []ast.Stmt, src *srcCache) []CostEvidence {
	if fi == nil || len(body) == 0 {
		return nil
	}
	w := &goEvidence{c: c, sc: newFnScope(fi, "", body), src: src}
	w.block(body, 0)
	return finishEvidence(w.out)
}

func (w *goEvidence) add(kind string, n ast.Node, depth int, note string) {
	line := w.c.m.fset.Position(n.Pos()).Line
	w.out = append(w.out, CostEvidence{Kind: kind, Line: line, Code: w.src.line(w.sc.fi.Rel, line), Depth: depth, Note: note})
}

func (w *goEvidence) block(list []ast.Stmt, depth int) {
	for _, s := range list {
		w.stmt(s, depth)
	}
}

func (w *goEvidence) stmt(s ast.Stmt, depth int) {
	switch x := s.(type) {
	case nil, *ast.EmptyStmt:
	case *ast.BlockStmt:
		w.block(x.List, depth)
	case *ast.LabeledStmt:
		w.stmt(x.Stmt, depth)
	case *ast.IfStmt:
		w.stmt(x.Init, depth)
		w.calls(x.Cond, depth)
		w.block(x.Body.List, depth)
		w.stmt(x.Else, depth)
	case *ast.ForStmt:
		w.stmt(x.Init, depth)
		_, data, unbounded := loopBound(x)
		inner := depth
		if data {
			inner++
			w.add(evidenceKindLoop, x, inner, loopNote(inner, unbounded))
		}
		w.calls(x.Cond, inner)
		w.block(x.Body.List, inner)
		w.stmt(x.Post, inner)
	case *ast.RangeStmt:
		w.calls(x.X, depth)
		inner := depth
		if _, data := rangeBoundIn(w.sc, x.X); data {
			inner++
			w.add(evidenceKindLoop, x, inner, loopNote(inner, false))
		}
		w.block(x.Body.List, inner)
	case *ast.SwitchStmt:
		w.stmt(x.Init, depth)
		w.calls(x.Tag, depth)
		w.clauses(x.Body, depth)
	case *ast.TypeSwitchStmt:
		w.stmt(x.Init, depth)
		w.clauses(x.Body, depth)
	case *ast.SelectStmt:
		w.clauses(x.Body, depth)
	case *ast.AssignStmt:
		for i, r := range x.Rhs {
			if lit, ok := r.(*ast.FuncLit); ok && i < len(x.Lhs) {
				if id, ok := x.Lhs[i].(*ast.Ident); ok {
					w.sc.closures[id.Name] = w.c.block(w.sc, lit.Body.List)
					w.block(lit.Body.List, depth)
					continue
				}
			}
			w.calls(r, depth)
		}
	case *ast.ReturnStmt:
		w.calls(x, 0) // `return` di dalam loop dijalankan paling banyak sekali
	default:
		w.calls(s, depth)
	}
}

func (w *goEvidence) clauses(body *ast.BlockStmt, depth int) {
	if body == nil {
		return
	}
	for _, cl := range body.List {
		switch x := cl.(type) {
		case *ast.CaseClause:
			w.block(x.Body, depth)
		case *ast.CommClause:
			w.stmt(x.Comm, depth)
			w.block(x.Body, depth)
		}
	}
}

// calls memeriksa setiap pemanggilan di dalam satu pernyataan/ekspresi dengan aturan yang sama dengan costCtx.call.
func (w *goEvidence) calls(n ast.Node, depth int) {
	if n == nil {
		return
	}
	if e, ok := n.(ast.Expr); ok && e == nil {
		return
	}
	ast.Inspect(n, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			w.block(x.Body.List, depth) // closure langsung (go func(){…}(), defer, argumen biasa)
			return false
		case *ast.CallExpr:
			return w.call(x, depth)
		}
		return true
	})
}

func (w *goEvidence) call(ce *ast.CallExpr, depth int) bool {
	c, sc := w.c, w.sc
	if id, ok := ce.Fun.(*ast.Ident); ok {
		if cl, ok := sc.closures[id.Name]; ok {
			if kind, note, ok := costNote(cl, depth, []string{id.Name}); ok {
				w.add(kind, ce, depth, note)
			}
			return true
		}
	}
	if fs := c.m.resolveFuncs(sc.fi, ce.Fun); len(fs) > 0 {
		best := newCst()
		var names []string
		var callee *CostCallee
		for _, f := range fs {
			if f == nil {
				continue
			}
			fc := c.funcCost(f.Key)
			best = heavier(best, fc)
			if (hasQueries(fc) || costDegree(fc) > 0) && len(names) < evidenceMaxCallName {
				names = append(names, f.Name)
				if callee == nil && f.Decl != nil && f.file != nil {
					callee = &CostCallee{Name: f.Name, File: f.file.Rel, Line: c.m.fset.Position(f.Decl.Pos()).Line}
				}
			}
		}
		if kind, note, ok := costNote(best, depth, names); ok {
			w.add(kind, ce, depth, note)
			w.out[len(w.out)-1].Callee = callee
		}
		return true
	}
	name := exprStr(ce.Fun)
	sel, _ := ce.Fun.(*ast.SelectorExpr)
	switch {
	case sortFuncs[name]:
		note := "pengurutan: n log n langkah"
		if depth > 0 {
			note += fmt.Sprintf(", di dalam loop data → dikalikan %s", powerLabel(depth))
		}
		w.add(evidenceKindCall, ce, depth, note)
		w.callbacks(ce, depth)
		return false
	case sel != nil && iterCallbackRe.MatchString(sel.Sel.Name) && hasFuncLit(ce):
		w.add(evidenceKindLoop, ce, depth+1, "callback dijalankan sekali per elemen: "+loopNote(depth+1, false))
		w.callbacks(ce, depth+1)
		w.calls(sel.X, depth)
		return false
	case sel != nil && isQueryCall(sel, len(ce.Args)):
		q := newCst()
		q.queries.addTerm(term{}, 1)
		if kind, note, ok := costNote(q, depth, nil); ok {
			w.add(kind, ce, depth, note)
		}
	}
	return true
}

func (w *goEvidence) callbacks(ce *ast.CallExpr, depth int) {
	for _, a := range ce.Args {
		if lit, ok := a.(*ast.FuncLit); ok {
			w.block(lit.Body.List, depth)
			continue
		}
		w.calls(a, depth)
	}
}

// ---------- PHP ----------

type phpEvidence struct {
	c   *phpCostCtx
	sc  *phpScope
	src *srcCache
	out []CostEvidence
}

// evidenceFor mengumpulkan bukti biaya dari badan fungsi/halaman PHP. Dipanggil setelah semua biaya dihitung
// (memo terisi), dengan lingkup baru agar tipe variabel dilacak ulang berurutan seperti saat menghitung.
func (c *phpCostCtx) evidenceFor(sc *phpScope, body []*phpStmt, src *srcCache) []CostEvidence {
	if sc == nil || sc.file == nil || len(body) == 0 {
		return nil
	}
	w := &phpEvidence{c: c, sc: sc, src: src}
	w.block(body, 0)
	return finishEvidence(w.out)
}

func (w *phpEvidence) add(kind string, line, depth int, note string) {
	w.out = append(w.out, CostEvidence{Kind: kind, Line: line, Code: w.src.line(w.sc.file.Rel, line), Depth: depth, Note: note})
}

func (w *phpEvidence) block(list []*phpStmt, depth int) {
	for _, s := range list {
		w.stmt(s, depth)
	}
}

func (w *phpEvidence) cost(cost cst, line, depth int) {
	if kind, note, ok := costNote(cost, depth, nil); ok {
		w.add(kind, line, depth, note)
	}
}

func (w *phpEvidence) stmt(s *phpStmt, depth int) {
	c, sc := w.c, w.sc
	switch s.Kind {
	case psBlock:
		w.block(s.Body, depth)
	case psIf:
		w.cost(c.expr(sc, s.Cond), s.Line, depth)
		w.block(s.Body, depth)
		w.block(s.Else, depth)
	case psSwitch:
		w.cost(c.expr(sc, s.Cond), s.Line, depth)
		for _, cs := range s.Cases {
			w.block(cs, depth)
		}
	case psTry:
		w.block(s.Body, depth)
		for _, cs := range s.Cases {
			w.block(cs, depth)
		}
		w.block(s.Else, depth)
	case psLoop:
		data, unbounded := phpLoopIsData(s)
		inner := depth
		if data {
			inner++
			w.add(evidenceKindLoop, s.Line, inner, loopNote(inner, unbounded))
		}
		w.block(s.Body, inner)
	case psExpr:
		if len(s.Expr) > 0 {
			w.cost(c.exprStmt(sc, s.Expr), s.Line, depth)
		}
	}
}

// phpLoopIsData: keputusan "loop atas data" yang sama dengan phpCostCtx.loop.
func phpLoopIsData(s *phpStmt) (data, unbounded bool) {
	switch s.Loop {
	case "foreach":
		subj := s.Cond
		for i, t := range s.Cond {
			if t.kw("as") {
				subj = s.Cond[:i]
				break
			}
		}
		_, fixed := phpFixedCount(subj)
		return !fixed, false
	case "for":
		parts := splitTop(s.Cond, ";")
		for len(parts) < 3 {
			parts = append(parts, nil)
		}
		_, data, unbounded = phpForBound(parts[0], parts[1])
		return data, unbounded
	}
	_, data, unbounded = phpForBound(nil, s.Cond)
	return data, unbounded
}
