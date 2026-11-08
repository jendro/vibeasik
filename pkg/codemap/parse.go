package codemap

import (
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/util"
)

// parsed adalah hasil ekstraksi satu berkas: simbol, spesifikasi impor mentah, rute/panggilan API, dan ringkasan.
type parsed struct {
	symbols []models.CodeSymbol
	refs    []string
	routes  []string
	summary string
}

const (
	maxSymbolsPerFile = 80
	maxSummaryLen     = 280
)

// langByExt memetakan ekstensi ke bahasa yang dipahami parser.
var langByExt = map[string]string{
	".go": "go", ".ts": "typescript", ".tsx": "typescript", ".mts": "typescript", ".js": "javascript", ".jsx": "javascript",
	".mjs": "javascript", ".cjs": "javascript", ".vue": "vue", ".svelte": "svelte", ".dart": "dart", ".php": "php", ".py": "python",
	".java": "java", ".kt": "kotlin", ".rs": "rust", ".rb": "ruby", ".cs": "csharp", ".swift": "swift", ".sql": "sql", ".md": "markdown",
	".css": "css", ".scss": "css",
}

// configFiles adalah berkas konfigurasi yang ikut dipetakan walau ekstensinya bukan kode.
var configFiles = map[string]string{
	"package.json": "json", "go.mod": "gomod", "pubspec.yaml": "yaml", "composer.json": "json", "docker-compose.yml": "yaml",
	"docker-compose.yaml": "yaml", "dockerfile": "docker", "caddyfile": "caddy", "makefile": "make", "requirements.txt": "text",
	"pyproject.toml": "toml", "cargo.toml": "toml", "tsconfig.json": "json", ".env.example": "env", "build.gradle": "gradle",
	"build.gradle.kts": "gradle", "androidmanifest.xml": "xml", "launch.json": "json",
}

type symRule struct {
	re       *regexp.Regexp
	kind     string
	nameIdx  int
	kindIdx  int      // >0: jenis diambil dari grup ini
	exported bool     // true: selalu diekspor; false: ditentukan huruf besar (Go) / kata export
	need     []string // pra-saring: baris harus mengandung salah satu kata ini
}

