<script setup lang="ts">
import type { LLMConfig } from '~/types'

const props = defineProps<{
  item: LLMConfig
  testing?: boolean
  copiedUrl?: string | null
}>()

const emit = defineEmits<{
  activate: [item: LLMConfig]
  edit: [item: LLMConfig]
  delete: [item: LLMConfig]
  test: [item: LLMConfig]
  copyUrl: [url: string]
  toggleVision: [item: LLMConfig, model: string, vision: boolean]
  setAlias: [item: LLMConfig, model: string, alias: string]
}>()

// Alias tampilan per model: nama model asli dari penyedia pihak ketiga bisa berbeda-beda.
const aliasOf = (mod: string) => props.item.aliases?.[mod] || ''
const editingAlias = ref<string | null>(null)
const aliasDraft = ref('')
const startAlias = (mod: string) => {
  editingAlias.value = mod
  aliasDraft.value = aliasOf(mod)
}
const saveAlias = (mod: string) => {
  if (editingAlias.value !== mod) return
  editingAlias.value = null
  const alias = aliasDraft.value.trim()
  if (alias !== aliasOf(mod)) emit('setAlias', props.item, mod, alias)
}

const supportsVision = (mod: string) => Boolean(props.item.vision?.[mod])
const visionTitle = (mod: string) => {
  const manual = props.item.vision_manual?.includes(mod)
    ? 'diatur manual'
    : props.item.vision_tested?.includes(mod) ? 'hasil uji gambar otomatis' : 'tebakan dari nama model, belum diuji'
  return supportsVision(mod)
    ? `Mendukung gambar/vision (${manual}). Klik untuk menandai tidak mendukung.`
    : `Tidak mendukung gambar (${manual}). Klik untuk menandai mendukung vision.`
}
</script>

