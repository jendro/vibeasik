// Package codectx menyusun konteks kode menyeluruh untuk audit — cara senior developer menilai satu temuan:
// membaca fungsi utuh di titik temuan, fungsi yang dipanggilnya sampai layer service/repository, siapa yang
// memanggilnya, test yang mengujinya, dan rute yang menjangkaunya. Go memakai call graph bertipe (routemap,
// go/types); bahasa lain (PHP/Laravel/CodeIgniter, JS/TS/Node, Python, Java/Kotlin, Dart, C#, Ruby, …) memakai
// peta kode + resolusi nama leksikal (prioritas berkas sama → berkas yang diimpor → direktori sama → nama unik).
package codectx

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/codemap"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/routemap"
)

// Span adalah satu fungsi beserta kodenya.
type Span struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Layer string `json:"layer,omitempty"`
	From  int    `json:"from"`
	To    int    `json:"to"`
	Depth int    `json:"depth,omitempty"` // 1 = dipanggil langsung oleh fungsi target
	Code  string `json:"code,omitempty"`
}

// CallSite adalah satu tempat fungsi target dipanggil.
type CallSite struct {
	Path   string `json:"path"`
	Caller string `json:"caller"`
	Line   int    `json:"line"`
	Text   string `json:"text"`
}

// Context adalah gambaran lengkap di sekitar satu temuan (atau satu berkas pada review berkas utuh).
type Context struct {
	Target      string     `json:"target"`
	Lang        string     `json:"lang"`
	Precise     bool       `json:"precise"` // call graph bertipe (Go) vs leksikal
	Unit        *Span      `json:"unit,omitempty"`
	Callees     []Span     `json:"callees,omitempty"`
	AlsoCalls   []string   `json:"also_calls,omitempty"` // callee lain (helper kecil / di luar batas) tanpa kode
	Callers     []CallSite `json:"callers,omitempty"`
	CallerCount int        `json:"caller_count"`
	Tests       []string   `json:"tests,omitempty"`
	Routes      []string   `json:"routes,omitempty"` // non-Go: rute ComponentRoute yang handler-nya fungsi ini
	FileUsers   []string   `json:"file_users,omitempty"`
}

// Options membatasi ukuran konteks agar prompt tetap wajar.
type Options struct {
	MaxUnitLines   int
	MaxCallees     int
	MaxCalleeLines int
	MaxCallers     int
	Depth          int
}

// DefaultOptions: batas konteks untuk satu temuan.
func DefaultOptions() Options {
	return Options{MaxUnitLines: 160, MaxCallees: 8, MaxCalleeLines: 45, MaxCallers: 6, Depth: 3}
}

type def struct {
	file *codemap.FileSymbols
	sym  models.CodeSymbol
}

type builder struct {
	db      *gorm.DB
	project *models.Project
	files   []codemap.FileSymbols
	byPath  map[string]*codemap.FileSymbols
	defs    map[string][]def
	src     sources
	opt     Options
	target  *codemap.FileSymbols
}

func newBuilder(db *gorm.DB, project *models.Project, opt Options) *builder {
	if db == nil || project == nil {
		return nil
	}
	files, err := codemap.Snapshot(db, project.ID)
	if err != nil || len(files) == 0 {
		return nil
	}
	b := &builder{db: db, project: project, files: files, byPath: map[string]*codemap.FileSymbols{}, defs: map[string][]def{}, src: sources{}, opt: opt}
	for i := range b.files {
		b.byPath[b.files[i].Path] = &b.files[i]
	}
	return b
}

// lookup mencari berkas berdasarkan path relatif root, absolut, atau akhiran path (terpendek bila ganda).
func (b *builder) lookup(file string) *codemap.FileSymbols {
	file = strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(file), "\\", "/"), "./")
	if f := b.byPath[file]; f != nil {
		return f
	}
	var best *codemap.FileSymbols
	for i := range b.files {
		f := &b.files[i]
		if f.AbsPath == file || strings.HasSuffix(f.Path, "/"+file) || strings.HasSuffix(file, "/"+f.Path) {
			if best == nil || len(f.Path) < len(best.Path) {
				best = f
			}
		}
	}
	return best
}

