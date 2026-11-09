package routemap

import (
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"strconv"
	"strings"
)

const maxInvokeDepth = 12

var (
	authRe     = regexp.MustCompile(`(?i)(auth|jwt|protect|guard|session|permission|rbac|acl|casbin|bearer|token|signin|login_?required|require_?(admin|role|user|login|exec)|localaccess|local_access|execaccess|exec_access)`)
	optionalRe = regexp.MustCompile(`(?i)optional|maybe|try|soft`)
)

// frameworkRoots: import path paket inti framework router (subpaket seperti fiber/middleware/recover bukan router).
var frameworkRoots = []struct {
	re *regexp.Regexp
	fw string
}{
	{regexp.MustCompile(`^github\.com/gofiber/fiber(/v\d+)?$`), "fiber"},
	{regexp.MustCompile(`^github\.com/gin-gonic/gin$`), "gin"},
	{regexp.MustCompile(`^github\.com/labstack/echo(/v\d+)?$`), "echo"},
	{regexp.MustCompile(`^github\.com/go-chi/chi(/v\d+)?$`), "chi"},
	{regexp.MustCompile(`^github\.com/gorilla/mux$`), "gorilla"},
	{regexp.MustCompile(`^net/http$`), "nethttp"},
}

// frameworkOf memetakan import path paket inti → nama framework router ("" bila bukan).
func frameworkOf(ip string) string {
	for _, f := range frameworkRoots {
		if f.re.MatchString(ip) {
			return f.fw
		}
	}
	return ""
}

var constructors = map[string]map[string]bool{
	"fiber":   {"New": true},
	"gin":     {"New": true, "Default": true},
	"echo":    {"New": true},
	"chi":     {"NewRouter": true, "NewMux": true},
	"gorilla": {"NewRouter": true},
	"nethttp": {"NewServeMux": true},
}

// tipe parameter yang menandakan router (untuk fungsi setup yang tidak terjangkau dari main)
var routerParamRe = regexp.MustCompile(`^\*?(fiber\.(App|Router)|gin\.(Engine|RouterGroup|IRouter|IRoutes)|echo\.(Echo|Group)|chi\.(Router|Mux)|mux\.Router|http\.ServeMux)$`)

var httpMethods = map[string]string{
	"Get": "GET", "Post": "POST", "Put": "PUT", "Delete": "DELETE", "Patch": "PATCH", "Head": "HEAD",
	"Options": "OPTIONS", "Connect": "CONNECT", "Trace": "TRACE", "All": "ANY", "Any": "ANY",
	"GET": "GET", "POST": "POST", "PUT": "PUT", "DELETE": "DELETE", "PATCH": "PATCH", "HEAD": "HEAD", "OPTIONS": "OPTIONS",
	"CONNECT": "CONNECT", "TRACE": "TRACE",
}

// valRef = nilai non-router yang disimpan di variabel (middleware, handler, config) beserta berkas asalnya.
type valRef struct {
	expr ast.Expr
	fi   *fileInfo
	sc   *scope
}

type scope struct {
	fi    *fileInfo
	vars  map[string]*Router
	vals  map[string]valRef
	stack []string
	ret   *Router
}

type walker struct {
	m         *module
	g         *Graph
	seq       int
	depth     int
	active    map[string]bool // fungsi yang sedang ditelusuri (cegah rekursi)
	visited   map[string]bool // fungsi yang pernah ditelusuri sebagai setup
	routeMW   map[*Route][]Middleware
	mounted   map[int]bool
	frameSeen map[string]bool
	fields    map[types.Object]*Router // field struct yang menyimpan router (s.app), lintas method (go/types)
}

func newScope(fi *fileInfo, stack []string) *scope {
	return &scope{fi: fi, vars: map[string]*Router{}, vals: map[string]valRef{}, stack: stack}
}

func exprStr(e ast.Expr) string {
	if e == nil {
		return ""
	}
	if _, ok := e.(*ast.FuncLit); ok {
		return "func literal"
	}
	return types.ExprString(e)
}

