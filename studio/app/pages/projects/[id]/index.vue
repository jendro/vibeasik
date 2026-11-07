<script setup lang="ts">
import type { Project, ProjectComponent, FastScanResult, ProjectBaseKnowledge, ProjectProfile } from '~/types'

definePageMeta({
  fluid: true
})

const route = useRoute()
const router = useRouter()
const toast = useToast()
const {
  getProject,
  addComponent,
  createFolderOnDisk,
  updateComponent,
  deleteComponent,
  deleteProject,
  checkPorts,
  updateComponentFastScan,
  applyFastScanParams,
  startProjectPipeline,
  getProjectBaseKnowledge,
  refreshProjectBaseKnowledge,
  getProjectProfile
} = useApi()

const { selectActiveProject } = useActiveProject()

const projectId = computed(() => Number(route.params.id))
const project = ref<Project | null>(null)
const loading = ref(true)
// Legacy tab ids keep old links (?tab=topology / ?tab=base_knowledge) working
const TAB_ALIASES: Record<string, string> = { topology: 'apps', base_knowledge: 'technical' }
const VALID_TABS = ['overview', 'apps', 'technical', 'synergy', 'ai_chat', 'visual_audit']
const normalizeTab = (tab: unknown) => {
  const t = typeof tab === 'string' ? (TAB_ALIASES[tab] || tab) : ''
  return VALID_TABS.includes(t) ? t : 'overview'
}
const activeTab = ref(normalizeTab(route.query.tab))

// Project profile (purpose, apps, merged features, roadmap) shared by Ringkasan & Aplikasi tabs
const profile = ref<ProjectProfile | null>(null)
const loadingProfile = ref(false)

// Profile generation runs on the engine's LLM queue; the page stays usable while it waits/runs.
const profileJob = useJobTarget('project_profile', () => ({ projectId: projectId.value }), async () => {
  await refreshAfterProfileJob()
})
const summaryJob = useJobTarget('project_summary', () => ({ projectId: projectId.value }), async () => {
  await refreshAfterProfileJob()
})
const generatingProfile = computed(() => profileJob.busy.value || summaryJob.busy.value)
const generatingProfileLabel = computed(() => profileJob.label.value || summaryJob.label.value)

