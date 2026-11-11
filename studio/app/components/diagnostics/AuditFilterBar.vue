<script setup lang="ts">
import AuditCustomSelect, { type SelectOption } from './AuditCustomSelect.vue'

const props = defineProps<{
  activeCategory: string
  activeSeverity: string
  activeStatus: string
  searchQuery: string
  totalFiltered: number
  statusCounts?: {
    active: number
    all: number
    resolved: number
    ignored: number
  }
  severityCounts?: Record<string, number>
  categoryCounts?: Record<string, number>
  groupSimilar?: boolean
  /** Jumlah kelompok temuan serupa yang bisa disatukan */
  mergeableGroups?: number
}>()

const emit = defineEmits<{
  (e: 'update:activeCategory', val: string): void
  (e: 'update:activeSeverity', val: string): void
  (e: 'update:activeStatus', val: string): void
  (e: 'update:searchQuery', val: string): void
  (e: 'toggleGroup'): void
}>()

const categories = [
  { id: 'all', label: 'Semua Kategori', icon: 'i-lucide-layers' },
  { id: 'deep', label: 'Deep Check', icon: 'i-lucide-scan-search' },
  { id: 'llm', label: 'Review LLM', icon: 'i-lucide-sparkles' },
  { id: 'runtime', label: 'Runtime', icon: 'i-lucide-play-circle' },
  { id: 'query', label: 'Database & Query', icon: 'i-lucide-database' },
  { id: 'router', label: 'Router & API', icon: 'i-lucide-route' },
  { id: 'frontend', label: 'Frontend & Tampilan', icon: 'i-lucide-layout-template' },
  { id: 'mobile', label: 'Android & Mobile', icon: 'i-lucide-smartphone' },
  { id: 'code_smell', label: 'Code Smells', icon: 'i-lucide-flame' },
  { id: 'security', label: 'Keamanan', icon: 'i-lucide-shield-alert' },
  { id: 'deployment', label: 'Deploy & Env', icon: 'i-lucide-hard-drive-upload' }
]

// Kategori utama tampil sebagai tab; sisanya di menu "Lainnya" agar baris filter ringkas.
const PRIMARY_CATEGORIES = ['all', 'deep', 'llm', 'code_smell', 'security']
const primaryCategories = categories.filter(c => PRIMARY_CATEGORIES.includes(c.id))
const extraCategories = categories.filter(c => !PRIMARY_CATEGORIES.includes(c.id))
const activeExtra = computed(() => extraCategories.find(c => c.id === props.activeCategory))

const categoryCount = (id: string) => {
  const c = props.categoryCounts || {}
  if (id === 'code_smell') return (c.code_smell || 0) + (c.architecture || 0)
  if (id === 'security') return (c.security || 0) + (c.port || 0)
  return c[id] || 0
}

const extraMenuItems = computed(() => [extraCategories.map(cat => ({
  label: `${cat.label} (${categoryCount(cat.id)})`,
  icon: cat.icon,
  onSelect: () => emit('update:activeCategory', cat.id)
}))])

const severities = computed<SelectOption[]>(() => [
  {
    value: 'all',
    label: 'Semua Keparahan',
    icon: 'i-lucide-sliders-horizontal'
  },
  {
    value: 'CRITICAL',
    label: props.severityCounts ? `Kritis (Critical) (${props.severityCounts.CRITICAL || 0})` : 'Kritis (Critical)',
    icon: 'i-lucide-alert-triangle',
    color: 'text-rose-500'
  },
  {
    value: 'WARNING',
    label: props.severityCounts ? `Peringatan (Warning) (${props.severityCounts.WARNING || 0})` : 'Peringatan (Warning)',
    icon: 'i-lucide-alert-circle',
    color: 'text-amber-500'
  },
  {
    value: 'INFO',
    label: props.severityCounts ? `Informasi (Info) (${props.severityCounts.INFO || 0})` : 'Informasi (Info)',
    icon: 'i-lucide-info',
    color: 'text-blue-500'
  }
])