var (
	goPackage = regexp.MustCompile(`(?m)^package\s+\w+`)
	goRules   = []symRule{
		{re: regexp.MustCompile(`(?m)^func\s+\(\s*\w*\s*\*?([\w.]+)(?:\[[^\]]*\])?\s*\)\s*(\w+)`), kind: "method", nameIdx: 2, need: []string{"func"}},
		{re: regexp.MustCompile(`(?m)^func\s+(\w+)`), kind: "func", nameIdx: 1, need: []string{"func"}},
		{re: regexp.MustCompile(`(?m)^type\s+(\w+)(?:\[[^\]]*\])?\s+(struct|interface)`), kind: "type", nameIdx: 1, kindIdx: 2, need: []string{"type"}},
		{re: regexp.MustCompile(`(?m)^type\s+(\w+)`), kind: "type", nameIdx: 1, need: []string{"type"}},
		{re: regexp.MustCompile(`(?m)^(?:const|var)\s+([A-Z]\w*)\s`), kind: "const", nameIdx: 1, need: []string{"const", "var"}},
	}
	goSelector = regexp.MustCompile(`\b([a-z_][A-Za-z0-9_]*)\.([A-Z][A-Za-z0-9_]*)`)
	goRoute    = regexp.MustCompile(`\.(Get|Post|Put|Delete|Patch|All|Head|Options)\(\s*"(/[^"]*)"`)

	jsImport = regexp.MustCompile(`(?:\bfrom|\bimport|\brequire)\s*\(?\s*['"]([^'"]+)['"]`)
	jsRules  = []symRule{
		{re: regexp.MustCompile(`(?m)^export\s+(?:default\s+)?(?:async\s+)?function\s*\*?\s*(\w+)`), kind: "func", nameIdx: 1, exported: true, need: []string{"export"}},
		{re: regexp.MustCompile(`(?m)^export\s+(?:default\s+)?(?:abstract\s+)?(class|interface|type|enum)\s+(\w+)`), nameIdx: 2, kindIdx: 1, exported: true, need: []string{"export"}},
		{re: regexp.MustCompile(`(?m)^export\s+(?:const|let|var)\s+(\w+)\s*(?::[^=]+)?=\s*(?:async\s*)?(?:\([^)]*\)|\w+)\s*(?::[^=]+)?=>`), kind: "func", nameIdx: 1, exported: true, need: []string{"export"}},
		{re: regexp.MustCompile(`(?m)^export\s+(?:const|let|var)\s+(\w+)`), kind: "const", nameIdx: 1, exported: true, need: []string{"export"}},
		{re: regexp.MustCompile(`(?m)^(?:async\s+)?function\s+(\w+)`), kind: "func", nameIdx: 1, need: []string{"function"}},
		{re: regexp.MustCompile(`(?m)^const\s+(\w+)\s*(?::[^=]+)?=\s*(?:async\s*)?(?:\([^)]*\)|\w+)\s*(?::[^=]+)?=>`), kind: "func", nameIdx: 1, need: []string{"=>"}},
		{re: regexp.MustCompile(`(?m)^(?:class|interface|enum)\s+(\w+)`), kind: "class", nameIdx: 1, need: []string{"class", "interface", "enum"}},
		{re: regexp.MustCompile(`defineStore\(\s*['"]([\w-]+)['"]`), kind: "store", nameIdx: 1, exported: true, need: []string{"defineStore"}},
	}
	jsServerRoute = regexp.MustCompile(`(?:app|router|server|api)\.(get|post|put|delete|patch|all)\(\s*['"` + "`" + `](/[^'"` + "`" + `]*)`)
	// Panggilan API dari frontend (fetch/$fetch/axios): dipakai agent untuk menelusuri FE → BE.
	apiCall     = regexp.MustCompile(`['"` + "`" + `](?:\$\{[^}]+\})?(/api/[\w/:.\-${}]*)`)
	tplVar      = regexp.MustCompile(`\$\{[^}]*\}`)
	vueTag      = regexp.MustCompile(`<([A-Z][A-Za-z0-9]+)(?:[\s/>]|$)`)
	composable  = regexp.MustCompile(`\b(use[A-Z][A-Za-z0-9]+)\s*\(`)
	htmlComment = regexp.MustCompile(`(?s)^\s*<!--(.*?)-->`)

	pyRules = []symRule{
		{re: regexp.MustCompile(`(?m)^class\s+(\w+)`), kind: "class", nameIdx: 1, need: []string{"class"}},
		{re: regexp.MustCompile(`(?m)^(?:async\s+)?def\s+(\w+)`), kind: "func", nameIdx: 1, need: []string{"def"}},
		{re: regexp.MustCompile(`(?m)^    (?:async\s+)?def\s+(\w+)`), kind: "method", nameIdx: 1, need: []string{"def"}},
	}
	pyImport = regexp.MustCompile(`(?m)^(?:from\s+([\w.]+)\s+import|import\s+([\w.]+))`)
	pyRoute  = regexp.MustCompile(`@\w+\.(get|post|put|delete|patch|route)\(\s*['"](/[^'"]*)`)
	pyDoc    = regexp.MustCompile(`(?s)^\s*(?:#[^\n]*\n\s*)*(?:"""|''')(.*?)(?:"""|''')`)

	phpRules = []symRule{
		{re: regexp.MustCompile(`(?m)^\s*(?:abstract\s+|final\s+)?(class|interface|trait|enum)\s+(\w+)`), nameIdx: 2, kindIdx: 1, need: []string{"class", "interface", "trait", "enum"}},
		{re: regexp.MustCompile(`(?m)^\s*(?:public\s+|protected\s+|private\s+)?(?:static\s+)?function\s+(\w+)`), kind: "func", nameIdx: 1, need: []string{"function"}},
	}
	phpUse   = regexp.MustCompile(`(?m)^\s*use\s+([A-Za-z0-9_\\]+)`)
	phpRoute = regexp.MustCompile(`Route::(get|post|put|delete|patch|any|resource)\(\s*['"]([^'"]+)`)
	// CodeIgniter 4: $routes->get('kontingen', 'Api\KontingenController::index') — rute + tautan ke controller.
	ci4Route = regexp.MustCompile(`\$routes->(get|post|put|delete|patch|options|match|resource|presenter|cli)\(\s*['"]([^'"]*)['"](?:\s*,\s*['"]\\?([\w\\]+)(?:::\w+)?['"])?`)

	dartRules = []symRule{
		{re: regexp.MustCompile(`(?m)^(?:abstract\s+|sealed\s+|final\s+|base\s+)*(class|mixin|enum|extension)\s+(\w+)`), nameIdx: 2, kindIdx: 1, need: []string{"class", "mixin", "enum", "extension"}},
		{re: regexp.MustCompile(`(?m)^(?:Future<[^>]*>|Stream<[^>]*>|void|[A-Z][\w<>?, ]*|int|double|bool|String)\s+(\w+)\s*\([^)]*\)\s*(?:async\s*)?\{`), kind: "func", nameIdx: 1, need: []string{"("}},
	}
	dartImport = regexp.MustCompile(`(?m)^(?:import|export|part)\s+['"]([^'"]+)['"]`)

	jvmRules = []symRule{
		{re: regexp.MustCompile(`(?m)^\s*(?:(?:public|private|protected|internal|open|abstract|data|sealed|final|static|enum|annotation)\s+)*(class|interface|object)\s+(\w+)`), nameIdx: 2, kindIdx: 1, need: []string{"class", "interface", "object"}},
		{re: regexp.MustCompile(`(?m)^\s*(?:(?:private|public|internal|protected|override|suspend|open|inline)\s+)*fun\s+(?:<[^>]+>\s*)?(?:[\w.]+\.)?(\w+)`), kind: "func", nameIdx: 1, need: []string{"fun"}},
		{re: regexp.MustCompile(`(?m)^\s+(?:public|protected|private)\s+(?:static\s+)?(?:final\s+)?(?:synchronized\s+)?[\w<>\[\], ?]+\s+(\w+)\s*\(`), kind: "method", nameIdx: 1, need: []string{"("}},
	}
	jvmImport = regexp.MustCompile(`(?m)^import\s+(?:static\s+)?([\w.]+)`)
	jvmRoute  = regexp.MustCompile(`@(Get|Post|Put|Delete|Patch|Request)Mapping\(\s*(?:value\s*=\s*|path\s*=\s*)?"([^"]*)"`)

	rustRules = []symRule{
		{re: regexp.MustCompile(`(?m)^\s*(?:pub(?:\([^)]*\))?\s+)?(?:async\s+)?fn\s+(\w+)`), kind: "func", nameIdx: 1, need: []string{"fn"}},
		{re: regexp.MustCompile(`(?m)^\s*(?:pub(?:\([^)]*\))?\s+)?(struct|enum|trait)\s+(\w+)`), nameIdx: 2, kindIdx: 1, need: []string{"struct", "enum", "trait"}},
	}
	rubyRules = []symRule{
		{re: regexp.MustCompile(`(?m)^\s*(class|module)\s+([\w:]+)`), nameIdx: 2, kindIdx: 1, need: []string{"class", "module"}},
		{re: regexp.MustCompile(`(?m)^\s*def\s+(?:self\.)?(\w+[?!]?)`), kind: "func", nameIdx: 1, need: []string{"def"}},
	}
	csRules = []symRule{
		{re: regexp.MustCompile(`(?m)^\s*(?:(?:public|private|protected|internal|static|sealed|abstract|partial)\s+)*(class|interface|record|struct|enum)\s+(\w+)`), nameIdx: 2, kindIdx: 1, need: []string{"class", "interface", "record", "struct", "enum"}},
		{re: regexp.MustCompile(`(?m)^\s+(?:public|protected|private|internal)\s+(?:static\s+|async\s+|override\s+|virtual\s+)*[\w<>\[\], ?]+\s+(\w+)\s*\(`), kind: "method", nameIdx: 1, need: []string{"("}},
	}
	swiftRules = []symRule{
		{re: regexp.MustCompile(`(?m)^\s*(?:(?:public|private|internal|open|final)\s+)*(class|struct|protocol|enum|extension)\s+(\w+)`), nameIdx: 2, kindIdx: 1, need: []string{"class", "struct", "protocol", "enum", "extension"}},
		{re: regexp.MustCompile(`(?m)^\s*(?:(?:public|private|internal|open|static|override|@\w+)\s+)*func\s+(\w+)`), kind: "func", nameIdx: 1, need: []string{"func"}},
	}
	sqlTable  = regexp.MustCompile("(?i)create\\s+table\\s+(?:if\\s+not\\s+exists\\s+)?[`\"]?(\\w+)")
	mdHeading = regexp.MustCompile(`(?m)^(#{1,3})\s+(.+?)\s*#*\s*$`)
	cssRule   = regexp.MustCompile(`(?m)^([.#][\w-]+)[^{\n]*\{`)
)

