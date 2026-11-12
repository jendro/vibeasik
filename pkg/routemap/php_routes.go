package routemap

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Rute PHP untuk penghitung biaya:
//
//	Laravel/Lumen  routes/*.php: Route::get/post/…/match/any/resource/apiResource, prefix/middleware/controller/group
//	               (closure, array atribut, require berkas lain), aksi [C::class, 'm'] / 'C@m' / C::class / closure;
//	               route model binding (show(User $user) + {user}) dihitung 1 query.
//	CodeIgniter 4  $routes->get('x', 'Home::index'), $routes->group('admin', ['filter' => 'auth'], fn)
//	Slim/FastRoute $app->get('/x', 'C:m'), $app->group('/api', fn), $r->addRoute('GET', '/x', h)
//	PHP biasa      tiap berkas .php yang dijalankan langsung (tidak di-include berkas lain) = satu halaman.

const (
	maxPHPFiles  = 4000
	maxPHPRoutes = 5000
)

var phpSkipDirs = map[string]bool{
	"vendor": true, "node_modules": true, "storage": true, "tests": true, "test": true, "database": true, "cache": true,
	".git": true, ".vibeasik": true, "dist": true, "build": true, "tmp": true, "logs": true, "lang": true,
}

// PHPModule = satu aplikasi PHP (satu composer.json, atau folder berisi berkas .php tanpa composer).
type PHPModule struct {
	Name      string `json:"name"`
	Dir       string `json:"dir"`
	Framework string `json:"framework"` // laravel, lumen, codeigniter4, slim, php
}

// findPHPModules: folder ber-composer.json (maks. kedalaman 4) yang memuat kode PHP sendiri; tanpa composer.json
// tetapi ada berkas .php di root → root dianggap satu aplikasi PHP biasa.
func findPHPModules(root string) []*phpModule {
	var dirs []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (phpSkipDirs[d.Name()] || skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			if rel, _ := filepath.Rel(root, p); strings.Count(rel, string(os.PathSeparator)) >= 4 {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "composer.json" {
			dirs = append(dirs, filepath.Dir(p))
		}
		return nil
	})
	if len(dirs) == 0 && hasPHPFiles(root) {
		dirs = []string{root}
	}
	sort.Strings(dirs)
	var mods []*phpModule
	for _, dir := range dirs {
		m := &phpModule{Dir: dir, root: root}
		m.Name, m.Framework = composerInfo(dir)
		mods = append(mods, m)
	}
	for _, m := range mods {
		var nested []string
		for _, o := range mods {
			if o != m && strings.HasPrefix(o.Dir, m.Dir+string(os.PathSeparator)) {
				nested = append(nested, o.Dir)
			}
		}
		loadPHPFiles(m, root, nested)
		m.index()
		m.aliases = laravelAliases(m)
	}
	out := mods[:0]
	for _, m := range mods {
		if len(m.files) > 0 {
			out = append(out, m)
		}
	}
	return out
}

func hasPHPFiles(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || found {
			return filepath.SkipDir
		}
		if d.IsDir() {
			if p != root && (phpSkipDirs[d.Name()] || skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			if rel, _ := filepath.Rel(root, p); strings.Count(rel, string(os.PathSeparator)) >= 3 {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".php") && !strings.HasSuffix(d.Name(), ".blade.php") {
			found = true
		}
		return nil
	})
	return found
}

func composerInfo(dir string) (name, framework string) {
	name, framework = filepath.Base(dir), "php"
	raw, err := os.ReadFile(filepath.Join(dir, "composer.json"))
	if err == nil {
		var c struct {
			Name    string            `json:"name"`
			Require map[string]string `json:"require"`
		}
		if json.Unmarshal(raw, &c) == nil {
			if c.Name != "" {
				name = c.Name
			}
			switch {
			case c.Require["laravel/framework"] != "":
				framework = "laravel"
			case c.Require["laravel/lumen-framework"] != "":
				framework = "lumen"
			case c.Require["codeigniter4/framework"] != "":
				framework = "codeigniter4"
			case c.Require["slim/slim"] != "":
				framework = "slim"
			}
		}
	}
	if framework == "php" {
		if _, err := os.Stat(filepath.Join(dir, "artisan")); err == nil {
			framework = "laravel"
		}
	}
	return name, framework
}

