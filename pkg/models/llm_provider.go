package models

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/util"

	"gorm.io/gorm"
)

// Kesehatan provider LLM: satu "provider" = alamat endpoint + API key (dua akun di URL yang sama
// dihitung terpisah karena kuotanya terpisah). Provider yang kuotanya habis ditandai Blocked dan tidak
// dipakai lagi oleh fitur apa pun (analisis, rapat PM, agenda pegawai) sampai diaktifkan lagi di
// Pengaturan LLM — atau otomatis setelah VIBEASIK_LLM_QUOTA_BLOCK_HOURS jam bila env itu diisi.
// Rate limit / key ditolak hanya memberi jeda singkat (CooldownUntil).

// LLMProviderState menyimpan status satu provider LLM (tanpa API key; hanya sidik key).
type LLMProviderState struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	ProviderKey string `gorm:"size:300;uniqueIndex" json:"provider_key"` // URL ternormalisasi + "#" + sidik key
	URL         string `gorm:"size:512" json:"url"`
	ConfigID    uint   `json:"config_id"`
	ConfigName  string `gorm:"size:255" json:"config_name"`
	// Blocked: kuota habis. BlockedUntil nil = sampai diaktifkan lagi oleh pengguna.
	Blocked       bool       `gorm:"index" json:"blocked"`
	BlockedUntil  *time.Time `json:"blocked_until,omitempty"`
	BlockedAt     *time.Time `json:"blocked_at,omitempty"`
	BlockReason   string     `gorm:"type:text" json:"block_reason,omitempty"`
	CooldownUntil *time.Time `json:"cooldown_until,omitempty"`
	LastError     string     `gorm:"type:text" json:"last_error,omitempty"`
	Failures      int        `json:"failures"`
	LastOKAt      *time.Time `json:"last_ok_at,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// LLMProviderKey membentuk kunci provider dari URL dan API key (key hanya disimpan sebagai sidik 8 hex).
func LLMProviderKey(url, apiKey string) string {
	u := strings.ToLower(strings.TrimRight(strings.TrimSpace(url), "/"))
	u = strings.TrimSuffix(u, "/chat/completions")
	if apiKey = strings.TrimSpace(apiKey); apiKey == "" {
		return u
	}
	sum := sha256.Sum256([]byte(apiKey))
	return u + "#" + hex.EncodeToString(sum[:4])
}

// quotaBlockDuration: 0 = blokir sampai diaktifkan lagi manual (bawaan).
func quotaBlockDuration() time.Duration {
	h, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("VIBEASIK_LLM_QUOTA_BLOCK_HOURS")))
	if h <= 0 {
		return 0
	}
	return time.Duration(h) * time.Hour
}

// LLMProviderUsable melaporkan apakah provider boleh dipakai sekarang; bila tidak, alasannya.
func LLMProviderUsable(db *gorm.DB, url, apiKey string) (bool, string) {
	if db == nil {
		return true, ""
	}
	var st LLMProviderState
	if db.Where("provider_key = ?", LLMProviderKey(url, apiKey)).Limit(1).Find(&st).Error != nil || st.ID == 0 {
		return true, ""
	}
	return st.Usable(time.Now())
}

// Usable: kuota habis (blokir belum kedaluwarsa) atau masih jeda rate limit → tidak dipakai.
func (st LLMProviderState) Usable(now time.Time) (bool, string) {
	if st.Blocked && (st.BlockedUntil == nil || now.Before(*st.BlockedUntil)) {
		return false, "kuota habis: " + st.BlockReason
	}
	if st.CooldownUntil != nil && now.Before(*st.CooldownUntil) {
		return false, "jeda sampai " + st.CooldownUntil.Format("15:04") + ": " + st.LastError
	}
	return true, ""
}

// LLMProviderStates memetakan provider_key → status untuk banyak pengecekan sekaligus.
func LLMProviderStates(db *gorm.DB) map[string]LLMProviderState {
	out := map[string]LLMProviderState{}
	if db == nil {
		return out
	}
	var rows []LLMProviderState
	db.Find(&rows)
	for _, r := range rows {
		out[r.ProviderKey] = r
	}
	return out
}

func upsertProviderState(db *gorm.DB, url, apiKey string, apply func(*LLMProviderState)) {
	if db == nil || strings.TrimSpace(url) == "" {
		return
	}
	key := LLMProviderKey(url, apiKey)
	var st LLMProviderState
	db.Where("provider_key = ?", key).Limit(1).Find(&st)
	if st.ID == 0 {
		st = LLMProviderState{ProviderKey: key, URL: strings.TrimRight(url, "/")}
		var cfg LLMConfig
		// kondisi struct: GORM meng-quote kolom `key` (kata tercadang MySQL)
		if db.Where(&LLMConfig{URL: url, Key: apiKey}).Limit(1).Find(&cfg).Error == nil && cfg.ID != 0 {
			st.ConfigID, st.ConfigName = cfg.ID, cfg.Name
		}
	}
	apply(&st)
	db.Save(&st)
}

// MarkLLMProviderQuota menandai provider kehabisan kuota: tidak dipakai lagi sampai diaktifkan ulang.
func MarkLLMProviderQuota(db *gorm.DB, url, apiKey, reason string) {
	upsertProviderState(db, url, apiKey, func(st *LLMProviderState) {
		now := time.Now()
		st.Blocked, st.BlockedAt, st.BlockReason = true, &now, util.ClipText(reason, 400)
		st.BlockedUntil = nil
		if d := quotaBlockDuration(); d > 0 {
			until := now.Add(d)
			st.BlockedUntil = &until
		}
		st.LastError, st.Failures = util.ClipText(reason, 400), st.Failures+1
	})
}

// MarkLLMProviderCooldown memberi jeda singkat (rate limit, key ditolak) tanpa memblokir permanen.
func MarkLLMProviderCooldown(db *gorm.DB, url, apiKey string, d time.Duration, reason string) {
	upsertProviderState(db, url, apiKey, func(st *LLMProviderState) {
		until := time.Now().Add(d)
		st.CooldownUntil, st.LastError, st.Failures = &until, util.ClipText(reason, 400), st.Failures+1
	})
}

// MarkLLMProviderOK mencatat jawaban sukses; hanya menulis bila provider pernah bermasalah.
func MarkLLMProviderOK(db *gorm.DB, url, apiKey string) {
	if db == nil {
		return
	}
	now := time.Now()
	db.Model(&LLMProviderState{}).Where("provider_key = ? AND (failures > 0 OR cooldown_until IS NOT NULL)", LLMProviderKey(url, apiKey)).
		Updates(map[string]interface{}{"failures": 0, "cooldown_until": nil, "last_ok_at": &now})
}

// ResetLLMProvider mengaktifkan lagi provider (kuota sudah diisi / dipulihkan pengguna).
func ResetLLMProvider(db *gorm.DB, id uint) error {
	return db.Model(&LLMProviderState{}).Where("id = ?", id).Updates(map[string]interface{}{
		"blocked": false, "blocked_until": nil, "blocked_at": nil, "block_reason": "", "cooldown_until": nil, "failures": 0,
	}).Error
}

var healthDB *gorm.DB

// SetLLMHealthDB dipasang database.InitDB agar setiap request LLM (paket ai) bisa mencatat kesehatan provider.
func SetLLMHealthDB(db *gorm.DB) { healthDB = db }

// LLMHealthDB mengembalikan database pencatat kesehatan provider (nil di unit test yang tidak memasangnya).
func LLMHealthDB() *gorm.DB { return healthDB }
