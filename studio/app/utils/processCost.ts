import type { CostClass } from '~/types'

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
