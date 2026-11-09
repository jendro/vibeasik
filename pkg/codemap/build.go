// Package codemap membangun "peta kode" per proyek: indeks per berkas (peran, simbol, impor, dipakai-oleh,
// rute, ringkasan) yang dicari dengan BM25 ala RAG, sehingga AI agent menemukan lokasi kode tanpa
// menjelajah seluruh repositori. Build bersifat inkremental (mtime + ukuran + hash isi).
package codemap

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	if err != nil && !os.IsNotExist(err) {
		log.Printf("[WARN] isDir: os.Stat(%q): %v", p, err)
	}
	return err == nil && fi.IsDir()
}

// exists melaporkan apakah path dapat diakses.
// Kontrak: os.IsNotExist adalah jalur normal (mengembalikan false tanpa noise);
// error sistem lain (izin, I/O) dicatat sebagai [WARN] dan tetap mengembalikan false
// agar pemanggil memperlakukan path sebagai tidak ada — perilaku aman-fail untuk
// pengecekan proyek-marker, bukan silent failure.
func exists(p string) bool {
	_, err := os.Stat(p)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("[WARN] exists: os.Stat(%q): %v", p, err)
	}
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

// shouldOverrideRootWithAncestor mengevaluasi apakah root proyek perlu digantikan oleh
// leluhur bersama komponen (mis. bila root berupa folder induk generik tanpa .git).
func shouldOverrideRootWithAncestor(root, anc string) bool {
	if anc == "" {
		return false
	}
	if root == "" {
		return true
	}
	if root == anc {
		return false
	}
	if !within(root, anc) {
		return false
	}
	return !exists(filepath.Join(root, ".git"))
}

// isRootValidWithoutComponents memvalidasi apakah root proyek sah digunakan saat proyek
// belum memiliki komponen terdaftar (harus memiliki project marker).
func isRootValidWithoutComponents(root string) bool {
	if root == "" {
		return false
	}
	return hasProjectMarker(root)
}

// collectComponentDirs mengumpulkan direktori absolut yang valid dari daftar komponen.
func collectComponentDirs(components []models.Component) []string {
	var compDirs []string
	for _, c := range components {
		if c.AbsolutePath != "" && isDir(c.AbsolutePath) {
			compDirs = append(compDirs, filepath.Clean(c.AbsolutePath))
		}
	}
	return compDirs
}

// resolveConfiguredRoot membersihkan path root terkonfigurasi jika valid dan berupa direktori.
func resolveConfiguredRoot(rootPath string) string {
	if rootPath == "" || rootPath == "-" || !isDir(rootPath) {
		return ""
	}
	return filepath.Clean(rootPath)
}

