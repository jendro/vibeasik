<script setup lang="ts">
import type {
  Project,
  ProjectComponent,
  ProjectSynergyMap,
  SynergyFrontendPage,
  SynergyRequestFlow
} from '~/types'

const props = defineProps<{
  project: Project
  portStatuses: Record<string, boolean>
  activePortsCount: number
  totalConfiguredPorts: number
}>()

defineEmits<{
  checkPorts: []
  openAddModal: [role?: string]
  openDetail: [comp: ProjectComponent]
  switchTab: [tab: string]
}>()

const toast = useToast()
const { getProjectSynergyMap, generateProjectSynergyMap } = useApi()

const synergyMap = ref<ProjectSynergyMap | null>(null)
const loading = ref(false)
// "Rangkai AI" is queued on the engine; the page stays usable while it waits and runs.
const synergyJob = useJobTarget('synergy_map', () => ({ projectId: props.project.id }), async () => {
  try {
    const result = await getProjectSynergyMap(props.project.id)
    synergyMap.value = result
    toast.add(result.llm_enriched
      ? {
          title: 'Pemetaan AI Berhasil Dirangkai!',
          description: 'Peta sinergi telah diperkaya dengan penalaran mendalam oleh LLM.',
          icon: 'i-lucide-check-circle',
          color: 'success'
        }
      : {
          title: 'Pemetaan Diperbarui Secara Heuristik',
          description: 'Model AI belum merespon; peta sinergi disusun berdasarkan ekstraksi kode sumber statis.',
          icon: 'i-lucide-info',
          color: 'warning'
        })
  } catch (err: unknown) {
    toast.add({
      title: 'Gagal Memuat Peta Sinergi',
      description: err instanceof Error ? err.message : 'Gagal memuat hasil peta sinergi',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  }
})
const generatingLLM = synergyJob.busy

// View Mode: 'pipeline' (Alur Visual) | 'matrix' (Matriks Audit Tabular) | 'topology' (Peta Relasi DB & API)
const activeView = ref<'pipeline' | 'matrix' | 'topology'>('pipeline')

// Filters
const searchQuery = ref('')
const filterMethod = ref<string>('all')
const filterTable = ref<string>('all')

// Interactive Flow Inspector Modal state
const isInspectorOpen = ref(false)
const inspectingFlow = ref<SynergyRequestFlow | null>(null)
const inspectingPage = ref<{ title: string, page_path: string, file_path: string } | null>(null)
const inspectorActiveTab = ref<'payload' | 'validation' | 'handler' | 'database' | 'response'>('payload')

// Page Collapsible state in visual pipeline
const collapsedPages = ref<Record<string, boolean>>({})

const togglePageCollapse = (pageId: string) => {
  collapsedPages.value[pageId] = !collapsedPages.value[pageId]
}

const collapseAllPages = () => {
  if (!synergyMap.value) return
  for (const page of synergyMap.value.pages) {
    collapsedPages.value[page.id] = true
  }
}

const expandAllPages = () => {
  collapsedPages.value = {}
}

const areAllPagesCollapsed = computed(() => {
  if (!synergyMap.value?.pages?.length) return false
  return synergyMap.value.pages.every(p => collapsedPages.value[p.id])
})

const openInspector = (
  flow: SynergyRequestFlow,
  page: { title: string, page_path: string, file_path: string },
  initialTab: 'payload' | 'validation' | 'handler' | 'database' | 'response' = 'payload'
) => {
  inspectingFlow.value = flow
  inspectingPage.value = page
  inspectorActiveTab.value = initialTab
  isInspectorOpen.value = true
}

const loadSynergyData = async (forceRefresh = false) => {
  loading.value = true
  try {
    if (forceRefresh) {
      synergyMap.value = await generateProjectSynergyMap(props.project.id, false)
      toast.add({
        title: 'Peta Sinergi Diperbarui',
        description: 'Pemetaan rute, file, dan alur database berhasil disinkronkan.',
        icon: 'i-lucide-check-circle',
        color: 'success'
      })
    } else {
      synergyMap.value = await getProjectSynergyMap(props.project.id)
    }
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Gagal memuat peta sinergi'
    toast.add({
      title: 'Gagal Memuat Peta Sinergi',
      description: msg,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}

const handleGenerateWithLLM = () => {
  synergyJob.start()
}

onMounted(() => {
  loadSynergyData(false)
})

// Available Database Tables across the project
const availableTables = computed<string[]>(() => {
  if (!synergyMap.value) return []
  const set = new Set<string>()
  for (const t of synergyMap.value.database_map || []) {
    if (t.table_name) set.add(t.table_name)
  }
  for (const p of synergyMap.value.pages || []) {
    for (const r of p.outbound_requests) {
      for (const tbl of r.database_operation?.target_tables || []) {
        if (tbl) set.add(tbl)
      }
    }
  }
  return Array.from(set).sort()
})

// Filtered Pages for Pipeline Visual & Topology
const filteredPages = computed<SynergyFrontendPage[]>(() => {
  if (!synergyMap.value?.pages) return []
  const q = searchQuery.value.trim().toLowerCase()
  const m = filterMethod.value.toUpperCase()
  const tbl = filterTable.value.toLowerCase()

  return synergyMap.value.pages
    .map((page) => {
      const pageMatch = !q || page.page_path.toLowerCase().includes(q) || page.title.toLowerCase().includes(q)

      const matchingRequests = page.outbound_requests.filter((req) => {
        // Method filter
        if (m !== 'ALL' && req.http_method.toUpperCase() !== m) {
          return false
        }

        // Table filter
        if (tbl !== 'all') {
          const reqTables = req.database_operation?.target_tables || []
          if (!reqTables.some(t => t.toLowerCase() === tbl)) {
            return false
          }
        }

        if (!q) return true

        return (
          req.action_title.toLowerCase().includes(q)
          || req.route_path.toLowerCase().includes(q)
          || req.handler_function.toLowerCase().includes(q)
          || req.http_method.toLowerCase().includes(q)
          || (req.middleware && req.middleware.toLowerCase().includes(q))
          || (req.database_operation?.target_tables && req.database_operation.target_tables.some(t => t.toLowerCase().includes(q)))
          || (req.database_operation?.stored_fields && req.database_operation.stored_fields.some(f => f.toLowerCase().includes(q)))
          || (req.validation_rules && req.validation_rules.some(r => r.toLowerCase().includes(q)))
          || pageMatch
        )
      })

      // When filtering by method or table, only keep pages that actually have matching requests
      if (m !== 'ALL' || tbl !== 'all') {
        if (matchingRequests.length === 0) return null
      } else if (!pageMatch && matchingRequests.length === 0) {
        return null
      }

      return {
        ...page,
        outbound_requests: matchingRequests
      }
    })
    .filter((p): p is SynergyFrontendPage => p !== null)
})

// Matrix Rows for Full-Width Audit Table
interface MatrixRow {
  pagePath: string
  pageTitle: string
  filePath: string
  navigationTargets: string[]
  method: string
  routePath: string
  handler: string
  middleware: string
  payload: string
  validations: string[]
  businessLogic: string[]
  dbEngine: string
  dbTables: string[]
  storedFields: string[]
  statusCode: number
  responseSample: string
  uiImpact: string
  rawFlow?: SynergyRequestFlow
  rawPage: { title: string, page_path: string, file_path: string }
}

const matrixRows = computed<MatrixRow[]>(() => {
  const rows: MatrixRow[] = []
  for (const p of filteredPages.value) {
    const navTextList = p.navigation_targets.map(n => `${n.target_path} (${n.target_label})`)
    const pageObj = { title: p.title, page_path: p.page_path, file_path: p.file_path }

    if (p.outbound_requests.length === 0) {
      rows.push({
        pagePath: p.page_path,
        pageTitle: p.title,
        filePath: p.file_path,
        navigationTargets: navTextList,
        method: 'NONE',
        routePath: 'Tanpa request API langsung',
        handler: '-',
        middleware: '-',
        payload: '-',
        validations: ['-'],
        businessLogic: ['Halaman statis / presentasi UI'],
        dbEngine: '-',
        dbTables: [],
        storedFields: [],
        statusCode: 200,
        responseSample: '-',
        uiImpact: 'Render antarmuka pengguna',
        rawPage: pageObj
      })
    } else {
      for (const req of p.outbound_requests) {
        rows.push({
          pagePath: p.page_path,
          pageTitle: p.title,
          filePath: p.file_path,
          navigationTargets: navTextList,
          method: req.http_method,
          routePath: req.route_path,
          handler: req.handler_function,
          middleware: req.middleware,
          payload: [req.request_payload?.params, req.request_payload?.body].filter(Boolean).join(' | ') || '-',
          validations: req.validation_rules || [],
          businessLogic: req.business_process || [],
          dbEngine: req.database_operation?.engine || 'SQLite',
          dbTables: req.database_operation?.target_tables || [],
          storedFields: req.database_operation?.stored_fields || [],
          statusCode: req.response_spec?.status_code || 200,
          responseSample: req.response_spec?.body_sample || '-',
          uiImpact: req.ui_impact || 'Perbarui state reaktif UI',
          rawFlow: req,
          rawPage: pageObj
        })
      }
    }
  }
  return rows
})

const resetFilters = () => {
  searchQuery.value = ''
  filterMethod.value = 'all'
  filterTable.value = 'all'
}

const copySummaryMarkdown = async () => {
  if (!synergyMap.value) return

  let md = `# Peta Sinergi Proyek: ${synergyMap.value.project_name}\n\n`
  md += `**Ringkasan**: ${synergyMap.value.summary}\n`
  md += `**Total Halaman**: ${synergyMap.value.total_pages} | **Total Alur**: ${synergyMap.value.total_flows} | **Tabel DB**: ${synergyMap.value.total_tables}\n\n`

  for (const p of synergyMap.value.pages) {
    md += `## 📱 Halaman: ${p.title} (\`${p.page_path}\`)\n`
    md += `- **File**: \`${p.file_path || 'Unknown'}\`\n`
    if (p.navigation_targets.length > 0) {
      md += `- **Navigasi Tujuan**:\n`
      for (const n of p.navigation_targets) {
        md += `  - ➔ \`${n.target_path}\`: ${n.target_label} (${n.trigger})\n`
      }
    }
    if (p.outbound_requests.length > 0) {
      md += `\n### Alur Request API:\n`
      for (const r of p.outbound_requests) {
        md += `#### [${r.http_method}] \`${r.route_path}\` - ${r.action_title}\n`
        md += `- **Pemicu**: ${r.trigger_event}\n`
        md += `- **Handler BE**: \`${r.handler_function}\` (${r.middleware})\n`
        md += `- **Data Dikirim**: ${r.request_payload?.params || ''} ${r.request_payload?.body || ''}\n`
        md += `- **Validasi**: ${r.validation_rules?.join('; ') || '-'}\n`
        md += `- **Proses**: ${r.business_process?.join(' ➔ ') || '-'}\n`
        md += `- **Database**: Tabel \`${r.database_operation?.target_tables?.join(', ')}\` | Kolom tersimpan: \`${r.database_operation?.stored_fields?.join(', ')}\`\n`
        md += `- **Response**: HTTP ${r.response_spec?.status_code} (${r.response_spec?.data_type})\n`
        md += `- **UI Impact**: ${r.ui_impact}\n\n`
      }
    }
    md += `---\n\n`
  }

  try {
    await navigator.clipboard.writeText(md)
    toast.add({
      title: 'Pemetaan Disalin ke Clipboard',
      description: 'Format Markdown lengkap berhasil disalin.',
      icon: 'i-lucide-clipboard-check',
      color: 'success'
    })
  } catch {
    toast.add({
      title: 'Gagal Menyalin',
      description: 'Silakan salin teks secara manual.',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  }
}

const downloadSynergyJson = () => {
  if (!synergyMap.value) return
  const dataStr = 'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(synergyMap.value, null, 2))
  const downloadAnchor = document.createElement('a')
  downloadAnchor.setAttribute('href', dataStr)
  downloadAnchor.setAttribute('download', `synergy-map-${props.project.slug || 'project'}.json`)
  document.body.appendChild(downloadAnchor)
  downloadAnchor.click()
  downloadAnchor.remove()
  toast.add({
    title: 'JSON Sinergi Berhasil Diunduh',
    description: `File synergy-map-${props.project.slug || 'project'}.json tersimpan.`,
    icon: 'i-lucide-download',
    color: 'success'
  })
}

const copyCurlCommand = async (req: SynergyRequestFlow) => {
  const method = req.http_method.toUpperCase()
  const port = props.project.components?.find((c: ProjectComponent) => c.role?.toLowerCase() === 'backend')?.port || 8080
  let cmd = `curl -X ${method} "http://localhost:${port}${req.route_path}"`
  if (req.request_payload?.headers) {
    for (const h of req.request_payload.headers.split(',')) {
      if (h.trim()) cmd += ` \\\n  -H "${h.trim()}"`
    }
  } else {
    cmd += ` \\\n  -H "Content-Type: application/json"`
  }
  if (req.request_payload?.body && method !== 'GET') {
    cmd += ` \\\n  -d '${req.request_payload.body}'`
  }

  try {
    await navigator.clipboard.writeText(cmd)
    toast.add({
      title: 'cURL Disalin!',
      description: `Perintah cURL untuk ${req.route_path} siap dijalankan di terminal.`,
      icon: 'i-lucide-terminal',
      color: 'success'
    })
  } catch {
    // fallback
  }
}

const getMethodColorClass = (method: string) => {
  const m = method.toUpperCase()
  if (m.startsWith('GET')) return 'bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-400 border-emerald-200 dark:border-emerald-800'
  if (m.startsWith('POST')) return 'bg-sky-50 dark:bg-sky-950/60 text-sky-700 dark:text-sky-400 border-sky-200 dark:border-sky-800'
  if (m.startsWith('PUT') || m.startsWith('PATCH')) return 'bg-amber-50 dark:bg-amber-950/60 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800'
  if (m.startsWith('DELETE')) return 'bg-rose-50 dark:bg-rose-950/60 text-rose-700 dark:text-rose-400 border-rose-200 dark:border-rose-800'
  return 'bg-purple-50 dark:bg-purple-950/60 text-purple-700 dark:text-purple-400 border-purple-200 dark:border-purple-800'
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header Hero Sinergi Proyek -->
    <div class="p-5 sm:p-6 rounded-2xl border border-neutral-200/80 dark:border-neutral-800 bg-white dark:bg-slate-900 shadow-xs space-y-4">
      <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
        <div class="space-y-1">
          <div class="flex items-center gap-2.5">
            <div class="p-2 rounded-xl bg-sky-500/10 text-sky-600 dark:text-sky-400 border border-sky-500/20">
              <UIcon name="i-lucide-git-fork" class="w-5 h-5" />
            </div>
            <div>
              <h2 class="text-base sm:text-lg font-bold text-neutral-900 dark:text-white flex items-center gap-2">
                <span>Peta Sinergi & Integrasi Sistem</span>
                <UBadge
                  v-if="synergyMap?.llm_enriched"
                  size="xs"
                  color="primary"
                  variant="subtle"
                  class="font-mono text-[10px] flex items-center gap-1"
                >
                  <UIcon name="i-lucide-sparkles" class="w-3 h-3" />
                  <span>AI Enriched</span>
                </UBadge>
              </h2>
              <p class="text-xs text-neutral-500 mt-0.5 max-w-3xl leading-relaxed">
                Pemetaan end-to-end terhubung: Halaman Frontend ➔ Navigasi ➔ Endpoint API ➔ Router & Controller ➔ Basis Data & Format Respon.
              </p>
            </div>
          </div>
        </div>

        <!-- Action Controls & Export Buttons -->
        <div class="flex flex-wrap items-center gap-2 shrink-0">
          <!-- Rangkai dengan AI (LLM) -->
          <UButton
            icon="i-lucide-sparkles"
            color="primary"
            variant="solid"
            size="xs"
            :loading="generatingLLM"
            title="Minta AI (LLM) menganalisis dan merangkai alur sinergi secara komprehensif"
            class="cursor-pointer"
            @click="handleGenerateWithLLM"
          >
            <span>{{ generatingLLM ? synergyJob.label.value : 'Rangkai AI' }}</span>
          </UButton>

          <!-- Segarkan Data Source -->
          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="outline"
            size="xs"
            :loading="loading"
            title="Sinkronkan ulang rute dan file source code"
            class="cursor-pointer"
            @click="loadSynergyData(true)"
          >
            <span>Segarkan</span>
          </UButton>

          <!-- Unduh JSON -->
          <UButton
            icon="i-lucide-download"
            color="neutral"
            variant="outline"
            size="xs"
            :disabled="!synergyMap"
            title="Unduh data sinergi lengkap dalam format JSON"
            class="cursor-pointer"
            @click="downloadSynergyJson"
          >
            <span class="hidden sm:inline">Unduh JSON</span>
          </UButton>

          <!-- Salin Markdown -->
          <UButton
            icon="i-lucide-clipboard"
            color="neutral"
            variant="ghost"
            size="xs"
            :disabled="!synergyMap"
            title="Salin pemetaan lengkap ke format Markdown"
            class="cursor-pointer"
            @click="copySummaryMarkdown"
          />
        </div>
      </div>

      <!-- Quick Metrics Grid -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 pt-3 border-t border-neutral-100 dark:border-neutral-800/80 text-xs">
        <div class="p-3 rounded-xl bg-neutral-50/70 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800/60">
          <span class="text-[10px] text-neutral-400 uppercase font-semibold">Halaman Frontend</span>
          <div class="font-bold text-neutral-800 dark:text-neutral-200 font-mono text-base mt-0.5 tabular-nums">
            {{ synergyMap?.total_pages || 0 }} halaman
          </div>
        </div>
        <div class="p-3 rounded-xl bg-neutral-50/70 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800/60">
          <span class="text-[10px] text-neutral-400 uppercase font-semibold">Alur Request API</span>
          <div class="font-bold font-mono text-base mt-0.5 text-sky-600 dark:text-sky-400 tabular-nums">
            {{ synergyMap?.total_flows || 0 }} alur terhubung
          </div>
        </div>
        <div class="p-3 rounded-xl bg-neutral-50/70 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800/60">
          <span class="text-[10px] text-neutral-400 uppercase font-semibold">Node Backend</span>
          <div class="font-bold text-neutral-800 dark:text-neutral-200 font-mono text-base mt-0.5 tabular-nums">
            {{ synergyMap?.backend_nodes?.length || 0 }} servis API
          </div>
        </div>
        <div class="p-3 rounded-xl bg-neutral-50/70 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800/60">
          <span class="text-[10px] text-neutral-400 uppercase font-semibold">Tabel Database</span>
          <div class="font-bold text-neutral-800 dark:text-neutral-200 font-mono text-base mt-0.5 tabular-nums">
            {{ synergyMap?.total_tables || 0 }} tabel
          </div>
        </div>
      </div>
    </div>

    <!-- Tampilan Mode Switcher (View Switcher Tab Bar) -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-1">
      <!-- 3 Pilihan Tampilan Utama: Pipeline Visual, Matriks Audit, Peta Dependensi -->
      <div class="flex items-center gap-1.5 p-1 rounded-xl bg-neutral-100 dark:bg-neutral-800/60 border border-neutral-200/60 dark:border-neutral-700/60">
        <button
          type="button"
          class="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
          :class="activeView === 'pipeline'
            ? 'bg-white dark:bg-slate-900 text-sky-600 dark:text-sky-400 shadow-2xs font-bold'
            : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
          @click="activeView = 'pipeline'"
        >
          <UIcon name="i-lucide-git-fork" class="w-4 h-4" />
          <span>Pipeline Visual</span>
        </button>

        <button
          type="button"
          class="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
          :class="activeView === 'matrix'
            ? 'bg-white dark:bg-slate-900 text-sky-600 dark:text-sky-400 shadow-2xs font-bold'
            : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
          @click="activeView = 'matrix'"
        >
          <UIcon name="i-lucide-table-properties" class="w-4 h-4" />
          <span>Matriks Audit Tabular</span>
          <UBadge size="xs" color="neutral" variant="subtle" class="font-mono text-[9px] px-1 py-0">
            {{ matrixRows.length }}
          </UBadge>
        </button>

        <button
          type="button"
          class="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
          :class="activeView === 'topology'
            ? 'bg-white dark:bg-slate-900 text-sky-600 dark:text-sky-400 shadow-2xs font-bold'
            : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
          @click="activeView = 'topology'"
        >
          <UIcon name="i-lucide-database" class="w-4 h-4" />
          <span>Relasi DB & API</span>
        </button>
      </div>

      <!-- Quick Summary Status -->
      <div class="text-xs text-neutral-400 flex items-center gap-2">
        <span>Menampilkan {{ filteredPages.length }} halaman</span>
        <span v-if="searchQuery || filterMethod !== 'all' || filterTable !== 'all'" class="text-sky-500 font-semibold">
          (Difilter)
        </span>
      </div>
    </div>

    <!-- Filter & Search Bar -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-3 p-3.5 rounded-xl border border-neutral-200/80 dark:border-neutral-800 bg-white dark:bg-slate-900">
      <div class="flex flex-wrap items-center gap-2 flex-1">
        <!-- Search Input -->
        <UInput
          v-model="searchQuery"
          icon="i-lucide-search"
          placeholder="Cari halaman, rute API, handler, tabel DB..."
          size="xs"
          class="w-full sm:w-64"
        />

        <!-- Method Selector -->
        <select
          v-model="filterMethod"
          aria-label="Filter Metode HTTP"
          class="text-xs rounded-lg px-2.5 py-1.5 border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-slate-800 text-neutral-700 dark:text-neutral-200 focus:outline-hidden cursor-pointer"
        >
          <option value="all">
            Semua Metode HTTP
          </option>
          <option value="GET">
            GET
          </option>
          <option value="POST">
            POST
          </option>
          <option value="PUT">
            PUT / PATCH
          </option>
          <option value="DELETE">
            DELETE
          </option>
        </select>

        <!-- Table Selector -->
        <select
          v-if="availableTables.length > 0"
          v-model="filterTable"
          aria-label="Filter Tabel Database"
          class="text-xs rounded-lg px-2.5 py-1.5 border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-slate-800 text-neutral-700 dark:text-neutral-200 focus:outline-hidden cursor-pointer"
        >
          <option value="all">
            Semua Tabel Database
          </option>
          <option v-for="t in availableTables" :key="t" :value="t">
            Tabel: {{ t }}
          </option>
        </select>

        <!-- Reset Button -->
        <UButton
          v-if="searchQuery || filterMethod !== 'all' || filterTable !== 'all'"
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-lucide-x"
          title="Reset filter pencarian"
          class="cursor-pointer"
          @click="resetFilters"
        >
          Reset
        </UButton>
      </div>

      <!-- Expand / Collapse Controls (Only relevant in Pipeline View) -->
      <div v-if="activeView === 'pipeline'" class="flex items-center gap-2 shrink-0">
        <UButton
          size="xs"
          color="neutral"
          variant="outline"
          :icon="areAllPagesCollapsed ? 'i-lucide-chevrons-down' : 'i-lucide-chevrons-up'"
          title="Ciutkan atau bentangkan seluruh modul halaman"
          class="cursor-pointer"
          @click="areAllPagesCollapsed ? expandAllPages() : collapseAllPages()"
        >
          {{ areAllPagesCollapsed ? 'Bentangkan Semua' : 'Ciutkan Semua' }}
        </UButton>
      </div>
    </div>

    <!-- Loading Skeleton -->
    <div v-if="loading && !synergyMap" class="space-y-4">
      <div v-for="i in 3" :key="i" class="h-44 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-neutral-100 dark:bg-neutral-800/40 animate-pulse" />
    </div>

    <!-- EMPTY STATE -->
    <div
      v-else-if="!filteredPages || filteredPages.length === 0"
      class="p-12 text-center rounded-2xl border border-dashed border-neutral-200 dark:border-neutral-800 bg-white dark:bg-slate-900/40 space-y-3"
    >
      <UIcon name="i-lucide-search-x" class="w-10 h-10 text-neutral-400 mx-auto" />
      <div class="text-sm font-semibold text-neutral-700 dark:text-neutral-300">
        Tidak ada data halaman atau alur yang cocok dengan kriteria filter
      </div>
      <p class="text-xs text-neutral-500 max-w-sm mx-auto">
        Ubah kata kunci pencarian, reset filter metode, atau klik "Segarkan" untuk memperbarui data dari kode sumber.
      </p>
      <div class="pt-2">
        <UButton size="xs" color="primary" variant="soft" class="cursor-pointer" @click="resetFilters">
          Hapus Semua Filter
        </UButton>
      </div>
    </div>

    <!-- VIEW 1: PIPELINE VISUAL (ALUR KARTU HALAMAN PER MODUL) -->
    <div v-else-if="activeView === 'pipeline'" class="space-y-4">
      <div
        v-for="p in filteredPages"
        :key="p.id"
        class="rounded-2xl border border-neutral-200/80 dark:border-neutral-800 bg-white dark:bg-slate-900 shadow-xs overflow-hidden transition-all duration-200"
      >
        <!-- Page Node Header (Accordion Toggle) -->
        <div
          class="p-4 sm:p-4.5 bg-neutral-50/70 dark:bg-neutral-800/40 border-b border-neutral-200/60 dark:border-neutral-800 flex flex-col sm:flex-row sm:items-center justify-between gap-3 cursor-pointer hover:bg-neutral-100/60 dark:hover:bg-neutral-800/60 transition-colors"
          @click="togglePageCollapse(p.id)"
        >
          <div class="flex items-center gap-3 min-w-0">
            <div class="p-2 rounded-xl bg-sky-500 text-white shrink-0 shadow-2xs">
              <UIcon name="i-lucide-monitor" class="w-4 h-4" />
            </div>
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="text-xs font-bold text-neutral-900 dark:text-white truncate">
                  {{ p.title }}
                </span>
                <span class="font-mono text-[11px] text-sky-600 dark:text-sky-400 font-semibold truncate">
                  {{ p.page_path }}
                </span>
              </div>
              <p v-if="p.file_path" class="text-[10px] font-mono text-neutral-400 truncate mt-0.5">
                {{ p.file_path }}
              </p>
            </div>
          </div>

          <div class="flex items-center gap-2 shrink-0">
            <!-- Navigation Destinations Count -->
            <UBadge size="xs" color="success" variant="subtle" class="font-mono text-[10px]">
              {{ p.navigation_targets.length }} Navigasi
            </UBadge>
            <!-- Outbound API Count -->
            <UBadge size="xs" color="primary" variant="subtle" class="font-mono text-[10px]">
              {{ p.outbound_requests.length }} API
            </UBadge>
            <!-- Toggle Chevron Icon -->
            <UIcon
              :name="collapsedPages[p.id] ? 'i-lucide-chevron-down' : 'i-lucide-chevron-up'"
              class="w-4 h-4 text-neutral-400"
            />
          </div>
        </div>

        <!-- Expanded Page Pipeline Body -->
        <div v-if="!collapsedPages[p.id]" class="p-4 sm:p-5 space-y-4">
          <!-- Peta Navigasi Antar Halaman -->
          <div class="p-3.5 rounded-xl bg-neutral-50/60 dark:bg-slate-950/40 border border-neutral-200/60 dark:border-neutral-800 space-y-2">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-1.5 text-xs font-bold text-neutral-700 dark:text-neutral-300">
                <UIcon name="i-lucide-compass" class="w-3.5 h-3.5 text-emerald-500" />
                <span>Peta Navigasi Keluar (Dari Halaman Ini Pengguna Bisa Berpindah Ke):</span>
              </div>
            </div>

            <div v-if="p.navigation_targets.length > 0" class="flex flex-wrap gap-2 pt-0.5">
              <div
                v-for="(nav, nIdx) in p.navigation_targets"
                :key="nIdx"
                class="px-2.5 py-1.5 rounded-lg bg-white dark:bg-slate-900 border border-neutral-200/70 dark:border-neutral-800 text-xs shadow-2xs space-y-0.5"
              >
                <div class="flex items-center gap-1 font-semibold text-neutral-800 dark:text-neutral-200 text-[11px]">
                  <UIcon name="i-lucide-arrow-right" class="w-3 h-3 text-emerald-500 shrink-0" />
                  <span class="font-mono text-emerald-700 dark:text-emerald-400">{{ nav.target_path }}</span>
                  <span class="text-neutral-500 font-normal">({{ nav.target_label }})</span>
                </div>
                <div class="text-[9px] text-neutral-400 pl-4">
                  {{ nav.trigger }}
                </div>
              </div>
            </div>
            <div v-else class="text-[11px] text-neutral-400 italic">
              Tidak ada navigasi keluar langsung dari halaman ini.
            </div>
          </div>

          <!-- Outbound API Branches -->
          <div class="space-y-3">
            <div class="flex items-center justify-between text-xs font-bold text-neutral-700 dark:text-neutral-300">
              <span class="flex items-center gap-1.5">
                <UIcon name="i-lucide-workflow" class="w-3.5 h-3.5 text-sky-500" />
                <span>Alur Cabang API (Klik Setiap Kotak Node Untuk Rincian Lengkap & cURL):</span>
              </span>
              <span class="text-[10px] text-neutral-400 font-mono">
                {{ p.outbound_requests.length }} alur request
              </span>
            </div>

            <div
              v-if="p.outbound_requests.length === 0"
              class="p-4 rounded-xl border border-dashed border-neutral-200 dark:border-neutral-800 text-xs text-neutral-400 text-center"
            >
              Halaman statis / presentasi UI (tidak memanggil endpoint API langsung).
            </div>

            <!-- Flow Cards -->
            <div
              v-for="r in p.outbound_requests"
              :key="r.id"
              class="p-4 rounded-xl bg-white dark:bg-slate-900 border border-neutral-200/80 dark:border-neutral-800 space-y-3 shadow-2xs hover:border-sky-400/50 transition-colors"
            >
              <!-- Request Header Bar -->
              <div class="flex flex-wrap items-center justify-between gap-2 pb-2 border-b border-neutral-100 dark:border-neutral-800/70">
                <div class="flex items-center gap-2">
                  <span
                    class="px-2 py-0.5 rounded text-[10px] font-bold font-mono border uppercase shrink-0"
                    :class="getMethodColorClass(r.http_method)"
                  >
                    {{ r.http_method }}
                  </span>
                  <span class="font-mono text-xs font-bold text-neutral-900 dark:text-neutral-100">
                    {{ r.route_path }}
                  </span>
                  <span class="text-xs text-neutral-500 hidden sm:inline">
                    • {{ r.action_title }}
                  </span>
                </div>
                <div class="flex items-center gap-2">
                  <span class="text-[10px] font-mono text-neutral-400 hidden md:inline">
                    Pemicu: {{ r.trigger_event }}
                  </span>
                  <!-- Tombol Buka Inspector Detail -->
                  <UButton
                    size="xs"
                    color="primary"
                    variant="soft"
                    icon="i-lucide-maximize-2"
                    title="Buka rincian lengkap alur & cURL"
                    class="cursor-pointer font-semibold text-[11px]"
                    @click="openInspector(r, { title: p.title, page_path: p.page_path, file_path: p.file_path }, 'payload')"
                  >
                    Detail Alur
                  </UButton>
                </div>
              </div>

              <!-- High-Performance Interactive Stage Pills (Click to Inspect) -->
              <div class="flex flex-wrap items-center gap-2 text-xs">
                <!-- Step 1: Payload Pill -->
                <button
                  type="button"
                  class="px-2.5 py-1.5 rounded-lg bg-sky-50/70 dark:bg-sky-950/40 text-sky-800 dark:text-sky-300 border border-sky-200 dark:border-sky-800/60 font-mono text-[11px] cursor-pointer hover:bg-sky-100 dark:hover:bg-sky-900/60 transition-colors flex items-center gap-1.5 shadow-2xs"
                  title="Klik untuk membuka detail payload"
                  @click="openInspector(r, { title: p.title, page_path: p.page_path, file_path: p.file_path }, 'payload')"
                >
                  <UIcon name="i-lucide-file-code" class="w-3.5 h-3.5 text-sky-500" />
                  <span>📤 Payload: {{ r.request_payload?.params || 'Data Kirim' }}</span>
                </button>

                <span class="text-neutral-400 text-xs">➔</span>

                <!-- Step 2: Validasi Pill -->
                <button
                  type="button"
                  class="px-2.5 py-1.5 rounded-lg bg-indigo-50/70 dark:bg-indigo-950/40 text-indigo-800 dark:text-indigo-300 border border-indigo-200 dark:border-indigo-800/60 font-mono text-[11px] cursor-pointer hover:bg-indigo-100 dark:hover:bg-indigo-900/60 transition-colors flex items-center gap-1.5 shadow-2xs"
                  title="Klik untuk membuka detail validasi"
                  @click="openInspector(r, { title: p.title, page_path: p.page_path, file_path: p.file_path }, 'validation')"
                >
                  <UIcon name="i-lucide-shield-check" class="w-3.5 h-3.5 text-indigo-500" />
                  <span>🛡️ Validasi: {{ r.validation_rules?.[0] ? `${r.validation_rules.length} aturan` : 'Validasi Schema' }}</span>
                </button>

                <span class="text-neutral-400 text-xs">➔</span>

                <!-- Step 3: Handler Pill -->
                <button
                  type="button"
                  class="px-2.5 py-1.5 rounded-lg bg-purple-50/70 dark:bg-purple-950/40 text-purple-800 dark:text-purple-300 border border-purple-200 dark:border-purple-800/60 font-mono text-[11px] cursor-pointer hover:bg-purple-100 dark:hover:bg-purple-900/60 transition-colors flex items-center gap-1.5 shadow-2xs"
                  title="Klik untuk membuka detail router & handler backend"
                  @click="openInspector(r, { title: p.title, page_path: p.page_path, file_path: p.file_path }, 'handler')"
                >
                  <UIcon name="i-lucide-server" class="w-3.5 h-3.5 text-purple-500" />
                  <span>⚡ Handler: {{ r.handler_function }}</span>
                </button>

                <span class="text-neutral-400 text-xs">➔</span>

                <!-- Step 4: Database Pill -->
                <button
                  type="button"
                  class="px-2.5 py-1.5 rounded-lg bg-amber-50/70 dark:bg-amber-950/40 text-amber-800 dark:text-amber-300 border border-amber-200 dark:border-amber-800/60 font-mono text-[11px] cursor-pointer hover:bg-amber-100 dark:hover:bg-amber-900/60 transition-colors flex items-center gap-1.5 shadow-2xs"
                  title="Klik untuk membuka detail query database"
                  @click="openInspector(r, { title: p.title, page_path: p.page_path, file_path: p.file_path }, 'database')"
                >
                  <UIcon name="i-lucide-database" class="w-3.5 h-3.5 text-amber-500" />
                  <span>🗄️ Tabel DB: {{ r.database_operation?.target_tables?.join(', ') || 'DB' }}</span>
                </button>

                <span class="text-neutral-400 text-xs">➔</span>

                <!-- Step 5: Respon Pill -->
                <button
                  type="button"
                  class="px-2.5 py-1.5 rounded-lg bg-emerald-50/70 dark:bg-emerald-950/40 text-emerald-800 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800/60 font-mono text-[11px] cursor-pointer hover:bg-emerald-100 dark:hover:bg-emerald-900/60 transition-colors flex items-center gap-1.5 shadow-2xs"
                  title="Klik untuk membuka format respon & dampak UI"
                  @click="openInspector(r, { title: p.title, page_path: p.page_path, file_path: p.file_path }, 'response')"
                >
                  <UIcon name="i-lucide-check-circle-2" class="w-3.5 h-3.5 text-emerald-500" />
                  <span>📥 Respon: HTTP {{ r.response_spec?.status_code || 200 }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- VIEW 2: MATRIKS AUDIT TABULAR (FULL-WIDTH DATA TABLE DENGAN FITUR LENGKAP) -->
    <div v-else-if="activeView === 'matrix'" class="space-y-4">
      <div class="rounded-2xl border border-neutral-200/80 dark:border-neutral-800 bg-white dark:bg-slate-900 shadow-xs overflow-hidden">
        <div class="p-3.5 sm:p-4 bg-neutral-50/60 dark:bg-neutral-800/40 border-b border-neutral-200/70 dark:border-neutral-800 flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-table-properties" class="w-4 h-4 text-sky-500" />
            <span class="text-xs font-bold text-neutral-800 dark:text-neutral-200">
              Matriks Audit Menyeluruh ({{ matrixRows.length }} Baris Terhubung)
            </span>
          </div>
          <div class="text-[11px] text-neutral-400 font-mono">
            Klik tombol "Periksa" pada cell Halaman FE untuk inspeksi alur & cURL
          </div>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs">
            <thead class="bg-neutral-50 dark:bg-neutral-800/80 text-neutral-500 dark:text-neutral-400 font-mono text-[11px] border-b border-neutral-200 dark:border-neutral-800">
              <tr>
                <th class="py-2.5 px-3 min-w-[200px]">
                  Halaman FE & Navigasi
                </th>
                <th class="py-2.5 px-3 min-w-[200px]">
                  Endpoint API & Handler
                </th>
                <th class="py-2.5 px-3 min-w-[160px]">
                  Data Dikirim & Validasi
                </th>
                <th class="py-2.5 px-3 min-w-[150px]">
                  Tabel DB & Kolom
                </th>
                <th class="py-2.5 px-3 min-w-[160px]">
                  Respon & UI Impact
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/80">
              <tr
                v-for="(row, rIdx) in matrixRows"
                :key="rIdx"
                class="hover:bg-neutral-50/70 dark:hover:bg-neutral-800/40 transition-colors"
              >
                <!-- Halaman FE & Navigasi + Tombol Periksa -->
                <td class="py-2.5 px-3 align-top space-y-1.5">
                  <div class="flex items-start justify-between gap-2">
                    <div class="min-w-0 flex-1">
                      <div class="font-bold text-neutral-900 dark:text-white text-xs leading-snug" :title="row.pageTitle">
                        {{ row.pageTitle }}
                      </div>
                      <div class="font-mono text-[11px] text-sky-600 dark:text-sky-400 truncate mt-0.5" :title="row.pagePath">
                        {{ row.pagePath }}
                      </div>
                    </div>
                    <UButton
                      v-if="row.rawFlow"
                      size="xs"
                      color="primary"
                      variant="soft"
                      icon="i-lucide-search"
                      class="cursor-pointer font-semibold text-[10px] px-2 py-0.5 shrink-0"
                      @click="openInspector(row.rawFlow, row.rawPage, 'payload')"
                    >
                      Periksa
                    </UButton>
                  </div>
                  <div v-if="row.filePath" class="text-[10px] font-mono text-neutral-400 truncate max-w-[220px]" :title="row.filePath">
                    {{ row.filePath }}
                  </div>
                  <div v-if="row.navigationTargets.length > 0" class="pt-0.5 text-[10px] text-neutral-500 space-y-0.5">
                    <div v-for="nt in row.navigationTargets.slice(0, 2)" :key="nt" class="truncate font-mono" :title="nt">
                      ➔ {{ nt }}
                    </div>
                  </div>
                </td>

                <!-- Endpoint API & Handler (Gabungan Metode, Rute, Handler & Middleware) -->
                <td class="py-2.5 px-3 align-top space-y-1">
                  <div class="flex items-center gap-1.5 flex-wrap">
                    <span
                      class="px-1.5 py-0.5 rounded text-[10px] font-bold font-mono border uppercase shrink-0"
                      :class="getMethodColorClass(row.method)"
                    >
                      {{ row.method }}
                    </span>
                    <span class="font-mono font-bold text-neutral-800 dark:text-neutral-200 text-[11px] break-all">
                      {{ row.routePath }}
                    </span>
                  </div>
                  <div class="font-mono text-[11px] text-neutral-700 dark:text-neutral-300 flex items-center gap-1.5 flex-wrap pt-0.5">
                    <span class="font-semibold text-neutral-800 dark:text-neutral-200">{{ row.handler }}</span>
                    <span
                      v-if="row.middleware && row.middleware !== '-'"
                      class="text-[10px] text-neutral-500 dark:text-neutral-400 font-sans px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700/60"
                    >
                      {{ row.middleware }}
                    </span>
                  </div>
                </td>

                <!-- Data Dikirim & Validasi -->
                <td class="py-2.5 px-3 align-top space-y-1">
                  <div class="font-mono text-[10px] text-neutral-700 dark:text-neutral-300 line-clamp-2" :title="row.payload">
                    {{ row.payload }}
                  </div>
                  <div class="text-[10px] text-neutral-500 line-clamp-2" :title="row.validations.join('; ')">
                    {{ row.validations[0] || '-' }}
                  </div>
                </td>

                <!-- Tabel DB & Kolom -->
                <td class="py-2.5 px-3 align-top space-y-1">
                  <div class="font-bold font-mono text-amber-700 dark:text-amber-400 text-[11px] break-words">
                    {{ row.dbTables.join(', ') || '-' }}
                  </div>
                  <div v-if="row.storedFields.length > 0" class="text-[10px] text-neutral-400 font-mono line-clamp-2" :title="row.storedFields.join(', ')">
                    Kolom: {{ row.storedFields.join(', ') }}
                  </div>
                </td>

                <!-- Respon & UI Impact -->
                <td class="py-2.5 px-3 align-top space-y-1">
                  <span class="px-1.5 py-0.5 rounded text-[10px] font-bold font-mono bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800 inline-block">
                    HTTP {{ row.statusCode }}
                  </span>
                  <p class="text-[11px] text-neutral-600 dark:text-neutral-400 line-clamp-2 pt-0.5 font-medium" :title="row.uiImpact">
                    {{ row.uiImpact }}
                  </p>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- VIEW 3: PETA RELASI & DEPENDENSI (TOPOLOGY OVERVIEW) -->
    <div v-else-if="activeView === 'topology'" class="space-y-5">
      <!-- Backend Nodes Summary -->
      <div class="p-5 rounded-2xl border border-neutral-200/80 dark:border-neutral-800 bg-white dark:bg-slate-900 shadow-xs space-y-3">
        <h3 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-server" class="w-4 h-4 text-purple-500" />
          <span>Servis & Node Komponen Backend</span>
        </h3>
        <p class="text-xs text-neutral-500">
          Daftar server backend yang melayani request API dan berinteraksi langsung dengan basis data.
        </p>

        <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3 pt-1">
          <div
            v-for="be in synergyMap?.backend_nodes || []"
            :key="be.component_name"
            class="p-4 rounded-xl bg-neutral-50/70 dark:bg-neutral-800/40 border border-neutral-200/70 dark:border-neutral-800 space-y-2"
          >
            <div class="flex items-center justify-between">
              <span class="font-bold text-xs text-neutral-900 dark:text-white">{{ be.component_name }}</span>
              <UBadge size="xs" color="primary" variant="subtle" class="font-mono text-[10px]">
                Port {{ be.port }}
              </UBadge>
            </div>
            <div class="text-[11px] text-neutral-500 flex items-center justify-between">
              <span>Framework:</span>
              <span class="font-semibold text-neutral-800 dark:text-neutral-200 font-mono">{{ be.framework || 'Go / Fiber' }}</span>
            </div>
            <div class="text-[11px] text-neutral-500 flex items-center justify-between">
              <span>Total Endpoint:</span>
              <span class="font-bold text-sky-600 dark:text-sky-400 font-mono">{{ be.routes_count }} rute</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Database Map Grid -->
      <div class="p-5 rounded-2xl border border-neutral-200/80 dark:border-neutral-800 bg-white dark:bg-slate-900 shadow-xs space-y-3">
        <h3 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-database" class="w-4 h-4 text-amber-500" />
          <span>Katalog Tabel Basis Data & Kolom Kunci</span>
        </h3>
        <p class="text-xs text-neutral-500">
          Entitas database yang menyimpan data hasil proses transaksi API dari seluruh aplikasi bundle.
        </p>

        <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3 pt-1">
          <div
            v-for="tbl in synergyMap?.database_map || []"
            :key="tbl.table_name"
            class="p-4 rounded-xl bg-neutral-50/70 dark:bg-neutral-800/40 border border-neutral-200/70 dark:border-neutral-800 space-y-2"
          >
            <div class="flex items-center justify-between pb-1 border-b border-neutral-200/60 dark:border-neutral-700/60">
              <span class="font-mono font-bold text-xs text-amber-700 dark:text-amber-400">
                {{ tbl.table_name }}
              </span>
              <span class="text-[10px] text-neutral-400 font-mono">
                {{ tbl.fields?.length || 0 }} field
              </span>
            </div>
            <div class="space-y-1">
              <span class="text-[10px] uppercase font-semibold text-neutral-400">Field Tersimpan:</span>
              <div class="flex flex-wrap gap-1">
                <span
                  v-for="f in tbl.fields"
                  :key="f"
                  class="px-1.5 py-0.5 rounded bg-white dark:bg-slate-900 border border-neutral-200/80 dark:border-neutral-700 font-mono text-[10px] text-neutral-700 dark:text-neutral-300"
                >
                  {{ f }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- DIALOG INSPECTOR MODAL: DETAIL ALUR & cURL GENERATOR -->
    <UModal
      v-model:open="isInspectorOpen"
      :title="`Inspeksi Alur: [${inspectingFlow?.http_method || 'API'}] ${inspectingFlow?.route_path || ''}`"
      :description="inspectingFlow?.action_title || 'Rincian alur end-to-end request, validasi, logika backend, dan database'"
      :ui="{ content: 'sm:max-w-4xl' }"
    >
      <template #body>
        <div v-if="inspectingFlow" class="space-y-5 text-xs max-h-[75vh] overflow-y-auto pr-1">
          <!-- Flow Context Summary Banner -->
          <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200/80 dark:border-neutral-800 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div class="space-y-0.5">
              <div class="text-[10px] text-neutral-400 uppercase font-semibold">
                Dipanggil dari Halaman:
              </div>
              <div class="font-bold text-neutral-900 dark:text-white flex items-center gap-1.5">
                <UIcon name="i-lucide-monitor" class="w-3.5 h-3.5 text-sky-500" />
                <span>{{ inspectingPage?.title }}</span>
                <span class="font-mono text-sky-600 dark:text-sky-400">({{ inspectingPage?.page_path }})</span>
              </div>
            </div>
            <!-- Tombol Salin cURL -->
            <UButton
              size="xs"
              color="primary"
              variant="outline"
              icon="i-lucide-terminal"
              class="cursor-pointer shrink-0"
              @click="copyCurlCommand(inspectingFlow)"
            >
              Salin Perintah cURL
            </UButton>
          </div>

          <!-- Inspector Stage Tab Navigation -->
          <div class="flex items-center gap-1 p-1 rounded-xl bg-neutral-100 dark:bg-neutral-800/80 border border-neutral-200 dark:border-neutral-700 overflow-x-auto">
            <button
              type="button"
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all cursor-pointer"
              :class="inspectorActiveTab === 'payload'
                ? 'bg-white dark:bg-slate-900 text-sky-600 dark:text-sky-400 shadow-2xs font-bold'
                : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
              @click="inspectorActiveTab = 'payload'"
            >
              <UIcon name="i-lucide-file-code" class="w-3.5 h-3.5 text-sky-500" />
              <span>1. Payload</span>
            </button>

            <button
              type="button"
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all cursor-pointer"
              :class="inspectorActiveTab === 'validation'
                ? 'bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 shadow-2xs font-bold'
                : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
              @click="inspectorActiveTab = 'validation'"
            >
              <UIcon name="i-lucide-shield-check" class="w-3.5 h-3.5 text-indigo-500" />
              <span>2. Validasi</span>
            </button>

            <button
              type="button"
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all cursor-pointer"
              :class="inspectorActiveTab === 'handler'
                ? 'bg-white dark:bg-slate-900 text-purple-600 dark:text-purple-400 shadow-2xs font-bold'
                : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
              @click="inspectorActiveTab = 'handler'"
            >
              <UIcon name="i-lucide-server" class="w-3.5 h-3.5 text-purple-500" />
              <span>3. Handler & Middleware</span>
            </button>

            <button
              type="button"
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all cursor-pointer"
              :class="inspectorActiveTab === 'database'
                ? 'bg-white dark:bg-slate-900 text-amber-600 dark:text-amber-400 shadow-2xs font-bold'
                : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
              @click="inspectorActiveTab = 'database'"
            >
              <UIcon name="i-lucide-database" class="w-3.5 h-3.5 text-amber-500" />
              <span>4. Basis Data (SQL/ORM)</span>
            </button>

            <button
              type="button"
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all cursor-pointer"
              :class="inspectorActiveTab === 'response'
                ? 'bg-white dark:bg-slate-900 text-emerald-600 dark:text-emerald-400 shadow-2xs font-bold'
                : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
              @click="inspectorActiveTab = 'response'"
            >
              <UIcon name="i-lucide-check-circle-2" class="w-3.5 h-3.5 text-emerald-500" />
              <span>5. Respon & UI</span>
            </button>
          </div>

          <!-- Tab Content 1: Payload -->
          <div v-if="inspectorActiveTab === 'payload'" class="space-y-4">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
                <span class="text-[10px] uppercase font-bold text-neutral-400">Pemicu Event Frontend:</span>
                <p class="font-semibold text-neutral-900 dark:text-white">
                  {{ inspectingFlow.trigger_event }}
                </p>
              </div>
              <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
                <span class="text-[10px] uppercase font-bold text-neutral-400">Metode & Endpoint:</span>
                <p class="font-mono font-bold text-sky-600 dark:text-sky-400">
                  {{ inspectingFlow.http_method }} {{ inspectingFlow.route_path }}
                </p>
              </div>
            </div>

            <div v-if="inspectingFlow.request_payload?.params" class="space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Parameter URL (Path Params):</span>
              <p class="p-2.5 rounded-lg bg-neutral-100 dark:bg-neutral-800 font-mono text-[11px] text-neutral-800 dark:text-neutral-200">
                {{ inspectingFlow.request_payload.params }}
              </p>
            </div>

            <div v-if="inspectingFlow.request_payload?.query" class="space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Query Parameters:</span>
              <p class="p-2.5 rounded-lg bg-neutral-100 dark:bg-neutral-800 font-mono text-[11px] text-neutral-800 dark:text-neutral-200">
                {{ inspectingFlow.request_payload.query }}
              </p>
            </div>

            <div v-if="inspectingFlow.request_payload?.headers" class="space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">HTTP Headers:</span>
              <p class="p-2.5 rounded-lg bg-neutral-100 dark:bg-neutral-800 font-mono text-[11px] text-neutral-800 dark:text-neutral-200">
                {{ inspectingFlow.request_payload.headers }}
              </p>
            </div>

            <div v-if="inspectingFlow.request_payload?.body" class="space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">JSON Request Body:</span>
              <pre class="p-3 rounded-xl bg-neutral-950 text-sky-400 font-mono text-[11px] overflow-x-auto border border-neutral-800 leading-relaxed">{{ inspectingFlow.request_payload.body }}</pre>
            </div>
          </div>

          <!-- Tab Content 2: Validasi -->
          <div v-else-if="inspectorActiveTab === 'validation'" class="space-y-3">
            <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-2">
              <span class="text-xs font-bold text-neutral-900 dark:text-white flex items-center gap-1.5">
                <UIcon name="i-lucide-shield-check" class="w-4 h-4 text-indigo-500" />
                <span>Aturan Validasi Integritas Data (Frontend & Backend):</span>
              </span>
              <ul v-if="inspectingFlow.validation_rules?.length" class="space-y-2 pt-1">
                <li
                  v-for="(rule, rIdx) in inspectingFlow.validation_rules"
                  :key="rIdx"
                  class="flex items-start gap-2 p-2 rounded-lg bg-white dark:bg-slate-900 border border-neutral-200/60 dark:border-neutral-800 text-neutral-700 dark:text-neutral-300"
                >
                  <UIcon name="i-lucide-check" class="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                  <span class="leading-relaxed">{{ rule }}</span>
                </li>
              </ul>
              <p v-else class="text-neutral-400 italic">
                Validasi tipe data default & unmarshaling JSON schema.
              </p>
            </div>
          </div>

          <!-- Tab Content 3: Handler & Middleware -->
          <div v-else-if="inspectorActiveTab === 'handler'" class="space-y-4">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
                <span class="text-[10px] uppercase font-bold text-neutral-400">Fungsi Handler Controller:</span>
                <p class="font-mono font-bold text-purple-700 dark:text-purple-300 text-xs">
                  {{ inspectingFlow.handler_function }}
                </p>
              </div>
              <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
                <span class="text-[10px] uppercase font-bold text-neutral-400">Guard / Middleware:</span>
                <p class="font-mono text-neutral-800 dark:text-neutral-200 text-xs">
                  {{ inspectingFlow.middleware }}
                </p>
              </div>
            </div>

            <div v-if="inspectingFlow.business_process?.length" class="space-y-2">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Tahapan Proses Logika Bisnis di Backend:</span>
              <ol class="space-y-2">
                <li
                  v-for="(step, sIdx) in inspectingFlow.business_process"
                  :key="sIdx"
                  class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200/70 dark:border-neutral-800 flex items-start gap-2.5"
                >
                  <span class="px-1.5 py-0.5 rounded font-mono font-bold text-[10px] bg-purple-100 dark:bg-purple-950 text-purple-700 dark:text-purple-300">
                    {{ sIdx + 1 }}
                  </span>
                  <span class="text-neutral-700 dark:text-neutral-300 leading-relaxed">{{ step }}</span>
                </li>
              </ol>
            </div>
          </div>

          <!-- Tab Content 4: Basis Data -->
          <div v-else-if="inspectorActiveTab === 'database'" class="space-y-4">
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <div class="p-3 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
                <span class="text-[10px] uppercase font-bold text-neutral-400">Database Engine:</span>
                <p class="font-bold text-neutral-900 dark:text-white">
                  {{ inspectingFlow.database_operation?.engine || 'SQLite' }}
                </p>
              </div>
              <div class="p-3 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
                <span class="text-[10px] uppercase font-bold text-neutral-400">Operasi:</span>
                <p class="font-mono font-bold text-amber-600 dark:text-amber-400">
                  {{ inspectingFlow.database_operation?.operation_type || 'SELECT' }}
                </p>
              </div>
              <div class="p-3 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
                <span class="text-[10px] uppercase font-bold text-neutral-400">Tabel Terhubung:</span>
                <p class="font-mono font-bold text-amber-700 dark:text-amber-300 truncate">
                  {{ inspectingFlow.database_operation?.target_tables?.join(', ') || '-' }}
                </p>
              </div>
            </div>

            <div v-if="inspectingFlow.database_operation?.stored_fields?.length" class="space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Kolom Data yang Disimpan / Dimutasi:</span>
              <div class="flex flex-wrap gap-1.5 p-2 rounded-lg bg-neutral-100 dark:bg-neutral-800">
                <span
                  v-for="sf in inspectingFlow.database_operation.stored_fields"
                  :key="sf"
                  class="px-2 py-0.5 rounded bg-white dark:bg-slate-900 border border-neutral-200 dark:border-neutral-700 font-mono text-[10px] font-bold text-amber-700 dark:text-amber-400"
                >
                  {{ sf }}
                </span>
              </div>
            </div>

            <div v-if="inspectingFlow.database_operation?.query_detail" class="space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Query SQL / Eksekusi ORM:</span>
              <pre class="p-3 rounded-xl bg-neutral-950 text-amber-300 font-mono text-[11px] overflow-x-auto border border-neutral-800 leading-relaxed">{{ inspectingFlow.database_operation.query_detail }}</pre>
            </div>
          </div>

          <!-- Tab Content 5: Respon & UI Impact -->
          <div v-else-if="inspectorActiveTab === 'response'" class="space-y-4">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
                <span class="text-[10px] uppercase font-bold text-neutral-400">Status Respon HTTP:</span>
                <p class="font-mono font-bold text-emerald-600 dark:text-emerald-400 text-sm">
                  HTTP {{ inspectingFlow.response_spec?.status_code || 200 }}
                </p>
              </div>
              <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
                <span class="text-[10px] uppercase font-bold text-neutral-400">Tipe Data:</span>
                <p class="font-mono font-semibold text-neutral-800 dark:text-neutral-200 text-sm">
                  {{ inspectingFlow.response_spec?.data_type || 'JSON Object' }}
                </p>
              </div>
            </div>

            <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Dampak pada Antarmuka (UI Reaction):</span>
              <p class="font-semibold text-neutral-900 dark:text-white leading-relaxed">
                {{ inspectingFlow.ui_impact || 'Perbarui state reaktif dan render ulang komponen tampilan.' }}
              </p>
            </div>

            <div v-if="inspectingFlow.response_spec?.body_sample" class="space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Contoh Payload Respon JSON:</span>
              <pre class="p-3 rounded-xl bg-neutral-950 text-emerald-400 font-mono text-[11px] overflow-x-auto border border-neutral-800 leading-relaxed">{{ inspectingFlow.response_spec.body_sample }}</pre>
            </div>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
