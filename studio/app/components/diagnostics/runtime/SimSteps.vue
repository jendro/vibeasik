<script setup lang="ts">
import type { RuntimeSimStep } from '~/types'

defineProps<{
  steps: RuntimeSimStep[]
}>()

const stepVisual = (status: string) => {
  switch (status) {
    case 'running':
      return { icon: 'i-lucide-loader-2', spin: true, dot: 'bg-primary-500 text-white ring-4 ring-primary-100 dark:ring-primary-950', text: 'text-primary-700 dark:text-primary-300' }
    case 'done':
      return { icon: 'i-lucide-check', spin: false, dot: 'bg-emerald-500 text-white', text: 'text-neutral-800 dark:text-neutral-200' }
    case 'failed':
      return { icon: 'i-lucide-x', spin: false, dot: 'bg-rose-500 text-white', text: 'text-rose-700 dark:text-rose-300' }
    case 'skipped':
      return { icon: 'i-lucide-minus', spin: false, dot: 'bg-neutral-200 dark:bg-neutral-700 text-neutral-500', text: 'text-neutral-400' }
    default:
      return { icon: '', spin: false, dot: 'bg-neutral-100 dark:bg-neutral-800 border border-neutral-300 dark:border-neutral-600', text: 'text-neutral-400' }
  }
}

const formatDuration = (ms?: number) => {
  if (!ms) return ''
  return ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${ms}ms`
}
</script>

<template>
  <ol class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 2xl:grid-cols-9 gap-2">
    <li
      v-for="(step, idx) in steps"
      :key="step.key"
      class="p-2.5 rounded-xl border transition-colors"
      :class="step.status === 'running'
        ? 'border-primary-300 dark:border-primary-800 bg-primary-50/60 dark:bg-primary-950/30'
        : 'border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/60'"
      :title="step.detail || step.title"
    >
      <div class="flex items-center gap-2">
        <span class="w-5 h-5 rounded-full flex items-center justify-center shrink-0" :class="stepVisual(step.status).dot">
          <UIcon
            v-if="stepVisual(step.status).icon"
            :name="stepVisual(step.status).icon"
            class="w-3 h-3"
            :class="{ 'animate-spin': stepVisual(step.status).spin }"
          />
          <span v-else class="text-[9px] font-bold font-mono">{{ idx + 1 }}</span>
        </span>
        <span class="text-[11px] font-semibold leading-tight" :class="stepVisual(step.status).text">
          {{ step.title }}
        </span>
      </div>
      <p v-if="step.detail || step.duration_ms" class="mt-1.5 text-[10px] leading-snug text-neutral-500 dark:text-neutral-400 line-clamp-2">
        <span v-if="step.duration_ms" class="font-mono text-neutral-400">{{ formatDuration(step.duration_ms) }} · </span>{{ step.detail }}
      </p>
    </li>
  </ol>
</template>
