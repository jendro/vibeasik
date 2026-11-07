<script setup lang="ts">
// Banner pembaruan: muncul bila ada versi baru. Pembaruan dipasang di latar bila diizinkan;
// pengguna tinggal menekan "Mulai ulang" untuk memakainya.
interface UpdateInfo {
  current: string
  latest?: string
  available: boolean
  can_apply: boolean
  reason?: string
  url?: string
  staged?: string
}
interface UpdateView { info: UpdateInfo, auto: boolean, restart_supported: boolean }

const { apiBase } = useApi()
const toast = useToast()
const view = ref<UpdateView | null>(null)
const busy = ref(false)
const restarting = ref(false)
const dismissed = ref('')

const KEY = 'vibeasik-update-dismissed'

async function load() {
  try {
    view.value = await $fetch<UpdateView>(`${apiBase}/api/update`, { timeout: 8000 })
  } catch { view.value = null }
}

onMounted(() => {
  try { dismissed.value = sessionStorage.getItem(KEY) || '' } catch { /* tanpa storage */ }
  load()
})

const info = computed(() => view.value?.info)
const ready = computed(() => Boolean(info.value?.latest && info.value.staged === info.value.latest))
const visible = computed(() => Boolean(info.value?.available && info.value.latest && dismissed.value !== info.value.latest))

function dismiss() {
  dismissed.value = info.value?.latest || ''
  try { sessionStorage.setItem(KEY, dismissed.value) } catch { /* abaikan */ }
}

async function install() {
  busy.value = true
  try {
    view.value = await $fetch<UpdateView>(`${apiBase}/api/update/apply`, { method: 'POST', timeout: 180000 })
  } catch (err: any) {
    toast.add({ title: 'Pembaruan gagal', description: err?.data?.error || String(err?.message || err), color: 'error' })
  } finally {
    busy.value = false
  }
}

async function restart() {
  restarting.value = true
  try {
    await $fetch(`${apiBase}/api/update/restart`, { method: 'POST' })
  } catch (err: any) {
    restarting.value = false
    return toast.add({ title: 'Tidak bisa mulai ulang otomatis', description: err?.data?.error || 'Hentikan lalu jalankan `vibeasik serve` lagi.', color: 'warning' })
  }
  await new Promise(r => setTimeout(r, 1500))
  for (let i = 0; i < 40; i++) {
    try {
      const h = await $fetch<{ status: string }>(`${apiBase}/health`, { timeout: 1500 })
      if (h.status === 'ok') return window.location.reload()
    } catch { /* belum hidup */ }
    await new Promise(r => setTimeout(r, 1000))
  }
  restarting.value = false
  toast.add({ title: 'Engine belum kembali', description: 'Jalankan `vibeasik serve` secara manual.', color: 'warning' })
}

async function toggleAuto(v: boolean | 'indeterminate') {
  const auto = v === true
  try {
    await $fetch(`${apiBase}/api/update/auto`, { method: 'PUT', body: { auto } })
    if (view.value) view.value.auto = auto
  } catch { /* abaikan */ }
}
</script>

<template>
  <div
    v-if="visible && info"
    role="status"
    class="flex flex-wrap items-center gap-x-4 gap-y-2 px-4 sm:px-6 py-2 text-sm bg-sky-600 text-white"
  >
    <UIcon name="i-lucide-download-cloud" class="w-4 h-4 shrink-0" />
    <span v-if="ready" class="font-medium">vibeasik {{ info.latest }} sudah terpasang. Mulai ulang untuk memakainya.</span>
    <span v-else class="font-medium">vibeasik {{ info.latest }} tersedia (terpasang {{ info.current }}).</span>

    <div class="flex items-center gap-2 ml-auto">
      <label v-if="info.can_apply && !ready" class="hidden sm:flex items-center gap-1.5 text-xs text-sky-100 cursor-pointer">
        <input type="checkbox" :checked="view?.auto" @change="toggleAuto(($event.target as HTMLInputElement).checked)">
        Pasang otomatis
      </label>
      <UButton v-if="ready && view?.restart_supported" size="xs" color="neutral" variant="solid" icon="i-lucide-rotate-cw" :loading="restarting" @click="restart">
        Mulai ulang sekarang
      </UButton>
      <span v-else-if="ready" class="text-xs text-sky-100">Hentikan lalu jalankan `vibeasik serve` lagi.</span>
      <UButton v-else-if="info.can_apply" size="xs" color="neutral" variant="solid" icon="i-lucide-download" :loading="busy" @click="install">
        Perbarui sekarang
      </UButton>
      <UButton v-else-if="info.url" size="xs" color="neutral" variant="solid" :to="info.url" target="_blank" trailing-icon="i-lucide-external-link">
        Lihat rilis
      </UButton>
      <UButton size="xs" color="neutral" variant="ghost" class="text-white hover:bg-sky-700" aria-label="Tutup" icon="i-lucide-x" @click="dismiss" />
    </div>
  </div>
</template>
