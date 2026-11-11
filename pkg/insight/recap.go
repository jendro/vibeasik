package insight

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// DayPoint = satu hari pada grafik kemajuan.
type DayPoint struct {
	Date             string `json:"date"` // YYYY-MM-DD
	Created          int    `json:"created"`
	Completed        int    `json:"completed"`
	BugsClosed       int    `json:"bugs_closed"`
	FindingsResolved int    `json:"findings_resolved"`
	Open             int    `json:"open"` // todo terbuka pada akhir hari itu
}

// TodoTotals = hitungan todo seluruh proyek (tanpa batas halaman).
type TodoTotals struct {
	Total            int `json:"total"` // selesai + terbuka
	Done             int `json:"done"`
	Open             int `json:"open"`
	InProgress       int `json:"in_progress"`
	Suggested        int `json:"suggested"`
	Percent          int `json:"percent"`
	DoneInPeriod     int `json:"done_in_period"`
	CreatedInPeriod  int `json:"created_in_period"`
	AvgScore         int `json:"avg_score"` // rata-rata nilai QA todo yang dinilai
	ScoredCount      int `json:"scored_count"`
	OpenHighPriority int `json:"open_high_priority"`
}

// CategoryStat = todo selesai/terbuka per kategori.
type CategoryStat struct {
	Category string `json:"category"`
	Done     int    `json:"done"`
	Open     int    `json:"open"`
}

