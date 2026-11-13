import type { DesignColor } from './design'

/**
 * Pelajaran Desain, dianalisis LLM vision. Dua jenis:
 * - before_after: gambar awal + gambar akhir + narasi konten.
 * - theory: satu teori desain + keterangan opsional + contoh gambar (tunggal benar/salah, atau komparasi benar vs salah).
 */
export type DesignLessonKind = 'before_after' | 'theory'
export type DesignLessonSide = 'before' | 'after'
/** Contoh pelajaran teori: tunggal (satu gambar benar ATAU salah) atau komparasi (gambar benar + gambar salah). */
export type DesignLessonExampleMode = 'single' | 'compare'
export type DesignLessonSlot = 'good' | 'bad'
export type DesignLessonPlatform = 'graphic' | 'web' | 'android' | 'ios'
export type DesignLessonStatus = 'new' | 'analyzed' | 'failed'

export interface DesignLessonMistake {
  area: string
  principle: string
  problem: string
  evidence: string
}

export interface DesignLessonStrength {
  area: string
  point: string
  evidence: string
}

export interface DesignLessonExample {
  id: string
  mode: DesignLessonExampleMode
  note?: string
  good_file?: string
  bad_file?: string
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
  /** Pelajaran teori: penerapan teori yang benar pada contoh. */
  strengths?: DesignLessonStrength[]
  matched?: boolean
}

export interface DesignLesson {
  id: number
  kind: DesignLessonKind
  title: string
  narrative: string
  source_url: string
  platform: DesignLessonPlatform
  tags: string[]
  before_file?: string
  after_file?: string
  /** Pelajaran teori: id konsep katalog teori desain. */
  theory_id?: string
  examples: DesignLessonExample[]
  status: DesignLessonStatus
  last_error?: string
  analysis?: DesignLessonAnalysis
  analysis_model?: string
  analyzed_at?: string
  created_at: string
  updated_at: string
}

export interface DesignLessonInput {
  /** Hanya dipakai saat membuat. */
  kind?: DesignLessonKind
  theory_id?: string
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
