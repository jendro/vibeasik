<script setup lang="ts">
import type { LLMConfig, CensoredKeyword } from '~/types'
import { useLLMState } from '~/composables/useLLMState'

const route = useRoute()

const {
  getLLMSettings,
  getLLMConfigs,
  createLLMConfig,
  updateLLMConfig,
  setLLMModelVision,
  setLLMModelAlias,
  deleteLLMConfig,
  activateLLMConfig,
  setActiveProvider,
  getCensoredKeywords,
  createCensoredKeyword,
  updateCensoredKeyword,
  deleteCensoredKeyword,
  getWorkspaceSettings,
  updateWorkspaceSettings
} = useApi()

const { triggerPing, refreshLLMState } = useLLMState()

// Page state
const loading = ref(true)
const saving = ref(false)
const savingWorkspace = ref(false)
const testingConfigId = ref<number | null>(null)
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

// Modal Add/Edit Config
const isModalOpen = ref(false)
const isEditing = ref(false)
const modalSaving = ref(false)
const editingConfig = ref<Partial<LLMConfig> | null>(null)

// Modal Delete Config
const isDeleteModalOpen = ref(false)
const configToDelete = ref<LLMConfig | null>(null)
const deleting = ref(false)

// Modal Add/Edit Keyword
const isKeywordModalOpen = ref(false)
const isEditingKeyword = ref(false)
const keywordModalSaving = ref(false)
const editingKeyword = ref<Partial<CensoredKeyword> | null>(null)

// Modal Delete Keyword
const isDeleteKeywordModalOpen = ref(false)
const keywordToDelete = ref<CensoredKeyword | null>(null)
const deletingKeyword = ref(false)

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
    await loadData()
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
    await loadData()
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
    await loadData()
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

// Keyword Modal Handlers
const openAddKeywordModal = (preset?: any) => {
  isEditingKeyword.value = false
  editingKeyword.value = preset
    ? {
        id: 0,
        keyword: preset.keyword,
        category: preset.category,
        description: preset.description,
        is_regex: preset.is_regex,
        is_active: true
      }
    : {
        id: 0,
        keyword: '',
        category: 'API_KEY',
        description: '',
        is_regex: false,
        is_active: true
      }
  isKeywordModalOpen.value = true
}

const openEditKeywordModal = (kw: CensoredKeyword) => {
  isEditingKeyword.value = true
  editingKeyword.value = { ...kw }
  isKeywordModalOpen.value = true
}

const handleSaveKeyword = async (formData: any) => {
  keywordModalSaving.value = true
  try {
    if (isEditingKeyword.value && formData.id) {
      await updateCensoredKeyword(formData.id, {
        keyword: formData.keyword,
        category: formData.category,
        description: formData.description,
        is_regex: formData.is_regex,
        is_active: formData.is_active
      })
      notify(`Keyword '${formData.keyword}' berhasil diperbarui.`)
    } else {
      await createCensoredKeyword({
        keyword: formData.keyword,
        category: formData.category,
        description: formData.description,
        is_regex: formData.is_regex,
        is_active: formData.is_active
      })
      notify(`Keyword '${formData.keyword}' berhasil ditambahkan ke vault.`)
    }
    isKeywordModalOpen.value = false
    await loadData()
  } catch (err: any) {
    notify(`Gagal menyimpan keyword sensor: ${err.message || err}`, 'error')
  } finally {
    keywordModalSaving.value = false
  }
}

const handleToggleKeywordStatus = async (kw: CensoredKeyword) => {
  try {
    await updateCensoredKeyword(kw.id, {
      ...kw,
      is_active: !kw.is_active
    })
    kw.is_active = !kw.is_active
    notify(`Status keyword '${kw.keyword}' diubah menjadi: ${kw.is_active ? 'Aktif' : 'Nonaktif'}`)
  } catch (err: any) {
    notify(`Gagal mengubah status: ${err.message || err}`, 'error')
  }
}

const confirmDeleteKeyword = (kw: CensoredKeyword) => {
  keywordToDelete.value = kw
  isDeleteKeywordModalOpen.value = true
}

