<script setup lang="ts">
import type { TrafficRecommendation, TrafficSeverity, TrafficWindow } from '~/types'
import type { StatItem } from '~/components/AppStatStrip.vue'

// Saran optimasi dari data traffic nyata. "Jadikan Todo" → todo saran (status suggested) di proyek sumber.
const props = defineProps<{ items: TrafficRecommendation[], sourceId: number, window: TrafficWindow, hasProject: boolean, projectId?: number | null }>()

const toast = useToast()
const { recommendationToTodo } = useTraffic()
const filter = ref<string>('all')
const busy = ref<string | null>(null)
const expanded = ref<Set<string>>(new Set())

const sevMeta: Record<TrafficSeverity, { label: string, icon: string, cls: string }> = {
  critical: { label: 'Kritis', icon: 'i-lucide-octagon-alert', cls: 'text-rose-700 dark:text-rose-300 bg-rose-50 dark:bg-rose-950/40 border-rose-200 dark:border-rose-900/60' },
  warning: { label: 'Perlu tindakan', icon: 'i-lucide-triangle-alert', cls: 'text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-950/40 border-amber-200 dark:border-amber-900/60' },
  info: { label: 'Info', icon: 'i-lucide-info', cls: 'text-sky-700 dark:text-sky-300 bg-sky-50 dark:bg-sky-950/40 border-sky-200 dark:border-sky-900/60' }
}

const categories = computed(() => {
  const counts = new Map<string, number>()
  for (const r of props.items) counts.set(r.category, (counts.get(r.category) || 0) + 1)
  return [...counts.entries()]
})
const visible = computed(() => filter.value === 'all' ? props.items : props.items.filter(r => r.category === filter.value))

