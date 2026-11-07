export type AgentRunStatus = 'queued' | 'running' | 'completed' | 'failed' | 'cancelled'
export type AgentRunKind = 'task' | 'merge'
export type MergeState = 'pending' | 'merging' | 'merged' | 'failed' | 'no_changes' | 'kept' | 'qa_review' | 'qa_rework'
export type RouterStrategy = 'rules' | 'jev' | 'llm'

/** Satu "kursi pekerja" orkestrator: CLI agent + model + effort + akun. */
export interface AgentProfile {
  id: number
  name: string
  agent_key: string
  model: string
  effort: string
  account_ref: string
  account_label: string
  autonomy: 'full' | 'edits'
  strengths: string
  notes: string
  sort_order: number
  max_concurrent: number
  daily_limit: number
  enabled: boolean
  cooldown_until?: string
  last_error?: string
  last_used_at?: string
  created_at: string
  updated_at: string
}

export interface AgentProfileView extends AgentProfile {
  running: number
  today_runs: number
  available: boolean
  unavailable_reason?: string
}

export type AgentProfileForm = Omit<AgentProfile, 'id' | 'created_at' | 'updated_at' | 'cooldown_until' | 'last_error' | 'last_used_at'>

export interface AgentCatalogEntry {
  key: string
  name: string
  installed: boolean
  path: string
  models: string[] | null
  efforts: string[] | null
  account_kind: '' | 'agy' | 'claude'
  model_hint: string
  is_custom: boolean
}

export interface OrchestratorSettings {
  id?: number
  router: RouterStrategy
  max_concurrent: number
  default_timeout_min: number
  cooldown_min: number
  auto_rotate: boolean
  auto_complete_todo: boolean
  extra_instructions: string
  /** Mode otomatis hanya memakai agent ini selama punya profil aktif (default 'agy') */
  preferred_agent: string
  /** Skor kerumitan minimal agar boleh dikerjakan agent lain; 0 = tidak pernah */
  escalate_min_score: number
  adaptive_effort: boolean
  /** Run di repositori yang sama boleh paralel bila berkas targetnya diketahui dan berbeda */
  parallel_same_dir: boolean
  /** worktree: tiap todo di branch sendiri lalu di-merge agent lain; shared: langsung di checkout utama */
  isolation: 'worktree' | 'shared'
  /** Profil khusus agent merge (kosong = dipilih router) */
  merge_profile_id?: number | null
  /** Tiap run membuka jendela terminal di desktop laptop engine yang menampilkan log live (juga run dari LAN) */
  show_terminal?: boolean
  /** Sebelum dispatch, todo pending yang serupa dilebur menjadi satu todo gabungan (Aturan 43) */
  merge_similar?: boolean
}

export interface OrchestratorOverview {
  settings: OrchestratorSettings
  profiles: AgentProfileView[]
  catalog: AgentCatalogEntry[]
  stats: Partial<Record<AgentRunStatus, number>>
}

export interface AgyAccount {
  id: string
  name: string
  email_masked: string
  status: string
  tier: string
  active: boolean
  ready: boolean
}

export interface ClaudeAccount {
  ref: string
  label: string
}

export interface AgentRun {
  id: number
  todo_id?: number
  project_id?: number
  component_id?: number
  project_name?: string
  work_dir: string
  /** Berkas target todo (path absolut, dipisah baris baru) */
  target_files?: string
  title: string
  prompt?: string
  category?: string
  priority?: string
  status: AgentRunStatus
  requested_profile_id?: number
  pool_agent_key?: string
  router: string
  profile_id?: number
  profile_name?: string
  agent_key?: string
  model?: string
  effort?: string
  routed_by?: string
  route_reason?: string
  complexity_level?: string
  complexity_score: number
  timeout_min: number
  attempts: number
  output?: string
  result_text?: string
  exit_code: number
  error?: string
  quota_hit: boolean
  created_at: string
  started_at?: string
  finished_at?: string
  duration_ms: number
  /** Alasan run masih antre (dari penjadwal engine) */
  wait_reason?: string
  /** task = agent mengerjakan todo; merge = agent lain menggabungkan & memeriksa branch run tugas */
  kind?: AgentRunKind
  parent_run_id?: number
  repo_root?: string
  /** Worktree tempat agent bekerja (run terisolasi) */
  exec_dir?: string
  branch?: string
  base_branch?: string
  base_commit?: string
  merge_state?: MergeState
  merge_run_id?: number
}

export interface AgentDispatchRequest {
  todo_id?: number
  project_id?: number
  component_id?: number
  work_dir?: string
  title?: string
  prompt?: string
  instructions?: string
  profile_id?: number
  pool_agent_key?: string
  router?: RouterStrategy | ''
  timeout_min?: number
}

export interface RouteDecision {
  profile_id: number
  profile_name: string
  by: string
  reason: string
}

export interface AgentBatchDispatchRequest {
  todo_ids?: number[]
  project_id?: number
  pool_agent_key?: string
  router?: RouterStrategy | ''
  instructions?: string
  timeout_min?: number
}

export interface AgentBatchItem {
  todo_id: number
  title: string
  run?: AgentRun
  reason?: string
}