// lineIndex mengubah offset byte menjadi nomor baris (mulai 1).
type lineIndex []int

func newLineIndex(src string) lineIndex {
	idx := lineIndex{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			idx = append(idx, i+1)
		}
	}
	return idx
}

func (l lineIndex) line(offset int) int {
	return sort.Search(len(l), func(i int) bool { return l[i] > offset })
}

// parseCtx membawa isi berkas dan indeks barisnya ke parser per bahasa.
type parseCtx struct {
	lang  string
	src   string
	lines lineIndex
	all   []string
}

func parseGo(p *parsed, c *parseCtx) {
	p.symbols = applyRules(c.all, goRules, goExported)
	p.refs = append(goImports(c.all), goSelectors(c.all)...)
	for _, m := range matchLines(c.all, goRoute, `("/`) {
		p.routes = append(p.routes, strings.ToUpper(m[1])+" "+m[2])
	}
	if loc := goPackage.FindStringIndex(c.src); loc != nil {
		p.summary = docAbove(c.all, c.lines.line(loc[0]))
	}
}

// vueMacroIndex mencari pemakaian makro compiler Vue (name( atau name<); -1 bila tidak ada.
func vueMacroIndex(src, name string) int {
	if i := strings.Index(src, name+"("); i >= 0 {
		return i
	}
	return strings.Index(src, name+"<")
}

