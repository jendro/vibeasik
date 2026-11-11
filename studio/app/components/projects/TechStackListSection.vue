<script setup lang="ts">
interface TechVersionItem {
  id: string
  name: string
  category: 'backend' | 'frontend' | 'mobile' | 'tool'
  categoryLabel: string
  icon: string
  color?: string
  latestVersion: string
  statusBadge: string
  badgeColor?: 'primary' | 'success' | 'info' | 'warning' | 'neutral'
  description: string
  adapterId?: string
}

const techItems: TechVersionItem[] = [
  {
    id: 'nuxt',
    name: 'Nuxt',
    category: 'frontend',
    categoryLabel: 'Fullstack Web',
    icon: 'i-simple-icons-nuxtdotjs',
    color: '#00DC82',
    latestVersion: 'v3.16.0 / v4.0',
    statusBadge: 'Nuxt 4 Ready',
    badgeColor: 'success',
    description: 'Framework Vue 3 SSR performa tinggi dengan Nitro server engine & Nuxt UI.',
    adapterId: 'nuxt'
  },
  {
    id: 'go',
    name: 'Go Web Service',
    category: 'backend',
    categoryLabel: 'Backend API',
    icon: 'i-simple-icons-go',
    color: '#00ADD8',
    latestVersion: 'v1.24.0',
    statusBadge: 'Latest Release',
    badgeColor: 'info',
    description: 'Backend ultra-cepat, memory-safe, dan konkurensi native (Fiber v2/v3, Gin, Chi).',
    adapterId: 'go'
  },
  {
    id: 'flutter',
    name: 'Flutter & Dart',
    category: 'mobile',
    categoryLabel: 'Mobile App',
    icon: 'i-simple-icons-flutter',
    color: '#02569B',
    latestVersion: 'Flutter 3.29 / Dart 3.7',
    statusBadge: 'Latest Stable',
    badgeColor: 'primary',
    description: 'Multiplatform client UI native untuk Android, iOS, dan desktop dengan Material 3.',
    adapterId: 'flutter'
  },
  {
    id: 'laravel',
    name: 'Laravel',
    category: 'backend',
    categoryLabel: 'Backend API',
    icon: 'i-simple-icons-laravel',
    color: '#FF2D20',
    latestVersion: 'v11.43.0 (PHP 8.2+)',
    statusBadge: 'LTS Active',
    badgeColor: 'warning',
    description: 'Framework PHP modern dengan Eloquent ORM, Sanctum guard, dan artisan routing.',
    adapterId: 'laravel'
  },
  {
    id: 'fastapi',
    name: 'FastAPI / Python',
    category: 'backend',
    categoryLabel: 'Backend & AI',
    icon: 'i-simple-icons-fastapi',
    color: '#009688',
    latestVersion: 'FastAPI 0.115 / Py 3.13',
    statusBadge: 'Production Ready',
    badgeColor: 'success',
    description: 'API Python asinkron berkecepatan tinggi dengan tipe Pydantic dan Swagger docs.',
    adapterId: 'python'
  },
  {
    id: 'nextjs',
    name: 'Next.js',
    category: 'frontend',
    categoryLabel: 'Frontend Web',
    icon: 'i-simple-icons-nextdotjs',
    color: '#000000',
    latestVersion: 'v15.2.0 (React 19)',
    statusBadge: 'Turbopack Ready',
    badgeColor: 'neutral',
    description: 'Framework React fullstack dengan App Router, Server Components, dan server actions.',
    adapterId: 'nextjs'
  },
  {
    id: 'nestjs',
    name: 'NestJS & Express',
    category: 'backend',
    categoryLabel: 'Backend API',
    icon: 'i-simple-icons-nestjs',
    color: '#E0234E',
    latestVersion: 'NestJS 11 / Express 5',
    statusBadge: 'Node 22 LTS',
    badgeColor: 'info',
    description: 'Arsitektur modular TypeScript decorator berbasis Node.js untuk enterprise microservices.',
    adapterId: 'express_nest'
  },
  {
    id: 'vue',
    name: 'Vue.js',
    category: 'frontend',
    categoryLabel: 'Frontend SPA',
    icon: 'i-simple-icons-vuedotjs',
    color: '#4FC08D',
    latestVersion: 'v3.5.13 (Vite 6)',
    statusBadge: 'Stable',
    badgeColor: 'success',
    description: 'Progressive JavaScript framework dengan Composition API dan reactivity engine.',
    adapterId: 'vue_router'
  },
  {
    id: 'react',
    name: 'React',
    category: 'frontend',
    categoryLabel: 'Frontend SPA',
    icon: 'i-simple-icons-react',
    color: '#61DAFB',
    latestVersion: 'v19.0.0',
    statusBadge: 'React 19',
    badgeColor: 'info',
    description: 'Library antarmuka pengguna deklaratif dengan server actions dan use hook modern.',
    adapterId: 'react_router'
  },
  {
    id: 'svelte',
    name: 'SvelteKit',
    category: 'frontend',
    categoryLabel: 'Frontend Web',
    icon: 'i-simple-icons-svelte',
    color: '#FF3E00',
    latestVersion: 'v2.17.0 (Svelte 5)',
    statusBadge: 'Svelte 5 Runes',
    badgeColor: 'warning',
    description: 'Framework web tanpa virtual DOM berbasis compiler dengan reaktivitas Runes.',
    adapterId: 'sveltekit'
  },
  {
    id: 'codeigniter',
    name: 'CodeIgniter',
    category: 'backend',
    categoryLabel: 'Backend API',
    icon: 'i-simple-icons-codeigniter',
    color: '#EF4223',
    latestVersion: 'v4.5.8 (PHP 8.2+)',
    statusBadge: 'Stable',
    badgeColor: 'neutral',
    description: 'Framework PHP berbobot ringan, jejak memori kecil, dan routing fleksibel.',
    adapterId: 'codeigniter'
  },
  {
    id: 'tailwind',
    name: 'Tailwind CSS',
    category: 'tool',
    categoryLabel: 'UI Styling',
    icon: 'i-simple-icons-tailwindcss',
    color: '#06B6D4',
    latestVersion: 'v4.0.9 (Oxide Engine)',
    statusBadge: 'Tailwind 4 Engine',
    badgeColor: 'primary',
    description: 'CSS framework utilitas modern tanpa konfigurasi JS berkat engine CSS native.',
    adapterId: ''
  }
]
</script>

