<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { Project, ProjectProfile, ConsolidateResult } from '~/types'
import ConsolidateProjectModal from '~/components/projects/ConsolidateProjectModal.vue'
import LastScanPopover from './LastScanPopover.vue'

const props = defineProps<{
  project: Project
  profile?: ProjectProfile | null
}>()

const TYPE_LABEL: Record<string, string> = { multi_fe: 'Multi-frontend', bundle: 'Backend + frontend', monorepo: 'Monorepo', single_app: 'Satu aplikasi' }
const STATUS: Record<string, { label: string, cls: string }> = {
  deprecated: { label: 'Usang', cls: 'bg-amber-500/10 text-amber-700 dark:text-amber-400' },
  hold: { label: 'Ditahan', cls: 'bg-amber-500/10 text-amber-700 dark:text-amber-400' },
  done: { label: 'Selesai', cls: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400' }
}
const MATURITY: Record<string, string> = { idea: 'Ide', prototype: 'Prototipe', mvp: 'MVP', beta: 'Beta', production: 'Produksi' }
const maturity = computed(() => {
  const st = props.profile?.maturity?.stage
  return st ? (MATURITY[st] || st) + (props.profile?.source === 'heuristic' ? ' (perkiraan)' : '') : ''
})
const score = computed(() => props.project.health_score ?? null)

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
      icon: 'i-lucide-plug-zap',
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
  <div class="space-y-4 pb-5">
    <NuxtLink
      to="/cockpit"
      class="inline-flex items-center gap-1.5 text-xs text-neutral-500 hover:text-neutral-900 dark:hover:text-white transition-colors"
    >
      <UIcon name="i-lucide-arrow-left" class="size-3.5" />
      Katalog proyek
    </NuxtLink>

    <div class="flex flex-col lg:flex-row lg:items-start justify-between gap-5">
      <div class="min-w-0 space-y-2.5">
        <div class="flex flex-wrap items-center gap-2">
          <h2 class="text-2xl sm:text-[28px] font-semibold tracking-tight text-neutral-950 dark:text-white">
            {{ project.name }}
          </h2>
          <span v-if="STATUS[project.status || '']" class="rounded-md px-2 py-0.5 text-xs font-medium" :class="STATUS[project.status || '']!.cls">
            {{ STATUS[project.status || '']!.label }}
          </span>
        </div>

        <div class="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs text-neutral-500 dark:text-slate-400">
          <span class="inline-flex items-center gap-1.5">
            <UIcon name="i-lucide-boxes" class="size-3.5" />
            {{ TYPE_LABEL[project.project_type] || project.project_type }} · {{ project.components?.length || 0 }} aplikasi
          </span>
          <span v-if="project.git_branch" class="inline-flex items-center gap-1.5">
            <UIcon name="i-lucide-git-branch" class="size-3.5" />{{ project.git_branch }}
          </span>
          <button
            v-if="project.root_path && project.root_path !== '-'"
            type="button"
            class="inline-flex items-center gap-1.5 font-mono min-w-0 max-w-full hover:text-neutral-800 dark:hover:text-slate-200"
            :title="`${project.root_path} — klik untuk menyalin`"
            @click="copyToClipboard(project.root_path)"
          >
            <UIcon name="i-lucide-folder" class="size-3.5 shrink-0" />
            <span class="truncate">{{ project.root_path }}</span>
          </button>
          <span
            v-if="!isProjectUnified(project)"
            class="inline-flex items-center gap-1.5 text-amber-600 dark:text-amber-400"
            title="Aplikasi proyek ini tersebar di beberapa folder — satukan lewat menu aksi"
          >
            <UIcon name="i-lucide-folders" class="size-3.5" />Folder terpencar
          </span>
          <span class="font-mono">#{{ project.id }}</span>
        </div>

        <div v-if="project.short_description" class="text-sm text-neutral-700 dark:text-slate-300 max-w-3xl">
          <MarkdownViewer :content="project.short_description" compact />
        </div>
      </div>

      <div class="flex flex-wrap lg:flex-col items-center lg:items-end gap-4 shrink-0">
        <div class="flex items-center gap-4 rounded-xl border border-neutral-200 dark:border-white/[0.08] px-4 py-3">
          <div class="text-right">
            <div class="text-3xl font-semibold tracking-tight tabular-nums leading-none" :class="healthTone(score)">
              {{ score ?? '—' }}
            </div>
            <div class="mt-1 text-xs text-neutral-500">
              skor kesehatan
            </div>
          </div>
          <div class="w-px self-stretch bg-neutral-200 dark:bg-white/[0.08]" />
          <div class="text-xs space-y-1">
            <div class="font-medium" :class="healthTone(score)">
              {{ healthLabel(score) }}
            </div>
            <div class="text-neutral-500">
              Tahap: <span class="text-neutral-800 dark:text-slate-200">{{ maturity || 'belum dianalisis' }}</span>
            </div>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <UButton to="/audit-report" size="sm" color="neutral" variant="outline" icon="i-lucide-shield-check">
            Temuan audit
          </UButton>
          <LastScanPopover
            :project="project"
            :is-project-level="true"
            align="end"
            side="bottom"
          >
            <AppActionButton
              kind="llm"
              label="Scan & profil"
              desc="Fast Scan tiap aplikasi, deteksi halaman & rute, lalu susun profil proyek"
              @click="$emit('startPipeline')"
            />
          </LastScanPopover>
          <UDropdownMenu :items="moreItems" :content="{ align: 'end' }">
            <UButton
              icon="i-lucide-ellipsis"
              color="neutral"
              variant="ghost"
              size="sm"
              :loading="initingProject"
              aria-label="Aksi lain"
            />
          </UDropdownMenu>
        </div>
      </div>
    </div>

    <ConsolidateProjectModal
      v-model:open="isConsolidateModalOpen"
      :project="project"
      @consolidated="handleConsolidated"
    />
  </div>
</template>
