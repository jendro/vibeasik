package routemap

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Batas agar proyek raksasa tidak membuat analisis lambat.
const (
	maxGoFiles   = 4000
	maxFileBytes = 600 * 1024
)

var skipDirs = map[string]bool{
	".git": true, "vendor": true, "node_modules": true, "testdata": true, ".vibeasik": true, "dist": true,
	".output": true, ".nuxt": true, "tmp": true, "bin": true,
}

// funcInfo = deklarasi fungsi/method lokal.
type funcInfo struct {
	Key  string // "<pkg>.Func" atau "<pkg>.Type.Method"
	Pkg  string
	Name string
	Recv string
	Pos  string
	End  int // baris penutup badan fungsi
	Decl *ast.FuncDecl
	file *fileInfo
}

// fileInfo = berkas Go ter-parse beserta peta alias impor → path paket.
type fileInfo struct {
	Rel     string
	Pkg     string // import path paket berkas
	PkgName string
	AST     *ast.File
	Imports map[string]string
}

type module struct {
	Path  string // module path dari go.mod
	Dir   string // abs
	fset  *token.FileSet
	base  string // root proyek abs (untuk path relatif)
	files []*fileInfo
	funcs map[string]*funcInfo
	// methods: nama method → key fungsi (untuk resolusi pemanggilan lewat variabel bertipe tidak diketahui)
	methods map[string][]string
	// pkgNames: import path paket lokal → nama paket
	pkgNames map[string]string
	types    *typeIndex
	// fixed: variabel tingkat paket berisi literal ("<import path>.<nama>" → jumlah elemen), dibangun sekali.
	fixed     map[string]int64
	fixedOnce sync.Once
}

var moduleRe = regexp.MustCompile(`^module\s+(\S+)`)

// findModules mencari go.mod di bawah root (maks. kedalaman 4).
func findModules(root string) []string {
	var mods []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			if rel, _ := filepath.Rel(root, p); strings.Count(rel, string(os.PathSeparator)) >= 4 {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "go.mod" {
			mods = append(mods, filepath.Dir(p))
		}
		return nil
	})
	return mods
}

func modulePath(dir string) string {
	f, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := moduleRe.FindStringSubmatch(strings.TrimSpace(sc.Text())); m != nil {
			return m[1]
		}
	}
	return ""
}

// loadModule mem-parse seluruh berkas Go non-test di modul (berhenti di go.mod bersarang).
func loadModule(dir, base string) *module {
	m := &module{Path: modulePath(dir), Dir: dir, fset: token.NewFileSet(), base: base,
		funcs: map[string]*funcInfo{}, methods: map[string][]string{}, pkgNames: map[string]string{}}
	if m.Path == "" {
		return nil
	}
	count := 0
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || count >= maxGoFiles {
			return nil
		}
		if d.IsDir() {
			if p != dir && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".") || exists(filepath.Join(p, "go.mod"))) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		if fi, err := d.Info(); err != nil || fi.Size() > maxFileBytes {
			return nil
		}
		f, err := parser.ParseFile(m.fset, p, nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil || f == nil {
			return nil
		}
		count++
		relDir, _ := filepath.Rel(dir, filepath.Dir(p))
		pkg := m.Path
		if relDir != "." {
			pkg = m.Path + "/" + filepath.ToSlash(relDir)
		}
		rel, _ := filepath.Rel(base, p)
		fi := &fileInfo{Rel: filepath.ToSlash(rel), Pkg: pkg, PkgName: f.Name.Name, AST: f, Imports: map[string]string{}}
		m.pkgNames[pkg] = f.Name.Name
		m.files = append(m.files, fi)
		return nil
	})
	for _, fi := range m.files {
		for _, im := range fi.AST.Imports {
			ip := strings.Trim(im.Path.Value, `"`)
			alias := ""
			if im.Name != nil {
				alias = im.Name.Name
			} else if n, ok := m.pkgNames[ip]; ok {
				alias = n
			} else {
				alias = defaultPkgName(ip)
			}
			if alias != "_" && alias != "." {
				fi.Imports[alias] = ip
			}
		}
		for _, d := range fi.AST.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			info := &funcInfo{Pkg: fi.Pkg, Name: fd.Name.Name, Decl: fd, file: fi, Pos: m.pos(fd.Pos()), End: m.fset.Position(fd.End()).Line}
			if fd.Recv != nil && len(fd.Recv.List) > 0 {
				info.Recv = recvTypeName(fd.Recv.List[0].Type)
				info.Key = fi.Pkg + "." + info.Recv + "." + info.Name
				m.methods[info.Name] = append(m.methods[info.Name], info.Key)
			} else {
				info.Key = fi.Pkg + "." + info.Name
			}
			m.funcs[info.Key] = info
		}
	}
	m.typeCheck()
	return m
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

