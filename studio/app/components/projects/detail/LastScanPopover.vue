<script setup lang="ts">
import type { FastScanResult, Project } from '~/types'

const props = withDefaults(
  defineProps<{
    scanData?: string | FastScanResult | null
    componentName?: string
    componentRole?: string
    project?: Project | null
    isProjectLevel?: boolean
    align?: 'start' | 'center' | 'end'
    side?: 'top' | 'bottom' | 'left' | 'right'
  }>(),
  {
    scanData: null,
    componentName: '',
    componentRole: '',
    project: null,
    isProjectLevel: false,
    align: 'end',
    side: 'bottom'
  }
)

const parsedScan = computed<FastScanResult | null>(() => {
  if (!props.scanData) return null
  if (typeof props.scanData === 'object') return props.scanData
  try {
    return JSON.parse(props.scanData)
  } catch {
    return null
  }
})

// Formatting timestamp
const formatDate = (dateStr?: string) => {
  if (!dateStr) return '-'
  try {
    const d = new Date(dateStr)
    return d.toLocaleString('id-ID', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  } catch {
    return dateStr
  }
}

// Project level computed metrics
const projectScannedCount = computed(() => {
  if (!props.project?.components) return 0
  return props.project.components.filter((c) => {
    if (!c.fast_scan_data) return false
    try {
      const data = JSON.parse(c.fast_scan_data)
      return Boolean(data)
    } catch {
      return false
    }
  }).length
})

const projectTotalComps = computed(() => props.project?.components?.length || 0)

const projectTotalRoutes = computed(() => {
  if (!props.project?.components) return 0
  let count = 0
  for (const c of props.project.components) {
    if (!c.fast_scan_data) continue
    try {
      const data = JSON.parse(c.fast_scan_data)
      if (Array.isArray(data.routes)) {
        count += data.routes.length
      }
    } catch {
      // ignore
    }
  }
  return count
})

const projectLatestScanDate = computed(() => {
  if (!props.project?.components) return ''
  let latest = ''
  for (const c of props.project.components) {
    if (!c.fast_scan_data) continue
    try {
      const data = JSON.parse(c.fast_scan_data)
      if (data.scanned_at && (!latest || new Date(data.scanned_at) > new Date(latest))) {
        latest = data.scanned_at
      }
    } catch {
      // ignore
    }
  }
  return latest
})
</script>

<template>
  <UPopover
    mode="hover"
    :content="{ align, side }"
    :open-delay="150"
    :close-delay="150"
  >
    <slot />

    <template #content>
      <!-- PROJECT LEVEL POPOVER -->
      <div v-if="isProjectLevel" class="p-3.5 w-80 sm:w-88 space-y-3 text-xs">
        <div class="flex items-start justify-between gap-2 pb-2 border-b border-neutral-100 dark:border-white/[0.08]">
          <div class="flex items-center gap-2">
            <div class="p-1.5 rounded-lg bg-sky-50 dark:bg-sky-950/50 text-sky-600 dark:text-sky-400">
              <UIcon name="i-lucide-play-circle" class="w-4 h-4" />
            </div>
            <div>
              <div class="font-bold text-neutral-900 dark:text-white leading-tight">
                Pipeline Scan LLM
              </div>
              <div class="text-xs text-neutral-500 font-mono">
                {{ project?.name || 'Proyek' }}
              </div>
            </div>
          </div>
          <UBadge
            size="xs"
            :color="projectScannedCount > 0 ? 'primary' : 'neutral'"
            variant="subtle"
            class="font-mono text-xs"
          >
            {{ projectScannedCount }}/{{ projectTotalComps }} Folder Ter-Scan
          </UBadge>
        </div>

        <div v-if="projectScannedCount > 0" class="space-y-2">
          <div class="grid grid-cols-2 gap-2 text-xs">
            <div class="p-2 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-100 dark:border-white/[0.08]">
              <span class="text-neutral-400 block text-xs">Total Rute Terpetakan</span>
              <span class="font-bold text-neutral-900 dark:text-white font-mono text-sm">
                {{ projectTotalRoutes }} rute
              </span>
            </div>
            <div class="p-2 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-100 dark:border-white/[0.08]">
              <span class="text-neutral-400 block text-xs">Scan Terakhir</span>
              <span class="font-bold text-neutral-800 dark:text-neutral-200 text-xs truncate block" :title="formatDate(projectLatestScanDate)">
                {{ projectLatestScanDate ? formatDate(projectLatestScanDate) : 'Baru saja' }}
              </span>
            </div>
          </div>

          <p class="text-xs text-neutral-500 leading-relaxed">
            Klik untuk menjalankan ulang pipeline lengkap: fast scan seluruh folder modul, ekstraksi rute database, dan pembaruan ringkasan proyek otomatis.
          </p>
        </div>

        <div v-else class="space-y-1.5 py-1">
          <div class="flex items-center gap-1.5 text-amber-600 dark:text-amber-400 font-medium text-xs">
            <UIcon name="i-lucide-info" class="w-3.5 h-3.5" />
            <span>Belum ada modul yang di-scan</span>
          </div>
          <p class="text-xs text-neutral-500 leading-relaxed">
            Klik tombol ini untuk menjalankan pipeline background: memindai seluruh folder proyek, membaca framework, mendeteksi port, dan mengidentifikasi rute API secara otomatis.
          </p>
        </div>
      </div>

      <!-- COMPONENT / FOLDER LEVEL POPOVER -->
      <div v-else class="p-3.5 w-76 sm:w-84 space-y-3 text-xs">
        <!-- Header -->
        <div class="flex items-start justify-between gap-2 pb-2 border-b border-neutral-100 dark:border-white/[0.08]">
          <div class="flex items-center gap-2 min-w-0">
            <div class="p-1.5 rounded-lg bg-sky-50 dark:bg-sky-950/50 text-sky-600 dark:text-sky-400 shrink-0">
              <UIcon name="i-lucide-scan-text" class="w-4 h-4" />
            </div>
            <div class="min-w-0">
              <div class="font-bold text-neutral-900 dark:text-white truncate leading-tight">
                {{ componentName || 'Scan Folder' }}
              </div>
              <div class="text-xs text-neutral-500 font-mono capitalize">
                {{ componentRole || 'Modul' }}
              </div>
            </div>
          </div>

          <UBadge
            size="xs"
            :color="parsedScan ? 'primary' : 'neutral'"
            variant="subtle"
            class="shrink-0 text-xs"
          >
            {{ parsedScan ? 'Terpindai' : 'Belum Scan' }}
          </UBadge>
        </div>

        <!-- Popover Body: Has Last Scan Data -->
        <div v-if="parsedScan" class="space-y-2.5">
          <div class="flex items-center justify-between text-xs font-mono text-neutral-400">
            <span>Waktu Scan:</span>
            <span class="text-neutral-700 dark:text-neutral-300 font-medium">
              {{ formatDate(parsedScan.scanned_at) }}
            </span>
          </div>

          <!-- Quick Stats Grid -->
          <div class="grid grid-cols-2 gap-1.5 text-xs">
            <div class="p-2 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-100 dark:border-white/[0.08]">
              <span class="text-neutral-400 block text-xs">Rute / Endpoint</span>
              <span class="font-bold font-mono text-neutral-900 dark:text-white">
                {{ parsedScan.routes?.length || 0 }} rute
              </span>
            </div>
            <div class="p-2 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-100 dark:border-white/[0.08]">
              <span class="text-neutral-400 block text-xs">Arsitektur</span>
              <span class="font-bold text-neutral-900 dark:text-white truncate block" :title="parsedScan.architecture?.design_pattern || 'MVC Clean'">
                {{ parsedScan.architecture?.design_pattern || 'Standar Modul' }}
              </span>
            </div>
          </div>

          <!-- Dev Command & Framework -->
          <div class="space-y-1 text-xs">
            <div v-if="parsedScan.detected_params?.dev_command" class="flex items-center gap-1.5 font-mono text-neutral-600 dark:text-neutral-400 truncate">
              <UIcon name="i-lucide-terminal" class="w-3.5 h-3.5 text-neutral-400 shrink-0" />
              <span class="truncate">{{ parsedScan.detected_params.dev_command }}</span>
            </div>
            <div v-if="parsedScan.extracted_env?.database_type" class="flex items-center gap-1.5 font-mono text-neutral-600 dark:text-neutral-400 truncate">
              <UIcon name="i-lucide-database" class="w-3.5 h-3.5 text-neutral-400 shrink-0" />
              <span class="truncate">{{ parsedScan.extracted_env.database_type }} ({{ parsedScan.extracted_env.orm_driver || 'Direct' }})</span>
            </div>
            <div v-if="parsedScan.security_audit?.redacted_count !== undefined" class="flex items-center gap-1.5 text-neutral-500 text-xs">
              <UIcon name="i-lucide-shield-check" class="w-3.5 h-3.5 text-sky-500 shrink-0" />
              <span>{{ parsedScan.security_audit.redacted_count }} kredensial .env terlindungi</span>
            </div>
          </div>

          <!-- Summary Snippet -->
          <p v-if="parsedScan.folder_structure_summary" class="text-xs text-neutral-500 dark:text-neutral-400 line-clamp-2 leading-relaxed italic border-t border-neutral-100 dark:border-neutral-800/80 pt-2">
            "{{ parsedScan.folder_structure_summary }}"
          </p>

          <div class="text-xs text-neutral-400 text-right pt-0.5">
            Klik tombol untuk memindai ulang modul ini
          </div>
        </div>

        <!-- Popover Body: No Scan Data (Empty State) -->
        <div v-else class="space-y-2 py-1">
          <div class="flex items-center gap-2 text-amber-600 dark:text-amber-400 font-semibold text-xs">
            <UIcon name="i-lucide-scan-line" class="w-4 h-4" />
            <span>Belum Ada Riwayat Scan LLM</span>
          </div>
          <p class="text-xs text-neutral-500 dark:text-neutral-400 leading-relaxed">
            Folder modul ini belum pernah dipindai oleh LLM. Klik tombol untuk memetakan rute, dependensi package, database, dan dev command secara otomatis.
          </p>
        </div>
      </div>
    </template>
  </UPopover>
</template>
