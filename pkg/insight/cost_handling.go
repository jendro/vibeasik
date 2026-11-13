package insight

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/diagnostics"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/routemap"
)

// Penanganan biaya tinggi: sumber biaya (fungsi tempat loop N+1 / O(n²) ditulis) otomatis ikut audit dengan
// aturan QUERY-NPLUS1-CHAIN / COST-POLYNOMIAL-GROWTH. Tab Biaya Proses menautkan tiap sumber ke temuan audit
// atau todo yang menanganinya; sumber yang belum tertangani bisa langsung dijadikan todo dengan kunci audit
// yang sama ("<aturan>|<berkas>"), jadi audit berikutnya menggabungkan temuannya ke todo itu, bukan membuat baru.

// Tempat penanganan sumber biaya.
const (
	HandledTodo    = "todo"    // todo terbuka (pending / sedang dikerjakan / saran)
	HandledAudit   = "audit"   // temuan audit terbuka, belum dijadikan todo
	HandledIgnored = "ignored" // ditandai bukan masalah di Audit
	HandledRecheck = "recheck" // todo/temuan sudah selesai, tetapi rumusnya masih tinggi
)

// CostSource = sumber biaya tinggi yang bisa dijadikan todo.
type CostSource struct {
	Kind  string `json:"kind"` // nplus1 | growth
	Rule  string `json:"rule"`
	Func  string `json:"func"`
	File  string `json:"file"`
	Line  int    `json:"line"`
	BigO  string `json:"big_o"`
	Query string `json:"query_formula,omitempty"`
	// Evidence: baris kode di fungsi sumber yang menaikkan biaya (bukti untuk pengguna & pengerja todo).
	Evidence []routemap.CostEvidence `json:"evidence,omitempty"`
}

// CostHandling = di mana sumber biaya sudah ditangani dan apa langkah berikutnya.
type CostHandling struct {
	Where      string     `json:"where"`
	TodoID     uint       `json:"todo_id,omitempty"`
	TodoTitle  string     `json:"todo_title,omitempty"`
	TodoStatus string     `json:"todo_status,omitempty"`
	Rule       string     `json:"rule"`
	Source     CostSource `json:"source"`
	Next       string     `json:"next"`
}

// RouteRow / FuncRow = baris tabel beserta sumber biaya yang belum ditangani dan penanganan yang sudah ada.
type RouteRow struct {
	routemap.RouteCost
	Source  *CostSource   `json:"source,omitempty"`
	Handled *CostHandling `json:"handled,omitempty"`
}

type FuncRow struct {
	routemap.FuncCost
	Source  *CostSource   `json:"source,omitempty"`
	Handled *CostHandling `json:"handled,omitempty"`
}

// CostTodoRef = todo terkait biaya proses untuk keterangan di atas tabel.
type CostTodoRef struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// CostHandlingSummary = rekap penanganan seluruh sumber biaya proyek (bukan hanya halaman yang termuat).
type CostHandlingSummary struct {
	Sources int           `json:"sources"`
	InTodo  int           `json:"in_todo"`
	InAudit int           `json:"in_audit"`
	Ignored int           `json:"ignored"`
	Recheck int           `json:"recheck"`
	Open    int           `json:"open"`
	Todos   []CostTodoRef `json:"todos"`
	AuditAt *time.Time    `json:"audit_at,omitempty"`
}

const maxSummaryTodos = 5

var openTodoStatus = map[string]bool{models.TodoStatusPending: true, models.TodoStatusInProgress: true, models.TodoStatusSuggested: true}

type costTodo struct {
	ID       uint
	Title    string
	Status   string
	AuditKey string
	ParentID *uint
}

// costLedger = todo & temuan audit biaya proses satu proyek.
type costLedger struct {
	todos    []costTodo
	byID     map[uint]costTodo
	findings []models.Finding
	auditAt  *time.Time
}

func costRulesFor(kind string) []string {
	if kind == "nplus1" {
		return []string{diagnostics.RuleNPlusOneChain, "QUERY-NPLUS1"}
	}
	return []string{diagnostics.RulePolynomialGrowth}
}

