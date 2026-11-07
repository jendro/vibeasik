<script setup lang="ts">
import type { Project, ProjectComponent } from '~/types'

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
  startProjectPipeline
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

// Profil proyek + cache Base Knowledge (state & API di composable)
const {
  profile, loadingProfile, generatingProfile, generatingProfileLabel,
  loadProfile, handleGenerateProfile, handleProfileUpdated,
  cachedBaseKnowledge, loadingBaseKnowledge, loadProjectBaseKnowledge
} = useProjectProfile(projectId, project)

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

// Fast Scan (antrean, modal blueprint, terapkan parameter)
const {
  scanningCompIds, fastScanModalOpen, activeFastScan, activeScanningComp, applyingParams, savingScanChanges,
  handleFastScanComponent, handleApplyFastScanParams, handleSaveFastScanChanges
} = useProjectFastScan({ projectId, project, loadProjectData, checkActivePorts })

const openExistingFastScan = (c: ProjectComponent) => {
  router.push(`/projects/${projectId.value}/apps/${c.id}?tab=fast_scan`)
}

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
  initialEditComponent.value = defaultComponentPreset(role, project.value?.root_path)
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
