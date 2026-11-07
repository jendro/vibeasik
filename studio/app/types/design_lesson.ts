import type { DesignColor } from './design'

/** Pelajaran Desain (before/after): gambar awal + gambar akhir + narasi konten, dianalisis LLM vision. */
export type DesignLessonSide = 'before' | 'after'
export type DesignLessonPlatform = 'graphic' | 'web' | 'android' | 'ios'
export type DesignLessonStatus = 'new' | 'analyzed' | 'failed'

export interface DesignLessonMistake {
  area: string
  principle: string
  problem: string
  evidence: string
}

export interface DesignLessonChange {
  area: string
  change: string
  why: string
}

export interface DesignLessonPrinciple {
  area: string
  rule: string
  how_to_apply: string
  theory_id?: string
  /** Kunci node prinsip pelajaran sebelumnya yang diperkuat (keputusan LLM). */
  same_as?: string
}

export interface DesignLessonAnalysis {
  summary: string
  before_desc: string
  after_desc: string
  mistakes: DesignLessonMistake[]
  changes: DesignLessonChange[]
  principles: DesignLessonPrinciple[]
  palette?: { before: DesignColor[], after: DesignColor[], notes: string }
  takeaway: string
  matched?: boolean
}

export interface DesignLesson {
  id: number
  title: string
  narrative: string
  source_url: string
  platform: DesignLessonPlatform
  tags: string[]
  before_file?: string
  after_file?: string
  status: DesignLessonStatus
  last_error?: string
  analysis?: DesignLessonAnalysis
  analysis_model?: string
  analyzed_at?: string
  created_at: string
  updated_at: string
}

export interface DesignLessonInput {
  title?: string
  narrative?: string
  source_url?: string
  platform?: DesignLessonPlatform
  tags?: string[]
}

/** Node Pusat Pengetahuan hasil satu pelajaran + tetangganya di jaring pengetahuan. */
export interface DesignLessonNeighbor {
  id: number
  title: string
  kind: string
  source: string
  strength: number
}

export interface DesignLessonNode {
  id: number
  key: string
  kind: string
  title: string
  status: string
  /** baru = node dibuat pelajaran ini; memperkuat = node pelajaran lain yang dikuatkan pelajaran ini */
  role: 'baru' | 'memperkuat'
  /** jumlah pelajaran yang mendukung node ini */
  supports: number
  weight: number
  uses: number
  helpful: number
  harmful: number
  cluster_id: number
  neighbors: DesignLessonNeighbor[]
}
