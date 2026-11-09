package repository_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
)

func setupTestDB(t *testing.T) *gorm.DB {
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

	return db
}

func TestRepositories(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.NewRepositories(db)
	ctx := context.Background()

	// 1. Test ProjectRepository
	t.Run("ProjectRepository CRUD", func(t *testing.T) {
		proj := models.Project{
			Name:        "Test Bundle",
			Slug:        "test-bundle",
			Description: "Test project for repo",
			ProjectType: models.TypeBundle,
			Status:      "active",
		}
		if err := repos.Projects.Create(ctx, &proj); err != nil {
			t.Fatalf("failed to create project: %v", err)
		}
		if proj.ID == 0 {
			t.Fatalf("expected non-zero project ID")
		}

		found, err := repos.Projects.FindByID(ctx, proj.ID)
		if err != nil || found.Name != "Test Bundle" {
			t.Fatalf("FindByID failed: %v", err)
		}

		bySlug, err := repos.Projects.FindBySlug(ctx, "test-bundle")
		if err != nil || bySlug.ID != proj.ID {
			t.Fatalf("FindBySlug failed: %v", err)
		}

		all, err := repos.Projects.FindAll(ctx, "", "")
		if err != nil || len(all) != 1 {
			t.Fatalf("FindAll failed: %v, len: %d", err, len(all))
		}

		// FindByIDs tests
		byEmptyIDs, err := repos.Projects.FindByIDs(ctx, []uint{})
		if err != nil || len(byEmptyIDs) != 0 {
			t.Fatalf("FindByIDs with empty slice failed: %v, len: %d", err, len(byEmptyIDs))
		}

		byIDs, err := repos.Projects.FindByIDs(ctx, []uint{proj.ID})
		if err != nil || len(byIDs) != 1 || byIDs[0].ID != proj.ID {
			t.Fatalf("FindByIDs with valid ID failed: %v, len: %d", err, len(byIDs))
		}

		byNonExistent, err := repos.Projects.FindByIDs(ctx, []uint{99999})
		if err != nil || len(byNonExistent) != 0 {
			t.Fatalf("FindByIDs with nonexistent ID failed: %v, len: %d", err, len(byNonExistent))
		}

		_ = repos.Projects.UpdateHealthScore(ctx, proj.ID, 95)
		updated, _ := repos.Projects.FindByID(ctx, proj.ID)
		if updated.HealthScore != 95 {
			t.Fatalf("expected health score 95, got %d", updated.HealthScore)
		}
	})

	// 2. Test ComponentRepository
	t.Run("ComponentRepository CRUD", func(t *testing.T) {
		comp := models.Component{
			ProjectID: 1,
			Name:      "Test Backend",
			Role:      models.RoleBackend,
			Language:  "go",
			Framework: "fiber",
		}
		if err := repos.Components.Create(ctx, &comp); err != nil {
			t.Fatalf("failed to create component: %v", err)
		}

		found, err := repos.Components.FindByID(ctx, comp.ID)
		if err != nil || found.Name != "Test Backend" {
			t.Fatalf("FindByID component failed: %v", err)
		}

		err = repos.Components.UpdateFields(ctx, comp.ID, map[string]interface{}{"port": 9000})
		if err != nil {
			t.Fatalf("UpdateFields failed: %v", err)
		}
		updated, _ := repos.Components.FindByID(ctx, comp.ID)
		if updated.Port != 9000 {
			t.Fatalf("expected port 9000, got %d", updated.Port)
		}
	})

	// 3. Test RouteRepository
	t.Run("RouteRepository CRUD", func(t *testing.T) {
		routes := []models.ComponentRoute{
			{ComponentID: 1, Method: "GET", Path: "/api/test", Handler: "TestHandler"},
			{ComponentID: 1, Method: "POST", Path: "/api/test", Handler: "CreateHandler"},
		}
		if err := repos.Routes.BulkCreate(ctx, routes); err != nil {
			t.Fatalf("BulkCreate failed: %v", err)
		}

		list, err := repos.Routes.FindByComponentID(ctx, 1)
		if err != nil || len(list) != 2 {
			t.Fatalf("FindByComponentID failed: %v, len: %d", err, len(list))
		}

		if err := repos.Routes.Delete(ctx, list[0].ID); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		afterDelete, _ := repos.Routes.FindByComponentID(ctx, 1)
		if len(afterDelete) != 1 {
			t.Fatalf("expected 1 route remaining, got %d", len(afterDelete))
		}
	})

	// 4. Test SettingsRepository
	t.Run("SettingsRepository Multi-Provider", func(t *testing.T) {
		cfg1 := models.LLMConfig{
			Provider:      models.ProviderSystemOne,
			Name:          "Local System One",
			URL:           "http://localhost:8750/v1/systemone",
			SelectedModel: "laya",
			IsActive:      true,
		}
		cfg2 := models.LLMConfig{
			Provider:      models.ProviderSystemOne,
			Name:          "Remote Jev",
			URL:           "https://api.typesafe.ai/v1/systemone",
			SelectedModel: "jev-latest",
			IsActive:      false,
		}
		_ = repos.Settings.CreateConfig(ctx, &cfg1)
		_ = repos.Settings.CreateConfig(ctx, &cfg2)

		active, err := repos.Settings.GetActiveConfig(ctx, models.ProviderSystemOne)
		if err != nil || active.ID != cfg1.ID {
			t.Fatalf("expected active config cfg1, got %v", active)
		}

		// Activate cfg2
		activated, err := repos.Settings.SetActiveConfig(ctx, cfg2.ID)
		if err != nil || !activated.IsActive {
			t.Fatalf("failed to activate cfg2: %v", err)
		}

		oldCfg, _ := repos.Settings.GetConfigByID(ctx, cfg1.ID)
		if oldCfg.IsActive {
			t.Fatalf("expected cfg1 to be deactivated")
		}
	})

	// 5. Test SecurityRepository
	t.Run("SecurityRepository Keywords", func(t *testing.T) {
		kw := models.CensoredKeyword{
			Keyword:  "SUPER_SECRET_KEY",
			Category: "token",
			IsActive: true,
		}
		if err := repos.Security.Create(ctx, &kw); err != nil {
			t.Fatalf("failed to create censored keyword: %v", err)
		}

		actives, err := repos.Security.GetActive(ctx)
		if err != nil || len(actives) == 0 {
			t.Fatalf("expected at least 1 active keyword, got %d", len(actives))
		}
	})
}
