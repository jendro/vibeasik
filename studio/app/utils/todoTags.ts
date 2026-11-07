// Tag fokus siklus pengembangan — sumber kebenaran di engine: local-engine/internal/models/todo_tags.go
export interface TodoTagMeta {
  key: string
  label: string
  icon: string
  class: string
}

export const TODO_FOCUS_TAGS: TodoTagMeta[] = [
  { key: 'fitur', label: 'Fitur', icon: 'i-lucide-sparkles', class: 'bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300 border-amber-200 dark:border-amber-900/60' },
  { key: 'keamanan-dasar', label: 'Keamanan Dasar', icon: 'i-lucide-shield', class: 'bg-rose-50 dark:bg-rose-950/40 text-rose-700 dark:text-rose-300 border-rose-200 dark:border-rose-900/60' },
  { key: 'optimasi-query', label: 'Optimasi Query', icon: 'i-lucide-database-zap', class: 'bg-sky-50 dark:bg-sky-950/40 text-sky-700 dark:text-sky-300 border-sky-200 dark:border-sky-900/60' },
  { key: 'rawat-ai', label: 'Rawat AI', icon: 'i-lucide-bot', class: 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300 border-emerald-200 dark:border-emerald-900/60' },
  { key: 'belajar', label: 'Belajar', icon: 'i-lucide-graduation-cap', class: 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 border-neutral-200 dark:border-white/10' }
]

export const parseTodoTags = (tags?: string): string[] =>
  (tags || '').split(',').map(t => t.trim()).filter(Boolean)

export const todoTagMeta = (key: string): TodoTagMeta =>
  TODO_FOCUS_TAGS.find(t => t.key === key)
  || { key, label: key, icon: 'i-lucide-tag', class: 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 border-neutral-200 dark:border-white/10' }
