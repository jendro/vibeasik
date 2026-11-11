// Package insight menyusun angka untuk pemilik proyek: biaya proses (tick & Big-O) tiap rute/fungsi dan
// rekap kemajuan (todo, temuan audit, kesehatan kode). Semua hitungan dikerjakan di server tanpa paging,
// supaya angka di layar tidak bergantung pada jumlah baris yang kebetulan termuat.
package insight

import (
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
	Supported      bool                `json:"supported"` // false: tidak ada backend Go ber-rute yang bisa ditelusuri
	Modules        []string            `json:"modules"`
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
	Routes  []routemap.RouteCost         `json:"routes"`
	Funcs   []routemap.FuncCost          `json:"funcs"`
	History []models.ProcessCostSnapshot `json:"history"`
}

const (
	defaultCostLimit = 50
	maxCostLimit     = 200
	costHistoryDays  = 60
)

// ProcessCost menghitung biaya proses proyek, mencatat potret hariannya, dan mengembalikan satu halaman daftar.
func ProcessCost(db *gorm.DB, projectID uint, q CostQuery, now time.Time) (*CostPage, error) {
	graphs, _, err := codemap.RouteGraphs(db, projectID)
	if err != nil {
		return nil, err
	}
	rep := routemap.Costs(graphs)
	page := &CostPage{Summary: summarize(graphs, rep), Kind: "routes", Routes: []routemap.RouteCost{}, Funcs: []routemap.FuncCost{}}
	if page.Summary.Supported {
		saveSnapshot(db, projectID, page.Summary, now)
	}
	page.History = costHistory(db, projectID, now)

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
		page.Funcs = append(page.Funcs, window(all, page.Offset, page.Limit)...)
		return page, nil
	}
	var all []routemap.RouteCost
	for _, r := range rep.Routes {
		if keep(r.Cost, r.Method+" "+r.Path+" "+r.Handler) {
			all = append(all, r)
		}
	}
	page.Total = len(all)
	page.Routes = append(page.Routes, window(all, page.Offset, page.Limit)...)
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

func summarize(graphs []*routemap.Graph, rep *routemap.CostReport) CostSummary {
	s := CostSummary{Modules: []string{}, RouteClasses: map[string]int{}, FuncClasses: map[string]int{}, Routes: len(rep.Routes), Funcs: len(rep.Funcs)}
	for _, g := range graphs {
		s.Modules = append(s.Modules, g.Module)
	}
	sort.Strings(s.Modules)
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
