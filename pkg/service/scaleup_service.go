package service

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
	"github.com/vibeasik/vibeasik/pkg/util"
)

// Catatan Audit Keamanan (ROUTER-UNGUARDED-MUTATION):
// RouterCodeSample dan ControllerCodeSample di berkas ini adalah string literal
// representasi pola arsitektur untuk edukasi (Layered, Hexagonal, Modular Monolith),
// bukan instansiasi router aktif. Router aplikasi riil dikonfigurasi di pkg/api/server.go
// dengan perlindungan perimeter global fail-closed (s.localAccess).
//
// ScaleUpService coordinates architectural pattern learning and codebase evaluation
type ScaleUpService interface {
	GetPatterns() []models.ArchitecturePattern
	EvaluateComponent(ctx context.Context, componentID uint) (*models.ArchitectureEvaluation, error)
	EvaluateProject(ctx context.Context, projectID uint) ([]models.ArchitectureEvaluation, error)
}

type scaleUpService struct {
	componentRepo repository.ComponentRepository
	projectRepo   repository.ProjectRepository
}

// NewScaleUpService initializes ScaleUpService
func NewScaleUpService(componentRepo repository.ComponentRepository, projectRepo repository.ProjectRepository) ScaleUpService {
	return &scaleUpService{
		componentRepo: componentRepo,
		projectRepo:   projectRepo,
	}
}

