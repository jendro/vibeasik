// Satu patokan warna & label skor kesehatan audit untuk seluruh Studio (Beranda, Audit, kartu proyek).
// null = belum pernah diaudit (jangan ditampilkan sebagai 100).
export const HEALTH_GOOD = 90
export const HEALTH_WARN = 70

export const healthTone = (s: number | null | undefined) => s == null
  ? 'text-neutral-400 dark:text-slate-500'
  : s >= HEALTH_GOOD ? 'text-emerald-600 dark:text-emerald-400' : s >= HEALTH_WARN ? 'text-amber-600 dark:text-amber-400' : 'text-rose-600 dark:text-rose-400'

export const healthBar = (s: number | null | undefined) => s == null
  ? 'bg-neutral-300 dark:bg-white/10'
  : s >= HEALTH_GOOD ? 'bg-emerald-500' : s >= HEALTH_WARN ? 'bg-amber-500' : 'bg-rose-500'

export const healthLabel = (s: number | null | undefined) => s == null
  ? 'Belum diaudit'
  : s >= HEALTH_GOOD ? 'Sehat · siap produksi' : s >= HEALTH_WARN ? 'Perlu perbaikan ringan' : 'Kritis · banyak temuan'
