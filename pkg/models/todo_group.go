package models

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Todo gabungan (Aturan 43): todo serupa yang konteksnya dekat dilebur menjadi satu todo induk (Source
// "todo_group") dengan sub todo (ParentID = induk, status "merged"). Induk adalah satu-satunya unit kerja —
// satu run/sub agent, satu branch, satu commit squash, satu review QA — sehingga alur todo lain (orkestrator,
// divisi, pegawai, PM, MCP) cukup melihat induknya. Status sub todo mengikuti induk lewat SettleSubTodos.
const (
	TodoStatusMerged = "merged" // sub todo yang dilebur ke todo gabungan (ParentID); dikerjakan lewat induknya
	SourceTodoGroup  = "todo_group"
)

// settleSkipKey menandai update dari SettleSubTodos sendiri agar callback tidak memanggilnya berulang.
const settleSkipKey = "todo_group:settling"

// RegisterSubTodoSettle memasang callback GORM: setiap todo diubah/dihapus (lewat jalur mana pun — REST,
// MCP, orkestrator, review QA, rapat PM) sub todo diselaraskan dengan induknya.
func RegisterSubTodoSettle(db *gorm.DB) {
	settle := func(tx *gorm.DB) {
		if tx.Error != nil || tx.Statement.Schema == nil || tx.Statement.Schema.Table != "todos" {
			return
		}
		if _, skip := tx.Statement.Get(settleSkipKey); skip {
			return
		}
		fresh := tx.Session(&gorm.Session{NewDB: true})
		SettleSubTodos(fresh)
		SettleSplitParents(fresh)
	}
	db.Callback().Update().After("gorm:update").Register("todo_group:settle_update", settle)
	db.Callback().Delete().After("gorm:delete").Register("todo_group:settle_delete", settle)
}

// SettleSubTodos menyelaraskan sub todo yang masih "merged" dengan induknya: induk selesai → sub todo selesai
// (waktu & agent penyelesaian sama, catatan penyelesaian merujuk induk); induk ditolak → sub todo ditolak (bahan
// penjaga todo sejenis); induk terhapus → sub todo kembali pending tanpa induk. Idempoten dan murah (satu query
// bila tidak ada sub todo). Mengembalikan jumlah sub todo yang berubah.
func SettleSubTodos(db *gorm.DB) int {
	var subs []Todo
	db.Select("id", "parent_id", "rationale").Where("status = ?", TodoStatusMerged).Find(&subs)
	if len(subs) == 0 {
		return 0
	}
	parentIDs := make([]uint, 0, len(subs))
	for _, s := range subs {
		if s.ParentID != nil {
			parentIDs = append(parentIDs, *s.ParentID)
		}
	}
	var parents []Todo
	db.Select("id", "status", "completed_at", "completed_by_agent", "completed_by_model", "rationale").Where("id IN ?", parentIDs).Find(&parents)
	byID := make(map[uint]Todo, len(parents))
	for _, p := range parents {
		byID[p.ID] = p
	}
	tx := db.Set(settleSkipKey, true).Session(&gorm.Session{}) // aman dipakai ulang di loop
	n := 0
	for _, s := range subs {
		var p Todo
		ok := false
		if s.ParentID != nil {
			p, ok = byID[*s.ParentID]
		}
		var updates map[string]interface{}
		switch {
		case !ok:
			updates = map[string]interface{}{"status": TodoStatusPending, "parent_id": nil}
		case p.Status == TodoStatusCompleted:
			done := p.CompletedAt
			if done == nil {
				now := time.Now()
				done = &now
			}
			updates = map[string]interface{}{
				"status": TodoStatusCompleted, "completed_at": done, "completed_by_agent": p.CompletedByAgent, "completed_by_model": p.CompletedByModel,
				"rationale": appendNote(s.Rationale, fmt.Sprintf("todo gabungan #%d", p.ID), *done, subTodoNote(p, "dikerjakan sekaligus")),
			}
		case p.Status == TodoStatusRejected:
			updates = map[string]interface{}{"status": TodoStatusRejected,
				"rationale": strings.TrimSpace(s.Rationale + fmt.Sprintf("\n\nDitolak bersama todo gabungan #%d.", p.ID))}
		default:
			continue
		}
		if tx.Model(&Todo{}).Where("id = ? AND status = ?", s.ID, TodoStatusMerged).Updates(updates).RowsAffected > 0 {
			n++
		}
	}
	return n
}

// subTodoNote: catatan penyelesaian sub todo — merujuk induk dan membawa laporan terakhir induk (diringkas)
// agar memori & RAG todo sub todo tetap berisi apa yang dikerjakan.
func subTodoNote(p Todo, verb string) string {
	note := fmt.Sprintf("Sub todo ini %s lewat todo gabungan #%d.", verb, p.ID)
	if last := p.LatestCompletionNote(); last != "" {
		r := []rune(strings.Join(strings.Fields(last), " "))
		if len(r) > 600 {
			r = append(r[:600], '…')
		}
		note += " Laporan induk: " + string(r)
	}
	return note
}

func appendNote(rationale, who string, when time.Time, text string) string {
	entry := fmt.Sprintf("%s%s, %s): %s", completionNoteMark, who, when.Format("2 Jan 2006 15:04"), text)
	if strings.TrimSpace(rationale) == "" {
		return entry
	}
	return strings.TrimRight(rationale, "\n") + "\n\n" + entry
}

// ReleaseSubTodos mengembalikan sub todo yang belum selesai menjadi todo biasa (sebelum todo gabungannya
// dihapus/dipecah): status ikut induk bila masih saran/pending, selain itu pending. Mengembalikan id-nya.
func ReleaseSubTodos(db *gorm.DB, parent Todo) []uint {
	status := TodoStatusPending
	if parent.Status == TodoStatusSuggested {
		status = TodoStatusSuggested
	}
	var ids []uint
	db.Model(&Todo{}).Where("parent_id = ? AND status = ?", parent.ID, TodoStatusMerged).Pluck("id", &ids)
	if len(ids) > 0 {
		db.Set(settleSkipKey, true).Model(&Todo{}).Where("id IN ?", ids).Updates(map[string]interface{}{"status": status, "parent_id": nil})
	}
	return ids
}

// DeleteTodo menghapus todo; bila todo gabungan, sub todo yang belum selesai dikembalikan dulu menjadi todo biasa
// (status sesuai induknya — saran tetap saran) agar tidak ada pekerjaan yang hilang.
func DeleteTodo(db *gorm.DB, id uint) (int64, error) {
	var t Todo
	if db.Select("id", "status").First(&t, id).Error == nil {
		ReleaseSubTodos(db, t)
	}
	r := db.Delete(&Todo{}, id)
	return r.RowsAffected, r.Error
}