func loadCostLedger(db *gorm.DB, projectID uint) *costLedger {
	l := &costLedger{byID: map[uint]costTodo{}}
	rules := []string{diagnostics.RuleNPlusOneChain, "QUERY-NPLUS1", diagnostics.RulePolynomialGrowth}
	q := db.Model(&models.Todo{}).Select("id", "title", "status", "audit_key", "parent_id").
		Where("project_id = ? AND status <> ?", projectID, models.TodoStatusRejected)
	cond := db.Where("audit_key LIKE ?", rules[0]+"|%")
	for _, r := range rules[1:] {
		cond = cond.Or("audit_key LIKE ?", r+"|%")
	}
	_ = q.Where(cond).Order("id DESC").Limit(500).Find(&l.todos).Error
	for _, t := range l.todos {
		l.byID[t.ID] = t
	}

	// Laporan audit terbaru per aplikasi (komponen) proyek.
	var ids []uint
	_ = db.Model(&models.Diagnostic{}).Select("MAX(id)").Where("project_id = ?", projectID).Group("component_id").Pluck("MAX(id)", &ids).Error
	if len(ids) == 0 {
		return l
	}
	var diags []models.Diagnostic
	_ = db.Select("id", "findings_json", "created_at").Where("id IN ?", ids).Find(&diags).Error
	var want []uint
	for _, d := range diags {
		if l.auditAt == nil || d.CreatedAt.After(*l.auditAt) {
			at := d.CreatedAt
			l.auditAt = &at
		}
		var fs []models.Finding
		if json.Unmarshal([]byte(d.FindingsJSON), &fs) != nil {
			continue
		}
		for _, f := range fs {
			if f.RuleID == diagnostics.RuleNPlusOneChain || f.RuleID == diagnostics.RulePolynomialGrowth || f.RuleID == "QUERY-NPLUS1" {
				l.findings = append(l.findings, f)
				if f.TodoID != nil {
					if _, ok := l.byID[*f.TodoID]; !ok {
						want = append(want, *f.TodoID)
					}
				}
			}
		}
	}
	if len(want) > 0 {
		var extra []costTodo
		_ = db.Model(&models.Todo{}).Select("id", "title", "status", "audit_key", "parent_id").Where("id IN ?", want).Find(&extra).Error
		for _, t := range extra {
			l.byID[t.ID] = t
		}
	}
	return l
}

// resolveTodo: sub todo yang dilebur ke induknya dirujuk lewat induknya.
func (l *costLedger) resolveTodo(t costTodo) costTodo {
	if t.ParentID != nil && !openTodoStatus[t.Status] && t.Status != models.TodoStatusCompleted {
		if p, ok := l.byID[*t.ParentID]; ok {
			return p
		}
	}
	return t
}

// samePath: berkas audit (relatif root proyek) dan berkas biaya bisa berbeda awalan folder aplikasi.
func samePath(a, b string) bool {
	a, b = filepath.ToSlash(filepath.Clean(a)), filepath.ToSlash(filepath.Clean(b))
	return a == b || strings.HasSuffix(a, "/"+b) || strings.HasSuffix(b, "/"+a)
}

func keyMatches(key string, rules []string, file string) bool {
	i := strings.Index(key, "|")
	if i < 0 {
		return false
	}
	rule, target := key[:i], key[i+1:]
	ok := false
	for _, r := range rules {
		ok = ok || r == rule
	}
	if !ok {
		return false
	}
	if strings.HasPrefix(target, "*") { // todo gabungan lintas berkas satu ekosistem ("*go", "*php")
		return strings.TrimPrefix(filepath.Ext(file), ".") == strings.TrimPrefix(target, "*")
	}
	return samePath(target, file)
}

func sourceOf(h routemap.Hotspot) CostSource {
	return CostSource{Kind: h.Kind, Rule: diagnostics.HotspotRule(h.Kind), Func: h.Func.Name, File: h.Func.File, Line: h.Func.Line,
		BigO: h.Func.BigO, Query: h.Func.QueryFormula, Evidence: h.Func.Evidence}
}

