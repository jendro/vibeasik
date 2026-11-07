package codemap

import (
	"path"
	"strings"
	"unicode"
)

// resolver memetakan spesifikasi impor mentah ke berkas lokal dalam peta (path relatif proyek).
// Hasil berupa path berkas, atau direktori berakhiran "/" untuk paket Go yang terdiri dari banyak berkas.
type resolver struct {
	files      map[string]bool
	dirs       map[string]bool
	nuxtComps  map[string]string // nama auto-import Nuxt (TodosTodoCard / TodoCard) → path
	composable map[string]string // useX → path
	suffix     map[string]string // akhiran path tanpa ekstensi (lowercase) → path, untuk PHP/JVM/Python
	roots      []rootInfo
}

type rootInfo struct {
	relBase  string // path komponen relatif root proyek ("" = root)
	goModule string // nama module dari go.mod
	dartPkg  string // nama paket dari pubspec.yaml
	srcDir   string // "app" bila Nuxt 4 memakai app/, selain itu ""
}

var resolveExts = []string{"", ".ts", ".js", ".vue", ".tsx", ".jsx", ".mjs", ".mts", "/index.ts", "/index.js", "/index.vue"}

func newResolver(paths []string, roots []rootInfo) *resolver {
	r := &resolver{files: map[string]bool{}, dirs: map[string]bool{}, nuxtComps: map[string]string{}, composable: map[string]string{}, suffix: map[string]string{}, roots: roots}
	for _, p := range paths {
		r.files[p] = true
		r.dirs[path.Dir(p)+"/"] = true
		base := path.Base(p)
		ext := path.Ext(base)
		stem := strings.TrimSuffix(base, ext)
		noExt := strings.ToLower(strings.TrimSuffix(p, ext))
		for i := 0; i < len(noExt); i++ {
			if i == 0 || noExt[i-1] == '/' {
				if _, dup := r.suffix[noExt[i:]]; !dup {
					r.suffix[noExt[i:]] = p
				}
			}
		}
		if len(stem) > 3 && strings.HasPrefix(stem, "use") && unicode.IsUpper(rune(stem[3])) {
			r.composable[stem] = p
		}
		if (ext == ".vue" || ext == ".tsx" || ext == ".jsx") && strings.Contains("/"+p, "/components/") {
			sub := p[strings.Index("/"+p, "/components/")+len("components/"):]
			sub = strings.TrimSuffix(sub, ext)
			name := nuxtName(sub)
			r.nuxtComps[name] = p
			if _, dup := r.nuxtComps[stem]; !dup {
				r.nuxtComps[stem] = p
			}
		}
	}
	return r
}

// nuxtName mengikuti aturan auto-import Nuxt: "todos/TodoCard" → "TodosTodoCard", "todos/todo-card" → "TodosTodoCard",
// awalan yang sudah sama dengan nama folder tidak diulang ("orchestrator/OrchestratorRunList" → "OrchestratorRunList").
func nuxtName(sub string) string {
	parts := strings.Split(sub, "/")
	var out string
	for _, part := range parts {
		if part == "index" && out != "" {
			continue
		}
		seg := pascal(part)
		if strings.HasPrefix(seg, out) && out != "" {
			out = seg
			continue
		}
		out += seg
	}
	return out
}

