package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestFastScanComponent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_comp_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create fake component files including a secret in .env
	_ = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{
		"name": "test-fe",
		"scripts": {
			"dev": "bun run dev",
			"build": "bun run build"
		}
	}`), 0644)

	_ = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(`
		API_KEY=sk-proj-99998888777766665555444433332222
		DATABASE_URL=postgres://root:SuperSecret123@localhost:5432/test
	`), 0644)

	comp := &models.Component{
		Name:         "Frontend Web",
		Role:         models.RoleFrontendClient,
		AbsolutePath: tmpDir,
	}

	res, err := FastScanComponent(nil, comp, nil)
	if err != nil {
		t.Fatalf("FastScanComponent failed: %v", err)
	}

	if res.DirectoryTree == "" {
		t.Errorf("Expected directory tree, got empty")
	}

	if res.HowToRun.DevCommand == "" {
		t.Errorf("Expected dev command to be detected")
	}

	// Verify that secret was sanitized in audit
	redactedCount, _ := res.SecurityAudit["redacted_count"].(int)
	if redactedCount < 1 {
		t.Errorf("Expected at least 1 redacted secret in audit, got %d", redactedCount)
	}

	// Verify raw key never appears in AI analysis or preview
	if strings.Contains(res.AiAnalysisMarkdown, "sk-proj-99998888") {
		t.Errorf("Leaked raw key in AI analysis markdown!")
	}

	// Verify extracted environment metadata
	if len(res.ExtractedEnv.EnvironmentKeys) < 2 {
		t.Errorf("Expected at least 2 environment keys, got %d", len(res.ExtractedEnv.EnvironmentKeys))
	}

	for _, ek := range res.ExtractedEnv.EnvironmentKeys {
		if strings.Contains(ek.SafeExample, "sk-proj-99998888") {
			t.Errorf("Leaked raw API key in SafeExample for %s!", ek.Key)
		}
		if strings.Contains(ek.SafeExample, "SuperSecret123") {
			t.Errorf("Leaked database password in SafeExample for %s!", ek.Key)
		}
	}

	if res.ExtractedEnv.DynamicMetadata == nil {
		t.Errorf("DynamicMetadata must not be nil")
	}
}

func TestExtractRoutesAndArchitecture_Gin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gin_proj_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`module myginapp

go 1.22

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/redis/go-redis/v9 v9.4.0
	gorm.io/driver/postgres v1.5.4
	gorm.io/gorm v1.25.5
)
`), 0644)

	mainCode := `package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/health", HealthCheck)

	api := r.Group("/api/v1")
	api.POST("/auth/login", HandleLogin)
	api.GET("/users", GetUsers)
	api.GET("/users/:id", GetUserByID)
	api.DELETE("/users/:id", DeleteUser)

	r.Run(":8080")
}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainCode), 0644)
	_ = os.Mkdir(filepath.Join(tmpDir, "controllers"), 0755)
	_ = os.Mkdir(filepath.Join(tmpDir, "models"), 0755)

	arch := detectArchitecture(tmpDir)
	if arch.Framework != "Gin" {
		t.Errorf("Expected framework Gin, got %s", arch.Framework)
	}
	if !containsString(arch.Plugins, "JWT Authentication (golang-jwt/jwt)") {
		t.Errorf("Expected JWT plugin, got %v", arch.Plugins)
	}
	if !containsString(arch.Plugins, "PostgreSQL Database Driver") {
		t.Errorf("Expected PostgreSQL driver, got %v", arch.Plugins)
	}
	if arch.DesignPattern != "MVC (Model-View-Controller / Model-Controller Pattern)" {
		t.Errorf("Expected MVC design pattern, got %s", arch.DesignPattern)
	}

	routes := extractRoutes(tmpDir)
	if len(routes) < 5 {
		t.Errorf("Expected at least 5 routes, got %d", len(routes))
	}

	foundLogin := false
	for _, r := range routes {
		if r.Path == "/api/v1/auth/login" && r.Method == "POST" {
			foundLogin = true
		}
	}
	if !foundLogin {
		t.Errorf("Expected route /api/v1/auth/login, found: %+v", routes)
	}
}

func TestExtractRoutesAndArchitecture_Echo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "echo_proj_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`module myechoapp

go 1.22

require (
	github.com/labstack/echo/v4 v4.11.4
	github.com/go-playground/validator/v10 v10.18.0
	github.com/jackc/pgx/v5 v5.5.3
)
`), 0644)

	mainCode := `package main

import "github.com/labstack/echo/v4"

