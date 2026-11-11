// Domain API "insight": rekap pemilik dan biaya proses (tick & Big-O).
import { useApiBase } from './base'
import type { OwnerRecap, ProcessCostPage } from '~/types'

export interface ProcessCostQuery {
  kind?: 'routes' | 'funcs'
  q?: string
  class?: string
  n_plus_one?: '1' | ''
  limit?: number
  offset?: number
}

export const useInsightApi = () => {
  const apiBase = useApiBase()

  const getOwnerRecap = async (projectId: number, days = 30): Promise<OwnerRecap> =>
    await $fetch<OwnerRecap>(`${apiBase}/api/projects/${projectId}/owner-recap`, { query: { days } })

  const getProcessCost = async (projectId: number, query: ProcessCostQuery = {}, signal?: AbortSignal): Promise<ProcessCostPage> =>
    await $fetch<ProcessCostPage>(`${apiBase}/api/projects/${projectId}/process-cost`, { query, signal, timeout: 120000 })

  return { getOwnerRecap, getProcessCost }
}
