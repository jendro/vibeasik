<script setup lang="ts">
import type { Project, Diagnostic } from '~/types'

definePageMeta({
  title: 'Local Projects Cockpit'
})

const { getProjects, runDiagnostics, diagnoseComponent, deleteProject, holdProject, doneProject, activateProject } = useApi()
const toast = useToast()

const projects = ref<Project[]>([])
const loading = ref(true)
const refreshing = ref(false)
const updatingProjectId = ref<number | null>(null)
const searchQuery = ref('')
const selectedType = ref('all')
const selectedStatus = ref('active')
const diagnosingId = ref<number | null>(null)
const diagReportModal = ref(false)
const activeDiagnostic = ref<Diagnostic | null>(null)
const activeProject = ref<Project | null>(null)
const activeComponentDiagnostics = ref<Record<number, Diagnostic>>({})
const diagnosingStatus = ref<string>('')
const currentlyCheckingCompId = ref<number | null>(null)
const diagnosingAll = ref(false)
const diagnoseAllStatus = ref<string>('')
const newProjectModalOpen = ref(false)
const similarityModalOpen = ref(false)
const deprecateModalOpen = ref(false)
const projectToDeprecate = ref<Project | null>(null)
const mergeModalOpen = ref(false)
const projectToMerge = ref<Project | null>(null)

