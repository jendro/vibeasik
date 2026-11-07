import type { ActorMood, AgentWorker, OfficeEmployee, OfficeEmployeeStatus } from '~/types'
import { breaksInGarden } from '~/utils/officeLayout'

/** Gaya karakter piksel per agent / penyedia LLM. */
export interface AgentLook {
  shirt: string
  hair: string
  skin: string
  pants: string
  accessory: 'none' | 'glasses' | 'headset' | 'cap' | 'beanie' | 'tie' | 'scarf'
  /** Gaya rambut pegawai (agent luar memakai `short`). */
  hairStyle?: 'short' | 'long' | 'bun' | 'bald' | 'curly' | 'hijab' | 'ponytail' | 'spiky'
  facial?: 'none' | 'beard' | 'mustache'
  icon: string
  label: string
}

const SKINS = ['#f2c7a5', '#e0ac85', '#c68863', '#8d5a3b', '#f5d6bf']
const HAIRS = ['#2b1d16', '#5a3a22', '#111827', '#a0522d', '#d6b16b', '#6b7280']

const LOOKS: Array<{ match: RegExp, look: Partial<AgentLook> }> = [
  { match: /anak buah/i, look: { pants: '#1e293b', accessory: 'cap', icon: 'i-lucide-hard-hat', label: 'Anak Buah' } },
  { match: /claude code/i, look: { shirt: '#d97757', hair: '#3b2a20', accessory: 'headset', icon: 'i-simple-icons-claude', label: 'Claude Code' } },
  { match: /claude desktop/i, look: { shirt: '#c96442', hair: '#5a3a22', accessory: 'none', icon: 'i-simple-icons-claude', label: 'Claude Desktop' } },
  { match: /antigravity/i, look: { shirt: '#4285f4', hair: '#111827', accessory: 'glasses', icon: 'i-simple-icons-google', label: 'Antigravity' } },
  { match: /gemini/i, look: { shirt: '#8e75ff', hair: '#1f2937', accessory: 'beanie', icon: 'i-simple-icons-googlegemini', label: 'Gemini CLI' } },
  { match: /cursor/i, look: { shirt: '#1f2937', hair: '#e5e7eb', accessory: 'glasses', icon: 'i-simple-icons-cursor', label: 'Cursor' } },
  { match: /windsurf/i, look: { shirt: '#0ea5a4', hair: '#5a3a22', accessory: 'none', icon: 'i-simple-icons-windsurf', label: 'Windsurf' } },
  { match: /copilot cli/i, look: { shirt: '#8957e5', hair: '#5a3a22', accessory: 'beanie', icon: 'i-simple-icons-githubcopilot', label: 'Copilot CLI' } },
  { match: /copilot/i, look: { shirt: '#6e40c9', hair: '#111827', accessory: 'headset', icon: 'i-simple-icons-githubcopilot', label: 'Copilot' } },
  { match: /aider/i, look: { shirt: '#16a34a', hair: '#a0522d', accessory: 'beanie', icon: 'i-lucide-terminal', label: 'Aider' } },
  { match: /cline/i, look: { shirt: '#7c3aed', hair: '#d6b16b', accessory: 'none', icon: 'i-lucide-bot', label: 'Cline' } },
  { match: /codex/i, look: { shirt: '#374151', hair: '#111827', accessory: 'glasses', icon: 'i-simple-icons-openai', label: 'Codex' } }
]

const PROVIDER_LOOKS: Array<{ match: RegExp, color: string, icon: string }> = [
  { match: /openai/i, color: '#10a37f', icon: 'i-simple-icons-openai' },
  { match: /anthropic/i, color: '#c96442', icon: 'i-simple-icons-anthropic' },
  { match: /gemini|google/i, color: '#8e75ff', icon: 'i-simple-icons-googlegemini' },
  { match: /openrouter/i, color: '#6467f2', icon: 'i-simple-icons-openrouter' },
  { match: /ollama/i, color: '#475569', icon: 'i-simple-icons-ollama' },
  { match: /deepseek/i, color: '#4d6bfe', icon: 'i-simple-icons-deepseek' },
  { match: /mistral/i, color: '#fa520f', icon: 'i-simple-icons-mistralai' },
  { match: /system one/i, color: '#f59e0b', icon: 'i-lucide-cpu' }
]

