<script setup lang="ts">
import type { CodeMapOverview } from '~/types'

/**
 * Peta Kode: indeks per berkas gaya RAG agar AI agent menemukan lokasi kode tanpa menjelajah seluruh repo.
 * Agent memakainya lewat MCP (vibeasik_code_map_*) atau .vibeasik/CODEMAP.md; halaman ini untuk memantau & menguji.
 */
definePageMeta({ fluid: true })

const { activeProjectId, activeProject, fetchProjects } = useActiveProject()
const { getOverview, build } = useCodeMap()
const { initing, initProject } = useInitVibeasik()
const toast = useToast()

const overview = ref<CodeMapOverview | null>(null)
const loading = ref(false)
const building = ref(false)
const error = ref('')
const openPath = ref<string | null>(null)

const load = async () => {
  if (!activeProjectId.value) return
  loading.value = true
  error.value = ''
  try {
    overview.value = await getOverview(activeProjectId.value)
  } catch (err: any) {
    error.value = err?.data?.error || err?.message || 'Gagal memuat peta kode'
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await fetchProjects()
  await load()
})
watch(activeProjectId, () => {
  overview.value = null
  load()
})

const rebuild = async (force: boolean) => {
  if (!activeProjectId.value) return
  building.value = true
  try {
    const r = await build(activeProjectId.value, force)
    toast.add({ title: 'Peta kode diperbarui', description: `${r.parsed} berkas dipindai ulang, ${r.unchanged} tetap, ${r.removed} dihapus (${r.duration_ms} ms)`, icon: 'i-lucide-map', color: 'success' })
    await load()
  } catch (err: any) {
    toast.add({ title: 'Gagal membangun peta', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' })
  } finally {
    building.value = false
  }
}

const runInit = async () => {
  if (!activeProjectId.value) return
  if (await initProject(activeProjectId.value, activeProject.value?.name)) await load()
}

const enrichJob = useJobTarget('code_map_enrich', () => (activeProjectId.value ? { projectId: activeProjectId.value } : null), async (job) => {
  toast.add({ title: 'Ringkasan peta kode selesai', description: job.message, icon: 'i-lucide-sparkles', color: 'success' })
  await load()
})

const fmtTime = (s?: string) => (s ? new Date(s).toLocaleString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) : '-')
const stats = computed(() => {
  const o = overview.value
  if (!o) return []
  return [
    { label: 'Berkas', value: o.file_count.toLocaleString('id-ID'), icon: 'i-lucide-files' },
    { label: 'Simbol', value: o.symbol_count.toLocaleString('id-ID'), icon: 'i-lucide-braces' },
    { label: 'Baris kode', value: o.total_lines.toLocaleString('id-ID'), icon: 'i-lucide-text' },
    { label: 'Rute & panggilan API', value: o.route_count.toLocaleString('id-ID'), icon: 'i-lucide-route' },
    { label: 'Ringkasan LLM', value: `${o.enriched_count}/${o.file_count}`, icon: 'i-lucide-sparkles' },
    { label: 'Dibangun', value: fmtTime(o.built_at), icon: 'i-lucide-clock' }
  ]
})

const mcpTools = ['vibeasik_code_map_search', 'vibeasik_code_map_file', 'vibeasik_code_map_overview', 'vibeasik_code_map_build', 'vibeasik_code_map_export']
</script>

<template>
  <div class="w-full px-4 sm:px-6 lg:px-8 pt-6 pb-20 space-y-6">
    <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4 p-5 sm:p-6 rounded-2xl bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 shadow-xs">
      <div class="flex items-center gap-3.5 min-w-0">
        <div class="size-11 shrink-0 rounded-xl bg-linear-to-br from-sky-500 to-violet-600 text-white flex items-center justify-center shadow-xs">
          <UIcon name="i-lucide-map" class="size-5.5" />
        </div>
        <div class="min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <h1 class="text-xl font-bold tracking-tight text-neutral-900 dark:text-white">
              Peta Kode untuk AI Agent
            </h1>
            <UBadge v-if="activeProject" size="sm" color="primary" variant="subtle">
              {{ activeProject.name }}
            </UBadge>
          </div>
          <p class="text-xs sm:text-sm text-neutral-500 dark:text-neutral-400 mt-1 leading-relaxed">
            Indeks per berkas (peran, simbol, impor, dipakai oleh, rute, ringkasan) yang dicari ala RAG — agent menemukan lokasi kode tanpa membaca seluruh repositori.
          </p>
        </div>
      </div>
      <div v-if="activeProjectId" class="flex flex-wrap items-center gap-2 shrink-0">
        <UButton color="neutral" variant="outline" icon="i-lucide-refresh-cw" label="Bangun ulang" :loading="building" @click="rebuild(false)" />
        <UButton
          color="neutral"
          variant="outline"
          icon="i-lucide-sparkles"
          :label="enrichJob.busy.value ? enrichJob.label.value : 'Ringkas dengan LLM'"
          :loading="enrichJob.busy.value"
          :disabled="enrichJob.busy.value"
          title="Masuk antrean: LLM menulis ringkasan & kata kunci per berkas agar pencarian agent lebih tepat"
          @click="enrichJob.start()"
        />
        <UButton
          color="primary"
          icon="i-lucide-sparkles"
          label="Init vibeasik"
          :loading="initing"
          title="Tulis .vibeasik.json (tautan AI) + .vibeasik/CODEMAP.md & codemap.jsonl; setelah itu peta diperbarui otomatis saat kode berubah"
          @click="runInit"
        />
      </div>
    </div>

    <UAlert v-if="!activeProjectId" color="neutral" variant="subtle" icon="i-lucide-folder-search" title="Pilih proyek aktif" description="Pilih proyek di sidebar untuk melihat peta kodenya." />
    <UAlert v-if="error" color="error" variant="subtle" icon="i-lucide-alert-circle" title="Peta kode tidak bisa dimuat" :description="error" />

    <div v-if="loading && !overview" class="py-16 flex flex-col items-center gap-2 text-neutral-500 text-sm">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
      Memindai berkas proyek…
    </div>

    <template v-if="overview && activeProjectId">
      <div class="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3 sm:gap-4">
        <div v-for="s in stats" :key="s.label" class="p-3.5 sm:p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 shadow-xs hover:border-neutral-300 dark:hover:border-neutral-700 transition-colors">
          <div class="flex items-center gap-1.5 text-[11px] font-medium text-neutral-500 dark:text-neutral-400">
            <UIcon :name="s.icon" class="size-3.5 text-neutral-400 dark:text-neutral-500" />
            {{ s.label }}
          </div>
          <p class="mt-1.5 text-base sm:text-lg font-bold font-mono text-neutral-900 dark:text-white">
            {{ s.value }}
          </p>
        </div>
      </div>

      <CodeMapSearch :project-id="activeProjectId" @open="openPath = $event" />
      <CodeMapInsights :overview="overview" @open="openPath = $event" />
      <CodeMapTree :overview="overview" @open="openPath = $event" />

      <section class="p-4 sm:p-5 rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700/80 bg-neutral-50/50 dark:bg-neutral-900/40 text-xs text-neutral-600 dark:text-neutral-300 space-y-2.5">
        <h3 class="font-semibold text-neutral-800 dark:text-neutral-100 flex items-center gap-2">
          <UIcon name="i-lucide-bot" class="size-4 text-violet-500" /> Cara agent memakai peta ini
        </h3>
        <p class="leading-relaxed">
          {{ overview.guide }}
        </p>
        <div class="flex flex-wrap gap-1.5 py-1">
          <code v-for="t in mcpTools" :key="t" class="px-2 py-0.5 rounded-md bg-neutral-100 dark:bg-neutral-800/80 text-neutral-800 dark:text-neutral-200 font-mono text-[11px] border border-neutral-200/60 dark:border-neutral-700/60">{{ t }}</code>
        </div>
        <p class="text-neutral-500 dark:text-neutral-400">
          Tanpa MCP: agent membaca <code class="px-1 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 font-mono text-[11px]">.vibeasik/CODEMAP.md</code> atau grep <code class="px-1 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 font-mono text-[11px]">.vibeasik/codemap.jsonl</code> (dibuat oleh Init vibeasik).
        </p>
        <p v-if="overview.exported_at" class="flex items-center gap-1.5 text-emerald-700 dark:text-emerald-400 font-medium">
          <UIcon name="i-lucide-refresh-cw" class="size-3.5 shrink-0" />
          Auto-sync aktif: engine memeriksa perubahan kode tiap 2 menit dan menulis ulang CODEMAP.md bila ada berkas berubah (terakhir {{ fmtTime(overview.exported_at) }}).
          Pencarian MCP selalu memakai peta ≤30 detik.
        </p>
        <p v-else class="flex items-center gap-1.5 text-amber-700 dark:text-amber-400 font-medium">
          <UIcon name="i-lucide-info" class="size-3.5 shrink-0" />
          Belum di-init: peta hanya ada di engine (MCP & halaman ini). Klik Init vibeasik agar peta ditulis ke repo dan diperbarui otomatis.
        </p>
      </section>
    </template>

    <CodeMapFileSlideover v-if="activeProjectId" v-model:path="openPath" :project-id="activeProjectId" />
  </div>
</template>
