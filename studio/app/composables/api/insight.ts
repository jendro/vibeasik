// Domain API "insight": rekap pemilik, biaya proses (tick & Big-O), dan penanganan risiko profil.
import { useApiBase } from './base'
import type { OwnerRecap, ProcessCostPage, ProjectProfile } from '~/types'

export interface ProcessCostQuery {
  kind?: 'routes' | 'funcs'
  q?: string
  class?: string
  n_plus_one?: '1' | ''
  limit?: number
  offset?: number
}

export interface RiskUpdate {
  title: string
  status?: 'open' | 'handled' | 'accepted'
  todo_id?: number
  note?: string
}

export const useInsightApi = () => {
  const apiBase = useApiBase()

  const getOwnerRecap = async (projectId: number, days = 30): Promise<OwnerRecap> =>
    await $fetch<OwnerRecap>(`${apiBase}/api/projects/${projectId}/owner-recap`, { query: { days } })

  const getProcessCost = async (projectId: number, query: ProcessCostQuery = {}, signal?: AbortSignal): Promise<ProcessCostPage> =>
    await $fetch<ProcessCostPage>(`${apiBase}/api/projects/${projectId}/process-cost`, { query, signal, timeout: 120000 })

  const updateProjectRisk = async (projectId: number, body: RiskUpdate): Promise<ProjectProfile> =>
    await $fetch<ProjectProfile>(`${apiBase}/api/projects/${projectId}/profile/risks`, { method: 'POST', body })

  return { getOwnerRecap, getProcessCost, updateProjectRisk }
}
