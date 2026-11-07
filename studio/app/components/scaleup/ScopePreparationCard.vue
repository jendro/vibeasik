<script setup lang="ts">
defineProps<{
  scope: 'mobile' | 'frontend'
}>()

const emit = defineEmits<{
  (e: 'backToBackend'): void
}>()

const mobilePillars = [
  {
    title: 'Clean Architecture (BLoC / Riverpod)',
    desc: 'Pemisahan Presentation (UI), Domain (UseCases murni), dan Data (Repository + Local/Remote DataSource).',
    icon: 'i-lucide-layers'
  },
  {
    title: 'Feature-First Folder Layout',
    desc: 'Pengelompokan folder per fitur (lib/features/...) untuk menghindari file jumping dan mempermudah AI agent.',
    icon: 'i-lucide-folder-tree'
  },
  {
    title: 'Offline-First & Background Sync',
    desc: 'Penyimpanan lokal cepat (Isar/Hive/Drift) dengan antrean sinkronisasi background saat koneksi offline.',
    icon: 'i-lucide-wifi-off'
  },
  {
    title: 'Declarative Routing & Deep Linking',
    desc: 'Navigasi deklaratif berbasis URL (go_router) dengan penanganan state query parameter dan back stack aman.',
    icon: 'i-lucide-network'
  }
]

const frontendPillars = [
  {
    title: 'Single Responsibility & Anti-Monolith',
    desc: 'Komponen antarmuka maksimal 250 baris, memisahkan Presentational Dumb Component dari Container Component.',
    icon: 'i-lucide-split'
  },
  {
    title: 'Automation-First Design',
    desc: 'Minimalkan informasi berlebih, otomatiskan alur di background, tampilkan hanya hal yang membutuhkan keputusan user.',
    icon: 'i-lucide-sparkles'
  },
  {
    title: 'Composable State & SSR Optimization',
    desc: 'Logika reaktif diisolasi ke dalam composables (useApi, useLLMState) dengan optimasi Core Web Vitals (LCP).',
    icon: 'i-lucide-cpu'
  },
  {
    title: 'Type-Safe Shared Domain Contracts',
    desc: 'Kontrak interface TypeScript terpusat di app/types/ yang sinkron dengan DTO backend tanpa any.',
    icon: 'i-lucide-shield-check'
  }
]
</script>

<template>
  <div class="p-6 sm:p-8 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 shadow-xs space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-6 border-b border-neutral-100 dark:border-neutral-800">
      <div class="flex items-start gap-3.5">
        <div
          class="w-12 h-12 rounded-xl flex items-center justify-center shrink-0 border"
          :class="scope === 'mobile'
            ? 'bg-blue-50 text-blue-600 dark:bg-blue-950/60 dark:text-blue-400 border-blue-500/20'
            : 'bg-sky-50 text-sky-600 dark:bg-sky-950/60 dark:text-sky-400 border border-sky-500/20'"
        >
          <UIcon :name="scope === 'mobile' ? 'i-lucide-smartphone' : 'i-lucide-layout'" class="w-6 h-6" />
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h2 class="text-lg font-bold text-neutral-900 dark:text-white">
              {{ scope === 'mobile' ? 'Scale Up Arsitektur Mobile (Flutter & React Native)' : 'Scale Up Arsitektur Frontend Web (Nuxt & Vue/React)' }}
            </h2>
            <span class="text-xs px-2 py-0.5 rounded-full font-semibold bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300">
              Disiapkan untuk Fase Selanjutnya
            </span>
          </div>
          <p class="text-xs sm:text-sm text-neutral-500 dark:text-neutral-400 mt-1 max-w-2xl">
            Modul best practice dunia nyata untuk {{ scope === 'mobile' ? 'aplikasi mobile' : 'antarmuka web frontend' }} sedang dalam persiapan. Saat ini Anda dapat mempelajari arsitektur Backend yang telah aktif penuh.
          </p>
        </div>
      </div>

      <button
        type="button"
        class="inline-flex items-center gap-2 px-4 py-2 text-xs font-semibold rounded-lg bg-sky-600 text-white hover:bg-sky-500 transition-colors cursor-pointer self-start sm:self-center shrink-0"
        @click="emit('backToBackend')"
      >
        <UIcon name="i-lucide-server" class="w-4 h-4" />
        <span>Buka Modul Backend (Aktif)</span>
      </button>
    </div>

    <!-- 4 Pillars Preview -->
    <div class="space-y-3">
      <h3 class="text-xs font-bold uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
        Standar Best Practice yang Sedang Disiapkan:
      </h3>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div
          v-for="p in (scope === 'mobile' ? mobilePillars : frontendPillars)"
          :key="p.title"
          class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-800/40 space-y-1.5"
        >
          <div class="flex items-center gap-2">
            <UIcon :name="p.icon" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
            <h4 class="text-xs font-bold text-neutral-900 dark:text-white">
              {{ p.title }}
            </h4>
          </div>
          <p class="text-xs text-neutral-600 dark:text-neutral-300 leading-relaxed">
            {{ p.desc }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
