export interface SynergyPageTransition {
  target_path: string
  target_label: string
  trigger: string
}

export interface SynergyPayloadInfo {
  params?: string
  query?: string
  body?: string
  headers?: string
}

export interface SynergyDatabaseOp {
  engine: string
  operation_type: 'SELECT' | 'INSERT' | 'UPDATE' | 'DELETE' | 'MULTI' | string
  target_tables: string[]
  stored_fields: string[]
  query_detail: string
}

export interface SynergyResponseInfo {
  status_code: number
  data_type: string
  body_sample: string
}

export interface SynergyRequestFlow {
  id: string
  action_title: string
  trigger_event: string
  backend_component: string
  http_method: string
  route_path: string
  handler_function: string
  middleware: string
  request_payload: SynergyPayloadInfo
  validation_rules: string[]
  business_process: string[]
  database_operation: SynergyDatabaseOp
  response_spec: SynergyResponseInfo
  ui_impact: string
}

export interface SynergyFrontendPage {
  id: string
  component_name: string
  page_path: string
  title: string
  file_path: string
  navigation_targets: SynergyPageTransition[]
  outbound_requests: SynergyRequestFlow[]
}

export interface SynergyBackendNode {
  component_name: string
  port: number
  routes_count: number
  framework: string
}

export interface SynergyDatabaseTable {
  table_name: string
  fields: string[]
  related_apps: string[]
}

export interface ProjectSynergyMap {
  project_id: number
  project_name: string
  compiled_at: string
  summary: string
  total_pages: number
  total_flows: number
  total_tables: number
  pages: SynergyFrontendPage[]
  backend_nodes: SynergyBackendNode[]
  database_map: SynergyDatabaseTable[]
  llm_enriched?: boolean
}
