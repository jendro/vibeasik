<script setup lang="ts">
import type { LLMConfig } from '~/types'

defineProps<{
  activeViewTab: 'system_one' | 'standard_llm'
  loading: boolean
  configs: LLMConfig[]
  testingConfigId: number | null
  copiedUrl: string | null
}>()
const emit = defineEmits<{
  addConfig: [provider: string]
  activate: [cfg: LLMConfig]
  edit: [cfg: LLMConfig]
  delete: [cfg: LLMConfig]
  setAlias: [cfg: LLMConfig, model: string, alias: string]
  test: [cfg: LLMConfig]
  copyUrl: [url: string]
  toggleVision: [cfg: LLMConfig, model: string, vision: boolean]
}>()
const openAddModal = (provider: string) => emit('addConfig', provider)
</script>

<template>
  <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
    <div v-for="i in 3" :key="i" class="p-5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 animate-pulse space-y-3">
      <div class="h-4 bg-neutral-200 dark:bg-neutral-800 rounded w-1/2" />
      <div class="h-8 bg-neutral-100 dark:bg-neutral-800/60 rounded" />
      <div class="h-4 bg-neutral-200 dark:bg-neutral-800 rounded w-3/4" />
    </div>
  </div>

  <div
    v-else-if="configs.length === 0"
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
      v-for="item in configs"
      :key="item.id"
      :item="item"
      :testing="testingConfigId === item.id"
      :copied-url="copiedUrl"
      @activate="emit('activate', $event)"
      @edit="emit('edit', $event)"
      @delete="emit('delete', $event)"
      @set-alias="(c: LLMConfig, m: string, a: string) => emit('setAlias', c, m, a)"
      @test="emit('test', $event)"
      @copy-url="emit('copyUrl', $event)"
      @toggle-vision="(c: LLMConfig, m: string, v: boolean) => emit('toggleVision', c, m, v)"
    />
  </div>
  <SettingsLlmPoolCard v-if="activeViewTab === 'standard_llm' && !loading" class="mt-4" />
  <SettingsLlmWeighingCard v-if="!loading" class="mt-4" />
</template>