// parseVueHeader mengisi ringkasan komentar, makro, dan tag komponen SFC lalu mengembalikan isi <script> + offsetnya.
func parseVueHeader(p *parsed, c *parseCtx) (string, int) {
	if m := htmlComment.FindStringSubmatch(c.src); m != nil {
		p.summary = cleanComment(m[1])
	}
	script, offset := vueScripts(c.src)
	for _, m := range []string{"defineProps", "defineEmits", "definePageMeta", "defineModel"} {
		if i := vueMacroIndex(c.src, m); i >= 0 {
			p.symbols = appendSym(p.symbols, models.CodeSymbol{Name: m, Kind: "macro", Line: c.lines.line(i)})
		}
	}
	for _, m := range matchLines(c.all, vueTag, "<") {
		p.refs = append(p.refs, "auto:"+m[1])
	}
	return script, offset
}

func parseJSLike(p *parsed, c *parseCtx) {
	script := c.src
	offset := 0
	if c.lang == "vue" || c.lang == "svelte" {
		script, offset = parseVueHeader(p, c)
	}
	scriptAll := strings.Split(script, "\n")
	base := c.lines.line(offset) - 1
	for _, s := range applyRules(scriptAll, jsRules, nil) {
		s.Line += base
		p.symbols = appendSym(p.symbols, s)
	}
	for _, m := range matchLines(scriptAll, jsImport, "import", "require", "from") {
		p.refs = append(p.refs, m[1])
	}
	for _, m := range matchLines(scriptAll, composable, "use") {
		p.refs = append(p.refs, "auto:"+m[1])
	}
	for _, m := range matchLines(scriptAll, jsServerRoute, ".get(", ".post(", ".put(", ".delete(", ".patch(", ".all(") {
		p.routes = append(p.routes, strings.ToUpper(m[1])+" "+m[2])
	}
	p.routes = append(p.routes, apiCalls(c.all)...)
	if p.summary == "" {
		p.summary = leadingComment(scriptAll)
	}
}

func notUnderscore(n string) bool { return !strings.HasPrefix(n, "_") }

func parsePython(p *parsed, c *parseCtx) {
	p.symbols = applyRules(c.all, pyRules, notUnderscore)
	for _, m := range pyImport.FindAllStringSubmatch(c.src, -1) {
		p.refs = append(p.refs, "py:"+m[1]+m[2])
	}
	p.routes = matchRoutes(pyRoute, c.src)
	if m := pyDoc.FindStringSubmatch(c.src); m != nil {
		p.summary = cleanComment(m[1])
	}
}

func parsePHP(p *parsed, c *parseCtx) {
	p.symbols = applyRules(c.all, phpRules, nil)
	for _, m := range phpUse.FindAllStringSubmatch(c.src, -1) {
		p.refs = append(p.refs, "php:"+m[1])
	}
	p.routes = matchRoutes(phpRoute, c.src)
	for _, m := range matchLines(c.all, ci4Route, "$routes->") {
		p.routes = append(p.routes, strings.ToUpper(m[1])+" /"+strings.TrimPrefix(m[2], "/"))
		if m[3] != "" {
			p.refs = append(p.refs, "php:"+m[3])
		}
	}
}

func parseDart(p *parsed, c *parseCtx) {
	p.symbols = applyRules(c.all, dartRules, notUnderscore)
	for _, m := range dartImport.FindAllStringSubmatch(c.src, -1) {
		p.refs = append(p.refs, "dart:"+m[1])
	}
}

func parseJVM(p *parsed, c *parseCtx) {
	p.symbols = applyRules(c.all, jvmRules, nil)
	for _, m := range jvmImport.FindAllStringSubmatch(c.src, -1) {
		p.refs = append(p.refs, "jvm:"+m[1])
	}
	p.routes = matchRoutes(jvmRoute, c.src)
}

