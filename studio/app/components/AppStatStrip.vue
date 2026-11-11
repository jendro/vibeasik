<script setup lang="ts">
// Deret angka ringkas (KPI) bersambung dalam satu bingkai, seperti di Beranda.
// Item dengan `to` menjadi tautan; dengan `action` menjadi tombol (mis. memfilter daftar).
export interface StatItem {
  label: string
  value: string | number
  hint?: string
  // Warna angka (kelas teks) — dipakai hemat: hanya untuk status (kritis/peringatan/sehat).
  tone?: string
  // Titik warna di depan label (kelas bg-*).
  dot?: string
  to?: string
  action?: () => void
  active?: boolean
  title?: string
}

const props = withDefaults(defineProps<{ items: StatItem[], cols?: string }>(), { cols: '' })

const gridCols = computed(() => props.cols || ({
  1: 'grid-cols-1',
  2: 'grid-cols-2',
  3: 'grid-cols-2 sm:grid-cols-3',
  4: 'grid-cols-2 lg:grid-cols-4',
  5: 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-5',
  6: 'grid-cols-2 sm:grid-cols-3 xl:grid-cols-6'
} as Record<number, string>)[props.items.length] || 'grid-cols-2 sm:grid-cols-3 xl:grid-cols-6')
</script>

<template>
  <section class="grid gap-px overflow-hidden rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-neutral-200 dark:bg-white/[0.08]" :class="gridCols">
    <component
      :is="s.to ? 'NuxtLink' : s.action ? 'button' : 'div'"
      v-for="s in items"
      :key="s.label"
      :to="s.to"
      :type="s.action ? 'button' : undefined"
      :title="s.title || s.hint"
      class="text-left bg-white dark:bg-[#04111f] px-4 py-3.5 sm:px-5 sm:py-4 space-y-0.5 transition-colors min-w-0"
      :class="[
        s.to || s.action ? 'hover:bg-neutral-50 dark:hover:bg-white/[0.03] focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-sky-500' : '',
        s.active ? 'bg-sky-50/60 dark:bg-sky-500/[0.07]' : ''
      ]"
      @click="s.action?.()"
    >
      <div class="text-xs font-medium text-neutral-500 dark:text-slate-400 flex items-center gap-1.5 truncate">
        <span v-if="s.dot" class="size-1.5 rounded-full shrink-0" :class="s.dot" />{{ s.label }}
      </div>
      <div class="text-2xl font-semibold tracking-tight tabular-nums truncate" :class="s.tone || 'text-neutral-950 dark:text-white'">
        {{ s.value }}
      </div>
      <div v-if="s.hint" class="text-xs text-neutral-500 dark:text-slate-500 truncate">
        {{ s.hint }}
      </div>
    </component>
  </section>
</template>
