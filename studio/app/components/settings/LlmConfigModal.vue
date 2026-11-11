<script setup lang="ts">
import type { LLMConfig } from '~/types'

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  isEditing: boolean
  initialConfig?: Partial<LLMConfig> | null
  saving?: boolean
}>()

const emit = defineEmits<{
  save: [data: {
    id: number
    provider: 'system_one' | 'standard_llm'
    name: string
    url: string
    key: string
    models: string[]
    selected_model: string
    is_active: boolean
    aliases: Record<string, string>
  }]
}>()

const toast = useToast()
const { discoverLLMModels } = useApi()

const showKey = ref(false)
const newModelInput = ref('')

const form = ref<{
  id: number
  provider: 'system_one' | 'standard_llm'
  name: string
  url: string
  key: string
  masked_key: string
  models: string[]
  selected_model: string
  is_active: boolean
  aliases: Record<string, string>
}>({
  id: 0,
  provider: 'system_one',
  name: '',
  url: '',
  key: '',
  masked_key: '',
  models: [],
  selected_model: '',
  is_active: false,
  aliases: {}
})

const systemOnePresets = [
  { label: 'TypeSafe Jev (Cloud Official)', url: 'https://api.typesafe.ai/v1/systemone', model: 'jev-latest' },
  { label: 'Laya Local (:8750)', url: 'http://192.168.1.9:8750/v1/systemone', model: 'laya' },
  { label: 'Local Default (:8750)', url: 'http://localhost:8750/v1/systemone', model: 'convaiinnovations/laya-multilingual' }
]

const standardLLMPresets = [
  { label: '9Router Local (:20128)', name: '9Router Local', url: 'http://localhost:20128/v1', defaultModel: 'cbai/glm-5.1', models: ['cbai/glm-5.2', 'cbai/glm-5.1', 'cbai/glm-5.0', 'cbai/glm-5.0-turbo', 'cbai/glm-5v-turbo', 'cbai/glm-4.7', 'cbai/minimax-m3', 'cbai/minimax-m2.7', 'cbai/kimi-k2.7', 'cbai/kimi-k2.6', 'cbai/kimi-k2.5', 'cbai/hy3-preview', 'cbai/deepseek-v4-pro', 'cbai/deepseek-v4.1-flash', 'cbai/deepseek-v3-2-volc', 'mmf/mimo-auto', 'oc/muse-spark-1.2-contributor-free', 'oc/muse-spark-1.3-contributor-free', 'oc/union-alpha', 'oc/jev-1.13-free'] },
  { label: 'LimitRouter', url: 'https://limitrouter.com/v1', defaultModel: 'gemini-3.8-flash', models: ['gemini-3.8-flash', 'deepseek-v4.1-flash', 'claude-opus-5.5', 'kimi-k3'] },
  { label: 'OpenAI', url: 'https://api.openai.com/v1', defaultModel: 'gpt-4o-mini', models: ['gpt-4o-mini', 'gpt-4o', 'o1-mini'] },
  { label: 'OpenRouter', url: 'https://openrouter.ai/api/v1', defaultModel: 'deepseek/deepseek-chat', models: ['deepseek/deepseek-chat', 'anthropic/claude-3.5-sonnet', 'meta-llama/llama-3.3-70b-instruct'] },
  { label: 'Ollama Local (:11434)', url: 'http://localhost:11434/v1', defaultModel: 'qwen2.5-coder:7b', models: ['qwen2.5-coder:7b', 'llama3.2:3b', 'deepseek-r1:7b'] },
  { label: 'Groq Cloud', url: 'https://api.groq.com/openai/v1', defaultModel: 'llama-3.3-70b-versatile', models: ['llama-3.3-70b-versatile', 'mixtral-8x7b-32768'] }
]

watch(() => props.initialConfig, (cfg) => {
  if (cfg) {
    form.value = {
      id: cfg.id || 0,
      provider: cfg.provider || 'system_one',
      name: cfg.name || '',
      url: cfg.url || '',
      key: '',
      masked_key: cfg.masked_key || cfg.key || '',
      models: cfg.models?.length ? [...cfg.models] : (cfg.selected_model ? [cfg.selected_model] : []),
      selected_model: cfg.selected_model || (cfg.models?.[0] || ''),
      is_active: cfg.is_active || false,
      aliases: { ...(cfg.aliases || {}) }
    }
  }
}, { immediate: true })

