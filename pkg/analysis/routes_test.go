package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	err = db.AutoMigrate(
		&models.Project{},
		&models.Component{},
		&models.ComponentRoute{},
		&models.LLMSettings{},
		&models.LLMConfig{},
		&models.CensoredKeyword{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestSyncComponentRoutes_AddUpdateDelete(t *testing.T) {
	db := setupTestDB(t)

	// Create a temporary project directory with a mock router file
	tmpDir, err := os.MkdirTemp("", "routes_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create initial Go file with 3 routes
	mainGoContent := `package main

import "github.com/gofiber/fiber/v2"

func main() {
	app := fiber.New()
	app.Get("/api/v1/health", healthCheck)
	app.Post("/api/v1/auth/login", loginHandler)
	app.Get("/api/v1/projects", getProjects)
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGoContent), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	proj := models.Project{
		Name:     "Test Project",
		Slug:     "test-project",
		RootPath: tmpDir,
	}
	if err := db.Create(&proj).Error; err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	comp := models.Component{
		ProjectID:    proj.ID,
		Name:         "Backend Service",
		Role:         models.RoleBackend,
		AbsolutePath: tmpDir,
		Language:     "go",
		Framework:    "fiber",
	}
	if err := db.Create(&comp).Error; err != nil {
		t.Fatalf("Failed to create component: %v", err)
	}

	// 1. Initial Sync: Should add 3 routes
	syncRes, err := SyncComponentRoutes(db, &comp, &proj)
	if err != nil {
		t.Fatalf("SyncComponentRoutes failed: %v", err)
	}

	if syncRes.AddedCount != 3 {
		t.Errorf("Expected 3 added routes, got %d", syncRes.AddedCount)
	}
	if syncRes.TotalCount != 3 {
		t.Errorf("Expected 3 total routes, got %d", syncRes.TotalCount)
	}

	// Verify routes in database
	var routes []models.ComponentRoute
	db.Where("component_id = ?", comp.ID).Find(&routes)
	if len(routes) != 3 {
		t.Fatalf("Expected 3 routes in DB, got %d", len(routes))
	}

	// Verify heuristic domain
	for _, r := range routes {
		if r.Path == "/api/v1/auth/login" {
			if r.Domain != "Autentikasi & Keamanan" {
				t.Errorf("Expected domain 'Autentikasi & Keamanan', got '%s'", r.Domain)
			}
		}
	}

	// 2. Repeat Sync without code change: should NOT duplicate, 0 added, 0 deleted
	syncRes2, err := SyncComponentRoutes(db, &comp, &proj)
	if err != nil {
		t.Fatalf("Second SyncComponentRoutes failed: %v", err)
	}
	if syncRes2.AddedCount != 0 {
		t.Errorf("Expected 0 added on repeat sync, got %d", syncRes2.AddedCount)
	}
	if syncRes2.DeletedCount != 0 {
		t.Errorf("Expected 0 deleted on repeat sync, got %d", syncRes2.DeletedCount)
	}
	if syncRes2.TotalCount != 3 {
		t.Errorf("Expected 3 total routes on repeat sync, got %d", syncRes2.TotalCount)
	}

	// 3. Modify code: Remove login route, and Add 1 new route (/api/v1/users)
	modifiedGoContent := `package main

import "github.com/gofiber/fiber/v2"

func main() {
	app := fiber.New()
	app.Get("/api/v1/health", healthCheck)
	app.Get("/api/v1/projects", getProjects)
	app.Post("/api/v1/users", createUser)
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(modifiedGoContent), 0644); err != nil {
		t.Fatalf("Failed to rewrite main.go: %v", err)
	}

	// 4. Sync again: 1 removed (/api/v1/auth/login), 1 added (/api/v1/users), 2 kept
	syncRes3, err := SyncComponentRoutes(db, &comp, &proj)
	if err != nil {
		t.Fatalf("Third SyncComponentRoutes failed: %v", err)
	}

	if syncRes3.AddedCount != 1 {
		t.Errorf("Expected 1 added route, got %d", syncRes3.AddedCount)
	}
	if syncRes3.DeletedCount != 1 {
		t.Errorf("Expected 1 deleted route (jika hilang dihapus), got %d", syncRes3.DeletedCount)
	}
	if syncRes3.TotalCount != 3 {
		t.Errorf("Expected 3 total routes, got %d", syncRes3.TotalCount)
	}

	// Verify that /api/v1/auth/login was deleted from DB
	var loginRoute models.ComponentRoute
	err = db.Where("component_id = ? AND path = ?", comp.ID, "/api/v1/auth/login").First(&loginRoute).Error
	if err == nil {
		t.Errorf("Expected /api/v1/auth/login to be deleted from DB, but it was found")
	}

	// Verify that /api/v1/users is present in DB
	var userRoute models.ComponentRoute
	err = db.Where("component_id = ? AND path = ?", comp.ID, "/api/v1/users").First(&userRoute).Error
	if err != nil {
		t.Errorf("Expected /api/v1/users to be present in DB: %v", err)
	}
}

func TestSyncComponentRoutes_FrontendWithTargetRequests(t *testing.T) {
	db := setupTestDB(t)

	tmpDir, err := os.MkdirTemp("", "fe_routes_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create composable with API calls
	compDir := filepath.Join(tmpDir, "app", "composables")
	_ = os.MkdirAll(compDir, 0755)
	useApiContent := `
export const useApi = () => {
  const getProjects = async () => {
    return await $fetch('/api/projects')
  }
  const checkPorts = async () => {
    return await $fetch('/api/ports/check', { method: 'POST' })
  }
  return { getProjects, checkPorts }
}
`
	_ = os.WriteFile(filepath.Join(compDir, "useApi.ts"), []byte(useApiContent), 0644)

	// Create pages
	pagesDir := filepath.Join(tmpDir, "app", "pages")
	_ = os.MkdirAll(pagesDir, 0755)

	indexVue := `
<script setup>
const { getProjects } = useApi()
</script>
<template><div>Index Page</div></template>
`
	_ = os.WriteFile(filepath.Join(pagesDir, "index.vue"), []byte(indexVue), 0644)

	portsVue := `
<script setup>
const { getProjects, checkPorts } = useApi()
</script>
<template><div>Ports Page</div></template>
`
	_ = os.WriteFile(filepath.Join(pagesDir, "ports.vue"), []byte(portsVue), 0644)

	proj := models.Project{
		Name:     "FE Test Project",
		Slug:     "fe-test-project",
		RootPath: tmpDir,
	}
	db.Create(&proj)

	comp := models.Component{
		ProjectID:    proj.ID,
		Name:         "Frontend Web",
		Role:         models.RoleFrontendClient,
		AbsolutePath: tmpDir,
		Language:     "typescript",
		Framework:    "nuxt",
	}
	db.Create(&comp)

	syncRes, err := SyncComponentRoutes(db, &comp, &proj)
	if err != nil {
		t.Fatalf("SyncComponentRoutes failed: %v", err)
	}

	if syncRes.TotalCount != 2 {
		t.Fatalf("Expected 2 frontend pages, got %d", syncRes.TotalCount)
	}

	for _, r := range syncRes.Routes {
		if r.Path == "/" {
			if !strings.Contains(r.TargetRequests, "/api/projects") {
				t.Errorf("Expected index page to request /api/projects, got: %s", r.TargetRequests)
			}
		}
		if r.Path == "/ports" {
			if !strings.Contains(r.TargetRequests, "/api/ports/check") {
				t.Errorf("Expected ports page to request /api/ports/check, got: %s", r.TargetRequests)
			}
		}
	}
}
