# Scanner Framework Adapters Ecosystem

Direktori ini (`internal/scanner/adapters`) mengelompokkan seluruh adapter scanner rute dan endpoint REST API ke dalam satu folder terisolasi yang rapi, modular, dan terstandardisasi. Tujuannya adalah agar **AI Coding Agent** maupun developer dapat melacak, memahami, dan menambahkan adapter framework baru dengan sangat mudah tanpa menyentuh bagian inti sistem.

---

## 🏛️ Arsitektur & Antarmuka Utama

Semua adapter mengimplementasikan antarmuka (`RouterScanner`) yang didefinisikan dalam [`types.go`](types.go):

```go
type RouterScanner interface {
    Name() string                                // Nama deskriptif framework
    Type() string                                // "backend" atau "frontend"
    Detect(dir string) bool                      // Heuristik deteksi apakah direktori menggunakan framework ini
    Scan(dir string) ([]models.RouteInfo, error) // Ekstraksi daftar rute/endpoint
}

type BackendAdapter interface {
    RouterScanner
}

type FrontendAdapter interface {
    RouterScanner
}
```

---

## 📂 Struktur Berkas dalam Folder Ini

| File | Kategori | Deskripsi |
| :--- | :--- | :--- |
| [`types.go`](types.go) | **Core Contract** | Definisi interface `RouterScanner`, `BackendAdapter`, `FrontendAdapter`, dan utilitas file I/O bersama (`CleanURLPath`, `fileExists`, `readFileString`, dll). |
| [`registry.go`](registry.go) | **Core Registry** | `ScannerRegistry` & `DefaultRegistry` untuk mengelola urutan prioritas eksekusi adapter dan scan otomatis. |
| [`metadata.go`](metadata.go) | **Catalog Metadata** | Data katalog kaya (deskripsi, ikon, aturan deteksi, fitur) untuk API `/api/adapters` dan UI dashboard. |
| [`api_extractor.go`](api_extractor.go) | **Helper AST/Regex** | Ekstraktor pemanggilan endpoint REST API (`useFetch`, `$fetch`, `axios`, `fetch`) dari kode frontend. |
| [`adapters_test.go`](adapters_test.go) | **Unit Tests** | Suite pengujian otomatis untuk memvalidasi parser tiap adapter. |
| **Backend Adapters** | | |
| [`go_adapter.go`](go_adapter.go) | Backend | Mendukung **Go**: Fiber v2/v3, Gin, Echo, Chi, net/http. |
| [`laravel_adapter.go`](laravel_adapter.go) | Backend | Mendukung **PHP**: Laravel `routes/api.php`, `routes/web.php`, `Route::resource`. |
| [`codeigniter_adapter.go`](codeigniter_adapter.go) | Backend | Mendukung **PHP**: CodeIgniter 4 (`php spark routes` & static AST) dan CodeIgniter 3. |
| [`express_nest_adapter.go`](express_nest_adapter.go) | Backend | Mendukung **Node.js**: NestJS `@Controller`, Express `router.get/post`, Fastify. |
| [`python_adapter.go`](python_adapter.go) | Backend | Mendukung **Python**: FastAPI `@app.get/@router.post`, Flask `@blueprint.route`, Django `urlpatterns`. |
| **Frontend Adapters** | | |
| [`nuxt_adapter.go`](nuxt_adapter.go) | Frontend | Mendukung **Nuxt 3/4**: file-system `pages/**/*.vue`, dynamic `[id].vue`, `server/api/`. |
| [`nextjs_adapter.go`](nextjs_adapter.go) | Frontend | Mendukung **Next.js**: App Router (`app/**/page.tsx`) dan Pages Router (`pages/**/*.tsx`). |
| [`sveltekit_adapter.go`](sveltekit_adapter.go) | Frontend | Mendukung **SvelteKit**: `src/routes/**/+page.svelte` dan `+server.ts`. |
| [`vue_router_adapter.go`](vue_router_adapter.go) | Frontend | Mendukung **Vue 3 Vite SPA**: parser array `createRouter({ routes: [...] })`. |
| [`react_router_adapter.go`](react_router_adapter.go) | Frontend | Mendukung **React SPA**: JSX `<Route path=...>` dan `createBrowserRouter(...)`. |
| [`generic_frontend_adapter.go`](generic_frontend_adapter.go) | Frontend | **Fallback universal**: Menemukan file statis HTML, template MPA, dan views kustom. |

---

## 🚀 Panduan AI Agent: Cara Menambahkan Adapter Baru dalam 3 Langkah

Jika sebuah AI Agent diminta untuk menambahkan dukungan framework baru (contoh: *Ruby on Rails*, *Spring Boot*, *ASP.NET Core*, *Astro*, atau *Remix*), ikuti 3 langkah sederhana berikut:

### Langkah 1: Buat Berkas Adapter Baru
Buat berkas `[framework]_adapter.go` di folder ini, misalnya `astro_adapter.go`:

```go
package adapters

import (
    "path/filepath"
    "github.com/vibeasik/vibeasik/pkg/models"
)

type AstroAdapter struct{}

func NewAstroAdapter() *AstroAdapter {
    return &AstroAdapter{}
}

func (a *AstroAdapter) Name() string {
    return "Astro (Content & SSR)"
}

func (a *AstroAdapter) Type() string {
    return "frontend" // atau "backend"
}

func (a *AstroAdapter) Detect(dir string) bool {
    return fileExists(filepath.Join(dir, "astro.config.mjs")) ||
           fileExists(filepath.Join(dir, "astro.config.ts"))
}

func (a *AstroAdapter) Scan(dir string) ([]models.RouteInfo, error) {
    var routes []models.RouteInfo
    // Lakukan static traversal direktori src/pages/ ...
    return routes, nil
}
```

### Langkah 2: Daftarkan di `registry.go`
Buka [`registry.go`](registry.go) dan tambahkan adapter ke `NewDefaultRegistry()`:

```go
// Untuk backend:
r.RegisterBackend(NewAstroAdapter())

// ATAU untuk frontend (urutkan sebelum generic fallback):
r.RegisterFrontend(NewAstroAdapter())
```

### Langkah 3: Daftarkan Metadata di `metadata.go`
Buka [`metadata.go`](metadata.go) dan tambahkan entri ke `GetRegisteredAdaptersMetadata()`:

```go
{
    ID:          "astro",
    Name:        "Astro",
    Type:        "frontend",
    Language:    "typescript",
    Framework:   "astro",
    Icon:        "i-simple-icons-astro",
    Description: "Pemindaian file-system pages Astro (*.astro, *.md, *.mdx) dan endpoints API.",
    DetectRules: []string{"astro.config.mjs atau astro.config.ts"},
    ScanMethods: []string{"File walk src/pages/"},
    Features:    []string{"Markdown/MDX routing", "Dynamic params", "API endpoints"},
    ExampleFile: "src/pages/**/*.astro",
    Status:      "active",
},
```

### Langkah 4: Jalankan Test
Jalankan perintah pengujian Go:
```bash
go test ./internal/scanner/adapters/...
```
