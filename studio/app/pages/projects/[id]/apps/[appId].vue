<script setup lang="ts">
import type { Project, ProjectComponent, FastScanResult } from '~/types'

definePageMeta({
  fluid: true
})

const route = useRoute()
const router = useRouter()
const toast = useToast()
const { getProject, checkPorts } = useApi()

const projectId = computed(() => Number(route.params.id))
const appId = computed(() => Number(route.params.appId))

const project = ref<Project | null>(null)
const loading = ref(true)
const activeTab = ref<'blueprint' | 'fast_scan' | 'diagnostics' | 'ai' | 'mcp'>('blueprint')

const comp = computed<ProjectComponent | null>(() => {
  return project.value?.components?.find(c => c.id === appId.value) ?? null
})

const isFrontend = computed(() => {
  const r = comp.value?.role || ''
  return r.includes('frontend') || r === 'frontend_client' || r === 'frontend_admin' || r === 'frontend_mobile'
})

// Ports state
const portStatuses = ref<Record<string, boolean>>({})
const portActive = computed(() => {
  if (!comp.value || comp.value.port <= 0) return false
  return !!portStatuses.value[String(comp.value.port)]
})

const checkActivePort = async () => {
  if (!comp.value || comp.value.port <= 0) return
  portStatuses.value = await checkPorts([comp.value.port])
}

// Blueprint / FastScan state
const parsedScanResult = computed<FastScanResult | null>(() => {
  if (!comp.value?.fast_scan_data) return null
  try {
    return JSON.parse(comp.value.fast_scan_data)
  } catch {
    return null
  }
})

const routeCatalogRef = ref<any>(null)
const routesCount = ref(0)
const onRoutesCountChanged = (count: number) => {
  routesCount.value = count
}

// Fast Scan (job antrean, terapkan parameter, simpan perubahan)
const {
  scanningComp, fastScanModalOpen, activeFastScan, applyingParams, savingScanChanges,
  handleFastScan, handleApplyFastScanParams, handleSaveFastScanChanges
} = useAppFastScan({ comp, project, parsedScanResult, routeCatalogRef, activeTab, reload: quiet => loadData(quiet), checkActivePort })

// Diagnostik kode & AI Advisor
const {
  diagnosing, componentDiagnostic, parsedFindings, handleRunComponentDiagnostics,
  reviewing, customAiQuery, componentAiReview, handleRunComponentAiReview, resetAnalysis
} = useAppAnalysis(comp)

// Dialog ubah/tambah aplikasi + hapus + buat folder
const {
  componentModalOpen, isEditingComp, savingComp, initialEditComponent, deleteConfirmOpen, deletingComp,
  openEditComponentModal, openAddComponentModal, handleSaveComponent, handleDeleteComponent, handleCreateDirOnDisk
} = useAppComponentEditor({ comp, project, projectId, checkActivePort })

