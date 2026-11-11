<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { Project, ConsolidateResult } from '~/types'
import ConsolidateProjectModal from '~/components/projects/ConsolidateProjectModal.vue'
import LastScanPopover from './LastScanPopover.vue'

const props = defineProps<{
  project: Project
}>()

const emit = defineEmits<{
  deleteProject: []
  startPipeline: []
  consolidated: [result: ConsolidateResult]
}>()

const isConsolidateModalOpen = ref(false)

const handleConsolidated = (res: ConsolidateResult) => {
  if (props.project) {
    props.project.root_path = res.project_root
    props.project.is_unified = res.is_unified
  }
  emit('consolidated', res)
}

const { copyToClipboard: copyText } = useCopyToClipboard()
const copyToClipboard = (text: string) => copyText(text, 'Root path proyek berhasil disalin.')

const { initing: initingProject, initProject } = useInitVibeasik()

const moreItems = computed<DropdownMenuItem[][]>(() => [
  [
    {
      label: 'Init vibeasik (tautan AI + peta kode)',
      icon: 'i-lucide-sparkles',
      onSelect: () => initProject(props.project.id, props.project.name)
    },
    ...(!isProjectUnified(props.project)
      ? [{
          label: 'Satukan ke 1 folder induk',
          icon: 'i-lucide-folder-input',
          onSelect: () => { isConsolidateModalOpen.value = true }
        }]
      : []),
    {
      label: 'Salin root path',
      icon: 'i-lucide-copy',
      disabled: !props.project.root_path || props.project.root_path === '-',
      onSelect: () => copyToClipboard(props.project.root_path)
    }
  ],
  [
    {
      label: 'Hapus dari pemantauan',
      icon: 'i-lucide-trash-2',
      color: 'error',
      onSelect: () => emit('deleteProject')
    }
  ]
])
</script>

<template>
  <div class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 p-4 sm:p-5 shadow-xs space-y-3">
    <div class="flex items-center justify-between gap-3 text-xs">
      <NuxtLink
        to="/cockpit"
        class="inline-flex items-center gap-1.5 text-neutral-500 hover:text-neutral-900 dark:hover:text-white transition-colors font-medium"
      >
        <UIcon name="i-lucide-arrow-left" class="w-3.5 h-3.5" />
        <span>Semua Proyek</span>
      </NuxtLink>

      <div class="flex items-center gap-2">
        <LastScanPopover
          :project="project"
          :is-project-level="true"
          align="end"
          side="bottom"
        >
          <UButton
            icon="i-lucide-play"
            color="primary"
            variant="soft"
            size="xs"
            title="Jalankan pipeline LLM: Fast Scan tiap aplikasi, deteksi halaman & rute, lalu susun profil proyek"
            @click="$emit('startPipeline')"
          >
            Jalankan Scan LLM
          </UButton>
        </LastScanPopover>
        <UDropdownMenu :items="moreItems" :content="{ align: 'end' }">
          <UButton
            icon="i-lucide-ellipsis"
            color="neutral"
            variant="outline"
            size="xs"
            :loading="initingProject"
            aria-label="Aksi lain"
          />
        </UDropdownMenu>
      </div>
    </div>

    <div class="space-y-2">
      <div class="flex flex-wrap items-center gap-2 sm:gap-2.5">
        <h1 class="text-xl sm:text-2xl font-bold tracking-tight text-neutral-900 dark:text-white">
          {{ project.name }}
        </h1>
        <UBadge
          size="sm"
          :color="project.project_type === 'multi_fe' ? 'secondary' : project.project_type === 'bundle' ? 'info' : 'warning'"
          variant="subtle"
        >
          {{ project.project_type === 'multi_fe' ? 'Multi-Frontend System' : project.project_type === 'bundle' ? 'Backend + Frontend Bundle' : project.project_type }}
        </UBadge>
        <UBadge v-if="project.status === 'deprecated'" size="sm" color="warning" variant="solid">
          DEPRECATED
        </UBadge>
        <UBadge v-else-if="project.status === 'hold'" size="sm" color="warning" variant="subtle">
          ON HOLD
        </UBadge>
        <UBadge v-else-if="project.status === 'done'" size="sm" color="success" variant="subtle">
          DONE (SELESAI)
        </UBadge>
        <UBadge v-if="project.git_branch" size="sm" color="success" variant="outline">
          <UIcon name="i-lucide-git-branch" class="w-3.5 h-3.5 mr-1" />
          {{ project.git_branch }}
        </UBadge>
        <UBadge
          v-if="!isProjectUnified(project)"
          size="sm"
          color="warning"
          variant="subtle"
          title="Komponen proyek masih terpencar di folder terpisah — satukan lewat menu aksi"
        >
          <UIcon name="i-lucide-folder-git-2" class="w-3.5 h-3.5 mr-1" />
          Folder Terpencar
        </UBadge>
      </div>

      <div v-if="project.root_path && project.root_path !== '-'" class="flex items-center gap-1.5 text-xs text-neutral-500 font-mono min-w-0">
        <UIcon name="i-lucide-folder" class="w-3.5 h-3.5 text-neutral-400 shrink-0" />
        <span class="truncate" :title="project.root_path">{{ project.root_path }}</span>
      </div>

      <div v-if="project.short_description" class="text-sm text-neutral-700 dark:text-neutral-300 max-w-4xl">
        <MarkdownViewer :content="project.short_description" compact />
      </div>
    </div>

    <ConsolidateProjectModal
      v-model:open="isConsolidateModalOpen"
      :project="project"
      @consolidated="handleConsolidated"
    />
  </div>
</template>
