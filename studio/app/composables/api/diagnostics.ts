// Domain API "diagnostics" — dipecah dari useApi.ts tanpa mengubah kontrak HTTP.
import { useApiBase } from './base'

export const useDiagnosticsApi = () => {
  const apiBase = useApiBase()

  const runDiagnostics = async (projectId: number): Promise<Diagnostic> => {
    return await $fetch<Diagnostic>(`${apiBase}/api/projects/${projectId}/diagnose`, {
      method: 'POST'
    })
  }

  const runDeepCheck = async (projectId: number): Promise<DeepCheckLatestResponse> => {
    return await $fetch<DeepCheckLatestResponse>(`${apiBase}/api/projects/${projectId}/deep-check`, {
      method: 'POST'
    })
  }

  const getLatestDeepCheck = async (projectId: number): Promise<DeepCheckLatestResponse> => {
    return await $fetch<DeepCheckLatestResponse>(`${apiBase}/api/projects/${projectId}/deep-check/latest`)
  }

  const getDeepCheckDownloadUrl = (projectId: number, format: 'json' | 'markdown' = 'markdown'): string => {
    return `${apiBase}/api/projects/${projectId}/deep-check/report/download?format=${format}`
  }

  const startRuntimeSimulation = async (projectId: number, options: RuntimeSimOptions = {}): Promise<RuntimeSimResponse> => {
    return await $fetch<RuntimeSimResponse>(`${apiBase}/api/projects/${projectId}/runtime-sim`, {
      method: 'POST',
      body: options
    })
  }

  const getRuntimeSimulation = async (projectId: number): Promise<RuntimeSimResponse> => {
    return await $fetch<RuntimeSimResponse>(`${apiBase}/api/projects/${projectId}/runtime-sim`)
  }

  const cancelRuntimeSimulation = async (projectId: number): Promise<{ success: boolean }> => {
    return await $fetch<{ success: boolean }>(`${apiBase}/api/projects/${projectId}/runtime-sim/cancel`, {
      method: 'POST'
    })
  }

  const getRuntimeSimDownloadUrl = (projectId: number, format: 'json' | 'markdown' | 'zip' | 'vision' = 'markdown'): string => {
    return `${apiBase}/api/projects/${projectId}/runtime-sim/report/download?format=${format}`
  }

  const getRuntimeSimShotUrl = (projectId: number, sessionId: string, file: string): string => {
    return `${apiBase}/api/projects/${projectId}/runtime-sim/shots/${encodeURIComponent(sessionId)}/${encodeURIComponent(file)}`
  }

  const getRuntimeTestProfile = async (projectId: number): Promise<RuntimeTestProfileResponse> => {
    return await $fetch<RuntimeTestProfileResponse>(`${apiBase}/api/projects/${projectId}/runtime-test/profile`)
  }

  const getLatestProjectDiagnostic = async (projectId: number): Promise<DiagnosticReportResponse> => {
    return await $fetch<DiagnosticReportResponse>(`${apiBase}/api/projects/${projectId}/diagnostics/latest`)
  }

  const listProjectDiagnostics = async (projectId: number, opts: { limit?: number, offset?: number, componentId?: number, full?: boolean } = {}): Promise<{ items: Diagnostic[], total: number, limit: number, offset: number }> => {
    return await $fetch(`${apiBase}/api/projects/${projectId}/diagnostics`, {
      query: { limit: opts.limit, offset: opts.offset, component_id: opts.componentId, full: opts.full ? 1 : undefined }
    })
  }

  const resolveDiagnosticIssue = async (projectId: number, finding: Finding, createTodo = true): Promise<ResolutionResponse> => {
    return await $fetch<ResolutionResponse>(`${apiBase}/api/projects/${projectId}/diagnostics/resolve-issue`, {
      method: 'POST',
      body: { finding, create_todo: createTodo }
    })
  }

  /** Selesaikan Grup (Satukan Serupa): semua temuan satu kelompok → satu panggilan LLM, satu todo. */

  const resolveDiagnosticGroup = async (projectId: number, findings: Finding[]): Promise<ResolutionResponse> => {
    return await $fetch<ResolutionResponse>(`${apiBase}/api/projects/${projectId}/diagnostics/resolve-group`, {
      method: 'POST',
      body: { findings, create_todo: true }
    })
  }

  const resolveAllDiagnosticIssues = async (projectId: number): Promise<{ success: boolean, project_id: number, resolved: number, resolutions: ResolutionResponse[] }> => {
    return await $fetch<{ success: boolean, project_id: number, resolved: number, resolutions: ResolutionResponse[] }>(`${apiBase}/api/projects/${projectId}/diagnostics/resolve-all`, {
      method: 'POST'
    })
  }

  const deleteDiagnosticFinding = async (projectId: number, finding: Finding): Promise<{ success: boolean, message: string, findings: Finding[], total_findings: number, score: number }> => {
    return await $fetch<{ success: boolean, message: string, findings: Finding[], total_findings: number, score: number }>(`${apiBase}/api/projects/${projectId}/diagnostics/delete-finding`, {
      method: 'POST',
      body: {
        finding_id: finding.id,
        rule_id: finding.rule_id,
        title: finding.title,
        target_file: finding.target_file,
        line: finding.line,
        finding
      }
    })
  }

  const ignoreDiagnosticFinding = async (projectId: number, finding: Finding, status?: string): Promise<{ success: boolean, message: string, status: string, findings: Finding[], score?: number }> => {
    return await $fetch<{ success: boolean, message: string, status: string, findings: Finding[], score?: number }>(`${apiBase}/api/projects/${projectId}/diagnostics/ignore-finding`, {
      method: 'POST',
      body: {
        finding_id: finding.id,
        rule_id: finding.rule_id,
        title: finding.title,
        target_file: finding.target_file,
        line: finding.line,
        status,
        finding
      }
    })
  }

  const verifyDiagnosticFinding = async (projectId: number, finding: Finding): Promise<FindingVerification> => {
    return await $fetch<FindingVerification>(`${apiBase}/api/projects/${projectId}/diagnostics/verify-finding`, {
      method: 'POST',
      body: { finding }
    })
  }

  // verdict kosong = cabut vonis
  const recordFindingFeedback = async (projectId: number, finding: Finding, input: { verdict: AuditVerdict | '', reason?: string, rule_advice?: string, source?: string, model?: string }): Promise<{ success: boolean, feedback: AuditFeedback | null, findings: Finding[], score: number }> => {
    return await $fetch<{ success: boolean, feedback: AuditFeedback | null, findings: Finding[], score: number }>(`${apiBase}/api/projects/${projectId}/diagnostics/finding-feedback`, {
      method: 'POST',
      body: { ...input, finding }
    })
  }

  const getAuditLearning = async (projectId?: number): Promise<AuditLearningResponse> => {
    return await $fetch<AuditLearningResponse>(`${apiBase}/api/audit-learning`, { query: projectId ? { project_id: projectId } : {} })
  }

  const markAuditRuleUpdated = async (ruleId: string, updated: boolean, projectId?: number): Promise<{ success: boolean, updated: number }> => {
    return await $fetch<{ success: boolean, updated: number }>(`${apiBase}/api/audit-learning/rules/updated`, {
      method: 'POST',
      body: { rule_id: ruleId, updated, project_id: projectId || 0 }
    })
  }

  const deleteAuditFeedback = async (id: number): Promise<{ success: boolean }> => {
    return await $fetch<{ success: boolean }>(`${apiBase}/api/audit-learning/${id}`, { method: 'DELETE' })
  }

  const getAuditLearningExportUrl = (projectId?: number): string => {
    return `${apiBase}/api/audit-learning/export${projectId ? `?project_id=${projectId}` : ''}`
  }

  const runAiReview = async (projectId: number, customQuery?: string): Promise<AIReviewResponse> => {
    return await $fetch<AIReviewResponse>(`${apiBase}/api/projects/${projectId}/ai-review`, {
      method: 'POST',
      body: { custom_query: customQuery }
    })
  }

  const diagnoseComponent = async (componentId: number): Promise<Diagnostic> => {
    return await $fetch<Diagnostic>(`${apiBase}/api/components/${componentId}/diagnose`, {
      method: 'POST'
    })
  }

  const getLatestComponentDiagnostic = async (componentId: number): Promise<{
    diagnostic: Diagnostic
    findings: Finding[]
    optimizations: Optimization[]
    total_findings: number
    score: number
    summary: string
  }> => {
    return await $fetch(`${apiBase}/api/components/${componentId}/diagnostics/latest`)
  }

  const aiReviewComponent = async (componentId: number, customQuery?: string): Promise<AIReviewResponse> => {
    return await $fetch<AIReviewResponse>(`${apiBase}/api/components/${componentId}/ai-review`, {
      method: 'POST',
      body: { custom_query: customQuery }
    })
  }

  // Visual Snapshots & Multimodal AI Visual Audit
  const captureProjectVisual = async (projectId: number, payload?: Partial<VisualCaptureRequest>): Promise<VisualCaptureResult> => {
    return await $fetch<VisualCaptureResult>(`${apiBase}/api/projects/${projectId}/visual-capture`, {
      method: 'POST',
      body: payload || {}
    })
  }

  const auditProjectVisual = async (projectId: number, payload?: Partial<VisualAuditRequest>): Promise<VisualAuditResponse> => {
    return await $fetch<VisualAuditResponse>(`${apiBase}/api/projects/${projectId}/visual-audit`, {
      method: 'POST',
      body: payload || {}
    })
  }

  const getProjectVisualSnapshots = async (projectId: number): Promise<ProjectSnapshotFile[]> => {
    return await $fetch<ProjectSnapshotFile[]>(`${apiBase}/api/projects/${projectId}/visual-snapshots`)
  }

  const deleteProjectVisualSnapshot = async (projectId: number, filename: string): Promise<{ message: string, filename: string }> => {
    return await $fetch<{ message: string, filename: string }>(`${apiBase}/api/projects/${projectId}/visual-snapshots/${encodeURIComponent(filename)}`, {
      method: 'DELETE'
    })
  }

  return {
    runDiagnostics,
    runDeepCheck,
    getLatestDeepCheck,
    getDeepCheckDownloadUrl,
    startRuntimeSimulation,
    getRuntimeSimulation,
    cancelRuntimeSimulation,
    getRuntimeSimDownloadUrl,
    getRuntimeSimShotUrl,
    getRuntimeTestProfile,
    getLatestProjectDiagnostic,
    listProjectDiagnostics,
    resolveDiagnosticIssue,
    resolveDiagnosticGroup,
    resolveAllDiagnosticIssues,
    deleteDiagnosticFinding,
    ignoreDiagnosticFinding,
    verifyDiagnosticFinding,
    recordFindingFeedback,
    getAuditLearning,
    markAuditRuleUpdated,
    deleteAuditFeedback,
    getAuditLearningExportUrl,
    runAiReview,
    diagnoseComponent,
    getLatestComponentDiagnostic,
    aiReviewComponent,
    captureProjectVisual,
    auditProjectVisual,
    getProjectVisualSnapshots,
    deleteProjectVisualSnapshot
  }
}
