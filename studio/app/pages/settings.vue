<script setup lang="ts">
import type { LLMConfig, CensoredKeyword } from '~/types'

const route = useRoute()

const {
  getLLMSettings,
  getLLMConfigs,
  setActiveProvider,
  getCensoredKeywords,
  getWorkspaceSettings,
  updateWorkspaceSettings
} = useApi()

// Page state
const loading = ref(true)
const saving = ref(false)
const savingWorkspace = ref(false)
const bannerMessage = ref<string | null>(null)
const bannerType = ref<'success' | 'info' | 'error'>('success')

// Active engine provider & tab
const activeProvider = ref<'system_one' | 'standard_llm'>('system_one')
const activeViewTab = ref<'system_one' | 'standard_llm' | 'censored_keys' | 'workspace_root'>('system_one')
const defaultProjectsRoot = ref('')

// All configs from BE
const configs = ref<LLMConfig[]>([])
const systemOneConfigs = computed(() => configs.value.filter(c => c.provider === 'system_one'))
const standardLLMConfigs = computed(() => configs.value.filter(c => c.provider === 'standard_llm'))
const currentGridConfigs = computed(() =>
  activeViewTab.value === 'system_one' ? systemOneConfigs.value : standardLLMConfigs.value
)

// Censored keywords state
const censoredKeywords = ref<CensoredKeyword[]>([])
const builtInPatterns = ref<string[]>([])

const toast = useToast()

const notify = (msg: string, type: 'success' | 'info' | 'error' = 'success') => {
  bannerMessage.value = msg
  bannerType.value = type
  toast.add({
    title: type === 'error' ? 'Pemberitahuan Gagal' : type === 'info' ? 'Informasi' : 'Berhasil',
    description: msg,
    icon: type === 'error' ? 'i-lucide-alert-circle' : type === 'info' ? 'i-lucide-info' : 'i-lucide-check-circle',
    color: type === 'error' ? 'error' : type === 'info' ? 'info' : 'success'
  })
  setTimeout(() => {
    if (bannerMessage.value === msg) {
      bannerMessage.value = null
    }
  }, 4000)
}

const loadData = async () => {
  loading.value = true
  try {
    const [settingsData, configsData, securityData, wsSettings] = await Promise.all([
      getLLMSettings(),
      getLLMConfigs(),
      getCensoredKeywords().catch((err) => {
        console.warn('Censored keywords not loaded:', err)
        return null
      }),
      getWorkspaceSettings().catch((err) => {
        console.warn('Workspace settings not loaded:', err)
        return null
      })
    ])
    if (settingsData?.active_provider) {
      activeProvider.value = settingsData.active_provider
      if (route.query.tab !== 'censored_keys' && route.query.tab !== 'workspace_root') {
        activeViewTab.value = settingsData.active_provider
      }
    }
    configs.value = configsData || []

    if (securityData) {
      censoredKeywords.value = securityData.keywords || []
      builtInPatterns.value = securityData.built_in_patterns || []
    }

    if (wsSettings?.default_projects_root) {
      defaultProjectsRoot.value = wsSettings.default_projects_root
    }

    if (route.query.tab === 'censored_keys') {
      activeViewTab.value = 'censored_keys'
    } else if (route.query.tab === 'workspace_root') {
      activeViewTab.value = 'workspace_root'
    }
  } catch (err: any) {
    console.error('Failed to load LLM settings:', err)
    notify(`Gagal memuat pengaturan: ${err.message || err}`, 'error')
  } finally {
    loading.value = false
  }
}

watch(() => route.query.tab, (tab) => {
  if (tab === 'censored_keys') {
    activeViewTab.value = 'censored_keys'
  } else if (tab === 'workspace_root') {
    activeViewTab.value = 'workspace_root'
  }
})

onMounted(() => {
  loadData()
})

watch(() => route.query.tab, (tab) => {
  if (tab === 'censored_keys') {
    activeViewTab.value = 'censored_keys'
  } else if (tab === 'workspace_root') {
    activeViewTab.value = 'workspace_root'
  }
})

onMounted(() => {
  loadData()
})

const handleSaveWorkspaceSettings = async () => {
  if (!defaultProjectsRoot.value?.trim()) {
    notify('Path folder induk proyek tidak boleh kosong', 'error')
    return
  }
  savingWorkspace.value = true
  try {
    const res = await updateWorkspaceSettings({
      default_projects_root: defaultProjectsRoot.value.trim()
    })
    if (res?.default_projects_root) {
      defaultProjectsRoot.value = res.default_projects_root
    }
    notify('Folder induk proyek berhasil disimpan!')
  } catch (err: any) {
    notify(`Gagal menyimpan folder induk: ${err.message || err}`, 'error')
  } finally {
    savingWorkspace.value = false
  }
}

