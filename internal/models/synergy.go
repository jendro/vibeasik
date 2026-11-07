package models

import "time"

// ProjectSynergyMap represents the end-to-end caller-to-callee trace across the project bundle
type ProjectSynergyMap struct {
	ProjectID    uint                   `json:"project_id"`
	ProjectName  string                 `json:"project_name"`
	CompiledAt   time.Time              `json:"compiled_at"`
	Summary      string                 `json:"summary"`
	TotalPages   int                    `json:"total_pages"`
	TotalFlows   int                    `json:"total_flows"`
	TotalTables  int                    `json:"total_tables"`
	Pages        []SynergyFrontendPage  `json:"pages"`
	BackendNodes []SynergyBackendNode   `json:"backend_nodes"`
	DatabaseMap  []SynergyDatabaseTable `json:"database_map"`
	LLMEnriched  bool                   `json:"llm_enriched"`
}

// SynergyFrontendPage represents an individual page/screen on the frontend and its outbound navigation & API requests
type SynergyFrontendPage struct {
	ID                string                  `json:"id"`
	ComponentName     string                  `json:"component_name"`     // e.g. "studio-frontend"
	PagePath          string                  `json:"page_path"`          // e.g. "/projects/:id", "/login"
	Title             string                  `json:"title"`              // e.g. "Detail Proyek & Ekosistem"
	FilePath          string                  `json:"file_path"`          // e.g. "app/pages/projects/[id]/index.vue"
	NavigationTargets []SynergyPageTransition `json:"navigation_targets"` // kemana saja user bisa berpindah dari halaman ini
	OutboundRequests  []SynergyRequestFlow    `json:"outbound_requests"`  // request API dari halaman ini ke BE
}

// SynergyPageTransition details navigation routes accessible from this page
type SynergyPageTransition struct {
	TargetPath  string `json:"target_path"`  // e.g. "/projects", "/settings"
	TargetLabel string `json:"target_label"` // e.g. "Daftar Proyek", "Pengaturan LLM"
	Trigger     string `json:"trigger"`      // e.g. "Klik tombol kembali", "Link navbar", "Redirect otomatis"
}

// SynergyRequestFlow represents an end-to-end trace from FE action -> BE route/handler -> DB -> response
type SynergyRequestFlow struct {
	ID                string              `json:"id"`
	ActionTitle       string              `json:"action_title"`       // e.g. "Muat Detail Proyek & Cek Port"
	TriggerEvent      string              `json:"trigger_event"`      // e.g. "onMounted", "Submit Form", "Klik Diagnostik"
	BackendComponent  string              `json:"backend_component"`  // e.g. "local-engine"
	HTTPMethod        string              `json:"http_method"`        // e.g. "GET", "POST", "PUT", "DELETE"
	RoutePath         string              `json:"route_path"`         // e.g. "/api/projects/:id"
	HandlerFunction   string              `json:"handler_function"`   // e.g. "s.getProject"
	Middleware        string              `json:"middleware"`         // e.g. "Public", "Auth Guard", "CORS"
	RequestPayload    SynergyPayloadInfo  `json:"request_payload"`    // params, query, body
	ValidationRules   []string            `json:"validation_rules"`   // validasi di FE dan BE
	BusinessProcess   []string            `json:"business_process"`   // tahapan alur proses di backend
	DatabaseOperation SynergyDatabaseOp   `json:"database_operation"` // tabel & data tersimpan
	ResponseSpec      SynergyResponseInfo `json:"response_spec"`      // format respon & status
	UIImpact          string              `json:"ui_impact"`          // update state di FE, modal buka, dsb.
}

// SynergyPayloadInfo documents query, params, body, and headers
type SynergyPayloadInfo struct {
	Params  string `json:"params,omitempty"`  // URL params e.g. ":id (uint)"
	Query   string `json:"query,omitempty"`   // Query params e.g. "?tab=topology"
	Body    string `json:"body,omitempty"`    // Body schema / example
	Headers string `json:"headers,omitempty"` // Custom headers / auth
}

// SynergyDatabaseOp documents database engine, operations, affected tables and stored fields
type SynergyDatabaseOp struct {
	Engine        string   `json:"engine"`         // SQLite, PostgreSQL, MySQL
	OperationType string   `json:"operation_type"` // SELECT, INSERT, UPDATE, DELETE, MULTI
	TargetTables  []string `json:"target_tables"`  // ["projects", "components", "diagnostics"]
	StoredFields  []string `json:"stored_fields"`  // ["health_score", "updated_at", "findings_json"]
	QueryDetail   string   `json:"query_detail"`   // e.g. "SELECT * FROM projects WHERE id = ? PRELOAD Components"
}

// SynergyResponseInfo describes status code and payload structure
type SynergyResponseInfo struct {
	StatusCode int    `json:"status_code"` // 200, 201, 400, 500
	DataType   string `json:"data_type"`   // JSON Object, Array, Blob
	BodySample string `json:"body_sample"` // schema / key return fields
}

// SynergyBackendNode summarizes backend application nodes
type SynergyBackendNode struct {
	ComponentName string `json:"component_name"`
	Port          int    `json:"port"`
	RoutesCount   int    `json:"routes_count"`
	Framework     string `json:"framework"`
}

// SynergyDatabaseTable maps tables and related apps
type SynergyDatabaseTable struct {
	TableName   string   `json:"table_name"`
	Fields      []string `json:"fields"`
	RelatedApps []string `json:"related_apps"`
}
