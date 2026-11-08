<script setup lang="ts">
import type { FastScanResult, Project, ProjectComponent } from '~/types'

const props = defineProps<{
  comp: ProjectComponent
  project: Project
  scanResult: FastScanResult | null
  scanning?: boolean
  applyingParams?: boolean
  savingScanChanges?: boolean
}>()

defineEmits<{
  fastScan: []
  applyParams: []
  saveChanges: []
}>()

type SubTabId = 'context_pack' | 'architecture' | 'how_to_run' | 'env_database' | 'routes' | 'censored_keys' | 'structure' | 'analysis' | 'tree'

const activeSubTab = ref<SubTabId>('architecture')

const { getRoleBadgeColor: getCompRoleBadgeColor, getCompRoleLabel } = useFormatters()

const subTabs = computed(() => [
  { id: 'architecture', label: 'Arsitektur & Pattern', icon: 'i-lucide-cpu' },
  { id: 'how_to_run', label: 'Cara Menjalankan', icon: 'i-lucide-play' },
  { id: 'env_database', label: 'Database & Env (Auth)', icon: 'i-lucide-database' },
  { id: 'routes', label: `Router & API (${props.scanResult?.routes?.length || 0})`, icon: 'i-lucide-route' },
  { id: 'censored_keys', label: `Sensor Kunci (${props.scanResult?.security_audit?.redacted_count || 0})`, icon: 'i-lucide-shield-alert', alert: true },
  { id: 'structure', label: 'Struktur Folder', icon: 'i-lucide-folder-tree' },
  { id: 'analysis', label: 'Analisis Lengkap AI', icon: 'i-lucide-bot' },
  { id: 'tree', label: 'Tree Filesystem', icon: 'i-lucide-list-tree' },
  { id: 'context_pack', label: 'AI Context Pack', icon: 'i-lucide-file-text' }
])
</script>

