// Domain API "projects" — dipecah dari useApi.ts tanpa mengubah kontrak HTTP.
import { useApiBase } from './base'

export const useProjectsApi = () => {
  const apiBase = useApiBase()

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

  const getGuidelines = async (): Promise<GuidelineCategory[]> => {
    return await $fetch<GuidelineCategory[]>(`${apiBase}/api/guidelines`)
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
    getProjects,
    getProject,
    createProject,
    updateProject,
    deleteProject,
    mergeProjects,
    generateProjectSummary,
    linkWorkspace,
    identifyProject,
    addComponent,
    getComponent,
    createFolderOnDisk,
    updateComponent,
    deleteComponent,
    getAdapters,
    scanDirectory,
    groupScanComponents,
    browseDirectories,
    fastScanComponent,
    applyFastScanParams,
    fastScanProject,
    updateComponentFastScan,
    getComponentRoutes,
    syncComponentRoutes,
    enrichComponentRoutes,
    updateComponentRoute,
    deleteComponentRoute,
    getGuidelines,
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
    consolidateProject
  }
}
