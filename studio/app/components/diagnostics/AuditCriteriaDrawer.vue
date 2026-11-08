<script setup lang="ts">
import type { GuidelineCategory } from '~/types'
import { AUDIT_CRITERIA as defaultCriteria } from '~/utils/auditCriteria'

const props = defineProps<{
  open: boolean
  categories?: GuidelineCategory[]
}>()

const emit = defineEmits<{
  (e: 'update:open', val: boolean): void
}>()

// State
const searchQuery = ref('')
const activeTab = ref<string>('all')
const activeSeverity = ref<string>('all')

const categoriesList = [
  { id: 'all', name: 'Semua Kriteria', icon: 'i-lucide-layers', count: 0 },
  { id: 'deep_only', name: '🔍 Deep Check Saja', icon: 'i-lucide-scan-search', count: 0 },
  { id: 'mechanism', name: '⚙️ Mekanisme Pengecekan', icon: 'i-lucide-cpu', count: '4 Layer' },
  { id: 'security', name: '🛡️ Keamanan & Kerentanan', icon: 'i-lucide-shield-alert', count: 0 },
  { id: 'critical_logic', name: '⚡ Logika Kritis & Kueri/Proses Berat', icon: 'i-lucide-zap', count: 0 },
  { id: 'backend', name: '⚙️ Backend (BE) & API', icon: 'i-lucide-server', count: 0 },
  { id: 'frontend', name: '💻 Frontend (FE) & Antarmuka', icon: 'i-lucide-layout', count: 0 },
  { id: 'mobile', name: '📱 Mobile (Android/Flutter)', icon: 'i-lucide-smartphone', count: 0 },
  { id: 'smelly_design', name: '🎨 Smelly Design & UX', icon: 'i-lucide-palette', count: 0 },
  { id: 'code_smell', name: '🧩 Smelly Code & Arsitektur', icon: 'i-lucide-code-2', count: 0 },
  { id: 'deployment', name: '🚀 Deploy: Multi Env & Lokasi Berkas', icon: 'i-lucide-hard-drive-upload', count: 0 }
]

// Tab counts computation
const categoriesWithCounts = computed(() => {
  return categoriesList.map((cat) => {
    if (cat.id === 'all') {
      return { ...cat, count: defaultCriteria.length }
    }
    if (cat.id === 'deep_only') {
      const count = defaultCriteria.filter(item => item.is_deep).length
      return { ...cat, count }
    }
    if (cat.id === 'mechanism') {
      return { ...cat, count: 'Engine' }
    }
    const count = defaultCriteria.filter(item => item.category === cat.id).length
    return { ...cat, count }
  })
})

// Filtered criteria list
const filteredCriteria = computed(() => {
  if (activeTab.value === 'mechanism') {
    return []
  }

  return defaultCriteria.filter((item) => {
    // Tab filter
    if (activeTab.value === 'deep_only') {
      if (!item.is_deep) return false
    } else if (activeTab.value !== 'all' && item.category !== activeTab.value) {
      return false
    }

    // Severity filter
    if (activeSeverity.value !== 'all' && item.severity !== activeSeverity.value) {
      return false
    }

    // Search query
    if (searchQuery.value.trim() !== '') {
      const q = searchQuery.value.toLowerCase()
      const matchId = item.id.toLowerCase().includes(q)
      const matchTitle = item.title.toLowerCase().includes(q)
      const matchRule = item.rule.toLowerCase().includes(q)
      const matchAction = item.suggested_action.toLowerCase().includes(q)
      const matchRationale = item.rationale.toLowerCase().includes(q)
      const matchScope = item.scope.toLowerCase().includes(q)
      if (!matchId && !matchTitle && !matchRule && !matchAction && !matchRationale && !matchScope) {
        return false
      }
    }

    return true
  })
})

