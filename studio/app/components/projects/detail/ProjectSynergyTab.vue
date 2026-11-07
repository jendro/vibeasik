<script setup lang="ts">
import type { Project, ProjectSynergyMap, SynergyRequestFlow } from '~/types'
import type { SynergyInspectPage, SynergyInspectTab } from '~/utils/synergy'

const props = defineProps<{
  project: Project
  portStatuses: Record<string, boolean>
  activePortsCount: number
  totalConfiguredPorts: number
}>()

defineEmits<{
  checkPorts: []
  openAddModal: [role?: string]
  openDetail: [comp: ProjectComponent]
  switchTab: [tab: string]
}>()

const toast = useToast()
const { getProjectSynergyMap, generateProjectSynergyMap } = useApi()

const synergyMap = ref<ProjectSynergyMap | null>(null)
const loading = ref(false)
// "Rangkai AI" is queued on the engine; the page stays usable while it waits and runs.
const synergyJob = useJobTarget('synergy_map', () => ({ projectId: props.project.id }), async () => {
  try {
    const result = await getProjectSynergyMap(props.project.id)
    synergyMap.value = result
    toast.add(result.llm_enriched
      ? {
          title: 'Pemetaan AI Berhasil Dirangkai!',
          description: 'Peta sinergi telah diperkaya dengan penalaran mendalam oleh LLM.',
          icon: 'i-lucide-check-circle',
          color: 'success'
        }
      : {
          title: 'Pemetaan Diperbarui Secara Heuristik',
          description: 'Model AI belum merespon; peta sinergi disusun berdasarkan ekstraksi kode sumber statis.',
          icon: 'i-lucide-info',
          color: 'warning'
        })
  } catch (err: unknown) {
    toast.add({
      title: 'Gagal Memuat Peta Sinergi',
      description: err instanceof Error ? err.message : 'Gagal memuat hasil peta sinergi',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  }
})
const generatingLLM = synergyJob.busy

// View Mode: 'pipeline' (Alur Visual) | 'matrix' (Matriks Audit Tabular) | 'topology' (Peta Relasi DB & API)
const activeView = ref<'pipeline' | 'matrix' | 'topology'>('pipeline')

// Filter + data turunan (halaman terfilter, baris matriks, daftar tabel)
const { searchQuery, filterMethod, filterTable, availableTables, filteredPages, matrixRows, resetFilters } = useSynergyFilters(synergyMap)

// Dialog inspektur (state di dalam komponen inspektur)
const inspector = ref<{ open: (flow: SynergyRequestFlow, page: SynergyInspectPage, tab?: SynergyInspectTab) => void } | null>(null)
const openInspector = (flow: SynergyRequestFlow, page: SynergyInspectPage, tab?: SynergyInspectTab) => inspector.value?.open(flow, page, tab)

// Page Collapsible state in visual pipeline
const collapsedPages = ref<Record<string, boolean>>({})

const togglePageCollapse = (pageId: string) => {
  collapsedPages.value[pageId] = !collapsedPages.value[pageId]
}

const collapseAllPages = () => {
  if (!synergyMap.value) return
  for (const page of synergyMap.value.pages) {
    collapsedPages.value[page.id] = true
  }
}

const expandAllPages = () => {
  collapsedPages.value = {}
}

const areAllPagesCollapsed = computed(() => {
  if (!synergyMap.value?.pages?.length) return false
  return synergyMap.value.pages.every(p => collapsedPages.value[p.id])
})

const loadSynergyData = async (forceRefresh = false) => {
  loading.value = true
  try {
    if (forceRefresh) {
      synergyMap.value = await generateProjectSynergyMap(props.project.id, false)
      toast.add({
        title: 'Peta Sinergi Diperbarui',
        description: 'Pemetaan rute, file, dan alur database berhasil disinkronkan.',
        icon: 'i-lucide-check-circle',
        color: 'success'
      })
    } else {
      synergyMap.value = await getProjectSynergyMap(props.project.id)
    }
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Gagal memuat peta sinergi'
    toast.add({
      title: 'Gagal Memuat Peta Sinergi',
      description: msg,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}

const handleGenerateWithLLM = () => {
  synergyJob.start()
}

onMounted(() => {
  loadSynergyData(false)
})

const copySummaryMarkdown = async () => {
  if (!synergyMap.value) return

  try {
    await navigator.clipboard.writeText(buildSynergyMarkdown(synergyMap.value))
    toast.add({
      title: 'Pemetaan Disalin ke Clipboard',
      description: 'Format Markdown lengkap berhasil disalin.',
      icon: 'i-lucide-clipboard-check',
      color: 'success'
    })
  } catch {
    toast.add({
      title: 'Gagal Menyalin',
      description: 'Silakan salin teks secara manual.',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  }
}

const downloadSynergyJson = () => {
  if (!synergyMap.value) return
  const dataStr = 'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(synergyMap.value, null, 2))
  const downloadAnchor = document.createElement('a')
  downloadAnchor.setAttribute('href', dataStr)
  downloadAnchor.setAttribute('download', `synergy-map-${props.project.slug || 'project'}.json`)
  document.body.appendChild(downloadAnchor)
  downloadAnchor.click()
  downloadAnchor.remove()
  toast.add({
    title: 'JSON Sinergi Berhasil Diunduh',
    description: `File synergy-map-${props.project.slug || 'project'}.json tersimpan.`,
    icon: 'i-lucide-download',
    color: 'success'
  })
}
</script>

<template>
  <div class="space-y-6">
    <ProjectSynergyHeader
      :synergy-map="synergyMap"
      :loading="loading"
      :generating-llm="generatingLLM"
      :job-label="synergyJob.label.value"
      @generate="handleGenerateWithLLM"
      @refresh="loadSynergyData"
      @download="downloadSynergyJson"
      @copy="copySummaryMarkdown"
    />

    <ProjectSynergyToolbar
      v-model:active-view="activeView"
      v-model:search-query="searchQuery"
      v-model:filter-method="filterMethod"
      v-model:filter-table="filterTable"
      :matrix-count="matrixRows.length"
      :page-count="filteredPages.length"
      :available-tables="availableTables"
      :are-all-pages-collapsed="areAllPagesCollapsed"
      @collapse-all="collapseAllPages"
      @expand-all="expandAllPages"
      @reset="resetFilters"
    />

    <!-- Loading Skeleton -->
    <div v-if="loading && !synergyMap" class="space-y-4">
      <div v-for="i in 3" :key="i" class="h-44 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-neutral-100 dark:bg-neutral-800/40 animate-pulse" />
    </div>

    <!-- EMPTY STATE -->
    <div
      v-else-if="!filteredPages || filteredPages.length === 0"
      class="p-12 text-center rounded-2xl border border-dashed border-neutral-200 dark:border-neutral-800 bg-white dark:bg-slate-900/40 space-y-3"
    >
      <UIcon name="i-lucide-search-x" class="w-10 h-10 text-neutral-400 mx-auto" />
      <div class="text-sm font-semibold text-neutral-700 dark:text-neutral-300">
        Tidak ada data halaman atau alur yang cocok dengan kriteria filter
      </div>
      <p class="text-xs text-neutral-500 max-w-sm mx-auto">
        Ubah kata kunci pencarian, reset filter metode, atau klik "Segarkan" untuk memperbarui data dari kode sumber.
      </p>
      <div class="pt-2">
        <UButton size="xs" color="primary" variant="soft" class="cursor-pointer" @click="resetFilters">
          Hapus Semua Filter
        </UButton>
      </div>
    </div>

    <ProjectSynergyPipelineView
      v-else-if="activeView === 'pipeline'"
      :filtered-pages="filteredPages"
      :collapsed-pages="collapsedPages"
      @toggle="togglePageCollapse"
      @inspect="openInspector"
    />

    <ProjectSynergyMatrixView
      v-else-if="activeView === 'matrix'"
      :matrix-rows="matrixRows"
      @inspect="openInspector"
    />

    <ProjectSynergyTopologyView v-else-if="activeView === 'topology'" :synergy-map="synergyMap" />

    <ProjectSynergyInspector ref="inspector" :project="project" />
  </div>
</template>