// handling: penanganan sumber biaya h, atau nil bila belum ditangani di mana pun.
func (l *costLedger) handling(h routemap.Hotspot) *CostHandling {
	src := sourceOf(h)
	rules := costRulesFor(h.Kind)
	mk := func(where string, t *costTodo, next string) *CostHandling {
		out := &CostHandling{Where: where, Rule: src.Rule, Source: src, Next: next}
		if t != nil {
			out.TodoID, out.TodoTitle, out.TodoStatus = t.ID, t.Title, t.Status
		}
		return out
	}
	todoNext := func(t costTodo) (string, string) {
		switch t.Status {
		case models.TodoStatusInProgress:
			return HandledTodo, fmt.Sprintf("Todo #%d sedang dikerjakan. Setelah selesai, jalankan Audit ulang dan muat ulang tab ini.", t.ID)
		case models.TodoStatusSuggested:
			return HandledTodo, fmt.Sprintf("Setujui saran todo #%d di halaman Todo, lalu kerjakan atau kirim ke agent.", t.ID)
		case models.TodoStatusCompleted:
			return HandledRecheck, fmt.Sprintf("Todo #%d sudah selesai, tetapi rumusnya masih tinggi. Jalankan Audit ulang; bila masih muncul, buka ulang todo itu.", t.ID)
		}
		return HandledTodo, fmt.Sprintf("Kerjakan todo #%d di halaman Todo atau kirim ke agent coding.", t.ID)
	}
	var done *costTodo
	for _, t := range l.todos {
		if !keyMatches(t.AuditKey, rules, h.Func.File) {
			continue
		}
		t = l.resolveTodo(t)
		if openTodoStatus[t.Status] {
			where, next := todoNext(t)
			return mk(where, &t, next)
		}
		if t.Status == models.TodoStatusCompleted && done == nil {
			tt := t
			done = &tt
		}
	}
	for _, f := range l.findings {
		in := false
		for _, r := range rules {
			in = in || r == f.RuleID
		}
		if !in || !samePath(f.TargetFile, h.Func.File) || (f.Line != 0 && h.Func.End > 0 && (f.Line < h.Func.Line || f.Line > h.Func.End)) {
			continue
		}
		if f.TodoID != nil {
			if t, ok := l.byID[*f.TodoID]; ok {
				t = l.resolveTodo(t)
				where, next := todoNext(t)
				return mk(where, &t, next)
			}
		}
		switch f.Status {
		case models.FindingStatusIgnored:
			return mk(HandledIgnored, nil, "Ditandai bukan masalah di Audit. Tidak perlu tindakan; buka lagi temuannya bila datanya ternyata besar.")
		case models.FindingStatusResolved:
			return mk(HandledRecheck, nil, "Temuan ditandai selesai di Audit, tetapi rumusnya masih tinggi. Jalankan Audit ulang untuk memastikan.")
		}
		return mk(HandledAudit, nil, "Sudah tercatat di Audit. Buka Audit lalu klik Selesaikan pada temuan ini agar menjadi todo, atau buat todo langsung dari sini.")
	}
	if done != nil {
		where, next := todoNext(*done)
		return mk(where, done, next)
	}
	return nil
}

// annotate mengisi sumber & penanganan tiap baris dan merangkum penanganan seluruh proyek.
func annotate(hots []routemap.Hotspot, l *costLedger) (func(routemap.RouteCost) RouteRow, func(routemap.FuncCost) FuncRow, CostHandlingSummary) {
	type hs struct {
		h   routemap.Hotspot
		got *CostHandling
	}
	all := make([]hs, 0, len(hots))
	byRoute := map[string][]int{}
	byFunc := map[string][]int{}
	sum := CostHandlingSummary{Todos: []CostTodoRef{}, AuditAt: l.auditAt}
	seenTodo := map[uint]bool{}
	for _, h := range hots {
		got := l.handling(h)
		i := len(all)
		all = append(all, hs{h, got})
		for _, r := range h.AllRoutes {
			byRoute[h.Func.Module+" "+r] = append(byRoute[h.Func.Module+" "+r], i)
		}
		fk := fmt.Sprintf("%s|%s|%d", h.Func.Module, h.Func.File, h.Func.Line)
		byFunc[fk] = append(byFunc[fk], i)
		sum.Sources++
		if got == nil {
			sum.Open++
			continue
		}
		switch got.Where {
		case HandledTodo:
			sum.InTodo++
		case HandledAudit:
			sum.InAudit++
		case HandledIgnored:
			sum.Ignored++
		case HandledRecheck:
			sum.Recheck++
		}
		if got.TodoID != 0 && got.Where == HandledTodo && !seenTodo[got.TodoID] && len(sum.Todos) < maxSummaryTodos {
			seenTodo[got.TodoID] = true
			sum.Todos = append(sum.Todos, CostTodoRef{ID: got.TodoID, Title: got.TodoTitle, Status: got.TodoStatus})
		}
	}
	pick := func(idx []int) (*CostSource, *CostHandling) {
		var src *CostSource
		var got *CostHandling
		for _, i := range idx {
			if all[i].got == nil {
				if src == nil {
					s := sourceOf(all[i].h)
					src = &s
				}
			} else if got == nil {
				got = all[i].got
			}
		}
		return src, got
	}
	route := func(r routemap.RouteCost) RouteRow {
		src, got := pick(byRoute[r.Module+" "+r.Method+" "+r.Path])
		return RouteRow{RouteCost: r, Source: src, Handled: got}
	}
	fn := func(f routemap.FuncCost) FuncRow {
		src, got := pick(byFunc[fmt.Sprintf("%s|%s|%d", f.Module, f.File, f.Line)])
		return FuncRow{FuncCost: f, Source: src, Handled: got}
	}
	return route, fn, sum
}

