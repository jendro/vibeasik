// Akun vibeasik di Studio. Akses Studio sendiri dijaga token engine (cookie dari `vibeasik open`);
// akun dipakai untuk membuka fitur sesuai paket & menyumbang Pusat Pengetahuan.

export interface AuthUser {
  id: number
  name: string
  email: string
  role: string
  avatar?: string
  toolPreference?: string
}

export interface AccountStatus {
  logged_in: boolean
  email?: string
  plan: string
  features: string[]
  device_id?: string
  expires_at?: string
  usable_until?: string
  offline: boolean
  cloud_url: string
  last_refresh?: string
  refresh_error?: string
}

export interface AccountView {
  account: AccountStatus
  edition: string
  features: Record<string, boolean>
}

export const useAuth = () => {
  const { apiBase } = useApi()
  const view = useState<AccountView | null>('vibeasik-account', () => null)
  // Diambil saat setup: composable Nuxt tidak boleh dipanggil setelah await.
  const edition = useEdition()

  const load = async () => {
    try {
      view.value = await $fetch<AccountView>(`${apiBase}/api/account`)
    } catch {
      view.value = null
    }
    return view.value
  }

  const account = computed(() => view.value?.account)
  const isAuthenticated = computed(() => Boolean(account.value?.logged_in))
  const user = computed<AuthUser | null>(() => account.value?.logged_in
    ? { id: 0, name: account.value.email?.split('@')[0] || 'vibecoder', email: account.value.email || '', role: account.value.plan }
    : null)

  const logout = async () => {
    try {
      view.value = await $fetch<AccountView>(`${apiBase}/api/account/logout`, { method: 'POST' })
    } catch {
      await load()
    }
    // Fitur Pro berubah: muat ulang status edisi.
    await edition.load(true)
  }

  if (import.meta.client && !view.value) load()

  return { view, account, user, isAuthenticated, load, logout }
}
