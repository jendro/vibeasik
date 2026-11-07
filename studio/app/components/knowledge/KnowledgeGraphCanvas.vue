<script setup lang="ts">
import { forceCollide, forceLink, forceManyBody, forceSimulation, forceX, forceY } from 'd3-force'
import type { Simulation, SimulationLinkDatum, SimulationNodeDatum } from 'd3-force'

/**
 * Kanvas jaring Pusat Pengetahuan: simulasi gaya d3-force (Barnes-Hut) digambar di <canvas> agar ribuan node tetap
 * mulus. Node berpengaruh (PageRank besar) tampil lebih besar + bercahaya; arahkan kursor untuk menyorot tetangga,
 * klik untuk memilih, seret node untuk memindah, seret latar untuk menggeser, roda tetikus untuk zoom.
 */
export interface GraphViewNode {
  id: number
  label: string
  color: string
  r: number
  rank: number
}
export interface GraphViewLink {
  source: number
  target: number
  strength: number
}

type SimNode = GraphViewNode & SimulationNodeDatum
type SimLink = SimulationLinkDatum<SimNode> & { strength: number }

// litIds: node yang "menyala" saat simulasi temu-kembali (urutan = urutan ditemukan); ghostIds: tetangga bertaut
// yang ikut dipertimbangkan (menyala samar); sisanya diredupkan.
const props = defineProps<{ nodes: GraphViewNode[], links: GraphViewLink[], selectedId: number | null, highlightId?: number | null, litIds?: number[], ghostIds?: number[], dark: boolean }>()
const emit = defineEmits<{ (e: 'select', id: number | null): void, (e: 'hover', id: number | null): void }>()

const wrap = ref<HTMLDivElement | null>(null)
const canvas = ref<HTMLCanvasElement | null>(null)
const hoverId = ref<number | null>(null)
const tip = reactive({ show: false, x: 0, y: 0, text: '' })

let sim: Simulation<SimNode, SimLink> | null = null
let simNodes: SimNode[] = []
let simLinks: SimLink[] = []
let byId = new Map<number, SimNode>()
let adj = new Map<number, Set<number>>()
let labelIds = new Set<number>()
const view = { k: 1, x: 0, y: 0 }
let width = 0
let height = 0
let dirty = true
let raf = 0
let resizeObs: ResizeObserver | null = null
let fitted = false

const build = () => {
  const prev = new Map(simNodes.map(n => [n.id, n]))
  simNodes = props.nodes.map((n) => {
    const p = prev.get(n.id)
    return { ...n, x: p?.x, y: p?.y, vx: p?.vx, vy: p?.vy }
  })
  byId = new Map(simNodes.map(n => [n.id, n]))
  simLinks = props.links.filter(l => byId.has(l.source) && byId.has(l.target)).map(l => ({ source: l.source, target: l.target, strength: l.strength })) as SimLink[]
  adj = new Map()
  for (const l of props.links) {
    if (!byId.has(l.source) || !byId.has(l.target)) continue
    if (!adj.has(l.source)) adj.set(l.source, new Set())
    if (!adj.has(l.target)) adj.set(l.target, new Set())
    adj.get(l.source)!.add(l.target)
    adj.get(l.target)!.add(l.source)
  }
  // Label tetap untuk node paling berpengaruh; sisanya muncul saat zoom cukup dekat.
  labelIds = new Set([...simNodes].sort((a, b) => b.rank - a.rank).slice(0, 12).map(n => n.id))

  sim?.stop()
  sim = forceSimulation<SimNode, SimLink>(simNodes)
    .force('link', forceLink<SimNode, SimLink>(simLinks).id(d => d.id).distance(l => 24 + (1 - l.strength) * 70 + ((l.source as SimNode).r + (l.target as SimNode).r)).strength(l => 0.15 + l.strength * 0.5))
    .force('charge', forceManyBody<SimNode>().strength(d => -18 - d.r * 5).distanceMax(420).theta(0.9))
    .force('collide', forceCollide<SimNode>(d => d.r + 1.5).iterations(1))
    .force('x', forceX<SimNode>(0).strength(0.035))
    .force('y', forceY<SimNode>(0).strength(0.035))
    .alpha(prev.size ? 0.6 : 1)
    .alphaDecay(0.025)
    .on('tick', () => { dirty = true })
    .on('end', () => {
      if (!fitted && !props.litIds?.length) fit()
      fitted = true
    })
}

