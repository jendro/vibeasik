package codemap

import (
	"encoding/json"
	"math"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/internal/models"
)

// doc adalah satu berkas di indeks memori beserta frekuensi term berbobot per field.
type doc struct {
	entry   models.CodeMapEntry
	symbols []models.CodeSymbol
	imports []string
	routes  []string
	tf      map[string]float64
	length  float64
}

type index struct {
	builtAt    time.Time
	docs       []*doc
	byPath     map[string]*doc
	df         map[string]int
	avgLen     float64
	dependents map[string][]string // path berkas → berkas yang mengimpornya (termasuk lewat direktori paket)
}

var indexes sync.Map // projectID → *index

func invalidate(projectID uint) { indexes.Delete(projectID) }

// Bobot field: nama berkas & simbol paling menentukan, lalu rute, ringkasan, dan kata kunci LLM.
const (
	wPath    = 3.0
	wSymbol  = 3.0
	wRoute   = 2.0
	wSummary = 1.5
	wKeyword = 1.5
	wDoc     = 0.7
	bm25K1   = 1.2
	bm25B    = 0.75
)

var stopwords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "from": true, "this": true, "that": true, "into": true, "func": true,
	"yang": true, "dan": true, "di": true, "ke": true, "dari": true, "untuk": true, "dengan": true, "ini": true, "itu": true,
	"atau": true, "pada": true, "agar": true, "bila": true, "saat": true, "oleh": true, "akan": true, "juga": true, "tidak": true,
	"ts": true, "js": true, "go": true, "vue": true, "app": true, "src": true, "internal": true, "index": true, "lib": true,
}

// synonyms menjembatani istilah Indonesia ↔ Inggris yang sering dipakai agent saat mencari.
var synonyms = map[string][]string{
	"hapus": {"delete", "remove"}, "simpan": {"save", "store"}, "ubah": {"update", "edit"}, "buat": {"create", "new"},
	"daftar": {"list"}, "masuk": {"login", "signin"}, "keluar": {"logout"}, "pengguna": {"user"}, "proyek": {"project"},
	"tugas": {"todo", "task"}, "antrean": {"queue"}, "antre": {"queue"}, "pengaturan": {"settings", "config"}, "halaman": {"page"},
	"tombol": {"button"}, "keamanan": {"security", "auth"}, "berkas": {"file"}, "peta": {"map"}, "rute": {"route", "router"},
	"komponen": {"component"}, "kerumitan": {"complexity"}, "dampak": {"impact"}, "agen": {"agent"}, "jalankan": {"run", "exec"},
	"cari": {"search", "find"}, "unggah": {"upload"}, "unduh": {"download"}, "laporan": {"report"}, "kantor": {"office"},
	"temuan": {"finding"}, "audit": {"diagnostic", "audit"}, "notifikasi": {"toast", "notification"}, "basis": {"database", "db"},
	"delete": {"hapus"}, "settings": {"pengaturan"}, "queue": {"antrean"}, "todo": {"tugas"}, "impact": {"dampak"},
}

// tokenize memecah teks menjadi term: kata utuh + potongan camelCase/snake_case, huruf kecil, tanpa stopword.
func tokenize(s string) []string {
	var out []string
	add := func(w string) {
		w = strings.ToLower(w)
		if len(w) >= 2 && !stopwords[w] {
			out = append(out, w)
		}
	}
	words := strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	for _, w := range words {
		add(w)
		var parts []string
		start := 0
		rs := []rune(w)
		for i := 1; i < len(rs); i++ {
			prev, cur := rs[i-1], rs[i]
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			if (unicode.IsLower(prev) && unicode.IsUpper(cur)) || (unicode.IsUpper(prev) && unicode.IsUpper(cur) && nextLower) ||
				(unicode.IsLetter(prev) && unicode.IsDigit(cur)) || (unicode.IsDigit(prev) && unicode.IsLetter(cur)) {
				parts = append(parts, string(rs[start:i]))
				start = i
			}
		}
		if start > 0 {
			parts = append(parts, string(rs[start:]))
			for _, p := range parts {
				add(p)
			}
		}
	}
	return out
}