// EffectiveRoot menentukan root peta: root proyek, kecuali bila root itu folder umum di atas komponennya
// (mis. proyek bundel ber-root ~/Documents) — maka dipakai leluhur bersama komponen.
func EffectiveRoot(p models.Project) string {
	compDirs := collectComponentDirs(p.Components)
	root := resolveConfiguredRoot(p.RootPath)
	anc := commonAncestor(compDirs)
	if shouldOverrideRootWithAncestor(root, anc) {
		root = anc
	}
	if len(compDirs) == 0 && !isRootValidWithoutComponents(root) {
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

func isIgnoredWalkDir(p string, info os.FileInfo, r scanRoot, rootDirs map[string]bool) bool {
	if p == r.absDir {
		return false
	}
	name := info.Name()
	if rootDirs[p] || ignoredDirs[name] {
		return true
	}
	if strings.HasPrefix(name, ".") && name != ".github" {
		return true
	}
	if r.componentID == 0 && exists(filepath.Join(p, ".git")) {
		return true
	}
	return false
}

func isValidCandidateFile(p string, info os.FileInfo, seen map[string]bool) bool {
	if seen[p] || info.Size() > maxFileSize || info.Size() == 0 || generatedFile.MatchString(info.Name()) {
		return false
	}
	return true
}

func makeCandidate(r scanRoot, p string, info os.FileInfo, lang string) candidate {
	rel, _ := filepath.Rel(r.absDir, p)
	rel = filepath.ToSlash(rel)
	if r.relBase != "" {
		rel = r.relBase + "/" + rel
	}
	return candidate{root: r, abs: p, rel: rel, lang: lang, size: info.Size(), modTime: info.ModTime().Unix()}
}

func collect(roots []scanRoot) []candidate {
	seen := map[string]bool{}
	rootDirs := map[string]bool{}
	for _, r := range roots {
		rootDirs[r.absDir] = true
	}
	var out []candidate
	for _, r := range roots {
		if err := filepath.Walk(r.absDir, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if isIgnoredWalkDir(p, info, r, rootDirs) {
					return filepath.SkipDir
				}
				return nil
			}
			if len(out) >= maxFiles {
				return filepath.SkipAll
			}
			if !isValidCandidateFile(p, info, seen) {
				return nil
			}
			lang := langFor(info.Name())
			if lang == "" {
				return nil
			}
			seen[p] = true
			out = append(out, makeCandidate(r, p, info, lang))
			return nil
		}); err != nil {
			log.Printf("[WARN] collect: filepath.Walk(%q): %v", r.absDir, err)
		}
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
		if isDir(filepath.Join(r.absDir, "app")) && exists(filepath.Join(r.absDir, "nuxt.config.ts")) {
			ri.srcDir = "app"
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

func canReuseEntry(e *models.CodeMapEntry, f candidate, force bool) bool {
	return e != nil && !force && e.ModTime == f.modTime && e.SizeBytes == f.size && e.AbsPath == f.abs
}

func populateParsedEntry(e *models.CodeMapEntry, f candidate, hash string, src string, p parsed) {
	e.AbsPath, e.ComponentID, e.ComponentName = f.abs, f.root.componentID, f.root.componentName
	e.Lang, e.Kind = f.lang, classify(f.rel, f.lang)
	e.Lines = strings.Count(src, "\n") + 1
	e.SizeBytes, e.ModTime, e.ContentHash = f.size, f.modTime, hash
	e.SymbolsJSON, e.RefsJSON, e.RoutesJSON = toJSON(p.symbols), toJSON(p.refs), toJSON(nuxtPageRoute(f.rel, e.Kind, p.routes))
	e.Summary = p.summary
}

func processCandidate(projectID uint, f candidate, e *models.CodeMapEntry, force bool) (*models.CodeMapEntry, bool, bool) {
	if canReuseEntry(e, f, force) {
		return e, false, false
	}
	data, err := os.ReadFile(f.abs)
	if err != nil {
		return nil, false, false
	}
	hash := hashBytes(data)
	if e != nil && e.ContentHash == hash && !force {
		e.ModTime, e.SizeBytes, e.AbsPath = f.modTime, f.size, f.abs
		e.ComponentID, e.ComponentName = f.root.componentID, f.root.componentName
		return e, true, false
	}
	if e == nil {
		e = &models.CodeMapEntry{ProjectID: projectID, Path: f.rel}
	}
	src := string(data)
	p := parseSource(f.lang, f.rel, src)
	populateParsedEntry(e, f, hash, src, p)
	return e, true, true
}

func scanCandidates(projectID uint, files []candidate, byPath map[string]*models.CodeMapEntry, force bool, res *BuildResult) ([]*models.CodeMapEntry, map[*models.CodeMapEntry]bool, map[string]bool) {
	entries := make([]*models.CodeMapEntry, 0, len(files))
	changed := map[*models.CodeMapEntry]bool{}
	seen := map[string]bool{}
	for _, f := range files {
		seen[f.rel] = true
		e, isChanged, isParsed := processCandidate(projectID, f, byPath[f.rel], force)
		if e == nil {
			continue
		}
		if isChanged {
			changed[e] = true
		}
		if isParsed {
			res.Parsed++
		} else {
			res.Unchanged++
		}
		entries = append(entries, e)
	}
	return entries, changed, seen
}

func resolveSingleEntryImports(e *models.CodeMapEntry, rs *resolver, goSyms map[string]map[string]string) []string {
	var refs []string
	if err := json.Unmarshal([]byte(e.RefsJSON), &refs); err != nil && e.RefsJSON != "" {
		log.Printf("[WARN] resolveSingleEntryImports: unmarshal refs for %q: %v", e.Path, err)
	}
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
	return imports
}

func resolveEntryImports(entries []*models.CodeMapEntry, roots []scanRoot, changed map[*models.CodeMapEntry]bool) {
	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.Path
	}
	rs := newResolver(paths, rootInfos(roots))
	goSyms := goSymbolIndex(entries)
	for _, e := range entries {
		imports := resolveSingleEntryImports(e, rs, goSyms)
		sort.Strings(imports)
		if js := toJSON(uniqueStrings(imports)); js != e.ImportsJSON {
			e.ImportsJSON = js
			changed[e] = true
		}
	}
}

func saveBuildTransaction(db *gorm.DB, projectID uint, proj models.Project, removed []uint, changed map[*models.CodeMapEntry]bool, entries []*models.CodeMapEntry, duration time.Duration) error {
	return db.Transaction(func(tx *gorm.DB) error {
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
		return saveMeta(tx, projectID, EffectiveRoot(proj), entries, duration)
	})
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
	entries, changed, seen := scanCandidates(projectID, files, byPath, force, res)

	// Resolusi impor diulang untuk semua berkas: berkas baru bisa melengkapi impor berkas lama.
	resolveEntryImports(entries, roots, changed)

	var removed []uint
	for p, e := range byPath {
		if !seen[p] {
			removed = append(removed, e.ID)
		}
	}
	res.Removed = len(removed)

	if err := saveBuildTransaction(db, projectID, proj, removed, changed, entries, time.Since(start)); err != nil {
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
		if err := json.Unmarshal([]byte(e.SymbolsJSON), &syms); err != nil && e.SymbolsJSON != "" {
			log.Printf("[WARN] goSymbolIndex: unmarshal symbols for %q: %v", e.Path, err)
		}
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
	if err := tx.Where("project_id = ?", projectID).First(&meta).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
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
	if _, err := Build(db, projectID, false); err != nil {
		return fmt.Errorf("gagal memperbarui code map proyek %d: %w", projectID, err)
	}
	return nil
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
