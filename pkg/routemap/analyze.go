package routemap

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type cached struct {
	fingerprint string
	graphs      []*Graph
}

var cache sync.Map // root abs → cached

// Analyze menelusuri semua modul Go di bawah root proyek. Hasil di-cache sampai berkas .go berubah.
func Analyze(root string) []*Graph {
	root, _ = filepath.Abs(root)
	fp := fingerprint(root)
	if c, ok := cache.Load(root); ok && c.(cached).fingerprint == fp {
		return c.(cached).graphs
	}
	var out []*Graph
	for _, dir := range findModules(root) {
		if g := analyzeModule(dir, root); g != nil && (len(g.Routes) > 0 || len(g.Routers) > 0) {
			out = append(out, g)
		}
	}
	cache.Store(root, cached{fingerprint: fp, graphs: out})
	return out
}

// fingerprint = jumlah & mtime terbaru berkas .go (murah: hanya stat).
func fingerprint(root string) string {
	n := 0
	var latest int64
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".go") || d.Name() == "go.mod" {
			n++
			if fi, err := d.Info(); err == nil && fi.ModTime().UnixNano() > latest {
				latest = fi.ModTime().UnixNano()
			}
		}
		return nil
	})
	return strconv.Itoa(n) + ":" + strconv.FormatInt(latest, 10)
}

func analyzeModule(dir, base string) *Graph {
	m := loadModule(dir, base)
	if m == nil {
		return nil
	}
	rel, _ := filepath.Rel(base, dir)
	g := &Graph{Module: m.Path, Dir: filepath.ToSlash(rel), funcs: m.funcs}
	g.buildCallGraph(m)
	w := &walker{m: m, g: g, active: map[string]bool{}, visited: map[string]bool{}, routeMW: map[*Route][]Middleware{}, mounted: map[int]bool{}, fields: map[types.Object]*Router{}}

	var mains []*funcInfo
	for _, f := range m.funcs {
		if f.Name == "main" && f.Recv == "" && f.file.PkgName == "main" {
			mains = append(mains, f)
		}
	}
	sort.Slice(mains, func(i, j int) bool { return mains[i].Pos < mains[j].Pos })
	for _, f := range mains {
		g.Entries = append(g.Entries, f.Pos)
		w.visited[f.Key] = true
		w.active[f.Key] = true
		w.walkStmts(f.Decl.Body.List, newScope(f.file, []string{f.Pos + " main()"}))
		delete(w.active, f.Key)
	}
	// Akar tidak langsung: closure di variabel paket (cobra RunE, tabel handler DI) lalu fungsi yang membuat
	// router sendiri (NewServer) — main sering hanya memanggil cmd.Execute() milik library.
	for _, fi := range m.files {
		for _, d := range fi.AST.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			ast.Inspect(gd, func(n ast.Node) bool {
				lit, ok := n.(*ast.FuncLit)
				if !ok {
					return true
				}
				if constructsOrCallsLocal(m, fi, lit.Body) {
					w.walkStmts(lit.Body.List, newScope(fi, []string{m.pos(lit.Pos()) + " func literal (variabel paket, mis. cobra RunE)"}))
				}
				return false
			})
		}
	}
	var builders []*funcInfo
	for _, f := range m.funcs {
		if !w.visited[f.Key] && constructsRouter(f.file, f.Decl.Body) {
			builders = append(builders, f)
		}
	}
	sort.Slice(builders, func(i, j int) bool { return builders[i].Pos < builders[j].Pos })
	for _, f := range builders {
		if w.visited[f.Key] {
			continue
		}
		w.visited[f.Key] = true
		w.active[f.Key] = true
		w.walkStmts(f.Decl.Body.List, newScope(f.file, []string{f.Pos + " " + shortKey(m, f.Key) + "() — pembuat router (tidak terjangkau langsung dari main)"}))
		delete(w.active, f.Key)
	}
	// Fungsi setup ber-parameter router yang tidak terjangkau dari main (library, wiring lewat DI/reflection).
	var orphans []*funcInfo
	for _, f := range m.funcs {
		if w.visited[f.Key] || f.Decl.Type.Params == nil {
			continue
		}
		for _, p := range f.Decl.Type.Params.List {
			if t := exprStr(p.Type); routerParamRe.MatchString(t) && len(p.Names) > 0 {
				orphans = append(orphans, f)
				break
			}
		}
	}
	sort.Slice(orphans, func(i, j int) bool { return orphans[i].Pos < orphans[j].Pos })
	for _, f := range orphans {
		if w.visited[f.Key] {
			continue
		}
		w.visited[f.Key] = true
		sc := newScope(f.file, []string{f.Pos + " " + f.Name + "() — tidak terjangkau dari main"})
		for _, p := range f.Decl.Type.Params.List {
			t := exprStr(p.Type)
			if !routerParamRe.MatchString(t) {
				continue
			}
			fw := frameworkOf(f.file.Imports[strings.TrimPrefix(strings.SplitN(t, ".", 2)[0], "*")])
			for _, n := range p.Names {
				sc.vars[n.Name] = w.newRouter(nil, fw, "/", n.Name, p.Pos(), sc)
			}
		}
		before := len(g.Routes)
		w.active[f.Key] = true
		w.walkStmts(f.Decl.Body.List, sc)
		delete(w.active, f.Key)
		if len(g.Routes) > before {
			for _, rt := range g.Routes[before:] {
				rt.Unreachable = true
			}
			g.Warnings = append(g.Warnings, "Fungsi setup "+shortKey(m, f.Key)+" ("+f.Pos+") tidak terjangkau dari main: "+
				strconv.Itoa(len(g.Routes)-before)+" rutenya kemungkinan kode mati (tidak pernah dipasang) atau dipasang dinamis; middleware induknya tidak diketahui.")
		}
	}
	w.finalize()
	fws := map[string]bool{}
	for _, r := range g.Routers {
		if r.Framework != "" {
			fws[r.Framework] = true
		}
	}
	for fw := range fws {
		g.Framework += fw + ","
	}
	g.Framework = strings.TrimSuffix(g.Framework, ",")
	findDuplicates(g, m)
	return g
}

