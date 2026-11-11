<script setup lang="ts">
defineProps<{
  categoryCounts: Record<string, number>
  severityCounts: Record<string, number>
  totalFindings: number
  activeCategory: string
}>()

const emit = defineEmits<{
  (e: 'selectCategory', cat: string): void
}>()
</script>

<template>
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-3.5">
    <!-- Card 1: Deep Check -->
    <div
      class="p-4 rounded-xl border transition-all cursor-pointer"
      :class="activeCategory === 'deep'
        ? 'border-sky-500 bg-sky-50/50 dark:bg-sky-950/30 ring-1 ring-sky-500'
        : 'border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] hover:border-neutral-300 dark:hover:border-neutral-700'"
      @click="emit('selectCategory', activeCategory === 'deep' ? 'all' : 'deep')"
    >
      <div class="flex items-center justify-between pb-2">
        <span class="text-xs font-semibold uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
          Deep Check & Simulasi
        </span>
        <UIcon name="i-lucide-scan-search" class="w-4 h-4 text-sky-500" />
      </div>
      <div class="flex items-baseline gap-2">
        <span class="text-2xl font-bold font-mono text-neutral-900 dark:text-white">
          {{ categoryCounts['deep'] || 0 }}
        </span>
        <span class="text-xs text-neutral-500">Isu Simulasi</span>
      </div>
      <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1 line-clamp-1">
        Latensi Rute API, Validasi Form, Query Risk
      </p>
    </div>

    <!-- Card 2: Database Query -->
    <div
      class="p-4 rounded-xl border transition-all cursor-pointer"
      :class="activeCategory === 'query'
        ? 'border-violet-500 bg-violet-50/50 dark:bg-violet-950/30 ring-1 ring-violet-500'
        : 'border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] hover:border-neutral-300 dark:hover:border-neutral-700'"
      @click="emit('selectCategory', activeCategory === 'query' ? 'all' : 'query')"
    >
      <div class="flex items-center justify-between pb-2">
        <span class="text-xs font-semibold uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
          Database & Query
        </span>
        <UIcon name="i-lucide-database" class="w-4 h-4 text-violet-500" />
      </div>
      <div class="flex items-baseline gap-2">
        <span class="text-2xl font-bold font-mono text-neutral-900 dark:text-white">
          {{ categoryCounts['query'] || 0 }}
        </span>
        <span class="text-xs text-neutral-500">Isu Query</span>
      </div>
      <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1 line-clamp-1">
        N+1 Loop, SQL Injection, SELECT *, Cursor Leaks
      </p>
    </div>

    <!-- Card 2: Router & API -->
    <div
      class="p-4 rounded-xl border transition-all cursor-pointer"
      :class="activeCategory === 'router'
        ? 'border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500'
        : 'border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] hover:border-neutral-300 dark:hover:border-neutral-700'"
      @click="emit('selectCategory', activeCategory === 'router' ? 'all' : 'router')"
    >
      <div class="flex items-center justify-between pb-2">
        <span class="text-xs font-semibold uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
          Router & REST API
        </span>
        <UIcon name="i-lucide-route" class="w-4 h-4 text-blue-500" />
      </div>
      <div class="flex items-baseline gap-2">
        <span class="text-2xl font-bold font-mono text-neutral-900 dark:text-white">
          {{ categoryCounts['router'] || 0 }}
        </span>
        <span class="text-xs text-neutral-500">Isu Rute</span>
      </div>
      <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1 line-clamp-1">
        Rute Mutasi Tanpa Auth, CORS Wildcard, Dangling Stubs
      </p>
    </div>

    <!-- Card 3: Code Smells -->
    <div
      class="p-4 rounded-xl border transition-all cursor-pointer"
      :class="activeCategory === 'code_smell'
        ? 'border-amber-500 bg-amber-50/50 dark:bg-amber-950/30 ring-1 ring-amber-500'
        : 'border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] hover:border-neutral-300 dark:hover:border-neutral-700'"
      @click="emit('selectCategory', activeCategory === 'code_smell' ? 'all' : 'code_smell')"
    >
      <div class="flex items-center justify-between pb-2">
        <span class="text-xs font-semibold uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
          Code Smells & Hygiene
        </span>
        <UIcon name="i-lucide-flame" class="w-4 h-4 text-amber-500" />
      </div>
      <div class="flex items-baseline gap-2">
        <span class="text-2xl font-bold font-mono text-neutral-900 dark:text-white">
          {{ (categoryCounts['code_smell'] || 0) + (categoryCounts['architecture'] || 0) }}
        </span>
        <span class="text-xs text-neutral-500">Isu Kualitas</span>
      </div>
      <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1 line-clamp-1">
        God Controller, Swallowed Error, Infinite Timeout
      </p>
    </div>

    <!-- Card 4: Security & Ports -->
    <div
      class="p-4 rounded-xl border transition-all cursor-pointer"
      :class="activeCategory === 'security'
        ? 'border-rose-500 bg-rose-50/50 dark:bg-rose-950/30 ring-1 ring-rose-500'
        : 'border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] hover:border-neutral-300 dark:hover:border-neutral-700'"
      @click="emit('selectCategory', activeCategory === 'security' ? 'all' : 'security')"
    >
      <div class="flex items-center justify-between pb-2">
        <span class="text-xs font-semibold uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
          Keamanan & Port
        </span>
        <UIcon name="i-lucide-shield-alert" class="w-4 h-4 text-rose-500" />
      </div>
      <div class="flex items-baseline gap-2">
        <span class="text-2xl font-bold font-mono text-neutral-900 dark:text-white">
          {{ (categoryCounts['security'] || 0) + (categoryCounts['port'] || 0) }}
        </span>
        <span class="text-xs text-neutral-500">Isu Keamanan</span>
      </div>
      <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1 line-clamp-1">
        Secret Leak, Proteksi .env, Port Collision
      </p>
    </div>

    <!-- Card 5: Tampilan & Performa UI (FE + Android/Mobile) -->
    <div
      class="p-4 rounded-xl border transition-all cursor-pointer"
      :class="activeCategory === 'ui'
        ? 'border-pink-500 bg-pink-50/50 dark:bg-pink-950/30 ring-1 ring-pink-500'
        : 'border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] hover:border-neutral-300 dark:hover:border-neutral-700'"
      @click="emit('selectCategory', activeCategory === 'ui' ? 'all' : 'ui')"
    >
      <div class="flex items-center justify-between pb-2">
        <span class="text-xs font-semibold uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
          UI & Tampilan
        </span>
        <UIcon name="i-lucide-layout-template" class="w-4 h-4 text-pink-500" />
      </div>
      <div class="flex items-baseline gap-2">
        <span class="text-2xl font-bold font-mono text-neutral-900 dark:text-white">
          {{ (categoryCounts['frontend'] || 0) + (categoryCounts['mobile'] || 0) }}
        </span>
        <span class="text-xs text-neutral-500">Isu FE & Mobile</span>
      </div>
      <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1 line-clamp-1">
        File Gemuk, Bundle/APK, A11y, Layout, Aset Berat
      </p>
    </div>
  </div>
</template>