func loadPHPFiles(m *phpModule, root string, nested []string) {
	_ = filepath.WalkDir(m.Dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || len(m.files) >= maxPHPFiles {
			return filepath.SkipDir
		}
		if d.IsDir() {
			if p == m.Dir {
				return nil
			}
			if phpSkipDirs[d.Name()] || skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			for _, n := range nested {
				if p == n {
					return filepath.SkipDir
				}
			}
			if rel, _ := filepath.Rel(m.Dir, p); rel == filepath.Join("resources", "views") || rel == filepath.Join("bootstrap", "cache") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".php") || strings.HasSuffix(p, ".blade.php") {
			return nil
		}
		if info, err := d.Info(); err != nil || info.Size() > maxFileBytes {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		m.files = append(m.files, parsePHPFile(filepath.ToSlash(rel), string(src)))
		return nil
	})
}

// laravelAliases: alias middleware → kelas, dari app/Http/Kernel.php ($middlewareAliases/$routeMiddleware)
// dan bootstrap/app.php (->alias([...])). Polanya: 'nama' => Kelas::class.
func laravelAliases(m *phpModule) map[string]string {
	out := map[string]string{}
	for _, f := range m.files {
		if !strings.HasSuffix(f.Rel, "Http/Kernel.php") && !strings.HasSuffix(f.Rel, "bootstrap/app.php") && !strings.HasSuffix(f.Rel, "Config/Filters.php") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(m.root, filepath.FromSlash(f.Rel)))
		if err != nil {
			continue
		}
		toks := lexPHP(string(src))
		for i := 0; i+4 < len(toks); i++ {
			if toks[i].K == ptStr && toks[i+1].is("=>") && toks[i+2].K == ptIdent && toks[i+3].is("::") && toks[i+4].kw("class") {
				out[toks[i].S] = phpShort(toks[i+2].S)
			}
		}
	}
	return out
}

// ---------- pengumpulan rute ----------

type phpRoute struct {
	Method, Path, Handler, Pos string
	fn                         *phpFunc   // handler method/fungsi lokal
	body                       []*phpStmt // handler closure
	bodyFile                   *phpFile
	bodyCls                    *phpClass
	entry                      *phpFile // halaman PHP biasa
	mws                        []string
	auth                       bool
	bindings                   int // route model binding: query implisit
}

type phpRouteCtx struct {
	prefix     string
	mws        []string
	controller string // Route::controller(C::class)->group(…)
	file       *phpFile
}

type phpCollector struct {
	m      *phpModule
	routes []*phpRoute
	seen   map[*phpFile]bool
}

func (pc *phpCollector) add(r *phpRoute) {
	if len(pc.routes) < maxPHPRoutes {
		pc.routes = append(pc.routes, r)
	}
}

func joinRoute(prefix, p string) string {
	out := "/" + strings.Trim(strings.Trim(prefix, "/")+"/"+strings.Trim(p, "/"), "/")
	for strings.Contains(out, "//") {
		out = strings.ReplaceAll(out, "//", "/")
	}
	return out
}

