import type { EnqueueJobType, QueueJob, QueueStats } from '~/types'

/**
 * Background LLM jobs. Every slow LLM action is queued on the engine (POST /api/queue/enqueue)
 * so the UI never waits on the request: the target keeps working, shows queue/progress state,
 * is blocked from being started twice, and reports back when the job ends.
 *
 * One shared poller (browser only) feeds `jobs`/`stats` for every consumer, including QueueFooterBar.
 */

export interface JobTarget {
  /** Required for project-level jobs (summary, profile, synergy). */
  projectId?: number
  /** Required for component-level jobs (fast scan, routes). */
  componentId?: number
  /** Jobs aimed outside projects, e.g. a Galeri Desain reference (design_analyze). */
  targetId?: number
}

interface FinishedEntry {
  seq: number
  job: QueueJob
}

const POLL_BUSY_MS = 2000
const POLL_IDLE_MS = 7000
const FINISHED_LOG_SIZE = 40

const isActive = (j: QueueJob) => j.status === 'queued' || j.status === 'running'

const matchesTarget = (j: QueueJob, t: JobTarget) => {
  if (t.targetId) return j.target_id === t.targetId
  if (t.componentId) return j.component_id === t.componentId
  return j.project_id === t.projectId && !j.component_id
}

// Browser-only singletons: one poll loop, and the job ids this tab is following.
let pollTimer: ReturnType<typeof setTimeout> | null = null
let polling = false
const seenActive = new Set<string>() // seen queued/running -> emit a "finished" entry when it ends
const startedHere = new Set<string>() // started from this tab -> toast on failure

export const useBackgroundJobs = () => {
  const { getQueueJobs, enqueueQueueJob } = useApi()
  const toast = useToast()

  const emptyStats = (): QueueStats => ({
    running_count: 0,
    queued_count: 0,
    completed_count: 0,
    failed_count: 0,
    max_concurrent_llm: 1,
    max_concurrent_total: 3
  })

  const jobs = useState<QueueJob[]>('bgjobs_list', () => [])
  const stats = useState<QueueStats>('bgjobs_stats', emptyStats)
  const finishedLog = useState<FinishedEntry[]>('bgjobs_finished', () => [])
  const finishedSeq = useState<number>('bgjobs_finished_seq', () => 0)

  const activeJobs = computed(() => jobs.value.filter(isActive))
  const hasActive = computed(() => activeJobs.value.length > 0)

  const applySnapshot = (list: QueueJob[], nextStats: QueueStats) => {
    const byId = new Map(list.map(j => [j.id, j]))
    const ended: QueueJob[] = []

    for (const id of [...seenActive]) {
      const j = byId.get(id)
      if (!j) {
        // Cleared from history while we were following it: nothing left to report.
        seenActive.delete(id)
        startedHere.delete(id)
        continue
      }
      if (isActive(j)) continue
      seenActive.delete(id)
      ended.push(j)
    }
    for (const j of list) {
      if (isActive(j)) seenActive.add(j.id)
    }

    jobs.value = list
    stats.value = nextStats

    if (!ended.length) return
    const entries = ended.map(job => ({ seq: ++finishedSeq.value, job }))
    finishedLog.value = [...finishedLog.value, ...entries].slice(-FINISHED_LOG_SIZE)

    for (const job of ended) {
      if (startedHere.delete(job.id) && job.status === 'failed') {
        toast.add({
          title: `${job.title} gagal`,
          description: job.error || job.message || 'Terjadi kesalahan saat memproses di antrean.',
          icon: 'i-lucide-alert-circle',
          color: 'error'
        })
      }
    }
  }

  // Antrean job adalah fitur Pro (orchestrator): tanpa fitur itu tidak perlu polling ke engine.
  const { isLocked } = useEdition()

  const refresh = async () => {
    if (isLocked('orchestrator')) return
    try {
      const res = await getQueueJobs('', 60)
      applySnapshot(res.jobs || [], res.stats)
    } catch {
      // Engine is restarting or offline; the next tick retries.
    }
  }

  const tick = async () => {
    await refresh()
    if (!polling) return
    pollTimer = setTimeout(tick, hasActive.value ? POLL_BUSY_MS : POLL_IDLE_MS)
  }

  const startPolling = () => {
    if (!import.meta.client || polling) return
    polling = true
    tick()
  }

  const stopPolling = () => {
    polling = false
    if (pollTimer) clearTimeout(pollTimer)
    pollTimer = null
  }

  /** Active (queued/running) job of this type for the target, if any. */
  const findActive = (type: EnqueueJobType, target: JobTarget) =>
    jobs.value.find(j => isActive(j) && j.type === type && matchesTarget(j, target)) ?? null

  /** Berapa job aktif di depan yang lane-nya bentrok dengan job ini (lane lain berjalan paralel). */
  const jobsAhead = (job: QueueJob) =>
    activeJobs.value.filter(j => j.id !== job.id && j.created_at < job.created_at && lanesConflict(j.lane, job.lane)).length

  const enqueue = async (type: EnqueueJobType, target: JobTarget): Promise<QueueJob | null> => {
    try {
      const res = await enqueueQueueJob({
        type,
        project_id: target.projectId,
        component_id: target.componentId,
        target_id: target.targetId
      })
      const job = res.job
      seenActive.add(job.id)
      startedHere.add(job.id)
      jobs.value = [job, ...jobs.value.filter(j => j.id !== job.id)]

      const ahead = jobsAhead(job)
      toast.add({
        title: res.already_queued ? 'Sudah dalam antrean' : 'Masuk antrean',
        description: ahead > 0
          ? `${job.title} — ${ahead} proses di depan. Anda bisa lanjut bekerja, hasilnya diberi tahu di sini.`
          : `${job.title} — mulai diproses di latar belakang.`,
        icon: 'i-lucide-list-todo',
        color: 'info'
      })

      startPolling()
      refresh()
      return job
    } catch (err: any) {
      toast.add({
        title: 'Gagal Memasukkan ke Antrean',
        description: err.data?.error || err.message || 'Terjadi kesalahan',
        icon: 'i-lucide-alert-circle',
        color: 'error'
      })
      return null
    }
  }

  return {
    jobs,
    stats,
    finishedLog,
    activeJobs,
    hasActive,
    findActive,
    jobsAhead,
    enqueue,
    refresh,
    startPolling,
    stopPolling
  }
}