func addTerms(tf map[string]float64, text string, weight float64) float64 {
	n := 0.0
	for _, t := range tokenize(text) {
		tf[t] += weight
		n += weight
	}
	return n
}

func newDoc(e models.CodeMapEntry) *doc {
	d := &doc{entry: e, tf: map[string]float64{}}
	_ = json.Unmarshal([]byte(e.SymbolsJSON), &d.symbols)
	_ = json.Unmarshal([]byte(e.ImportsJSON), &d.imports)
	_ = json.Unmarshal([]byte(e.RoutesJSON), &d.routes)
	d.length += addTerms(d.tf, strings.ReplaceAll(e.Path, "/", " "), wPath)
	d.length += addTerms(d.tf, e.Kind, 1)
	for _, s := range d.symbols {
		d.length += addTerms(d.tf, s.Name, wSymbol)
		if s.Doc != "" {
			d.length += addTerms(d.tf, s.Doc, wDoc)
		}
	}
	for _, r := range d.routes {
		d.length += addTerms(d.tf, r, wRoute)
	}
	d.length += addTerms(d.tf, e.Summary, wSummary)
	d.length += addTerms(d.tf, e.AISummary, wSummary)
	d.length += addTerms(d.tf, e.Keywords, wKeyword)
	return d
}

func loadIndex(db *gorm.DB, projectID uint) (*index, error) {
	var meta models.CodeMap
	if err := db.Where("project_id = ?", projectID).First(&meta).Error; err != nil {
		return nil, err
	}
	if v, ok := indexes.Load(projectID); ok {
		if ix := v.(*index); ix.builtAt.Equal(meta.BuiltAt) {
			return ix, nil
		}
	}
	var entries []models.CodeMapEntry
	if err := db.Where("project_id = ?", projectID).Order("path").Find(&entries).Error; err != nil {
		return nil, err
	}
	ix := &index{builtAt: meta.BuiltAt, byPath: map[string]*doc{}, df: map[string]int{}, dependents: map[string][]string{}}
	total := 0.0
	byDir := map[string][]string{}
	for _, e := range entries {
		d := newDoc(e)
		ix.docs = append(ix.docs, d)
		ix.byPath[e.Path] = d
		byDir[path.Dir(e.Path)+"/"] = append(byDir[path.Dir(e.Path)+"/"], e.Path)
		total += d.length
		for t := range d.tf {
			ix.df[t]++
		}
	}
	if len(ix.docs) > 0 {
		ix.avgLen = total / float64(len(ix.docs))
	}
	for _, d := range ix.docs {
		for _, imp := range d.imports {
			targets := []string{imp}
			if strings.HasSuffix(imp, "/") {
				targets = byDir[imp]
			}
			for _, t := range targets {
				if t != d.entry.Path {
					ix.dependents[t] = append(ix.dependents[t], d.entry.Path)
				}
			}
		}
	}
	indexes.Store(projectID, ix)
	return ix, nil
}

// SearchOptions mengatur pencarian peta kode.
type SearchOptions struct {
	Query       string
	Kind        string
	ComponentID uint
	Limit       int
	Snippet     bool
}

// SearchHit adalah satu chunk hasil pencarian: berkas + simbol yang cocok (+ cuplikan kode bila diminta).
type SearchHit struct {
	Path         string              `json:"path"`
	AbsPath      string              `json:"abs_path"`
	Component    string              `json:"component,omitempty"`
	Kind         string              `json:"kind"`
	Lang         string              `json:"lang"`
	Lines        int                 `json:"lines"`
	Score        float64             `json:"score"`
	Summary      string              `json:"summary,omitempty"`
	Symbols      []models.CodeSymbol `json:"symbols,omitempty"`
	Routes       []string            `json:"routes,omitempty"`
	Imports      int                 `json:"imports"`
	UsedBy       []string            `json:"used_by,omitempty"`
	UsedByCount  int                 `json:"used_by_count"`
	Snippet      string              `json:"snippet,omitempty"`
	SnippetStart int                 `json:"snippet_start,omitempty"`
}

func summaryOf(e models.CodeMapEntry) string {
	if e.AISummary != "" {
		if e.Summary != "" && !strings.EqualFold(e.Summary, e.AISummary) && len(e.Summary) < 120 {
			return e.AISummary + " (" + e.Summary + ")"
		}
		return e.AISummary
	}
	return e.Summary
}