// rulesOnly membuat parser yang hanya menerapkan aturan simbol.
func rulesOnly(rules []symRule) func(p *parsed, c *parseCtx) {
	return func(p *parsed, c *parseCtx) { p.symbols = applyRules(c.all, rules, nil) }
}

func parseSQL(p *parsed, c *parseCtx) {
	for _, m := range sqlTable.FindAllStringSubmatchIndex(c.src, -1) {
		p.symbols = appendSym(p.symbols, models.CodeSymbol{Name: c.src[m[2]:m[3]], Kind: "table", Line: c.lines.line(m[0]), Exported: true})
	}
}

func parseMarkdown(p *parsed, c *parseCtx) {
	for _, m := range mdHeading.FindAllStringSubmatchIndex(c.src, -1) {
		p.symbols = appendSym(p.symbols, models.CodeSymbol{Name: strings.TrimSpace(c.src[m[4]:m[5]]), Kind: "heading", Line: c.lines.line(m[0]), Exported: true})
	}
	p.summary = markdownSummary(c.all)
}

func parseCSS(p *parsed, c *parseCtx) {
	for _, m := range cssRule.FindAllStringSubmatchIndex(c.src, -1) {
		if len(p.symbols) >= 30 {
			break
		}
		p.symbols = appendSym(p.symbols, models.CodeSymbol{Name: c.src[m[2]:m[3]], Kind: "selector", Line: c.lines.line(m[0])})
	}
}

// langParsers memetakan bahasa ke parser khususnya.
var langParsers = map[string]func(p *parsed, c *parseCtx){
	"go": parseGo, "typescript": parseJSLike, "javascript": parseJSLike, "vue": parseJSLike, "svelte": parseJSLike,
	"python": parsePython, "php": parsePHP, "dart": parseDart, "java": parseJVM, "kotlin": parseJVM,
	"rust": rulesOnly(rustRules), "ruby": rulesOnly(rubyRules), "csharp": rulesOnly(csRules), "swift": rulesOnly(swiftRules),
	"sql": parseSQL, "markdown": parseMarkdown, "css": parseCSS,
}

// parseSource mengekstrak simbol, impor, rute, dan ringkasan dari isi sebuah berkas.
func parseSource(lang, relPath, src string) parsed {
	var p parsed
	c := &parseCtx{lang: lang, src: src, lines: newLineIndex(src), all: strings.Split(src, "\n")}
	if parse := langParsers[lang]; parse != nil {
		parse(&p, c)
	}
	if p.summary == "" && lang != "markdown" {
		p.summary = leadingComment(c.all)
	}
	if p.summary == "" {
		// Dokumen simbol publik pertama sering menjelaskan tujuan berkas (mis. berkas Go tanpa package doc).
		p.summary = firstExportedDoc(p.symbols)
	}
	p.summary = truncate(p.summary, maxSummaryLen)
	p.refs = uniqueStrings(p.refs)
	p.routes = uniqueStrings(p.routes)
	return p
}

// firstExportedDoc mengembalikan dokumen simbol publik pertama yang punya dokumen.
func firstExportedDoc(symbols []models.CodeSymbol) string {
	for _, s := range symbols {
		if s.Exported && s.Doc != "" {
			return s.Doc
		}
	}
	return ""
}

// goImports membaca deklarasi import Go (tunggal maupun blok) per baris.
func goImports(all []string) []string {
	var out []string
	inBlock := false
	for _, line := range all {
		t := strings.TrimSpace(line)
		switch {
		case inBlock:
			if strings.HasPrefix(t, ")") {
				inBlock = false
				continue
			}
			out = appendGoImport(out, t)
		case strings.HasPrefix(t, "import ("):
			inBlock = true
		case strings.HasPrefix(t, "import "):
			out = appendGoImport(out, strings.TrimSpace(t[len("import "):]))
		case strings.HasPrefix(t, "func ") || strings.HasPrefix(t, "type ") || strings.HasPrefix(t, "var ") || strings.HasPrefix(t, "const "):
			return out
		}
	}
	return out
}

// appendGoImport menyimpan impor sebagai "goimp:<alias>|<path>" (alias kosong = nama paket bawaan).
func appendGoImport(out []string, spec string) []string {
	q := quotedPart(spec)
	if q == "" {
		return out
	}
	alias := strings.TrimSpace(spec[:strings.IndexByte(spec, '"')])
	return append(out, "goimp:"+alias+"|"+q)
}

