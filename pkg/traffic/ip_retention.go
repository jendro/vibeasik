package traffic

import (
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

const (
	// IPRetentionEnv = jumlah hari IP penuh disimpan sebelum disamarkan (/24 untuk IPv4, /48 untuk IPv6).
	IPRetentionEnv         = "VIBEASIK_TRAFFIC_IP_RETENTION_DAYS"
	DefaultIPRetentionDays = 7
	maskBatch              = 500
)

// IPRetention = masa simpan IP penuh; nilai env yang tidak valid atau < 1 memakai bawaan 7 hari.
func IPRetention() time.Duration {
	days := DefaultIPRetentionDays
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(IPRetentionEnv))); err == nil && v >= 1 {
		days = v
	}
	return time.Duration(days) * 24 * time.Hour
}

// MaskIP menyamarkan IP ke jaringannya: IPv4 → a.b.c.0/24, IPv6 → /48. Nilai yang bukan IP polos
// (sudah disamarkan atau kosong) dikembalikan apa adanya dengan ok=false.
func MaskIP(ip string) (string, bool) {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return ip, false
	}
	addr = addr.Unmap()
	bits := 48
	if addr.Is4() {
		bits = 24
	}
	p, err := addr.Prefix(bits)
	if err != nil {
		return ip, false
	}
	return p.String(), true
}

// maskOldIPs menyamarkan IP penuh yang lebih tua dari masa simpan pada dimensi ip/auth_ip dan ekor event.
// Hitungan agregat tetap: baris yang bertabrakan pada indeks unik digabung.
func maskOldIPs(db *gorm.DB, now time.Time) {
	cut := now.Add(-IPRetention())
	maskDimIPs(db, cut.Unix())
	maskRecentIPs(db, cut)
}

func maskDimIPs(db *gorm.DB, cutUnix int64) {
	var lastID uint
	for {
		var rows []models.TrafficDimStat
		if err := db.Where("id > ? AND dim IN ? AND bucket < ? AND value NOT LIKE ?", lastID, []string{"ip", "auth_ip"}, cutUnix, "%/%").
			Order("id").Limit(maskBatch).Find(&rows).Error; err != nil || len(rows) == 0 {
			return
		}
		lastID = rows[len(rows)-1].ID
		err := db.Transaction(func(tx *gorm.DB) error {
			for _, r := range rows {
				if err := maskDimRow(tx, r); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return // coba lagi pada putaran pangkas berikutnya
		}
	}
}

// maskDimRow mengganti nilai IP dengan jaringannya; bila baris tujuan sudah ada, hitungan digabung.
// Nilai yang bukan IP (mis. samaran dari SDK) dibiarkan.
func maskDimRow(tx *gorm.DB, r models.TrafficDimStat) error {
	masked, ok := MaskIP(r.Value)
	if !ok {
		return nil
	}
	var existing models.TrafficDimStat
	err := tx.Where("source_id = ? AND gran = ? AND bucket = ? AND dim = ? AND value = ?", r.SourceID, r.Gran, r.Bucket, r.Dim, masked).Take(&existing).Error
	if err == nil {
		if err := tx.Model(&existing).Updates(map[string]any{"count": existing.Count + r.Count, "err": existing.Err + r.Err}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.TrafficDimStat{}, r.ID).Error
	}
	return tx.Model(&models.TrafficDimStat{}).Where("id = ?", r.ID).Update("value", masked).Error
}

func maskRecentIPs(db *gorm.DB, cut time.Time) {
	var lastID uint
	for {
		var rows []models.TrafficRecentEvent
		if err := db.Select("id", "ip").Where("id > ? AND at < ? AND ip <> '' AND ip NOT LIKE ?", lastID, cut, "%/%").
			Order("id").Limit(maskBatch).Find(&rows).Error; err != nil || len(rows) == 0 {
			return
		}
		lastID = rows[len(rows)-1].ID
		for _, r := range rows {
			if masked, ok := MaskIP(r.IP); ok {
				db.Model(&models.TrafficRecentEvent{}).Where("id = ?", r.ID).Update("ip", masked)
			}
		}
	}
}