// TodoBrief = todo ringkas untuk daftar sorotan.
type TodoBrief struct {
	ID          uint       `json:"id"`
	Title       string     `json:"title"`
	Category    string     `json:"category"`
	Priority    string     `json:"priority"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Score       *int       `json:"score,omitempty"`
}

// WorkStat = kelompok pekerjaan (bug, perbaikan algoritma): selesai, masih terbuka, dan yang terbaru.
type WorkStat struct {
	Done         int         `json:"done"`
	Open         int         `json:"open"`
	DoneInPeriod int         `json:"done_in_period"`
	Latest       []TodoBrief `json:"latest"`
}

// FindingStats = temuan audit terbaru tiap aplikasi.
type FindingStats struct {
	Open             int            `json:"open"`
	Resolved         int            `json:"resolved"`
	Ignored          int            `json:"ignored"`
	OpenCritical     int            `json:"open_critical"`
	ResolvedInPeriod int            `json:"resolved_in_period"`
	OpenBySeverity   map[string]int `json:"open_by_severity"`
}

// HealthPoint = skor kesehatan kode dari satu audit.
type HealthPoint struct {
	Date  string `json:"date"`
	Score int    `json:"score"`
}

// OwnerRecap = rekap satu proyek untuk pemiliknya.
type OwnerRecap struct {
	ProjectID   uint           `json:"project_id"`
	Days        int            `json:"days"`
	GeneratedAt time.Time      `json:"generated_at"`
	Todos       TodoTotals     `json:"todos"`
	Timeline    []DayPoint     `json:"timeline"`
	Categories  []CategoryStat `json:"categories"`
	Bugs        WorkStat       `json:"bugs"`
	Algorithm   WorkStat       `json:"algorithm"`
	Findings    FindingStats   `json:"findings"`
	Health      []HealthPoint  `json:"health"`
	HealthScore int            `json:"health_score"`
	Highlights  []TodoBrief    `json:"highlights"`
}

var (
	// Todo yang menutup bug: berasal dari audit/scanner, atau judulnya menyebut perbaikan galat.
	bugTitleRe = regexp.MustCompile(`(?i)\b(bug|fix|hotfix|perbaik\w*|galat|error|crash|gagal|rusak|regresi)\b`)
	// Todo perbaikan algoritma/performa di luar kategori performance.
	algoTitleRe = regexp.MustCompile(`(?i)(n\s*\+\s*1|big[- ]?o\b|kompleksitas|complexity|algoritm|optimas|optimi[sz]|\bquery\b|\bindex\b|\bindeks\b|\bcache\b|paging|pagination|performa|lambat|\bslow\b|loop bersarang)`)
)

const (
	recapLatest     = 6
	recapHealthMax  = 60
	defaultRecapDay = 30
	maxRecapDays    = 180
)

type recapTodo struct {
	ID          uint
	Title       string
	Category    string
	Status      string
	Priority    string
	Source      string
	AuditKey    string
	Tags        string
	Score       *int
	CreatedAt   time.Time
	CompletedAt *time.Time
}

func (t recapTodo) brief() TodoBrief {
	return TodoBrief{ID: t.ID, Title: t.Title, Category: t.Category, Priority: t.Priority, CompletedAt: t.CompletedAt, Score: t.Score}
}

func isBugTodo(t recapTodo) bool {
	return t.AuditKey != "" || t.Source == "scanner" || bugTitleRe.MatchString(t.Title)
}

func isAlgoTodo(t recapTodo) bool {
	return t.Category == models.CategoryPerformance || strings.Contains(t.Tags, "optimasi") || algoTitleRe.MatchString(t.Title)
}

// BuildOwnerRecap menyusun rekap proyek untuk `days` hari terakhir (bawaan 30).
func BuildOwnerRecap(db *gorm.DB, projectID uint, days int, now time.Time) (*OwnerRecap, error) {
	if days <= 0 {
		days = defaultRecapDay
	}
	if days > maxRecapDays {
		days = maxRecapDays
	}
	var proj models.Project
	if err := db.Select("id", "health_score").First(&proj, projectID).Error; err != nil {
		return nil, err
	}
	out := &OwnerRecap{ProjectID: projectID, Days: days, GeneratedAt: now, HealthScore: proj.HealthScore,
		Categories: []CategoryStat{}, Health: []HealthPoint{}, Highlights: []TodoBrief{},
		Bugs: WorkStat{Latest: []TodoBrief{}}, Algorithm: WorkStat{Latest: []TodoBrief{}},
		Findings: FindingStats{OpenBySeverity: map[string]int{}}}

	// Hanya kolom kecil yang dibaca; seluruh todo proyek dihitung di sini agar angkanya tidak terpotong limit.
	var todos []recapTodo
	if err := db.Model(&models.Todo{}).
		Select("id", "title", "category", "status", "priority", "source", "audit_key", "tags", "score", "created_at", "completed_at").
		Where("project_id = ?", projectID).Find(&todos).Error; err != nil {
		return nil, err
	}

	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	start := today.AddDate(0, 0, -(days - 1))
	idx := map[string]int{}
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		idx[d] = i
		out.Timeline = append(out.Timeline, DayPoint{Date: d})
	}
	day := func(t time.Time) (*DayPoint, bool) {
		i, ok := idx[t.In(loc).Format("2006-01-02")]
		if !ok {
			return nil, false
		}
		return &out.Timeline[i], true
	}

	cats := map[string]*CategoryStat{}
	var bugsDone, algoDone, done []recapTodo
	scoreSum := 0
	for _, t := range todos {
		completed := t.Status == models.TodoStatusCompleted
		open := t.Status == models.TodoStatusPending || t.Status == models.TodoStatusInProgress
		if t.Status == models.TodoStatusSuggested {
			out.Todos.Suggested++
		}
		if !completed && !open {
			continue // rejected, merged, split: bukan beban kerja nyata
		}
		cat := t.Category
		if cat == "" {
			cat = models.CategoryGeneral
		}
		cs := cats[cat]
		if cs == nil {
			cs = &CategoryStat{Category: cat}
			cats[cat] = cs
		}
		if d, ok := day(t.CreatedAt); ok {
			d.Created++
			out.Todos.CreatedInPeriod++
		}
		bug, algo := isBugTodo(t), isAlgoTodo(t)
		if open {
			out.Todos.Open++
			cs.Open++
			if t.Status == models.TodoStatusInProgress {
				out.Todos.InProgress++
			}
			if t.Priority == models.TodoPriorityHigh || t.Priority == models.TodoPriorityCritical {
				out.Todos.OpenHighPriority++
			}
			if bug {
				out.Bugs.Open++
			}
			if algo {
				out.Algorithm.Open++
			}
			continue
		}
		out.Todos.Done++
		cs.Done++
		done = append(done, t)
		if t.Score != nil {
			scoreSum += *t.Score
			out.Todos.ScoredCount++
		}
		inPeriod := false
		if t.CompletedAt != nil {
			if d, ok := day(*t.CompletedAt); ok {
				inPeriod = true
				d.Completed++
				out.Todos.DoneInPeriod++
				if bug {
					d.BugsClosed++
				}
			}
		}
		if bug {
			out.Bugs.Done++
			bugsDone = append(bugsDone, t)
			if inPeriod {
				out.Bugs.DoneInPeriod++
			}
		}
		if algo {
			out.Algorithm.Done++
			algoDone = append(algoDone, t)
			if inPeriod {
				out.Algorithm.DoneInPeriod++
			}
		}
	}
	out.Todos.Total = out.Todos.Done + out.Todos.Open
	if out.Todos.Total > 0 {
		out.Todos.Percent = out.Todos.Done * 100 / out.Todos.Total
	}
	if out.Todos.ScoredCount > 0 {
		out.Todos.AvgScore = scoreSum / out.Todos.ScoredCount
	}
	for _, cs := range cats {
		out.Categories = append(out.Categories, *cs)
	}
	sort.Slice(out.Categories, func(i, j int) bool {
		a, b := out.Categories[i], out.Categories[j]
		if a.Done+a.Open != b.Done+b.Open {
			return a.Done+a.Open > b.Done+b.Open
		}
		return a.Category < b.Category
	})
	out.Highlights = latest(done)
	out.Bugs.Latest = latest(bugsDone)
	out.Algorithm.Latest = latest(algoDone)

	// Todo terbuka di akhir tiap hari: mundur dari jumlah terbuka saat ini.
	open := out.Todos.Open
	for i := len(out.Timeline) - 1; i >= 0; i-- {
		out.Timeline[i].Open = open
		open = open - out.Timeline[i].Created + out.Timeline[i].Completed
		if open < 0 {
			open = 0
		}
	}

	if err := fillFindings(db, projectID, out, day); err != nil {
		return nil, err
	}
	return out, nil
}

func latest(ts []recapTodo) []TodoBrief {
	sort.SliceStable(ts, func(i, j int) bool {
		a, b := ts[i].CompletedAt, ts[j].CompletedAt
		switch {
		case a == nil || b == nil:
			return a != nil
		case !a.Equal(*b):
			return a.After(*b)
		}
		return ts[i].ID > ts[j].ID
	})
	out := []TodoBrief{}
	for i := 0; i < len(ts) && i < recapLatest; i++ {
		out = append(out, ts[i].brief())
	}
	return out
}

// fillFindings membaca skor tiap audit (tren kesehatan) dan temuan audit TERBARU per aplikasi.
func fillFindings(db *gorm.DB, projectID uint, out *OwnerRecap, day func(time.Time) (*DayPoint, bool)) error {
	var scores []models.Diagnostic
	if err := db.Model(&models.Diagnostic{}).Select("id", "component_id", "score", "created_at").
		Where("project_id = ?", projectID).Order("created_at DESC, id DESC").Limit(recapHealthMax).Find(&scores).Error; err != nil {
		return err
	}
	latestID := map[uint]uint{} // component (0 = proyek) → audit terbaru
	for i := len(scores) - 1; i >= 0; i-- {
		d := scores[i]
		out.Health = append(out.Health, HealthPoint{Date: d.CreatedAt.Format("2006-01-02"), Score: d.Score})
		cid := uint(0)
		if d.ComponentID != nil {
			cid = *d.ComponentID
		}
		latestID[cid] = d.ID
	}
	if len(latestID) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(latestID))
	for _, id := range latestID {
		ids = append(ids, id)
	}
	var full []models.Diagnostic
	if err := db.Model(&models.Diagnostic{}).Select("id", "findings_json").Where("id IN ?", ids).Find(&full).Error; err != nil {
		return err
	}
	for _, d := range full {
		var fs []models.Finding
		if d.FindingsJSON == "" || json.Unmarshal([]byte(d.FindingsJSON), &fs) != nil {
			continue
		}
		for _, f := range fs {
			switch f.Status {
			case models.FindingStatusResolved:
				out.Findings.Resolved++
				if f.ResolvedAt != nil {
					if p, ok := day(*f.ResolvedAt); ok {
						p.FindingsResolved++
						out.Findings.ResolvedInPeriod++
					}
				}
			case models.FindingStatusIgnored:
				out.Findings.Ignored++
			default:
				out.Findings.Open++
				sev := strings.ToUpper(f.Severity)
				out.Findings.OpenBySeverity[sev]++
				if sev == "CRITICAL" {
					out.Findings.OpenCritical++
				}
			}
		}
	}
	return nil
}
