/** Pegawai vibeasik: agent AI berkepribadian & berpengetahuan sendiri (GET /api/staff). */

export type StaffDivision = 'product' | 'engineering' | 'quality' | 'platform' | 'business' | 'executive'

export interface EmployeeAvatar {
  skin: string
  hair: string
  hair_style: 'short' | 'long' | 'bun' | 'bald' | 'curly' | 'hijab' | 'ponytail' | 'spiky'
  shirt: string
  pants: string
  accessory: 'none' | 'glasses' | 'headset' | 'cap' | 'beanie' | 'tie' | 'scarf'
  facial: 'none' | 'beard' | 'mustache'
  emoji: string
}

export interface Employee {
  id: number
  name: string
  role: string
  title: string
  division: StaffDivision
  seniority: 'junior' | 'mid' | 'senior' | 'lead'
  specialty: string
  profile_type?: 'specialist' | 'generalist'
  stack: string[] | null
  brief: string
  personality: string
  motto: string
  skills: Record<string, number> | null
  traits: Record<string, number> | null
  /** Kunci prinsip kerja yang dipegang pegawai (maks. 3) */
  principles?: string[] | null
  /** Pola pikir: sumbu dua kutub 0–100 */
  mind?: Record<string, number> | null
  /** Selera tampilan: sumbu dua kutub 0–100 */
  taste?: Record<string, number> | null
  avatar: EmployeeAvatar
  desk: number
  /** Divisi yang dipimpin (leader menimbang siapa mengerjakan todo divisinya). */
  leads_division?: StaffDivision
  status: 'active' | 'resigned'
  /** SP yang sedang berlaku dari HR (0 = tidak ada, 1–3) */
  warning_level?: number
  /** Proyek terikat (untuk eksekutif proyek: CEO, CTO, PO, CMO) */
  project_id?: number
  project_name?: string
  xp: number
  level: number
  tasks_done: number
  tasks_reviewed: number
  avg_score: number
  last_review_at?: string
  /** Catatan eks pegawai yang digantikan pegawai baru ini */
  replaced_employee_id?: number
  replaced_employee_name?: string
  replacement_reason?: string
  inherited_notes_count?: number
  inherited_summary?: string
  hired_at: string
}

/** Satu langkah kerja pegawai beserta riwayat pemeriksaannya. */
export interface EmployeePlaybookStep {
  id: string
  phase: 'pahami' | 'rancang' | 'kerjakan' | 'periksa' | 'lapor'
  title: string
  instruction: string
  must: boolean
  risk: number
  checked_by_program: boolean
  skill: string
  /** Kotak ingatan: 0 belum pernah diperiksa, 1 baru gagal, makin tinggi makin dikuasai */
  box: number
  passed: number
  failed: number
  last_note?: string
  /** Tumbuh dari pengalaman (bukan bawaan) */
  learned?: boolean
  /** Milik pegawai ini saja */
  personal?: boolean
  /** Masih dicoba diam-diam: belum memengaruhi nilai */
  trial?: boolean
  trial_checks?: number
  trial_hits?: number
  origin?: string
}

/** Satu sumbu dua kutub (0 = kutub low, 100 = kutub high). */
export interface EmployeeAxis {
  key: string
  group: string
  low: string
  high: string
  value: number
}

/** Satu pilihan gaya pegawai di dalam rentang galeri. */
export interface EmployeeStylePick {
  key: string
  label: string
  value: string
  range: string
  why: string
}

/** Pola pikir & selera seorang pegawai. */
export interface EmployeeMind {
  code: string
  type: string
  lines: string[] | null
  strengths: string[] | null
  blind_spots: string[] | null
  axes: EmployeeAxis[]
  taste: EmployeeAxis[]
  taste_words: string[] | null
  style?: { words: string[] | null, numbers: EmployeeStylePick[] | null, accent?: EmployeeStylePick, styles: string[] | null, basis: string }
}

/** Langkah kerja, prinsip, dan kondisi seorang pegawai. */
export interface EmployeePlaybook {
  employee_id: number
  name: string
  role: string
  /** false = peran ini baru memakai langkah umum */
  own_book: boolean
  work_style: string
  principles: string[]
  principle_keys: string[]
  principle_options: { key: string, text: string }[] | null
  condition: 'mantap' | 'waspada' | 'tertekan' | 'jenuh'
  condition_reason?: string
  habits: string[]
  mind: EmployeeMind
  steps: EmployeePlaybookStep[] | null
}

