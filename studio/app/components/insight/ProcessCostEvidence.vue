<script setup lang="ts">
import type { CostEvidence } from '~/types'

// Bukti rumus biaya: baris kode (loop atas data, query, pemanggilan berbiaya) yang membentuk T(n) dan Q(n),
// agar angka Big-O bisa diperiksa langsung pada kodenya.
const props = defineProps<{ title: string, file?: string, evidence?: CostEvidence[], emptyText?: string }>()

const KIND: Record<CostEvidence['kind'], { label: string, icon: string }> = {
  loop: { label: 'Loop data', icon: 'i-lucide-repeat' },
  query: { label: 'Query', icon: 'i-lucide-database' },
  call: { label: 'Pemanggilan', icon: 'i-lucide-corner-down-right' }
}
const tone = (e: CostEvidence) => {
  if (e.kind === 'query') {
    return e.depth > 0
      ? 'bg-rose-50 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300'
      : 'bg-neutral-100 text-neutral-700 dark:bg-white/[0.06] dark:text-neutral-200'
  }
  if (e.kind === 'loop') {
    return e.depth > 1
      ? 'bg-amber-100 text-amber-800 dark:bg-amber-950/50 dark:text-amber-300'
      : 'bg-amber-50 text-amber-700 dark:bg-amber-950/30 dark:text-amber-300'
  }
  return 'bg-sky-50 text-sky-700 dark:bg-sky-950/40 dark:text-sky-300'
}
const rows = computed(() => props.evidence || [])
</script>

<template>
  <div class="rounded-lg border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] overflow-hidden">
    <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 px-3 py-2 border-b border-neutral-200 dark:border-white/[0.08] text-xs">
      <span class="font-semibold text-neutral-900 dark:text-white">{{ title }} ({{ rows.length }})</span>
      <span v-if="file" class="font-mono text-neutral-500 truncate max-w-full" :title="file">{{ file }}</span>
    </div>
    <p v-if="!rows.length" class="px-3 py-2.5 text-xs text-neutral-500">
      {{ emptyText || 'Tidak ada baris yang menaikkan biaya.' }}
    </p>
    <ol v-else>
      <li
        v-for="e in rows"
        :key="`${e.kind}-${e.line}`"
        class="grid grid-cols-[3.25rem_minmax(0,1fr)] gap-x-3 px-3 py-2 border-b border-neutral-100 dark:border-white/[0.05] last:border-b-0"
      >
        <span class="font-mono text-xs tabular-nums text-neutral-400 text-right pt-px" :title="`baris ${e.line}`">L{{ e.line }}</span>
        <div class="min-w-0 space-y-1">
          <code class="block font-mono text-xs text-neutral-900 dark:text-white whitespace-pre overflow-x-auto">{{ e.code }}</code>
          <div class="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-neutral-600 dark:text-neutral-300">
            <span class="inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 font-medium" :class="tone(e)">
              <UIcon :name="KIND[e.kind].icon" class="size-3" />{{ KIND[e.kind].label }}
            </span>
            <span>{{ e.note }}</span>
          </div>
        </div>
      </li>
    </ol>
  </div>
</template>
