import type { ComputedRef } from 'vue'
import type { AIReviewResponse, Diagnostic, Finding, ProjectComponent } from '~/types'

// Diagnostik kode & AI Advisor untuk satu aplikasi.
export function useAppAnalysis(comp: ComputedRef<ProjectComponent | null>) {
  const toast = useToast()
  const { diagnoseComponent, aiReviewComponent } = useApi()

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

  const resetAnalysis = () => {
    componentDiagnostic.value = null
    componentAiReview.value = null
    customAiQuery.value = ''
  }

  return {
    diagnosing, componentDiagnostic, parsedFindings, handleRunComponentDiagnostics,
    reviewing, customAiQuery, componentAiReview, handleRunComponentAiReview, resetAnalysis
  }
}
