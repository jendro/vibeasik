package codemap

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/internal/models"
	"github.com/vibeasik/vibeasik/internal/routemap"
)

// hotspotLines adalah batas baris berkas yang mulai mahal dibaca utuh oleh AI agent (konteks/token).
const hotspotLines = 400

// OverviewFile adalah ringkasan satu berkas di peta.
type OverviewFile struct {
	Path    string   `json:"path"`
	Kind    string   `json:"kind"`
	Lang    string   `json:"lang"`
	Lines   int      `json:"lines"`
	Summary string   `json:"summary,omitempty"`
	Symbols []string `json:"symbols,omitempty"`
	UsedBy  int      `json:"used_by"`
	Routes  int      `json:"routes,omitempty"`
}

// OverviewDir mengelompokkan berkas per direktori.
type OverviewDir struct {
	Path     string         `json:"path"`
	Files    int            `json:"files"`
	Lines    int            `json:"lines"`
	Kinds    map[string]int `json:"kinds"`
	KeyFiles []string       `json:"key_files"` // berkas yang paling banyak dipakai di direktori ini
	Entries  []OverviewFile `json:"entries,omitempty"`
}

// OverviewComponent adalah satu aplikasi/komponen di peta.
type OverviewComponent struct {
	ID    uint           `json:"id"`
	Name  string         `json:"name"`
	Path  string         `json:"path"`
	Files int            `json:"files"`
	Lines int            `json:"lines"`
	Kinds map[string]int `json:"kinds"`
	Dirs  []OverviewDir  `json:"dirs"`
}

// RankedFile dipakai untuk daftar modul inti, hotspot, dan titik masuk.
type RankedFile struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Lines  int    `json:"lines"`
	UsedBy int    `json:"used_by"`
	Note   string `json:"note,omitempty"`
}

// Overview adalah "peta" tingkat atas sebuah proyek untuk agent: struktur, modul inti, hotspot, rute.
type Overview struct {
	ProjectID     uint                `json:"project_id"`
	ProjectName   string              `json:"project_name"`
	RootPath      string              `json:"root_path"`
	BuiltAt       time.Time           `json:"built_at"`
	ExportedAt    *time.Time          `json:"exported_at,omitempty"`
	FileCount     int                 `json:"file_count"`
	SymbolCount   int                 `json:"symbol_count"`
	EnrichedCount int                 `json:"enriched_count"`
	TotalLines    int                 `json:"total_lines"`
	RouteCount    int                 `json:"route_count"`
	Languages     map[string]int      `json:"languages"`
	Kinds         map[string]int      `json:"kinds"`
	Components    []OverviewComponent `json:"components"`
	Core          []RankedFile        `json:"core"`
	Hotspots      []RankedFile        `json:"hotspots"`
	EntryPoints   []RankedFile        `json:"entry_points"`
	Docs          []RankedFile        `json:"docs"`
	Guide         string              `json:"guide"`
}

const agentGuide = "Peta kode untuk AI agent. Cari lokasi kode dengan vibeasik_code_map_search (query bebas Indonesia/Inggris, filter kind), " +
	"lihat impor & 'dipakai oleh' sebuah berkas dengan vibeasik_code_map_file sebelum mengubahnya (perkirakan dampak), " +
	"dan baca berkas 'docs' (AI_CONTEXT.md, README) untuk aturan proyek. Berkas di 'hotspots' terlalu besar untuk dibaca utuh: baca per rentang baris."

