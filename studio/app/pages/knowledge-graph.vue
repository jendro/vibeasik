<script setup lang="ts">
import type { KnowledgeGraph, KnowledgeGraphNode, KnowledgeKind, KnowledgePromptPreview } from '~/types'
import type { GraphViewLink, GraphViewNode } from '~/components/knowledge/KnowledgeGraphCanvas.vue'
import { KIND_META, SOURCE_LABEL } from '~/utils/knowledge'

/**
 * Peta Jaring Pusat Pengetahuan: seluruh butir aktif sebagai node, tautan kemiripan (relink kurator) sebagai garis.
 * Ukuran node = pengaruh: PageRank berbobot pada jaring tautan, dengan lompatan acak yang dibias kekuatan butir
 * (bobot awal, jumlah dipakai, bukti ✓/✗) — node yang banyak bertaut ke butir penting & sering terbukti membesar.
 */
definePageMeta({ fluid: true })

const api = useApi()
const colorMode = useColorMode()
const dark = computed(() => colorMode.value === 'dark')

const graph = ref<KnowledgeGraph | null>(null)
const loading = ref(false)
const error = ref('')

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    graph.value = await api.getKnowledgeGraph()
  } catch (err: any) {
    error.value = err?.data?.error || err?.message || 'Gagal memuat jaring pengetahuan'
  } finally {
    loading.value = false
  }
}
// Tautan dari Pelajaran Desain: ?node=<id> membuka detail & menyorot node setelah jaring dimuat.
const route = useRoute()
onMounted(async () => {
  await load()
  const id = Number(route.query.node)
  if (id) nextTick(() => selectNode(id))
})

// Filter tampilan.
const includePersonal = ref(false)
const hideIsolated = ref(true)
const domain = ref('all')
const kind = ref('all')
const colorBy = ref<'kind' | 'domain' | 'cluster'>('domain')
const search = ref('')

const PALETTE = ['#6366f1', '#10b981', '#f59e0b', '#ec4899', '#0ea5e9', '#8b5cf6', '#ef4444', '#14b8a6', '#f97316', '#84cc16', '#06b6d4', '#d946ef']
const OTHER = '#94a3b8'
const KIND_COLOR: Record<KnowledgeKind, string> = { rule: '#6366f1', procedure: '#0ea5e9', lesson: '#f59e0b', concept: '#a855f7' }

// Simulasi: butir hasil temu-kembali selalu tampil walau tersaring filter. litIds = yang sudah menyala.
const simForce = ref<Set<number>>(new Set())
const litIds = ref<number[]>([])
const ghostIds = ref<number[]>([])
const simPreview = ref<KnowledgePromptPreview | null>(null)
const simRef = ref<{ text: string } | null>(null)
// Tetangga bertaut pustaka bersama (temu-kembali tanpa pegawai tidak memakai keahlian pribadi).
const neighborsOf = (ids: number[]) => {
  const seed = new Set(ids)
  const out = new Set<number>()
  for (const l of graph.value?.links || []) {
    const other = seed.has(l.source) ? l.target : seed.has(l.target) ? l.source : 0
    if (other && !seed.has(other) && !nodeById.value.get(other)?.personal) out.add(other)
  }
  return [...out]
}

const filtered = computed(() => {
  const g = graph.value
  if (!g) return { nodes: [] as KnowledgeGraphNode[], links: [] as KnowledgeGraph['links'] }
  const force = simForce.value
  let nodes = g.nodes.filter(n => force.has(n.id) || ((includePersonal.value || !n.personal) && (domain.value === 'all' || n.domain === domain.value) && (kind.value === 'all' || n.kind === kind.value)))
  const ids = new Set(nodes.map(n => n.id))
  const links = g.links.filter(l => ids.has(l.source) && ids.has(l.target))
  if (hideIsolated.value) {
    const linked = new Set<number>()
    for (const l of links) {
      linked.add(l.source)
      linked.add(l.target)
    }
    nodes = nodes.filter(n => linked.has(n.id) || force.has(n.id))
  }
  return { nodes, links }
})