// Provider switcher
const handleSelectActiveProvider = async (provider: 'system_one' | 'standard_llm') => {
  if (activeProvider.value === provider) {
    activeViewTab.value = provider
    return
  }
  saving.value = true
  try {
    await setActiveProvider(provider)
    activeProvider.value = provider
    activeViewTab.value = provider
    notify(`Provider AI aktif dialihkan ke: ${provider === 'system_one' ? 'System One / Jev' : 'LLM Biasa (OpenAI API)'}`)
  } catch (err: any) {
    notify(`Gagal mengubah provider aktif: ${err.message || err}`, 'error')
  } finally {
    saving.value = false
  }
}

// Dialog & aksi grid konfigurasi LLM serta keyword sensor (state + API di composable)
const {
  testingConfigId, isModalOpen, isEditing, modalSaving, editingConfig, isDeleteModalOpen, configToDelete, deleting, copiedUrl,
  openAddModal, openEditModal, handleSaveConfig, handleActivateConfig, handleToggleVision, handleSetAlias,
  confirmDelete, handleDeleteConfig, handleTestConfig, handleCopyUrl
} = useSettingsConfigs({ notify, reload: loadData, configs, activeViewTab, systemOneConfigs, standardLLMConfigs })

const {
  isKeywordModalOpen, isEditingKeyword, keywordModalSaving, editingKeyword,
  isDeleteKeywordModalOpen, keywordToDelete, deletingKeyword,
  openAddKeywordModal, openEditKeywordModal, handleSaveKeyword, handleToggleKeywordStatus, confirmDeleteKeyword, handleDeleteKeyword
} = useSettingsKeywords(notify, loadData)
</script>

<template>
  <div class="space-y-6">
    <!-- Header with Provider Selector -->
    <SettingsHeader
      :active-provider="activeProvider"
      :saving="saving"
      :banner-message="bannerMessage"
      :banner-type="bannerType"
      @select-provider="handleSelectActiveProvider"
    />

    <!-- View Switcher & Tabs Toolbar -->
    <div class="space-y-4 pt-2">
      <SettingsTabsToolbar
        v-model:active-view-tab="activeViewTab"
        :system-one-count="systemOneConfigs.length"
        :standard-llm-count="standardLLMConfigs.length"
        :keyword-count="censoredKeywords.length"
        @add-config="openAddModal"
        @add-keyword="openAddKeywordModal()"
      />

      <!-- Config Grid (System One / Standard LLM) -->
      <SettingsConfigGrid
        v-if="activeViewTab === 'system_one' || activeViewTab === 'standard_llm'"
        :active-view-tab="activeViewTab"
        :loading="loading"
        :configs="currentGridConfigs"
        :testing-config-id="testingConfigId"
        :copied-url="copiedUrl"
        @add-config="openAddModal"
        @activate="handleActivateConfig"
        @edit="openEditModal"
        @delete="confirmDelete"
        @set-alias="handleSetAlias"
        @test="handleTestConfig"
        @copy-url="handleCopyUrl"
        @toggle-vision="handleToggleVision"
      />

      <!-- Censored Keywords Tab Component -->
      <SettingsCensoredKeywordsTab
        v-else-if="activeViewTab === 'censored_keys'"
        :keywords="censoredKeywords"
        :built-in-patterns="builtInPatterns"
        :loading="loading"
        @add="openAddKeywordModal()"
        @edit="openEditKeywordModal"
        @delete="confirmDeleteKeyword"
        @toggle-active="handleToggleKeywordStatus"
        @apply-preset="openAddKeywordModal"
      />

      <!-- Workspace Root Settings Tab Component -->
      <SettingsWorkspaceTab
        v-else-if="activeViewTab === 'workspace_root'"
        v-model:root="defaultProjectsRoot"
        :saving="savingWorkspace"
        @save="handleSaveWorkspaceSettings"
      />
    </div>

    <!-- Modals -->
    <SettingsLlmConfigModal
      v-model:open="isModalOpen"
      :is-editing="isEditing"
      :initial-config="editingConfig"
      :saving="modalSaving"
      @save="handleSaveConfig"
    />

    <SettingsLlmConfigDeleteModal
      v-model:open="isDeleteModalOpen"
      :config="configToDelete"
      :deleting="deleting"
      @confirm="handleDeleteConfig"
    />

    <SettingsCensoredKeywordModal
      v-model:open="isKeywordModalOpen"
      :is-editing="isEditingKeyword"
      :initial-keyword="editingKeyword"
      :saving="keywordModalSaving"
      @save="handleSaveKeyword"
    />

    <SettingsCensoredKeywordDeleteModal
      v-model:open="isDeleteKeywordModalOpen"
      :keyword="keywordToDelete"
      :deleting="deletingKeyword"
      @confirm="handleDeleteKeyword"
    />
  </div>
</template>
