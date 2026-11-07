package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/security"
)

var DB *gorm.DB

// InitDB initializes database connection based on config (.env) or explicit path override
func InitDB(dbPathOverride string) (*gorm.DB, error) {
	cfg := config.GetConfig()

	var db *gorm.DB
	var err error

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	// 1. If explicit SQLite flag --db was provided, respect it
	if dbPathOverride != "" {
		dir := filepath.Dir(dbPathOverride)
		_ = os.MkdirAll(dir, 0755)
		log.Printf("[Database] Connecting to explicit SQLite database at: %s\n", dbPathOverride)
		db, err = gorm.Open(sqlite.Open(dbPathOverride), gormConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to open SQLite database (%s): %w", dbPathOverride, err)
		}
	} else if strings.ToLower(cfg.DBDriver) == "mysql" {
		// 2. Connect to MySQL via environment variables
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

		log.Printf("[Database] Connecting to MySQL (%s:%s/%s)...\n", cfg.DBHost, cfg.DBPort, cfg.DBName)
		db, err = gorm.Open(mysql.Open(dsn), gormConfig)
		if err != nil {
			log.Printf("⚠️ [Database] Failed to connect to MySQL (%s:%s/%s): %v\n", cfg.DBHost, cfg.DBPort, cfg.DBName, err)
			log.Printf("⚠️ [Database] Falling back to SQLite (%s)...\n", cfg.DBPath)

			dir := filepath.Dir(cfg.DBPath)
			if dir != "" && dir != "." {
				_ = os.MkdirAll(dir, 0755)
			}
			db, err = gorm.Open(sqlite.Open(cfg.DBPath), gormConfig)
			if err != nil {
				return nil, fmt.Errorf("database initialization failed: mysql error (%v) and sqlite fallback error (%w)", err, err)
			}
		} else {
			log.Printf("✅ [Database] Successfully connected to MySQL database '%s'\n", cfg.DBName)
		}
	} else {
		// 3. SQLite configured in DB_DRIVER
		dbPath := cfg.DBPath
		if dbPath == "" {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				appDir := filepath.Join(homeDir, ".projectmanager")
				_ = os.MkdirAll(appDir, 0755)
				dbPath = filepath.Join(appDir, "projects.db")
			} else {
				_ = os.MkdirAll("data", 0755)
				dbPath = "data/projects.db"
			}
		} else {
			dir := filepath.Dir(dbPath)
			if dir != "" && dir != "." {
				_ = os.MkdirAll(dir, 0755)
			}
		}

		log.Printf("[Database] Connecting to SQLite at %s\n", dbPath)
		db, err = gorm.Open(sqlite.Open(dbPath), gormConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to open SQLite database (%s): %w", dbPath, err)
		}
	}

	// Migrate schema
	err = db.AutoMigrate(
		&models.Project{},
		&models.Component{},
		&models.ComponentRoute{},
		&models.Diagnostic{},
		&models.LLMSettings{},
		&models.LLMConfig{},
		&models.CensoredKeyword{},
		&models.LocalAgentConfig{},
		&models.Todo{},
		&models.DeepCheckReport{},
		&models.RuntimeSimReport{},
		&models.RuntimeTestProfile{},
		&models.ProjectChatMessage{},
		&models.WorkspaceSettings{},
		&models.AgentWorker{},
		&models.AgentProfile{},
		&models.AgentRun{},
		&models.OrchestratorSettings{},
		&models.CodeMap{},
		&models.CodeMapEntry{},
		&models.PlanningSession{},
		&models.PlannerSettings{},
		&models.AuditFeedback{},
		&models.DesignReference{},
		&models.DesignPage{},
		&models.DesignLesson{},
		&models.Employee{},
		&models.EmployeeKnowledge{},
		&models.EmployeeTask{},
		&models.EmployeeReview{},
		&models.EmployeeChatMessage{},
		&models.StaffSettings{},
		&models.EmployeeAgenda{},
		&models.EmployeeConversation{},
		&models.EmployeeHRAction{},
		&models.EmployeeRevisit{},
		&models.TodoQAGate{},
		&models.LLMProviderState{},
		&models.EmployeeTurnover{},
		&models.ProjectExecutive{},
		&models.ExecutiveKnowledge{},
		&models.ExecutiveChatMessage{},
		&models.KnowledgeNode{},
		&models.KnowledgeLink{},
		&models.KnowledgeCluster{},
		&models.KnowledgeUse{},
		&models.KnowledgeGap{},
		&models.TrafficSource{},
		&models.TrafficRouteStat{},
		&models.TrafficDimStat{},
		&models.TrafficRecentEvent{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database schema: %w", err)
	}

	// Ensure high-capacity text storage (LONGTEXT) in MySQL for JSON payloads & trees
	if db.Dialector.Name() == "mysql" {
		_ = db.Exec("ALTER TABLE diagnostics MODIFY COLUMN findings_json LONGTEXT, MODIFY COLUMN optimizations LONGTEXT, MODIFY COLUMN structure_tree LONGTEXT, MODIFY COLUMN summary LONGTEXT").Error
		_ = db.Exec("ALTER TABLE deep_check_reports MODIFY COLUMN findings_json LONGTEXT, MODIFY COLUMN route_simulations_json LONGTEXT, MODIFY COLUMN summary LONGTEXT").Error
		_ = db.Exec("ALTER TABLE runtime_sim_reports MODIFY COLUMN result_json LONGTEXT, MODIFY COLUMN summary LONGTEXT").Error
		_ = db.Exec("ALTER TABLE components MODIFY COLUMN fast_scan_data LONGTEXT, MODIFY COLUMN env_summary LONGTEXT").Error
		_ = db.Exec("ALTER TABLE component_routes MODIFY COLUMN target_requests LONGTEXT").Error
		_ = db.Exec("ALTER TABLE projects MODIFY COLUMN synergy_map_data LONGTEXT").Error
	}

	// Backfill short_description for existing projects where empty
	var emptyShortProjects []models.Project
	if err := db.Where("short_description = '' OR short_description IS NULL").Find(&emptyShortProjects).Error; err == nil {
		for _, ep := range emptyShortProjects {
			if ep.Description != "" {
				short := strings.TrimSpace(ep.Description)
				if idx := strings.Index(short, "\n"); idx != -1 {
					short = strings.TrimSpace(short[:idx])
				}
				if len(short) > 220 {
					short = short[:217] + "..."
				}
				_ = db.Model(&ep).Update("short_description", short).Error
			}
		}
	}

	DB = db
	ensureDefaultLLMSettings(db)
	ensureDefaultLLMConfigs(db)
	ensureDefaultWorkspaceSettings(db)
	ensureDefaultCensoredKeywords(db)
	ensureDefaultTodos(db)
	// Todo gabungan (Aturan 43): sub todo mengikuti status induknya di semua jalur update/hapus todo.
	models.RegisterSubTodoSettle(db)
	if n := models.SettleSubTodos(db); n > 0 {
		log.Printf("[Database] %d sub todo diselaraskan dengan todo gabungannya", n)
	}

	models.SetLLMHealthDB(db)
	return db, nil
}