const toWorld = (sx: number, sy: number) => ({ x: (sx - view.x) / view.k, y: (sy - view.y) / view.k })

const nodeAt = (sx: number, sy: number): SimNode | null => {
  const p = toWorld(sx, sy)
  let best: SimNode | null = null
  let bestD = Infinity
  for (const n of simNodes) {
    const dx = (n.x ?? 0) - p.x
    const dy = (n.y ?? 0) - p.y
    const d = dx * dx + dy * dy
    const hit = n.r + 4 / view.k
    if (d < hit * hit && d < bestD) {
      best = n
      bestD = d
    }
  }
  return best
}

const resize = () => {
  const el = wrap.value
  const c = canvas.value
  if (!el || !c) return
  const first = !width
  width = el.clientWidth
  height = el.clientHeight
  const dpr = window.devicePixelRatio || 1
  c.width = width * dpr
  c.height = height * dpr
  c.style.width = `${width}px`
  c.style.height = `${height}px`
  if (first) {
    view.x = width / 2
    view.y = height / 2
    view.k = 0.7
  }
  dirty = true
}

const hexA = (hex: string, a: number) => {
  const v = hex.replace('#', '')
  const r = Number.parseInt(v.slice(0, 2), 16)
  const g = Number.parseInt(v.slice(2, 4), 16)
  const b = Number.parseInt(v.slice(4, 6), 16)
  return `rgba(${r},${g},${b},${a})`
}

