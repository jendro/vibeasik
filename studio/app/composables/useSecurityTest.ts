import type { SecurityTestReport, SecurityTestRequest, Todo } from '~/types'

/** Uji keamanan proyek: audit router + probe server berjalan (lokal/live). */
export const useSecurityTest = () => {
  const { apiBase } = useApi()
  const url = (projectId: number) => `${apiBase}/api/projects/${projectId}/security-test`

  return {
    getLast: (projectId: number) => $fetch<{ report: SecurityTestReport | null }>(url(projectId)),
    run: (projectId: number, body: SecurityTestRequest) =>
      $fetch<SecurityTestReport>(url(projectId), { method: 'POST', body, timeout: 180000 }),
    /** Pemeriksaan gagal/lemah di hasil uji terakhir → satu todo keamanan per pemeriksaan (tanpa LLM). */
    createTodos: (projectId: number) =>
      $fetch<{ created: number, existed: number, todos: Todo[] }>(`${url(projectId)}/todos`, { method: 'POST' })
  }
}