func (s *scaleUpService) GetPatterns() []models.ArchitecturePattern {
	return []models.ArchitecturePattern{
		{
			ID:                        "layered",
			Name:                      "Traditional Layered (N-Tier)",
			Badge:                     "Fondasi Standar",
			ShortDesc:                 "Pemisahan horizontal berdasarkan tanggung jawab teknis: Router -> Controller -> Service -> Repository -> Model.",
			Philosophy:                "Memisahkan kode ke dalam lapisan horizontal. Setiap lapisan hanya boleh berkomunikasi dengan lapisan tepat di bawahnya. Mudah dipahami oleh pemula dan tim kecil.",
			RouterCharacteristics:     "Router mendefinisikan rute dan memanggil method handler/controller per domain bisnis (misal: /api/projects -> projectHandler).",
			ControllerCharacteristics: "Handler mem-parsing DTO dari HTTP request, memvalidasi input dasar, memanggil service layer, dan membalas dengan status code HTTP yang sesuai.",
			FunctionVsClassPhilosophy: "Service dan Repository menggunakan Struct/Class dengan Interface untuk Dependency Injection. Fungsi-fungsi utilitas dan kalkulasi murni dijadikan pure functions.",
			McpIntegration:            "Biasanya belum terintegrasi secara bawaan. MCP dapat ditambahkan sebagai handler terpisah di samping HTTP router.",
			MvppRole:                  "Dasar dari MVPP, di mana Presenter = Controller, Model = Entities/DB, namun belum memiliki layer Planner mandiri.",
			ComplexityScore:           2,
			TestabilityScore:          4,
			AiReadinessScore:          3,
			ScalabilityScore:          3,
			Pros: []string{
				"Sangat mudah dipelajari dan diimplementasikan oleh tim baru",
				"Struktur folder intuitif dan teratur per jenis file",
				"Mudah memisahkan query SQL dari antarmuka HTTP",
				"Didukung luas oleh framework Go (Fiber/Gin) dan Node.js (Express/Nest)",
			},
			Cons: []string{
				"Lapisan horizontal bisa menjadi bottleneck jika fitur bertambah ratusan",
				"Risiko Service layer menjadi god-object yang terlalu gemuk",
				"Developer sering tergoda melempar entitas database langsung ke response klien",
			},
			WhenToUse: []string{
				"Aplikasi CRUD standar dengan skala kecil hingga menengah",
				"Pengembangan prototipe cepat yang tetap butuh struktur rapi",
				"Tim developer yang sedang bertransisi dari skrip monolit ke arsitektur bersih",
			},
			FolderBlueprintTree: `backend/
├── cmd/
│   └── api/
│       └── main.go           # Entrypoint & inisialisasi server
├── internal/
│   ├── api/                  # [ROUTER & CONTROLLER] Transport Layer
│   │   ├── server.go         # Router definition & middleware
│   │   ├── project_handler.go
│   │   └── user_handler.go
│   ├── service/              # [SERVICE] Business Logic Layer
│   │   ├── project_service.go
│   │   └── user_service.go
│   ├── repository/           # [REPOSITORY] Data Access Layer (GORM/SQL)
│   │   ├── project_repo.go
│   │   └── user_repo.go
│   └── models/               # [MODEL] Domain Entities & DTOs
│       ├── project.go
│       └── user.go
└── pkg/
    └── utils/                # Pure helper functions
        └── slug.go`,
			FolderExplanation: []models.FolderAnnotation{
				{Path: "internal/api/", Role: "Router & Controller", Description: "Menerima request HTTP, validasi payload, dan formatting response JSON."},
				{Path: "internal/service/", Role: "Business Logic", Description: "Berisi aturan bisnis dan orkestrasi operasi antarrepositori."},
				{Path: "internal/repository/", Role: "Persistence", Description: "Mengenkapsulasi kueri database GORM/SQL ke dalam antarmuka interface."},
				{Path: "internal/models/", Role: "Domain Model", Description: "Definisi struct database dan kontrak payload request/response."},
			},
			RouterCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "Router Terstruktur di Layered Architecture",
				Code: `// internal/api/server.go
func (s *Server) setupRoutes() {
    api := s.app.Group("/api")
    
    // Group per domain
    projects := api.Group("/projects")
    projects.Get("/", s.getProjects)
    projects.Post("/", s.createProject)
    projects.Get("/:id", s.getProject)
}`,
				Explanation: "Router memetakan URL ke handler struct tanpa ada logika bisnis inline.",
			},
			ControllerCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "Thin Controller Mendelegasikan ke Service",
				Code: `// internal/api/project_handler.go
func (s *Server) createProject(c *fiber.Ctx) error {
    var req models.CreateProjectDTO
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
    }

    // Panggil Service Layer via Interface
    res, err := s.services.Projects.Create(c.Context(), req)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
    }
    return c.Status(fiber.StatusCreated).JSON(res)
}`,
				Explanation: "Controller hanya bertugas binding DTO dan memanggil service.",
			},
			LogicCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "Service Struct Method vs Pure Function",
				Code: `// Pure Function: Stateless, deterministik, tanpa I/O
func Slugify(title string) string {
    return strings.ToLower(strings.ReplaceAll(title, " ", "-"))
}

// Service Method: Menginjeksi dependency Repository
func (s *projectService) Create(ctx context.Context, dto models.CreateProjectDTO) (*models.Project, error) {
    slug := Slugify(dto.Name)
    p := &models.Project{Name: dto.Name, Slug: slug}
    return s.repo.Save(ctx, p)
}`,
				Explanation: "Pemisahan tegas: kalkulasi stateless jadi pure function, operasi I/O jadi struct method.",
			},
		},
		{
			ID:                        "clean_hexagonal",
			Name:                      "Clean / Hexagonal (Ports & Adapters)",
			Badge:                     "Enterprise Grade",
			ShortDesc:                 "Domain entity murni berada di pusat sistem, terisolasi 100% dari framework, database, dan protokol luar.",
			Philosophy:                "Inversion of Control tingkat tinggi. Inti domain mendefinisikan port (interface), dan dunia luar (HTTP, CLI, DB, AI) adalah adapter yang mencolokkan diri ke port tersebut.",
			RouterCharacteristics:     "Router bertindak murni sebagai Inbound/Driving Adapter yang mengonversi request luar menjadi domain interactor commands.",
			ControllerCharacteristics: "Adapter controller sangat tipis, hanya menjadi penerjemah protokol ke Use Case yang ditentukan di core domain.",
			FunctionVsClassPhilosophy: "Core domain mengedepankan Rich Domain Models (entities yang memiliki method invariant sendiri). Ports didefinisikan sebagai interface murni.",
			McpIntegration:            "Sangat alami: MCP Server cukup dianggap sebagai Inbound Adapter lain di samping HTTP Adapter, mengeksekusi Use Case yang sama persis.",
			MvppRole:                  "Sangat sejalan dengan MVPP: Presenter = Inbound HTTP Adapter, Planner = Inbound AI Adapter, Model = Core Domain Entity.",
			ComplexityScore:           4,
			TestabilityScore:          5,
			AiReadinessScore:          4,
			ScalabilityScore:          5,
			Pros: []string{
				"Core domain 100% independen dari framework web atau database tertentu",
				"Unit testing inti bisnis dapat dijalankan instan tanpa mock DB atau HTTP",
				"Sangat mudah mengganti database (misal SQLite ke Postgres/MongoDB) tanpa menyentuh bisnis logic",
				"Memudahkan penambahan protokol baru (gRPC, MCP, CLI) tanpa duplikasi logika",
			},
			Cons: []string{
				"Memerlukan banyak berkas mapping DTO dan interfaces (boiler plate tinggi)",
				"Overengineering untuk aplikasi CRUD sederhana",
				"Membutuhkan kedisiplinan tinggi dari seluruh anggota tim",
			},
			WhenToUse: []string{
				"Sistem dengan aturan bisnis kompleks dan umur proyek jangka panjang",
				"Aplikasi yang memerlukan banyak pintu masuk (REST API, MCP, WebSocket, CLI)",
				"Sistem perbankan, fintech, atau supply-chain yang menuntut uji integritas ketat",
			},
			FolderBlueprintTree: `backend/
├── cmd/
│   └── main.go               # Composition Root (Dependency Wiring)
├── internal/
│   ├── core/                 # [DOMAIN CORE] Nol dependensi ke luar
│   │   ├── domain/           # Entitas bisnis murni
│   │   │   └── project.go
│   │   ├── ports/            # Interfaces
│   │   │   ├── inbound.go    # UseCase interfaces (Driving)
│   │   │   └── outbound.go   # Repository & Gateway interfaces (Driven)
│   │   └── services/         # Implementasi Use Case bisnis
│   │       └── project_usecase.go
│   └── adapters/             # [ADAPTERS] Mengimplementasikan / Menggunakan Ports
│       ├── inbound/          # Driving Adapters
│       │   ├── http/         # Fiber / Express Router & Controller
│       │   └── mcp/          # Model Context Protocol Handler
│       └── outbound/         # Driven Adapters
│           ├── gorm_repo/    # SQLite / PostgreSQL DB Adapter
│           └── cloud_client/ # External HTTP API Adapter`,
			FolderExplanation: []models.FolderAnnotation{
				{Path: "internal/core/domain/", Role: "Core Domain", Description: "Entitas dan invariant bisnis yang tidak mengenal database atau framework sama sekali."},
				{Path: "internal/core/ports/", Role: "Contract Ports", Description: "Definisi antarmuka Inbound (UseCases) dan Outbound (Persistence/External)."},
				{Path: "internal/adapters/inbound/", Role: "Driving Adapters", Description: "Pintu masuk sistem: HTTP Fiber, MCP Stdio/SSE, CLI commands."},
				{Path: "internal/adapters/outbound/", Role: "Driven Adapters", Description: "Implementasi persistensi database GORM dan klien cloud."},
			},
			RouterCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "HTTP Driving Adapter Router",
				Code: `// internal/adapters/inbound/http/router.go
func NewHTTPAdapter(app *fiber.App, projectUseCase ports.ProjectUseCase) *HTTPAdapter {
    adapter := &HTTPAdapter{useCase: projectUseCase}
    api := app.Group("/api/v1")
    api.Post("/projects", adapter.CreateProject)
    return adapter
}`,
				Explanation: "HTTP Adapter menerima Port interface (Use Case) saat inisialisasi.",
			},
			ControllerCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "Inbound Adapter Controller",
				Code: `// internal/adapters/inbound/http/handler.go
func (h *HTTPAdapter) CreateProject(c *fiber.Ctx) error {
    var cmd domain.CreateProjectCommand
    if err := c.BodyParser(&cmd); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": err.Error()})
    }
    
    project, err := h.useCase.Execute(c.Context(), cmd)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
    return c.Status(201).JSON(project)
}`,
				Explanation: "Adapter hanya mengubah format HTTP menjadi command domain dan memanggil port.",
			},
			LogicCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "Core Domain Entity Invariant Method",
				Code: `// internal/core/domain/project.go
type Project struct {
    ID   string
    Name string
    Slug string
}

// Invariant logic di dalam entitas domain murni
func NewProject(name string) (*Project, error) {
    if strings.TrimSpace(name) == "" {
        return nil, errors.New("nama proyek tidak boleh kosong")
    }
    return &Project{
        Name: name,
        Slug: strings.ToLower(strings.ReplaceAll(name, " ", "-")),
    }, nil
}`,
				Explanation: "Entitas domain memvalidasi integritasnya sendiri tanpa dependensi eksternal.",
			},
		},
		{
			ID:                        "vertical_slice",
			Name:                      "Vertical Slice Architecture",
			Badge:                     "Modular Fitur",
			ShortDesc:                 "Mengelompokkan kode berdasarkan fitur/use-case vertikal, bukan memotongnya menjadi lapisan teknis horizontal.",
			Philosophy:                "Tinggalkan pembagian Controller-Service-Repository yang membuat developer harus melompat ke 4 folder untuk mengedit 1 fitur. Satukan handler, command, validasi, dan kueri SQL fitur tersebut dalam satu folder.",
			RouterCharacteristics:     "Router mendaftarkan command/query handler per slice fitur (misal: POST /projects memanggil CreateProjectSlice).",
			ControllerCharacteristics: "Handler bertindak sebagai dispatcher ringkas yang langsung memetakan request ke slice yang bersangkutan.",
			FunctionVsClassPhilosophy: "Mengutamakan fungsi murni dan struct request/response khusus per slice fitur. Menghindari inheritance dan shared abstraction yang tidak perlu.",
			McpIntegration:            "Setiap slice fitur dapat mengekspos endpoint HTTP sekaligus fungsi MCP Tool di file yang sama.",
			MvppRole:                  "Setiap slice merepresentasikan satu unit MVPP kecil yang mandiri.",
			ComplexityScore:           3,
			TestabilityScore:          5,
			AiReadinessScore:          5,
			ScalabilityScore:          4,
			Pros: []string{
				"Sangat ramah AI Coding Agent: Semua konteks fitur ada dalam 1 folder/file",
				"Tidak ada overhead file jumping saat mengembangkan atau refactor fitur",
				"Menghilangkan god-service dan kueri repo raksasa yang saling senggol",
				"Fitur baru dapat dihapus atau ditambah tanpa risiko merusak fitur lain",
			},
			Cons: []string{
				"Dapat terjadi sedikit duplikasi kode DTO jika dua fitur mirip",
				"Membutuhkan konvensi bersama agar struktur internal per slice konsisten",
				"Belum umum di framework Go konvensional yang terbiasa dengan MVC",
			},
			WhenToUse: []string{
				"Aplikasi bertenaga AI dengan siklus iterasi fitur sangat cepat",
				"Tim lintas fungsi yang mengelola domain-domain independen",
				"Aplikasi yang sering menambah/mengurangi fitur musiman",
			},
			FolderBlueprintTree: `backend/
├── cmd/main.go
├── common/                   # Shared cross-cutting (DB, middleware, errors)
└── features/                 # SLICES PER FITUR
    ├── create_project/
    │   ├── endpoint.go       # HTTP route & handler
    │   ├── mcp_tool.go       # MCP Tool definition & executor
    │   ├── command.go        # Request DTO & validation
    │   └── handler.go        # Business logic & SQL query langsung
    ├── get_project_detail/
    │   ├── endpoint.go
    │   ├── query.go
    │   └── handler.go
    └── delete_project/
        ├── endpoint.go
        └── handler.go`,
			FolderExplanation: []models.FolderAnnotation{
				{Path: "features/create_project/", Role: "Feature Slice", Description: "Semua hal tentang pembuatan proyek (HTTP, MCP, Validasi, SQL) terkumpul di sini."},
				{Path: "features/get_project_detail/", Role: "Feature Slice", Description: "Semua logika query spesifik detail proyek."},
				{Path: "common/", Role: "Cross-Cutting", Description: "Koneksi database GORM, utilitas logger, dan auth middleware bersama."},
			},
			RouterCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "Vertical Slice Feature Registration",
				Code: `// cmd/main.go
func registerFeatures(app *fiber.App, db *gorm.DB) {
    api := app.Group("/api")
    
    // Setiap slice mendaftarkan endpointnya sendiri
    create_project.RegisterEndpoint(api, db)
    get_project_detail.RegisterEndpoint(api, db)
}`,
				Explanation: "Router hanya bertindak sebagai wadah registrasi slice fitur.",
			},
			ControllerCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "Feature Slice Endpoint & Handler",
				Code: `// features/create_project/endpoint.go
func RegisterEndpoint(router fiber.Router, db *gorm.DB) {
    router.Post("/projects", func(c *fiber.Ctx) error {
        var cmd Command
        if err := c.BodyParser(&cmd); err != nil {
            return c.Status(400).JSON(fiber.Map{"error": err.Error()})
        }
        res, err := Handle(c.Context(), db, cmd)
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }
        return c.Status(201).JSON(res)
    })
}`,
				Explanation: "Handler, validasi, dan eksekutor ada di folder fitur yang sama.",
			},
			LogicCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "Direct Query & Logic in Slice Handler",
				Code: `// features/create_project/handler.go
func Handle(ctx context.Context, db *gorm.DB, cmd Command) (*Result, error) {
    if err := cmd.Validate(); err != nil {
        return nil, err
    }
    // Query langsung yang spesifik hanya untuk use case ini
    p := &models.Project{Name: cmd.Name}
    if err := db.WithContext(ctx).Create(p).Error; err != nil {
        return nil, err
    }
    return &Result{ID: p.ID, Name: p.Name}, nil
}`,
				Explanation: "Tidak perlu abstraksi repository generik jika hanya dipakai oleh fitur ini.",
			},
		},
		{
			ID:                        "mcp_native",
			Name:                      "MCP-Native Backend Architecture",
			Badge:                     "AI First",
			ShortDesc:                 "Backend dirancang dari awal untuk melayani klien manusia (REST HTTP) dan AI Coding Agent (MCP JSON-RPC) secara setara.",
			Philosophy:                "AI Coding Agent (Antigravity, Cursor, Claude Code) bukan sekadar pengakses web biasa. Backend menyediakan Model Context Protocol (stdio & SSE) yang mengekspos Tools, Resources, dan Prompts secara terstruktur.",
			RouterCharacteristics:     "Menyediakan dua gerbang transport: HTTP REST (/api/*) untuk frontend web/mobile, dan JSON-RPC 2.0 (/mcp, /mcp/sse) untuk AI agent.",
			ControllerCharacteristics: "Controller HTTP dan MCP Tool Executor berbagi layer logika bisnis yang sama. Tool call menerima schema JSON dan mengembalikan konten terstruktur.",
			FunctionVsClassPhilosophy: "Tools didefinisikan dengan JSON Schema yang ketat. Menggunakan struct method untuk mengelola state koneksi MCP dan pure functions untuk format konten teks.",
			McpIntegration:            "Inti dari arsitektur: backend berperan sebagai MCP Server resmi yang mengekspos kapabilitas sistem.",
			MvppRole:                  "Mewakili kombinasi Model-View-Presenter terpadu dengan kapabilitas AI Agent.",
			ComplexityScore:           3,
			TestabilityScore:          5,
			AiReadinessScore:          5,
			ScalabilityScore:          4,
			Pros: []string{
				"Agen AI dapat mendiagnosa, memindai, dan mengoperasikan backend tanpa error atau halusinasi",
				"Satu domain service digunakan bersama oleh UI dashboard dan AI agent",
				"Mendukung transport lokal cepat (stdio) maupun jaringan jarak jauh (HTTP SSE)",
				"Standar terbuka industri yang didukung Antigravity, Anthropic, dan Google",
			},
			Cons: []string{
				"Perlu mendefinisikan JSON Schema input yang detail untuk setiap Tool",
				"Memerlukan pemahaman protokol JSON-RPC 2.0",
			},
			WhenToUse: []string{
				"Platform Vibecoding dan alat pengembang (seperti Vibeasik)",
				"Backend internal yang ingin diautomasi oleh agent autonomous",
				"Sistem diagnostik, audit kode, dan orkestrasi cloud devops",
			},
			FolderBlueprintTree: `backend/
├── cmd/
│   ├── serve.go              # Menjalankan HTTP server & MCP SSE
│   └── mcp.go                # Menjalankan MCP stdio untuk agent lokal
├── internal/
│   ├── api/                  # Inbound Port 1: REST API (Browser UI)
│   │   ├── server.go
│   │   └── project_handler.go
│   ├── mcp/                  # Inbound Port 2: MCP Server (AI Agents)
│   │   ├── server.go         # JSON-RPC 2.0 Dispatcher
│   │   ├── sse.go            # Server-Sent Events Transport
│   │   ├── types.go          # Tool & Resource Schemas
│   │   └── tools.go          # Tool declarations & handlers
│   ├── service/              # Shared Core Domain Logic
│   │   └── project_service.go
│   ├── repository/           # Data Persistence
│   │   └── project_repo.go
│   └── models/               # Domain Models
│       └── project.go`,
			FolderExplanation: []models.FolderAnnotation{
				{Path: "internal/mcp/", Role: "MCP Protocol Server", Description: "Menangani JSON-RPC 2.0, validasi schema tool, dan pengiriman respon SSE/stdio ke AI agent."},
				{Path: "internal/api/", Role: "REST API Transport", Description: "Menangani request standar dari UI Nuxt / web browser."},
				{Path: "internal/service/", Role: "Shared Domain Service", Description: "Logika bisnis yang dieksekusi baik oleh tombol di web maupun oleh perintah AI agent."},
			},
			RouterCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "MCP Server Protocol Registration",
				Code: `// internal/mcp/server.go
func (s *Server) RegisterTools() {
    s.tools["scan_directory"] = Tool{
        Name: "scan_directory",
        Description: "Memindai arsitektur folder proyek lokal",
        InputSchema: ToolSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "path": map[string]string{"type": "string"},
            },
            Required: []string{"path"},
        },
    }
}`,
				Explanation: "Mendefinisikan kemampuan sistem secara tipe-aman untuk agen AI.",
			},
			ControllerCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "Dual Execution: HTTP & MCP Calling Same Service",
				Code: `// internal/mcp/server.go (Tool Executor)
func (s *Server) ExecuteTool(name string, args map[string]interface{}) (ToolCallResult, error) {
    if name == "list_projects" {
        projects, err := s.services.Projects.GetAll(context.Background(), "", "")
        if err != nil { return ToolCallResult{IsError: true}, err }
        jsonBytes, _ := json.Marshal(projects)
        return ToolCallResult{
            Content: []ToolContent{{Type: "text", Text: string(jsonBytes)}},
        }, nil
    }
    return ToolCallResult{}, fmt.Errorf("tool not found")
}`,
				Explanation: "MCP Tool memanggil service yang sama dengan HTTP handler.",
			},
			LogicCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "Shared Service Logic",
				Code: `// internal/service/project_service.go
func (s *projectService) GetAll(ctx context.Context, pType, status string) ([]models.Project, error) {
    // Dipanggil oleh:
    // 1. GET /api/projects (Nuxt Frontend)
    // 2. tool vibeasik_list_projects (Antigravity / Cursor)
    return s.projectRepo.FindAll(ctx, pType, status)
}`,
				Explanation: "Logika tidak diduplikasi, cukup diakses via dua adapter berbeda.",
			},
		},
		{
			ID:                        "mvpp",
			Name:                      "MVPP (Model - View - Presenter - Planner)",
			Badge:                     "AI Cognitive Era",
			ShortDesc:                 "Memisahkan orkestrasi kontrol deterministik (Presenter) dari layer penalaran probabilistik AI (Planner).",
			Philosophy:                "Di era modern, backend tidak hanya memproses query SQL, tetapi juga memanggil LLM atau model kognitif (System One). MVPP memisahkan logika deterministik bisnis dari layer Planner kognitif agar sistem tetap stabil dan teruji.",
			RouterCharacteristics:     "Router meneruskan request ke Presenter. Presenter mengarahkan flow kapan harus memanggil Service deterministik vs kapan harus berkonsultasi ke Planner AI.",
			ControllerCharacteristics: "Presenter (Controller) bertindak sebagai konduktor orkestra yang menggabungkan output dari Model bisnis dan hasil penalaran dari Planner.",
			FunctionVsClassPhilosophy: "Planner mengisolasi primitif System One (choice, noul, score) atau LLM prompt ke dalam class/struct reasoning terdedikasi.",
			McpIntegration:            "Planner dapat menggunakan MCP Tools eksternal untuk memperkaya konteks sebelum memberikan rekomendasi.",
			MvppRole:                  "Arsitektur referensi utama untuk sistem bertenaga AI.",
			ComplexityScore:           4,
			TestabilityScore:          5,
			AiReadinessScore:          5,
			ScalabilityScore:          5,
			Pros: []string{
				"Logika probabilistik AI tidak mengotori atau membingungkan kode bisnis inti",
				"Jika server AI/LLM down atau lambat, Presenter dapat mengaktifkan fallback otomatis",
				"Planner dapat diuji dengan evaluasi akurasi, sementara Service diuji dengan unit test biasa",
				"Mendukung primitif System One / Jev (choice, noul, score) secara modular",
			},
			Cons: []string{
				"Memerlukan pemahaman pemisahan logika deterministik vs probabilistik",
				"Menambah satu layer abstraksi di atas Presenter/Controller",
			},
			WhenToUse: []string{
				"Aplikasi cerdas yang memiliki fitur AI rekomendasi, audit kode, atau auto-fix",
				"Sistem dengan arsitektur System One / Jev atau integrasi LLM",
				"Aplikasi yang memerlukan AI autonomous tanpa mengorbankan stabilitas data",
			},
			FolderBlueprintTree: `backend/
├── cmd/main.go
├── internal/
│   ├── models/               # [M] MODEL: Persistent Entities & State
│   │   ├── project.go
│   │   └── audit_finding.go
│   ├── views/                # [V] VIEW / SERIALIZER: DTO Output Formatter
│   │   ├── project_view.go
│   │   └── mcp_view.go
│   ├── presenters/           # [P] PRESENTER: Controller & Flow Orchestrator
│   │   ├── project_presenter.go
│   │   └── audit_presenter.go
│   ├── planners/             # [P] PLANNER: AI Cognitive & Reasoning Engine
│   │   ├── systemone_client.go # Primitives: choice, noul, score
│   │   ├── prompt_templates.go # Structured AI prompts
│   │   └── scaleup_planner.go  # Autonomous reasoning workflow
│   ├── services/             # Deterministic Business Operations
│   │   └── project_service.go
│   └── repository/           # Data Storage Layer
│       └── project_repo.go`,
			FolderExplanation: []models.FolderAnnotation{
				{Path: "internal/models/", Role: "Model (M)", Description: "Entitas data bisnis murni dan skema persistensi basis data."},
				{Path: "internal/views/", Role: "View (V)", Description: "Serializer dan formatting response: JSON API envelope, SSE, atau Markdown."},
				{Path: "internal/presenters/", Role: "Presenter (P)", Description: "Mengatur alur kontrol, otentikasi, HTTP status, dan koordinasi Service + Planner."},
				{Path: "internal/planners/", Role: "Planner (P)", Description: "Lapisan kognitif: Menjalankan primitif AI System One, heuristik cerdas, dan prompt LLM."},
			},
			RouterCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "MVPP Router Setup",
				Code: `// internal/api/router.go
func SetupMVPPRoutes(app *fiber.App, p *presenters.ProjectPresenter) {
    api := app.Group("/api/projects")
    api.Post("/:id/auto-scale", p.HandleAutoScaleUp)
}`,
				Explanation: "Router hanya memanggil Presenter yang mengoordinasikan Planner dan Service.",
			},
			ControllerCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "Presenter Orchestrating Service + Planner",
				Code: `// internal/presenters/project_presenter.go
func (p *ProjectPresenter) HandleAutoScaleUp(c *fiber.Ctx) error {
    id, _ := c.ParamsInt("id")
    
    // 1. Ambil data deterministik via Service
    project, err := p.services.GetByID(c.Context(), uint(id))
    if err != nil { return c.Status(404).JSON(fiber.Map{"error": "Not found"}) }
    
    // 2. Minta penalaran cerdas ke layer Planner (AI)
    plan, err := p.planners.ScaleUp.Evaluate(c.Context(), project)
    if err != nil {
        // Fallback deterministik jika Planner gagal
        plan = p.planners.ScaleUp.DefaultHeuristicPlan(project)
    }
    
    // 3. Serialisasi respon via View
    return c.JSON(views.FormatScaleUpPlan(project, plan))
}`,
				Explanation: "Presenter memadukan Service deterministik dan Planner AI secara aman.",
			},
			LogicCodeSample: models.CodeSnippet{
				Language: "go",
				Title:    "Planner Layer with System One Primitives",
				Code: `// internal/planners/scaleup_planner.go
func (pl *ScaleUpPlanner) ShouldMigrateToClean(ctx context.Context, metrics CodeMetrics) (bool, error) {
    // Menggunakan primitif System One 'noul' (keputusan biner probabilistik)
    payload := map[string]interface{}{
        "files_count": metrics.FileCount,
        "coupling_ratio": metrics.Coupling,
        "question": "Apakah codebase ini sudah membutuhkan arsitektur Clean/Hexagonal?",
    }
    return pl.systemOne.Noul(ctx, payload)
}`,
				Explanation: "Logika kognitif terisolasi sepenuhnya di dalam Planner.",
			},
		},
	}
}

