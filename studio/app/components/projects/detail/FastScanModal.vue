<script setup lang="ts">
import type { FastScanResult, ProjectComponent } from '~/types'

const open = defineModel<boolean>('open', { default: false })

defineProps<{
  scanResult: FastScanResult | null
  activeComp: ProjectComponent | null
  scanningCompIds?: number[]
  applyingParams?: boolean
  savingScanChanges?: boolean
}>()

defineEmits<{
  applyParams: []
  saveChanges: []
  reScan: [comp: ProjectComponent]
}>()

const activeScanTab = ref<'architecture' | 'how_to_run' | 'env_database' | 'routes' | 'censored_keys' | 'structure' | 'analysis' | 'tree'>('architecture')
const showSecurityAuditDetails = ref(false)

const { getRoleBadgeColor: getCompRoleBadgeColor } = useFormatters()

const getCompRoleLabel = (role: string) => {
  switch (role) {
    case 'backend': return 'BE (Backend)'
    case 'frontend_admin': return 'FE Admin'
    case 'frontend_client': return 'FE Web'
    case 'frontend_mobile': return 'Mobile App'
    case 'docs': return 'Docs'
    case 'cli': return 'CLI'
    case 'worker': return 'Worker'
    default: return role === 'other' ? 'Folder Lain' : role
  }
}
</script>

