// Package codemap membangun "peta kode" per proyek: indeks per berkas (peran, simbol, impor, dipakai-oleh,
// rute, ringkasan) yang dicari dengan BM25 ala RAG, sehingga AI agent menemukan lokasi kode tanpa
// menjelajah seluruh repositori. Build bersifat inkremental (mtime + ukuran + hash isi).
package codemap

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

const (
	maxFiles    = 6000
	maxFileSize = 400 * 1024
	// ParserVersion dinaikkan setiap aturan parse/klasifikasi berubah agar peta lama di-parse ulang penuh.
	ParserVersion = 2
	// StaleAfter adalah umur peta sebelum pencarian memicu build inkremental otomatis.
	StaleAfter = 30 * time.Second
)

var ignoredDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".nuxt": true, ".output": true, "dist": true, "build": true, "bin": true,
	".cache": true, "tmp": true, ".vscode": true, ".idea": true, "coverage": true, ".dart_tool": true, ".next": true, ".svelte-kit": true,
	"__pycache__": true, ".venv": true, "venv": true, "target": true, ".gradle": true, "Pods": true, ".pub-cache": true, ".turbo": true,
	".vibeasik": true, "storage": true, ".expo": true, "out": true, ".logs": true, ".pids": true,
}

var generatedFile = regexp.MustCompile(`(\.pb\.go|\.g\.dart|\.freezed\.dart|\.gen\.go|_generated\.\w+|\.min\.(js|css)|\.d\.ts\.map|-lock\.json|\.lock)$`)

// BuildResult merangkum satu build peta kode.
type BuildResult struct {
	ProjectID  uint      `json:"project_id"`
	FileCount  int       `json:"file_count"`
	Parsed     int       `json:"parsed"`
	Unchanged  int       `json:"unchanged"`
	Removed    int       `json:"removed"`
	DurationMs int64     `json:"duration_ms"`
	BuiltAt    time.Time `json:"built_at"`
}

var buildLocks sync.Map // projectID → *sync.Mutex

func lockFor(projectID uint) *sync.Mutex {
	m, _ := buildLocks.LoadOrStore(projectID, &sync.Mutex{})
	return m.(*sync.Mutex)
}

type scanRoot struct {
	componentID   uint
	componentName string
	absDir        string
	relBase       string
}

type candidate struct {
	root    scanRoot
	abs     string
	rel     string
	lang    string
	size    int64
	modTime int64
}

// projectMarkers menandai direktori yang memang root proyek (bukan folder umum seperti ~/Documents).
var projectMarkers = []string{".git", "package.json", "go.mod", "pubspec.yaml", "composer.json", "pyproject.toml", "Cargo.toml", "AI_CONTEXT.md", "CLAUDE.md", "README.md"}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func hasProjectMarker(dir string) bool {
	for _, m := range projectMarkers {
		if exists(filepath.Join(dir, m)) {
			return true
		}
	}
	return false
}

func within(parent, child string) bool {
	r, err := filepath.Rel(parent, child)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}

func commonAncestor(dirs []string) string {
	if len(dirs) == 0 {
		return ""
	}
	anc := dirs[0]
	for _, d := range dirs[1:] {
		for !within(anc, d) {
			parent := filepath.Dir(anc)
			if parent == anc {
				return ""
			}
			anc = parent
		}
	}
	return anc
}

// EffectiveRoot menentukan root peta: root proyek, kecuali bila root itu folder umum di atas komponennya
// (mis. proyek bundel ber-root ~/Documents) — maka dipakai leluhur bersama komponen.
func EffectiveRoot(p models.Project) string {
	var compDirs []string
	for _, c := range p.Components {
		if c.AbsolutePath != "" && isDir(c.AbsolutePath) {
			compDirs = append(compDirs, filepath.Clean(c.AbsolutePath))
		}
	}
	root := ""
	if p.RootPath != "" && p.RootPath != "-" && isDir(p.RootPath) {
		root = filepath.Clean(p.RootPath)
	}
	anc := commonAncestor(compDirs)
	if anc != "" && (root == "" || root != anc && within(root, anc) && !exists(filepath.Join(root, ".git"))) {
		root = anc
	}
	if root != "" && len(compDirs) == 0 && !hasProjectMarker(root) {
		return ""
	}
	return root
}

