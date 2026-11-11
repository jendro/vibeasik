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

// openExplicitSQLite membuka koneksi SQLite pada path eksplisit (misal dari flag CLI --db).
func openExplicitSQLite(path string, gormConfig *gorm.Config) (*gorm.DB, error) {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)
	log.Printf("[Database] Connecting to explicit SQLite database at: %s\n", path)
	db, err := gorm.Open(sqlite.Open(path), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database (%s): %w", path, err)
	}
	return db, nil
}

// openMySQLOrFallback menghubungkan ke MySQL dan otomatis fallback ke SQLite jika gagal.
func openMySQLOrFallback(cfg *config.Config, gormConfig *gorm.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	log.Printf("[Database] Connecting to MySQL (%s:%s/%s)...\n", cfg.DBHost, cfg.DBPort, cfg.DBName)
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err == nil {
		log.Printf("✅ [Database] Successfully connected to MySQL database '%s'\n", cfg.DBName)
		return db, nil
	}

	log.Printf("⚠️ [Database] Failed to connect to MySQL (%s:%s/%s): %v\n", cfg.DBHost, cfg.DBPort, cfg.DBName, err)
	log.Printf("⚠️ [Database] Falling back to SQLite (%s)...\n", cfg.DBPath)

	dir := filepath.Dir(cfg.DBPath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	fallbackDB, fallbackErr := gorm.Open(sqlite.Open(cfg.DBPath), gormConfig)
	if fallbackErr != nil {
		return nil, fmt.Errorf("database initialization failed: mysql error (%v) and sqlite fallback error (%w)", err, fallbackErr)
	}
	return fallbackDB, nil
}

// resolveDefaultSQLitePath menentukan path penyimpanan SQLite default bila tidak ditentukan.
func resolveDefaultSQLitePath(configuredPath string) string {
	if configuredPath != "" {
		dir := filepath.Dir(configuredPath)
		if dir != "" && dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}
		return configuredPath
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		_ = os.MkdirAll("data", 0755)
		return "data/projects.db"
	}

	appDir := filepath.Join(homeDir, ".projectmanager")
	_ = os.MkdirAll(appDir, 0755)
	return filepath.Join(appDir, "projects.db")
}

// openDefaultSQLite membuka SQLite default berdasarkan konfigurasi sistem.
func openDefaultSQLite(cfg *config.Config, gormConfig *gorm.Config) (*gorm.DB, error) {
	dbPath := resolveDefaultSQLitePath(cfg.DBPath)
	log.Printf("[Database] Connecting to SQLite at %s\n", dbPath)
	db, err := gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database (%s): %w", dbPath, err)
	}
	return db, nil
}

// openDatabaseConnection memilih driver database yang sesuai secara linear.
func openDatabaseConnection(dbPathOverride string, cfg *config.Config, gormConfig *gorm.Config) (*gorm.DB, error) {
	if dbPathOverride != "" {
		return openExplicitSQLite(dbPathOverride, gormConfig)
	}
	if strings.ToLower(cfg.DBDriver) == "mysql" {
		return openMySQLOrFallback(cfg, gormConfig)
	}
	return openDefaultSQLite(cfg, gormConfig)
}

