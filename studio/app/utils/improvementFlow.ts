import type { OwnerRecap } from '~/types'

// Area perbaikan temuan audit — sama dengan diagnostics.Areas di engine (kunci open_by_area di rekap).
export const FINDING_AREAS = [
  { key: 'speed', label: 'Performa & trafik', icon: 'i-lucide-gauge', desc: 'Query/N+1, Big-O, latensi rute, ukuran aset, rate limit' },
  { key: 'data', label: 'Kualitas data', icon: 'i-lucide-database', desc: 'Validasi input & transaksi' },
  { key: 'ui', label: 'Tampilan', icon: 'i-lucide-panels-top-left', desc: 'Frontend, mobile, halaman rusak saat dijalankan' },
  { key: 'security', label: 'Keamanan', icon: 'i-lucide-shield', desc: 'Injeksi, rute tanpa proteksi, rahasia, CORS' },
  { key: 'code', label: 'Kode', icon: 'i-lucide-code', desc: 'Kompleksitas, duplikasi, struktur' },
  { key: 'ops', label: 'Operasional', icon: 'i-lucide-server-cog', desc: 'Git, env, port, skrip jalan & deploy' }
] as const

export type ImprovementStep = 'check' | 'work' | 'result'

export interface NextImprovementStep {
  step: ImprovementStep
  /** tombol utama di langkah check: periksa ulang (scan) atau olah temuan (process) */
  action?: 'scan' | 'process'
  title: string
  reason: string
  icon: string
}

/** Pemeriksaan dianggap basi setelah sekian hari: kode sudah berubah, temuan lama mungkin tidak berlaku. */
export const STALE_AUDIT_DAYS = 7

// Langkah yang perlu dikerjakan sekarang: periksa bila belum/basi, olah temuan yang belum jadi todo,
// kerjakan todo terbuka, selain itu lihat hasilnya.
export const nextImprovementStep = (r: OwnerRecap, now: number): NextImprovementStep => {
  const at = r.findings.last_audit_at ? new Date(r.findings.last_audit_at).getTime() : 0
  const ageDays = at ? Math.floor((now - at) / 86_400_000) : Infinity
  if (!at) {
    return { step: 'check', action: 'scan', icon: 'i-lucide-scan-search', title: 'periksa menyeluruh', reason: 'Proyek ini belum pernah diperiksa, jadi belum ada temuan untuk diperbaiki.' }
  }
  if (ageDays >= STALE_AUDIT_DAYS) {
    return { step: 'check', action: 'scan', icon: 'i-lucide-scan-search', title: 'periksa ulang', reason: `Pemeriksaan terakhir ${ageDays} hari lalu; kode mungkin sudah berubah.` }
  }
  if (r.findings.open > 0) {
    const crit = r.findings.open_critical ? `, ${r.findings.open_critical} di antaranya kritis` : ''
    return { step: 'check', action: 'process', icon: 'i-lucide-list-checks', title: 'tinjau temuan', reason: `${r.findings.open} temuan belum dijadikan todo${crit}.` }
  }
  if (r.todos.open > 0) {
    const high = r.todos.open_high_priority ? ` (${r.todos.open_high_priority} prioritas tinggi)` : ''
    return { step: 'work', icon: 'i-lucide-clipboard-check', title: 'kerjakan todo', reason: `${r.todos.open} todo terbuka${high} menunggu dikerjakan.` }
  }
  return { step: 'result', icon: 'i-lucide-badge-check', title: 'lihat hasil', reason: 'Semua temuan sudah diproses dan tidak ada todo terbuka.' }
}