<template>
  <div class="space-y-6">
    <!-- State Kosong: Belum Pernah Di-Scan -->
    <div
      v-if="!scanResult"
      class="p-10 text-center rounded-xl border border-dashed border-neutral-200 dark:border-neutral-800 bg-white/50 dark:bg-neutral-900/50 space-y-4"
    >
      <div class="w-12 h-12 mx-auto rounded-xl bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center text-neutral-400">
        <UIcon name="i-lucide-scan" class="w-6 h-6 text-primary-500" />
      </div>
      <div class="space-y-1.5">
        <h4 class="text-base font-bold text-neutral-900 dark:text-white">
          Detail Hasil Fast Scan LLM Belum Tersedia
        </h4>
        <p class="text-xs text-neutral-500 max-w-lg mx-auto leading-relaxed">
          Pindai aplikasi <b>{{ comp.name }}</b> dengan model LLM untuk mengekstrak arsitektur teknis, perintah dev & build, konfigurasi database, daftar router/halaman, kredensial tersensor, serta tree filesystem secara komprehensif.
        </p>
      </div>
      <div class="pt-2">
        <UButton
          size="sm"
          color="primary"
          icon="i-lucide-sparkles"
          :loading="scanning"
          @click="$emit('fastScan')"
        >
          Jalankan Fast Scan LLM Sekarang
        </UButton>
      </div>
    </div>

    <!-- State Berisi: Tampilkan Detail Fast Scan Lengkap -->
    <div v-else class="space-y-6">
      <!-- Top Action & Info Bar -->
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-2xs space-y-3.5">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-xl bg-sky-50 dark:bg-sky-950/60 border border-sky-200 dark:border-sky-800/80 flex items-center justify-center text-sky-600 dark:text-sky-400 shrink-0">
              <UIcon name="i-lucide-scan-text" class="w-5 h-5" />
            </div>
            <div>
              <div class="flex items-center gap-2">
                <h3 class="text-sm font-bold text-neutral-900 dark:text-white">
                  Hasil Inspeksi Fast Scan LLM
                </h3>
                <UBadge size="xs" :color="getCompRoleBadgeColor(scanResult.role)" variant="subtle">
                  {{ getCompRoleLabel(scanResult.role) }}
                </UBadge>
              </div>
              <p class="text-xs text-neutral-500 mt-0.5 font-mono">
                Dipindai: {{ new Date(scanResult.scanned_at).toLocaleString() }}
              </p>
            </div>
          </div>

          <!-- Action Buttons -->
          <div class="flex flex-wrap items-center gap-2">
            <UButton
              v-if="scanResult.detected_params?.dev_command || scanResult.detected_params?.build_command || scanResult.detected_params?.port"
              icon="i-lucide-check-check"
              color="primary"
              variant="solid"
              size="xs"
              :loading="applyingParams"
              title="Terapkan dev command, build command, dan port hasil analisis LLM ini ke profil aplikasi"
              @click="$emit('applyParams')"
            >
              Terapkan Parameter ke Profil
            </UButton>

            <UButton
              icon="i-lucide-refresh-cw"
              color="neutral"
              variant="outline"
              size="xs"
              :loading="scanning"
              title="Baca ulang kode aplikasi dengan LLM; hasil di halaman ini akan diganti"
              @click="$emit('fastScan')"
            >
              Pindai Ulang
            </UButton>
          </div>
        </div>

        <!-- Global Security Sanitizer Strip -->
        <div class="p-3 rounded-lg border border-sky-200 dark:border-sky-900/60 bg-sky-50/70 dark:bg-sky-950/40 flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 text-xs">
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-shield-check" class="w-4 h-4 text-sky-600 dark:text-sky-400 shrink-0" />
            <span class="text-sky-900 dark:text-sky-200 font-medium">
              Global Security Sanitizer:
              <b v-if="scanResult.security_audit && scanResult.security_audit.redacted_count > 0">
                {{ scanResult.security_audit.redacted_count }} kredensial sensitif berhasil disensor & dialiaskan secara aman.
              </b>
              <span v-else>
                Payload bersih tanpa kebocoran data rahasia (Zero Leak Guarantee).
              </span>
            </span>
          </div>

          <div class="flex items-center gap-2 shrink-0">
            <button
              v-if="scanResult.security_audit?.records && scanResult.security_audit.records.length > 0"
              type="button"
              class="text-sky-700 dark:text-sky-300 font-semibold hover:underline cursor-pointer text-xs"
              @click="activeSubTab = 'censored_keys'"
            >
              Buka Sensor Kunci ({{ scanResult.security_audit.records.length }})
            </button>
          </div>
        </div>
      </div>

      <!-- Navigation Sub-Tabs -->
      <div class="flex items-center gap-1.5 border-b border-neutral-200 dark:border-neutral-800 pb-2 text-xs font-medium overflow-x-auto">
        <button
          v-for="st in subTabs"
          :key="st.id"
          type="button"
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-colors cursor-pointer shrink-0 text-xs"
          :class="activeSubTab === st.id
            ? (st.alert ? 'bg-amber-100 dark:bg-amber-950/70 text-amber-800 dark:text-amber-200 font-bold border border-amber-300 dark:border-amber-700' : 'bg-primary-50 dark:bg-primary-950/50 text-primary-700 dark:text-primary-300 font-bold')
            : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
          @click="activeSubTab = st.id as SubTabId"
        >
          <UIcon :name="st.icon" class="w-3.5 h-3.5" />
          <span>{{ st.label }}</span>
        </button>
      </div>

      <!-- SUB-TAB: AI CONTEXT PACK -->
      <ProjectsDetailAiContextPack
        v-if="activeSubTab === 'context_pack'"
        :comp="comp"
        :project="project"
        :scan-result="scanResult"
      />

      <!-- SUB-TAB 1: ARSITEKTUR & PATTERN -->
      <FastScanDetailArchitecture
        v-if="activeSubTab === 'architecture'"
        :scan-result="scanResult"
      />

      <!-- SUB-TAB 2: CARA MENJALANKAN (HOW TO RUN) -->
      <FastScanDetailHowToRun
        v-if="activeSubTab === 'how_to_run'"
        :scan-result="scanResult"
      />

      <!-- SUB-TAB 3: DATABASE & ENV -->
      <FastScanDetailEnvDatabase
        v-if="activeSubTab === 'env_database'"
        :scan-result="scanResult"
        :saving-scan-changes="savingScanChanges"
        @save-changes="$emit('saveChanges')"
      />

      <!-- SUB-TAB 4: ROUTER & API -->
      <FastScanDetailRoutes
        v-if="activeSubTab === 'routes'"
        :scan-result="scanResult"
      />

      <!-- SUB-TAB 5: SENSORED KEYS & SECURITY VAULT -->
      <FastScanDetailCensoredKeys
        v-if="activeSubTab === 'censored_keys'"
        :scan-result="scanResult"
      />

      <!-- SUB-TAB 6: STRUKTUR FOLDER -->
      <FastScanDetailStructure
        v-if="activeSubTab === 'structure'"
        :scan-result="scanResult"
      />

      <!-- SUB-TAB 7: ANALISIS AI LENGKAP -->
      <FastScanDetailAnalysis
        v-if="activeSubTab === 'analysis'"
        :scan-result="scanResult"
      />

      <!-- SUB-TAB 8: TREE FILESYSTEM -->
      <FastScanDetailTree
        v-if="activeSubTab === 'tree'"
        :scan-result="scanResult"
      />
    </div>
  </div>
</template>