const handleDeleteKeyword = async () => {
  if (!keywordToDelete.value) return
  deletingKeyword.value = true
  try {
    await deleteCensoredKeyword(keywordToDelete.value.id)
    notify(`Keyword '${keywordToDelete.value.keyword}' telah dihapus dari vault.`)
    isDeleteKeywordModalOpen.value = false
    keywordToDelete.value = null
    await loadData()
  } catch (err: any) {
    notify(`Gagal menghapus keyword: ${err.message || err}`, 'error')
  } finally {
    deletingKeyword.value = false
  }
}
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
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-2 border-b border-neutral-200 dark:border-white/10">
        <!-- View Tabs -->
        <div class="flex items-center gap-1.5 p-1 rounded-lg bg-neutral-100 dark:bg-neutral-800/80 w-fit">
          <button
            type="button"
            class="px-3 py-1.5 rounded-md text-xs font-medium transition-all inline-flex items-center gap-2 cursor-pointer"
            :class="activeViewTab === 'system_one'
              ? 'bg-white dark:bg-slate-900/40 text-neutral-900 dark:text-white shadow-xs font-semibold'
              : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="activeViewTab = 'system_one'"
          >
            <UIcon name="i-lucide-cpu" class="w-3.5 h-3.5 text-blue-500" />
            <span>Grid System One</span>
            <span class="px-1.5 py-0.5 rounded-full text-[10px] bg-neutral-200 dark:bg-neutral-700 font-mono">
              {{ systemOneConfigs.length }}
            </span>
          </button>

          <button
            type="button"
            class="px-3 py-1.5 rounded-md text-xs font-medium transition-all inline-flex items-center gap-2 cursor-pointer"
            :class="activeViewTab === 'standard_llm'
              ? 'bg-white dark:bg-slate-900/40 text-neutral-900 dark:text-white shadow-xs font-semibold'
              : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="activeViewTab = 'standard_llm'"
          >
            <UIcon name="i-lucide-bot" class="w-3.5 h-3.5 text-sky-500" />
            <span>Grid LLM Biasa</span>
            <span class="px-1.5 py-0.5 rounded-full text-[10px] bg-neutral-200 dark:bg-neutral-700 font-mono">
              {{ standardLLMConfigs.length }}
            </span>
          </button>

          <button
            type="button"
            class="px-3 py-1.5 rounded-md text-xs font-medium transition-all inline-flex items-center gap-2 cursor-pointer"
            :class="activeViewTab === 'censored_keys'
              ? 'bg-white dark:bg-slate-900/40 text-amber-700 dark:text-amber-300 shadow-xs font-semibold'
              : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="activeViewTab = 'censored_keys'"
          >
            <UIcon name="i-lucide-shield-alert" class="w-3.5 h-3.5 text-amber-500" />
            <span>Sensored Keys Vault</span>
            <span class="px-1.5 py-0.5 rounded-full text-[10px] bg-amber-100 dark:bg-amber-950/60 text-amber-800 dark:text-amber-300 font-mono">
              {{ censoredKeywords.length }}
            </span>
          </button>

          <button
            type="button"
            class="px-3 py-1.5 rounded-md text-xs font-medium transition-all inline-flex items-center gap-2 cursor-pointer"
            :class="activeViewTab === 'workspace_root'
              ? 'bg-white dark:bg-slate-900/40 text-emerald-700 dark:text-emerald-400 shadow-xs font-semibold'
              : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="activeViewTab = 'workspace_root'"
          >
            <UIcon name="i-lucide-folder-tree" class="w-3.5 h-3.5 text-emerald-500" />
            <span>Folder Induk Proyek</span>
          </button>
        </div>

        <!-- Add Button -->
        <div class="flex items-center gap-2">
          <button
            v-if="activeViewTab === 'system_one' || activeViewTab === 'standard_llm'"
            type="button"
            class="text-xs px-3 py-1.5 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-slate-900/40 hover:bg-neutral-50 dark:hover:bg-neutral-800 text-neutral-700 dark:text-neutral-300 inline-flex items-center gap-1.5 transition-colors cursor-pointer"
            @click="openAddModal(activeViewTab)"
          >
            <UIcon name="i-lucide-plus" class="w-3.5 h-3.5 text-sky-600 dark:text-sky-400" />
            <span>Tambah ke Grid Ini</span>
          </button>

          <button
            v-else-if="activeViewTab === 'censored_keys'"
            type="button"
            class="text-xs px-3.5 py-1.5 rounded-lg border border-amber-400 dark:border-amber-600 bg-amber-600 hover:bg-amber-700 text-white font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer shadow-xs"
            @click="openAddKeywordModal()"
          >
            <UIcon name="i-lucide-plus" class="w-3.5 h-3.5" />
            <span>Tambah Keyword Sensor</span>
          </button>
        </div>
      </div>

      <!-- Config Grid (System One / Standard LLM) -->
      <template v-if="activeViewTab === 'system_one' || activeViewTab === 'standard_llm'">
        <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <div v-for="i in 3" :key="i" class="p-5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 animate-pulse space-y-3">
            <div class="h-4 bg-neutral-200 dark:bg-neutral-800 rounded w-1/2" />
            <div class="h-8 bg-neutral-100 dark:bg-neutral-800/60 rounded" />
            <div class="h-4 bg-neutral-200 dark:bg-neutral-800 rounded w-3/4" />
          </div>
        </div>

        <div
          v-else-if="currentGridConfigs.length === 0"
          class="p-12 rounded-xl border border-dashed border-neutral-300 dark:border-white/10 bg-white/60 dark:bg-slate-900/60 text-center space-y-3"
        >
          <div class="w-10 h-10 rounded-full bg-neutral-100 dark:bg-neutral-800 text-neutral-400 flex items-center justify-center mx-auto">
            <UIcon name="i-lucide-server-crash" class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-sm font-semibold text-neutral-800 dark:text-neutral-200">
              Belum ada konfigurasi untuk {{ activeViewTab === 'system_one' ? 'System One' : 'LLM Biasa' }}
            </h3>
            <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1 max-w-sm mx-auto">
              Tambahkan konfigurasi baru untuk menyimpan endpoint URL, API key, dan daftar model yang didukung.
            </p>
          </div>
          <button
            type="button"
            class="inline-flex items-center gap-1.5 px-3.5 py-1.5 bg-sky-600 hover:bg-sky-500 text-white text-xs font-medium rounded-lg transition-colors cursor-pointer"
            @click="openAddModal(activeViewTab)"
          >
            <UIcon name="i-lucide-plus" class="w-3.5 h-3.5" />
            <span>Tambah Konfigurasi Pertama</span>
          </button>
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4.5">
          <SettingsLlmConfigCard
            v-for="item in currentGridConfigs"
            :key="item.id"
            :item="item"
            :testing="testingConfigId === item.id"
            :copied-url="copiedUrl"
            @activate="handleActivateConfig"
            @edit="openEditModal"
            @delete="confirmDelete"
            @set-alias="handleSetAlias"
            @test="handleTestConfig"
            @copy-url="handleCopyUrl"
            @toggle-vision="handleToggleVision"
          />
        </div>
        <SettingsLlmPoolCard v-if="activeViewTab === 'standard_llm' && !loading" class="mt-4" />
        <SettingsLlmWeighingCard v-if="!loading" class="mt-4" />
      </template>

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
      <div v-else-if="activeViewTab === 'workspace_root'" class="space-y-6 max-w-4xl">
        <div class="p-6 rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs space-y-5">
          <div class="flex items-start gap-3.5">
            <div class="w-10 h-10 rounded-xl bg-emerald-50 dark:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 flex items-center justify-center shrink-0">
              <UIcon name="i-lucide-folder-cog" class="w-5 h-5" />
            </div>
            <div>
              <h3 class="text-base font-semibold text-neutral-900 dark:text-white">
                Folder Induk Proyek (Default Root Projects)
              </h3>
              <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1">
                Lokasi utama tempat semua proyek terintegrasi dikumpulkan dalam 1 folder induk (misalnya <code class="font-mono text-emerald-600 dark:text-emerald-400">~/Documents/Projects</code>). Folder induk ini menjadi jangkar ekosistem AI Agent untuk multi-service atau fullstack.
              </p>
            </div>
          </div>

          <div class="space-y-2">
            <label class="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
              Absolute Directory Path
            </label>
            <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
              <div class="relative flex-1">
                <input
                  v-model="defaultProjectsRoot"
                  type="text"
                  placeholder="~/Documents/Projects"
                  class="w-full px-3.5 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-slate-900 text-sm font-mono text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                >
              </div>
              <button
                type="button"
                :disabled="savingWorkspace"
                class="px-5 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-xs font-semibold inline-flex items-center justify-center gap-2 transition-colors cursor-pointer shadow-xs shrink-0"
                @click="handleSaveWorkspaceSettings"
              >
                <UIcon v-if="savingWorkspace" name="i-lucide-loader-2" class="w-4 h-4 animate-spin" />
                <UIcon v-else name="i-lucide-save" class="w-4 h-4" />
                <span>{{ savingWorkspace ? 'Menyimpan...' : 'Simpan Lokasi' }}</span>
              </button>
            </div>
            <p class="text-[11px] text-neutral-500 dark:text-neutral-400">
              Setiap kali Anda menekan tombol "Satukan Folder" pada kartu atau detail proyek, dialog pemindahan akan otomatis mengusulkan subdirektori di bawah folder induk ini.
            </p>
          </div>
        </div>

        <!-- Info / Benefits Card -->
        <div class="p-5 rounded-2xl border border-neutral-200 dark:border-white/10 bg-neutral-50/70 dark:bg-neutral-800/40 space-y-3">
          <h4 class="text-xs font-semibold text-neutral-800 dark:text-neutral-200 flex items-center gap-2">
            <UIcon name="i-lucide-sparkles" class="w-4 h-4 text-emerald-500" />
            Keuntungan Arsitektur 1 Folder Induk bagi AI Agent:
          </h4>
          <ul class="text-xs text-neutral-600 dark:text-neutral-300 space-y-2 list-disc pl-5">
            <li>
              <strong>Konteks Penuh Fullstack:</strong> AI Agent dapat membaca skema backend (misal Go, Python) dan frontend (misal Nuxt, Flutter) secara bersamaan dalam satu sesi.
            </li>
            <li>
              <strong>Tautan Otomatis:</strong> File konfigurasi <code class="font-mono text-emerald-600 dark:text-emerald-400">.vibeasik.json</code> akan diletakkan langsung di akar folder induk ini.
            </li>
            <li>
              <strong>Git History Tetap Utuh:</strong> Folder komponen backend/frontend tetap menjaga repositori <code class="font-mono">.git</code> masing-masing tanpa merusak riwayat commit.
            </li>
          </ul>
        </div>
      </div>
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