// goSelectors mengumpulkan pemakaian "paket.Nama" agar dependensi antarpaket bisa ditautkan per berkas.
func goSelectors(all []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range matchLines(all, goSelector, ".") {
		k := "sel:" + m[1] + "." + m[2]
		if !seen[k] && len(out) < 600 {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

func quotedPart(s string) string {
	i := strings.IndexByte(s, '"')
	if i < 0 {
		return ""
	}
	j := strings.IndexByte(s[i+1:], '"')
	if j < 0 {
		return ""
	}
	return s[i+1 : i+1+j]
}

// vueScripts menggabungkan isi semua blok <script> SFC (tanpa regex lazy yang lambat di RE2) dan
// mengembalikan offset blok pertama untuk menghitung nomor baris.
func vueScripts(src string) (string, int) {
	var b strings.Builder
	first := 0
	rest, base := src, 0
	for {
		i := strings.Index(rest, "<script")
		if i < 0 {
			break
		}
		gt := strings.IndexByte(rest[i:], '>')
		if gt < 0 {
			break
		}
		start := i + gt + 1
		end := strings.Index(rest[start:], "</script>")
		if end < 0 {
			end = len(rest) - start
		}
		if first == 0 {
			first = base + start
		}
		b.WriteString(rest[start : start+end])
		b.WriteString("\n")
		base += start + end
		rest = rest[start+end:]
	}
	return b.String(), first
}

func goExported(name string) bool { return name != "" && unicode.IsUpper(rune(name[0])) }

// skipRuleLine: baris kosong atau berawalan komentar/penutup tidak diperiksa aturan simbol.
func skipRuleLine(t string) bool {
	return t == "" || strings.IndexByte("/*#})<", t[0]) >= 0
}

// ruleSymbol mencocokkan satu aturan pada satu baris; ok=false bila tak cocok.
func ruleSymbol(r symRule, line string, exported func(string) bool) (name, kind string, exp, ok bool) {
	if !util.ContainsAny(line, r.need...) {
		return
	}
	m := r.re.FindStringSubmatchIndex(line)
	if m == nil || m[2*r.nameIdx] < 0 {
		return
	}
	name = line[m[2*r.nameIdx]:m[2*r.nameIdx+1]]
	kind = r.kind
	if r.kindIdx > 0 && m[2*r.kindIdx] >= 0 {
		kind = line[m[2*r.kindIdx]:m[2*r.kindIdx+1]]
	}
	if kind == "struct" {
		kind = "type"
	}
	exp = r.exported
	if !exp && exported != nil {
		exp = exported(name)
	}
	return name, kind, exp, true
}

// applyRules mencocokkan aturan simbol per baris (jauh lebih cepat daripada regex (?m)^ atas seluruh berkas);
// simbol pada baris yang sama hanya dicatat sekali, aturan yang lebih awal menang.
func applyRules(all []string, rules []symRule, exported func(string) bool) []models.CodeSymbol {
	var out []models.CodeSymbol
	for i, line := range all {
		if len(out) >= maxSymbolsPerFile {
			break
		}
		if skipRuleLine(strings.TrimLeft(line, " \t")) {
			continue
		}
		for _, r := range rules {
			if name, kind, exp, ok := ruleSymbol(r, line, exported); ok {
				out = append(out, models.CodeSymbol{Name: name, Kind: kind, Line: i + 1, Exported: exp, Doc: truncate(docAbove(all, i+1), 160)})
				break
			}
		}
	}
	return out
}

// matchLines menjalankan re per baris yang mengandung salah satu needle (pra-saring murah).
func matchLines(all []string, re *regexp.Regexp, needles ...string) [][]string {
	var out [][]string
	for _, line := range all {
		for _, n := range needles {
			if strings.Contains(line, n) {
				out = append(out, re.FindAllStringSubmatch(line, -1)...)
				break
			}
		}
	}
	return out
}

func appendSym(list []models.CodeSymbol, s models.CodeSymbol) []models.CodeSymbol {
	if len(list) >= maxSymbolsPerFile {
		return list
	}
	return append(list, s)
}

func matchRoutes(re *regexp.Regexp, src string) []string {
	var out []string
	for _, m := range re.FindAllStringSubmatch(src, -1) {
		out = append(out, strings.ToUpper(m[1])+" "+m[2])
	}
	return out
}

// apiCalls mencatat endpoint /api/... yang dipanggil berkas (variabel template dinormalkan menjadi :param).
func apiCalls(all []string) []string {
	var out []string
	for _, m := range matchLines(all, apiCall, "/api/") {
		p := tplVar.ReplaceAllString(m[1], ":param")
		p = strings.TrimRight(p, "/?")
		if len(p) > 5 {
			out = append(out, "CALL "+p)
		}
	}
	return out
}

var commentPrefix = regexp.MustCompile(`^\s*(?://+|#+|/\*+|\*+/?|--|;+)\s?`)

// docAbove mengambil blok komentar tepat di atas baris (1-based) ln.
func docAbove(all []string, ln int) string {
	var parts []string
	for i := ln - 2; i >= 0 && i >= ln-15; i-- {
		t := strings.TrimSpace(all[i])
		if t == "" || !(strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "*") || strings.HasPrefix(t, "/*")) {
			break
		}
		if strings.HasPrefix(t, "#[") || strings.HasPrefix(t, "#!") || strings.HasPrefix(t, "#include") {
			break
		}
		parts = append([]string{commentPrefix.ReplaceAllString(t, "")}, parts...)
	}
	return cleanComment(strings.Join(parts, " "))
}

// isLeadingComment: baris komentar (awalan * hanya dihitung setelah komentar dimulai).
func isLeadingComment(t string, started bool) bool {
	if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "/*") || (started && strings.HasPrefix(t, "*")) {
		return true
	}
	return strings.HasPrefix(t, "#") && !strings.HasPrefix(t, "#!") && !strings.HasPrefix(t, "#include")
}

// lintDirective: komentar direktif lint/generator yang dilewati.
func lintDirective(lc string) bool {
	return containsSub(lc, "eslint", "@ts-", "prettier", "nolint") || strings.HasPrefix(lc, "go:")
}

// preambleLine: baris pembuka berkas (kosong, shebang, 'use strict', <?php, package) yang dilewati.
func preambleLine(t string) bool {
	return t == "" || hasAnyPrefix(t, "#!", "'use ", "\"use ", "<?php", "package ")
}

// importLine: baris impor/skrip; komentar boleh berada setelahnya.
func importLine(t string) bool {
	return hasAnyPrefix(t, "import ", "from ", "<script", "@file:")
}

func containsSub(s string, subs ...string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, x := range prefixes {
		if strings.HasPrefix(s, x) {
			return true
		}
	}
	return false
}

// leadingComment mengambil blok komentar pertama di awal berkas (melewati shebang, lisensi, dan direktif lint).
func leadingComment(all []string) string {
	var parts []string
	started := false
	for i, raw := range all {
		if i > 60 {
			break
		}
		t := strings.TrimSpace(raw)
		switch {
		case isLeadingComment(t, started):
			c := commentPrefix.ReplaceAllString(t, "")
			lc := strings.ToLower(c)
			if lintDirective(lc) {
				continue
			}
			if containsSub(lc, "copyright", "license", "spdx") {
				parts = nil
				continue
			}
			started = true
			parts = append(parts, c)
		case started:
			return cleanComment(strings.Join(parts, " "))
		case preambleLine(t), importLine(t):
			// komentar boleh berada setelah impor; lanjut mencari
			continue
		default:
			return ""
		}
	}
	return cleanComment(strings.Join(parts, " "))
}

func markdownSummary(all []string) string {
	var title, para string
	for _, raw := range all {
		t := strings.TrimSpace(raw)
		switch {
		case t == "":
			if para != "" {
				return strings.TrimSpace(title + " — " + para)
			}
		case strings.HasPrefix(t, "#"):
			if title == "" {
				title = strings.TrimSpace(strings.TrimLeft(t, "#"))
			} else if para != "" {
				return strings.TrimSpace(title + " — " + para)
			}
		case strings.HasPrefix(t, "```") || strings.HasPrefix(t, "|") || strings.HasPrefix(t, "<") || strings.HasPrefix(t, "!["):
			continue
		default:
			para = strings.TrimSpace(para + " " + strings.TrimLeft(t, "-*> "))
			if len(para) > maxSummaryLen {
				return strings.TrimSpace(title + " — " + para)
			}
		}
	}
	if para == "" {
		return title
	}
	return strings.TrimSpace(title + " — " + para)
}

var spaces = regexp.MustCompile(`\s+`)

func cleanComment(s string) string {
	s = strings.ReplaceAll(s, "*/", "")
	s = strings.ReplaceAll(s, "/**", "")
	return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n-1])) + "…"
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := in[:0]
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// clsCtx: bentuk path yang dipakai aturan klasifikasi berkas.
type clsCtx struct {
	p    string // "/" + path huruf kecil
	base string
	stem string
	orig string // nama berkas asli (huruf asli)
	lang string
}