const statusOptions = computed<SelectOption[]>(() => [
  {
    value: 'active',
    label: props.statusCounts ? `Aktif (${props.statusCounts.active})` : 'Aktif Saja',
    icon: 'i-lucide-activity'
  },
  {
    value: 'all',
    label: props.statusCounts ? `Semua Status (${props.statusCounts.all})` : 'Semua Status',
    icon: 'i-lucide-list'
  },
  {
    value: 'resolved',
    label: props.statusCounts ? `Selesai (${props.statusCounts.resolved})` : 'Sudah Selesai',
    icon: 'i-lucide-check-circle-2',
    color: 'text-sky-500'
  },
  {
    value: 'ignored',
    label: props.statusCounts ? `Diabaikan (${props.statusCounts.ignored})` : 'Diabaikan',
    icon: 'i-lucide-eye-off',
    color: 'text-neutral-400'
  }
])
const handleStatusChange = (val: string | number) => {
  emit('update:activeStatus', String(val))
}

const handleSeverityChange = (val: string | number) => {
  emit('update:activeSeverity', String(val))
}
</script>

<template>
  <div class="flex flex-col 2xl:flex-row 2xl:items-center justify-between gap-3 pb-3 border-b border-neutral-200 dark:border-white/[0.08]">
    <!-- Category Tabs -->
    <div class="flex items-center gap-1.5 overflow-x-auto scrollbar-none py-1 shrink-0">
      <button
        v-for="cat in primaryCategories"
        :key="cat.id"
        type="button"
        class="px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors whitespace-nowrap cursor-pointer shrink-0"
        :class="activeCategory === cat.id
          ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900'
          : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
        @click="emit('update:activeCategory', cat.id)"
      >
        <UIcon :name="cat.icon" class="w-3.5 h-3.5" />
        {{ cat.label }}
        <span v-if="cat.id !== 'all' && categoryCounts" class="opacity-60 tabular-nums">{{ categoryCount(cat.id) }}</span>
      </button>
      <UDropdownMenu :items="extraMenuItems" :content="{ align: 'start' }">
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors whitespace-nowrap cursor-pointer shrink-0"
          :class="activeExtra
            ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900'
            : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
        >
          <UIcon :name="activeExtra?.icon || 'i-lucide-ellipsis'" class="w-3.5 h-3.5" />
          {{ activeExtra?.label || 'Lainnya' }}
          <span v-if="activeExtra && categoryCounts" class="opacity-60 tabular-nums">{{ categoryCount(activeExtra.id) }}</span>
          <UIcon name="i-lucide-chevron-down" class="w-3 h-3" />
        </button>
      </UDropdownMenu>
    </div>

    <!-- Filter Controls Row: Status, Severity, & Search Input sejajar di pojok kanan -->
    <div class="flex items-center gap-2 flex-wrap 2xl:justify-end 2xl:ml-auto">
      <!-- Satukan Serupa: temuan rule + ekosistem sama jadi satu temuan utama dengan sub temuan -->
      <UButton
        size="sm"
        :color="groupSimilar ? 'primary' : 'neutral'"
        :variant="groupSimilar ? 'solid' : 'outline'"
        :icon="groupSimilar ? 'i-lucide-ungroup' : 'i-lucide-group'"
        :disabled="!groupSimilar && !mergeableGroups"
        :title="groupSimilar ? 'Tampilkan semua temuan satu per satu' : `${mergeableGroups || 0} kelompok temuan serupa bisa disatukan`"
        @click="emit('toggleGroup')"
      >
        {{ groupSimilar ? 'Pisahkan' : 'Satukan Serupa' }}
        <UBadge v-if="!groupSimilar && mergeableGroups" size="xs" color="primary" variant="soft">
          {{ mergeableGroups }}
        </UBadge>
      </UButton>
      <!-- Status Filter Custom Select (Lebar atas compact, opsi dropdown terlihat lengkap) -->
      <AuditCustomSelect
        :model-value="activeStatus"
        :options="statusOptions"
        button-class="w-36 sm:w-40"
        placeholder="Status Temuan"
        @update:model-value="handleStatusChange"
      />

      <!-- Severity Filter Custom Select (Lebar atas compact, opsi dropdown terlihat lengkap) -->
      <AuditCustomSelect
        :model-value="activeSeverity"
        :options="severities"
        button-class="w-36 sm:w-40"
        placeholder="Filter Keparahan"
        @update:model-value="handleSeverityChange"
      />

      <!-- Search Input: Sejajar berdampingan -->
      <div class="w-48 sm:w-64 shrink-0">
        <UInput
          :model-value="searchQuery"
          icon="i-lucide-search"
          placeholder="Cari file, rule, judul..."
          size="sm"
          class="w-full"
          @update:model-value="val => emit('update:searchQuery', String(val ?? ''))"
        />
      </div>
    </div>
  </div>
</template>
