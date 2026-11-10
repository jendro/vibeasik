package models

import "time"

// Status grup error.
const (
	TrafficErrorOpen     = "open"
	TrafficErrorResolved = "resolved"
	TrafficErrorIgnored  = "ignored"
)

// TrafficErrorGroup = error aplikasi live yang dikelompokkan lewat sidik jari (tipe + frame teratas ternormalisasi).
// Stack hanya berisi frame (fungsi + berkas), tidak pernah nilai variabel, body, header, maupun query.
type TrafficErrorGroup struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	SourceID     uint      `gorm:"uniqueIndex:idx_traffic_err_key,priority:1;index;not null" json:"-"`
	Fingerprint  string    `gorm:"size:40;uniqueIndex:idx_traffic_err_key,priority:2" json:"fingerprint"`
	ErrType      string    `gorm:"size:128" json:"err_type"`
	Message      string    `gorm:"size:500" json:"message"` // pesan contoh (kejadian pertama)
	TopFrame     string    `gorm:"size:300" json:"top_frame"`
	Stack        string    `gorm:"size:8000" json:"stack,omitempty"` // stack contoh; tidak ikut di daftar
	Route        string    `gorm:"size:255" json:"route"`
	Method       string    `gorm:"size:10" json:"method"`
	Side         string    `gorm:"size:8" json:"side"` // server | browser
	Count        int64     `json:"count"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `gorm:"index" json:"last_seen"`
	FirstRelease string    `gorm:"size:64" json:"first_release"`
	LastRelease  string    `gorm:"size:64" json:"last_release"`
	Status       string    `gorm:"size:12;default:'open'" json:"status"`
}

// TrafficProcess = status proses per instance dari heartbeat SDK.
type TrafficProcess struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	SourceID  uint      `gorm:"uniqueIndex:idx_traffic_proc_key,priority:1;not null" json:"-"`
	Instance  string    `gorm:"size:128;uniqueIndex:idx_traffic_proc_key,priority:2" json:"instance"`
	Release   string    `gorm:"size:64" json:"release"`
	MemBytes  int64     `json:"mem_bytes"`
	UptimeS   int64     `json:"uptime_s"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}