const draw = () => {
  raf = requestAnimationFrame(draw)
  const simSet = props.litIds?.length ? new Set(props.litIds) : null
  const ghostSet = new Set(simSet ? props.ghostIds || [] : [])
  // Mode simulasi berdenyut: gambar ulang tiap frame.
  if (!dirty && !simSet) return
  dirty = false
  const c = canvas.value
  if (!c) return
  const ctx = c.getContext('2d')
  if (!ctx) return
  const dpr = window.devicePixelRatio || 1
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, width, height)

  const focus = hoverId.value ?? props.highlightId ?? props.selectedId
  const near = focus != null ? adj.get(focus) || new Set<number>() : null
  const lit = (id: number) => near ? id === focus || near.has(id) : simSet ? simSet.has(id) || ghostSet.has(id) : true

  ctx.save()
  ctx.translate(view.x, view.y)
  ctx.scale(view.k, view.k)

  // Tautan: tipis & samar; yang menyentuh node fokus menyala dengan warna node fokus.
  const linkBase = props.dark ? '148,163,184' : '100,116,139'
  ctx.lineWidth = 0.6 / view.k
  ctx.strokeStyle = `rgba(${linkBase},${near || simSet ? 0.05 : 0.16})`
  ctx.beginPath()
  for (const l of simLinks) {
    const s = l.source as SimNode
    const t = l.target as SimNode
    if (near && (s.id === focus || t.id === focus)) continue
    ctx.moveTo(s.x!, s.y!)
    ctx.lineTo(t.x!, t.y!)
  }
  ctx.stroke()
  if (near && focus != null) {
    const f = byId.get(focus)
    if (f) {
      ctx.strokeStyle = hexA(f.color, 0.75)
      for (const l of simLinks) {
        const s = l.source as SimNode
        const t = l.target as SimNode
        if (s.id !== focus && t.id !== focus) continue
        ctx.lineWidth = (0.8 + l.strength * 2.2) / view.k
        ctx.beginPath()
        ctx.moveTo(s.x!, s.y!)
        ctx.lineTo(t.x!, t.y!)
        ctx.stroke()
      }
    }
  }

  // Simulasi: tautan di antara node yang menyala = jalur penyusuran pengetahuan.
  if (simSet && !near) {
    // Tautan ke tetangga yang dipertimbangkan: putus-putus.
    ctx.setLineDash([4 / view.k, 4 / view.k])
    ctx.lineWidth = 1 / view.k
    for (const l of simLinks) {
      const s = l.source as SimNode
      const t = l.target as SimNode
      const from = simSet.has(s.id) ? s : simSet.has(t.id) ? t : null
      const to = from === s ? t : s
      if (!from || !ghostSet.has(to.id)) continue
      ctx.strokeStyle = hexA(from.color, 0.55)
      ctx.beginPath()
      ctx.moveTo(from.x!, from.y!)
      ctx.lineTo(to.x!, to.y!)
      ctx.stroke()
    }
    ctx.setLineDash([])
    for (const l of simLinks) {
      const s = l.source as SimNode
      const t = l.target as SimNode
      if (!simSet.has(s.id) || !simSet.has(t.id)) continue
      ctx.strokeStyle = hexA(s.color, 0.8)
      ctx.lineWidth = (1 + l.strength * 2.5) / view.k
      ctx.beginPath()
      ctx.moveTo(s.x!, s.y!)
      ctx.lineTo(t.x!, t.y!)
      ctx.stroke()
    }
  }

  // Node: kecil dulu, besar terakhir (di atas); node menyala selalu di atas yang redup. Node besar diberi cahaya.
  const order = [...simNodes].sort((a, b) => (lit(a.id) ? 1000 : 0) + a.r - (lit(b.id) ? 1000 : 0) - b.r)
  for (const n of order) {
    const on = lit(n.id)
    const a = !on ? 0.12 : ghostSet.has(n.id) && !near ? 0.45 : 1
    if (on && n.r > 9 && !(ghostSet.has(n.id) && !near)) {
      const g = ctx.createRadialGradient(n.x!, n.y!, n.r * 0.6, n.x!, n.y!, n.r * 2.4)
      g.addColorStop(0, hexA(n.color, props.dark ? 0.35 : 0.22))
      g.addColorStop(1, hexA(n.color, 0))
      ctx.fillStyle = g
      ctx.beginPath()
      ctx.arc(n.x!, n.y!, n.r * 2.4, 0, Math.PI * 2)
      ctx.fill()
    }
    ctx.fillStyle = hexA(n.color, a * (props.dark ? 0.92 : 0.88))
    ctx.beginPath()
    ctx.arc(n.x!, n.y!, n.r, 0, Math.PI * 2)
    ctx.fill()
    if (simSet?.has(n.id) && !near) {
      // Cincin berdenyut menandai butir yang terambil untuk pekerjaan ini.
      const ph = (performance.now() / 1400 + props.litIds!.indexOf(n.id) * 0.17) % 1
      ctx.lineWidth = 1.5 / view.k
      ctx.strokeStyle = hexA(n.color, 0.9 * (1 - ph))
      ctx.beginPath()
      ctx.arc(n.x!, n.y!, n.r + (3 + ph * 14) / view.k, 0, Math.PI * 2)
      ctx.stroke()
      ctx.beginPath()
      ctx.arc(n.x!, n.y!, n.r, 0, Math.PI * 2)
    }
    if (n.id === focus || n.id === props.selectedId) {
      ctx.lineWidth = 2 / view.k
      ctx.strokeStyle = props.dark ? '#ffffff' : '#0f172a'
      ctx.stroke()
    } else if (on && n.r > 5) {
      ctx.lineWidth = 1 / view.k
      ctx.strokeStyle = props.dark ? 'rgba(15,23,42,0.6)' : 'rgba(255,255,255,0.85)'
      ctx.stroke()
    }
  }

  // Label berukuran tetap di layar.
  const fs = 11 / view.k
  ctx.font = `500 ${fs}px ui-sans-serif, system-ui, sans-serif`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'top'
  ctx.lineJoin = 'round'
  // Label didahulukan untuk node terbesar; label yang bertabrakan dengan label sebelumnya dilewati.
  const placed: { x0: number, y0: number, x1: number, y1: number }[] = []
  for (let i = order.length - 1; i >= 0; i--) {
    const n = order[i]!
    const show = near
      ? (n.id === focus || near.has(n.id)) && (n.r * view.k > 4 || n.id === focus || near.size < 14)
      : simSet ? simSet.has(n.id) : labelIds.has(n.id) || n.r * view.k > 22
    if (!show) continue
    const text = n.label.length > 42 ? `${n.label.slice(0, 40)}…` : n.label
    const y = n.y! + n.r + 3 / view.k
    const half = ctx.measureText(text).width / 2
    const box = { x0: n.x! - half, y0: y, x1: n.x! + half, y1: y + fs * 1.2 }
    if (n.id !== focus && placed.some(b => box.x0 < b.x1 && box.x1 > b.x0 && box.y0 < b.y1 && box.y1 > b.y0)) continue
    placed.push(box)
    ctx.lineWidth = 3 / view.k
    ctx.strokeStyle = props.dark ? 'rgba(10,10,10,0.85)' : 'rgba(255,255,255,0.9)'
    ctx.strokeText(text, n.x!, y)
    ctx.fillStyle = props.dark ? '#e5e7eb' : '#1f2937'
    ctx.fillText(text, n.x!, y)
  }
  ctx.restore()
}

