<script setup lang="ts">
defineProps<{
  matrixCount: number
  pageCount: number
  availableTables: string[]
  areAllPagesCollapsed: boolean
}>()
const activeView = defineModel<'pipeline' | 'matrix' | 'topology'>('activeView', { required: true })
const searchQuery = defineModel<string>('searchQuery', { required: true })
const filterMethod = defineModel<string>('filterMethod', { required: true })
const filterTable = defineModel<string>('filterTable', { required: true })
const emit = defineEmits<{ collapseAll: [], expandAll: [], reset: [] }>()
const collapseAllPages = () => emit('collapseAll')
const expandAllPages = () => emit('expandAll')
const resetFilters = () => emit('reset')
</script>

<template>
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-1">
    <!-- 3 Pilihan Tampilan Utama: Pipeline Visual, Matriks Audit, Peta Dependensi -->
    <div class="flex items-center gap-1.5 p-1 rounded-xl bg-neutral-100 dark:bg-neutral-800/60 border border-neutral-200/60 dark:border-neutral-700/60">
      <button
        type="button"
        class="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
        :class="activeView === 'pipeline'
          ? 'bg-white dark:bg-[#04111f] text-sky-600 dark:text-sky-400 font-bold'
          : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
        @click="activeView = 'pipeline'"
      >
        <UIcon name="i-lucide-git-fork" class="w-4 h-4" />
        <span>Pipeline Visual</span>
      </button>

      <button
        type="button"
        class="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
        :class="activeView === 'matrix'
          ? 'bg-white dark:bg-[#04111f] text-sky-600 dark:text-sky-400 font-bold'
          : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
        @click="activeView = 'matrix'"
      >
        <UIcon name="i-lucide-table-properties" class="w-4 h-4" />
        <span>Matriks Audit Tabular</span>
        <UBadge size="xs" color="neutral" variant="subtle" class="font-mono text-xs px-1 py-0">
          {{ matrixCount }}
        </UBadge>
      </button>

      <button
        type="button"
        class="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
        :class="activeView === 'topology'
          ? 'bg-white dark:bg-[#04111f] text-sky-600 dark:text-sky-400 font-bold'
          : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
        @click="activeView = 'topology'"
      >
        <UIcon name="i-lucide-database" class="w-4 h-4" />
        <span>Relasi DB & API</span>
      </button>
    </div>

    <!-- Quick Summary Status -->
    <div class="text-xs text-neutral-400 flex items-center gap-2">
      <span>Menampilkan {{ pageCount }} halaman</span>
      <span v-if="searchQuery || filterMethod !== 'all' || filterTable !== 'all'" class="text-sky-500 font-semibold">
        (Difilter)
      </span>
    </div>
  </div>

  <!-- Filter & Search Bar -->
  <div class="flex flex-col md:flex-row md:items-center justify-between gap-3 p-3.5 rounded-xl border border-neutral-200/80 dark:border-white/[0.08] bg-white dark:bg-[#04111f]">
    <div class="flex flex-wrap items-center gap-2 flex-1">
      <!-- Search Input -->
      <UInput
        v-model="searchQuery"
        icon="i-lucide-search"
        placeholder="Cari halaman, rute API, handler, tabel DB..."
        size="xs"
        class="w-full sm:w-64"
      />

      <!-- Method Selector -->
      <select
        v-model="filterMethod"
        aria-label="Filter Metode HTTP"
        class="text-xs rounded-lg px-2.5 py-1.5 border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-slate-800 text-neutral-700 dark:text-neutral-200 focus:outline-hidden cursor-pointer"
      >
        <option value="all">
          Semua Metode HTTP
        </option>
        <option value="GET">
          GET
        </option>
        <option value="POST">
          POST
        </option>
        <option value="PUT">
          PUT / PATCH
        </option>
        <option value="DELETE">
          DELETE
        </option>
      </select>

      <!-- Table Selector -->
      <select
        v-if="availableTables.length > 0"
        v-model="filterTable"
        aria-label="Filter Tabel Database"
        class="text-xs rounded-lg px-2.5 py-1.5 border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-slate-800 text-neutral-700 dark:text-neutral-200 focus:outline-hidden cursor-pointer"
      >
        <option value="all">
          Semua Tabel Database
        </option>
        <option v-for="t in availableTables" :key="t" :value="t">
          Tabel: {{ t }}
        </option>
      </select>

      <!-- Reset Button -->
      <UButton
        v-if="searchQuery || filterMethod !== 'all' || filterTable !== 'all'"
        size="xs"
        color="neutral"
        variant="ghost"
        icon="i-lucide-x"
        title="Reset filter pencarian"
        class="cursor-pointer"
        @click="resetFilters"
      >
        Reset
      </UButton>
    </div>

    <!-- Expand / Collapse Controls (Only relevant in Pipeline View) -->
    <div v-if="activeView === 'pipeline'" class="flex items-center gap-2 shrink-0">
      <UButton
        size="xs"
        color="neutral"
        variant="outline"
        :icon="areAllPagesCollapsed ? 'i-lucide-chevrons-down' : 'i-lucide-chevrons-up'"
        title="Ciutkan atau bentangkan seluruh modul halaman"
        class="cursor-pointer"
        @click="areAllPagesCollapsed ? expandAllPages() : collapseAllPages()"
      >
        {{ areAllPagesCollapsed ? 'Bentangkan Semua' : 'Ciutkan Semua' }}
      </UButton>
    </div>
  </div>
</template>
