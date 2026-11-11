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
	// verdictWait: batas tunggu hasil untuk konteks selain todo (review audit tanpa vonis, perencanaan).
	verdictWait = 7 * 24 * time.Hour
	// reviewGrace: jeda setelah todo selesai & bernilai sebelum pemakaian yang tersisa dianggap tidak akan dinilai.
	reviewGrace = 24 * time.Hour
	settleBatch = 500
)

// SettleStale menutup pemakaian yang tidak akan pernah mendapat hasil sebagai netral, sehingga hitungan
// "menunggu nilai" hanya berisi pekerjaan yang memang masih berjalan. ✓/✗ node tidak berubah: tanpa hasil kerja
// yang dinilai tidak ada bukti untuk node itu. Yang ditutup:
//   - konteks selain todo yang lebih tua dari verdictWait (berkas audit tanpa vonis, perencanaan);
//   - todo yang sudah dihapus;
//   - todo selesai yang sudah bernilai lebih dari reviewGrace: penilaiannya sudah lewat, jadi pemakaian yang
//     tersisa berasal dari pengerjaan ulang yang tidak dinilai lagi.
//
// Todo yang masih dikerjakan, atau selesai tetapi belum dinilai, tidak disentuh. Mengembalikan jumlah pemakaian
// yang ditutup.
func SettleStale(db *gorm.DB, now time.Time) int {
	if db == nil {
		return 0
	}
	settled := 0
	res := db.Model(&models.KnowledgeUse{}).
		Where("outcome = ? AND context NOT LIKE ? AND created_at < ?", "", "todo:%", now.Add(-verdictWait)).
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
	cutoff := now.Add(-reviewGrace)
	for _, t := range todos {
		reviewed := t.Status == models.TodoStatusCompleted && t.Score != nil && t.CompletedAt != nil && t.CompletedAt.Before(cutoff)
		if !reviewed {
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