const applyPreset = (preset: any) => {
  form.value.url = preset.url
  if (preset.name && !form.value.name) {
    form.value.name = preset.name
  }
  if (preset.models && preset.models.length > 0) {
    form.value.models = [...preset.models]
  } else if (preset.model) {
    form.value.models = [preset.model]
  }
  if (preset.defaultModel) {
    form.value.selected_model = preset.defaultModel
  } else if (preset.model) {
    form.value.selected_model = preset.model
  }
}

// Nama tampilan (display_name) hanya untuk vibeasik; yang dikirim ke LLM luar tetap id model.
const nameOf = (mod: string) => form.value.aliases[mod] || ''

const discovering = ref(false)
const discoverModels = async () => {
  if (!form.value.url.trim() && !form.value.id) {
    toast.add({ title: 'Isi Endpoint URL dulu', description: 'Daftar model diambil dari {URL}/models.', icon: 'i-lucide-alert-circle', color: 'warning' })
    return
  }
  discovering.value = true
  try {
    const res = await discoverLLMModels({ url: form.value.url.trim(), key: form.value.key.trim(), config_id: form.value.id || undefined })
    const list = res.models || []
    if (!list.length) {
      toast.add({ title: 'Katalog kosong', description: 'Provider tidak mengembalikan model.', icon: 'i-lucide-info', color: 'warning' })
      return
    }
    const ids = list.map(m => m.id)
    const added = ids.filter(id => !form.value.models.includes(id)).length
    form.value.models = [...form.value.models.filter(m => !ids.includes(m)), ...ids]
    const aliases = { ...form.value.aliases }
    for (const m of list) {
      if (m.display_name && !aliases[m.id]) aliases[m.id] = m.display_name
    }
    form.value.aliases = aliases
    if (!form.value.selected_model || !form.value.models.includes(form.value.selected_model)) {
      form.value.selected_model = form.value.models[0] || ''
    }
    toast.add({ title: 'Daftar model diambil', description: `${list.length} model dari provider (${added} baru).`, icon: 'i-lucide-check', color: 'success' })
  } catch (err: any) {
    toast.add({ title: 'Gagal mengambil daftar model', description: err?.data?.error || err?.message || String(err), icon: 'i-lucide-alert-circle', color: 'error' })
  } finally {
    discovering.value = false
  }
}

const addModel = () => {
  const trimmed = newModelInput.value.trim()
  if (!trimmed) return
  if (!form.value.models.includes(trimmed)) {
    form.value.models.push(trimmed)
    if (!form.value.selected_model) {
      form.value.selected_model = trimmed
    }
  }
  newModelInput.value = ''
}

const removeModel = (mod: string) => {
  if (form.value.models.length <= 1) {
    toast.add({
      title: 'Validasi Model',
      description: 'Minimal harus ada 1 model terdaftar untuk konfigurasi ini.',
      icon: 'i-lucide-alert-circle',
      color: 'warning'
    })
    return
  }
  form.value.models = form.value.models.filter(m => m !== mod)
  const { [mod]: _removed, ...rest } = form.value.aliases
  form.value.aliases = rest
  if (form.value.selected_model === mod) {
    form.value.selected_model = form.value.models[0] || ''
  }
}

const handleSubmit = () => {
  if (!form.value.name.trim() || !form.value.url.trim()) {
    toast.add({
      title: 'Validasi Form',
      description: 'Nama konfigurasi dan URL endpoint wajib diisi.',
      icon: 'i-lucide-alert-circle',
      color: 'warning'
    })
    return
  }
  if (form.value.models.length === 0) {
    toast.add({
      title: 'Validasi Form',
      description: 'Harap daftarkan minimal satu model yang bisa digunakan.',
      icon: 'i-lucide-alert-circle',
      color: 'warning'
    })
    return
  }

  emit('save', {
    id: form.value.id,
    provider: form.value.provider,
    name: form.value.name.trim(),
    url: form.value.url.trim(),
    key: form.value.key.trim(),
    models: form.value.models,
    selected_model: form.value.selected_model,
    is_active: form.value.is_active,
    aliases: Object.fromEntries(Object.entries(form.value.aliases).filter(([m, a]) => form.value.models.includes(m) && a.trim()))
  })
}
</script>

