// Domain API "llm" — dipecah dari useApi.ts tanpa mengubah kontrak HTTP.
import { useApiBase } from './base'

export const useLlmApi = () => {
  const apiBase = useApiBase()

  const getLLMWeighing = async (limit = 40): Promise<LLMWeighingReport> =>
    await $fetch<LLMWeighingReport>(`${apiBase}/api/settings/llm/weighing`, { query: { limit } })

  const getLLMSettings = async (): Promise<LLMSettings> => {
    return await $fetch<LLMSettings>(`${apiBase}/api/settings/llm`)
  }

  const updateLLMSettings = async (data: Partial<LLMSettings>): Promise<LLMSettings> => {
    return await $fetch<LLMSettings>(`${apiBase}/api/settings/llm`, {
      method: 'PUT',
      body: data
    })
  }

  const getLLMConfigs = async (provider?: string): Promise<LLMConfig[]> => {
    const query = provider ? `?provider=${encodeURIComponent(provider)}` : ''
    return await $fetch<LLMConfig[]>(`${apiBase}/api/settings/llm/configs${query}`)
  }

  const createLLMConfig = async (data: Partial<LLMConfig>): Promise<LLMConfig> => {
    return await $fetch<LLMConfig>(`${apiBase}/api/settings/llm/configs`, {
      method: 'POST',
      body: data
    })
  }

  const updateLLMConfig = async (id: number, data: Partial<LLMConfig>): Promise<LLMConfig> => {
    return await $fetch<LLMConfig>(`${apiBase}/api/settings/llm/configs/${id}`, {
      method: 'PUT',
      body: data
    })
  }

  /** Tandai satu model konfigurasi menerima gambar (vision) atau tidak. */

  const setLLMModelVision = async (id: number, model: string, vision: boolean): Promise<LLMConfig> => {
    return await $fetch<LLMConfig>(`${apiBase}/api/settings/llm/configs/${id}/vision`, {
      method: 'PUT',
      body: { model, vision }
    })
  }

  /** Beri alias tampilan untuk satu model konfigurasi (alias kosong = hapus alias). */

  const setLLMModelAlias = async (id: number, model: string, alias: string): Promise<LLMConfig> => {
    return await $fetch<LLMConfig>(`${apiBase}/api/settings/llm/configs/${id}/alias`, {
      method: 'PUT',
      body: { model, alias }
    })
  }

  const deleteLLMConfig = async (id: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/settings/llm/configs/${id}`, {
      method: 'DELETE'
    })
  }

  const activateLLMConfig = async (id: number): Promise<{ message: string, config: LLMConfig }> => {
    return await $fetch<{ message: string, config: LLMConfig }>(`${apiBase}/api/settings/llm/configs/${id}/activate`, {
      method: 'POST'
    })
  }

  const setActiveProvider = async (provider: 'system_one' | 'standard_llm'): Promise<{ message: string, active_provider: string }> => {
    return await $fetch<{ message: string, active_provider: string }>(`${apiBase}/api/settings/llm/active-provider`, {
      method: 'POST',
      body: { provider }
    })
  }

  const selectActiveModel = async (model: string): Promise<{ message: string, selected_model: string }> => {
    return await $fetch<{ message: string, selected_model: string }>(`${apiBase}/api/settings/llm/select-model`, {
      method: 'POST',
      body: { model }
    })
  }

  const testLLMConnection = async (payload: { config_id?: number, provider?: string, url?: string, model?: string, key?: string }): Promise<LLMTestResult> => {
    return await $fetch<LLMTestResult>(`${apiBase}/api/settings/llm/test`, {
      method: 'POST',
      body: payload
    })
  }

  /** Ambil katalog model provider (GET {url}/models): id dikirim ke LLM, display_name hanya untuk tampilan. */
  const discoverLLMModels = async (payload: { url?: string, key?: string, config_id?: number }): Promise<{ models: DiscoveredLLMModel[] }> =>
    await $fetch<{ models: DiscoveredLLMModel[] }>(`${apiBase}/api/settings/llm/models/discover`, { method: 'POST', body: payload })

  const getLLMPool = async (): Promise<LLMPoolResponse> => await $fetch<LLMPoolResponse>(`${apiBase}/api/llm/pool`)

  const resetLLMProvider = async (id: number) => await $fetch(`${apiBase}/api/llm/providers/${id}/reset`, { method: 'POST' })

  return {
    getLLMWeighing,
    getLLMSettings,
    updateLLMSettings,
    getLLMConfigs,
    createLLMConfig,
    updateLLMConfig,
    setLLMModelVision,
    setLLMModelAlias,
    deleteLLMConfig,
    activateLLMConfig,
    setActiveProvider,
    selectActiveModel,
    testLLMConnection,
    discoverLLMModels,
    getLLMPool,
    resetLLMProvider
  }
}
