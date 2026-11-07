<script setup lang="ts">
defineProps<{
  systemOneCount: number
  standardLlmCount: number
  keywordCount: number
}>()
const activeViewTab = defineModel<'system_one' | 'standard_llm' | 'censored_keys' | 'workspace_root'>('activeViewTab', { required: true })
const emit = defineEmits<{
  addConfig: [provider: string]
  addKeyword: []
}>()
const openAddModal = (provider: string) => emit('addConfig', provider)
const openAddKeywordModal = () => emit('addKeyword')
</script>

<template>
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
          {{ systemOneCount }}
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
          {{ standardLlmCount }}
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
          {{ keywordCount }}
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
</template>
