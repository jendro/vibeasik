package routemap

import (
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

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
		if sel, isSel := ce.Fun.(*ast.SelectorExpr); isSel && w.directResponse(ce, sel, fb, env, via, ctx, add) {
			return true
		}
		// helper lokal yang menerima ctx: fail(c, 400, "...")
		if depth < maxHelperDepth {
			w.helperResponses(ce, fb, env, via, depth, ctx, add)
		}
		return true
	})
	sort.SliceStable(out, func(i, j int) bool { return out[i].Status < out[j].Status })
	return out
}

// directResponse mengenali pemanggilan yang langsung menulis response (NewError, http.Error, c.JSON, c.Next).
// Mengembalikan true bila pemanggilan sudah ditangani.
func (w *walker) directResponse(ce *ast.CallExpr, sel *ast.SelectorExpr, fb fnBody, env map[string]binding, via string, ctx map[string]bool, add func(Response)) bool {
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
		add(w.writeResponse(ce, sel, wm.status, wm.kind, fb, env, via))
		return true
	}
	if name == "Next" && ctx[rootIdent(sel.X)] {
		add(Response{Kind: "next", Via: via, Pos: w.m.pos(ce.Pos())})
		return true
	}
	return false
}

// writeResponse membentuk Response dari pemanggilan method penulis response pada ctx.
func (w *walker) writeResponse(ce *ast.CallExpr, sel *ast.SelectorExpr, defStatus int, kind string, fb fnBody, env map[string]binding, via string) Response {
	r := Response{Status: defStatus, Kind: kind, Via: via, Pos: w.m.pos(ce.Pos())}
	if inner, ok := sel.X.(*ast.CallExpr); ok { // c.Status(400).JSON(...)
		if is, ok := inner.Fun.(*ast.SelectorExpr); ok && is.Sel.Name == "Status" {
			r.Status = w.status(argAt(inner.Args, 0), fb.fi, env)
		}
	}
	r.Status = w.writeStatus(ce, sel.Sel.Name, defStatus, kind, r.Status, fb, env)
	if len(ce.Args) > 0 && r.Kind != "status" && r.Kind != "redirect" {
		r.Body = describeBody(ce.Args[len(ce.Args)-1])
	}
	return r
}

// helperResponses menelusuri helper lokal yang menerima ctx sebagai argumen.
func (w *walker) helperResponses(ce *ast.CallExpr, fb fnBody, env map[string]binding, via string, depth int, ctx map[string]bool, add func(Response)) {
	passesCtx := false
	for _, a := range ce.Args {
		if ctx[rootIdent(a)] {
			passesCtx = true
			break
		}
	}
	if !passesCtx {
		return
	}
	f := w.m.resolveFunc(fb.fi, ce.Fun)
	if f == nil {
		return
	}
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

// writeStatus menentukan status response dari argumen pemanggilan penulis response (cur = status sementara).
func (w *walker) writeStatus(ce *ast.CallExpr, name string, defStatus int, kind string, cur int, fb fnBody, env map[string]binding) int {
	switch {
	case defStatus == 0: // status di argumen pertama
		return w.status(argAt(ce.Args, 0), fb.fi, env)
	case name == "Redirect":
		if st := w.status(argAt(ce.Args, 0), fb.fi, env); st >= 300 { // gin Redirect(code, loc)
			return st
		} else if st := w.status(argAt(ce.Args, 1), fb.fi, env); st >= 300 { // fiber Redirect(loc, code)
			return st
		}
	case len(ce.Args) >= 2 && (kind == "json" || kind == "string" || kind == "xml" || kind == "html"):
		if st := w.status(ce.Args[0], fb.fi, env); st > 0 { // gin/echo c.JSON(code, body)
			return st
		}
	}
	return cur
}
