import type { ComputedRef } from 'vue'
import type { Diagnostic, Finding, ProjectComponent } from '~/types'

// Diagnostik kode untuk satu aplikasi.
export function useAppAnalysis(comp: ComputedRef<ProjectComponent | null>) {
  const toast = useToast()
  const { diagnoseComponent } = useApi()

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

  const resetAnalysis = () => {
    componentDiagnostic.value = null
  }

  return {
    diagnosing, componentDiagnostic, parsedFindings, handleRunComponentDiagnostics, resetAnalysis
  }
}
