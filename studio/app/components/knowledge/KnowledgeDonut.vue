<script setup lang="ts">
// Grafik pie (donat) SVG ringan beserta legenda dan persentase. Irisan kecil digabung menjadi "Lainnya".
const props = withDefaults(defineProps<{
  items: { label: string, value: number }[]
  palette: string[]
  emptyText?: string
  max?: number
}>(), { emptyText: 'Belum ada data.', max: 6 })

const slices = computed(() => {
  const sorted = props.items.filter(i => i.value > 0).sort((a, b) => b.value - a.value)
  const head = sorted.slice(0, props.max)
  const rest = sorted.slice(props.max).reduce((s, i) => s + i.value, 0)
  const list = rest > 0 ? [...head, { label: 'Lainnya', value: rest }] : head
  const total = list.reduce((s, i) => s + i.value, 0)
  let acc = 0
  return list.map((i, idx) => {
    const pct = total ? i.value / total * 100 : 0
    const slice = { ...i, pct, offset: -acc, color: props.palette[idx % props.palette.length] }
    acc += pct
    return slice
  })
})
const total = computed(() => slices.value.reduce((s, i) => s + i.value, 0))
</script>

<template>
  <div v-if="!slices.length" class="text-xs text-neutral-400 py-6 text-center">
    {{ emptyText }}
  </div>
  <div v-else class="flex items-center gap-4">
    <svg viewBox="0 0 42 42" class="size-28 shrink-0 -rotate-90" role="img" aria-label="Grafik pie">
      <circle cx="21" cy="21" r="15.9155" fill="none" stroke="currentColor" stroke-width="6" class="text-neutral-100 dark:text-neutral-800" />
      <circle
        v-for="s in slices"
        :key="s.label"
        cx="21"
        cy="21"
        r="15.9155"
        fill="none"
        stroke-width="6"
        :stroke="s.color"
        :stroke-dasharray="`${s.pct} ${100 - s.pct}`"
        :stroke-dashoffset="s.offset"
      />
    </svg>
    <ul class="flex-1 min-w-0 space-y-1">
      <li v-for="s in slices" :key="s.label" class="flex items-center gap-2 text-xs">
        <span class="size-2.5 rounded-sm shrink-0" :style="{ background: s.color }" />
        <span class="flex-1 truncate text-neutral-700 dark:text-neutral-300">{{ s.label }}</span>
        <span class="font-mono text-neutral-500">{{ s.value }}</span>
        <span class="font-mono w-10 text-right font-semibold text-neutral-800 dark:text-neutral-200">{{ s.pct.toFixed(s.pct < 10 ? 1 : 0) }}%</span>
      </li>
      <li class="text-xs text-neutral-400 pt-0.5">
        Total {{ total }}
      </li>
    </ul>
  </div>
</template>