func (s *scaleUpService) EvaluateComponent(ctx context.Context, componentID uint) (*models.ArchitectureEvaluation, error) {
	comp, err := s.componentRepo.FindByID(ctx, componentID)
	if err != nil {
		return nil, fmt.Errorf("komponen tidak ditemukan: %w", err)
	}

	eval := &models.ArchitectureEvaluation{
		ComponentID:            comp.ID,
		ComponentName:          comp.Name,
		Role:                   comp.Role,
		Language:               comp.Language,
		Framework:              comp.Framework,
		AbsolutePath:           comp.AbsolutePath,
		DetectedTraits:         models.EvaluationTraits{},
		FolderSummary:          make([]string, 0),
		KeyFindings:            make([]string, 0),
		ScaleUpRecommendations: make([]models.ScaleUpAction, 0),
	}

	targetDir := comp.AbsolutePath
	if targetDir == "" || targetDir == "-" {
		setEarlyEvaluation(eval, "unknown", "Tidak Diketahui (Path Kosong)", 20)
		return eval, nil
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		setEarlyEvaluation(eval, "unreachable", "Direktori Tidak Dapat Dibaca", 10)
		return eval, nil
	}

	folderSet := collectFolderSet(targetDir, entries, eval)
	stats := scanSourceStats(targetDir)
	traits := newArchTraits(folderSet, stats)
	applyArchTraits(eval, traits, stats)

	scores := computeArchScores(traits, stats.fatFiles)
	eval.Metrics.RouterScore = scores.router
	eval.Metrics.ControllerScore = scores.controller
	eval.Metrics.ServiceScore = scores.service
	eval.Metrics.AiMcpScore = scores.mcp
	eval.MaturityScore = scores.router + scores.controller + scores.service + scores.mcp

	classifyArchPattern(eval, traits, folderSet)
	eval.ScoreLevel = archScoreLevel(eval.MaturityScore)
	eval.KeyFindings = append(eval.KeyFindings, archKeyFindings(traits)...)
	eval.ScaleUpRecommendations = append(eval.ScaleUpRecommendations, archRecommendations(traits, stats.fatFiles)...)

	return eval, nil
}

