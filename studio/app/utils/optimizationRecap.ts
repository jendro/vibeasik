// Rekap optimasi per project: todo yang sudah selesai dikelompokkan menurut kategori.
import { CATEGORY_LABEL } from './recap'

export interface RecapTodo {
  id: number
  title: string
  category?: string
  status: string
  priority?: string
  component_name?: string
  completed_at?: string
}

export interface CategoryRecap {
  category: string
  label: string
  done: number
  open: number
  percent: number
  latest: RecapTodo[]
}

export interface OptimizationRecap {
  done: number
  open: number
  total: number
  percent: number
  categories: CategoryRecap[]
}

// Status yang masih dikerjakan/menunggu; rejected, merged, split tidak dihitung (bukan beban kerja nyata).
const OPEN_STATUSES = new Set(['pending', 'in_progress'])

const labelOf = (category: string) => CATEGORY_LABEL[category] || CATEGORY_LABEL.general!

const pct = (done: number, total: number) => (total > 0 ? Math.round((done / total) * 100) : 0)

const byCompletedDesc = (a: RecapTodo, b: RecapTodo) =>
  (b.completed_at || '').localeCompare(a.completed_at || '') || b.id - a.id

/** Susun rekap dari daftar todo satu project; kategori terbanyak selesai di atas, kategori tanpa todo dilewati. */
export function buildOptimizationRecap(todos: RecapTodo[], latestPerCategory = 5): OptimizationRecap {
  const groups = new Map<string, { done: RecapTodo[], open: number }>()
  for (const t of todos) {
    const completed = t.status === 'completed'
    if (!completed && !OPEN_STATUSES.has(t.status)) continue
    const key = t.category && CATEGORY_LABEL[t.category] ? t.category : 'general'
    const g = groups.get(key) ?? { done: [], open: 0 }
    if (completed) g.done.push(t)
    else g.open++
    groups.set(key, g)
  }

  const categories: CategoryRecap[] = [...groups.entries()]
    .map(([category, g]) => ({
      category,
      label: labelOf(category),
      done: g.done.length,
      open: g.open,
      percent: pct(g.done.length, g.done.length + g.open),
      latest: [...g.done].sort(byCompletedDesc).slice(0, latestPerCategory)
    }))
    .sort((a, b) => b.done - a.done || a.label.localeCompare(b.label))

  const done = categories.reduce((n, c) => n + c.done, 0)
  const open = categories.reduce((n, c) => n + c.open, 0)
  return { done, open, total: done + open, percent: pct(done, done + open), categories }
}