func shorten(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}

// key untuk variabel: Ident atau selector field (s.app).
func varKey(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		if k := varKey(x.X); k != "" {
			return k + "." + x.Sel.Name
		}
	case *ast.StarExpr:
		return varKey(x.X)
	case *ast.ParenExpr:
		return varKey(x.X)
	}
	return ""
}

func strLit(e ast.Expr) (string, bool) {
	if bl, ok := e.(*ast.BasicLit); ok && bl.Kind == token.STRING {
		if s, err := strconv.Unquote(bl.Value); err == nil {
			return s, true
		}
	}
	return "", false
}

func (w *walker) newRouter(parent *Router, fw, prefix, v string, at token.Pos, sc *scope) *Router {
	r := &Router{ID: len(w.g.Routers), Parent: -1, Framework: fw, Prefix: prefix, Var: v, Pos: w.m.pos(at), Stack: append([]string(nil), sc.stack...)}
	if parent != nil {
		r.Parent = parent.ID
		r.Framework = parent.Framework
		r.Prefix = joinPath(parent.Prefix, prefix)
	}
	if r.Prefix == "" {
		r.Prefix = "/"
	}
	w.g.Routers = append(w.g.Routers, r)
	return r
}

// evalRouter mengevaluasi ekspresi; mengembalikan router bila ekspresi bernilai router. Pemanggilan
// (Group/Use/Get/fungsi setup) dijalankan sebagai efek samping tepat satu kali.
func (w *walker) evalRouter(e ast.Expr, sc *scope) *Router {
	switch x := e.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		if r := sc.vars[varKey(x)]; r != nil {
			return r
		}
		if obj := w.fieldOf(x); obj != nil {
			return w.fields[obj]
		}
		return nil
	case *ast.CompositeLit: // &Server{app: app}: simpan router per field
		w.storeFields(x, sc)
		return nil
	case *ast.ParenExpr:
		return w.evalRouter(x.X, sc)
	case *ast.StarExpr:
		return w.evalRouter(x.X, sc)
	case *ast.UnaryExpr:
		return w.evalRouter(x.X, sc)
	case *ast.CallExpr:
		return w.call(x, sc)
	case *ast.FuncLit:
		return nil
	}
	return nil
}

func (w *walker) assign(lhs []ast.Expr, rhs []ast.Expr, sc *scope) {
	if len(rhs) == 1 && len(lhs) >= 1 {
		r := w.evalRouter(rhs[0], sc)
		k := varKey(lhs[0])
		if k == "" || k == "_" {
			return
		}
		if r != nil {
			sc.vars[k] = r
			if obj := w.fieldOf(lhs[0]); obj != nil {
				w.fields[obj] = r
			}
		} else if len(lhs) == 1 {
			delete(sc.vars, k)
			sc.vals[k] = valRef{expr: rhs[0], fi: sc.fi, sc: sc}
		}
		return
	}
	for i := range lhs {
		if i >= len(rhs) {
			break
		}
		w.assign(lhs[i:i+1], rhs[i:i+1], sc)
	}
}

func (w *walker) walkStmts(list []ast.Stmt, sc *scope) {
	for _, s := range list {
		w.walkStmt(s, sc)
	}
}