// ErrCostSourceGone: sumber biaya yang diminta tidak ada lagi (kode sudah berubah).
var ErrCostSourceGone = errors.New("sumber biaya ini tidak ada lagi; muat ulang tab Biaya Proses")

// CreateCostTodo menjadikan satu sumber biaya tinggi sebagai todo. Bila sudah ada todo terbuka untuk sumber
// itu, todo tersebut yang dikembalikan (existed = true).
func CreateCostTodo(db *gorm.DB, projectID uint, file string, line int, kind string) (*models.Todo, bool, error) {
	a, err := analyze(db, projectID)
	if err != nil {
		return nil, false, err
	}
	var h *routemap.Hotspot
	for i := range a.hots {
		if a.hots[i].Kind == kind && a.hots[i].Func.File == file && a.hots[i].Func.Line == line {
			h = &a.hots[i]
			break
		}
	}
	if h == nil {
		return nil, false, ErrCostSourceGone
	}
	l := loadCostLedger(db, projectID)
	if got := l.handling(*h); got != nil && got.TodoID != 0 && got.Where == HandledTodo {
		var t models.Todo
		if db.First(&t, got.TodoID).Error == nil {
			return &t, true, nil
		}
	}

	var project models.Project
	if err := db.Preload("Components").First(&project, projectID).Error; err != nil {
		return nil, false, err
	}
	f := diagnostics.HotspotFinding(*h, "")
	todo := models.Todo{
		Level: models.TodoLevelProject, ProjectID: &project.ID, ProjectName: project.Name,
		TargetApps:  fmt.Sprintf("%s (%s:L%d)", project.Name, filepath.Base(file), line),
		Title:       costTodoTitle(*h),
		Description: costTodoDescription(*h, f),
		Status:      models.TodoStatusPending, Priority: models.TodoPriorityMedium, Category: models.CategoryPerformance,
		CreatorType: models.CreatorHuman, CreatorName: "Biaya Proses",
		GuidelineRef: f.RuleID, Source: "process_cost", AuditKey: f.RuleID + "|" + filepath.ToSlash(filepath.Clean(file)),
		Tags:      models.TagOptimasiQuery,
		Rationale: "Dihitung statis dari rumus tick T(n) & query Q(n) di tab Biaya Proses (batas atas kasus terburuk).",
	}
	if h.Kind == "nplus1" || h.Func.Degree >= 3 {
		todo.Priority = models.TodoPriorityHigh
	}
	if c := componentOf(project, a.root, file); c != nil {
		todo.Level, todo.ComponentID, todo.ComponentName = models.TodoLevelComponent, &c.ID, c.Name
	}
	if err := db.Create(&todo).Error; err != nil {
		return nil, false, err
	}
	return &todo, false, nil
}

func costTodoTitle(h routemap.Hotspot) string {
	name := h.Func.Name
	if i := strings.LastIndex(name, "."); i >= 0 && h.Func.Lang == "go" {
		name = name[i+1:]
	}
	if h.Kind == "nplus1" {
		return fmt.Sprintf("Hilangkan N+1 query di %s (Q(n) = %s)", name, h.Func.QueryFormula)
	}
	return fmt.Sprintf("Turunkan biaya %s di %s", h.Func.BigO, name)
}

func costTodoDescription(h routemap.Hotspot, f models.Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n**Lokasi:** `%s:%d` (fungsi `%s`)\n\n", f.Description, h.Func.File, h.Func.Line, h.Func.Name)
	fmt.Fprintf(&b, "**Rumus saat ini:** T(n) = %s · Q(n) = %s · %s\n\n", h.Func.Formula, h.Func.QueryFormula, h.Func.BigO)
	fmt.Fprintf(&b, "**Langkah:** %s\n\n", f.Suggestion)
	b.WriteString("**Selesai bila:** tab Biaya Proses tidak lagi menunjuk fungsi ini sebagai sumber biaya (kelasnya turun atau N+1 hilang), dan test yang ada tetap lulus.\n\n")
	fmt.Fprintf(&b, "**Landasan:** %s\n", f.TheoryBasis)
	return b.String()
}

// componentOf: aplikasi (komponen) yang foldernya memuat berkas relatif root proyek.
func componentOf(p models.Project, root, file string) *models.Component {
	var best *models.Component
	bestLen := -1
	for i := range p.Components {
		c := &p.Components[i]
		if c.AbsolutePath == "" {
			continue
		}
		rel, err := filepath.Rel(root, c.AbsolutePath)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		rel = filepath.ToSlash(rel)
		if (rel == "." || strings.HasPrefix(file, rel+"/")) && len(rel) > bestLen {
			best, bestLen = c, len(rel)
		}
	}
	return best
}
