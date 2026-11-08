<script setup lang="ts">
import type { Project } from '~/types'

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  sourceProject: Project | null
  projects: Project[]
}>()

const emit = defineEmits<{
  merged: [targetProject: Project]
}>()

const { mergeProjects } = useApi()
const toast = useToast()

const searchQuery = ref('')
const selectedTarget = ref<Project | null>(null)
const isMerging = ref(false)

watch(open, (isOpen) => {
  if (isOpen) {
    searchQuery.value = ''
    selectedTarget.value = null
    isMerging.value = false
  }
})

const availableTargetProjects = computed(() => {
  if (!props.projects) return []
  const q = searchQuery.value.trim().toLowerCase()
  return props.projects.filter((p) => {
    // Cannot merge into self
    if (props.sourceProject && p.id === props.sourceProject.id) return false

    if (!q) return true
    return (
      p.name.toLowerCase().includes(q)
      || (p.tags && p.tags.toLowerCase().includes(q))
      || (p.root_path && p.root_path.toLowerCase().includes(q))
      || (p.short_description && p.short_description.toLowerCase().includes(q))
      || (p.description && p.description.toLowerCase().includes(q))
      || p.components?.some(c =>
        c.name.toLowerCase().includes(q)
        || c.language.toLowerCase().includes(q)
        || (c.framework && c.framework.toLowerCase().includes(q))
      )
    )
  })
})

const getScoreColor = (score: number) => {
  if (score >= 90) return 'text-sky-600 dark:text-sky-400 border-sky-500/30 bg-sky-50 dark:bg-sky-950/30'
  if (score >= 70) return 'text-amber-600 dark:text-amber-400 border-amber-500/30 bg-amber-50 dark:bg-amber-950/30'
  return 'text-rose-600 dark:text-rose-400 border-rose-500/30 bg-rose-50 dark:bg-rose-950/30'
}

const { getRoleBadgeColor } = useFormatters()

const { getRoleLabel: roleLabel } = useFormatters()
const getRoleLabel = (role: string) => roleLabel(role, { other: 'Lain' })

const handleSelectTarget = (target: Project) => {
  selectedTarget.value = target
}

const handleCancelSelection = () => {
  selectedTarget.value = null
}