const toggle = (id: string) => {
  const s = new Set(expanded.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  expanded.value = s
}

const makeTodo = async (r: TrafficRecommendation) => {
  busy.value = r.id
  try {
    const res = await recommendationToTodo(props.sourceId, r.id, props.window)
    toast.add({
      title: res.already_exists ? 'Todo sudah ada' : 'Todo saran dibuat',
      description: res.todo.title,
      icon: 'i-lucide-clipboard-check',
      color: res.already_exists ? 'info' : 'success'
    })
  } catch (err: any) {
    toast.add({ title: 'Gagal membuat todo', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' })
  } finally {
    busy.value = null
  }
}

// Rekap saran → "Jadikan semua todo": tiap saran dijadikan todo satu per satu (tanpa LLM; saran yang sudah punya
// todo terbuka dilewati oleh engine).
const sevCount = (s: TrafficSeverity) => props.items.filter(r => r.severity === s).length
const recapStats = computed<StatItem[]>(() => [
  { label: 'Saran', value: props.items.length, hint: `dari traffic ${props.window}` },
  { label: 'Kritis', value: sevCount('critical'), dot: 'bg-rose-500', tone: sevCount('critical') ? 'text-rose-600 dark:text-rose-400' : undefined },
  { label: 'Perlu tindakan', value: sevCount('warning'), dot: 'bg-amber-500' },
  { label: 'Info', value: sevCount('info'), dot: 'bg-sky-500' }
])
const allBusy = ref(false)
const makeAllTodos = async () => {
  allBusy.value = true
  let created = 0
  let existed = 0
  const failed: string[] = []
  for (const r of props.items) {
    try {
      const res = await recommendationToTodo(props.sourceId, r.id, props.window)
      if (res.already_exists) existed++
      else created++
    } catch (err: any) {
      failed.push(`${r.title}: ${err?.data?.error || err?.message || 'gagal'}`)
    }
  }
  allBusy.value = false
  toast.add({
    title: created ? `${created} todo saran dibuat` : 'Tidak ada todo baru',
    description: [existed ? `${existed} saran sudah punya todo.` : '', failed.length ? `${failed.length} gagal — ${failed[0]}` : ''].filter(Boolean).join(' ') || undefined,
    icon: 'i-lucide-clipboard-check',
    color: failed.length ? 'warning' : created ? 'success' : 'neutral',
    actions: [{ label: 'Buka Todo', onClick: () => { navigateTo('/todos') } }]
  })
}
</script>

<template>
  <div class="space-y-3">
    <FindingsTodoBar
      v-if="items.length"
      :project-id="hasProject ? (projectId ?? sourceId) : null"
      :stats="recapStats"
      :count="hasProject ? items.length : 0"
      :note="hasProject ? 'Setiap saran menjadi todo saran berisi bukti traffic nyata; saran yang sudah punya todo dilewati.' : 'Hubungkan sumber traffic ini ke proyek agar sarannya bisa dijadikan todo.'"
      todo-desc="Semua saran traffic dijadikan todo saran (status disarankan) di proyek sumber — tanpa LLM"
      :busy="allBusy"
      @todo-all="makeAllTodos"
    />
    <div class="flex flex-wrap gap-1.5">
      <UButton size="xs" :color="filter === 'all' ? 'primary' : 'neutral'" :variant="filter === 'all' ? 'soft' : 'ghost'" @click="filter = 'all'">
        Semua · {{ items.length }}
      </UButton>
      <UButton
        v-for="[cat, n] in categories"
        :key="cat"
        size="xs"
        :icon="trafficCategoryMeta[cat]?.icon"
        :color="filter === cat ? 'primary' : 'neutral'"
        :variant="filter === cat ? 'soft' : 'ghost'"
        @click="filter = cat"
      >
        {{ trafficCategoryMeta[cat]?.label || cat }} · {{ n }}
      </UButton>
    </div>

    <div class="space-y-2">
      <div v-for="r in visible" :key="r.id" class="rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f]">
        <button type="button" class="w-full text-left p-3 flex items-start gap-3 cursor-pointer" @click="toggle(r.id)">
          <span class="shrink-0 inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md border text-xs font-semibold" :class="sevMeta[r.severity].cls">
            <UIcon :name="sevMeta[r.severity].icon" class="w-3 h-3" />{{ sevMeta[r.severity].label }}
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-1.5 text-sm font-semibold text-neutral-900 dark:text-white">
              <UIcon :name="trafficCategoryMeta[r.category]?.icon || 'i-lucide-info'" class="w-4 h-4 shrink-0 text-neutral-400" />
              <span class="break-words">{{ r.title }}</span>
            </div>
            <p class="text-xs text-neutral-600 dark:text-neutral-300 mt-1 leading-relaxed" :class="expanded.has(r.id) ? '' : 'line-clamp-2'">
              {{ r.detail }}
            </p>
          </div>
          <UIcon :name="expanded.has(r.id) ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" class="w-4 h-4 shrink-0 text-neutral-400 mt-0.5" />
        </button>
        <div v-if="expanded.has(r.id)" class="px-3 pb-3 space-y-2 text-xs">
          <div class="rounded-lg bg-neutral-50 dark:bg-neutral-800/60 p-2.5 text-neutral-700 dark:text-neutral-200 leading-relaxed">
            <span class="font-semibold">Langkah:</span> {{ r.action }}
          </div>
          <ul class="space-y-0.5 text-neutral-600 dark:text-neutral-300">
            <li v-for="e in r.evidence" :key="e" class="flex gap-1.5">
              <UIcon name="i-lucide-chart-no-axes-column" class="w-3.5 h-3.5 mt-0.5 shrink-0 text-neutral-400" /><span class="font-mono text-xs break-all">{{ e }}</span>
            </li>
          </ul>
          <div v-if="r.routes?.length" class="flex flex-wrap gap-1">
            <UBadge v-for="rt in r.routes" :key="rt" size="sm" color="neutral" variant="soft" class="font-mono">
              {{ rt }}
            </UBadge>
          </div>
          <div class="flex items-center justify-between gap-2 pt-1">
            <span class="text-xs font-mono text-neutral-400">{{ r.rule }}</span>
            <AppActionButton
              v-if="r.category !== 'info'"
              kind="process"
              size="xs"
              label="Jadikan todo"
              desc="Saran trafik ini menjadi todo proyek — tanpa LLM"
              :loading="busy === r.id"
              :disabled="!hasProject"
              @click="makeTodo(r)"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
