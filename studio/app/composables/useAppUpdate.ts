// Pembaruan aplikasi (satu sumber untuk badge versi di sidebar dan banner): periksa versi, unduh + pasang,
// lalu mulai ulang engine otomatis dan muat ulang Studio. Logika unduh, verifikasi sha256, dan penggantian biner
// ada di engine (pkg/update); di sini hanya memanggil API-nya.
export interface UpdateInfo {
  current: string
  latest?: string
  available: boolean
  can_apply: boolean
  reason?: string
  url?: string
  staged?: string
  error?: string
}
export interface UpdateView { info: UpdateInfo, auto: boolean, restart_supported: boolean }

export type UpdateStage = 'idle' | 'downloading' | 'restarting'

export function useAppUpdate() {
  const { apiBase } = useApi()
  const toast = useToast()
  const view = useState<UpdateView | null>('app-update-view', () => null)
  const stage = useState<UpdateStage>('app-update-stage', () => 'idle')

  const info = computed(() => view.value?.info)
  // Terpasang di disk tetapi belum dipakai (menunggu mulai ulang).
  const ready = computed(() => Boolean(info.value?.latest && info.value.staged === info.value.latest))
  const available = computed(() => Boolean(info.value?.available && info.value.latest))

  async function load(force = false) {
    try {
      view.value = force
        ? await $fetch<UpdateView>(`${apiBase}/api/update/check`, { method: 'POST', timeout: 15000 })
        : await $fetch<UpdateView>(`${apiBase}/api/update`, { timeout: 8000 })
    } catch {
      if (!view.value) view.value = null
    }
  }

  async function waitHealthy(): Promise<boolean> {
    await new Promise(r => setTimeout(r, 1500))
    for (let i = 0; i < 40; i++) {
      try {
        const h = await $fetch<{ status: string }>(`${apiBase}/health`, { timeout: 1500 })
        if (h.status === 'ok') return true
      } catch { /* belum hidup */ }
      await new Promise(r => setTimeout(r, 1000))
    }
    return false
  }

  async function restart() {
    stage.value = 'restarting'
    try {
      await $fetch(`${apiBase}/api/update/restart`, { method: 'POST' })
    } catch (err: any) {
      stage.value = 'idle'
      toast.add({ title: 'Tidak bisa mulai ulang otomatis', description: err?.data?.error || 'Hentikan lalu jalankan `vibeasik serve` lagi.', color: 'warning' })
      return
    }
    if (await waitHealthy()) return window.location.reload()
    stage.value = 'idle'
    toast.add({ title: 'Engine belum kembali', description: 'Jalankan `vibeasik serve` secara manual.', color: 'warning' })
  }

  // Klik sekali: unduh + verifikasi + pasang, lalu mulai ulang otomatis bila engine mendukung.
  async function updateNow() {
    if (stage.value !== 'idle') return
    try {
      if (!ready.value) {
        stage.value = 'downloading'
        view.value = await $fetch<UpdateView>(`${apiBase}/api/update/apply`, { method: 'POST', timeout: 180000 })
      }
    } catch (err: any) {
      stage.value = 'idle'
      toast.add({ title: 'Pembaruan gagal', description: err?.data?.error || String(err?.message || err), color: 'error' })
      return
    }
    if (view.value?.restart_supported) return restart()
    stage.value = 'idle'
    toast.add({ title: `vibeasik ${info.value?.latest} terpasang`, description: 'Hentikan lalu jalankan `vibeasik serve` lagi untuk memakainya.', color: 'success' })
  }

  async function setAuto(auto: boolean) {
    try {
      await $fetch(`${apiBase}/api/update/auto`, { method: 'PUT', body: { auto } })
      if (view.value) view.value = { ...view.value, auto }
    } catch { /* abaikan */ }
  }

  return { view, info, stage, ready, available, load, restart, updateNow, setAuto }
}