/** Hasil ekspor paket pegawai: isi berkas bertanda tangan dan ringkasannya. */
export interface EmployeePackageExport {
  file: { package: unknown, public_key: string, signature: string }
  passport: { format: string, serial: string, copy_no: number, license: string, origin: string, exported_at: string }
  knowledge: number
  /** Butir pengetahuan yang tidak ikut karena menyebut proyek atau path */
  dropped: number
  steps: number
}

/** Pegawai + angka ringkas (daftar). */
export interface EmployeeSummary extends Employee {
  knowledge: number
  active_tasks: number
  pending_review: number
  reviewing: boolean
  chatting: boolean
}

export interface EmployeeKnowledge {
  id: number
  employee_id: number
  topic: string
  content: string
  kind: 'technical' | 'process' | 'lesson' | 'preference'
  source: 'seed' | 'qa_review' | 'manual' | 'agenda' | 'chat' | 'team' | 'alumni'
  weight: number
  source_todo_id?: number
  /** Catatan tim (source team): anggota yang dicatat leader */
  about_employee_id?: number
  review_id?: number
  status: 'active' | 'removed' | 'merged'
  merged_into_id?: number
  removed_reason?: string
  removed_at?: string
  created_at: string
}

export interface EmployeeTask {
  id: number
  employee_id: number
  todo_id: number
  title: string
  project_name?: string
  category: string
  assigned_by?: string
  /** Area spesialisasi todo, mis. "backend_engineer/go". */
  area?: string
  leader_id?: number
  leader_name?: string
  assign_reason?: string
  status: 'assigned' | 'completed' | 'reviewed'
  completed_at?: string
  review_id?: number
  score?: number
  qa_comment?: string
  high_score_skills?: string[] | null
  high_score_notes?: string
  success_factors?: string[] | null
  /** Nilai panel QC per bidang (architecture, security, performance, data, testing, ux): level 0–4 + catatan */
  qc_fields?: Record<string, QCFieldScore> | null
  /** Nasib todo setelah nilai QA rendah: dibuka lagi lewat leader, atau batas pengerjaan ulang habis */
  rework?: 'reopened' | 'limit'
  /** Level dampak todo ke pengguna akhir & performa (0–4, Aturan 38) — bobot rata-rata nilai pegawai */
  impact?: number
  impact_reason?: string
  /** relevansi terhadap kondisi pegawai sekarang (0,15–1), pengali bobot di avg_score */
  relevance?: number
  created_at: string
}

export interface QCFieldScore {
  level: number
  note?: string
}

export interface KnowledgeChange {
  knowledge_id?: number
  topic: string
  content?: string
  reason?: string
  todo_id?: number
}

export interface EmployeeReview {
  id: number
  employee_id: number
  employee_name: string
  reviewer_id: number
  reviewer_name: string
  status: 'queued' | 'running' | 'completed' | 'failed'
  task_ids: number[] | null
  score: number
  summary?: string
  added: KnowledgeChange[] | null
  removed: KnowledgeChange[] | null
  skill_changes: Array<{ skill: string, delta: number, reason?: string }> | null
  xp_gained: number
  /** Todo bernilai rendah yang dibuka lagi lewat leader setelah review ini */
  rework_todo_ids?: number[] | null
  relevance?: number
  model?: string
  error?: string
  created_at: string
  finished_at?: string
}

export interface StaffRole {
  key: string
  title: string
  division: StaffDivision
  icon: string
  technical: boolean
  brief: string
  skills: Record<string, number>
  categories: string[] | null
  planner_role?: string
}