/** Runs `handler` for every job that completes successfully while the caller is mounted. */
export const useJobCompleted = (handler: (job: QueueJob) => void | Promise<void>) => {
  const { finishedLog, startPolling } = useBackgroundJobs()
  startPolling()

  let cursor = finishedLog.value.at(-1)?.seq ?? 0
  watch(finishedLog, (log) => {
    for (const entry of log) {
      if (entry.seq <= cursor) continue
      cursor = entry.seq
      if (entry.job.status === 'completed') handler(entry.job)
    }
  })
}

/**
 * Binds one queued action to one target (a project or a component): reactive busy/progress state
 * for the UI, `start()` to queue it, and `onCompleted` to refresh whatever the job updated.
 */
export const useJobTarget = (
  type: EnqueueJobType,
  target: MaybeRefOrGetter<JobTarget | null | undefined>,
  onCompleted?: (job: QueueJob) => void | Promise<void>
) => {
  const bg = useBackgroundJobs()

  const job = computed(() => {
    const t = toValue(target)
    return t ? bg.findActive(type, t) : null
  })
  const busy = computed(() => job.value !== null)
  const waiting = computed(() => job.value?.status === 'queued')

  /** Short text for buttons/badges: queue position while waiting, progress while running. */
  const label = computed(() => {
    const j = job.value
    if (!j) return ''
    if (j.status === 'queued') {
      const ahead = bg.jobsAhead(j)
      return ahead > 0 ? `Antre · ${ahead} di depan` : 'Menunggu giliran'
    }
    return j.progress > 0 ? `Memproses ${j.progress}%` : 'Memproses'
  })

  const start = async () => {
    const t = toValue(target)
    if (!t || busy.value) return null
    return await bg.enqueue(type, t)
  }

  useJobCompleted((done) => {
    const t = toValue(target)
    if (t && done.type === type && matchesTarget(done, t)) onCompleted?.(done)
  })

  return { job, busy, waiting, label, start }
}

/** Sama dengan engine (queue/lanes.go): bentrok bila lane sama atau salah satunya induk yang lain. */
export const lanesConflict = (a?: string, b?: string) =>
  !a || !b || a === b || a.startsWith(`${b}/`) || b.startsWith(`${a}/`)