// clsRule cocok bila SALAH SATU kondisi terpenuhi: path memuat pathHas, nama dasar memuat baseHas /
// berawalan basePrefix / berakhiran baseSuffix / sama dengan baseIs, bahasa ada di langs, atau extra true.
type clsRule struct {
	kind       string
	pathHas    []string
	baseHas    []string
	basePrefix []string
	baseSuffix []string
	baseIs     []string
	langs      []string
	extra      func(c *clsCtx) bool
}

func (r clsRule) match(c *clsCtx) bool {
	if containsSub(c.p, r.pathHas...) || containsSub(c.base, r.baseHas...) ||
		hasAnyPrefix(c.base, r.basePrefix...) || hasAnySuffix(c.base, r.baseSuffix...) {
		return true
	}
	for _, x := range r.baseIs {
		if c.base == x {
			return true
		}
	}
	for _, x := range r.langs {
		if c.lang == x {
			return true
		}
	}
	return r.extra != nil && r.extra(c)
}

func hasAnySuffix(s string, suffixes ...string) bool {
	for _, x := range suffixes {
		if strings.HasSuffix(s, x) {
			return true
		}
	}
	return false
}

func isConfigBase(c *clsCtx) bool { return configFiles[c.base] != "" }

func isDartTest(c *clsCtx) bool { return strings.HasSuffix(c.stem, "test") && c.lang == "dart" }