// collectPHPRoutes mengumpulkan rute modul sesuai frameworknya; tanpa rute router → halaman PHP biasa.
func collectPHPRoutes(m *phpModule) []*phpRoute {
	pc := &phpCollector{m: m, seen: map[*phpFile]bool{}}
	var routeFiles []*phpFile
	for _, f := range m.files {
		if strings.Contains("/"+f.Rel, "/routes/") && !strings.Contains(f.Rel, "/app/") {
			routeFiles = append(routeFiles, f)
		}
	}
	// Laravel: web.php & api.php dulu (api.php mendapat prefix /api), lalu berkas lain yang belum di-require.
	sort.SliceStable(routeFiles, func(i, j int) bool { return routeRank(routeFiles[i].Rel) < routeRank(routeFiles[j].Rel) })
	for _, f := range routeFiles {
		if pc.seen[f] {
			continue
		}
		ctx := phpRouteCtx{file: f}
		if path.Base(f.Rel) == "api.php" && (m.Framework == "laravel" || m.Framework == "php") {
			ctx.prefix, ctx.mws = "/api", []string{"api"}
		}
		pc.seen[f] = true
		pc.walk(ctx, f.Stmts)
	}
	// Router umum (CodeIgniter 4, Slim, FastRoute) di berkas mana pun: hanya pernyataan tingkat atas
	// (dan closure `return function ($app) {…}`) supaya rute di dalam group tidak terhitung dua kali.
	for _, f := range m.files {
		if pc.seen[f] {
			continue
		}
		before := len(pc.routes)
		pc.walk(phpRouteCtx{file: f}, f.Stmts)
		if len(pc.routes) > before {
			pc.seen[f] = true
		}
	}
	if len(pc.routes) == 0 && m.Framework != "laravel" && m.Framework != "lumen" {
		pc.pages()
	}
	return pc.routes
}

func routeRank(rel string) int {
	switch path.Base(rel) {
	case "web.php":
		return 0
	case "api.php":
		return 1
	}
	return 2
}

// walk menafsirkan pernyataan berkas rute.
func (pc *phpCollector) walk(ctx phpRouteCtx, list []*phpStmt) {
	for _, s := range list {
		switch s.Kind {
		case psBlock:
			pc.walk(ctx, s.Body)
		case psIf:
			pc.walk(ctx, s.Body)
			pc.walk(ctx, s.Else)
		case psExpr:
			pc.exprStmt(ctx, s.Expr)
		}
	}
}

func (pc *phpCollector) exprStmt(ctx phpRouteCtx, toks []phpTok) {
	if len(toks) == 0 {
		return
	}
	first := strings.ToLower(toks[0].S)
	if toks[0].K == ptIdent && (first == "require" || first == "require_once" || first == "include" || first == "include_once") {
		c := &phpCostCtx{m: pc.m}
		if f := c.resolveInclude(ctx.file, toks[1:]); f != nil && !pc.seen[f] {
			pc.seen[f] = true
			sub := ctx
			sub.file = f
			pc.walk(sub, f.Stmts)
		}
		return
	}
	if toks[0].kw("return") && isClosureStart(toks, 1) { // Slim: return function (App $app) { … };
		pc.closureBody(ctx, toks[1:])
		return
	}
	ch, _ := parsePHPChain(toks, 0)
	if ch == nil || len(ch.Segs) < 2 {
		return
	}
	switch {
	case ch.RootKind == "name" && strings.EqualFold(phpShort(ch.Root.S), "Route") && ch.Segs[0].Op == "::":
		pc.laravel(ctx, ch)
	case ch.RootKind == "var" && ch.Segs[0].Op == "->":
		pc.generic(ctx, ch)
	}
}

// calls: pasangan (nama method, argumen) dari rantai.
func chainCalls(ch *phpChain) (names []string, args [][][]phpTok, lines []int) {
	for j, s := range ch.Segs {
		if s.Op == "call" && j > 0 && (ch.Segs[j-1].Op == "->" || ch.Segs[j-1].Op == "::") {
			names = append(names, strings.ToLower(ch.Segs[j-1].Name))
			args = append(args, s.Args)
			lines = append(lines, s.Line)
		}
	}
	return
}

func strArg(a []phpTok) (string, bool) {
	a = stripNamedArg(a)
	if len(a) == 1 && a[0].K == ptStr && !a[0].Interp {
		return a[0].S, true
	}
	return "", false
}

