export interface BaseKnowledgeApp {
  id: number
  name: string
  role: string
  relative_path: string
  absolute_path: string
  language: string
  framework: string
  package_manager: string
  dev_command: string
  build_command: string
  port: number
  is_port_live: boolean
  health_url: string
  routes_count: number
  key_files: string[]
  env_summary: string
}

export interface BaseKnowledgeFilesScope {
  total_files: number
  directory_tree: string
  key_files: string[]
  config_files: string[]
  file_types_count: Record<string, number>
}

export interface BaseKnowledgeRoute {
  component_id: number
  component_name: string
  method: string
  path: string
  handler: string
  middleware: string
  description: string
  domain: string
  function_group: string
  target_requests?: string[]
}

export interface BaseKnowledgeRoutersScope {
  total_routes: number
  backend_routes_count: number
  frontend_pages_count: number
  domain_groups: Record<string, number>
  routes: BaseKnowledgeRoute[]
}

export interface BaseKnowledgeEnvKey {
  key: string
  purpose: string
  required: boolean
  is_secret: boolean
  safe_example: string
}

export interface BaseKnowledgeEnvScope {
  database_type: string
  orm_driver: string
  database_name: string
  auth_methods: string[]
  login_endpoints: string[]
  third_party_services: string[]
  detected_env_keys: BaseKnowledgeEnvKey[]
}

export interface ProjectBaseKnowledge {
  project_id: number
  project_name: string
  slug: string
  project_type: string
  root_path: string
  git_branch: string
  git_remote: string
  status: string
  health_score: number
  apps_count: number
  apps: BaseKnowledgeApp[]
  files_scope: BaseKnowledgeFilesScope
  routers_scope: BaseKnowledgeRoutersScope
  environment_scope: BaseKnowledgeEnvScope
  summary_markdown: string
  compiled_at: string
}

export interface LLMPlannerTodoItem {
  title: string
  category: string
  level: string
  target_apps: string
  priority: string
  description: string
  rationale: string
  guideline_ref: string
  action_decision?: 'do' | 'defer' | 'skip' | string
  decision_reason?: string
}

export interface ProjectChatMessageDTO {
  id: number
  project_id: number
  role: 'user' | 'assistant' | 'system'
  content: string
  todos?: LLMPlannerTodoItem[]
  model_name?: string
  created_at: string
}

export interface ProjectChatResponse {
  reply: string
  analysis: string
  todos: LLMPlannerTodoItem[]
  model: string
  message_id: number
}
