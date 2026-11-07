package models

import "time"

// ProjectBaseKnowledge represents the comprehensive, accurate knowledge base
// of a project bundle, capturing its applications, files, routers, database, and stack.
type ProjectBaseKnowledge struct {
	ProjectID        uint                      `json:"project_id"`
	ProjectName      string                    `json:"project_name"`
	Slug             string                    `json:"slug"`
	ProjectType      string                    `json:"project_type"`
	RootPath         string                    `json:"root_path"`
	GitBranch        string                    `json:"git_branch"`
	GitRemote        string                    `json:"git_remote"`
	Status           string                    `json:"status"`
	HealthScore      int                       `json:"health_score"`
	AppsCount        int                       `json:"apps_count"`
	Apps             []BaseKnowledgeApp        `json:"apps"`
	FilesScope       BaseKnowledgeFilesScope   `json:"files_scope"`
	RoutersScope     BaseKnowledgeRoutersScope `json:"routers_scope"`
	EnvironmentScope BaseKnowledgeEnvScope     `json:"environment_scope"`
	SummaryMarkdown  string                    `json:"summary_markdown"`
	CompiledAt       time.Time                 `json:"compiled_at"`
}

// BaseKnowledgeApp summarizes each application / component participating in the project
type BaseKnowledgeApp struct {
	ID             uint     `json:"id"`
	Name           string   `json:"name"`
	Role           string   `json:"role"` // backend, frontend_client, frontend_admin, frontend_mobile, worker, cli, etc.
	RelativePath   string   `json:"relative_path"`
	AbsolutePath   string   `json:"absolute_path"`
	Language       string   `json:"language"`
	Framework      string   `json:"framework"`
	PackageManager string   `json:"package_manager"`
	DevCommand     string   `json:"dev_command"`
	BuildCommand   string   `json:"build_command"`
	Port           int      `json:"port"`
	RunPort        int      `json:"run_port"`
	IsPortLive     bool     `json:"is_port_live"`
	HealthURL      string   `json:"health_url"`
	RoutesCount    int      `json:"routes_count"`
	KeyFiles       []string `json:"key_files"`
	EnvSummary     string   `json:"env_summary"`
}

// BaseKnowledgeFilesScope captures filesystem metrics and structural hierarchy
type BaseKnowledgeFilesScope struct {
	TotalFiles     int            `json:"total_files"`
	DirectoryTree  string         `json:"directory_tree"`
	KeyFiles       []string       `json:"key_files"`
	ConfigFiles    []string       `json:"config_files"`
	FileTypesCount map[string]int `json:"file_types_count"`
}

// BaseKnowledgeRoutersScope captures all API routes, page routes, handlers, and functional domains
type BaseKnowledgeRoutersScope struct {
	TotalRoutes        int                  `json:"total_routes"`
	BackendRoutesCount int                  `json:"backend_routes_count"`
	FrontendPagesCount int                  `json:"frontend_pages_count"`
	DomainGroups       map[string]int       `json:"domain_groups"`
	Routes             []BaseKnowledgeRoute `json:"routes"`
}

// BaseKnowledgeRoute represents an individual endpoint or page route
type BaseKnowledgeRoute struct {
	ComponentID    uint     `json:"component_id"`
	ComponentName  string   `json:"component_name"`
	Method         string   `json:"method"` // GET, POST, PUT, DELETE, PAGE, etc.
	Path           string   `json:"path"`
	Handler        string   `json:"handler"`
	Middleware     string   `json:"middleware"`
	Description    string   `json:"description"`
	Domain         string   `json:"domain"`
	FunctionGroup  string   `json:"function_group"`
	TargetRequests []string `json:"target_requests,omitempty"`
}

// BaseKnowledgeEnvScope captures database, auth, and environment parameters
type BaseKnowledgeEnvScope struct {
	DatabaseType       string              `json:"database_type"`
	ORMDriver          string              `json:"orm_driver"`
	DatabaseName       string              `json:"database_name"`
	AuthMethods        []string            `json:"auth_methods"`
	LoginEndpoints     []string            `json:"login_endpoints"`
	ThirdPartyServices []string            `json:"third_party_services"`
	DetectedEnvKeys    []EnvKeyDescription `json:"detected_env_keys"`
}
