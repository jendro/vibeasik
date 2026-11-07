import type { LLMPlannerTodoItem } from './base_knowledge'

export interface ProjectComponent {
  id: number
  project_id: number
  name: string
  role: 'backend' | 'frontend_admin' | 'frontend_client' | 'frontend_mobile' | 'cli' | 'worker' | 'docs' | 'other' | string
  relative_path: string
  absolute_path: string
  language: string
  framework: string
  package_manager: string
  dev_command: string
  build_command: string
  port: number
  run_port?: number
  health_url: string
  health_score?: number
  env_file: string
  env_summary: string
  fast_scan_data?: string
  created_at?: string
  updated_at?: string
}

export interface ComponentDraft {
  name: string
  role: string
  absolute_path: string
  port: number | null
  language: string
  framework: string
  dev_command: string
}

export interface Diagnostic {
  id: number
  project_id: number
  component_id?: number
  summary: string
  score: number
  findings_json: string
  optimizations: string
  structure_tree: string
  created_at: string
}

export interface Project {
  id: number
  name: string
  slug?: string
  short_description?: string
  description: string
  root_path: string
  project_type: 'bundle' | 'multi_fe' | 'monorepo' | 'single_app' | 'microservice' | string
  status: 'active' | 'hold' | 'done' | 'deprecated' | 'paused' | 'archived' | string
  git_branch: string
  git_remote?: string
  tags: string
  health_score: number
  is_unified?: boolean
  autonomous?: boolean
  deprecated_at?: string
  deprecated_reason?: string
  last_deep_check_at?: string
  last_deep_check_score?: number
  created_at?: string
  updated_at?: string
  components: ProjectComponent[]
  diagnostics?: Diagnostic[]
}

export interface ExecutiveKnowledgeItem {
  id: number
  executive_id: number
  project_id: number
  topic: string
  content: string
  kind: 'app_reality' | 'product_vision' | 'roadmap' | 'technical_decision' | 'market' | 'forecast' | 'user_feedback' | 'lesson' | string
  source: 'seed' | 'todo_completed' | 'meeting' | 'profile' | 'manual' | 'chat' | 'autonomous' | 'market_research' | 'qa_review' | string
  weight: number
  source_todo_id?: number
  source_session_id?: number
  status: 'active' | 'archived'
  created_at: string
  updated_at?: string
}

export interface ExecutiveTodoAnalysisResult {
  analysis: string
  summary?: string
  model: string
  total_todos: number
  pending_count: number
  suggested_todos?: LLMPlannerTodoItem[]
  actions?: {
    reprioritize?: Array<{
      todo_id: number
      title: string
      todo_title?: string
      old_priority: string
      new_priority: string
      rationale: string
      reason?: string
    }>
    pause?: Array<{
      todo_id: number
      title: string
      todo_title?: string
      rationale: string
      reason?: string
    }>
    resume?: Array<{
      todo_id: number
      title: string
      todo_title?: string
      rationale: string
      reason?: string
    }>
    delete?: Array<{
      todo_id: number
      title: string
      todo_title?: string
      rationale: string
      reason?: string
    }>
    complete?: Array<{
      todo_id: number
      title: string
      todo_title?: string
      rationale: string
      reason?: string
    }>
  }
}

export interface ActiveExecutiveMeeting {
  project_id: number
  project_name: string
  status: 'preparing' | 'meeting' | 'deliberating' | 'formulating' | 'done' | string
  step: string
  topic: string
  started_at: string
  invited_employee_ids?: number[]
  invited_employees?: any[]
}

export interface ExecutiveAutonomousStatus {
  project_id: number
  autonomous: boolean
  last_run_at?: string
  next_eligible_at?: string
  total_auto_todos: number
  total_notes: number
  active_meeting?: ActiveExecutiveMeeting | null
}

export interface CreateProjectPayload {
  name: string
  short_description?: string
  description?: string
  root_path?: string
  status?: string
  project_type?: string
  components?: Partial<ProjectComponent>[]
}

export interface DirectoryItem {
  name: string
  path: string
  is_dir?: boolean
}

export interface BrowseDirectoryResponse {
  current_path: string
  parent_path: string
  directories: DirectoryItem[]
  quick_paths?: DirectoryItem[]
}

export interface ScanResult {
  root_path: string
  suggested_name: string
  suggested_slug: string
  project_type: string
  git_branch: string
  git_remote: string
  detected_stack: string[]
  components: Omit<ProjectComponent, 'id' | 'project_id'>[]
  directory_tree: string
}

export type RoleBadgeColor = 'info' | 'secondary' | 'success' | 'warning' | 'neutral'

export interface ScanGroupProposal {
  id: string
  name: string
  slug?: string
  project_type: string
  description?: string
  reason?: string
  matched_existing_project_id?: number
  matched_existing_project_name?: string
  component_indices: number[]
}

export interface ScanGroupingResult {
  groups: ScanGroupProposal[]
  llm_used: boolean
  summary: string
}

export interface ManualComponentItem {
  original_index: number
  selected: boolean
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
  health_url: string
  env_file: string
  env_summary: string
}

export interface ManualProjectGroup {
  id: string
  is_existing: boolean
  existing_project_id?: number
  name: string
  project_type: string
  reason?: string
  components: ManualComponentItem[]
}

export interface WorkspaceSettings {
  id?: number
  default_projects_root: string
  updated_at?: string
}

export interface ConsolidateMoveItem {
  id?: string
  component_id?: number
  component_ids?: number[]
  component_name?: string
  role?: string
  current_path: string
  target_path: string
  relative_target_path: string
  is_git_repo?: boolean
  has_uncommitted_changes?: boolean
  status: 'pending' | 'already_in_place' | 'moves_with_parent' | 'success' | 'conflict' | 'failed' | string
  error?: string
  warning?: string
  size_bytes?: number
}

export interface ConsolidatePreview {
  project_id: number
  project_name: string
  project_slug: string
  is_unified: boolean
  target_parent_dir: string
  project_target_dir: string
  items: ConsolidateMoveItem[]
  has_conflict: boolean
  can_consolidate: boolean
  total_size_bytes?: number
}

export interface ConsolidateResult {
  success: boolean
  message: string
  project_id: number
  project_name: string
  project_root: string
  is_unified: boolean
  moved_count: number
  items: ConsolidateMoveItem[]
  linked_files?: string[]
  code_map_markdown?: string
  warnings?: string[]
}
