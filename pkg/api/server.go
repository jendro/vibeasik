package api

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	fiberRecover "github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/activity"
	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/mcp"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/repository"
	"github.com/vibeasik/vibeasik/pkg/runner"
	"github.com/vibeasik/vibeasik/pkg/service"
	"github.com/vibeasik/vibeasik/pkg/update"
)

// Server coordinates the Fiber REST API and MCP endpoints
type Server struct {
	token     string // token akses lokal (pkg/localauth)
	lan       bool   // engine dibuka ke LAN: Host IP privat diterima
	app       *fiber.App
	db        *gorm.DB
	repos     *repository.Repositories
	services  *service.Services
	mcpServer *mcp.Server
	runner    *runner.Service
	updater   *update.Updater
	restart   func() error // mulai ulang engine (dipasang serve); nil bila tidak didukung
	enqueueAssignment func(db *gorm.DB, todo *models.Todo) error // seam enqueue assignment (fallback ke registry.EnqueueTodoAssignment)
}

const (
	envCORSOrigins     = "VIBEASIK_CORS_ORIGINS"
	envFrontendOrigins = "FRONTEND_ORIGINS"
	envAllowedOrigins  = "ALLOWED_ORIGINS"
	defaultCORSOrigins = "http://localhost:8760,http://127.0.0.1:8760"
)

// validateOrigin memeriksa dan menormalisasi URL origin sesuai RFC 6454.
// Origin wildcard (*) dilarang untuk mencegah risiko keamanan kredensial dan CSRF (K36).
// Origin harus memiliki skema http/https dan host yang valid (K3158).
func validateOrigin(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "*" {
		return "", false
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	if u.Host == "" {
		return "", false
	}
	return fmt.Sprintf("%s://%s", scheme, strings.ToLower(u.Host)), true
}

// corsOrigins membaca allowlist origin dari environment variable (VIBEASIK_CORS_ORIGINS, FRONTEND_ORIGINS, ALLOWED_ORIGINS).
// Jika disetel, hanya origin yang tervalidasi yang diizinkan dan origin liar ditolak (K36, K4942).
// Jika tidak disetel, digunakan fallback aman untuk local dev Studio (defaultCORSOrigins).
func corsOrigins() string {
	for _, envKey := range []string{envCORSOrigins, envFrontendOrigins, envAllowedOrigins} {
		if raw := strings.TrimSpace(os.Getenv(envKey)); raw != "" {
			var valid []string
			seen := make(map[string]bool)
			for _, part := range strings.Split(raw, ",") {
				if origin, ok := validateOrigin(part); ok && !seen[origin] {
					seen[origin] = true
					valid = append(valid, origin)
				}
			}
			if len(valid) > 0 {
				return strings.Join(valid, ",")
			}
			// Env disetel tetapi tidak ada origin valid (misal hanya wildcard '*');
			// tolak semua cross-origin request dengan mengembalikan allowlist kosong.
			return ""
		}
	}
	return defaultCORSOrigins
}

// NewServer builds and configures the Fiber application with layered architecture
func NewServer(db *gorm.DB) *Server {
	app := fiber.New(fiber.Config{
		AppName:      "vibeasik",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		BodyLimit:    40 << 20, // unggahan screenshot Galeri Desain (≤8 MB per gambar, beberapa sekaligus)
	})

	// Middlewares
	app.Use(fiberRecover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))
	app.Use(securityHeaders)
	app.Use(cors.New(cors.Config{
		AllowOrigins:  corsOrigins(),
		AllowMethods:  "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:  "Origin, Content-Type, Accept, Authorization, Idempotency-Key, X-Idempotency-Key, " + headerMCPSession + ", " + headerExecToken + ", " + localauth.Header,
		ExposeHeaders: headerMCPSession + ", X-Todo-Status-Counts",
	}))

	repos := repository.NewRepositories(db)
	services := service.NewServices(repos, db)
	activity.Init(db)
	mcpSrv := mcp.NewServer(db)

	token, err := localauth.Token()
	if err != nil {
		log.Fatalf("token akses lokal tidak bisa dibuat di %s: %v", localauth.Path(), err)
	}

	s := &Server{
		token:     token,
		app:       app,
		db:        db,
		repos:     repos,
		services:  services,
		mcpServer: mcpSrv,
		runner:    runner.NewService(db),
	}

	// Perimeter Keamanan Global Fail-Closed (localAccess):
	// Dipasang sebelum rute mana pun (termasuk Studio statis dan seluruh sub-group API).
	// Mengamankan seluruh rute mutasi (POST/PUT/DELETE/PATCH) dan read tanpa perlu duplikasi middleware auth per-group:
	// 1. Host Validation (mitigasi DNS-Rebinding): Host wajib loopback (localhost, 127.0.0.1, ::1).
	//    Host non-loopback (mis. 192.168.x.x, 10.x.x.x, domain publik) langsung ditolak HTTP 403 Forbidden
	//    kecuali mode LAN sengaja diaktifkan via serve --host 0.0.0.0 (SetLAN).
	// 2. Local Token Authentication: Seluruh endpoint /api/* dan /mcp wajib membawa token lokal yang valid
	//    (header X-Vibeasik-Token, Bearer authorization, atau cookie sesi Studio), mengembalikan HTTP 401 bila tidak sah.
	// 3. CORS & CSRF Mitigation: Dikombinasikan dengan corsOrigins() allowlist ketat dan securityHeaders
	//    untuk mencegah cross-origin request dari website pihak ketiga di browser lokal.
	app.Use(s.localAccess)
	s.setupRoutes()
	return s
}

// Start begins listening on the requested address
func (s *Server) Start(addr string) error {
	return s.app.Listen(addr)
}

// Shutdown stops dev servers launched from vibeasik, then the HTTP listener.
func (s *Server) Shutdown() error {
	runner.Default().StopAll()
	return s.app.Shutdown()
}

// SetEnqueueAssignment menyetel fungsi kustom untuk enqueue assignment todo (untuk testing atau seam injeksi).
func (s *Server) SetEnqueueAssignment(fn func(db *gorm.DB, todo *models.Todo) error) {
	s.enqueueAssignment = fn
}

func (s *Server) enqueueTodoAssignment(tx *gorm.DB, todo *models.Todo) error {
	if s.enqueueAssignment != nil {
		return s.enqueueAssignment(tx, todo)
	}
	return registry.EnqueueTodoAssignment(tx, todo)
}

func (s *Server) safeEnqueueTodoAssignment(tx *gorm.DB, todo *models.Todo) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in assignment enqueue: %v", r)
		}
	}()
	return s.enqueueTodoAssignment(tx, todo)
}

