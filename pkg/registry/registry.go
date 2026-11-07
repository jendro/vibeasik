// Package registry adalah titik daftar modul vibeasik.
//
// Build Community hanya mendaftarkan modul inti. Build Official (vibeasik-pro) mengimpor modul Pro
// yang mendaftarkan dirinya lewat Register di init(): model DB, route API, tool MCP, worker latar,
// dan hook yang memperkaya alur inti. Arah dependency selalu Pro → registry, tidak pernah sebaliknya.
package registry

import (
	"context"
	"sort"
	"sync"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/mcp/protocol"
)

// Module adalah satu kesatuan fitur. Kemampuan tambahan dipasang dengan mengimplementasikan
// interface opsional di bawah (ModelProvider, RouteProvider, ToolProvider, Starter).
type Module interface {
	// Name nama unik modul, mis. "staff".
	Name() string
	// Feature kunci entitlement yang dibutuhkan; "" = selalu aktif (Community).
	Feature() string
}

// ModelProvider mendaftarkan model GORM untuk AutoMigrate.
type ModelProvider interface {
	Models() []any
}

// RouteProvider memasang endpoint di bawah /api. Router otomatis membungkus tiap route dengan gate fitur modul.
type RouteProvider interface {
	Routes(r Router, db *gorm.DB)
}

// ToolProvider menyediakan tool MCP. CallTool mengembalikan handled=false bila name bukan miliknya.
type ToolProvider interface {
	Tools() []protocol.Tool
	CallTool(db *gorm.DB, name string, args map[string]any) (res *protocol.ToolCallResult, handled bool, err error)
}

// Starter menjalankan worker latar (antrean, agenda, …) saat engine start.
type Starter interface {
	Start(ctx context.Context, db *gorm.DB) error
}

var (
	mu      sync.RWMutex
	modules = map[string]Module{}
)

// Register mendaftarkan modul. Nama ganda menimpa yang lama (berguna untuk build Official yang
// mengganti implementasi Community).
func Register(m Module) {
	mu.Lock()
	defer mu.Unlock()
	modules[m.Name()] = m
}

// All mengembalikan semua modul terdaftar, urut nama agar perilaku start deterministik.
func All() []Module {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Module, 0, len(modules))
	for _, m := range modules {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// Models mengumpulkan model dari semua modul untuk AutoMigrate.
func Models() []any {
	var out []any
	for _, m := range All() {
		if mp, ok := m.(ModelProvider); ok {
			out = append(out, mp.Models()...)
		}
	}
	return out
}
