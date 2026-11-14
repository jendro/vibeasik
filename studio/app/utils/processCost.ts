import type { CostClass, CostHandling, CostHandlingSummary } from '~/types'

// Kelas pertumbuhan: label manusiawi + warna status. Warna selalu ditemani label (tidak bergantung warna saja).
export const COST_CLASSES: { key: CostClass, bigO: string, label: string, meaning: string, bar: string, text: string }[] = [
  { key: 'constant', bigO: 'O(1)', label: 'Tetap', meaning: 'Biaya tidak naik walau data bertambah.', bar: 'bg-emerald-500', text: 'text-emerald-700 dark:text-emerald-400' },
  { key: 'log', bigO: 'O(log n)', label: 'Logaritmik', meaning: 'Naik sangat lambat (pencarian biner).', bar: 'bg-emerald-400', text: 'text-emerald-700 dark:text-emerald-400' },
  { key: 'linear', bigO: 'O(n)', label: 'Linear', meaning: 'Data 10× → biaya 10×. Wajar untuk daftar.', bar: 'bg-sky-500', text: 'text-sky-700 dark:text-sky-400' },
  { key: 'linearithmic', bigO: 'O(n log n)', label: 'Pengurutan', meaning: 'Sedikit di atas linear (sort).', bar: 'bg-indigo-500', text: 'text-indigo-700 dark:text-indigo-400' },
  { key: 'quadratic', bigO: 'O(n²)', label: 'Kuadratik', meaning: 'Data 10× → biaya 100×. Loop di dalam loop.', bar: 'bg-amber-500', text: 'text-amber-700 dark:text-amber-400' },
  { key: 'cubic', bigO: 'O(n³)', label: 'Kubik', meaning: 'Data 10× → biaya 1.000×. Perlu dirombak.', bar: 'bg-rose-500', text: 'text-rose-700 dark:text-rose-400' },
  { key: 'polynomial', bigO: 'O(nᵏ)', label: 'Polinomial tinggi', meaning: 'Tumbuh sangat cepat. Perlu dirombak.', bar: 'bg-rose-600', text: 'text-rose-700 dark:text-rose-400' }
]

export const costClassInfo = (key: string) => COST_CLASSES.find(c => c.key === key) ?? COST_CLASSES[0]!

/** Angka besar ringkas: 30205 → 30,2 rb; 3e8 → 300 jt. */
export const fmtTicks = (v: number) => new Intl.NumberFormat('id-ID', { notation: v >= 100000 ? 'compact' : 'standard', maximumFractionDigits: 1 }).format(Math.round(v))

/** Rumus panjang diringkas ke suku-suku tertinggi (penentu pertumbuhan); rumus utuh tampil saat baris dibuka. */
export const shortFormula = (formula: string, terms = 3) => {
  const parts = formula.split(' + ')
  return parts.length <= terms + 1 ? formula : `${parts.slice(0, terms).join(' + ')} + …`
}

/** Label bahasa/framework baris biaya: Go, Laravel, CodeIgniter 4, PHP … */
export const costStackLabel = (r: { lang?: string, framework?: string }) =>
  r.lang === 'php' ? (r.framework || 'PHP') : 'Go'

const TODO_STATUS: Record<string, string> = { pending: 'menunggu', in_progress: 'dikerjakan', suggested: 'saran', completed: 'selesai' }

// Penanda penanganan satu baris: di mana sumber biayanya sudah ditangani (label pendek + warna + ikon).
export const costHandledBadge = (h: CostHandling) => {
  switch (h.where) {
    case 'todo':
      return { label: `Todo #${h.todo_id} · ${TODO_STATUS[h.todo_status || ''] || h.todo_status}`, icon: 'i-lucide-list-checks', tone: 'bg-sky-50 text-sky-700 dark:bg-sky-950/40 dark:text-sky-300' }
    case 'audit':
      return { label: 'Di Audit', icon: 'i-lucide-shield-check', tone: 'bg-amber-50 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300' }
    case 'ignored':
      return { label: 'Bukan masalah', icon: 'i-lucide-circle-slash', tone: 'bg-neutral-100 text-neutral-600 dark:bg-white/[0.06] dark:text-neutral-300' }
  }
  return { label: h.todo_id ? `Cek ulang #${h.todo_id}` : 'Cek ulang', icon: 'i-lucide-rotate-ccw', tone: 'bg-rose-50 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300' }
}

// Keterangan di atas tabel: sudah ditangani di mana, lalu langkah yang perlu dilakukan.
export const costHandlingNote = (s: CostHandlingSummary) => {
  const handled = s.in_todo + s.in_audit + s.ignored + s.recheck
  const where: string[] = []
  if (s.in_todo) {
    const more = s.todo_count > s.todos.length ? `, +${s.todo_count - s.todos.length} lain` : ''
    where.push(`${s.in_todo} di ${s.todo_count} todo${s.todos.length ? ` (${s.todos.map(t => `#${t.id}`).join(', ')}${more})` : ''}`)
  }
  if (s.in_audit) where.push(`${s.in_audit} di temuan Audit`)
  if (s.ignored) where.push(`${s.ignored} ditandai bukan masalah`)
  if (s.recheck) where.push(`${s.recheck} perlu dicek ulang`)
  const todo: string[] = []
  if (s.in_todo) todo.push('kerjakan todo-nya di halaman Todo atau kirim ke agent coding')
  if (s.in_audit) todo.push('buka Audit lalu klik Selesaikan pada temuannya agar menjadi todo')
  if (s.recheck) todo.push('jalankan Audit ulang untuk yang sudah selesai tetapi rumusnya masih tinggi')
  if (s.open) todo.push(`buat todo untuk ${s.open} sumber yang belum ditangani lewat tombol di barisnya`)
  return {
    title: handled ? `Sudah ditangani ${handled} dari ${s.sources} sumber biaya tinggi` : `${s.sources} sumber biaya tinggi belum ditangani`,
    where: where.join(' · '),
    next: todo.length ? `Lakukan: ${todo.join('; ')}.` : 'Tidak ada yang perlu dilakukan.'
  }
}
