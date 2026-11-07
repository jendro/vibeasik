export * from '../types'

export const useApi = () => {
  const config = useRuntimeConfig()
  let apiBase = (config.public.apiBase as string) || 'http://localhost:8765'

  // Dynamic origin adaptation:
  // When served directly by vibeasik local-engine (e.g. http://localhost:8765 or LAN IP:port),
  // automatically use window.location.origin so all API calls match current host and port.
  // When running in the Nuxt dev server (any port), adapt LAN IP while targeting the engine port.
  if (import.meta.client && typeof window !== 'undefined') {
    if (!import.meta.dev) {
      apiBase = window.location.origin
    } else {
      try {
        const parsed = new URL(apiBase)
        if (
          (parsed.hostname === 'localhost' || parsed.hostname === '127.0.0.1')
          && window.location.hostname !== 'localhost'
          && window.location.hostname !== '127.0.0.1'
        ) {
          parsed.hostname = window.location.hostname
          apiBase = parsed.origin
        }
      } catch {
        // Keep original apiBase if URL parsing fails
      }
    }
  }

  const getProjects = async (params?: { type?: string, status?: string }): Promise<Project[]> => {
    const query = new URLSearchParams()
    if (params?.type) query.append('type', params.type)
    if (params?.status) query.append('status', params.status)
    const url = `${apiBase}/api/projects${query.toString() ? '?' + query.toString() : ''}`
    return await $fetch<Project[]>(url)
  }

  const getProject = async (id: number | string): Promise<Project> => {
    return await $fetch<Project>(`${apiBase}/api/projects/${id}`)
  }

  const createProject = async (data: Partial<Project> & { create_directories?: boolean }): Promise<Project> => {
    return await $fetch<Project>(`${apiBase}/api/projects`, {
      method: 'POST',
      body: data
    })
  }

  const updateProject = async (id: number, data: Partial<Project>): Promise<Project> => {
    return await $fetch<Project>(`${apiBase}/api/projects/${id}`, {
      method: 'PUT',
      body: data
    })
  }

  const deleteProject = async (id: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/projects/${id}`, {
      method: 'DELETE'
    })
  }

  const mergeProjects = async (sourceId: number, targetId: number): Promise<{ message: string, target_project: Project }> => {
    try {
      return await $fetch<{ message: string, target_project: Project }>(`${apiBase}/api/projects/${sourceId}/merge`, {
        method: 'POST',
        body: { target_project_id: targetId }
      })
    } catch {
      // Fallback if backend server endpoint is temporarily not restarted
      const source = await getProject(sourceId)
      if (source && source.components && source.components.length > 0) {
        for (const comp of source.components) {
          await addComponent(targetId, {
            name: comp.name,
            role: comp.role,
            relative_path: comp.relative_path,
            absolute_path: comp.absolute_path,
            language: comp.language,
            framework: comp.framework,
            package_manager: comp.package_manager,
            dev_command: comp.dev_command,
            build_command: comp.build_command,
            port: comp.port,
            health_url: comp.health_url,
            env_file: comp.env_file,
            env_summary: comp.env_summary,
            fast_scan_data: comp.fast_scan_data
          })
          await deleteComponent(comp.id).catch(() => {})
        }
      }
      await deleteProject(sourceId)
      await generateProjectSummary(targetId).catch(() => {})
      const updatedTarget = await getProject(targetId)
      return {
        message: 'Proyek berhasil digabungkan',
        target_project: updatedTarget
      }
    }
  }

  const generateProjectSummary = async (projectId: number): Promise<{
    message: string
    short_description: string
    description: string
    project: Project
  }> => {
    return await $fetch<{
      message: string
      short_description: string
      description: string
      project: Project
    }>(`${apiBase}/api/projects/${projectId}/summary`, {
      method: 'POST'
    })
  }

  const linkWorkspace = async (projectId: number, componentId?: number): Promise<{
    message: string
    project_id: number
    files_created: string[]
    identifier: any
  }> => {
    return await $fetch<{
      message: string
      project_id: number
      files_created: string[]
      identifier: any
    }>(`${apiBase}/api/projects/${projectId}/link-workspace`, {
      method: 'POST',
      body: { component_id: componentId }
    })
  }

  const identifyProject = async (path?: string): Promise<{
    matched: boolean
    match_source: string
    identifier_file_path?: string
    project?: Project
    current_component?: ProjectComponent
    identifier?: any
    active_todos?: Todo[]
    todos_count: number
    pending_todos_count: number
    suggested_todos_count: number
    ai_agent_instructions?: string
  }> => {
    const query = path ? `?path=${encodeURIComponent(path)}` : ''
    return await $fetch(`${apiBase}/api/projects/identify${query}`)
  }

  const addComponent = async (projectId: number, comp: Partial<ProjectComponent> & { create_folder?: boolean }): Promise<ProjectComponent> => {
    return await $fetch<ProjectComponent>(`${apiBase}/api/projects/${projectId}/components`, {
      method: 'POST',
      body: comp
    })
  }

  const getComponent = async (componentId: number): Promise<ProjectComponent> => {
    return await $fetch<ProjectComponent>(`${apiBase}/api/components/${componentId}`)
  }

  const createFolderOnDisk = async (componentId: number): Promise<{ message: string, path: string }> => {
    return await $fetch<{ message: string, path: string }>(`${apiBase}/api/components/${componentId}/create-dir`, {
      method: 'POST'
    })
  }

  const updateComponent = async (componentId: number, comp: Partial<ProjectComponent>): Promise<ProjectComponent> => {
    return await $fetch<ProjectComponent>(`${apiBase}/api/components/${componentId}`, {
      method: 'PUT',
      body: comp
    })
  }

  const deleteComponent = async (componentId: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/components/${componentId}`, {
      method: 'DELETE'
    })
  }

  const getAdapters = async (): Promise<AdaptersCatalogSummary> => {
    return await $fetch<AdaptersCatalogSummary>(`${apiBase}/api/adapters`)
  }

  const scanDirectory = async (path: string): Promise<ScanResult> => {
    return await $fetch<ScanResult>(`${apiBase}/api/scan`, {
      method: 'POST',
      body: { path }
    })
  }

  const groupScanComponents = async (data: { root_path: string, components: any[] }): Promise<ScanGroupingResult> => {
    return await $fetch<ScanGroupingResult>(`${apiBase}/api/scan/group-ai`, {
      method: 'POST',
      body: data
    })
  }

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

  const getLLMWeighing = async (limit = 40): Promise<LLMWeighingReport> =>
    await $fetch<LLMWeighingReport>(`${apiBase}/api/settings/llm/weighing`, { query: { limit } })

  // Pusat Pengetahuan (satu pintu patokan & pengetahuan pekerja AI)
  const getKnowledge = async (): Promise<KnowledgeOverview> => await $fetch<KnowledgeOverview>(`${apiBase}/api/knowledge`)
  const getKnowledgeGraph = async (limit = 5000): Promise<KnowledgeGraph> => await $fetch<KnowledgeGraph>(`${apiBase}/api/knowledge/graph`, { query: { limit } })
  const listKnowledgeNodes = async (query: { q?: string, domain?: string, kind?: string, source?: string, cluster_id?: number, status?: string, limit?: number }): Promise<KnowledgeHit[]> =>
    await $fetch<KnowledgeHit[]>(`${apiBase}/api/knowledge/nodes`, { query })
  const getKnowledgeNode = async (id: number): Promise<KnowledgeDetail> => await $fetch<KnowledgeDetail>(`${apiBase}/api/knowledge/nodes/${id}`)
  const teachKnowledge = async (body: KnowledgeNoteInput): Promise<{ node: KnowledgeNode, merged: boolean }> =>
    await $fetch<{ node: KnowledgeNode, merged: boolean }>(`${apiBase}/api/knowledge/nodes`, { method: 'POST', body })
  const knowledgeFeedback = async (id: number, helpful: boolean): Promise<KnowledgeNode> =>
    await $fetch<KnowledgeNode>(`${apiBase}/api/knowledge/nodes/${id}/feedback`, { method: 'POST', body: { helpful } })
  const retireKnowledgeNode = async (id: number, reason?: string): Promise<{ ok: boolean }> =>
    await $fetch<{ ok: boolean }>(`${apiBase}/api/knowledge/nodes/${id}`, { method: 'DELETE', query: reason ? { reason } : {} })
  const recallKnowledge = async (body: { text: string, domains?: string[], employee_id?: number, limit?: number }): Promise<KnowledgeRecallResult> =>
    await $fetch<KnowledgeRecallResult>(`${apiBase}/api/knowledge/recall`, { method: 'POST', body })
  const previewKnowledgePrompt = async (body: { text: string, employee_id?: number, project_id?: number }): Promise<KnowledgePromptPreview> =>
    await $fetch<KnowledgePromptPreview>(`${apiBase}/api/knowledge/preview`, { method: 'POST', body })
  const curateKnowledge = async (): Promise<KnowledgeCurateReport> => await $fetch<KnowledgeCurateReport>(`${apiBase}/api/knowledge/curate`, { method: 'POST' })
  const reportKnowledgeGap = async (body: { topic: string, domain?: string, sample?: string }): Promise<KnowledgeGap> =>
    await $fetch<KnowledgeGap>(`${apiBase}/api/knowledge/gaps`, { method: 'POST', body })
  const dismissKnowledgeGap = async (id: number, note?: string): Promise<{ ok: boolean }> =>
    await $fetch<{ ok: boolean }>(`${apiBase}/api/knowledge/gaps/${id}/dismiss`, { method: 'POST', query: note ? { note } : {} })

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

  const checkPorts = async (ports: number[]): Promise<Record<string, boolean>> => {
    if (!ports.length) return {}
    const portStr = ports.filter(p => p > 0).join(',')
    const res = await $fetch<{ ports: Record<string, boolean> }>(`${apiBase}/api/system/ports?ports=${portStr}`)
    return res.ports || {}
  }

  const callMcp = async (method: string, params?: any): Promise<any> => {
    return await $fetch(`${apiBase}/mcp`, {
      method: 'POST',
      body: {
        jsonrpc: '2.0',
        id: Date.now(),
        method,
        params
      }
    })
  }

  const getMCPInfo = async (): Promise<any> => {
    return await $fetch(`${apiBase}/api/mcp/info`)
  }

  const getLLMSettings = async (): Promise<LLMSettings> => {
    return await $fetch<LLMSettings>(`${apiBase}/api/settings/llm`)
  }

  const updateLLMSettings = async (data: Partial<LLMSettings>): Promise<LLMSettings> => {
    return await $fetch<LLMSettings>(`${apiBase}/api/settings/llm`, {
      method: 'PUT',
      body: data
    })
  }

  const getLLMConfigs = async (provider?: string): Promise<LLMConfig[]> => {
    const query = provider ? `?provider=${encodeURIComponent(provider)}` : ''
    return await $fetch<LLMConfig[]>(`${apiBase}/api/settings/llm/configs${query}`)
  }

  const createLLMConfig = async (data: Partial<LLMConfig>): Promise<LLMConfig> => {
    return await $fetch<LLMConfig>(`${apiBase}/api/settings/llm/configs`, {
      method: 'POST',
      body: data
    })
  }

  const updateLLMConfig = async (id: number, data: Partial<LLMConfig>): Promise<LLMConfig> => {
    return await $fetch<LLMConfig>(`${apiBase}/api/settings/llm/configs/${id}`, {
      method: 'PUT',
      body: data
    })
  }

  /** Tandai satu model konfigurasi menerima gambar (vision) atau tidak. */
  const setLLMModelVision = async (id: number, model: string, vision: boolean): Promise<LLMConfig> => {
    return await $fetch<LLMConfig>(`${apiBase}/api/settings/llm/configs/${id}/vision`, {
      method: 'PUT',
      body: { model, vision }
    })
  }

  /** Beri alias tampilan untuk satu model konfigurasi (alias kosong = hapus alias). */
  const setLLMModelAlias = async (id: number, model: string, alias: string): Promise<LLMConfig> => {
    return await $fetch<LLMConfig>(`${apiBase}/api/settings/llm/configs/${id}/alias`, {
      method: 'PUT',
      body: { model, alias }
    })
  }

  const deleteLLMConfig = async (id: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/settings/llm/configs/${id}`, {
      method: 'DELETE'
    })
  }

  const activateLLMConfig = async (id: number): Promise<{ message: string, config: LLMConfig }> => {
    return await $fetch<{ message: string, config: LLMConfig }>(`${apiBase}/api/settings/llm/configs/${id}/activate`, {
      method: 'POST'
    })
  }

  const setActiveProvider = async (provider: 'system_one' | 'standard_llm'): Promise<{ message: string, active_provider: string }> => {
    return await $fetch<{ message: string, active_provider: string }>(`${apiBase}/api/settings/llm/active-provider`, {
      method: 'POST',
      body: { provider }
    })
  }

  const selectActiveModel = async (model: string): Promise<{ message: string, selected_model: string }> => {
    return await $fetch<{ message: string, selected_model: string }>(`${apiBase}/api/settings/llm/select-model`, {
      method: 'POST',
      body: { model }
    })
  }

  const testLLMConnection = async (payload: { config_id?: number, provider?: string, url?: string, model?: string, key?: string }): Promise<LLMTestResult> => {
    return await $fetch<LLMTestResult>(`${apiBase}/api/settings/llm/test`, {
      method: 'POST',
      body: payload
    })
  }

  const browseDirectories = async (path?: string, search?: string): Promise<BrowseDirectoryResponse> => {
    const query = new URLSearchParams()
    if (path) query.append('path', path)
    if (search) query.append('search', search)
    const url = `${apiBase}/api/system/directories${query.toString() ? '?' + query.toString() : ''}`
    return await $fetch<BrowseDirectoryResponse>(url)
  }

  const fastScanComponent = async (componentId: number): Promise<FastScanResult> => {
    return await $fetch<FastScanResult>(`${apiBase}/api/components/${componentId}/fast-scan`, {
      method: 'POST'
    })
  }

  const applyFastScanParams = async (componentId: number, params: Partial<ProjectComponent>): Promise<ProjectComponent> => {
    return await $fetch<ProjectComponent>(`${apiBase}/api/components/${componentId}/apply-scan-params`, {
      method: 'POST',
      body: params
    })
  }

  const fastScanProject = async (projectId: number): Promise<ProjectFastScanResponse> => {
    return await $fetch<ProjectFastScanResponse>(`${apiBase}/api/projects/${projectId}/fast-scan`, {
      method: 'POST'
    })
  }

  const updateComponentFastScan = async (componentId: number, data: FastScanResult): Promise<FastScanResult> => {
    return await $fetch<FastScanResult>(`${apiBase}/api/components/${componentId}/fast-scan`, {
      method: 'PUT',
      body: data
    })
  }

  const getComponentRoutes = async (componentId: number): Promise<ComponentRoute[]> => {
    return await $fetch<ComponentRoute[]>(`${apiBase}/api/components/${componentId}/routes`)
  }

  const syncComponentRoutes = async (componentId: number): Promise<RouteSyncResult> => {
    return await $fetch<RouteSyncResult>(`${apiBase}/api/components/${componentId}/routes/sync`, {
      method: 'POST'
    })
  }

  const enrichComponentRoutes = async (componentId: number): Promise<{ message: string, count: number, routes: ComponentRoute[] }> => {
    return await $fetch<{ message: string, count: number, routes: ComponentRoute[] }>(`${apiBase}/api/components/${componentId}/routes/enrich`, {
      method: 'POST'
    })
  }

  const updateComponentRoute = async (componentId: number, routeId: number, data: Partial<ComponentRoute>): Promise<ComponentRoute> => {
    return await $fetch<ComponentRoute>(`${apiBase}/api/components/${componentId}/routes/${routeId}`, {
      method: 'PUT',
      body: data
    })
  }

  const deleteComponentRoute = async (componentId: number, routeId: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/components/${componentId}/routes/${routeId}`, {
      method: 'DELETE'
    })
  }

  const getSecurityStatus = async (): Promise<SecurityStatus> => {
    return await $fetch<SecurityStatus>(`${apiBase}/api/security/status`)
  }

  const getCensoredKeywords = async (): Promise<CensoredKeywordsResponse> => {
    return await $fetch<CensoredKeywordsResponse>(`${apiBase}/api/security/censored-keywords`)
  }

  const createCensoredKeyword = async (payload: { keyword: string, category?: string, description?: string, is_regex?: boolean, is_active?: boolean }): Promise<CensoredKeyword> => {
    return await $fetch<CensoredKeyword>(`${apiBase}/api/security/censored-keywords`, {
      method: 'POST',
      body: payload
    })
  }

  const updateCensoredKeyword = async (id: number, payload: Partial<CensoredKeyword>): Promise<CensoredKeyword> => {
    return await $fetch<CensoredKeyword>(`${apiBase}/api/security/censored-keywords/${id}`, {
      method: 'PUT',
      body: payload
    })
  }

  const deleteCensoredKeyword = async (id: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/security/censored-keywords/${id}`, {
      method: 'DELETE'
    })
  }

  const testCensorPayload = async (text: string): Promise<TestCensorResponse> => {
    return await $fetch<TestCensorResponse>(`${apiBase}/api/security/test-censor`, {
      method: 'POST',
      body: { text }
    })
  }

  // Background Job Queue & Pipeline Monitor
  const getQueueJobs = async (status?: string, limit = 50): Promise<QueueResponse> => {
    const query = new URLSearchParams()
    if (status) query.append('status', status)
    if (limit) query.append('limit', String(limit))
    const url = `${apiBase}/api/queue${query.toString() ? '?' + query.toString() : ''}`
    return await $fetch<QueueResponse>(url)
  }

  // Kantor Agent: pekerja AI aktif (agent coding, CLI lokal, request LLM) + antrean & todo berjalan
  const getAgentOffice = async (params?: { project_id?: number }): Promise<AgentOfficeSnapshot> => {
    return await $fetch<AgentOfficeSnapshot>(`${apiBase}/api/agents/office`, {
      query: params?.project_id ? { project_id: params.project_id } : undefined
    })
  }

  // Rekap progres harian: commit, todo dibuat/selesai, run agent (n hari terakhir termasuk hari ini)
  const getAgentRecap = async (days = 14): Promise<AgentRecap> => {
    return await $fetch<AgentRecap>(`${apiBase}/api/agents/recap`, { query: { days } })
  }

  // Pegawai vibeasik: agent berkepribadian & berpengetahuan sendiri, review QA, obrolan
  const getStaff = async (): Promise<StaffOverview> => await $fetch<StaffOverview>(`${apiBase}/api/staff`)
  const getEmployee = async (id: number): Promise<EmployeeDetail> => await $fetch<EmployeeDetail>(`${apiBase}/api/staff/${id}`)
  const hireEmployee = async (role: string): Promise<Employee> => await $fetch<Employee>(`${apiBase}/api/staff`, { method: 'POST', body: { role } })
  const updateEmployee = async (id: number, body: Partial<Employee>): Promise<Employee> => await $fetch<Employee>(`${apiBase}/api/staff/${id}`, { method: 'PUT', body })
  const resignEmployee = async (id: number) => await $fetch(`${apiBase}/api/staff/${id}`, { method: 'DELETE' })
  const teachEmployee = async (id: number, body: { topic: string, content: string, kind?: string, weight?: number }): Promise<EmployeeKnowledge> =>
    await $fetch<EmployeeKnowledge>(`${apiBase}/api/staff/${id}/knowledge`, { method: 'POST', body })
  const consolidateKnowledge = async (id: number): Promise<{ merged_count: number }> =>
    await $fetch<{ merged_count: number }>(`${apiBase}/api/staff/${id}/knowledge/consolidate`, { method: 'POST' })
  const forgetKnowledge = async (id: number, kid: number, reason?: string) =>
    await $fetch(`${apiBase}/api/staff/${id}/knowledge/${kid}`, { method: 'DELETE', query: { reason } })
  const getEmployeeChat = async (id: number): Promise<EmployeeChatMessage[]> => await $fetch<EmployeeChatMessage[]>(`${apiBase}/api/staff/${id}/chat`)
  const sendEmployeeChat = async (id: number, message: string): Promise<{ message: EmployeeChatMessage, job_id: string, warning?: string }> =>
    await $fetch(`${apiBase}/api/staff/${id}/chat`, { method: 'POST', body: { message } })
  const clearEmployeeChat = async (id: number) => await $fetch(`${apiBase}/api/staff/${id}/chat`, { method: 'DELETE' })
  const requestEmployeeReview = async (id: number): Promise<{ job_id: string }> => await $fetch(`${apiBase}/api/staff/${id}/review`, { method: 'POST' })
  const getStaffTeams = async (): Promise<StaffTeam[]> => await $fetch<StaffTeam[]>(`${apiBase}/api/staff/teams`)
  const setStaffLeader = async (id: number, lead: boolean): Promise<StaffTeam[]> =>
    await $fetch<StaffTeam[]>(`${apiBase}/api/staff/${id}/leader`, { method: 'PUT', body: { lead } })
  const updateStaffSettings = async (body: StaffSettings): Promise<StaffSettings> => await $fetch<StaffSettings>(`${apiBase}/api/staff/settings`, { method: 'PUT', body })
  // Eksekutif proyek (CEO, CTO, Product Owner)
  const getExecutives = async (query: { project_id?: number } = {}): Promise<{ executives: ProjectExecutive[], roles: ExecRole[] }> =>
    await $fetch<{ executives: ProjectExecutive[], roles: ExecRole[] }>(`${apiBase}/api/staff/executives`, { query })
  const getExecutive = async (id: number): Promise<ExecutiveDetail> =>
    await $fetch<ExecutiveDetail>(`${apiBase}/api/staff/executives/${id}`)
  const provisionExecutives = async (projectId: number): Promise<{ project_id: number, project_name: string, executives: ProjectExecutive[] }> =>
    await $fetch<{ project_id: number, project_name: string, executives: ProjectExecutive[] }>(`${apiBase}/api/staff/executives/provision/${projectId}`, { method: 'POST' })
  // Agenda pribadi pegawai (terpisah dari todo proyek) & pool LLM
  const getAgendas = async (query: { employee_id?: number, status?: string, limit?: number } = {}): Promise<AgendaOverview> =>
    await $fetch<AgendaOverview>(`${apiBase}/api/staff/agenda`, { query })
  const addAgenda = async (employeeId: number, body: AgendaInput): Promise<EmployeeAgenda> =>
    await $fetch<EmployeeAgenda>(`${apiBase}/api/staff/${employeeId}/agenda`, { method: 'POST', body })
  const startAgenda = async (id: number): Promise<EmployeeAgenda> => await $fetch<EmployeeAgenda>(`${apiBase}/api/staff/agenda/${id}/start`, { method: 'POST' })
  const dropAgenda = async (id: number) => await $fetch(`${apiBase}/api/staff/agenda/${id}`, { method: 'DELETE' })
  // Obrolan santai antar pegawai (model obrolan ringan)
  const getHR = async (query: { employee_id?: number, kind?: string, status?: string, limit?: number } = {}): Promise<HROverview> =>
    await $fetch<HROverview>(`${apiBase}/api/staff/hr`, { query })
  const evaluateHR = async (): Promise<{ created: EmployeeHRAction[] }> => await $fetch(`${apiBase}/api/staff/hr/evaluate`, { method: 'POST' })
  const decideHR = async (id: number, approve: boolean, note = ''): Promise<EmployeeHRAction> =>
    await $fetch<EmployeeHRAction>(`${apiBase}/api/staff/hr/${id}/${approve ? 'approve' : 'reject'}`, { method: 'POST', body: { note } })
  const getTurnovers = async (query: { limit?: number } = {}): Promise<{ turnovers: EmployeeTurnover[] }> =>
    await $fetch<{ turnovers: EmployeeTurnover[] }>(`${apiBase}/api/staff/turnovers`, { query })
  // Antrean perbaikan tertunda (todo selesai bernilai kurang, dibuka lagi setelah skill naik)
  const getRevisits = async (query: { status?: string, cause?: string, employee_id?: number, limit?: number } = {}): Promise<RevisitOverview> =>
    await $fetch<RevisitOverview>(`${apiBase}/api/staff/revisits`, { query })
  const processRevisits = async (): Promise<RevisitReport> => await $fetch<RevisitReport>(`${apiBase}/api/staff/revisits/process`, { method: 'POST' })
  const revisitAction = async (id: number, action: 'reopen' | 'dismiss', note = ''): Promise<EmployeeRevisit> =>
    await $fetch<EmployeeRevisit>(`${apiBase}/api/staff/revisits/${id}/${action}`, { method: 'POST', body: { note } })
  const getConversations = async (query: { employee_id?: number, limit?: number } = {}): Promise<ConversationOverview> =>
    await $fetch<ConversationOverview>(`${apiBase}/api/staff/conversations`, { query })
  const startConversation = async (aId: number, bId: number): Promise<EmployeeConversation> =>
    await $fetch<EmployeeConversation>(`${apiBase}/api/staff/conversations`, { method: 'POST', body: { a_id: aId, b_id: bId } })
  const getLLMPool = async (): Promise<LLMPoolResponse> => await $fetch<LLMPoolResponse>(`${apiBase}/api/llm/pool`)
  const resetLLMProvider = async (id: number) => await $fetch(`${apiBase}/api/llm/providers/${id}/reset`, { method: 'POST' })
  const assignTodoEmployee = async (todoId: number, employeeId?: number | null) =>
    await $fetch(`${apiBase}/api/todos/${todoId}/employee`, { method: 'PUT', body: employeeId === undefined ? {} : { employee_id: employeeId ?? 0 } })

  // Agent Project Manager: rapat perencanaan dengan spesialis memakai LLM berkecerdasan tinggi
  const getPMOverview = async (): Promise<PlannerOverview> => {
    return await $fetch<PlannerOverview>(`${apiBase}/api/pm/overview`)
  }

  const updatePMSettings = async (body: PlannerSettings): Promise<PlannerOverview> => {
    return await $fetch<PlannerOverview>(`${apiBase}/api/pm/settings`, { method: 'PUT', body })
  }

  const requestPMPlan = async (body: { project_id: number, note?: string, todo_id?: number }): Promise<{ session: PlanningSession, job_id: string }> => {
    return await $fetch(`${apiBase}/api/pm/request`, { method: 'POST', body: { ...body, source: 'manual' } })
  }

  const getPMSessions = async (projectId?: number, limit = 20): Promise<PlanningSession[]> => {
    return await $fetch<PlanningSession[]>(`${apiBase}/api/pm/sessions`, { query: { project_id: projectId || undefined, limit } })
  }

  const getPMSession = async (id: number): Promise<PlanningSession> => {
    return await $fetch<PlanningSession>(`${apiBase}/api/pm/sessions/${id}`)
  }

  // Orkestrator agent: profil worker (agent + model + effort + akun) & run yang dikirim ke laptop
  const getOrchestratorOverview = async (): Promise<OrchestratorOverview> => {
    return await $fetch<OrchestratorOverview>(`${apiBase}/api/orchestrator/overview`)
  }

  const getOrchestratorAccounts = async (): Promise<{ agy: AgyAccount[], claude: ClaudeAccount[] }> => {
    return await $fetch(`${apiBase}/api/orchestrator/accounts`)
  }

  const updateOrchestratorSettings = async (body: OrchestratorSettings): Promise<OrchestratorSettings> => {
    return await $fetch<OrchestratorSettings>(`${apiBase}/api/orchestrator/settings`, { method: 'PUT', body })
  }

  const saveAgentProfile = async (body: AgentProfileForm, id?: number): Promise<AgentProfile> => {
    const url = id ? `${apiBase}/api/orchestrator/profiles/${id}` : `${apiBase}/api/orchestrator/profiles`
    return await $fetch<AgentProfile>(url, { method: id ? 'PUT' : 'POST', body })
  }

  const deleteAgentProfile = async (id: number): Promise<{ success: boolean }> => {
    return await $fetch(`${apiBase}/api/orchestrator/profiles/${id}`, { method: 'DELETE' })
  }

  const resetAgentProfileCooldown = async (id: number): Promise<{ success: boolean }> => {
    return await $fetch(`${apiBase}/api/orchestrator/profiles/${id}/reset-cooldown`, { method: 'POST' })
  }

  const importAgentProfiles = async (body: { source: 'agy' | 'claude' | 'opencode' | 'cline', model?: string, effort?: string, daily_limit?: number }): Promise<{ created: AgentProfile[], count: number }> => {
    return await $fetch(`${apiBase}/api/orchestrator/profiles/import`, { method: 'POST', body })
  }

  const getAgentRuns = async (params?: { status?: string, todo_id?: number, limit?: number }): Promise<{ runs: AgentRun[] }> => {
    const query = new URLSearchParams()
    if (params?.status) query.append('status', params.status)
    if (params?.todo_id) query.append('todo_id', String(params.todo_id))
    if (params?.limit) query.append('limit', String(params.limit))
    return await $fetch(`${apiBase}/api/orchestrator/runs${query.toString() ? '?' + query.toString() : ''}`)
  }

  const getAgentRun = async (id: number): Promise<{ run: AgentRun, live_lines?: string[] }> => {
    return await $fetch(`${apiBase}/api/orchestrator/runs/${id}`)
  }

  const dispatchAgentRun = async (body: AgentDispatchRequest): Promise<{ run: AgentRun }> => {
    return await $fetch(`${apiBase}/api/orchestrator/runs`, { method: 'POST', body })
  }

  const dispatchAgentBatch = async (body: AgentBatchDispatchRequest): Promise<{ created: AgentBatchItem[], skipped: AgentBatchItem[] }> => {
    return await $fetch(`${apiBase}/api/orchestrator/dispatch-batch`, { method: 'POST', body })
  }

  const previewAgentRoute = async (body: AgentDispatchRequest): Promise<{ decision: RouteDecision | null, message?: string }> => {
    return await $fetch(`${apiBase}/api/orchestrator/route-preview`, { method: 'POST', body })
  }

  const cancelAgentRun = async (id: number): Promise<{ success: boolean }> => {
    return await $fetch(`${apiBase}/api/orchestrator/runs/${id}/cancel`, { method: 'POST' })
  }

  const retryAgentRun = async (id: number): Promise<{ run: AgentRun }> => {
    return await $fetch(`${apiBase}/api/orchestrator/runs/${id}/retry`, { method: 'POST' })
  }

  const deleteAgentRun = async (id: number): Promise<{ success: boolean }> => {
    return await $fetch(`${apiBase}/api/orchestrator/runs/${id}`, { method: 'DELETE' })
  }

  const getQueueStats = async (): Promise<QueueStats> => {
    return await $fetch<QueueStats>(`${apiBase}/api/queue/stats`)
  }

  const enqueueQueueJob = async (req: EnqueueJobRequest): Promise<EnqueueJobResponse> => {
    return await $fetch<EnqueueJobResponse>(`${apiBase}/api/queue/enqueue`, {
      method: 'POST',
      body: req
    })
  }

  const deleteQueueJob = async (id: string): Promise<{ message: string, id: string }> => {
    return await $fetch<{ message: string, id: string }>(`${apiBase}/api/queue/${id}`, {
      method: 'DELETE'
    })
  }

  const clearQueueHistory = async (): Promise<{ cleared: number }> => {
    return await $fetch<{ cleared: number }>(`${apiBase}/api/queue/clear`, {
      method: 'DELETE'
    })
  }

  const startProjectPipeline = async (projectId: number): Promise<{ message: string, project_id: number, jobs: string[] }> => {
    return await $fetch<{ message: string, project_id: number, jobs: string[] }>(`${apiBase}/api/projects/${projectId}/start-pipeline`, {
      method: 'POST'
    })
  }

  const cancelProjectPipeline = async (projectId: number): Promise<{ cancelled: number }> => {
    return await $fetch<{ cancelled: number }>(`${apiBase}/api/projects/${projectId}/cancel-pipeline`, {
      method: 'POST'
    })
  }

  const getTodos = async (params?: { project_id?: number, component_id?: number, level?: string, category?: string, status?: string, source?: string, is_ai_suggestion?: boolean, limit?: number, offset?: number }): Promise<Todo[]> => {
    const query = new URLSearchParams()
    if (params?.project_id) query.append('project_id', String(params.project_id))
    if (params?.component_id) query.append('component_id', String(params.component_id))
    if (params?.level) query.append('level', params.level)
    if (params?.category) query.append('category', params.category)
    if (params?.status) query.append('status', params.status)
    if (params?.source) query.append('source', params.source)
    if (params?.is_ai_suggestion !== undefined) query.append('is_ai_suggestion', String(params.is_ai_suggestion))
    if (params?.limit) query.append('limit', String(params.limit))
    if (params?.offset !== undefined) query.append('offset', String(params.offset))
    const url = `${apiBase}/api/todos${query.toString() ? '?' + query.toString() : ''}`
    return await $fetch<Todo[]>(url)
  }

  const getTodo = async (id: number): Promise<Todo> => {
    return await $fetch<Todo>(`${apiBase}/api/todos/${id}`)
  }

  const createTodo = async (todo: Partial<Todo>): Promise<Todo> => {
    return await $fetch<Todo>(`${apiBase}/api/todos`, {
      method: 'POST',
      body: todo
    })
  }

  const updateTodo = async (id: number, todo: Partial<Todo>): Promise<Todo> => {
    return await $fetch<Todo>(`${apiBase}/api/todos/${id}`, {
      method: 'PUT',
      body: todo
    })
  }

  const deleteTodo = async (id: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/todos/${id}`, {
      method: 'DELETE'
    })
  }

  const suggestTodos = async (projectId?: number, componentId?: number, saveToDB = true): Promise<Todo[]> => {
    const res = await $fetch<{ count?: number, new_count?: number, saved_to_db?: boolean, suggestions?: Todo[] } | Todo[]>(`${apiBase}/api/todos/suggest`, {
      method: 'POST',
      body: {
        project_id: projectId,
        component_id: componentId,
        save_to_db: saveToDB
      }
    })
    if (res && 'suggestions' in res && Array.isArray(res.suggestions)) {
      return res.suggestions
    }
    if (Array.isArray(res)) {
      return res
    }
    return []
  }

  /** Saran AI: saran aturan instan + rapat Project Manager di antrean untuk proyek (bila PM aktif). */
  const requestTodoSuggestions = async (projectId: number): Promise<{ suggestions: Todo[], new_count?: number, pm_session?: PlanningSession | null, pm_job_id?: string, pm_error?: string }> => {
    const res = await $fetch<{ suggestions?: Todo[], new_count?: number, pm_session?: PlanningSession | null, pm_job_id?: string, pm_error?: string }>(`${apiBase}/api/todos/suggest`, {
      method: 'POST',
      body: { project_id: projectId, save_to_db: true }
    })
    return { ...res, suggestions: res.suggestions ?? [] }
  }

  /** Lebur todo serupa (Aturan 43): dry_run = pratinjau kelompok tanpa mengubah todo. */
  const mergeTodos = async (body: { project_id?: number, todo_ids?: number[], dry_run?: boolean }): Promise<TodoMergeResult> => {
    return await $fetch<TodoMergeResult>(`${apiBase}/api/todos/merge`, { method: 'POST', body })
  }

  /** Satukan Serupa laporan audit: semua todo satu kelompok temuan → satu todo gabungan (tanpa batas sub todo). */
  const groupTodos = async (todoIds: number[]): Promise<TodoMergeGroup> => {
    return await $fetch<TodoMergeGroup>(`${apiBase}/api/todos/group`, { method: 'POST', body: { todo_ids: todoIds } })
  }

  /** Pecah todo gabungan yang belum dikerjakan kembali menjadi todo biasa. */
  const splitTodoGroup = async (id: number): Promise<{ status: string, todo_ids: number[] }> => {
    return await $fetch<{ status: string, todo_ids: number[] }>(`${apiBase}/api/todos/${id}/split`, { method: 'POST' })
  }

  const approveTodo = async (id: number): Promise<Todo> => {
    return await $fetch<Todo>(`${apiBase}/api/todos/${id}/approve`, {
      method: 'POST'
    })
  }

  const rejectTodo = async (id: number): Promise<Todo> => {
    return await $fetch<Todo>(`${apiBase}/api/todos/${id}/reject`, {
      method: 'POST'
    })
  }

  const getGuidelines = async (): Promise<GuidelineCategory[]> => {
    return await $fetch<GuidelineCategory[]>(`${apiBase}/api/guidelines`)
  }

  const getScaleUpPatterns = async (): Promise<ArchitecturePattern[]> => {
    return await $fetch<ArchitecturePattern[]>(`${apiBase}/api/scaleup/patterns`)
  }

  const evaluateComponentArchitecture = async (componentId: number): Promise<ArchitectureEvaluation> => {
    return await $fetch<ArchitectureEvaluation>(`${apiBase}/api/scaleup/evaluate/${componentId}`)
  }

  const evaluateProjectArchitecture = async (projectId: number): Promise<ArchitectureEvaluation[]> => {
    return await $fetch<ArchitectureEvaluation[]>(`${apiBase}/api/scaleup/project-evaluate/${projectId}`)
  }

  const scanSimilarities = async (): Promise<SimilarityScanResult> => {
    return await $fetch<SimilarityScanResult>(`${apiBase}/api/projects/similarity-scan`)
  }

  const compareProjects = async (idA: number, idB: number): Promise<SimilarityMatch> => {
    return await $fetch<SimilarityMatch>(`${apiBase}/api/projects/${idA}/compare/${idB}`)
  }

  const deprecateProject = async (id: number, reason?: string): Promise<{ message: string, project: Project }> => {
    return await $fetch<{ message: string, project: Project }>(`${apiBase}/api/projects/${id}/deprecate`, {
      method: 'POST',
      body: { reason }
    })
  }

  const undeprecateProject = async (id: number): Promise<{ message: string, project: Project }> => {
    return await $fetch<{ message: string, project: Project }>(`${apiBase}/api/projects/${id}/undeprecate`, {
      method: 'POST'
    })
  }

  const holdProject = async (id: number): Promise<{ message: string, project: Project }> => {
    return await $fetch<{ message: string, project: Project }>(`${apiBase}/api/projects/${id}/hold`, {
      method: 'POST'
    })
  }

  const doneProject = async (id: number): Promise<{ message: string, project: Project }> => {
    return await $fetch<{ message: string, project: Project }>(`${apiBase}/api/projects/${id}/done`, {
      method: 'POST'
    })
  }

  const activateProject = async (id: number): Promise<{ message: string, project: Project }> => {
    return await $fetch<{ message: string, project: Project }>(`${apiBase}/api/projects/${id}/activate`, {
      method: 'POST'
    })
  }

  const generateReorgPlan = async (payload?: ReorgPlanRequest): Promise<ReorgPlan> => {
    return await $fetch<ReorgPlan>(`${apiBase}/api/organizer/plan`, {
      method: 'POST',
      body: payload || {}
    })
  }

  const executeReorgMoves = async (payload: ExecuteMovePayload): Promise<ExecuteMoveResult> => {
    return await $fetch<ExecuteMoveResult>(`${apiBase}/api/organizer/execute`, {
      method: 'POST',
      body: payload
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

  // Project Base Knowledge & AI Chat
  const getProjectBaseKnowledge = async (projectId: number): Promise<ProjectBaseKnowledge> => {
    return await $fetch<ProjectBaseKnowledge>(`${apiBase}/api/projects/${projectId}/base-knowledge`)
  }

  const refreshProjectBaseKnowledge = async (projectId: number): Promise<ProjectBaseKnowledge> => {
    return await $fetch<ProjectBaseKnowledge>(`${apiBase}/api/projects/${projectId}/base-knowledge/refresh`, {
      method: 'POST'
    })
  }

  // Project Synergy Map (End-to-End Caller-to-Callee Trace)
  const getProjectSynergyMap = async (projectId: number): Promise<ProjectSynergyMap> => {
    return await $fetch<ProjectSynergyMap>(`${apiBase}/api/projects/${projectId}/synergy-map`)
  }

  const generateProjectSynergyMap = async (projectId: number, useLLM = false): Promise<ProjectSynergyMap> => {
    return await $fetch<ProjectSynergyMap>(`${apiBase}/api/projects/${projectId}/synergy-map/generate`, {
      method: 'POST',
      body: { use_llm: useLLM }
    })
  }

  const getProjectProfile = async (projectId: number): Promise<ProjectProfile> => {
    return await $fetch<ProjectProfile>(`${apiBase}/api/projects/${projectId}/profile`)
  }

  const generateProjectProfile = async (projectId: number): Promise<ProjectProfile> => {
    return await $fetch<ProjectProfile>(`${apiBase}/api/projects/${projectId}/profile/generate`, {
      method: 'POST',
      timeout: 200000
    })
  }

  const saveProjectProfile = async (projectId: number, profile: Partial<ProjectProfile>, agent: string): Promise<ProjectProfile> => {
    return await $fetch<ProjectProfile>(`${apiBase}/api/projects/${projectId}/profile`, {
      method: 'PUT',
      query: { agent },
      body: profile
    })
  }

  const getProjectProfilePrompt = async (projectId: number): Promise<ProjectProfilePrompt> => {
    return await $fetch<ProjectProfilePrompt>(`${apiBase}/api/projects/${projectId}/profile/prompt`)
  }

  const chatProject = async (projectId: number, payload: { message: string, model?: string }): Promise<ProjectChatResponse> => {
    return await $fetch<ProjectChatResponse>(`${apiBase}/api/projects/${projectId}/chat`, {
      method: 'POST',
      body: payload
    })
  }

  const getProjectChatHistory = async (projectId: number): Promise<ProjectChatMessageDTO[]> => {
    return await $fetch<ProjectChatMessageDTO[]>(`${apiBase}/api/projects/${projectId}/chat/history`)
  }

  const clearProjectChatHistory = async (projectId: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/projects/${projectId}/chat/history`, {
      method: 'DELETE'
    })
  }

  const chatProjectExecutives = async (projectId: number, payload: ExecutiveChatRequestDTO): Promise<ExecutiveChatResponseDTO> => {
    return await $fetch<ExecutiveChatResponseDTO>(`${apiBase}/api/projects/${projectId}/executives/chat`, {
      method: 'POST',
      body: payload
    })
  }

  const getProjectExecutivesChatHistory = async (projectId: number): Promise<ExecutiveChatMessageDTO[]> => {
    return await $fetch<ExecutiveChatMessageDTO[]>(`${apiBase}/api/projects/${projectId}/executives/chat/history`)
  }

  const clearProjectExecutivesChatHistory = async (projectId: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/projects/${projectId}/executives/chat/history`, {
      method: 'DELETE'
    })
  }

  const getProjectExecutiveKnowledge = async (projectId: number): Promise<ExecutiveKnowledgeItem[]> => {
    return await $fetch<ExecutiveKnowledgeItem[]>(`${apiBase}/api/projects/${projectId}/executives/knowledge`)
  }

  const createProjectExecutiveKnowledge = async (projectId: number, payload: Partial<ExecutiveKnowledgeItem>): Promise<ExecutiveKnowledgeItem> => {
    return await $fetch<ExecutiveKnowledgeItem>(`${apiBase}/api/projects/${projectId}/executives/knowledge`, {
      method: 'POST',
      body: payload
    })
  }

  const deleteProjectExecutiveKnowledge = async (projectId: number, knowledgeId: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/projects/${projectId}/executives/knowledge/${knowledgeId}`, {
      method: 'DELETE'
    })
  }

  const analyzeProjectTodosByExecutives = async (projectId: number): Promise<ExecutiveTodoAnalysisResult> => {
    return await $fetch<ExecutiveTodoAnalysisResult>(`${apiBase}/api/projects/${projectId}/executives/todos/analyze`, {
      method: 'POST'
    })
  }

  const batchActionProjectTodosByExecutives = async (
    projectId: number,
    payload: { action: string, todo_ids: number[], new_priority?: string, reason?: string, executive_role?: string }
  ): Promise<{ message: string, action: string, affected: number, todo_ids: number[] }> => {
    return await $fetch<{ message: string, action: string, affected: number, todo_ids: number[] }>(`${apiBase}/api/projects/${projectId}/executives/todos/batch-action`, {
      method: 'POST',
      body: payload
    })
  }

  const triggerExecutiveAutonomousCycle = async (projectId: number, payload?: { invited_employee_ids?: number[], topic?: string }): Promise<any> => {
    return await $fetch<any>(`${apiBase}/api/projects/${projectId}/executives/autonomous/cycle`, {
      method: 'POST',
      body: payload
    })
  }

  const getExecutiveAutonomousStatus = async (projectId: number): Promise<ExecutiveAutonomousStatus> => {
    return await $fetch<ExecutiveAutonomousStatus>(`${apiBase}/api/projects/${projectId}/executives/autonomous/status`)
  }

  const getCurrentExecutiveMeeting = async (projectId: number): Promise<{ active_meeting: ActiveExecutiveMeeting | null }> => {
    return await $fetch<{ active_meeting: ActiveExecutiveMeeting | null }>(`${apiBase}/api/projects/${projectId}/executives/meeting/current`)
  }

  const getEligibleMeetingStaff = async (projectId: number): Promise<OfficeEmployee[]> => {
    return await $fetch<OfficeEmployee[]>(`${apiBase}/api/projects/${projectId}/executives/meeting/eligible-staff`)
  }

  const getWorkspaceSettings = async (): Promise<WorkspaceSettings> => {
    return await $fetch<WorkspaceSettings>(`${apiBase}/api/settings/workspace`)
  }

  const updateWorkspaceSettings = async (payload: { default_projects_root: string } | string): Promise<WorkspaceSettings> => {
    const root = typeof payload === 'string' ? payload : payload.default_projects_root
    return await $fetch<WorkspaceSettings>(`${apiBase}/api/settings/workspace`, {
      method: 'PUT',
      body: { default_projects_root: root }
    })
  }

  const previewConsolidateProject = async (projectId: number, targetParentDir?: string): Promise<ConsolidatePreview> => {
    const query = new URLSearchParams()
    if (targetParentDir) query.append('target_parent_dir', targetParentDir)
    const url = `${apiBase}/api/projects/${projectId}/consolidate-preview${query.toString() ? '?' + query.toString() : ''}`
    return await $fetch<ConsolidatePreview>(url)
  }

  const consolidateProject = async (projectId: number, targetParentDir?: string): Promise<ConsolidateResult> => {
    return await $fetch<ConsolidateResult>(`${apiBase}/api/projects/${projectId}/consolidate`, {
      method: 'POST',
      body: { target_parent_dir: targetParentDir }
    })
  }

  return {
    apiBase,
    getProjectBaseKnowledge,
    refreshProjectBaseKnowledge,
    getProjectSynergyMap,
    generateProjectSynergyMap,
    getProjectProfile,
    generateProjectProfile,
    getProjectProfilePrompt,
    saveProjectProfile,
    chatProject,
    getProjectChatHistory,
    clearProjectChatHistory,
    chatProjectExecutives,
    getProjectExecutivesChatHistory,
    clearProjectExecutivesChatHistory,
    getProjectExecutiveKnowledge,
    createProjectExecutiveKnowledge,
    deleteProjectExecutiveKnowledge,
    analyzeProjectTodosByExecutives,
    batchActionProjectTodosByExecutives,
    triggerExecutiveAutonomousCycle,
    getExecutiveAutonomousStatus,
    getCurrentExecutiveMeeting,
    getEligibleMeetingStaff,
    getScaleUpPatterns,
    evaluateComponentArchitecture,
    evaluateProjectArchitecture,
    getTodos,
    getTodo,
    createTodo,
    updateTodo,
    deleteTodo,
    suggestTodos,
    requestTodoSuggestions,
    approveTodo,
    rejectTodo,
    mergeTodos,
    groupTodos,
    splitTodoGroup,
    getGuidelines,
    getProjects,
    getProject,
    createProject,
    updateProject,
    deleteProject,
    mergeProjects,
    generateProjectSummary,
    addComponent,
    getComponent,
    createFolderOnDisk,
    updateComponent,
    deleteComponent,
    getAdapters,
    scanDirectory,
    groupScanComponents,
    browseDirectories,
    linkWorkspace,
    identifyProject,
    runDiagnostics,
    runDeepCheck,
    getLatestDeepCheck,
    getDeepCheckDownloadUrl,
    verifyDiagnosticFinding,
    recordFindingFeedback,
    getAuditLearning,
    markAuditRuleUpdated,
    deleteAuditFeedback,
    getAuditLearningExportUrl,
    getLLMWeighing,
    getKnowledge,
    getKnowledgeGraph,
    previewKnowledgePrompt,
    listKnowledgeNodes,
    getKnowledgeNode,
    teachKnowledge,
    knowledgeFeedback,
    retireKnowledgeNode,
    recallKnowledge,
    curateKnowledge,
    reportKnowledgeGap,
    dismissKnowledgeGap,
    startRuntimeSimulation,
    getRuntimeSimulation,
    cancelRuntimeSimulation,
    getRuntimeSimDownloadUrl,
    getRuntimeSimShotUrl,
    getRuntimeTestProfile,
    getLatestProjectDiagnostic,
    resolveDiagnosticIssue,
    resolveAllDiagnosticIssues,
    resolveDiagnosticGroup,
    deleteDiagnosticFinding,
    ignoreDiagnosticFinding,
    runAiReview,
    diagnoseComponent,
    getLatestComponentDiagnostic,
    aiReviewComponent,
    checkPorts,
    callMcp,
    getMCPInfo,
    getLLMSettings,
    updateLLMSettings,
    getLLMConfigs,
    createLLMConfig,
    updateLLMConfig,
    setLLMModelVision,
    setLLMModelAlias,
    deleteLLMConfig,
    activateLLMConfig,
    setActiveProvider,
    selectActiveModel,
    testLLMConnection,
    fastScanComponent,
    updateComponentFastScan,
    getComponentRoutes,
    syncComponentRoutes,
    enrichComponentRoutes,
    updateComponentRoute,
    deleteComponentRoute,
    applyFastScanParams,
    fastScanProject,
    getSecurityStatus,
    getCensoredKeywords,
    createCensoredKeyword,
    updateCensoredKeyword,
    deleteCensoredKeyword,
    testCensorPayload,
    getQueueJobs,
    getAgentOffice,
    getAgentRecap,
    getStaff,
    getEmployee,
    hireEmployee,
    updateEmployee,
    resignEmployee,
    teachEmployee,
    consolidateKnowledge,
    forgetKnowledge,
    getEmployeeChat,
    sendEmployeeChat,
    clearEmployeeChat,
    requestEmployeeReview,
    updateStaffSettings,
    getAgendas,
    addAgenda,
    startAgenda,
    getConversations,
    startConversation,
    getHR,
    evaluateHR,
    decideHR,
    getTurnovers,
    getRevisits,
    processRevisits,
    revisitAction,
    dropAgenda,
    getLLMPool,
    resetLLMProvider,
    getStaffTeams,
    setStaffLeader,
    getExecutives,
    getExecutive,
    provisionExecutives,
    assignTodoEmployee,
    getPMOverview,
    updatePMSettings,
    requestPMPlan,
    getPMSessions,
    getPMSession,
    getOrchestratorOverview,
    getOrchestratorAccounts,
    updateOrchestratorSettings,
    saveAgentProfile,
    deleteAgentProfile,
    resetAgentProfileCooldown,
    importAgentProfiles,
    getAgentRuns,
    getAgentRun,
    dispatchAgentRun,
    dispatchAgentBatch,
    previewAgentRoute,
    cancelAgentRun,
    retryAgentRun,
    deleteAgentRun,
    getQueueStats,
    enqueueQueueJob,
    deleteQueueJob,
    clearQueueHistory,
    startProjectPipeline,
    cancelProjectPipeline,
    scanSimilarities,
    compareProjects,
    deprecateProject,
    undeprecateProject,
    holdProject,
    doneProject,
    activateProject,
    generateReorgPlan,
    executeReorgMoves,
    getWorkspaceSettings,
    updateWorkspaceSettings,
    previewConsolidateProject,
    consolidateProject,
    captureProjectVisual,
    auditProjectVisual,
    getProjectVisualSnapshots,
    deleteProjectVisualSnapshot
  }
}
