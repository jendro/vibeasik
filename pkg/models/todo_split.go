package models

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Todo dipecah per aplikasi: todo yang menyentuh beberapa aplikasi berbeda di satu proyek (mis. backend, frontend,
// mobile) dipecah menjadi sub todo per aplikasi (SplitFromID = induk), masing-masing dikerjakan pegawai yang
// keahliannya cocok — seperti tiket per engineer. Induknya berstatus "split" (bukan unit kerja: tidak di-dispatch,
// tidak dinilai QA) dan selesai sendiri begitu semua sub todo-nya tuntas (SettleSplitParents). Kebalikan todo
// gabungan (todo_group.go): di sana banyak todo → satu unit kerja; di sini satu todo → banyak unit kerja.
const (
	TodoStatusSplit = "split" // induk yang dipecah per aplikasi; dikerjakan lewat sub todo (SplitFromID)
	SourceTodoSplit = "todo_split"
)

// SettleSplitParents menyelaraskan induk "split" dengan sub todo-nya: semua sub selesai/ditolak (minimal satu
// selesai) → induk selesai dengan nilai rata-rata & catatan per aplikasi; semua sub ditolak → induk ditolak; semua
// sub terhapus → induk kembali pending. Idempoten; satu query bila tidak ada induk split. Mengembalikan jumlah induk
// yang berubah.
func SettleSplitParents(db *gorm.DB) int {
	var parents []Todo
	db.Select("id", "rationale").Where("status = ?", TodoStatusSplit).Find(&parents)
	if len(parents) == 0 {
		return 0
	}
	tx := db.Set(settleSkipKey, true).Session(&gorm.Session{})
	n := 0
	for _, p := range parents {
		var subs []Todo
		db.Select("id", "title", "status", "score", "employee_name", "completed_at", "completed_by_agent", "completed_by_model").
			Where("split_from_id = ?", p.ID).Order("id").Find(&subs)
		updates := settleSplit(p, subs)
		if updates == nil {
			continue
		}
		if tx.Model(&Todo{}).Where("id = ? AND status = ?", p.ID, TodoStatusSplit).Updates(updates).RowsAffected > 0 {
			n++
		}
	}
	return n
}

func settleSplit(p Todo, subs []Todo) map[string]interface{} {
	if len(subs) == 0 {
		return map[string]interface{}{"status": TodoStatusPending}
	}
	done, rejected := 0, 0
	var last *time.Time
	scoreSum, scored := 0, 0
	var agents, modelNames, lines []string
	for _, s := range subs {
		switch s.Status {
		case TodoStatusCompleted:
			done++
			if s.CompletedAt != nil && (last == nil || s.CompletedAt.After(*last)) {
				last = s.CompletedAt
			}
			if s.Score != nil {
				scoreSum += *s.Score
				scored++
			}
			agents = appendUniq(agents, s.CompletedByAgent)
			modelNames = appendUniq(modelNames, s.CompletedByModel)
			line := fmt.Sprintf("#%d %s", s.ID, s.Title)
			if s.EmployeeName != "" {
				line += " — " + s.EmployeeName
			}
			if s.Score != nil {
				line += fmt.Sprintf(" (nilai QA %d)", *s.Score)
			}
			lines = append(lines, line)
		case TodoStatusRejected:
			rejected++
		default:
			return nil // masih ada sub todo yang belum tuntas
		}
	}
	if done == 0 {
		return map[string]interface{}{"status": TodoStatusRejected,
			"rationale": strings.TrimSpace(p.Rationale + "\n\nSemua sub todo per aplikasinya ditolak.")}
	}
	if last == nil {
		now := time.Now()
		last = &now
	}
	updates := map[string]interface{}{
		"status": TodoStatusCompleted, "completed_at": last,
		"completed_by_agent": strings.Join(agents, ", "), "completed_by_model": strings.Join(modelNames, ", "),
		"rationale": appendNote(p.Rationale, "pecahan per aplikasi", *last,
			fmt.Sprintf("Dikerjakan lewat %d sub todo per aplikasi: %s.", done, strings.Join(lines, "; "))),
	}
	if scored > 0 {
		updates["score"] = (scoreSum + scored/2) / scored
	}
	return updates
}

func appendUniq(xs []string, x string) []string {
	x = strings.TrimSpace(x)
	if x == "" {
		return xs
	}
	for _, y := range xs {
		if y == x {
			return xs
		}
	}
	return append(xs, x)
}
