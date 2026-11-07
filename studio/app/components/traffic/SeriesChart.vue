<script setup lang="ts">
import type { TrafficPoint } from '~/types'

// Satu grafik = satu ukuran (tidak ada dua sumbu Y): "volume" = batang request (5xx ditumpuk, warna status),
// "latency" = garis p95. Hover menampilkan crosshair + tooltip per ember waktu.
const props = defineProps<{ points: TrafficPoint[], mode: 'volume' | 'latency', stepSec: number }>()

const wrap = ref<HTMLElement | null>(null)
const width = ref(600)
const height = 180
const pad = { l: 44, r: 8, t: 10, b: 22 }
const hover = ref<number | null>(null)
let ro: ResizeObserver | null = null

onMounted(() => {
  if (!wrap.value) return
  ro = new ResizeObserver(([e]) => { if (e) width.value = Math.max(240, e.contentRect.width) })
  ro.observe(wrap.value)
})
onBeforeUnmount(() => ro?.disconnect())

const value = (p: TrafficPoint) => props.mode === 'volume' ? p.requests : p.p95
const maxY = computed(() => {
  const m = Math.max(0, ...props.points.map(value))
  if (m <= 0) return props.mode === 'volume' ? 10 : 100
  const mag = 10 ** Math.floor(Math.log10(m))
  return Math.ceil(m / mag * 1.1) * mag
})
const innerW = computed(() => width.value - pad.l - pad.r)
const innerH = height - pad.t - pad.b
const slot = computed(() => innerW.value / Math.max(1, props.points.length))
const x = (i: number) => pad.l + i * slot.value
const y = (v: number) => pad.t + innerH - (v / maxY.value) * innerH
const barW = computed(() => Math.max(1, slot.value - 2)) // 2px celah permukaan antar batang

const ticks = computed(() => [0, 0.5, 1].map(f => ({ v: maxY.value * f, y: y(maxY.value * f) })))
const fmtTick = (v: number) => props.mode === 'volume' ? fmtNum(v) : fmtMs(v)

