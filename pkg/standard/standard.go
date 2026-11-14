// Package standard menjalankan ciri khas vibeasik pada setiap proyek (Aturan 45): blok port otomatis
// (PORTS.md), berkas env lokal/server yang dibuat engine, dan todo pending untuk berkas yang dikerjakan agent
// (dev.sh, dev-mobile.sh untuk proyek Flutter, deploy.sh, .gitignore env, .env.example) serta perbaikan kode default (URL API, duplicate import).
// Dipicu saat proyek/komponen dibuat (hook models.ProjectChanged), setiap audit, "Saran Todo", dan saat engine start.
package standard

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/diagnostics"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/runner"
)

// Result = apa yang dilakukan Ensure untuk satu proyek.
type Result struct {
	ProjectID    uint          `json:"project_id"`
	PortsChanged int           `json:"ports_changed"`
	EnvActions   []string      `json:"env_actions,omitempty"`
	Todos        []models.Todo `json:"todos,omitempty"` // todo standar yang terbuka (baru atau diperbarui)
	Closed       []uint        `json:"closed,omitempty"`
}

// recentDone: todo standar yang baru selesai tidak dibuat ulang selama ini, karena hasil agent di worktree
// baru masuk folder proyek setelah run merge.
const recentDone = 24 * time.Hour

// StandardTitle = judul todo gabungan standar berkas (stabil per proyek).
func StandardTitle(projectName string) string {
	return fmt.Sprintf("Lengkapi standar vibeasik: dev.sh, deploy.sh & env (%s)", projectName)
}

// Ensure menyiapkan port, env, dan todo standar satu proyek. Proyek non-aktif hanya diurus portnya.
func Ensure(db *gorm.DB, projectID uint) (*Result, error) {
	ensureMu.Lock() // pemicu bersamaan (hook, audit, start) tidak boleh membuat todo ganda
	defer ensureMu.Unlock()
	res := &Result{ProjectID: projectID}
	changed, err := runner.EnsureProjectPorts(db, projectID)
	if err != nil {
		return res, err
	}
	res.PortsChanged = len(changed)

	var p models.Project
	if err := db.Preload("Components", func(q *gorm.DB) *gorm.DB { return q.Order("id ASC") }).First(&p, projectID).Error; err != nil {
		return res, err
	}
	if p.Status != "" && p.Status != models.StatusActive {
		return res, nil
	}
	if p.PortSlot != nil && *p.PortSlot < 0 {
		return res, nil // vibeasik sendiri: punya dev.sh/deploy.sh & port 87xx sendiri (Aturan 8)
	}
	res.EnvActions = diagnostics.EnsureEnvFiles(&p)

	if t, ok := standardTodo(&p); ok {
		saved, err := upsert(db, t)
		if err != nil {
			return res, err
		}
		if saved != nil {
			res.Todos = append(res.Todos, *saved)
		}
	} else if id := closeStandardTodo(db, &p); id > 0 {
		res.Closed = append(res.Closed, id)
	}
	fresh := diagnostics.CodeTodos(&p)
	for _, t := range fresh {
		saved, err := upsert(db, t)
		if err != nil {
			return res, err
		}
		if saved != nil {
			res.Todos = append(res.Todos, *saved)
		}
	}
	res.Closed = append(res.Closed, closeGoneCodeTodos(db, &p, fresh)...)
	return res, nil
}

// closeGoneCodeTodos menutup todo kode standar yang masih terbuka tetapi masalahnya tidak lagi terdeteksi di disk
// (sudah diperbaiki, atau pemeriksaannya dulu keliru). Pemeriksaan ini statis, jadi "tidak muncul lagi" = dasar
// todonya hilang; tanpa penutupan, todo lama tetap pending dan dikerjakan agent untuk masalah yang tidak ada.
func closeGoneCodeTodos(db *gorm.DB, p *models.Project, fresh []models.Todo) []uint {
	keep := map[string]bool{}
	for _, t := range fresh {
		keep[codeTodoKey(t.Title, t.ComponentID)] = true
	}
	var open []models.Todo
	db.Select("id", "title", "component_id", "rationale").
		Where("project_id = ? AND source = ? AND title <> ? AND status IN ?", p.ID, "vibeasik_standard", StandardTitle(p.Name),
			[]string{models.TodoStatusPending, models.TodoStatusSuggested}).Find(&open)
	onDisk := map[uint]bool{}
	for _, c := range p.Components {
		if c.AbsolutePath != "" {
			if fi, err := os.Stat(c.AbsolutePath); err == nil && fi.IsDir() {
				onDisk[c.ID] = true
			}
		}
	}
	var closed []uint
	now := time.Now()
	for _, t := range open {
		// Hanya todo komponen yang foldernya terbaca: folder hilang sementara ≠ masalah sudah beres.
		if t.ComponentID == nil || !onDisk[*t.ComponentID] || keep[codeTodoKey(t.Title, t.ComponentID)] {
			continue
		}
		db.Model(&models.Todo{}).Where("id = ?", t.ID).Updates(map[string]interface{}{
			"status": models.TodoStatusCompleted, "completed_at": &now,
			"rationale": strings.TrimSpace(t.Rationale + "\n\nCatatan penyelesaian: pemeriksaan standar tidak lagi menemukan masalah ini di disk (diperiksa otomatis oleh engine)."),
		})
		closed = append(closed, t.ID)
	}
	return closed
}

