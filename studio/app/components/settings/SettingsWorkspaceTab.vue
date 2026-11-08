<script setup lang="ts">
defineProps<{ saving: boolean }>()
const defaultProjectsRoot = defineModel<string>('root', { required: true })
const emit = defineEmits<{ save: [] }>()
const handleSaveWorkspaceSettings = () => emit('save')
</script>

<template>
  <div class="space-y-6 max-w-4xl">
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
            :disabled="saving"
            class="px-5 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-xs font-semibold inline-flex items-center justify-center gap-2 transition-colors cursor-pointer shadow-xs shrink-0"
            @click="handleSaveWorkspaceSettings"
          >
            <UIcon v-if="saving" name="i-lucide-loader-2" class="w-4 h-4 animate-spin" />
            <UIcon v-else name="i-lucide-save" class="w-4 h-4" />
            <span>{{ saving ? 'Menyimpan...' : 'Simpan Lokasi' }}</span>
          </button>
        </div>
        <p class="text-xs text-neutral-500 dark:text-neutral-400">
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
</template>
