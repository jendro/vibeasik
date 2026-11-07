package models

import (
	"time"
)

// ScanResult is returned by the scanner before a project is registered or during re-scan
type ScanResult struct {
	RootPath       string              `json:"root_path"`
	SuggestedName  string              `json:"suggested_name"`
	SuggestedSlug  string              `json:"suggested_slug"`
	ProjectType    string              `json:"project_type"`
	GitBranch      string              `json:"git_branch"`
	GitRemote      string              `json:"git_remote"`
	DetectedStack  []string            `json:"detected_stack"`
	Components     []ComponentProposal `json:"components"`
	TotalFiles     int                 `json:"total_files"`
	IssuesDetected []Finding           `json:"issues_detected"`
	DirectoryTree  string              `json:"directory_tree"`
}

// ComponentProposal represents a proposed component detected in a directory
type ComponentProposal struct {
	Name           string `json:"name"`
	Role           string `json:"role"`
	RelativePath   string `json:"relative_path"`
	AbsolutePath   string `json:"absolute_path"`
	Language       string `json:"language"`
	Framework      string `json:"framework"`
	PackageManager string `json:"package_manager"`
	DevCommand     string `json:"dev_command"`
	BuildCommand   string `json:"build_command"`
	Port           int    `json:"port"`
	HealthURL      string `json:"health_url"`
	EnvFile        string `json:"env_file"`
	EnvSummary     string `json:"env_summary"`
}

// ArchitectureInfo represents detected framework, plugins/libraries, and design patterns
type ArchitectureInfo struct {
	Framework      string   `json:"framework"`       // e.g. "Fiber v2", "Gin", "Echo", "Chi", "net/http"
	FrameworkType  string   `json:"framework_type"`  // e.g. "High-Performance Asynchronous HTTP Engine"
	Plugins        []string `json:"plugins"`         // Detected plugins, middlewares, and key packages
	DesignPattern  string   `json:"design_pattern"`  // e.g. "MVC (Model-Controller Architecture)", "Clean Architecture"
	PatternDetails string   `json:"pattern_details"` // Detailed explanation of the architectural pattern
}

// FastScanResult represents the output of an AI fast scan for an application component
type FastScanResult struct {
	ComponentID            uint                   `json:"component_id"`
	ComponentName          string                 `json:"component_name"`
	Role                   string                 `json:"role"`
	RelativePath           string                 `json:"relative_path"`
	AbsolutePath           string                 `json:"absolute_path"`
	DirectoryTree          string                 `json:"directory_tree"`
	FolderStructureSummary string                 `json:"folder_structure_summary"`
	HowToRun               HowToRunInfo           `json:"how_to_run"`
	ExtractedEnv           ExtractedEnvInfo       `json:"extracted_env"`
	DetectedParams         DetectedParams         `json:"detected_params"`
	Routes                 []RouteInfo            `json:"routes"`
	Architecture           ArchitectureInfo       `json:"architecture"`
	AiAnalysisMarkdown     string                 `json:"ai_analysis_markdown"`
	SecurityAudit          map[string]interface{} `json:"security_audit"`
	ScannedAt              time.Time              `json:"scanned_at"`
}

// ExtractedEnvInfo contains environment, database, authentication, and dynamic attributes
type ExtractedEnvInfo struct {
	DatabaseType        string              `json:"database_type"`          // PostgreSQL, MySQL, SQLite, MongoDB, Redis, etc.
	ORMDriver           string              `json:"orm_driver"`             // GORM, Prisma, Drizzle, TypeORM, SQLx, etc.
	DatabaseName        string              `json:"database_name"`          // e.g., myapp_db
	AuthMethods         []string            `json:"auth_methods"`           // JWT, Session, OAuth2 (Google/GitHub), Firebase, API Key, etc.
	LoginEndpoints      []string            `json:"login_endpoints"`        // /api/auth/login, /api/login, etc.
	ThirdPartyServices  []DetectedService   `json:"third_party_services"`   // S3, Stripe, OpenAI, Redis, Mailgun, etc.
	EnvironmentKeys     []EnvKeyDescription `json:"environment_keys"`       // Key, Purpose, Required, AliasedExample
	LoginAndAccessGuide string              `json:"login_and_access_guide"` // Step-by-step summary on how to authenticate/login
	DynamicMetadata     map[string]string   `json:"dynamic_metadata"`       // Flexible dynamic key-value pairs extracted by LLM
}

// DetectedService represents third-party APIs or infrastructure components
type DetectedService struct {
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
}

// EnvKeyDescription represents an individual environment variable extracted
type EnvKeyDescription struct {
	Key         string `json:"key"`
	Purpose     string `json:"purpose"`
	Required    bool   `json:"required"`
	IsSecret    bool   `json:"is_secret"`
	SafeExample string `json:"safe_example"`
}

// HowToRunInfo contains structured run instructions
type HowToRunInfo struct {
	Prerequisites  []string `json:"prerequisites"`
	InstallCommand string   `json:"install_command"`
	DevCommand     string   `json:"dev_command"`
	BuildCommand   string   `json:"build_command"`
	TestCommand    string   `json:"test_command"`
	Steps          []string `json:"steps"`
}

// DetectedParams contains parameters auto-detected by LLM
type DetectedParams struct {
	Language       string `json:"language"`
	Framework      string `json:"framework"`
	PackageManager string `json:"package_manager"`
	DevCommand     string `json:"dev_command"`
	BuildCommand   string `json:"build_command"`
	Port           int    `json:"port"`
}

// ScanGroupProposal represents a proposed project grouping for scanned components
type ScanGroupProposal struct {
	ID                         string `json:"id"`
	Name                       string `json:"name"`
	Slug                       string `json:"slug"`
	ProjectType                string `json:"project_type"`
	Description                string `json:"description"`
	Reason                     string `json:"reason"`
	MatchedExistingProjectID   uint   `json:"matched_existing_project_id"`
	MatchedExistingProjectName string `json:"matched_existing_project_name"`
	ComponentIndices           []int  `json:"component_indices"`
}

// ScanGroupingResult is the output of AI/heuristic grouping of scanned components
type ScanGroupingResult struct {
	Groups  []ScanGroupProposal `json:"groups"`
	LlmUsed bool                `json:"llm_used"`
	Summary string              `json:"summary"`
}

// GroupScanRequest is the payload to request AI grouping of components
type GroupScanRequest struct {
	RootPath   string              `json:"root_path"`
	Components []ComponentProposal `json:"components"`
}
