<script setup lang="ts">
import type { TrafficMeta, TrafficRecentEvent, TrafficReport, TrafficSource, TrafficWindow } from '~/types'

definePageMeta({
  fluid: true
})

// Traffic Analyzer (Aturan 48): BE live mengirim statistik request lewat SDK → vibeasik live → engine ini.
// Halaman ini merangkum rute, asal traffic, kecepatan & error, lalu memberi saran rate limit, cache,
// load balancer dan pemecahan service berdasarkan data nyata.
const route = useRoute()
const toast = useToast()
const { activeProjectId, fetchProjects, selectActiveProject } = useActiveProject()
const { getMeta, listSources, updateSource, deleteSource, syncSource, getReport, getRecent } = useTraffic()

const meta = ref<TrafficMeta | null>(null)
const sources = ref<TrafficSource[]>([])
const sourceId = ref<number | null>(null)
const windowKey = ref<TrafficWindow>('1h')
const report = ref<TrafficReport | null>(null)
const recent = ref<TrafficRecentEvent[]>([])
const tab = ref<'advice' | 'routes' | 'origin' | 'live' | 'setup'>('advice')
const loading = ref(false)
const syncing = ref(false)
const showCreate = ref(false)
const confirmDelete = ref(false)
const autoRefresh = ref(true)
let timer: ReturnType<typeof setInterval> | null = null

const source = computed(() => sources.value.find(s => s.id === sourceId.value) || null)
const sourceItems = computed(() => sources.value.map(s => ({ label: `${s.name}${s.enabled ? '' : ' (jeda)'}`, value: s.id })))
const windowItems = computed(() => (meta.value?.windows || ['15m', '1h', '6h', '24h', '7d', '30d']).map(w => ({ label: trafficWindowLabel[w], value: w })))
const intervalItems = computed(() => (meta.value?.sync_intervals || [5, 60, 300]).map(v => ({ label: trafficIntervalLabel(v), value: v })))

// Penyegaran halaman mengikuti interval tarik sumber (realtime = 5 dtk), minimal 5 dtk, maksimal 60 dtk.
const refreshMs = computed(() => {
  if (!source.value) return 30000
  const sec = source.value.mode === 'direct' ? 5 : source.value.sync_interval_sec
  return Math.min(60, Math.max(5, sec)) * 1000
})

const loadSources = async () => {
  sources.value = await listSources(activeProjectId.value || undefined)
  if (!sources.value.some(s => s.id === sourceId.value)) sourceId.value = sources.value[0]?.id ?? null
}

