<script setup lang="ts">
import type { RecapWorkStat } from '~/types'

// Satu kelompok pekerjaan pada rekap pemilik (bug ditutup, perbaikan algoritma): angka + daftar terbaru.
defineProps<{
  title: string
  icon: string
  iconClass?: string
  stat: RecapWorkStat
  empty: string
  hint?: string
  actionLabel?: string
}>()
const emit = defineEmits<{ action: [] }>()

const fmtDate = (iso?: string) => iso ? new Date(iso).toLocaleDateString('id-ID', { day: 'numeric', month: 'short' }) : ''
</script>

<template>
  <section class="min-w-0 rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] p-5 space-y-3">
    <div class="flex items-start justify-between gap-2">
      <h3 class="text-sm font-semibold text-neutral-900 dark:text-white flex items-center gap-2">
        <UIcon :name="icon" class="size-4" :class="iconClass" />{{ title }}
      </h3>
      <div class="text-right leading-none">
        <span class="text-2xl font-semibold tabular-nums text-neutral-900 dark:text-white">{{ stat.done }}</span>
        <div class="text-xs text-neutral-500 mt-1">
          {{ stat.done_in_period }} di periode ini · {{ stat.open }} terbuka
        </div>
      </div>
    </div>
    <p v-if="hint" class="text-xs text-neutral-500">
      {{ hint }}
    </p>
    <ul v-if="stat.latest.length" class="space-y-1.5">
      <li v-for="t in stat.latest" :key="t.id" class="flex gap-2 text-sm text-neutral-700 dark:text-neutral-200">
        <UIcon name="i-lucide-check" class="size-4 shrink-0 mt-0.5 text-sky-500" />
        <span class="min-w-0 flex-1 truncate" :title="t.title">{{ t.title }}</span>
        <span class="shrink-0 text-xs text-neutral-500">{{ fmtDate(t.completed_at) }}</span>
      </li>
    </ul>
    <p v-else class="text-sm text-neutral-500">
      {{ empty }}
    </p>
    <UButton v-if="actionLabel" size="xs" color="neutral" variant="outline" trailing-icon="i-lucide-arrow-right" @click="emit('action')">
      {{ actionLabel }}
    </UButton>
  </section>
</template>