func (w *walker) walkStmt(s ast.Stmt, sc *scope) {
	switch x := s.(type) {
	case *ast.AssignStmt:
		w.assign(x.Lhs, x.Rhs, sc)
	case *ast.DeclStmt:
		if gd, ok := x.Decl.(*ast.GenDecl); ok {
			for _, sp := range gd.Specs {
				if vs, ok := sp.(*ast.ValueSpec); ok && len(vs.Values) > 0 {
					lhs := make([]ast.Expr, len(vs.Names))
					for i, n := range vs.Names {
						lhs[i] = n
					}
					w.assign(lhs, vs.Values, sc)
				}
			}
		}
	case *ast.ExprStmt:
		w.evalRouter(x.X, sc)
	case *ast.ReturnStmt:
		for _, r := range x.Results {
			if rt := w.evalRouter(r, sc); rt != nil {
				sc.ret = rt
			}
		}
	case *ast.BlockStmt:
		w.walkStmts(x.List, sc)
	case *ast.IfStmt:
		if x.Init != nil {
			w.walkStmt(x.Init, sc)
		}
		w.evalRouter(x.Cond, sc)
		w.walkStmts(x.Body.List, sc)
		if x.Else != nil {
			w.walkStmt(x.Else, sc)
		}
	case *ast.ForStmt:
		w.walkStmts(x.Body.List, sc)
	case *ast.RangeStmt:
		w.walkStmts(x.Body.List, sc)
	case *ast.SwitchStmt:
		for _, c := range x.Body.List {
			if cc, ok := c.(*ast.CaseClause); ok {
				w.walkStmts(cc.Body, sc)
			}
		}
	case *ast.TypeSwitchStmt:
		for _, c := range x.Body.List {
			if cc, ok := c.(*ast.CaseClause); ok {
				w.walkStmts(cc.Body, sc)
			}
		}
	case *ast.LabeledStmt:
		w.walkStmt(x.Stmt, sc)
	case *ast.GoStmt:
		w.evalRouter(x.Call, sc)
	case *ast.DeferStmt:
		w.evalRouter(x.Call, sc)
	}
}

// call memproses satu pemanggilan: konstruktor router, method router, atau fungsi setup lokal.
func (w *walker) call(ce *ast.CallExpr, sc *scope) *Router {
	if lit, ok := ce.Fun.(*ast.FuncLit); ok { // (func(){...})()
		ns := w.childScope(sc, sc.fi, nil)
		w.walkStmts(lit.Body.List, ns)
		return ns.ret
	}
	if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
		if ip := importOf(sc.fi, sel); ip != "" {
			if fw := frameworkOf(ip); fw != "" {
				if constructors[fw][sel.Sel.Name] {
					return w.newRouter(nil, fw, "/", "", ce.Pos(), sc)
				}
				return nil
			}
		}
		// gorilla: r.HandleFunc(p, h).Methods("POST")
		if sel.Sel.Name == "Methods" {
			if inner, ok := sel.X.(*ast.CallExpr); ok {
				if isel, ok := inner.Fun.(*ast.SelectorExpr); ok {
					if r := w.evalRouter(isel.X, sc); r != nil {
						var ms []string
						for _, a := range ce.Args {
							if s, ok := strLit(a); ok {
								ms = append(ms, strings.ToUpper(s))
							}
						}
						w.routerMethod(r, isel.Sel.Name, inner, sc, strings.Join(ms, ","))
						return nil
					}
				}
			}
		}
		if r := w.evalRouter(sel.X, sc); r != nil {
			return w.routerMethod(r, sel.Sel.Name, ce, sc, "")
		}
	}
	if f := w.m.resolveFunc(sc.fi, ce.Fun); f != nil {
		return w.invoke(f, ce, sc)
	}
	// method lewat interface (module.RegisterRoutes(router)): bila ada argumen router, telusuri semua
	// implementasi bernama sama di modul (dispatch konservatif ala CHA, golang.org/x/tools/go/callgraph/cha).
	if fs := w.m.resolveFuncs(sc.fi, ce.Fun); len(fs) > 1 && w.hasRouterArg(ce, sc) {
		for _, f := range fs {
			w.invoke(f, ce, sc)
		}
		return nil
	}
	// pemanggilan lain: telusuri argumen yang berupa pemanggilan (log.Fatal(setup(app)), must(...))
	for _, a := range ce.Args {
		if c, ok := a.(*ast.CallExpr); ok {
			w.call(c, sc)
		}
	}
	return nil
}

// fieldOf: objek field struct yang dirujuk selector (s.app) menurut go/types.
func (w *walker) fieldOf(e ast.Expr) types.Object {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || w.m.types == nil || w.fields == nil {
		return nil
	}
	if s := w.m.types.info.Selections[sel]; s != nil && s.Kind() == types.FieldVal {
		return s.Obj()
	}
	return nil
}