// strList: 'a' | ['a', 'b'] | 'a', 'b' (beberapa argumen).
func strList(args [][]phpTok) []string {
	var out []string
	for _, a := range args {
		a = stripNamedArg(a)
		if s, ok := strArg(a); ok {
			out = append(out, s)
			continue
		}
		if len(a) >= 2 && (a[0].is("[") || a[0].kw("array")) {
			inner := a[1 : len(a)-1]
			if a[0].kw("array") && len(a) >= 3 {
				inner = a[2 : len(a)-1]
			}
			for _, el := range splitTop(inner, ",") {
				if s, ok := strArg(el); ok {
					out = append(out, s)
				} else if len(el) >= 3 && el[0].K == ptIdent && el[1].is("::") && el[2].kw("class") {
					out = append(out, phpShort(el[0].S))
				}
			}
			continue
		}
		if len(a) >= 3 && a[0].K == ptIdent && a[1].is("::") && a[2].kw("class") {
			out = append(out, phpShort(a[0].S))
		}
	}
	return out
}

// arrayAttrs: ['prefix' => 'admin', 'middleware' => ['auth'], 'filter' => 'auth'] → atribut group.
func arrayAttrs(a []phpTok) map[string][]string {
	out := map[string][]string{}
	if len(a) < 2 || !(a[0].is("[") || a[0].kw("array")) {
		return out
	}
	inner := a[1 : len(a)-1]
	if a[0].kw("array") && len(a) >= 3 {
		inner = a[2 : len(a)-1]
	}
	for _, el := range splitTop(inner, ",") {
		kv := splitTop(el, "=>")
		if len(kv) != 2 {
			continue
		}
		k, ok := strArg(kv[0])
		if !ok {
			continue
		}
		out[strings.ToLower(k)] = strList([][]phpTok{kv[1]})
	}
	return out
}

var httpVerbs = map[string]string{"get": "GET", "post": "POST", "put": "PUT", "patch": "PATCH", "delete": "DELETE",
	"options": "OPTIONS", "any": "ANY", "head": "HEAD", "cli": "", "match": "", "map": ""}

func (pc *phpCollector) laravel(ctx phpRouteCtx, ch *phpChain) {
	names, args, lines := chainCalls(ch)
	attrs := ctx
	var res *resourceSpec
	for i, name := range names {
		a := args[i]
		switch name {
		case "prefix":
			if s, ok := strArg(firstArg(a)); ok {
				attrs.prefix = joinRoute(attrs.prefix, s)
			}
		case "middleware":
			attrs.mws = append(append([]string{}, attrs.mws...), strList(a)...)
		case "controller":
			if l := strList(a); len(l) > 0 {
				attrs.controller = phpShort(l[0])
			}
		case "group":
			var cb []phpTok
			for _, x := range a {
				x = stripNamedArg(x)
				if isClosureStart(x, 0) {
					cb = x
				} else if at := arrayAttrs(x); len(at) > 0 {
					if p := at["prefix"]; len(p) > 0 {
						attrs.prefix = joinRoute(attrs.prefix, p[0])
					}
					attrs.mws = append(append([]string{}, attrs.mws...), at["middleware"]...)
				} else if s, ok := strArg(x); ok { // ->group(base_path('routes/x.php')) / group('routes/x.php')
					pc.groupFile(attrs, s)
				} else if f := (&phpCostCtx{m: pc.m}).resolveInclude(ctx.file, x); f != nil {
					pc.groupFile(attrs, f.Rel)
				}
			}
			if cb != nil {
				pc.closureBody(attrs, cb)
			}
		case "resource", "apiresource", "resources", "apiresources":
			l := strList(a)
			if len(l) >= 2 {
				res = &resourceSpec{name: l[0], controller: l[1], api: strings.HasPrefix(name, "api"), line: lines[i]}
			}
		case "only", "except":
			if res != nil {
				set := map[string]bool{}
				for _, s := range strList(a) {
					set[strings.ToLower(s)] = true
				}
				if name == "only" {
					res.only = set
				} else {
					res.except = set
				}
			}
		case "view", "redirect", "permanentredirect", "fallback":
			if s, ok := strArg(firstArg(a)); ok || name == "fallback" {
				method := "GET"
				if name == "fallback" {
					s = "{fallback}"
				}
				pc.add(&phpRoute{Method: method, Path: joinRoute(attrs.prefix, s), Handler: "Route::" + name,
					Pos: posOf(ctx.file, lines[i]), mws: attrs.mws, auth: hasAuth(attrs.mws)})
			}
		default:
			verb, ok := httpVerbs[name]
			if !ok {
				continue
			}
			methods := []string{verb}
			rest := a
			if name == "match" && len(a) >= 3 {
				methods = nil
				for _, v := range strList(a[:1]) {
					methods = append(methods, strings.ToUpper(v))
				}
				rest = a[1:]
			}
			if verb == "" && name != "match" || len(rest) < 1 {
				continue
			}
			uri, ok := strArg(rest[0])
			if !ok {
				continue
			}
			var action []phpTok
			if len(rest) > 1 {
				action = stripNamedArg(rest[1])
			}
			// middleware/nama yang dirantai SETELAH verb tetap berlaku untuk rute ini
			mws := append([]string{}, attrs.mws...)
			for k := i + 1; k < len(names); k++ {
				if names[k] == "middleware" {
					mws = append(mws, strList(args[k])...)
				}
			}
			for _, mth := range methods {
				r := &phpRoute{Method: mth, Path: joinRoute(attrs.prefix, uri), Pos: posOf(ctx.file, lines[i]), mws: mws}
				pc.action(attrs, r, action)
				r.auth = hasAuth(r.mws)
				pc.add(r)
			}
		}
	}
	if res != nil {
		pc.resource(attrs, res, names, args)
	}
}

