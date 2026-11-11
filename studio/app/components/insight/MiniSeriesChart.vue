<script setup lang="ts">
// Grafik kecil satu ukuran (tanpa dua sumbu Y): batang atau garis, dengan crosshair + tooltip saat hover.
const props = withDefaults(defineProps<{
  values: number[]
  labels: string[]
  mode?: 'bar' | 'line'
  // Kelas warna Tailwind untuk isi/garis, mis. "fill-sky-500" / "stroke-sky-500".
  tone?: 'sky' | 'emerald' | 'amber' | 'rose'
  unit?: string
  height?: number
  label: string
  // Batas atas tetap (mis. 100 untuk skor); kosong = menyesuaikan data.
  max?: number
}>(), { mode: 'bar', tone: 'sky', unit: '', height: 150, max: 0 })

const TONES = {
  sky: { fill: 'fill-sky-500 dark:fill-sky-400', stroke: 'stroke-sky-500 dark:stroke-sky-400', area: 'fill-sky-500/10' },
  emerald: { fill: 'fill-emerald-500 dark:fill-emerald-400', stroke: 'stroke-emerald-500 dark:stroke-emerald-400', area: 'fill-emerald-500/10' },
  amber: { fill: 'fill-amber-500 dark:fill-amber-400', stroke: 'stroke-amber-500 dark:stroke-amber-400', area: 'fill-amber-500/10' },
  rose: { fill: 'fill-rose-500 dark:fill-rose-400', stroke: 'stroke-rose-500 dark:stroke-rose-400', area: 'fill-rose-500/10' }
}
const tone = computed(() => TONES[props.tone])

const wrap = ref<HTMLElement | null>(null)
const width = ref(480)
const pad = { l: 34, r: 8, t: 10, b: 20 }
const hover = ref<number | null>(null)
let ro: ResizeObserver | null = null
onMounted(() => {
  if (!wrap.value) return
  ro = new ResizeObserver(([e]) => { if (e) width.value = Math.max(200, e.contentRect.width) })
  ro.observe(wrap.value)
})
onBeforeUnmount(() => ro?.disconnect())

const maxY = computed(() => {
  if (props.max > 0) return props.max
  const m = Math.max(0, ...props.values)
  if (m <= 4) return 4
  const mag = 10 ** Math.floor(Math.log10(m))
  return Math.ceil(m / mag * 1.1) * mag
})
const innerW = computed(() => width.value - pad.l - pad.r)
const innerH = computed(() => props.height - pad.t - pad.b)
const slot = computed(() => innerW.value / Math.max(1, props.values.length))
const cx = (i: number) => pad.l + i * slot.value + slot.value / 2
const y = (v: number) => pad.t + innerH.value - (v / maxY.value) * innerH.value
const barW = computed(() => Math.max(1, Math.min(22, slot.value - 2)))

const ticks = computed(() => [0, 0.5, 1].map(f => ({ v: Math.round(maxY.value * f), y: y(maxY.value * f) })))
const xTicks = computed(() => {
  const n = props.labels.length
  const every = Math.max(1, Math.ceil(n / Math.max(2, Math.floor(innerW.value / 70))))
  return props.labels.map((label, i) => ({ i, label })).filter(({ i }) => i % every === 0)
})
const linePath = computed(() => props.values.map((v, i) => `${i ? 'L' : 'M'}${cx(i).toFixed(1)},${y(v).toFixed(1)}`).join(''))
const areaPath = computed(() => props.values.length
  ? `${linePath.value}L${cx(props.values.length - 1).toFixed(1)},${y(0)}L${cx(0).toFixed(1)},${y(0)}Z`
  : '')

const onMove = (e: MouseEvent) => {
  const rect = (e.currentTarget as SVGElement).getBoundingClientRect()
  const i = Math.floor((e.clientX - rect.left - pad.l) / slot.value)
  hover.value = i >= 0 && i < props.values.length ? i : null
}
const tipLeft = computed(() => hover.value === null ? 0 : Math.min(width.value - 130, Math.max(0, cx(hover.value) - 65)))
</script>

<template>
  <div ref="wrap" class="relative w-full min-w-0 overflow-hidden select-none">
    <svg :width="width" :height="height" class="block" role="img" :aria-label="label" @mousemove="onMove" @mouseleave="hover = null">
      <line v-for="t in ticks" :key="t.v" :x1="pad.l" :x2="width - pad.r" :y1="t.y" :y2="t.y" class="stroke-neutral-200 dark:stroke-neutral-800" stroke-width="1" />
      <text v-for="t in ticks" :key="'l' + t.v" :x="pad.l - 6" :y="t.y + 3" text-anchor="end" class="fill-neutral-500 text-xs">{{ t.v }}</text>
      <text v-for="t in xTicks" :key="'x' + t.i" :x="cx(t.i)" :y="height - 4" text-anchor="middle" class="fill-neutral-500 text-xs">{{ t.label }}</text>
      <line v-if="hover !== null" :x1="cx(hover)" :x2="cx(hover)" :y1="pad.t" :y2="pad.t + innerH" class="stroke-neutral-400 dark:stroke-neutral-500" stroke-width="1" stroke-dasharray="3 3" />
      <template v-if="mode === 'bar'">
        <rect
          v-for="(v, i) in values"
          :key="i"
          :x="cx(i) - barW / 2"
          :y="y(v)"
          :width="barW"
          :height="Math.max(0, (v / maxY) * innerH)"
          rx="2"
          :class="tone.fill"
          :opacity="hover === null || hover === i ? 1 : 0.5"
        />
      </template>
      <template v-else>
        <path :d="areaPath" :class="tone.area" />
        <path :d="linePath" fill="none" :class="tone.stroke" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />
        <circle v-if="values.length === 1" :cx="cx(0)" :cy="y(values[0]!)" r="3" :class="tone.fill" />
        <circle v-if="hover !== null" :cx="cx(hover)" :cy="y(values[hover]!)" r="4" :class="tone.fill" class="stroke-white dark:stroke-[#04111f]" stroke-width="2" />
      </template>
    </svg>
    <div
      v-if="hover !== null"
      class="pointer-events-none absolute top-1 w-[130px] rounded-lg border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#0a1a2b] px-2.5 py-1.5 shadow-sm"
      :style="{ left: `${tipLeft}px` }"
    >
      <div class="text-xs text-neutral-500">
        {{ labels[hover] }}
      </div>
      <div class="text-sm font-semibold tabular-nums text-neutral-900 dark:text-white">
        {{ values[hover] }}<span v-if="unit" class="font-normal text-neutral-500"> {{ unit }}</span>
      </div>
    </div>
  </div>
</template>
