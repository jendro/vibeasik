package models

import (
	"time"
)

// RouteInfo represents an individual API route or endpoint in a component (scanned or inferred)
type RouteInfo struct {
	Method         string   `json:"method"`                    // GET, POST, PUT, DELETE, PATCH, OPTIONS, PAGE (GET), etc.
	Path           string   `json:"path"`                      // e.g. /api/auth/register, /projects/:id
	Handler        string   `json:"handler"`                   // e.g. controllers.Register, controllers.GetAsset
	Description    string   `json:"description"`               // Penjelasan fungsi & tujuan router
	Middleware     string   `json:"middleware"`                // Public, Protected (JWT), dsb.
	TargetRequests []string `json:"target_requests,omitempty"` // Daftar endpoint API yang di-request oleh halaman ini (untuk FE)
}

// ComponentRoute represents an API route or endpoint persisted in database
type ComponentRoute struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ComponentID    uint      `gorm:"index;not null" json:"component_id"`
	Method         string    `gorm:"size:32;not null" json:"method"`                   // GET, POST, PUT, DELETE, PATCH, OPTIONS, PAGE (GET), etc.
	Path           string    `gorm:"size:512;not null" json:"path"`                    // e.g. /api/auth/register, /projects/:id
	Handler        string    `gorm:"size:255;default:''" json:"handler"`               // controller func or file path
	Middleware     string    `gorm:"size:255;default:''" json:"middleware"`            // Public, Protected (JWT), dsb.
	Description    string    `gorm:"type:text" json:"description"`                     // Keterangan / penjelasan fungsi & tanggung jawab endpoint
	Domain         string    `gorm:"size:128;default:'Umum'" json:"domain"`            // Kelompok domain bisnis (e.g. Autentikasi, Manajemen Proyek, Aset & Inventori)
	FunctionGroup  string    `gorm:"size:128;default:'Lainnya'" json:"function_group"` // Kelompok fungsi teknis (e.g. Otentikasi, Read Data, Create/Update Data)
	TargetRequests string    `gorm:"type:longtext" json:"target_requests"`             // JSON string array endpoint API yang di-request oleh halaman ini
	Tags           string    `gorm:"size:255;default:''" json:"tags"`                  // Opsional kata kunci dipisahkan koma
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RouteSyncResult represents the result of route synchronization between source code and database
type RouteSyncResult struct {
	ComponentID  uint             `json:"component_id"`
	AddedCount   int              `json:"added_count"`
	UpdatedCount int              `json:"updated_count"`
	DeletedCount int              `json:"deleted_count"`
	TotalCount   int              `json:"total_count"`
	Routes       []ComponentRoute `json:"routes"`
}
