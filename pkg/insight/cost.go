// Package insight menyusun angka untuk pemilik proyek: biaya proses (tick & Big-O) tiap rute/fungsi dan
// rekap kemajuan (todo, temuan audit, kesehatan kode). Semua hitungan dikerjakan di server tanpa paging,
// supaya angka di layar tidak bergantung pada jumlah baris yang kebetulan termuat.
package insight

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/codemap"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/routemap"
)

// CostQuery memilih daftar (rute/fungsi), menyaring, dan membatasi halaman.
type CostQuery struct {
	Kind     string // routes (bawaan) | funcs
	Search   string
	Class    string // constant, linear, linearithmic, quadratic, cubic, polynomial
	NPlusOne bool
	Limit    int
	Offset   int
}

// CostSummary = sebaran kelas pertumbuhan seluruh proyek (bukan hanya halaman yang termuat).
type CostSummary struct {
	Supported      bool                `json:"supported"` // false: tidak ada backend Go/PHP ber-rute yang bisa ditelusuri
	Modules        []string            `json:"modules"`
	Frameworks     []string            `json:"frameworks"` // Go (fiber, gin …), Laravel, CodeIgniter 4, Slim, PHP
	Routes         int                 `json:"routes"`
	Funcs          int                 `json:"funcs"`
	RouteClasses   map[string]int      `json:"route_classes"`
	FuncClasses    map[string]int      `json:"func_classes"`
	NPlusOneRoutes int                 `json:"n_plus_one_routes"`
	NPlusOneFuncs  int                 `json:"n_plus_one_funcs"`
	Recursive      int                 `json:"recursive"`
	Unbounded      int                 `json:"unbounded"`
	RouteTicks1    int64               `json:"route_ticks_1"` // jumlah T(1) semua rute
	WorstRoute     *routemap.RouteCost `json:"worst_route,omitempty"`
}

// CostPage = jawaban endpoint biaya proses.
type CostPage struct {
	Summary CostSummary                  `json:"summary"`
	Kind    string                       `json:"kind"`
	Total   int                          `json:"total"`
	Limit   int                          `json:"limit"`
	Offset  int                          `json:"offset"`
	Routes  []RouteRow                   `json:"routes"`
	Funcs   []FuncRow                    `json:"funcs"`
	History []models.ProcessCostSnapshot `json:"history"`
	// Handling: sumber biaya tinggi yang sudah ditangani (todo / audit) dan yang belum.
	Handling CostHandlingSummary `json:"handling"`
	// ComputedAt & DurationMs: kapan rumus ini dihitung dari kode di disk dan berapa lama. Setiap permintaan
	// menghitung ulang dari kode terbaru (tidak ada cache), jadi "cek ulang" = memuat ulang endpoint ini.
	ComputedAt time.Time `json:"computed_at"`
	DurationMs int64     `json:"duration_ms"`
}

const (
	defaultCostLimit = 50
	maxCostLimit     = 200
	costHistoryDays  = 60
)

type costAnalysis struct {
	root string
	rep  *routemap.CostReport
	mods []string
	fws  []string
	hots []routemap.Hotspot
}

// analyze menghitung biaya Go dan PHP (Laravel, CodeIgniter, Slim, PHP biasa) dengan model tick yang sama,
// lalu menggabungkannya, beserta sumber biaya (hotspot) keduanya.
func analyze(db *gorm.DB, projectID uint) (*costAnalysis, error) {
	graphs, root, err := codemap.RouteGraphs(db, projectID)
	if err != nil {
		return nil, err
	}
	keep := componentDirFilter(db, projectID)
	if keep != nil {
		kept := graphs[:0]
		for _, g := range graphs {
			if keep(filepath.Join(root, filepath.FromSlash(g.Dir))) {
				kept = append(kept, g)
			}
		}
		graphs = kept
	}
	php := routemap.AnalyzePHPIn(root, keep)
	a := &costAnalysis{root: root, rep: routemap.MergeCosts(routemap.Costs(graphs), php.Report)}
	for _, g := range graphs {
		a.mods = append(a.mods, g.Module)
		if g.Framework != "" {
			a.fws = append(a.fws, "Go ("+g.Framework+")")
		}
	}
	for _, m := range php.Modules {
		a.mods = append(a.mods, m.Name)
		a.fws = append(a.fws, phpFrameworkName(m.Framework))
	}
	a.hots = append(routemap.Hotspots(graphs), php.Hotspots...)
	return a, nil
}

