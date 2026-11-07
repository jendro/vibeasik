// Package activity mencatat siapa saja "pekerja" AI yang sedang aktif (agent coding via MCP/hook,
// CLI agent lokal, dan request LLM luar) untuk divisualisasikan di halaman Kantor Agent.
//
// Semua fungsi bersifat best-effort: kegagalan menulis aktivitas tidak boleh mengganggu alur utama,
// jadi error DB diabaikan. Bila Init belum dipanggil (mis. di unit test), seluruh fungsi menjadi no-op.
package activity

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Batas waktu tampil di kantor.
const (
	IdleAfter        = 3 * time.Minute  // agent tanpa aktivitas selama ini dianggap duduk santai
	CodingAgentTTL   = 30 * time.Minute // agent tanpa aktivitas selama ini dianggap sudah pulang
	FinishedLinger   = 20 * time.Second // pekerja selesai/keluar tetap tampil sebentar (animasi pulang)
	IdleHomeAfter    = FinishedLinger   // agent coding yang selesai menjawab (menunggu chat baru) lalu pulang
	ErrorLinger      = 10 * time.Minute // pekerja gagal tetap di mejanya agar error-nya sempat diperiksa
	StuckLLMTTL      = 10 * time.Minute // request LLM "working" lebih lama dari ini dianggap proses mati
	pruneInterval    = time.Minute
	llmRetention     = time.Hour
	workerRetention  = 7 * 24 * time.Hour
	maxTaskLength    = 500
	maxActionLength  = 180
	maxGenericLength = 120
	maxPathLength    = 1000
)

var (
	mu          sync.RWMutex
	db          *gorm.DB
	llmContext  func() (task, project string)
	seq         atomic.Uint64
	lastPruneAt atomic.Int64
	// livenessEnabled: hanya engine (StartLocalWatchers) yang mencocokkan pekerja dengan proses
	// asli di laptop, agar unit test tidak bergantung pada proses yang kebetulan berjalan.
	livenessEnabled atomic.Bool
)

// Caller mengidentifikasi satu koneksi MCP (stdio per proses, HTTP per sesi/IP).
type Caller struct {
	Key       string
	Transport string
}

// Init mengaktifkan pencatatan aktivitas ke database.
func Init(d *gorm.DB) {
	mu.Lock()
	db = d
	mu.Unlock()
}

// SetLLMContextProvider memberi label pekerjaan untuk request LLM (mis. judul job antrean yang sedang berjalan).
func SetLLMContextProvider(fn func() (task, project string)) {
	mu.Lock()
	llmContext = fn
	mu.Unlock()
}

func getDB() *gorm.DB {
	mu.RLock()
	defer mu.RUnlock()
	return db
}

func newKey(prefix string) string {
	return fmt.Sprintf("%s:%d:%d:%d", prefix, os.Getpid(), time.Now().UnixNano(), seq.Add(1))
}

// truncate memotong string secara aman untuk UTF-8.
func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:max-1])) + "…"
}

// firstLine mengambil baris pertama yang tidak kosong.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

// upsertWorker membuat baris pekerja bila belum ada, lalu menerapkan mutate dan menyimpannya.
func upsertWorker(d *gorm.DB, key string, defaults models.AgentWorker, mutate func(w *models.AgentWorker)) *models.AgentWorker {
	now := time.Now()
	var w models.AgentWorker
	if err := d.Where("worker_key = ?", key).First(&w).Error; err != nil {
		w = defaults
		w.Key = key
		w.StartedAt = now
		if w.Status == "" {
			w.Status = models.WorkerStatusWorking
		}
	}
	// Sesi yang sempat ditandai pulang lalu aktif lagi dianggap masuk kerja kembali.
	if w.Status == models.WorkerStatusLeft || w.FinishedAt != nil {
		w.FinishedAt = nil
		w.Status = models.WorkerStatusWorking
	}
	apply := func(w *models.AgentWorker) {
		w.LastSeenAt = now
		if mutate != nil {
			mutate(w)
		}
		w.Task = truncate(w.Task, maxTaskLength)
		w.Action = truncate(w.Action, maxActionLength)
		w.LastError = truncate(w.LastError, maxTaskLength)
	}
	apply(&w)
	if w.ID != 0 {
		d.Save(&w)
		return &w
	}
	if res := d.Clauses(clause.OnConflict{DoNothing: true}).Create(&w); res.Error == nil && res.RowsAffected > 0 {
		return &w
	}
	// Baris dengan key sama baru saja dibuat proses lain (stdio & engine berbagi DB): ulangi sebagai update.
	var existing models.AgentWorker
	if err := d.Where("worker_key = ?", key).First(&existing).Error; err != nil {
		return &w
	}
	apply(&existing)
	d.Save(&existing)
	return &existing
}

