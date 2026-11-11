// Alamat engine yang sebenarnya (host:port) untuk ditampilkan: dari apiBase bila absolut,
// selain itu origin halaman (Studio produksi disajikan engine sendiri). Diisi saat mount agar SSR/hidrasi sama.
export const useEngineHost = () => {
  const { apiBase } = useApi()
  const host = ref('')
  onMounted(() => {
    try {
      host.value = apiBase && !apiBase.startsWith('/') ? new URL(apiBase).host : window.location.host
    } catch {
      host.value = window.location.host
    }
  })
  return host
}
