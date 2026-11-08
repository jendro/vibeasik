<script setup lang="ts">
// Banner pembaruan: muncul bila ada versi baru. "Perbarui sekarang" = unduh, pasang, mulai ulang otomatis
// (logika bersama dengan badge versi di sidebar: useAppUpdate).
const { view, info, stage, ready, available, load, restart, updateNow, setAuto } = useAppUpdate()
const dismissed = ref('')

const KEY = 'vibeasik-update-dismissed'

onMounted(() => {
  try { dismissed.value = sessionStorage.getItem(KEY) || '' } catch { /* tanpa storage */ }
  load()
})

const visible = computed(() => Boolean(available.value && info.value?.latest && dismissed.value !== info.value.latest))

function dismiss() {
  dismissed.value = info.value?.latest || ''
  try { sessionStorage.setItem(KEY, dismissed.value) } catch { /* abaikan */ }
}

const busy = computed(() => stage.value === 'downloading')
const restarting = computed(() => stage.value === 'restarting')
const toggleAuto = (v: boolean | 'indeterminate') => setAuto(v === true)
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
      <UButton v-else-if="info.can_apply" size="xs" color="neutral" variant="solid" icon="i-lucide-download" :loading="busy" @click="updateNow">
        Perbarui sekarang
      </UButton>
      <UButton v-else-if="info.url" size="xs" color="neutral" variant="solid" :to="info.url" target="_blank" trailing-icon="i-lucide-external-link">
        Lihat rilis
      </UButton>
      <UButton size="xs" color="neutral" variant="ghost" class="text-white hover:bg-sky-700" aria-label="Tutup" icon="i-lucide-x" @click="dismiss" />
    </div>
  </div>
</template>