func ensureDefaultLLMSettings(db *gorm.DB) {
	var count int64
	db.Model(&models.LLMSettings{}).Count(&count)
	if count > 0 {
		return
	}

	log.Println("[Database] Initializing default LLM settings...")
	defaultModels := "[\"gemini-3.8-flash\", \"deepseek-v4.1-flash\", \"claude-opus-5.5\", \"kimi-k3\"]"
	defaultSettings := models.LLMSettings{
		ActiveProvider:      models.ProviderSystemOne,
		SystemOneURL:        "https://api.typesafe.ai/v1/systemone",
		SystemOneModel:      "jev-latest",
		SystemOneKey:        "",
		StandardLLMURL:      "https://limitrouter.com/v1",
		StandardLLMKey:      "",
		StandardLLMModels:   defaultModels,
		StandardLLMSelected: "gemini-3.8-flash",
	}
	db.Create(&defaultSettings)
}

func ensureDefaultLLMConfigs(db *gorm.DB) {
	var count int64
	db.Model(&models.LLMConfig{}).Count(&count)
	if count > 0 {
		return
	}

	log.Println("[Database] Initializing default multi-provider LLM configurations grid...")

	// Default System One & Standard LLM configs
	defaultConfigs := []models.LLMConfig{
		{
			Provider:      models.ProviderSystemOne,
			Name:          "Laya Local",
			URL:           "http://192.168.1.9:8750/v1/systemone",
			Key:           "",
			Models:        "[\"laya\"]",
			SelectedModel: "laya",
			IsActive:      false,
		},
		{
			Provider:      models.ProviderSystemOne,
			Name:          "TypeSafe Jev (Cloud Official)",
			URL:           "https://api.typesafe.ai/v1/systemone",
			Key:           "",
			Models:        "[\"jev-latest\"]",
			SelectedModel: "jev-latest",
			IsActive:      true,
		},
		{
			Provider:      models.ProviderSystemOne,
			Name:          "Laya Local 2",
			URL:           "http://192.168.1.9:8750/v1/systemone",
			Key:           "sk-laya-systemone-local",
			Models:        "[\"laya\"]",
			SelectedModel: "laya",
			IsActive:      false,
		},
		{
			Provider:      models.ProviderStandardLLM,
			Name:          "Gemini",
			URL:           "https://limitrouter.com/v1",
			Key:           "",
			Models:        "[\"gemini-3.8-flash\", \"deepseek-v4.1-flash\", \"claude-opus-5.5\", \"kimi-k3\"]",
			SelectedModel: "gemini-3.8-flash",
			IsActive:      true,
		},
	}

	for _, cfg := range defaultConfigs {
		db.Create(&cfg)
	}
}