// lookupFunc: temuan yang TargetFile-nya nama fungsi (mis. "routeManifestHandler" atau "handlers.AdminX").
func (b *builder) lookupFunc(name string) (*codemap.FileSymbols, int) {
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	for i := range b.files {
		f := &b.files[i]
		if !f.IsAppCode() {
			continue
		}
		for _, s := range f.Symbols {
			if isCallable(s.Kind) && s.Name == name {
				return f, s.Line
			}
		}
	}
	return nil, 0
}

// inScope: berkas kode aplikasi di komponen & bahasa yang sama dengan berkas target.
func (b *builder) inScope(f *codemap.FileSymbols) bool {
	t := b.target
	if f.Lang != t.Lang || !f.IsAppCode() {
		return false
	}
	if t.Component != "" {
		return f.Component == t.Component
	}
	return topDir(f.Path) == topDir(t.Path)
}

func (b *builder) testInScope(f *codemap.FileSymbols) bool {
	t := b.target
	return f.IsTest() && f.Lang == t.Lang && (t.Component == "" || f.Component == t.Component)
}

func topDir(p string) string {
	if i := strings.Index(p, "/"); i > 0 {
		return p[:i]
	}
	return ""
}

func isCallable(kind string) bool { return kind == "func" || kind == "method" }

func (b *builder) indexDefs() {
	for i := range b.files {
		f := &b.files[i]
		if !b.inScope(f) {
			continue
		}
		for _, s := range f.Symbols {
			if isCallable(s.Kind) {
				b.defs[s.Name] = append(b.defs[s.Name], def{file: f, sym: s})
			}
		}
	}
}

// spanOf menghitung rentang fungsi: dari baris simbol sampai penutup bloknya, dibatasi simbol berikutnya.
func (b *builder) spanOf(f *codemap.FileSymbols, s models.CodeSymbol) (int, int) {
	lines := b.src.lines(f.AbsPath)
	limit := len(lines)
	for _, o := range f.Symbols {
		if o.Line > s.Line && o.Line-1 < limit && (isCallable(o.Kind) || o.Kind == "type" || o.Kind == "class") {
			limit = o.Line - 1
		}
	}
	return s.Line, blockEnd(lines, s.Line, f.Lang, limit)
}

// enclosing: fungsi yang memuat baris line.
func (b *builder) enclosing(f *codemap.FileSymbols, line int) (models.CodeSymbol, int, int, bool) {
	var best models.CodeSymbol
	found := false
	for _, s := range f.Symbols {
		if isCallable(s.Kind) && s.Line <= line && (!found || s.Line > best.Line) {
			best, found = s, true
		}
	}
	if !found {
		return best, 0, 0, false
	}
	from, to := b.spanOf(f, best)
	if line > to {
		return best, 0, 0, false
	}
	return best, from, to, true
}

func (b *builder) span(f *codemap.FileSymbols, name string, from, to, mark, max, depth int) Span {
	lines := b.src.lines(f.AbsPath)
	return Span{Path: f.Path, Name: name, Layer: f.Layer, From: from, To: to, Depth: depth, Code: numbered(lines, from, to, mark, max)}
}

// callRe: pemanggilan fungsi/method — grup 1 = penerima (obj., $this->, Class::), grup 2 = nama.
var callRe = regexp.MustCompile(`(?:([A-Za-z_$][\w$]*)\s*(?:\?->|->|::|\.)\s*)?([A-Za-z_][\w]*)\s*\(`)

// notCalls: kata kunci & konstruksi bahasa yang bentuknya mirip pemanggilan.
var notCalls = map[string]bool{
	"if": true, "for": true, "foreach": true, "while": true, "switch": true, "return": true, "func": true, "function": true,
	"catch": true, "elseif": true, "match": true, "fn": true, "def": true, "print": true, "echo": true, "isset": true,
	"empty": true, "array": true, "list": true, "new": true, "typeof": true, "sizeof": true, "defer": true, "go": true,
	"make": true, "len": true, "cap": true, "append": true, "panic": true, "recover": true, "delete": true, "copy": true,
	"super": true, "this": true, "await": true, "async": true, "lambda": true, "with": true, "assert": true, "unset": true,
	"require": true, "include": true, "import": true, "select": true, "case": true, "when": true, "do": true, "else": true,
}

