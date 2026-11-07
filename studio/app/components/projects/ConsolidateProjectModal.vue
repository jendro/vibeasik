<script setup lang="ts">
import type { Project, ConsolidatePreview, ConsolidateResult, RoleBadgeColor } from '~/types'

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  project: Project | null
}>()

const emit = defineEmits<{
  consolidated: [result: ConsolidateResult]
}>()

const { getWorkspaceSettings, previewConsolidateProject, consolidateProject } = useApi()
const toast = useToast()

const targetParentDir = ref('')
const loadingPreview = ref(false)
const executing = ref(false)
const preview = ref<ConsolidatePreview | null>(null)
const errorMsg = ref<string | null>(null)

const loadPreview = async () => {
  if (!props.project?.id) return
  loadingPreview.value = true
  errorMsg.value = null

  try {
    if (!targetParentDir.value) {
      const ws = await getWorkspaceSettings().catch(() => null)
      if (ws?.default_projects_root) {
        targetParentDir.value = ws.default_projects_root
      } else {
        targetParentDir.value = '~/Documents/Projects'
      }
    }

    const res = await previewConsolidateProject(props.project.id, targetParentDir.value)
    preview.value = res
  } catch (err: any) {
    errorMsg.value = err.message || 'Gagal memuat rencana penyatuan folder'
  } finally {
    loadingPreview.value = false
  }
}

watch(open, (isOpen) => {
  if (isOpen) {
    targetParentDir.value = ''
    preview.value = null
    executing.value = false
    loadPreview()
  }
})

const getRoleBadgeColor = (role?: string): RoleBadgeColor => {
  switch (role) {
    case 'backend': return 'info'
    case 'frontend_admin': return 'secondary'
    case 'frontend_client': return 'success'
    case 'frontend_mobile': return 'warning'
    default: return 'neutral'
  }
}

const getRoleLabel = (role?: string) => {
  switch (role) {
    case 'backend': return 'BE'
    case 'frontend_admin': return 'FE Admin'
    case 'frontend_client': return 'FE Web'
    case 'frontend_mobile': return 'Mobile'
    case 'docs': return 'Docs'
    case 'cli': return 'CLI'
    case 'worker': return 'Worker'
    default: return role || 'Komponen'
  }
}

