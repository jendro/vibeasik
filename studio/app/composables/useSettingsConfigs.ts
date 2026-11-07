import type { Ref, ComputedRef } from 'vue'
import type { LLMConfig } from '~/types'
import { useLLMState } from '~/composables/useLLMState'

type Notify = (msg: string, type?: 'success' | 'info' | 'error') => void

// Grid konfigurasi LLM: dialog tambah/ubah/hapus, aktifkan, vision, alias, uji ping, salin URL.
export function useSettingsConfigs(opts: {
  notify: Notify
  reload: () => Promise<void>
  configs: Ref<LLMConfig[]>
  activeViewTab: Ref<string>
  systemOneConfigs: ComputedRef<LLMConfig[]>
  standardLLMConfigs: ComputedRef<LLMConfig[]>
}) {
  const { notify, configs, activeViewTab, systemOneConfigs, standardLLMConfigs } = opts
  const { createLLMConfig, updateLLMConfig, setLLMModelVision, setLLMModelAlias, deleteLLMConfig, activateLLMConfig } = useApi()
  const { triggerPing, refreshLLMState } = useLLMState()

  const testingConfigId = ref<number | null>(null)

  // Modal Add/Edit Config
  const isModalOpen = ref(false)
  const isEditing = ref(false)
  const modalSaving = ref(false)
  const editingConfig = ref<Partial<LLMConfig> | null>(null)

  // Modal Delete Config
  const isDeleteModalOpen = ref(false)
  const configToDelete = ref<LLMConfig | null>(null)
  const deleting = ref(false)

  // Config Modal Handlers
  const openAddModal = (provider?: string) => {
    const targetProvider: 'system_one' | 'standard_llm' = (provider === 'standard_llm' || activeViewTab.value === 'standard_llm') ? 'standard_llm' : 'system_one'
    isEditing.value = false
    editingConfig.value = {
      id: 0,
      provider: targetProvider,
      name: '',
      url: targetProvider === 'system_one' ? 'http://localhost:8750/v1/systemone' : 'https://api.openai.com/v1',
      key: '',
      models: targetProvider === 'system_one' ? ['convaiinnovations/laya-multilingual'] : ['gpt-4o-mini', 'gpt-4o'],
      selected_model: targetProvider === 'system_one' ? 'convaiinnovations/laya-multilingual' : 'gpt-4o-mini',
      is_active: targetProvider === 'system_one' ? systemOneConfigs.value.length === 0 : standardLLMConfigs.value.length === 0
    }
    isModalOpen.value = true
  }

  const openEditModal = (cfg: LLMConfig) => {
    isEditing.value = true
    editingConfig.value = { ...cfg }
    isModalOpen.value = true
  }

  const handleSaveConfig = async (formData: any) => {
    modalSaving.value = true
    try {
      if (isEditing.value && formData.id) {
        await updateLLMConfig(formData.id, {
          name: formData.name,
          url: formData.url,
          key: formData.key,
          models: formData.models,
          selected_model: formData.selected_model,
          is_active: formData.is_active
        })
        notify(`Konfigurasi '${formData.name}' berhasil diperbarui.`)
      } else {
        await createLLMConfig({
          provider: formData.provider,
          name: formData.name,
          url: formData.url,
          key: formData.key,
          models: formData.models,
          selected_model: formData.selected_model,
          is_active: formData.is_active
        })
        notify(`Konfigurasi '${formData.name}' berhasil ditambahkan ke grid.`)
      }
      isModalOpen.value = false
      await opts.reload()
      await refreshLLMState()
    } catch (err: any) {
      notify(`Gagal menyimpan konfigurasi: ${err.message || err}`, 'error')
    } finally {
      modalSaving.value = false
    }
  }

  const handleActivateConfig = async (cfg: LLMConfig) => {
    try {
      await activateLLMConfig(cfg.id)
      notify(`Konfigurasi '${cfg.name}' sekarang aktif sebagai engine ${cfg.provider === 'system_one' ? 'System One' : 'LLM Biasa'}.`)
      await opts.reload()
      await refreshLLMState()
    } catch (err: any) {
      notify(`Gagal mengaktifkan konfigurasi: ${err.message || err}`, 'error')
    }
  }

  const handleToggleVision = async (cfg: LLMConfig, model: string, vision: boolean) => {
    try {
      const updated = await setLLMModelVision(cfg.id, model, vision)
      configs.value = configs.value.map(c => (c.id === cfg.id ? { ...c, vision: updated.vision, vision_manual: updated.vision_manual } : c))
      notify(`${model} ditandai ${vision ? 'mendukung' : 'tidak mendukung'} gambar (vision).`)
    } catch (err: any) {
      notify(`Gagal mengubah dukungan vision: ${err?.data?.error || err.message || err}`, 'error')
    }
  }

  const handleSetAlias = async (cfg: LLMConfig, model: string, alias: string) => {
    try {
      const updated = await setLLMModelAlias(cfg.id, model, alias)
      configs.value = configs.value.map(c => (c.id === cfg.id ? { ...c, aliases: updated.aliases } : c))
      notify(alias ? `${model} sekarang tampil sebagai "${alias}".` : `Alias ${model} dihapus.`)
      if (cfg.is_active) await refreshLLMState()
    } catch (err: any) {
      notify(`Gagal menyimpan alias: ${err?.data?.error || err.message || err}`, 'error')
    }
  }

  const confirmDelete = (cfg: LLMConfig) => {
    configToDelete.value = cfg
    isDeleteModalOpen.value = true
  }

  const handleDeleteConfig = async () => {
    if (!configToDelete.value) return
    deleting.value = true
    try {
      await deleteLLMConfig(configToDelete.value.id)
      notify(`Konfigurasi '${configToDelete.value.name}' telah dihapus dari grid.`)
      isDeleteModalOpen.value = false
      configToDelete.value = null
      await opts.reload()
      await refreshLLMState()
    } catch (err: any) {
      notify(`Gagal menghapus konfigurasi: ${err.message || err}`, 'error')
    } finally {
      deleting.value = false
    }
  }

  const handleTestConfig = async (cfg: LLMConfig) => {
    testingConfigId.value = cfg.id
    try {
      await triggerPing(cfg.id)
    } finally {
      testingConfigId.value = null
    }
  }

  const copiedUrl = ref<string | null>(null)
  const handleCopyUrl = async (url: string) => {
    try {
      await navigator.clipboard.writeText(url)
      copiedUrl.value = url
      setTimeout(() => {
        if (copiedUrl.value === url) copiedUrl.value = null
      }, 2000)
    } catch {
      // ignore
    }
  }

  return {
    testingConfigId, isModalOpen, isEditing, modalSaving, editingConfig, isDeleteModalOpen, configToDelete, deleting, copiedUrl,
    openAddModal, openEditModal, handleSaveConfig, handleActivateConfig, handleToggleVision, handleSetAlias,
    confirmDelete, handleDeleteConfig, handleTestConfig, handleCopyUrl
  }
}