func isUseHook(c *clsCtx) bool {
	return len(c.orig) > 4 && strings.HasPrefix(c.orig, "use") && unicode.IsUpper(rune(c.orig[3]))
}

func isTypesStem(c *clsCtx) bool { return c.stem == "types" }

// classifyRules: urut prioritas; aturan pertama yang cocok menentukan peran berkas.
var classifyRules = []clsRule{
	{kind: "doc", langs: []string{"markdown"}},
	{kind: "config", basePrefix: []string{"nuxt.config", "vite.config", "tailwind.config", "next.config"}, extra: isConfigBase},
	{kind: "test", baseSuffix: []string{"_test.go"}, baseHas: []string{".test.", ".spec."}, basePrefix: []string{"test_"},
		pathHas: []string{"/__tests__/", "/tests/"}, extra: isDartTest},
	{kind: "migration", langs: []string{"sql"}, pathHas: []string{"/migrations/", "/migration/"}},
	{kind: "style", langs: []string{"css"}},
	{kind: "page", pathHas: []string{"/pages/", "/views/", "/screens/", "/app/routes/"}},
	{kind: "layout", pathHas: []string{"/layouts/"}},
	{kind: "composable", pathHas: []string{"/composables/", "/hooks/"}, extra: isUseHook},
	{kind: "store", pathHas: []string{"/stores/", "/store/"}},
	{kind: "component", pathHas: []string{"/components/", "/widgets/"}, langs: []string{"vue", "svelte"}, baseSuffix: []string{".tsx", ".jsx"}},
	{kind: "model", pathHas: []string{"/models/", "/model/", "/entities/", "/entity/", "/schemas/"}},
	{kind: "type", pathHas: []string{"/types/"}, baseSuffix: []string{".d.ts"}, extra: isTypesStem},
	{kind: "middleware", pathHas: []string{"/middleware"}, baseHas: []string{"middleware", "guard"}},
	{kind: "handler", pathHas: []string{"/handlers/", "/controllers/", "/server/api/"}, baseHas: []string{"handler", "controller"}},
	{kind: "route", baseHas: []string{"route", "router"}, pathHas: []string{"/routes/"}},
	{kind: "service", pathHas: []string{"/services/", "/service/", "/usecase"}, baseHas: []string{"service"}},
	{kind: "repository", pathHas: []string{"/repositor"}, baseHas: []string{"repository", "repo."}},
	{kind: "entry", pathHas: []string{"/cmd/"}, baseIs: []string{"main.go", "main.dart", "main.py", "app.vue", "index.php", "main.ts"}},
	{kind: "util", pathHas: []string{"/utils/", "/helpers/", "/lib/", "/pkg/"}, baseHas: []string{"util", "helper"}},
	{kind: "config", pathHas: []string{"/config/"}, baseHas: []string{"config", "settings"}},
	{kind: "script", pathHas: []string{"/scripts/"}, langs: []string{"shell"}},
}

// classify menentukan peran berkas dari path & bahasanya, agar agent bisa menyaring (mis. kind=handler).
func classify(relPath, lang string) string {
	p := "/" + strings.ToLower(relPath)
	base := path.Base(p)
	c := &clsCtx{p: p, base: base, stem: strings.TrimSuffix(base, path.Ext(base)), orig: path.Base(relPath), lang: lang}
	for _, r := range classifyRules {
		if r.match(c) {
			return r.kind
		}
	}
	return "module"
}
