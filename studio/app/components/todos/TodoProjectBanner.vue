<script setup lang="ts">
import type { Project } from '~/types'

defineProps<{
  activeProject: Project | null
  activeProjectId: number | null
  projects: Project[]
  remainingTasksCount: number
}>()
const emit = defineEmits<{ switchProject: [id: string] }>()
const handleSwitchProject = (id: string) => emit('switchProject', id)
</script>

<template>
  <div class="p-3.5 rounded-xl border border-blue-200 dark:border-blue-900/60 bg-blue-50/60 dark:bg-blue-950/20 text-xs flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-2xs">
    <div class="flex items-center gap-2.5 min-w-0">
      <UIcon name="i-lucide-folder-git-2" class="w-5 h-5 text-blue-600 dark:text-blue-400 shrink-0" />
      <div class="min-w-0">
        <div class="flex items-center gap-2 flex-wrap">
          <span class="text-neutral-500 dark:text-neutral-400 font-medium">Proyek Aktif:</span>
          <span class="font-bold text-sm text-blue-950 dark:text-blue-100 truncate">
            {{ activeProject ? activeProject.name : 'Belum Ada Proyek Terpilih' }}
          </span>
          <span
            v-if="activeProject"
            class="px-2 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900/50 text-blue-800 dark:text-blue-300 text-xs font-semibold border border-blue-200 dark:border-blue-800"
          >
            {{ activeProject.components?.length || 0 }} Komponen
          </span>
          <span
            v-if="remainingTasksCount > 0"
            class="px-2 py-0.5 rounded-full bg-amber-100 dark:bg-amber-900/50 text-amber-800 dark:text-amber-300 text-xs font-semibold border border-amber-200 dark:border-amber-800"
          >
            {{ remainingTasksCount }} tugas perlu diselesaikan
          </span>
        </div>
        <p v-if="activeProject && activeProject.components && activeProject.components.length > 0" class="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5 truncate">
          Aplikasi: {{ activeProject.components.map(c => c.name).join(', ') }}
        </p>
      </div>
    </div>

    <!-- Quick Switch Active Project on this page -->
    <div class="flex items-center gap-2 shrink-0">
      <label for="project-switcher" class="text-xs text-neutral-500 dark:text-neutral-400 font-medium hidden sm:inline">
        Ganti:
      </label>
      <select
        id="project-switcher"
        :value="activeProjectId ?? ''"
        class="h-8 pl-2.5 pr-7 py-0 text-xs font-mono font-medium rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 hover:border-neutral-300 dark:hover:border-neutral-600 focus:outline-none focus:ring-1 focus:ring-blue-500 cursor-pointer max-w-[200px] truncate"
        title="Pilih proyek aktif untuk perencanaan agenda"
        @change="(e) => handleSwitchProject((e.target as HTMLSelectElement).value)"
      >
        <option value="" disabled>
          -- Pilih Proyek Aktif --
        </option>
        <option v-for="p in projects" :key="p.id" :value="p.id">
          {{ p.name }} ({{ p.components?.length || 0 }} apps)
        </option>
      </select>
    </div>
  </div>
</template>
