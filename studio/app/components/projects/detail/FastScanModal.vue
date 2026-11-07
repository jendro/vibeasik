<script setup lang="ts">
import type { FastScanResult, ProjectComponent, RoleBadgeColor } from '~/types'

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
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

const toast = useToast()

const activeScanTab = ref<'architecture' | 'how_to_run' | 'env_database' | 'routes' | 'censored_keys' | 'structure' | 'analysis' | 'tree'>('architecture')
const showSecurityAuditDetails = ref(false)
const routeSearchQuery = ref('')
const newMetaKey = ref('')
const newMetaVal = ref('')

const copyToClipboard = async (text: string, label = 'Copied!') => {
  try {
    await navigator.clipboard.writeText(text)
    toast.add({
      title: 'Disalin ke Clipboard',
      description: label,
      icon: 'i-lucide-clipboard-check',
      color: 'neutral'
    })
  } catch {
    // ignore
  }
}

const getCompRoleBadgeColor = (role: string): RoleBadgeColor => {
  switch (role) {
    case 'backend': return 'info'
    case 'frontend_admin': return 'secondary'
    case 'frontend_client': return 'success'
    case 'frontend_mobile': return 'warning'
    default: return 'neutral'
  }
}

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

const getMethodBadgeColor = (method: string): 'success' | 'info' | 'warning' | 'error' | 'neutral' | 'primary' => {
  const m = method.toUpperCase()
  if (m === 'GET') return 'success'
  if (m === 'POST') return 'info'
  if (m === 'PUT' || m === 'PATCH') return 'warning'
  if (m === 'DELETE') return 'error'
  if (m.includes('PAGE') || m.includes('VIEW') || m.includes('SCREEN')) return 'primary'
  return 'neutral'
}

const filteredRoutes = computed(() => {
  if (!props.scanResult?.routes) return []
  const q = routeSearchQuery.value.trim().toLowerCase()
  if (!q) return props.scanResult.routes
  return props.scanResult.routes.filter(r =>
    r.path.toLowerCase().includes(q)
    || r.method.toLowerCase().includes(q)
    || r.handler.toLowerCase().includes(q)
    || r.description.toLowerCase().includes(q)
    || (r.middleware && r.middleware.toLowerCase().includes(q))
  )
})

const addDynamicMeta = () => {
  if (!props.scanResult) return
  if (!props.scanResult.extracted_env) {
    props.scanResult.extracted_env = {
      database_type: '',
      orm_driver: '',
      database_name: '',
      auth_methods: [],
      login_endpoints: [],
      third_party_services: [],
      environment_keys: [],
      login_and_access_guide: '',
      dynamic_metadata: {}
    }
  }
  if (!props.scanResult.extracted_env.dynamic_metadata) {
    props.scanResult.extracted_env.dynamic_metadata = {}
  }
  const k = newMetaKey.value.trim()
  const v = newMetaVal.value.trim()
  if (!k) {
    toast.add({
      title: 'Validasi Gagal',
      description: 'Nama properti / atribut wajib diisi',
      icon: 'i-lucide-alert-circle',
      color: 'warning'
    })
    return
  }
  props.scanResult.extracted_env.dynamic_metadata[k] = v
  newMetaKey.value = ''
  newMetaVal.value = ''
}

