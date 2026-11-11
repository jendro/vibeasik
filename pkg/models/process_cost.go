package models

import "time"

// ProcessCostSnapshot menyimpan sebaran kelas pertumbuhan (Big-O) rute satu proyek per hari, agar perbaikan
// algoritma terlihat sebagai tren: berapa rute yang masih O(n²) atau N+1 query dibanding hari-hari sebelumnya.
type ProcessCostSnapshot struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ProjectID    uint      `gorm:"uniqueIndex:idx_cost_snap_project_date;not null" json:"project_id"`
	SnapDate     string    `gorm:"size:10;uniqueIndex:idx_cost_snap_project_date;not null" json:"snap_date"` // YYYY-MM-DD
	Routes       int       `json:"routes"`
	Funcs        int       `json:"funcs"`
	Constant     int       `json:"constant"`     // rute O(1)
	Linear       int       `json:"linear"`       // rute O(n)
	Linearithmic int       `json:"linearithmic"` // rute O(n log n)
	Quadratic    int       `json:"quadratic"`    // rute O(n²)
	Worse        int       `json:"worse"`        // rute O(n³) ke atas
	NPlusOne     int       `json:"n_plus_one"`   // rute dengan query di dalam loop
	Ticks1       int64     `json:"ticks_1"`      // jumlah T(1) semua rute
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
