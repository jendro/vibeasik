package models

// ArchitecturePattern defines metadata, characteristics, and blueprints for backend design patterns
type ArchitecturePattern struct {
	ID                        string             `json:"id"`
	Name                      string             `json:"name"`
	Badge                     string             `json:"badge"`
	ShortDesc                 string             `json:"short_desc"`
	Philosophy                string             `json:"philosophy"`
	RouterCharacteristics     string             `json:"router_characteristics"`
	ControllerCharacteristics string             `json:"controller_characteristics"`
	FunctionVsClassPhilosophy string             `json:"function_vs_class_philosophy"`
	McpIntegration            string             `json:"mcp_integration"`
	MvppRole                  string             `json:"mvpp_role"`
	ComplexityScore           int                `json:"complexity_score"`   // 1 - 5
	TestabilityScore          int                `json:"testability_score"`  // 1 - 5
	AiReadinessScore          int                `json:"ai_readiness_score"` // 1 - 5
	ScalabilityScore          int                `json:"scalability_score"`  // 1 - 5
	Pros                      []string           `json:"pros"`
	Cons                      []string           `json:"cons"`
	WhenToUse                 []string           `json:"when_to_use"`
	FolderBlueprintTree       string             `json:"folder_blueprint_tree"`
	FolderExplanation         []FolderAnnotation `json:"folder_explanation"`
	RouterCodeSample          CodeSnippet        `json:"router_code_sample"`
	ControllerCodeSample      CodeSnippet        `json:"controller_code_sample"`
	LogicCodeSample           CodeSnippet        `json:"logic_code_sample"`
}

// FolderAnnotation explains specific directories in a folder blueprint
type FolderAnnotation struct {
	Path        string `json:"path"`
	Role        string `json:"role"`
	Description string `json:"description"`
}

// CodeSnippet represents code examples in Go or TypeScript
type CodeSnippet struct {
	Language    string `json:"language"`
	Title       string `json:"title"`
	Code        string `json:"code"`
	Explanation string `json:"explanation"`
}

// ArchitectureEvaluation contains analysis results of a component's backend structure
type ArchitectureEvaluation struct {
	ComponentID            uint              `json:"component_id"`
	ComponentName          string            `json:"component_name"`
	Role                   string            `json:"role"`
	Language               string            `json:"language"`
	Framework              string            `json:"framework"`
	AbsolutePath           string            `json:"absolute_path"`
	DetectedPattern        string            `json:"detected_pattern"`
	DetectedPatternName    string            `json:"detected_pattern_name"`
	MaturityScore          int               `json:"maturity_score"` // 0 - 100
	ScoreLevel             string            `json:"score_level"`    // Beginner, Intermediate, Production, Enterprise
	Metrics                EvaluationMetrics `json:"metrics"`
	DetectedTraits         EvaluationTraits  `json:"detected_traits"`
	FolderSummary          []string          `json:"folder_summary"`
	KeyFindings            []string          `json:"key_findings"`
	ScaleUpRecommendations []ScaleUpAction   `json:"scale_up_recommendations"`
	RecommendedNextPattern string            `json:"recommended_next_pattern"`
}

// EvaluationMetrics holds 4-dimensional score cards
type EvaluationMetrics struct {
	RouterScore     int `json:"router_score"`     // 0 - 25
	ControllerScore int `json:"controller_score"` // 0 - 25
	ServiceScore    int `json:"service_score"`    // 0 - 25
	AiMcpScore      int `json:"ai_mcp_score"`     // 0 - 25
}

// EvaluationTraits records binary detection flags
type EvaluationTraits struct {
	HasModularRouting  bool `json:"has_modular_routing"`
	HasThinControllers bool `json:"has_thin_controllers"`
	HasServiceLayer    bool `json:"has_service_layer"`
	HasRepositoryLayer bool `json:"has_repository_layer"`
	HasPureFunctions   bool `json:"has_pure_functions"`
	HasMcpSupport      bool `json:"has_mcp_support"`
	HasPlannerLayer    bool `json:"has_planner_layer"`
	TotalSourceFiles   int  `json:"total_source_files"`
	TotalRoutes        int  `json:"total_routes"`
}

// ScaleUpAction represents an actionable step to scale the codebase
type ScaleUpAction struct {
	Phase       string `json:"phase"` // Tahap 1, Tahap 2, dll.
	Title       string `json:"title"`
	Description string `json:"description"`
	TargetDir   string `json:"target_dir"`
	Priority    string `json:"priority"` // urgent, high, medium, low
}