export interface StaffSettings {
  enabled: boolean
  auto_review: boolean
  review_min_tasks: number
  review_settle_min: number
  max_knowledge: number
  /** Agenda pribadi pegawai: dibuat dari pemicu otomatis & dikerjakan saat senggang */
  agenda_enabled: boolean
  agenda_auto_run: boolean
  /** Agenda investigasi boleh memakai agent CLI lokal (read-only, jendela terminal dibuka di laptop) */
  agenda_allow_cli: boolean
  agenda_daily_limit?: number
  /** Seberapa sering agenda dikerjakan per pegawai — sedikit, normal, sering, sangat_sering */
  agenda_rate?: AgendaRate
  /** Nilai QA di bawah ambang ini memicu agenda belajar */
  agenda_low_score: number
  /** Todo bernilai QA di bawah rework_score dibuka lagi & diserahkan ke leader divisinya (arahan + pegawai baru) */
  rework_enabled: boolean
  rework_score: number
  /** Batas pengerjaan ulang per todo */
  rework_max: number
  /** QA menilai diff branch todo sebelum di-squash ke branch utama; kurang dari ambang → dikerjakan ulang di branch yang sama */
  qa_gate_enabled: boolean
  qa_gate_score: number
  /** Putaran QA gate per todo sebelum dieskalasi ke leader */
  qa_gate_max_rounds: number
  /** Todo selesai bernilai kurang masuk antrean perbaikan tertunda; dibuka lagi setelah ada bukti skill naik */
  revisit_enabled: boolean
  /** Nilai di bawah ambang ini (tapi ≥ rework_score) dianggap lulus tipis */
  revisit_score: number
  /** Todo yang dibuka lagi dari antrean per hari */
  revisit_daily: number
  /** Todo yang selesai lebih lama dari ini dianggap usang */
  revisit_max_age_days: number
  /** Maksimal berapa siklus pengulangan revisit untuk satu todo bila nilainya masih rendah */
  revisit_max_cycles: number
  /** Pegawai santai (tanpa todo aktif) boleh mengobrol sesama pegawai & upgrade skill sendiri */
  idle_social: boolean
  idle_social_rate: SocialRate
  /** Model obrolan "<config_id>:<model>"; kosong = otomatis (9Router lokal deepseek flash) */
  conversation_model: string
  /** Pegawai HR mengevaluasi kinerja (SP), relevansi (usulan pecat) & beban tim (usulan rekrut) */
  hr_enabled: boolean
  /** Usulan rekrut & pecat langsung dijalankan tanpa persetujuan */
  hr_auto_approve: boolean
  /** Rata-rata nilai QA (min. 3 todo) di bawah ini memicu SP */
  hr_warn_score: number
  /** Pegawai teknis tanpa todo sekian hari (peran masih diisi rekan) diusulkan dipecat */
  hr_idle_days: number
  /** Rata-rata todo aktif per pegawai seperan di atas ini memicu usulan rekrut */
  hr_hire_load: number
}

export type SocialRate = 'sedikit' | 'normal' | 'banyak' | 'sangat_sering'
export type AgendaRate = 'sedikit' | 'normal' | 'sering' | 'sangat_sering' | 'banyak'

export interface ConvLine {
  employee_id: number
  name: string
  text: string
}

/** Obrolan santai dua pegawai (GET /api/staff/conversations). */
export interface EmployeeConversation {
  id: number
  a_id: number
  a_name: string
  b_id: number
  b_name: string
  topic: string
  lines: ConvLine[] | null
  /** Pengetahuan yang dipetik dari obrolan, satu baris per pegawai */
  learned?: string
  trigger: 'idle' | 'manual'
  status: 'queued' | 'running' | 'done' | 'failed'
  model?: string
  error?: string
  finished_at?: string
  created_at: string
}

export interface ConversationOverview {
  conversations: EmployeeConversation[]
  /** Model obrolan yang sedang dipakai (pilihan pengaturan atau otomatis) */
  model: string
  rates: SocialRate[]
}

export type AgendaKind = 'reflect' | 'learn' | 'investigate' | 'improve'
export type AgendaStatus = 'pending' | 'running' | 'done' | 'failed' | 'dropped'
export type AgendaNeed = 'light' | 'standard' | 'deep'

/** Todo pribadi pegawai — terpisah dari todo proyek (GET /api/staff/agenda). */
export interface EmployeeAgenda {
  id: number
  employee_id: number
  employee_name: string
  kind: AgendaKind
  title: string
  detail: string
  /** todo_completed, qa_low_score, run_failed, self, agent, manual */
  trigger: string
  source_ref?: string
  project_id?: number
  project_name?: string
  work_dir?: string
  priority: number
  status: AgendaStatus
  executor: 'llm' | 'cli'
  need: AgendaNeed
  prefer_model?: string
  model_used?: string
  /** JSON jejak percobaan model (pool LLM) */
  attempts?: string
  run_id?: number
  result?: string
  error?: string
  created_by?: string
  started_at?: string
  finished_at?: string
  created_at: string
}