// rootsFor menentukan direktori yang dipindai: setiap komponen + root efektif (untuk dokumen & konfigurasi
// di luar komponen). Path entry selalu relatif terhadap root bila komponen berada di dalamnya.
func rootsFor(p models.Project) []scanRoot {
	root := EffectiveRoot(p)
	var out []scanRoot
	rootIsComponent := false
	for _, c := range p.Components {
		if c.AbsolutePath == "" || !isDir(c.AbsolutePath) {
			continue
		}
		abs := filepath.Clean(c.AbsolutePath)
		rel := c.Name
		if root != "" && within(root, abs) {
			r, _ := filepath.Rel(root, abs)
			rel = filepath.ToSlash(r)
			if rel == "." {
				rel = ""
				rootIsComponent = true
			}
		}
		out = append(out, scanRoot{componentID: c.ID, componentName: c.Name, absDir: abs, relBase: rel})
	}
	if root != "" && !rootIsComponent {
		out = append(out, scanRoot{absDir: root})
	}
	// Komponen terdalam dipindai dulu agar berkasnya diatribusikan ke komponen yang paling spesifik.
	sort.SliceStable(out, func(i, j int) bool { return len(out[i].absDir) > len(out[j].absDir) })
	return out
}

func langFor(name string) string {
	lower := strings.ToLower(name)
	if l := configFiles[lower]; l != "" {
		return l
	}
	return langByExt[strings.ToLower(filepath.Ext(name))]
}

func collect(roots []scanRoot) []candidate {
	seen := map[string]bool{}
	rootDirs := map[string]bool{}
	for _, r := range roots {
		rootDirs[r.absDir] = true
	}
	var out []candidate
	for _, r := range roots {
		_ = filepath.Walk(r.absDir, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				// Direktori komponen lain sudah dipindai sebagai root tersendiri; repo git bersarang di root
				// proyek adalah proyek lain, bukan bagian peta ini.
				if p != r.absDir && (rootDirs[p] || ignoredDirs[info.Name()] || strings.HasPrefix(info.Name(), ".") && info.Name() != ".github" ||
					r.componentID == 0 && exists(filepath.Join(p, ".git"))) {
					return filepath.SkipDir
				}
				return nil
			}
			if len(out) >= maxFiles {
				return filepath.SkipAll
			}
			if seen[p] || info.Size() > maxFileSize || info.Size() == 0 || generatedFile.MatchString(info.Name()) {
				return nil
			}
			lang := langFor(info.Name())
			if lang == "" {
				return nil
			}
			seen[p] = true
			rel, _ := filepath.Rel(r.absDir, p)
			rel = filepath.ToSlash(rel)
			if r.relBase != "" {
				rel = r.relBase + "/" + rel
			}
			out = append(out, candidate{root: r, abs: p, rel: rel, lang: lang, size: info.Size(), modTime: info.ModTime().Unix()})
			return nil
		})
	}
	return out
}

var (
	goModLine   = regexp.MustCompile(`(?m)^module\s+(\S+)`)
	pubspecName = regexp.MustCompile(`(?m)^name:\s*(\S+)`)
)

func rootInfos(roots []scanRoot) []rootInfo {
	var out []rootInfo
	for _, r := range roots {
		ri := rootInfo{relBase: r.relBase}
		if data, err := os.ReadFile(filepath.Join(r.absDir, "go.mod")); err == nil {
			if m := goModLine.FindSubmatch(data); m != nil {
				ri.goModule = string(m[1])
			}
		}
		if data, err := os.ReadFile(filepath.Join(r.absDir, "pubspec.yaml")); err == nil {
			if m := pubspecName.FindSubmatch(data); m != nil {
				ri.dartPkg = string(m[1])
			}
		}
		if fi, err := os.Stat(filepath.Join(r.absDir, "app")); err == nil && fi.IsDir() {
			if _, err := os.Stat(filepath.Join(r.absDir, "nuxt.config.ts")); err == nil {
				ri.srcDir = "app"
			}
		}
		out = append(out, ri)
	}
	return out
}

func hashBytes(b []byte) string {
	h := sha1.Sum(b)
	return hex.EncodeToString(h[:])
}

func toJSON(v interface{}) string {
	if v == nil {
		return "[]"
	}
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return "[]"
	}
	return string(b)
}

