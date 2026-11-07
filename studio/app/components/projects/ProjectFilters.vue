<script setup lang="ts">
const searchQuery = defineModel<string>('searchQuery', { default: '' })
const selectedType = defineModel<string>('selectedType', { default: 'all' })
const selectedStatus = defineModel<string>('selectedStatus', { default: 'active' })

const typeOptions = [
  { id: 'all', label: 'All' },
  { id: 'multi_fe', label: 'Multi-FE' },
  { id: 'bundle', label: 'Bundle' },
  { id: 'monorepo', label: 'Monorepo' }
]

const statusOptions = [
  { id: 'active', label: 'Aktif' },
  { id: 'hold', label: 'Hold' },
  { id: 'done', label: 'Done' },
  { id: 'deprecated', label: 'Deprecated' },
  { id: 'all', label: 'Semua' }
]
</script>

<template>
  <div class="flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3">
    <!-- Search Input -->
    <div class="relative flex-1 max-w-md">
      <UInput
        v-model="searchQuery"
        icon="i-lucide-search"
        placeholder="Filter by name, path, language (go, nuxt, etc.)..."
        size="md"
        class="w-full"
      />
    </div>

    <div class="flex flex-wrap items-center gap-2 self-start md:self-auto">
      <!-- Status Filter Tabs -->
      <div class="flex items-center gap-1 p-1 bg-neutral-200/60 dark:bg-neutral-800/60 rounded-lg text-xs font-medium">
        <button
          v-for="s in statusOptions"
          :key="s.id"
          class="px-2.5 py-1.5 rounded-md transition-all cursor-pointer flex items-center gap-1"
          :class="selectedStatus === s.id
            ? 'bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white shadow-xs font-semibold'
            : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
          @click="selectedStatus = s.id"
        >
          <span v-if="s.id === 'active'" class="w-1.5 h-1.5 rounded-full bg-emerald-500" />
          <span v-else-if="s.id === 'hold'" class="w-1.5 h-1.5 rounded-full bg-amber-500" />
          <span v-else-if="s.id === 'done'" class="w-1.5 h-1.5 rounded-full bg-sky-500" />
          <span v-else-if="s.id === 'deprecated'" class="w-1.5 h-1.5 rounded-full bg-neutral-400" />
          <span>{{ s.label }}</span>
        </button>
      </div>

      <!-- Type Filter Tabs -->
      <div class="flex items-center gap-1 p-1 bg-neutral-200/60 dark:bg-neutral-800/60 rounded-lg text-xs font-medium">
        <button
          v-for="t in typeOptions"
          :key="t.id"
          class="px-3 py-1.5 rounded-md transition-all cursor-pointer"
          :class="selectedType === t.id
            ? 'bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white shadow-xs font-semibold'
            : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
          @click="selectedType = t.id"
        >
          {{ t.label }}
        </button>
      </div>
    </div>
  </div>
</template>