// Pengaruh: PageRank berbobot (redaman 0.85) dengan vektor lompatan = kekuatan intrinsik butir.
const influence = computed(() => {
  const { nodes, links } = filtered.value
  const idx = new Map(nodes.map((n, i) => [n.id, i]))
  const N = nodes.length
  const rank = new Map<number, number>()
  const degree = new Map<number, number>()
  if (!N) return { rank, degree }
  const intrinsic = nodes.map(n => 1 + n.weight / 25 + Math.log1p(n.uses) * 1.5 + Math.max(0, n.helpful - n.harmful) * 0.8)
  const tot = intrinsic.reduce((a, b) => a + b, 0)
  const tele = intrinsic.map(v => v / tot)
  const out = new Float64Array(N)
  const edges: [number, number, number][] = []
  for (const l of links) {
    const a = idx.get(l.source)!
    const b = idx.get(l.target)!
    edges.push([a, b, l.strength])
    out[a] = (out[a] ?? 0) + l.strength
    out[b] = (out[b] ?? 0) + l.strength
    degree.set(l.source, (degree.get(l.source) || 0) + 1)
    degree.set(l.target, (degree.get(l.target) || 0) + 1)
  }
  let pr = Float64Array.from(tele)
  for (let it = 0; it < 40; it++) {
    const next = new Float64Array(N)
    let dangling = 0
    for (let i = 0; i < N; i++) if (!out[i]) dangling += pr[i]!
    for (const [a, b, w] of edges) {
      next[b] = next[b]! + 0.85 * pr[a]! * w / out[a]!
      next[a] = next[a]! + 0.85 * pr[b]! * w / out[b]!
    }
    for (let i = 0; i < N; i++) next[i] = next[i]! + (0.15 + 0.85 * dangling) * tele[i]!
    pr = next
  }
  const max = Math.max(...pr)
  nodes.forEach((n, i) => rank.set(n.id, pr[i]! / max))
  return { rank, degree }
})

const domainColor = computed(() => {
  const counts = new Map<string, number>()
  for (const n of filtered.value.nodes) counts.set(n.domain || 'umum', (counts.get(n.domain || 'umum') || 0) + 1)
  const top = [...counts.entries()].sort((a, b) => b[1] - a[1]).slice(0, PALETTE.length)
  return new Map(top.map(([d], i) => [d, PALETTE[i]!]))
})

const clusterLabel = computed(() => new Map((graph.value?.clusters || []).map(c => [c.id, c.label])))
const clusterColor = computed(() => {
  const counts = new Map<number, number>()
  for (const n of filtered.value.nodes) if (n.cluster_id) counts.set(n.cluster_id, (counts.get(n.cluster_id) || 0) + 1)
  const top = [...counts.entries()].sort((a, b) => b[1] - a[1]).slice(0, PALETTE.length)
  return new Map(top.map(([c], i) => [c, PALETTE[i]!]))
})

const colorOf = (n: KnowledgeGraphNode) => {
  if (colorBy.value === 'kind') return KIND_COLOR[n.kind] || OTHER
  if (colorBy.value === 'cluster') return clusterColor.value.get(n.cluster_id) || OTHER
  return domainColor.value.get(n.domain || 'umum') || OTHER
}

const viewNodes = computed<GraphViewNode[]>(() => filtered.value.nodes.map((n) => {
  const rk = influence.value.rank.get(n.id) || 0
  return { id: n.id, label: n.title, color: colorOf(n), rank: rk, r: 2 + rk ** 1.2 * 26 }
}))
const viewLinks = computed<GraphViewLink[]>(() => filtered.value.links)

