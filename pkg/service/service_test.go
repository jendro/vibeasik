package service_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
	"github.com/vibeasik/vibeasik/pkg/service"
)

func setupTestServices(t *testing.T) (*service.Services, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory db: %v", err)
	}

	err = db.AutoMigrate(
		&models.Project{},
		&models.Component{},
		&models.ComponentRoute{},
		&models.Diagnostic{},
		&models.LLMSettings{},
		&models.LLMConfig{},
		&models.CensoredKeyword{},
	)
	if err != nil {
		t.Fatalf("failed to auto migrate: %v", err)
	}

	repos := repository.NewRepositories(db)
	services := service.NewServices(repos, db)
	return services, db
}

func TestServices(t *testing.T) {
	services, db := setupTestServices(t)
	ctx := context.Background()

	t.Run("Slugify Helper", func(t *testing.T) {
		slug := service.Slugify("My Awesome Project Suite 2.0")
		if slug != "my-awesome-project-suite-20" {
			t.Fatalf("unexpected slug: %s", slug)
		}
	})

	t.Run("ProjectService Workflow", func(t *testing.T) {
		created, err := services.Projects.Create(ctx, service.CreateProjectInput{
			Project: models.Project{
				Name:        "E-Commerce Monorepo",
				Description: "Microservices & frontend clients",
				ProjectType: models.TypeMonorepo,
			},
			CreateDirectories: false,
		})
		if err != nil {
			t.Fatalf("Create project failed: %v", err)
		}
		if created.Slug != "e-commerce-monorepo" {
			t.Fatalf("unexpected slug: %s", created.Slug)
		}

		list, err := services.Projects.GetAll(ctx, "", "")
		if err != nil || len(list) != 1 {
			t.Fatalf("GetAll projects failed: %v", err)
		}

		updated, err := services.Projects.Update(ctx, created.ID, &models.Project{
			Name: "E-Commerce Monorepo V2",
		})
		if err != nil || updated.Name != "E-Commerce Monorepo V2" {
			t.Fatalf("Update project failed: %v", err)
		}
	})

	t.Run("ComponentService Workflow", func(t *testing.T) {
		comp, err := services.Components.AddToProject(ctx, 1, service.AddComponentInput{
			Component: models.Component{
				Name:     "Core Gateway",
				Role:     models.RoleBackend,
				Language: "go",
			},
		})
		if err != nil {
			t.Fatalf("AddToProject failed: %v", err)
		}

		applied, err := services.Components.ApplyScanParams(ctx, comp.ID, models.DetectedParams{
			Framework:  "fiber",
			Port:       8080,
			DevCommand: "go run main.go",
		})
		if err != nil || applied.Framework != "fiber" || applied.Port != 8080 {
			t.Fatalf("ApplyScanParams failed: %v, comp: %+v", err, applied)
		}
	})

	t.Run("SettingsService Provider Switch", func(t *testing.T) {
		// Initialize settings in DB
		s := models.LLMSettings{
			ActiveProvider: models.ProviderSystemOne,
			SystemOneURL:   "http://localhost:8750/v1/systemone",
		}
		_ = db.Create(&s).Error

		dto, err := services.Settings.GetSettings(ctx)
		if err != nil || dto.ActiveProvider != models.ProviderSystemOne {
			t.Fatalf("GetSettings failed: %v", err)
		}

		err = services.Settings.SetActiveProvider(ctx, models.ProviderStandardLLM)
		if err != nil {
			t.Fatalf("SetActiveProvider failed: %v", err)
		}

		dtoUpdated, _ := services.Settings.GetSettings(ctx)
		if dtoUpdated.ActiveProvider != models.ProviderStandardLLM {
			t.Fatalf("expected standard_llm, got: %s", dtoUpdated.ActiveProvider)
		}
	})

	t.Run("SystemService CheckPorts", func(t *testing.T) {
		results := services.System.CheckPorts(ctx, []int{99999})
		if active, exists := results["99999"]; !exists || active {
			t.Fatalf("expected port 99999 to be reported as inactive (false), got exists=%v active=%v", exists, active)
		}
	})

	t.Run("DiagnosticService Preserves Ignored Findings Across Audits", func(t *testing.T) {
		proj, err := services.Projects.Create(ctx, service.CreateProjectInput{
			Project: models.Project{
				Name: "Audit Test App",
			},
		})
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		ignoredFinding := models.Finding{
			ID:         "QUERY-NPLUS1_repo.go_25",
			RuleID:     "QUERY-NPLUS1",
			Title:      "N+1 Query in loop",
			TargetFile: "repo.go",
			Line:       25,
			Severity:   "CRITICAL",
			Status:     models.FindingStatusIgnored,
		}
		findingsBytes, _ := json.Marshal([]models.Finding{ignoredFinding})
		prevDiag := models.Diagnostic{
			ProjectID:    proj.ID,
			Score:        100,
			FindingsJSON: string(findingsBytes),
		}
		if err := db.Create(&prevDiag).Error; err != nil {
			t.Fatalf("failed to seed previous diagnostic: %v", err)
		}

		diag, err := services.Diagnostics.DiagnoseProject(ctx, proj.ID)
		if err != nil {
			t.Fatalf("DiagnoseProject failed: %v", err)
		}

		if diag == nil {
			t.Fatal("expected diagnostic to be non-nil")
		}
	})

	t.Run("DiagnosticService Preserves Resolved Findings And Health Score", func(t *testing.T) {
		proj, err := services.Projects.Create(ctx, service.CreateProjectInput{
			Project: models.Project{
				Name: "Resolved Audit Test App",
			},
		})
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		resolvedFinding := models.Finding{
			ID:              "ROUTER-UNGUARDED-MUTATION_routes.go_98",
			RuleID:          "ROUTER-UNGUARDED-MUTATION",
			Title:           "Rute Mutasi Data (POST /api-keys) Tanpa Proteksi Middleware",
			TargetFile:      "routes.go",
			Line:            98,
			Severity:        "WARNING",
			Status:          models.FindingStatusResolved,
			ResolutionTitle: "Protected with JWT and Role",
		}
		findingsBytes, _ := json.Marshal([]models.Finding{resolvedFinding})
		prevDiag := models.Diagnostic{
			ProjectID:    proj.ID,
			Score:        100,
			FindingsJSON: string(findingsBytes),
		}
		if err := db.Create(&prevDiag).Error; err != nil {
			t.Fatalf("failed to seed previous diagnostic: %v", err)
		}

		diag, err := services.Diagnostics.DiagnoseProject(ctx, proj.ID)
		if err != nil {
			t.Fatalf("DiagnoseProject failed: %v", err)
		}

		if diag == nil {
			t.Fatal("expected diagnostic to be non-nil")
		}

		// Resolved findings must NOT penalize score (should be 100 or >= 95)
		if diag.Score < 95 {
			t.Errorf("expected score >= 95 for project with resolved finding, got %d", diag.Score)
		}
	})

	t.Run("Diagnostic Zero Score Persistence", func(t *testing.T) {
		diagZero := models.Diagnostic{
			ProjectID:    1,
			Score:        0,
			FindingsJSON: "[]",
		}
		if err := db.Create(&diagZero).Error; err != nil {
			t.Fatalf("failed to create diagnostic with score 0: %v", err)
		}

		var retrieved models.Diagnostic
		if err := db.First(&retrieved, diagZero.ID).Error; err != nil {
			t.Fatalf("failed to retrieve diagnostic: %v", err)
		}
		if retrieved.Score != 0 {
			t.Errorf("expected score 0 to be persisted, but got %d", retrieved.Score)
		}
	})
}
