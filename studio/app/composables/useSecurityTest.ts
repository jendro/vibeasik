import type { SecurityTestReport, SecurityTestRequest } from '~/types'

/** Uji keamanan proyek: audit router + probe server berjalan (lokal/live). */
export const useSecurityTest = () => {
  const { apiBase } = useApi()
  const url = (projectId: number) => `${apiBase}/api/projects/${projectId}/security-test`

  return {
    getLast: (projectId: number) => $fetch<{ report: SecurityTestReport | null }>(url(projectId)),
    run: (projectId: number, body: SecurityTestRequest) =>
      $fetch<SecurityTestReport>(url(projectId), { method: 'POST', body, timeout: 180000 })
  }
}