const loadProfile = async () => {
  loadingProfile.value = true
  try {
    profile.value = await getProjectProfile(projectId.value)
  } catch (err: any) {
    toast.add({
      title: 'Gagal Memuat Profil Proyek',
      description: err.message || 'Terjadi kesalahan saat memuat profil proyek',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    loadingProfile.value = false
  }
}

const handleGenerateProfile = () => {
  if (generatingProfile.value) return
  profileJob.start()
}

// Called when a queued profile/summary job finishes: reload what it wrote and report the outcome.
const refreshAfterProfileJob = async () => {
  await loadProfile()
  const res = profile.value
  if (!res) return
  // llm_error means the LLM did not answer; the backend then keeps the previously saved profile
  const ok = res.source === 'llm' && !res.llm_error
  if (project.value && ok) {
    project.value.short_description = res.short_description
    project.value.description = res.full_description
  }
  toast.add({
    title: ok ? 'Profil Proyek Diperbarui' : 'AI Tidak Merespons',
    description: ok
      ? `${res.features.length} fitur dan ${res.roadmap.length} langkah roadmap berhasil disusun.`
      : `${res.source === 'heuristic' ? 'Profil disusun secara heuristik.' : 'Profil sebelumnya tetap dipakai.'} ${res.llm_error || 'Periksa konfigurasi LLM di Settings.'}`,
    icon: ok ? 'i-lucide-check-circle' : 'i-lucide-info',
    color: ok ? 'success' : 'warning'
  })
}

// Profile saved by an external AI agent (MCP save_project_profile) or pasted JSON
const handleProfileUpdated = (res: ProjectProfile) => {
  profile.value = res
  if (project.value) {
    project.value.short_description = res.short_description
    project.value.description = res.full_description
  }
}

// Cached Base Knowledge state on page level so switching tabs does not reload from BE
const cachedBaseKnowledge = ref<ProjectBaseKnowledge | null>(null)
const loadingBaseKnowledge = ref(false)

const loadProjectBaseKnowledge = async (forceRefresh = false) => {
  if (cachedBaseKnowledge.value && !forceRefresh) {
    return cachedBaseKnowledge.value
  }
  loadingBaseKnowledge.value = true
  try {
    if (forceRefresh) {
      cachedBaseKnowledge.value = await refreshProjectBaseKnowledge(projectId.value)
    } else {
      cachedBaseKnowledge.value = await getProjectBaseKnowledge(projectId.value)
    }
    return cachedBaseKnowledge.value
  } catch (err: any) {
    toast.add({
      title: 'Gagal Memuat Base Knowledge',
      description: err.message || 'Terjadi kesalahan saat memuat base knowledge',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    loadingBaseKnowledge.value = false
  }
}

watch(() => route.query.tab, (newTab) => {
  if (newTab) {
    activeTab.value = normalizeTab(newTab)
  }
})

watch(activeTab, (newTab) => {
  if (route.query.tab !== newTab) {
    router.replace({ query: { ...route.query, tab: newTab } })
  }
  if (import.meta.client) window.scrollTo({ top: 0 })
})

// Application Detail navigation
const handleOpenAppDetail = (c: ProjectComponent) => {
  router.push(`/projects/${projectId.value}/apps/${c.id}`)
}

// Fast Scan state
// Components with a queued/running Fast Scan (several can wait in the LLM queue at once)
const { jobs: queueJobs, enqueue } = useBackgroundJobs()
const scanningCompIds = computed(() =>
  queueJobs.value
    .filter(j => j.type === 'fast_scan' && j.project_id === projectId.value && (j.status === 'queued' || j.status === 'running'))
    .map(j => j.component_id as number)
)
// Components whose scan result should pop open the blueprint modal once the job finishes
const openModalAfterScan = new Set<number>()
const fastScanModalOpen = ref(false)
const activeFastScan = ref<FastScanResult | null>(null)
const activeScanningComp = ref<ProjectComponent | null>(null)
const applyingParams = ref(false)
const savingScanChanges = ref(false)

// Ports state
const portStatuses = ref<Record<string, boolean>>({})
const activePortsCount = computed(() => Object.values(portStatuses.value).filter(Boolean).length)
const totalConfiguredPorts = computed(() => project.value?.components?.filter(c => c.port > 0).length || 0)

// Component Modal state
const componentModalOpen = ref(false)
const isEditingComp = ref(false)
const savingComp = ref(false)
const creatingDirId = ref<number | null>(null)
const initialEditComponent = ref<Partial<ProjectComponent> | null>(null)

// Data loading
const checkActivePorts = async () => {
  if (!project.value?.components) return
  const ports = project.value.components.map(c => c.port).filter(p => p > 0)
  if (ports.length > 0) {
    portStatuses.value = await checkPorts(ports)
  }
}

// quiet = background refresh after a queued job: no page skeleton, no redirect on a failed reload
const loadProjectData = async (quiet = false) => {
  if (!quiet) loading.value = true
  try {
    const data = await getProject(projectId.value)
    project.value = data
    selectActiveProject(projectId.value)
    await checkActivePorts()
  } catch (err: any) {
    if (quiet) return
    toast.add({
      title: 'Gagal Memuat Proyek',
      description: `Failed to load project: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
    router.push('/')
  } finally {
    if (!quiet) loading.value = false
  }
}

onMounted(() => {
  loadProjectData()
  loadProfile()
})

// Delete confirmations (Zero Alert Policy: no window.confirm)
const deleteProjectOpen = ref(false)
const deletingProject = ref(false)
const compToDelete = ref<ProjectComponent | null>(null)
const deleteCompOpen = computed({
  get: () => compToDelete.value !== null,
  set: (v: boolean) => { if (!v) compToDelete.value = null }
})
const deletingComp = ref(false)

const handleDeleteProject = async () => {
  deletingProject.value = true
  try {
    await deleteProject(projectId.value)
    deleteProjectOpen.value = false
    router.push('/')
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menghapus Pantauan',
      description: err.message || 'Terjadi kesalahan',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    deletingProject.value = false
  }
}

const handleStartPipeline = async () => {
  try {
    const res = await startProjectPipeline(projectId.value)
    toast.add({
      title: 'Pipeline Berhasil Dipicu',
      description: `Pipeline background berhasil dipicu (${res.jobs?.length || 0} tugas masuk antrean). Pantau proses di footer status bar!`,
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
  } catch (err: any) {
    toast.add({
      title: 'Gagal Memicu Pipeline',
      description: `Gagal memicu pipeline: ${err.message || 'Terjadi kesalahan'}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  }
}

// Component Actions
const openAddComponentModal = (role = 'backend') => {
  isEditingComp.value = false
  const root = project.value?.root_path || '~/Documents'
  const defaults: Record<string, Partial<ProjectComponent>> = {
    backend: {
      name: 'Backend API (BE)',
      role: 'backend',
      relative_path: 'BE',
      absolute_path: `${root}/BE`,
      language: 'go',
      framework: 'fiber',
      package_manager: 'go',
      dev_command: 'go run main.go',
      port: 8080,
      health_url: 'http://localhost:8080/health',
      env_file: '.env',
      env_summary: ''
    },
    frontend_client: {
      name: 'Frontend Web (FE)',
      role: 'frontend_client',
      relative_path: 'FE',
      absolute_path: `${root}/FE`,
      language: 'typescript',
      framework: 'nuxt',
      package_manager: 'bun',
      dev_command: 'bun run dev',
      port: 3000,
      health_url: 'http://localhost:3000',
      env_file: '.env',
      env_summary: ''
    },
    frontend_admin: {
      name: 'Admin Dashboard',
      role: 'frontend_admin',
      relative_path: 'admin',
      absolute_path: `${root}/admin`,
      language: 'typescript',
      framework: 'nuxt',
      package_manager: 'bun',
      dev_command: 'bun run dev',
      port: 3001,
      health_url: 'http://localhost:3001',
      env_file: '.env',
      env_summary: ''
    },
    frontend_mobile: {
      name: 'Mobile App',
      role: 'frontend_mobile',
      relative_path: 'mobile',
      absolute_path: `${root}/mobile`,
      language: 'dart',
      framework: 'flutter',
      package_manager: 'pub',
      dev_command: 'flutter run',
      port: 0,
      health_url: '',
      env_file: '.env',
      env_summary: ''
    }
  }

  initialEditComponent.value = defaults[role] || {
    name: 'Folder Lain',
    role: role || 'other',
    relative_path: 'docs',
    absolute_path: `${root}/docs`,
    language: 'auto',
    port: 0
  }
  componentModalOpen.value = true
}

const openEditComponentModal = (c: ProjectComponent) => {
  isEditingComp.value = true
  initialEditComponent.value = { ...c }
  componentModalOpen.value = true
}

const handleSaveComponent = async (data: Partial<ProjectComponent>, createFolder: boolean) => {
  savingComp.value = true
  try {
    if (isEditingComp.value && data.id) {
      await updateComponent(data.id, data)
    } else {
      await addComponent(projectId.value, {
        ...data,
        create_folder: createFolder
      })
    }
    componentModalOpen.value = false
    await loadProjectData()
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menyimpan Komponen',
      description: `Failed to save component: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    savingComp.value = false
  }
}

const handleDeleteComp = async () => {
  if (!compToDelete.value) return
  deletingComp.value = true
  try {
    await deleteComponent(compToDelete.value.id)
    compToDelete.value = null
    await loadProjectData()
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

const handleCreateDirOnDisk = async (comp: ProjectComponent) => {
  creatingDirId.value = comp.id
  try {
    const res = await createFolderOnDisk(comp.id)
    toast.add({
      title: 'Folder Berhasil Dibuat',
      description: `Folder berhasil dibuat di filesystem:\n${res.path}`,
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
    await loadProjectData()
  } catch (err: any) {
    toast.add({
      title: 'Gagal Membuat Folder',
      description: `Gagal membuat folder: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    creatingDirId.value = null
  }
}

// Fast Scan Actions
const handleFastScanComponent = async (c: ProjectComponent) => {
  if (scanningCompIds.value.includes(c.id)) return
  activeScanningComp.value = c
  openModalAfterScan.add(c.id)
  const job = await enqueue('fast_scan', { componentId: c.id })
  if (!job) openModalAfterScan.delete(c.id)
}

useJobCompleted(async (job) => {
  if (job.type !== 'fast_scan' || job.project_id !== projectId.value || !job.component_id) return
  await loadProjectData(true)
  const comp = project.value?.components?.find(item => item.id === job.component_id)
  if (!comp?.fast_scan_data) return
  if (activeScanningComp.value?.id === comp.id) activeScanningComp.value = comp
  if (!openModalAfterScan.delete(comp.id)) return
  try {
    activeFastScan.value = JSON.parse(comp.fast_scan_data) as FastScanResult
    fastScanModalOpen.value = true
  } catch {
    // Unreadable scan data: the component card still shows its scan state after reload.
  }
})

const openExistingFastScan = (c: ProjectComponent) => {
  router.push(`/projects/${projectId.value}/apps/${c.id}?tab=fast_scan`)
}

const handleApplyFastScanParams = async () => {
  if (!activeFastScan.value || !activeScanningComp.value) return
  applyingParams.value = true
  try {
    const updated = await applyFastScanParams(activeFastScan.value.component_id, {
      dev_command: activeFastScan.value.detected_params.dev_command,
      build_command: activeFastScan.value.detected_params.build_command,
      port: activeFastScan.value.detected_params.port,
      language: activeFastScan.value.detected_params.language,
      framework: activeFastScan.value.detected_params.framework,
      package_manager: activeFastScan.value.detected_params.package_manager
    })

    if (project.value?.components) {
      const idx = project.value.components.findIndex(item => item.id === updated.id)
      if (idx !== -1) {
        project.value.components[idx] = { ...project.value.components[idx], ...updated }
      }
    }
    activeScanningComp.value = updated
    toast.add({
      title: 'Parameter Berhasil Diterapkan',
      description: 'Parameter berhasil diperbarui ke modul komponen!',
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
    await checkActivePorts()
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

const handleSaveFastScanChanges = async () => {
  if (!activeFastScan.value) return
  savingScanChanges.value = true
  try {
    const updated = await updateComponentFastScan(activeFastScan.value.component_id, activeFastScan.value)
    activeFastScan.value = updated
    if (project.value?.components) {
      const compItem = project.value.components.find(item => item.id === updated.component_id)
      if (compItem) {
        compItem.fast_scan_data = JSON.stringify(updated)
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
</script>

<template>
  <div v-if="loading" class="p-4 sm:p-6 lg:p-8 space-y-6">
    <div class="h-28 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 animate-pulse" />
    <div class="h-96 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 animate-pulse" />
  </div>

  <div v-else-if="project" class="flex-1 flex flex-col lg:flex-row items-stretch w-full min-h-[calc(100vh-4rem)] relative">
    <!-- Main column: project header (always visible) + active tab -->
    <div class="flex-1 min-w-0 w-full p-4 sm:p-6 lg:p-8 space-y-6 lg:mr-72 xl:mr-80">
      <ProjectsDetailProjectDetailHeader
        :project="project"
        @delete-project="deleteProjectOpen = true"
        @start-pipeline="handleStartPipeline"
      />

      <ProjectsDetailProjectOverviewTab
        v-if="activeTab === 'overview'"
        :project="project"
        :profile="profile"
        :loading="loadingProfile"
        :generating="generatingProfile"
        :generating-label="generatingProfileLabel"
        :active-ports-count="activePortsCount"
        @generate="handleGenerateProfile"
        @updated="handleProfileUpdated"
        @switch-tab="activeTab = $event"
      />

      <ProjectsDetailComponentTopologyTab
        v-else-if="activeTab === 'apps'"
        :project="project"
        :profile="profile"
        :port-statuses="portStatuses"
        :active-ports-count="activePortsCount"
        :total-configured-ports="totalConfiguredPorts"
        :scanning-comp-ids="scanningCompIds"
        :creating-dir-id="creatingDirId"
        @open-detail="handleOpenAppDetail"
        @fast-scan="handleFastScanComponent"
        @open-existing-scan="openExistingFastScan"
        @create-dir="handleCreateDirOnDisk"
        @edit="openEditComponentModal"
        @delete="compToDelete = $event"
        @open-add-modal="openAddComponentModal"
        @switch-tab="activeTab = $event"
      />

      <ProjectsDetailBaseKnowledgeTab
        v-else-if="activeTab === 'technical'"
        :project="project"
        :cached-data="cachedBaseKnowledge"
        :loading-data="loadingBaseKnowledge"
        @load="loadProjectBaseKnowledge(false)"
        @refresh="loadProjectBaseKnowledge(true)"
        @switch-tab="activeTab = $event"
      />

      <ProjectsDetailProjectSynergyTab
        v-else-if="activeTab === 'synergy'"
        :project="project"
        :port-statuses="portStatuses"
        :active-ports-count="activePortsCount"
        :total-configured-ports="totalConfiguredPorts"
        @check-ports="checkActivePorts"
        @open-add-modal="openAddComponentModal"
        @open-detail="handleOpenAppDetail"
        @switch-tab="activeTab = $event"
      />

      <ProjectsDetailProjectChatTab
        v-else-if="activeTab === 'ai_chat'"
        :project="project"
        :cached-base-knowledge="cachedBaseKnowledge"
      />

      <ProjectsDetailVisualAuditTab
        v-else-if="activeTab === 'visual_audit'"
        :project="project"
        :port-statuses="portStatuses"
      />
    </div>

    <ProjectsDetailProjectDetailSidebar
      v-model:active-tab="activeTab"
      :project="project"
      :profile="profile"
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

    <ProjectsDetailFastScanModal
      v-model:open="fastScanModalOpen"
      :scan-result="activeFastScan"
      :active-comp="activeScanningComp"
      :scanning-comp-ids="scanningCompIds"
      :applying-params="applyingParams"
      :saving-scan-changes="savingScanChanges"
      @apply-params="handleApplyFastScanParams"
      @save-changes="handleSaveFastScanChanges"
      @re-scan="handleFastScanComponent"
    />

    <ProjectsDetailConfirmActionModal
      v-model:open="deleteProjectOpen"
      title="Hapus Proyek dari Pemantauan"
      confirm-label="Hapus Pantauan"
      :loading="deletingProject"
      @confirm="handleDeleteProject"
    >
      <p>Proyek <strong>{{ project.name }}</strong> beserta {{ project.components?.length || 0 }} aplikasinya akan dihapus dari Vibeasik.</p>
      <p class="text-neutral-500">
        File di disk Anda <strong>tidak</strong> disentuh sama sekali.
      </p>
    </ProjectsDetailConfirmActionModal>

    <ProjectsDetailConfirmActionModal
      v-model:open="deleteCompOpen"
      title="Hapus Aplikasi dari Proyek"
      confirm-label="Hapus Aplikasi"
      :loading="deletingComp"
      @confirm="handleDeleteComp"
    >
      <p>Aplikasi <strong>{{ compToDelete?.name }}</strong> beserta rute & hasil scan-nya akan dihapus dari proyek ini.</p>
      <p class="text-neutral-500">
        Folder di disk tetap aman.
      </p>
    </ProjectsDetailConfirmActionModal>
  </div>
</template>