// Build memindai proyek dan memperbarui peta kode secara inkremental. force memaksa parse ulang semua berkas.
func Build(db *gorm.DB, projectID uint, force bool) (*BuildResult, error) {
	mu := lockFor(projectID)
	mu.Lock()
	defer mu.Unlock()
	start := time.Now()

	var proj models.Project
	if err := db.Preload("Components").First(&proj, projectID).Error; err != nil {
		return nil, fmt.Errorf("proyek %d tidak ditemukan: %w", projectID, err)
	}
	var prev models.CodeMap
	if db.Where("project_id = ?", projectID).First(&prev).Error == nil && prev.ParserVersion != ParserVersion {
		force = true
	}
	roots := rootsFor(proj)
	if len(roots) == 0 {
		return nil, fmt.Errorf("proyek %s tidak punya direktori yang bisa dipindai (root/komponen tidak ditemukan di disk)", proj.Name)
	}
	files := collect(roots)

	var existing []models.CodeMapEntry
	if err := db.Where("project_id = ?", projectID).Find(&existing).Error; err != nil {
		return nil, err
	}
	byPath := make(map[string]*models.CodeMapEntry, len(existing))
	for i := range existing {
		byPath[existing[i].Path] = &existing[i]
	}

	res := &BuildResult{ProjectID: projectID, FileCount: len(files)}
	entries := make([]*models.CodeMapEntry, 0, len(files))
	changed := map[*models.CodeMapEntry]bool{}
	seen := map[string]bool{}
	for _, f := range files {
		seen[f.rel] = true
		e := byPath[f.rel]
		if e != nil && !force && e.ModTime == f.modTime && e.SizeBytes == f.size && e.AbsPath == f.abs {
			res.Unchanged++
			entries = append(entries, e)
			continue
		}
		data, err := os.ReadFile(f.abs)
		if err != nil {
			continue
		}
		hash := hashBytes(data)
		if e == nil {
			e = &models.CodeMapEntry{ProjectID: projectID, Path: f.rel}
		} else if e.ContentHash == hash && !force {
			e.ModTime, e.SizeBytes, e.AbsPath = f.modTime, f.size, f.abs
			e.ComponentID, e.ComponentName = f.root.componentID, f.root.componentName
			changed[e] = true
			res.Unchanged++
			entries = append(entries, e)
			continue
		}
		src := string(data)
		p := parseSource(f.lang, f.rel, src)
		e.AbsPath, e.ComponentID, e.ComponentName = f.abs, f.root.componentID, f.root.componentName
		e.Lang, e.Kind = f.lang, classify(f.rel, f.lang)
		e.Lines = strings.Count(src, "\n") + 1
		e.SizeBytes, e.ModTime, e.ContentHash = f.size, f.modTime, hash
		e.SymbolsJSON, e.RefsJSON, e.RoutesJSON = toJSON(p.symbols), toJSON(p.refs), toJSON(nuxtPageRoute(f.rel, e.Kind, p.routes))
		e.Summary = p.summary
		changed[e] = true
		res.Parsed++
		entries = append(entries, e)
	}

	// Resolusi impor diulang untuk semua berkas: berkas baru bisa melengkapi impor berkas lama.
	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.Path
	}
	rs := newResolver(paths, rootInfos(roots))
	goSyms := goSymbolIndex(entries)
	for _, e := range entries {
		var refs []string
		_ = json.Unmarshal([]byte(e.RefsJSON), &refs)
		var imports []string
		sels := map[string]map[string]bool{}
		for _, spec := range refs {
			if strings.HasPrefix(spec, "sel:") {
				pkg, name, _ := strings.Cut(spec[4:], ".")
				if sels[pkg] == nil {
					sels[pkg] = map[string]bool{}
				}
				sels[pkg][name] = true
			}
		}
		for _, spec := range refs {
			if strings.HasPrefix(spec, "sel:") {
				continue
			}
			if strings.HasPrefix(spec, "goimp:") {
				imports = append(imports, resolveGoImport(rs, goSyms, e.Path, spec[6:], sels)...)
				continue
			}
			if t := rs.resolve(e.Path, spec); t != "" && t != e.Path {
				imports = append(imports, t)
			}
		}
		sort.Strings(imports)
		if js := toJSON(uniqueStrings(imports)); js != e.ImportsJSON {
			e.ImportsJSON = js
			changed[e] = true
		}
	}

	var removed []uint
	for p, e := range byPath {
		if !seen[p] {
			removed = append(removed, e.ID)
		}
	}
	res.Removed = len(removed)

	err := db.Transaction(func(tx *gorm.DB) error {
		if len(removed) > 0 {
			if err := tx.Where("id IN ?", removed).Delete(&models.CodeMapEntry{}).Error; err != nil {
				return err
			}
		}
		var fresh []*models.CodeMapEntry
		for e := range changed {
			if e.ID == 0 {
				fresh = append(fresh, e)
			} else if err := tx.Save(e).Error; err != nil {
				return err
			}
		}
		if len(fresh) > 0 {
			if err := tx.CreateInBatches(fresh, 200).Error; err != nil {
				return err
			}
		}
		return saveMeta(tx, projectID, EffectiveRoot(proj), entries, time.Since(start))
	})
	if err != nil {
		return nil, err
	}
	invalidate(projectID)
	res.DurationMs = time.Since(start).Milliseconds()
	res.BuiltAt = time.Now()
	return res, nil
}