const handleExecuteMerge = async () => {
  if (!props.sourceProject || !selectedTarget.value) return
  isMerging.value = true

  const sourceName = props.sourceProject.name
  const targetName = selectedTarget.value.name

  try {
    const res = await mergeProjects(props.sourceProject.id, selectedTarget.value.id)
    toast.add({
      title: 'Proyek Berhasil Digabungkan',
      description: `Aplikasi dari '${sourceName}' berhasil dipindahkan ke '${targetName}'. Keterangan proyek telah dibuat ulang oleh AI.`,
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
    emit('merged', res.target_project)
    open.value = false
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menggabungkan Proyek',
      description: err.message || 'Terjadi kesalahan saat menggabungkan proyek',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    isMerging.value = false
  }
}
</script>

<template>
  <UModal
    v-model:open="open"
    title="Gabungkan Proyek"
    description="Pindahkan semua aplikasi/layanan dari proyek ini ke dalam proyek target."
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #body>
      <div class="space-y-4">
        <!-- Source Project Banner -->
        <div v-if="sourceProject" class="p-3.5 rounded-xl border border-sky-500/30 bg-sky-50/50 dark:bg-sky-950/20 text-xs flex items-start gap-3">
          <div class="w-8 h-8 rounded-lg bg-sky-500/15 text-sky-600 dark:text-sky-400 flex items-center justify-center shrink-0 mt-0.5">
            <UIcon name="i-lucide-git-merge" class="w-4 h-4" />
          </div>
          <div class="space-y-1 min-w-0 flex-1">
            <div class="flex items-center gap-2 flex-wrap">
              <span class="text-neutral-500 dark:text-neutral-400">Proyek Sumber:</span>
              <span class="font-bold text-neutral-900 dark:text-white">{{ sourceProject.name }}</span>
              <UBadge size="xs" color="primary" variant="subtle">
                {{ sourceProject.components?.length || 0 }} Layanan
              </UBadge>
            </div>
            <p class="text-neutral-600 dark:text-neutral-300 leading-relaxed text-xs">
              Semua aplikasi dari proyek ini akan dipindahkan ke proyek target, lalu proyek <strong>{{ sourceProject.name }}</strong> akan dilepas dari pemantauan (file laptop tetap aman). Keterangan proyek target akan dibuat ulang oleh AI.
            </p>
          </div>
        </div>

        <!-- Mode 1: Confirmation Review when a target is selected -->
        <div v-if="selectedTarget" class="space-y-3.5 duration-200">
          <div class="p-4 rounded-xl border-2 border-primary-500/40 bg-white dark:bg-slate-900/80 shadow-xs space-y-3">
            <div class="flex items-center justify-between pb-2.5 border-b border-neutral-100 dark:border-neutral-800">
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-arrow-right-circle" class="w-4 h-4 text-primary-500" />
                <span class="text-xs font-semibold text-neutral-900 dark:text-white uppercase tracking-wider">
                  Target Penggabungan Terpilih
                </span>
              </div>
              <UButton
                color="neutral"
                variant="ghost"
                size="xs"
                icon="i-lucide-rotate-ccw"
                :disabled="isMerging"
                @click="handleCancelSelection"
              >
                Ganti Target
              </UButton>
            </div>

            <!-- Target Project Info Row -->
            <div class="flex items-center justify-between gap-3 text-xs">
              <div class="min-w-0 space-y-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-bold text-sm text-neutral-900 dark:text-white">
                    {{ selectedTarget.name }}
                  </span>
                  <UBadge size="xs" color="info" variant="subtle">
                    {{ selectedTarget.project_type }}
                  </UBadge>
                  <span
                    class="px-2 py-0.5 rounded-full text-xs font-mono font-bold border"
                    :class="getScoreColor(selectedTarget.health_score)"
                  >
                    {{ selectedTarget.health_score }}/100
                  </span>
                </div>
                <p class="text-xs text-neutral-500 line-clamp-1">
                  {{ selectedTarget.short_description || selectedTarget.description || 'Tidak ada deskripsi.' }}
                </p>
              </div>

              <div class="text-right shrink-0 font-mono text-xs text-neutral-500">
                <div>{{ selectedTarget.components?.length || 0 }} layanan saat ini</div>
                <div class="text-primary-600 dark:text-primary-400 font-semibold">
                  + {{ sourceProject?.components?.length || 0 }} layanan baru
                </div>
              </div>
            </div>

            <!-- List of services to be transferred -->
            <div class="pt-2 border-t border-neutral-100 dark:border-neutral-800 space-y-1.5">
              <span class="text-xs font-medium text-neutral-500">
                Aplikasi yang akan dipindahkan ke target:
              </span>
              <div class="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto">
                <div
                  v-for="c in sourceProject?.components"
                  :key="c.id"
                  class="inline-flex items-center gap-1.5 px-2 py-1 rounded-md bg-neutral-100 dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 text-xs font-mono border border-neutral-200 dark:border-neutral-700"
                >
                  <UBadge size="xs" :color="getRoleBadgeColor(c.role)" variant="subtle" class="text-xs px-1 py-0">
                    {{ getRoleLabel(c.role) }}
                  </UBadge>
                  <span>{{ c.name }}</span>
                  <span v-if="c.port > 0" class="text-neutral-400">:{{ c.port }}</span>
                </div>
              </div>
            </div>

            <!-- AI Summary Regeneration Notice -->
            <div class="flex items-center gap-2 p-2.5 rounded-lg bg-sky-50/60 dark:bg-sky-950/30 border border-sky-500/20 text-xs text-sky-800 dark:text-sky-300">
              <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5 shrink-0 text-sky-500" />
              <span>AI akan otomatis memperbarui ringkasan arsitektur proyek <strong>{{ selectedTarget.name }}</strong> setelah penggabungan selesai.</span>
            </div>
          </div>

          <!-- Bottom Action Buttons -->
          <div class="flex items-center justify-end gap-2 pt-1">
            <UButton
              color="neutral"
              variant="outline"
              size="sm"
              :disabled="isMerging"
              @click="handleCancelSelection"
            >
              Pilih Target Lain
            </UButton>
            <UButton
              color="primary"
              variant="solid"
              size="sm"
              icon="i-lucide-git-merge"
              :loading="isMerging"
              @click="handleExecuteMerge"
            >
              Konfirmasi & Gabungkan
            </UButton>
          </div>
        </div>

        <!-- Mode 2: Search & Selection of Target Project -->
        <div v-else class="space-y-3">
          <!-- Search input -->
          <div class="relative">
            <UInput
              v-model="searchQuery"
              icon="i-lucide-search"
              placeholder="Cari nama proyek target, stack teknologi, atau path..."
              size="sm"
              class="w-full"
              autofocus
            />
          </div>

          <!-- List Header with count -->
          <div class="flex items-center justify-between text-xs text-neutral-500 font-mono px-0.5">
            <span>Pilih salah satu proyek target di bawah ini:</span>
            <span>{{ availableTargetProjects.length }} proyek tersedia</span>
          </div>

          <!-- Target Projects List (Card seperti pada catalog tapi versi lebih simple dalam bentuk baris) -->
          <div class="space-y-2 max-h-[360px] overflow-y-auto pr-1">
            <div
              v-if="availableTargetProjects.length === 0"
              class="text-center py-10 px-4 rounded-xl border border-dashed border-neutral-300 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-900/30"
            >
              <UIcon name="i-lucide-search-x" class="w-7 h-7 mx-auto text-neutral-400 mb-1.5" />
              <p class="text-xs font-semibold text-neutral-800 dark:text-neutral-200">
                Tidak ada proyek target yang cocok
              </p>
              <p class="text-xs text-neutral-500 mt-0.5">
                Coba ubah kata kunci pencarian nama proyek.
              </p>
            </div>

            <!-- Simplified Row Card -->
            <div
              v-for="target in availableTargetProjects"
              :key="target.id"
              class="group p-3 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-900/60 hover:border-sky-500/40 hover:bg-neutral-50/70 dark:hover:bg-neutral-800/50 transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2.5"
            >
              <!-- Info Column -->
              <div class="min-w-0 flex-1 space-y-1">
                <!-- Top Row: Name, Badges, Health Score -->
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-semibold text-sm text-neutral-900 dark:text-white group-hover:text-sky-600 dark:group-hover:text-sky-400 transition-colors truncate">
                    {{ target.name }}
                  </span>

                  <UBadge
                    v-if="target.status === 'deprecated'"
                    size="xs"
                    color="warning"
                    variant="solid"
                  >
                    DEPRECATED
                  </UBadge>

                  <UBadge
                    size="xs"
                    :color="target.project_type === 'multi_fe' ? 'secondary' : target.project_type === 'bundle' ? 'info' : 'warning'"
                    variant="subtle"
                  >
                    {{ target.project_type === 'multi_fe' ? 'Multi-FE' : target.project_type === 'bundle' ? 'Bundle' : target.project_type }}
                  </UBadge>

                  <div
                    class="px-2 py-0.5 rounded-full text-xs font-mono font-bold border shrink-0"
                    :class="getScoreColor(target.health_score)"
                  >
                    {{ target.health_score }}/100
                  </div>
                </div>

                <!-- Meta Row: Services count & Folder name -->
                <div class="flex items-center gap-2.5 text-xs text-neutral-500 font-mono">
                  <span class="flex items-center gap-1">
                    <UIcon name="i-lucide-layers" class="w-3.5 h-3.5 text-sky-500 shrink-0" />
                    <span>{{ target.components?.length || 0 }} linked services</span>
                  </span>
                  <span
                    v-if="target.root_path && target.root_path !== '-'"
                    class="truncate text-xs text-neutral-400 max-w-[200px]"
                    :title="target.root_path"
                  >
                    📁 {{ target.root_path.split('/').filter(Boolean).pop() }}
                  </span>
                </div>

                <!-- Short description preview -->
                <p class="text-xs text-neutral-600 dark:text-neutral-400 line-clamp-1">
                  {{ target.short_description || target.description || 'Polyglot application bundle managed locally.' }}
                </p>
              </div>

              <!-- Action Column: Pilih Target -->
              <div class="shrink-0 flex items-center sm:self-center">
                <UButton
                  color="primary"
                  variant="soft"
                  size="xs"
                  icon="i-lucide-git-merge"
                  @click="handleSelectTarget(target)"
                >
                  Pilih Target
                </UButton>
              </div>
            </div>
          </div>

          <!-- Modal Footer Info -->
          <div class="flex items-center justify-between pt-2 border-t border-neutral-100 dark:border-neutral-800 text-xs">
            <span class="text-neutral-500 text-xs">
              Klik "Pilih Target" untuk melihat ringkasan sebelum konfirmasi
            </span>
            <UButton
              color="neutral"
              variant="outline"
              size="xs"
              @click="open = false"
            >
              Tutup
            </UButton>
          </div>
        </div>
      </div>
    </template>
  </UModal>
</template>
