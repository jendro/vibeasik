<script setup lang="ts">
import type { Todo } from '~/types'

// Sub todo sebuah todo gabungan (Aturan 43): dikerjakan sekaligus lewat induknya, statusnya mengikuti induk.
const props = defineProps<{ subTodos: Todo[] }>()

const open = ref(false)
const doneCount = computed(() => props.subTodos.filter(s => s.status === 'completed').length)

const statusIcon = (s: Todo) => {
  switch (s.status) {
    case 'completed':
      return { icon: 'i-lucide-check-circle-2', class: 'text-emerald-500', title: 'Selesai bersama todo gabungan' }
    case 'rejected':
      return { icon: 'i-lucide-x-circle', class: 'text-neutral-400', title: 'Ditolak bersama todo gabungan' }
    default:
      return { icon: 'i-lucide-circle-dashed', class: 'text-violet-500', title: 'Dikerjakan sekaligus lewat todo gabungan' }
  }
}

const fileOf = (s: Todo) => {
  const key = s.audit_key || ''
  const i = key.indexOf('|')
  if (i > 0) return key.slice(i + 1).split('/').pop()
  return /\(([^():]+):L\d+\)/.exec(s.target_apps || '')?.[1]
}
</script>

<template>
  <div class="mt-2 rounded-lg border border-violet-200/70 dark:border-violet-900/50 bg-violet-50/50 dark:bg-violet-950/20">
    <button
      type="button"
      class="w-full flex items-center justify-between gap-2 px-2.5 py-1.5 text-xs font-semibold text-violet-800 dark:text-violet-300 cursor-pointer"
      @click="open = !open"
    >
      <span class="flex items-center gap-1.5">
        <UIcon name="i-lucide-layers" class="w-3.5 h-3.5" />
        {{ subTodos.length }} sub todo dikerjakan sekaligus
        <span v-if="doneCount" class="font-normal text-emerald-600 dark:text-emerald-400">· {{ doneCount }} selesai</span>
      </span>
      <UIcon :name="open ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" class="w-3.5 h-3.5" />
    </button>
    <ul v-if="open" class="px-2.5 pb-2 space-y-1">
      <li v-for="s in subTodos" :key="s.id" class="flex items-start gap-1.5 text-xs text-neutral-700 dark:text-neutral-300">
        <UIcon :name="statusIcon(s).icon" :class="statusIcon(s).class" class="w-3.5 h-3.5 mt-px shrink-0" :title="statusIcon(s).title" />
        <span class="min-w-0">
          <span class="font-mono text-neutral-400">#{{ s.id }}</span>
          {{ s.title }}
          <code v-if="fileOf(s)" class="ml-1 text-xs text-sky-700 dark:text-sky-400">{{ fileOf(s) }}</code>
        </span>
      </li>
    </ul>
  </div>
</template>
