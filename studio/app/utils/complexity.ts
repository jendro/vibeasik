// Tampilan tingkat kerumitan todo.
export const complexityMeta = (level?: string) => {
  switch (level) {
    case 'quick':
      return { label: 'Ringan', icon: 'i-lucide-feather', class: 'text-emerald-700 bg-emerald-50 dark:bg-emerald-950/40 dark:text-emerald-300 border-emerald-200 dark:border-emerald-800' }
    case 'complex':
      return { label: 'Rumit', icon: 'i-lucide-brain-circuit', class: 'text-rose-700 bg-rose-50 dark:bg-rose-950/40 dark:text-rose-300 border-rose-200 dark:border-rose-800' }
    case 'normal':
      return { label: 'Sedang', icon: 'i-lucide-gauge', class: 'text-amber-700 bg-amber-50 dark:bg-amber-950/40 dark:text-amber-300 border-amber-200 dark:border-amber-800' }
    default:
      return { label: 'Belum dinilai', icon: 'i-lucide-circle-help', class: 'text-neutral-600 bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-400 border-neutral-200 dark:border-neutral-700' }
  }
}