func codeTodoKey(title string, comp *uint) string {
	if comp == nil {
		return title + "|"
	}
	return fmt.Sprintf("%s|%d", title, *comp)
}

// standardTodo menyusun satu todo gabungan dari semua kekurangan standar berkas proyek.
func standardTodo(p *models.Project) (models.Todo, bool) {
	findings := diagnostics.StandardFindings(p)
	if len(findings) == 0 {
		return models.Todo{}, false
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Standar vibeasik (Aturan 45) untuk proyek %s belum lengkap. Kerjakan semua butir dalam satu run, satu commit.\n\n", p.Name)
	var targets []string
	examples := map[string]string{}
	for i, f := range findings {
		fmt.Fprintf(&b, "%d. **%s** — %s\n   Langkah: %s\n", i+1, f.Title, f.Description, f.Suggestion)
		if f.TargetFile != "" {
			targets = append(targets, "`"+f.TargetFile+"`")
		}
		if f.SuggestedFix != "" && f.TargetFile != "" {
			examples[f.TargetFile] = f.SuggestedFix
		}
	}
	if len(targets) > 0 {
		fmt.Fprintf(&b, "\nFile Target: %s\n", strings.Join(dedup(targets), ", "))
	}
	if ports := portLines(p); ports != "" {
		fmt.Fprintf(&b, "\nPort blok proyek (sudah dialokasikan engine, PORTS.md slot %d) — pakai di dev.sh, .env.example & URL API:\n%s", *p.PortSlot, ports)
	}
	b.WriteString("\nJangan membuat atau meng-commit .env, .env.local, .env.server: engine membuatnya sendiri (ter-ignore git); cukup pastikan .gitignore memuat `.env`, `.env.*`, `!.env.example` dan .env.example berisi semua kunci tanpa nilai rahasia.\n")
	names := make([]string, 0, len(examples))
	for n := range examples {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(&b, "\nContoh %s (sesuaikan folder komponen & perintah build):\n```bash\n%s\n```\n", n, examples[n])
	}
	return models.Todo{
		Level:        models.TodoLevelProject,
		Category:     models.CategoryDevOps,
		ProjectID:    &p.ID,
		ProjectName:  p.Name,
		TargetApps:   fmt.Sprintf("Seluruh Aplikasi (Bundle %s)", p.Name),
		Title:        StandardTitle(p.Name),
		Description:  b.String(),
		Status:       models.TodoStatusPending,
		Priority:     models.TodoPriorityHigh,
		CreatorType:  models.CreatorAI,
		CreatorName:  "vibeasik-standard",
		GuidelineRef: "ARCH-002",
		Rationale:    "Patokan: [ARCH-002][DEV-007][SEC-002] default proyek vibeasik (Aturan 45): dev.sh menjalankan semua komponen, dev-mobile.sh menjalankan Flutter ke HP (flutter-win/laptop, Aturan 49), deploy.sh memakai .env.<tujuan> lalu kembali ke .env.local, hanya .env.example yang di-commit.",
		Source:       "vibeasik_standard",
	}, true
}

func portLines(p *models.Project) string {
	if p.PortSlot == nil || *p.PortSlot < 0 {
		return ""
	}
	var b strings.Builder
	for _, c := range p.Components {
		if c.RunPort > 0 {
			fmt.Fprintf(&b, "- %s (%s, %s) → PORT=%d\n", c.Name, c.Role, c.RelativePath, c.RunPort)
		}
	}
	return b.String()
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func sameTodo(db *gorm.DB, t models.Todo) *gorm.DB {
	q := db.Model(&models.Todo{}).Where("project_id = ? AND title = ?", *t.ProjectID, t.Title)
	if t.ComponentID != nil {
		q = q.Where("component_id = ?", *t.ComponentID)
	}
	return q
}

// upsert: todo terbuka dengan judul sama diperbarui (in_progress tidak disentuh); todo yang ditolak manusia
// atau baru selesai (< recentDone) tidak dibuat ulang; selain itu dibuat baru.
func upsert(db *gorm.DB, t models.Todo) (*models.Todo, error) {
	var open models.Todo
	var merged int64
	sameTodo(db, t).Where("status = ?", models.TodoStatusMerged).Count(&merged)
	if merged > 0 {
		return nil, nil // sudah dilebur ke todo gabungan yang masih berjalan; jangan buat salinan
	}
	err := sameTodo(db, t).Where("status IN ?", []string{models.TodoStatusPending, models.TodoStatusInProgress, models.TodoStatusSuggested}).
		Order("id DESC").First(&open).Error
	if err == nil {
		if open.Status != models.TodoStatusInProgress && (open.Description != t.Description || open.Status != models.TodoStatusPending) {
			if err := db.Model(&open).Updates(map[string]interface{}{"description": t.Description, "status": models.TodoStatusPending}).Error; err != nil {
				return nil, err
			}
		}
		return &open, nil
	}
	var blocked int64
	sameTodo(db, t).Where("status = ?", models.TodoStatusRejected).Count(&blocked)
	if blocked > 0 {
		return nil, nil
	}
	// todo baru selesai hanya menahan butir yang sudah dikerjakannya; butir standar baru (mis. dev-mobile.sh) tetap dibuat
	var done []models.Todo
	sameTodo(db, t).Where("status = ? AND completed_at > ?", models.TodoStatusCompleted, time.Now().Add(-recentDone)).Find(&done)
	for _, d := range done {
		if covers(d.Description, t.Description) {
			return nil, nil
		}
	}
	if err := db.Create(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

var itemRe = regexp.MustCompile(`\*\*([^*]+)\*\*`)

// covers: semua butir (**judul temuan**) todo baru sudah ada di todo lama. Todo tanpa butir (todo kode,
// judulnya sudah memuat berkas sasaran) dianggap sama bila judulnya sama.
func covers(oldDesc, newDesc string) bool {
	for _, m := range itemRe.FindAllStringSubmatch(newDesc, -1) {
		if !strings.Contains(oldDesc, "**"+m[1]+"**") {
			return false
		}
	}
	return true
}

// closeStandardTodo menyelesaikan todo gabungan pending bila standar berkas sudah lengkap di disk.
func closeStandardTodo(db *gorm.DB, p *models.Project) uint {
	var open models.Todo
	if db.Where("project_id = ? AND title = ? AND status IN ?", p.ID, StandardTitle(p.Name),
		[]string{models.TodoStatusPending, models.TodoStatusSuggested}).First(&open).Error != nil {
		return 0
	}
	now := time.Now()
	db.Model(&open).Updates(map[string]interface{}{
		"status":       models.TodoStatusCompleted,
		"completed_at": &now,
		"rationale":    open.Rationale + "\n\nCatatan penyelesaian: standar sudah lengkap di disk (diperiksa otomatis oleh engine).",
	})
	return open.ID
}

// ---------------------------------------------------------------------------------------------
// Pemicu otomatis
// ---------------------------------------------------------------------------------------------

var (
	ensureMu sync.Mutex
	mu       sync.Mutex
	pending  = map[uint]*time.Timer{}
	debounce = 3 * time.Second
)

// Init memasang hook pembuatan proyek/komponen dan menjalankan Ensure untuk semua proyek di latar belakang
// (migrasi port lama ke blok slot + env + todo standar). Dipanggil sekali dari `vibeasik serve`.
func Init(db *gorm.DB) {
	models.ProjectChanged = func(projectID uint) { Schedule(db, projectID) }
	go func() {
		if changed, err := runner.EnsureAllProjectPorts(db); err != nil {
			log.Printf("standard: alokasi blok port gagal: %v", err)
		} else if len(changed) > 0 {
			log.Printf("standard: %d komponen dipindah ke port blok proyeknya (PORTS.md)", len(changed))
		}
		var ids []uint
		db.Model(&models.Project{}).Where("status = ? OR status = ''", models.StatusActive).Order("id ASC").Pluck("id", &ids)
		for _, id := range ids {
			if _, err := Ensure(db, id); err != nil {
				log.Printf("standard: proyek #%d: %v", id, err)
			}
		}
	}()
}

// Schedule menjalankan Ensure beberapa detik setelah perubahan terakhir (scan menambah banyak komponen
// berturut-turut; transaksi pembuatan juga sudah selesai saat itu).
func Schedule(db *gorm.DB, projectID uint) {
	if projectID == 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if t, ok := pending[projectID]; ok {
		t.Stop()
	}
	pending[projectID] = time.AfterFunc(debounce, func() {
		mu.Lock()
		delete(pending, projectID)
		mu.Unlock()
		if _, err := Ensure(db, projectID); err != nil {
			log.Printf("standard: proyek #%d: %v", projectID, err)
		}
	})
}