<template>
  <div
    class="rounded-xl border bg-white dark:bg-neutral-900 p-4.5 shadow-xs flex flex-col justify-between transition-all relative hover:border-neutral-300 dark:hover:border-neutral-700 space-y-4"
    :class="item.is_active
      ? 'border-sky-500 dark:border-sky-500/80 ring-1 ring-sky-500/30'
      : 'border-neutral-200 dark:border-white/10'"
  >
    <!-- Card Header -->
    <div class="space-y-3">
      <div class="flex items-start justify-between gap-2">
        <div class="flex items-center gap-2 min-w-0">
          <div
            class="w-7 h-7 rounded-lg flex items-center justify-center text-xs shrink-0"
            :class="item.provider === 'system_one'
              ? 'bg-blue-100 text-blue-700 dark:bg-blue-950/60 dark:text-blue-300'
              : 'bg-sky-100 text-sky-700 dark:bg-sky-950/60 dark:text-sky-300'"
          >
            <UIcon :name="item.provider === 'system_one' ? 'i-lucide-cpu' : 'i-lucide-bot'" class="w-4 h-4" />
          </div>
          <div class="min-w-0">
            <h4 class="text-xs font-bold text-neutral-900 dark:text-white truncate" :title="item.name">
              {{ item.name }}
            </h4>
            <span class="text-xs text-neutral-500 dark:text-neutral-400 font-mono">
              ID #{{ item.id }}
            </span>
          </div>
        </div>

        <!-- Active Status Badge / Switch -->
        <div>
          <span
            v-if="item.is_active"
            class="px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-sky-100 text-sky-800 dark:bg-sky-950 dark:text-sky-300 inline-flex items-center gap-1 shrink-0"
          >
            <span class="w-1.5 h-1.5 rounded-full bg-sky-500" />
            Aktif
          </span>
          <button
            v-else
            type="button"
            class="px-2 py-0.5 rounded text-xs font-medium text-neutral-600 dark:text-neutral-400 hover:text-sky-700 dark:hover:text-sky-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 transition-colors shrink-0 cursor-pointer"
            title="Jadikan konfigurasi ini aktif"
            @click="$emit('activate', item)"
          >
            Gunakan
          </button>
        </div>
      </div>

      <!-- URL Display with Copy Button -->
      <div class="space-y-1">
        <span class="text-xs uppercase font-semibold text-neutral-500 dark:text-neutral-400 tracking-wider">
          Endpoint URL
        </span>
        <div class="flex items-center justify-between gap-1 px-2.5 py-1.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200/80 dark:border-white/10 text-xs font-mono text-neutral-800 dark:text-neutral-200">
          <span class="truncate text-xs" :title="item.url">{{ item.url }}</span>
          <button
            type="button"
            class="text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 p-0.5 rounded transition-colors shrink-0 cursor-pointer"
            :title="copiedUrl === item.url ? 'Disalin!' : 'Salin URL'"
            @click="$emit('copyUrl', item.url)"
          >
            <UIcon :name="copiedUrl === item.url ? 'i-lucide-check' : 'i-lucide-copy'" class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      <!-- Masked API Key Section -->
      <div class="space-y-1">
        <span class="text-xs uppercase font-semibold text-neutral-500 dark:text-neutral-400 tracking-wider">
          API Key / Auth Token
        </span>
        <div class="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200/80 dark:border-white/10 text-xs">
          <div class="flex items-center gap-1.5 min-w-0">
            <UIcon name="i-lucide-key-round" class="w-3.5 h-3.5 text-neutral-400 shrink-0" />
            <span
              v-if="item.has_key"
              class="font-mono text-xs text-neutral-700 dark:text-neutral-300 tracking-widest truncate"
              title="API key tersimpan dan selalu termasking demi keamanan"
            >
              {{ item.masked_key || '••••••••••••' }}
            </span>
            <span v-else class="text-xs text-neutral-400 dark:text-neutral-500 italic">
              Tanpa Key (Auth Bebas / Server Lokal)
            </span>
          </div>
          <span
            v-if="item.has_key"
            class="text-xs uppercase px-1.5 py-0.5 rounded bg-neutral-200 dark:bg-neutral-700 text-neutral-600 dark:text-neutral-300 font-mono shrink-0"
          >
            Masked
          </span>
        </div>
      </div>

      <!-- Usable Models Section -->
      <div class="space-y-1.5">
        <div class="flex items-center justify-between text-xs">
          <span class="uppercase font-semibold text-neutral-500 dark:text-neutral-400 tracking-wider">
            Model yang Bisa Digunakan ({{ item.models?.length || 1 }})
          </span>
          <span v-if="item.provider === 'standard_llm'" class="inline-flex items-center gap-0.5 text-emerald-600 dark:text-emerald-400" title="Model berikon mata mendukung gambar (dipakai Galeri Desain)">
            <UIcon name="i-lucide-eye" class="w-3 h-3" /> {{ item.models.filter(m => item.vision?.[m]).length }} vision
          </span>
          <span v-if="item.selected_model" class="text-neutral-400 font-mono">
            Default: {{ aliasOf(item.selected_model) || item.selected_model }}
          </span>
        </div>

        <!-- Model Chips -->
        <div class="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto">
          <span
            v-for="mod in item.models"
            :key="mod"
            class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-mono border transition-colors"
            :class="item.selected_model === mod
              ? 'bg-sky-50 dark:bg-sky-950/60 border-sky-300 dark:border-sky-700 text-sky-800 dark:text-sky-300 font-semibold'
              : 'bg-neutral-100 dark:bg-neutral-800 border-neutral-200 dark:border-neutral-700 text-neutral-700 dark:text-neutral-300'"
          >
            <span
              class="w-1.5 h-1.5 rounded-full"
              :class="item.selected_model === mod ? 'bg-sky-500' : 'bg-neutral-400 dark:bg-neutral-500'"
            />
            <input
              v-if="editingAlias === mod"
              v-model="aliasDraft"
              type="text"
              class="w-28 px-1 py-0 rounded border border-sky-300 dark:border-sky-700 bg-white dark:bg-neutral-900 font-sans text-xs focus:outline-none focus:ring-1 focus:ring-sky-500"
              :placeholder="mod"
              :aria-label="`Alias untuk ${mod}`"
              autofocus
              @keydown.enter.prevent="saveAlias(mod)"
              @keydown.esc.prevent="editingAlias = null"
              @blur="saveAlias(mod)"
            >
            <span v-else-if="aliasOf(mod)" :title="`Alias untuk ${mod}`">
              <span class="font-sans font-semibold">{{ aliasOf(mod) }}</span>
              <span class="opacity-60"> · {{ mod }}</span>
            </span>
            <span v-else>{{ mod }}</span>
            <button
              v-if="editingAlias !== mod"
              type="button"
              class="inline-flex items-center rounded p-0.5 text-neutral-400 hover:text-sky-600 hover:bg-neutral-200 dark:hover:bg-neutral-700 transition-colors cursor-pointer"
              :title="aliasOf(mod) ? `Ubah alias (${aliasOf(mod)}); kosongkan untuk menghapus` : 'Beri alias: nama yang tampil di pemilih model & Kantor Agent'"
              :aria-label="`Atur alias ${mod}`"
              @click="startAlias(mod)"
            >
              <UIcon name="i-lucide-tag" class="w-3 h-3" />
            </button>
            <span v-if="item.selected_model === mod" class="text-xs uppercase text-sky-600 dark:text-sky-400 font-sans font-bold">
              (default)
            </span>
            <button
              v-if="item.provider === 'standard_llm'"
              type="button"
              class="ml-0.5 inline-flex items-center rounded p-0.5 transition-colors cursor-pointer"
              :class="supportsVision(mod) ? 'text-emerald-600 dark:text-emerald-400 hover:bg-emerald-100 dark:hover:bg-emerald-950/50' : 'text-neutral-400 hover:text-neutral-600 hover:bg-neutral-200 dark:hover:bg-neutral-700'"
              :title="visionTitle(mod)"
              :aria-label="visionTitle(mod)"
              @click="emit('toggleVision', item, mod, !supportsVision(mod))"
            >
              <UIcon :name="supportsVision(mod) ? 'i-lucide-eye' : 'i-lucide-eye-off'" class="w-3 h-3" />
            </button>
          </span>
        </div>
      </div>
    </div>

    <!-- Card Action Buttons Footer -->
    <div class="pt-3 border-t border-neutral-200 dark:border-white/10 flex items-center justify-between gap-2">
      <!-- Test Connection Button -->
      <button
        type="button"
        class="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg border border-neutral-200 dark:border-neutral-700 hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-700 dark:text-neutral-300 transition-colors disabled:opacity-50 cursor-pointer"
        :disabled="testing"
        @click="$emit('test', item)"
      >
        <UIcon
          :name="testing ? 'i-lucide-loader-2' : 'i-lucide-activity'"
          class="w-3.5 h-3.5 text-neutral-500"
          :class="testing ? 'animate-spin' : ''"
        />
        <span>{{ testing ? 'Testing...' : 'Test Ping' }}</span>
      </button>

      <div class="flex items-center gap-1">
        <!-- Edit Button -->
        <button
          type="button"
          class="p-1.5 rounded-lg text-neutral-500 hover:text-neutral-900 dark:hover:text-white hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors cursor-pointer"
          title="Edit konfigurasi ini"
          @click="$emit('edit', item)"
        >
          <UIcon name="i-lucide-pencil" class="w-3.5 h-3.5" />
        </button>

        <!-- Delete Button -->
        <button
          type="button"
          class="p-1.5 rounded-lg text-neutral-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition-colors cursor-pointer"
          title="Hapus konfigurasi ini"
          @click="$emit('delete', item)"
        >
          <UIcon name="i-lucide-trash-2" class="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  </div>
</template>