func (w *walker) storeFields(cl *ast.CompositeLit, sc *scope) {
	if w.m.types == nil {
		return
	}
	for _, el := range cl.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		id, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if r := w.evalRouter(kv.Value, sc); r != nil {
			if obj, ok := w.m.types.info.Uses[id].(*types.Var); ok && obj.IsField() {
				w.fields[obj] = r
			}
		}
	}
}

func (w *walker) hasRouterArg(ce *ast.CallExpr, sc *scope) bool {
	for _, a := range ce.Args {
		if k := varKey(a); k != "" && sc.vars[k] != nil {
			return true
		}
	}
	return false
}

func (w *walker) childScope(sc *scope, fi *fileInfo, frame []string) *scope {
	ns := newScope(fi, append(append([]string(nil), sc.stack...), frame...))
	if fi == sc.fi {
		for k, v := range sc.vars {
			ns.vars[k] = v
		}
		for k, v := range sc.vals {
			ns.vals[k] = v
		}
	}
	return ns
}

func paramNames(ft *ast.FuncType) []string {
	var names []string
	if ft == nil || ft.Params == nil {
		return names
	}
	for _, f := range ft.Params.List {
		if len(f.Names) == 0 {
			names = append(names, "_")
		}
		for _, n := range f.Names {
			names = append(names, n.Name)
		}
	}
	return names
}

// invoke menelusuri fungsi setup lokal dengan parameter router terikat ke argumen pemanggil.
func (w *walker) invoke(f *funcInfo, ce *ast.CallExpr, sc *scope) *Router {
	if w.active[f.Key] || w.depth >= maxInvokeDepth {
		return nil
	}
	params := paramNames(f.Decl.Type)
	bound := false
	ns := newScope(f.file, append(append([]string(nil), sc.stack...), w.m.pos(ce.Pos())+" "+shorten(exprStr(ce.Fun), 60)+"()"))
	for i, a := range ce.Args {
		if i >= len(params) {
			break
		}
		if r := w.evalRouter(a, sc); r != nil {
			ns.vars[params[i]] = r
			bound = true
		} else {
			ns.vals[params[i]] = valRef{expr: a, fi: sc.fi, sc: sc}
		}
	}
	// method: salin field router milik receiver (srv.app → s.app)
	if f.Decl.Recv != nil && len(f.Decl.Recv.List) > 0 && len(f.Decl.Recv.List[0].Names) > 0 {
		if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
			if rk := varKey(sel.X); rk != "" {
				rn := f.Decl.Recv.List[0].Names[0].Name
				for k, r := range sc.vars {
					if strings.HasPrefix(k, rk+".") {
						ns.vars[rn+k[len(rk):]] = r
						bound = true
					}
				}
			}
		}
	}
	// fungsi tanpa router terikat (config.Load(), newServer()) ditelusuri sekali saja: mungkin membuat router sendiri
	if !bound && w.visited[f.Key] {
		return nil
	}
	w.visited[f.Key] = true
	w.active[f.Key] = true
	w.depth++
	w.walkStmts(f.Decl.Body.List, ns)
	w.depth--
	delete(w.active, f.Key)
	return ns.ret
}

func (w *walker) nextSeq() int { w.seq++; return w.seq }

