<script setup lang="ts">
import type { Todo } from '~/types'
import { impactAudienceLabel, impactMeta, impactPerEffort } from '~/composables/useTodoImpact'

/** Chip skor dampak todo (0–100) + popover rincian 7 dimensi, bobot kategori, dan apa yang meningkat. */
const props = defineProps<{ todo: Todo }>()

const { info, values } = useTodoImpact()

const meta = computed(() => impactMeta(props.todo.impact_level))
const groups = computed(() => {
  const dims = info.value?.dimensions || []
  const vals = values(props.todo)
  const weights = info.value?.weights?.[props.todo.category || 'general'] || info.value?.weights?.general || []
  const maxW = Math.max(...weights, 0.0001)
  const out: { audience: string, items: { key: string, label: string, hint: string, value: number, weight: number }[] }[] = []
  dims.forEach((d, i) => {
    let g = out.find(x => x.audience === d.audience)
    if (!g) out.push(g = { audience: d.audience, items: [] })
    g.items.push({ key: d.key, label: d.label, hint: d.hint, value: vals[i] ?? 0, weight: (weights[i] ?? 0) / maxW })
  })
  return out
})
const sourceLabel = computed(() => ({ llm: 'Dinilai LLM', manual: 'Dinilai manual/agent', heuristic: 'Taksiran otomatis' } as Record<string, string>)[props.todo.impact_source || ''] || 'Taksiran otomatis')
const perEffort = computed(() => impactPerEffort(props.todo).toFixed(0))
</script>

<template>
  <UPopover v-if="todo.impact_level" mode="hover" :content="{ side: 'top', align: 'start' }">
    <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-medium text-xs border cursor-default" :class="meta.class">
      <UIcon :name="meta.icon" class="w-3.5 h-3.5" />
      {{ meta.label }} · {{ todo.impact_score }}
    </span>
    <template #content>
      <div class="p-3 w-72 space-y-2.5 text-xs">
        <div class="flex items-center justify-between font-semibold text-neutral-800 dark:text-neutral-100">
          <span>{{ meta.label }}</span>
          <span class="font-mono">{{ todo.impact_score }} / 100</span>
        </div>
        <p v-if="todo.impact_summary" class="text-xs text-neutral-700 dark:text-neutral-200 leading-snug">
          {{ todo.impact_summary }}
        </p>

        <div v-for="g in groups" :key="g.audience" class="space-y-1">
          <div class="text-xs font-semibold text-neutral-400">
            {{ impactAudienceLabel[g.audience] || g.audience }}
          </div>
          <div v-for="d in g.items" :key="d.key" :title="d.hint" class="space-y-0.5">
            <div class="flex justify-between text-xs text-neutral-600 dark:text-neutral-300">
              <span :class="{ 'text-neutral-400 dark:text-neutral-500': d.value === 0 }">
                {{ d.label }}
                <span v-if="d.weight >= 0.99" class="ml-0.5 text-xs text-violet-500" title="Bobot terbesar untuk kategori todo ini">★</span>
              </span>
              <span class="font-mono">{{ d.value }}</span>
            </div>
            <div class="h-1.5 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
              <div class="h-full rounded-full bg-linear-to-r from-sky-400 to-violet-500" :style="{ width: `${(d.value / 5) * 100}%`, opacity: 0.45 + 0.55 * d.weight }" />
            </div>
          </div>
        </div>

        <div class="pt-1.5 border-t border-neutral-100 dark:border-white/[0.08] text-xs text-neutral-500 space-y-0.5">
          <div v-if="todo.complexity_score > 0" class="flex justify-between">
            <span title="Skor dampak dibagi skor kerumitan: makin tinggi makin layak dikerjakan dulu">Dampak per usaha</span>
            <span class="font-mono">{{ perEffort }}</span>
          </div>
          <p>{{ sourceLabel }} · bobot mengikuti kategori todo (★ = paling menentukan)</p>
        </div>
      </div>
    </template>
  </UPopover>
</template>
