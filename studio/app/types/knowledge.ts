/** Pusat Pengetahuan: satu pintu patokan & pengetahuan umum pekerja AI vibeasik (GET /api/knowledge). */

export type KnowledgeKind = 'rule' | 'procedure' | 'lesson' | 'concept'
export type KnowledgeSource = 'guideline' | 'audit_checklist' | 'audit_verdict' | 'employee' | 'promoted' | 'gap_research' | 'qa_review' | 'design_library' | 'design_theory' | 'design_lesson' | 'manual' | 'agent' | 'hub'
export type KnowledgeGapStatus = 'open' | 'learning' | 'learned' | 'dismissed'

export interface KnowledgeNode {
  id: number
  key?: string
  kind: KnowledgeKind
  domain: string
  tags: string[] | null
  title: string
  content: string
  source: KnowledgeSource
  owner_employee_id?: number
  owner_name?: string
  weight: number
  helpful: number
  harmful: number
  uses: number
  last_used_at?: string
  cluster_id: number
  merged_into_id?: number
  needs_review: boolean
  status: 'active' | 'merged' | 'retired'
  status_reason?: string
  created_at: string
  updated_at: string
}

/** Node hasil temu-kembali / daftar beserta rincian skornya. */
export interface KnowledgeHit extends KnowledgeNode {
  score: number
  relevance: number
  utility: number
  via?: string
}

export interface KnowledgeCluster {
  id: number
  label: string
  domain: string
  terms: string[] | null
  size: number
  utility: number
}

export interface KnowledgeGap {
  id: number
  domain: string
  topic: string
  sample?: string
  hits: number
  auto: boolean
  attempts: number
  status: KnowledgeGapStatus
  employee_id?: number
  employee_name?: string
  agenda_id?: number
  nodes_learned: number
  note?: string
  learned_at?: string
  updated_at: string
}

export interface KnowledgeOverview {
  total: number
  shared: number
  personal: number
  by_kind: Record<string, number>
  by_source: Record<string, number>
  domains: { domain: string, nodes: number, ecosystem: boolean }[] | null
  clusters: KnowledgeCluster[] | null
  gaps: KnowledgeGap[] | null
  needs_review: KnowledgeNode[] | null
  tested: number
  utility: number
  links: number
  uses_pending: number
}

export interface KnowledgeRecallResult {
  hits: KnowledgeHit[] | null
  domains: string[] | null
  gaps?: string[] | null
}

export interface KnowledgeDetail {
  node: KnowledgeNode
  utility: number
  neighbors: (KnowledgeNode & { strength: number })[] | null
  uses: { id: number, node_id: number, context: string, outcome?: string, created_at: string }[] | null
  merged_to?: KnowledgeNode
  merged?: KnowledgeNode[] | null
}

export interface KnowledgeCurateReport {
  synced: number
  merged: number
  promoted: number
  retired: number
  flagged: number
  links: number
  clusters: number
  nodes: number
  took: string
}

export interface KnowledgeNoteInput {
  title: string
  content: string
  kind: KnowledgeKind
  domain?: string
}

/** Jaring pengetahuan (GET /api/knowledge/graph): node ringkas + tautan kemiripan, untuk peta jaring. */
export interface KnowledgeGraphNode {
  id: number
  title: string
  kind: KnowledgeKind
  domain: string
  source: KnowledgeSource
  cluster_id: number
  personal: boolean
  weight: number
  helpful: number
  harmful: number
  uses: number
  utility: number
}

export interface KnowledgeGraph {
  nodes: KnowledgeGraphNode[]
  links: { source: number, target: number, strength: number }[]
  clusters: KnowledgeCluster[] | null
}

/** Pratinjau prompt sebuah pekerjaan (POST /api/knowledge/preview) — tanpa efek samping. */
export interface KnowledgePromptPreview {
  employee?: { id: number, name: string, role: string }
  reason?: string
  prompt: string
  knowledge_section: string
  hits: KnowledgeHit[] | null
  domains: string[] | null
  gaps: string[] | null
  context: string
  /** Terisi bila pekerjaan menyentuh beberapa aplikasi berbeda di proyek: satu prompt per pegawai per aplikasi. */
  parts?: KnowledgePromptPart[]
}

export interface KnowledgePromptPart {
  class: 'backend' | 'frontend' | 'mobile'
  apps: string[]
  title: string
  employee?: { id: number, name: string, role: string }
  reason?: string
  prompt: string
  knowledge_section: string
  hits: KnowledgeHit[] | null
}
