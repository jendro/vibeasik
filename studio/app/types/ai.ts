export interface HowToRunInfo {
  prerequisites: string[]
  install_command: string
  dev_command: string
  build_command: string
  test_command: string
  steps: string[]
}

export interface DetectedParams {
  language: string
  framework: string
  package_manager: string
  dev_command: string
  build_command: string
  port: number
}

export interface RedactedRecord {
  category: string
  alias: string
  masked_preview: string
}

export interface SecurityAudit {
  redacted_count: number
  custom_matches?: number
  built_in_matches?: number
  records: RedactedRecord[]
  message: string
  safe_preview?: string
  double_security_level?: string
}

export interface DetectedService {
  name: string
  purpose: string
  config_key?: string
}

export interface EnvKeyDescription {
  key: string
  purpose: string
  required: boolean
  is_secret: boolean
  safe_example: string
}

export interface ExtractedEnvInfo {
  database_type: string
  orm_driver: string
  database_name: string
  auth_methods: string[]
  login_endpoints: string[]
  third_party_services: DetectedService[]
  environment_keys: EnvKeyDescription[]
  login_and_access_guide: string
  dynamic_metadata: Record<string, string>
}

export interface RouteInfo {
  method: string
  path: string
  handler: string
  description: string
  middleware?: string
  target_requests?: string[]
}

export interface ComponentRoute {
  id: number
  component_id: number
  method: string
  path: string
  handler: string
  middleware?: string
  description: string
  domain: string
  function_group: string
  target_requests?: string
  tags?: string
  created_at?: string
  updated_at?: string
}

export interface RouteSyncResult {
  component_id: number
  added_count: number
  updated_count: number
  deleted_count: number
  total_count: number
  routes: ComponentRoute[]
}

export interface ArchitectureInfo {
  framework: string
  framework_type: string
  plugins: string[]
  design_pattern: string
  pattern_details: string
}

export interface FastScanResult {
  component_id: number
  component_name: string
  role: string
  relative_path: string
  absolute_path: string
  directory_tree: string
  folder_structure_summary: string
  how_to_run: HowToRunInfo
  extracted_env: ExtractedEnvInfo
  detected_params: DetectedParams
  routes?: RouteInfo[]
  architecture?: ArchitectureInfo
  ai_analysis_markdown: string
  security_audit: SecurityAudit
  scanned_at: string
}

export interface ProjectFastScanResponse {
  project_id: number
  project_name: string
  components: FastScanResult[]
  scanned_at: string
}

export interface AIReviewResponse {
  project_name: string
  architecture_type: string
  summary: string
  strengths: string[]
  weaknesses: string[]
  action_plan: {
    target_component: string
    title: string
    description: string
    priority: 'P0' | 'P1' | 'P2'
  }[]
  suggested_features: string[]
  ai_agent_prompt: string
  generated_at: string
}