func queryTerms(q string) ([]string, map[string]float64) {
	weights := map[string]float64{}
	var order []string
	add := func(t string, w float64) {
		if _, ok := weights[t]; !ok {
			order = append(order, t)
		}
		weights[t] = math.Max(weights[t], w)
	}
	for _, t := range tokenize(q) {
		add(t, 1)
		for _, s := range synonyms[t] {
			add(s, 0.6)
		}
	}
	return order, weights
}

// queryConcepts mengelompokkan setiap kata query bersama sinonimnya.
func queryConcepts(q string) [][]string {
	var out [][]string
	seen := map[string]bool{}
	for _, t := range tokenize(q) {
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, append([]string{t}, synonyms[t]...))
	}
	return out
}

// symbolCoverage menghitung jumlah konsep query terbanyak yang termuat dalam satu nama simbol.
func symbolCoverage(d *doc, concepts [][]string) int {
	best := 0
	for _, sym := range d.symbols {
		toks := tokenize(sym.Name)
		n := 0
		for _, c := range concepts {
		found:
			for _, alt := range c {
				for _, x := range toks {
					if x == alt {
						n++
						break found
					}
				}
			}
		}
		best = max(best, n)
	}
	return best
}

func (ix *index) score(d *doc, terms []string, weights map[string]float64) float64 {
	n := float64(len(ix.docs))
	s := 0.0
	for _, t := range terms {
		f := d.tf[t]
		if f == 0 {
			// Awalan identifier (mis. "orchestr" → "orchestrator") diberi nilai sebagian.
			if len(t) >= 4 {
				for term, v := range d.tf {
					if len(term) > len(t) && strings.HasPrefix(term, t) {
						f = math.Max(f, v*0.5)
					}
				}
			}
			if f == 0 {
				continue
			}
		}
		df := float64(ix.df[t])
		if df == 0 {
			df = 1
		}
		idf := math.Log(1 + (n-df+0.5)/(df+0.5))
		s += weights[t] * idf * (f * (bm25K1 + 1)) / (f + bm25K1*(1-bm25B+bm25B*d.length/math.Max(ix.avgLen, 1)))
	}
	return s
}

func matchedSymbols(d *doc, terms []string, q string) []models.CodeSymbol {
	type scored struct {
		s models.CodeSymbol
		v int
	}
	ql := strings.ToLower(strings.TrimSpace(q))
	var hits []scored
	for _, s := range d.symbols {
		v := 0
		name := strings.ToLower(s.Name)
		if name == ql {
			v += 10
		}
		toks := tokenize(s.Name + " " + s.Doc)
		for _, t := range terms {
			for _, x := range toks {
				if x == t || len(t) >= 4 && strings.HasPrefix(x, t) {
					v++
					break
				}
			}
		}
		if v > 0 {
			hits = append(hits, scored{s, v})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].v > hits[j].v })
	out := []models.CodeSymbol{}
	for i, h := range hits {
		if i >= 6 {
			break
		}
		out = append(out, h.s)
	}
	return out
}

