<script setup lang="ts">
import type { AdaptersCatalogSummary, AdapterMetadata } from '~/types'

const { getAdapters } = useApi()

const catalog = ref<AdaptersCatalogSummary | null>(null)
const loading = ref(true)
const error = ref('')

const searchQuery = ref('')
const selectedType = ref<'all' | 'backend' | 'frontend'>('all')
const selectedLanguage = ref<string>('all')
const expandedCards = ref<Record<string, boolean>>({})

const loadCatalog = async () => {
  loading.value = true
  error.value = ''
  try {
    catalog.value = await getAdapters()
    // By default expand top 2 or expand all for fast visibility
    if (catalog.value?.adapters) {
      catalog.value.adapters.forEach((a) => {
        expandedCards.value[a.id] = true
      })
    }
  } catch (err: any) {
    error.value = err?.data?.error || err?.message || 'Gagal memuat katalog adapter dari server'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadCatalog()
})

const toggleExpand = (id: string) => {
  expandedCards.value[id] = !expandedCards.value[id]
}

const toggleAll = () => {
  if (!catalog.value) return
  const anyClosed = filteredAdapters.value.some(a => !expandedCards.value[a.id])
  filteredAdapters.value.forEach((a) => {
    expandedCards.value[a.id] = anyClosed
  })
}

const areAllExpanded = computed(() => {
  if (!filteredAdapters.value.length) return false
  return filteredAdapters.value.every(a => expandedCards.value[a.id])
})

const availableLanguages = computed(() => {
  if (!catalog.value) return []
  const set = new Set<string>()
  catalog.value.adapters.forEach((a) => {
    if (a.language && a.language !== 'all') {
      set.add(a.language)
    }
  })
  return Array.from(set).sort()
})

const filteredAdapters = computed<AdapterMetadata[]>(() => {
  if (!catalog.value) return []
  return catalog.value.adapters.filter((item) => {
    // Type filter
    if (selectedType.value !== 'all' && item.type !== selectedType.value) {
      return false
    }

    // Language filter
    if (selectedLanguage.value !== 'all' && item.language !== selectedLanguage.value) {
      return false
    }

    // Search query filter
    if (searchQuery.value.trim()) {
      const q = searchQuery.value.toLowerCase().trim()
      const inName = item.name.toLowerCase().includes(q)
      const inId = item.id.toLowerCase().includes(q)
      const inDesc = item.description.toLowerCase().includes(q)
      const inLang = item.language.toLowerCase().includes(q)
      const inFramework = item.framework.toLowerCase().includes(q)
      const inRules = item.detect_rules.some(r => r.toLowerCase().includes(q))
      const inMethods = item.scan_methods.some(m => m.toLowerCase().includes(q))
      const inFeatures = item.features.some(f => f.toLowerCase().includes(q))
      return inName || inId || inDesc || inLang || inFramework || inRules || inMethods || inFeatures
    }

    return true
  })
})

const resetFilters = () => {
  searchQuery.value = ''
  selectedType.value = 'all'
  selectedLanguage.value = 'all'
}

const DEFAULT_VERSIONS: Record<string, { version: string, badge: string }> = {
  go: { version: 'Go 1.24.0 (Fiber v2.52/v3, Gin v1.10)', badge: 'Latest Release' },
  nuxt: { version: 'v3.16.0 / v4.0 (Vue 3.5+, Nitro 2.10)', badge: 'Nuxt 4 Ready' },
  nextjs: { version: 'v15.2.0 (React 19 / Turbopack)', badge: 'Latest Release' },
  laravel: { version: 'v11.43.0 (PHP 8.2+)', badge: 'LTS Active' },
  codeigniter: { version: 'v4.5.8 (PHP 8.2+)', badge: 'Stable' },
  python: { version: 'FastAPI v0.115.8 (Python 3.13)', badge: 'Production Ready' },
  express_nest: { version: 'NestJS v11.0 / Express v5.0 (Node 22 LTS)', badge: 'Active LTS' },
  flutter: { version: 'v3.29.0 (Dart 3.7.0)', badge: 'Latest Stable' },
  sveltekit: { version: 'v2.17.0 (Svelte 5 Runes)', badge: 'Svelte 5 Modern' },
  vue_router: { version: 'Vue Router v4.5.0 (Vue 3.5.13)', badge: 'Stable' },
  react_router: { version: 'React Router v7.2.0 (React 19)', badge: 'v7 Framework Ready' },
  generic_frontend: { version: 'HTML5 / ES2024 Standards', badge: 'Universal' }
}

