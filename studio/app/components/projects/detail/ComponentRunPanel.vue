<script setup lang="ts">
import type { ProjectComponent, RunStatus } from '~/types'

const props = defineProps<{
  comp: ProjectComponent
}>()

const { getRunStatus, runComponent, stopComponent } = useRunner()
const toast = useToast()

const status = ref<RunStatus | null>(null)
const busy = ref(false)
const showLogs = ref(false)
const logBox = ref<HTMLElement | null>(null)
let pollTimer: ReturnType<typeof setTimeout> | null = null

const isActive = computed(() => status.value?.state === 'starting' || status.value?.state === 'running')
const port = computed(() => status.value?.port || props.comp.run_port || 0)
const openUrl = computed(() => {
  const host = import.meta.client ? window.location.hostname : 'localhost'
  return port.value ? `http://${host}:${port.value}` : ''
})

const stateConfig = computed(() => {
  switch (status.value?.state) {
    case 'running':
      return { label: 'RUNNING', dot: 'bg-emerald-500 animate-pulse', text: 'text-emerald-600 dark:text-emerald-400' }
    case 'starting':
      return { label: 'STARTING', dot: 'bg-amber-400 animate-pulse', text: 'text-amber-600 dark:text-amber-400' }
    case 'exited':
      return { label: `EXIT ${status.value?.exit_code ?? ''}`.trim(), dot: 'bg-rose-500', text: 'text-rose-600 dark:text-rose-400' }
    case 'stopped':
      return { label: 'STOPPED', dot: 'bg-neutral-400', text: 'text-neutral-400' }
    default:
      return { label: 'IDLE', dot: 'bg-neutral-300 dark:bg-neutral-600', text: 'text-neutral-400' }
  }
})

const refresh = async () => {
  try {
    status.value = await getRunStatus(props.comp.id, showLogs.value)
    if (showLogs.value) {
      await nextTick()
      if (logBox.value) logBox.value.scrollTop = logBox.value.scrollHeight
    }
  } catch {
    // engine lama tanpa endpoint runner: panel tetap tampil dalam keadaan idle
  }
}

// Polling hanya selama proses aktif atau log sedang dibuka, supaya tidak membebani engine.
const schedulePoll = () => {
  if (pollTimer) clearTimeout(pollTimer)
  if (!isActive.value && !showLogs.value) return
  pollTimer = setTimeout(async () => {
    await refresh()
    schedulePoll()
  }, 2000)
}

const errorMessage = (err: any) => err?.data?.error || err?.message || 'Terjadi kesalahan'

const start = async (reassign = false) => {
  busy.value = true
  try {
    status.value = await runComponent(props.comp.id, reassign)
    toast.add({
      title: `${props.comp.name} dijalankan`,
      description: `Port :${status.value.port}${status.value.notes?.length ? ' — ' + status.value.notes.join('; ') : ''}`,
      icon: 'i-lucide-play',
      color: 'success'
    })
    schedulePoll()
  } catch (err: any) {
    if (err?.data?.code === 'port_busy') {
      toast.add({
        title: `Port :${err.data.port} sedang dipakai`,
        description: 'Ada proses lain (di luar vibeasik) di port ini. Pindahkan komponen ke port bebas?',
        icon: 'i-lucide-triangle-alert',
        color: 'warning',
        actions: [{ label: 'Pakai port lain & jalankan', color: 'warning', variant: 'solid', onClick: () => { start(true) } }]
      })
    } else {
      toast.add({ title: 'Gagal menjalankan', description: errorMessage(err), icon: 'i-lucide-circle-x', color: 'error' })
    }
  } finally {
    busy.value = false
  }
}

const stop = async () => {
  busy.value = true
  try {
    status.value = await stopComponent(props.comp.id)
    toast.add({ title: `${props.comp.name} dihentikan`, icon: 'i-lucide-square', color: 'neutral' })
  } catch (err: any) {
    toast.add({ title: 'Gagal menghentikan', description: errorMessage(err), icon: 'i-lucide-circle-x', color: 'error' })
    await refresh()
  } finally {
    busy.value = false
  }
}

const toggleLogs = async () => {
  showLogs.value = !showLogs.value
  if (showLogs.value) await refresh()
  schedulePoll()
}

onMounted(async () => {
  await refresh()
  schedulePoll()
})

onBeforeUnmount(() => {
  if (pollTimer) clearTimeout(pollTimer)
})
</script>

<template>
  <div class="p-2.5 rounded-lg border border-neutral-100 dark:border-neutral-800 text-xs space-y-2">
    <!-- flex-wrap: di kartu sempit tombol turun ke baris baru, bukan menimpa label status -->
    <div class="flex flex-wrap items-center justify-between gap-x-2 gap-y-1.5">
      <div class="flex items-center gap-2 whitespace-nowrap">
        <span class="text-[10px] uppercase text-neutral-400">Run</span>
        <span
          v-if="port"
          class="px-1.5 py-0.5 rounded font-mono font-bold bg-neutral-100 dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200"
          title="Port unik yang dialokasikan vibeasik (disuntik sebagai env PORT)"
        >
          :{{ port }}
        </span>
        <span class="flex items-center gap-1 font-mono text-[10px] font-semibold" :class="stateConfig.text">
          <span class="w-2 h-2 rounded-full" :class="stateConfig.dot" />
          {{ stateConfig.label }}
        </span>
      </div>

      <div class="flex items-center gap-1 ml-auto">
        <UButton
          v-if="isActive && status?.listening"
          :to="openUrl"
          target="_blank"
          icon="i-lucide-external-link"
          color="neutral"
          variant="ghost"
          size="xs"
          title="Buka aplikasi yang sedang berjalan"
        />
        <UButton
          icon="i-lucide-scroll-text"
          :color="showLogs ? 'primary' : 'neutral'"
          variant="ghost"
          size="xs"
          title="Tampilkan log dev server"
          @click="toggleLogs"
        />
        <UButton
          v-if="isActive"
          icon="i-lucide-square"
          color="error"
          variant="soft"
          size="xs"
          :loading="busy"
          @click="stop"
        >
          Stop
        </UButton>
        <UButton
          v-else
          icon="i-lucide-play"
          color="success"
          variant="soft"
          size="xs"
          :loading="busy"
          :disabled="!comp.dev_command"
          :title="comp.dev_command ? 'Jalankan dev command di port unik dari vibeasik' : 'Isi dev command dulu lewat tombol edit'"
          @click="start()"
        >
          Run
        </UButton>
      </div>
    </div>

    <p v-if="status?.warning" class="text-[11px] text-amber-700 dark:text-amber-400 flex gap-1.5">
      <UIcon name="i-lucide-triangle-alert" class="w-3.5 h-3.5 shrink-0 mt-px" />
      <span>{{ status.warning }}</span>
    </p>

    <div v-if="showLogs" class="space-y-1">
      <div v-if="status?.command" class="font-mono text-[10px] text-neutral-400 truncate" :title="status.command">
        $ {{ status.command }}
      </div>
      <pre
        ref="logBox"
        class="max-h-56 overflow-auto rounded bg-neutral-950 text-neutral-200 p-2 font-mono text-[10.5px] leading-relaxed whitespace-pre-wrap break-all"
      >{{ status?.logs?.length ? status.logs.join('\n') : 'Belum ada log. Tekan Run untuk menjalankan.' }}</pre>
    </div>
  </div>
</template>
