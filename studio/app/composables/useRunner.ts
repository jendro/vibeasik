import type { RunStatus, PortRegistryResponse, ProjectRunItem } from '~/types'

// Port registry & dev runner: setiap komponen mendapat port unik yang stabil dan bisa dijalankan dari vibeasik.
export const useRunner = () => {
  const { apiBase } = useApi()

  const getRunStatus = async (componentId: number, withLogs = true): Promise<RunStatus> => {
    return await $fetch<RunStatus>(`${apiBase}/api/components/${componentId}/run?logs=${withLogs ? 1 : 0}`)
  }

  // reassign=true memindahkan komponen ke port bebas bila port tetapnya dipakai proses lain.
  const runComponent = async (componentId: number, reassign = false): Promise<RunStatus> => {
    return await $fetch<RunStatus>(`${apiBase}/api/components/${componentId}/run`, {
      method: 'POST',
      body: { reassign }
    })
  }

  const stopComponent = async (componentId: number): Promise<RunStatus> => {
    return await $fetch<RunStatus>(`${apiBase}/api/components/${componentId}/stop`, { method: 'POST' })
  }

  const assignPort = async (componentId: number, payload: { port?: number, reassign?: boolean }): Promise<{ component_id: number, run_port: number }> => {
    return await $fetch(`${apiBase}/api/components/${componentId}/ports`, { method: 'POST', body: payload })
  }

  const getPortRegistry = async (): Promise<PortRegistryResponse> => {
    return await $fetch<PortRegistryResponse>(`${apiBase}/api/ports/registry`)
  }

  const runProject = async (projectId: number, reassign = false): Promise<ProjectRunItem[]> => {
    const res = await $fetch<{ results: ProjectRunItem[] }>(`${apiBase}/api/projects/${projectId}/run`, {
      method: 'POST',
      body: { reassign }
    })
    return res.results || []
  }

  const stopProject = async (projectId: number): Promise<ProjectRunItem[]> => {
    const res = await $fetch<{ results: ProjectRunItem[] }>(`${apiBase}/api/projects/${projectId}/stop`, { method: 'POST' })
    return res.results || []
  }

  return {
    getRunStatus,
    runComponent,
    stopComponent,
    assignPort,
    getPortRegistry,
    runProject,
    stopProject
  }
}