// constructsRouter: badan fungsi memanggil konstruktor router (fiber.New, gin.Default, chi.NewRouter, …).
func constructsRouter(fi *fileInfo, body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		if ce, ok := n.(*ast.CallExpr); ok {
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
				if fw := frameworkOf(importOf(fi, sel)); fw != "" && constructors[fw][sel.Sel.Name] {
					found = true
				}
			}
		}
		return true
	})
	return found
}

func constructsOrCallsLocal(m *module, fi *fileInfo, body *ast.BlockStmt) bool {
	return constructsRouter(fi, body) || len(directCallees(m, fi, body)) > 0
}

// FindRoutes memfilter rute: method (opsional), path (substring atau pola persis), function (nama fungsi,
// key, atau path berkas — rute yang handler-nya menjangkau fungsi itu, ditelusuri balik lewat call graph).
func FindRoutes(graphs []*Graph, method, pathQ, function string) []*Route {
	var out []*Route
	for _, g := range graphs {
		var reach map[string]bool
		if function != "" {
			reach = g.reaching(function)
		}
		for _, rt := range g.Routes {
			if method != "" && !strings.EqualFold(rt.Method, method) && rt.Method != "ANY" {
				continue
			}
			if pathQ != "" && !strings.Contains(rt.Path, pathQ) {
				continue
			}
			if function != "" && !reach[rt.handlerKey] && !routeMentions(rt, function) {
				continue
			}
			out = append(out, rt)
		}
	}
	return out
}

// reaching: semua fungsi yang (langsung/tidak langsung) memanggil fungsi target, termasuk target.
func (g *Graph) reaching(q string) map[string]bool {
	targets := map[string]bool{}
	for k, f := range g.funcs {
		short := f.file.PkgName + "." + f.Name
		if f.Recv != "" {
			short = f.file.PkgName + "." + f.Recv + "." + f.Name
		}
		file := strings.SplitN(f.Pos, ":", 2)[0]
		if k == q || f.Name == q || short == q || strings.HasSuffix(short, "."+q) || file == q || strings.HasSuffix(file, "/"+q) {
			targets[k] = true
		}
	}
	seen := map[string]bool{}
	frontier := make([]string, 0, len(targets))
	for k := range targets {
		frontier = append(frontier, k)
	}
	for len(frontier) > 0 {
		k := frontier[0]
		frontier = frontier[1:]
		if seen[k] {
			continue
		}
		seen[k] = true
		frontier = append(frontier, g.callers[k]...)
	}
	return seen
}

func routeMentions(rt *Route, q string) bool {
	if strings.Contains(rt.Handler, q) || strings.HasPrefix(rt.HandlerDecl, q+":") || strings.HasSuffix(strings.SplitN(rt.HandlerDecl, ":", 2)[0], "/"+q) {
		return true
	}
	for _, c := range rt.Calls {
		if strings.Contains(c, q) {
			return true
		}
	}
	return false
}

// RouteAt mencari rute yang didaftarkan pada berkas:baris (relatif root proyek, cocok akhiran path).
func RouteAt(graphs []*Graph, file string, line int) *Route {
	want := ":" + strconv.Itoa(line)
	for _, g := range graphs {
		for _, rt := range g.Routes {
			if strings.HasSuffix(rt.Pos, want) {
				f := strings.TrimSuffix(rt.Pos, want)
				if f == file || strings.HasSuffix(f, "/"+file) || strings.HasSuffix(file, "/"+f) {
					return rt
				}
			}
		}
	}
	return nil
}

// FileRoutes memetakan berkas → rute yang melewatinya (handler atau fungsi yang dipanggil handler).
func FileRoutes(graphs []*Graph) map[string][]string {
	out := map[string][]string{}
	for _, g := range graphs {
		for _, rt := range g.Routes {
			label := rt.Method + " " + rt.Path
			files := map[string]bool{}
			if f := strings.SplitN(rt.HandlerDecl, ":", 2)[0]; f != "" {
				files[f] = true
			}
			for _, c := range rt.Calls {
				if i := strings.LastIndex(c, "("); i >= 0 {
					files[strings.SplitN(strings.TrimSuffix(c[i+1:], ")"), ":", 2)[0]] = true
				}
			}
			for f := range files {
				out[f] = append(out[f], label)
			}
		}
	}
	return out
}
