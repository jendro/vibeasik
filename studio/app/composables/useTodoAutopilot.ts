import type { Ref } from 'vue'
import type { AgentDispatchRequest, Project, Todo } from '~/types'

// Autopilot: kerjakan todo proyek aktif satu per satu lewat agent coding + pantau run yang berjalan.
export function useTodoAutopilot(opts: {
  projectTodos: Ref<Todo[]>
  filteredTodos: Ref<Todo[]>
  activeProject: Ref<Project | null>
  loadData: () => Promise<void>
}) {
  const toast = useToast()
  const { dispatchAgentRun } = useApi()

  // Autopilot: Kerjakan todo proyek aktif satu per satu secara otomatis
  const autopilotEnabled = ref(false)
  const autopilotAgent = ref<'agy' | 'cline' | 'opencode' | 'auto'>('agy') // default: agy (Google Antigravity CLI dengan 7 akun aktif & terbukti mengubah kode)
  const autopilotRunning = ref(false)
  const autopilotCurrentTodoId = ref<number | null>(null)
  const autopilotProcessedCount = ref(0)

  const { byTodo: agentRunsByTodo, refresh: refreshAgentRuns } = useTodoAgentRuns(async () => {
    await opts.loadData()
    if (autopilotEnabled.value) {
      await runNextAutopilot()
    }
  })

  const activeRunningTodo = computed(() => {
    return opts.projectTodos.value.find((t) => {
      const run = agentRunsByTodo.value[t.id]
      return run && (run.status === 'running' || run.status === 'queued')
    })
  })

  const runNextAutopilot = async () => {
    if (!autopilotEnabled.value || autopilotRunning.value) return

    // Tunggu jika sedang ada tugas yang berjalan
    if (activeRunningTodo.value) {
      autopilotCurrentTodoId.value = activeRunningTodo.value.id
      return
    }

    // Muat data todo terbaru agar status yang baru selesai tidak terbaca pending
    await opts.loadData()

    const currentlyRunning = activeRunningTodo.value as Todo | undefined
    if (currentlyRunning) {
      autopilotCurrentTodoId.value = currentlyRunning.id
      return
    }

    // Ambil todo pending berikutnya (prioritas tertinggi dulu)
    const candidate = opts.filteredTodos.value.find(t => t.status === 'pending' && !agentRunsByTodo.value[t.id])
      || opts.projectTodos.value.find(t => t.status === 'pending' && !agentRunsByTodo.value[t.id])

    if (!candidate) {
      autopilotEnabled.value = false
      autopilotCurrentTodoId.value = null
      toast.add({
        title: 'Autopilot Selesai',
        description: `Seluruh agenda pada proyek ${opts.activeProject.value?.name || ''} telah selesai dikerjakan!`,
        icon: 'i-lucide-check-circle-2',
        color: 'success'
      })
      return
    }

    autopilotRunning.value = true
    autopilotCurrentTodoId.value = candidate.id

    try {
      const payload: AgentDispatchRequest = {
        todo_id: candidate.id,
        router: 'rules'
      }

      if (autopilotAgent.value === 'agy') {
        payload.pool_agent_key = 'agy'
      } else if (autopilotAgent.value === 'cline') {
        payload.pool_agent_key = 'cline'
      } else if (autopilotAgent.value === 'opencode') {
        payload.pool_agent_key = 'opencode'
      }

      const res = await dispatchAgentRun(payload)
      autopilotProcessedCount.value++
      toast.add({
        title: `Autopilot: Menjalankan Agenda #${candidate.id}`,
        description: `"${candidate.title.slice(0, 45)}..." dikerjakan oleh ${res?.run?.profile_name || autopilotAgent.value}`,
        icon: 'i-lucide-bot',
        color: 'info'
      })
      await refreshAgentRuns()
    } catch (err: unknown) {
      const e = err as { data?: { error?: string }, message?: string }
      console.error('Gagal menjalankan autopilot todo:', err)
      toast.add({
        title: 'Autopilot Dijeda',
        description: `Gagal mengirim todo #${candidate.id}: ${e?.data?.error || e?.message || 'Error'}`,
        icon: 'i-lucide-alert-circle',
        color: 'error'
      })
      autopilotEnabled.value = false
      autopilotCurrentTodoId.value = null
    } finally {
      autopilotRunning.value = false
    }
  }

  // Watcher ketika tugas selesai dijalankan
  watch(activeRunningTodo, (cur, prev) => {
    if (autopilotEnabled.value && !cur && prev) {
      setTimeout(async () => {
        if (autopilotEnabled.value) {
          await runNextAutopilot()
        }
      }, 1500)
    }
  })

  const handleToggleAutopilot = async (val: boolean) => {
    autopilotEnabled.value = val
    if (val) {
      toast.add({
        title: 'Autopilot Aktif',
        description: `Agenda proyek aktif akan dikerjakan satu per satu via ${autopilotAgent.value === 'opencode' ? 'OpenCode' : autopilotAgent.value === 'agy' ? 'Antigravity' : 'Auto'}.`,
        icon: 'i-lucide-play',
        color: 'success'
      })
      await runNextAutopilot()
    } else {
      autopilotCurrentTodoId.value = null
      toast.add({
        title: 'Autopilot Dimatikan',
        description: 'Pengerjaan otomatis dihentikan. Tugas yang sedang aktif berjalan tetap diselesaikan.',
        icon: 'i-lucide-square',
        color: 'neutral'
      })
    }
  }

  return { autopilotEnabled, autopilotAgent, activeRunningTodo, agentRunsByTodo, refreshAgentRuns, handleToggleAutopilot }
}
