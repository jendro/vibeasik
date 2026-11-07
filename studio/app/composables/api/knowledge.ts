// Domain API "knowledge" — dipecah dari useApi.ts tanpa mengubah kontrak HTTP.
import { useApiBase } from './base'

export const useKnowledgeApi = () => {
  const apiBase = useApiBase()

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

  return {
    getKnowledge,
    getKnowledgeGraph,
    listKnowledgeNodes,
    getKnowledgeNode,
    teachKnowledge,
    knowledgeFeedback,
    retireKnowledgeNode,
    recallKnowledge,
    previewKnowledgePrompt,
    curateKnowledge,
    reportKnowledgeGap,
    dismissKnowledgeGap,
    getProjectBaseKnowledge,
    refreshProjectBaseKnowledge,
    getProjectSynergyMap,
    generateProjectSynergyMap,
    getProjectProfile,
    generateProjectProfile,
    saveProjectProfile,
    getProjectProfilePrompt,
    chatProject,
    getProjectChatHistory,
    clearProjectChatHistory
  }
}