export interface AgendaOverview {
  agendas: EmployeeAgenda[]
  counts: { status: AgendaStatus, n: number }[] | null
  settings: StaffSettings
}

export interface AgendaInput {
  kind: AgendaKind
  title: string
  detail?: string
  need?: AgendaNeed
  executor?: 'llm' | 'cli'
  prefer_model?: string
  project_id?: number
  priority?: number
}

export interface EmployeeTurnover {
  id: number
  former_employee_id: number
  former_employee_name: string
  former_role: string
  former_division: StaffDivision
  dismiss_reason: string
  dismissed_at: string
  successor_id?: number
  successor_name?: string
  hired_at?: string
  inherited_notes_count: number
  inherited_skills?: string[] | null
  inherited_summary?: string
  created_at: string
  updated_at: string
}

export interface StaffOverview {
  employees: EmployeeSummary[]
  roles: StaffRole[]
  divisions: Record<StaffDivision, string>
  settings: StaffSettings
  reviews: EmployeeReview[] | null
  turnovers?: EmployeeTurnover[] | null
}

export interface EmployeeDetail extends EmployeeSummary {
  role_info: StaffRole
  knowledge_items: EmployeeKnowledge[] | null
  removed_knowledge: EmployeeKnowledge[] | null
  tasks: EmployeeTask[] | null
  reviews: EmployeeReview[] | null
  reviews_given: EmployeeReview[] | null
}

export interface EmployeeChatMessage {
  id: number
  employee_id: number
  role: 'user' | 'assistant'
  content: string
  status: 'pending' | 'answered' | 'failed'
  model?: string
  error?: string
  created_at: string
}

/** Keadaan pegawai di Kantor Agent (bagian `staff` GET /api/agents/office). */
export type OfficeEmployeeStatus = 'idle' | 'working' | 'meeting' | 'reviewing' | 'learning' | 'chatting' | 'social' | 'break' | 'studying'

export interface OfficeEmployee {
  id: number
  name: string
  title: string
  role: string
  division: StaffDivision
  specialty: string
  desk: number
  leader: boolean
  level: number
  avatar: EmployeeAvatar
  status: OfficeEmployeeStatus
  action?: string
  /** Pekerja nyata (agent coding / CLI / peserta rapat) yang diwakili pegawai ini. */
  worker_key?: string
  todo_id?: number
  todo_title?: string
  knowledge: number
  /** Obrolan santai / rehat: di meja tetangga (desk) atau pantry */
  place?: 'desk' | 'pantry'
  partner_id?: number
  /** Tuan rumah obrolan di meja: tetap duduk, rekannya berdiri di samping mejanya */
  host?: boolean
}

export interface OfficeStaffState {
  enabled: boolean
  employees: OfficeEmployee[] | null
  divisions: Record<StaffDivision, string>
  executive_room?: OfficeExecutive[] | null
}

/** Pengalaman satu pegawai di satu area spesialisasi. */
export interface AreaStat {
  area: string
  done: number
  reviewed: number
  avg_score: number
}

export interface TeamMember {
  id: number
  name: string
  title: string
  role: string
  specialty: string
  level: number
  leader: boolean
  active_tasks: number
  tasks_done: number
  avg_score: number
  areas: AreaStat[] | null
  top_skills: string
}

/** Peta keahlian satu divisi (GET /api/staff/teams). */
export interface StaffTeam {
  division: StaffDivision
  name: string
  leader_id?: number
  leader_avatar?: EmployeeAvatar
  members: TeamMember[]
}

export type HRKind = 'hire' | 'warning' | 'dismiss'
export type HRStatus = 'proposed' | 'active' | 'done' | 'rejected' | 'expired' | 'cleared' | 'superseded'

/** Angka yang dibaca HR saat mengambil tindakan. */
export interface HRMetrics {
  reviewed?: number
  avg_score?: number
  idle_days?: number
  members?: number
  active_tasks?: number
  borrowed?: number
  peer_tasks?: number
  unmatched_todo_id?: number
  missing_skills?: string[]
  recommended_specialty?: string
}

