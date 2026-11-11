import type { OwnerRecap } from '~/types'

// Kalimat & label rekap pemilik (murni, tanpa state) agar bisa diuji tanpa Nuxt.
const dayLabel = (iso: string) => {
  const [, m, d] = iso.split('-')
  return `${Number(d)}/${Number(m)}`
}

/** Label sumbu X tanggal singkat (11/3) dari YYYY-MM-DD. */
export const recapDayLabels = (dates: string[]) => dates.map(dayLabel)

/** Satu kalimat yang bisa dibaca pemilik tanpa membuka angka lain. */
export function recapHeadline(r: OwnerRecap): string {
  const t = r.todos
  if (t.total === 0) return 'Belum ada pekerjaan tercatat di proyek ini. Tambahkan todo atau jalankan audit untuk mulai merekam kemajuan.'
  const parts = [`${t.done_in_period} pekerjaan selesai`]
  if (r.bugs.done_in_period > 0) parts.push(`${r.bugs.done_in_period} bug ditutup`)
  if (r.algorithm.done_in_period > 0) parts.push(`${r.algorithm.done_in_period} perbaikan algoritma`)
  if (r.findings.resolved_in_period > 0) parts.push(`${r.findings.resolved_in_period} temuan audit dibereskan`)
  const rest = t.open === 0
    ? 'Tidak ada pekerjaan yang tertunda.'
    : `${t.open} masih terbuka${t.open_high_priority ? `, ${t.open_high_priority} di antaranya prioritas tinggi` : ''}.`
  return `Dalam ${r.days} hari terakhir: ${parts.join(', ')}. ${rest}`
}
