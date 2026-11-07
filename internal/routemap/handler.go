package routemap

import (
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

const (
	maxHelperDepth = 3
	maxCallDepth   = 4
	maxCalls       = 30
)

// Konstanta status net/http & fiber yang umum (fiber.StatusX dan http.StatusX bernama sama).
var statusConsts = map[string]int{
	"StatusContinue": 100, "StatusSwitchingProtocols": 101,
	"StatusOK": 200, "StatusCreated": 201, "StatusAccepted": 202, "StatusNonAuthoritativeInfo": 203,
	"StatusNoContent": 204, "StatusResetContent": 205, "StatusPartialContent": 206,
	"StatusMultipleChoices": 300, "StatusMovedPermanently": 301, "StatusFound": 302, "StatusSeeOther": 303,
	"StatusNotModified": 304, "StatusTemporaryRedirect": 307, "StatusPermanentRedirect": 308,
	"StatusBadRequest": 400, "StatusUnauthorized": 401, "StatusPaymentRequired": 402, "StatusForbidden": 403,
	"StatusNotFound": 404, "StatusMethodNotAllowed": 405, "StatusNotAcceptable": 406, "StatusRequestTimeout": 408,
	"StatusConflict": 409, "StatusGone": 410, "StatusLengthRequired": 411, "StatusPreconditionFailed": 412,
	"StatusRequestEntityTooLarge": 413, "StatusUnsupportedMediaType": 415, "StatusUnprocessableEntity": 422,
	"StatusLocked": 423, "StatusTooEarly": 425, "StatusPreconditionRequired": 428, "StatusTooManyRequests": 429,
	"StatusInternalServerError": 500, "StatusNotImplemented": 501, "StatusBadGateway": 502,
	"StatusServiceUnavailable": 503, "StatusGatewayTimeout": 504,
}

// method ctx yang menulis response; nilai = status bawaan bila tidak disebut.
var writeMethods = map[string]struct {
	kind   string
	status int
}{
	"JSON": {"json", 200}, "JSONP": {"json", 200}, "IndentedJSON": {"json", 200}, "PureJSON": {"json", 200},
	"SecureJSON": {"json", 200}, "AsciiJSON": {"json", 200}, "XML": {"xml", 200}, "YAML": {"yaml", 200},
	"SendString": {"string", 200}, "String": {"string", 200}, "Send": {"body", 200}, "Blob": {"body", 200},
	"Data": {"body", 200}, "HTML": {"html", 200}, "Render": {"render", 200}, "SendFile": {"file", 200},
	"File": {"file", 200}, "Download": {"file", 200}, "Attachment": {"file", 200}, "Stream": {"stream", 200},
	"SendStream": {"stream", 200}, "SSEvent": {"stream", 200}, "Format": {"body", 200},
	"SendStatus": {"status", 0}, "AbortWithStatus": {"status", 0}, "NoContent": {"status", 0},
	"WriteHeader": {"status", 0}, "AbortWithStatusJSON": {"json", 0}, "Redirect": {"redirect", 302},
}

// fnBody = fungsi yang dianalisis (deklarasi atau func literal).
type fnBody struct {
	body *ast.BlockStmt
	typ  *ast.FuncType
	fi   *fileInfo
}

// binding = argumen pemanggil untuk parameter helper (agar status yang dikirim lewat parameter terbaca).
type binding struct {
	expr ast.Expr
	fi   *fileInfo
	env  map[string]binding
}

// describeHandler mengisi Handler/HandlerDecl/Responses/Calls rute dari ekspresi handler.
func (w *walker) describeHandler(rt *Route, e ast.Expr, sc *scope) {
	rt.Handler = shorten(exprStr(e), 80)
	fb, key, decl := w.resolveHandler(e, sc.fi, sc, 0)
	rt.HandlerDecl, rt.handlerKey = decl, key
	if fb.body == nil {
		return
	}
	rt.body, rt.bodyFile = fb.body, fb.fi
	rt.Responses = w.responses(fb, nil, "", 0)
	if key == "" { // func literal: telusuri pemanggilan langsung di badan literal
		rt.Calls = w.bfsCalls(directCallees(w.m, fb.fi, fb.body))
	} else {
		rt.Calls = w.bfsCalls(w.g.callees[key])
	}
}

func (w *walker) resolveHandler(e ast.Expr, fi *fileInfo, sc *scope, depth int) (fnBody, string, string) {
	if depth > 4 || e == nil {
		return fnBody{}, "", ""
	}
	switch x := e.(type) {
	case *ast.FuncLit:
		return fnBody{x.Body, x.Type, fi}, "", w.m.pos(x.Pos())
	case *ast.Ident:
		if sc != nil {
			if v, ok := sc.vals[x.Name]; ok {
				return w.resolveHandler(v.expr, v.fi, v.sc, depth+1)
			}
		}
	case *ast.CallExpr:
		// factory: handlers.Make(db) → func literal yang dikembalikan; adaptor http.HandlerFunc(h) → h
		if f := w.m.resolveFunc(fi, x.Fun); f != nil {
			if lit := returnedFuncLit(f.Decl.Body); lit != nil {
				return fnBody{lit.Body, lit.Type, f.file}, "", w.m.pos(lit.Pos())
			}
			return fnBody{}, f.Key, f.Pos
		}
		if len(x.Args) == 1 {
			return w.resolveHandler(x.Args[0], fi, sc, depth+1)
		}
		return fnBody{}, "", ""
	}
	if f := w.m.resolveFunc(fi, e); f != nil {
		return fnBody{f.Decl.Body, f.Decl.Type, f.file}, f.Key, f.Pos
	}
	return fnBody{}, "", ""
}

func returnedFuncLit(b *ast.BlockStmt) *ast.FuncLit {
	var lit *ast.FuncLit
	ast.Inspect(b, func(n ast.Node) bool {
		if lit != nil {
			return false
		}
		if r, ok := n.(*ast.ReturnStmt); ok && len(r.Results) == 1 {
			if l, ok := r.Results[0].(*ast.FuncLit); ok {
				lit = l
			}
		}
		return true
	})
	return lit
}

// ctxParams: nama parameter yang bertipe context request/response writer.
func ctxParams(ft *ast.FuncType) map[string]bool {
	out := map[string]bool{}
	if ft == nil || ft.Params == nil {
		return out
	}
	for _, f := range ft.Params.List {
		t := exprStr(f.Type)
		if strings.Contains(t, "Ctx") || strings.Contains(t, "Context") || strings.Contains(t, "ResponseWriter") {
			for _, n := range f.Names {
				out[n.Name] = true
			}
		}
	}
	return out
}

func rootIdent(e ast.Expr) string {
	for {
		switch x := e.(type) {
		case *ast.Ident:
			return x.Name
		case *ast.SelectorExpr:
			e = x.X
		case *ast.CallExpr:
			e = x.Fun
		case *ast.ParenExpr:
			e = x.X
		case *ast.StarExpr:
			e = x.X
		default:
			return ""
		}
	}
}

// responses mengumpulkan cara fungsi menulis response, mengikuti helper lokal yang menerima ctx.
func (w *walker) responses(fb fnBody, env map[string]binding, via string, depth int) []Response {
	ctx := ctxParams(fb.typ)
	var out []Response
	seen := map[string]bool{}
	add := func(r Response) {
		k := strconv.Itoa(r.Status) + r.Kind + r.Via + r.Body
		if !seen[k] {
			seen[k] = true
			out = append(out, r)
		}
	}
	ast.Inspect(fb.body, func(n ast.Node) bool {
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, isSel := ce.Fun.(*ast.SelectorExpr)
		if isSel {
			name := sel.Sel.Name
			ip := importOf(fb.fi, sel)
			switch {
			case (name == "NewError" || name == "NewHTTPError") && frameworkOf(ip) != "":
				add(Response{Status: w.status(argAt(ce.Args, 0), fb.fi, env), Kind: "error", Body: shorten(exprStr(argAt(ce.Args, 1)), 60), Via: via, Pos: w.m.pos(ce.Pos())})
				return true
			case name == "Error" && ip == "net/http":
				add(Response{Status: w.status(argAt(ce.Args, 2), fb.fi, env), Kind: "error", Body: shorten(exprStr(argAt(ce.Args, 1)), 60), Via: via, Pos: w.m.pos(ce.Pos())})
				return true
			}
			if wm, ok := writeMethods[name]; ok && ctx[rootIdent(sel.X)] {
				r := Response{Status: wm.status, Kind: wm.kind, Via: via, Pos: w.m.pos(ce.Pos())}
				if inner, ok := sel.X.(*ast.CallExpr); ok { // c.Status(400).JSON(...)
					if is, ok := inner.Fun.(*ast.SelectorExpr); ok && is.Sel.Name == "Status" {
						r.Status = w.status(argAt(inner.Args, 0), fb.fi, env)
					}
				}
				switch {
				case wm.status == 0: // status di argumen pertama
					r.Status = w.status(argAt(ce.Args, 0), fb.fi, env)
				case name == "Redirect":
					if st := w.status(argAt(ce.Args, 0), fb.fi, env); st >= 300 { // gin Redirect(code, loc)
						r.Status = st
					} else if st := w.status(argAt(ce.Args, 1), fb.fi, env); st >= 300 { // fiber Redirect(loc, code)
						r.Status = st
					}
				case len(ce.Args) >= 2 && (wm.kind == "json" || wm.kind == "string" || wm.kind == "xml" || wm.kind == "html"):
					if st := w.status(ce.Args[0], fb.fi, env); st > 0 { // gin/echo c.JSON(code, body)
						r.Status = st
					}
				}
				if len(ce.Args) > 0 && r.Kind != "status" && r.Kind != "redirect" {
					r.Body = describeBody(ce.Args[len(ce.Args)-1])
				}
				add(r)
				return true
			}
			if name == "Next" && ctx[rootIdent(sel.X)] {
				add(Response{Kind: "next", Via: via, Pos: w.m.pos(ce.Pos())})
				return true
			}
		}
		// helper lokal yang menerima ctx: fail(c, 400, "...")
		if depth < maxHelperDepth {
			passesCtx := false
			for _, a := range ce.Args {
				if ctx[rootIdent(a)] {
					passesCtx = true
					break
				}
			}
			if passesCtx {
				if f := w.m.resolveFunc(fb.fi, ce.Fun); f != nil {
					ne := map[string]binding{}
					for i, p := range paramNames(f.Decl.Type) {
						if i < len(ce.Args) {
							ne[p] = binding{expr: ce.Args[i], fi: fb.fi, env: env}
						}
					}
					v := f.Name
					if via != "" {
						v = via + "→" + f.Name
					}
					for _, r := range w.responses(fnBody{f.Decl.Body, f.Decl.Type, f.file}, ne, v, depth+1) {
						add(r)
					}
				}
			}
		}
		return true
	})
	sort.SliceStable(out, func(i, j int) bool { return out[i].Status < out[j].Status })
	return out
}

// status mengevaluasi ekspresi kode status: literal, fiber/http.StatusX, konstanta lokal, atau parameter helper.
func (w *walker) status(e ast.Expr, fi *fileInfo, env map[string]binding) int {
	switch x := e.(type) {
	case *ast.BasicLit:
		if x.Kind == token.INT {
			n, _ := strconv.Atoi(x.Value)
			return n
		}
	case *ast.SelectorExpr:
		return statusConsts[x.Sel.Name]
	case *ast.Ident:
		if b, ok := env[x.Name]; ok {
			return w.status(b.expr, b.fi, b.env)
		}
		return statusConsts[x.Name]
	case *ast.ParenExpr:
		return w.status(x.X, fi, env)
	}
	return 0
}

func describeBody(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.CompositeLit:
		var keys []string
		for _, el := range x.Elts {
			if kv, ok := el.(*ast.KeyValueExpr); ok {
				k := exprStr(kv.Key)
				if s, ok := strLit(kv.Key); ok {
					k = s
				}
				keys = append(keys, k)
			}
		}
		t := exprStr(x.Type)
		if len(keys) > 0 {
			return shorten(t+"{"+strings.Join(keys, ", ")+"}", 80)
		}
		return shorten(t+"{…}", 80)
	case *ast.UnaryExpr:
		return describeBody(x.X)
	case *ast.CallExpr:
		return shorten(exprStr(x.Fun)+"(…)", 60)
	}
	return shorten(exprStr(e), 60)
}

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