/** Tim perencana vibeasik: Project Manager & spesialis rapat, dikenali dari client_name (key peran). */
const PLANNER_LOOKS: Record<string, Partial<AgentLook>> = {
  pm: { shirt: '#1e293b', pants: '#0f172a', hair: '#6b7280', accessory: 'tie', icon: 'i-lucide-briefcase', label: 'Project Manager' },
  business: { shirt: '#059669', hair: '#2b1d16', accessory: 'glasses', icon: 'i-lucide-chart-line', label: 'Analis Bisnis' },
  design: { shirt: '#db2777', hair: '#a0522d', accessory: 'beanie', icon: 'i-lucide-palette', label: 'Desainer UI/UX' },
  architect: { shirt: '#2563eb', hair: '#111827', accessory: 'glasses', icon: 'i-lucide-blocks', label: 'Arsitek Teknis' },
  security: { shirt: '#b91c1c', hair: '#111827', accessory: 'cap', icon: 'i-lucide-shield-check', label: 'Spesialis Keamanan' },
  qa: { shirt: '#7c3aed', hair: '#d6b16b', accessory: 'headset', icon: 'i-lucide-flask-conical', label: 'QA & Reliability' },
  devops: { shirt: '#ea580c', hair: '#5a3a22', accessory: 'cap', icon: 'i-lucide-rocket', label: 'DevOps & Rilis' }
}

export const PLANNER_ROLE_ICON = (role: string) => PLANNER_LOOKS[role]?.icon ?? 'i-lucide-user-round'

function hash(s: string): number {
  let h = 0
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0
  return h
}

/** Mode divisi: koordinator mengambil todo dan menyuruh sub agent (anak buah) mengerjakannya. */
export const COORDINATOR_ROLE = 'Pengambil todo'
export const isCoordinator = (w: Pick<AgentWorker, 'role'>) => w.role === COORDINATOR_ROLE
export const isSubagent = (w: Pick<AgentWorker, 'parent_key'>) => !!w.parent_key

/** Penampilan karakter untuk seorang pekerja (stabil untuk nama/key yang sama). */
export function agentLook(w: Pick<AgentWorker, 'kind' | 'name' | 'key'> & { client_name?: string, role?: string }): AgentLook {
  const h = hash(w.key || w.name)
  const base: AgentLook = {
    shirt: `hsl(${h % 360} 55% 48%)`,
    hair: HAIRS[h % HAIRS.length]!,
    skin: SKINS[(h >>> 3) % SKINS.length]!,
    pants: '#334155',
    accessory: 'none',
    icon: 'i-lucide-bot',
    label: w.name || 'Agent'
  }
  if (w.kind === 'llm_call') {
    const p = PROVIDER_LOOKS.find(p => p.match.test(w.name))
    return { ...base, shirt: p?.color ?? '#64748b', pants: '#1e293b', accessory: 'cap', icon: p?.icon ?? 'i-lucide-sparkles' }
  }
  if (w.kind === 'planner') return { ...base, ...PLANNER_LOOKS[w.client_name ?? ''] }
  const known = LOOKS.find(l => l.match.test(w.name))
  const look = { ...base, ...known?.look }
  return isCoordinator(w) ? { ...look, accessory: 'tie', icon: 'i-lucide-clipboard-list' } : look
}

/** Penampilan pegawai vibeasik dari avatar yang disimpan engine. */
export function employeeLook(e: Pick<OfficeEmployee, 'avatar' | 'title'>): AgentLook {
  const a = e.avatar
  return {
    shirt: a.shirt, hair: a.hair, skin: a.skin, pants: a.pants, accessory: a.accessory, hairStyle: a.hair_style, facial: a.facial,
    icon: 'i-lucide-id-card', label: e.title
  }
}

const EMPLOYEE_ACTION: Record<OfficeEmployeeStatus, string> = {
  idle: 'Siap menerima tugas',
  working: 'Mengerjakan todo',
  meeting: 'Rapat perencanaan',
  reviewing: 'Menilai hasil kerja rekan',
  learning: 'Memperbarui pengetahuan',
  chatting: 'Membalas obrolan…',
  social: 'Mengobrol santai',
  break: '☕ Ngopi di pantry',
  studying: 'Mengerjakan agenda pribadi'
}

/** Pegawai yang rehat di taman (lihat `breaksInGarden`) tidak sedang di pantry. */
const breakAction = (e: OfficeEmployee) => e.status === 'break' && breaksInGarden(e.id) ? '🌳 Rehat di taman' : ''

/**
 * Pegawai sebagai "pekerja" yang digambar: bila mewakili pekerja nyata (agent coding / CLI / peserta rapat),
 * status & aksinya diambil dari pekerja itu; bila tidak, dari keadaan pegawai (santai, review QA, obrolan).
 */
export function employeeWorker(e: OfficeEmployee, src: AgentWorker | undefined, nowIso: string): AgentWorker {
  if (src) {
    return { ...src, key: `emp:${e.id}`, name: e.name, role: e.title, parent_key: undefined, task: e.todo_title || src.task }
  }
  return {
    id: 0, key: `emp:${e.id}`, kind: 'employee', name: e.name, client_name: e.role, role: e.title,
    status: e.status === 'idle' || e.status === 'break' ? 'idle' : 'working', action: breakAction(e) || e.action || EMPLOYEE_ACTION[e.status], task: e.todo_title, todo_id: e.todo_id,
    tool_calls: 0, tasks_done: 0, started_at: nowIso, last_seen_at: nowIso
  }
}