<template>
  <UModal
    v-model:open="open"
    :title="`Fast Scan LLM: ${scanResult?.component_name || 'Komponen'}`"
    :description="`Struktur folder, cara menjalankan, dan rekomendasi runtime (${scanResult?.role || ''})`"
    :ui="{ content: 'sm:max-w-6xl xl:max-w-7xl w-full' }"
  >
    <template #body>
      <div v-if="scanResult" class="space-y-4 max-h-[82vh] overflow-y-auto pr-1">
        <!-- Global Security Sanitizer Banner -->
        <div class="p-3.5 rounded-xl border border-sky-200 dark:border-sky-900/60 bg-sky-50/70 dark:bg-sky-950/40 space-y-2">
          <div class="flex items-start justify-between gap-3">
            <div class="flex items-start gap-2.5">
              <UIcon name="i-lucide-shield-check" class="w-5 h-5 text-sky-600 dark:text-sky-400 shrink-0 mt-0.5" />
              <div>
                <div class="flex items-center gap-2">
                  <span class="font-bold text-xs text-sky-900 dark:text-sky-100">Global Security Sanitizer Aktif</span>
                  <UBadge size="xs" color="success" variant="subtle">
                    Zero Leak Guarantee
                  </UBadge>
                </div>
                <p class="text-[11px] text-sky-800 dark:text-sky-300 mt-0.5">
                  Semua API key server dan AI telah disensor secara global sebelum dikirim ke LLM.
                  <span v-if="scanResult.security_audit && scanResult.security_audit.redacted_count > 0">
                    Terdeteksi <b>{{ scanResult.security_audit.redacted_count }}</b> credential sensitif yang berhasil dialiaskan dengan safe alias.
                  </span>
                  <span v-else>
                    Payload bersih (tidak ada raw secret yang terkirim).
                  </span>
                </p>
              </div>
            </div>

            <UButton
              v-if="scanResult.security_audit && scanResult.security_audit.records && scanResult.security_audit.records.length > 0"
              variant="ghost"
              color="neutral"
              size="xs"
              @click="showSecurityAuditDetails = !showSecurityAuditDetails"
            >
              {{ showSecurityAuditDetails ? 'Sembunyikan' : 'Lihat Kunci Tersensor' }}
            </UButton>
          </div>

          <!-- Expandable Security Audit Table -->
          <div v-if="showSecurityAuditDetails && scanResult.security_audit?.records" class="mt-2 pt-2 border-t border-sky-200/70 dark:border-sky-900/40 space-y-1.5 text-xs">
            <div class="text-[10px] uppercase font-bold text-sky-900 dark:text-sky-200 tracking-wider">
              Daftar Kredensial yang Dialiaskan Sebelum Masuk ke LLM:
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-1.5 font-mono text-[11px]">
              <div
                v-for="(rec, rIdx) in scanResult.security_audit.records"
                :key="rIdx"
                class="flex items-center justify-between p-1.5 rounded bg-white/70 dark:bg-black/30 border border-sky-100 dark:border-sky-900/40"
              >
                <span class="text-neutral-500 text-[10px]">{{ rec.category }}:</span>
                <span class="text-sky-700 dark:text-sky-300 font-semibold">{{ rec.alias }}</span>
                <span class="text-neutral-400 text-[10px]">{{ rec.masked_preview }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Component Path & Role Strip -->
        <div class="flex flex-wrap items-center justify-between gap-2 p-2.5 rounded-lg bg-neutral-100 dark:bg-neutral-800 text-xs font-mono">
          <span class="flex items-center gap-1.5 text-neutral-600 dark:text-neutral-300 truncate">
            <UIcon name="i-lucide-folder" class="w-4 h-4 text-neutral-400" />
            <span>{{ scanResult.absolute_path }}</span>
          </span>
          <div class="flex items-center gap-2">
            <TechBadge v-if="scanResult.detected_params?.language" :tech="scanResult.detected_params.language" size="xs" />
            <TechBadge v-if="scanResult.detected_params?.framework" :tech="scanResult.detected_params.framework" size="xs" />
            <UBadge size="xs" :color="getCompRoleBadgeColor(scanResult.role)" variant="subtle">
              {{ getCompRoleLabel(scanResult.role) }}
            </UBadge>
            <span class="text-neutral-400 text-[10px]">
              Scanned: {{ new Date(scanResult.scanned_at).toLocaleTimeString() }}
            </span>
          </div>
        </div>

        <!-- Fast Scan Tab Bar -->
        <div class="flex items-center gap-1.5 border-b border-neutral-200 dark:border-neutral-800 pb-2 text-xs font-medium overflow-x-auto">
          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-colors cursor-pointer shrink-0"
            :class="activeScanTab === 'architecture' ? 'bg-primary-50 dark:bg-primary-950/50 text-primary-700 dark:text-primary-300 font-bold' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="activeScanTab = 'architecture'"
          >
            <UIcon name="i-lucide-cpu" class="w-3.5 h-3.5" />
            <span>Arsitektur & Pattern</span>
          </button>
          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-colors cursor-pointer shrink-0"
            :class="activeScanTab === 'how_to_run' ? 'bg-primary-50 dark:bg-primary-950/50 text-primary-700 dark:text-primary-300 font-bold' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="activeScanTab = 'how_to_run'"
          >
            <UIcon name="i-lucide-play" class="w-3.5 h-3.5" />
            <span>Cara Menjalankan</span>
          </button>
          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-colors cursor-pointer shrink-0"
            :class="activeScanTab === 'env_database' ? 'bg-primary-50 dark:bg-primary-950/50 text-primary-700 dark:text-primary-300 font-bold' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="activeScanTab = 'env_database'"
          >
            <UIcon name="i-lucide-database" class="w-3.5 h-3.5" />
            <span>Database & Env (Auth)</span>
          </button>
          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-colors cursor-pointer shrink-0"
            :class="activeScanTab === 'routes' ? 'bg-primary-50 dark:bg-primary-950/50 text-primary-700 dark:text-primary-300 font-bold' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="activeScanTab = 'routes'"
          >
            <UIcon name="i-lucide-route" class="w-3.5 h-3.5" />
            <span>Router & API ({{ scanResult.routes?.length || 0 }})</span>
          </button>
          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-colors cursor-pointer shrink-0"
            :class="activeScanTab === 'censored_keys' ? 'bg-amber-50 dark:bg-amber-950/50 text-amber-700 dark:text-amber-300 font-bold border border-amber-200/60 dark:border-amber-800/60' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="activeScanTab = 'censored_keys'"
          >
            <UIcon name="i-lucide-shield-alert" class="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" />
            <span>Sensor Kunci ({{ scanResult.security_audit?.redacted_count || 0 }})</span>
          </button>
          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-colors cursor-pointer shrink-0"
            :class="activeScanTab === 'structure' ? 'bg-primary-50 dark:bg-primary-950/50 text-primary-700 dark:text-primary-300 font-bold' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="activeScanTab = 'structure'"
          >
            <UIcon name="i-lucide-folder-tree" class="w-3.5 h-3.5" />
            <span>Struktur Folder</span>
          </button>
          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-colors cursor-pointer shrink-0"
            :class="activeScanTab === 'analysis' ? 'bg-primary-50 dark:bg-primary-950/50 text-primary-700 dark:text-primary-300 font-bold' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="activeScanTab = 'analysis'"
          >
            <UIcon name="i-lucide-bot" class="w-3.5 h-3.5" />
            <span>Analisis Lengkap AI</span>
          </button>
          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-colors cursor-pointer shrink-0"
            :class="activeScanTab === 'tree' ? 'bg-primary-50 dark:bg-primary-950/50 text-primary-700 dark:text-primary-300 font-bold' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="activeScanTab = 'tree'"
          >
            <UIcon name="i-lucide-list-tree" class="w-3.5 h-3.5" />
            <span>Tree Filesystem</span>
          </button>
        </div>

        <!-- TAB CONTENT 0: ARSITEKTUR & PATTERN -->
        <FastScanModalArchitecture
          v-if="activeScanTab === 'architecture'"
          :scan-result="scanResult"
        />

        <!-- TAB CONTENT 1: CARA MENJALANKAN (HOW TO RUN) -->
        <FastScanModalHowToRun
          v-if="activeScanTab === 'how_to_run'"
          :scan-result="scanResult"
        />

        <!-- TAB CONTENT: DATABASE, AUTH & ENV -->
        <FastScanModalEnvDatabase
          v-if="activeScanTab === 'env_database'"
          :scan-result="scanResult"
          :saving-scan-changes="savingScanChanges"
          @save-changes="$emit('saveChanges')"
        />

        <!-- TAB CONTENT: ROUTER & API -->
        <FastScanModalRoutes
          v-if="activeScanTab === 'routes'"
          :scan-result="scanResult"
        />

        <!-- TAB CONTENT: SENSORED KEYS & SECURITY VAULT -->
        <FastScanModalCensoredKeys
          v-if="activeScanTab === 'censored_keys'"
          :scan-result="scanResult"
        />

        <!-- TAB CONTENT 2: STRUKTUR FOLDER -->
        <FastScanModalStructure
          v-if="activeScanTab === 'structure'"
          :scan-result="scanResult"
        />

        <!-- TAB CONTENT 3: ANALISIS AI -->
        <FastScanModalAnalysis
          v-if="activeScanTab === 'analysis'"
          :scan-result="scanResult"
        />

        <!-- TAB CONTENT 4: DIRECTORY TREE -->
        <FastScanModalTree
          v-if="activeScanTab === 'tree'"
          :scan-result="scanResult"
        />
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-between w-full">
        <div>
          <UButton
            v-if="scanResult && (scanResult.detected_params.dev_command || scanResult.detected_params.build_command || scanResult.detected_params.port)"
            icon="i-lucide-check-check"
            color="primary"
            variant="solid"
            size="sm"
            :loading="applyingParams"
            title="Simpan dev command, build command, dan port hasil analisis AI ini ke profil pemantauan komponen"
            @click="$emit('applyParams')"
          >
            Simpan Parameter ke Profil Modul
          </UButton>
        </div>

        <div class="flex items-center gap-2">
          <UButton
            v-if="activeComp"
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="outline"
            size="sm"
            :loading="scanningCompIds?.includes(activeComp.id)"
            title="Pindai ulang modul ini dengan LLM untuk memperbarui rute API, arsitektur, dan env"
            @click="$emit('reScan', activeComp)"
          >
            Pindai Ulang (Re-Scan LLM)
          </UButton>
          <UButton
            color="neutral"
            variant="ghost"
            size="sm"
            title="Tutup jendela blueprint AI"
            @click="open = false"
          >
            Tutup Jendela
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
