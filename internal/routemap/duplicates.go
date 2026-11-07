package routemap

import (
	"go/ast"
	"go/token"
	"regexp"
	"sort"
	"strings"
)

// Duplicate = pemeriksaan yang sama dikerjakan dua kali di jalur satu rute (patokan ARCH-004): middleware yang
// terpasang ulang di rantai efektif, atau kondisi validasi yang sama di handler dan fungsi yang dipanggilnya.
type Duplicate struct {
	Kind   string   `json:"kind"`   // middleware, validation
	Name   string   `json:"name"`   // nama middleware / kondisi validasi ternormalisasi
	First  string   `json:"first"`  // pemeriksaan pertama di jalur request
	Pos    string   `json:"pos"`    // salinan berikutnya (kandidat dihapus)
	Detail string   `json:"detail"` // scope/fungsi kedua lokasi
	Routes []string `json:"routes"` // contoh rute terdampak (maks. maxDupRoutes)
	Count  int      `json:"count"`  // jumlah rute terdampak
}

const maxDupRoutes = 3

// findDuplicates mengisi g.Duplicates: satu entri per pasangan lokasi, dengan contoh rute yang terdampak.
func findDuplicates(g *Graph, m *module) {
	byKey := map[string]*Duplicate{}
	add := func(d Duplicate, route string) {
		key := d.Kind + "|" + d.Name + "|" + d.First + "|" + d.Pos
		cur := byKey[key]
		if cur == nil {
			cur = &d
			byKey[key] = cur
		}
		cur.Count++
		if len(cur.Routes) < maxDupRoutes {
			cur.Routes = append(cur.Routes, route)
		}
	}
	guards := map[string][]guard{} // cache per fungsi
	byPos := make(map[string]string, len(g.funcs))
	for k, f := range g.funcs {
		byPos[f.Pos] = k
	}
	for _, rt := range g.Routes {
		if rt.Unreachable {
			continue
		}
		label := rt.Method + " " + rt.Path
		for i, j := range repeated(rt) {
			a, b := rt.Chain[j], rt.Chain[i]
			add(Duplicate{Kind: "middleware", Name: b.Name, First: a.Pos, Pos: b.Pos, Detail: a.Scope + " → " + b.Scope}, label)
		}
		for _, d := range validationDups(g, m, rt, byPos, guards) {
			add(d, label)
		}
	}
	g.Duplicates = make([]Duplicate, 0, len(byKey))
	for _, d := range byKey {
		g.Duplicates = append(g.Duplicates, *d)
	}
	sort.Slice(g.Duplicates, func(i, j int) bool {
		a, b := g.Duplicates[i], g.Duplicates[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Pos < b.Pos
	})
}

// guard = if yang memeriksa input lalu langsung return (pola validasi).
type guard struct {
	norm, pos, fn string
}

// validationDups: kondisi validasi yang sama muncul di dua fungsi berbeda pada jalur handler rute
// (handler → service → …). Pemeriksaan di fungsi pertama sudah menolak input yang sama.
func validationDups(g *Graph, m *module, rt *Route, byPos map[string]string, cache map[string][]guard) []Duplicate {
	if rt.body == nil {
		return nil
	}
	chain := [][]guard{guardsOf(m, rt.bodyFile, rt.body, shorten(rt.Handler, 60))}
	for _, c := range rt.Calls {
		// entri rt.Calls = "handlers.X (berkas.go:12)": posisi deklarasi → key fungsi
		i := strings.LastIndex(c, " (")
		if i < 0 {
			continue
		}
		key := byPos[strings.TrimSuffix(c[i+2:], ")")]
		if key == "" || key == rt.handlerKey {
			continue
		}
		gs, ok := cache[key]
		if !ok {
			f := g.funcs[key]
			gs = guardsOf(m, f.file, f.Decl.Body, shortKey(m, key))
			cache[key] = gs
		}
		chain = append(chain, gs)
	}
	seen := map[string]guard{}
	var out []Duplicate
	for _, gs := range chain {
		local := map[string]bool{}
		for _, gd := range gs {
			if first, ok := seen[gd.norm]; ok && first.fn != gd.fn && !local[gd.norm] {
				out = append(out, Duplicate{Kind: "validation", Name: gd.norm, First: first.pos, Pos: gd.pos, Detail: first.fn + " → " + gd.fn})
				local[gd.norm] = true
				continue
			}
			if _, ok := seen[gd.norm]; !ok {
				seen[gd.norm] = gd
			}
		}
	}
	return out
}

var (
	selectorRoot = regexp.MustCompile(`\b(?:[A-Za-z_]\w*\.)+`)
	validCallRe  = regexp.MustCompile(`(?i)valid|empty|blank`)
	allowedCalls = map[string]bool{"len": true, "TrimSpace": true, "RuneCountInString": true, "IsZero": true}
)

// guardsOf mengumpulkan if-validasi di badan fungsi (closure di dalamnya bukan jalur langsung, dilewati).
func guardsOf(m *module, fi *fileInfo, body *ast.BlockStmt, fn string) []guard {
	if body == nil {
		return nil
	}
	var out []guard
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		is, ok := n.(*ast.IfStmt)
		if !ok || !returnsEarly(is.Body) {
			return true
		}
		if norm := guardKey(fi, is); norm != "" {
			out = append(out, guard{norm: norm, pos: m.pos(is.Pos()), fn: fn})
		}
		return true
	})
	return out
}

