<script setup lang="ts">
import type { ArchitecturePattern } from '~/types'

const props = defineProps<{
  patterns: ArchitecturePattern[]
  selectedPatternId: string
}>()

const emit = defineEmits<{
  (e: 'selectPattern', id: string): void
  (e: 'viewBlueprint', id: string): void
}>()

const filterType = ref<'all' | 'ai' | 'clean' | 'simple'>('all')

const filteredPatterns = computed(() => {
  if (filterType.value === 'ai') {
    return props.patterns.filter(p => p.ai_readiness_score >= 4)
  }
  if (filterType.value === 'clean') {
    return props.patterns.filter(p => p.testability_score >= 4)
  }
  if (filterType.value === 'simple') {
    return props.patterns.filter(p => p.complexity_score <= 3)
  }
  return props.patterns
})
</script>

<template>
  <div class="space-y-6">
    <!-- Filter bar & concept note -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-900/40">
      <div class="flex items-center gap-2">
        <UIcon name="i-lucide-sparkles" class="w-4 h-4 text-sky-600 dark:text-sky-400 shrink-0" />
        <span class="text-xs sm:text-sm font-medium text-neutral-700 dark:text-neutral-300">
          Filter Pola Arsitektur:
        </span>
      </div>
      <div class="flex flex-wrap items-center gap-1.5">
        <button
          type="button"
          class="px-2.5 py-1 text-xs rounded-md font-medium transition-colors cursor-pointer"
          :class="filterType === 'all'
            ? 'bg-sky-600 text-white font-semibold'
            : 'bg-white dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white border border-neutral-200 dark:border-neutral-700'"
          @click="filterType = 'all'"
        >
          Semua ({{ patterns.length }})
        </button>
        <button
          type="button"
          class="px-2.5 py-1 text-xs rounded-md font-medium transition-colors cursor-pointer"
          :class="filterType === 'ai'
            ? 'bg-sky-600 text-white font-semibold'
            : 'bg-white dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white border border-neutral-200 dark:border-neutral-700'"
          @click="filterType = 'ai'"
        >
          AI-Native & MCP (High)
        </button>
        <button
          type="button"
          class="px-2.5 py-1 text-xs rounded-md font-medium transition-colors cursor-pointer"
          :class="filterType === 'clean'
            ? 'bg-sky-600 text-white font-semibold'
            : 'bg-white dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white border border-neutral-200 dark:border-neutral-700'"
          @click="filterType = 'clean'"
        >
          High Testability
        </button>
        <button
          type="button"
          class="px-2.5 py-1 text-xs rounded-md font-medium transition-colors cursor-pointer"
          :class="filterType === 'simple'
            ? 'bg-sky-600 text-white font-semibold'
            : 'bg-white dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white border border-neutral-200 dark:border-neutral-700'"
          @click="filterType = 'simple'"
        >
          Ringan & Cepat
        </button>
      </div>
    </div>

    <!-- Patterns Grid -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
      <div
        v-for="p in filteredPatterns"
        :key="p.id"
        class="flex flex-col justify-between p-5 rounded-xl border transition-all duration-200 bg-white dark:bg-neutral-900 shadow-2xs"
        :class="selectedPatternId === p.id
          ? 'border-sky-500 ring-2 ring-sky-500/20 shadow-md'
          : 'border-neutral-200 dark:border-neutral-800 hover:border-neutral-300 dark:hover:border-neutral-700'"
      >
        <div class="space-y-4">
          <!-- Card Header -->
          <div class="flex items-start justify-between gap-3">
            <div>
              <div class="flex items-center gap-2">
                <span class="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold bg-sky-50 text-sky-700 dark:bg-sky-950/60 dark:text-sky-300 border border-sky-500/20">
                  {{ p.badge }}
                </span>
                <span v-if="p.ai_readiness_score >= 5" class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-purple-50 text-purple-700 dark:bg-purple-950/60 dark:text-purple-300 border border-purple-500/20">
                  <UIcon name="i-lucide-sparkles" class="w-3 h-3" />
                  AI Era
                </span>
              </div>
              <h2 class="text-base sm:text-lg font-bold text-neutral-900 dark:text-white mt-1.5">
                {{ p.name }}
              </h2>
            </div>
            <span class="text-xs font-mono text-neutral-400 dark:text-neutral-500 uppercase px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800">
              #{{ p.id }}
            </span>
          </div>

          <p class="text-xs sm:text-sm text-neutral-600 dark:text-neutral-300 leading-relaxed">
            {{ p.short_desc }}
          </p>

          <!-- Metrics Matrix -->
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-1 pb-1">
            <div class="p-2 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-100 dark:border-neutral-800 text-center">
              <span class="text-[10px] text-neutral-500 dark:text-neutral-400 uppercase tracking-wider block">Kompleksitas</span>
              <span class="text-xs font-bold text-neutral-800 dark:text-neutral-200">{{ p.complexity_score }} / 5</span>
            </div>
            <div class="p-2 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-100 dark:border-neutral-800 text-center">
              <span class="text-[10px] text-neutral-500 dark:text-neutral-400 uppercase tracking-wider block">Testability</span>
              <span class="text-xs font-bold text-sky-600 dark:text-sky-400">{{ p.testability_score }} / 5</span>
            </div>
            <div class="p-2 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-100 dark:border-neutral-800 text-center">
              <span class="text-[10px] text-neutral-500 dark:text-neutral-400 uppercase tracking-wider block">AI-Ready</span>
              <span class="text-xs font-bold text-purple-600 dark:text-purple-400">{{ p.ai_readiness_score }} / 5</span>
            </div>
            <div class="p-2 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-100 dark:border-neutral-800 text-center">
              <span class="text-[10px] text-neutral-500 dark:text-neutral-400 uppercase tracking-wider block">Skalabilitas</span>
              <span class="text-xs font-bold text-blue-600 dark:text-blue-400">{{ p.scalability_score }} / 5</span>
            </div>
          </div>

          <!-- Pros & Cons Snippets -->
          <div class="space-y-2 pt-2 border-t border-neutral-100 dark:border-neutral-800/80">
            <div class="flex items-start gap-2 text-xs text-neutral-700 dark:text-neutral-300">
              <UIcon name="i-lucide-check-circle-2" class="w-4 h-4 text-sky-400 shrink-0 mt-0.5" />
              <span><strong>Kelebihan:</strong> {{ p.pros[0] }}</span>
            </div>
            <div class="flex items-start gap-2 text-xs text-neutral-700 dark:text-neutral-300">
              <UIcon name="i-lucide-alert-circle" class="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
              <span><strong>Trade-off:</strong> {{ p.cons[0] }}</span>
            </div>
          </div>
        </div>

        <!-- Card Footer Actions -->
        <div class="flex items-center justify-between gap-3 pt-4 mt-4 border-t border-neutral-100 dark:border-neutral-800">
          <button
            type="button"
            class="inline-flex items-center gap-1.5 text-xs font-semibold text-sky-600 dark:text-sky-400 hover:underline cursor-pointer"
            @click="emit('viewBlueprint', p.id)"
          >
            <UIcon name="i-lucide-folder-tree" class="w-3.5 h-3.5" />
            <span>Lihat Desain Folder</span>
          </button>

          <button
            type="button"
            class="px-3 py-1.5 text-xs font-semibold rounded-lg transition-colors cursor-pointer"
            :class="selectedPatternId === p.id
              ? 'bg-sky-600 text-white'
              : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-200 dark:hover:bg-neutral-700'"
            @click="emit('selectPattern', p.id)"
          >
            {{ selectedPatternId === p.id ? 'Sedang Dipilih' : 'Pilih Pola' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
