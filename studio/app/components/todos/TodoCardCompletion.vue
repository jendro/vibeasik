<script setup lang="ts">
import type { Todo } from '~/types'

defineProps<{ todo: Todo }>()
const { formatDateTime } = useFormatters()
</script>

<template>
  <div
    class="space-y-2 p-2.5 rounded-lg bg-emerald-50/70 dark:bg-emerald-950/20 border border-emerald-200/60 dark:border-emerald-800/40 text-xs"
  >
    <div class="flex flex-wrap items-center gap-2">
      <div class="flex items-center gap-1.5 font-medium text-emerald-800 dark:text-emerald-300">
        <UIcon name="i-lucide-check-circle-2" class="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
        <span>Selesai:</span>
        <span class="font-normal text-neutral-700 dark:text-neutral-300">
          {{ todo.completed_at ? formatDateTime(todo.completed_at) : 'Selesai' }}
        </span>
      </div>

      <div
        v-if="todo.completed_by_agent"
        class="flex items-center gap-1 text-xs px-2 py-0.5 rounded-md bg-white dark:bg-slate-900/40 border border-emerald-200 dark:border-emerald-800 text-emerald-800 dark:text-emerald-300"
      >
        <UIcon name="i-lucide-bot" class="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
        <span>Agent:</span>
        <span class="font-semibold">{{ todo.completed_by_agent }}</span>
      </div>

      <div
        v-if="todo.completed_by_model"
        class="flex items-center gap-1 font-mono text-xs px-2 py-0.5 rounded-md bg-white dark:bg-slate-900/40 border border-emerald-200 dark:border-emerald-800 text-emerald-700 dark:text-emerald-400"
      >
        <UIcon name="i-lucide-cpu" class="w-3 h-3 text-sky-500" />
        <span>Model:</span>
        <span class="font-semibold">{{ todo.completed_by_model }}</span>
      </div>

      <!-- QA Score Badge -->
      <span
        v-if="todo.score !== undefined && todo.score !== null"
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-semibold text-xs border ml-auto"
        :class="todo.score >= 80
          ? 'bg-amber-100 text-amber-900 border-amber-300 dark:bg-amber-950/60 dark:text-amber-200 dark:border-amber-800'
          : todo.score >= 60
            ? 'bg-blue-100 text-blue-900 border-blue-300 dark:bg-blue-950/60 dark:text-blue-200 dark:border-blue-800'
            : 'bg-red-100 text-red-900 border-red-300 dark:bg-red-950/60 dark:text-red-200 dark:border-red-800'"
      >
        <UIcon :name="todo.score >= 80 ? 'i-lucide-star' : 'i-lucide-award'" class="w-3 h-3 text-amber-500" />
        QA Score: {{ todo.score }}
      </span>

      <span
        v-if="todo.inherited_from"
        class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs bg-purple-100 text-purple-800 dark:bg-purple-950/50 dark:text-purple-300 border border-purple-200 dark:border-purple-800"
      >
        <UIcon name="i-lucide-scroll" class="w-3 h-3" />
        Warisan: {{ todo.inherited_from }}
      </span>
    </div>

    <!-- High Score Skills & Success Factors (Penyebab Nilai Tinggi) -->
    <div
      v-if="(todo.high_score_skills && todo.high_score_skills.length) || todo.high_score_notes"
      class="mt-1.5 p-2 rounded-md bg-white/80 dark:bg-neutral-900/60 border border-emerald-200 dark:border-emerald-800/60 space-y-1.5"
    >
      <div class="flex items-center justify-between text-xs font-medium text-emerald-900 dark:text-emerald-200">
        <span class="flex items-center gap-1">
          <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5 text-amber-500" />
          Penyebab Nilai Tinggi / Warisan Pengetahuan:
        </span>
        <span v-if="todo.high_score_skills && todo.high_score_skills.length" class="text-xs text-neutral-400">
          {{ todo.high_score_skills.length }} skill kunci
        </span>
      </div>

      <div v-if="todo.high_score_skills && todo.high_score_skills.length" class="flex flex-wrap gap-1">
        <span
          v-for="sk in todo.high_score_skills"
          :key="sk"
          class="px-1.5 py-0.5 rounded bg-amber-50 dark:bg-amber-950/40 text-amber-800 dark:text-amber-300 border border-amber-200 dark:border-amber-900 text-xs font-mono flex items-center gap-1"
        >
          <UIcon name="i-lucide-zap" class="w-2.5 h-2.5 text-amber-500" />
          {{ sk }}
        </span>
      </div>

      <p v-if="todo.high_score_notes" class="text-xs text-neutral-700 dark:text-neutral-300 leading-relaxed">
        {{ todo.high_score_notes }}
      </p>

      <ul v-if="todo.success_factors && todo.success_factors.length" class="list-disc list-inside text-xs text-neutral-600 dark:text-neutral-400 space-y-0.5">
        <li v-for="(factor, idx) in todo.success_factors" :key="idx">
          {{ factor }}
        </li>
      </ul>
    </div>
  </div>
</template>