const timeLabel = (t: number) => {
  const d = new Date(t * 1000)
  return props.stepSec >= 3600 * 6 || (props.stepSec >= 3600 && props.points.length > 48)
    ? d.toLocaleDateString('id-ID', { day: 'numeric', month: 'short' }) + (props.stepSec < 86400 ? ' ' + d.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' }) : '')
    : d.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
}
const xTicks = computed(() => {
  const n = props.points.length
  if (!n) return []
  const every = Math.max(1, Math.ceil(n / Math.max(2, Math.floor(innerW.value / 90))))
  return props.points.map((p, i) => ({ i, p })).filter(({ i }) => i % every === 0)
})

const linePath = computed(() => {
  let d = ''
  let pen = false
  props.points.forEach((p, i) => {
    if (p.requests === 0) { pen = false; return } // tanpa request = tidak ada latensi, garis diputus
    const cx = x(i) + slot.value / 2
    d += `${pen ? 'L' : 'M'}${cx.toFixed(1)},${y(p.p95).toFixed(1)}`
    pen = true
  })
  return d
})

const isolated = computed(() => props.points
  .map((p, i) => (p.requests > 0 && !(props.points[i - 1]?.requests) && !(props.points[i + 1]?.requests) ? i : -1))
  .filter(i => i >= 0))

const barPath = (x0: number, y0: number, w: number, h: number, roundTop: boolean) => {
  if (h <= 0) return ''
  const r = roundTop ? Math.min(4, w / 2, h) : 0
  return `M${x0},${y0 + h}V${y0 + r}${r ? `Q${x0},${y0} ${x0 + r},${y0}` : ''}H${x0 + w - r}${r ? `Q${x0 + w},${y0} ${x0 + w},${y0 + r}` : ''}V${y0 + h}Z`
}

const onMove = (e: MouseEvent) => {
  const rect = (e.currentTarget as SVGElement).getBoundingClientRect()
  const i = Math.floor((e.clientX - rect.left - pad.l) / slot.value)
  hover.value = i >= 0 && i < props.points.length ? i : null
}
const hp = computed(() => hover.value !== null ? props.points[hover.value] : null)
const tipLeft = computed(() => hover.value === null ? 0 : Math.min(width.value - 170, Math.max(0, x(hover.value) + slot.value / 2 - 85)))
</script>

<template>
  <div ref="wrap" class="relative w-full select-none">
    <svg :width="width" :height="height" class="block" role="img" :aria-label="mode === 'volume' ? 'Jumlah request per waktu' : 'Latensi p95 per waktu'" @mousemove="onMove" @mouseleave="hover = null">
      <g>
        <line v-for="t in ticks" :key="t.v" :x1="pad.l" :x2="width - pad.r" :y1="t.y" :y2="t.y" class="stroke-neutral-200 dark:stroke-neutral-800" stroke-width="1" />
        <text v-for="t in ticks" :key="'l' + t.v" :x="pad.l - 6" :y="t.y + 3" text-anchor="end" class="fill-neutral-500 text-[10px]">{{ fmtTick(t.v) }}</text>
        <text v-for="t in xTicks" :key="'x' + t.i" :x="x(t.i) + slot / 2" :y="height - 6" text-anchor="middle" class="fill-neutral-500 text-[10px]">{{ timeLabel(t.p.t) }}</text>
      </g>
      <line v-if="hover !== null" :x1="x(hover) + slot / 2" :x2="x(hover) + slot / 2" :y1="pad.t" :y2="pad.t + innerH" class="stroke-neutral-400 dark:stroke-neutral-500" stroke-width="1" stroke-dasharray="3 3" />
      <template v-if="mode === 'volume'">
        <g v-for="(p, i) in points" :key="p.t">
          <!-- batang OK (non-5xx) di bawah, 5xx di atasnya dengan celah 2px -->
          <path
            :d="barPath(x(i) + 1, y(p.requests - p.err_5xx), barW, (p.requests - p.err_5xx) / maxY * innerH, p.err_5xx === 0)"
            class="fill-primary-500 dark:fill-primary-400"
            :opacity="hover === null || hover === i ? 1 : 0.55"
          />
          <path
            v-if="p.err_5xx > 0"
            :d="barPath(x(i) + 1, y(p.requests), barW, Math.max(0, p.err_5xx / maxY * innerH - 2), true)"
            class="fill-rose-500 dark:fill-rose-400"
            :opacity="hover === null || hover === i ? 1 : 0.55"
          />
        </g>
      </template>
      <template v-else>
        <path :d="linePath" fill="none" class="stroke-amber-500 dark:stroke-amber-400" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />
        <!-- titik tanpa tetangga tidak membentuk garis: tampilkan sebagai penanda -->
        <circle
          v-for="i in isolated"
          :key="'iso' + i"
          :cx="x(i) + slot / 2"
          :cy="y(points[i]!.p95)"
          r="3"
          class="fill-amber-500 dark:fill-amber-400"
        />
        <circle
          v-if="hp && hp.requests > 0 && hover !== null"
          :cx="x(hover) + slot / 2"
          :cy="y(hp.p95)"
          r="4"
          class="fill-amber-500 dark:fill-amber-400 stroke-white dark:stroke-neutral-900"
          stroke-width="2"
        />
      </template>
    </svg>
    <div
      v-if="hp"
      class="pointer-events-none absolute top-1 z-10 w-[170px] rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white/95 dark:bg-neutral-900/95 shadow-sm px-2.5 py-1.5 text-[11px] text-neutral-700 dark:text-neutral-200"
      :style="{ left: tipLeft + 'px' }"
    >
      <div class="font-semibold text-neutral-900 dark:text-white">
        {{ timeLabel(hp.t) }}
      </div>
      <div>{{ fmtNum(hp.requests) }} request</div>
      <div v-if="hp.err_5xx || hp.err_4xx" class="text-neutral-500">
        5xx {{ fmtNum(hp.err_5xx) }} · 4xx {{ fmtNum(hp.err_4xx) }}
      </div>
      <div v-if="hp.requests">
        p95 {{ fmtMs(hp.p95) }} · rata² {{ fmtMs(hp.avg_ms) }}
      </div>
    </div>
  </div>
</template>
