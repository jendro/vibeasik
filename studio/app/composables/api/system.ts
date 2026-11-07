// Domain API "system" — dipecah dari useApi.ts tanpa mengubah kontrak HTTP.
import { useApiBase } from './base'

export const useSystemApi = () => {
  const apiBase = useApiBase()

  const checkPorts = async (ports: number[]): Promise<Record<string, boolean>> => {
    if (!ports.length) return {}
    const portStr = ports.filter(p => p > 0).join(',')
    const res = await $fetch<{ ports: Record<string, boolean> }>(`${apiBase}/api/system/ports?ports=${portStr}`)
    return res.ports || {}
  }

  const callMcp = async (method: string, params?: any): Promise<any> => {
    return await $fetch(`${apiBase}/mcp`, {
      method: 'POST',
      body: {
        jsonrpc: '2.0',
        id: Date.now(),
        method,
        params
      }
    })
  }

  const getMCPInfo = async (): Promise<any> => {
    return await $fetch(`${apiBase}/api/mcp/info`)
  }

  const getSecurityStatus = async (): Promise<SecurityStatus> => {
    return await $fetch<SecurityStatus>(`${apiBase}/api/security/status`)
  }

  const getCensoredKeywords = async (): Promise<CensoredKeywordsResponse> => {
    return await $fetch<CensoredKeywordsResponse>(`${apiBase}/api/security/censored-keywords`)
  }

  const createCensoredKeyword = async (payload: { keyword: string, category?: string, description?: string, is_regex?: boolean, is_active?: boolean }): Promise<CensoredKeyword> => {
    return await $fetch<CensoredKeyword>(`${apiBase}/api/security/censored-keywords`, {
      method: 'POST',
      body: payload
    })
  }

  const updateCensoredKeyword = async (id: number, payload: Partial<CensoredKeyword>): Promise<CensoredKeyword> => {
    return await $fetch<CensoredKeyword>(`${apiBase}/api/security/censored-keywords/${id}`, {
      method: 'PUT',
      body: payload
    })
  }

  const deleteCensoredKeyword = async (id: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/security/censored-keywords/${id}`, {
      method: 'DELETE'
    })
  }

  const testCensorPayload = async (text: string): Promise<TestCensorResponse> => {
    return await $fetch<TestCensorResponse>(`${apiBase}/api/security/test-censor`, {
      method: 'POST',
      body: { text }
    })
  }

  return {
    checkPorts,
    callMcp,
    getMCPInfo,
    getSecurityStatus,
    getCensoredKeywords,
    createCensoredKeyword,
    updateCensoredKeyword,
    deleteCensoredKeyword,
    testCensorPayload
  }
}