// Search mencari berkas paling relevan (BM25 berbobot field + bonus nama simbol/path persis).
func Search(db *gorm.DB, projectID uint, opt SearchOptions) ([]SearchHit, error) {
	if err := EnsureFresh(db, projectID, StaleAfter); err != nil {
		return nil, err
	}
	ix, err := loadIndex(db, projectID)
	if err != nil {
		return nil, err
	}
	if opt.Limit <= 0 || opt.Limit > 50 {
		opt.Limit = 10
	}
	terms, weights := queryTerms(opt.Query)
	concepts := queryConcepts(opt.Query)
	ql := strings.ToLower(strings.TrimSpace(opt.Query))
	wantsTests := strings.Contains(ql, "test")
	type cand struct {
		d *doc
		s float64
	}
	var cands []cand
	for _, d := range ix.docs {
		if opt.Kind != "" && d.entry.Kind != opt.Kind {
			continue
		}
		if opt.ComponentID > 0 && d.entry.ComponentID != opt.ComponentID {
			continue
		}
		s := ix.score(d, terms, weights)
		if ql != "" && strings.Contains(strings.ToLower(d.entry.Path), ql) {
			s += 3
		}
		for _, sym := range d.symbols {
			if strings.EqualFold(sym.Name, strings.TrimSpace(opt.Query)) {
				s += 4
				break
			}
		}
		// Satu simbol yang memuat semua konsep query ("hapus todo" → deleteTodo) lebih tepat daripada berkas yang
		// sekadar menyebut kata-katanya di tempat berbeda.
		if cov := symbolCoverage(d, concepts); len(concepts) >= 2 && cov >= 2 {
			if cov == len(concepts) {
				s += 3
			} else {
				s += 1
			}
		}
		if s <= 0 {
			continue
		}
		if d.entry.Kind == "test" && !wantsTests {
			s *= 0.6
		}
		// Berkas yang banyak dipakai sedikit diutamakan (modul inti).
		s *= 1 + math.Min(float64(len(ix.dependents[d.entry.Path])), 20)/100
		cands = append(cands, cand{d, s})
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].s > cands[j].s })
	hits := []SearchHit{}
	for i, c := range cands {
		if i >= opt.Limit {
			break
		}
		h := hitFor(ix, c.d, terms, opt.Query)
		h.Score = math.Round(c.s*100) / 100
		if opt.Snippet && i < 5 {
			from := 1
			if len(h.Symbols) > 0 {
				from = h.Symbols[0].Line - 3
			}
			h.Snippet, h.SnippetStart = readLines(c.d.entry.AbsPath, from, from+39)
		}
		hits = append(hits, h)
	}
	return hits, nil
}

func hitFor(ix *index, d *doc, terms []string, q string) SearchHit {
	used := ix.dependents[d.entry.Path]
	h := SearchHit{Path: d.entry.Path, AbsPath: d.entry.AbsPath, Component: d.entry.ComponentName, Kind: d.entry.Kind, Lang: d.entry.Lang,
		Lines: d.entry.Lines, Summary: summaryOf(d.entry), Imports: len(d.imports), UsedByCount: len(used)}
	h.Symbols = matchedSymbols(d, terms, q)
	if len(h.Symbols) == 0 {
		for _, s := range d.symbols {
			if s.Exported && len(h.Symbols) < 5 {
				h.Symbols = append(h.Symbols, s)
			}
		}
	}
	h.Routes = d.routes
	if len(h.Routes) > 8 {
		h.Routes = h.Routes[:8]
	}
	h.UsedBy = used
	if len(h.UsedBy) > 8 {
		h.UsedBy = h.UsedBy[:8]
	}
	return h
}

// FileDetail adalah isi lengkap satu entry peta: semua simbol, impor, dipakai-oleh, dan rute.
type FileDetail struct {
	models.CodeMapEntry
	Symbols []models.CodeSymbol `json:"symbols"`
	Imports []string            `json:"imports"`
	UsedBy  []string            `json:"used_by"`
	Routes  []string            `json:"routes"`
}

// File mengambil detail berkas berdasarkan path relatif, path absolut, atau akhiran path unik.
func File(db *gorm.DB, projectID uint, file string) (*FileDetail, []string, error) {
	if err := EnsureFresh(db, projectID, StaleAfter); err != nil {
		return nil, nil, err
	}
	ix, err := loadIndex(db, projectID)
	if err != nil {
		return nil, nil, err
	}
	file = strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(file), "\\", "/"), "./")
	d := ix.byPath[file]
	var candidates []string
	if d == nil {
		for _, x := range ix.docs {
			if x.entry.AbsPath == file || strings.HasSuffix(x.entry.Path, "/"+file) || strings.HasSuffix(file, "/"+x.entry.Path) {
				candidates = append(candidates, x.entry.Path)
				d = x
			}
		}
		if len(candidates) > 1 {
			return nil, candidates, nil
		}
	}
	if d == nil {
		return nil, nil, nil
	}
	fd := &FileDetail{CodeMapEntry: d.entry, Symbols: d.symbols, Imports: d.imports, UsedBy: ix.dependents[d.entry.Path], Routes: d.routes}
	if fd.Symbols == nil {
		fd.Symbols = []models.CodeSymbol{}
	}
	return fd, nil, nil
}
