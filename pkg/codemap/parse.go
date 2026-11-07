package codemap

import (
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/vibeasik/vibeasik/pkg/models"
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

// parseSource mengekstrak simbol, impor, rute, dan ringkasan dari isi sebuah berkas.
func parseSource(lang, relPath, src string) parsed {
	var p parsed
	lines := newLineIndex(src)
	allLines := strings.Split(src, "\n")
	switch lang {
	case "go":
		p.symbols = applyRules(allLines, goRules, goExported)
		p.refs = append(goImports(allLines), goSelectors(allLines)...)
		for _, m := range matchLines(allLines, goRoute, `("/`) {
			p.routes = append(p.routes, strings.ToUpper(m[1])+" "+m[2])
		}
		if loc := goPackage.FindStringIndex(src); loc != nil {
			p.summary = docAbove(allLines, lines.line(loc[0]))
		}
	case "typescript", "javascript", "vue", "svelte":
		script := src
		offset := 0
		if lang == "vue" || lang == "svelte" {
			if m := htmlComment.FindStringSubmatch(src); m != nil {
				p.summary = cleanComment(m[1])
			}
			script, offset = vueScripts(src)
			for _, m := range []string{"defineProps", "defineEmits", "definePageMeta", "defineModel"} {
				if i := strings.Index(src, m+"("); i >= 0 || strings.Index(src, m+"<") >= 0 {
					if i < 0 {
						i = strings.Index(src, m+"<")
					}
					p.symbols = appendSym(p.symbols, models.CodeSymbol{Name: m, Kind: "macro", Line: lines.line(i)})
				}
			}
			for _, m := range matchLines(allLines, vueTag, "<") {
				p.refs = append(p.refs, "auto:"+m[1])
			}
		}
		scriptAll := strings.Split(script, "\n")
		base := lines.line(offset) - 1
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
		p.routes = append(p.routes, apiCalls(allLines)...)
		if p.summary == "" {
			p.summary = leadingComment(scriptAll)
		}
	case "python":
		p.symbols = applyRules(allLines, pyRules, func(n string) bool { return !strings.HasPrefix(n, "_") })
		for _, m := range pyImport.FindAllStringSubmatch(src, -1) {
			p.refs = append(p.refs, "py:"+m[1]+m[2])
		}
		p.routes = matchRoutes(pyRoute, src)
		if m := pyDoc.FindStringSubmatch(src); m != nil {
			p.summary = cleanComment(m[1])
		}
	case "php":
		p.symbols = applyRules(allLines, phpRules, nil)
		for _, m := range phpUse.FindAllStringSubmatch(src, -1) {
			p.refs = append(p.refs, "php:"+m[1])
		}
		p.routes = matchRoutes(phpRoute, src)
		for _, m := range matchLines(allLines, ci4Route, "$routes->") {
			p.routes = append(p.routes, strings.ToUpper(m[1])+" /"+strings.TrimPrefix(m[2], "/"))
			if m[3] != "" {
				p.refs = append(p.refs, "php:"+m[3])
			}
		}
	case "dart":
		p.symbols = applyRules(allLines, dartRules, func(n string) bool { return !strings.HasPrefix(n, "_") })
		for _, m := range dartImport.FindAllStringSubmatch(src, -1) {
			p.refs = append(p.refs, "dart:"+m[1])
		}
	case "java", "kotlin":
		p.symbols = applyRules(allLines, jvmRules, nil)
		for _, m := range jvmImport.FindAllStringSubmatch(src, -1) {
			p.refs = append(p.refs, "jvm:"+m[1])
		}
		p.routes = matchRoutes(jvmRoute, src)
	case "rust":
		p.symbols = applyRules(allLines, rustRules, nil)
	case "ruby":
		p.symbols = applyRules(allLines, rubyRules, nil)
	case "csharp":
		p.symbols = applyRules(allLines, csRules, nil)
	case "swift":
		p.symbols = applyRules(allLines, swiftRules, nil)
	case "sql":
		for _, m := range sqlTable.FindAllStringSubmatchIndex(src, -1) {
			p.symbols = appendSym(p.symbols, models.CodeSymbol{Name: src[m[2]:m[3]], Kind: "table", Line: lines.line(m[0]), Exported: true})
		}
	case "markdown":
		for _, m := range mdHeading.FindAllStringSubmatchIndex(src, -1) {
			p.symbols = appendSym(p.symbols, models.CodeSymbol{Name: strings.TrimSpace(src[m[4]:m[5]]), Kind: "heading", Line: lines.line(m[0]), Exported: true})
		}
		p.summary = markdownSummary(allLines)
	case "css":
		for _, m := range cssRule.FindAllStringSubmatchIndex(src, -1) {
			if len(p.symbols) >= 30 {
				break
			}
			p.symbols = appendSym(p.symbols, models.CodeSymbol{Name: src[m[2]:m[3]], Kind: "selector", Line: lines.line(m[0])})
		}
	}
	if p.summary == "" && lang != "markdown" {
		p.summary = leadingComment(allLines)
	}
	if p.summary == "" {
		// Dokumen simbol publik pertama sering menjelaskan tujuan berkas (mis. berkas Go tanpa package doc).
		for _, s := range p.symbols {
			if s.Exported && s.Doc != "" {
				p.summary = s.Doc
				break
			}
		}
	}
	p.summary = truncate(p.summary, maxSummaryLen)
	p.refs = uniqueStrings(p.refs)
	p.routes = uniqueStrings(p.routes)
	return p
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

// applyRules mencocokkan aturan simbol per baris (jauh lebih cepat daripada regex (?m)^ atas seluruh berkas);
// simbol pada baris yang sama hanya dicatat sekali, aturan yang lebih awal menang.
func applyRules(all []string, rules []symRule, exported func(string) bool) []models.CodeSymbol {
	var out []models.CodeSymbol
	for i, line := range all {
		if len(out) >= maxSymbolsPerFile {
			break
		}
		t := strings.TrimLeft(line, " \t")
		if t == "" || t[0] == '/' || t[0] == '*' || t[0] == '#' || t[0] == '}' || t[0] == ')' || t[0] == '<' {
			continue
		}
		for _, r := range rules {
			if !containsAny(line, r.need) {
				continue
			}
			m := r.re.FindStringSubmatchIndex(line)
			if m == nil || m[2*r.nameIdx] < 0 {
				continue
			}
			name := line[m[2*r.nameIdx]:m[2*r.nameIdx+1]]
			kind := r.kind
			if r.kindIdx > 0 && m[2*r.kindIdx] >= 0 {
				kind = line[m[2*r.kindIdx]:m[2*r.kindIdx+1]]
			}
			if kind == "struct" {
				kind = "type"
			}
			exp := r.exported
			if !exp && exported != nil {
				exp = exported(name)
			}
			out = append(out, models.CodeSymbol{Name: name, Kind: kind, Line: i + 1, Exported: exp, Doc: truncate(docAbove(all, i+1), 160)})
			break
		}
	}
	return out
}

func containsAny(s string, needles []string) bool {
	if len(needles) == 0 {
		return true
	}
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
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

// leadingComment mengambil blok komentar pertama di awal berkas (melewati shebang, lisensi, dan direktif lint).
func leadingComment(all []string) string {
	var parts []string
	started := false
	for i, raw := range all {
		if i > 60 {
			break
		}
		t := strings.TrimSpace(raw)
		isComment := strings.HasPrefix(t, "//") || strings.HasPrefix(t, "/*") || (started && strings.HasPrefix(t, "*")) ||
			(strings.HasPrefix(t, "#") && !strings.HasPrefix(t, "#!") && !strings.HasPrefix(t, "#include"))
		switch {
		case isComment:
			c := commentPrefix.ReplaceAllString(t, "")
			lc := strings.ToLower(c)
			if strings.Contains(lc, "eslint") || strings.Contains(lc, "@ts-") || strings.Contains(lc, "prettier") || strings.HasPrefix(lc, "go:") || strings.Contains(lc, "nolint") {
				continue
			}
			if strings.Contains(lc, "copyright") || strings.Contains(lc, "license") || strings.Contains(lc, "spdx") {
				parts = nil
				continue
			}
			started = true
			parts = append(parts, c)
		case started:
			return cleanComment(strings.Join(parts, " "))
		case t == "" || strings.HasPrefix(t, "#!") || strings.HasPrefix(t, "'use ") || strings.HasPrefix(t, "\"use ") || strings.HasPrefix(t, "<?php") || strings.HasPrefix(t, "package "):
			continue
		case strings.HasPrefix(t, "import ") || strings.HasPrefix(t, "from ") || strings.HasPrefix(t, "<script") || strings.HasPrefix(t, "@file:"):
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

// classify menentukan peran berkas dari path & bahasanya, agar agent bisa menyaring (mis. kind=handler).
func classify(relPath, lang string) string {
	p := "/" + strings.ToLower(relPath)
	base := path.Base(p)
	stem := strings.TrimSuffix(base, path.Ext(base))
	orig := path.Base(relPath)
	switch {
	case lang == "markdown":
		return "doc"
	case configFiles[base] != "" || strings.HasPrefix(base, "nuxt.config") || strings.HasPrefix(base, "vite.config") || strings.HasPrefix(base, "tailwind.config") || strings.HasPrefix(base, "next.config"):
		return "config"
	case strings.HasSuffix(base, "_test.go") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") || strings.HasPrefix(base, "test_") || strings.Contains(p, "/__tests__/") || strings.Contains(p, "/tests/") || strings.HasSuffix(stem, "test") && lang == "dart":
		return "test"
	case lang == "sql" || strings.Contains(p, "/migrations/") || strings.Contains(p, "/migration/"):
		return "migration"
	case lang == "css":
		return "style"
	case strings.Contains(p, "/pages/") || strings.Contains(p, "/views/") || strings.Contains(p, "/screens/") || strings.Contains(p, "/app/routes/"):
		return "page"
	case strings.Contains(p, "/layouts/"):
		return "layout"
	case strings.Contains(p, "/composables/") || strings.Contains(p, "/hooks/") || len(orig) > 4 && strings.HasPrefix(orig, "use") && unicode.IsUpper(rune(orig[3])):
		return "composable"
	case strings.Contains(p, "/stores/") || strings.Contains(p, "/store/"):
		return "store"
	case strings.Contains(p, "/components/") || strings.Contains(p, "/widgets/") || lang == "vue" || lang == "svelte" || strings.HasSuffix(base, ".tsx") || strings.HasSuffix(base, ".jsx"):
		return "component"
	case strings.Contains(p, "/models/") || strings.Contains(p, "/model/") || strings.Contains(p, "/entities/") || strings.Contains(p, "/entity/") || strings.Contains(p, "/schemas/"):
		return "model"
	case strings.Contains(p, "/types/") || strings.HasSuffix(base, ".d.ts") || stem == "types":
		return "type"
	case strings.Contains(p, "/middleware") || strings.Contains(base, "middleware") || strings.Contains(base, "guard"):
		return "middleware"
	case strings.Contains(p, "/handlers/") || strings.Contains(p, "/controllers/") || strings.Contains(base, "handler") || strings.Contains(base, "controller") || strings.Contains(p, "/server/api/"):
		return "handler"
	case strings.Contains(base, "route") || strings.Contains(base, "router") || strings.Contains(p, "/routes/"):
		return "route"
	case strings.Contains(p, "/services/") || strings.Contains(p, "/service/") || strings.Contains(base, "service") || strings.Contains(p, "/usecase"):
		return "service"
	case strings.Contains(p, "/repositor") || strings.Contains(base, "repository") || strings.Contains(base, "repo."):
		return "repository"
	case strings.Contains(p, "/cmd/") || base == "main.go" || base == "main.dart" || base == "main.py" || base == "app.vue" || base == "index.php" || base == "main.ts":
		return "entry"
	case strings.Contains(p, "/utils/") || strings.Contains(p, "/helpers/") || strings.Contains(p, "/lib/") || strings.Contains(p, "/pkg/") || strings.Contains(base, "util") || strings.Contains(base, "helper"):
		return "util"
	case strings.Contains(p, "/config/") || strings.Contains(base, "config") || strings.Contains(base, "settings"):
		return "config"
	case strings.Contains(p, "/scripts/") || lang == "shell":
		return "script"
	}
	return "module"
}
