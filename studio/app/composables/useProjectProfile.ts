import type { Ref } from 'vue'
import type { Project, ProjectBaseKnowledge, ProjectProfile } from '~/types'

// Profil proyek (tujuan, fitur, roadmap) + cache Base Knowledge untuk halaman detail proyek.
export function useProjectProfile(projectId: Ref<number>, project: Ref<Project | null>) {
  const toast = useToast()
  const { getProjectBaseKnowledge, refreshProjectBaseKnowledge, getProjectProfile } = useApi()

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

  return {
    profile, loadingProfile, generatingProfile, generatingProfileLabel,
    loadProfile, handleGenerateProfile, handleProfileUpdated,
    cachedBaseKnowledge, loadingBaseKnowledge, loadProjectBaseKnowledge
  }
}
