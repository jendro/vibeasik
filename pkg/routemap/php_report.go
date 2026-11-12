package routemap

import (
	"path/filepath"
	"strings"
)

var phpFrameworkLabel = map[string]string{"laravel": "Laravel", "lumen": "Lumen", "codeigniter4": "CodeIgniter 4", "slim": "Slim", "php": "PHP"}

// PHPCosts menghitung biaya proses (tick & query) rute dan fungsi semua aplikasi PHP di bawah root, dengan
// model yang sama dengan Costs untuk Go. Aplikasi tanpa rute/halaman yang bisa ditelusuri tidak dilaporkan.
func PHPCosts(root string) (*CostReport, []PHPModule) {
	rep := &CostReport{Routes: []RouteCost{}, Funcs: []FuncCost{}}
	var mods []PHPModule
	for _, m := range findPHPModules(root) {
		routes := collectPHPRoutes(m)
		if len(routes) == 0 {
			continue
		}
		dir, _ := filepath.Rel(root, m.Dir)
		mods = append(mods, PHPModule{Name: m.Name, Dir: filepath.ToSlash(dir), Framework: m.Framework})
		label := phpFrameworkLabel[m.Framework]
		c := newPHPCostCtx(m)
		reach := map[string]int{}
		for i, r := range routes {
			key := "route#" + itoa(i)
			c.cur = []string{key}
			total := newCst()
			decl := ""
			switch {
			case r.fn != nil:
				c.edge(phpNodeKey(r.fn))
				total.add(c.funcCost(r.fn))
				decl = r.fn.file.Rel + ":" + itoa(r.fn.Line)
			case r.body != nil:
				total.add(c.block(&phpScope{file: r.bodyFile, cls: r.bodyCls, vars: map[string]string{}, closures: map[string]cst{}}, r.body))
			case r.entry != nil:
				c.edge("file:" + r.entry.Rel)
				total.add(c.fileCost(r.entry))
			default:
				total.tick(1)
			}
			total.queries.addTerm(term{}, int64(r.bindings))
			mws := 0
			for _, fn := range c.middlewareFuncs(r.mws) {
				c.edge(phpNodeKey(fn))
				total.add(c.funcCost(fn))
				mws++
			}
			c.cur = nil
			for k := range phpReach(c.edges, key) {
				reach[k]++
			}
			rep.Routes = append(rep.Routes, RouteCost{Module: m.Name, Lang: "php", Framework: label, Method: r.Method, Path: r.Path,
				Handler: r.Handler, HandlerDecl: decl, Auth: r.auth || m.authAlias(r.mws), Middlewares: mws, Cost: total.export()})
		}
		for _, fn := range m.all {
			rep.Funcs = append(rep.Funcs, FuncCost{Module: m.Name, Lang: "php", Name: fn.Key, File: fn.file.Rel, Line: fn.Line,
				End: fn.End, Routes: reach[phpNodeKey(fn)], Cost: c.funcCost(fn).export()})
		}
	}
	finishCosts(rep)
	return rep, mods
}

// middlewareFuncs: middleware lokal (alias Kernel / nama kelas) → method handle()/before() miliknya.
func (c *phpCostCtx) middlewareFuncs(names []string) []*phpFunc {
	var out []*phpFunc
	seen := map[*phpFunc]bool{}
	for _, n := range names {
		n = strings.TrimPrefix(n, "filter:")
		base := strings.SplitN(n, ":", 2)[0]
		cls := c.m.aliases[base]
		if cls == "" {
			cls = phpShort(base)
		}
		for _, k := range c.m.classes[strings.ToLower(cls)] {
			for _, mname := range []string{"handle", "before", "process", "__invoke"} {
				if fn := c.m.method(k, mname); fn != nil && !seen[fn] {
					seen[fn] = true
					out = append(out, fn)
					break
				}
			}
			break
		}
	}
	return out
}

// authAlias: alias middleware yang menunjuk kelas autentikasi ('admin' => AuthenticateAdmin::class).
func (m *phpModule) authAlias(names []string) bool {
	for _, n := range names {
		cls := strings.ToLower(m.aliases[strings.SplitN(strings.TrimPrefix(n, "filter:"), ":", 2)[0]])
		if strings.Contains(cls, "auth") && !strings.Contains(cls, "optional") {
			return true
		}
	}
	return false
}

// phpReach: semua simpul (fungsi/berkas) yang dijangkau dari key.
func phpReach(edges map[string]map[string]bool, key string) map[string]bool {
	seen := map[string]bool{}
	stack := []string{key}
	for len(stack) > 0 {
		k := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for to := range edges[k] {
			if !seen[to] {
				seen[to] = true
				stack = append(stack, to)
			}
		}
	}
	return seen
}

// MergeCosts menggabungkan laporan beberapa bahasa (Go + PHP) dan mengurutkan ulang dari yang termahal.
func MergeCosts(reps ...*CostReport) *CostReport {
	out := &CostReport{Routes: []RouteCost{}, Funcs: []FuncCost{}}
	for _, r := range reps {
		if r == nil {
			continue
		}
		out.Routes = append(out.Routes, r.Routes...)
		out.Funcs = append(out.Funcs, r.Funcs...)
	}
	finishCosts(out)
	return out
}