// routerMethod menangani method pada router: Group/Route/With/Use/Mount/PathPrefix dan pendaftaran rute.
func (w *walker) routerMethod(r *Router, name string, ce *ast.CallExpr, sc *scope, methodOverride string) *Router {
	args := ce.Args
	fw := r.Framework
	switch name {
	case "Group":
		if fw == "chi" { // r.Group(func(r chi.Router){...})
			child := w.newRouter(r, fw, "", "", ce.Pos(), sc)
			if len(args) > 0 {
				w.invokeLit(args[0], child, sc)
			}
			return child
		}
		if len(args) == 0 {
			return nil
		}
		p, _ := strLit(args[0])
		child := w.newRouter(r, fw, p, "", ce.Pos(), sc)
		seq := w.nextSeq()
		for _, a := range args[1:] {
			child.Middlewares = append(child.Middlewares, w.middleware(a, sc, "group", child, seq))
		}
		return child
	case "Route":
		if len(args) >= 2 && (fw == "chi" || fw == "fiber") {
			p, _ := strLit(args[0])
			child := w.newRouter(r, fw, p, "", ce.Pos(), sc)
			w.invokeLit(args[len(args)-1], child, sc)
			return child
		}
	case "With":
		child := w.newRouter(r, fw, "", "", ce.Pos(), sc)
		seq := w.nextSeq()
		for _, a := range args {
			child.Middlewares = append(child.Middlewares, w.middleware(a, sc, "with", child, seq))
		}
		return child
	case "PathPrefix":
		if len(args) > 0 {
			p, _ := strLit(args[0])
			return w.newRouter(r, fw, p, "", ce.Pos(), sc)
		}
	case "Subrouter":
		return r
	case "Use", "UseRouter", "Pre":
		scopeName := "group"
		if r.Parent < 0 {
			scopeName = "global"
		}
		usePrefix := ""
		if len(args) > 0 {
			if s, ok := strLit(args[0]); ok {
				usePrefix, args = s, args[1:]
				scopeName = "prefix"
			}
		}
		seq := w.nextSeq()
		for _, a := range args {
			if sub := w.evalRouter(a, sc); sub != nil && sub != r { // fiber app.Use("/x", subApp) = mount
				w.mount(r, sub, usePrefix)
				continue
			}
			mw := w.middleware(a, sc, scopeName, r, seq)
			if usePrefix != "" {
				mw.ScopePath = joinPath(r.Prefix, usePrefix)
			}
			r.Middlewares = append(r.Middlewares, mw)
		}
		return nil
	case "Mount":
		if len(args) >= 2 {
			p, _ := strLit(args[0])
			if sub := w.evalRouter(args[1], sc); sub != nil {
				w.mount(r, sub, p)
			}
		}
		return nil
	}
	w.route(r, name, ce, sc, methodOverride)
	return nil
}

// invokeLit menjalankan callback router (func(r chi.Router){...}) dengan parameter pertama = child.
func (w *walker) invokeLit(e ast.Expr, child *Router, sc *scope) {
	switch fn := e.(type) {
	case *ast.FuncLit:
		ns := w.childScope(sc, sc.fi, nil)
		if ps := paramNames(fn.Type); len(ps) > 0 {
			ns.vars[ps[0]] = child
		}
		w.walkStmts(fn.Body.List, ns)
	default:
		if f := w.m.resolveFunc(sc.fi, e); f != nil && !w.active[f.Key] {
			ns := newScope(f.file, append(append([]string(nil), sc.stack...), w.m.pos(e.Pos())+" "+exprStr(e)+"()"))
			if ps := paramNames(f.Decl.Type); len(ps) > 0 {
				ns.vars[ps[0]] = child
			}
			w.active[f.Key] = true
			w.walkStmts(f.Decl.Body.List, ns)
			delete(w.active, f.Key)
		}
	}
}

// mount memasang sub-router di bawah prefix router lain (prefix seluruh turunan & rutenya disesuaikan).
func (w *walker) mount(parent, sub *Router, p string) {
	if w.mounted[sub.ID] {
		return
	}
	w.mounted[sub.ID] = true
	base := joinPath(parent.Prefix, p)
	old := sub.Prefix
	ids := map[int]bool{sub.ID: true}
	for _, r := range w.g.Routers { // urutan ID = urutan pembuatan, jadi turunan selalu sesudah induknya
		if ids[r.Parent] && r.ID != sub.ID {
			ids[r.ID] = true
		}
	}
	rebase := func(s string) string { return joinPath(base, strings.TrimPrefix(s, strings.TrimSuffix(old, "/"))) }
	for _, r := range w.g.Routers {
		if ids[r.ID] {
			r.Prefix = rebase(r.Prefix)
			for i := range r.Middlewares {
				if r.Middlewares[i].ScopePath != "" {
					r.Middlewares[i].ScopePath = rebase(r.Middlewares[i].ScopePath)
				}
			}
		}
	}
	for _, rt := range w.g.Routes {
		if ids[rt.RouterID] {
			rt.Path = rebase(rt.Path)
		}
	}
	sub.Parent = parent.ID
}

