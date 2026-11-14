package insight

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
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
	Sources   int           `json:"sources"`
	InTodo    int           `json:"in_todo"`    // jumlah SUMBER yang ditangani todo
	TodoCount int           `json:"todo_count"` // jumlah TODO unik (satu todo bisa menampung banyak sumber satu berkas)
	InAudit   int           `json:"in_audit"`
	Ignored   int           `json:"ignored"`
	Recheck   int           `json:"recheck"`
	Open      int           `json:"open"`
	Todos     []CostTodoRef `json:"todos"`
	AuditAt   *time.Time    `json:"audit_at,omitempty"`
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
	// Todo gabungan lintas berkas satu ekosistem ("*go", "*php") tidak dihitung menangani sumber biaya tertentu:
	// isinya bisa hal lain (mis. memvalidasi false positive), jadi sumbernya tetap perlu todo per berkas.
	if strings.HasPrefix(target, "*") {
		return false
	}
	return samePath(target, file)
}

// isWildcardKey: kunci audit todo gabungan satu ekosistem ("QUERY-NPLUS1|*go").
func isWildcardKey(key string) bool {
	i := strings.Index(key, "|")
	return i >= 0 && strings.HasPrefix(key[i+1:], "*")
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
				if isWildcardKey(t.AuditKey) {
					return mk(HandledAudit, nil, fmt.Sprintf("Temuan ini hanya masuk todo gabungan #%d (lintas berkas). Buat todo khusus berkas ini dari sini.", t.ID))
				}
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
		if got.TodoID != 0 && got.Where == HandledTodo && !seenTodo[got.TodoID] {
			seenTodo[got.TodoID] = true
			sum.TodoCount++
			if len(sum.Todos) < maxSummaryTodos {
				sum.Todos = append(sum.Todos, CostTodoRef{ID: got.TodoID, Title: got.TodoTitle, Status: got.TodoStatus})
			}
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

// CreateCostTodo menjadikan sumber biaya tinggi sebagai todo. Semua sumber biaya di berkas yang sama (N+1
// maupun O(n²), fungsi mana pun) dilebur menjadi SATU todo: deskripsinya dimulai dari fungsi sumber, lalu naik
// ke rute yang menjangkaunya. Aturan audit lain di berkas itu dicatat sebagai sub todo "merged" (kunci audit
// "<aturan>|<berkas>") agar audit dan tab Biaya Proses menunjuk todo yang sama. Bila sudah ada todo terbuka
// untuk salah satu sumber itu, sumber lainnya dilebur ke todo tersebut (existed = true).
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
	return createCostTodoFrom(db, projectID, a, loadCostLedger(db, projectID), *h, file)
}

// CostTodoBatch = hasil "Jadikan semua todo" di halaman Biaya Proses.
type CostTodoBatch struct {
	Created int           `json:"created"` // todo baru
	Merged  int           `json:"merged"`  // dilebur ke todo terbuka yang sudah ada
	Files   int           `json:"files"`   // berkas yang diproses (satu todo per berkas)
	Todos   []CostTodoRef `json:"todos"`
}

// CreateAllCostTodos menjadikan todo semua sumber biaya yang belum punya todo terbuka (belum ditangani, atau baru
// tercatat sebagai temuan audit), satu todo per berkas — sama dengan menekan "Jadikan todo" di tiap baris. Sumber
// yang diabaikan atau menunggu cek ulang dilewati. Tanpa LLM.
func CreateAllCostTodos(db *gorm.DB, projectID uint) (*CostTodoBatch, error) {
	a, err := analyze(db, projectID)
	if err != nil {
		return nil, err
	}
	l := loadCostLedger(db, projectID)
	out := &CostTodoBatch{}
	done := map[string]bool{}
	for _, h := range a.hots {
		if done[h.Func.File] {
			continue
		}
		if got := l.handling(h); got != nil && got.Where != HandledAudit {
			continue
		}
		done[h.Func.File] = true
		t, existed, err := createCostTodoFrom(db, projectID, a, l, h, h.Func.File)
		if err != nil {
			return out, err
		}
		out.Files++
		if existed {
			out.Merged++
		} else {
			out.Created++
		}
		out.Todos = append(out.Todos, CostTodoRef{ID: t.ID, Title: t.Title, Status: t.Status})
	}
	return out, nil
}

func createCostTodoFrom(db *gorm.DB, projectID uint, a *costAnalysis, l *costLedger, h routemap.Hotspot, file string) (*models.Todo, bool, error) {
	group := costGroup(a.hots, h)

	// Todo terbuka yang sudah menangani salah satu sumber di berkas ini → sumber lain dilebur ke sana.
	// Todo gabungan lintas berkas ("<aturan>|*go") tidak menjadi tempat lebur: todo dari Biaya Proses menunjuk
	// berkas tertentu, sedangkan todo gabungan bisa berisi hal lain (mis. memvalidasi false positive).
	for _, g := range group {
		got := l.handling(g)
		if got == nil || got.TodoID == 0 || got.Where != HandledTodo || isWildcardKey(l.byID[got.TodoID].AuditKey) {
			continue
		}
		var t models.Todo
		if db.First(&t, got.TodoID).Error != nil {
			continue
		}
		if t.Source == "process_cost" && (t.Status == models.TodoStatusPending || t.Status == models.TodoStatusSuggested) {
			t.Title, t.Description = costGroupTitle(group), costGroupDescription(group)
			t.Priority = costGroupPriority(group, t.Priority)
			if err := db.Model(&t).Select("title", "description", "priority").Updates(&t).Error; err != nil {
				return nil, false, err
			}
		}
		if err := mergeCostRules(db, t, group, file); err != nil {
			return nil, false, err
		}
		return &t, true, nil
	}

	var project models.Project
	if err := db.Preload("Components").First(&project, projectID).Error; err != nil {
		return nil, false, err
	}
	rule := diagnostics.HotspotRule(h.Kind)
	todo := models.Todo{
		Level: models.TodoLevelProject, ProjectID: &project.ID, ProjectName: project.Name,
		TargetApps:  fmt.Sprintf("%s (%s:L%d)", project.Name, filepath.Base(file), group[0].Func.Line),
		Title:       costGroupTitle(group),
		Description: costGroupDescription(group),
		Status:      models.TodoStatusPending, Priority: costGroupPriority(group, models.TodoPriorityMedium), Category: models.CategoryPerformance,
		CreatorType: models.CreatorHuman, CreatorName: "Biaya Proses",
		GuidelineRef: rule, Source: "process_cost", AuditKey: costKey(rule, file),
		Tags:      models.TagOptimasiQuery,
		Rationale: "Dihitung statis dari rumus tick T(n) & query Q(n) di tab Biaya Proses (batas atas kasus terburuk).",
	}
	if c := componentOf(project, a.root, file); c != nil {
		todo.Level, todo.ComponentID, todo.ComponentName = models.TodoLevelComponent, &c.ID, c.Name
	}
	if err := db.Create(&todo).Error; err != nil {
		return nil, false, err
	}
	if err := mergeCostRules(db, todo, group, file); err != nil {
		return nil, false, err
	}
	return &todo, false, nil
}

func costKey(rule, file string) string { return rule + "|" + filepath.ToSlash(filepath.Clean(file)) }

// costGroup: semua sumber biaya di berkas yang sama dengan h, urut dari bawah rantai pemanggilan ke atas.
// Pangkat sumber biaya selalu lebih tinggi daripada fungsi yang dipanggilnya, jadi pangkat terendah (fungsi
// terdalam) dikerjakan dulu, baru pemanggilnya; seri diurutkan baris. Satu fungsi yang punya N+1 sekaligus O(n²)
// tetap berdampingan (N+1 dulu) supaya dilaporkan sebagai satu fungsi.
func costGroup(hots []routemap.Hotspot, h routemap.Hotspot) []routemap.Hotspot {
	var out []routemap.Hotspot
	for _, x := range hots {
		if x.Func.Module == h.Func.Module && x.Func.File == h.Func.File {
			out = append(out, x)
		}
	}
	degree := map[int]int{} // pangkat tertinggi per fungsi (baris), agar N+1 & O(n²) satu fungsi tidak terpisah
	for _, x := range out {
		if d := max(x.Func.Degree, 1); d > degree[x.Func.Line] {
			degree[x.Func.Line] = d
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Func, out[j].Func
		if degree[a.Line] != degree[b.Line] {
			return degree[a.Line] < degree[b.Line]
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return out[i].Kind == "nplus1" && out[j].Kind != "nplus1"
	})
	return out
}

// costFunc = satu fungsi sumber beserta semua jenis biayanya.
type costFunc struct {
	hots []routemap.Hotspot
}

func groupByFunc(group []routemap.Hotspot) []costFunc {
	var out []costFunc
	idx := map[int]int{}
	for _, h := range group {
		if i, ok := idx[h.Func.Line]; ok {
			out[i].hots = append(out[i].hots, h)
			continue
		}
		idx[h.Func.Line] = len(out)
		out = append(out, costFunc{hots: []routemap.Hotspot{h}})
	}
	return out
}

func costGroupPriority(group []routemap.Hotspot, current string) string {
	for _, h := range group {
		if h.Kind == "nplus1" || h.Func.Degree >= 3 {
			return models.TodoPriorityHigh
		}
	}
	return current
}

func funcShortName(h routemap.Hotspot) string {
	name := h.Func.Name
	if i := strings.LastIndex(name, "."); i >= 0 && h.Func.Lang == "go" {
		name = name[i+1:]
	}
	return name
}

func costGroupTitle(group []routemap.Hotspot) string {
	if len(group) == 1 {
		return costTodoTitle(group[0])
	}
	fns := groupByFunc(group)
	if len(fns) == 1 {
		h := fns[0].hots[0]
		big := h.Func.BigO
		for _, x := range fns[0].hots {
			if x.Kind == "growth" {
				big = x.Func.BigO
			}
		}
		return fmt.Sprintf("Hilangkan N+1 query & turunkan biaya %s di %s", big, funcShortName(h))
	}
	names := make([]string, 0, 3)
	for i, f := range fns {
		if i == 3 {
			names = append(names, "…")
			break
		}
		names = append(names, funcShortName(f.hots[0]))
	}
	return fmt.Sprintf("Turunkan biaya proses di %s (%d fungsi: %s)", filepath.Base(group[0].Func.File), len(fns), strings.Join(names, ", "))
}

func costTodoTitle(h routemap.Hotspot) string {
	name := funcShortName(h)
	if h.Kind == "nplus1" {
		if !routemap.OnlyQueries(h.Func.IOKinds) {
			return fmt.Sprintf("Keluarkan I/O per item (%s) dari loop di %s (Q(n) = %s)", strings.Join(routemap.IOLabels(h.Func.IOKinds), ", "), name, h.Func.QueryFormula)
		}
		return fmt.Sprintf("Hilangkan N+1 query di %s (Q(n) = %s)", name, h.Func.QueryFormula)
	}
	return fmt.Sprintf("Turunkan biaya %s di %s", h.Func.BigO, name)
}

const maxCostTodoRoutes = 30

// costGroupDescription: langkah dari bawah ke atas — fungsi sumber dulu (tempat loop/query ditulis), lalu rute
// yang menjangkaunya untuk diperiksa ulang setelah fungsinya diperbaiki.
func costGroupDescription(group []routemap.Hotspot) string {
	var b strings.Builder
	fns := groupByFunc(group)
	file := group[0].Func.File
	routeFuncs := map[string][]string{}
	for _, f := range fns {
		name := funcShortName(f.hots[0])
		seen := map[string]bool{}
		for _, h := range f.hots {
			for _, r := range h.AllRoutes {
				if !seen[r] {
					seen[r] = true
					routeFuncs[r] = append(routeFuncs[r], name)
				}
			}
		}
	}
	routes := make([]string, 0, len(routeFuncs))
	for r := range routeFuncs {
		routes = append(routes, r)
	}
	sort.Strings(routes)

	if len(group) > 1 {
		fmt.Fprintf(&b, "Sumber biaya tinggi di `%s` dilebur menjadi satu todo: %d fungsi, %d rute terdampak. ", file, len(fns), len(routes))
	}
	b.WriteString("Kerjakan dari bawah ke atas: perbaiki fungsi sumbernya dulu, baru naik ke rute yang memanggilnya.\n\n")

	b.WriteString("## 1. Mulai dari fungsi\n\n")
	var theories []string
	for i, f := range fns {
		h0 := f.hots[0]
		fmt.Fprintf(&b, "### 1.%d `%s` — `%s:%d`\n\n", i+1, h0.Func.Name, h0.Func.File, h0.Func.Line)
		for _, h := range f.hots {
			fd := diagnostics.HotspotFinding(h, "")
			label := "Pertumbuhan " + h.Func.BigO
			if h.Kind == "nplus1" {
				label = "N+1 query"
			}
			fmt.Fprintf(&b, "**%s.** %s\n\n", label, fd.Description)
			fmt.Fprintf(&b, "**Rumus saat ini:** T(n) = %s · Q(n) = %s · %s\n\n", h.Func.Formula, h.Func.QueryFormula, h.Func.BigO)
			fmt.Fprintf(&b, "**Langkah:** %s\n\n", fd.Suggestion)
			if fd.TheoryBasis != "" && !containsStr(theories, fd.TheoryBasis) {
				theories = append(theories, fd.TheoryBasis)
			}
		}
	}

	if len(routes) > 0 {
		b.WriteString("## 2. Naik ke rute\n\n")
		b.WriteString("Setelah fungsi di atas diperbaiki, periksa tiap rute yang menjangkaunya (handler, middleware, dan pemanggilan lain di jalurnya) agar biayanya tidak naik lagi di tempat lain, lalu klik **Cek ulang Big-O** di tab Biaya Proses.\n\n")
	} else {
		b.WriteString("## 2. Periksa pemanggilnya\n\n")
	}
	for i, r := range routes {
		if i == maxCostTodoRoutes {
			fmt.Fprintf(&b, "- … dan %d rute lain\n", len(routes)-maxCostTodoRoutes)
			break
		}
		fmt.Fprintf(&b, "- `%s` (lewat `%s`)\n", r, strings.Join(routeFuncs[r], "`, `"))
	}
	if len(routes) > 0 {
		b.WriteString("\n")
	} else {
		b.WriteString("Tidak ada rute HTTP yang menjangkau fungsi ini: ia dijalankan di luar rute (worker antrean, goroutine/job dari main atau init). Periksa pemanggil & pendaftarnya (mis. `RegisterHandler`, `go …`) setelah fungsinya diperbaiki.\n\n")
	}
	b.WriteString("**Selesai bila:** tab Biaya Proses tidak lagi menunjuk fungsi-fungsi ini sebagai sumber biaya (kelasnya turun atau N+1 hilang), rute di atas ikut turun, dan test yang ada tetap lulus.\n\n")
	if len(theories) > 0 {
		fmt.Fprintf(&b, "**Landasan:** %s\n", strings.Join(theories, " "))
	}
	return b.String()
}

func containsStr(in []string, s string) bool {
	for _, x := range in {
		if x == s {
			return true
		}
	}
	return false
}

// mergeCostRules mencatat aturan audit lain di kelompok sebagai sub todo "merged" dari t, supaya kunci audit
// "<aturan>|<berkas>" itu menunjuk todo yang sama (tidak ada todo ganda untuk satu berkas).
func mergeCostRules(db *gorm.DB, t models.Todo, group []routemap.Hotspot, file string) error {
	for _, h := range group {
		rule := diagnostics.HotspotRule(h.Kind)
		key := costKey(rule, file)
		if key == t.AuditKey {
			continue
		}
		var n int64
		db.Model(&models.Todo{}).Where("audit_key = ? AND (id = ? OR parent_id = ?) AND status <> ?", key, t.ID, t.ID, models.TodoStatusRejected).Count(&n)
		if n > 0 {
			continue
		}
		parent := t.ID
		sub := models.Todo{
			Level: t.Level, ProjectID: t.ProjectID, ProjectName: t.ProjectName, ComponentID: t.ComponentID, ComponentName: t.ComponentName,
			TargetApps: t.TargetApps, ParentID: &parent, Status: models.TodoStatusMerged,
			Title:       fmt.Sprintf("%s (dilebur ke todo #%d)", costTodoTitle(h), t.ID),
			Description: fmt.Sprintf("Dilebur ke todo #%d karena berada di berkas yang sama (`%s`). Kerjakan lewat todo induknya.", t.ID, file),
			Priority:    t.Priority, Category: models.CategoryPerformance, CreatorType: models.CreatorHuman, CreatorName: "Biaya Proses",
			GuidelineRef: rule, Source: "process_cost", AuditKey: key, Tags: models.TagOptimasiQuery,
		}
		if err := db.Create(&sub).Error; err != nil {
			return err
		}
	}
	return nil
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
