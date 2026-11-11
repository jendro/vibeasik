<script setup lang="ts">
// Lonceng Notifikasi di menu atas: jumlah request LLM luar yang gagal berulang dan menunggu dieksekusi dari
// lokal. Dicek tiap menit selama tab terlihat; endpoint hanya ada di edisi dengan LLM Router — bila tidak
// tersedia lonceng disembunyikan. Jumlah yang naik dikabarkan sekali lewat toast.
const api = useApi()
const toast = useToast()
const available = ref(false)
const open = ref(0)
const answered = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

const load = async () => {
  if (typeof document !== 'undefined' && document.visibilityState === 'hidden') return
  try {
    const res = await api.getNotifications({ limit: 1 })
    const n = (res.counts?.open ?? 0) + (res.counts?.running ?? 0)
    if (available.value && n > open.value) {
      toast.add({
        title: 'Request ke LLM API gagal',
        description: 'Semua model gagal menjawab beberapa kali. Eksekusi prompt-nya dari lokal di halaman Notifikasi.',
        icon: 'i-lucide-bell-ring',
        color: 'warning',
        actions: [{ label: 'Buka Notifikasi', onClick: () => { navigateTo('/notifications') } }]
      })
    }
    open.value = n
    answered.value = res.counts?.answered ?? 0
    available.value = true
  } catch {
    available.value = false
  }
}

onMounted(() => {
  load()
  timer = setInterval(load, 60_000)
})
onBeforeUnmount(() => { if (timer) clearInterval(timer) })

const title = computed(() => open.value
  ? `${open.value} request LLM gagal menunggu dieksekusi dari lokal. Klik untuk membuka Notifikasi.`
  : 'Notifikasi: tidak ada request LLM yang gagal.')
</script>

<template>
  <UButton
    v-if="available"
    to="/notifications"
    size="sm"
    color="neutral"
    variant="ghost"
    :icon="open ? 'i-lucide-bell-ring' : 'i-lucide-bell'"
    :title="title"
    :aria-label="title"
    class="relative"
  >
    <span
      v-if="open"
      class="absolute -top-0.5 -right-0.5 min-w-4 h-4 px-1 rounded-full bg-rose-500 text-white text-[10px] font-semibold leading-4 text-center tabular-nums"
    >{{ open > 99 ? '99+' : open }}</span>
  </UButton>
</template>
