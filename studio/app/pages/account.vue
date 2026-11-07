<script setup lang="ts">
// Akun vibeasik: login perangkat (kode ditampilkan di sini, disetujui di halaman server), paket & fitur aktif.
const { apiBase } = useApi()
const { view, account, load, logout } = useAuth()
const { load: loadEdition } = useEdition()
const toast = useToast()

interface DeviceCodeResp {
  device_code: string
  user_code: string
  verification_uri: string
  verification_uri_complete: string
  expires_in: number
  interval: number
}

const pending = ref<DeviceCodeResp | null>(null)
const busy = ref(false)
let timer: ReturnType<typeof setTimeout> | null = null

onMounted(load)

const fmt = (s?: string) => s ? new Date(s).toLocaleString('id-ID') : '-'

const stopPolling = () => {
  if (timer) clearTimeout(timer)
  timer = null
}

const poll = async () => {
  if (!pending.value) return
  try {
    const res = await $fetch<{ status: string, error?: string }>(`${apiBase}/api/account/login/poll`, {
      method: 'POST', body: { device_code: pending.value.device_code }
    })
    if (res.status === 'done') {
      pending.value = null
      await load()
      await loadEdition(true)
      toast.add({ title: 'Akun tersambung', description: `Paket ${account.value?.plan}`, color: 'success', icon: 'i-lucide-check' })
      return
    }
  } catch (err: any) {
    pending.value = null
    toast.add({ title: 'Login gagal', description: err?.data?.error || 'Mulai ulang login.', color: 'error' })
    return
  }
  timer = setTimeout(poll, Math.max(pending.value?.interval || 5, 2) * 1000)
}

const startLogin = async () => {
  busy.value = true
  try {
    pending.value = await $fetch<DeviceCodeResp>(`${apiBase}/api/account/login`, { method: 'POST' })
    window.open(pending.value.verification_uri_complete, '_blank', 'noopener')
    timer = setTimeout(poll, pending.value.interval * 1000)
  } catch (err: any) {
    toast.add({ title: 'Server vibeasik tidak bisa dihubungi', description: err?.data?.error || String(err), color: 'error' })
  } finally {
    busy.value = false
  }
}

const refresh = async () => {
  busy.value = true
  try {
    view.value = await $fetch(`${apiBase}/api/account/refresh`, { method: 'POST' })
    await loadEdition(true)
  } catch (err: any) {
    toast.add({ title: 'Gagal memperbarui', description: err?.data?.error || String(err), color: 'warning' })
  } finally {
    busy.value = false
  }
}

const doLogout = async () => {
  await logout()
  toast.add({ title: 'Keluar dari akun', description: 'Fitur berbayar terkunci kembali.', icon: 'i-lucide-log-out' })
}

onBeforeUnmount(stopPolling)
useHead({ title: 'Akun vibeasik' })
</script>

<template>
  <div class="max-w-2xl mx-auto py-10 px-4 space-y-6">
    <div>
      <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
        Akun vibeasik
      </h1>
      <p class="text-sm text-neutral-500 dark:text-neutral-400">
        Edisi engine: <b>{{ view?.edition || '-' }}</b> · server: <code class="font-mono">{{ account?.cloud_url }}</code>
      </p>
    </div>

    <ClientOnly>
      <UCard v-if="account?.logged_in">
        <div class="space-y-3">
          <div class="flex items-center gap-3">
            <UIcon name="i-lucide-user-round-check" class="w-8 h-8 text-emerald-500" />
            <div>
              <div class="font-semibold text-neutral-900 dark:text-white">
                {{ account.email }}
              </div>
              <div class="text-sm text-neutral-500">
                Paket <UBadge :color="account.plan === 'free' ? 'neutral' : 'warning'" variant="subtle">
                  {{ account.plan }}
                </UBadge>
                <span v-if="account.offline" class="ml-2 text-amber-600">offline — memakai masa tenggang</span>
              </div>
            </div>
          </div>
          <dl class="text-xs grid grid-cols-2 gap-2 text-neutral-600 dark:text-neutral-400">
            <dt>Perangkat</dt><dd class="font-mono">
              {{ account.device_id }}
            </dd>
            <dt>Diperbarui</dt><dd>{{ fmt(account.last_refresh) }}</dd>
            <dt>Berlaku sampai</dt><dd>{{ fmt(account.usable_until) }}</dd>
          </dl>
          <p v-if="account.refresh_error" class="text-xs text-amber-600">
            {{ account.refresh_error }}
          </p>
          <div v-if="account.features.length" class="flex flex-wrap gap-1">
            <UBadge v-for="f in account.features" :key="f" size="sm" variant="soft">
              {{ PRO_FEATURES[f]?.label || f }}
            </UBadge>
          </div>
          <div class="flex gap-2 pt-2">
            <UButton :loading="busy" icon="i-lucide-refresh-cw" variant="soft" @click="refresh">
              Perbarui paket
            </UButton>
            <UButton color="error" variant="ghost" icon="i-lucide-log-out" @click="doLogout">
              Keluar
            </UButton>
          </div>
        </div>
      </UCard>

      <UCard v-else>
        <div class="space-y-4">
          <p class="text-sm text-neutral-600 dark:text-neutral-400">
            Belum masuk. Fitur Community tetap jalan tanpa akun. Masuk (gratis) untuk menyumbang Pusat Pengetahuan,
            atau dengan paket Pro untuk membuka fitur berbayar.
          </p>
          <div v-if="pending" class="rounded-lg border border-sky-300 dark:border-sky-700 p-4 space-y-2 text-center">
            <p class="text-sm">
              Masukkan kode ini di halaman yang terbuka:
            </p>
            <div class="text-3xl font-mono font-bold tracking-widest text-sky-600 dark:text-sky-400">
              {{ pending.user_code }}
            </div>
            <a :href="pending.verification_uri_complete" target="_blank" rel="noopener" class="text-xs underline">{{ pending.verification_uri }}</a>
            <p class="text-xs text-neutral-500">
              Menunggu persetujuan…
            </p>
          </div>
          <UButton v-else :loading="busy" icon="i-lucide-log-in" @click="startLogin">
            Masuk dengan akun vibeasik
          </UButton>
          <p class="text-xs text-neutral-500">
            Dari terminal: <code class="font-mono">vibeasik login</code>
          </p>
        </div>
      </UCard>
    </ClientOnly>
  </div>
</template>
