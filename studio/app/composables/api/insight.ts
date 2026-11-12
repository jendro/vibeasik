// Domain API "insight": rekap pemilik dan biaya proses (tick & Big-O).
import { useApiBase } from './base'
import type { CostSource, OwnerRecap, ProcessCostPage, Todo } from '~/types'

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

  /** Jadikan satu sumber biaya tinggi sebagai todo; todo terbuka yang sudah ada dikembalikan (already_exists). */
  const createProcessCostTodo = async (projectId: number, src: Pick<CostSource, 'file' | 'line' | 'kind'>): Promise<{ todo: Todo, already_exists: boolean }> =>
    await $fetch(`${apiBase}/api/projects/${projectId}/process-cost/todo`, { method: 'POST', body: { file: src.file, line: src.line, kind: src.kind }, timeout: 120000 })

  return { getOwnerRecap, getProcessCost, createProcessCostTodo }
}
