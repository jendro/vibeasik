package models

import "time"

// ProjectProfile is the evidence-based "dossier" of a project: why it exists, which apps
// make it up, which features it offers (merged across apps), and where it is heading.
// It is produced by the LLM when available, with a deterministic heuristic fallback.
type ProjectProfile struct {
	ProjectID        uint                 `json:"project_id"`
	ShortDescription string               `json:"short_description"`
	FullDescription  string               `json:"full_description"`
	Purpose          ProfilePurpose       `json:"purpose"`
	Maturity         ProfileMaturity      `json:"maturity"`
	Apps             []ProfileApp         `json:"apps"`
	Features         []ProfileFeature     `json:"features"`
	Integrations     []ProfileIntegration `json:"integrations"`
	DataEntities     []ProfileDataEntity  `json:"data_entities"`
	ExternalServices []ProfileExternalSvc `json:"external_services"`
	Roadmap          []ProfileRoadmapItem `json:"roadmap"`
	Risks            []ProfileRisk        `json:"risks"`
	OpenQuestions    []string             `json:"open_questions"`
	Confidence       float64              `json:"confidence"`
	Stats            ProfileStats         `json:"stats"`
	Source           string               `json:"source"` // llm | heuristic
	Model            string               `json:"model,omitempty"`
	LLMError         string               `json:"llm_error,omitempty"`
	GeneratedAt      time.Time            `json:"generated_at"`
}

type ProfilePurpose struct {
	Problem          string   `json:"problem"`
	Solution         string   `json:"solution"`
	TargetUsers      []string `json:"target_users"`
	ValueProposition string   `json:"value_proposition"`
}

type ProfileMaturity struct {
	Stage  string `json:"stage"` // idea | prototype | mvp | beta | production
	Reason string `json:"reason"`
}

type ProfileApp struct {
	ComponentID      uint              `json:"component_id"`
	ComponentName    string            `json:"component_name"`
	Role             string            `json:"role"`
	Summary          string            `json:"summary"`
	Responsibilities []string          `json:"responsibilities"`
	KeyPages         []ProfileRouteRef `json:"key_pages"`
	KeyEndpoints     []ProfileRouteRef `json:"key_endpoints"`
	TalksTo          []string          `json:"talks_to"`
}

type ProfileRouteRef struct {
	Method  string `json:"method,omitempty"`
	Path    string `json:"path"`
	Purpose string `json:"purpose"`
}

type ProfileFeature struct {
	Name        string   `json:"name"`
	Domain      string   `json:"domain"`
	Description string   `json:"description"`
	Status      string   `json:"status"` // done | partial | planned
	Apps        []string `json:"apps"`
	Pages       []string `json:"pages"`
	Endpoints   []string `json:"endpoints"`
	Tables      []string `json:"tables"`
	Evidence    []string `json:"evidence"`
}

type ProfileIntegration struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Via     string `json:"via"`
	Purpose string `json:"purpose"`
}

type ProfileDataEntity struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	UsedBy      []string `json:"used_by"`
}

type ProfileExternalSvc struct {
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
}

type ProfileRoadmapItem struct {
	Title           string   `json:"title"`
	Horizon         string   `json:"horizon"` // now | next | later
	Rationale       string   `json:"rationale"`
	RelatedFeatures []string `json:"related_features"`
}

// Risk handling status constants
const (
	RiskStatusOpen       = "open"        // belum ada tindak lanjut
	RiskStatusPlanned    = "planned"     // sudah dijadikan todo, belum dikerjakan
	RiskStatusInProgress = "in_progress" // todo-nya sedang dikerjakan
	RiskStatusHandled    = "handled"     // todo-nya selesai atau ditandai sudah ditangani
	RiskStatusAccepted   = "accepted"    // disadari & diterima pemilik, tidak ditindaklanjuti
)

type ProfileRisk struct {
	Title    string `json:"title"`
	Severity string `json:"severity"` // high | medium | low
	Detail   string `json:"detail"`
	// Penanganan: diisi pemilik (atau diturunkan dari status todo yang ditautkan), bukan oleh LLM.
	Status    string     `json:"status,omitempty"` // open | planned | in_progress | handled | accepted
	TodoID    *uint      `json:"todo_id,omitempty"`
	Note      string     `json:"note,omitempty"`
	HandledAt *time.Time `json:"handled_at,omitempty"`
}

// ProfileStats are computed from facts (never from the LLM) so the numbers stay trustworthy.
type ProfileStats struct {
	Apps       int `json:"apps"`
	LiveApps   int `json:"live_apps"`
	Pages      int `json:"pages"`
	Endpoints  int `json:"endpoints"`
	Tables     int `json:"tables"`
	Domains    int `json:"domains"`
	OpenTodos  int `json:"open_todos"`
	DoneTodos  int `json:"done_todos"`
	TotalFiles int `json:"total_files"`
}

// ProjectProfilePrompt exposes the exact prompts so they can be reused by external agents.
type ProjectProfilePrompt struct {
	SystemPrompt string `json:"system_prompt"`
	UserPrompt   string `json:"user_prompt"`
	AgentPrompt  string `json:"agent_prompt"`
	EvidenceSize int    `json:"evidence_size"`
}