func firstArg(a [][]phpTok) []phpTok {
	if len(a) == 0 {
		return nil
	}
	return a[0]
}

func posOf(f *phpFile, line int) string {
	if f == nil {
		return ""
	}
	return f.Rel + ":" + itoa(line)
}

func hasAuth(mws []string) bool {
	for _, m := range mws {
		l := strings.ToLower(m)
		if l == "auth" || strings.HasPrefix(l, "auth:") || strings.HasPrefix(l, "auth.") || l == "authenticate" ||
			strings.HasPrefix(l, "jwt") || l == "sanctum" || strings.HasPrefix(l, "filter:auth") || l == "session" {
			return true
		}
	}
	return false
}

func (pc *phpCollector) groupFile(ctx phpRouteCtx, rel string) {
	rel = strings.TrimPrefix(rel, "/")
	for _, f := range pc.m.files {
		if (f.Rel == rel || strings.HasSuffix(f.Rel, "/"+rel)) && !pc.seen[f] {
			pc.seen[f] = true
			sub := ctx
			sub.file = f
			pc.walk(sub, f.Stmts)
			return
		}
	}
}

// closureBody menafsirkan badan closure group sebagai pernyataan rute.
func (pc *phpCollector) closureBody(ctx phpRouteCtx, toks []phpTok) {
	i := 0
	if toks[i].kw("static") {
		i++
	}
	if toks[i].kw("fn") { // fn () => Route::get(…) — satu ekspresi
		for i < len(toks) && !toks[i].is("=>") {
			i++
		}
		if i+1 < len(toks) {
			pc.exprStmt(ctx, toks[i+1:])
		}
		return
	}
	for i < len(toks) && !toks[i].is("{") {
		i++
	}
	end := matchingClose(toks, i)
	if i >= len(toks) || end < 0 {
		return
	}
	pc.walk(ctx, parsePHPTokens(ctx.file, nil, toks[i+1:end]))
}

