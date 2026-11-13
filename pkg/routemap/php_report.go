package routemap

import (
	"path/filepath"
	"strings"
)

var phpFrameworkLabel = map[string]string{"laravel": "Laravel", "lumen": "Lumen", "codeigniter4": "CodeIgniter 4", "slim": "Slim", "php": "PHP"}

// PHPAnalysis = hasil satu kali analisis biaya semua aplikasi PHP di bawah root.
type PHPAnalysis struct {
	Report   *CostReport
	Modules  []PHPModule
	Hotspots []Hotspot // sumber N+1 & pertumbuhan O(n²) ke atas, seperti Hotspots untuk Go
}

// PHPCosts menghitung biaya proses (tick & query) rute dan fungsi semua aplikasi PHP di bawah root, dengan
// model yang sama dengan Costs untuk Go. Aplikasi tanpa rute/halaman yang bisa ditelusuri tidak dilaporkan.
func PHPCosts(root string) (*CostReport, []PHPModule) {
	a := AnalyzePHP(root)
	return a.Report, a.Modules
}

// PHPHotspots: sumber biaya PHP saja (untuk audit).
func PHPHotspots(root string) []Hotspot { return AnalyzePHP(root).Hotspots }

// AnalyzePHP menghitung laporan biaya dan sumber biayanya sekaligus.
func AnalyzePHP(root string) PHPAnalysis {
	rep := &CostReport{Routes: []RouteCost{}, Funcs: []FuncCost{}}
	var mods []PHPModule
	var hots []Hotspot
	for _, m := range findPHPModules(root) {
		routes := collectPHPRoutes(m)
		if len(routes) == 0 {
			continue
		}
		dir, _ := filepath.Rel(root, m.Dir)
		mods = append(mods, PHPModule{Name: m.Name, Dir: filepath.ToSlash(dir), Framework: m.Framework})
		label := phpFrameworkLabel[m.Framework]
		c := newPHPCostCtx(m)
		src := newSrcCache(m.root)
		type pendingRoute struct {
			idx int
			r   *phpRoute
		}
		var pending []pendingRoute
		reach := map[string]int{}
		routesOf := map[string][]string{}
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
				routesOf[k] = append(routesOf[k], r.Method+" "+r.Path)
			}
			rep.Routes = append(rep.Routes, RouteCost{Module: m.Name, Lang: "php", Framework: label, Method: r.Method, Path: r.Path,
				Handler: r.Handler, HandlerDecl: decl, Auth: r.auth || m.authAlias(r.mws), Middlewares: mws, Cost: total.export()})
			pending = append(pending, pendingRoute{len(rep.Routes) - 1, r})
		}
		// Bukti dikumpulkan setelah semua biaya termemo (c.cur kosong: tidak menambah sisi graf).
		for _, p := range pending {
			rc := &rep.Routes[p.idx]
			if !rc.Cost.needsEvidence() {
				continue
			}
			switch r := p.r; {
			case r.fn != nil:
				rc.Evidence, rc.EvidenceFile = c.evidenceFor(c.funcScope(r.fn), r.fn.Body, src), r.fn.file.Rel
			case r.body != nil && r.bodyFile != nil:
				rc.Evidence, rc.EvidenceFile = c.evidenceFor(&phpScope{file: r.bodyFile, cls: r.bodyCls, vars: map[string]string{}, closures: map[string]cst{}}, r.body, src), r.bodyFile.Rel
			case r.entry != nil:
				rc.Evidence, rc.EvidenceFile = c.evidenceFor(&phpScope{file: r.entry, vars: map[string]string{}, closures: map[string]cst{}}, r.entry.Stmts, src), r.entry.Rel
			}
		}
		for _, fn := range m.all {
			cost := c.funcCost(fn).export()
			if cost.needsEvidence() {
				cost.Evidence, cost.EvidenceFile = c.evidenceFor(c.funcScope(fn), fn.Body, src), fn.file.Rel
			}
			rep.Funcs = append(rep.Funcs, FuncCost{Module: m.Name, Lang: "php", Name: fn.Key, File: fn.file.Rel, Line: fn.Line,
				End: fn.End, Routes: reach[phpNodeKey(fn)], Cost: cost})
		}
		hots = append(hots, c.hotspots(routesOf)...)
	}
	finishCosts(rep)
	sortHotspots(hots)
	return PHPAnalysis{Report: rep, Modules: mods, Hotspots: hots}
}

// hotspots: fungsi/halaman yang pangkat Q(n) atau T(n)-nya lebih tinggi daripada semua yang dipanggilnya,
// yaitu tempat loop yang menaikkan biaya benar-benar ditulis.
func (c *phpCostCtx) hotspots(routesOf map[string][]string) []Hotspot {
	src := newSrcCache(c.m.root)
	type node struct {
		fc   FuncCost
		cost Cost
	}
	nodes := map[string]node{}
	for _, fn := range c.m.all {
		k := phpNodeKey(fn)
		cost := c.funcCost(fn).export()
		if cost.needsEvidence() {
			cost.Evidence, cost.EvidenceFile = c.evidenceFor(c.funcScope(fn), fn.Body, src), fn.file.Rel
		}
		nodes[k] = node{FuncCost{Module: c.m.Name, Lang: "php", Name: fn.Key, File: fn.file.Rel, Line: fn.Line, End: fn.End, Cost: cost}, cost}
	}
	for f, v := range c.fileMemo {
		cost := v.export()
		if cost.needsEvidence() {
			cost.Evidence, cost.EvidenceFile = c.evidenceFor(&phpScope{file: f, vars: map[string]string{}, closures: map[string]cst{}}, f.Stmts, src), f.Rel
		}
		nodes["file:"+f.Rel] = node{FuncCost{Module: c.m.Name, Lang: "php", Name: f.Rel, File: f.Rel, Line: 1, Cost: cost}, cost}
	}
	var out []Hotspot
	for k, n := range nodes {
		routes := uniqueSorted(append([]string(nil), routesOf[k]...))
		if len(routes) == 0 {
			continue
		}
		maxQ, maxT, viaQ, viaT := 0, 0, "", ""
		var bestQ, bestT float64
		for to := range c.edges[k] {
			cn, ok := nodes[to]
			if !ok || to == k {
				continue
			}
			maxQ, maxT = max(maxQ, cn.cost.qDegree), max(maxT, cn.cost.Degree)
			if q := float64(cn.cost.Queries1); q > bestQ {
				bestQ, viaQ = q, cn.fc.Name
			}
			if cn.cost.Degree >= 1 && cn.cost.rank > bestT {
				bestT, viaT = cn.cost.rank, cn.fc.Name
			}
		}
		fn := n.fc
		fn.Routes = len(routes)
		sample := routes
		if len(sample) > hotspotRouteSamples {
			sample = sample[:hotspotRouteSamples]
		}
		if n.cost.qDegree >= 1 && n.cost.qDegree > maxQ {
			out = append(out, Hotspot{Kind: "nplus1", Func: fn, Via: viaQ, Routes: sample, RouteCount: len(routes), AllRoutes: routes})
		}
		if n.cost.Degree >= 2 && n.cost.Degree > maxT {
			out = append(out, Hotspot{Kind: "growth", Func: fn, Via: viaT, Routes: sample, RouteCount: len(routes), AllRoutes: routes})
		}
	}
	return out
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