const loadProjects = async (showSkeleton = false) => {
  if (showSkeleton || projects.value.length === 0) {
    loading.value = true
  } else {
    refreshing.value = true
  }
  try {
    projects.value = await getProjects()
  } catch (err) {
    console.error('Failed to load projects:', err)
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

const updateProjectInList = (updatedProject: Project) => {
  const currentScroll = typeof window !== 'undefined' ? window.scrollY : 0
  const index = projects.value.findIndex(p => p.id === updatedProject.id)
  if (index !== -1) {
    const existing = projects.value[index]
    if (existing) {
      projects.value[index] = {
        ...existing,
        ...updatedProject,
        components: (updatedProject.components && updatedProject.components.length > 0)
          ? updatedProject.components
          : existing.components
      }
    }
  }
  nextTick(() => {
    if (typeof window !== 'undefined' && Math.abs(window.scrollY - currentScroll) > 5) {
      window.scrollTo({ top: currentScroll, behavior: 'instant' })
    }
  })
}

onMounted(() => {
  loadProjects(true)
})

const filteredProjects = computed(() => {
  return projects.value.filter((p) => {
    const q = searchQuery.value.toLowerCase()
    const matchesSearch
      = searchQuery.value === ''
        || p.name.toLowerCase().includes(q)
        || (p.root_path && p.root_path.toLowerCase().includes(q))
        || (p.tags && p.tags.toLowerCase().includes(q))
        || p.components?.some(c =>
          c.name.toLowerCase().includes(q)
          || c.language.toLowerCase().includes(q)
          || (c.absolute_path && c.absolute_path.toLowerCase().includes(q))
          || (c.relative_path && c.relative_path.toLowerCase().includes(q))
        )

    const matchesType = selectedType.value === 'all' || p.project_type === selectedType.value
    const matchesStatus
      = selectedStatus.value === 'all'
        || (selectedStatus.value === 'deprecated' && p.status === 'deprecated')
        || (selectedStatus.value === 'hold' && p.status === 'hold')
        || (selectedStatus.value === 'done' && p.status === 'done')
        || (selectedStatus.value === 'active' && (p.status === 'active' || (!p.status || !['deprecated', 'hold', 'done'].includes(p.status))))

    return matchesSearch && matchesType && matchesStatus
  })
})

const stats = computed(() => {
  const total = projects.value.length
  const active = projects.value.filter(p => p.status === 'active' || (!p.status || !['deprecated', 'hold', 'done'].includes(p.status))).length
  const hold = projects.value.filter(p => p.status === 'hold').length
  const done = projects.value.filter(p => p.status === 'done').length
  const deprecated = projects.value.filter(p => p.status === 'deprecated').length
  const multiFe = projects.value.filter(p => p.project_type === 'multi_fe').length
  const bundles = projects.value.filter(p => p.project_type === 'bundle').length
  const monorepos = projects.value.filter(p => p.project_type === 'monorepo').length
  const avgHealth = total > 0
    ? Math.round(projects.value.reduce((acc, p) => acc + (p.health_score || 0), 0) / total)
    : 0

  return { total, active, hold, done, deprecated, multiFe, bundles, monorepos, avgHealth }
})

const handleOpenMerge = (p: Project) => {
  projectToMerge.value = p
  mergeModalOpen.value = true
}

const handleMerged = async () => {
  await loadProjects()
}

const handleOpenDeprecate = (p: Project) => {
  projectToDeprecate.value = p
  deprecateModalOpen.value = true
}

const handleDeprecateRequest = (projectId: number) => {
  const target = projects.value.find(p => p.id === projectId)
  if (target) {
    handleOpenDeprecate(target)
  }
}

const handleProjectUpdated = (updatedProject: Project) => {
  updateProjectInList(updatedProject)
}

const handleActivate = async (p: Project) => {
  if (updatingProjectId.value === p.id) return
  updatingProjectId.value = p.id
  try {
    const res = await activateProject(p.id)
    toast.add({
      title: 'Status Diaktifkan Kembali',
      description: `Proyek '${p.name}' telah diaktifkan kembali ke daftar aktif.`,
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
    updateProjectInList(res.project || { ...p, status: 'active', deprecated_reason: '' })
  } catch (err: any) {
    toast.add({
      title: 'Gagal Mengaktifkan Proyek',
      description: err.message || 'Terjadi kesalahan',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    updatingProjectId.value = null
  }
}

const handleUndeprecate = handleActivate

const handleHold = async (p: Project) => {
  if (updatingProjectId.value === p.id) return
  updatingProjectId.value = p.id
  try {
    const res = await holdProject(p.id)
    toast.add({
      title: 'Proyek Di-Hold',
      description: `Proyek '${p.name}' berhasil di-hold dan disembunyikan dari daftar aktif.`,
      icon: 'i-lucide-pause-circle',
      color: 'warning'
    })
    updateProjectInList(res.project || { ...p, status: 'hold' })
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menandai Hold',
      description: err.message || 'Terjadi kesalahan saat mengubah status proyek',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    updatingProjectId.value = null
  }
}

const handleDone = async (p: Project) => {
  if (updatingProjectId.value === p.id) return
  updatingProjectId.value = p.id
  try {
    const res = await doneProject(p.id)
    toast.add({
      title: 'Proyek Selesai (Done)',
      description: `Proyek '${p.name}' ditandai selesai dan dalam mode pemeliharaan.`,
      icon: 'i-lucide-check-circle-2',
      color: 'success'
    })
    updateProjectInList(res.project || { ...p, status: 'done' })
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menandai Selesai',
      description: err.message || 'Terjadi kesalahan saat mengubah status proyek',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    updatingProjectId.value = null
  }
}

const handleDiagnose = async (p: Project) => {
  diagnosingId.value = p.id
  activeProject.value = p
  const compReports: Record<number, Diagnostic> = {}

  try {
    const comps = p.components || []
    if (comps.length > 0) {
      for (let i = 0; i < comps.length; i++) {
        const comp = comps[i]
        if (!comp) continue
        currentlyCheckingCompId.value = comp.id
        diagnosingStatus.value = `Mengecek ${comp.name} (${i + 1}/${comps.length})...`

        try {
          const compReport = await diagnoseComponent(comp.id)
          compReports[comp.id] = compReport
          comp.health_score = compReport.score
        } catch (e: any) {
          console.warn(`Gagal memeriksa aplikasi ${comp.name}:`, e)
        }

        // Delay visual singkat agar transisi pengecekan tiap aplikasi terlihat jelas
        await new Promise(resolve => setTimeout(resolve, 200))
      }
    }

    currentlyCheckingCompId.value = null
    diagnosingStatus.value = 'Konsolidasi & Audit Proyek...'

    const report = await runDiagnostics(p.id)
    activeDiagnostic.value = report
    activeComponentDiagnostics.value = compReports
    p.health_score = report.score
    diagReportModal.value = true
  } catch (err: any) {
    toast.add({
      title: 'Diagnostik Gagal',
      description: `Diagnostics failed: ${err.message || 'unknown error'}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    diagnosingId.value = null
    diagnosingStatus.value = ''
    currentlyCheckingCompId.value = null
  }
}

const handleDiagnoseAll = async () => {
  if (projects.value.length === 0 || diagnosingAll.value) return
  diagnosingAll.value = true
  let auditedAppsCount = 0
  let auditedProjectsCount = 0

  const targetProjects = projects.value.filter(p => p.status === 'active' || (!p.status || !['deprecated', 'hold'].includes(p.status)))
  const totalApps = targetProjects.reduce((acc, p) => acc + (p.components?.length || 0), 0)

  try {
    for (const p of targetProjects) {
      diagnosingId.value = p.id
      const comps = p.components || []

      for (let i = 0; i < comps.length; i++) {
        const comp = comps[i]
        if (!comp) continue
        currentlyCheckingCompId.value = comp.id
        auditedAppsCount++
        diagnoseAllStatus.value = `${auditedAppsCount}/${totalApps}`
        diagnosingStatus.value = `Mengecek ${comp.name} (${i + 1}/${comps.length})...`

        try {
          const compReport = await diagnoseComponent(comp.id)
          comp.health_score = compReport.score
        } catch (e: any) {
          console.warn(`Gagal memeriksa aplikasi ${comp.name}:`, e)
        }

        await new Promise(resolve => setTimeout(resolve, 150))
      }

      currentlyCheckingCompId.value = null
      diagnosingStatus.value = 'Konsolidasi...'
      const report = await runDiagnostics(p.id)
      p.health_score = report.score
      auditedProjectsCount++
    }

    toast.add({
      title: 'Audit Seluruh Aplikasi Selesai',
      description: `Berhasil memeriksa ${auditedAppsCount} aplikasi di ${auditedProjectsCount} proyek satu persatu!`,
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
  } catch (err: any) {
    toast.add({
      title: 'Cek Kesehatan Massal Gagal',
      description: err.message || 'Terjadi kesalahan saat memeriksa aplikasi',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    diagnosingAll.value = false
    diagnoseAllStatus.value = ''
    diagnosingId.value = null
    diagnosingStatus.value = ''
    currentlyCheckingCompId.value = null
  }
}

const handleDelete = async (id: number) => {
  if (confirm('Are you sure you want to remove this project from manager? Files on disk will remain untouched.')) {
    const currentScroll = typeof window !== 'undefined' ? window.scrollY : 0
    try {
      await deleteProject(id)
      projects.value = projects.value.filter(p => p.id !== id)
      toast.add({
        title: 'Proyek Dilepas',
        description: 'Proyek berhasil dilepas dari pemantauan lokal.',
        icon: 'i-lucide-check-circle',
        color: 'success'
      })
      nextTick(() => {
        if (typeof window !== 'undefined' && Math.abs(window.scrollY - currentScroll) > 5) {
          window.scrollTo({ top: currentScroll, behavior: 'instant' })
        }
      })
    } catch (err: any) {
      toast.add({
        title: 'Gagal Menghapus Proyek',
        description: `Delete failed: ${err.message}`,
        icon: 'i-lucide-alert-circle',
        color: 'error'
      })
    }
  }
}
</script>

<template>
  <div class="space-y-8">
    <!-- Hero / Header Section -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-2 border-b border-neutral-200 dark:border-neutral-800">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white flex items-center gap-2.5">
          Local Projects Cockpit
          <UBadge color="primary" variant="subtle" size="sm">
            {{ stats.total }} Terdaftar
          </UBadge>
        </h1>
        <p class="text-sm text-neutral-500 dark:text-neutral-400 mt-1">
          Observasi arsitektur, rute API, port aktif, dan dependensi proyek lokal Anda.
        </p>
      </div>

      <div class="flex items-center gap-2 flex-wrap">
        <UButton
          icon="i-lucide-stethoscope"
          color="neutral"
          variant="outline"
          size="sm"
          :loading="diagnosingAll"
          title="Pindai kesehatan seluruh aplikasi di semua proyek satu persatu"
          @click="handleDiagnoseAll"
        >
          {{ diagnoseAllStatus ? `Audit (${diagnoseAllStatus})` : 'Cek Kesehatan Semua' }}
        </UButton>
        <UButton
          icon="i-lucide-refresh-cw"
          color="neutral"
          variant="outline"
          size="sm"
          :loading="refreshing || loading"
          title="Perbarui daftar proyek dan sinkronkan port aktif"
          @click="loadProjects(false)"
        >
          Segarkan Data
        </UButton>
        <UButton
          icon="i-lucide-copy"
          color="neutral"
          variant="outline"
          size="sm"
          title="Pindai potensi duplikasi nama, folder yang sama, atau hubungan BE/FE antar proyek"
          @click="similarityModalOpen = true"
        >
          Pindai Kemiripan
        </UButton>
        <UButton
          icon="i-lucide-plus"
          color="primary"
          variant="solid"
          size="sm"
          title="Daftarkan proyek baru ke daftar pemantauan secara manual"
          @click="newProjectModalOpen = true"
        >
          + Daftarkan Proyek
        </UButton>
      </div>
    </div>

    <!-- Quick Metrics Row Component -->
    <ProjectsProjectMetrics :stats="stats" />

    <!-- Filters & Search Toolbar Component -->
    <ProjectsProjectFilters
      v-model:search-query="searchQuery"
      v-model:selected-type="selectedType"
      v-model:selected-status="selectedStatus"
    />

    <!-- Projects Grid -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div v-for="i in 4" :key="i" class="h-56 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 animate-pulse" />
    </div>

    <div v-else-if="filteredProjects.length === 0" class="text-center py-16 px-4 rounded-xl border border-dashed border-neutral-300 dark:border-neutral-800 bg-white/50 dark:bg-neutral-900/50">
      <div class="w-12 h-12 mx-auto rounded-full bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center text-neutral-400 mb-3">
        <UIcon name="i-lucide-folder-search" class="w-6 h-6" />
      </div>
      <h3 class="text-base font-semibold text-neutral-900 dark:text-white">
        Tidak ada proyek yang sesuai filter
      </h3>
      <p class="text-sm text-neutral-500 max-w-sm mx-auto mt-1 mb-4">
        Coba ubah kata kunci pencarian atau ganti filter status ke "Semua" atau "Deprecated".
      </p>
      <UButton to="/scan" icon="i-lucide-sparkles" color="primary">
        Scan Local Project
      </UButton>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <ProjectsProjectCard
        v-for="project in filteredProjects"
        :key="project.id"
        :project="project"
        :diagnosing="diagnosingId === project.id"
        :diagnosing-status="diagnosingId === project.id ? diagnosingStatus : ''"
        :currently-checking-comp-id="diagnosingId === project.id ? currentlyCheckingCompId : null"
        :status-loading="updatingProjectId === project.id"
        @diagnose="handleDiagnose"
        @delete="handleDelete"
        @deprecate="handleOpenDeprecate"
        @undeprecate="handleUndeprecate"
        @activate="handleActivate"
        @hold="handleHold"
        @done="handleDone"
        @merge="handleOpenMerge"
      />
    </div>

    <!-- Ekosistem Teknologi & Versi Terbaru -->
    <ProjectsTechStackListSection />

    <!-- Quick Diagnostics Modal -->
    <ProjectsDiagnosticModal
      v-model:open="diagReportModal"
      :diagnostic="activeDiagnostic"
      :project="activeProject"
      :component-diagnostics="activeComponentDiagnostics"
    />

    <!-- New Project Modal -->
    <ProjectsNewProjectModal
      v-model:open="newProjectModalOpen"
      @created="() => loadProjects(false)"
    />

    <!-- Merge Project Modal -->
    <ProjectsMergeProjectModal
      v-model:open="mergeModalOpen"
      :source-project="projectToMerge"
      :projects="projects"
      @merged="handleMerged"
    />

    <!-- Similarity Scanner Modal -->
    <ProjectsSimilarityScannerModal
      v-model:open="similarityModalOpen"
      @deprecate-request="handleDeprecateRequest"
      @refresh-projects="() => loadProjects(false)"
    />

    <!-- Deprecate Project Modal -->
    <ProjectsDeprecateModal
      v-model:open="deprecateModalOpen"
      :project="projectToDeprecate"
      @updated="handleProjectUpdated"
    />
  </div>
</template>