func main() {
	e := echo.New()
	e.GET("/ping", PingHandler)

	admin := e.Group("/admin")
	admin.GET("/metrics", GetMetrics)
	admin.POST("/settings", UpdateSettings)

	e.Start(":1323")
}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainCode), 0644)
	_ = os.Mkdir(filepath.Join(tmpDir, "handlers"), 0755)
	_ = os.Mkdir(filepath.Join(tmpDir, "services"), 0755)

	arch := detectArchitecture(tmpDir)
	if arch.Framework != "Echo v4" {
		t.Errorf("Expected Echo v4, got %s", arch.Framework)
	}
	if !containsString(arch.Plugins, "PostgreSQL Database Driver") {
		t.Errorf("Expected PostgreSQL Driver, got %v", arch.Plugins)
	}
	if arch.DesignPattern != "3-Tier Layered Architecture (Handler - Service - Repository)" {
		t.Errorf("Expected 3-Tier Layered Architecture, got %s", arch.DesignPattern)
	}

	routes := extractRoutes(tmpDir)
	if len(routes) < 3 {
		t.Errorf("Expected at least 3 routes, got %d", len(routes))
	}
}

func TestExtractRoutesAndArchitecture_Chi(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chi_proj_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`module mychiapp

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.12
	go.uber.org/zap v1.27.0
)
`), 0644)

	mainCode := `package main

import "github.com/go-chi/chi/v5"

func main() {
	r := chi.NewRouter()
	r.Get("/health", HealthHandler)
	r.Route("/items", func(r chi.Router) {
		r.Get("/", ListItems)
		r.Post("/", CreateItem)
	})
}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainCode), 0644)
	_ = os.Mkdir(filepath.Join(tmpDir, "domain"), 0755)
	_ = os.Mkdir(filepath.Join(tmpDir, "usecase"), 0755)
	_ = os.Mkdir(filepath.Join(tmpDir, "repository"), 0755)

	arch := detectArchitecture(tmpDir)
	if arch.Framework != "Chi Router" {
		t.Errorf("Expected Chi Router, got %s", arch.Framework)
	}
	if !containsString(arch.Plugins, "Uber Zap (Structured Fast Logger)") {
		t.Errorf("Expected Zap logger, got %v", arch.Plugins)
	}
	if !strings.Contains(arch.DesignPattern, "Clean Architecture") {
		t.Errorf("Expected Clean Architecture, got %s", arch.DesignPattern)
	}

	routes := extractRoutes(tmpDir)
	if len(routes) < 3 {
		t.Errorf("Expected at least 3 routes, got %d", len(routes))
	}
}

func TestExtractRoutesAndArchitecture_StandardNetHttp(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nethttp_proj_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`module mynethttpapp

go 1.22
`), 0644)

	mainCode := `package main

import "net/http"

func main() {
	http.HandleFunc("GET /api/v1/ping", PingHandler)
	http.HandleFunc("POST /api/v1/echo", EchoHandler)
	http.ListenAndServe(":8080", nil)
}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainCode), 0644)

	arch := detectArchitecture(tmpDir)
	if arch.Framework != "Standard net/http" {
		t.Errorf("Expected Standard net/http, got %s", arch.Framework)
	}

	routes := extractRoutes(tmpDir)
	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}
	if routes[0].Method != "GET" || routes[0].Path != "/api/v1/ping" {
		t.Errorf("Unexpected route 0: %+v", routes[0])
	}
	if routes[1].Method != "POST" || routes[1].Path != "/api/v1/echo" {
		t.Errorf("Unexpected route 1: %+v", routes[1])
	}
}

func TestExtractRoutesAndArchitecture_GorillaMux(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mux_proj_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`module mymuxapp

go 1.22

require (
	github.com/gorilla/mux v1.8.1
	github.com/redis/go-redis/v9 v9.4.0
)
`), 0644)

	mainCode := `package main

import "github.com/gorilla/mux"

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/users", GetUsers).Methods("GET")
	r.HandleFunc("/users", CreateUser).Methods("POST")
	r.HandleFunc("/users/{id}", UpdateUser).Methods("PUT", "PATCH")

	sub := r.PathPrefix("/admin").Subrouter()
	sub.HandleFunc("/dashboard", AdminDashboard).Methods("GET")
}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainCode), 0644)

	arch := detectArchitecture(tmpDir)
	if arch.Framework != "Gorilla Mux" {
		t.Errorf("Expected Gorilla Mux, got %s", arch.Framework)
	}
	if !containsString(arch.Plugins, "Redis Client (Cache & Session Storage)") {
		t.Errorf("Expected Redis Client in plugins, got %v", arch.Plugins)
	}

	routes := extractRoutes(tmpDir)
	if len(routes) < 5 {
		t.Errorf("Expected at least 5 routes, got %d (routes: %+v)", len(routes), routes)
	}
}

func TestFastScanFrontendNuxtApp(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_nuxt_fe_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create Nuxt package.json
	pkgJSON := `{
		"name": "my-nuxt-portal",
		"dependencies": {
			"@nuxt/ui": "^4.11.3",
			"@iconify-json/lucide": "^1.2.139",
			"nuxt": "^4.6.0",
			"tailwindcss": "^4.3.3",
			"pinia": "^2.1.7",
			"zod": "^3.22.4"
		},
		"devDependencies": {
			"typescript": "^5.4.0",
			"eslint": "^9.0.0"
		},
		"packageManager": "bun@1.1.0"
	}`
	_ = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(pkgJSON), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "nuxt.config.ts"), []byte(`export default defineNuxtConfig({})`), 0644)

	// Create pages hierarchy
	pagesDir := filepath.Join(tmpDir, "app", "pages")
	_ = os.MkdirAll(filepath.Join(pagesDir, "projects"), 0755)
	_ = os.WriteFile(filepath.Join(pagesDir, "index.vue"), []byte(`<template><div>Home</div></template>`), 0644)
	_ = os.WriteFile(filepath.Join(pagesDir, "projects", "[id].vue"), []byte(`<template><div>Project Detail</div></template>`), 0644)
	_ = os.WriteFile(filepath.Join(pagesDir, "settings.vue"), []byte(`<template><div>Settings</div></template>`), 0644)
	_ = os.WriteFile(filepath.Join(pagesDir, "ports.vue"), []byte(`<template><div>Ports</div></template>`), 0644)

	// Create composable
	compDir := filepath.Join(tmpDir, "app", "composables")
	_ = os.MkdirAll(compDir, 0755)
	_ = os.WriteFile(filepath.Join(compDir, "useApi.ts"), []byte(`export const useApi = () => {}`), 0644)

	// Create .env with secret
	_ = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(`
