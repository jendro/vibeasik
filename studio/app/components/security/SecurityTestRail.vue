<script setup lang="ts">
import type { SecurityTestSession } from '~/types'
import { securityGradeColor } from '~/utils/securityTest'

// Sidebar kanan Uji Keamanan (bentuk sama dengan Uji Runtime): status & progres, langkah, dan aktivitas
// request ke server. Uji berjalan di engine, jadi halaman boleh ditinggal.
const props = defineProps<{
  session: SecurityTestSession
  running: boolean
}>()

const steps = computed(() => props.session.steps || [])
const doneCount = computed(() => steps.value.filter(s => s.status !== 'pending' && s.status !== 'running').length)
const current = computed(() => {
  const idx = steps.value.findIndex(s => s.status === 'running')
  return idx >= 0 ? idx : Math.max(0, doneCount.value - 1)
})
const percent = computed(() => {
  if (!steps.value.length) return 0
  if (!props.running) return 100
  return Math.round(((doneCount.value + 0.5) / steps.value.length) * 100)
})

const now = ref(Date.now())
let tick: ReturnType<typeof setInterval> | null = null
watch(() => props.running, (on) => {
  if (tick) clearInterval(tick)
  tick = on ? setInterval(() => { now.value = Date.now() }, 1000) : null
}, { immediate: true })
onUnmounted(() => tick && clearInterval(tick))

const elapsed = computed(() => {
  const start = new Date(props.session.started_at).getTime()
  const end = props.session.finished_at ? new Date(props.session.finished_at).getTime() : now.value
  const sec = Math.max(0, Math.round((end - start) / 1000))
  return sec >= 60 ? `${Math.floor(sec / 60)}m ${String(sec % 60).padStart(2, '0')}s` : `${sec}s`
})

const statusText = computed(() => ({ running: 'Sedang berjalan', completed: 'Selesai', cancelled: 'Dibatalkan' } as Record<string, string>)[props.session.status] || 'Gagal')

const stepIcon = (status: string) => ({
  running: 'i-lucide-loader-2',
  done: 'i-lucide-check',
  failed: 'i-lucide-x',
  skipped: 'i-lucide-minus'
} as Record<string, string>)[status] || ''

const stepDot = (status: string) => ({
  running: 'bg-sky-500 text-white ring-4 ring-sky-100 dark:ring-sky-950',
  done: 'bg-emerald-500 text-white',
  failed: 'bg-rose-500 text-white',
  skipped: 'bg-neutral-200 dark:bg-neutral-700 text-neutral-500'
} as Record<string, string>)[status] || 'bg-neutral-100 dark:bg-neutral-800 border border-neutral-300 dark:border-neutral-600 text-neutral-400'