// route mendaftarkan endpoint dari method router (Get/Post/Handle/HandleFunc/Add/Match/Method/Static…).
func (w *walker) route(r *Router, name string, ce *ast.CallExpr, sc *scope, methodOverride string) {
	args := ce.Args
	method, ok := httpMethods[name]
	pathIdx, rest := 0, 1
	switch {
	case ok:
	case name == "Handle" || name == "HandleFunc" || name == "Handler":
		method = "ANY"
	case name == "Add" || name == "Method" || name == "MethodFunc" || name == "Match":
		if len(args) < 2 {
			return
		}
		method = methodName(args[0])
		pathIdx, rest = 1, 2
	case name == "Static" || name == "StaticFS" || name == "StaticFile" || name == "File":
		method = "GET"
		if len(args) < 1 {
			return
		}
		p, _ := strLit(args[0])
		rt := &Route{Method: method, Path: joinPath(r.Prefix, p), Pos: w.m.pos(ce.Pos()), RouterID: r.ID,
			Stack: append([]string(nil), sc.stack...), Handler: "static " + shorten(exprStr(argAt(args, 1)), 60), seq: w.nextSeq()}
		w.g.Routes = append(w.g.Routes, rt)
		return
	default:
		return
	}
	if methodOverride != "" {
		method = methodOverride
	}
	if len(args) <= pathIdx {
		return
	}
	p, ok := strLit(args[pathIdx])
	if !ok {
		p = "{" + shorten(exprStr(args[pathIdx]), 40) + "}"
	}
	if r.Framework == "nethttp" { // Go 1.22: "POST /x"
		if m, rest, ok := strings.Cut(p, " "); ok && strings.ToUpper(m) == m {
			method, p = m, strings.TrimSpace(rest)
		}
	}
	hs := args[rest:]
	if len(hs) == 0 {
		return
	}
	var handler ast.Expr
	var mws []ast.Expr
	if r.Framework == "echo" { // e.GET(path, h, m...)
		handler, mws = hs[0], hs[1:]
	} else { // fiber/gin: (path, mw..., handler)
		handler, mws = hs[len(hs)-1], hs[:len(hs)-1]
	}
	rt := &Route{Method: method, Path: joinPath(r.Prefix, p), Pos: w.m.pos(ce.Pos()), RouterID: r.ID,
		Stack: append([]string(nil), sc.stack...), seq: w.nextSeq()}
	for _, m := range mws {
		w.routeMW[rt] = append(w.routeMW[rt], w.middleware(m, sc, "route", r, rt.seq))
	}
	w.describeHandler(rt, handler, sc)
	w.g.Routes = append(w.g.Routes, rt)
}

func argAt(args []ast.Expr, i int) ast.Expr {
	if i < len(args) {
		return args[i]
	}
	return nil
}

func methodName(e ast.Expr) string {
	if s, ok := strLit(e); ok {
		return strings.ToUpper(s)
	}
	if cl, ok := e.(*ast.CompositeLit); ok { // gin Match([]string{"GET","POST"}, ...)
		var ms []string
		for _, el := range cl.Elts {
			ms = append(ms, methodName(el))
		}
		return strings.Join(ms, ",")
	}
	s := exprStr(e)
	if i := strings.LastIndex(s, "."); i >= 0 {
		s = s[i+1:]
	}
	s = strings.TrimPrefix(s, "Method")
	return strings.ToUpper(s)
}