// Data loading
// quiet = background refresh after a queued job: no page skeleton, no redirect on a failed reload
const loadData = async (quiet = false) => {
  if (!quiet) loading.value = true
  try {
    const proj = await getProject(projectId.value)
    project.value = proj
    if (comp.value && comp.value.port > 0) {
      await checkActivePort()
    }
  } catch (err: any) {
    if (quiet) return
    toast.add({
      title: 'Gagal Memuat Detail',
      description: `Gagal memuat detail aplikasi: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
    router.push(`/projects/${projectId.value}`)
  } finally {
    if (!quiet) loading.value = false
  }
}

onMounted(() => {
  if (route.query.tab && ['blueprint', 'fast_scan', 'diagnostics', 'ai', 'mcp'].includes(route.query.tab as string)) {
    activeTab.value = route.query.tab as any
  }
  loadData()
})

watch(() => route.query.tab, (newTab) => {
  if (newTab && ['blueprint', 'fast_scan', 'diagnostics', 'ai', 'mcp'].includes(newTab as string)) {
    activeTab.value = newTab as any
  }
})

watch(activeTab, (newTab) => {
  if (route.query.tab !== newTab) {
    router.replace({ query: { ...route.query, tab: newTab } })
  }
})

watch(() => route.params.appId, () => {
  resetAnalysis()
  checkActivePort()
})
</script>

<template>
  <div v-if="loading" class="p-4 sm:p-6 lg:p-8 space-y-6">
    <div class="h-28 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] animate-pulse" />
    <div class="h-96 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] animate-pulse" />
  </div>

  <div v-else-if="project && comp" class="flex-1 flex flex-col lg:flex-row items-stretch w-full min-h-[calc(100vh-4rem)] relative">
    <!-- Main Left Column -->
    <div class="flex-1 min-w-0 w-full p-4 sm:p-6 lg:p-8 space-y-6 lg:mr-80 xl:mr-88 2xl:mr-96">
      <!-- Application Detail Header & Sub-Tabs -->
      <ProjectsDetailApplicationDetailHeader
        v-model:active-tab="activeTab"
        :comp="comp"
        :project="project"
        :port-active="portActive"
        :routes-count="routesCount || parsedScanResult?.routes?.length || 0"
        :findings-count="parsedFindings.length"
        :has-fast-scan="!!parsedScanResult"
        @edit="openEditComponentModal"
        @delete="deleteConfirmOpen = true"
      />

      <!-- TAB 1: BLUEPRINT & RUTE -->
      <div v-if="activeTab === 'blueprint'" class="space-y-6">
        <ProjectsDetailAppScanBar
          :scan-result="parsedScanResult"
          :scanning="scanningComp"
          :is-frontend="isFrontend"
          @scan="handleFastScan"
          @open-result="activeTab = 'fast_scan'"
        />

        <ProjectsDetailAppDesignTagsCard v-if="isFrontend" :component-id="comp.id" />

        <!-- Ringkasan hasil pindai -->
        <div v-if="parsedScanResult" class="space-y-6">
          <!-- Grid Info Arsitektur & Lingkungan -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <!-- Card 1: Cara Menjalankan -->
            <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] space-y-2.5 text-xs">
              <h4 class="font-bold text-neutral-900 dark:text-white flex items-center gap-2">
                <UIcon name="i-lucide-terminal" class="w-4 h-4 text-sky-400" />
                <span>Instruksi Menjalankan</span>
              </h4>
              <div class="space-y-1.5 font-mono text-xs">
                <div v-if="parsedScanResult.how_to_run?.dev_command" class="flex justify-between items-center p-1.5 rounded bg-neutral-50 dark:bg-neutral-800">
                  <span class="text-neutral-400">Dev:</span>
                  <span class="font-bold text-sky-600 dark:text-sky-400">{{ parsedScanResult.how_to_run.dev_command }}</span>
                </div>
                <div v-if="parsedScanResult.how_to_run?.build_command" class="flex justify-between items-center p-1.5 rounded bg-neutral-50 dark:bg-neutral-800">
                  <span class="text-neutral-400">Build:</span>
                  <span class="text-neutral-700 dark:text-neutral-300">{{ parsedScanResult.how_to_run.build_command }}</span>
                </div>
                <div v-if="parsedScanResult.how_to_run?.install_command" class="flex justify-between items-center p-1.5 rounded bg-neutral-50 dark:bg-neutral-800">
                  <span class="text-neutral-400">Install:</span>
                  <span class="text-neutral-700 dark:text-neutral-300">{{ parsedScanResult.how_to_run.install_command }}</span>
                </div>
              </div>
            </div>

            <!-- Card 2: Database & Lingkungan -->
            <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] space-y-2.5 text-xs">
              <h4 class="font-bold text-neutral-900 dark:text-white flex items-center gap-2">
                <UIcon name="i-lucide-database" class="w-4 h-4 text-blue-500" />
                <span>Database & Environment</span>
              </h4>
              <div class="space-y-1.5 font-mono text-xs">
                <div class="flex justify-between items-center p-1.5 rounded bg-neutral-50 dark:bg-neutral-800">
                  <span class="text-neutral-400">Database:</span>
                  <span class="font-bold text-neutral-800 dark:text-neutral-200">{{ parsedScanResult.extracted_env?.database_type || 'Tidak terdeteksi' }}</span>
                </div>
                <div class="flex justify-between items-center p-1.5 rounded bg-neutral-50 dark:bg-neutral-800">
                  <span class="text-neutral-400">ORM / Driver:</span>
                  <span class="text-neutral-700 dark:text-neutral-300">{{ parsedScanResult.extracted_env?.orm_driver || '-' }}</span>
                </div>
                <div class="flex justify-between items-center p-1.5 rounded bg-neutral-50 dark:bg-neutral-800">
                  <span class="text-neutral-400">Total Env Keys:</span>
                  <span class="text-neutral-700 dark:text-neutral-300">{{ parsedScanResult.extracted_env?.environment_keys?.length || 0 }} parameter</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Katalog Rute & API Endpoints Database (Tersimpan Permanen & Sinkron) -->
        <ProjectsDetailRouteCatalogSection
          ref="routeCatalogRef"
          :comp="comp"
          :project="project"
          @routes-count-changed="onRoutesCountChanged"
        />
      </div>

      <!-- TAB FAST SCAN LLM -->
      <ProjectsDetailFastScanDetailTab
        v-else-if="activeTab === 'fast_scan'"
        :comp="comp"
        :project="project"
        :scan-result="parsedScanResult"
        :scanning="scanningComp"
        :applying-params="applyingParams"
        :saving-scan-changes="savingScanChanges"
        @fast-scan="handleFastScan"
        @apply-params="handleApplyFastScanParams"
        @save-changes="handleSaveFastScanChanges"
      />

      <!-- TAB 2: CODEBASE DIAGNOSTICS -->
      <ProjectsDetailAppDetailDiagnosticsTab
        v-else-if="activeTab === 'diagnostics'"
        :comp="comp"
        :loading="loading"
        :diagnosing="diagnosing"
        :component-diagnostic="componentDiagnostic"
        :parsed-findings="parsedFindings"
        @run="handleRunComponentDiagnostics"
      />


      <!-- TAB 3: AI ADVISOR & OPTIMASI -->
      <ProjectsDetailAppDetailAiAdvisorTab
        v-else-if="activeTab === 'ai'"
        v-model:query="customAiQuery"
        :comp="comp"
        :loading="loading"
        :reviewing="reviewing"
        :component-ai-review="componentAiReview"
        @run="handleRunComponentAiReview"
      />


      <!-- TAB 4: AGENT MCP HUB -->
      <ProjectsDetailAppDetailMcpTab v-else-if="activeTab === 'mcp'" :comp="comp" />
    </div>


    <!-- Right Sidebar: Full ke atas sampai batas navbar dan fixed -->
    <ProjectsDetailApplicationDetailSidebar
      :comp="comp"
      :project="project"
      :port-active="portActive"
      :component-diagnostic="componentDiagnostic"
      :has-blueprint="!!parsedScanResult"
      @check-port="checkActivePort"
      @edit="openEditComponentModal"
      @create-dir="handleCreateDirOnDisk"
      @delete="deleteConfirmOpen = true"
      @open-add-modal="openAddComponentModal"
      @switch-tab="activeTab = $event as any"
    />

    <!-- Modals -->
    <ProjectsDetailComponentEditModal
      v-model:open="componentModalOpen"
      :is-editing="isEditingComp"
      :initial-component="initialEditComponent"
      :project-root-path="project?.root_path || ''"
      :saving="savingComp"
      @save="handleSaveComponent"
    />

    <ProjectsDetailConfirmActionModal
      v-model:open="deleteConfirmOpen"
      :title="`Hapus aplikasi '${comp.name}'?`"
      description="Aplikasi dilepas dari proyek ini. File fisik di disk tidak dihapus."
      :loading="deletingComp"
      @confirm="handleDeleteComponent"
    />

    <ProjectsDetailFastScanModal
      v-model:open="fastScanModalOpen"
      :scan-result="activeFastScan"
      :active-comp="comp"
      :scanning-comp-ids="scanningComp && comp ? [comp.id] : []"
      :applying-params="applyingParams"
      :saving-scan-changes="savingScanChanges"
      @apply-params="handleApplyFastScanParams"
      @save-changes="handleSaveFastScanChanges"
      @re-scan="handleFastScan"
    />
  </div>
</template>
