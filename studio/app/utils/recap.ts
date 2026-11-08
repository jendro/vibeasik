import type { AgentRecap } from '~/types'

export interface RecapHighlight {
  icon: string
  text: string
  tone: string
}

const fmt = (n: number) => n.toLocaleString('id-ID')

type Highlighter = (r: AgentRecap) => RecapHighlight | null

const hl = (icon: string, tone: string, text: string): RecapHighlight => ({ icon, tone, text })

const streakHighlight: Highlighter = (r) => {
  const { current, longest } = r.streak
  if (current < 2) return null
  return hl('i-lucide-flame', 'text-orange-500', `${current} hari produktif beruntun${current >= longest ? ' — rekor terpanjang!' : ''}`)
}

const commitsHighlight: Highlighter = (r) => {
  const t = r.today
  if (t.commits <= 0) return null
  const lines = t.additions > 0 ? ` (+${fmt(t.additions)} baris)` : ''
  const up = t.commits > r.yesterday.commits ? `, naik dari ${r.yesterday.commits} kemarin` : ''
  return hl('i-lucide-git-commit-horizontal', 'text-sky-500', `${t.commits} commit hari ini${lines}${up}`)
}

const todosDoneHighlight: Highlighter = r =>
  r.today.todos_completed > 0 ? hl('i-lucide-circle-check-big', 'text-emerald-500', `${r.today.todos_completed} todo diselesaikan hari ini`) : null

const aiTodosHighlight: Highlighter = r =>
  r.today.todos_completed_by_ai > 0 ? hl('i-lucide-bot', 'text-indigo-500', `Agent AI menuntaskan ${r.today.todos_completed_by_ai} todo hari ini`) : null

const aiCommitsHighlight: Highlighter = r =>
  r.today.ai_commits > 0 ? hl('i-lucide-sparkles', 'text-violet-500', `${r.today.ai_commits} commit hari ini ditulis bersama agent AI`) : null

const runsHighlight: Highlighter = r =>
  r.today.runs_completed > 0 ? hl('i-lucide-network', 'text-teal-500', `${r.today.runs_completed} run Orkestrator Agent selesai sukses`) : null

const ideasHighlight: Highlighter = r =>
  r.today.todos_created > 0 ? hl('i-lucide-lightbulb', 'text-amber-500', `${r.today.todos_created} ide/todo baru dicatat hari ini`) : null

const backlogHighlight: Highlighter = (r) => {
  const { todos_completed: done, todos_created: created } = r.totals
  if (done <= 0 || done - created < 0) return null
  return hl('i-lucide-trending-down', 'text-emerald-500', `Backlog terkendali: ${fmt(done)} todo selesai vs ${fmt(created)} dibuat`)
}

const priorityHighlight: Highlighter = (r) => {
  const prio = (r.priorities ?? []).filter(p => p.key === 'critical' || p.key === 'high').reduce((s, p) => s + p.count, 0)
  return prio > 0 ? hl('i-lucide-shield-check', 'text-rose-500', `${prio} todo prioritas tinggi/kritis berhasil ditutup`) : null
}

const securityHighlight: Highlighter = (r) => {
  const security = (r.categories ?? []).find(c => c.key === 'security')?.count ?? 0
  return security > 0 ? hl('i-lucide-lock', 'text-rose-500', `${security} perbaikan keamanan selesai`) : null
}

const bestDayHighlight: Highlighter = (r) => {
  const b = r.best_day
  if (!b || b.date !== r.to || (b.commits <= 0 && b.todos_completed <= 0)) return null
  return hl('i-lucide-trophy', 'text-yellow-500', 'Hari ini adalah hari terbaik di rentang rekap!')
}

const reposHighlight: Highlighter = (r) => {
  const repos = (r.repos ?? []).filter(x => x.today_commits > 0).length
  return repos >= 2 ? hl('i-lucide-folder-git-2', 'text-sky-500', `Kemajuan di ${repos} repo sekaligus hari ini`) : null
}

const HIGHLIGHTERS: Highlighter[] = [
  streakHighlight, commitsHighlight, todosDoneHighlight, aiTodosHighlight, aiCommitsHighlight, runsHighlight,
  ideasHighlight, backlogHighlight, priorityHighlight, securityHighlight, bestDayHighlight, reposHighlight
]

/** Kalimat pencapaian yang patut dibanggakan, hanya dari angka asli rekap (yang nol/negatif dilewati). */
export function recapHighlights(r: AgentRecap): RecapHighlight[] {
  return HIGHLIGHTERS.map(h => h(r)).filter((x): x is RecapHighlight => x !== null)
}

export const CATEGORY_LABEL: Record<string, string> = {
  feature: 'Fitur',
  security: 'Keamanan',
  ui_ux: 'UI/UX',
  architecture: 'Arsitektur',
  performance: 'Performa',
  backend: 'Backend',
  devops: 'DevOps',
  general: 'Umum'
}

export const PRIORITY_LABEL: Record<string, string> = { critical: 'Kritis', high: 'Tinggi', medium: 'Sedang', low: 'Rendah' }