// commonNames: nama method umum framework/ORM; hanya dihubungkan bila didefinisikan di berkas sama/diimpor.
var commonNames = map[string]bool{
	"get": true, "set": true, "find": true, "first": true, "all": true, "where": true, "save": true, "create": true,
	"update": true, "delete": true, "json": true, "status": true, "send": true, "query": true, "exec": true, "scan": true,
	"close": true, "error": true, "string": true, "push": true, "map": true, "filter": true, "then": true, "log": true,
	"info": true, "warn": true, "debug": true, "parse": true, "format": true, "validate": true, "handle": true, "run": true,
	"init": true, "load": true, "count": true, "list": true, "show": true, "index": true, "store": true, "destroy": true,
	"render": true, "view": true, "response": true, "request": true, "input": true, "next": true, "bind": true,
}

type callRef struct {
	qual, name string
	line       int
}

func callsIn(lines []string, from, to int) []callRef {
	var out []callRef
	seen := map[string]bool{}
	for n := from + 1; n <= to && n <= len(lines); n++ { // baris signature dilewati
		l := lines[n-1]
		if t := strings.TrimSpace(l); strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "*") {
			continue
		}
		for _, m := range callRe.FindAllStringSubmatch(l, -1) {
			name := m[2]
			if notCalls[strings.ToLower(name)] || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, callRef{qual: strings.TrimPrefix(m[1], "$"), name: name, line: n})
		}
	}
	return out
}

// resolve memetakan satu pemanggilan ke definisinya (maks. 2 kandidat) dengan prioritas: berkas sama →
// berkas yang diimpor → petunjuk penerima ($this->userService->x → UserService) → direktori sama → nama unik.
func (b *builder) resolve(c callRef, from *codemap.FileSymbols, self string) []def {
	cands := b.eligibleCandidates(c, from, self)
	if len(cands) == 0 {
		return nil
	}
	qual := strings.ToLower(c.qual)
	selfRecv := isSelfReceiver(qual)
	tiers := resolutionTiers(from, c, qual, selfRecv)
	if hit := matchFirstTier(cands, tiers); hit != nil {
		return hit
	}
	if len(cands) == 1 && !commonNames[strings.ToLower(c.name)] && !selfRecv {
		return cands
	}
	return nil
}

func (b *builder) eligibleCandidates(c callRef, from *codemap.FileSymbols, self string) []def {
	var cands []def
	for _, d := range b.defs[c.name] {
		if !(d.file == from && d.sym.Name == self) {
			cands = append(cands, d)
		}
	}
	return cands
}

func isSelfReceiver(qual string) bool {
	return qual == "this" || qual == "self" || qual == "static" || qual == "parent"
}

func matchesReceiverStem(filePath, qual string) bool {
	if qual == "" || len(qual) < 3 {
		return false
	}
	stem := strings.ToLower(strings.TrimSuffix(path.Base(filePath), path.Ext(filePath)))
	return strings.Contains(strings.ReplaceAll(stem, "_", ""), strings.ReplaceAll(qual, "_", ""))
}

func matchesSameDir(candidatePath, fromPath, callName string) bool {
	return path.Dir(candidatePath) == path.Dir(fromPath) && !commonNames[strings.ToLower(callName)]
}

func resolutionTiers(from *codemap.FileSymbols, c callRef, qual string, selfRecv bool) []func(d def) bool {
	sameFile := func(d def) bool { return d.file == from }
	if selfRecv {
		return []func(d def) bool{sameFile}
	}
	return []func(d def) bool{
		sameFile,
		func(d def) bool { return imports(from, d.file.Path) },
		func(d def) bool { return matchesReceiverStem(d.file.Path, qual) },
		func(d def) bool { return matchesSameDir(d.file.Path, from.Path, c.name) },
	}
}

