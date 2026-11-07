// Edisi engine (Community / Official) dan fitur berbayar yang aktif untuk akun ini.

export interface EditionInfo {
  edition: string
  features: Record<string, boolean>
  installed: Record<string, boolean>
  upgrade_url: string
}

export const useEdition = () => {
  const { apiBase } = useApi()
  const info = useState<EditionInfo | null>('vibeasik-edition', () => null)
  const loading = useState<boolean>('vibeasik-edition-loading', () => false)

  const load = async (force = false) => {
    if ((info.value && !force) || loading.value) return info.value
    loading.value = true
    try {
      info.value = await $fetch<EditionInfo>(`${apiBase}/api/edition`, { timeout: 3000 })
    } catch {
      // Engine lama tanpa /api/edition: anggap semua fitur terpasang & aktif agar tidak mengunci tanpa alasan.
      info.value = null
    } finally {
      loading.value = false
    }
    return info.value
  }

  // Fitur dianggap terbuka bila engine tidak melapor (kompatibel) atau melaporkannya aktif.
  const isEnabled = (feature?: string) => !feature || !info.value || info.value.features[feature] !== false
  const isInstalled = (feature?: string) => !feature || !info.value || info.value.installed[feature] !== false
  const isLocked = (feature?: string) => !isEnabled(feature) || !isInstalled(feature)
  const upgradeUrl = computed(() => info.value?.upgrade_url || 'https://vibeasik.com/pricing')
  const edition = computed(() => info.value?.edition || '')

  return { info, load, isEnabled, isInstalled, isLocked, upgradeUrl, edition }
}
