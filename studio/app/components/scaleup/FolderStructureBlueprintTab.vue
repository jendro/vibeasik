<script setup lang="ts">
import type { ArchitecturePattern } from '~/types'

const props = defineProps<{
  patterns: ArchitecturePattern[]
  selectedPatternId: string
}>()

const emit = defineEmits<{
  (e: 'update:selectedPatternId', id: string): void
}>()

const copied = ref(false)

const currentPattern = computed(() => {
  return props.patterns.find(p => p.id === props.selectedPatternId) || props.patterns[0]
})

const handleCopyTree = () => {
  if (!currentPattern.value) return
  if (navigator?.clipboard?.writeText) {
    navigator.clipboard.writeText(currentPattern.value.folder_blueprint_tree)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  }
}
</script>

<template>
  <div v-if="currentPattern" class="space-y-6">
    <!-- Pattern Switcher Pills -->
    <div class="flex items-center gap-2 overflow-x-auto pb-1 no-scrollbar">
      <button
        v-for="p in patterns"
        :key="p.id"
        type="button"
        class="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs sm:text-sm font-semibold transition-all shrink-0 cursor-pointer"
        :class="selectedPatternId === p.id
          ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900 shadow-xs'
          : 'bg-white dark:bg-neutral-900 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white border border-neutral-200 dark:border-neutral-800'"
        @click="emit('update:selectedPatternId', p.id)"
      >
        <span>{{ p.name }}</span>
        <span
          class="text-[10px] px-1.5 py-0.5 rounded"
          :class="selectedPatternId === p.id ? 'bg-neutral-700 text-neutral-200 dark:bg-neutral-200 dark:text-neutral-800' : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-500'"
        >
          {{ p.badge }}
        </span>
      </button>
    </div>

    <!-- Main Blueprint Layout -->
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-6">
      <!-- Folder Tree (7 cols) -->
      <div class="lg:col-span-7 flex flex-col rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-900 text-neutral-100 overflow-hidden shadow-xs">
        <div class="flex items-center justify-between px-4 py-3 bg-neutral-950/80 border-b border-neutral-800">
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-folder-tree" class="w-4 h-4 text-sky-400" />
            <span class="text-xs font-mono font-semibold text-neutral-300">
              Folder Blueprint: {{ currentPattern.name }}
            </span>
          </div>
          <button
            type="button"
            class="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-md bg-neutral-800 hover:bg-neutral-700 text-neutral-200 transition-colors cursor-pointer"
            @click="handleCopyTree"
          >
            <UIcon :name="copied ? 'i-lucide-check' : 'i-lucide-copy'" class="w-3.5 h-3.5" :class="{ 'text-sky-400': copied }" />
            <span>{{ copied ? 'Tersalin!' : 'Salin Struktur' }}</span>
          </button>
        </div>

        <div class="p-4 overflow-x-auto font-mono text-xs text-sky-300/90 leading-relaxed bg-[#010810] border border-white/10">
          <pre>{{ currentPattern.folder_blueprint_tree }}</pre>
        </div>
      </div>

      <!-- Folder Directory Annotations (5 cols) -->
      <div class="lg:col-span-5 space-y-4">
        <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
          <h3 class="text-xs font-bold uppercase tracking-wider text-neutral-500 dark:text-neutral-400 mb-3 flex items-center gap-2">
            <UIcon name="i-lucide-info" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
            <span>Penjelasan Peran Direktori</span>
          </h3>

          <div class="space-y-3">
            <div
              v-for="anno in currentPattern.folder_explanation"
              :key="anno.path"
              class="p-3 rounded-lg border border-neutral-100 dark:border-neutral-800/80 bg-neutral-50/60 dark:bg-neutral-800/40"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="font-mono text-xs font-bold text-sky-600 dark:text-sky-400">
                  {{ anno.path }}
                </span>
                <span class="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-neutral-200/80 dark:bg-neutral-700 text-neutral-700 dark:text-neutral-300">
                  {{ anno.role }}
                </span>
              </div>
              <p class="text-xs text-neutral-600 dark:text-neutral-400 mt-1.5 leading-relaxed">
                {{ anno.description }}
              </p>
            </div>
          </div>
        </div>

        <!-- Scale-Up Insight Card -->
        <div class="p-4 rounded-xl border border-sky-500/20 bg-sky-50/50 dark:bg-sky-950/20">
          <h4 class="text-xs font-bold text-sky-800 dark:text-sky-300 flex items-center gap-1.5 mb-1.5">
            <UIcon name="i-lucide-compass" class="w-4 h-4" />
            <span>Kaidah Penamaan & Refactor</span>
          </h4>
          <p class="text-xs text-sky-700/90 dark:text-sky-300/80 leading-relaxed">
            Dalam arsitektur modern Go & TypeScript, jangan pernah membuat folder raksasa berisi ratusan file. Selalu kelompokkan file berdasarkan domain atau use-case agar AI Coding Agent tidak kehilangan konteks saat proses perbaikan.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
