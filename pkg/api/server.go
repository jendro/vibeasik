package api

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/activity"
	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/mcp"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/repository"
	"github.com/vibeasik/vibeasik/pkg/runner"
	"github.com/vibeasik/vibeasik/pkg/service"
	"github.com/vibeasik/vibeasik/pkg/ui"
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
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))
	app.Use(securityHeaders)
	app.Use(cors.New(cors.Config{
		AllowOrigins:  corsOrigins(),
		AllowMethods:  "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:  "Origin, Content-Type, Accept, Authorization, " + headerMCPSession + ", " + headerExecToken + ", " + localauth.Header,
		ExposeHeaders: headerMCPSession,
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

	// Token & Host diperiksa sebelum route mana pun (termasuk Studio statis).
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

func (s *Server) setupRoutes() {
	// Health
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "time": time.Now()})
	})

	// MCP Protocol Endpoints
	s.app.Post("/mcp", s.handleMCPPost)
	s.app.Delete("/mcp", s.handleMCPDelete)
	s.app.Get("/mcp/sse", s.handleMCPSSE)

	// API Group
	api := s.app.Group("/api")

	// MCP Info & Config
	api.Get("/mcp/info", s.getMCPInfo)

	// Projects
	api.Get("/projects", s.getProjects)
	api.Post("/projects", s.createProject)
	api.Get("/projects/identify", s.identifyProject)
	api.Get("/projects/similarity-scan", s.scanSimilarities)
	api.Post("/projects/similarity-scan", s.scanSimilarities)
	api.Get("/projects/:idA/compare/:idB", s.compareProjects)
	api.Post("/projects/:id/deprecate", s.deprecateProject)
	api.Post("/projects/:id/undeprecate", s.undeprecateProject)
	api.Post("/projects/:id/hold", s.holdProject)
	api.Post("/projects/:id/done", s.doneProject)
	api.Post("/projects/:id/activate", s.activateProject)
	api.Get("/projects/:id", s.getProject)
	api.Put("/projects/:id", s.updateProject)
	api.Delete("/projects/:id", s.deleteProject)
	api.Post("/projects/:id/merge", s.mergeProject)
	api.Post("/projects/:id/summary", s.generateProjectSummary)
	api.Get("/projects/:id/todos", s.getProjectTodos)
	api.Get("/projects/:id/code-map", s.getCodeMap)
	api.Post("/projects/:id/code-map/build", s.buildCodeMap)
	api.Get("/projects/:id/code-map/search", s.searchCodeMap)
	api.Get("/projects/:id/code-map/file", s.getCodeMapFile)
	api.Post("/projects/:id/code-map/export", s.exportCodeMap)
	api.Post("/projects/:id/link-workspace", s.linkProjectWorkspace)
	api.Post("/projects/:id/init-vibeasik", s.initProject)
	api.Get("/projects/:id/consolidate-preview", s.previewConsolidateProject)
	api.Post("/projects/:id/consolidate", s.consolidateProject)

	// Workspace Folder Organizer
	api.Post("/organizer/plan", s.generateReorgPlan)
	api.Post("/organizer/execute", s.executeReorgMoves)

	// Guidelines & Standards (Rubrics)
	api.Get("/guidelines", s.getGuidelines)

	// Todos & Task Tracking (3-Tier: Global, Project, Component)
	api.Get("/todos", s.getTodos)
	api.Get("/todos/impact-dimensions", s.getImpactDimensions)
	api.Post("/todos", s.createTodo)
	api.Get("/todos/:id", s.getTodo)
	api.Put("/todos/:id", s.updateTodo)
	api.Delete("/todos/:id", s.deleteTodo)
	api.Post("/todos/suggest", s.suggestTodos)
	api.Post("/todos/:id/approve", s.approveTodo)
	api.Post("/todos/:id/reject", s.rejectTodo)

	// Components
	api.Post("/projects/:id/components", s.addComponent)
	api.Get("/components/:id", s.getComponent)
	api.Put("/components/:id", s.updateComponent)
	api.Delete("/components/:id", s.deleteComponent)
	api.Post("/components/:id/create-dir", s.createComponentDir)

	// Scanner & Diagnostics
	api.Get("/adapters", s.listAdapters)
	api.Post("/scan", s.scanPath)
	api.Post("/scan/group-ai", s.groupScanComponents)
	api.Post("/projects/:id/diagnose", s.diagnoseProject)
	api.Get("/projects/:id/diagnostics/latest", s.getLatestProjectDiagnostic)
	api.Post("/projects/:id/diagnostics/delete-finding", s.deleteDiagnosticFinding)
	api.Delete("/projects/:id/diagnostics/findings", s.deleteDiagnosticFinding)
	api.Delete("/projects/:id/diagnostics/findings/:findingKey", s.deleteDiagnosticFinding)
	api.Post("/projects/:id/diagnostics/ignore-finding", s.ignoreDiagnosticFinding)
	api.Post("/projects/:id/diagnostics/ignore-finding/:findingKey", s.ignoreDiagnosticFinding)
	// Pusat Pengetahuan (satu pintu patokan & pengetahuan pekerja AI)
	api.Get("/knowledge", s.getKnowledgeOverview)
	api.Get("/knowledge/graph", s.getKnowledgeGraph)
	api.Get("/knowledge/nodes", s.listKnowledgeNodes)
	api.Post("/knowledge/nodes", s.teachKnowledge)
	api.Get("/knowledge/nodes/:id", s.getKnowledgeNode)
	api.Post("/knowledge/nodes/:id/feedback", s.knowledgeFeedback)
	api.Delete("/knowledge/nodes/:id", s.retireKnowledgeNode)
	api.Post("/knowledge/recall", s.recallKnowledge)
	api.Post("/knowledge/curate", s.curateKnowledge)
	api.Get("/knowledge/gaps", s.listKnowledgeGaps)
	api.Post("/knowledge/gaps", s.reportKnowledgeGap)
	api.Post("/knowledge/gaps/:gid/dismiss", s.dismissKnowledgeGap)

	api.Post("/components/:id/diagnose", s.diagnoseComponent)
	api.Get("/components/:id/diagnostics/latest", s.getLatestComponentDiagnostic)
	api.Post("/components/:id/fast-scan", s.fastScanComponent)
	api.Put("/components/:id/fast-scan", s.updateComponentFastScan)
	api.Post("/components/:id/apply-scan-params", s.applyFastScanParams)
	api.Post("/projects/:id/fast-scan", s.fastScanProject)

	// Scale-Up Architecture & Pattern Evaluator
	api.Get("/scaleup/patterns", s.getScaleUpPatterns)
	api.Get("/scaleup/evaluate/:componentId", s.evaluateComponentArchitecture)
	api.Get("/scaleup/project-evaluate/:projectId", s.evaluateProjectArchitecture)

	// Component Routes & Endpoints Management
	api.Get("/components/:id/routes", s.getComponentRoutes)
	api.Post("/components/:id/routes/sync", s.syncComponentRoutes)
	api.Put("/components/:id/routes/:routeId", s.updateComponentRoute)
	api.Delete("/components/:id/routes/:routeId", s.deleteComponentRoute)

	// Security & Censoring
	api.Get("/security/status", s.getSecurityStatus)
	api.Get("/security/censored-keywords", s.getCensoredKeywords)
	api.Post("/security/censored-keywords", s.createCensoredKeyword)
	api.Put("/security/censored-keywords/:id", s.updateCensoredKeyword)
	api.Delete("/security/censored-keywords/:id", s.deleteCensoredKeyword)
	api.Post("/security/test-censor", s.testCensorPayload)

	// System Port Checker
	api.Get("/system/ports", s.checkPorts)

	// Port Registry & Dev Runner (vibeasik mengalokasikan port unik lalu menjalankan dev command).
	// Route yang mengeksekusi perintah dijaga requireExecAccess (lihat exec_guard.go).
	api.Get("/ports/registry", s.getPortRegistry)
	api.Get("/runs", s.listRuns)
	api.Post("/components/:id/port", s.assignComponentPort)
	api.Get("/components/:id/run", s.getComponentRunStatus)
	api.Post("/components/:id/run", requireExecAccess, s.runComponent)
	api.Post("/components/:id/stop", s.stopComponent)
	api.Post("/projects/:id/run", requireExecAccess, s.runProject)
	api.Post("/projects/:id/stop", s.stopProject)
	api.Post("/projects/:id/ensure-unique-ports", s.ensureUniquePorts)
	api.Post("/ports/ensure-unique", s.ensureUniquePorts)

	// System Directory Browser & Search
	api.Get("/system/directories", s.browseDirectories)

	// LLM Settings & Configurations Grid
	api.Get("/settings/llm", s.getLLMSettings)
	api.Put("/settings/llm", s.updateLLMSettings)
	api.Post("/settings/llm/active-provider", s.setActiveProvider)
	api.Post("/settings/llm/select-model", s.selectActiveModel)
	api.Get("/settings/llm/configs", s.getLLMConfigs)
	api.Post("/settings/llm/configs", s.createLLMConfig)
	api.Get("/settings/llm/configs/:id", s.getLLMConfig)
	api.Put("/settings/llm/configs/:id", s.updateLLMConfig)
	api.Delete("/settings/llm/configs/:id", s.deleteLLMConfig)
	api.Post("/settings/llm/configs/:id/activate", s.activateLLMConfig)
	api.Put("/settings/llm/configs/:id/vision", s.setLLMModelVision)
	api.Put("/settings/llm/configs/:id/alias", s.setLLMModelAlias)
	api.Get("/settings/workspace", s.getWorkspaceSettings)
	api.Put("/settings/workspace", s.updateWorkspaceSettings)

	// vibeasik.com Delivery, Academy, Cloud Security & AI Agent Hub
	api.Get("/vibeasik/tutorials", s.getVibeTutorials)
	api.Get("/vibeasik/domains/search", s.searchVibeDomains)
	api.Post("/vibeasik/deploy/generate-plan", s.generateVibeDeployPlan)
	api.Post("/vibeasik/security/cloud-audit", s.runCloudSecurityAudit)
	api.Get("/vibeasik/ai-templates", s.getVibeAITemplates)
	api.Post("/vibeasik/ai-templates/inject", s.injectVibeAITemplate)

	// Route aliases directly on app without /api prefix
	s.app.Get("/vibeasik/tutorials", s.getVibeTutorials)
	s.app.Get("/vibeasik/domains/search", s.searchVibeDomains)
	s.app.Post("/vibeasik/deploy/generate-plan", s.generateVibeDeployPlan)
	s.app.Post("/vibeasik/security/cloud-audit", s.runCloudSecurityAudit)
	s.app.Get("/vibeasik/ai-templates", s.getVibeAITemplates)
	s.app.Post("/vibeasik/ai-templates/inject", s.injectVibeAITemplate)

	// Plug-and-Play Local Terminal AI Agents (agy, claude, aider, custom)
	api.Get("/agents", s.listLocalAgents)
	api.Post("/agents/execute", requireExecAccess, s.executeLocalAgent)
	api.Post("/agents/custom", requireExecAccess, s.createCustomAgent)
	api.Delete("/agents/custom/:key", s.deleteCustomAgent)

	// Traffic Analyzer: statistik request aplikasi live → saran rate limit, cache, LB, pecah service (Aturan 48)
	api.Get("/traffic/meta", s.getTrafficMeta)
	api.Post("/traffic/ingest", s.ingestTrafficDirect)
	api.Get("/traffic/sources", s.listTrafficSources)
	api.Post("/traffic/sources", s.createTrafficSource)
	api.Put("/traffic/sources/:id", s.updateTrafficSource)
	api.Delete("/traffic/sources/:id", s.deleteTrafficSource)
	api.Post("/traffic/sources/:id/sync", s.syncTrafficSource)
	api.Post("/traffic/sources/:id/rotate-key", s.rotateTrafficKey)
	api.Get("/traffic/sources/:id/report", s.getTrafficReport)
	api.Get("/traffic/sources/:id/recent", s.getTrafficRecent)
	api.Get("/traffic/sources/:id/sdk", s.getTrafficSDK)
	api.Post("/traffic/sources/:id/sdk/install", s.installTrafficSDK)
	api.Post("/traffic/sources/:id/todo", s.trafficRecommendationTodo)

	// REA Reverse Engineering Toolkit
	api.Get("/rea/status", s.getReaStatus)
	api.Post("/rea/analyze", s.runReaAnalysis)

	// Route modul (Pro & tambahan) dipasang lewat registry, tiap route dibungkus gate fitur modulnya.
	for _, m := range registry.All() {
		if rp, ok := m.(registry.RouteProvider); ok {
			rp.Routes(registry.NewRouter(api, m.Feature()), s.db)
		}
	}
	// Route fitur berbayar yang modulnya tidak ada di build ini → 402 upgrade_required (bukan 404).
	// Bila modulnya terpasang, route modul di atas terdaftar lebih dulu sehingga ini tidak pernah tercapai.
	for _, r := range registry.ProCatalog {
		feature := r.Feature
		api.Add(r.Method, r.Path, func(c *fiber.Ctx) error { return registry.RespondUpgrade(c, feature) })
	}

	// Edisi & fitur aktif untuk Studio (gembok menu Pro).
	api.Get("/edition", func(c *fiber.Ctx) error {
		features := registry.Features()
		installed := map[string]bool{}
		for f := range features {
			installed[f] = registry.Installed(f)
		}
		return c.JSON(fiber.Map{"edition": registry.Edition, "features": features, "installed": installed, "upgrade_url": registry.UpgradeURL})
	})

	// Mount Downloads directory if available on disk
	downloadDirs := []string{
		"../studio-frontend/public/downloads",
		"studio-frontend/public/downloads",
		"public/downloads",
	}
	for _, dir := range downloadDirs {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			s.app.Static("/downloads", dir, fiber.Static{
				Download: true,
				MaxAge:   3600,
			})
			break
		}
	}

	// Mount Static Studio Frontend UI (Embedded or Local)
	if httpFS, ok := ui.GetFileSystem(); ok {
		// Aset ber-hash (/_nuxt/) boleh di-cache permanen; HTML & lainnya selalu divalidasi ulang agar
		// Studio baru langsung terpakai setelah engine diperbarui (HTML lama menunjuk aset yang sudah tiada).
		s.app.Use("/", func(c *fiber.Ctx) error {
			// Aset ber-hash yang tidak ada: 404, jangan 200.html (bisa ter-cache sebagai JS).
			if strings.HasPrefix(c.Path(), "/_nuxt/") {
				f, openErr := httpFS.Open(c.Path())
				if openErr != nil {
					c.Set(fiber.HeaderCacheControl, "no-store")
					return c.SendStatus(fiber.StatusNotFound)
				}
				_ = f.Close()
			}
			err := c.Next()
			if strings.HasPrefix(c.Path(), "/_nuxt/") && c.Response().StatusCode() == fiber.StatusOK {
				c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
			} else {
				c.Set(fiber.HeaderCacheControl, "no-cache")
			}
			return err
		})
		s.app.Use("/", filesystem.New(filesystem.Config{
			Root:         httpFS,
			Index:        "index.html",
			NotFoundFile: "200.html",
			Next: func(c *fiber.Ctx) bool {
				path := c.Path()
				return strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/health") || strings.HasPrefix(path, "/downloads")
			},
		}))
	}
}