func returnsEarly(b *ast.BlockStmt) bool {
	for _, s := range b.List {
		if _, ok := s.(*ast.ReturnStmt); ok {
			return true
		}
	}
	return false
}

// guardKey menormalkan kondisi validasi agar nama variabel tiap lapisan (req.Email vs in.Email) tidak
// membedakan: akar selector dibuang ("Email == \"\""). Hanya kondisi yang membaca field input dibandingkan
// dengan literal, atau pemanggilan validator. Pemeriksaan error (err != nil, .Error) dan cek nil defensif di
// awal fungsi publik (db == nil || p == nil) bukan validasi input, jadi dilewati.
func guardKey(fi *fileInfo, is *ast.IfStmt) string {
	if as, ok := is.Init.(*ast.AssignStmt); ok && len(as.Rhs) == 1 {
		if ce, ok := as.Rhs[0].(*ast.CallExpr); ok {
			if name := exprStr(ce.Fun); validCallRe.MatchString(name) {
				return "validator " + selectorRoot.ReplaceAllString(name, "") + "(" + selectorRoot.ReplaceAllString(exprList(ce.Args), "") + ")"
			}
		}
	}
	field, compare, ok := false, false, true
	ast.Inspect(is.Cond, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			if x.Name == "err" || x.Name == "nil" {
				ok = false
			}
		case *ast.SelectorExpr:
			if x.Sel.Name == "Error" || x.Sel.Name == "Err" {
				ok = false
			}
			if id, isID := x.X.(*ast.Ident); isID && fi.Imports[id.Name] == "" {
				field = true
			}
		case *ast.CallExpr:
			name := exprStr(x.Fun)
			if i := strings.LastIndex(name, "."); i >= 0 {
				name = name[i+1:]
			}
			if !allowedCalls[name] && !validCallRe.MatchString(name) {
				ok = false
			}
			compare = compare || validCallRe.MatchString(name)
		case *ast.BinaryExpr:
			switch x.Op {
			case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ:
				compare = compare || isLiteral(x.X) || isLiteral(x.Y)
			}
		case *ast.UnaryExpr:
			compare = compare || x.Op == token.NOT
		}
		return ok
	})
	if !ok || !field || !compare {
		return ""
	}
	return selectorRoot.ReplaceAllString(shorten(exprStr(is.Cond), 120), "")
}

func isLiteral(e ast.Expr) bool {
	_, ok := e.(*ast.BasicLit)
	return ok
}

func exprList(es []ast.Expr) string {
	parts := make([]string, len(es))
	for i, e := range es {
		parts[i] = exprStr(e)
	}
	return strings.Join(parts, ", ")
}