// action menautkan aksi rute ke handler lokal: [C::class, 'm'], 'C@m', 'C::m' (CI4), 'C:m' (Slim), C::class, closure.
func (pc *phpCollector) action(ctx phpRouteCtx, r *phpRoute, a []phpTok) {
	cls, method := "", ""
	switch {
	case len(a) == 0:
		r.Handler = "-"
		return
	case isClosureStart(a, 0):
		r.Handler = "closure"
		r.bodyFile = ctx.file
		r.body = closureStmts(ctx.file, a)
		return
	case a[0].is("[") || a[0].kw("array"):
		l := strList([][]phpTok{a})
		if len(l) >= 2 {
			cls, method = l[0], l[1]
		} else if len(l) == 1 {
			cls, method = l[0], "__invoke"
		}
	case len(a) >= 3 && a[0].K == ptIdent && a[1].is("::") && a[2].kw("class"):
		cls, method = phpShort(a[0].S), "__invoke"
	default:
		s, ok := strArg(a)
		if !ok {
			r.Handler = "?"
			return
		}
		s = strings.SplitN(s, "/", 2)[0] // CI4: 'Blog::view/$1'
		switch {
		case strings.Contains(s, "@"):
			p := strings.SplitN(s, "@", 2)
			cls, method = p[0], p[1]
		case strings.Contains(s, "::"):
			p := strings.SplitN(s, "::", 2)
			cls, method = p[0], p[1]
		case strings.Contains(s, ":"):
			p := strings.SplitN(s, ":", 2)
			cls, method = p[0], p[1]
		case ctx.controller != "":
			cls, method = ctx.controller, s
		default:
			cls, method = s, "__invoke"
		}
	}
	cls = phpShort(strings.ReplaceAll(cls, "/", "\\"))
	r.Handler = cls + "@" + method
	c := pc.m.class(ctx.file, cls)
	if c == nil {
		return
	}
	fn := pc.m.method(c, method)
	if fn == nil {
		return
	}
	r.fn = fn
	r.Handler = fn.Key
	// middleware dari konstruktor controller: $this->middleware('auth')
	if ctor := pc.m.method(c, "__construct"); ctor != nil {
		r.mws = append(r.mws, ctorMiddlewares(ctor.Body)...)
	}
	// route model binding: parameter bertipe model yang namanya cocok dengan {param} di URI
	for _, p := range fn.Params {
		if p.Type != "" && strings.Contains(r.Path, "{"+strings.TrimPrefix(p.Name, "$")) && pc.m.isModel(fn.file, p.Type) {
			r.bindings++
		}
	}
}

func closureStmts(f *phpFile, toks []phpTok) []*phpStmt {
	i := 0
	if toks[i].kw("static") {
		i++
	}
	if toks[i].kw("fn") {
		for i < len(toks) && !toks[i].is("=>") {
			i++
		}
		if i+1 >= len(toks) {
			return nil
		}
		return []*phpStmt{{Kind: psExpr, Line: toks[i].Line, Expr: toks[i+1:]}}
	}
	for i < len(toks) && !toks[i].is("{") {
		i++
	}
	end := matchingClose(toks, i)
	if i >= len(toks) || end < 0 {
		return nil
	}
	return parsePHPTokens(f, nil, toks[i+1:end])
}

func ctorMiddlewares(list []*phpStmt) []string {
	var out []string
	for _, s := range list {
		if s.Kind != psExpr || len(s.Expr) < 4 || s.Expr[0].S != "$this" {
			continue
		}
		ch, _ := parsePHPChain(s.Expr, 0)
		names, args, _ := chainCalls(ch)
		for i, n := range names {
			if n == "middleware" {
				out = append(out, strList(args[i])...)
			}
		}
	}
	return out
}

type resourceSpec struct {
	name, controller string
	api              bool
	only, except     map[string]bool
	line             int
}

func singular(s string) string {
	switch {
	case strings.HasSuffix(s, "ies"):
		return strings.TrimSuffix(s, "ies") + "y"
	case strings.HasSuffix(s, "ses"), strings.HasSuffix(s, "xes"):
		return strings.TrimSuffix(s, "es")
	case strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss"):
		return strings.TrimSuffix(s, "s")
	}
	return s
}