func ensureDefaultWorkspaceSettings(db *gorm.DB) {
	var count int64
	db.Model(&models.WorkspaceSettings{}).Count(&count)
	if count > 0 {
		return
	}

	defaultRoot := models.DefaultProjectsRoot()

	log.Printf("[Database] Initializing default workspace settings (default root: %s)...\n", defaultRoot)
	ws := models.WorkspaceSettings{
		DefaultProjectsRoot: defaultRoot,
	}
	db.Create(&ws)
}

func ensureDefaultCensoredKeywords(db *gorm.DB) {
	var count int64
	db.Model(&models.CensoredKeyword{}).Count(&count)
	if count > 0 {
		SyncCensoredKeywords(db)
		return
	}

	log.Println("[Database] Initializing default censored keywords & patterns...")
	defaults := []models.CensoredKeyword{
		{
			Keyword:     "ADMIN_PASSWORD",
			Category:    "password",
			Description: "Master administrative secret / password parameter",
			IsRegex:     false,
			IsActive:    true,
		},
		{
			Keyword:     "JWT_SECRET",
			Category:    "token",
			Description: "JSON Web Token symmetric signing secret key",
			IsRegex:     false,
			IsActive:    true,
		},
		{
			Keyword:     "DATABASE_PASSWORD",
			Category:    "password",
			Description: "Primary database credential password",
			IsRegex:     false,
			IsActive:    true,
		},
		{
			Keyword:     "PRIVATE_KEY",
			Category:    "private_pattern",
			Description: "Cryptographic asymmetric private key phrase",
			IsRegex:     false,
			IsActive:    true,
		},
		{
			Keyword:     `(?i)session_secret\s*[:=]\s*["']?[^"'\r\n\s]{6,}["']?`,
			Category:    "token",
			Description: "Regex detector for HTTP cookie session encryption secret",
			IsRegex:     true,
			IsActive:    true,
		},
	}

	for _, item := range defaults {
		db.Create(&item)
	}

	SyncCensoredKeywords(db)
}

// SyncCensoredKeywords reloads all active censored keywords from database into the GlobalSanitizer
func SyncCensoredKeywords(db *gorm.DB) {
	if db == nil {
		return
	}
	var list []models.CensoredKeyword
	if err := db.Where("is_active = ?", true).Find(&list).Error; err == nil {
		rules := make([]security.CustomKeywordRule, 0, len(list))
		for _, item := range list {
			rules = append(rules, security.CustomKeywordRule{
				ID:          item.ID,
				Keyword:     item.Keyword,
				Category:    item.Category,
				Description: item.Description,
				IsRegex:     item.IsRegex,
				IsActive:    item.IsActive,
			})
		}
		security.GlobalSanitizer.SetCustomKeywords(rules)
	}
}

