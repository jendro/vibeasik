package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestOpenExplicitSQLite_Boundary(t *testing.T) {
	tmpDir := t.TempDir()
	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}

	// 1. Valid case: valid path in writable temp dir
	validPath := filepath.Join(tmpDir, "sub", "valid.db")
	db, err := openExplicitSQLite(validPath, gormCfg)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if db == nil {
		t.Fatal("expected non-nil db instance")
	}

	// 2. Invalid case: directory path where file creation fails (e.g. invalid dev path)
	invalidPath := "/dev/null/sub/invalid.db"
	_, err = openExplicitSQLite(invalidPath, gormCfg)
	if err == nil {
		t.Error("expected error for invalid sqlite path, got nil")
	}
}

func TestResolveDefaultSQLitePath_Boundary(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Valid configured path
	customPath := filepath.Join(tmpDir, "custom", "test.db")
	resolved := resolveDefaultSQLitePath(customPath)
	if resolved != customPath {
		t.Errorf("expected %s, got %s", customPath, resolved)
	}
	if _, err := os.Stat(filepath.Dir(customPath)); err != nil {
		t.Errorf("expected directory to be created: %v", err)
	}

	// 2. Empty configured path -> falls back to user home dir or data/projects.db
	defaultResolved := resolveDefaultSQLitePath("")
	if defaultResolved == "" {
		t.Error("expected non-empty default sqlite path")
	}
	if !strings.HasSuffix(defaultResolved, "projects.db") {
		t.Errorf("expected path ending in projects.db, got %s", defaultResolved)
	}
}

func TestOpenDefaultSQLite_Boundary(t *testing.T) {
	tmpDir := t.TempDir()
	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}

	// 1. Valid case
	cfg := &config.Config{
		DBPath: filepath.Join(tmpDir, "default.db"),
	}
	db, err := openDefaultSQLite(cfg, gormCfg)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if db == nil {
		t.Fatal("expected non-nil db")
	}

	// 2. Invalid case
	invalidCfg := &config.Config{
		DBPath: "/dev/null/sub/invalid.db",
	}
	_, err = openDefaultSQLite(invalidCfg, gormCfg)
	if err == nil {
		t.Error("expected error for invalid db path, got nil")
	}
}

func TestOpenMySQLOrFallback_Boundary(t *testing.T) {
	tmpDir := t.TempDir()
	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}

	// 1. Fallback to valid SQLite when MySQL unreachable
	fallbackPath := filepath.Join(tmpDir, "fallback.db")
	cfg := &config.Config{
		DBDriver:   "mysql",
		DBHost:     "127.0.0.1",
		DBPort:     "59999", // non-existent MySQL port
		DBUser:     "invalid_user",
		DBPassword: "invalid_password",
		DBName:     "invalid_db",
		DBPath:     fallbackPath,
	}

	db, err := openMySQLOrFallback(cfg, gormCfg)
	if err != nil {
		t.Fatalf("expected fallback to sqlite to succeed, got: %v", err)
	}
	if db == nil {
		t.Fatal("expected non-nil fallback db")
	}

	// 2. Invalid case: MySQL unreachable and fallback path is invalid
	invalidCfg := &config.Config{
		DBDriver:   "mysql",
		DBHost:     "127.0.0.1",
		DBPort:     "59999",
		DBUser:     "invalid_user",
		DBPassword: "invalid_password",
		DBName:     "invalid_db",
		DBPath:     "/dev/null/sub/invalid.db",
	}
	_, err = openMySQLOrFallback(invalidCfg, gormCfg)
	if err == nil {
		t.Error("expected error when both mysql and sqlite fallback fail, got nil")
	}
}

func TestOpenDatabaseConnection_Boundary(t *testing.T) {
	tmpDir := t.TempDir()
	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}

	// 1. Explicit path override takes precedence
	overridePath := filepath.Join(tmpDir, "override.db")
	cfg := &config.Config{
		DBDriver: "mysql",
		DBPath:   filepath.Join(tmpDir, "other.db"),
	}
	db, err := openDatabaseConnection(overridePath, cfg, gormCfg)
	if err != nil {
		t.Fatalf("expected override connection success, got: %v", err)
	}
	if db == nil {
		t.Fatal("expected non-nil db")
	}

	// 2. Default SQLite driver
	defaultPath := filepath.Join(tmpDir, "default_driver.db")
	sqliteCfg := &config.Config{
		DBDriver: "sqlite",
		DBPath:   defaultPath,
	}
	db2, err := openDatabaseConnection("", sqliteCfg, gormCfg)
	if err != nil {
		t.Fatalf("expected default sqlite success, got: %v", err)
	}
	if db2 == nil {
		t.Fatal("expected non-nil db2")
	}
}

func TestBackfillProjectShortDescriptions_Boundary(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "backfill.db")
	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}

	db, err := openExplicitSQLite(dbPath, gormCfg)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	_ = db.AutoMigrate(&models.Project{})

	longDesc := strings.Repeat("A", 250)
	multiLineDesc := "First line summary\nSecond line details\nThird line notes"

	p1 := models.Project{Name: "P1", Slug: "p1", Description: longDesc, ShortDescription: ""}
	p2 := models.Project{Name: "P2", Slug: "p2", Description: multiLineDesc, ShortDescription: ""}
	p3 := models.Project{Name: "P3", Slug: "p3", Description: "", ShortDescription: ""}
	p4 := models.Project{Name: "P4", Slug: "p4", Description: "Already short", ShortDescription: "Custom short"}

	if err := db.Create(&p1).Error; err != nil {
		t.Fatalf("failed to create p1: %v", err)
	}
	if err := db.Create(&p2).Error; err != nil {
		t.Fatalf("failed to create p2: %v", err)
	}
	if err := db.Create(&p3).Error; err != nil {
		t.Fatalf("failed to create p3: %v", err)
	}
	if err := db.Create(&p4).Error; err != nil {
		t.Fatalf("failed to create p4: %v", err)
	}

	backfillProjectShortDescriptions(db)

	var res1, res2, res3, res4 models.Project
	db.First(&res1, p1.ID)
	db.First(&res2, p2.ID)
	db.First(&res3, p3.ID)
	db.First(&res4, p4.ID)

	if len(res1.ShortDescription) > 220 || !strings.HasSuffix(res1.ShortDescription, "...") {
		t.Errorf("expected truncated short description with ..., got length %d", len(res1.ShortDescription))
	}
	if res2.ShortDescription != "First line summary" {
		t.Errorf("expected first line only, got %q", res2.ShortDescription)
	}
	if res3.ShortDescription != "" {
		t.Errorf("expected empty short description for empty description, got %q", res3.ShortDescription)
	}
	if res4.ShortDescription != "Custom short" {
		t.Errorf("expected unchanged short description, got %q", res4.ShortDescription)
	}
}

func TestInitDB_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "init_test.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	if db == nil {
		t.Fatal("expected non-nil DB instance")
	}
	if DB == nil {
		t.Fatal("expected global DB pointer to be set")
	}

	// Verify schema migration by checking table existence
	if !db.Migrator().HasTable(&models.Project{}) {
		t.Error("expected projects table to exist")
	}
	if !db.Migrator().HasTable(&models.Todo{}) {
		t.Error("expected todos table to exist")
	}
	if !db.Migrator().HasTable(&models.LLMSettings{}) {
		t.Error("expected llm_settings table to exist")
	}

	// Verify default configurations were populated
	var llmCount int64
	db.Model(&models.LLMSettings{}).Count(&llmCount)
	if llmCount == 0 {
		t.Error("expected default LLM settings to be initialized")
	}
}