const removeDynamicMeta = (key: string) => {
  if (!props.scanResult?.extracted_env?.dynamic_metadata) return
  delete props.scanResult.extracted_env.dynamic_metadata[key]
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
        <div v-if="activeScanTab === 'architecture'" class="space-y-4">
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-cpu" class="w-4 h-4 text-primary-600 dark:text-primary-400" />
                <span class="text-xs font-bold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">Framework & Engine HTTP</span>
              </div>
              <TechBadge
                :tech="scanResult.architecture?.framework || scanResult.detected_params.framework"
                size="sm"
              />
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs font-mono">
              <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200 dark:border-neutral-700/60">
                <div class="text-[10px] uppercase font-bold text-neutral-400">
                  Framework Name
                </div>
                <div class="mt-1 flex items-center gap-1.5 font-semibold text-neutral-800 dark:text-neutral-100">
                  <TechBadge :tech="scanResult.architecture?.framework || scanResult.detected_params.framework || 'Standard Go'" size="xs" />
                </div>
              </div>

              <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200 dark:border-neutral-700/60 sm:col-span-2">
                <div class="text-[10px] uppercase font-bold text-neutral-400">
                  Engine / Architecture Type
                </div>
                <div class="mt-1 font-semibold text-neutral-800 dark:text-neutral-100 truncate" :title="scanResult.architecture?.framework_type">
                  {{ scanResult.architecture?.framework_type || 'HTTP Server & RESTful API Engine' }}
                </div>
              </div>
            </div>
          </div>

          <!-- Design Pattern Card -->
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-network" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
                <span class="text-xs font-bold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">Desain Pola Arsitektur (Design Pattern)</span>
              </div>
              <UBadge
                size="xs"
                color="success"
                variant="subtle"
              >
                {{ scanResult.architecture?.design_pattern || 'MVC Pattern' }}
              </UBadge>
            </div>

            <div class="p-3.5 rounded-lg bg-sky-50/70 dark:bg-sky-950/40 border border-sky-200/70 dark:border-sky-900/40 space-y-1.5 text-xs">
              <div class="font-bold text-sky-900 dark:text-sky-200">
                {{ scanResult.architecture?.design_pattern || 'Model-Controller Architecture' }}
              </div>
              <p class="text-sky-800 dark:text-sky-300 text-[11px] leading-relaxed whitespace-pre-line">
                {{ scanResult.architecture?.pattern_details || scanResult.folder_structure_summary }}
              </p>
            </div>
          </div>

          <!-- Plugins Card -->
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-blocks" class="w-4 h-4 text-purple-600 dark:text-purple-400" />
                <span class="text-xs font-bold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">Plugins, Middlewares & Libraries</span>
              </div>
              <span class="text-[11px] text-neutral-400 font-mono">
                {{ scanResult.architecture?.plugins?.length || 0 }} library terdeteksi
              </span>
            </div>

            <div v-if="scanResult.architecture?.plugins && scanResult.architecture.plugins.length > 0" class="flex flex-wrap gap-2">
              <TechBadge
                v-for="(plug, pIdx) in scanResult.architecture.plugins"
                :key="pIdx"
                :tech="plug"
                size="sm"
              />
            </div>
            <div v-else class="text-xs text-neutral-500 italic">
              Tidak ada plugin atau middleware pihak ketiga tambahan yang terdeteksi.
            </div>
          </div>
        </div>

        <!-- TAB CONTENT 1: CARA MENJALANKAN (HOW TO RUN) -->
        <div v-if="activeScanTab === 'how_to_run'" class="space-y-4">
          <div v-if="scanResult.how_to_run.prerequisites && scanResult.how_to_run.prerequisites.length > 0" class="space-y-1.5">
            <label class="block text-xs font-semibold text-neutral-500 uppercase tracking-wider">Prasyarat Runtime:</label>
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="(p, pIdx) in scanResult.how_to_run.prerequisites"
                :key="pIdx"
                class="px-2 py-0.5 rounded text-xs font-mono bg-neutral-100 dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 border border-neutral-200 dark:border-neutral-700"
              >
                {{ p }}
              </span>
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs font-mono">
            <div class="p-3 rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-800/40 space-y-1.5">
              <div class="flex items-center justify-between text-neutral-500 text-[10px] uppercase font-bold">
                <span class="flex items-center gap-1 text-sky-600 dark:text-sky-400">
                  <UIcon name="i-lucide-terminal" class="w-3.5 h-3.5" />
                  Dev Command
                </span>
                <button
                  class="text-sky-600 dark:text-sky-400 hover:underline cursor-pointer"
                  @click="copyToClipboard(scanResult.how_to_run.dev_command, 'Dev command copied!')"
                >
                  copy
                </button>
              </div>
              <div class="p-2 rounded bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-700 select-all font-bold text-neutral-900 dark:text-white truncate">
                {{ scanResult.how_to_run.dev_command || 'Tidak terdeteksi' }}
              </div>
            </div>

            <div class="p-3 rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-800/40 space-y-1.5">
              <div class="flex items-center justify-between text-neutral-500 text-[10px] uppercase font-bold">
                <span class="flex items-center gap-1 text-blue-600 dark:text-blue-400">
                  <UIcon name="i-lucide-download" class="w-3.5 h-3.5" />
                  Install Dependencies
                </span>
                <button
                  class="text-blue-600 dark:text-blue-400 hover:underline cursor-pointer"
                  @click="copyToClipboard(scanResult.how_to_run.install_command, 'Install command copied!')"
                >
                  copy
                </button>
              </div>
              <div class="p-2 rounded bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-700 select-all text-neutral-800 dark:text-neutral-200 truncate">
                {{ scanResult.how_to_run.install_command || 'Tidak terdeteksi' }}
              </div>
            </div>

            <div class="p-3 rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-800/40 space-y-1.5">
              <div class="flex items-center justify-between text-neutral-500 text-[10px] uppercase font-bold">
                <span class="flex items-center gap-1 text-amber-600 dark:text-amber-400">
                  <UIcon name="i-lucide-hammer" class="w-3.5 h-3.5" />
                  Build Command
                </span>
                <button
                  class="text-amber-600 dark:text-amber-400 hover:underline cursor-pointer"
                  @click="copyToClipboard(scanResult.how_to_run.build_command, 'Build command copied!')"
                >
                  copy
                </button>
              </div>
              <div class="p-2 rounded bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-700 select-all text-neutral-800 dark:text-neutral-200 truncate">
                {{ scanResult.how_to_run.build_command || 'Tidak terdeteksi' }}
              </div>
            </div>

            <div class="p-3 rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-800/40 space-y-1.5">
              <div class="flex items-center justify-between text-neutral-500 text-[10px] uppercase font-bold">
                <span class="flex items-center gap-1 text-purple-600 dark:text-purple-400">
                  <UIcon name="i-lucide-check-circle-2" class="w-3.5 h-3.5" />
                  Test Command
                </span>
                <button
                  class="text-purple-600 dark:text-purple-400 hover:underline cursor-pointer"
                  @click="copyToClipboard(scanResult.how_to_run.test_command, 'Test command copied!')"
                >
                  copy
                </button>
              </div>
              <div class="p-2 rounded bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-700 select-all text-neutral-800 dark:text-neutral-200 truncate">
                {{ scanResult.how_to_run.test_command || 'Tidak terdeteksi' }}
              </div>
            </div>
          </div>

          <div v-if="scanResult.how_to_run.steps && scanResult.how_to_run.steps.length > 0" class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-2">
            <div class="text-xs font-semibold text-neutral-700 dark:text-neutral-300">
              Langkah Menjalankan (Step-by-Step):
            </div>
            <ul class="space-y-1.5 text-xs text-neutral-600 dark:text-neutral-400">
              <li v-for="(s, sIdx) in scanResult.how_to_run.steps" :key="sIdx" class="flex items-start gap-2">
                <span class="font-bold text-sky-600 dark:text-sky-400 mt-0.5">•</span>
                <span>{{ s }}</span>
              </li>
            </ul>
          </div>
        </div>

        <!-- TAB CONTENT: DATABASE, AUTH & ENV -->
        <div v-if="activeScanTab === 'env_database'" class="space-y-4">
          <div class="flex flex-wrap items-center justify-between gap-2 p-3 rounded-xl bg-neutral-50 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-800 text-xs">
            <div class="flex items-center gap-2 text-neutral-600 dark:text-neutral-300">
              <UIcon name="i-lucide-database-zap" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
              <span class="font-medium">Informasi diekstrak dari .env, file konfigurasi, manifest, dan arsitektur kode.</span>
            </div>
            <UButton
              icon="i-lucide-save"
              color="primary"
              size="xs"
              variant="solid"
              :loading="savingScanChanges"
              @click="$emit('saveChanges')"
            >
              Simpan Perubahan Dinamis
            </UButton>
          </div>

          <!-- Database Section -->
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-database" class="w-4 h-4 text-primary-600 dark:text-primary-400" />
                <span class="text-xs font-bold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">Database & Persistence</span>
              </div>
              <TechBadge
                v-if="scanResult.extracted_env?.database_type"
                :tech="scanResult.extracted_env.database_type"
                size="sm"
              />
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs font-mono">
              <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200 dark:border-neutral-700/60">
                <div class="text-[10px] uppercase font-bold text-neutral-400">
                  Database Engine
                </div>
                <div class="mt-1 flex items-center gap-1.5 font-semibold text-neutral-800 dark:text-neutral-100">
                  <TechBadge :tech="scanResult.extracted_env?.database_type || 'None'" size="xs" />
                </div>
              </div>

              <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200 dark:border-neutral-700/60">
                <div class="text-[10px] uppercase font-bold text-neutral-400">
                  ORM / Driver / Client
                </div>
                <div class="mt-1 flex items-center gap-1.5 font-semibold text-neutral-800 dark:text-neutral-100">
                  <TechBadge :tech="scanResult.extracted_env?.orm_driver || 'Standar / Raw SQL'" size="xs" />
                </div>
              </div>

              <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200 dark:border-neutral-700/60">
                <div class="text-[10px] uppercase font-bold text-neutral-400">
                  Nama DB / Connection Target
                </div>
                <div class="mt-1 font-semibold text-neutral-800 dark:text-neutral-100 truncate" :title="scanResult.extracted_env?.database_name">
                  {{ scanResult.extracted_env?.database_name || '-' }}
                </div>
              </div>
            </div>
          </div>

          <!-- Login & Auth -->
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3">
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-key-round" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
              <span class="text-xs font-bold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">Mekanisme Login & Autentikasi</span>
            </div>

            <div class="space-y-1.5">
              <label class="block text-[11px] font-semibold text-neutral-500 uppercase tracking-wider">Metode Login yang Tersedia:</label>
              <div v-if="scanResult.extracted_env?.auth_methods && scanResult.extracted_env.auth_methods.length > 0" class="flex flex-wrap gap-2">
                <TechBadge
                  v-for="(auth, aIdx) in scanResult.extracted_env.auth_methods"
                  :key="aIdx"
                  :tech="auth"
                  size="sm"
                />
              </div>
              <div v-else class="text-xs text-neutral-500 italic">
                Tidak terdeteksi autentikasi khusus atau endpoint login publik.
              </div>
            </div>

            <div v-if="scanResult.extracted_env?.login_endpoints && scanResult.extracted_env.login_endpoints.length > 0" class="space-y-1.5">
              <label class="block text-[11px] font-semibold text-neutral-500 uppercase tracking-wider">Endpoint Login / Auth:</label>
              <div class="flex flex-wrap gap-2 font-mono text-xs">
                <span
                  v-for="(ep, epIdx) in scanResult.extracted_env.login_endpoints"
                  :key="epIdx"
                  class="px-2 py-1 rounded bg-neutral-100 dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 border border-neutral-200 dark:border-neutral-700 select-all flex items-center gap-1.5"
                >
                  <UIcon name="i-lucide-send" class="w-3 h-3 text-neutral-400" />
                  <span>{{ ep }}</span>
                </span>
              </div>
            </div>

            <div v-if="scanResult.extracted_env?.login_and_access_guide" class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1.5 text-xs">
              <div class="flex items-center gap-1 text-[11px] font-bold text-neutral-600 dark:text-neutral-300">
                <UIcon name="i-lucide-info" class="w-3.5 h-3.5 text-primary-500" />
                <span>Petunjuk Akses / Kredensial Default / Cara Login:</span>
              </div>
              <p class="leading-relaxed text-neutral-700 dark:text-neutral-300 whitespace-pre-line">
                {{ scanResult.extracted_env.login_and_access_guide }}
              </p>
            </div>
          </div>

          <!-- Dynamic Custom Metadata Manager -->
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-tag" class="w-4 h-4 text-purple-600 dark:text-purple-400" />
                <span class="text-xs font-bold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">Metadata & Atribut Dinamis Komponen</span>
              </div>
              <span class="text-[10px] text-neutral-400">Dapat ditambah & disimpan fleksibel</span>
            </div>

            <div
              v-if="scanResult.extracted_env?.dynamic_metadata && Object.keys(scanResult.extracted_env.dynamic_metadata).length > 0"
              class="grid grid-cols-1 sm:grid-cols-2 gap-2"
            >
              <div
                v-for="(val, key) in scanResult.extracted_env.dynamic_metadata"
                :key="key"
                class="flex items-center justify-between gap-2 p-2.5 rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-50/70 dark:bg-neutral-800/40 text-xs"
              >
                <div class="min-w-0 flex-1">
                  <div class="font-bold text-neutral-800 dark:text-neutral-200 truncate">
                    {{ key }}
                  </div>
                  <div class="text-[11px] text-neutral-600 dark:text-neutral-400 truncate">
                    {{ val }}
                  </div>
                </div>
                <button
                  type="button"
                  class="text-neutral-400 hover:text-red-500 p-1 cursor-pointer transition-colors"
                  title="Hapus properti"
                  @click="removeDynamicMeta(String(key))"
                >
                  <UIcon name="i-lucide-x" class="w-4 h-4" />
                </button>
              </div>
            </div>

            <!-- Add Form Inline -->
            <div class="pt-2 border-t border-neutral-100 dark:border-neutral-800 flex flex-wrap sm:flex-nowrap items-center gap-2 text-xs">
              <input
                v-model="newMetaKey"
                type="text"
                placeholder="Nama Properti (e.g. redis_prefix, queue_driver)"
                class="flex-1 px-3 py-1.5 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 text-xs focus:outline-none focus:ring-1 focus:ring-primary-500"
                @keydown.enter.prevent="addDynamicMeta"
              >
              <input
                v-model="newMetaVal"
                type="text"
                placeholder="Nilai / Catatan"
                class="flex-1 px-3 py-1.5 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 text-xs focus:outline-none focus:ring-1 focus:ring-primary-500"
                @keydown.enter.prevent="addDynamicMeta"
              >
              <UButton
                icon="i-lucide-plus"
                size="xs"
                color="neutral"
                variant="subtle"
                @click="addDynamicMeta"
              >
                Tambah
              </UButton>
            </div>
          </div>
        </div>

        <!-- TAB CONTENT: ROUTER & API -->
        <div v-if="activeScanTab === 'routes'" class="space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-2 p-3 rounded-xl bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800">
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-route" class="w-4 h-4 text-primary-600 dark:text-primary-400" />
              <span class="text-xs font-bold text-neutral-800 dark:text-neutral-200">
                Daftar Router & API Endpoints
              </span>
              <UBadge size="xs" color="primary" variant="subtle">
                {{ filteredRoutes.length }} dari {{ scanResult.routes?.length || 0 }} Endpoints
              </UBadge>
            </div>

            <div class="w-full sm:w-64">
              <UInput
                v-model="routeSearchQuery"
                icon="i-lucide-search"
                placeholder="Cari router, path, handler..."
                size="xs"
              />
            </div>
          </div>

          <div v-if="filteredRoutes.length > 0" class="overflow-x-auto rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40">
            <table class="w-full text-left text-xs">
              <thead class="bg-neutral-50 dark:bg-neutral-800 text-neutral-500 uppercase text-[10px] tracking-wider border-b border-neutral-200 dark:border-neutral-800">
                <tr>
                  <th class="p-3 w-20">
                    Method
                  </th>
                  <th class="p-3">
                    Endpoint Path
                  </th>
                  <th class="p-3">
                    Fungsi yang Diakses (Handler)
                  </th>
                  <th class="p-3">
                    Keamanan / Auth
                  </th>
                  <th class="p-3">
                    Penjelasan Fungsi
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/60 font-mono text-xs">
                <tr
                  v-for="(r, rIdx) in filteredRoutes"
                  :key="rIdx"
                  class="hover:bg-neutral-50/60 dark:hover:bg-neutral-800/40 transition-colors"
                >
                  <td class="p-3 whitespace-nowrap">
                    <UBadge
                      size="xs"
                      :color="getMethodBadgeColor(r.method)"
                      variant="subtle"
                      class="font-mono font-bold uppercase tracking-wider"
                    >
                      {{ r.method }}
                    </UBadge>
                  </td>

                  <td class="p-3 font-semibold text-neutral-900 dark:text-neutral-100 whitespace-nowrap">
                    <div class="flex items-center gap-1.5">
                      <span class="select-all">{{ r.path }}</span>
                      <button
                        type="button"
                        class="text-neutral-400 hover:text-primary-600 transition-colors cursor-pointer"
                        title="Copy path"
                        @click="copyToClipboard(r.path, 'Path copied!')"
                      >
                        <UIcon name="i-lucide-copy" class="w-3 h-3" />
                      </button>
                    </div>
                  </td>

                  <td class="p-3 whitespace-nowrap">
                    <div class="flex items-center gap-1 text-purple-700 dark:text-purple-300 font-semibold bg-purple-50 dark:bg-purple-950/40 px-2 py-0.5 rounded border border-purple-200/50 dark:border-purple-900/50 w-fit">
                      <UIcon name="i-lucide-code-2" class="w-3 h-3 text-purple-500" />
                      <span>{{ r.handler }}</span>
                    </div>
                  </td>

                  <td class="p-3 whitespace-nowrap">
                    <span
                      class="text-[11px] px-2 py-0.5 rounded font-sans inline-flex items-center gap-1"
                      :class="r.middleware && r.middleware.includes('Protected') ? 'bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300 border border-amber-200/60 dark:border-amber-900/50' : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400'"
                    >
                      <UIcon
                        :name="r.middleware && r.middleware.includes('Protected') ? 'i-lucide-lock' : 'i-lucide-globe'"
                        class="w-3 h-3"
                      />
                      <span>{{ r.middleware || 'Public' }}</span>
                    </span>
                  </td>

                  <td class="p-3 font-sans text-xs text-neutral-600 dark:text-neutral-300 leading-relaxed min-w-[240px]">
                    {{ r.description }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-else class="p-8 text-center text-xs text-neutral-500 italic bg-white dark:bg-neutral-900 rounded-xl border border-neutral-200 dark:border-neutral-800">
            Tidak ada router yang cocok dengan pencarian "{{ routeSearchQuery }}".
          </div>
        </div>

        <!-- TAB CONTENT: SENSORED KEYS & SECURITY VAULT -->
        <div v-if="activeScanTab === 'censored_keys'" class="space-y-4">
          <div class="p-4 rounded-xl border border-amber-200 dark:border-amber-900/60 bg-amber-50/50 dark:bg-amber-950/20 space-y-3">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="flex items-center gap-2.5">
                <div class="w-7 h-7 rounded-lg bg-amber-500/10 border border-amber-300 dark:border-amber-700 flex items-center justify-center text-amber-600 dark:text-amber-400">
                  <UIcon name="i-lucide-shield-check" class="w-4 h-4" />
                </div>
                <div>
                  <div class="text-xs font-bold text-amber-900 dark:text-amber-200 flex items-center gap-2">
                    <span>Double Security Censorship Shield</span>
                    <UBadge size="xs" color="warning" variant="subtle">
                      {{ scanResult.security_audit?.double_security_level || 'ACTIVE_ENFORCED' }}
                    </UBadge>
                  </div>
                  <p class="text-[11px] text-amber-700 dark:text-amber-300 mt-0.5">
                    {{ scanResult.security_audit?.message || 'Semua kunci rahasia, token otentikasi, dan password disensor secara berlapis sebelum dikirim ke LLM.' }}
                  </p>
                </div>
              </div>

              <NuxtLink
                to="/settings?tab=censored_keys"
                class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold bg-white dark:bg-neutral-800 border border-amber-300 dark:border-amber-700 text-amber-800 dark:text-amber-300 hover:bg-amber-100/50 dark:hover:bg-neutral-700 transition-colors shadow-xs"
              >
                <UIcon name="i-lucide-sliders" class="w-3.5 h-3.5" />
                <span>Buka Security Vault Settings</span>
              </NuxtLink>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5 pt-1">
              <div class="p-2.5 rounded-lg bg-white/80 dark:bg-neutral-900/80 border border-amber-200/60 dark:border-amber-900/40">
                <div class="text-[10px] uppercase font-bold text-neutral-400">
                  Total Kunci Tersensor
                </div>
                <div class="text-lg font-bold font-mono text-amber-600 dark:text-amber-400 mt-0.5">
                  {{ scanResult.security_audit?.redacted_count || 0 }} rahasia
                </div>
              </div>
              <div class="p-2.5 rounded-lg bg-white/80 dark:bg-neutral-900/80 border border-amber-200/60 dark:border-amber-900/40">
                <div class="text-[10px] uppercase font-bold text-neutral-400">
                  Match Keyword Khusus (Pass 1)
                </div>
                <div class="text-lg font-bold font-mono text-purple-600 dark:text-purple-400 mt-0.5">
                  {{ scanResult.security_audit?.custom_matches || 0 }} kecocokan
                </div>
              </div>
              <div class="p-2.5 rounded-lg bg-white/80 dark:bg-neutral-900/80 border border-amber-200/60 dark:border-amber-900/40">
                <div class="text-[10px] uppercase font-bold text-neutral-400">
                  Pola Ancaman Bawaan (Pass 2)
                </div>
                <div class="text-lg font-bold font-mono text-sky-600 dark:text-sky-400 mt-0.5">
                  {{ scanResult.security_audit?.built_in_matches || 0 }} dinetralkan
                </div>
              </div>
            </div>
          </div>

          <!-- Redacted Secrets Table -->
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-key-round" class="w-4 h-4 text-amber-500" />
                <span class="text-xs font-bold uppercase tracking-wider text-neutral-800 dark:text-neutral-200">
                  Daftar Token & Kunci yang Dialias / Disensor Pada Komponen Ini
                </span>
              </div>
              <span class="text-[11px] text-neutral-500 font-mono">
                {{ scanResult.security_audit?.records?.length || 0 }} item terdaftar
              </span>
            </div>

            <div v-if="scanResult.security_audit?.records && scanResult.security_audit.records.length > 0" class="overflow-x-auto rounded-lg border border-neutral-200 dark:border-neutral-800">
              <table class="w-full text-left text-xs font-mono">
                <thead class="bg-neutral-50 dark:bg-neutral-800 text-neutral-500 uppercase text-[10px] tracking-wider border-b border-neutral-200 dark:border-neutral-800">
                  <tr>
                    <th class="p-2.5 w-1/4">
                      Alias Terkirim ke LLM
                    </th>
                    <th class="p-2.5 w-1/4">
                      Kategori Kunci
                    </th>
                    <th class="p-2.5 w-1/2">
                      Masking Nilai Asli (Aman)
                    </th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800">
                  <tr
                    v-for="(rec, rIdx) in scanResult.security_audit.records"
                    :key="rIdx"
                    class="hover:bg-neutral-50/60 dark:hover:bg-neutral-800/40 transition-colors"
                  >
                    <td class="p-2.5 font-bold text-amber-600 dark:text-amber-400 whitespace-nowrap">
                      <div class="flex items-center gap-1.5">
                        <span>{{ rec.alias }}</span>
                        <button
                          type="button"
                          class="text-neutral-400 hover:text-amber-600 transition-colors cursor-pointer"
                          title="Copy alias"
                          @click="copyToClipboard(rec.alias, 'Alias copied!')"
                        >
                          <UIcon name="i-lucide-copy" class="w-3 h-3" />
                        </button>
                      </div>
                    </td>
                    <td class="p-2.5 whitespace-nowrap">
                      <UBadge size="xs" color="neutral" variant="subtle">
                        {{ rec.category }}
                      </UBadge>
                    </td>
                    <td class="p-2.5 text-neutral-500 dark:text-neutral-400">
                      <span class="bg-neutral-100 dark:bg-neutral-800 px-2 py-0.5 rounded text-[11px]">
                        {{ rec.masked_preview }}
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div v-else class="p-6 text-center text-xs text-neutral-500 italic bg-neutral-50/50 dark:bg-neutral-800/30 rounded-lg border border-dashed border-neutral-200 dark:border-neutral-800">
              Tidak ada kredensial atau string rahasia terdeteksi pada scan komponen ini (Aman dan bersih).
            </div>
          </div>
        </div>

        <!-- TAB CONTENT 2: STRUKTUR FOLDER -->
        <div v-if="activeScanTab === 'structure'" class="space-y-3">
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-2">
            <div class="text-xs font-bold text-neutral-900 dark:text-white">
              Deskripsi Struktur & Arsitektur Modul:
            </div>
            <p class="text-xs leading-relaxed text-neutral-600 dark:text-neutral-400 whitespace-pre-line">
              {{ scanResult.folder_structure_summary }}
            </p>
          </div>

          <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-900 text-neutral-200 space-y-2">
            <div class="flex items-center justify-between text-xs font-mono text-neutral-400">
              <span>Struktur Direktori Komponen</span>
              <button
                class="text-sky-400 hover:underline cursor-pointer"
                @click="copyToClipboard(scanResult.directory_tree, 'Tree copied!')"
              >
                copy tree
              </button>
            </div>
            <pre class="font-mono text-xs max-h-72 overflow-y-auto leading-relaxed text-sky-300/90 whitespace-pre">{{ scanResult.directory_tree }}</pre>
          </div>
        </div>

        <!-- TAB CONTENT 3: ANALISIS AI -->
        <div v-if="activeScanTab === 'analysis'" class="space-y-3">
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3">
            <MarkdownViewer :content="scanResult.ai_analysis_markdown" />
          </div>
        </div>

        <!-- TAB CONTENT 4: DIRECTORY TREE -->
        <div v-if="activeScanTab === 'tree'" class="space-y-3">
          <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-900 text-neutral-200">
            <div class="flex items-center justify-between text-xs font-mono text-neutral-400 mb-2">
              <span>Directory Tree Snapshot</span>
              <button
                class="text-sky-400 hover:underline cursor-pointer"
                @click="copyToClipboard(scanResult.directory_tree, 'Tree copied!')"
              >
                copy tree
              </button>
            </div>
            <pre class="font-mono text-xs max-h-80 overflow-y-auto leading-relaxed text-neutral-300 whitespace-pre">{{ scanResult.directory_tree }}</pre>
          </div>
        </div>
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
