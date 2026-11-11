package knowledge

import (
	"log"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

const (
	// verdictWait: batas tunggu hasil untuk konteks apa pun (berkas audit tanpa vonis, perencanaan, todo yang
	// prompt-nya sudah dibangun tetapi tidak kunjung selesai). Pekerjaan yang dilanjutkan mencatat pemakaian baru.
	verdictWait = 7 * 24 * time.Hour
	// reviewGrace: jeda setelah todo selesai & bernilai sebelum pemakaian yang tersisa dianggap tidak akan dinilai.
	reviewGrace = 24 * time.Hour
	// reviewWait: todo selesai yang selama ini belum dinilai tidak akan dinilai lagi (review datang dalam sehari).
	reviewWait  = 48 * time.Hour
	settleBatch = 500
)

// SettleStale menutup pemakaian yang tidak akan pernah mendapat hasil sebagai netral, sehingga hitungan
// "menunggu nilai" hanya berisi pekerjaan yang memang masih berjalan. ✓/✗ node tidak berubah: tanpa hasil kerja
// yang dinilai tidak ada bukti untuk node itu. Yang ditutup:
//   - pemakaian yang lebih tua dari verdictWait, apa pun konteksnya;
//   - todo yang sudah dihapus;
//   - todo selesai yang sudah bernilai lebih dari reviewGrace: penilaiannya sudah lewat, jadi pemakaian yang
//     tersisa berasal dari pengerjaan ulang yang tidak dinilai lagi;
//   - todo selesai tanpa nilai lebih dari reviewWait: penilaiannya tidak pernah datang.
//
// Todo yang masih dikerjakan atau baru saja selesai tidak disentuh. Mengembalikan jumlah pemakaian yang ditutup.
func SettleStale(db *gorm.DB, now time.Time) int {
	if db == nil {
		return 0
	}
	settled := 0
	res := db.Model(&models.KnowledgeUse{}).
		Where("outcome = ? AND created_at < ?", "", now.Add(-verdictWait)).
		Update("outcome", models.KnowledgeOutcomeNeutral)
	if res.Error != nil {
		log.Printf("[Knowledge] menutup pemakaian tanpa hasil gagal: %v", res.Error)
	} else {
		settled += int(res.RowsAffected)
	}

	var contexts []string
	if err := db.Model(&models.KnowledgeUse{}).Where("outcome = ? AND context LIKE ?", "", "todo:%").
		Distinct().Pluck("context", &contexts).Error; err != nil {
		log.Printf("[Knowledge] membaca pemakaian todo yang menunggu gagal: %v", err)
		return settled
	}
	byTodo := map[uint]string{}
	ids := make([]uint, 0, len(contexts))
	for _, c := range contexts {
		id, err := strconv.ParseUint(strings.TrimPrefix(c, "todo:"), 10, 64)
		if err != nil {
			continue // kunci tak dikenal: dibiarkan sampai masa simpan habis
		}
		byTodo[uint(id)] = c
		ids = append(ids, uint(id))
	}
	if len(ids) == 0 {
		return settled
	}
	// Todo yang gagal dibaca tidak boleh dianggap terhapus: pemakaiannya dibiarkan menunggu.
	var todos []models.Todo
	if err := db.Select("id", "status", "score", "completed_at").Where("id IN ?", ids).Find(&todos).Error; err != nil {
		log.Printf("[Knowledge] membaca todo pemakaian yang menunggu gagal: %v", err)
		return settled
	}
	for _, t := range todos {
		if !todoSettled(t, now) {
			delete(byTodo, t.ID)
		}
	}
	stale := make([]string, 0, len(byTodo))
	for _, c := range byTodo {
		stale = append(stale, c)
	}
	for i := 0; i < len(stale); i += settleBatch {
		res := db.Model(&models.KnowledgeUse{}).
			Where("outcome = ? AND context IN ?", "", stale[i:min(i+settleBatch, len(stale))]).
			Update("outcome", models.KnowledgeOutcomeNeutral)
		if res.Error != nil {
			log.Printf("[Knowledge] menutup pemakaian todo tanpa hasil gagal: %v", res.Error)
			continue
		}
		settled += int(res.RowsAffected)
	}
	return settled
}

// todoSettled: todo selesai yang masa tunggu penilaiannya sudah habis.
func todoSettled(t models.Todo, now time.Time) bool {
	if t.Status != models.TodoStatusCompleted || t.CompletedAt == nil {
		return false
	}
	wait := reviewWait
	if t.Score != nil {
		wait = reviewGrace
	}
	return t.CompletedAt.Before(now.Add(-wait))
}
