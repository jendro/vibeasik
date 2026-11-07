# Panduan Komparasi Arsitektur & Desain Pola Backend (Scale-Up Guide)

> **Tujuan Dokumen:**  
> Dokumen ini merupakan referensi teknis dan catatan pembanding komprehensif untuk mempelajari, menganalisis, dan meningkatkan skala (*scale up*) arsitektur Backend (BE).  
> Dokumen ini membedah arsitektur backend dari perspektif **Router**, **Controller/Handler**, penggunaan **Fungsi vs Class/Struct**, serta integrasi protokol modern seperti **MCP (Model Context Protocol)** dan paradigma **MVPP (Model-View-Presenter-Planner)**.

---

## 📑 Daftar Isi
1. [Mengapa Perlu Scale Up Backend?](#1-mengapa-perlu-scale-up-backend)
2. [Peta Anatomi Backend: 3 Sudut Pandang Kritis](#2-peta-anatomi-backend-3-sudut-pandang-kritis)
   - [2.1 Sudut Pandang Router](#21-sudut-pandang-router)
   - [2.2 Sudut Pandang Controller / Handler](#22-sudut-pandang-controller--handler)
   - [2.3 Sudut Pandang Penggunaan Fungsi vs Class / Struct](#23-sudut-pandang-penggunaan-fungsi-vs-class--struct)
3. [Matriks Komparasi 5 Design Pattern Backend Populer](#3-matriks-komparasi-5-design-pattern-backend-populer)
4. [Evolusi & Blueprint Desain Struktur Folder](#4-evolusi--blueprint-desain-struktur-folder)
   - [Pattern 1: Traditional Layered (N-Tier)](#pattern-1-traditional-layered-n-tier)
   - [Pattern 2: Clean / Hexagonal (Ports & Adapters)](#pattern-2-clean--hexagonal-ports--adapters)
   - [Pattern 3: Vertical Slice Architecture](#pattern-3-vertical-slice-architecture)
   - [Pattern 4: MCP-Native Backend Architecture](#pattern-4-mcp-native-backend-architecture)
   - [Pattern 5: MVPP (Model - View - Presenter - Planner)](#pattern-5-mvpp-model---view---presenter---planner)
5. [Membedah MVPP & MCP: Apa & Mengapa Relevan Sekarang?](#5-membedah-mvpp--mcp-apa--mengapa-relevan-sekarang)
6. [Studi Kasus: Arsitektur Backend Vibeasik Saat Ini](#6-studi-kasus-arsitektur-backend-vibeasik-saat-ini)
7. [Panduan Langkah Demi Langkah (Scale-Up Roadmap)](#7-panduan-langkah-demi-langkah-scale-up-roadmap)

---

## 1. Mengapa Perlu Scale Up Backend?

Saat membangun aplikasi backend tahap awal (MVP atau prototipe), developer cenderung menempatkan semua kode dalam beberapa file besar (misal satu file `main.go` atau `index.ts` yang berisi rute, query SQL, manipulasi JSON, dan logika bisnis).

Seiring bertambahnya fitur:
1. **Coupling Tinggi:** Mengubah skema database merusak handler HTTP.
2. **Sulit Diuji:** Logika bisnis tidak bisa diuji unit test tanpa harus menyalakan server HTTP dan database fisik.
3. **Konflik Tim & AI Context Overload:** File besar melebihi batas token AI coding agent, memicu halusinasi dan kegagalan refactor.
4. **Keterbatasan Ekosistem AI:** AI agent modern membutuhkan endpoint tipe-aman (seperti MCP dan System One) bukan sekadar REST API monolitik.

Scale-Up bukan berarti langsung membuat microservices yang rumit, melainkan merapikan batasan tanggung jawab (*Separation of Concerns*) dan menerapkan pola yang tepat sesuai ukuran sistem.

---

## 2. Peta Anatomi Backend: 3 Sudut Pandang Kritis

Untuk menganalisis pola arsitektur sebuah codebase backend, kita cukup menginspeksi **tiga titik kritis**:

```
[ Klien / AI Agent ]
        │
        ▼ (1. Router)
┌────────────────────────────────────────────────────────┐
│ Router: Menerima request, validasi jalur, middleware   │
└───────────────────────┬────────────────────────────────┘
                        │
                        ▼ (2. Controller / Handler)
┌────────────────────────────────────────────────────────┐
│ Controller: Parsing DTO, validasi input, delegasi data │
└───────────────────────┬────────────────────────────────┘
                        │
                        ▼ (3. Fungsi vs Class / Struct)
┌────────────────────────────────────────────────────────┐
│ Service & Domain: Logika bisnis murni & persistensi    │
│ • Pure Functions untuk kalkulasi stateless             │
│ • Struct/Class + Interface untuk state & dependency    │
└────────────────────────────────────────────────────────┘
```

---

### 2.1 Sudut Pandang Router

Router adalah gerbang depan backend. Cara router diorganisir mencerminkan kematangan arsitektur sistem.

| Indikator Evaluasi | Kode Awal / Berbau (*Smelly*) | Kode yang Sudah Di-Scale Up (*Clean*) |
| :--- | :--- | :--- |
| **Pendaftaran Rute** | Semua endpoint ditulis datar dalam satu fungsi `main()` sepanjang 500 baris. | Rute dikelompokkan modular per domain (`api.Group("/projects")`, `api.Group("/diagnostics")`). |
| **Logika di Router** | Route handler berupa inline anonymous closure dengan query database langsung: `app.Get("/users", func(c) { db.Find(...) })`. | Router hanya memetakan URL ke method controller/handler terpisah: `api.Get("/projects", s.getProjects)`. |
| **Middleware Pipeline** | Validasi token, CORS, dan rate limit dicampur ke dalam body handler rute. | Middleware dirangkai deklaratif: `api.Use(corsMiddleware, authMiddleware, rateLimiter)`. |
| **Protokol Eksternal** | Hanya mendukung HTTP REST standar. | Menyediakan transport ganda: REST HTTP (`/api/*`) dan MCP JSON-RPC (`/mcp` stdio/SSE) untuk AI Agent. |

---

### 2.2 Sudut Pandang Controller / Handler

Controller (atau Handler di Fiber/Gin/Express) bertindak sebagai mediator antara protokol transport (HTTP, WebSocket, gRPC, CLI) dan logika bisnis.

#### Ciri Fat Controller (Anti-Pattern):
```go
// ❌ CONTOH FAT CONTROLLER (BURUK):
func CreateOrderHandler(c *fiber.Ctx) error {
    var req OrderRequest
    c.BodyParser(&req)
    
    // 1. Validasi manual bercampur
    if req.Total <= 0 { return c.Status(400).SendString("Invalid total") }
    
    // 2. Query SQL langsung di handler
    db.Exec("INSERT INTO orders ...")
    
    // 3. Side-effect langsung (kirim email, buat log)
    smtp.SendMail(...)
    
    return c.JSON(req)
}
```
*Dampak buruk:* Tidak bisa diuji secara unit test tanpa server HTTP dan koneksi database nyata.

#### Ciri Thin Controller (Best Practice Scale-Up):
```go
// ✅ CONTOH THIN CONTROLLER (TER-SCALE):
func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
    var req dto.CreateOrderRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
    }
    
    // Delegasikan 100% logika ke Service Layer via Interface
    order, err := h.orderService.PlaceOrder(c.Context(), req.ToDomain())
    if err != nil {
        return handleServiceError(c, err)
    }
    
    return c.Status(fiber.StatusCreated).JSON(dto.FromDomain(order))
}
```
*Kelebihan:* Controller hanya peduli serialisasi/deserialisasi HTTP. Logika `PlaceOrder` dapat diuji unit test dengan mock service dalam milidetik.

---

### 2.3 Sudut Pandang Penggunaan Fungsi vs Class / Struct

Di bahasa modern (seperti Go, TypeScript, Python), muncul pertanyaan krusial: **kapan menggunakan pure functions, dan kapan menggunakan struct/class dengan interface?**

#### A. Kapan Menggunakan Pure Functions?
Pure function adalah fungsi murni deterministik tanpa *side effect*, tanpa mengakses database atau state global. Outputnya 100% hanya bergantung pada input argumennya.
- **Contoh Penggunaan:**
  - `service.Slugify(text string) string`
  - `sanitizer.MaskSecret(val string) string`
  - `validator.ValidateEmail(email string) bool`
  - `calculator.ComputeHealthScore(findings []Finding) int`
- **Keunggulan:** Sangat cepat, hemat memori, aman terhadap concurency (thread-safe), dan pengujian unit test 100% tidak memerlukan mocking.

#### B. Kapan Menggunakan Class / Struct dengan Interface?
Gunakan Struct/Class ber-state saat unit kode membutuhkan:
1. **Dependency Injection:** Memerlukan akses ke koneksi database, client HTTP cloud, logger, atau konfigurasi runtime.
2. **Inversion of Control (IoC):** Memerlukan abstraksi interface agar implementasinya bisa ditukar (misalnya: `ProjectRepository` bisa berupa `GORMProjectRepository` atau `MockProjectRepository` saat unit testing).
3. **Lifecycle Management:** Komponen yang memiliki inisialisasi awal (`NewServer()`) dan penutupan koneksi (`Close()`).

---

## 3. Matriks Komparasi 5 Design Pattern Backend Populer

| Kriteria Evaluasi | 1. Traditional Layered (N-Tier) | 2. Clean / Hexagonal (Ports & Adapters) | 3. Vertical Slice Architecture | 4. MCP-Native Architecture | 5. MVPP (Model-View-Presenter-Planner) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Filosofi Utama** | Pemisahan layer horizontal teknis (Controller -> Service -> Repo -> DB). | Domain di inti, dependensi mengarah ke dalam (*Inversion of Control*). | Pengelompokan berdasarkan fitur/use-case, bukan layer teknis. | Backend mengekspos REST API sekaligus Tool AI berstandar JSON-RPC 2.0. | Memisahkan orkestrasi kontrol (Presenter) dari penalaran AI (Planner). |
| **Karakteristik Router** | Route mapping langsung ke Handler per domain. | Primary Driving Adapter yang meneruskan input ke Use Case. | Setiap rute langsung memanggil Command/Query handler spesifik fitur. | Route HTTP standar didampingi oleh transport `/mcp` (stdio & SSE). | Router meneruskan request ke Presenter; Presenter membagi tugas ke Service & Planner. |
| **Karakteristik Controller** | Thin/Semi-fat controller memanggil Service. | Driving Adapter mentransformasikan DTO ke Request Interactor. | Handler sangat ramping (MediatR / minimal dispatcher). | Controller HTTP & MCP Tool Executor berbagi service domain yang sama. | Presenter mengorkestrasi alur data deterministik dan hasil penalaran dari Planner. |
| **Karakteristik Logic & Func** | Service struct mengelola sekumpulan operasi CRUD domain. | Domain Entity kaya aturan (*Rich Domain Model*), bebas library pihak ketiga. | Pure functions & command structs per use-case (1 file = 1 feature). | Tool Call functions memetakan schema JSON input ke domain logic. | Planner menjalankan primitif AI (*choice, noul, score*) atau LLM reasoning chain. |
| **Tingkat Kompleksitas** | ⭐⭐ Rendah (Sangat mudah dipelajari). | ⭐⭐⭐⭐ Menengah-Tinggi (Banyak interface & boiler plate DTO). | ⭐⭐⭐ Menengah (Sangat modular, minim file jumping). | ⭐⭐⭐ Menengah (Sangat ramah AI Agent). | ⭐⭐⭐ Menengah-Tinggi (Kombinasi deterministik + probabilistik). |
| **Kemudahan Testing** | Mudah jika interface repository diinjeksi. | Sangat mudah (Core domain 100% terisolasi dari framework). | Sangat mudah (Test fokus pada input/output per slice). | Sangat mudah (Tool call dapat diuji langsung via payload JSON-RPC). | Luar biasa: Deterministik & AI logic diuji pada suite terpisah. |
| **Kapan Tepat Dipakai?** | Aplikasi CRUD standar, tim kecil, produk tahap awal - menengah. | Sistem core enterprise, aturan bisnis kompleks, database dinamis. | Aplikasi skala besar dengan banyak developer dan fitur independen. | Aplikasi yang dibangun bersama AI Agent (Vibecoding, Cursor, Antigravity). | Sistem yang memadukan automasi bisnis dengan keputusan cerdas LLM. |

---

## 4. Evolusi & Blueprint Desain Struktur Folder

Berikut adalah cetak biru struktur folder untuk masing-masing pola agar dapat diaplikasikan langsung pada kode Go atau TypeScript:

### Pattern 1: Traditional Layered (N-Tier)
```
backend/
├── cmd/
│   └── api/main.go             # Inisialisasi koneksi DB dan HTTP server
├── config/                     # Konfigurasi environment (.env)
├── internal/
│   ├── api/                    # HTTP Handlers / Controllers
│   │   ├── project_handler.go
│   │   └── user_handler.go
│   ├── service/                # Business Logic Services
│   │   ├── project_service.go
│   │   └── user_service.go
│   ├── repository/             # Database Access Layer (SQL / GORM)
│   │   ├── project_repo.go
│   │   └── user_repo.go
│   └── models/                 # Domain Entities & DTOs
│       ├── project.go
│       └── user.go
```

---

### Pattern 2: Clean / Hexagonal (Ports & Adapters)
```
backend/
├── cmd/
│   └── main.go                 # Dependency wiring (Composition Root)
├── internal/
│   ├── core/                   # INTI DOMAIN (Zero dependency ke luar)
│   │   ├── domain/             # Entitas bisnis murni
│   │   │   └── project.go
│   │   ├── ports/              # Interfaces (Kontrak)
│   │   │   ├── inbound.go      # Driving Ports (Use Cases / Services)
│   │   │   └── outbound.go     # Driven Ports (Repo, Cloud Client, Mailer)
│   │   └── services/           # Implementasi Use Case bisnis
│   │       └── project_usecase.go
│   └── adapters/               # ADAPTER TEKNIS (Depend ke Core Ports)
│       ├── inbound/            # Driving Adapters
│       │   ├── http/           # Fiber / Gin HTTP Handlers
│       │   └── cli/            # CLI Command runner
│       └── outbound/           # Driven Adapters
│           ├── persistence/    # PostgreSQL / SQLite GORM Repository
│           └── cloud/          # HTTP Client ke API eksternal
```

---

### Pattern 3: Vertical Slice Architecture
```
backend/
├── cmd/main.go
├── common/                     # Cross-cutting concerns (DB conn, logger, errors)
└── features/                   # SLICE PER FITUR (Semua hal tentang fitur ada di sini!)
    ├── create_project/
    │   ├── handler.go          # HTTP Handler khusus create project
    │   ├── command.go          # Command DTO & Validasi input
    │   ├── repository.go       # Kueri SQL spesifik create project
    │   └── service.go          # Logika spesifik pembuatan project
    ├── list_projects/
    │   ├── handler.go
    │   ├── query.go
    │   └── repository.go
    └── delete_project/
        ├── handler.go
        └── command.go
```

---

### Pattern 4: MCP-Native Backend Architecture
*(Pola yang diterapkan pada Vibeasik Local Engine)*
```
backend/
├── cmd/                        # CLI Entrypoint (serve, scan, mcp stdio)
├── internal/
│   ├── api/                    # Inbound Port 1: REST API (Fiber HTTP)
│   │   ├── server.go
│   │   └── project_handler.go
│   ├── mcp/                    # Inbound Port 2: Model Context Protocol Server
│   │   ├── server.go           # JSON-RPC 2.0 Handler
│   │   ├── sse.go              # Server-Sent Events Transport
│   │   └── tools.go            # Tool definitions (list_projects, run_diagnostics)
│   ├── service/                # Core Business Logic (Disharing oleh API & MCP)
│   │   └── project_service.go
│   ├── repository/             # Outbound Port: Data Persistence
│   │   └── project_repo.go
│   └── models/                 # Domain Models & Shared Schemas
```

---

### Pattern 5: MVPP (Model - View - Presenter - Planner)
```
backend/
├── cmd/main.go
├── internal/
│   ├── models/                 # [M] MODEL: Persistent Data & State
│   │   ├── project.go          # GORM Schema & Invariants
│   │   └── audit_finding.go
│   │
│   ├── views/                  # [V] VIEW / SERIALIZER: Respon DTO & Formatter
│   │   ├── project_view.go     # JSON Response Envelope, XML/SSE formats
│   │   └── mcp_view.go         # MCP Tool Output Formatter
│   │
│   ├── presenters/             # [P] PRESENTER: Controller & Request Orchestrator
│   │   ├── project_presenter.go# Handle HTTP status, cookie, route params
│   │   └── dispatch_presenter.go
│   │
│   ├── planners/               # [P] PLANNER: AI & Cognitive Reasoning Engine
│   │   ├── systemone_client.go # Primitives: choice, noul, score
│   │   ├── heuristic_engine.go # Heuristic rule evaluators
│   │   ├── prompt_templates.go # LLM Prompts & Cognitive graphs
│   │   └── scaleup_planner.go  # Rekomendasi scale-up berbasis AI
│   │
│   └── services/               # Deterministic Business Services
│       └── project_service.go
```

---

## 5. Membedah MVPP & MCP: Apa & Mengapa Relevan Sekarang?

### Apa itu MVPP?
**MVPP** merupakan evolusi dari pola MVC/MVP untuk era aplikasi modern bertenaga kecerdasan buatan (*AI-Native Applications*):

1. **Model (M):** Sumber kebenaran data (*Single Source of Truth*), skema basis data, dan aturan validasi entitas.
2. **View / Serializer (V):** Format tampilan respon data (REST JSON, GraphQL, Server-Sent Events, atau Markdown/JSON-RPC untuk AI Agent).
3. **Presenter / Controller (P):** Mengelola siklus request, otentikasi, validasi parameter rute, dan mengorkestrasi kapan harus memanggil logika bisnis vs kapan harus meminta pertimbangan AI.
4. **Planner (P):** **Lapisan Kognitif Terisolasi.** Alih-alih menulis *prompt* LLM acak di dalam handler controller, layer Planner mengonsentrasikan seluruh logika probabilistik:
   - Evaluasi kelayakan (*score*).
   - Pengambilan keputusan biner (*noul*).
   - Klasifikasi terstruktur (*choice*).
   - Chain-of-Thought / ReAct planning loop.

*Keuntungan Utama:* Jika provider AI mati atau lambat, Presenter memiliki fallback deterministik yang aman, dan kode bisnis tidak terkontaminasi oleh prompt engineering.

### Apa itu MCP (Model Context Protocol)?
MCP adalah standar terbuka yang digagas Anthropic dan didukung luas oleh ekosistem developer (termasuk Antigravity, Claude Code, Cursor, Cline).
- Menghilangkan integrasi API kustom untuk setiap AI tool.
- Memungkinkan backend bertindak sebagai **Server Konteks** yang mengekspos:
  1. **Tools:** Fungsi yang bisa dieksekusi agent (misal `vibeasik_run_local_agent`, `scan_directory`).
  2. **Resources:** Data yang bisa dibaca agent (misal isi file konfigurasi, log server).
  3. **Prompts:** Template instruksi bawaan sistem.

---

## 6. Studi Kasus: Arsitektur Backend Vibeasik Saat Ini

Codebase `vibeasik/local-engine` saat ini mengadopsi kombinasi **Layered Clean Architecture** dengan **MCP-Native Adapter**:

1. **Router & Handler:**
   - Didefinisikan di `internal/api/server.go`. Menggunakan Fiber v2.
   - Handlers dibagi rapi per domain (`project_handler.go`, `route_handler.go`, `diagnostic_handler.go`).
2. **Business Services & Repositories:**
   - Menggunakan interface terpisah (`internal/service/` dan `internal/repository/`).
   - Menerapkan *Dependency Injection* melalui constructor struct (`NewProjectService`).
3. **MCP Integration:**
   - Menyediakan `internal/mcp/server.go` dengan 13 tools terdaftar. Mendukung stdio CLI dan HTTP SSE.
4. **Peluang Scale-Up ke Depan:**
   - Mentransformasikan Heuristic & AI Review ke dalam layer **Planner** terpisah (MVPP).
   - Memindahkan validasi pembuatan direktori fisik dari handler ke domain service.

---

## 7. Panduan Langkah Demi Langkah (Scale-Up Roadmap)

Jika Anda memiliki backend monolitik atau berantakan, ikuti 4 langkah scale-up teruji berikut:

```
Langkah 1: Rapikan Router & Handler
  ├── Pisahkan rute ke dalam Route Groups per domain
  └── Pindahkan inline handler ke Controller struct
         │
         ▼
Langkah 2: Ekstraksi Business Logic ke Service Layer
  ├── Keluarkan query SQL dari Handler
  ├── Buat Struct Service dengan interface
  └── Controller hanya bertugas mem-parse input & format output
         │
         ▼
Langkah 3: Abstraksi Database ke Repository Layer
  ├── Buat Repository Interface (CRUD & kueri spesifik)
  └── Implementasikan GORM / SQL Driver di package terisolasi
         │
         ▼
Langkah 4: Tambahkan Lapisan Modern (MCP & Planner / MVPP)
---

## 8. Visi Skalabilitas Menyeluruh: Ekosistem Multi-Tier (BE, Mobile, dan Frontend)

Platform Scale-Up Vibeasik dirancang untuk mematangkan aplikasi secara *end-to-end* sesuai *real-world production best practice*:

1. **Backend (BE) — Fondasi Logika & Data:**
   - Memastikan pemisahan Router, Thin Controller, Service, Repository, serta kesiapan dual-transport (HTTP REST & MCP JSON-RPC).
   - Mengadopsi isolasi kognitif MVPP untuk integrasi AI System One / Jev.

2. **Mobile Apps (Flutter / React Native) — Fondasi Portabilitas & Offline:**
   - **Clean Architecture + BLoC / Riverpod:** Memisahkan lapisan Presentation, Domain (UseCases murni tanpa dependensi framework), dan Data (Repository + DataSources).
   - **Feature-First Layout:** Struktur `lib/features/[nama_fitur]/` untuk mencegah konflik tim dan memudahkan AI coding agent.
   - **Offline-First & Local Storage:** Integrasi database lokal cepat (Isar/Hive/Drift) dengan antrean sinkronisasi background.
   - **Declarative Navigation:** Rute bertipe aman (`go_router`) dengan dukungan deep linking.

3. **Frontend Web (Nuxt 4 / Vue / React) — Fondasi UI & Performa:**
   - **Single Responsibility Components:** Komponen di bawah 250 baris dengan pemisahan Presentational (Dumb) dan Container (Smart).
   - **Automation-First Design:** Meminimalkan informasi berlebih di layar, mengotomasi proses di background, dan menyajikan drill-down hanya saat dibutuhkan.
   - **Composable State & SSR:** Pengelolaan state reaktif modular (`useApi`, `useVibeasik`) dan optimasi Core Web Vitals (LCP < 2.5s).
   - **Type-Safe Domain Contracts:** Berbagi definisi DTO antarmuka di `app/types/` tanpa penggunaan `any`.

---
*Dokumen ini dikelola secara terpusat di `guidelines/08_scale_up_architecture_patterns.md`.*