const formatBytes = (bytes?: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

const handleConsolidate = async () => {
  if (!props.project?.id || !preview.value?.can_consolidate) return
  executing.value = true

  try {
    const res = await consolidateProject(props.project.id, targetParentDir.value)
    toast.add({
      title: 'Folder Proyek Berhasil Disatukan',
      description: `Seluruh komponen '${props.project.name}' kini berada di ${res.project_root}; peta kode & tautan AI ada di root folder induk.`,
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
    if (res.warnings?.length) {
      toast.add({
        title: 'Catatan Penyatuan',
        description: res.warnings.join(' · '),
        icon: 'i-lucide-info',
        color: 'warning'
      })
    }
    emit('consolidated', res)
    open.value = false
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menyatukan Folder',
      description: err.message || 'Terjadi kesalahan saat memindahkan folder',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    executing.value = false
  }
}
</script>

<template>
  <UModal
    v-model:open="open"
    title="Satukan Folder Proyek"
    description="Kelompokkan semua komponen proyek ke dalam 1 folder induk dan tautkan AI ke parent workspace."
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #body>
      <div v-if="loadingPreview" class="p-8 text-center space-y-3">
        <UIcon name="i-lucide-loader-2" class="w-8 h-8 animate-spin mx-auto text-primary-500" />
        <p class="text-sm text-neutral-500 font-mono">
          Memeriksa struktur folder dan status git...
        </p>
      </div>

      <div v-else-if="errorMsg" class="p-4 rounded-xl border border-rose-500/30 bg-rose-50/50 dark:bg-rose-950/20 text-rose-800 dark:text-rose-200 text-xs space-y-2">
        <div class="flex items-center gap-2 font-semibold">
          <UIcon name="i-lucide-alert-circle" class="w-4 h-4 text-rose-500" />
          <span>Gagal Menganalisis Folder</span>
        </div>
        <p>{{ errorMsg }}</p>
      </div>

      <div v-else-if="preview" class="space-y-4">
        <!-- Target Directory Input Box -->
        <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-50/50 dark:bg-slate-900/60 space-y-2">
          <label class="text-xs font-semibold text-neutral-700 dark:text-neutral-300 flex items-center justify-between">
            <span class="flex items-center gap-1.5">
              <UIcon name="i-lucide-folder" class="w-3.5 h-3.5 text-primary-500" />
              <span>Lokasi Folder Induk Proyek</span>
            </span>
            <span class="text-[11px] text-neutral-400 font-mono">
              Total: {{ formatBytes(preview.total_size_bytes) }}
            </span>
          </label>
          <div class="flex items-center gap-2">
            <UInput
              v-model="targetParentDir"
              class="w-full font-mono text-xs"
              placeholder="~/Documents/Projects"
              :disabled="executing"
              @change="loadPreview"
            />
            <UButton
              color="neutral"
              variant="outline"
              size="xs"
              icon="i-lucide-refresh-cw"
              :disabled="executing"
              @click="loadPreview"
            >
              Cek Ulang
            </UButton>
          </div>
          <div class="text-[11px] font-mono text-neutral-500 dark:text-neutral-400 flex items-center gap-1">
            <span>Hasil Akhir:</span>
            <span class="text-primary-600 dark:text-primary-400 font-bold break-all">{{ preview.project_target_dir }}</span>
          </div>
        </div>

        <!-- Conflict Banner if any -->
        <div
          v-if="preview.has_conflict"
          class="p-3 rounded-lg border border-rose-500/40 bg-rose-50 dark:bg-rose-950/20 text-rose-800 dark:text-rose-200 text-xs space-y-1"
        >
          <div class="flex items-center gap-2 font-bold">
            <UIcon name="i-lucide-alert-triangle" class="w-4 h-4 text-rose-500 shrink-0" />
            <span>Konflik Folder Terdeteksi</span>
          </div>
          <p class="text-[11px]">
            Target folder tujuan sudah ada di komputer Anda (dan bukan sisa penyatuan proyek ini) atau folder sumber tidak ditemukan. Silakan ganti direktori induk proyek atau periksa path di filesystem.
          </p>
        </div>

        <!-- Components Move List -->
        <div class="space-y-2">
          <div class="flex items-center justify-between text-xs text-neutral-600 dark:text-neutral-400 font-medium">
            <span>Daftar Komponen yang Disatukan:</span>
            <span>{{ preview.items.length }} folder</span>
          </div>

          <div class="max-h-60 overflow-y-auto space-y-2 pr-1">
            <div
              v-for="item in preview.items"
              :key="item.id || item.current_path"
              class="p-2.5 rounded-lg border text-xs space-y-1.5 transition-colors"
              :class="item.status === 'conflict'
                ? 'border-rose-500/40 bg-rose-50/30 dark:bg-rose-950/10'
                : item.status === 'already_in_place'
                  ? 'border-emerald-500/40 bg-emerald-50/20 dark:bg-emerald-950/10'
                  : 'border-neutral-200 dark:border-neutral-800 bg-white dark:bg-slate-900/40'"
            >
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-bold text-neutral-900 dark:text-white">{{ item.component_name || 'Root Proyek' }}</span>
                  <UBadge size="xs" :color="getRoleBadgeColor(item.role)" variant="subtle">
                    {{ getRoleLabel(item.role) }}
                  </UBadge>
                  <UBadge v-if="item.is_git_repo" size="xs" color="success" variant="outline">
                    <UIcon name="i-lucide-git-branch" class="w-3 h-3 mr-1" />
                    Git History Aman
                  </UBadge>
                </div>
                <span class="font-mono text-[11px] text-neutral-400">
                  {{ formatBytes(item.size_bytes) }}
                </span>
              </div>

              <!-- Path Flow -->
              <div class="font-mono text-[11px] space-y-0.5">
                <div class="text-neutral-400 truncate flex items-center gap-1.5">
                  <span class="text-[10px] uppercase font-bold text-neutral-500">Asal:</span>
                  <span class="truncate">{{ item.current_path }}</span>
                </div>
                <div class="text-primary-600 dark:text-primary-400 font-semibold truncate flex items-center gap-1.5">
                  <span class="text-[10px] uppercase font-bold text-neutral-500">Tujuan:</span>
                  <span class="truncate">{{ item.target_path }}</span>
                </div>
              </div>

              <!-- Status or Warning Note -->
              <div v-if="item.status === 'already_in_place'" class="text-[10px] text-emerald-600 dark:text-emerald-400 font-medium flex items-center gap-1">
                <UIcon name="i-lucide-check" class="w-3 h-3" />
                <span>{{ item.warning || 'Sudah berada di folder tujuan.' }}</span>
              </div>
              <div v-else-if="item.warning" class="text-[10px] text-amber-600 dark:text-amber-400 flex items-center gap-1">
                <UIcon name="i-lucide-info" class="w-3 h-3" />
                <span>{{ item.warning }}</span>
              </div>
              <div v-else-if="item.error" class="text-[10px] text-rose-600 dark:text-rose-400 font-bold flex items-center gap-1">
                <UIcon name="i-lucide-x" class="w-3 h-3" />
                <span>{{ item.error }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Explanation Footer Note -->
        <div class="p-3 rounded-lg bg-sky-50/60 dark:bg-sky-950/20 border border-sky-500/20 text-xs text-sky-900 dark:text-sky-200 space-y-1">
          <div class="font-bold flex items-center gap-1.5">
            <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5 text-sky-600 dark:text-sky-400" />
            <span>Otomasi Ekosistem & Tautan AI</span>
          </div>
          <p class="text-[11px] leading-relaxed text-neutral-600 dark:text-neutral-400">
            Setelah disatukan, Vibeasik memperbarui database, memindahkan <code class="text-sky-600 dark:text-sky-400 font-mono">.vibeasik/</code> (peta kode & base knowledge) dari komponen ke folder induk, lalu menjalankan ulang Init vibeasik di sana: tautan AI (<code class="text-sky-600 dark:text-sky-400 font-mono">.vibeasik.json</code>) dan <code class="text-sky-600 dark:text-sky-400 font-mono">.vibeasik/CODEMAP.md</code> berada di root folder induk. AI coding agent (Antigravity/Cursor) akan memiliki visibilitas penuh atas seluruh komponen proyek.
          </p>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-end gap-2.5 w-full">
        <UButton
          color="neutral"
          variant="outline"
          size="sm"
          :disabled="executing"
          @click="open = false"
        >
          Batal
        </UButton>
        <UButton
          color="primary"
          variant="solid"
          size="sm"
          icon="i-lucide-folder-input"
          :loading="executing"
          :disabled="!preview || !preview.can_consolidate"
          @click="handleConsolidate"
        >
          Satukan Sekarang
        </UButton>
      </div>
    </template>
  </UModal>
</template>