func setEarlyEvaluation(eval *models.ArchitectureEvaluation, pattern, name string, score int) {
	eval.DetectedPattern = pattern
	eval.DetectedPatternName = name
	eval.MaturityScore = score
	eval.ScoreLevel = "Initial"
}

// collectFolderSet mencatat folder tingkat atas (dan internal/*) ke folderSet
// sekaligus mengisi eval.FolderSummary.
func collectFolderSet(targetDir string, entries []os.DirEntry, eval *models.ArchitectureEvaluation) map[string]bool {
	folderSet := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() {
			folderSet[strings.ToLower(e.Name())] = true
			eval.FolderSummary = append(eval.FolderSummary, e.Name())
		}
	}
	internalDir := filepath.Join(targetDir, "internal")
	if fi, err := os.Stat(internalDir); err == nil && fi.IsDir() {
		intEntries, _ := os.ReadDir(internalDir)
		for _, ie := range intEntries {
			if ie.IsDir() {
				folderSet[strings.ToLower(ie.Name())] = true
				eval.FolderSummary = append(eval.FolderSummary, "internal/"+ie.Name())
			}
		}
	}
	return folderSet
}

type sourceStats struct {
	totalFiles        int
	fatFiles          int
	hasModularRouting bool
}

var scanSkipDirs = map[string]bool{".git": true, "node_modules": true, "vendor": true, ".nuxt": true, "dist": true, "bin": true}
var scanSourceExts = map[string]bool{".go": true, ".ts": true, ".js": true, ".py": true, ".php": true}