// resource: Route::resource('photos', C::class) → 7 rute (apiResource: 5), sesuai only/except.
func (pc *phpCollector) resource(ctx phpRouteCtx, res *resourceSpec, names []string, args [][][]phpTok) {
	base := ""
	param := ""
	for i, part := range strings.Split(res.name, ".") {
		p := strings.ReplaceAll(singular(part), "-", "_")
		if i > 0 {
			base += "/{" + param + "}"
		}
		base += "/" + part
		param = p
	}
	actions := []struct{ name, method, suffix string }{
		{"index", "GET", ""}, {"create", "GET", "/create"}, {"store", "POST", ""}, {"show", "GET", "/{" + param + "}"},
		{"edit", "GET", "/{" + param + "}/edit"}, {"update", "PUT", "/{" + param + "}"}, {"destroy", "DELETE", "/{" + param + "}"},
	}
	mws := append([]string{}, ctx.mws...)
	for i, n := range names {
		if n == "middleware" {
			mws = append(mws, strList(args[i])...)
		}
	}
	cls := pc.m.class(ctx.file, res.controller)
	for _, a := range actions {
		if res.api && (a.name == "create" || a.name == "edit") {
			continue
		}
		if res.only != nil && !res.only[a.name] || res.except != nil && res.except[a.name] {
			continue
		}
		if cls != nil && pc.m.method(cls, a.name) == nil {
			continue // Laravel memanggil method ini; bila controller lokal tidak punya, rute itu tidak dipakai
		}
		r := &phpRoute{Method: a.method, Path: joinRoute(ctx.prefix, base+a.suffix), Pos: posOf(ctx.file, res.line), mws: append([]string{}, mws...)}
		pc.action(ctx, r, []phpTok{{K: ptStr, S: res.controller + "@" + a.name}})
		r.auth = hasAuth(r.mws)
		pc.add(r)
	}
}

// generic: $routes->get('x', 'C::m') (CI4), $app->get('/x', handler) (Slim), $r->addRoute('GET', '/x', h).
func (pc *phpCollector) generic(ctx phpRouteCtx, ch *phpChain) {
	names, args, lines := chainCalls(ch)
	if len(names) == 0 {
		return
	}
	name, a := names[0], args[0]
	attrs := ctx
	switch name {
	case "group":
		rest := a
		if len(rest) > 0 {
			if s, ok := strArg(rest[0]); ok {
				attrs.prefix = joinRoute(attrs.prefix, s)
				rest = rest[1:]
			}
		}
		for _, x := range rest {
			x = stripNamedArg(x)
			if at := arrayAttrs(x); len(at) > 0 {
				if p := at["prefix"]; len(p) > 0 { // Lumen: $router->group(['prefix' => 'v1'], …)
					attrs.prefix = joinRoute(attrs.prefix, p[0])
				}
				for _, f := range at["filter"] {
					attrs.mws = append(attrs.mws, "filter:"+f)
				}
				attrs.mws = append(attrs.mws, at["middleware"]...)
			}
		}
		for k := 1; k < len(names); k++ { // Slim: ->group(…)->add(Auth::class)
			if names[k] == "add" || names[k] == "middleware" || names[k] == "addmiddleware" {
				attrs.mws = append(attrs.mws, strList(args[k])...)
			}
		}
		for _, x := range rest {
			if x = stripNamedArg(x); isClosureStart(x, 0) {
				pc.closureBody(attrs, x)
			}
		}
		return
	case "addroute", "map":
		if len(a) < 3 {
			return
		}
		methods := strList(a[:1])
		uri, ok := strArg(a[1])
		if !ok {
			return
		}
		for _, mth := range methods {
			pc.genericRoute(attrs, strings.ToUpper(mth), uri, stripNamedArg(a[2]), lines[0], names, args)
		}
		return
	}
	verb, ok := httpVerbs[name]
	if !ok || verb == "" || len(a) < 2 {
		return
	}
	uri, ok := strArg(a[0])
	if !ok {
		return
	}
	pc.genericRoute(attrs, verb, uri, stripNamedArg(a[1]), lines[0], names, args)
	for _, x := range a[2:] { // CI4: $routes->get('x', 'C::m', ['filter' => 'auth'])
		for _, f := range arrayAttrs(stripNamedArg(x))["filter"] {
			last := pc.routes[len(pc.routes)-1]
			last.mws = append(last.mws, "filter:"+f)
			last.auth = hasAuth(last.mws)
		}
	}
}