func matchFirstTier(cands []def, tiers []func(d def) bool) []def {
	for _, tier := range tiers {
		var hit []def
		for _, d := range cands {
			if tier(d) {
				hit = append(hit, d)
			}
		}
		if len(hit) > 0 {
			if len(hit) > 2 {
				return nil // ambigu: lebih baik tidak menebak
			}
			return hit
		}
	}
	return nil
}

func imports(from *codemap.FileSymbols, p string) bool {
	for _, imp := range from.Imports {
		if imp == p || (strings.HasSuffix(imp, "/") && strings.HasPrefix(p, imp) && !strings.Contains(strings.TrimPrefix(p, imp), "/")) {
			return true
		}
	}
	return false
}

// Build menyusun konteks menyeluruh untuk satu temuan di berkas:baris. target boleh path berkas atau nama
// fungsi. Mengembalikan nil bila peta kode belum ada atau berkas tidak dikenal.
func Build(db *gorm.DB, project *models.Project, target string, line int) *Context {
	return BuildWith(db, project, target, line, DefaultOptions())
}

// BuildWith = Build dengan batas ukuran sendiri.
func BuildWith(db *gorm.DB, project *models.Project, target string, line int, opt Options) *Context {
	b := newBuilder(db, project, opt)
	if b == nil {
		return nil
	}
	f := b.lookup(target)
	if f == nil && !strings.Contains(target, "/") {
		f, line = b.lookupFunc(target) // TargetFile berupa nama fungsi/handler
	}
	if f == nil || !f.IsAppCode() {
		return nil
	}
	b.target = f
	b.indexDefs()
	c := &Context{Target: f.Path, Lang: f.Lang}
	if f.Lang == "go" && b.buildGo(c, line) {
		return c
	}
	sym, from, to, ok := b.enclosing(f, line)
	if !ok {
		// baris di luar fungsi (deklarasi, konfigurasi rute, …): jendela di sekitar baris
		lines := b.src.lines(f.AbsPath)
		if line < 1 {
			line = 1
		}
		s := b.span(f, path.Base(f.Path), max(1, line-25), min(len(lines), line+35), line, opt.MaxUnitLines, 0)
		c.Unit = &s
		return c
	}
	s := b.span(f, sym.Name, from, to, line, opt.MaxUnitLines, 0)
	c.Unit = &s
	b.lexicalCallees(c, f, sym.Name, from, to)
	b.lexicalCallers(c, f, sym.Name, sym.Line)
	b.routesFor(c, f, sym.Name)
	return c
}

func (b *builder) lexicalCallees(c *Context, f *codemap.FileSymbols, self string, from, to int) {
	type item struct {
		f     *codemap.FileSymbols
		s     models.CodeSymbol
		depth int
	}
	seen := map[string]bool{f.Path + ":" + self: true}
	frontier := []item{{f, models.CodeSymbol{Name: self, Line: from}, 0}}
	limit := b.opt.MaxCallees * 3
	var all []Span
	defer func() { b.selectCallees(c, all) }()
	for len(frontier) > 0 && len(all) < limit {
		cur := frontier[0]
		frontier = frontier[1:]
		if cur.depth >= b.opt.Depth {
			continue
		}
		cf, ct := from, to
		if cur.depth > 0 {
			cf, ct = b.spanOf(cur.f, cur.s)
		}
		for _, call := range callsIn(b.src.lines(cur.f.AbsPath), cf, ct) {
			for _, d := range b.resolve(call, cur.f, cur.s.Name) {
				key := d.file.Path + ":" + d.sym.Name
				if seen[key] || len(all) >= limit {
					continue
				}
				seen[key] = true
				df, dt := b.spanOf(d.file, d.sym)
				all = append(all, b.span(d.file, d.sym.Name, df, dt, 0, b.opt.MaxCalleeLines, cur.depth+1))
				frontier = append(frontier, item{d.file, d.sym, cur.depth + 1})
			}
		}
	}
}

