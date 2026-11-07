import type { AgentRecap } from '~/types'

export interface RecapHighlight {
  icon: string
  text: string
  tone: string
}

const fmt = (n: number) => n.toLocaleString('id-ID')

/** Kalimat pencapaian yang patut dibanggakan, hanya dari angka asli rekap (yang nol/negatif dilewati). */
export function recapHighlights(r: AgentRecap): RecapHighlight[] {
  const out: RecapHighlight[] = []
  const t = r.today
  const y = r.yesterday
  if (r.streak.current >= 2) {
    out.push({ icon: 'i-lucide-flame', tone: 'text-orange-500', text: `${r.streak.current} hari produktif beruntun${r.streak.current >= r.streak.longest ? ' — rekor terpanjang!' : ''}` })
  }
  if (t.commits > 0) {
    const lines = t.additions > 0 ? ` (+${fmt(t.additions)} baris)` : ''
    const up = t.commits > y.commits ? `, naik dari ${y.commits} kemarin` : ''
    out.push({ icon: 'i-lucide-git-commit-horizontal', tone: 'text-sky-500', text: `${t.commits} commit hari ini${lines}${up}` })
  }
  if (t.todos_completed > 0) {
    out.push({ icon: 'i-lucide-circle-check-big', tone: 'text-emerald-500', text: `${t.todos_completed} todo diselesaikan hari ini` })
  }
  if (t.todos_completed_by_ai > 0) {
    out.push({ icon: 'i-lucide-bot', tone: 'text-indigo-500', text: `Agent AI menuntaskan ${t.todos_completed_by_ai} todo hari ini` })
  }
  if (t.ai_commits > 0) {
    out.push({ icon: 'i-lucide-sparkles', tone: 'text-violet-500', text: `${t.ai_commits} commit hari ini ditulis bersama agent AI` })
  }
  if (t.runs_completed > 0) {
    out.push({ icon: 'i-lucide-network', tone: 'text-teal-500', text: `${t.runs_completed} run Orkestrator Agent selesai sukses` })
  }
  if (t.todos_created > 0) {
    out.push({ icon: 'i-lucide-lightbulb', tone: 'text-amber-500', text: `${t.todos_created} ide/todo baru dicatat hari ini` })
  }
  const net = r.totals.todos_completed - r.totals.todos_created
  if (r.totals.todos_completed > 0 && net >= 0) {
    out.push({ icon: 'i-lucide-trending-down', tone: 'text-emerald-500', text: `Backlog terkendali: ${fmt(r.totals.todos_completed)} todo selesai vs ${fmt(r.totals.todos_created)} dibuat` })
  }
  const prio = (r.priorities ?? []).filter(p => p.key === 'critical' || p.key === 'high').reduce((s, p) => s + p.count, 0)
  if (prio > 0) {
    out.push({ icon: 'i-lucide-shield-check', tone: 'text-rose-500', text: `${prio} todo prioritas tinggi/kritis berhasil ditutup` })
  }
  const security = (r.categories ?? []).find(c => c.key === 'security')?.count ?? 0
  if (security > 0) {
    out.push({ icon: 'i-lucide-lock', tone: 'text-rose-500', text: `${security} perbaikan keamanan selesai` })
  }
  if (r.best_day && r.best_day.date === r.to && (r.best_day.commits > 0 || r.best_day.todos_completed > 0)) {
    out.push({ icon: 'i-lucide-trophy', tone: 'text-yellow-500', text: 'Hari ini adalah hari terbaik di rentang rekap!' })
  }
  const repos = (r.repos ?? []).filter(x => x.today_commits > 0).length
  if (repos >= 2) {
    out.push({ icon: 'i-lucide-folder-git-2', tone: 'text-sky-500', text: `Kemajuan di ${repos} repo sekaligus hari ini` })
  }
  return out
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