const formatMs = (ms?: number) => !ms ? '' : ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${ms}ms`

const events = computed(() => (props.session.events || []).slice(-150))
const feed = ref<HTMLElement | null>(null)
const stick = ref(true)
const onScroll = () => {
  const el = feed.value
  if (el) stick.value = el.scrollHeight - el.scrollTop - el.clientHeight < 40
}
watch(() => props.session.events?.length, async () => {
  if (!stick.value) return
  await nextTick()
  if (feed.value) feed.value.scrollTop = feed.value.scrollHeight
}, { immediate: true })

const levelCls: Record<string, string> = {
  ok: 'text-emerald-600 dark:text-emerald-400',
  warn: 'text-amber-600 dark:text-amber-300',
  error: 'text-rose-600 dark:text-rose-400',
  info: 'text-neutral-600 dark:text-neutral-300'
}
const time = (iso: string) => {
  try {
    return new Date(iso).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    return ''
  }
}
</script>

<template>
  <aside class="rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] flex flex-col overflow-hidden lg:max-h-[calc(100vh-5rem)]">
    <div class="px-4 py-3 border-b border-neutral-200 dark:border-white/[0.08] space-y-2 shrink-0">
      <div class="flex items-center justify-between gap-2">
        <span class="inline-flex items-center gap-1.5 text-sm font-semibold text-neutral-900 dark:text-white">
          <span class="size-2 rounded-full" :class="running ? 'bg-sky-500 animate-pulse' : session.status === 'completed' ? 'bg-emerald-500' : 'bg-amber-500'" />
          {{ statusText }}
        </span>
        <span class="text-xs tabular-nums text-neutral-500">{{ elapsed }}</span>
      </div>
      <div class="h-1.5 rounded-full bg-neutral-100 dark:bg-white/10 overflow-hidden">
        <div class="h-full rounded-full transition-all duration-500" :class="running ? 'bg-sky-500' : 'bg-emerald-500'" :style="{ width: `${percent}%` }" />
      </div>
      <p class="text-xs text-neutral-500 truncate">
        <template v-if="running">
          Langkah {{ current + 1 }}/{{ steps.length }} · {{ steps[current]?.title }}
        </template>
        <template v-else-if="session.report">
          {{ session.report.summary.fail }} gagal · {{ session.report.summary.warn }} lemah · skor
          <b class="tabular-nums" :class="securityGradeColor(session.report.score)">{{ session.report.score }}/100</b>
        </template>
        <template v-else>
          {{ session.summary }}
        </template>
      </p>
      <p v-if="running" class="flex items-start gap-1.5 text-xs text-sky-700 dark:text-sky-300 bg-sky-50 dark:bg-sky-500/10 rounded-lg px-2.5 py-1.5">
        <UIcon name="i-lucide-info" class="size-3.5 mt-0.5 shrink-0" />
        Boleh ditinggal: uji tetap berjalan di engine; buka halaman ini lagi untuk melihat hasilnya.
      </p>
    </div>

    <ol class="px-4 py-3 space-y-1.5 border-b border-neutral-200 dark:border-white/[0.08] shrink-0">
      <li v-for="(step, idx) in steps" :key="step.key" class="flex items-start gap-2.5" :title="step.detail || step.title">
        <span class="size-5 rounded-full flex items-center justify-center shrink-0 mt-px" :class="stepDot(step.status)">
          <UIcon v-if="stepIcon(step.status)" :name="stepIcon(step.status)" class="size-3" :class="{ 'animate-spin': step.status === 'running' }" />
          <span v-else class="text-xs font-bold font-mono">{{ idx + 1 }}</span>
        </span>
        <div class="min-w-0 flex-1">
          <div class="text-xs font-semibold leading-5" :class="step.status === 'running' ? 'text-sky-700 dark:text-sky-300' : step.status === 'pending' || step.status === 'skipped' ? 'text-neutral-400' : step.status === 'failed' ? 'text-rose-700 dark:text-rose-300' : 'text-neutral-800 dark:text-neutral-200'">
            {{ step.title }}
            <span v-if="step.duration_ms" class="font-normal font-mono text-neutral-400"> · {{ formatMs(step.duration_ms) }}</span>
          </div>
          <p v-if="step.detail && step.status !== 'pending'" class="text-xs text-neutral-500 leading-snug line-clamp-2">
            {{ step.detail }}
          </p>
        </div>
      </li>
    </ol>

    <div class="flex items-center gap-1.5 px-4 pt-3 pb-1.5 text-xs font-semibold text-neutral-700 dark:text-neutral-200 shrink-0">
      <UIcon :name="running ? 'i-lucide-loader-2' : 'i-lucide-activity'" class="size-3.5 text-sky-500" :class="{ 'animate-spin': running }" />
      Aktivitas pengecekan
      <span class="font-normal text-neutral-400 tabular-nums">{{ session.events?.length || 0 }}</span>
    </div>
    <div ref="feed" class="flex-1 min-h-0 max-h-80 lg:max-h-none overflow-y-auto px-4 pb-3 space-y-1 font-mono text-xs leading-relaxed" @scroll="onScroll">
      <div v-if="!events.length" class="text-neutral-400">
        Menunggu uji dimulai...
      </div>
      <div v-for="(ev, idx) in events" :key="idx" class="flex gap-2 break-words">
        <span class="text-neutral-400 shrink-0">{{ time(ev.at) }}</span>
        <span class="min-w-0" :class="levelCls[ev.level] || levelCls.info">{{ ev.message }}</span>
      </div>
    </div>
  </aside>
</template>
