// Prompt siap tempel untuk menyuruh agent coding mengerjakan todo vibeasik lewat MCP. Prompt pendek seperti
// "kerjakan todo di vibeasik" kadang dibaca agent sebagai "periksa repo", jadi prompt ini menyebut tool pertama,
// project_id, kunci koordinator, dan larangan scan secara eksplisit.

export interface AgentPromptTarget {
  key: string
  label: string
  icon: string
  /** true: agent punya tool sub agent sendiri (Claude Code) → mode divisi; selain itu mode worker. */
  subagents: boolean
  hint: string
}

export const agentPromptTargets: AgentPromptTarget[] = [
  { key: 'claude', label: 'Claude Code', icon: 'i-simple-icons-claude', subagents: true, hint: 'Punya sub agent sendiri: tiap todo dikerjakan satu sub agent (tool Agent/Task).' },
  { key: 'agy', label: 'Antigravity', icon: 'i-simple-icons-google', subagents: false, hint: 'Mode worker: todo dikirim ke proses agent baru lewat orkestrator vibeasik.' },
  { key: 'cline', label: 'Cline', icon: 'i-lucide-terminal', subagents: false, hint: 'Mode worker: todo dikirim ke proses agent baru lewat orkestrator vibeasik.' },
  { key: 'cursor', label: 'Cursor', icon: 'i-simple-icons-cursor', subagents: false, hint: 'Mode worker: todo dikirim ke proses agent baru lewat orkestrator vibeasik.' },
  { key: 'codex', label: 'Codex', icon: 'i-lucide-code-xml', subagents: false, hint: 'Mode worker: todo dikirim ke proses agent baru lewat orkestrator vibeasik.' },
  { key: 'other', label: 'Agent lain', icon: 'i-lucide-bot', subagents: false, hint: 'Agent MCP apa pun (Windsurf, OpenCode, …): mode worker.' }
]

export interface AgentPromptProject {
  id: number
  name?: string
}

const intro = (p: AgentPromptProject) =>
  `Kerjakan todo pending proyek vibeasik "${p.name || `#${p.id}`}" (project_id ${p.id}) lewat MCP server \`vibeasik\`.\n`
  + 'Ini perintah mengerjakan todo, BUKAN audit: jangan scan atau menjelajah repo dulu, dan jangan memanggil scan_directory, fast_scan, run_diagnostics, atau run_deep_check.\n'
  + 'Bila tool MCP vibeasik tidak tersedia, berhenti dan minta saya menyalakan `vibeasik serve`.\n\n'

const closing = 'Untuk setiap todo: ikuti brief pegawai vibeasik (persona & pengetahuannya), tulis PEGAWAI:, PENGETAHUAN:, dan PELAJARAN: di completion_note. Jangan membuat todo baru sendiri; temuan lanjutan tulis di completion_note. Di akhir, laporkan per todo: pegawai, hasil, nilai QA, hash commit.'

const divisionSteps = (p: AgentPromptProject) =>
  `1. Panggil \`vibeasik_todo_dispatch_plan\` {project_id: ${p.id}, coordinator: "claude"} sebagai langkah PERTAMA, lalu ikuti \`coordinator_steps\`-nya. Kamu koordinator: jangan mengedit kode sendiri.\n`
  + '2. Untuk tiap todo di `next_wave`: `update_todo` {todo_id, status: "in_progress"}, lalu jalankan SATU sub agent baru (tool Agent/Task, foreground) dengan `subagent_prompt` apa adanya dan model `subagent_model`. Paralel hanya bila `next_wave.parallel` = true.\n'
  + '3. Setelah sub agent melapor: commit sisa perubahan di worktree todo, build/test, lalu `vibeasik_qa_gate`. next=rework → sub agent baru dengan `feedback_section`; next=merge → squash satu todo = satu commit; next=escalated → simpan branch, jangan squash.\n'
  + '4. `update_todo` status "completed" (atau kembali "pending" beserta alasan bila gagal).\n'
  + '5. Panggil `vibeasik_todo_dispatch_plan` lagi sampai `next_wave` kosong.\n\n'

const workerSteps = (p: AgentPromptProject, key: string) =>
  `1. Panggil \`vibeasik_todo_dispatch_plan\` {project_id: ${p.id}, coordinator: "${key}"} sebagai langkah PERTAMA. Kamu koordinator mode worker: jangan mengedit kode sendiri.\n`
  + '2. Bila hasilnya punya `note` "delegate/workers diabaikan", berhenti dan laporkan note itu ke saya — jangan mengerjakan todo sendiri tanpa persetujuan saya.\n'
  + '3. Untuk tiap todo di `next_wave` yang punya `delegate`: kirim lewat `vibeasik_dispatch_agent` {todo_id, profile_id: delegate.profile_id}. Jangan memanggil `update_todo` untuk todo itu — orkestrator mengurus worktree, QA gate, merge, dan statusnya.\n'
  + '4. Tunggu dengan `vibeasik_agent_runs` {todo_ids: [...], wait_sec: 300}, ulangi sampai `active` = 0, lalu periksa hasil tiap run.\n'
  + '5. Todo tanpa `delegate`: kerjakan sendiri satu per satu SEBAGAI pegawai di kolom `employee` — `update_todo` in_progress dulu (brief pegawai), ikuti `subagent_prompt`, build/test, `vibeasik_qa_gate`, lalu `update_todo` completed.\n'
  + '6. Panggil `vibeasik_todo_dispatch_plan` lagi sampai `next_wave` kosong.\n\n'

export function buildAgentPrompt(target: AgentPromptTarget, project: AgentPromptProject): string {
  const steps = target.subagents ? divisionSteps(project) : workerSteps(project, target.key)
  return intro(project) + 'Langkah:\n' + steps + closing
}
