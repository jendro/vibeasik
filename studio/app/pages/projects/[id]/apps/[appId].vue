<script setup lang="ts">
import type { Project, ProjectComponent, FastScanResult, Diagnostic, Finding, AIReviewResponse } from '~/types'

definePageMeta({
  fluid: true
})

const route = useRoute()
const router = useRouter()
const toast = useToast()
const {
  getProject,
  updateComponent,
  deleteComponent,
  createFolderOnDisk,
  updateComponentFastScan,
  applyFastScanParams,
  diagnoseComponent,
  aiReviewComponent,
  checkPorts,
  addComponent
} = useApi()

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

// Fast Scan Modal state
// Fast Scan runs on the engine's LLM queue; the page stays usable while it waits and runs.
const fastScanJob = useJobTarget('fast_scan', () => ({ componentId: comp.value?.id }), async () => {
  await loadData(true)
  if (!comp.value?.fast_scan_data) return
  try {
    activeFastScan.value = JSON.parse(comp.value.fast_scan_data) as FastScanResult
  } catch {
    return
  }
  routeCatalogRef.value?.loadRoutes()
  activeTab.value = 'fast_scan'
})
const scanningComp = fastScanJob.busy
const fastScanModalOpen = ref(false)
const activeFastScan = ref<FastScanResult | null>(null)
const applyingParams = ref(false)
const savingScanChanges = ref(false)

const handleFastScan = () => {
  if (!comp.value) return
  fastScanJob.start()
}