// finish menandai pekerja sekali-jalan (LLM / CLI lokal) sudah selesai.
func finish(d *gorm.DB, key string, failed bool, errMsg string) {
	now := time.Now()
	status := models.WorkerStatusDone
	if failed {
		status = models.WorkerStatusError
	}
	d.Model(&models.AgentWorker{}).Where("worker_key = ?", key).Updates(map[string]interface{}{
		"status":       status,
		"last_error":   truncate(errMsg, maxTaskLength),
		"finished_at":  now,
		"last_seen_at": now,
	})
}

// VisibleWorkers mengembalikan pekerja yang layak tampil di kantor saat ini.
func VisibleWorkers(now time.Time) ([]models.AgentWorker, error) {
	d := getDB()
	if d == nil {
		return []models.AgentWorker{}, nil
	}
	prune(d, now)

	var rows []models.AgentWorker
	if err := d.Where("last_seen_at >= ?", now.Add(-CodingAgentTTL)).Order("started_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	var procs *procSnapshot
	if livenessEnabled.Load() {
		procs = currentProcs(now)
	}
	// Koordinator tetap di kantor selama anak buahnya masih bekerja walau gilirannya sudah selesai.
	supervising := map[string]bool{}
	for _, w := range rows {
		if w.ParentKey != "" && w.Status == models.WorkerStatusWorking {
			supervising[w.ParentKey] = true
		}
	}
	visible := make([]models.AgentWorker, 0, len(rows))
	for _, w := range rows {
		if !isVisible(w, now) || (wentHome(w, now) && !supervising[w.Key]) {
			continue
		}
		if alive, known := processAlive(w, now, procs); known && !alive {
			markGone(d, w)
			continue
		}
		visible = append(visible, w)
	}
	return mergeDuplicateMCP(visible), nil
}

func isVisible(w models.AgentWorker, now time.Time) bool {
	switch w.Status {
	case models.WorkerStatusDone, models.WorkerStatusError, models.WorkerStatusLeft:
		end := w.LastSeenAt
		if w.FinishedAt != nil {
			end = *w.FinishedAt
		}
		if w.Status == models.WorkerStatusError {
			return now.Sub(end) <= ErrorLinger
		}
		return now.Sub(end) <= FinishedLinger
	}
	if w.Kind == models.WorkerKindLLMCall || w.Kind == models.WorkerKindLocalAgent || w.Kind == models.WorkerKindPlanner {
		return now.Sub(w.LastSeenAt) <= StuckLLMTTL
	}
	return now.Sub(w.LastSeenAt) <= CodingAgentTTL
}

// wentHome: agent coding yang sudah selesai menjawab (status idle, menunggu chat baru) kemungkinan
// tidak dipakai lagi, jadi pulang alih-alih santai/tidur di lounge. Chat baru di sesi yang sama
// mengubah statusnya kembali "working" sehingga ia datang lagi. Menunggu izin pengguna bukan selesai.
func wentHome(w models.AgentWorker, now time.Time) bool {
	return w.Kind == models.WorkerKindCodingAgent && w.Status == models.WorkerStatusIdle &&
		w.Action != actionNeedsPermission && now.Sub(w.LastSeenAt) > IdleHomeAfter
}

// markGone menutup pekerja yang prosesnya sudah tidak ada (engine/CLI mati sebelum sempat melapor
// selesai) agar tidak tampil lagi. finished_at = aktivitas terakhir sehingga tidak ada animasi pulang
// palsu; event baru dari sesi yang sama akan membukanya kembali lewat upsertWorker.
func markGone(d *gorm.DB, w models.AgentWorker) {
	if w.Status == models.WorkerStatusDone || w.Status == models.WorkerStatusError || w.Status == models.WorkerStatusLeft {
		return
	}
	updates := map[string]interface{}{"finished_at": w.LastSeenAt}
	if w.Kind == models.WorkerKindLLMCall || w.Kind == models.WorkerKindLocalAgent || w.Kind == models.WorkerKindPlanner {
		updates["status"] = models.WorkerStatusError
		updates["last_error"] = "Proses berhenti sebelum selesai"
	} else {
		updates["status"] = models.WorkerStatusLeft
	}
	d.Model(&models.AgentWorker{}).Where("id = ? AND status = ?", w.ID, w.Status).Updates(updates)
}

// prune menghapus baris lama paling banyak sekali per menit agar tabel tetap kecil.
func prune(d *gorm.DB, now time.Time) {
	last := lastPruneAt.Load()
	if now.UnixNano()-last < int64(pruneInterval) || !lastPruneAt.CompareAndSwap(last, now.UnixNano()) {
		return
	}
	d.Where("kind = ? AND last_seen_at < ?", models.WorkerKindLLMCall, now.Add(-llmRetention)).Delete(&models.AgentWorker{})
	d.Where("last_seen_at < ?", now.Add(-workerRetention)).Delete(&models.AgentWorker{})
}
