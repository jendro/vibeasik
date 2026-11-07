<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { Project, ProjectComponent, ProjectProfile, ProfileApp } from '~/types'

const props = defineProps<{
  project: Project
  profile: ProjectProfile | null
  portStatuses: Record<string, boolean>
  activePortsCount: number
  totalConfiguredPorts: number
  scanningCompIds: number[]
  creatingDirId: number | null
}>()

const emit = defineEmits<{
  openDetail: [comp: ProjectComponent]
  fastScan: [comp: ProjectComponent]
  openExistingScan: [comp: ProjectComponent]
  createDir: [comp: ProjectComponent]
  edit: [comp: ProjectComponent]
  delete: [comp: ProjectComponent]
  openAddModal: [role?: string]
  switchTab: [tab: string]
}>()

const appProfiles = computed(() => {
  const map: Record<number, ProfileApp> = {}
  for (const a of props.profile?.apps || []) map[a.component_id] = a
  return map
})

const addItems = computed<DropdownMenuItem[]>(() => [
  { label: 'Backend / API', icon: 'i-lucide-server', onSelect: () => emit('openAddModal', 'backend') },
  { label: 'Frontend Web', icon: 'i-lucide-globe', onSelect: () => emit('openAddModal', 'frontend_client') },
  { label: 'Admin Dashboard', icon: 'i-lucide-layout-dashboard', onSelect: () => emit('openAddModal', 'frontend_admin') },
  { label: 'Mobile App', icon: 'i-lucide-smartphone', onSelect: () => emit('openAddModal', 'frontend_mobile') },
  { label: 'Modul lain', icon: 'i-lucide-folder', onSelect: () => emit('openAddModal', 'other') }
])
</script>

<template>
  <div class="space-y-5">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div>
        <h2 class="text-base font-bold text-neutral-900 dark:text-white flex items-center gap-2">
          <span>Aplikasi dalam Proyek</span>
          <UBadge size="xs" color="neutral" variant="subtle">
            {{ project.components?.length || 0 }} aplikasi
          </UBadge>
        </h2>
        <p class="text-xs text-neutral-500">
          Backend, frontend, mobile, dan modul pendukung —
          <span :class="activePortsCount > 0 ? 'text-sky-600 dark:text-sky-400 font-semibold' : ''">{{ activePortsCount }}</span>/{{ totalConfiguredPorts }} port sedang aktif.
        </p>
      </div>
      <UDropdownMenu :items="addItems" :content="{ align: 'end' }">
        <UButton icon="i-lucide-plus" color="primary" variant="soft" size="xs" trailing-icon="i-lucide-chevron-down">
          Tambah Aplikasi
        </UButton>
      </UDropdownMenu>
    </div>

    <!-- Empty State -->
    <div
      v-if="!project.components || project.components.length === 0"
      class="p-12 text-center rounded-xl border border-dashed border-neutral-300 dark:border-neutral-800 bg-white/50 dark:bg-neutral-900/50 space-y-3"
    >
      <div class="w-10 h-10 mx-auto rounded-full bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center text-neutral-400">
        <UIcon name="i-lucide-layers" class="w-5 h-5" />
      </div>
      <div class="space-y-1">
        <h3 class="text-sm font-semibold text-neutral-900 dark:text-white">
          Belum ada aplikasi yang dipantau
        </h3>
        <p class="text-xs text-neutral-500 max-w-sm mx-auto">
          Tambahkan modul Backend, Frontend Web, atau Mobile untuk memantau rute dan port.
        </p>
      </div>
      <div class="flex flex-wrap items-center justify-center gap-2 pt-2">
        <UButton size="xs" color="primary" @click="$emit('openAddModal', 'backend')">
          + Tambah BE
        </UButton>
        <UButton size="xs" color="neutral" variant="outline" @click="$emit('openAddModal', 'frontend_client')">
          + Tambah FE
        </UButton>
        <UButton size="xs" color="neutral" variant="outline" @click="$emit('openAddModal', 'frontend_mobile')">
          + Tambah Mobile
        </UButton>
      </div>
    </div>

    <!-- Component Cards Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 2xl:grid-cols-3 gap-4">
      <ProjectsDetailComponentCard
        v-for="c in project.components"
        :key="c.id"
        :comp="c"
        :project-id="project.id"
        :app-profile="appProfiles[c.id]"
        :port-active="portStatuses[String(c.port)]"
        :scanning="scanningCompIds.includes(c.id)"
        :creating-dir="creatingDirId === c.id"
        @open-detail="$emit('openDetail', c)"
        @fast-scan="$emit('fastScan', c)"
        @open-existing-scan="$emit('openExistingScan', c)"
        @create-dir="$emit('createDir', c)"
        @edit="$emit('edit', c)"
        @delete="$emit('delete', c)"
      />
    </div>

    <ProjectsDetailProjectIntegrationsPanel
      v-if="profile && (project.components?.length || 0) > 0"
      :profile="profile"
      @switch-tab="$emit('switchTab', $event)"
    />
  </div>
</template>
