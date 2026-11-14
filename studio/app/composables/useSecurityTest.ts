import type { SecurityTestReport, SecurityTestRequest, SecurityTestSession, Todo } from '~/types'

const POLL_MS = 1000

/** Uji keamanan proyek: audit router + server yang dinyalakan & dipantau di engine (lokal/live). */
export const useSecurityTest = () => {
  const { apiBase } = useApi()
  const url = (projectId: number) => `${apiBase}/api/projects/${projectId}/security-test`

  return {
    /** Hasil uji terakhir dan sesi uji (berjalan atau selesai) sejak engine menyala. */
    getState: (projectId: number) => $fetch<{ report: SecurityTestReport | null, session: SecurityTestSession | null }>(url(projectId)),
    start: (projectId: number, body: SecurityTestRequest) =>
      $fetch<{ session: SecurityTestSession }>(`${url(projectId)}/start`, { method: 'POST', body }),
    cancel: (projectId: number) => $fetch<{ ok: boolean }>(`${url(projectId)}/cancel`, { method: 'POST' }),
    /** Pemeriksaan gagal/lemah di hasil uji terakhir → satu todo keamanan per pemeriksaan (tanpa LLM). */
    createTodos: (projectId: number) =>
      $fetch<{ created: number, existed: number, todos: Todo[] }>(`${url(projectId)}/todos`, { method: 'POST' })
  }
}

/**
 * State & polling satu proyek: sesi berjalan di engine, halaman mem-poll tiap detik agar langkah, aktivitas,
 * dan log server tampil live. Halaman boleh ditinggal; saat dibuka lagi sesi yang berjalan dilanjutkan.
 */
export const useSecurityTestSession = (onFinished?: (s: SecurityTestSession) => void) => {
  const api = useSecurityTest()
  const report = ref<SecurityTestReport | null>(null)
  const session = ref<SecurityTestSession | null>(null)
  const loading = ref(false)
  const error = ref('')
  const projectId = ref<number | null>(null)
  let timer: ReturnType<typeof setTimeout> | null = null

  const isRunning = computed(() => session.value?.status === 'running')
  const stop = () => {
    if (timer) clearTimeout(timer)
    timer = null
  }

  const poll = async () => {
    const pid = projectId.value
    if (!pid) return
    try {
      const res = await api.getState(pid)
      if (pid !== projectId.value) return // proyek diganti saat request berjalan
      const wasRunning = isRunning.value
      session.value = res.session
      report.value = res.report
      if (res.session?.status === 'running') timer = setTimeout(poll, POLL_MS)
      else if (wasRunning && res.session) onFinished?.(res.session)
    } catch (err: any) {
      if (isRunning.value) timer = setTimeout(poll, POLL_MS * 3) // gangguan sesaat saat uji berjalan
      else error.value = err?.data?.error || err?.message || 'Gagal memuat hasil uji'
    }
  }

  const load = async (pid: number) => {
    stop()
    projectId.value = pid
    session.value = null
    report.value = null
    error.value = ''
    loading.value = true
    await poll()
    loading.value = false
  }

  const starting = ref(false)
  const start = async (body: SecurityTestRequest) => {
    const pid = projectId.value
    if (!pid || starting.value) return
    starting.value = true
    error.value = ''
    try {
      session.value = (await api.start(pid, body)).session
    } catch (err: any) {
      if (err?.data?.code === 'security_test_busy' && err?.data?.session) session.value = err.data.session
      else error.value = err?.data?.error || err?.message || 'Uji keamanan gagal dimulai'
    } finally {
      starting.value = false
    }
    if (isRunning.value) {
      stop()
      timer = setTimeout(poll, POLL_MS)
    }
  }

  const cancelling = ref(false)
  const cancel = async () => {
    const pid = projectId.value
    if (!pid) return
    cancelling.value = true
    try {
      await api.cancel(pid)
    } catch (err: any) {
      error.value = err?.data?.error || err?.message || 'Gagal membatalkan uji'
    } finally {
      cancelling.value = false
    }
  }

  onUnmounted(stop)
  return { report, session, loading, error, isRunning, starting, cancelling, load, start, cancel, createTodos: api.createTodos }
}