const handleApplyFastScanParams = async (scanData?: FastScanResult) => {
  const scan = scanData || activeFastScan.value || parsedScanResult.value
  if (!scan || !comp.value) return
  applyingParams.value = true
  try {
    const updated = await applyFastScanParams(scan.component_id, {
      dev_command: scan.detected_params.dev_command,
      build_command: scan.detected_params.build_command,
      port: scan.detected_params.port,
      language: scan.detected_params.language,
      framework: scan.detected_params.framework,
      package_manager: scan.detected_params.package_manager
    })

    if (project.value?.components) {
      const idx = project.value.components.findIndex(item => item.id === updated.id)
      if (idx !== -1) {
        project.value.components[idx] = { ...project.value.components[idx], ...updated }
      }
    }
    toast.add({
      title: 'Parameter Berhasil Diterapkan',
      description: 'Parameter berhasil diperbarui ke aplikasi!',
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
    await checkActivePort()
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menerapkan Parameter',
      description: `Gagal menerapkan parameter: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    applyingParams.value = false
  }
}

const handleSaveFastScanChanges = async (scanData?: FastScanResult) => {
  const scan = scanData || activeFastScan.value || parsedScanResult.value
  if (!scan) return
  savingScanChanges.value = true
  try {
    const updated = await updateComponentFastScan(scan.component_id, scan)
    activeFastScan.value = updated
    if (project.value?.components) {
      const idx = project.value.components.findIndex(item => item.id === updated.component_id)
      if (idx !== -1 && project.value.components[idx]) {
        project.value.components[idx].fast_scan_data = JSON.stringify(updated)
      }
    }
    toast.add({
      title: 'Perubahan Disimpan',
      description: 'Informasi env dan metadata dinamis berhasil disimpan!',
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menyimpan Perubahan',
      description: `Gagal menyimpan perubahan: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    savingScanChanges.value = false
  }
}

// Diagnostics state
const diagnosing = ref(false)
const componentDiagnostic = ref<Diagnostic | null>(null)

const parsedFindings = computed<Finding[]>(() => {
  if (!componentDiagnostic.value?.findings_json) return []
  try {
    return JSON.parse(componentDiagnostic.value.findings_json)
  } catch {
    return []
  }
})

const handleRunComponentDiagnostics = async () => {
  if (!comp.value) return
  diagnosing.value = true
  try {
    const res = await diagnoseComponent(comp.value.id)
    componentDiagnostic.value = res
  } catch (err: any) {
    toast.add({
      title: 'Diagnostik Gagal',
      description: `Diagnostik aplikasi gagal: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    diagnosing.value = false
  }
}

// AI Advisor state
const reviewing = ref(false)
const customAiQuery = ref('')
const componentAiReview = ref<AIReviewResponse | null>(null)

const handleRunComponentAiReview = async () => {
  if (!comp.value) return
  reviewing.value = true
  try {
    const res = await aiReviewComponent(comp.value.id, customAiQuery.value)
    componentAiReview.value = res
  } catch (err: any) {
    toast.add({
      title: 'AI Review Gagal',
      description: `AI review aplikasi gagal: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    reviewing.value = false
  }
}

// MCP Config
const mcpConfigJson = computed(() => {
  const compDir = comp.value?.absolute_path || '.'
  return JSON.stringify({
    mcpServers: {
      vibeasik: {
        command: 'vibeasik',
        args: ['mcp'],
        env: {
          ACTIVE_APP_PATH: compDir,
          ACTIVE_APP_NAME: comp.value?.name || ''
        }
      }
    }
  }, null, 2)
})

// Component Edit Modal state
const componentModalOpen = ref(false)
const isEditingComp = ref(false)
const savingComp = ref(false)
const initialEditComponent = ref<Partial<ProjectComponent> | null>(null)

const openEditComponentModal = () => {
  if (!comp.value) return
  isEditingComp.value = true
  initialEditComponent.value = { ...comp.value }
  componentModalOpen.value = true
}

const openAddComponentModal = () => {
  isEditingComp.value = false
  const root = project.value?.root_path || '~/Documents'
  initialEditComponent.value = {
    name: 'Aplikasi Baru',
    role: 'backend',
    relative_path: 'backend',
    absolute_path: `${root}/backend`,
    language: 'go',
    port: 8080
  }
  componentModalOpen.value = true
}

const handleSaveComponent = async (data: Partial<ProjectComponent>, createFolder: boolean) => {
  savingComp.value = true
  try {
    if (isEditingComp.value && data.id) {
      const updated = await updateComponent(data.id, data)
      if (project.value?.components) {
        const idx = project.value.components.findIndex(item => item.id === updated.id)
        if (idx !== -1) {
          project.value.components[idx] = updated
        }
      }
    } else {
      const created = await addComponent(projectId.value, {
        ...data,
        create_folder: createFolder
      })
      if (project.value?.components) {
        project.value.components.push(created)
      }
    }
    componentModalOpen.value = false
    await checkActivePort()
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menyimpan Komponen',
      description: `Gagal menyimpan komponen: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    savingComp.value = false
  }
}

const deleteConfirmOpen = ref(false)
const deletingComp = ref(false)

const handleDeleteComponent = async () => {
  if (!comp.value) return
  deletingComp.value = true
  try {
    await deleteComponent(comp.value.id)
    deleteConfirmOpen.value = false
    router.push(`/projects/${projectId.value}`)
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menghapus Aplikasi',
      description: err.message || 'Terjadi kesalahan',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    deletingComp.value = false
  }
}

const handleCreateDirOnDisk = async () => {
  if (!comp.value) return
  try {
    const res = await createFolderOnDisk(comp.value.id)
    toast.add({
      title: 'Folder Berhasil Dibuat',
      description: `Folder berhasil dibuat di filesystem:\n${res.path}`,
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
  } catch (err: any) {
    toast.add({
      title: 'Gagal Membuat Folder',
      description: `Gagal membuat folder: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  }
}

const copyToClipboard = async (text: string, label = 'Teks berhasil disalin!') => {
  try {
    await navigator.clipboard.writeText(text)
    toast.add({
      title: 'Disalin ke Clipboard',
      description: label,
      icon: 'i-lucide-clipboard-check',
      color: 'neutral'
    })
  } catch {
    // ignore
  }
}

const getScoreColor = (score: number) => {
  if (score >= 90) return 'text-sky-600 dark:text-sky-400 border-sky-500/30 bg-sky-50 dark:bg-sky-950/30'
  if (score >= 70) return 'text-amber-600 dark:text-amber-400 border-amber-500/30 bg-amber-50 dark:bg-amber-950/30'
  return 'text-rose-600 dark:text-rose-400 border-rose-500/30 bg-rose-50 dark:bg-rose-950/30'
}

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
  componentDiagnostic.value = null
  componentAiReview.value = null
  customAiQuery.value = ''
  checkActivePort()
})
</script>

<template>
  <div v-if="loading" class="p-4 sm:p-6 lg:p-8 space-y-6">
    <div class="h-28 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 animate-pulse" />
    <div class="h-96 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 animate-pulse" />
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
            <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-2.5 text-xs shadow-2xs">
              <h4 class="font-bold text-neutral-900 dark:text-white flex items-center gap-2">
                <UIcon name="i-lucide-terminal" class="w-4 h-4 text-sky-400" />
                <span>Instruksi Menjalankan</span>
              </h4>
              <div class="space-y-1.5 font-mono text-[11px]">
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
            <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-2.5 text-xs shadow-2xs">
              <h4 class="font-bold text-neutral-900 dark:text-white flex items-center gap-2">
                <UIcon name="i-lucide-database" class="w-4 h-4 text-blue-500" />
                <span>Database & Environment</span>
              </h4>
              <div class="space-y-1.5 font-mono text-[11px]">
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
      <div v-else-if="activeTab === 'diagnostics'" class="space-y-6">
        <!-- Action Header -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4.5 rounded-xl bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 shadow-xs">
          <div>
            <h4 class="font-bold text-neutral-900 dark:text-white text-sm flex items-center gap-2">
              <UIcon name="i-lucide-shield-alert" class="w-4 h-4 text-sky-400" />
              <span>Audit Kesehatan Kode Aplikasi</span>
            </h4>
            <p class="text-xs text-neutral-500 mt-0.5">
              Audit statis integritas path, konfigurasi .env, test suites, dan deteksi bentrok port untuk {{ comp.name }}.
            </p>
          </div>

          <UButton
            icon="i-lucide-activity"
            color="primary"
            size="xs"
            :loading="diagnosing"
            title="Jalankan audit kesehatan kode aplikasi ini sekarang"
            @click="handleRunComponentDiagnostics"
          >
            {{ componentDiagnostic ? 'Audit Ulang Aplikasi' : 'Jalankan Diagnostik' }}
          </UButton>
        </div>

        <!-- State Belum Diaudit -->
        <div v-if="!componentDiagnostic && !diagnosing" class="p-8 text-center rounded-xl border border-dashed border-neutral-200 dark:border-neutral-800 bg-white/50 dark:bg-neutral-900/50 space-y-3">
          <div class="w-10 h-10 mx-auto rounded-full bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center text-neutral-400">
            <UIcon name="i-lucide-stethoscope" class="w-5 h-5" />
          </div>
          <div class="space-y-1">
            <h4 class="text-sm font-semibold text-neutral-900 dark:text-white">
              Aplikasi ini belum diaudit secara terpisah
            </h4>
            <p class="text-xs text-neutral-500 max-w-md mx-auto">
              Klik tombol di atas untuk menjalankan diagnosa integritas kode, git status, dan deteksi port untuk aplikasi ini.
            </p>
          </div>
          <UButton size="xs" color="primary" @click="handleRunComponentDiagnostics">
            Jalankan Diagnostik Sekarang
          </UButton>
        </div>

        <!-- Hasil Diagnostik -->
        <div v-else-if="componentDiagnostic" class="space-y-4">
          <!-- Score Banner -->
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 flex items-center gap-4 shadow-xs">
            <div
              class="w-14 h-14 rounded-xl flex flex-col items-center justify-center font-mono font-bold border shrink-0"
              :class="getScoreColor(componentDiagnostic.score)"
            >
              <span class="text-xl">{{ componentDiagnostic.score }}</span>
              <span class="text-[8px] uppercase font-normal">Score</span>
            </div>
            <div class="space-y-0.5 min-w-0">
              <div class="font-bold text-sm text-neutral-900 dark:text-white">
                {{ componentDiagnostic.score >= 90 ? 'Kondisi Kode Optimal' : componentDiagnostic.score >= 70 ? 'Perlu Optimasi' : 'Peringatan Kesehatan' }}
              </div>
              <p class="text-xs text-neutral-500 line-clamp-2">
                {{ componentDiagnostic.summary }}
              </p>
            </div>
          </div>

          <!-- Findings List -->
          <div class="space-y-3">
            <h5 class="text-xs font-semibold uppercase tracking-wider text-neutral-400">
              Temuan Audit ({{ parsedFindings.length }})
            </h5>

            <div v-if="parsedFindings.length === 0" class="p-6 rounded-xl border border-dashed border-neutral-200 dark:border-neutral-800 text-center text-xs text-neutral-500 bg-white/50 dark:bg-neutral-900/50">
              Tidak ada masalah terdeteksi. Konfigurasi dan port aplikasi aman.
            </div>

            <div
              v-for="(f, idx) in parsedFindings"
              :key="idx"
              class="p-4 rounded-xl border space-y-2 text-xs"
              :class="f.severity === 'CRITICAL'
                ? 'border-rose-300 dark:border-rose-900/60 bg-rose-50/40 dark:bg-rose-950/20'
                : f.severity === 'WARNING'
                  ? 'border-amber-300 dark:border-amber-900/60 bg-amber-50/40 dark:bg-amber-950/20'
                  : 'border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40'"
            >
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <UIcon
                    :name="f.severity === 'CRITICAL' ? 'i-lucide-alert-triangle' : f.severity === 'WARNING' ? 'i-lucide-alert-circle' : 'i-lucide-info'"
                    class="w-4 h-4"
                    :class="f.severity === 'CRITICAL' ? 'text-rose-500' : f.severity === 'WARNING' ? 'text-amber-500' : 'text-blue-500'"
                  />
                  <span class="font-bold text-neutral-900 dark:text-neutral-100">{{ f.title }}</span>
                  <span class="text-[10px] uppercase font-mono px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 text-neutral-500">
                    {{ f.category }}
                  </span>
                </div>

                <UBadge size="xs" :color="f.severity === 'CRITICAL' ? 'error' : f.severity === 'WARNING' ? 'warning' : 'neutral'">
                  {{ f.severity }}
                </UBadge>
              </div>

              <p class="text-neutral-600 dark:text-neutral-300 leading-relaxed">
                {{ f.description }}
              </p>

              <div class="flex items-center gap-1.5 font-mono text-sky-700 dark:text-sky-400 pt-1">
                <UIcon name="i-lucide-check-circle" class="w-3.5 h-3.5 shrink-0" />
                <span>Saran: {{ f.suggestion }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- TAB 3: AI ADVISOR & OPTIMASI -->
      <div v-else-if="activeTab === 'ai'" class="space-y-6">
        <!-- Action Header & Custom Query -->
        <div class="p-5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs space-y-3.5">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <h4 class="font-bold text-neutral-900 dark:text-white text-sm flex items-center gap-2">
                <UIcon name="i-lucide-bot" class="w-4 h-4 text-sky-400" />
                <span>AI Architecture & Code Advisor untuk {{ comp.name }}</span>
              </h4>
              <p class="text-xs text-neutral-500 mt-0.5">
                Menganalisis stack {{ comp.language }}/{{ comp.framework }} dan menghasilkan saran arsitektur serta instruksi prompt AI.
              </p>
            </div>

            <UButton
              icon="i-lucide-sparkles"
              color="primary"
              size="xs"
              :loading="reviewing"
              title="Jalankan review AI untuk aplikasi ini"
              @click="handleRunComponentAiReview"
            >
              {{ componentAiReview ? 'Review Ulang AI' : 'Jalankan Review AI' }}
            </UButton>
          </div>

          <!-- Custom query input -->
          <div class="flex items-center gap-2 pt-1 border-t border-neutral-100 dark:border-neutral-800">
            <UInput
              v-model="customAiQuery"
              placeholder="Ajukan pertanyaan khusus seputar aplikasi ini (opsional)..."
              size="xs"
              class="flex-1"
            />
          </div>
        </div>

        <!-- State Belum Ada Review -->
        <div v-if="!componentAiReview && !reviewing" class="p-8 text-center rounded-xl border border-dashed border-neutral-200 dark:border-neutral-800 bg-white/50 dark:bg-neutral-900/50 space-y-3">
          <div class="w-10 h-10 mx-auto rounded-full bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center text-neutral-400">
            <UIcon name="i-lucide-sparkles" class="w-5 h-5" />
          </div>
          <div class="space-y-1">
            <h4 class="text-sm font-semibold text-neutral-900 dark:text-white">
              Belum ada analisis AI untuk aplikasi ini
            </h4>
            <p class="text-xs text-neutral-500 max-w-md mx-auto">
              Klik Jalankan Review AI untuk mendapatkan analisis kekuatan, rencana perbaikan, dan instruksi prompt agent.
            </p>
          </div>
          <UButton size="xs" color="primary" @click="handleRunComponentAiReview">
            Mulai Review AI
          </UButton>
        </div>

        <!-- Hasil Review AI -->
        <div v-else-if="componentAiReview" class="space-y-4">
          <!-- Summary Box -->
          <div class="p-4 rounded-xl border border-sky-200 dark:border-sky-800/60 bg-sky-50/70 dark:bg-sky-950/40 text-xs space-y-1.5 shadow-2xs">
            <div class="font-bold text-sky-900 dark:text-sky-200 flex items-center gap-1.5">
              <UIcon name="i-lucide-check-circle" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
              <span>Ringkasan Evaluasi AI:</span>
            </div>
            <p class="text-neutral-700 dark:text-neutral-300 leading-relaxed">
              {{ componentAiReview.summary }}
            </p>
          </div>

          <!-- Action Plan Grid -->
          <div v-if="componentAiReview.action_plan?.length" class="space-y-2.5">
            <h5 class="text-xs font-semibold uppercase tracking-wider text-neutral-400">
              Rencana Aksi & Rekomendasi Fitur
            </h5>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
              <div
                v-for="(plan, idx) in componentAiReview.action_plan"
                :key="idx"
                class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-1.5 shadow-2xs"
              >
                <div class="flex items-center justify-between">
                  <span class="font-bold text-neutral-900 dark:text-white">{{ plan.title }}</span>
                  <UBadge size="xs" :color="plan.priority === 'P0' ? 'error' : 'warning'">
                    {{ plan.priority }}
                  </UBadge>
                </div>
                <p class="text-neutral-600 dark:text-neutral-400 text-[11px] leading-relaxed">
                  {{ plan.description }}
                </p>
              </div>
            </div>
          </div>

          <!-- AI Agent Ready Prompt -->
          <div v-if="componentAiReview.ai_agent_prompt" class="p-4 rounded-xl bg-neutral-950 text-neutral-200 space-y-2 text-xs shadow-xs">
            <div class="flex items-center justify-between">
              <span class="font-mono text-sky-400 font-semibold text-[11px]">Prompt AI Agent (Antigravity / Claude Code)</span>
              <UButton
                icon="i-lucide-copy"
                color="neutral"
                variant="ghost"
                size="xs"
                @click="copyToClipboard(componentAiReview.ai_agent_prompt, 'Prompt AI berhasil disalin!')"
              >
                Salin Prompt
              </UButton>
            </div>
            <pre class="p-3 rounded-lg bg-black/40 font-mono text-[11px] text-neutral-300 overflow-x-auto whitespace-pre-wrap max-h-48 overflow-y-auto">{{ componentAiReview.ai_agent_prompt }}</pre>
          </div>
        </div>
      </div>

      <!-- TAB 4: AGENT MCP HUB -->
      <div v-else-if="activeTab === 'mcp'" class="space-y-6">
        <div class="p-5 sm:p-6 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs space-y-4">
          <div>
            <h4 class="font-bold text-neutral-900 dark:text-white text-sm flex items-center gap-2">
              <UIcon name="i-lucide-cpu" class="w-4 h-4 text-sky-400" />
              <span>Model Context Protocol (MCP) untuk {{ comp.name }}</span>
            </h4>
            <p class="text-xs text-neutral-500 mt-1 leading-relaxed">
              Gunakan konfigurasi ini agar AI agent (Google Antigravity, Claude Code, Cursor) memiliki konteks langsung dan direktori kerja di aplikasi ini.
            </p>
          </div>

          <!-- Target Directory Indicator -->
          <div class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-700/60 text-xs font-mono flex items-center justify-between">
            <span class="text-neutral-500">Direktori Kerja Agent:</span>
            <span class="text-neutral-900 dark:text-neutral-100 font-bold truncate max-w-sm">{{ comp.absolute_path }}</span>
          </div>

          <!-- MCP Configuration JSON -->
          <div class="p-4 rounded-xl bg-neutral-950 text-neutral-200 space-y-2 text-xs">
            <div class="flex items-center justify-between">
              <span class="font-mono text-sky-400 font-semibold text-[11px]">stdio Config JSON</span>
              <UButton
                icon="i-lucide-copy"
                color="neutral"
                variant="ghost"
                size="xs"
                @click="copyToClipboard(mcpConfigJson, 'Konfigurasi MCP disalin!')"
              >
                Salin Konfigurasi
              </UButton>
            </div>
            <pre class="p-3 rounded-lg bg-[#010810] font-mono text-sky-300 border border-white/10 text-[11px] overflow-x-auto">{{ mcpConfigJson }}</pre>
          </div>

          <!-- Agent Usage Guidance -->
          <div class="p-4 rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-200 dark:border-neutral-800 text-xs space-y-2">
            <div class="font-bold text-neutral-900 dark:text-white">
              Cara Menggunakan:
            </div>
            <ul class="list-disc list-inside space-y-1 text-neutral-600 dark:text-neutral-300">
              <li>Buka terminal di folder <code class="font-mono text-sky-600 dark:text-sky-400">{{ comp.absolute_path }}</code></li>
              <li>Jalankan agent dengan perintah <code class="font-mono">agy</code> atau <code class="font-mono">claude</code></li>
              <li>Agent akan otomatis memanfaatkan tool MCP vibeasik untuk memeriksa rute, port, dan dependensi.</li>
            </ul>
          </div>
        </div>
      </div>
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
