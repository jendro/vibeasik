import type { RuntimeSimOptions, RuntimeSimSession } from '~/types'

const POLL_INTERVAL_MS = 1000

/**
 * State & polling Simulasi Runtime untuk satu proyek: engine menyalakan aplikasi sungguhan,
 * frontend mem-poll snapshot tiap detik agar timeline, log, dan hasil request tampil live.
 */
export const useRuntimeSimulation = (onFinished?: (session: RuntimeSimSession) => void) => {
  const toast = useToast()
  const { startRuntimeSimulation, getRuntimeSimulation, cancelRuntimeSimulation, getRuntimeSimDownloadUrl, getRuntimeSimShotUrl } = useApi()

  const session = ref<RuntimeSimSession | null>(null)
  const starting = ref(false)
  const cancelling = ref(false)
  const projectId = ref<number | null>(null)
  let timer: ReturnType<typeof setTimeout> | null = null

  const isRunning = computed(() => session.value?.status === 'running')

  const stopPolling = () => {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
  }

  const poll = async () => {
    const pid = projectId.value
    if (!pid) return
    try {
      const res = await getRuntimeSimulation(pid)
      if (pid !== projectId.value) return // proyek sudah diganti saat request berjalan
      const wasRunning = isRunning.value
      session.value = res.session
      if (res.session?.status === 'running') {
        timer = setTimeout(poll, POLL_INTERVAL_MS)
      } else if (wasRunning && res.session) {
        onFinished?.(res.session)
      }
    } catch (err) {
      console.warn('Gagal memuat status simulasi runtime:', err)
      // Hanya coba lagi bila simulasi sedang berjalan (gangguan sesaat); engine lama tanpa endpoint ini tidak di-poll terus.
      if (isRunning.value) timer = setTimeout(poll, POLL_INTERVAL_MS * 3)
    }
  }

  const load = async (pid: number) => {
    stopPolling()
    projectId.value = pid
    session.value = null
    await poll()
  }

  const start = async (options: RuntimeSimOptions) => {
    const pid = projectId.value
    if (!pid || starting.value) return
    starting.value = true
    try {
      const res = await startRuntimeSimulation(pid, options)
      session.value = res.session
      stopPolling()
      timer = setTimeout(poll, POLL_INTERVAL_MS)
    } catch (err: any) {
      const busy = err?.data?.code === 'runtime_sim_busy'
      if (busy && err?.data?.session) {
        session.value = err.data.session
        stopPolling()
        timer = setTimeout(poll, POLL_INTERVAL_MS)
      }
      toast.add({
        title: busy ? 'Simulasi Sedang Berjalan' : 'Simulasi Runtime Gagal Dimulai',
        description: err?.data?.error || err?.message || 'Local engine tidak merespons.',
        icon: busy ? 'i-lucide-loader-2' : 'i-lucide-alert-circle',
        color: busy ? 'info' : 'error'
      })
    } finally {
      starting.value = false
    }
  }

  const cancel = async () => {
    const pid = projectId.value
    if (!pid || cancelling.value) return
    cancelling.value = true
    try {
      await cancelRuntimeSimulation(pid)
      toast.add({
        title: 'Simulasi Dibatalkan',
        description: 'Pengujian dihentikan; aplikasi yang dinyalakan simulasi sedang dimatikan.',
        icon: 'i-lucide-square',
        color: 'warning'
      })
    } catch (err: any) {
      toast.add({
        title: 'Gagal Membatalkan',
        description: err?.data?.error || err?.message,
        icon: 'i-lucide-alert-circle',
        color: 'error'
      })
    } finally {
      cancelling.value = false
    }
  }

  const download = (format: 'markdown' | 'json' | 'zip' = 'markdown') => {
    const pid = projectId.value
    if (!pid) return
    const ext = { markdown: 'md', json: 'json', zip: 'zip' }[format]
    const link = document.createElement('a')
    link.href = getRuntimeSimDownloadUrl(pid, format)
    link.setAttribute('download', `uji-runtime-${pid}.${ext}`)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  // URL berkas screenshot sesi saat ini (data URL dari laporan lama dipakai apa adanya).
  const shotUrl = (file?: string) => {
    if (!file) return ''
    if (file.startsWith('data:')) return file
    return session.value && projectId.value ? getRuntimeSimShotUrl(projectId.value, session.value.id, file) : ''
  }

  onUnmounted(stopPolling)

  return { session, isRunning, starting, cancelling, load, start, cancel, download, shotUrl }
}