const getAdapterVersion = (adapter: AdapterMetadata) => {
  return adapter.latest_version || DEFAULT_VERSIONS[adapter.id]?.version || 'Terbaru'
}

const getAdapterBadge = (adapter: AdapterMetadata) => {
  return adapter.status_badge || DEFAULT_VERSIONS[adapter.id]?.badge || 'Aktif'
}

const getLanguageLabel = (lang: string) => {
  switch (lang.toLowerCase()) {
    case 'php': return 'PHP'
    case 'go': return 'Go'
    case 'typescript': return 'TypeScript'
    case 'javascript': return 'JavaScript'
    case 'python': return 'Python'
    case 'dart': return 'Dart'
    case 'svelte': return 'Svelte'
    case 'all': return 'Semua Bahasa'
    default: return lang
  }
}
</script>

<template>
  <div class="space-y-6 max-w-6xl mx-auto pb-12">
    <!-- Top Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-neutral-200 dark:border-white/10">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white flex items-center gap-2.5">
          <div class="p-2 rounded-lg bg-sky-50 dark:bg-sky-950/40 text-sky-600 dark:text-sky-400 border border-sky-200/60 dark:border-sky-800/60">
            <UIcon name="i-lucide-layers" class="w-5 h-5" />
          </div>
          Katalog Adapter Scanner
        </h1>
        <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1">
          Daftar adapter resmi untuk mendeteksi rute endpoint backend, file routing frontend, dan integrasi API otomatis.
        </p>
      </div>

      <div class="flex items-center gap-2">
        <UButton
          icon="i-lucide-refresh-cw"
          color="neutral"
          variant="outline"
          size="sm"
          :loading="loading"
          title="Muat ulang daftar adapter"
          @click="loadCatalog"
        >
          Muat Ulang
        </UButton>

        <UButton
          to="/scan"
          icon="i-lucide-folder-search"
          color="primary"
          size="sm"
        >
          Pindai Laptop
        </UButton>
      </div>
    </div>

    <!-- Error Alert -->
    <div
      v-if="error"
      class="p-4 rounded-xl border border-rose-300 dark:border-rose-900/60 bg-rose-50 dark:bg-rose-950/20 text-xs text-rose-800 dark:text-rose-300 flex items-center justify-between"
    >
      <div class="flex items-center gap-2">
        <UIcon name="i-lucide-alert-circle" class="w-4 h-4 text-rose-600 dark:text-rose-400" />
        <span>{{ error }}</span>
      </div>
      <UButton size="xs" color="error" variant="subtle" @click="loadCatalog">
        Coba Lagi
      </UButton>
    </div>

    <!-- Summary Metrics Bar -->
    <div v-if="catalog" class="grid grid-cols-2 sm:grid-cols-4 gap-3">
      <!-- Metric 1: Total -->
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 flex items-center gap-3">
        <div class="p-2.5 rounded-lg bg-neutral-100 dark:bg-slate-800/40 text-neutral-700 dark:text-neutral-300">
          <UIcon name="i-lucide-box" class="w-5 h-5" />
        </div>
        <div>
          <div class="text-xs text-neutral-500 dark:text-neutral-400 font-medium">
            Total Adapter
          </div>
          <div class="text-xl font-bold text-neutral-900 dark:text-white">
            {{ catalog.total }} Mesin
          </div>
        </div>
      </div>

      <!-- Metric 2: Backend -->
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 flex items-center gap-3">
        <div class="p-2.5 rounded-lg bg-sky-50 dark:bg-sky-950/30 text-sky-600 dark:text-sky-400">
          <UIcon name="i-lucide-server" class="w-5 h-5" />
        </div>
        <div>
          <div class="text-xs text-neutral-500 dark:text-neutral-400 font-medium">
            Backend Engines
          </div>
          <div class="text-xl font-bold text-sky-600 dark:text-sky-400">
            {{ catalog.backend }} Framework
          </div>
        </div>
      </div>

      <!-- Metric 3: Frontend -->
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 flex items-center gap-3">
        <div class="p-2.5 rounded-lg bg-sky-50 dark:bg-sky-950/30 text-sky-600 dark:text-sky-400">
          <UIcon name="i-lucide-app-window" class="w-5 h-5" />
        </div>
        <div>
          <div class="text-xs text-neutral-500 dark:text-neutral-400 font-medium">
            Frontend Engines
          </div>
          <div class="text-xl font-bold text-sky-600 dark:text-sky-400">
            {{ catalog.frontend }} Framework
          </div>
        </div>
      </div>

      <!-- Metric 4: Languages -->
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 flex items-center gap-3">
        <div class="p-2.5 rounded-lg bg-amber-50 dark:bg-amber-950/30 text-amber-600 dark:text-amber-400">
          <UIcon name="i-lucide-binary" class="w-5 h-5" />
        </div>
        <div>
          <div class="text-xs text-neutral-500 dark:text-neutral-400 font-medium">
            Bahasa Didukung
          </div>
          <div class="text-xl font-bold text-amber-600 dark:text-amber-400">
            {{ catalog.languages?.length || 5 }} Bahasa
          </div>
        </div>
      </div>
    </div>

    <!-- Filter & Search Controls -->
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <!-- Search Input -->
        <div class="relative flex-1 max-w-md">
          <UIcon name="i-lucide-search" class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-neutral-400" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Cari adapter, framework, file contoh, atau fitur..."
            class="w-full pl-9 pr-8 py-2 text-xs rounded-lg border border-neutral-200 dark:border-neutral-700 bg-neutral-50 dark:bg-slate-800/40 text-neutral-900 dark:text-white placeholder:text-neutral-400 focus:outline-none focus:ring-2 focus:ring-sky-500"
          >
          <button
            v-if="searchQuery"
            class="absolute right-2.5 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200"
            @click="searchQuery = ''"
          >
            <UIcon name="i-lucide-x" class="w-3.5 h-3.5" />
          </button>
        </div>

        <!-- Type Selector Tabs -->
        <div class="flex items-center gap-1 p-1 bg-neutral-100 dark:bg-slate-800/40 rounded-lg text-xs self-start sm:self-auto">
          <button
            type="button"
            class="px-3 py-1.5 rounded-md font-medium transition-colors"
            :class="selectedType === 'all' ? 'bg-white dark:bg-neutral-700 text-neutral-900 dark:text-white shadow-xs' : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="selectedType = 'all'"
          >
            Semua ({{ catalog?.total || 0 }})
          </button>
          <button
            type="button"
            class="px-3 py-1.5 rounded-md font-medium transition-colors"
            :class="selectedType === 'backend' ? 'bg-white dark:bg-neutral-700 text-sky-600 dark:text-sky-400 shadow-xs' : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="selectedType = 'backend'"
          >
            Backend ({{ catalog?.backend || 0 }})
          </button>
          <button
            type="button"
            class="px-3 py-1.5 rounded-md font-medium transition-colors"
            :class="selectedType === 'frontend' ? 'bg-white dark:bg-neutral-700 text-sky-600 dark:text-sky-400 shadow-xs' : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="selectedType = 'frontend'"
          >
            Frontend ({{ catalog?.frontend || 0 }})
          </button>
        </div>

        <!-- Expand / Collapse All -->
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          :icon="areAllExpanded ? 'i-lucide-chevrons-up' : 'i-lucide-chevrons-down'"
          @click="toggleAll"
        >
          {{ areAllExpanded ? 'Tutup Semua Detail' : 'Buka Semua Detail' }}
        </UButton>
      </div>

      <!-- Language Filter Pills -->
      <div class="flex items-center gap-1.5 flex-wrap pt-2 border-t border-neutral-100 dark:border-white/10 text-xs">
        <span class="text-neutral-400 dark:text-neutral-500 font-medium mr-1 flex items-center gap-1">
          <UIcon name="i-lucide-filter" class="w-3.5 h-3.5" />
          Bahasa:
        </span>
        <button
          type="button"
          class="px-2.5 py-1 rounded-full text-xs font-medium border transition-colors"
          :class="selectedLanguage === 'all' ? 'bg-neutral-900 dark:bg-white text-white dark:text-neutral-900 border-neutral-900 dark:border-white' : 'bg-transparent text-neutral-600 dark:text-neutral-400 border-neutral-200 dark:border-neutral-700 hover:border-neutral-300 dark:hover:border-neutral-600'"
          @click="selectedLanguage = 'all'"
        >
          Semua
        </button>
        <button
          v-for="lang in availableLanguages"
          :key="lang"
          type="button"
          class="px-2.5 py-1 rounded-full text-xs font-medium border transition-colors flex items-center gap-1"
          :class="selectedLanguage === lang ? 'bg-neutral-900 dark:bg-white text-white dark:text-neutral-900 border-neutral-900 dark:border-white' : 'bg-transparent text-neutral-600 dark:text-neutral-400 border-neutral-200 dark:border-neutral-700 hover:border-neutral-300 dark:hover:border-neutral-600'"
          @click="selectedLanguage = lang"
        >
          {{ getLanguageLabel(lang) }}
        </button>
      </div>
    </div>

    <!-- Loading Skeleton -->
    <div v-if="loading && !catalog" class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div v-for="i in 4" :key="i" class="p-5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3 animate-pulse">
        <div class="h-6 bg-neutral-200 dark:bg-slate-800/40 rounded w-1/3" />
        <div class="h-4 bg-neutral-100 dark:bg-slate-800/40 rounded w-3/4" />
        <div class="h-16 bg-neutral-100 dark:bg-slate-800/40 rounded" />
      </div>
    </div>

    <!-- Empty State -->
    <div
      v-else-if="filteredAdapters.length === 0"
      class="p-12 text-center rounded-xl border border-dashed border-neutral-300 dark:border-neutral-700 bg-white/50 dark:bg-slate-900/50 space-y-3"
    >
      <div class="inline-flex p-3 rounded-full bg-neutral-100 dark:bg-slate-800/40 text-neutral-400">
        <UIcon name="i-lucide-search-x" class="w-6 h-6" />
      </div>
      <h3 class="text-sm font-semibold text-neutral-900 dark:text-white">
        Tidak ada adapter yang cocok
      </h3>
      <p class="text-xs text-neutral-500 dark:text-neutral-400 max-w-sm mx-auto">
        Kriteria pencarian "{{ searchQuery }}" atau filter yang dipilih tidak menemukan adapter terdaftar.
      </p>
      <UButton size="xs" color="neutral" variant="outline" @click="resetFilters">
        Reset Filter
      </UButton>
    </div>

    <!-- Adapters Grid -->
    <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <div
        v-for="adapter in filteredAdapters"
        :key="adapter.id"
        class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 transition-all hover:border-neutral-300 dark:hover:border-neutral-700 shadow-xs flex flex-col justify-between"
      >
        <!-- Card Header & Core Info -->
        <div class="p-5 space-y-3">
          <div class="flex items-start justify-between gap-3">
            <!-- Framework & Icon -->
            <div class="flex items-center gap-2.5">
              <TechBadge
                :tech="adapter.framework || adapter.id"
                size="md"
                variant="subtle"
                :show-label="false"
              />
              <div>
                <h3 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-1.5">
                  {{ adapter.name }}
                </h3>
                <div class="flex items-center gap-1.5 mt-0.5">
                  <span class="text-xs text-neutral-500 dark:text-neutral-400 font-mono">
                    {{ adapter.id }}
                  </span>
                  <span class="text-neutral-300 dark:text-neutral-700">•</span>
                  <span class="text-xs text-neutral-500 dark:text-neutral-400 capitalize">
                    {{ getLanguageLabel(adapter.language) }}
                  </span>
                </div>
              </div>
            </div>

            <!-- Badges: Type & Active Status -->
            <div class="flex items-center gap-1.5">
              <span
                class="px-2 py-0.5 text-xs font-semibold rounded-md border"
                :class="adapter.type === 'backend'
                  ? 'bg-sky-50 dark:bg-sky-950/40 text-sky-700 dark:text-sky-300 border-sky-200 dark:border-sky-800/60'
                  : 'bg-sky-50 dark:bg-sky-950/40 text-sky-700 dark:text-sky-300 border-sky-200 dark:border-sky-800/60'"
              >
                {{ adapter.type === 'backend' ? 'Backend' : 'Frontend' }}
              </span>

              <span
                class="inline-flex items-center gap-1 px-1.5 py-0.5 text-xs font-medium rounded-md bg-sky-500/10 text-sky-600 dark:text-sky-400 border border-sky-500/20"
                title="Adapter aktif & siap digunakan"
              >
                <span class="w-1.5 h-1.5 rounded-full bg-sky-500 animate-pulse" />
                Aktif
              </span>
            </div>
          </div>

          <!-- Description -->
          <p class="text-xs text-neutral-600 dark:text-neutral-300 leading-relaxed">
            {{ adapter.description }}
          </p>

          <!-- Latest Version Info -->
          <div class="flex flex-wrap items-center gap-2 pt-0.5">
            <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-xs font-mono font-medium bg-neutral-100 dark:bg-slate-800 text-neutral-800 dark:text-neutral-200 border border-neutral-200/80 dark:border-neutral-700/80">
              <UIcon name="i-lucide-tag" class="w-3.5 h-3.5 text-sky-500" />
              <span>Versi Terbaru: <strong class="text-neutral-900 dark:text-white">{{ getAdapterVersion(adapter) }}</strong></span>
            </span>
            <UBadge
              size="xs"
              color="primary"
              variant="subtle"
              class="text-xs"
            >
              {{ getAdapterBadge(adapter) }}
            </UBadge>
          </div>

          <!-- Example File Tag -->
          <div class="flex items-center gap-2 pt-1 text-xs text-neutral-500 dark:text-neutral-400">
            <span class="font-medium text-neutral-700 dark:text-neutral-300 flex items-center gap-1">
              <UIcon name="i-lucide-file-code-2" class="w-3.5 h-3.5 text-neutral-400" />
              File Contoh:
            </span>
            <code class="px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-slate-800/40 font-mono text-xs text-neutral-800 dark:text-neutral-200 border border-neutral-200/60 dark:border-neutral-700/60">
              {{ adapter.example_file }}
            </code>
          </div>

          <!-- Accordion Toggle Button -->
          <div class="pt-2 border-t border-neutral-100 dark:border-white/10 flex items-center justify-between">
            <button
              type="button"
              class="text-xs font-medium text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white flex items-center gap-1 transition-colors"
              @click="toggleExpand(adapter.id)"
            >
              <span>{{ expandedCards[adapter.id] ? 'Sembunyikan Spesifikasi' : 'Lihat Aturan & Fitur Lengkap' }}</span>
              <UIcon
                :name="expandedCards[adapter.id] ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
                class="w-3.5 h-3.5"
              />
            </button>

            <span class="text-xs text-neutral-400">
              {{ adapter.features.length }} Fitur Terintegrasi
            </span>
          </div>

          <!-- Expandable Technical Details -->
          <div v-show="expandedCards[adapter.id]" class="space-y-3 pt-2 text-xs">
            <!-- Detection Rules -->
            <div class="p-3 rounded-lg bg-neutral-50 dark:bg-slate-800/50 border border-neutral-100 dark:border-white/10">
              <div class="font-semibold text-neutral-800 dark:text-neutral-200 flex items-center gap-1.5 mb-1.5 text-xs">
                <UIcon name="i-lucide-scan-search" class="w-3.5 h-3.5 text-amber-500" />
                Aturan Deteksi Otomatis (Trigger):
              </div>
              <ul class="space-y-1 pl-4 list-disc text-neutral-600 dark:text-neutral-400 text-xs">
                <li v-for="(rule, idx) in adapter.detect_rules" :key="idx">
                  {{ rule }}
                </li>
              </ul>
            </div>

            <!-- Scan Methods -->
            <div class="p-3 rounded-lg bg-neutral-50 dark:bg-slate-800/50 border border-neutral-100 dark:border-white/10">
              <div class="font-semibold text-neutral-800 dark:text-neutral-200 flex items-center gap-1.5 mb-1.5 text-xs">
                <UIcon name="i-lucide-cpu" class="w-3.5 h-3.5 text-sky-500" />
                Metode Ekstraksi Rute:
              </div>
              <ul class="space-y-1 pl-4 list-disc text-neutral-600 dark:text-neutral-400 text-xs">
                <li v-for="(method, idx) in adapter.scan_methods" :key="idx">
                  {{ method }}
                </li>
              </ul>
            </div>

            <!-- Features Checklist -->
            <div class="p-3 rounded-lg bg-neutral-50 dark:bg-slate-800/50 border border-neutral-100 dark:border-white/10">
              <div class="font-semibold text-neutral-800 dark:text-neutral-200 flex items-center gap-1.5 mb-2 text-xs">
                <UIcon name="i-lucide-badge-check" class="w-3.5 h-3.5 text-sky-400" />
                Kapabilitas & Dukungan Sintaks:
              </div>
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-1.5 text-xs">
                <div
                  v-for="(feature, idx) in adapter.features"
                  :key="idx"
                  class="flex items-start gap-1.5 text-neutral-700 dark:text-neutral-300"
                >
                  <UIcon name="i-lucide-check" class="w-3.5 h-3.5 text-sky-400 shrink-0 mt-0.5" />
                  <span>{{ feature }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Card Footer Actions -->
        <div class="px-5 py-3 border-t border-neutral-100 dark:border-white/10 bg-neutral-50/50 dark:bg-slate-800/20 rounded-b-xl flex items-center justify-between gap-2">
          <div class="text-xs text-neutral-400 flex items-center gap-1 font-mono">
            <span class="w-2 h-2 rounded-full bg-sky-500/60 inline-block" />
            Default Registry: OK
          </div>

          <UButton
            to="/scan"
            size="xs"
            color="neutral"
            variant="ghost"
            icon="i-lucide-arrow-right"
            trailing
          >
            Gunakan di Scanner
          </UButton>
        </div>
      </div>
    </div>
  </div>
</template>
