import type { AgentRun } from '~/types'

// Run agent per todo (orkestrator). Di build Community API-nya menjawab 402, sehingga daftar kosong.

export function useTodoAgentRuns(onFinished: () => void) {
  const { getAgentRuns } = useApi()
  const byTodo = ref<Record<number, AgentRun>>({})
  let timer: ReturnType<typeof setTimeout> | null = null

  const { isLocked } = useEdition()

  const load = async () => {
    if (isLocked('orchestrator')) return
    try {
      const { runs } = await getAgentRuns({ status: 'queued,running', limit: 100 })
      const next: Record<number, AgentRun> = {}
      for (const r of runs ?? []) {
        if (r.todo_id) next[r.todo_id] = r
      }
      const finished = Object.keys(byTodo.value).some(id => !next[Number(id)])
      byTodo.value = next
      if (finished) onFinished()
    } catch {
      // engine lama tanpa orkestrator: badge tidak ditampilkan
    }
  }

  const schedule = () => {
    const delay = Object.keys(byTodo.value).length ? 4000 : 15000
    timer = setTimeout(async () => {
      await load()
      schedule()
    }, delay)
  }

  onMounted(() => {
    load().then(schedule)
  })
  onBeforeUnmount(() => {
    if (timer) clearTimeout(timer)
  })

  return { byTodo, refresh: load }
}