// defaultPkgName: nama paket bawaan dari import path (github.com/gofiber/fiber/v2 → fiber, go-chi/chi/v5 → chi).
func defaultPkgName(ip string) string {
	parts := strings.Split(ip, "/")
	name := parts[len(parts)-1]
	if len(parts) > 1 && len(name) > 1 && name[0] == 'v' && strings.Trim(name[1:], "0123456789") == "" {
		name = parts[len(parts)-2]
	}
	name = strings.TrimPrefix(name, "go-")
	name = strings.TrimSuffix(name, ".go")
	return strings.ReplaceAll(name, "-", "")
}

func recvTypeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return recvTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return recvTypeName(t.X)
	case *ast.IndexListExpr:
		return recvTypeName(t.X)
	}
	return ""
}

// pos = "berkas/relatif.go:baris" terhadap root proyek.
func (m *module) pos(p token.Pos) string {
	if !p.IsValid() {
		return ""
	}
	tp := m.fset.Position(p)
	rel, err := filepath.Rel(m.base, tp.Filename)
	if err != nil {
		rel = tp.Filename
	}
	return filepath.ToSlash(rel) + ":" + strconv.Itoa(tp.Line)
}

// resolveFuncs mencari fungsi lokal yang dirujuk ekspresi pemanggil. Utamanya lewat go/types (presisi, termasuk
// method lewat variabel bertipe); method interface → semua implementasi lokal (CHA). Tanpa info tipe dipakai
// heuristik: Ident di paket sama, alias.Func, atau nama method yang unik di modul.
func (m *module) resolveFuncs(fi *fileInfo, fun ast.Expr) []*funcInfo {
	for {
		switch x := fun.(type) {
		case *ast.ParenExpr:
			fun = x.X
			continue
		case *ast.IndexExpr: // fungsi generik Foo[T]
			fun = x.X
			continue
		case *ast.IndexListExpr:
			fun = x.X
			continue
		}
		break
	}
	if f := m.typedFunc(fun); f != nil {
		if isInterfaceMethod(f) {
			var out []*funcInfo
			for _, k := range m.implementations(f) {
				out = append(out, m.funcs[k])
			}
			return out
		}
		if fn := m.funcs[funcKey(f)]; fn != nil {
			return []*funcInfo{fn}
		}
		return nil
	}
	switch f := fun.(type) {
	case *ast.Ident:
		if fn := m.funcs[fi.Pkg+"."+f.Name]; fn != nil {
			return []*funcInfo{fn}
		}
	case *ast.SelectorExpr:
		if x, ok := f.X.(*ast.Ident); ok {
			if ip, ok := fi.Imports[x.Name]; ok {
				if fn := m.funcs[ip+"."+f.Sel.Name]; fn != nil {
					return []*funcInfo{fn}
				}
				return nil
			}
		}
		if m.types == nil {
			if keys := m.methods[f.Sel.Name]; len(keys) == 1 {
				return []*funcInfo{m.funcs[keys[0]]}
			}
		}
	}
	return nil
}

// resolveFunc = resolveFuncs bila tepat satu target.
func (m *module) resolveFunc(fi *fileInfo, fun ast.Expr) *funcInfo {
	if fs := m.resolveFuncs(fi, fun); len(fs) == 1 {
		return fs[0]
	}
	return nil
}

// importOf: path impor untuk alias paket pada selector (fiber.New → github.com/gofiber/fiber/v2).
func importOf(fi *fileInfo, e ast.Expr) string {
	if s, ok := e.(*ast.SelectorExpr); ok {
		if x, ok := s.X.(*ast.Ident); ok {
			return fi.Imports[x.Name]
		}
	}
	return ""
}

func joinPath(prefix, p string) string {
	if p == "" {
		if prefix == "" {
			return "/"
		}
		return prefix
	}
	j := path.Join("/"+strings.TrimPrefix(prefix, "/"), p)
	if strings.HasSuffix(p, "/") && !strings.HasSuffix(j, "/") {
		j += "/"
	}
	return j
}