<template>
  <UModal
    v-model:open="open"
    :title="isEditing ? 'Edit Konfigurasi AI' : 'Tambah Konfigurasi AI Baru'"
    :description="isEditing ? 'Perbarui endpoint, URL, model, dan API key untuk konfigurasi ini.' : 'Daftarkan endpoint LLM baru ke dalam grid konfigurasi.'"
    :ui="{ content: 'sm:max-w-xl' }"
  >
    <template #body>
      <div class="space-y-4 text-xs">
        <!-- Provider Selector (Only when creating) -->
        <div v-if="!isEditing" class="space-y-1.5">
          <label class="block font-medium text-neutral-700 dark:text-neutral-300">Jenis Provider</label>
          <div class="grid grid-cols-2 gap-2">
            <button
              type="button"
              class="p-2.5 rounded-lg border text-left flex items-center gap-2 cursor-pointer transition-colors"
              :class="form.provider === 'system_one'
                ? 'border-sky-500 bg-sky-50/50 dark:bg-sky-950/40 text-sky-900 dark:text-sky-200 font-semibold'
                : 'border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800/40 text-neutral-700 dark:text-neutral-300'"
              @click="form.provider = 'system_one'"
            >
              <UIcon name="i-lucide-cpu" class="w-4 h-4 text-blue-500" />
              <span>System One / Jev</span>
            </button>

            <button
              type="button"
              class="p-2.5 rounded-lg border text-left flex items-center gap-2 cursor-pointer transition-colors"
              :class="form.provider === 'standard_llm'
                ? 'border-sky-500 bg-sky-50/50 dark:bg-sky-950/40 text-sky-900 dark:text-sky-200 font-semibold'
                : 'border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800/40 text-neutral-700 dark:text-neutral-300'"
              @click="form.provider = 'standard_llm'"
            >
              <UIcon name="i-lucide-bot" class="w-4 h-4 text-sky-500" />
              <span>LLM Biasa (OpenAI)</span>
            </button>
          </div>
        </div>

        <!-- Quick Presets -->
        <div class="space-y-1">
          <span class="text-xs font-medium text-neutral-500 dark:text-neutral-400">Preset Cepat:</span>
          <div class="flex flex-wrap gap-1.5">
            <button
              v-for="p in (form.provider === 'system_one' ? systemOnePresets : standardLLMPresets)"
              :key="p.label"
              type="button"
              class="px-2 py-0.5 text-xs font-mono rounded bg-neutral-100 hover:bg-neutral-200 dark:bg-neutral-800 dark:hover:bg-neutral-700 text-neutral-700 dark:text-neutral-300 border border-neutral-200 dark:border-neutral-700 transition-colors cursor-pointer"
              @click="applyPreset(p)"
            >
              {{ p.label }}
            </button>
          </div>
        </div>

        <!-- Nama Konfigurasi -->
        <div class="space-y-1">
          <label class="block font-medium text-neutral-700 dark:text-neutral-300">Nama Konfigurasi *</label>
          <input
            v-model="form.name"
            type="text"
            placeholder="Contoh: Local Laya Bare, OpenAI Production, Ollama Server"
            class="w-full px-3 py-2 text-xs rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-sky-500"
          >
        </div>

        <!-- URL Endpoint -->
        <div class="space-y-1">
          <label class="block font-medium text-neutral-700 dark:text-neutral-300">Endpoint URL *</label>
          <input
            v-model="form.url"
            type="text"
            placeholder="http://localhost:8750/v1/systemone atau https://api.openai.com/v1"
            class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-sky-500"
          >
        </div>

        <!-- API Key -->
        <div class="space-y-1">
          <div class="flex items-center justify-between">
            <label class="block font-medium text-neutral-700 dark:text-neutral-300">
              API Key / Auth Token (Selalu Termasking)
            </label>
            <span v-if="isEditing && form.masked_key" class="text-xs text-neutral-500 font-mono">
              Tersimpan: {{ form.masked_key }}
            </span>
          </div>
          <div class="relative">
            <input
              v-model="form.key"
              :type="showKey ? 'text' : 'password'"
              :placeholder="isEditing ? 'Biarkan kosong untuk mempertahankan key lama yang sudah termasking' : 'sk-... atau API Key Anda (opsional jika lokal)'"
              class="w-full pl-3 pr-10 py-2 text-xs font-mono rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-sky-500"
            >
            <button
              type="button"
              class="absolute right-2.5 top-2 text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 cursor-pointer"
              :title="showKey ? 'Sembunyikan' : 'Tampilkan teks'"
              @click="showKey = !showKey"
            >
              <UIcon :name="showKey ? 'i-lucide-eye-off' : 'i-lucide-eye'" class="w-4 h-4" />
            </button>
          </div>
          <p class="text-xs text-neutral-500 dark:text-neutral-400">
            Key selalu termasking saat dikembalikan dari server dan tidak akan terekspos di browser.
          </p>
        </div>

        <!-- Model yang Bisa Digunakan -->
        <div class="space-y-2 pt-1 border-t border-neutral-100 dark:border-white/10">
          <div class="flex items-center justify-between">
            <label class="block font-medium text-neutral-700 dark:text-neutral-300">
              Model yang Bisa Digunakan pada Konfigurasi Ini *
            </label>
            <UButton
              v-if="form.provider === 'standard_llm'"
              size="xs"
              color="neutral"
              variant="outline"
              icon="i-lucide-download"
              :loading="discovering"
              @click="discoverModels"
            >
              Ambil dari provider
            </UButton>
          </div>
          <p class="text-xs text-neutral-500 dark:text-neutral-400">
            Klik model untuk menjadikannya default. Nama tampilan diambil dari <code class="font-mono">display_name</code> katalog provider; yang dikirim ke LLM tetap id model.
          </p>

          <!-- List of Models -->
          <div class="flex flex-wrap gap-1.5 min-h-8 p-2 rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-200 dark:border-neutral-700">
            <span
              v-for="mod in form.models"
              :key="mod"
              class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs border transition-colors cursor-pointer select-none"
              :class="form.selected_model === mod
                ? 'bg-sky-100 dark:bg-sky-950 border-sky-400 dark:border-sky-700 text-sky-900 dark:text-sky-200 font-bold'
                : 'bg-white dark:bg-neutral-900 border-neutral-200 dark:border-neutral-700 text-neutral-700 dark:text-neutral-300'"
              @click="form.selected_model = mod"
            >
              <span v-if="nameOf(mod)" :title="mod">{{ nameOf(mod) }}</span>
              <span :class="nameOf(mod) ? 'font-mono text-neutral-400 dark:text-neutral-500' : 'font-mono'">{{ mod }}</span>
              <span v-if="form.selected_model === mod" class="text-xs uppercase font-sans text-sky-700 dark:text-sky-300">
                ✓
              </span>
              <button
                type="button"
                class="text-neutral-400 hover:text-rose-600 dark:hover:text-rose-400 ml-1 p-0.5 rounded cursor-pointer"
                title="Hapus model ini"
                @click.stop="removeModel(mod)"
              >
                <UIcon name="i-lucide-x" class="w-3 h-3" />
              </button>
            </span>

            <span v-if="form.models.length === 0" class="text-xs text-neutral-400 italic">
              Belum ada model ditambahkan
            </span>
          </div>

          <!-- Add Model Input -->
          <div class="flex items-center gap-2">
            <input
              v-model="newModelInput"
              type="text"
              placeholder="Ketik nama model (contoh: gpt-4o, claude-3-5-sonnet)..."
              class="flex-1 px-3 py-1.5 text-xs font-mono rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-sky-500"
              @keydown.enter.prevent="addModel"
            >
            <button
              type="button"
              class="px-3 py-1.5 text-xs font-medium rounded-lg bg-neutral-100 hover:bg-neutral-200 dark:bg-neutral-800 dark:hover:bg-neutral-700 text-neutral-700 dark:text-neutral-200 border border-neutral-200 dark:border-neutral-700 transition-colors inline-flex items-center gap-1 cursor-pointer"
              @click="addModel"
            >
              <UIcon name="i-lucide-plus" class="w-3.5 h-3.5" />
              <span>Tambah Model</span>
            </button>
          </div>
        </div>

        <!-- Active Checkbox -->
        <div class="flex items-center gap-2 pt-2">
          <input
            id="is_active_checkbox"
            v-model="form.is_active"
            type="checkbox"
            class="w-4 h-4 text-sky-600 rounded border-neutral-300 focus:ring-sky-500 cursor-pointer"
          >
          <label for="is_active_checkbox" class="text-xs text-neutral-700 dark:text-neutral-300 cursor-pointer select-none font-medium">
            Jadikan konfigurasi ini sebagai engine aktif untuk {{ form.provider === 'system_one' ? 'System One' : 'LLM Biasa' }}
          </label>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-end gap-2.5 w-full">
        <UButton color="neutral" variant="outline" @click="open = false">
          Batal
        </UButton>
        <UButton color="primary" :loading="saving" @click="handleSubmit">
          {{ isEditing ? 'Simpan Perubahan' : 'Tambahkan ke Grid' }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