<template>
  <div class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] p-5 space-y-4">
    <!-- Section Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-3 border-b border-neutral-100 dark:border-white/[0.08]">
      <div>
        <h2 class="text-sm sm:text-base font-bold text-neutral-900 dark:text-white flex items-center gap-2">
          <div class="p-1.5 rounded-lg bg-sky-50 dark:bg-sky-950/40 text-sky-600 dark:text-sky-400 border border-sky-200/60 dark:border-sky-800/60">
            <UIcon name="i-lucide-layers" class="w-4 h-4" />
          </div>
          <span>Ekosistem Teknologi & Versi Terbaru</span>
        </h2>
        <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5">
          Daftar teknologi dan framework utama yang didukung oleh Vibeasik. Cek versi terkini agar proyek Anda tetap up-to-date.
        </p>
      </div>

      <div class="flex items-center gap-2 shrink-0">
        <NuxtLink
          to="/adapters"
          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold text-sky-600 dark:text-sky-400 bg-sky-50 dark:bg-sky-950/40 hover:bg-sky-100 dark:hover:bg-sky-900/40 border border-sky-200/60 dark:border-sky-800/60 transition-colors"
        >
          <span>Katalog Adapter Scanner</span>
          <UIcon name="i-lucide-arrow-right" class="w-3.5 h-3.5" />
        </NuxtLink>
      </div>
    </div>

    <!-- Tech Cards Grid -->
    <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 gap-3">
      <NuxtLink
        v-for="item in techItems"
        :key="item.id"
        :to="item.adapterId ? `/adapters` : '#'"
        class="group p-3 rounded-xl border border-neutral-200/80 dark:border-white/[0.08] bg-neutral-50/50 dark:bg-neutral-800/30 hover:bg-white dark:hover:bg-slate-800/60 hover:border-neutral-300 dark:hover:border-neutral-700 transition-all flex flex-col justify-between space-y-2 cursor-pointer"
      >
        <div class="space-y-2">
          <!-- Logo & Status Badge -->
          <div class="flex items-start justify-between gap-1.5">
            <div
              class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0 border border-neutral-200/80 dark:border-neutral-700/60 bg-white dark:bg-neutral-800 group-hover:scale-105 transition-transform"
            >
              <UIcon
                :name="item.icon"
                class="w-4.5 h-4.5"
                :style="item.color ? { color: item.color } : undefined"
              />
            </div>
            <UBadge
              size="xs"
              :color="item.badgeColor || 'primary'"
              variant="subtle"
              class="text-xs px-1.5 py-0 font-medium truncate max-w-[95px]"
            >
              {{ item.statusBadge }}
            </UBadge>
          </div>

          <!-- Name & Category -->
          <div>
            <div class="font-bold text-xs text-neutral-900 dark:text-white truncate group-hover:text-sky-600 dark:group-hover:text-sky-400 transition-colors">
              {{ item.name }}
            </div>
            <div class="text-xs text-neutral-400 dark:text-neutral-500 font-mono">
              {{ item.categoryLabel }}
            </div>
          </div>
        </div>

        <!-- Latest Version Tag -->
        <div class="pt-2 border-t border-neutral-200/60 dark:border-neutral-700/50">
          <div class="text-xs text-neutral-400 uppercase font-mono">Versi Terbaru:</div>
          <div class="text-xs font-mono font-bold text-neutral-800 dark:text-neutral-200 truncate mt-0.5" :title="item.latestVersion">
            {{ item.latestVersion }}
          </div>
        </div>
      </NuxtLink>
    </div>
  </div>
</template>