// ProcessCost menghitung biaya proses proyek, mencatat potret hariannya, dan mengembalikan satu halaman daftar.
func ProcessCost(db *gorm.DB, projectID uint, q CostQuery, now time.Time) (*CostPage, error) {
	started := time.Now()
	a, err := analyze(db, projectID)
	if err != nil {
		return nil, err
	}
	rep := a.rep
	page := &CostPage{Summary: summarize(a.mods, a.fws, rep), Kind: "routes", Routes: []RouteRow{}, Funcs: []FuncRow{},
		ComputedAt: now, DurationMs: time.Since(started).Milliseconds()}
	if page.Summary.Supported {
		saveSnapshot(db, projectID, page.Summary, now)
	}
	page.History = costHistory(db, projectID, now)
	routeRow, funcRow, handling := annotate(a.hots, loadCostLedger(db, projectID))
	page.Handling = handling

	page.Limit, page.Offset = q.Limit, q.Offset
	if page.Limit <= 0 {
		page.Limit = defaultCostLimit
	}
	if page.Limit > maxCostLimit {
		page.Limit = maxCostLimit
	}
	if page.Offset < 0 {
		page.Offset = 0
	}
	needle := strings.ToLower(strings.TrimSpace(q.Search))
	keep := func(c routemap.Cost, hay string) bool {
		if q.Class != "" && c.Class != q.Class {
			return false
		}
		if q.NPlusOne && !c.NPlusOne {
			return false
		}
		return needle == "" || strings.Contains(strings.ToLower(hay), needle)
	}
	if q.Kind == "funcs" {
		page.Kind = "funcs"
		var all []routemap.FuncCost
		for _, f := range rep.Funcs {
			if keep(f.Cost, f.Name+" "+f.File) {
				all = append(all, f)
			}
		}
		page.Total = len(all)
		for _, f := range window(all, page.Offset, page.Limit) {
			page.Funcs = append(page.Funcs, funcRow(f))
		}
		return page, nil
	}
	var all []routemap.RouteCost
	for _, r := range rep.Routes {
		if keep(r.Cost, r.Method+" "+r.Path+" "+r.Handler) {
			all = append(all, r)
		}
	}
	page.Total = len(all)
	for _, r := range window(all, page.Offset, page.Limit) {
		page.Routes = append(page.Routes, routeRow(r))
	}
	return page, nil
}

func window[T any](all []T, offset, limit int) []T {
	if offset >= len(all) {
		return nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end]
}

func phpFrameworkName(fw string) string {
	switch fw {
	case "laravel":
		return "Laravel"
	case "lumen":
		return "Lumen"
	case "codeigniter4":
		return "CodeIgniter 4"
	case "slim":
		return "Slim"
	}
	return "PHP"
}

func uniqSorted(in []string) []string {
	sort.Strings(in)
	out := []string{}
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}

func summarize(mods, fws []string, rep *routemap.CostReport) CostSummary {
	s := CostSummary{Modules: uniqSorted(mods), Frameworks: uniqSorted(fws), RouteClasses: map[string]int{}, FuncClasses: map[string]int{}, Routes: len(rep.Routes), Funcs: len(rep.Funcs)}
	s.Supported = len(rep.Routes) > 0
	for _, r := range rep.Routes {
		s.RouteClasses[r.Class]++
		s.RouteTicks1 += r.Ticks1
		if r.NPlusOne {
			s.NPlusOneRoutes++
		}
	}
	for _, f := range rep.Funcs {
		s.FuncClasses[f.Class]++
		if f.NPlusOne {
			s.NPlusOneFuncs++
		}
		if f.Recursive {
			s.Recursive++
		}
		if f.Unbounded {
			s.Unbounded++
		}
	}
	if len(rep.Routes) > 0 {
		w := rep.Routes[0]
		s.WorstRoute = &w
	}
	return s
}

func saveSnapshot(db *gorm.DB, projectID uint, s CostSummary, now time.Time) {
	snap := models.ProcessCostSnapshot{
		Routes: s.Routes, Funcs: s.Funcs,
		Constant: s.RouteClasses["constant"] + s.RouteClasses["log"], Linear: s.RouteClasses["linear"],
		Linearithmic: s.RouteClasses["linearithmic"], Quadratic: s.RouteClasses["quadratic"],
		Worse: s.RouteClasses["cubic"] + s.RouteClasses["polynomial"], NPlusOne: s.NPlusOneRoutes, Ticks1: s.RouteTicks1,
	}
	key := models.ProcessCostSnapshot{ProjectID: projectID, SnapDate: now.Format("2006-01-02")}
	// Potret hanyalah pelengkap grafik tren: kegagalan menyimpan tidak boleh menggagalkan laporan.
	_ = db.Where(&key).Assign(snap).FirstOrCreate(&models.ProcessCostSnapshot{}).Error
}

func costHistory(db *gorm.DB, projectID uint, now time.Time) []models.ProcessCostSnapshot {
	out := []models.ProcessCostSnapshot{}
	since := now.AddDate(0, 0, -costHistoryDays).Format("2006-01-02")
	_ = db.Where("project_id = ? AND snap_date >= ?", projectID, since).Order("snap_date ASC").Find(&out).Error
	return out
}

// componentDirFilter: aplikasi (go.mod / composer.json) yang dihitung hanya yang berada di folder komponen
// terdaftar, atau yang memuat folder komponen. Tanpa filter, salinan sistem lama di bawah root (mis. legacy/ yang
// sudah digantikan) ikut menghasilkan todo biaya proses untuk kode yang tidak dipakai lagi. nil = proyek tanpa
// folder komponen yang dikenal, semua aplikasi dihitung.
func componentDirFilter(db *gorm.DB, projectID uint) func(dir string) bool {
	var comps []models.Component
	db.Select("absolute_path").Where("project_id = ?", projectID).Find(&comps)
	var dirs []string
	for _, c := range comps {
		if p := strings.TrimSpace(c.AbsolutePath); p != "" && p != "-" {
			dirs = append(dirs, filepath.Clean(p))
		}
	}
	if len(dirs) == 0 {
		return nil
	}
	return func(dir string) bool {
		dir = filepath.Clean(dir)
		for _, c := range dirs {
			if pathWithin(c, dir) || pathWithin(dir, c) {
				return true
			}
		}
		return false
	}
}

func pathWithin(parent, child string) bool {
	r, err := filepath.Rel(parent, child)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}