func (pc *phpCollector) genericRoute(ctx phpRouteCtx, method, uri string, action []phpTok, line int, names []string, args [][][]phpTok) {
	r := &phpRoute{Method: method, Path: joinRoute(ctx.prefix, uri), Pos: posOf(ctx.file, line), mws: append([]string{}, ctx.mws...)}
	for k := 1; k < len(names); k++ {
		if names[k] == "add" || names[k] == "middleware" || names[k] == "addmiddleware" {
			r.mws = append(r.mws, strList(args[k])...)
		}
	}
	pc.action(ctx, r, action)
	r.auth = hasAuth(r.mws)
	pc.add(r)
}

// pages: aplikasi PHP biasa — berkas yang dijalankan langsung (punya pernyataan tingkat atas dan tidak
// di-include berkas lain) menjadi satu "rute" per halaman.
func (pc *phpCollector) pages() {
	c := &phpCostCtx{m: pc.m}
	included := map[*phpFile]bool{}
	var scan func(f *phpFile, list []*phpStmt)
	scan = func(f *phpFile, list []*phpStmt) {
		for _, s := range list {
			if s.Kind == psExpr && len(s.Expr) > 0 && s.Expr[0].K == ptIdent {
				switch strings.ToLower(s.Expr[0].S) {
				case "require", "require_once", "include", "include_once":
					if g := c.resolveInclude(f, s.Expr[1:]); g != nil && g != f {
						included[g] = true
					}
				}
			}
			scan(f, s.Body)
			scan(f, s.Else)
			for _, cs := range s.Cases {
				scan(f, cs)
			}
		}
	}
	for _, f := range pc.m.files {
		scan(f, f.Stmts)
		for _, fn := range pc.m.all {
			if fn.file == f {
				scan(f, fn.Body)
			}
		}
	}
	docroot := ""
	for _, d := range []string{"public", "public_html", "htdocs", "www", "web"} {
		for _, f := range pc.m.files {
			if strings.Contains("/"+f.Rel, "/"+d+"/") {
				docroot = d
				break
			}
		}
		if docroot != "" {
			break
		}
	}
	for _, f := range pc.m.files {
		if included[f] || !executable(f.Stmts) {
			continue
		}
		rel := f.Rel
		if docroot != "" {
			i := strings.Index("/"+rel, "/"+docroot+"/")
			if i < 0 {
				continue // berkas di luar docroot tidak bisa dibuka lewat browser
			}
			rel = rel[i+len(docroot)+1:]
		}
		p := "/" + rel
		if path.Base(p) == "index.php" {
			p = strings.TrimSuffix(p, "index.php")
		}
		pc.add(&phpRoute{Method: "ANY", Path: p, Handler: f.Rel, Pos: f.Rel + ":1", entry: f})
	}
}

// executable: berkas punya pernyataan yang dijalankan (bukan hanya deklarasi/`return [...]` konfigurasi).
func executable(list []*phpStmt) bool {
	for _, s := range list {
		if s.Kind != psExpr {
			return true
		}
		if len(s.Expr) == 0 {
			continue
		}
		t := s.Expr[0]
		if t.K == ptIdent {
			switch strings.ToLower(t.S) {
			case "require", "require_once", "include", "include_once", "return", "define", "defined", "const":
				continue
			}
		}
		return true
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
