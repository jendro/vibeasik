export interface FolderMoveItem {
  id: string
  project_id: number
  project_name: string
  component_id?: number
  component_name?: string
  role?: string
  current_path: string
  target_path: string
  relative_target_path: string
  is_git_repo: boolean
  has_uncommitted_changes: boolean
  status: 'pending' | 'success' | 'skipped' | 'conflict' | 'failed'
  error?: string
  warning?: string
  size_bytes: number
}

export interface ReorgPlanRequest {
  target_directory?: string
  strategy?: 'product_centric' | 'grouped_by_role' | string
  project_ids?: number[]
}

export interface ReorgPlan {
  target_directory: string
  strategy: string
  total_items: number
  total_size: number
  planned_moves: FolderMoveItem[]
  generated_script: string
  created_at: string
}

export interface ExecuteMovePayload {
  moves: FolderMoveItem[]
  update_db: boolean
}

export interface ExecuteMoveResult {
  success_count: number
  failed_count: number
  skipped_count: number
  moves: FolderMoveItem[]
  message: string
}
