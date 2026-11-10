package models

import "time"

// Traffic Analyzer (Aturan 48): statistik request aplikasi live per sumber.
// Alur: SDK di BE live → vibeasik live (cloud-server, antrean sementara) → engine ini menarik berkala →
// dirangkum per menit/jam di sini. Mode "direct": BE (dev/staging) mengirim langsung ke engine ini.
const (
	TrafficModeCloud  = "cloud"
	TrafficModeDirect = "direct"

	TrafficGranMinute = "m"
	TrafficGranHour   = "h"
)

type TrafficSource struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	ProjectID       *uint      `gorm:"index" json:"project_id,omitempty"`
	ComponentID     *uint      `gorm:"index" json:"component_id,omitempty"`
	Name            string     `gorm:"size:255;not null" json:"name"`
	Mode            string     `gorm:"size:16;default:'cloud'" json:"mode"`
	CloudURL        string     `gorm:"size:512" json:"cloud_url"` // URL publik vibeasik live (cloud-server)
	CloudUID        string     `gorm:"size:64" json:"cloud_uid,omitempty"`
	IngestKey       string     `gorm:"size:128;index" json:"-"` // kunci SDK; hanya tampil di endpoint SDK
	PullKey         string     `gorm:"size:128" json:"-"`       // kunci tarik dari vibeasik live; tidak pernah keluar dari engine
	SyncIntervalSec int        `gorm:"default:60" json:"sync_interval_sec"`
	Enabled         bool       `gorm:"default:true" json:"enabled"`
	Cursor          uint       `gorm:"default:0" json:"-"`
	Pending         int64      `json:"pending"`        // event yang masih menunggu di vibeasik live
	Dropped         int64      `json:"dropped"`        // event dibuang vibeasik live karena antrean penuh
	TotalEvents     int64      `json:"total_events"`   // event yang sudah diolah engine ini
	LastSyncAt      *time.Time `json:"last_sync_at"`   // tarikan terakhir (berhasil maupun gagal)
	LastEventAt     *time.Time `json:"last_event_at"`  // event terbaru yang diterima
	LastIngestAt    *time.Time `json:"last_ingest_at"` // aplikasi live terakhir mengirim ke vibeasik live
	LastInstance    string     `gorm:"size:128" json:"last_instance"`
	LastError       string     `gorm:"size:1000" json:"last_error"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TrafficRouteStat = rangkuman per (menit|jam, method, rute). Hist = CSV jumlah request per ember latensi
// (batas di traffic.LatencyBounds) sehingga p50/p95/p99 bisa digabung lintas ember waktu.
type TrafficRouteStat struct {
	ID       uint    `gorm:"primaryKey" json:"-"`
	SourceID uint    `gorm:"uniqueIndex:idx_traffic_route_key,priority:1;not null" json:"-"`
	Gran     string  `gorm:"size:1;uniqueIndex:idx_traffic_route_key,priority:2" json:"-"`
	Bucket   int64   `gorm:"uniqueIndex:idx_traffic_route_key,priority:3;index" json:"bucket"`
	Method   string  `gorm:"size:10;uniqueIndex:idx_traffic_route_key,priority:4" json:"method"`
	Route    string  `gorm:"size:255;uniqueIndex:idx_traffic_route_key,priority:5" json:"route"`
	Count    int64   `json:"count"`
	Err4xx   int64   `json:"err_4xx"`
	Err5xx   int64   `json:"err_5xx"`
	SumMS    float64 `json:"sum_ms"`
	MaxMS    float64 `json:"max_ms"`
	SumIn    int64   `json:"sum_in"`
	SumOut   int64   `json:"sum_out"`
	Hist     string  `gorm:"size:255" json:"-"`
}

// TrafficDimStat = hitungan per dimensi (ip, country, city, asn, org, net_type, ua, client, referer, status, instance, auth_ip, notfound, gpath).
type TrafficDimStat struct {
	ID       uint   `gorm:"primaryKey" json:"-"`
	SourceID uint   `gorm:"uniqueIndex:idx_traffic_dim_key,priority:1;not null" json:"-"`
	Gran     string `gorm:"size:1;uniqueIndex:idx_traffic_dim_key,priority:2" json:"-"`
	Bucket   int64  `gorm:"uniqueIndex:idx_traffic_dim_key,priority:3;index" json:"bucket"`
	Dim      string `gorm:"size:16;uniqueIndex:idx_traffic_dim_key,priority:4" json:"dim"`
	Value    string `gorm:"size:255;uniqueIndex:idx_traffic_dim_key,priority:5" json:"value"`
	Count    int64  `json:"count"`
	Err      int64  `json:"err"`
}

// TrafficRecentEvent = ekor event terbaru (maks traffic.RecentKeep per sumber) untuk tampilan live.
type TrafficRecentEvent struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	SourceID uint      `gorm:"index;not null" json:"-"`
	At       time.Time `json:"at"`
	Method   string    `gorm:"size:10" json:"method"`
	Route    string    `gorm:"size:255" json:"route"`
	Path     string    `gorm:"size:255" json:"path"`
	Status   int       `json:"status"`
	DurMS    float64   `json:"dur_ms"`
	BytesOut int64     `json:"bytes_out"`
	IP       string    `gorm:"size:64" json:"ip"`
	Country  string    `gorm:"size:8" json:"country"`
	City     string    `gorm:"size:64" json:"city,omitempty"`
	ASN      uint32    `json:"asn,omitempty"`
	Org      string    `gorm:"size:128" json:"org,omitempty"`
	NetType  string    `gorm:"size:16" json:"net_type,omitempty"` // mobile | residential | hosting | vpn | unknown
	UAClass  string    `gorm:"size:16" json:"ua_class"`
	Client   string    `gorm:"size:32" json:"client"`
	Referer  string    `gorm:"size:255" json:"referer"`
	Instance string    `gorm:"size:128" json:"instance"`
}
