<script setup lang="ts">
import type { Project } from '~/types'

defineProps<{
  activeProject: Project | null
  activeTab: string
  selectedScope: 'backend' | 'mobile' | 'frontend'
  evaluating: boolean
  backendCount: number
}>()

const emit = defineEmits<{
  (e: 'update:activeTab', val: string): void
  (e: 'update:selectedScope', val: 'backend' | 'mobile' | 'frontend'): void
  (e: 'reEvaluate'): void
}>()

const scopes = [
  { id: 'backend' as const, label: 'Backend (BE)', icon: 'i-lucide-server', badge: 'Aktif', active: true },
  { id: 'mobile' as const, label: 'Mobile Apps', icon: 'i-lucide-smartphone', badge: 'Segera', active: false },
  { id: 'frontend' as const, label: 'Frontend Web', icon: 'i-lucide-layout', badge: 'Segera', active: false }
]

const tabs = [
  { id: 'patterns', label: 'Katalog Design Pattern', icon: 'i-lucide-layers', desc: '5 pola arsitektur utama' },
  { id: 'blueprints', label: 'Desain Struktur Folder', icon: 'i-lucide-folder-tree', desc: 'Blueprint tree per pattern' },
  { id: 'dissection', label: 'Bedah Router, Controller & Fungsi', icon: 'i-lucide-code-xml', desc: 'Analisis anatomis kode' },
  { id: 'mvpp-mcp', label: 'Konsep MVPP & MCP', icon: 'i-lucide-bot', desc: 'Paradigma AI-native' },
  { id: 'audit', label: 'Evaluasi Proyek Aktif', icon: 'i-lucide-gauge', desc: 'Maturity audit BE lokal' }
]
</script>

<template>
  <div class="space-y-4">
    <!-- Top Action Bar -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white/70 dark:bg-slate-900/60 backdrop-blur-md">
      <div class="flex items-start gap-3.5">
        <div class="w-11 h-11 rounded-xl bg-sky-600/10 text-sky-600 dark:text-sky-400 flex items-center justify-center shrink-0 border border-sky-500/20">
          <UIcon name="i-lucide-trending-up" class="w-6 h-6" />
        </div>
        <div>
          <div class="flex items-center gap-2.5">
            <h1 class="text-xl font-bold tracking-tight text-neutral-900 dark:text-white">
              Scale Up & Best Practice Aplikasi
            </h1>
            <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-sky-100 text-sky-800 dark:bg-sky-950/60 dark:text-sky-300">
              Enterprise & Real-World Hub
            </span>
          </div>
          <p class="text-xs sm:text-sm text-neutral-500 dark:text-neutral-400 mt-1 max-w-2xl">
            Mematangkan arsitektur sistem (Backend, Mobile, dan Frontend) sesuai standar best practice dunia nyata.
          </p>
        </div>
      </div>

      <!-- Right Active Status & Action -->
      <div class="flex items-center gap-3 self-end sm:self-center">
        <div v-if="activeProject" class="hidden md:flex flex-col items-end text-xs">
          <span class="text-neutral-400">Proyek Aktif:</span>
          <span class="font-semibold text-neutral-800 dark:text-neutral-200 max-w-[160px] truncate">
            {{ activeProject.name }}
          </span>
        </div>

        <button
          type="button"
          :disabled="evaluating"
          class="inline-flex items-center gap-2 px-3.5 py-2 text-xs sm:text-sm font-semibold rounded-lg text-white bg-sky-600 hover:bg-sky-500 active:scale-95 disabled:opacity-50 transition-all shadow-xs cursor-pointer"
          @click="emit('reEvaluate')"
        >
          <UIcon
            :name="evaluating ? 'i-lucide-loader-2' : 'i-lucide-refresh-cw'"
            class="w-4 h-4"
            :class="{ 'animate-spin': evaluating }"
          />
          <span>{{ evaluating ? 'Menganalisis...' : 'Audit Arsitektur' }}</span>
        </button>
      </div>
    </div>

    <!-- Tier Domain Scope Selector (BE / Mobile / FE) -->
    <div class="flex flex-wrap items-center gap-2 p-1.5 rounded-xl border border-neutral-200 dark:border-white/10 bg-neutral-100/70 dark:bg-slate-900/60">
      <span class="text-xs font-bold text-neutral-500 dark:text-neutral-400 px-3 uppercase tracking-wider">
        Target Domain:
      </span>
      <button
        v-for="sc in scopes"
        :key="sc.id"
        type="button"
        class="inline-flex items-center gap-2 px-3.5 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer"
        :class="selectedScope === sc.id
          ? 'bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white shadow-xs border border-neutral-200/80 dark:border-neutral-700'
          : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
        @click="emit('update:selectedScope', sc.id)"
      >
        <UIcon :name="sc.icon" class="w-4 h-4" />
        <span>{{ sc.label }}</span>
        <span
          class="text-[10px] px-1.5 py-0.5 rounded"
          :class="sc.active ? 'bg-sky-100 text-sky-800 dark:bg-sky-950 dark:text-sky-300 font-bold' : 'bg-neutral-200/80 dark:bg-neutral-800 text-neutral-500'"
        >
          {{ sc.badge }}
        </span>
      </button>
    </div>

    <!-- Navigation Tabs (Active in Backend) -->
    <div v-if="selectedScope === 'backend'" class="flex items-center gap-1.5 overflow-x-auto pb-1 border-b border-neutral-200 dark:border-white/10 no-scrollbar">
      <button
        v-for="t in tabs"
        :key="t.id"
        type="button"
        class="inline-flex items-center gap-2 px-3.5 py-2 text-xs sm:text-sm font-medium rounded-lg shrink-0 transition-all cursor-pointer"
        :class="activeTab === t.id
          ? 'bg-sky-50 text-sky-700 dark:bg-sky-950/60 dark:text-sky-300 border border-sky-500/20 font-semibold shadow-2xs'
          : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white hover:bg-neutral-100 dark:hover:bg-neutral-800/60'"
        @click="emit('update:activeTab', t.id)"
      >
        <UIcon :name="t.icon" class="w-4 h-4" />
        <span>{{ t.label }}</span>
      </button>
    </div>
  </div>
</template>