// BuildOverview menyusun peta proyek. withFiles=true menyertakan daftar berkas per direktori (bisa besar).
func BuildOverview(db *gorm.DB, projectID uint, componentID uint, withFiles bool) (*Overview, error) {
	if err := EnsureFresh(db, projectID, StaleAfter); err != nil {
		return nil, err
	}
	ix, err := loadIndex(db, projectID)
	if err != nil {
		return nil, err
	}
	var proj models.Project
	db.Select("id, name").First(&proj, projectID)
	var meta models.CodeMap
	db.Where("project_id = ?", projectID).First(&meta)

	ov := &Overview{ProjectID: projectID, ProjectName: proj.Name, RootPath: meta.RootPath, BuiltAt: meta.BuiltAt, ExportedAt: meta.ExportedAt,
		SymbolCount: meta.SymbolCount, EnrichedCount: meta.EnrichedCount, Languages: map[string]int{}, Kinds: map[string]int{}, Guide: agentGuide}
	comps := map[uint]*OverviewComponent{}
	dirs := map[string]*OverviewDir{}
	var order []uint
	var core, hot, entry, docs []RankedFile
	for _, d := range ix.docs {
		e := d.entry
		if componentID > 0 && e.ComponentID != componentID {
			continue
		}
		ov.FileCount++
		ov.TotalLines += e.Lines
		ov.Languages[e.Lang]++
		ov.Kinds[e.Kind]++
		ov.RouteCount += len(d.routes)
		c := comps[e.ComponentID]
		if c == nil {
			name := e.ComponentName
			if name == "" {
				name = "(root proyek)"
			}
			c = &OverviewComponent{ID: e.ComponentID, Name: name, Kinds: map[string]int{}}
			comps[e.ComponentID] = c
			order = append(order, e.ComponentID)
		}
		c.Files++
		c.Lines += e.Lines
		c.Kinds[e.Kind]++
		dk := fmt.Sprintf("%d|%s", e.ComponentID, path.Dir(e.Path))
		dr := dirs[dk]
		if dr == nil {
			dr = &OverviewDir{Path: path.Dir(e.Path), Kinds: map[string]int{}}
			dirs[dk] = dr
			c.Dirs = append(c.Dirs, OverviewDir{}) // placeholder, diisi setelah loop
		}
		dr.Files++
		dr.Lines += e.Lines
		dr.Kinds[e.Kind]++
		used := len(ix.dependents[e.Path])
		of := OverviewFile{Path: e.Path, Kind: e.Kind, Lang: e.Lang, Lines: e.Lines, Summary: summaryOf(e), UsedBy: used, Routes: len(d.routes)}
		for _, s := range d.symbols {
			if (s.Exported || e.Lang == "vue") && len(of.Symbols) < 8 && s.Kind != "macro" {
				of.Symbols = append(of.Symbols, s.Name)
			}
		}
		dr.Entries = append(dr.Entries, of)
		rf := RankedFile{Path: e.Path, Kind: e.Kind, Lines: e.Lines, UsedBy: used}
		if used >= 3 && e.Kind != "test" {
			core = append(core, rf)
		}
		if e.Lines >= hotspotLines && e.Kind != "doc" && e.Kind != "test" && e.Kind != "migration" {
			rf.Note = fmt.Sprintf("%d baris — baca per rentang baris; kandidat dipecah agar hemat konteks AI", e.Lines)
			hot = append(hot, rf)
		}
		if e.Kind == "entry" || len(d.routes) >= 5 && (e.Kind == "route" || e.Kind == "handler" || e.Kind == "module") {
			rf.Note = "titik masuk aplikasi/CLI"
			if len(d.routes) > 0 {
				rf.Note = fmt.Sprintf("%d rute terdaftar", len(d.routes))
			}
			entry = append(entry, rf)
		}
		if e.Kind == "doc" && strings.Count(e.Path, "/") <= 1 {
			rf.Note = summaryOf(e)
			docs = append(docs, rf)
		}
	}
	sort.SliceStable(core, func(i, j int) bool { return core[i].UsedBy > core[j].UsedBy })
	sort.SliceStable(hot, func(i, j int) bool { return hot[i].Lines > hot[j].Lines })
	sort.SliceStable(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	ov.Core, ov.Hotspots, ov.EntryPoints, ov.Docs = capRanked(core, 15), capRanked(hot, 15), capRanked(entry, 15), capRanked(docs, 15)

	for _, id := range order {
		c := comps[id]
		c.Dirs = c.Dirs[:0]
		for k, dr := range dirs {
			if !strings.HasPrefix(k, fmt.Sprintf("%d|", id)) {
				continue
			}
			sort.SliceStable(dr.Entries, func(i, j int) bool { return dr.Entries[i].UsedBy > dr.Entries[j].UsedBy })
			for i := 0; i < len(dr.Entries) && i < 3; i++ {
				if dr.Entries[i].UsedBy > 0 || i == 0 {
					dr.KeyFiles = append(dr.KeyFiles, path.Base(dr.Entries[i].Path))
				}
			}
			sort.SliceStable(dr.Entries, func(i, j int) bool { return dr.Entries[i].Path < dr.Entries[j].Path })
			if !withFiles {
				dr.Entries = nil
			}
			c.Dirs = append(c.Dirs, *dr)
		}
		sort.SliceStable(c.Dirs, func(i, j int) bool { return c.Dirs[i].Path < c.Dirs[j].Path })
		c.Path = commonDir(c.Dirs)
		ov.Components = append(ov.Components, *c)
	}
	sort.SliceStable(ov.Components, func(i, j int) bool { return ov.Components[i].Path < ov.Components[j].Path })
	return ov, nil
}

func capRanked(list []RankedFile, n int) []RankedFile {
	if list == nil {
		return []RankedFile{}
	}
	if len(list) > n {
		return list[:n]
	}
	return list
}

func commonDir(dirs []OverviewDir) string {
	if len(dirs) == 0 {
		return ""
	}
	prefix := dirs[0].Path
	for _, d := range dirs[1:] {
		for prefix != "." && prefix != "" && d.Path != prefix && !strings.HasPrefix(d.Path, prefix+"/") {
			prefix = path.Dir(prefix)
		}
	}
	if prefix == "." {
		return ""
	}
	return prefix
}

// ExportResult menjelaskan berkas peta yang ditulis ke repositori.
type ExportResult struct {
	MarkdownPath string `json:"markdown_path"`
	JSONLPath    string `json:"jsonl_path"`
	Files        int    `json:"files"`
	RoutesPath   string `json:"routes_path,omitempty"`
	Routes       int    `json:"routes,omitempty"`
}

func routeCount(gs []*routemap.Graph) int {
	n := 0
	for _, g := range gs {
		n += len(g.Routes)
	}
	return n
}

type jsonlChunk struct {
	Path      string   `json:"path"`
	Component string   `json:"component,omitempty"`
	Kind      string   `json:"kind"`
	Lang      string   `json:"lang"`
	Lines     int      `json:"lines"`
	Summary   string   `json:"summary,omitempty"`
	Keywords  string   `json:"keywords,omitempty"`
	Symbols   []string `json:"symbols,omitempty"`
	Routes    []string `json:"routes,omitempty"`
	Imports   []string `json:"imports,omitempty"`
	UsedBy    []string `json:"used_by,omitempty"`
	// RouteTrace: rute HTTP yang handler-nya (atau fungsi yang dipanggil handler) ada di berkas ini (routemap)
	RouteTrace []string `json:"route_trace,omitempty"`
}

// Export menulis peta ke <root>/.vibeasik/CODEMAP.md (dibaca agent tanpa MCP) dan codemap.jsonl
// (satu chunk per baris, mudah di-grep atau dimuat ke RAG lain).
func Export(db *gorm.DB, projectID uint) (*ExportResult, error) {
	ov, err := BuildOverview(db, projectID, 0, true)
	if err != nil {
		return nil, err
	}
	ix, err := loadIndex(db, projectID)
	if err != nil {
		return nil, err
	}
	root := ov.RootPath
	if fi, err := os.Stat(root); root == "" || root == "-" || err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("root proyek tidak ditemukan di disk; peta tidak bisa ditulis ke repositori")
	}
	dir := filepath.Join(root, ".vibeasik")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	var md strings.Builder
	fmt.Fprintf(&md, "# Peta Kode — %s\n\n", ov.ProjectName)
	fmt.Fprintf(&md, "> Dibuat otomatis oleh vibeasik pada %s (%d berkas, %d simbol). Jangan diedit manual — bangun ulang dari Studio (/code-map) atau MCP `vibeasik_code_map_export`.\n\n",
		ov.BuiltAt.Format("2006-01-02 15:04"), ov.FileCount, ov.SymbolCount)
	md.WriteString("## Cara pakai untuk AI agent\n\n")
	md.WriteString("1. Cari dulu di sini (atau `grep` di `.vibeasik/codemap.jsonl`, satu berkas per baris) sebelum menjelajah repositori.\n")
	md.WriteString("2. Bila MCP vibeasik tersedia: `vibeasik_code_map_search` (pencarian BM25 + cuplikan kode) dan `vibeasik_code_map_file` (impor & dipakai-oleh).\n")
	md.WriteString("3. Sebelum mengubah berkas, cek 'dipakai oleh' untuk memperkirakan dampak perubahan.\n")
	md.WriteString("4. Berkas di bagian Hotspot terlalu besar untuk dibaca utuh: baca per rentang baris.\n")
	md.WriteString("5. Menilai rute/endpoint (auth, validasi, response)? Baca bagian Peta Rute: middleware global & group tidak terlihat di baris rute itu sendiri.\n\n")
	writeRanked := func(title string, list []RankedFile, note func(RankedFile) string) {
		if len(list) == 0 {
			return
		}
		fmt.Fprintf(&md, "## %s\n\n", title)
		for _, r := range list {
			fmt.Fprintf(&md, "- `%s` — %s\n", r.Path, note(r))
		}
		md.WriteString("\n")
	}
	writeRanked("Dokumen konteks", ov.Docs, func(r RankedFile) string { return r.Note })
	writeRanked("Titik masuk & rute", ov.EntryPoints, func(r RankedFile) string { return r.Kind + ", " + r.Note })
	writeRanked("Modul inti (paling banyak dipakai)", ov.Core, func(r RankedFile) string { return fmt.Sprintf("%s, dipakai %d berkas", r.Kind, r.UsedBy) })
	writeRanked("Hotspot (berkas besar)", ov.Hotspots, func(r RankedFile) string { return r.Note })
	graphs := routemap.Analyze(root)
	md.WriteString(routemap.Markdown(graphs))
	for _, c := range ov.Components {
		fmt.Fprintf(&md, "## Aplikasi: %s (`%s`, %d berkas, %d baris)\n\n", c.Name, c.Path, c.Files, c.Lines)
		for _, d := range c.Dirs {
			fmt.Fprintf(&md, "### %s/\n\n", d.Path)
			for _, f := range d.Entries {
				fmt.Fprintf(&md, "- `%s` (%s, %d baris", path.Base(f.Path), f.Kind, f.Lines)
				if f.UsedBy > 0 {
					fmt.Fprintf(&md, ", dipakai %d", f.UsedBy)
				}
				md.WriteString(")")
				if f.Summary != "" {
					md.WriteString(" — " + f.Summary)
				}
				if len(f.Symbols) > 0 {
					md.WriteString(" · " + strings.Join(f.Symbols, ", "))
				}
				md.WriteString("\n")
			}
			md.WriteString("\n")
		}
	}

	var jl strings.Builder
	enc := json.NewEncoder(&jl)
	enc.SetEscapeHTML(false)
	fileRoutes := routemap.FileRoutes(graphs)
	for _, d := range ix.docs {
		ch := jsonlChunk{RouteTrace: fileRoutes[d.entry.Path], Path: d.entry.Path, Component: d.entry.ComponentName, Kind: d.entry.Kind, Lang: d.entry.Lang, Lines: d.entry.Lines,
			Summary: summaryOf(d.entry), Keywords: d.entry.Keywords, Routes: d.routes, Imports: d.imports, UsedBy: ix.dependents[d.entry.Path]}
		for _, s := range d.symbols {
			ch.Symbols = append(ch.Symbols, fmt.Sprintf("%s:%s@%d", s.Kind, s.Name, s.Line))
		}
		_ = enc.Encode(ch)
	}

	res := &ExportResult{MarkdownPath: filepath.Join(dir, "CODEMAP.md"), JSONLPath: filepath.Join(dir, "codemap.jsonl"), Files: ov.FileCount}
	if err := os.WriteFile(res.MarkdownPath, []byte(md.String()), 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(res.JSONLPath, []byte(jl.String()), 0o644); err != nil {
		return nil, err
	}
	if len(graphs) > 0 {
		res.RoutesPath, res.Routes = filepath.Join(dir, "routes.jsonl"), routeCount(graphs)
		if err := os.WriteFile(res.RoutesPath, []byte(routemap.JSONL(graphs)), 0o644); err != nil {
			return nil, err
		}
	}
	now := time.Now()
	db.Model(&models.CodeMap{}).Where("project_id = ?", projectID).Update("exported_at", &now)
	return res, nil
}
