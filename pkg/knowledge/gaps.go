package knowledge

import (
	"errors"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/util"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Celah pengetahuan = kurikulum otomatis (Voyager): ekosistem yang muncul di pekerjaan tanpa satu node pun
// (mis. todo menyentuh src/main.gleam), atau topik yang dilaporkan agent/manusia. Paket staff mengubah celah
// terbuka menjadi agenda belajar pegawai yang paling cocok; hasil agendanya masuk pustaka bersama lalu celah
// ditutup. Agenda yang gagal/tidak menghasilkan node dicoba ulang, maks maxGapAttempts kali.

const maxGapAttempts = 3

// autoTopic: topik celah yang dideteksi otomatis.
func autoTopic(domain string) string { return "Teknologi baru: " + domain }

// recordAutoGap mencatat (atau menambah hitungan) celah ekosistem tanpa node. Celah yang pernah diabaikan tidak dibuka lagi.
func recordAutoGap(db *gorm.DB, domain, sample string) {
	var g models.KnowledgeGap
	if db.Where("domain = ? AND auto = ? AND status IN ?", domain, true,
		[]string{models.KnowledgeGapOpen, models.KnowledgeGapLearning, models.KnowledgeGapDismissed}).Limit(1).Find(&g).Error == nil && g.ID != 0 {
		if g.Status != models.KnowledgeGapDismissed {
			db.Model(&g).UpdateColumn("hits", gorm.Expr("hits + 1"))
		}
		return
	}
	db.Create(&models.KnowledgeGap{Domain: domain, Topic: autoTopic(domain), Sample: clip(sample, 600), Hits: 1, Auto: true, Status: models.KnowledgeGapOpen})
}

// ReportGap mencatat celah yang dilaporkan agent/manusia (mis. metode baru di ekosistem yang sudah dikenal).
func ReportGap(db *gorm.DB, domain, topic, sample string) (*models.KnowledgeGap, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	topic = clip(topic, 180)
	if topic == "" {
		return nil, errors.New("topik celah wajib diisi")
	}
	if domain == "" {
		domain = primaryDomain(topic + "\n" + sample)
	}
	var g models.KnowledgeGap
	if db.Where("domain = ? AND LOWER(topic) = ? AND status IN ?", domain, strings.ToLower(topic),
		[]string{models.KnowledgeGapOpen, models.KnowledgeGapLearning}).Limit(1).Find(&g).Error == nil && g.ID != 0 {
		db.Model(&g).UpdateColumn("hits", gorm.Expr("hits + 1"))
		g.Hits++
		return &g, nil
	}
	g = models.KnowledgeGap{Domain: domain, Topic: topic, Sample: clip(sample, 600), Hits: 1, Status: models.KnowledgeGapOpen}
	return &g, db.Create(&g).Error
}

// fillGaps menutup celah otomatis sebuah ekosistem begitu node pertamanya masuk.
func fillGaps(db *gorm.DB, domain string) {
	if domain == "" {
		return
	}
	now := time.Now()
	db.Model(&models.KnowledgeGap{}).Where("domain = ? AND auto = ? AND status IN ?", domain, true,
		[]string{models.KnowledgeGapOpen, models.KnowledgeGapLearning}).
		Updates(map[string]interface{}{"status": models.KnowledgeGapLearned, "learned_at": &now})
}

// OpenGaps: celah terbuka, yang paling sering muncul lebih dulu.
func OpenGaps(db *gorm.DB, limit int) []models.KnowledgeGap {
	var out []models.KnowledgeGap
	db.Where("status = ?", models.KnowledgeGapOpen).Order("hits DESC, id ASC").Limit(limit).Find(&out)
	return out
}

// ListGaps: semua celah (status kosong = semua) terbaru di atas.
func ListGaps(db *gorm.DB, status string, limit int) []models.KnowledgeGap {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := db.Order("CASE status WHEN 'open' THEN 0 WHEN 'learning' THEN 1 ELSE 2 END, updated_at DESC").Limit(limit)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var out []models.KnowledgeGap
	q.Find(&out)
	return out
}

// MarkGapLearning menautkan celah ke agenda belajar pegawai.
func MarkGapLearning(db *gorm.DB, gapID, employeeID uint, employeeName string, agendaID uint) {
	db.Model(&models.KnowledgeGap{}).Where("id = ?", gapID).Updates(map[string]interface{}{
		"status": models.KnowledgeGapLearning, "employee_id": employeeID, "employee_name": employeeName, "agenda_id": agendaID,
	})
}

// ResolveGap menutup celah setelah agenda belajarnya menghasilkan n node.
func ResolveGap(db *gorm.DB, gapID uint, n int, note string) {
	if n <= 0 {
		return
	}
	now := time.Now()
	db.Model(&models.KnowledgeGap{}).Where("id = ?", gapID).Updates(map[string]interface{}{
		"status": models.KnowledgeGapLearned, "nodes_learned": gorm.Expr("nodes_learned + ?", n), "learned_at": &now, "note": clip(note, 1000),
	})
}

// GapByID mengembalikan satu celah.
func GapByID(db *gorm.DB, id uint) (*models.KnowledgeGap, error) {
	var g models.KnowledgeGap
	if err := db.First(&g, id).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

// DismissGap mengabaikan celah (mis. ekstensi yang ternyata bukan bahasa pemrograman).
func DismissGap(db *gorm.DB, id uint, note string) error {
	res := db.Model(&models.KnowledgeGap{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": models.KnowledgeGapDismissed, "note": clip(util.FirstNonBlank(note, "Diabaikan manual"), 500)})
	if res.RowsAffected == 0 {
		return errors.New("celah tidak ditemukan")
	}
	return res.Error
}

// CheckGapAgendas membuka lagi celah yang agendanya gagal, dibatalkan, atau selesai tanpa node baru.
func CheckGapAgendas(db *gorm.DB) {
	var gaps []models.KnowledgeGap
	db.Where("status = ? AND agenda_id IS NOT NULL", models.KnowledgeGapLearning).Find(&gaps)
	for _, g := range gaps {
		var a models.EmployeeAgenda
		missing := db.Select("id", "status").First(&a, *g.AgendaID).Error != nil
		if !missing && a.Status != models.AgendaFailed && a.Status != models.AgendaDropped && a.Status != models.AgendaDone {
			continue
		}
		attempts := g.Attempts + 1
		updates := map[string]interface{}{"attempts": attempts, "agenda_id": nil, "status": models.KnowledgeGapOpen}
		if attempts >= maxGapAttempts {
			updates["status"], updates["note"] = models.KnowledgeGapDismissed, "Belum berhasil dipelajari setelah beberapa agenda — tambahkan pengetahuan manual atau laporkan ulang."
		}
		db.Model(&g).Updates(updates)
	}
}
