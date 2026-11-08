<script setup lang="ts">
import { TODO_FOCUS_TAGS } from '~/utils/todoTags'

const model = defineModel<string>({ default: 'all' })
</script>

<template>
  <div class="flex flex-wrap items-center gap-1.5 text-xs pb-1">
    <span class="text-neutral-400 text-xs font-semibold uppercase tracking-wider mr-1">Fokus:</span>
    <button
      type="button"
      class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors cursor-pointer"
      :class="model === 'all' ? 'bg-neutral-800 text-white dark:bg-neutral-100 dark:text-neutral-900' : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:bg-neutral-200 dark:hover:bg-neutral-700'"
      @click="model = 'all'"
    >
      Semua Tag
    </button>
    <button
      v-for="tag in TODO_FOCUS_TAGS"
      :key="tag.key"
      type="button"
      class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer border"
      :class="model === tag.key ? 'ring-2 ring-offset-1 ring-neutral-400 dark:ring-offset-neutral-900 ' + tag.class : tag.class"
      @click="model = tag.key"
    >
      <UIcon :name="tag.icon" class="w-3 h-3" />
      <span>{{ tag.label }}</span>
    </button>
    <button
      type="button"
      class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors cursor-pointer border border-dashed border-neutral-300 dark:border-white/15 text-neutral-500"
      :class="model === 'untagged' ? 'bg-neutral-200 dark:bg-neutral-700' : ''"
      @click="model = 'untagged'"
    >
      Tanpa Tag
    </button>
  </div>
</template>