// middleware mendeskripsikan argumen middleware: nama, deklarasi lokal, auth, dan kondisi skip.
func (w *walker) middleware(e ast.Expr, sc *scope, scopeName string, r *Router, seq int) Middleware {
	mw := Middleware{Scope: scopeName, Pos: w.m.pos(e.Pos()), seq: seq, router: r.ID}
	if scopeName == "group" || scopeName == "with" {
		mw.ScopePath = r.Prefix
	}
	expr, fi := e, sc.fi
	name := exprStr(e)
	if id, ok := e.(*ast.Ident); ok {
		if v, ok := sc.vals[id.Name]; ok {
			expr, fi = v.expr, v.fi
			name = id.Name + " = " + exprStr(v.expr)
		}
	}
	mw.Name = shorten(name, 90)
	var declName string
	var target ast.Expr = expr
	if c, ok := expr.(*ast.CallExpr); ok {
		target = c.Fun
	}
	if f := w.m.resolveFunc(fi, target); f != nil {
		mw.Decl, declName = f.Pos, f.Name
		pre, exact := skipConditions(f.Decl.Body, false)
		mw.SkipPrefixes, mw.SkipPaths = append(mw.SkipPrefixes, pre...), append(mw.SkipPaths, exact...)
	}
	if lit, ok := expr.(*ast.FuncLit); ok {
		mw.Decl = w.m.pos(lit.Pos())
		pre, exact := skipConditions(lit.Body, false)
		mw.SkipPrefixes, mw.SkipPaths = append(mw.SkipPrefixes, pre...), append(mw.SkipPaths, exact...)
	}
	// Config{Next: func(c) bool {...}} / Skipper
	ast.Inspect(expr, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		if k, ok := kv.Key.(*ast.Ident); ok && (k.Name == "Next" || k.Name == "Skipper" || k.Name == "Filter" || k.Name == "SkipFunc") {
			if lit, ok := kv.Value.(*ast.FuncLit); ok {
				pre, exact := skipConditions(lit.Body, true)
				mw.SkipPrefixes, mw.SkipPaths = append(mw.SkipPrefixes, pre...), append(mw.SkipPaths, exact...)
			}
		}
		return true
	})
	label := name + " " + declName
	if authRe.MatchString(label) {
		mw.Auth = true
		mw.Optional = optionalRe.MatchString(declName) || optionalRe.MatchString(exprStr(target))
	}
	return mw
}

// skipConditions mengumpulkan path yang dilewati middleware: strings.HasPrefix(c.Path(), "/x") dan c.Path() == "/x".
// predicate=true: seluruh fungsi adalah predikat skip (Next/Skipper); false: hanya if yang isinya c.Next().
func skipConditions(body *ast.BlockStmt, predicate bool) (prefixes, exact []string) {
	if body == nil {
		return
	}
	collect := func(n ast.Node) {
		ast.Inspect(n, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "HasPrefix" && len(x.Args) == 2 && mentionsPath(x.Args[0]) {
					if s, ok := strLit(x.Args[1]); ok {
						prefixes = append(prefixes, s)
					}
				}
			case *ast.BinaryExpr:
				if x.Op == token.EQL {
					if mentionsPath(x.X) {
						if s, ok := strLit(x.Y); ok && strings.HasPrefix(s, "/") {
							exact = append(exact, s)
						}
					} else if mentionsPath(x.Y) {
						if s, ok := strLit(x.X); ok && strings.HasPrefix(s, "/") {
							exact = append(exact, s)
						}
					}
				}
			}
			return true
		})
	}
	if predicate {
		collect(body)
		return
	}
	ast.Inspect(body, func(n ast.Node) bool {
		ifs, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		if callsNext(ifs.Body) {
			collect(ifs.Cond)
		}
		return true
	})
	return
}

func mentionsPath(e ast.Expr) bool {
	s := exprStr(e)
	return strings.Contains(s, "Path") || strings.Contains(s, "URL") || strings.Contains(s, "OriginalURL") || strings.Contains(s, "RequestURI")
}

func callsNext(b *ast.BlockStmt) bool {
	found := false
	ast.Inspect(b, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Next" {
				found = true
			}
		}
		return !found
	})
	return found
}