func scanSourceStats(targetDir string) sourceStats {
	var st sourceStats
	_ = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			if scanSkipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !scanSourceExts[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		st.totalFiles++
		if countLines(path) > 300 {
			st.fatFiles++
		}
		lowerPath := strings.ToLower(path)
		if util.ContainsAny(lowerPath, "route", "api", "server") {
			st.hasModularRouting = true
		}
		return nil
	})
	return st
}

func anyFolder(folderSet map[string]bool, names ...string) bool {
	for _, n := range names {
		if folderSet[n] {
			return true
		}
	}
	return false
}

type archTraits struct {
	routing, service, repo, mcp, planner bool
}

func newArchTraits(folderSet map[string]bool, st sourceStats) archTraits {
	return archTraits{
		routing: st.hasModularRouting,
		service: anyFolder(folderSet, "service", "services", "usecase", "usecases"),
		repo:    anyFolder(folderSet, "repository", "repositories", "repo", "db"),
		mcp:     anyFolder(folderSet, "mcp", "agents"),
		planner: anyFolder(folderSet, "planners", "ai", "cognitive"),
	}
}

func applyArchTraits(eval *models.ArchitectureEvaluation, t archTraits, st sourceStats) {
	eval.DetectedTraits.TotalSourceFiles = st.totalFiles
	eval.DetectedTraits.HasModularRouting = t.routing
	eval.DetectedTraits.HasServiceLayer = t.service
	eval.DetectedTraits.HasRepositoryLayer = t.repo
	eval.DetectedTraits.HasMcpSupport = t.mcp
	eval.DetectedTraits.HasPlannerLayer = t.planner
	eval.DetectedTraits.HasThinControllers = st.fatFiles <= 2
	eval.DetectedTraits.HasPureFunctions = true
}