func buildLexicalRegexes(filePath, name string) (*regexp.Regexp, *regexp.Regexp) {
	re := regexp.MustCompile(`(^|[^\w$])` + regexp.QuoteMeta(name) + `\s*\(|['"]` + regexp.QuoteMeta(name) + `['"]`)
	testRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
	if stem := strings.TrimSuffix(path.Base(filePath), path.Ext(filePath)); stem != "" && stem[0] >= 'A' && stem[0] <= 'Z' {
		testRe = regexp.MustCompile(`\b(` + regexp.QuoteMeta(name) + `|` + regexp.QuoteMeta(stem) + `)\b`)
	}
	return re, testRe
}

func (b *builder) recordLexicalCaller(c *Context, o *codemap.FileSymbols, lineNum int, lineText string) {
	c.CallerCount++
	if len(c.Callers) < b.opt.MaxCallers {
		caller := "(level berkas)"
		if s, _, _, ok := b.enclosing(o, lineNum); ok {
			caller = s.Name
		}
		c.Callers = append(c.Callers, CallSite{Path: o.Path, Caller: caller, Line: lineNum, Text: clip(strings.TrimSpace(lineText), 160)})
	}
}

func (b *builder) scanFileLexicalCallers(c *Context, f, o *codemap.FileSymbols, defLine int, re, testRe *regexp.Regexp) {
	isTest := b.testInScope(o)
	if !isTest && !b.inScope(o) {
		return
	}
	lines := b.src.lines(o.AbsPath)
	match := re
	if isTest {
		match = testRe
	}
	for n, l := range lines {
		if (o == f && n+1 == defLine) || !match.MatchString(l) {
			continue
		}
		if isTest {
			if len(c.Tests) < 4 && (len(c.Tests) == 0 || c.Tests[len(c.Tests)-1] != o.Path) {
				c.Tests = append(c.Tests, o.Path)
			}
			break
		}
		b.recordLexicalCaller(c, o, n+1, l)
	}
}

// lexicalCallers mencari pemanggilan nama fungsi di berkas lingkup (kode aplikasi) & test.
func (b *builder) lexicalCallers(c *Context, f *codemap.FileSymbols, name string, defLine int) {
	if len(name) < 3 {
		return
	}
	re, testRe := buildLexicalRegexes(f.Path, name)
	for i := range b.files {
		b.scanFileLexicalCallers(c, f, &b.files[i], defLine, re, testRe)
	}
}

func (b *builder) collectGoCallees(g *routemap.Graph, fr *routemap.FuncRef) []Span {
	seen := map[string]bool{fr.Key: true}
	frontier := []routemap.FuncRef{*fr}
	limit := b.opt.MaxCallees * 3
	var all []Span
	for depth := 1; depth <= b.opt.Depth && len(frontier) > 0; depth++ {
		var next []routemap.FuncRef
		for _, cur := range frontier {
			for _, ce := range g.Callees(cur.Key) {
				if seen[ce.Key] || len(all) >= limit {
					continue
				}
				seen[ce.Key] = true
				if cf := b.lookup(ce.File); cf != nil {
					all = append(all, b.span(cf, ce.Name, ce.Line, ce.End, 0, b.opt.MaxCalleeLines, depth))
				}
				next = append(next, ce)
			}
		}
		frontier = next
	}
	return all
}

func (b *builder) resolveGoCallers(c *Context, g *routemap.Graph, fr *routemap.FuncRef, short string) {
	callers := g.Callers(fr.Key)
	c.CallerCount = len(callers)
	for _, cr := range callers {
		if len(c.Callers) >= b.opt.MaxCallers {
			break
		}
		site := CallSite{Path: cr.File, Caller: cr.Name, Line: cr.Line}
		if cf := b.lookup(cr.File); cf != nil {
			lines := b.src.lines(cf.AbsPath)
			for n := cr.Line; n <= cr.End && n <= len(lines); n++ {
				if strings.Contains(lines[n-1], short+"(") {
					site.Line, site.Text = n, clip(strings.TrimSpace(lines[n-1]), 160)
					break
				}
			}
		}
		c.Callers = append(c.Callers, site)
	}
}