// Close on Escape
const onKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.open) {
    emit('update:open', false)
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeyDown)
})
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-center justify-end bg-black/60 backdrop-blur-xs transition-opacity duration-200"
    @click.self="emit('update:open', false)"
  >
    <div
      class="w-full max-w-3xl h-full bg-white dark:bg-neutral-900 border-l border-neutral-200 dark:border-neutral-800 shadow-2xl flex flex-col duration-200 overflow-hidden"
    >
      <!-- Top Header -->
      <div class="p-5 border-b border-neutral-200 dark:border-neutral-800 flex items-start justify-between gap-4 bg-white dark:bg-neutral-900 shrink-0">
        <div class="flex items-start gap-3 min-w-0">
          <span class="p-2.5 rounded-xl bg-violet-100 dark:bg-violet-950/60 text-violet-700 dark:text-violet-400 border border-violet-200 dark:border-violet-800 shrink-0 mt-0.5">
            <UIcon name="i-lucide-list-checks" class="w-5 h-5" />
          </span>
          <div class="space-y-1 min-w-0">
            <div class="flex items-center gap-2 flex-wrap">
              <h2 class="text-base font-bold text-neutral-900 dark:text-white truncate">
                Daftar Cek & Standar Audit Diagnostik
              </h2>
              <span class="px-2 py-0.5 rounded-full text-xs font-bold font-mono bg-violet-100 dark:bg-violet-950/60 text-violet-700 dark:text-violet-300 border border-violet-200 dark:border-violet-800">
                {{ defaultCriteria.length }} Aturan Diperiksa
              </span>
            </div>
            <p class="text-xs text-neutral-500 dark:text-neutral-400 leading-relaxed">
              Kamus standar pemeriksaan menyeluruh untuk Backend (BE), Frontend (FE), Mobile, keamanan kritis (SQLi/RCE), kueri/proses berat, serta celah smelly code & design.
            </p>
          </div>
        </div>

        <button
          type="button"
          class="p-1.5 rounded-lg text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 cursor-pointer shrink-0 transition-colors"
          title="Tutup Panel"
          @click="emit('update:open', false)"
        >
          <UIcon name="i-lucide-x" class="w-5 h-5" />
        </button>
      </div>

      <!-- Search and Filter Bar -->
      <div class="p-4 border-b border-neutral-200 dark:border-neutral-800 bg-neutral-50/70 dark:bg-neutral-900/60 flex flex-col sm:flex-row items-stretch sm:items-center gap-3 shrink-0">
        <!-- Search Input -->
        <div class="relative flex-1">
          <UIcon name="i-lucide-search" class="w-4 h-4 text-neutral-400 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
          <input
            v-model="searchQuery"
            type="text"
            :placeholder="activeTab === 'mechanism' ? 'Cari topik mekanisme, traversal, layer, regex, atau tips akurasi...' : 'Cari ID celah, kata kunci kueri, keamanan, smell, atau solusi...'"
            class="w-full pl-9 pr-8 py-1.5 text-xs rounded-lg border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white placeholder-neutral-400 focus:outline-none focus:ring-1 focus:ring-violet-500"
          >
          <button
            v-if="searchQuery"
            type="button"
            class="absolute right-2.5 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200 text-xs"
            @click="searchQuery = ''"
          >
            <UIcon name="i-lucide-x" class="w-3.5 h-3.5" />
          </button>
        </div>

        <!-- Severity Filter Dropdown (hidden on mechanism tab) -->
        <div v-if="activeTab !== 'mechanism'" class="flex items-center gap-1.5 shrink-0 text-xs">
          <span class="text-neutral-500 dark:text-neutral-400 text-xs font-medium hidden sm:inline">Keparahan:</span>
          <select
            v-model="activeSeverity"
            class="px-2.5 py-1.5 text-xs rounded-lg border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 cursor-pointer focus:outline-none focus:ring-1 focus:ring-violet-500"
          >
            <option value="all">
              Semua Keparahan
            </option>
            <option value="critical">
              🚨 Kritis (Critical)
            </option>
            <option value="high">
              ⚠️ Tinggi (High)
            </option>
            <option value="warning">
              ⚡ Peringatan (Warning)
            </option>
            <option value="info">
              ℹ️ Info
            </option>
          </select>
        </div>
        <div v-else class="flex items-center gap-2 shrink-0">
          <span class="px-2.5 py-1 rounded-md text-xs font-medium bg-violet-100 dark:bg-violet-950/50 text-violet-700 dark:text-violet-300 border border-violet-200 dark:border-violet-800 flex items-center gap-1.5">
            <UIcon name="i-lucide-gauge" class="w-3.5 h-3.5" />
            <span>High-Precision Engine</span>
          </span>
        </div>
      </div>

      <!-- Categories Tabs (Horizontal Scrollable) -->
      <div class="px-4 py-2.5 border-b border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 flex gap-1.5 overflow-x-auto scrollbar-thin shrink-0 text-xs">
        <button
          v-for="cat in categoriesWithCounts"
          :key="cat.id"
          type="button"
          class="px-3 py-1.5 rounded-lg whitespace-nowrap transition-all flex items-center gap-1.5 font-medium shrink-0 cursor-pointer text-xs"
          :class="activeTab === cat.id
            ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900 shadow-xs font-semibold'
            : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
          @click="activeTab = cat.id"
        >
          <span>{{ cat.name }}</span>
          <span
            class="text-xs font-mono px-1.5 py-0.5 rounded-full"
            :class="activeTab === cat.id
              ? 'bg-neutral-700 text-white dark:bg-neutral-200 dark:text-neutral-900'
              : 'bg-neutral-200/80 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-300'"
          >
            {{ cat.count }}
          </span>
        </button>
      </div>

      <!-- Criteria List Body -->
      <div class="flex-1 overflow-y-auto p-5 space-y-4">
        <!-- Results count indicator -->
        <div class="flex items-center justify-between text-xs text-neutral-500 dark:text-neutral-400 pb-1">
          <span v-if="activeTab === 'mechanism'">
            Arsitektur Mesin Diagnostik Vibeasik &middot; <strong>4 Layer Pemeriksaan Terarah</strong>
          </span>
          <span v-else>
            Menampilkan <strong>{{ filteredCriteria.length }}</strong> dari total {{ defaultCriteria.length }} kriteria
          </span>
          <span
            v-if="searchQuery || activeSeverity !== 'all'"
            role="button"
            tabindex="0"
            class="text-violet-600 dark:text-violet-400 font-medium cursor-pointer hover:underline"
            @click="searchQuery = ''; activeSeverity = 'all'; activeTab = 'all'"
            @keydown.enter="searchQuery = ''; activeSeverity = 'all'; activeTab = 'all'"
            @keydown.space.prevent="searchQuery = ''; activeSeverity = 'all'; activeTab = 'all'"
          >
            Reset Filter
          </span>
        </div>

        <!-- Mechanism View -->
        <DiagnosticsAuditMechanismView v-if="activeTab === 'mechanism'" />

        <!-- Criteria List (Regular Tabs) -->
        <div v-else class="space-y-4">
          <!-- Empty State -->
          <div
            v-if="filteredCriteria.length === 0"
            class="p-10 text-center rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 bg-neutral-50/50 dark:bg-neutral-800/20 space-y-2"
          >
            <UIcon name="i-lucide-filter-x" class="w-8 h-8 text-neutral-400 mx-auto" />
            <h4 class="text-sm font-bold text-neutral-800 dark:text-neutral-200">
              Tidak ada kriteria yang cocok
            </h4>
            <p class="text-xs text-neutral-500 max-w-sm mx-auto">
              Coba ubah kata kunci pencarian atau ganti filter keparahan untuk melihat kriteria lainnya.
            </p>
            <button
              type="button"
              class="mt-2 text-xs font-semibold text-violet-600 dark:text-violet-400 hover:underline"
              @click="searchQuery = ''; activeSeverity = 'all'; activeTab = 'all'"
            >
              Tampilkan Semua Kriteria
            </button>
          </div>

          <!-- Criteria Cards -->
          <DiagnosticsAuditCriteriaCard v-for="item in filteredCriteria" :key="item.id" :item="item" />
        </div>
      </div>

      <!-- Drawer Footer -->
      <div class="p-4 border-t border-neutral-200 dark:border-neutral-800 bg-neutral-50/80 dark:bg-neutral-900/80 flex items-center justify-between text-xs shrink-0">
        <span class="text-neutral-500 dark:text-neutral-400">
          Standar disinkronkan dengan <code class="font-mono bg-neutral-200/60 dark:bg-neutral-800 px-1 py-0.5 rounded text-xs">vibeasik/guidelines/*.json</code>
        </span>
        <UButton
          size="xs"
          color="neutral"
          variant="solid"
          @click="emit('update:open', false)"
        >
          Tutup
        </UButton>
      </div>
    </div>
  </div>
</template>
