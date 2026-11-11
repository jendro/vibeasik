// Prompt siap tempel untuk menyuruh agent coding mengerjakan todo vibeasik lewat MCP. Prompt pendek seperti
// "kerjakan todo di vibeasik" kadang dibaca agent sebagai "periksa repo", jadi prompt ini menyebut tool pertama,
// project_id, kunci koordinator, dan larangan scan secara eksplisit.

export interface AgentPromptTarget {
  key: string
  label: string
  icon: string
  /** true: agent punya sub agent sendiri di aplikasinya (Claude Code, Antigravity) → mode divisi; selain itu mode worker. */
  subagents: boolean
  /** Cara menjalankan sub agent di aplikasi itu (untuk mode divisi). */
  subagentTool?: string
  hint: string
}

export const agentPromptTargets: AgentPromptTarget[] = [
  { key: 'claude', label: 'Claude Code', icon: 'i-simple-icons-claude', subagents: true, subagentTool: 'tool Agent/Task, foreground', hint: 'Punya sub agent sendiri: tiap todo dikerjakan satu sub agent baru (konteks bersih) yang berpasangan dengan pegawai vibeasik sebagai navigator; koordinator hanya menerima ringkasannya.' },
  { key: 'agy', label: 'Antigravity', icon: 'i-simple-icons-google', subagents: true, subagentTool: 'agent/sub agent baru di Antigravity', hint: 'Punya sub agent sendiri: tiap todo dikerjakan satu sub agent baru di Antigravity (konteks bersih) sebagai pegawai vibeasik, koordinator hanya menerima ringkasannya.' },
  { key: 'cline', label: 'Cline', icon: 'i-lucide-terminal', subagents: false, hint: 'Mode worker: tiap todo dikirim ke proses agent baru lewat orkestrator vibeasik (konteks bersih per todo).' },
  { key: 'cursor', label: 'Cursor', icon: 'i-simple-icons-cursor', subagents: false, hint: 'Mode worker: tiap todo dikirim ke proses agent baru lewat orkestrator vibeasik (konteks bersih per todo).' },
  { key: 'codex', label: 'Codex', icon: 'i-lucide-code-xml', subagents: false, hint: 'Mode worker: tiap todo dikirim ke proses agent baru lewat orkestrator vibeasik (konteks bersih per todo).' },
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

const closing = 'Pair programming: sub agent/worker adalah DRIVER, pegawai vibeasik yang ditugaskan adalah NAVIGATOR & mentor-nya (menjawab lewat `vibeasik_staff_consult` dengan model & pengetahuannya sendiri). '
  + 'Hemat konteks: kamu hanya koordinator. Jangan membaca kode, diff lengkap, berkas prompt sub agent, atau log panjang sendiri; cukup ringkasan dari sub agent/run dan hasil tool vibeasik. '
  + 'Tiap completion_note memuat KONSULTASI:, PEGAWAI:, PENGETAHUAN:, dan PELAJARAN:. Jangan membuat todo baru sendiri; temuan lanjutan tulis di completion_note. '
  + 'Di akhir, laporkan per todo: pegawai, konsultasi (plan/stuck/review + verdict), hasil, nilai QA, hash commit.'

const divisionSteps = (p: AgentPromptProject, t: AgentPromptTarget) =>
  `1. Panggil \`vibeasik_todo_dispatch_plan\` {project_id: ${p.id}, coordinator: "${t.key}", workers: false} sebagai langkah PERTAMA, lalu ikuti \`coordinator_steps\`-nya. Kamu koordinator: jangan mengedit kode sendiri.\n`
  + `2. Untuk tiap todo di \`next_wave\`: \`update_todo\` {todo_id, status: "in_progress"}, lalu jalankan SATU sub agent baru (${t.subagentTool || 'sub agent di aplikasimu'}) dengan \`subagent_launch\` apa adanya sebagai prompt-nya — sub agent membaca \`subagent_prompt_file\` utuh (persona, standar, pengetahuan pegawai, aturan konsultasi); jangan diringkas dan jangan kamu baca sendiri. Tanpa berkas: \`subagent_prompt\` apa adanya. Pakai model \`subagent_model\` bila ada. Jangan memakai \`vibeasik_dispatch_agent\`. Satu sub agent = satu todo: tutup setelah melapor. Paralel hanya bila \`next_wave.parallel\` = true.\n`
  + '3. Pair programming: sub agent wajib berkonsultasi dengan pegawai di kolom `employee` lewat `vibeasik_staff_consult` — `plan` sebelum edit pertama, `stuck` saat buntu, `review` sebelum melapor. Setelah sub agent melapor, cek `vibeasik_staff_consults` {todo_id}: review belum ada → panggil sendiri `vibeasik_staff_consult` {todo_id, phase: "review", repo, branch, base}; verdict `revise` → sub agent baru di worktree yang sama dengan arahan itu. Sub agent buntu/gagal → tanyakan dulu ke pegawai (phase "stuck") sebelum mengembalikan todo.\n'
  + '4. Commit sisa perubahan di worktree todo, build/test, lalu `vibeasik_qa_gate`. next=rework → sub agent baru dengan `feedback_section`; next=merge → squash satu todo = satu commit; next=escalated → simpan branch, jangan squash.\n'
  + '5. `update_todo` status "completed" (atau kembali "pending" beserta alasan bila gagal).\n'
  + '6. Panggil `vibeasik_todo_dispatch_plan` lagi sampai `next_wave` kosong.\n\n'

const workerSteps = (p: AgentPromptProject, key: string) =>
  `1. Panggil \`vibeasik_todo_dispatch_plan\` {project_id: ${p.id}, coordinator: "${key}"} sebagai langkah PERTAMA. Kamu koordinator mode worker: jangan mengedit kode sendiri.\n`
  + '2. Bila hasilnya punya `note` "delegate/workers diabaikan" atau "PERINGATAN", atau tidak ada satu pun todo di `next_wave` yang punya `delegate`, berhenti dan laporkan itu ke saya — jangan mengerjakan todo sendiri tanpa persetujuan saya.\n'
  + '3. Buat daftar tugas (task list/checklist) berisi tiap todo `next_wave`: `#todo_id judul — employee → delegate.profile_name`, lalu perbarui statusnya (dikirim, berjalan, selesai + nilai QA) sepanjang percakapan.\n'
  + '4. Untuk tiap todo di `next_wave` yang punya `delegate`: kirim lewat `vibeasik_dispatch_agent` {todo_id, profile_id: delegate.profile_id} — proses agent baru itulah sub agent-nya (driver), berpasangan dengan pegawai di kolom `employee` sebagai navigator lewat `vibeasik_staff_consult`. Jangan memanggil `update_todo` untuk todo itu — orkestrator mengurus worktree, QA gate, merge, dan statusnya.\n'
  + '5. Tunggu dengan `vibeasik_agent_runs` {todo_ids: [...], wait_sec: 300}, ulangi sampai `active` = 0, lalu periksa hasil tiap run.\n'
  + '6. Todo tanpa `delegate` (tidak ada worker): jangan kerjakan di percakapan ini agar konteks tidak menumpuk. Laporkan daftarnya (todo_id + judul) dan minta saya membuka percakapan baru per todo.\n'
  + '7. Panggil `vibeasik_todo_dispatch_plan` lagi sampai `next_wave` kosong.\n\n'

export function buildAgentPrompt(target: AgentPromptTarget, project: AgentPromptProject): string {
  const steps = target.subagents ? divisionSteps(project, target) : workerSteps(project, target.key)
  return intro(project) + 'Langkah:\n' + steps + closing
}
