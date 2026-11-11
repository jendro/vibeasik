<script setup lang="ts">
defineProps<{
  stats: {
    total: number
    active?: number
    hold?: number
    done?: number
    deprecated?: number
    multiFe: number
    bundles: number
    monorepos: number
    avgHealth: number
  }
}>()
</script>

<template>
  <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f]">
      <div class="flex items-center justify-between text-neutral-500 dark:text-neutral-400 text-xs font-medium">
        <span>Total Proyek</span>
        <UIcon name="i-lucide-box" class="w-4 h-4 text-sky-500" />
      </div>
      <div class="mt-2 text-2xl font-bold font-mono">
        {{ stats.total }}
      </div>
      <div class="mt-0.5 text-xs text-neutral-500 truncate" :title="`${stats.active ?? stats.total} aktif, ${stats.hold || 0} hold, ${stats.done || 0} done`">
        {{ stats.active ?? stats.total }} aktif<span v-if="stats.hold || stats.done"> ({{ stats.hold || 0 }} hold, {{ stats.done || 0 }} done)</span>
      </div>
    </div>

    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f]">
      <div class="flex items-center justify-between text-neutral-500 dark:text-neutral-400 text-xs font-medium">
        <span>Multi-FE</span>
        <UIcon name="i-lucide-layers" class="w-4 h-4 text-purple-500" />
      </div>
      <div class="mt-2 text-2xl font-bold font-mono text-purple-600 dark:text-purple-400">
        {{ stats.multiFe }}
      </div>
      <div class="mt-0.5 text-xs text-neutral-500">
        Admin + Client
      </div>
    </div>

    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f]">
      <div class="flex items-center justify-between text-neutral-500 dark:text-neutral-400 text-xs font-medium">
        <span>Bundles</span>
        <UIcon name="i-lucide-cpu" class="w-4 h-4 text-blue-500" />
      </div>
      <div class="mt-2 text-2xl font-bold font-mono text-blue-600 dark:text-blue-400">
        {{ stats.bundles }}
      </div>
      <div class="mt-0.5 text-xs text-neutral-500">
        Pasangan BE + FE
      </div>
    </div>

    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f]">
      <div class="flex items-center justify-between text-neutral-500 dark:text-neutral-400 text-xs font-medium">
        <span>Monorepo</span>
        <UIcon name="i-lucide-git-fork" class="w-4 h-4 text-amber-500" />
      </div>
      <div class="mt-2 text-2xl font-bold font-mono text-amber-600 dark:text-amber-400">
        {{ stats.monorepos }}
      </div>
      <div class="mt-0.5 text-xs text-neutral-500">
        Polyglot repos
      </div>
    </div>

    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f]">
      <div class="flex items-center justify-between text-neutral-500 dark:text-neutral-400 text-xs font-medium">
        <span>Deprecated / Non-Aktif</span>
        <UIcon name="i-lucide-archive" class="w-4 h-4 text-amber-500" />
      </div>
      <div class="mt-2 text-2xl font-bold font-mono text-amber-600 dark:text-amber-400">
        {{ stats.deprecated || 0 }}
      </div>
      <div class="mt-0.5 text-xs text-neutral-500 truncate" :title="`${stats.deprecated || 0} deprecated, ${stats.hold || 0} hold, ${stats.done || 0} selesai`">
        {{ stats.deprecated || 0 }} deprecated<span v-if="stats.hold || stats.done"> • {{ (stats.hold || 0) + (stats.done || 0) }} hold/done</span>
      </div>
    </div>

    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f]">
      <div class="flex items-center justify-between text-neutral-500 dark:text-neutral-400 text-xs font-medium">
        <span>Skor Kesehatan</span>
        <UIcon name="i-lucide-activity" class="w-4 h-4 text-sky-400" />
      </div>
      <div class="mt-2 text-2xl font-bold font-mono text-sky-600 dark:text-sky-400">
        {{ stats.avgHealth }}/100
      </div>
      <div class="mt-0.5 text-xs text-neutral-500">
        Audit rata-rata
      </div>
    </div>
  </div>
</template>