func (b *builder) findGoTests(c *Context, short string) {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(short) + `\b`)
	for i := range b.files {
		o := &b.files[i]
		if len(c.Tests) >= 4 || !b.testInScope(o) {
			continue
		}
		for _, l := range b.src.lines(o.AbsPath) {
			if re.MatchString(l) {
				c.Tests = append(c.Tests, o.Path)
				break
			}
		}
	}
}

// buildGo: call graph bertipe dari routemap (presisi untuk method lewat interface/variabel bertipe).
func (b *builder) buildGo(c *Context, line int) bool {
	graphs := routemap.Analyze(codemap.EffectiveRoot(*b.project))
	g, fr := routemap.FuncAt(graphs, b.target.Path, line)
	if fr == nil {
		return false
	}
	c.Precise = true
	s := b.span(b.target, fr.Name, fr.Line, fr.End, line, b.opt.MaxUnitLines, 0)
	c.Unit = &s

	all := b.collectGoCallees(g, fr)
	b.selectCallees(c, all)

	short := fr.Name[strings.LastIndex(fr.Name, ".")+1:]
	b.resolveGoCallers(c, g, fr, short)
	b.findGoTests(c, short)

	return true
}

// routesFor (non-Go): rute dari pemindai framework (ComponentRoute: Laravel Controller@method, CI4
// Controller::method, Express/Nest, FastAPI/Django) yang handler-nya fungsi ini.
func (b *builder) routesFor(c *Context, f *codemap.FileSymbols, name string) {
	ids := make([]uint, 0, len(b.project.Components))
	for _, comp := range b.project.Components {
		ids = append(ids, comp.ID)
	}
	if len(ids) == 0 {
		return
	}
	var rows []models.ComponentRoute
	b.db.Where("component_id IN ? AND handler <> ''", ids).Find(&rows)
	class := strings.ToLower(strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)))
	for _, r := range rows {
		cls, method := splitHandler(r.Handler)
		if !strings.EqualFold(method, name) || (cls != "" && !strings.EqualFold(cls, class) && !strings.HasSuffix(class, "."+strings.ToLower(cls))) {
			continue
		}
		line := r.Method + " " + r.Path
		if r.Middleware != "" {
			line += " · middleware: " + r.Middleware
		}
		c.Routes = append(c.Routes, line+" · handler "+r.Handler)
		if len(c.Routes) >= 6 {
			break
		}
	}
}

