import type { ComputedRef, Ref } from 'vue'
import type { FastScanResult, Project, ProjectComponent } from '~/types'

// Fast Scan LLM satu aplikasi: job antrean, terapkan parameter, simpan perubahan env/metadata.
export function useAppFastScan(opts: {
  comp: ComputedRef<ProjectComponent | null>
  project: Ref<Project | null>
  parsedScanResult: ComputedRef<FastScanResult | null>
  routeCatalogRef: Ref<any>
  activeTab: Ref<string>
  reload: (quiet?: boolean) => Promise<void>
  checkActivePort: () => Promise<void>
}) {
  const { comp, project, parsedScanResult, routeCatalogRef, activeTab, checkActivePort } = opts
  const toast = useToast()
  const { updateComponentFastScan, applyFastScanParams } = useApi()

  // Fast Scan Modal state
  // Fast Scan runs on the engine's LLM queue; the page stays usable while it waits and runs.
  const fastScanJob = useJobTarget('fast_scan', () => ({ componentId: comp.value?.id }), async () => {
    await opts.reload(true)
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

  return { scanningComp, fastScanModalOpen, activeFastScan, applyingParams, savingScanChanges, handleFastScan, handleApplyFastScanParams, handleSaveFastScanChanges }
}