const loadReport = async (quiet = false) => {
  if (!sourceId.value) {
    report.value = null
    recent.value = []
    return
  }
  if (!quiet) loading.value = true
  const id = sourceId.value
  try {
    const [rep, rc] = await Promise.all([getReport(id, windowKey.value), getRecent(id, 100)])
    if (id !== sourceId.value) return
    report.value = rep
    recent.value = rc.events
    const i = sources.value.findIndex(s => s.id === id)
    if (i >= 0) sources.value[i] = { ...sources.value[i], ...rc.source }
  } catch (err: any) {
    if (!quiet) toast.add({ title: 'Gagal memuat laporan traffic', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' })
  } finally {
    loading.value = false
  }
}

const restartTimer = () => {
  if (timer) clearInterval(timer)
  timer = null
  if (autoRefresh.value && import.meta.client) timer = setInterval(() => loadReport(true), refreshMs.value)
}

watch(sourceId, () => {
  confirmDelete.value = false
  loadReport()
})
watch(windowKey, () => loadReport())
watch([autoRefresh, refreshMs], restartTimer)
watch(activeProjectId, async () => {
  try {
    await loadSources()
  } catch (err: any) {
    toast.add({ title: 'Gagal memuat sumber traffic', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' })
  }
  if (!sourceId.value) loadReport()
})


const handleCreated = async (src: TrafficSource) => {
  await loadSources()
  sourceId.value = src.id
  tab.value = 'setup'
}

const syncNow = async () => {
  if (!source.value) return
  syncing.value = true
  try {
    const res = await syncSource(source.value.id)
    toast.add({ title: 'Sinkron selesai', description: `${res.pulled} event ditarik dari vibeasik live`, icon: 'i-lucide-refresh-cw', color: 'success' })
    await loadReport(true)
  } catch (err: any) {
    toast.add({ title: 'Sinkron gagal', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' })
  } finally {
    syncing.value = false
  }
}

const patchSource = async (body: { sync_interval_sec?: number, enabled?: boolean }) => {
  if (!source.value) return
  try {
    const updated = await updateSource(source.value.id, body)
    const i = sources.value.findIndex(s => s.id === updated.id)
    if (i >= 0) sources.value[i] = { ...sources.value[i], ...updated }
  } catch (err: any) {
    toast.add({ title: 'Gagal menyimpan', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' })
  }
}

const removeSource = async () => {
  if (!source.value) return
  try {
    const res = await deleteSource(source.value.id)
    toast.add({ title: 'Sumber traffic dihapus', description: res.warning || undefined, icon: 'i-lucide-trash-2', color: res.warning ? 'warning' : 'success' })
    confirmDelete.value = false
    await loadSources()
    loadReport()
  } catch (err: any) {
    toast.add({ title: 'Gagal menghapus', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' })
  }
}

const t = computed(() => report.value?.totals)
const alertCount = computed(() => (report.value?.recommendations || []).filter(r => r.severity !== 'info').length)
const connection = computed(() => {
  const s = source.value
  if (!s) return null
  if (s.last_error) return { cls: 'text-rose-600 dark:text-rose-400', icon: 'i-lucide-circle-x', text: s.last_error }
  if (!s.last_ingest_at) return { cls: 'text-neutral-500', icon: 'i-lucide-circle-dashed', text: 'Menunggu kiriman pertama dari aplikasi — pasang SDK di tab Pemasangan.' }
  const age = (Date.now() - new Date(s.last_ingest_at).getTime()) / 1000
  if (age > 900) return { cls: 'text-amber-600 dark:text-amber-400', icon: 'i-lucide-circle-alert', text: `Aplikasi terakhir mengirim ${fmtAgo(s.last_ingest_at)} — cek server live atau env SDK.` }
  return { cls: 'text-emerald-600 dark:text-emerald-400', icon: 'i-lucide-circle-check', text: `Tersambung · aplikasi terakhir mengirim ${fmtAgo(s.last_ingest_at)}` }
})

const tabs = computed(() => [
  { value: 'advice', label: 'Saran Optimasi', icon: 'i-lucide-lightbulb', badge: alertCount.value || undefined },
  { value: 'routes', label: 'Endpoint', icon: 'i-lucide-route' },
  { value: 'origin', label: 'Asal Traffic', icon: 'i-lucide-globe' },
  { value: 'live', label: 'Live', icon: 'i-lucide-radio' },
  { value: 'setup', label: 'Pemasangan', icon: 'i-lucide-plug' }
])

onMounted(async () => {
  await fetchProjects()
  const queryPId = route.query.project_id ? Number(route.query.project_id) : null
  if (queryPId) selectActiveProject(queryPId)
  try {
    meta.value = await getMeta()
    await loadSources()
  } catch (err: any) {
    toast.add({ title: 'Traffic Analyzer tidak tersedia', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' })
  }
  await loadReport()
  if (source.value && !source.value.last_ingest_at) tab.value = 'setup'
  restartTimer()
})
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="space-y-5 w-full max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 pt-6 pb-16">
    <AppPageHeader
      title="Traffic Analyzer"
      description="Statistik request nyata dari aplikasi live: endpoint yang diakses, asal, kecepatan, dan tingkat gagal — dasar saran rate limit, cache, load balancer, atau pemecahan service."
    >
      <template #actions>
        <UButton size="sm" color="neutral" icon="i-lucide-plus" @click="showCreate = true">
          Hubungkan aplikasi live
        </UButton>
      </template>
    </AppPageHeader>

    <!-- Belum ada sumber -->
    <section v-if="!sources.length" class="p-8 rounded-xl border border-dashed border-neutral-300 dark:border-neutral-700 text-center space-y-4">
      <UIcon name="i-lucide-radar" class="w-10 h-10 text-neutral-400 mx-auto" />
      <div class="space-y-1">
        <h2 class="text-sm font-bold text-neutral-900 dark:text-white">
          Belum ada aplikasi live yang terhubung
        </h2>
        <p class="text-xs text-neutral-500 max-w-xl mx-auto leading-relaxed">
          Pasang SDK kecil di BE produksi (Go, Node, Nuxt/Nitro, Python, Laravel). SDK mengirim statistik request per batch ke vibeasik live — realtime atau tiap beberapa menit — lalu engine ini menariknya dan menganalisis.
        </p>
      </div>
      <UButton color="primary" icon="i-lucide-plus" @click="showCreate = true">
        Hubungkan Aplikasi Live
      </UButton>
    </section>

    <template v-else>
      <!-- Sumber & kontrol -->
      <section class="p-4 bg-white dark:bg-[#04111f] border border-neutral-200 dark:border-white/[0.08] rounded-xl space-y-3">
        <div class="flex flex-col xl:flex-row xl:items-center justify-between gap-3">
          <div class="flex flex-wrap items-center gap-2">
            <USelect :model-value="sourceId ?? undefined" :items="sourceItems" size="sm" class="w-56" icon="i-lucide-server" @update:model-value="v => sourceId = Number(v)" />
            <UBadge v-if="source" size="sm" color="neutral" variant="soft" :icon="source.mode === 'cloud' ? 'i-lucide-cloud' : 'i-lucide-plug'">
              {{ source.mode === 'cloud' ? 'via vibeasik live' : 'langsung' }}
            </UBadge>
            <span v-if="source?.project_name" class="text-xs text-neutral-500">{{ source.project_name }}<template v-if="source.component_name"> · {{ source.component_name }}</template></span>
          </div>
          <div v-if="source" class="flex flex-wrap items-center gap-2">
            <USelect
              v-if="source.mode === 'cloud'"
              :model-value="source.sync_interval_sec"
              :items="intervalItems"
              size="sm"
              class="w-40"
              icon="i-lucide-timer"
              @update:model-value="v => patchSource({ sync_interval_sec: Number(v) })"
            />
            <UButton v-if="source.mode === 'cloud'" size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="syncing" @click="syncNow">
              Tarik sekarang
            </UButton>
            <UButton size="sm" color="neutral" variant="ghost" :icon="source.enabled ? 'i-lucide-pause' : 'i-lucide-play'" @click="patchSource({ enabled: !source.enabled })">
              {{ source.enabled ? 'Jeda' : 'Lanjutkan' }}
            </UButton>
            <UButton v-if="!confirmDelete" size="sm" color="error" variant="ghost" icon="i-lucide-trash-2" aria-label="Hapus sumber" @click="confirmDelete = true" />
            <template v-else>
              <span class="text-xs text-rose-600 dark:text-rose-400">Hapus sumber & seluruh statistiknya?</span>
              <UButton size="sm" color="error" @click="removeSource">
                Hapus
              </UButton>
              <UButton size="sm" color="neutral" variant="ghost" @click="confirmDelete = false">
                Batal
              </UButton>
            </template>
          </div>
        </div>
        <div v-if="source && connection" class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs">
          <span class="flex items-center gap-1.5 min-w-0" :class="connection.cls">
            <UIcon :name="connection.icon" class="w-4 h-4 shrink-0" /><span class="break-all">{{ connection.text }}</span>
          </span>
          <span class="text-neutral-500">Diolah {{ fmtNum(source.total_events) }} event</span>
          <span v-if="source.mode === 'cloud'" class="text-neutral-500">Ditarik {{ fmtAgo(source.last_sync_at) }}</span>
          <span v-if="source.pending > 0" class="text-amber-600 dark:text-amber-400">{{ fmtNum(source.pending) }} event masih menunggu di vibeasik live</span>
          <span v-if="source.dropped > 0" class="text-rose-600 dark:text-rose-400">{{ fmtNum(source.dropped) }} event dibuang (antrean live penuh — perpendek interval tarik)</span>
          <span v-if="source.last_instance" class="text-neutral-500">Instance terakhir: <span class="font-mono">{{ source.last_instance }}</span></span>
        </div>
      </section>

      <!-- Jendela waktu -->
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="flex flex-wrap gap-1">
          <UButton
            v-for="w in windowItems"
            :key="w.value"
            size="xs"
            :color="windowKey === w.value ? 'primary' : 'neutral'"
            :variant="windowKey === w.value ? 'soft' : 'ghost'"
            @click="windowKey = w.value"
          >
            {{ w.label }}
          </UButton>
        </div>
        <div class="flex items-center gap-2 text-xs text-neutral-500">
          <USwitch v-model="autoRefresh" size="sm" />
          <span>Segarkan otomatis tiap {{ refreshMs / 1000 }} dtk</span>
          <UIcon v-if="loading" name="i-lucide-loader-2" class="w-4 h-4 animate-spin" />
        </div>
      </div>

      <template v-if="report && t">
        <!-- KPI -->
        <section class="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3">
          <div class="p-3 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f]">
            <div class="text-xs text-neutral-500">
              Request
            </div>
            <div class="text-xl font-bold tabular-nums text-neutral-900 dark:text-white">
              {{ fmtNum(t.requests) }}
            </div>
            <div class="text-xs text-neutral-500">
              {{ t.rps_avg }} req/dtk rata²
            </div>
          </div>
          <div class="p-3 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f]">
            <div class="text-xs text-neutral-500">
              Puncak
            </div>
            <div class="text-xl font-bold tabular-nums text-neutral-900 dark:text-white">
              {{ fmtNum(t.rpm_peak) }}<span class="text-xs font-normal text-neutral-500"> /mnt</span>
            </div>
            <div class="text-xs text-neutral-500">
              {{ t.peak_at ? new Date(t.peak_at * 1000).toLocaleString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) : '–' }}
            </div>
          </div>
          <div class="p-3 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f]">
            <div class="text-xs text-neutral-500">
              Latensi p95
            </div>
            <div class="text-xl font-bold tabular-nums" :class="t.p95 >= 1000 ? 'text-amber-600 dark:text-amber-400' : 'text-neutral-900 dark:text-white'">
              {{ fmtMs(t.p95) }}
            </div>
            <div class="text-xs text-neutral-500">
              p50 {{ fmtMs(t.p50) }} · p99 {{ fmtMs(t.p99) }}
            </div>
          </div>
          <div class="p-3 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f]">
            <div class="text-xs text-neutral-500">
              Error server (5xx)
            </div>
            <div class="text-xl font-bold tabular-nums" :class="t.err_rate >= 0.01 ? 'text-rose-600 dark:text-rose-400' : 'text-neutral-900 dark:text-white'">
              {{ fmtPct(t.err_rate) }}
            </div>
            <div class="text-xs text-neutral-500">
              4xx {{ fmtPct(t.client_err_rate) }}
            </div>
          </div>
          <div class="p-3 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f]">
            <div class="text-xs text-neutral-500" title="Perkiraan request yang diproses bersamaan (Little's law)">
              Request bersamaan
            </div>
            <div class="text-xl font-bold tabular-nums text-neutral-900 dark:text-white">
              {{ t.concurrency_peak }}
            </div>
            <div class="text-xs text-neutral-500">
              rata² {{ t.concurrency_avg }} · {{ fmtPct(t.read_share) }} baca
            </div>
          </div>
          <div class="p-3 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f]">
            <div class="text-xs text-neutral-500">
              Pengunjung
            </div>
            <div class="text-xl font-bold tabular-nums text-neutral-900 dark:text-white">
              {{ fmtNum(t.unique_ips) }}<span class="text-xs font-normal text-neutral-500"> IP</span>
            </div>
            <div class="text-xs text-neutral-500">
              {{ t.instances || 0 }} instance · respons {{ fmtBytes(t.avg_out) }}
            </div>
          </div>
        </section>

        <!-- Grafik -->
        <section class="grid grid-cols-1 xl:grid-cols-2 gap-4">
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] space-y-2 min-w-0">
            <div class="flex items-center justify-between gap-2">
              <h3 class="text-xs font-bold text-neutral-800 dark:text-neutral-100">
                Request per {{ report.step_sec >= 3600 ? `${report.step_sec / 3600} jam` : `${report.step_sec / 60} menit` }}
              </h3>
              <div class="flex items-center gap-3 text-xs text-neutral-500">
                <span class="flex items-center gap-1"><span class="w-2.5 h-2.5 rounded-sm bg-primary-500 dark:bg-primary-400" /> Berhasil & 4xx</span>
                <span class="flex items-center gap-1"><span class="w-2.5 h-2.5 rounded-sm bg-rose-500 dark:bg-rose-400" /> Error 5xx</span>
              </div>
            </div>
            <TrafficSeriesChart :points="report.series" mode="volume" :step-sec="report.step_sec" />
          </div>
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] space-y-2 min-w-0">
            <h3 class="text-xs font-bold text-neutral-800 dark:text-neutral-100">
              Latensi p95
            </h3>
            <TrafficSeriesChart :points="report.series" mode="latency" :step-sec="report.step_sec" />
          </div>
        </section>

        <!-- Tab -->
        <section class="p-4 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] space-y-4">
          <div class="flex flex-wrap gap-1 border-b border-neutral-100 dark:border-white/[0.08] pb-2">
            <UButton
              v-for="tb in tabs"
              :key="tb.value"
              size="sm"
              :icon="tb.icon"
              :color="tab === tb.value ? 'primary' : 'neutral'"
              :variant="tab === tb.value ? 'soft' : 'ghost'"
              @click="tab = tb.value as typeof tab"
            >
              {{ tb.label }}
              <UBadge v-if="tb.badge" size="sm" color="warning" variant="solid" class="ml-1">
                {{ tb.badge }}
              </UBadge>
            </UButton>
          </div>
          <TrafficRecommendations
            v-if="tab === 'advice'"
            :items="report.recommendations"
            :source-id="report.source.id"
            :window="windowKey"
            :has-project="Boolean(source?.project_id)"
          />
          <TrafficRouteTable v-else-if="tab === 'routes'" :routes="report.routes" :groups="report.groups" />
          <div v-else-if="tab === 'origin'" class="space-y-4">
            <TrafficBreakdown :report="report" />
            <TrafficGeoBreakdown :report="report" />
            <p class="text-xs text-neutral-500">
              <a href="https://db-ip.com" target="_blank" rel="noopener" class="underline">IP Geolocation by DB-IP</a>
              · basis data lokasi offline berlisensi CC BY 4.0
            </p>
          </div>
          <TrafficLiveTail v-else-if="tab === 'live'" :events="recent" />
          <TrafficSetupPanel v-else-if="source" :source="source" @rotated="loadSources" />
        </section>
      </template>
    </template>

    <TrafficSourceFormModal v-model:open="showCreate" :meta="meta" @created="handleCreated" />
  </div>
</template>
