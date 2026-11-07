// Domain API "queue" — dipecah dari useApi.ts tanpa mengubah kontrak HTTP.
import { useApiBase } from './base'

export const useQueueApi = () => {
  const apiBase = useApiBase()

  // Background Job Queue & Pipeline Monitor
  const getQueueJobs = async (status?: string, limit = 50): Promise<QueueResponse> => {
    const query = new URLSearchParams()
    if (status) query.append('status', status)
    if (limit) query.append('limit', String(limit))
    const url = `${apiBase}/api/queue${query.toString() ? '?' + query.toString() : ''}`
    return await $fetch<QueueResponse>(url)
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

  return {
    getQueueJobs,
    getQueueStats,
    enqueueQueueJob,
    deleteQueueJob,
    clearQueueHistory,
    startProjectPipeline,
    cancelProjectPipeline
  }
}