/** Tindakan HR (GET /api/staff/hr): SP, usulan pecat, atau pemberitahuan butuh pegawai baru. */
export interface EmployeeHRAction {
  id: number
  kind: HRKind
  level?: number
  employee_id?: number
  employee_name?: string
  role: string
  division: StaffDivision
  cause?: 'performance' | 'irrelevant' | 'unmatched_todo'
  title: string
  reason: string
  letter: string
  metrics: HRMetrics
  status: HRStatus
  issuer_name?: string
  hired_id?: number
  hired_name?: string
  model?: string
  expires_at?: string
  decided_at?: string
  decided_by?: string
  decide_note?: string
  created_at: string
}

export interface HROverview {
  actions: EmployeeHRAction[]
  counts: { kind: HRKind, status: HRStatus, n: number }[]
  officer: Employee | null
}

export type RevisitCause = 'limit' | 'borderline' | 'skipped'
export type RevisitStatus = 'waiting' | 'ready' | 'hold' | 'reopened' | 'done' | 'stale' | 'dismissed'

/** Satu todo di antrean perbaikan tertunda (GET /api/staff/revisits). Nilai lama tidak pernah ditimpa. */
export interface EmployeeRevisit {
  id: number
  todo_id: number
  task_id: number
  title: string
  project_name?: string
  area?: string
  employee_id: number
  employee_name: string
  cause: RevisitCause
  old_score: number
  old_comment?: string
  priority: number
  status: RevisitStatus
  /** Bukti skill naik yang membangunkan todo */
  trigger?: string
  stale_reason?: string
  assignee_id?: number
  assignee_name?: string
  new_score?: number
  reopened_at?: string
  finished_at?: string
  /** Status todo saat ini (pending/in_progress/completed) — diisi backend via join saat status revisit = reopened */
  todo_status?: string
  /** Siklus perbaikan tertunda ke-N untuk todo ini (1 = pertama, 2 = setelah siklus 1 masih rendah, dst.) */
  revisit_cycle?: number
  created_at: string
}

export interface RevisitOverview {
  revisits: EmployeeRevisit[]
  counts: Partial<Record<RevisitStatus, number>>
}

export interface RevisitReport {
  queued: number
  woken: number
  reopened: number
  stale: number
  done: number
}

/** Peran eksekutif per proyek (CEO, CTO, Product Owner, CMO). */
export type ExecutiveRoleKey = 'ceo' | 'cto' | 'product_owner' | 'cmo'

export interface ExecRole {
  key: ExecutiveRoleKey
  title: string
  icon: string
  brief: string
  skills: Record<string, number>
  categories: string[]
  planner_role: string
}

export interface ProjectExecutive {
  id: number
  project_id: number
  project_name: string
  role: ExecutiveRoleKey
  employee_id?: number
  name: string
  title: string
  brief: string
  personality: string
  motto: string
  avatar: EmployeeAvatar
  skills: Record<string, number> | null
  level?: number
  xp?: number
  tasks_done?: number
  avg_score?: number
  status: 'active' | 'inactive'
  created_at: string
  updated_at: string
}

export type ExecKnowledgeKind = 'app_reality' | 'product_vision' | 'roadmap' | 'technical_decision' | 'market' | 'forecast' | 'user_feedback' | 'lesson'
export type ExecKnowledgeSource = 'seed' | 'todo_completed' | 'meeting' | 'profile' | 'manual' | 'chat' | 'autonomous' | 'market_research' | 'qa_review'

export interface ExecutiveKnowledge {
  id: number
  executive_id: number
  project_id: number
  topic: string
  content: string
  kind: ExecKnowledgeKind
  source: ExecKnowledgeSource
  weight: number
  source_todo_id?: number
  source_session_id?: number
  status: 'active' | 'archived'
  created_at: string
  updated_at: string
}

export interface ExecutiveDetail extends ProjectExecutive {
  knowledge_count: number
  knowledge: ExecutiveKnowledge[]
}

export interface OfficeExecutive {
  id: number
  project_id: number
  project_name: string
  role: string
  name: string
  title: string
  avatar: EmployeeAvatar
  knowledge: number
  status: 'idle' | 'meeting' | 'reviewing_strategy'
  action?: string
}
