import type { Ref } from 'vue'
import type { FastScanResult, Project, ProjectComponent } from '~/types'

// Fast Scan LLM per komponen: antrean, modal blueprint, terapkan parameter, simpan perubahan env.
export function useProjectFastScan(opts: {
  projectId: Ref<number>
  project: Ref<Project | null>
  loadProjectData: (quiet?: boolean) => Promise<void>
  checkActivePorts: () => Promise<void>
}) {
  const { projectId, project, loadProjectData, checkActivePorts } = opts
  const toast = useToast()
  const { updateComponentFastScan, applyFastScanParams } = useApi()

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

  return {
    scanningCompIds, fastScanModalOpen, activeFastScan, activeScanningComp, applyingParams, savingScanChanges,
    handleFastScanComponent, handleApplyFastScanParams, handleSaveFastScanChanges
  }
}