const legend = computed(() => {
  if (colorBy.value === 'kind') return (Object.keys(KIND_META) as KnowledgeKind[]).map(k => ({ label: KIND_META[k].label, color: KIND_COLOR[k] }))
  if (colorBy.value === 'cluster') return [...clusterColor.value.entries()].map(([c, color]) => ({ label: clusterLabel.value.get(c) || `Klaster #${c}`, color }))
  return [...domainColor.value.entries()].map(([d, color]) => ({ label: d, color }))
})

const nodeById = computed(() => new Map((graph.value?.nodes || []).map(n => [n.id, n])))
const topNodes = computed(() => [...filtered.value.nodes]
  .sort((a, b) => (influence.value.rank.get(b.id) || 0) - (influence.value.rank.get(a.id) || 0))
  .slice(0, 12))

const domainItems = computed(() => {
  const counts = new Map<string, number>()
  for (const n of graph.value?.nodes || []) if (includePersonal.value || !n.personal) counts.set(n.domain || '', (counts.get(n.domain || '') || 0) + 1)
  return [{ label: 'Semua domain', value: 'all' }, ...[...counts.entries()].filter(([d]) => d).sort((a, b) => b[1] - a[1]).map(([d, c]) => ({ label: `${d} (${c})`, value: d }))]
})
const kindItems = [{ label: 'Semua jenis', value: 'all' }, ...(Object.keys(KIND_META) as KnowledgeKind[]).map(k => ({ label: KIND_META[k].label, value: k }))]
const colorItems = [{ label: 'Warna: domain', value: 'domain' }, { label: 'Warna: jenis', value: 'kind' }, { label: 'Warna: klaster konsep', value: 'cluster' }]

const canvasRef = ref<{ focusNode: (id: number) => void, fit: (ids?: number[]) => void, reheat: () => void } | null>(null)
const aside = ref<'sim' | 'top'>('sim')
const showWhy = ref(true)

const simStart = async (ids: number[]) => {
  selectedId.value = null
  // Butir yang lebih baru dari peta yang dimuat → muat ulang peta sekali agar ikut tampil.
  if (ids.some(id => !nodeById.value.has(id))) await load()
  simForce.value = new Set(ids)
}
// Tunggu tata letak menyesuaikan butir yang baru ikut tampil, lalu bingkai butir hasil simulasi.
const simFit = (ids: number[]) => setTimeout(() => canvasRef.value?.fit(ids), 400)
const simReset = () => {
  litIds.value = []
  ghostIds.value = []
  simForce.value = new Set()
  canvasRef.value?.fit()
}
const selectedId = ref<number | null>(null)
const hoverId = ref<number | null>(null)
const listHoverId = ref<number | null>(null)
const detailOpen = computed({ get: () => !!selectedId.value, set: (v) => { if (!v) selectedId.value = null } })
const selectNode = (id: number | null) => {
  selectedId.value = id
  if (id) canvasRef.value?.focusNode(id)
}

const searchItems = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (q.length < 2) return []
  return filtered.value.nodes.filter(n => n.title.toLowerCase().includes(q)).slice(0, 8)
})

// Siklus hidup pengetahuan, ditampilkan di kartu "gunanya apa".
const FLOW = [
  { title: 'Pekerjaan masuk', icon: 'i-lucide-inbox', color: '#6366f1', desc: 'Todo, audit kode, rapat PM, atau penilaian QA meminta pengetahuan.' },
  { title: 'Temu-kembali', icon: 'i-lucide-search', color: '#0ea5e9', desc: 'Butir yang cocok dicari, lalu tautannya disusuri ke butir tetangga.' },
  { title: 'Dibawa ke AI', icon: 'i-lucide-bot', color: '#f59e0b', desc: 'Hanya butir paling relevan & terbukti masuk prompt pekerja AI.' },
  { title: 'Hasil dinilai', icon: 'i-lucide-badge-check', color: '#10b981', desc: 'Nilai QA & vonis audit memberi bukti ✓ membantu / ✗ keliru.' },
  { title: 'Belajar', icon: 'i-lucide-graduation-cap', color: '#ec4899', desc: 'Butir ✓ membesar, ✗ ditinjau; topik kosong jadi celah untuk dipelajari.' }
]
const pct = (v: number) => `${Math.round(v * 100)}%`
const info = computed(() => {
  const id = hoverId.value ?? listHoverId.value ?? selectedId.value
  return id ? nodeById.value.get(id) || null : null
})
</script>

