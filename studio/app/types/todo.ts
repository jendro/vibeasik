export type TodoLevel = 'global' | 'project' | 'component'
/**
 * merged = sub todo yang dilebur ke todo gabungan (parent_id); dikerjakan & selesai lewat induknya.
 * split = induk lintas aplikasi yang dipecah per aplikasi (sub todo: split_from_id); selesai sendiri saat semua sub selesai.
 */
export type TodoStatus = 'pending' | 'in_progress' | 'completed' | 'suggested' | 'rejected' | 'merged' | 'split'
export type TodoPriority = 'low' | 'medium' | 'high' | 'critical'
export type TodoCreatorType = 'human' | 'ai_agent'
export type TodoComplexityLevel = 'quick' | 'normal' | 'complex'
export type TodoImpactLevel = 'high' | 'medium' | 'low' | 'minimal'

export type TodoCategory = 'feature' | 'security' | 'ui_ux' | 'architecture' | 'performance' | 'backend' | 'devops' | 'general'

export interface Todo {
  id: number
  level: TodoLevel
  category?: TodoCategory | string
  parent_id?: number
  split_from_id?: number
  project_id?: number
  project_name?: string
  component_id?: number
  component_name?: string
  target_apps?: string
  title: string
  description: string
  status: TodoStatus
  priority: TodoPriority
  creator_type: TodoCreatorType
  creator_name: string
  is_ai_suggestion: boolean
  guideline_ref?: string
  /** CSV tag fokus: fitur, keamanan-dasar, optimasi-query, rawat-ai, belajar (utils/todoTags.ts). */
  tags?: string
  audit_key?: string
  rationale?: string
  source: string
  due_date?: string
  completed_at?: string
  completed_by_agent?: string
  /** Pegawai vibeasik yang ditugaskan (persona agent coding yang mengerjakannya). */
  employee_id?: number
  employee_name?: string
  employee_leader_id?: number
  /** Alasan leader divisi memilih pegawai ini. */
  employee_reason?: string
  /** Berapa kali todo dibuka lagi karena nilai QA rendah. */
  rework_count?: number
  /** Arahan leader untuk pengerjaan ulang terakhir. */
  rework_note?: string
  completed_by_model?: string
  /** Nilai hasil review QA (0–100) */
  score?: number
  /** Skill penentu nilai tinggi */
  high_score_skills?: string[] | null
  /** Catatan detail kenapa pengerjaan ini berhasil/bernilai tinggi */
  high_score_notes?: string
  /** List butir-butir faktor keberhasilan */
  success_factors?: string[] | null
  /** Nama eks pegawai yang standarnya diwarisi */
  inherited_from?: string
  /** Kerumitan 1–5 per dimensi; diisi otomatis (heuristik, atau LLM saat dibuat dari audit/saran AI) */
  complexity_logic: number
  complexity_coding: number
  complexity_scope: number
  complexity_risk: number
  complexity_score: number
  complexity_level: TodoComplexityLevel | ''
  complexity_note?: string
  complexity_source?: 'heuristic' | 'llm' | 'manual' | ''
  /** Dampak 0–5 per dimensi (kode dinilai dari manfaatnya bagi AI agent; manusia: tampilan & kecepatan) */
  impact_ai_context: number
  impact_maintainability: number
  impact_reliability: number
  impact_security: number
  impact_speed: number
  impact_ux: number
  impact_feature: number
  /** Skor final 0–100, berbobot sesuai kategori todo */
  impact_score: number
  impact_level: TodoImpactLevel | ''
  /** Apa yang meningkat pada aplikasi setelah todo selesai */
  impact_summary?: string
  impact_source?: 'heuristic' | 'llm' | 'manual' | ''
  created_at: string
  updated_at: string
}

export interface GuidelineItem {
  id: string
  level: string
  title: string
  rule: string
  rationale: string
  inspection_check: string
  suggested_action: string
  severity: string
}

export interface GuidelineCategory {
  category: string
  name: string
  description: string
  file_path?: string
  guidelines: GuidelineItem[]
}

export type ImpactDimensionKey = 'ai_context' | 'maintainability' | 'reliability' | 'security' | 'speed' | 'ux' | 'feature'

export interface ImpactDimension {
  key: ImpactDimensionKey
  label: string
  audience: 'ai_agent' | 'app' | 'human'
  hint: string
}

/** GET /api/todos/impact-dimensions — definisi dimensi & bobot per kategori dari engine */
export interface ImpactDimensionsInfo {
  dimensions: ImpactDimension[]
  /** Bobot per kategori, urutan sama dengan `dimensions` */
  weights: Record<string, number[]>
  levels: Record<TodoImpactLevel, number>
  formula: string
}

/** Satu todo gabungan hasil "Lebur Todo Serupa" (Aturan 43): todo serupa dikerjakan satu agent sekaligus. */
export interface TodoMergeGroup {
  /** Kosong pada pratinjau, kecuali todo baru bergabung ke todo gabungan yang sudah ada */
  parent_id?: number
  title: string
  /** Alasan dilebur: rule sama, berkas sama, atau isi & judul mirip (RAG todo) */
  reason: string
  todo_ids: number[]
  titles: string[]
  joined?: boolean
}

/** POST /api/todos/merge */
export interface TodoMergeResult {
  groups: TodoMergeGroup[]
  /** Jumlah todo yang baru dilebur */
  merged: number
}