// splitHandler: "App\Http\Controllers\UserController@index" → ("usercontroller", "index").
func splitHandler(h string) (string, string) {
	h = strings.TrimSpace(h)
	for _, sep := range []string{"@", "::", "#"} {
		if i := strings.LastIndex(h, sep); i >= 0 {
			cls := h[:i]
			if j := strings.LastIndexAny(cls, `\/.`); j >= 0 {
				cls = cls[j+1:]
			}
			return strings.ToLower(cls), h[i+len(sep):]
		}
	}
	if i := strings.LastIndex(h, "."); i >= 0 {
		return strings.ToLower(h[:i]), h[i+1:]
	}
	return "", h
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// Prompt merender konteks untuk prompt LLM audit / verifikasi / perbaikan.
func (c *Context) Prompt() string {
	if c == nil || c.Unit == nil {
		return ""
	}
	var b strings.Builder
	mode := "peta kode + resolusi nama leksikal; pemanggilan yang namanya ambigu tidak ditampilkan"
	if c.Precise {
		mode = "call graph bertipe go/types"
	}
	fmt.Fprintf(&b, "\nKonteks Kode Menyeluruh (%s). Nilai dari alur nyata — fungsi utuh, yang dipanggilnya, pemanggilnya, test, dan rute — bukan dari satu baris. Bila masalah sudah ditangani di fungsi lain dalam alur ini, itu bukan temuan.\n", mode)
	if len(c.Routes) > 0 {
		b.WriteString("Rute yang menjangkau fungsi ini:\n")
		for _, r := range c.Routes {
			b.WriteString("- " + r + "\n")
		}
	}
	layer := ""
	if c.Unit.Layer != "" {
		layer = ", layer " + c.Unit.Layer
	}
	fmt.Fprintf(&b, "Fungsi target %s (%s:%d-%d%s):\n%s", c.Unit.Name, c.Unit.Path, c.Unit.From, c.Unit.To, layer, c.Unit.Code)
	if len(c.Callees) > 0 {
		b.WriteString("\nDipanggil oleh fungsi target (kedalaman 1 = langsung):\n")
		for _, s := range c.Callees {
			fmt.Fprintf(&b, "· %s — %s:%d-%d (kedalaman %d", s.Name, s.Path, s.From, s.To, s.Depth)
			if s.Layer != "" {
				b.WriteString(", layer " + s.Layer)
			}
			b.WriteString(")\n" + s.Code)
		}
	}
	if len(c.AlsoCalls) > 0 {
		b.WriteString("Juga memanggil (kode tidak disertakan): " + strings.Join(c.AlsoCalls, ", ") + "\n")
	}
	switch {
	case c.CallerCount == 0:
		b.WriteString("\nPemanggil: tidak ditemukan pemanggil langsung di kode aplikasi (bisa handler rute, callback framework, atau kode mati).\n")
	default:
		fmt.Fprintf(&b, "\nPemanggil (%d tempat):\n", c.CallerCount)
		for _, cs := range c.Callers {
			fmt.Fprintf(&b, "- %s:%d di %s: %s\n", cs.Path, cs.Line, cs.Caller, cs.Text)
		}
	}
	if len(c.Tests) > 0 {
		b.WriteString("Diuji oleh: " + strings.Join(c.Tests, ", ") + "\n")
	} else {
		b.WriteString("Diuji oleh: belum ada test yang merujuk fungsi ini.\n")
	}
	return b.String()
}

// calleeRank: layer dalam (repository → service → model) & fungsi bermuatan lebih dulu, helper 1–4 baris belakangan.
func calleeRank(s Span) int {
	rank := map[string]int{"repository": 0, "service": 1, "model": 2, "util": 4}
	r, ok := rank[s.Layer]
	if !ok {
		r = 3
	}
	r = r*10 + s.Depth
	if s.To-s.From < 4 {
		r += 50
	}
	return r
}

func sortSpans(list []Span) {
	sort.SliceStable(list, func(i, j int) bool { return calleeRank(list[i]) < calleeRank(list[j]) })
}

// selectCallees: MaxCallees callee paling penting beserta kodenya; sisanya hanya disebut namanya.
func (b *builder) selectCallees(c *Context, all []Span) {
	sortSpans(all)
	for i, s := range all {
		if i < b.opt.MaxCallees {
			c.Callees = append(c.Callees, s)
		} else {
			c.AlsoCalls = append(c.AlsoCalls, fmt.Sprintf("%s (%s:%d)", s.Name, s.Path, s.From))
		}
	}
	// tampilkan berurutan kedalaman agar alur terbaca dari atas
	sort.SliceStable(c.Callees, func(i, j int) bool { return c.Callees[i].Depth < c.Callees[j].Depth })
}

// ImpactLine: ringkasan dampak perubahan pada fungsi ini (pemanggil & test) untuk penilai QC — kode fungsinya
// sendiri sudah ada di diff.
func (c *Context) ImpactLine() string {
	if c == nil || c.Unit == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "- %s (%s:%d-%d", c.Unit.Name, c.Unit.Path, c.Unit.From, c.Unit.To)
	if c.Unit.Layer != "" {
		b.WriteString(", layer " + c.Unit.Layer)
	}
	fmt.Fprintf(&b, "): dipanggil di %d tempat", c.CallerCount)
	for i, cs := range c.Callers {
		if i >= 4 {
			break
		}
		fmt.Fprintf(&b, "; %s:%d (%s)", cs.Path, cs.Line, cs.Caller)
	}
	if len(c.Routes) > 0 {
		b.WriteString("; rute: " + strings.Join(c.Routes, " | "))
	}
	if len(c.Tests) > 0 {
		b.WriteString("; diuji oleh " + strings.Join(c.Tests, ", "))
	} else {
		b.WriteString("; belum ada test yang merujuknya")
	}
	return b.String() + "\n"
}
