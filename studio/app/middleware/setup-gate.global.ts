// Penyiapan awal wajib: akun (opsional), Syarat & Ketentuan, dan database. Selama belum selesai,
// semua halaman Studio mengarah ke /setup.
const OPEN = ['/setup']

export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server || OPEN.includes(to.path)) return
  const { apiBase } = useApi()
  const state = useState<{ complete: boolean } | null>('vibeasik-setup', () => null)
  if (state.value?.complete) return
  try {
    state.value = await $fetch<{ complete: boolean }>(`${apiBase}/api/setup`, { timeout: 3000 })
  } catch {
    return // engine tidak terjangkau: biarkan halaman menampilkan status engine sendiri
  }
  if (!state.value?.complete) return navigateTo('/setup', { replace: true })
})