func pascal(s string) string {
	var b strings.Builder
	up := true
	for _, r := range s {
		if r == '-' || r == '_' || r == '.' || r == ' ' {
			up = true
			continue
		}
		if up {
			b.WriteRune(unicode.ToUpper(r))
			up = false
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (r *resolver) rootOf(p string) rootInfo {
	best := rootInfo{}
	for _, ri := range r.roots {
		if ri.relBase == "" || p == ri.relBase || strings.HasPrefix(p, ri.relBase+"/") {
			if len(ri.relBase) >= len(best.relBase) {
				best = ri
			}
		}
	}
	return best
}

func joinRel(base, rel string) string {
	if base == "" {
		return path.Clean(rel)
	}
	return path.Clean(base + "/" + rel)
}

func (r *resolver) tryFile(p string) string {
	for _, ext := range resolveExts {
		if r.files[p+ext] {
			return p + ext
		}
	}
	return ""
}

// resolve mengembalikan target lokal sebuah impor dari berkas from, atau "" bila eksternal/tidak dikenal.
func (r *resolver) resolve(from, spec string) string {
	root := r.rootOf(from)
	dir := path.Dir(from)
	switch {
	case strings.HasPrefix(spec, "auto:"):
		name := spec[5:]
		if strings.HasPrefix(name, "use") {
			return r.composable[name]
		}
		if p := r.nuxtComps[name]; p != "" {
			return p
		}
		// Komponen lazy Nuxt (LazyFoo) memakai berkas Foo.
		return r.nuxtComps[strings.TrimPrefix(name, "Lazy")]
	case strings.HasPrefix(spec, "py:"):
		mod := spec[3:]
		lead := len(mod) - len(strings.TrimLeft(mod, "."))
		rel := strings.ReplaceAll(strings.TrimLeft(mod, "."), ".", "/")
		base := root.relBase
		if lead > 0 {
			base = dir
			for i := 1; i < lead; i++ {
				base = path.Dir(base)
			}
		}
		p := joinRel(base, rel)
		if r.files[p+".py"] {
			return p + ".py"
		}
		if r.files[p+"/__init__.py"] {
			return p + "/__init__.py"
		}
		return r.suffixMatch(rel, ".py")
	case strings.HasPrefix(spec, "php:"):
		return r.suffixMatch(strings.ReplaceAll(spec[4:], "\\", "/"), ".php")
	case strings.HasPrefix(spec, "jvm:"):
		rel := strings.ReplaceAll(spec[4:], ".", "/")
		if p := r.suffixMatch(rel, ".kt"); p != "" {
			return p
		}
		return r.suffixMatch(rel, ".java")
	case strings.HasPrefix(spec, "dart:"):
		s := spec[5:]
		if strings.HasPrefix(s, "package:") {
			s = s[8:]
			pkg, rest, _ := strings.Cut(s, "/")
			if root.dartPkg != "" && pkg == root.dartPkg {
				p := joinRel(root.relBase, "lib/"+rest)
				if r.files[p] {
					return p
				}
			}
			return ""
		}
		if strings.Contains(s, ":") {
			return ""
		}
		if p := joinRel(dir, s); r.files[p] {
			return p
		}
		return ""
	case strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../"):
		return r.tryFile(joinRel(dir, spec))
	case strings.HasPrefix(spec, "~~/") || strings.HasPrefix(spec, "@@/"):
		return r.tryFile(joinRel(root.relBase, spec[3:]))
	case strings.HasPrefix(spec, "~/") || strings.HasPrefix(spec, "@/"):
		if p := r.tryFile(joinRel(joinRel(root.relBase, root.srcDir), spec[2:])); p != "" {
			return p
		}
		if p := r.tryFile(joinRel(root.relBase, "src/"+spec[2:])); p != "" {
			return p
		}
		return r.tryFile(joinRel(root.relBase, spec[2:]))
	}
	// Paket Go lokal: "<module>/internal/models" → direktori paket.
	for _, ri := range r.roots {
		if ri.goModule == "" {
			continue
		}
		if spec == ri.goModule || strings.HasPrefix(spec, ri.goModule+"/") {
			d := joinRel(ri.relBase, strings.TrimPrefix(strings.TrimPrefix(spec, ri.goModule), "/")) + "/"
			if d == "./" {
				d = ""
			}
			if r.dirs[d] {
				return d
			}
		}
	}
	return ""
}

// suffixMatch mencari berkas yang path-nya berakhiran rel (mis. PSR-4 "App/Models/User" → app/Models/User.php).
func (r *resolver) suffixMatch(rel, ext string) string {
	p := r.suffix[strings.ToLower(rel)]
	if p != "" && strings.HasSuffix(p, ext) {
		return p
	}
	// Namespace PHP/JVM sering diawali segmen yang tidak ada di disk (App\ → app/, com.example → src/main/...).
	if i := strings.Index(rel, "/"); i > 0 {
		if p := r.suffix[strings.ToLower(rel[i+1:])]; p != "" && strings.HasSuffix(p, ext) {
			return p
		}
	}
	return ""
}