/** Mood karakter: status dari engine + lama tanpa aktivitas (jam server). */
export function actorMood(w: AgentWorker, serverNowMs: number, idleAfterSec: number): ActorMood {
  if (w.status === 'done' || w.status === 'error' || w.status === 'left') return w.status
  if (w.kind !== 'coding_agent') return 'working'
  const quietSec = (serverNowMs - Date.parse(w.last_seen_at)) / 1000
  if (w.status === 'idle' || quietSec > idleAfterSec) {
    return quietSec > 600 ? 'sleeping' : 'idle'
  }
  return 'working'
}

export const MOOD_LABEL: Record<ActorMood, string> = {
  working: 'Bekerja',
  idle: 'Santai',
  sleeping: 'Tertidur',
  done: 'Selesai',
  error: 'Gagal',
  left: 'Pulang'
}

export const MOOD_COLOR: Record<ActorMood, 'success' | 'warning' | 'neutral' | 'info' | 'error'> = {
  working: 'success',
  idle: 'warning',
  sleeping: 'neutral',
  done: 'info',
  error: 'error',
  left: 'neutral'
}

export const KIND_LABEL: Record<AgentWorker['kind'], string> = {
  employee: 'Pegawai vibeasik',
  coding_agent: 'Agent Coding',
  local_agent: 'CLI Lokal',
  llm_call: 'Request LLM',
  planner: 'Tim Perencana'
}

/** Teks gelembung ucapan karakter. */
export function bubbleText(w: AgentWorker, mood: ActorMood): string {
  if (mood === 'error') return w.last_error || 'Ada yang gagal…'
  if (mood === 'done') {
    if (w.kind === 'llm_call') return 'Jawaban terkirim ✓'
    if (w.kind === 'planner') return w.client_name === 'pm' ? 'Keputusan rapat siap ✓' : 'Pendapat disampaikan ✓'
    if (isSubagent(w)) return 'Lapor: tugas selesai ✓'
    return 'Selesai ✓'
  }
  if (mood === 'left') return 'Sampai jumpa 👋'
  if (mood === 'idle' || mood === 'sleeping') {
    // Status masih "working" tapi lama tanpa event: tampilkan apa adanya, bukan aksi terakhirnya.
    if (w.status === 'working') {
      return `Tidak ada aktivitas sejak ${new Date(w.last_seen_at).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })}`
    }
    return w.action || 'Menunggu instruksi'
  }
  return w.action || w.task || 'Sedang bekerja…'
}

/** Baris kecil di bawah aksi: task yang sedang dikerjakan (hanya jika berbeda dari aksi). */
export function bubbleSubtext(w: AgentWorker, mood: ActorMood): string {
  if (mood !== 'working' || !w.task || w.task === w.action) return ''
  return w.task
}

const ACTION_ICONS: Array<[RegExp, string]> = [
  [/menerima tugas|melapor|todo_dispatch_plan/i, '📋'],
  [/^mengedit|^menulis|edit/i, '✏️'],
  [/^membaca|read/i, '📖'],
  [/^mencari|search|grep/i, '🔍'],
  [/^menjalankan|running|test|build/i, '⚡'],
  [/^browsing|web/i, '🌐'],
  [/subagent|mendelegasikan/i, '🤝'],
  [/daftar tugas|todo/i, '📋'],
  [/^berpikir|thinking/i, '💭'],
  [/menunggu jawaban/i, '📨'],
  [/^rapat dewan|^musyawarah|^dewan eksekutif/i, '🏛️'],
  [/^diundang|^mengundang|undangan/i, '📢'],
  [/^mengkaji visi|^memantau dampak|visi|roadmap/i, '👑'],
  [/^menjaga integritas|^mengawasi kualitas|arsitektur/i, '💻'],
  [/^menyusun prioritas|^memvalidasi kriteria/i, '📊'],
  [/rapat|memutuskan|pendapat|usulan/i, '🗂️'],
  [/menganalisa|konteks/i, '🧐'],
  [/membagi tugas|mencatat keputusan/i, '📌'],
  [/^mcp:/i, '🔌'],
  [/terhubung/i, '🔗']
]

/** Ikon kecil yang mewakili jenis aksi terakhir agent. */
export function actionIcon(action: string | undefined): string {
  if (!action) return ''
  return ACTION_ICONS.find(([re]) => re.test(action))?.[1] ?? '🛠️'
}

export function relativeTime(iso: string | undefined, nowMs: number): string {
  if (!iso) return '-'
  const sec = Math.max(0, Math.round((nowMs - Date.parse(iso)) / 1000))
  if (sec < 5) return 'baru saja'
  if (sec < 60) return `${sec} dtk lalu`
  if (sec < 3600) return `${Math.floor(sec / 60)} mnt lalu`
  return `${Math.floor(sec / 3600)} jam lalu`
}
