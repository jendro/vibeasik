import type { ImpactDimensionsInfo, Todo } from '~/types'

/** Definisi dimensi dampak (sekali muat per sesi) + helper tampilan badge dampak todo. */
export const useTodoImpact = () => {
  const { apiBase } = useApi()
  const info = useState<ImpactDimensionsInfo | null>('todo-impact-dimensions', () => null)
  const loading = useState<Promise<void> | null>('todo-impact-dimensions-loading', () => null)

  if (import.meta.client && !info.value && !loading.value) {
    loading.value = $fetch<ImpactDimensionsInfo>(`${apiBase}/api/todos/impact-dimensions`)
      .then((res) => { info.value = res })
      .catch(() => { loading.value = null })
  }

  /** Nilai dimensi todo dengan urutan `info.dimensions` */
  const values = (t: Todo) => (info.value?.dimensions || []).map(d => Number(t[`impact_${d.key}` as keyof Todo]) || 0)

  return { info, values }
}

export const impactMeta = (level?: string) => {
  switch (level) {
    case 'high':
      return { label: 'Dampak Besar', icon: 'i-lucide-rocket', class: 'text-violet-700 bg-violet-50 dark:bg-violet-950/40 dark:text-violet-300 border-violet-200 dark:border-violet-800' }
    case 'medium':
      return { label: 'Dampak Sedang', icon: 'i-lucide-trending-up', class: 'text-sky-700 bg-sky-50 dark:bg-sky-950/40 dark:text-sky-300 border-sky-200 dark:border-sky-800' }
    case 'low':
      return { label: 'Dampak Kecil', icon: 'i-lucide-arrow-up-right', class: 'text-teal-700 bg-teal-50 dark:bg-teal-950/40 dark:text-teal-300 border-teal-200 dark:border-teal-800' }
    case 'minimal':
      return { label: 'Dampak Minim', icon: 'i-lucide-minus', class: 'text-neutral-600 bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-400 border-neutral-200 dark:border-neutral-700' }
    default:
      return { label: 'Dampak belum dinilai', icon: 'i-lucide-circle-help', class: 'text-neutral-600 bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-400 border-neutral-200 dark:border-neutral-700' }
  }
}

export const impactAudienceLabel: Record<string, string> = {
  ai_agent: 'Kode · untuk AI Agent',
  app: 'Kualitas Aplikasi',
  human: 'Pengguna (Manusia)'
}

/** Dampak per satuan usaha: skor dampak dibagi skor kerumitan (1–5). */
export const impactPerEffort = (t: Todo) => {
  const imp = Number(t.impact_score) || 0
  const comp = Number(t.complexity_score) || 0
  return comp > 0 ? imp / comp : imp
}
