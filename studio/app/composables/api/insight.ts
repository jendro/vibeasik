// Domain API "insight": rekap pemilik dan biaya proses (tick & Big-O).
import { useApiBase } from './base'
import type { CostSource, CostSourceWindow, CostTodoBatch, FindingsTodoPrompt, OwnerRecap, ProcessCostPage, Todo } from '~/types'

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

  /** "Jadikan semua todo": semua sumber biaya yang belum punya todo terbuka, satu todo per berkas (tanpa LLM). */
  const createAllProcessCostTodos = async (projectId: number): Promise<CostTodoBatch> =>
    await $fetch(`${apiBase}/api/projects/${projectId}/process-cost/todos`, { method: 'POST', timeout: 180000 })

  /** Potongan berkas (baris from..to, maks 400) untuk tampilan editor bukti biaya. */
  const getProcessCostSource = async (projectId: number, file: string, from: number, to: number): Promise<CostSourceWindow> =>
    await $fetch(`${apiBase}/api/projects/${projectId}/process-cost/source`, { query: { file, from, to } })

  /** Prompt siap salin untuk agent coding: temuan satu sumber (audit, cost, runtime) → todo lewat MCP, tanpa LLM luar. */
  const getFindingsTodoPrompt = async (projectId: number, scope: string): Promise<FindingsTodoPrompt> =>
    await $fetch(`${apiBase}/api/projects/${projectId}/findings/todo-prompt`, { query: { scope } })

  return { getOwnerRecap, getProcessCost, createProcessCostTodo, createAllProcessCostTodos, getProcessCostSource, getFindingsTodoPrompt }
}