func ensureDefaultTodos(db *gorm.DB) {
	// Backfill any existing todos with missing fields
	db.Model(&models.Todo{}).Where("level = '' OR level IS NULL").Where("component_id IS NOT NULL AND component_id > 0").Update("level", models.TodoLevelComponent)
	db.Model(&models.Todo{}).Where("level = '' OR level IS NULL").Where("project_id IS NOT NULL AND project_id > 0").Update("level", models.TodoLevelProject)
	db.Model(&models.Todo{}).Where("level = '' OR level IS NULL").Update("level", models.TodoLevelGlobal)
	db.Model(&models.Todo{}).Where("creator_type = '' OR creator_type IS NULL").Update("creator_type", models.CreatorHuman)
	db.Model(&models.Todo{}).Where("creator_name = '' OR creator_name IS NULL").Update("creator_name", "developer")

	// Backfill category for existing todos
	db.Model(&models.Todo{}).Where("(category = '' OR category IS NULL) AND guideline_ref LIKE 'SEC%'").Update("category", models.CategorySecurity)
	db.Model(&models.Todo{}).Where("(category = '' OR category IS NULL) AND (guideline_ref LIKE 'ARCH%' OR title LIKE '%Arsitektur%' OR title LIKE '%runner%' OR title LIKE '%dev.sh%')").Update("category", models.CategoryArchitecture)
	db.Model(&models.Todo{}).Where("(category = '' OR category IS NULL) AND (guideline_ref LIKE 'FE%' OR title LIKE '%UI%' OR title LIKE '%Composable%')").Update("category", models.CategoryUIUX)
	db.Model(&models.Todo{}).Where("(category = '' OR category IS NULL) AND (guideline_ref LIKE 'API%' OR title LIKE '%endpoint%' OR title LIKE '%health%')").Update("category", models.CategoryBackend)
	db.Model(&models.Todo{}).Where("(category = '' OR category IS NULL) AND (guideline_ref LIKE 'NET%' OR title LIKE '%port%' OR title LIKE '%Proxy%')").Update("category", models.CategoryPerformance)
	db.Model(&models.Todo{}).Where("(category = '' OR category IS NULL) AND (guideline_ref LIKE 'AI%' OR guideline_ref LIKE 'PLAN%' OR title LIKE '%MCP%' OR title LIKE '%Neural%')").Update("category", models.CategoryFeature)
	db.Model(&models.Todo{}).Where("(category = '' OR category IS NULL) AND (guideline_ref LIKE 'CICD%' OR title LIKE '%CI%')").Update("category", models.CategoryDevOps)
	db.Model(&models.Todo{}).Where("category = '' OR category IS NULL").Update("category", models.CategoryGeneral)

	// Backfill target_apps for existing todos
	db.Model(&models.Todo{}).Where("(target_apps = '' OR target_apps IS NULL) AND component_name != ''").Update("target_apps", gorm.Expr("CONCAT(component_name, ' [1 App]')"))
	db.Model(&models.Todo{}).Where("(target_apps = '' OR target_apps IS NULL) AND project_name != ''").Update("target_apps", gorm.Expr("CONCAT('Seluruh Aplikasi (Bundle ', project_name, ')')"))
	db.Model(&models.Todo{}).Where("target_apps = '' OR target_apps IS NULL").Update("target_apps", "Global Platform")

	// Memori todo (ringkasan riwayat pengerjaan) untuk todo lama yang sudah pernah dikerjakan.
	if n := models.BackfillTodoMemory(db); n > 0 {
		log.Printf("[Database] memori todo diisi untuk %d todo", n)
	}

	// Backfill kerumitan (logika, koding, cakupan, risiko) agar orkestrator bisa memilih agent. Taksiran heuristik
	// dihitung ulang tiap start supaya ikut perbaikan estimator; penilaian LLM/manual tidak disentuh.
	var unrated []models.Todo
	db.Where("complexity_level = '' OR complexity_level IS NULL OR complexity_source = ?", models.ComplexitySourceHeuristic).Find(&unrated)
	for i := range unrated {
		c := models.EstimateComplexity(&unrated[i])
		if c == unrated[i].TodoComplexity {
			continue
		}
		db.Model(&models.Todo{}).Where("id = ?", unrated[i].ID).Updates(map[string]interface{}{
			"complexity_logic": c.ComplexityLogic, "complexity_coding": c.ComplexityCoding, "complexity_scope": c.ComplexityScope,
			"complexity_risk": c.ComplexityRisk, "complexity_score": c.ComplexityScore, "complexity_level": c.ComplexityLevel,
			"complexity_note": c.ComplexityNote, "complexity_source": c.ComplexitySource,
		})
	}

	// Backfill dampak (7 dimensi → skor 0–100) dengan aturan yang sama: taksiran heuristik selalu diperbarui.
	var noImpact []models.Todo
	db.Where("impact_source = '' OR impact_source IS NULL OR impact_source = ?", models.ComplexitySourceHeuristic).Find(&noImpact)
	for i := range noImpact {
		imp := models.EstimateImpact(&noImpact[i])
		if imp == noImpact[i].TodoImpact {
			continue
		}
		db.Model(&models.Todo{}).Where("id = ?", noImpact[i].ID).Updates(imp.Columns())
	}
}
