// Package routemap menelusuri rute HTTP proyek Go secara statis dengan parser resmi Go (go/parser + go/ast,
// dasar yang sama dengan gopls/staticcheck; tanpa CGO, tanpa perlu mengunduh dependensi proyek):
//
//	main() → fungsi setup (routes.SetupRoutes(app)) → Group/Use/Route/With → rute → handler → response.
//
// Untuk tiap rute dihitung rantai middleware efektif (global, group, prefix, rute) sesuai semantik framework
// (Fiber/Gin/Echo/Chi/net/http), middleware yang terdaftar tetapi tidak aktif (didaftarkan setelah rute atau
// dilewati lewat Next/Skipper/skip prefix), status response handler (termasuk lewat helper), dan fungsi lokal
// yang dipanggil. Call graph lokal memungkinkan penelusuran balik: fungsi → handler → rute → group → main.
// Hasilnya ditulis ke peta kode (.vibeasik/CODEMAP.md & routes.jsonl) agar AI agent tidak menyimpulkan dari
// satu baris saja (mis. mengira rute tanpa auth padahal auth dipasang di group atau global).
package routemap

import "go/ast"

// Graph = hasil analisis satu modul Go (satu go.mod).
type Graph struct {
	Module    string    `json:"module"`
	Dir       string    `json:"dir"`       // direktori go.mod relatif terhadap root proyek ("." = root)
	Framework string    `json:"framework"` // fiber, gin, echo, chi, gorilla, nethttp (campuran dipisah koma)
	Entries   []string  `json:"entries"`   // posisi func main yang ditelusuri
	Routers   []*Router `json:"routers"`
	Routes    []*Route  `json:"routes"`
	Warnings  []string  `json:"warnings,omitempty"`
	// Duplicates: pemeriksaan yang dikerjakan dua kali di jalur rute yang sama (middleware/validasi ganda)
	Duplicates []Duplicate `json:"duplicates,omitempty"`

	funcs   map[string]*funcInfo
	callees map[string][]string
	callers map[string][]string
}

// Router = app/engine/group/sub-router beserta middleware yang dipasang langsung padanya (urut pendaftaran).
type Router struct {
	ID          int          `json:"id"`
	Parent      int          `json:"parent"` // -1 = akar
	Var         string       `json:"var"`
	Framework   string       `json:"framework"`
	Prefix      string       `json:"prefix"` // prefix penuh
	Pos         string       `json:"pos"`
	Stack       []string     `json:"stack,omitempty"` // rantai pemanggilan setup dari main sampai router dibuat
	Middlewares []Middleware `json:"middlewares,omitempty"`
}

// Middleware terdaftar (Use/Group/With/argumen rute).
type Middleware struct {
	Name         string   `json:"name"`
	Scope        string   `json:"scope"` // global, group, prefix, route
	ScopePath    string   `json:"scope_path,omitempty"`
	Pos          string   `json:"pos"`            // tempat didaftarkan
	Decl         string   `json:"decl,omitempty"` // tempat didefinisikan (fungsi lokal)
	Auth         bool     `json:"auth,omitempty"`
	Optional     bool     `json:"optional,omitempty"` // auth opsional (OptionalAuth): tidak menolak request anonim
	SkipPrefixes []string `json:"skip_prefixes,omitempty"`
	SkipPaths    []string `json:"skip_paths,omitempty"`
	Reason       string   `json:"reason,omitempty"` // untuk middleware tidak aktif: kenapa
	seq          int
	router       int
}

// Response = cara handler menjawab request.
type Response struct {
	Status int    `json:"status,omitempty"` // 0 = tidak diketahui statis
	Kind   string `json:"kind"`             // json, status, string, file, redirect, error, render, next, stream
	Body   string `json:"body,omitempty"`
	Via    string `json:"via,omitempty"` // helper yang menghasilkan response
	Pos    string `json:"pos"`
}

// Route = satu endpoint.
type Route struct {
	Method      string       `json:"method"`
	Path        string       `json:"path"`
	Pos         string       `json:"pos"`
	RouterID    int          `json:"router_id"`
	Stack       []string     `json:"stack,omitempty"`
	Handler     string       `json:"handler"`
	HandlerDecl string       `json:"handler_decl,omitempty"`
	Chain       []Middleware `json:"chain,omitempty"`       // middleware efektif, urut eksekusi
	Inactive    []Middleware `json:"inactive,omitempty"`    // terdaftar pada prefix rute ini tetapi tidak berjalan
	Auth        bool         `json:"auth"`                  // ada middleware auth wajib di rantai
	Unreachable bool         `json:"unreachable,omitempty"` // didaftarkan di fungsi setup yang tidak terjangkau dari main (kode mati / wiring dinamis)
	Responses   []Response   `json:"responses,omitempty"`
	Calls       []string     `json:"calls,omitempty"` // fungsi lokal yang dijangkau handler (BFS, terbatas)
	seq         int
	handlerKey  string
	body        *ast.BlockStmt // badan handler (untuk deteksi validasi ganda)
	bodyFile    *fileInfo
}