// migrateDatabaseSchema menjalankan migrasi model dan optimasi kolom pada MySQL.
func migrateDatabaseSchema(db *gorm.DB) error {
	err := db.AutoMigrate(
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
		&models.ProcessCostSnapshot{},
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
		&models.EmployeeConsult{},
		&models.LLMProviderState{},
		&models.EmployeeTurnover{},
		&models.EmployeeStepStat{},
		&models.EmployeeLearnedStep{},
		&models.EmployeeCustomRole{},
		&models.EmployeePassport{},
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
		&models.TrafficErrorGroup{},
		&models.TrafficProcess{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database schema: %w", err)
	}

	if db.Dialector.Name() == "mysql" {
		tuneMySQLStorage(db)
	}
	return nil
}

// tuneMySQLStorage memastikan kolom teks berkapasitas tinggi (LONGTEXT) di MySQL.
func tuneMySQLStorage(db *gorm.DB) {
	_ = db.Exec("ALTER TABLE diagnostics MODIFY COLUMN findings_json LONGTEXT, MODIFY COLUMN optimizations LONGTEXT, MODIFY COLUMN structure_tree LONGTEXT, MODIFY COLUMN summary LONGTEXT").Error
	_ = db.Exec("ALTER TABLE deep_check_reports MODIFY COLUMN findings_json LONGTEXT, MODIFY COLUMN route_simulations_json LONGTEXT, MODIFY COLUMN summary LONGTEXT").Error
	_ = db.Exec("ALTER TABLE runtime_sim_reports MODIFY COLUMN result_json LONGTEXT, MODIFY COLUMN summary LONGTEXT").Error
	_ = db.Exec("ALTER TABLE components MODIFY COLUMN fast_scan_data LONGTEXT, MODIFY COLUMN env_summary LONGTEXT").Error
	_ = db.Exec("ALTER TABLE component_routes MODIFY COLUMN target_requests LONGTEXT").Error
	_ = db.Exec("ALTER TABLE projects MODIFY COLUMN synergy_map_data LONGTEXT").Error
}

// backfillProjectShortDescriptions mengisi short_description untuk proyek lama yang masih kosong.
func backfillProjectShortDescriptions(db *gorm.DB) {
	var emptyShortProjects []models.Project
	if err := db.Where("short_description = '' OR short_description IS NULL").Find(&emptyShortProjects).Error; err != nil {
		return
	}
	for _, ep := range emptyShortProjects {
		if ep.Description == "" {
			continue
		}
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

// InitDB initializes database connection based on config (.env) or explicit path override
func InitDB(dbPathOverride string) (*gorm.DB, error) {
	cfg := config.GetConfig()

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	db, err := openDatabaseConnection(dbPathOverride, cfg, gormConfig)
	if err != nil {
		return nil, err
	}

	if err := migrateDatabaseSchema(db); err != nil {
		return nil, err
	}

	backfillProjectShortDescriptions(db)

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
	err := db.
		Select("id", "keyword", "category", "description", "is_regex", "is_active").
		Where("is_active = ?", true).
		Find(&list).Error
	if err != nil {
		return
	}

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

type legacyCategoryRule struct {
	prefix   string
	keywords []string
	category string
}

var legacyCategoryRules = []legacyCategoryRule{
	{prefix: "SEC", category: models.CategorySecurity},
	{prefix: "ARCH", keywords: []string{"arsitektur", "runner", "dev.sh"}, category: models.CategoryArchitecture},
	{prefix: "FE", keywords: []string{"ui", "composable"}, category: models.CategoryUIUX},
	{prefix: "API", keywords: []string{"endpoint", "health"}, category: models.CategoryBackend},
	{prefix: "NET", keywords: []string{"port", "proxy"}, category: models.CategoryPerformance},
	{prefix: "AI", keywords: []string{"mcp", "neural"}, category: models.CategoryFeature},
	{prefix: "PLAN", keywords: []string{"mcp", "neural"}, category: models.CategoryFeature},
	{prefix: "CICD", keywords: []string{"ci"}, category: models.CategoryDevOps},
}

func matchLegacyRule(rule legacyCategoryRule, text, g string) bool {
	if rule.prefix != "" && strings.HasPrefix(g, rule.prefix) {
		return true
	}
	for _, kw := range rule.keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// classifyLegacyTodoCategory menentukan kategori todo lama dari judul dan guideline_ref.
// Urutan case = prioritas (SEC, ARCH, FE, API, NET, AI/PLAN, CICD); fallback ke General.
func classifyLegacyTodoCategory(title, guidelineRef string) string {
	text := strings.ToLower(title)
	g := strings.ToUpper(guidelineRef)
	for _, rule := range legacyCategoryRules {
		if matchLegacyRule(rule, text, g) {
			return rule.category
		}
	}
	return models.CategoryGeneral
}

func ensureDefaultTodos(db *gorm.DB) {
	// Backfill any existing todos with missing fields
	db.Model(&models.Todo{}).Where("level = '' OR level IS NULL").Where("component_id IS NOT NULL AND component_id > 0").Update("level", models.TodoLevelComponent)
	db.Model(&models.Todo{}).Where("level = '' OR level IS NULL").Where("project_id IS NOT NULL AND project_id > 0").Update("level", models.TodoLevelProject)
	db.Model(&models.Todo{}).Where("level = '' OR level IS NULL").Update("level", models.TodoLevelGlobal)
	db.Model(&models.Todo{}).Where("creator_type = '' OR creator_type IS NULL").Update("creator_type", models.CreatorHuman)
	db.Model(&models.Todo{}).Where("creator_name = '' OR creator_name IS NULL").Update("creator_name", "developer")

	// Backfill category for existing todos
	// Klasifikasi dilakukan di Go (satu pembacaan, lalu update per baris) agar tidak memakai
	// LIKE dengan wildcard di awal yang mematikan indeks; semantik contains + case-insensitive dipertahankan.
	var uncategorized []models.Todo
	db.Select("id", "title", "guideline_ref").Where("category = '' OR category IS NULL").Find(&uncategorized)
	for i := range uncategorized {
		t := &uncategorized[i]
		db.Model(&models.Todo{}).Where("id = ?", t.ID).Update("category", classifyLegacyTodoCategory(t.Title, t.GuidelineRef))
	}

	// Backfill target_apps for existing todos
	db.Model(&models.Todo{}).Where("(target_apps = '' OR target_apps IS NULL) AND component_name != ''").Update("target_apps", gorm.Expr("CONCAT(component_name, ' [1 App]')"))
	db.Model(&models.Todo{}).Where("(target_apps = '' OR target_apps IS NULL) AND project_name != ''").Update("target_apps", gorm.Expr("CONCAT('Seluruh Aplikasi (Bundle ', project_name, ')')"))
	db.Model(&models.Todo{}).Where("target_apps = '' OR target_apps IS NULL").Update("target_apps", "Global Platform")

	// Memori todo (ringkasan riwayat pengerjaan) untuk todo lama yang sudah pernah dikerjakan.
	if n := models.BackfillTodoMemory(db); n > 0 {
		log.Printf("[Database] memori todo diisi untuk %d todo", n)
	}

	// Todo tanpa taksiran kerumitan/dampak diisi sekarang (cepat: hanya yang kosong). Penghitungan ulang taksiran
	// heuristik untuk semua todo berat (regex per todo), jadi hanya `vibeasik serve` yang menjalankannya di latar
	// lewat RefreshTodoEstimates — proses singkat seperti `vibeasik mcp` tidak boleh tertahan puluhan detik.
	refreshTodoEstimates(db, false)
}

// RefreshTodoEstimates menghitung ulang taksiran heuristik kerumitan (logika, koding, cakupan, risiko) dan dampak
// (7 dimensi → skor 0–100) semua todo agar ikut perbaikan estimator; penilaian LLM/manual tidak disentuh.
// Dipanggil `vibeasik serve` di latar setiap start. Mengembalikan jumlah todo yang kerumitan & dampaknya berubah.
func RefreshTodoEstimates(db *gorm.DB) (complexity, impact int) {
	return refreshTodoEstimates(db, true)
}

func refreshTodoEstimates(db *gorm.DB, all bool) (complexity, impact int) {
	var unrated []models.Todo
	q := db.Where("complexity_level = '' OR complexity_level IS NULL")
	if all {
		q = q.Or("complexity_source = ?", models.ComplexitySourceHeuristic)
	}
	q.Find(&unrated)
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
		complexity++
	}

	var noImpact []models.Todo
	q = db.Where("impact_source = '' OR impact_source IS NULL")
	if all {
		q = q.Or("impact_source = ?", models.ComplexitySourceHeuristic)
	}
	q.Find(&noImpact)
	for i := range noImpact {
		imp := models.EstimateImpact(&noImpact[i])
		if imp == noImpact[i].TodoImpact {
			continue
		}
		db.Model(&models.Todo{}).Where("id = ?", noImpact[i].ID).Updates(imp.Columns())
		impact++
	}
	return complexity, impact
}
