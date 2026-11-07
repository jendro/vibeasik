import type { QueueJob, QueueStats } from './queue'
import type { Todo } from './todo'
import type { OfficePlanningState } from './planner'
import type { EmployeeAvatar, OfficeEmployee, OfficeStaffState } from './staff'
import type { LLMPlannerTodoItem } from './base_knowledge'
import type { ActiveExecutiveMeeting } from './project'

/** `employee` = pegawai vibeasik yang sedang tidak mewakili pekerja nyata (hanya di studio). */
export type WorkerKind = 'coding_agent' | 'local_agent' | 'llm_call' | 'planner' | 'employee'
export type WorkerStatus = 'working' | 'idle' | 'done' | 'error' | 'left'

/** Satu "orang" di Kantor Agent (GET /api/agents/office). */
export interface AgentWorker {
  id: number
  key: string
  kind: WorkerKind
  name: string
  client_name: string
  model?: string
  /** Mode divisi: "Pengambil todo" (koordinator) atau "Anak buah · <tipe>" (sub agent). */
  role?: string
  /** Key koordinator bila pekerja ini sub agent suruhannya. */
  parent_key?: string
  transport?: string
  status: WorkerStatus
  action?: string
  task?: string
  todo_id?: number
  project_id?: number
  project_name?: string
  project_path?: string
  tool_calls: number
  tasks_done: number
  last_error?: string
  started_at: string
  last_seen_at: string
  finished_at?: string
}

/** Transkrip agent di laptop yang dipantau engine (Claude Code, Antigravity, Copilot). */
export interface WatchedSource {
  agent: string
  path: string
  found: boolean
  active_sessions: number
}

export interface AgentOfficeSnapshot {
  server_time: string
  workers: AgentWorker[]
  queue: { running: QueueJob[] | null, queued: QueueJob[] | null, stats: QueueStats }
  active_todos: Todo[] | null
  sources: WatchedSource[] | null
  thresholds: { idle_after_sec: number, finished_linger_sec: number, error_linger_sec?: number }
  planning?: OfficePlanningState
  staff?: OfficeStaffState
}

/** Keadaan yang digambar untuk karakter, diturunkan dari status + waktu aktivitas terakhir. */
export type ActorMood = 'working' | 'idle' | 'sleeping' | 'done' | 'error' | 'left'

export interface OfficeActor {
  key: string
  worker: AgentWorker
  mood: ActorMood
  x: number
  y: number
  /** Lantai (0 = lantai dasar & luar gedung). */
  f: number
  /** Arah hadap saat diam (radian, lihat `Spot.face`). */
  face?: number
  /** Duduk di balik meja (badan bawah tertutup meja). */
  seated: boolean
  walking: boolean
  moveMs: number
  facing: 'left' | 'right'
  leaving: boolean
  celebrating: boolean
  /** Duduk di kursinya sendiri (pegawai di zona divisi). */
  atDesk?: boolean
  /** Key pekerja nyata yang sedang diwakili pegawai ini. */
  srcKey?: string
  /** Pegawai vibeasik yang digambar actor ini (kursi tetap di zona divisinya). */
  employee?: OfficeEmployee
}

/** Proyek yang sedang dikerjakan agent AI (dikelompokkan dari pekerja aktif di kantor). */
export interface OfficeProject {
  key: string
  name: string
  path?: string
  workers: AgentWorker[]
  working: number
  todos: number
  last_seen_at: string
}

/** Angka satu hari kalender pada rekap progres (GET /api/agents/recap). */
export interface RecapDay {
  date: string
  commits: number
  ai_commits: number
  additions: number
  deletions: number
  files_changed: number
  todos_completed: number
  todos_created: number
  todos_completed_by_ai: number
  runs_completed: number
  agent_sessions: number
}

export interface RecapRepo {
  name: string
  path: string
  project_name?: string
  commits: number
  today_commits: number
  additions: number
  deletions: number
  last_commit_at?: string
}

export interface RecapCommit {
  hash: string
  subject: string
  author: string
  at: string
  repo: string
  project_name?: string
  additions: number
  deletions: number
  files: number
  agent?: string
}

export interface RecapCompletedTodo {
  id: number
  title: string
  project_name?: string
  category: string
  priority: string
  completed_at: string
  completed_by_agent?: string
}

export interface RecapFileCluster {
  file: string
  touch_count: number
  todo_ids: number[]
  sample_titles: string[]
  recommendation: string
}

export interface RecapCategorizedTodo {
  id: number
  title: string
  project_name?: string
  category: string
  priority: string
  completed_at: string
  completed_by_agent?: string
  impact_score: number
  impact_type: 'ui_ux' | 'performance' | 'code_only'
  impact_label: string
  impact_tier: 'high' | 'medium' | 'low'
  impact_rationale: string
  file_target?: string
  action_strategy: string
  is_cluster_hotspot: boolean
}

export interface RecapImpactAnalysis {
  summary: string
  total_analyzed: number
  ui_ux_count: number
  performance_count: number
  code_only_count: number
  ui_ux_percent: number
  performance_percent: number
  code_only_percent: number
  potential_savings_percent: number
  file_clusters: RecapFileCluster[]
  categorized_todos: RecapCategorizedTodo[]
}

export interface AgentRecap {
  generated_at: string
  from: string
  to: string
  days: RecapDay[]
  today: RecapDay
  yesterday: RecapDay
  totals: RecapDay
  best_day?: RecapDay
  streak: { current: number, longest: number }
  repos: RecapRepo[] | null
  contributors: Array<{ name: string, todos_completed: number, runs_completed: number, commits: number }> | null
  categories: Array<{ key: string, count: number }> | null
  priorities: Array<{ key: string, count: number }> | null
  recent_commits: RecapCommit[] | null
  completed_todos: RecapCompletedTodo[] | null
  impact_analysis?: RecapImpactAnalysis | null
  open_todos: number
}

/** Jenis permintaan klien di Ruang Klien; menentukan cara dewan menanggapi & menyusun todo. */
export type ClientRequestKind = 'bug' | 'question' | 'feature' | 'change' | 'other'

/** Pekerjaan dewan yang sedang berjalan di Ruang Klien (dilaporkan komponen chat ke panel kanan). */
export interface ClientBoardActivity {
  meeting: ActiveExecutiveMeeting | null
  cycleRunning: boolean
  analyzing: boolean
  replying: { who: string, kind: string } | null
  autonomous: boolean
  lastRunAt?: string
}

/** Todo yang lahir dari permintaan klien beserta status terkininya. */
export interface ClientTodoRef {
  id: number
  title: string
  status: string
  priority: string
}

export interface ExecutiveChatMessageDTO {
  id: number
  project_id: number
  role: 'user' | 'assistant'
  target: 'all' | 'ceo' | 'cto' | 'product_owner' | 'cmo'
  kind?: ClientRequestKind
  speaker_role?: string
  speaker_name?: string
  speaker_title?: string
  avatar?: EmployeeAvatar
  content: string
  todos?: LLMPlannerTodoItem[]
  board_notes?: any[]
  saved_todos?: ClientTodoRef[]
  model_name?: string
  created_at: string
}

export interface ExecutiveChatRequestDTO {
  message: string
  target?: string
  kind?: ClientRequestKind
  model?: string
  auto_save_todos?: boolean
}

export interface ExecutiveChatResponseDTO {
  reply: string
  analysis: string
  speaker_role: string
  speaker_name: string
  speaker_title: string
  avatar?: EmployeeAvatar
  todos?: LLMPlannerTodoItem[]
  board_notes?: any[]
  saved_todos?: any[]
  model: string
  message_id: number
}
