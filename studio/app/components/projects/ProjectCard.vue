<script setup lang="ts">
import type { Project, ConsolidateResult } from '~/types'
import ConsolidateProjectModal from '~/components/projects/ConsolidateProjectModal.vue'

const props = defineProps<{
  project: Project
  diagnosing?: boolean
  diagnosingStatus?: string
  currentlyCheckingCompId?: number | null
  statusLoading?: boolean
}>()

const emit = defineEmits<{
  diagnose: [project: Project]
  delete: [id: number]
  deprecate: [project: Project]
  undeprecate: [project: Project]
  hold: [project: Project]
  done: [project: Project]
  activate: [project: Project]
  merge: [project: Project]
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

const { getRoleBadgeColor } = useFormatters()

const { getRoleLabel: roleLabel } = useFormatters()
const getRoleLabel = (role: string) => roleLabel(role, { mobile: 'Mobile App', other: 'Folder Lain' })

const getScoreColor = (score: number) => {
  if (score >= 90) return 'text-sky-600 dark:text-sky-400 border-sky-500/30 bg-sky-50 dark:bg-sky-950/30'
  if (score >= 70) return 'text-amber-600 dark:text-amber-400 border-amber-500/30 bg-amber-50 dark:bg-amber-950/30'
  return 'text-rose-600 dark:text-rose-400 border-rose-500/30 bg-rose-50 dark:bg-rose-950/30'
}

const { getProject } = useApi()
const toast = useToast()

// Summary refresh runs on the engine's LLM queue; the card shows queue/progress and stays blocked meanwhile.
const summaryJob = useJobTarget('project_summary', () => ({ projectId: props.project?.id }), async () => {
  try {
    const fresh = await getProject(props.project.id)
    props.project.short_description = fresh.short_description
    props.project.description = fresh.description
    toast.add({
      title: 'Ringkasan Berhasil Diperbarui',
      description: `Ringkasan AI untuk proyek '${props.project.name}' berhasil diperbarui!`,
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
  } catch (err: any) {
    toast.add({
      title: 'Gagal Memuat Ringkasan Baru',
      description: `Ringkasan '${props.project.name}' selesai dibuat tetapi gagal dimuat: ${err.message || 'Terjadi kesalahan'}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  }
})
const refreshingSummary = summaryJob.busy

const handleCardRefreshSummary = () => {
  if (!props.project?.id) return
  summaryJob.start()
}
</script>

<template>
  <div
    class="group relative rounded-xl border p-5 shadow-xs transition-all flex flex-col justify-between"
    :class="project.status === 'deprecated'
      ? 'border-dashed border-amber-500/40 bg-neutral-50/50 dark:bg-neutral-900/30 opacity-75 hover:opacity-100'
      : project.status === 'hold'
        ? 'border-dashed border-amber-500/50 bg-amber-50/20 dark:bg-amber-950/10 hover:border-amber-500/60'
        : project.status === 'done'
          ? 'border-emerald-500/30 bg-emerald-50/20 dark:bg-emerald-950/10 hover:border-emerald-500/50'
          : 'border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 hover:border-sky-500/30 dark:hover:border-sky-500/40'"
  >
    <div>
      <!-- Header: Title, Health Score & Top Badges -->
      <div class="flex items-start justify-between gap-3">
        <div class="space-y-1">
          <div class="flex items-center gap-2 flex-wrap">
            <NuxtLink
              :to="`/projects/${project.id}`"
              class="font-semibold text-lg transition-colors"
              :class="project.status === 'deprecated'
                ? 'line-through text-neutral-500 dark:text-neutral-400'
                : project.status === 'hold'
                  ? 'italic text-neutral-600 dark:text-neutral-300'
                  : 'text-neutral-900 dark:text-white group-hover:text-sky-600 dark:group-hover:text-sky-400'"
            >
              {{ project.name }}
            </NuxtLink>
            <UBadge
              v-if="project.status === 'deprecated'"
              size="xs"
              color="warning"
              variant="solid"
            >
              DEPRECATED
            </UBadge>
            <UBadge
              v-else-if="project.status === 'hold'"
              size="xs"
              color="warning"
              variant="subtle"
            >
              ON HOLD
            </UBadge>
            <UBadge
              v-else-if="project.status === 'done'"
              size="xs"
              color="success"
              variant="subtle"
            >
              DONE (SELESAI)
            </UBadge>
            <UBadge
              size="xs"
              :color="project.project_type === 'multi_fe' ? 'secondary' : project.project_type === 'bundle' ? 'info' : 'warning'"
              variant="subtle"
            >
              {{ project.project_type === 'multi_fe' ? 'Multi-FE Bundle' : project.project_type === 'bundle' ? 'BE + FE Bundle' : project.project_type }}
            </UBadge>
            <UBadge
              v-if="project.is_unified"
              size="xs"
              color="success"
              variant="subtle"
              title="1 Folder Induk Terpadu"
            >
              <UIcon name="i-lucide-folder-check" class="w-3 h-3 mr-0.5" />
              1 Induk
            </UBadge>
            <UBadge
              v-else
              size="xs"
              color="warning"
              variant="subtle"
              title="Folder Terpencar"
            >
              <UIcon name="i-lucide-folder-git-2" class="w-3 h-3 mr-0.5" />
              Terpencar
            </UBadge>
          </div>

          <div class="flex items-center gap-2.5 text-xs text-neutral-500 font-mono">
            <span class="flex items-center gap-1.5">
              <UIcon name="i-lucide-layers" class="w-3.5 h-3.5 text-sky-500 shrink-0" />
              <span>{{ project.components?.length || 0 }} linked services</span>
            </span>
            <span v-if="project.git_branch" class="flex items-center gap-1 text-neutral-400">
              <UIcon name="i-lucide-git-branch" class="w-3 h-3 text-sky-500" />
              {{ project.git_branch }}
            </span>
          </div>
        </div>

        <!-- Health Score Pill -->
        <div
          class="px-2.5 py-1 rounded-full text-xs font-mono font-bold border shrink-0"
          :class="getScoreColor(project.health_score)"
          :title="`Health Score: ${project.health_score}/100`"
        >
          {{ project.health_score }}/100
        </div>
      </div>

      <!-- Deprecated Reason Banner -->
      <div
        v-if="project.status === 'deprecated'"
        class="mt-2.5 p-2 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-900 dark:text-amber-200 text-xs flex items-start gap-2"
      >
        <UIcon name="i-lucide-alert-circle" class="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
        <div class="space-y-0.5">
          <div class="font-semibold text-xs text-amber-800 dark:text-amber-300">
            Status Deprecated
          </div>
          <p class="text-xs leading-relaxed text-neutral-600 dark:text-neutral-400">
            {{ project.deprecated_reason || 'Proyek ditandai sebagai tidak aktif/deprecated.' }}
          </p>
        </div>
      </div>

      <!-- Hold Status Banner -->
      <div
        v-else-if="project.status === 'hold'"
        class="mt-2.5 p-2 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-900 dark:text-amber-200 text-xs flex items-start gap-2"
      >
        <UIcon name="i-lucide-pause-circle" class="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
        <div class="space-y-0.5">
          <div class="font-semibold text-xs text-amber-800 dark:text-amber-300">
            Status On Hold
          </div>
          <p class="text-xs leading-relaxed text-neutral-600 dark:text-neutral-400">
            Proyek sedang ditangguhkan sementara dan disembunyikan dari daftar proyek aktif.
          </p>
        </div>
      </div>

      <!-- Done Status Banner -->
      <div
        v-else-if="project.status === 'done'"
        class="mt-2.5 p-2 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-900 dark:text-emerald-200 text-xs flex items-start gap-2"
      >
        <UIcon name="i-lucide-check-circle-2" class="w-4 h-4 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5" />
        <div class="space-y-0.5">
          <div class="font-semibold text-xs text-emerald-800 dark:text-emerald-300">
            Status Selesai (Maintenance)
          </div>
          <p class="text-xs leading-relaxed text-neutral-600 dark:text-neutral-400">
            Proyek telah rampung dan dalam mode pemeliharaan (maintenance).
          </p>
        </div>
      </div>

      <!-- Keterangan Singkat (AI Generated / Formatted in Markdown) -->
      <div class="mt-3 pt-2.5 border-t border-neutral-100/80 dark:border-neutral-800/60">
        <div class="flex items-center justify-between text-xs font-medium text-neutral-500 mb-1.5">
          <span class="flex items-center gap-1.5">
            <UIcon name="i-lucide-file-text" class="w-3.5 h-3.5 text-sky-500" />
            <span class="font-semibold text-neutral-700 dark:text-neutral-300">Keterangan Singkat</span>
          </span>
          <div class="flex items-center gap-1.5">
            <span
              v-if="project.short_description"
              class="inline-flex items-center gap-1 text-xs font-mono text-sky-600 dark:text-sky-400 bg-sky-50 dark:bg-sky-950/40 px-1.5 py-0.5 rounded border border-sky-500/20"
              title="Keterangan singkat dibuat oleh AI"
            >
              <UIcon name="i-lucide-sparkles" class="w-2.5 h-2.5" />
              AI Summary
            </span>
            <button
              type="button"
              class="p-1 rounded-md hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-400 hover:text-sky-600 dark:hover:text-sky-400 transition-colors cursor-pointer"
              :title="refreshingSummary ? `Ringkasan AI '${project.name}': ${summaryJob.label.value}` : `Perbarui ringkasan AI khusus untuk proyek '${project.name}' ini saja`"
              :disabled="refreshingSummary"
              @click.stop="handleCardRefreshSummary"
            >
              <UIcon
                :name="refreshingSummary ? 'i-lucide-loader-2' : 'i-lucide-refresh-cw'"
                class="w-3 h-3"
                :class="refreshingSummary ? 'animate-spin text-sky-500' : ''"
              />
            </button>
          </div>
        </div>

        <div class="bg-neutral-50/60 dark:bg-neutral-800/30 rounded-lg p-2.5 border border-neutral-100 dark:border-neutral-800/60">
          <MarkdownViewer
            :content="project.short_description || project.description || 'Polyglot application bundle managed locally.'"
            compact
            :max-lines="3"
            empty-text="Belum ada keterangan singkat."
          />
        </div>
      </div>

      <!-- Component Architecture Breakdown -->
      <div class="mt-4 pt-3 border-t border-neutral-100 dark:border-neutral-800/80">
        <div class="text-xs font-semibold text-neutral-500 uppercase tracking-wider mb-2 flex items-center justify-between">
          <span>Bundle Topology ({{ project.components?.length || 0 }} Services)</span>
          <span class="text-xs lowercase font-normal text-neutral-400">ports & roles</span>
        </div>

        <div class="space-y-1.5">
          <div
            v-for="comp in project.components"
            :key="comp.id"
            class="flex items-center justify-between text-xs px-2.5 py-1.5 rounded-lg border font-mono transition-all"
            :class="currentlyCheckingCompId === comp.id
              ? 'bg-sky-50 dark:bg-sky-950/40 border-sky-400 dark:border-sky-500/50 shadow-xs'
              : 'bg-neutral-50 dark:bg-neutral-800/40 border-neutral-100 dark:border-neutral-800'"
          >
            <div class="flex items-center gap-2 truncate">
              <UBadge size="xs" :color="getRoleBadgeColor(comp.role)" variant="subtle">
                {{ getRoleLabel(comp.role) }}
              </UBadge>
              <span class="font-medium text-neutral-800 dark:text-neutral-200 truncate">{{ comp.name }}</span>
              <span
                v-if="currentlyCheckingCompId === comp.id"
                class="inline-flex items-center gap-1 text-xs text-sky-600 dark:text-sky-400 font-sans font-medium animate-pulse"
              >
                <UIcon name="i-lucide-loader-2" class="w-3 h-3 animate-spin" />
                Mengecek...
              </span>
              <span
                v-else-if="comp.relative_path || comp.absolute_path"
                class="text-xs text-neutral-400 dark:text-neutral-500 truncate max-w-[140px]"
                :title="comp.absolute_path || comp.relative_path"
              >
                📁 {{ comp.relative_path || (comp.absolute_path ? comp.absolute_path.split('/').filter(Boolean).pop() : '') }}
              </span>
            </div>

            <div class="flex items-center gap-1.5 shrink-0">
              <span
                v-if="comp.health_score !== undefined && comp.health_score !== null"
                class="px-1.5 py-0.5 rounded text-xs font-mono font-bold border"
                :class="getScoreColor(comp.health_score)"
                :title="`Skor Kesehatan ${comp.name}: ${comp.health_score}/100`"
              >
                {{ comp.health_score }}/100
              </span>
              <TechBadge :tech="comp.language" size="xs" />
              <TechBadge
                v-if="comp.framework && comp.framework.toLowerCase() !== comp.language.toLowerCase()"
                :tech="comp.framework"
                size="xs"
              />
              <span
                v-if="comp.port > 0"
                class="px-1.5 py-0.5 rounded text-xs font-mono font-bold bg-neutral-200 dark:bg-neutral-700 text-neutral-800 dark:text-neutral-200"
              >
                :{{ comp.port }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Action Buttons Footer -->
    <div class="mt-5 pt-3 border-t border-neutral-100 dark:border-neutral-800 flex items-center justify-between gap-2">
      <div class="flex items-center gap-1.5 flex-wrap">
        <UButton
          :to="`/projects/${project.id}`"
          icon="i-lucide-arrow-right"
          color="primary"
          variant="soft"
          size="xs"
          title="Buka cockpit detail arsitektur, rute API, port, dan analisis AI proyek ini"
        >
          Buka Monitor
        </UButton>

        <UButton
          icon="i-lucide-git-merge"
          color="neutral"
          variant="outline"
          size="xs"
          title="Gabungkan aplikasi dari proyek ini ke proyek target lain"
          @click="$emit('merge', project)"
        >
          Gabungkan dengan...
        </UButton>

        <UButton
          icon="i-lucide-activity"
          color="neutral"
          variant="outline"
          size="xs"
          :loading="diagnosing"
          title="Audit kesehatan seluruh aplikasi dalam proyek ini satu persatu"
          @click="$emit('diagnose', project)"
        >
          {{ diagnosing ? (diagnosingStatus || 'Memeriksa...') : 'Cek Kesehatan' }}
        </UButton>

        <!-- Status Actions: Deprecate, Hold, Done, or Aktifkan Kembali -->
        <template v-if="project.status === 'deprecated'">
          <UButton
            icon="i-lucide-rotate-ccw"
            color="warning"
            variant="soft"
            size="xs"
            :loading="statusLoading"
            :disabled="statusLoading"
            title="Kembalikan status proyek ini ke aktif"
            @click="$emit('activate', project)"
          >
            Aktifkan Kembali
          </UButton>
          <UButton
            icon="i-lucide-pause-circle"
            color="warning"
            variant="ghost"
            size="xs"
            :loading="statusLoading"
            :disabled="statusLoading"
            title="Ubah status proyek ini ke Hold"
            @click="$emit('hold', project)"
          >
            Hold
          </UButton>
          <UButton
            icon="i-lucide-check-circle-2"
            color="success"
            variant="ghost"
            size="xs"
            :loading="statusLoading"
            :disabled="statusLoading"
            title="Tandai proyek telah selesai (Done)"
            @click="$emit('done', project)"
          >
            Done
          </UButton>
        </template>

        <template v-else-if="project.status === 'hold'">
          <UButton
            icon="i-lucide-play"
            color="warning"
            variant="soft"
            size="xs"
            :loading="statusLoading"
            :disabled="statusLoading"
            title="Lanjutkan proyek (aktifkan kembali ke daftar aktif)"
            @click="$emit('activate', project)"
          >
            Aktifkan Kembali
          </UButton>
          <UButton
            icon="i-lucide-check-circle-2"
            color="success"
            variant="ghost"
            size="xs"
            :loading="statusLoading"
            :disabled="statusLoading"
            title="Tandai proyek telah selesai"
            @click="$emit('done', project)"
          >
            Done
          </UButton>
          <UButton
            icon="i-lucide-archive"
            color="neutral"
            variant="ghost"
            size="xs"
            :loading="statusLoading"
            :disabled="statusLoading"
            title="Tandai proyek sebagai deprecated"
            @click="$emit('deprecate', project)"
          >
            Deprecate
          </UButton>
        </template>

        <template v-else-if="project.status === 'done'">
          <UButton
            icon="i-lucide-rotate-ccw"
            color="success"
            variant="soft"
            size="xs"
            :loading="statusLoading"
            :disabled="statusLoading"
            title="Kembalikan status ke aktif untuk pengembangan lebih lanjut"
            @click="$emit('activate', project)"
          >
            Aktifkan Kembali
          </UButton>
          <UButton
            icon="i-lucide-pause-circle"
            color="warning"
            variant="ghost"
            size="xs"
            :loading="statusLoading"
            :disabled="statusLoading"
            title="Hold proyek ini"
            @click="$emit('hold', project)"
          >
            Hold
          </UButton>
          <UButton
            icon="i-lucide-archive"
            color="neutral"
            variant="ghost"
            size="xs"
            :loading="statusLoading"
            :disabled="statusLoading"
            title="Tandai proyek sebagai deprecated"
            @click="$emit('deprecate', project)"
          >
            Deprecate
          </UButton>
        </template>

        <template v-else>
          <UButton
            icon="i-lucide-archive"
            color="neutral"
            variant="ghost"
            size="xs"
            :loading="statusLoading"
            :disabled="statusLoading"
            title="Tandai proyek sebagai deprecated (arsipkan dari daftar aktif)"
            @click="$emit('deprecate', project)"
          >
            Deprecate
          </UButton>
          <UButton
            icon="i-lucide-pause-circle"
            color="warning"
            variant="ghost"
            size="xs"
            :loading="statusLoading"
            :disabled="statusLoading"
            title="Hold proyek agar tidak muncul di daftar aktif"
            @click="$emit('hold', project)"
          >
            Hold
          </UButton>
          <UButton
            icon="i-lucide-check-circle-2"
            color="success"
            variant="ghost"
            size="xs"
            :loading="statusLoading"
            :disabled="statusLoading"
            title="Tandai proyek telah selesai (hanya butuh maintenance)"
            @click="$emit('done', project)"
          >
            Done
          </UButton>
        </template>
      </div>

      <div class="flex items-center gap-1">
        <UButton
          v-if="!project.is_unified"
          icon="i-lucide-folder-input"
          color="warning"
          variant="subtle"
          size="xs"
          title="Satukan seluruh komponen proyek ke dalam 1 folder induk"
          @click="isConsolidateModalOpen = true"
        >
          Satukan
        </UButton>
        <UButton
          icon="i-lucide-trash-2"
          color="error"
          variant="ghost"
          size="xs"
          title="Hapus proyek ini dari pemantauan (file asli di laptop tetap aman)"
          @click="$emit('delete', project.id)"
        >
          Lepas
        </UButton>
      </div>
    </div>

    <!-- Consolidate Project Modal -->
    <ConsolidateProjectModal
      v-model:open="isConsolidateModalOpen"
      :project="project"
      @consolidated="handleConsolidated"
    />
  </div>
</template>
