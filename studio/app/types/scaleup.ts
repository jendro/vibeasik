export interface FolderAnnotation {
  path: string
  role: string
  description: string
}

export interface CodeSnippet {
  language: string
  title: string
  code: string
  explanation: string
}

export interface ArchitecturePattern {
  id: string
  name: string
  badge: string
  short_desc: string
  philosophy: string
  router_characteristics: string
  controller_characteristics: string
  function_vs_class_philosophy: string
  mcp_integration: string
  mvpp_role: string
  complexity_score: number // 1 - 5
  testability_score: number // 1 - 5
  ai_readiness_score: number // 1 - 5
  scalability_score: number // 1 - 5
  pros: string[]
  cons: string[]
  when_to_use: string[]
  folder_blueprint_tree: string
  folder_explanation: FolderAnnotation[]
  router_code_sample: CodeSnippet
  controller_code_sample: CodeSnippet
  logic_code_sample: CodeSnippet
}

export interface ScaleUpAction {
  phase: string
  title: string
  description: string
  target_dir: string
  priority: 'urgent' | 'high' | 'medium' | 'low'
}

export interface EvaluationMetrics {
  router_score: number // 0 - 25
  controller_score: number // 0 - 25
  service_score: number // 0 - 25
  ai_mcp_score: number // 0 - 25
}

export interface EvaluationTraits {
  has_modular_routing: boolean
  has_thin_controllers: boolean
  has_service_layer: boolean
  has_repository_layer: boolean
  has_pure_functions: boolean
  has_mcp_support: boolean
  has_planner_layer: boolean
  total_source_files: number
  total_routes: number
}

export interface ArchitectureEvaluation {
  component_id: number
  component_name: string
  role: string
  language: string
  framework: string
  absolute_path: string
  detected_pattern: string
  detected_pattern_name: string
  maturity_score: number // 0 - 100
  score_level: string
  metrics: EvaluationMetrics
  detected_traits: EvaluationTraits
  folder_summary: string[]
  key_findings: string[]
  scale_up_recommendations: ScaleUpAction[]
  recommended_next_pattern: string
}