type archScores struct{ router, controller, service, mcp int }

func pickScore(def int, cases ...[2]int) int {
	// cases: pasangan {kondisi(0/1), nilai}; kondisi pertama yang benar menang.
	for _, c := range cases {
		if c[0] == 1 {
			return c[1]
		}
	}
	return def
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func computeArchScores(t archTraits, fat int) archScores {
	return archScores{
		router:     pickScore(15, [2]int{b2i(t.routing), 24}),
		controller: pickScore(22, [2]int{b2i(fat > 3), 12}, [2]int{b2i(fat > 0), 18}),
		service:    pickScore(10, [2]int{b2i(t.service && t.repo), 25}, [2]int{b2i(t.service || t.repo), 18}),
		mcp: pickScore(5, [2]int{b2i(t.mcp && t.planner), 25}, [2]int{b2i(t.mcp), 20},
			[2]int{b2i(t.planner), 15}),
	}
}

func classifyArchPattern(eval *models.ArchitectureEvaluation, t archTraits, folderSet map[string]bool) {
	switch {
	case t.mcp && t.service && t.repo:
		eval.DetectedPattern, eval.DetectedPatternName, eval.RecommendedNextPattern = "mcp_native", "MCP-Native Architecture", "mvpp"
	case folderSet["core"] && folderSet["adapters"]:
		eval.DetectedPattern, eval.DetectedPatternName, eval.RecommendedNextPattern = "clean_hexagonal", "Clean / Hexagonal Architecture", "mcp_native"
	case t.service && t.repo:
		eval.DetectedPattern, eval.DetectedPatternName, eval.RecommendedNextPattern = "layered", "Traditional Layered (N-Tier)", "mcp_native"
	case folderSet["features"] || folderSet["slices"]:
		eval.DetectedPattern, eval.DetectedPatternName, eval.RecommendedNextPattern = "vertical_slice", "Vertical Slice Architecture", "mcp_native"
	default:
		eval.DetectedPattern, eval.DetectedPatternName, eval.RecommendedNextPattern = "monolithic_initial", "Arsitektur Awal / Monolitik Sederhana", "layered"
	}
}

func archScoreLevel(total int) string {
	switch {
	case total >= 85:
		return "Production & Enterprise Ready"
	case total >= 70:
		return "Solid Architecture (Intermediate)"
	case total >= 50:
		return "Modularizing (Need Refactor)"
	}
	return "Initial Stage (High Tech Debt)"
}

func pickText(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}

func archKeyFindings(t archTraits) []string {
	return []string{
		pickText(t.routing, "Router sudah terorganisir modular dan terpisah dari main entrypoint.", "Router masih banyak terkonsentrasi di satu file utama."),
		pickText(t.service, "Logika bisnis telah diekstraksi ke Service Layer independen.", "Belum terdeteksi Service Layer mandiri; handler mungkin masih menangani logika bisnis langsung."),
		pickText(t.repo, "Kueri database telah diisolasi di Repository Layer terpisah.", "Kueri basis data belum dienkapsulasi ke dalam antarmuka Repository terpisah."),
		pickText(t.mcp, "Backend sudah siap terhubung dengan AI Coding Agent via Model Context Protocol (MCP).", "Belum memiliki adapter MCP; AI coding agent harus mengandalkan pembacaan file manual."),
	}
}

func archRecommendations(t archTraits, fat int) []models.ScaleUpAction {
	var recs []models.ScaleUpAction
	add := func(cond bool, a models.ScaleUpAction) {
		if cond {
			recs = append(recs, a)
		}
	}
	add(!t.service, models.ScaleUpAction{
		Phase:       "Tahap 1",
		Title:       "Ekstraksi Business Logic ke Service Layer",
		Description: "Pindahkan kalkulasi dan validasi dari handler ke folder internal/service.",
		TargetDir:   "internal/service",
		Priority:    "urgent",
	})
	add(!t.repo, models.ScaleUpAction{
		Phase:       "Tahap 2",
		Title:       "Abstraksi Data Access ke Repository Layer",
		Description: "Bungkus kueri GORM/SQL ke dalam interface repository agar mudah diuji dengan mock.",
		TargetDir:   "internal/repository",
		Priority:    "high",
	})
	add(fat > 0, models.ScaleUpAction{
		Phase:       "Tahap 3",
		Title:       "Refactor Thin Controller",
		Description: fmt.Sprintf("Ditemukan %d file sumber dengan ukuran > 300 baris. Pecah controller menjadi sub-handler per use-case.", fat),
		TargetDir:   "internal/api",
		Priority:    "medium",
	})
	add(!t.mcp, models.ScaleUpAction{
		Phase:       "Tahap 4",
		Title:       "Tambahkan Model Context Protocol (MCP) Adapter",
		Description: "Buat internal/mcp/server.go untuk mengekspos domain service sebagai MCP Tools untuk Antigravity/Cursor.",
		TargetDir:   "internal/mcp",
		Priority:    "medium",
	})
	add(!t.planner, models.ScaleUpAction{
		Phase:       "Tahap 5",
		Title:       "Adopsi Paradigma MVPP (AI Cognitive Layer)",
		Description: "Pisahkan prompt dan integrasi LLM/System One ke dalam layer Planner terisolasi.",
		TargetDir:   "internal/planners",
		Priority:    "low",
	})
	return recs
}

func (s *scaleUpService) EvaluateProject(ctx context.Context, projectID uint) ([]models.ArchitectureEvaluation, error) {
	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("proyek tidak ditemukan: %w", err)
	}

	var results []models.ArchitectureEvaluation
	for _, comp := range proj.Components {
		// Only evaluate backend or cli or other code-containing components
		if comp.Role == models.RoleBackend || comp.Role == models.RoleCLI || comp.Role == models.RoleWorker || comp.Language == "go" || comp.Language == "typescript" {
			eval, err := s.EvaluateComponent(ctx, comp.ID)
			if err == nil && eval != nil {
				results = append(results, *eval)
			}
		}
	}

	// Fallback if no backend component explicitly tagged
	if len(results) == 0 && len(proj.Components) > 0 {
		eval, err := s.EvaluateComponent(ctx, proj.Components[0].ID)
		if err == nil && eval != nil {
			results = append(results, *eval)
		}
	}

	return results, nil
}

func countLines(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines
}

var _ = regexp.MustCompile