// Interaksi: geser latar, seret node, zoom roda tetikus, sorot saat hover.
let drag: { mode: 'pan' | 'node', node?: SimNode, sx: number, sy: number, vx: number, vy: number, moved: boolean } | null = null

const local = (e: MouseEvent) => {
  const r = canvas.value!.getBoundingClientRect()
  return { x: e.clientX - r.left, y: e.clientY - r.top }
}

const onDown = (e: PointerEvent) => {
  const p = local(e)
  const n = nodeAt(p.x, p.y)
  canvas.value?.setPointerCapture(e.pointerId)
  drag = { mode: n ? 'node' : 'pan', node: n || undefined, sx: p.x, sy: p.y, vx: view.x, vy: view.y, moved: false }
  if (n) {
    n.fx = n.x
    n.fy = n.y
    sim?.alphaTarget(0.25).restart()
  }
}

const onMove = (e: PointerEvent) => {
  const p = local(e)
  if (drag) {
    if (Math.abs(p.x - drag.sx) + Math.abs(p.y - drag.sy) > 3) drag.moved = true
    if (drag.mode === 'pan') {
      view.x = drag.vx + (p.x - drag.sx)
      view.y = drag.vy + (p.y - drag.sy)
    } else if (drag.node) {
      const w = toWorld(p.x, p.y)
      drag.node.fx = w.x
      drag.node.fy = w.y
    }
    dirty = true
    tip.show = false
    return
  }
  const n = nodeAt(p.x, p.y)
  const id = n?.id ?? null
  if (id !== hoverId.value) {
    hoverId.value = id
    emit('hover', id)
    dirty = true
  }
  if (n) Object.assign(tip, { show: true, x: p.x + 14, y: p.y + 14, text: n.label })
  else tip.show = false
  if (canvas.value) canvas.value.style.cursor = n ? 'pointer' : 'grab'
}

const onUp = () => {
  if (!drag) return
  if (drag.node) {
    drag.node.fx = null
    drag.node.fy = null
    sim?.alphaTarget(0)
  }
  if (!drag.moved) emit('select', drag.node?.id ?? null)
  drag = null
}

