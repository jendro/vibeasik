/** Agent Project Manager: rapat perencanaan dengan spesialis (GET /api/pm/*). */

export type PlanningStatus = 'queued' | 'running' | 'completed' | 'failed'
export type PlanningTrigger = 'manual' | 'run_completed' | 'agent_request' | 'todo_suggest'
export type PlannerRoleKey = 'pm' | 'business' | 'design' | 'architect' | 'security' | 'qa' | 'devops'

export interface PlanningRequest {
  source: PlanningTrigger
  requester?: string
  note?: string
  todo_id?: number
  run_id?: number
  at: string
}

export interface PlanningProposal {
  title: string
  description?: string
  category?: string
  priority?: string
  rationale?: string
}

export interface PlanningOpinion {
  role: PlannerRoleKey
  role_name: string
  question?: string
  model?: string
  opinion?: string
  concerns?: string[] | null
  proposals?: PlanningProposal[] | null
  new_features?: boolean | null
  error?: string
  duration_ms: number
}

export interface PlanningDecisionTodo {
  title: string
  category: string
  priority: string
  proposed_by?: string[] | null
  assign_hint?: string
  todo_id?: number
  outcome: 'created' | 'duplicate' | 'assigned' | 'failed'
  run_id?: number
  assign_error?: string
}

export interface PlanningRejected {
  title: string
  reason: string
  kind?: 'duplicate' | 'fulfilled' | 'invalid' | 'deferred' | 'low_value'
  todo_id?: number
  /** Saran lama sudah ditandai rejected oleh PM */
  applied: boolean
  note?: string
}

export interface PlanningSession {
  id: number
  project_id: number
  project_name: string
  status: PlanningStatus
  trigger: PlanningTrigger
  requests: PlanningRequest[] | null
  step?: string
  pm_model?: string
  situation?: string
  add_features: boolean
  feature_decision?: string
  summary?: string
  lookups: Array<{ tool: string, query: string, result: string }> | null
  opinions: PlanningOpinion[] | null
  decisions: PlanningDecisionTodo[] | null
  rejected: PlanningRejected[] | null
  todos_created: number
  /** Saran lama yang ditandai rejected oleh PM (duplikat / sudah terpenuhi / tidak valid) */
  todos_rejected: number
  todos_assigned: number
  error?: string
  started_at?: string
  finished_at?: string
  created_at: string
}

export interface PlannerSettings {
  id?: number
  enabled: boolean
  /** "auto" atau "<config_id>:<model>" */
  pm_model: string
  min_intelligence: number
  /** CSV key spesialis, kosong = semua */
  specialists: string
  max_specialists: number
  max_todos: number
  review_after_run: boolean
  auto_assign: boolean
}

/** Satu model LLM yang bisa dipakai rapat, beserta taksiran kecerdasannya. */
export interface PlannerBrain {
  config_id: number
  config: string
  model: string
  score: number
  tier: 'frontier' | 'tinggi' | 'menengah' | 'ringan'
}

export interface PlannerRole {
  key: PlannerRoleKey
  name: string
  focus: string
}

export interface PlannerOverview {
  settings: PlannerSettings
  brains: PlannerBrain[] | null
  lineup?: { lead: PlannerBrain, backup: PlannerBrain[] | null, pool: PlannerBrain[] | null }
  lineup_error?: string
  roles: PlannerRole[]
  active: PlanningSession[] | null
  latest?: PlanningSession
}

/** Bagian `planning` di GET /api/agents/office. */
export interface OfficePlanningState {
  enabled: boolean
  active: PlanningSession[] | null
  latest?: PlanningSession
}
