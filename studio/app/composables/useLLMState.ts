import type { LLMConfig, LLMTestResult } from '~/composables/useApi'

export const useLLMState = () => {
  const { getLLMConfigs, getLLMSettings, selectActiveModel, testLLMConnection } = useApi()

  // Shared reactive states across all components and pages
  const activeStandardConfig = useState<LLMConfig | null>('llm_active_standard_config', () => null)
  const availableModels = useState<string[]>('llm_available_models', () => [])
  const selectedModel = useState<string>('llm_selected_model', () => '')
  const activeProfileName = useState<string>('llm_active_profile_name', () => '')
  const isSwitchingModel = useState<boolean>('llm_is_switching_model', () => false)
  const isLoading = useState<boolean>('llm_state_loading', () => false)

  // Mini popup ping notification state
  const pingResult = useState<LLMTestResult | null>('llm_ping_result', () => null)
  const isPinging = useState<boolean>('llm_is_pinging', () => false)

  const refreshLLMState = async () => {
    isLoading.value = true
    try {
      const configs = await getLLMConfigs('standard_llm')
      const active = configs.find(c => c.is_active) || configs[0] || null
      activeStandardConfig.value = active

      if (active) {
        activeProfileName.value = active.name
        availableModels.value = active.models || []
        selectedModel.value = active.selected_model || (active.models?.[0] || '')
      } else {
        const settings = await getLLMSettings()
        availableModels.value = settings.standard_llm_models || []
        selectedModel.value = settings.standard_llm_selected || (settings.standard_llm_models?.[0] || '')
        activeProfileName.value = 'Standard LLM'
      }
    } catch (err) {
      console.warn('Failed to fetch active LLM state:', err)
    } finally {
      isLoading.value = false
    }
  }

  const handleSelectModel = async (model: string) => {
    if (!model || model === selectedModel.value) return
    isSwitchingModel.value = true
    const previous = selectedModel.value
    selectedModel.value = model
    try {
      await selectActiveModel(model)
      if (activeStandardConfig.value) {
        activeStandardConfig.value.selected_model = model
      }
    } catch (err) {
      console.error('Failed to update selected model:', err)
      selectedModel.value = previous
    } finally {
      isSwitchingModel.value = false
    }
  }

  /** Nama tampilan model: alias dari konfigurasi aktif bila ada, selain itu nama model asli. */
  const modelLabel = (model: string) => activeStandardConfig.value?.aliases?.[model] || model

  const dismissPing = () => {
    pingResult.value = null
  }

  let dismissTimeout: ReturnType<typeof setTimeout> | null = null

  const triggerPing = async (configId?: number, model?: string): Promise<LLMTestResult> => {
    if (dismissTimeout) {
      clearTimeout(dismissTimeout)
      dismissTimeout = null
    }
    isPinging.value = true
    pingResult.value = null

    try {
      const payload: { config_id?: number, provider?: string, model?: string } = {}
      if (configId) {
        payload.config_id = configId
      } else {
        payload.provider = 'standard_llm'
        payload.model = model || selectedModel.value
      }

      const res = await testLLMConnection(payload)
      pingResult.value = res

      dismissTimeout = setTimeout(() => {
        pingResult.value = null
      }, 7000)
      return res
    } catch (err: any) {
      const errRes: LLMTestResult = {
        success: false,
        latency_ms: 0,
        // Engine menolak dengan {error, code} (mis. exec_origin_forbidden 403); tampilkan alasannya, bukan pesan fetch mentah.
        message: err.data?.error || err.data?.message || err.message || 'Koneksi ke endpoint LLM gagal.'
      }
      pingResult.value = errRes
      dismissTimeout = setTimeout(() => {
        pingResult.value = null
      }, 8000)
      return errRes
    } finally {
      isPinging.value = false
    }
  }

  return {
    activeStandardConfig,
    availableModels,
    selectedModel,
    activeProfileName,
    isSwitchingModel,
    isLoading,
    pingResult,
    isPinging,
    refreshLLMState,
    handleSelectModel,
    modelLabel,
    triggerPing,
    dismissPing
  }
}
