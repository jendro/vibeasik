<script setup lang="ts">
import type { Todo } from '~/types'
import { complexityMeta } from '~/utils/complexity'

/** Chip kerumitan todo + popover rincian 4 dimensi (logika, koding, cakupan, risiko). */
const props = defineProps<{ todo: Todo }>()

const meta = computed(() => complexityMeta(props.todo.complexity_level))
const dims = computed(() => [
  { label: 'Logika', value: props.todo.complexity_logic, hint: 'Algoritma, state, alur yang harus dipahami' },
  { label: 'Koding', value: props.todo.complexity_coding, hint: 'Volume & kesulitan kode yang ditulis' },
  { label: 'Cakupan', value: props.todo.complexity_scope, hint: 'Jumlah file, komponen, aplikasi tersentuh' },
  { label: 'Risiko', value: props.todo.complexity_risk, hint: 'Regresi, keamanan, kerusakan data' }
])
const sourceLabel = computed(() => ({ llm: 'Dinilai LLM', manual: 'Dinilai manual', heuristic: 'Taksiran otomatis' } as Record<string, string>)[props.todo.complexity_source || ''] || 'Taksiran otomatis')
</script>

<template>
  <UPopover v-if="todo.complexity_level" mode="hover" :content="{ side: 'top', align: 'start' }">
    <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-medium text-xs border cursor-default" :class="meta.class">
      <UIcon :name="meta.icon" class="w-3.5 h-3.5" />
      {{ meta.label }} · {{ (todo.complexity_score ?? 0).toFixed(1) }}
    </span>
    <template #content>
      <div class="p-3 w-64 space-y-2 text-xs">
        <div class="flex items-center justify-between font-semibold text-neutral-800 dark:text-neutral-100">
          <span>Kerumitan {{ meta.label }}</span>
          <span class="font-mono">{{ (todo.complexity_score ?? 0).toFixed(1) }} / 5</span>
        </div>
        <div v-for="d in dims" :key="d.label" :title="d.hint" class="space-y-0.5">
          <div class="flex justify-between text-xs text-neutral-600 dark:text-neutral-300">
            <span>{{ d.label }}</span>
            <span class="font-mono">{{ d.value }}</span>
          </div>
          <div class="h-1.5 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
            <div class="h-full rounded-full bg-linear-to-r from-emerald-400 via-amber-400 to-rose-500" :style="{ width: `${(d.value / 5) * 100}%` }" />
          </div>
        </div>
        <p class="pt-1.5 border-t border-neutral-100 dark:border-white/[0.08] text-xs text-neutral-500">
          {{ sourceLabel }}<template v-if="todo.complexity_note">
            — {{ todo.complexity_note }}
          </template>
        </p>
      </div>
    </template>
  </UPopover>
</template>
