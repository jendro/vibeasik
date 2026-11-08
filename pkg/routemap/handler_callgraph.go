package routemap

import (
	"go/ast"
	"sort"
	"strings"
)

// directCallees: fungsi lokal yang dipanggil langsung di dalam node.
func directCallees(m *module, fi *fileInfo, n ast.Node) []string {
	var out []string
	seen := map[string]bool{}
	ast.Inspect(n, func(n ast.Node) bool {
		if ce, ok := n.(*ast.CallExpr); ok {
			for _, f := range m.resolveFuncs(fi, ce.Fun) {
				if !seen[f.Key] {
					seen[f.Key] = true
					out = append(out, f.Key)
				}
			}
		}
		return true
	})
	return out
}

// buildCallGraph menyusun call graph lokal modul (dipakai untuk Calls rute & penelusuran balik fungsi → rute).
func (g *Graph) buildCallGraph(m *module) {
	g.callees = map[string][]string{}
	g.callers = map[string][]string{}
	for k, f := range m.funcs {
		cs := directCallees(m, f.file, f.Decl.Body)
		g.callees[k] = cs
		for _, c := range cs {
			if c != k {
				g.callers[c] = append(g.callers[c], k)
			}
		}
	}
}

func (w *walker) bfsCalls(start []string) []string {
	var out []string
	seen := map[string]bool{}
	frontier := start
	for d := 0; d < maxCallDepth && len(frontier) > 0 && len(out) < maxCalls; d++ {
		var next []string
		for _, k := range frontier {
			if seen[k] {
				continue
			}
			seen[k] = true
			if f := w.m.funcs[k]; f != nil {
				out = append(out, shortKey(w.m, k)+" ("+f.Pos+")")
				if len(out) >= maxCalls {
					break
				}
			}
			next = append(next, w.g.callees[k]...)
		}
		frontier = next
	}
	return out
}

// shortKey: bimtekapsiBE/handlers.AdminUpload → handlers.AdminUpload.
func shortKey(m *module, k string) string {
	if f := m.funcs[k]; f != nil {
		name := f.Name
		if f.Recv != "" {
			name = f.Recv + "." + f.Name
		}
		return f.file.PkgName + "." + name
	}
	return k
}

func routerRootOf(byID map[int]*Router, r *Router) *Router {
	for r.Parent >= 0 && byID[r.Parent] != nil {
		r = byID[r.Parent]
	}
	return r
}

func routerAncestors(byID map[int]*Router, r *Router) map[int]bool {
	out := map[int]bool{}
	for r != nil {
		out[r.ID] = true
		if r.Parent < 0 {
			break
		}
		r = byID[r.Parent]
	}
	return out
}

func isMiddlewareApplicable(r *Router, rt *Route, mw Middleware, o *Router, anc map[int]bool) bool {
	scopePath := mw.ScopePath
	if scopePath == "" {
		scopePath = o.Prefix
	}
	if r.Framework == "fiber" {
		if mw.Scope == "with" && !anc[o.ID] {
			return false
		}
		return hasPathPrefix(rt.Path, scopePath)
	}
	if !anc[o.ID] {
		return false
	}
	if mw.Scope == "prefix" && !hasPathPrefix(rt.Path, scopePath) {
		return false
	}
	return true
}

func (w *walker) collectRouteMiddlewares(byID map[int]*Router, r *Router, rt *Route) []Middleware {
	root := routerRootOf(byID, r)
	anc := routerAncestors(byID, r)
	var cands []Middleware
	for _, o := range w.g.Routers {
		if routerRootOf(byID, o) != root {
			continue
		}
		for _, mw := range o.Middlewares {
			if isMiddlewareApplicable(r, rt, mw, o, anc) {
				cands = append(cands, mw)
			}
		}
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].seq < cands[j].seq })
	return append(cands, w.routeMW[rt]...)
}

func applyRouteMiddlewares(r *Router, rt *Route, cands []Middleware) {
	for _, mw := range cands {
		switch {
		case mw.seq > rt.seq && r.Framework != "echo":
			mw.Reason = "didaftarkan setelah rute ini (hanya berjalan bila handler memanggil Next)"
			rt.Inactive = append(rt.Inactive, mw)
		case skipped(rt.Path, mw):
			mw.Reason = "dilewati untuk path ini (kondisi Next/skip di middleware)"
			rt.Inactive = append(rt.Inactive, mw)
		default:
			rt.Chain = append(rt.Chain, mw)
			if mw.Auth && !mw.Optional {
				rt.Auth = true
			}
		}
	}
}

// finalize menghitung rantai middleware efektif tiap rute sesuai semantik framework.
func (w *walker) finalize() {
	byID := map[int]*Router{}
	for _, r := range w.g.Routers {
		byID[r.ID] = r
	}
	for _, rt := range w.g.Routes {
		r := byID[rt.RouterID]
		if r == nil {
			continue
		}
		cands := w.collectRouteMiddlewares(byID, r, rt)
		applyRouteMiddlewares(r, rt, cands)
	}
	sort.SliceStable(w.g.Routes, func(i, j int) bool {
		if w.g.Routes[i].Path != w.g.Routes[j].Path {
			return w.g.Routes[i].Path < w.g.Routes[j].Path
		}
		return w.g.Routes[i].Method < w.g.Routes[j].Method
	})
}

func hasPathPrefix(p, prefix string) bool {
	prefix = strings.TrimSuffix(prefix, "/")
	return prefix == "" || p == prefix || strings.HasPrefix(p, prefix+"/")
}

func skipped(p string, mw Middleware) bool {
	for _, s := range mw.SkipPaths {
		if p == s {
			return true
		}
	}
	for _, s := range mw.SkipPrefixes {
		if strings.HasPrefix(p, s) {
			return true
		}
	}
	return false
}
