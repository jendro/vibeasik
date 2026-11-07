<script setup lang="ts">
import type { RuntimeSimComponent, RuntimeSimEvent } from '~/types'

const props = defineProps<{
  events: RuntimeSimEvent[]
  components: RuntimeSimComponent[]
  running: boolean
}>()

// 'activity' = apa yang sedang dicek simulasi; selain itu = component_id untuk log proses
const activeTab = ref<'activity' | number>('activity')
const scroller = ref<HTMLElement | null>(null)
const stickToBottom = ref(true)

const logTabs = computed(() => props.components.filter(c => (c.logs?.length || 0) > 0))

const activeLines = computed(() => {
  if (activeTab.value === 'activity') return []
  return props.components.find(c => c.component_id === activeTab.value)?.logs || []
})

const levelClass: Record<string, string> = {
  ok: 'text-emerald-400',
  warn: 'text-amber-300',
  error: 'text-rose-400',
  info: 'text-neutral-300'
}

const formatTime = (iso: string) => {
  try {
    return new Date(iso).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    return ''
  }
}

const isErrorLine = (line: string) => /(panic:|fatal|uncaught|unhandled|traceback|error|exception)/i.test(line)

const onScroll = () => {
  const el = scroller.value
  if (!el) return
  stickToBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < 40
}

watch(
  () => [props.events.length, activeLines.value.length, activeTab.value],
  async () => {
    if (!stickToBottom.value) return
    await nextTick()
    if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight
  },
  { immediate: true }
)
</script>

<template>
  <div class="rounded-xl overflow-hidden border border-neutral-800 bg-neutral-950 flex flex-col h-[420px]">
    <div class="flex items-center gap-1 px-2 pt-2 bg-neutral-900 border-b border-neutral-800 overflow-x-auto shrink-0">
      <span class="flex items-center gap-1 px-1.5 mr-1">
        <span class="w-2.5 h-2.5 rounded-full bg-rose-500" /><span class="w-2.5 h-2.5 rounded-full bg-amber-400" /><span class="w-2.5 h-2.5 rounded-full bg-emerald-500" />
      </span>
      <button
        type="button"
        class="px-2.5 py-1 rounded-t-md text-[11px] font-mono whitespace-nowrap cursor-pointer"
        :class="activeTab === 'activity' ? 'bg-neutral-950 text-white' : 'text-neutral-400 hover:text-neutral-200'"
        @click="activeTab = 'activity'"
      >
        <UIcon v-if="running" name="i-lucide-loader-2" class="w-3 h-3 animate-spin inline -mt-0.5 mr-1" />aktivitas pengecekan
      </button>
      <button
        v-for="c in logTabs"
        :key="c.component_id"
        type="button"
        class="px-2.5 py-1 rounded-t-md text-[11px] font-mono whitespace-nowrap cursor-pointer"
        :class="activeTab === c.component_id ? 'bg-neutral-950 text-white' : 'text-neutral-400 hover:text-neutral-200'"
        @click="activeTab = c.component_id"
      >
        {{ c.name }}.log
        <span v-if="c.error_lines" class="ml-1 px-1 rounded bg-rose-500/20 text-rose-300">{{ c.error_lines }}</span>
      </button>
    </div>

    <div ref="scroller" class="flex-1 overflow-y-auto p-3 font-mono text-[11px] leading-relaxed" @scroll="onScroll">
      <template v-if="activeTab === 'activity'">
        <div v-if="events.length === 0" class="text-neutral-500">
          Menunggu simulasi dimulai...
        </div>
        <div v-for="(ev, idx) in events" :key="idx" class="flex gap-2 whitespace-pre-wrap break-all">
          <span class="text-neutral-600 shrink-0">{{ formatTime(ev.at) }}</span>
          <span :class="[levelClass[ev.level] || 'text-neutral-300', ev.message.startsWith('—') ? 'font-bold text-sky-300' : '']">{{ ev.message }}</span>
        </div>
        <div v-if="running" class="text-neutral-500 animate-pulse">
          ▌
        </div>
      </template>
      <template v-else>
        <div
          v-for="(line, idx) in activeLines"
          :key="idx"
          class="whitespace-pre-wrap break-all"
          :class="idx === 0 && line.startsWith('$ ') ? 'text-sky-300' : isErrorLine(line) ? 'text-rose-400' : 'text-neutral-300'"
        >
          {{ line || ' ' }}
        </div>
      </template>
    </div>
  </div>
</template>
