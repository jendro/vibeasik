// Label tampilan Pusat Pengetahuan (jenis node, asal, status celah).
import type { KnowledgeGapStatus, KnowledgeKind, KnowledgeSource } from '~/types'

type BadgeColor = 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'

export const KIND_META: Record<KnowledgeKind, { label: string, icon: string, color: BadgeColor }> = {
  rule: { label: 'Patokan', icon: 'i-lucide-scale', color: 'primary' },
  procedure: { label: 'Prosedur', icon: 'i-lucide-list-checks', color: 'info' },
  lesson: { label: 'Pelajaran', icon: 'i-lucide-lightbulb', color: 'warning' },
  concept: { label: 'Konsep', icon: 'i-lucide-book-open', color: 'secondary' }
}

export const SOURCE_LABEL: Record<KnowledgeSource, string> = {
  guideline: 'Guideline',
  audit_checklist: 'Checklist audit',
  audit_verdict: 'Vonis audit',
  employee: 'Keahlian pegawai',
  promoted: 'Dipromosikan',
  gap_research: 'Belajar teknologi baru',
  qa_review: 'Best practice penilai QA',
  design_library: 'Galeri Desain',
  design_theory: 'Teori Desain & Warna',
  design_lesson: 'Pelajaran Desain',
  manual: 'Diajarkan manusia',
  agent: 'Agent MCP',
  hub: 'Paket komunitas'
}

export const GAP_STATUS: Record<KnowledgeGapStatus, { label: string, color: BadgeColor }> = {
  open: { label: 'Terbuka', color: 'warning' },
  learning: { label: 'Sedang dipelajari', color: 'info' },
  learned: { label: 'Sudah dipelajari', color: 'success' },
  dismissed: { label: 'Diabaikan', color: 'neutral' }
}
