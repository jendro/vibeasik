import type {
  Tutorial,
  DomainSearchResponse,
  DeployPlanRequest,
  DeployPlanResponse,
  CloudAuditResponse,
  AITemplate,
  AgentStatusDTO,
  AgentExecutionRequest,
  AgentExecutionResult,
  CustomAgentForm,
  ReaStatus
} from '~/types'

export const useVibeasik = () => {
  const { apiBase } = useApi()

  const fetchTutorials = async (category = ''): Promise<Tutorial[]> => {
    const url = category ? `${apiBase}/api/vibeasik/tutorials?category=${encodeURIComponent(category)}` : `${apiBase}/api/vibeasik/tutorials`
    const data = await $fetch<{ tutorials: Tutorial[] }>(url)
    return data.tutorials || []
  }

  const searchDomains = async (query: string): Promise<DomainSearchResponse> => {
    return await $fetch<DomainSearchResponse>(`${apiBase}/api/vibeasik/domains/search?query=${encodeURIComponent(query)}`)
  }

  const generateDeployPlan = async (payload: DeployPlanRequest): Promise<DeployPlanResponse> => {
    return await $fetch<DeployPlanResponse>(`${apiBase}/api/vibeasik/deploy/generate-plan`, {
      method: 'POST',
      body: payload
    })
  }

  const runCloudSecurityAudit = async (payload: any): Promise<CloudAuditResponse> => {
    return await $fetch<CloudAuditResponse>(`${apiBase}/api/vibeasik/security/cloud-audit`, {
      method: 'POST',
      body: payload
    })
  }

  const fetchAITemplates = async (): Promise<AITemplate[]> => {
    const data = await $fetch<{ templates: AITemplate[] }>(`${apiBase}/api/vibeasik/ai-templates`)
    return data.templates || []
  }

  const injectAITemplate = async (payload: { component_id: number, template_id: string, target_path: string }) => {
    return await $fetch<{ success: boolean, message: string, created_files: string[], install_guide: string }>(`${apiBase}/api/vibeasik/ai-templates/inject`, {
      method: 'POST',
      body: payload
    })
  }

  const fetchLocalAgents = async (): Promise<AgentStatusDTO[]> => {
    const data = await $fetch<{ status: string, agents: AgentStatusDTO[] }>(`${apiBase}/api/agents`)
    return data.agents || []
  }

  const executeLocalAgent = async (req: AgentExecutionRequest): Promise<AgentExecutionResult> => {
    return await $fetch<AgentExecutionResult>(`${apiBase}/api/agents/execute`, {
      method: 'POST',
      body: req
    })
  }

  const createCustomAgent = async (form: CustomAgentForm) => {
    return await $fetch<{ success: boolean, message: string }>(`${apiBase}/api/agents/custom`, {
      method: 'POST',
      body: form
    })
  }

  const deleteCustomAgent = async (key: string) => {
    return await $fetch<{ success: boolean, message: string }>(`${apiBase}/api/agents/custom/${encodeURIComponent(key)}`, {
      method: 'DELETE'
    })
  }

  const getReaStatus = async (): Promise<ReaStatus> => {
    const data = await $fetch<ReaStatus>(`${apiBase}/api/rea/status`)
    return data
  }

  const runReaAnalysis = async (targetPath: string, analysisType?: string): Promise<{ output: string }> => {
    return await $fetch<{ output: string }>(`${apiBase}/api/rea/analyze`, {
      method: 'POST',
      body: { target_path: targetPath, analysis_type: analysisType || 'javascript' }
    })
  }

  return {
    fetchTutorials,
    searchDomains,
    generateDeployPlan,
    runCloudSecurityAudit,
    fetchAITemplates,
    injectAITemplate,
    fetchLocalAgents,
    executeLocalAgent,
    createCustomAgent,
    deleteCustomAgent,
    getReaStatus,
    runReaAnalysis
  }
}