NUXT_PUBLIC_API_URL=http://localhost:8765
AUTH_CLIENT_SECRET=sk-proj-feSecretKey1234567890abcdef12345
`), 0644)

	comp := &models.Component{
		Name:         "Portal Nuxt Frontend",
		Role:         models.RoleFrontendClient,
		AbsolutePath: tmpDir,
	}

	res, err := FastScanComponent(nil, comp, nil)
	if err != nil {
		t.Fatalf("FastScanComponent failed on Nuxt frontend: %v", err)
	}

	// 1. Verify Architecture & Framework
	if !strings.Contains(res.Architecture.Framework, "Nuxt") {
		t.Errorf("Expected framework Nuxt, got: %s", res.Architecture.Framework)
	}
	if !strings.Contains(res.Architecture.DesignPattern, "Composition API") {
		t.Errorf("Expected Composition API design pattern, got: %s", res.Architecture.DesignPattern)
	}

	// 2. Verify Plugins
	foundNuxtUI := false
	foundTailwind := false
	foundPinia := false
	foundZod := false
	for _, p := range res.Architecture.Plugins {
		if strings.Contains(p, "Nuxt UI") {
			foundNuxtUI = true
		}
		if strings.Contains(p, "Tailwind") {
			foundTailwind = true
		}
		if strings.Contains(p, "Pinia") {
			foundPinia = true
		}
		if strings.Contains(p, "Zod") {
			foundZod = true
		}
	}
	if !foundNuxtUI {
		t.Errorf("Expected Nuxt UI in plugins: %v", res.Architecture.Plugins)
	}
	if !foundTailwind {
		t.Errorf("Expected Tailwind in plugins: %v", res.Architecture.Plugins)
	}
	if !foundPinia {
		t.Errorf("Expected Pinia in plugins: %v", res.Architecture.Plugins)
	}
	if !foundZod {
		t.Errorf("Expected Zod in plugins: %v", res.Architecture.Plugins)
	}

	// 3. Verify Routes extraction
	if len(res.Routes) < 4 {
		t.Errorf("Expected at least 4 page routes, got %d (routes: %+v)", len(res.Routes), res.Routes)
	}
	foundHome := false
	foundProjectID := false
	foundSettings := false
	for _, r := range res.Routes {
		if r.Path == "/" {
			foundHome = true
		}
		if r.Path == "/projects/:id" {
			foundProjectID = true
		}
		if r.Path == "/settings" {
			foundSettings = true
		}
	}
	if !foundHome {
		t.Errorf("Expected root route '/' to be extracted")
	}
	if !foundProjectID {
		t.Errorf("Expected parameterized route '/projects/:id' to be extracted")
	}
	if !foundSettings {
		t.Errorf("Expected '/settings' route to be extracted")
	}

	// 4. Verify Capabilities & Env extraction
	if res.ExtractedEnv.DatabaseType != "Client Cache & Remote REST API" {
		t.Errorf("Expected frontend DB type, got: %s", res.ExtractedEnv.DatabaseType)
	}
	caps, ok := res.ExtractedEnv.DynamicMetadata["Frontend Capabilities"]
	if !ok || !strings.Contains(caps, "Tailwind CSS") {
		t.Errorf("Expected Frontend Capabilities in DynamicMetadata, got: %v", res.ExtractedEnv.DynamicMetadata)
	}

	// 5. Verify Double Security Sanitization
	redactedCount, _ := res.SecurityAudit["redacted_count"].(int)
	if redactedCount < 1 {
		t.Errorf("Expected at least 1 redacted key in audit, got %d", redactedCount)
	}
	if strings.Contains(res.AiAnalysisMarkdown, "sk-proj-feSecretKey") {
		t.Errorf("Raw key leaked in AI analysis markdown!")
	}
}
