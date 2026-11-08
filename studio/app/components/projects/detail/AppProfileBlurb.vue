<script setup lang="ts">
import type { ProfileApp } from '~/types'

defineProps<{
  app: ProfileApp
}>()
</script>

<template>
  <div class="space-y-2 text-xs">
    <p v-if="app.summary" class="text-neutral-600 dark:text-neutral-300 leading-relaxed line-clamp-3">
      {{ app.summary }}
    </p>
    <ul v-if="app.responsibilities.length" class="space-y-0.5">
      <li
        v-for="r in app.responsibilities.slice(0, 4)"
        :key="r"
        class="flex items-start gap-1.5 text-xs text-neutral-600 dark:text-neutral-400"
      >
        <UIcon name="i-lucide-check" class="w-3 h-3 mt-0.5 text-sky-500 shrink-0" />
        <span class="line-clamp-1">{{ r }}</span>
      </li>
    </ul>
    <div v-if="app.talks_to.length" class="flex flex-wrap items-center gap-1 text-xs">
      <span class="text-neutral-400">Terhubung ke</span>
      <span
        v-for="t in app.talks_to"
        :key="t"
        class="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-300"
      >
        <UIcon name="i-lucide-arrow-right" class="w-2.5 h-2.5" />{{ t }}
      </span>
    </div>
  </div>
</template>
