<script setup lang="ts">
defineProps<{
  filteredCount: number
  totalCount: number
}>()
const selectedLevel = defineModel<string>('level', { required: true })
const selectedCategory = defineModel<string>('category', { required: true })
const selectedCreator = defineModel<string>('creator', { required: true })
const selectedTag = defineModel<string>('tag', { required: true })
const selectedStatus = defineModel<string>('status', { required: true })
const selectedOrderBy = defineModel<TodoOrderBy>('orderBy', { required: true })
const searchQuery = defineModel<string>('search', { required: true })

const setLevel = (level: string) => {
  selectedLevel.value = level
}
const setCategory = (category: string) => {
  selectedCategory.value = category
}
const toggleUncompletedStatus = () => {
  selectedStatus.value = selectedStatus.value === 'uncompleted' ? 'all' : 'uncompleted'
}
</script>

<template>
  <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] space-y-3">
    <!-- Row 1: Level Tabs (Dikhususkan untuk Proyek Aktif) -->
    <div class="flex flex-wrap items-center justify-between gap-2 border-b border-neutral-100 dark:border-white/10 pb-3">
      <div class="flex flex-wrap gap-1 text-xs">
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg font-medium transition-colors cursor-pointer"
          :class="selectedLevel === 'all' ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
          @click="setLevel('all')"
        >
          Semua Tingkat Proyek
        </button>
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
          :class="selectedLevel === 'project' ? 'bg-blue-600 text-white' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
          @click="setLevel('project')"
        >
          <UIcon name="i-lucide-folder-kanban" class="w-3.5 h-3.5" />
          <span>Proyek (Bundle)</span>
        </button>
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
          :class="selectedLevel === 'component' ? 'bg-sky-600 text-white' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
          @click="setLevel('component')"
        >
          <UIcon name="i-lucide-layers" class="w-3.5 h-3.5" />
          <span>Komponen (Module)</span>
        </button>
      </div>

      <!-- Quick Toggle Sembunyikan Selesai & Counter -->
      <div class="flex items-center gap-3">
        <button
          type="button"
          class="px-2.5 py-1 rounded-lg text-xs font-medium transition-all flex items-center gap-1.5 border cursor-pointer"
          :class="selectedStatus === 'uncompleted'
            ? 'bg-amber-100 text-amber-900 border-amber-300 dark:bg-amber-950/60 dark:text-amber-200 dark:border-amber-800 font-semibold'
            : 'border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800/80 text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-700'"
          title="Klik untuk menyembunyikan atau menampilkan tugas yang sudah selesai"
          @click="toggleUncompletedStatus"
        >
          <UIcon :name="selectedStatus === 'uncompleted' ? 'i-lucide-eye-off' : 'i-lucide-check-circle'" class="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" />
          <span>{{ selectedStatus === 'uncompleted' ? 'Selesai Tersembunyi' : 'Sembunyikan Selesai' }}</span>
        </button>

        <div class="text-xs font-mono text-neutral-400">
          Menampilkan {{ filteredCount }} dari {{ totalCount }} agenda
        </div>
      </div>
    </div>

    <!-- Row 2: Category Filter Pills -->
    <div class="flex flex-wrap items-center gap-1.5 text-xs pb-1">
      <span class="text-neutral-400 text-xs font-semibold mr-1">Kategori:</span>
      <button
        type="button"
        class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors cursor-pointer"
        :class="selectedCategory === 'all' ? 'bg-neutral-800 text-white dark:bg-neutral-100 dark:text-neutral-900' : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:bg-neutral-200 dark:hover:bg-neutral-700'"
        @click="setCategory('all')"
      >
        Semua Kategori
      </button>
      <button
        type="button"
        class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
        :class="selectedCategory === 'security' ? 'bg-rose-600 text-white' : 'bg-rose-50 dark:bg-rose-950/40 text-rose-700 dark:text-rose-300 hover:bg-rose-100 dark:hover:bg-rose-900/50 border border-rose-200 dark:border-rose-900/60'"
        @click="setCategory('security')"
      >
        <UIcon name="i-lucide-shield-check" class="w-3 h-3" />
        <span>Keamanan</span>
      </button>
      <button
        type="button"
        class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
        :class="selectedCategory === 'ui_ux' ? 'bg-fuchsia-600 text-white' : 'bg-fuchsia-50 dark:bg-fuchsia-950/40 text-fuchsia-700 dark:text-fuchsia-300 hover:bg-fuchsia-100 dark:hover:bg-fuchsia-900/50 border border-fuchsia-200 dark:border-fuchsia-900/60'"
        @click="setCategory('ui_ux')"
      >
        <UIcon name="i-lucide-palette" class="w-3 h-3" />
        <span>UI / UX</span>
      </button>
      <button
        type="button"
        class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
        :class="selectedCategory === 'feature' ? 'bg-amber-600 text-white' : 'bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300 hover:bg-amber-100 dark:hover:bg-amber-900/50 border border-amber-200 dark:border-amber-900/60'"
        @click="setCategory('feature')"
      >
        <UIcon name="i-lucide-sparkles" class="w-3 h-3" />
        <span>Fitur</span>
      </button>
      <button
        type="button"
        class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
        :class="selectedCategory === 'architecture' ? 'bg-blue-600 text-white' : 'bg-blue-50 dark:bg-blue-950/40 text-blue-700 dark:text-blue-300 hover:bg-blue-100 dark:hover:bg-blue-900/50 border border-blue-200 dark:border-blue-900/60'"
        @click="setCategory('architecture')"
      >
        <UIcon name="i-lucide-network" class="w-3 h-3" />
        <span>Arsitektur</span>
      </button>
      <button
        type="button"
        class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
        :class="selectedCategory === 'performance' ? 'bg-sky-600 text-white' : 'bg-sky-50 dark:bg-sky-950/40 text-sky-700 dark:text-sky-300 hover:bg-sky-100 dark:hover:bg-sky-900/50 border border-sky-200 dark:border-sky-900/60'"
        @click="setCategory('performance')"
      >
        <UIcon name="i-lucide-zap" class="w-3 h-3" />
        <span>Performa</span>
      </button>
      <button
        type="button"
        class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
        :class="selectedCategory === 'backend' ? 'bg-sky-600 text-white' : 'bg-sky-50 dark:bg-sky-950/40 text-sky-700 dark:text-sky-300 hover:bg-sky-100 dark:hover:bg-sky-900/50 border border-sky-200 dark:border-sky-900/60'"
        @click="setCategory('backend')"
      >
        <UIcon name="i-lucide-server" class="w-3 h-3" />
        <span>Backend</span>
      </button>
      <button
        type="button"
        class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
        :class="selectedCategory === 'devops' ? 'bg-violet-600 text-white' : 'bg-violet-50 dark:bg-violet-950/40 text-violet-700 dark:text-violet-300 hover:bg-violet-100 dark:hover:bg-violet-900/50 border border-violet-200 dark:border-violet-900/60'"
        @click="setCategory('devops')"
      >
        <UIcon name="i-lucide-git-branch" class="w-3 h-3" />
        <span>DevOps</span>
      </button>
    </div>

    <TodosTodoTagFilter v-model="selectedTag" />

    <!-- Row 3: Secondary Filters & Order By (Prioritas, dll.) -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2.5 text-xs">
      <!-- Search -->
      <div class="relative">
        <UIcon name="i-lucide-search" class="w-4 h-4 absolute left-2.5 top-2.5 text-neutral-400" />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Cari agenda, kode [SEC-001], dll..."
          class="w-full pl-8 pr-3 py-1.5 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800 text-neutral-900 dark:text-white"
        >
      </div>

      <!-- Order By Selector (Prioritas, Tanggal, dll.) -->
      <div class="relative flex items-center">
        <span class="absolute left-2.5 pointer-events-none text-neutral-400">
          <UIcon name="i-lucide-arrow-down-up" class="w-3.5 h-3.5 text-violet-500" />
        </span>
        <select
          v-model="selectedOrderBy"
          class="w-full pl-8 pr-7 py-1.5 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800 text-neutral-900 dark:text-white appearance-none cursor-pointer"
          title="Urutkan agenda berdasarkan"
        >
          <option value="priority_desc">
            ⚡ Urut: Prioritas (Tertinggi)
          </option>
          <option value="priority_asc">
            ⚡ Urut: Prioritas (Terendah)
          </option>
          <option value="impact_desc">
            🚀 Urut: Dampak (Terbesar)
          </option>
          <option value="value_desc">
            🎯 Urut: Dampak per Usaha
          </option>
          <option value="created_desc">
            🕒 Urut: Terbaru Dibuat
          </option>
          <option value="created_asc">
            🕒 Urut: Terlama Dibuat
          </option>
          <option value="title_asc">
            🔤 Urut: Judul (A - Z)
          </option>
          <option value="status">
            📊 Urut: Status Tugas
          </option>
        </select>
        <span class="absolute right-2 pointer-events-none text-neutral-400">
          <UIcon name="i-lucide-chevron-down" class="w-3.5 h-3.5" />
        </span>
      </div>

      <!-- Status Filter (Termasuk Opsi Belum Selesai) -->
      <div class="relative flex items-center">
        <span class="absolute left-2.5 pointer-events-none text-neutral-400">
          <UIcon name="i-lucide-filter" class="w-3.5 h-3.5 text-amber-500" />
        </span>
        <select
          v-model="selectedStatus"
          class="w-full pl-8 pr-7 py-1.5 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800 text-neutral-900 dark:text-white appearance-none cursor-pointer"
          title="Filter berdasarkan status pengerjaan"
        >
          <option value="uncompleted">
            ⏳ Belum Selesai (Sembunyikan Selesai)
          </option>
          <option value="all">
            Semua Status (Termasuk Selesai)
          </option>
          <option value="in_progress">
            Sedang Dikerjakan
          </option>
          <option value="pending">
            Menunggu (Pending)
          </option>
          <option value="suggested">
            Saran AI (Menunggu Approval)
          </option>
          <option value="completed">
            Selesai
          </option>
          <option value="rejected">
            Ditolak
          </option>
        </select>
        <span class="absolute right-2 pointer-events-none text-neutral-400">
          <UIcon name="i-lucide-chevron-down" class="w-3.5 h-3.5" />
        </span>
      </div>

      <!-- Origin / Asal Todo Filter -->
      <div class="relative flex items-center">
        <span class="absolute left-2.5 pointer-events-none text-neutral-400">
          <UIcon name="i-lucide-layers" class="w-3.5 h-3.5 text-blue-500" />
        </span>
        <select
          v-model="selectedCreator"
          class="w-full pl-8 pr-7 py-1.5 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800 text-neutral-900 dark:text-white appearance-none cursor-pointer"
        >
          <option value="all">
            🌐 Semua Asal Todo
          </option>
          <option value="executive">
            🏛️ Dewan Eksekutif
          </option>
          <option value="audit">
            🔍 Optimasi Audit
          </option>
          <option value="pm">
            📋 Project Manager
          </option>
          <option value="ai_suggestion">
            💡 Rekomendasi AI
          </option>
          <option value="agent">
            💻 Agent Coding
          </option>
          <option value="human">
            👤 Developer (Manual)
          </option>
        </select>
        <span class="absolute right-2 pointer-events-none text-neutral-400">
          <UIcon name="i-lucide-chevron-down" class="w-3.5 h-3.5" />
        </span>
      </div>
    </div>
  </div>
</template>