// finalize menghitung rantai middleware efektif tiap rute sesuai semantik framework.
func (w *walker) finalize() {
	byID := map[int]*Router{}
	for _, r := range w.g.Routers {
		byID[r.ID] = r
	}
	rootOf := func(r *Router) *Router {
		for r.Parent >= 0 && byID[r.Parent] != nil {
			r = byID[r.Parent]
		}
		return r
	}
	ancestors := func(r *Router) map[int]bool {
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
	for _, rt := range w.g.Routes {
		r := byID[rt.RouterID]
		if r == nil {
			continue
		}
		root, anc := rootOf(r), ancestors(r)
		var cands []Middleware
		for _, o := range w.g.Routers {
			if rootOf(o) != root {
				continue
			}
			for _, mw := range o.Middlewares {
				scopePath := mw.ScopePath
				if scopePath == "" {
					scopePath = o.Prefix
				}
				switch {
				case r.Framework == "fiber": // fiber: middleware group/Use berlaku berdasarkan prefix path, lintas objek group
					if mw.Scope == "with" && !anc[o.ID] {
						continue
					}
					if !hasPathPrefix(rt.Path, scopePath) {
						continue
					}
				default: // gin/echo/chi/gorilla: hanya middleware milik leluhur router rute
					if !anc[o.ID] {
						continue
					}
					if mw.Scope == "prefix" && !hasPathPrefix(rt.Path, scopePath) {
						continue
					}
				}
				cands = append(cands, mw)
			}
		}
		sort.SliceStable(cands, func(i, j int) bool { return cands[i].seq < cands[j].seq })
		cands = append(cands, w.routeMW[rt]...)
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