// goSymbolIndex memetakan direktori paket Go → nama simbol tingkat paket → berkas yang mendefinisikannya.
func goSymbolIndex(entries []*models.CodeMapEntry) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, e := range entries {
		if e.Lang != "go" || e.Kind == "test" {
			continue
		}
		var syms []models.CodeSymbol
		_ = json.Unmarshal([]byte(e.SymbolsJSON), &syms)
		dir := path.Dir(e.Path) + "/"
		if out[dir] == nil {
			out[dir] = map[string]string{}
		}
		for _, s := range syms {
			if s.Kind != "method" {
				out[dir][s.Name] = e.Path
			}
		}
	}
	return out
}

var goVersionSuffix = regexp.MustCompile(`^v\d+$`)

// resolveGoImport menautkan impor paket Go lokal ke berkas yang simbolnya benar-benar dipakai (paket.Nama);
// bila tidak ada selector yang cocok (impor _ / .), seluruh direktori paket ditautkan.
func resolveGoImport(rs *resolver, goSyms map[string]map[string]string, from, spec string, sels map[string]map[string]bool) []string {
	alias, imp, _ := strings.Cut(spec, "|")
	dir := rs.resolve(from, imp)
	if dir == "" {
		return nil
	}
	name := alias
	if name == "" {
		parts := strings.Split(imp, "/")
		name = parts[len(parts)-1]
		if goVersionSuffix.MatchString(name) && len(parts) > 1 {
			name = parts[len(parts)-2]
		}
		name = strings.ReplaceAll(name, "-", "")
	}
	var out []string
	for sym := range sels[name] {
		if f := goSyms[dir][sym]; f != "" && f != from {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		return []string{dir}
	}
	return out
}

func saveMeta(tx *gorm.DB, projectID uint, root string, entries []*models.CodeMapEntry, took time.Duration) error {
	langs, kinds := map[string]int{}, map[string]int{}
	symbols, enriched := 0, 0
	for _, e := range entries {
		langs[e.Lang]++
		kinds[e.Kind]++
		symbols += strings.Count(e.SymbolsJSON, `"name":`)
		if e.AISummary != "" && e.AIHash == e.ContentHash {
			enriched++
		}
	}
	var meta models.CodeMap
	tx.Where("project_id = ?", projectID).First(&meta)
	meta.ProjectID, meta.RootPath = projectID, root
	meta.FileCount, meta.SymbolCount, meta.EnrichedCount = len(entries), symbols, enriched
	meta.LanguagesJSON, meta.KindsJSON = toJSON(langs), toJSON(kinds)
	meta.BuildMs, meta.BuiltAt, meta.ParserVersion = took.Milliseconds(), time.Now(), ParserVersion
	return tx.Save(&meta).Error
}

// nuxtPageRoute menambahkan rute halaman yang diturunkan dari path berkas (pages/projects/[id].vue → PAGE /projects/:id).
func nuxtPageRoute(rel, kind string, routes []string) []string {
	if kind != "page" || !strings.HasSuffix(rel, ".vue") {
		return routes
	}
	i := strings.Index("/"+rel, "/pages/")
	if i < 0 {
		return routes
	}
	r := strings.TrimSuffix(rel[i+len("pages/"):], ".vue")
	r = strings.TrimSuffix(strings.TrimSuffix(r, "/index"), "index")
	r = bracketParam.ReplaceAllString(r, ":$1")
	return append([]string{"PAGE /" + strings.TrimPrefix(r, "/")}, routes...)
}

var bracketParam = regexp.MustCompile(`\[\.{0,3}(\w+)\]`)

// EnsureFresh membangun peta bila belum ada atau sudah lebih tua dari maxAge (build inkremental, cepat).
func EnsureFresh(db *gorm.DB, projectID uint, maxAge time.Duration) error {
	var meta models.CodeMap
	if err := db.Where("project_id = ?", projectID).First(&meta).Error; err == nil && time.Since(meta.BuiltAt) < maxAge {
		return nil
	}
	_, err := Build(db, projectID, false)
	return err
}

// readLines membaca rentang baris [from, to] (1-based) sebuah berkas untuk cuplikan hasil pencarian.
func readLines(abs string, from, to int) (string, int) {
	f, err := os.Open(abs)
	if err != nil {
		return "", 0
	}
	defer f.Close()
	if from < 1 {
		from = 1
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	var b strings.Builder
	n := 0
	for sc.Scan() {
		n++
		if n < from {
			continue
		}
		if n > to {
			break
		}
		fmt.Fprintf(&b, "%5d | %s\n", n, sc.Text())
	}
	return b.String(), from
}
