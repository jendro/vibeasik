<script setup lang="ts">
import type { GuidelineCategory } from '~/types'

const props = defineProps<{
  open: boolean
  categories: GuidelineCategory[]
}>()

const emit = defineEmits<{
  (e: 'update:open', val: boolean): void
}>()

const activeCategoryIndex = ref(0)
const activeCategory = computed(() => props.categories[activeCategoryIndex.value])

const severityClass = (s: string) => {
  switch (s) {
    case 'critical':
      return 'text-red-700 bg-red-100 dark:bg-red-950/50 dark:text-red-300 border-red-200 dark:border-red-800'
    case 'high':
      return 'text-amber-700 bg-amber-100 dark:bg-amber-950/50 dark:text-amber-300 border-amber-200 dark:border-amber-800'
    case 'medium':
      return 'text-blue-700 bg-blue-100 dark:bg-blue-950/50 dark:text-blue-300 border-blue-200 dark:border-blue-800'
    default:
      return 'text-neutral-600 bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-400 border-neutral-200 dark:border-neutral-700'
  }
}
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-center justify-end bg-black/50 backdrop-blur-xs"
  >
    <div class="w-full max-w-2xl h-full bg-white dark:bg-slate-900/40 border-l border-neutral-200 dark:border-white/10 shadow-2xl flex flex-col duration-200">
      <!-- Header -->
      <div class="p-5 border-b border-neutral-200 dark:border-white/10 flex items-center justify-between">
        <div>
          <h2 class="text-base font-bold text-neutral-900 dark:text-white flex items-center gap-2">
            <UIcon name="i-lucide-book-open-check" class="w-5 h-5 text-violet-500" />
            Rubrik & Patokan Saran AI
          </h2>
          <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5">
            Folder: <code class="font-mono text-[11px] bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">vibeasik/guidelines/*.json</code>
          </p>
        </div>

        <button
          type="button"
          class="p-1 rounded-lg text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200"
          @click="emit('update:open', false)"
        >
          <UIcon name="i-lucide-x" class="w-5 h-5" />
        </button>
      </div>

      <!-- Categories Tabs -->
      <div class="px-5 py-3 border-b border-neutral-100 dark:border-white/10 bg-neutral-50/50 dark:bg-slate-900/50 flex gap-1.5 overflow-x-auto text-xs">
        <button
          v-for="(cat, idx) in categories"
          :key="cat.category"
          type="button"
          class="px-3 py-1.5 rounded-lg whitespace-nowrap transition-colors flex items-center gap-1.5 font-medium"
          :class="activeCategoryIndex === idx
            ? 'bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white shadow-xs border border-neutral-200 dark:border-neutral-700'
            : 'text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-300'"
          @click="activeCategoryIndex = idx"
        >
          <span>{{ cat.name }}</span>
          <span class="text-[10px] font-mono px-1 rounded bg-neutral-200/60 dark:bg-neutral-700">
            {{ cat.guidelines?.length || 0 }}
          </span>
        </button>
      </div>

      <!-- Body / Active Category Guidelines List -->
      <div class="flex-1 overflow-y-auto p-5 space-y-4 text-xs">
        <div v-if="activeCategory">
          <div class="mb-4 p-3.5 rounded-xl bg-violet-50/50 dark:bg-violet-950/20 border border-violet-200/60 dark:border-violet-900/40">
            <h3 class="font-bold text-violet-900 dark:text-violet-200 text-sm">
              {{ activeCategory.name }}
            </h3>
            <p class="text-neutral-600 dark:text-neutral-300 text-xs mt-1 leading-relaxed">
              {{ activeCategory.description }}
            </p>
            <div class="mt-2 text-[10px] font-mono text-violet-700 dark:text-violet-400">
              Berkas: {{ activeCategory.file_path }}
            </div>
          </div>

          <!-- Guidelines Cards in this category -->
          <div class="space-y-3">
            <div
              v-for="item in activeCategory.guidelines"
              :key="item.id"
              class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-2xs space-y-2.5"
            >
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-2">
                  <span class="px-2 py-0.5 rounded font-mono font-bold text-[11px] bg-neutral-100 dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 border border-neutral-200 dark:border-neutral-700">
                    {{ item.id }}
                  </span>
                  <span class="font-bold text-neutral-900 dark:text-white text-xs">
                    {{ item.title }}
                  </span>
                </div>

                <span
                  class="px-2 py-0.5 rounded text-[10px] font-medium border uppercase"
                  :class="severityClass(item.severity)"
                >
                  {{ item.severity }}
                </span>
              </div>

              <!-- Rule -->
              <div class="space-y-1">
                <div class="text-[11px] font-semibold text-neutral-500 uppercase tracking-wider">
                  Aturan (Rule):
                </div>
                <p class="text-neutral-700 dark:text-neutral-300 leading-relaxed font-sans">
                  {{ item.rule }}
                </p>
              </div>

              <!-- Rationale -->
              <div class="space-y-1">
                <div class="text-[11px] font-semibold text-neutral-500 uppercase tracking-wider">
                  Rasional / Alasan:
                </div>
                <p class="text-neutral-600 dark:text-neutral-400 leading-relaxed font-sans text-[11px]">
                  {{ item.rationale }}
                </p>
              </div>

              <!-- Inspection Check -->
              <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200/60 dark:border-neutral-700/60 space-y-1">
                <div class="flex items-center gap-1 font-semibold text-[11px] text-neutral-600 dark:text-neutral-300">
                  <UIcon name="i-lucide-search" class="w-3.5 h-3.5 text-neutral-500" />
                  <span>Metode Pemeriksaan Otomatis:</span>
                </div>
                <p class="text-[11px] font-mono text-neutral-600 dark:text-neutral-300">
                  {{ item.inspection_check }}
                </p>
              </div>

              <!-- Suggested Action -->
              <div class="p-2.5 rounded-lg bg-sky-50/50 dark:bg-sky-950/20 border border-sky-200/60 dark:border-sky-900/40 space-y-1">
                <div class="flex items-center gap-1 font-semibold text-[11px] text-sky-800 dark:text-sky-300">
                  <UIcon name="i-lucide-wrench" class="w-3.5 h-3.5 text-sky-500" />
                  <span>Saran Tindakan Solusi:</span>
                </div>
                <p class="text-[11px] text-sky-900 dark:text-sky-200 font-sans">
                  {{ item.suggested_action }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