const onLeave = () => {
  hoverId.value = null
  tip.show = false
  emit('hover', null)
  dirty = true
}

const onWheel = (e: WheelEvent) => {
  e.preventDefault()
  const p = local(e)
  const k = Math.min(6, Math.max(0.08, view.k * Math.exp(-e.deltaY * 0.0015)))
  view.x = p.x - ((p.x - view.x) / view.k) * k
  view.y = p.y - ((p.y - view.y) / view.k) * k
  view.k = k
  dirty = true
}

// Animasi halus pindah kamera ke satu node / ke seluruh jaring.
const animateTo = (tx: number, ty: number, tk: number) => {
  const from = { ...view }
  const t0 = performance.now()
  const step = (now: number) => {
    const t = Math.min(1, (now - t0) / 550)
    const e = 1 - (1 - t) ** 3
    view.k = from.k + (tk - from.k) * e
    view.x = from.x + (tx - from.x) * e
    view.y = from.y + (ty - from.y) * e
    dirty = true
    if (t < 1) requestAnimationFrame(step)
  }
  requestAnimationFrame(step)
}

const focusNode = (id: number) => {
  const n = byId.get(id)
  if (!n) return
  const k = Math.max(view.k, 1.6)
  animateTo(width / 2 - n.x! * k, height / 2 - n.y! * k, k)
}

const fit = (ids?: number[]) => {
  const pick = ids?.length ? ids.map(id => byId.get(id)).filter((n): n is SimNode => !!n) : simNodes
  if (!pick.length) return
  let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity
  for (const n of pick) {
    x0 = Math.min(x0, n.x! - n.r)
    y0 = Math.min(y0, n.y! - n.r)
    x1 = Math.max(x1, n.x! + n.r)
    y1 = Math.max(y1, n.y! + n.r)
  }
  const pad = ids?.length ? 260 : 0
  const k = Math.min(ids?.length ? 2.2 : 3, 0.9 * Math.min(width / (x1 - x0 + pad || 1), height / (y1 - y0 + pad || 1)))
  animateTo(width / 2 - ((x0 + x1) / 2) * k, height / 2 - ((y0 + y1) / 2) * k, k)
}

const reheat = () => sim?.alpha(0.8).restart()

defineExpose({ focusNode, fit, reheat })

watch(() => [props.nodes, props.links], build)
watch(() => [props.selectedId, props.highlightId, props.litIds, props.ghostIds, props.dark], () => { dirty = true })

onMounted(() => {
  resize()
  build()
  resizeObs = new ResizeObserver(resize)
  if (wrap.value) resizeObs.observe(wrap.value)
  canvas.value?.addEventListener('wheel', onWheel, { passive: false })
  raf = requestAnimationFrame(draw)
  // Bingkai seluruh jaring saat tata letak awal mulai terbentuk; sekali lagi saat simulasi tenang (on end).
  setTimeout(fit, 1200)
})

onBeforeUnmount(() => {
  sim?.stop()
  cancelAnimationFrame(raf)
  resizeObs?.disconnect()
  canvas.value?.removeEventListener('wheel', onWheel)
})
</script>

<template>
  <div ref="wrap" class="relative w-full h-full overflow-hidden">
    <canvas
      ref="canvas"
      class="block touch-none cursor-grab"
      @pointerdown="onDown"
      @pointermove="onMove"
      @pointerup="onUp"
      @pointercancel="onUp"
      @pointerleave="onLeave"
    />
    <div
      v-if="tip.show"
      class="pointer-events-none absolute z-10 max-w-xs px-2.5 py-1.5 rounded-md text-xs font-medium bg-neutral-900/90 text-white dark:bg-white/95 dark:text-neutral-900 shadow-lg"
      :style="{ left: `${tip.x}px`, top: `${tip.y}px` }"
    >
      {{ tip.text }}
    </div>
  </div>
</template>
