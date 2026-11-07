// Domain API "agents" — dipecah dari useApi.ts tanpa mengubah kontrak HTTP.
import { useApiBase } from './base'

export const useAgentsApi = () => {
  const apiBase = useApiBase()

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

  return {
    getAgentOffice,
    getAgentRecap,
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
    deleteAgentRun
  }
}