<template>
  <div class="w-full px-4 sm:px-6 lg:px-8 pt-6 pb-8 space-y-4">
    <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4 p-5 sm:p-6 rounded-2xl bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 shadow-xs">
      <div class="flex items-center gap-3.5 min-w-0">
        <div class="size-11 shrink-0 rounded-xl bg-linear-to-br from-indigo-500 to-emerald-500 text-white flex items-center justify-center shadow-xs">
          <UIcon name="i-lucide-waypoints" class="size-5.5" />
        </div>
        <div class="min-w-0">
          <h1 class="text-xl font-bold tracking-tight text-neutral-900 dark:text-white">
            Jaring Pengetahuan
          </h1>
          <p class="text-xs text-neutral-500 mt-0.5">
            Peta tautan antar butir Pusat Pengetahuan. Node membesar sesuai pengaruhnya: banyak bertaut ke butir penting, sering dipakai, dan terbukti membantu.
          </p>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2 shrink-0">
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-book-marked" to="/knowledge-docs">
          Dokumentasi
        </UButton>
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-library" to="/knowledge">
          Daftar butir
        </UButton>
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="loading" @click="load">
          Muat ulang
        </UButton>
      </div>
    </div>

    <p v-if="error" class="text-sm text-rose-600 dark:text-rose-400">
      {{ error }}
    </p>

    <div class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 p-4 sm:p-5">
      <button type="button" class="w-full flex items-center justify-between gap-2 text-left" @click="showWhy = !showWhy">
        <span class="text-sm font-semibold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-circle-help" class="size-4 text-primary-500" /> Jaring ini gunanya apa?
        </span>
        <UIcon :name="showWhy ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" class="size-4 text-neutral-400" />
      </button>
      <div v-if="showWhy" class="mt-3 space-y-4">
        <p class="text-sm text-neutral-600 dark:text-neutral-400 leading-relaxed max-w-4xl">
          <b class="text-neutral-900 dark:text-white">Masalahnya:</b> tanpa pusat ini, tiap pekerja AI mulai dari nol — lupa standar keamanan, mengulang bug yang sama, atau prompt-nya dijejali ratusan guideline sekaligus.
          <b class="text-neutral-900 dark:text-white">Solusinya:</b> setiap butir (patokan, prosedur, pelajaran, konsep) disimpan sebagai node dan ditautkan ke butir yang mirip. Saat ada pekerjaan, hanya ±10 butir paling relevan & terbukti yang dibawa, lalu hasil kerjanya menilai balik butir itu.
        </p>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-2">
          <div v-for="(f, i) in FLOW" :key="f.title" class="relative p-3 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-100 dark:border-neutral-800">
            <div class="flex items-center gap-2">
              <span class="size-6 rounded-lg flex items-center justify-center text-white shrink-0" :style="{ background: f.color }">
                <UIcon :name="f.icon" class="size-3.5" />
              </span>
              <span class="text-xs font-semibold text-neutral-900 dark:text-white">{{ i + 1 }}. {{ f.title }}</span>
            </div>
            <p class="mt-1.5 text-xs text-neutral-500 leading-relaxed">
              {{ f.desc }}
            </p>
          </div>
        </div>
        <p class="text-xs text-neutral-500">
          Coba langsung: pilih contoh pekerjaan di panel <b>Simulasi</b> sebelah kanan peta, lalu lihat butir mana yang menyala.
        </p>
      </div>
    </div>

    <div class="flex flex-col lg:flex-row gap-2">
      <div class="relative flex-1">
        <UInput v-model="search" icon="i-lucide-search" placeholder="Cari butir lalu lompat ke node…" class="w-full" />
        <div v-if="searchItems.length" class="absolute z-20 mt-1 w-full rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 shadow-lg overflow-hidden">
          <button
            v-for="n in searchItems"
            :key="n.id"
            type="button"
            class="w-full text-left px-3 py-2 text-sm hover:bg-neutral-100 dark:hover:bg-neutral-800 flex items-center gap-2"
            @click="() => { selectNode(n.id); search = '' }"
          >
            <span class="size-2 rounded-full shrink-0" :style="{ background: colorOf(n) }" />
            <span class="truncate">{{ n.title }}</span>
            <span class="ml-auto text-xs text-neutral-500">K{{ n.id }}</span>
          </button>
        </div>
      </div>
      <USelect v-model="domain" :items="domainItems" class="w-full lg:w-48" />
      <USelect v-model="kind" :items="kindItems" class="w-full lg:w-40" />
      <USelect v-model="colorBy" :items="colorItems" class="w-full lg:w-52" />
    </div>
    <div class="flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-neutral-600 dark:text-neutral-400">
      <USwitch v-model="includePersonal" label="Sertakan keahlian pribadi pegawai" size="sm" />
      <USwitch v-model="hideIsolated" label="Sembunyikan node tanpa tautan" size="sm" />
      <span class="ml-auto">{{ filtered.nodes.length }} node · {{ filtered.links.length }} tautan</span>
    </div>

    <div class="grid grid-cols-1 xl:grid-cols-[1fr_340px] gap-4">
      <div class="relative h-[70vh] min-h-[480px] rounded-2xl border border-neutral-200 dark:border-neutral-800 overflow-hidden bg-[radial-gradient(ellipse_at_center,var(--color-white)_0%,var(--color-neutral-100)_100%)] dark:bg-[radial-gradient(ellipse_at_center,#172033_0%,#0a0a0a_100%)]">
        <ClientOnly>
          <KnowledgeGraphCanvas
            v-if="viewNodes.length"
            ref="canvasRef"
            :nodes="viewNodes"
            :links="viewLinks"
            :selected-id="selectedId"
            :highlight-id="listHoverId"
            :lit-ids="litIds"
            :ghost-ids="ghostIds"
            :dark="dark"
            @select="selectNode"
            @hover="hoverId = $event"
          />
        </ClientOnly>
        <div v-if="loading && !graph" class="absolute inset-0 flex items-center justify-center text-sm text-neutral-500">
          <UIcon name="i-lucide-loader-circle" class="size-5 animate-spin mr-2" /> Merajut jaring…
        </div>
        <p v-else-if="graph && !viewNodes.length" class="absolute inset-0 flex items-center justify-center text-sm text-neutral-500 px-6 text-center">
          Tidak ada node bertaut untuk filter ini. Tautan dihitung kurator (tiap 15 menit, atau "Kurasi sekarang" di Pusat Pengetahuan).
        </p>

        <div class="absolute top-3 right-3 flex flex-col gap-1.5">
          <UButton size="xs" color="neutral" variant="subtle" icon="i-lucide-scan" title="Bingkai seluruh jaring" @click="canvasRef?.fit()" />
          <UButton size="xs" color="neutral" variant="subtle" icon="i-lucide-wind" title="Guncang ulang tata letak" @click="canvasRef?.reheat()" />
        </div>

        <div v-if="legend.length" class="absolute bottom-3 left-3 max-w-[70%] flex flex-wrap gap-x-3 gap-y-1 px-3 py-2 rounded-lg bg-white/80 dark:bg-neutral-900/80 backdrop-blur text-xs text-neutral-700 dark:text-neutral-300">
          <span v-for="l in legend" :key="l.label" class="flex items-center gap-1.5">
            <span class="size-2.5 rounded-full" :style="{ background: l.color }" />{{ l.label }}
          </span>
        </div>

        <div v-if="info" class="absolute top-3 left-3 max-w-sm px-3 py-2.5 rounded-lg bg-white/90 dark:bg-neutral-900/90 backdrop-blur border border-neutral-200 dark:border-neutral-800 text-xs space-y-1">
          <p class="font-semibold text-sm text-neutral-900 dark:text-white leading-snug">
            {{ info.title }}
          </p>
          <p class="text-neutral-500">
            {{ KIND_META[info.kind]?.label }} · {{ info.domain || 'umum' }} · {{ SOURCE_LABEL[info.source] || info.source }}
          </p>
          <p class="text-neutral-600 dark:text-neutral-400">
            Pengaruh {{ pct(influence.rank.get(info.id) || 0) }} · {{ influence.degree.get(info.id) || 0 }} tautan · dipakai {{ info.uses }}× · ✓{{ info.helpful }} ✗{{ info.harmful }}
          </p>
        </div>
      </div>

      <aside class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 p-4 space-y-3 h-fit">
        <UTabs
          v-model="aside"
          :items="[{ label: 'Simulasi', value: 'sim', icon: 'i-lucide-play' }, { label: 'Berpengaruh', value: 'top', icon: 'i-lucide-trophy' }]"
          size="xs"
          :content="false"
          class="w-full"
        />
        <KnowledgeSimulation
          v-show="aside === 'sim'"
          ref="simRef"
          :total="graph?.nodes.length || 0"
          :neighbors-of="neighborsOf"
          @start="simStart"
          @lit="litIds = $event"
          @ghost="ghostIds = $event"
          @preview="simPreview = $event"
          @fit="simFit"
          @open="(id) => { selectedId = id }"
          @reset="simReset"
        />
        <template v-if="aside === 'top'">
          <button
            v-for="(n, i) in topNodes"
            :key="n.id"
            type="button"
            class="w-full text-left flex items-start gap-2.5 p-2 -mx-2 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 transition"
            :class="{ 'bg-neutral-100 dark:bg-neutral-800': selectedId === n.id }"
            @click="selectNode(n.id)"
            @mouseenter="listHoverId = n.id"
            @mouseleave="listHoverId = null"
          >
            <span class="text-xs font-mono text-neutral-400 w-4 shrink-0 pt-0.5">{{ i + 1 }}</span>
            <span class="size-2.5 rounded-full mt-1.5 shrink-0" :style="{ background: colorOf(n) }" />
            <span class="min-w-0 flex-1">
              <span class="block text-sm font-medium text-neutral-900 dark:text-white leading-snug line-clamp-2">{{ n.title }}</span>
              <span class="block mt-1 h-1 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
                <span class="block h-full rounded-full" :style="{ width: pct(influence.rank.get(n.id) || 0), background: colorOf(n) }" />
              </span>
              <span class="block mt-1 text-xs text-neutral-500">{{ influence.degree.get(n.id) || 0 }} tautan · {{ n.domain || 'umum' }}</span>
            </span>
          </button>
        </template>
        <p class="text-xs text-neutral-500 leading-relaxed pt-1 border-t border-neutral-100 dark:border-neutral-800">
          Arahkan kursor untuk menyorot tetangga, klik untuk detail, seret node untuk memindah, gulir untuk zoom.
        </p>
      </aside>
    </div>

    <KnowledgeSimulationOutput v-if="simPreview" :preview="simPreview" :task="simRef?.text || ''" @open="(id) => { selectedId = id }" />

    <USlideover v-model:open="detailOpen" :title="selectedId ? `Node K${selectedId}` : 'Node'" :overlay="false" :modal="false" :ui="{ content: 'max-w-md' }">
      <template #body>
        <KnowledgeNodeDetail v-if="selectedId" :id="selectedId" @open="selectNode" />
      </template>
    </USlideover>
  </div>
</template>
