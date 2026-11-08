package routemap

import (
	"go/ast"
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
