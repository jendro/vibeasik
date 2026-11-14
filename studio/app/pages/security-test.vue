<script setup lang="ts">
import type { RuntimeSimComponent, SecurityCheckStatus, SecurityTestSession } from '~/types'
import type { SecurityTestOptions } from '~/components/security/SecurityTestConfig.vue'
import { SEC_STATUS_META, SEC_STATUS_ORDER, SEC_TAB_META, securityBaseUrlProblem, countSecurityByTab, filterSecurityChecks, securityGradeColor, securityGradeLabel, type SecurityFilter } from '~/utils/securityTest'

/**
 * Uji Keamanan (bentuk sama dengan Uji Runtime): engine menyalakan server backend proyek bila belum berjalan,
 * membaca router & kode, menguji server lewat request nyata (header, CORS, input, rate limit), lalu mematikannya
 * lagi. Halaman memantau langkah, aktivitas request, dan log server tiap detik.
 */
definePageMeta({ fluid: true })

const route = useRoute()
const toast = useToast()
const { projects, activeProjectId, activeProject, fetchProjects, selectActiveProject } = useActiveProject()

const OPTS_KEY = 'vibeasik_sectest_opts'
const defaultOptions = (): SecurityTestOptions => ({ component_id: 0, base_url: '', start_server: true, burst: true, keep_running: false, attacks: true, auth_header: '' })
const options = ref<SecurityTestOptions>(defaultOptions())
const showConfig = ref(true)

const { report, session, loading, error, isRunning, starting, cancelling, load, start, cancel, createTodos } = useSecurityTestSession((s: SecurityTestSession) => {
  const ok = s.status === 'completed'
  const sm = s.report?.summary
  toast.add({
    title: ok ? `Uji keamanan selesai — skor ${s.report?.score ?? '-'}` : s.status === 'cancelled' ? 'Uji keamanan dibatalkan' : 'Uji keamanan gagal',
    description: sm ? `${sm.fail} gagal, ${sm.warn} lemah, ${sm.pass} lulus` : s.summary,
    icon: ok ? 'i-lucide-shield' : 'i-lucide-alert-circle',
    color: ok ? (sm?.fail ? 'warning' : 'success') : s.status === 'cancelled' ? 'neutral' : 'error'
  })
})

const loadProject = async (pid: number) => {
  await load(pid)
  showConfig.value = !session.value && !report.value
}

onMounted(async () => {
  try {
    const saved = JSON.parse(localStorage.getItem(OPTS_KEY) || '{}')
    options.value = { ...defaultOptions(), start_server: saved.start_server ?? true, burst: saved.burst ?? true, keep_running: !!saved.keep_running, attacks: saved.attacks ?? true }
  } catch { /* penyimpanan tidak tersedia */ }
  await fetchProjects()
  const queryPId = route.query.project_id ? Number(route.query.project_id) : null
  if (queryPId) selectActiveProject(queryPId)
  else if (!activeProjectId.value && projects.value[0]) selectActiveProject(projects.value[0].id)
  if (activeProjectId.value) await loadProject(activeProjectId.value)
})
// Header login sengaja tidak disimpan di browser.
watch(() => [options.value.start_server, options.value.burst, options.value.keep_running, options.value.attacks], ([start_server, burst, keep_running, attacks]) => {
  try {
    localStorage.setItem(OPTS_KEY, JSON.stringify({ start_server, burst, keep_running, attacks }))
  } catch { /* abaikan */ }
})
watch(activeProjectId, (pid, old) => {
  if (pid && pid !== old) {
    options.value = { ...options.value, component_id: 0, base_url: '', auth_header: '' }
    loadProject(pid)
  }
})

const components = computed(() => activeProject.value?.components || [])
const useLive = computed(() => options.value.component_id === -1)
const optionsProblem = computed(() => (useLive.value ? (options.value.base_url.trim() ? securityBaseUrlProblem(options.value.base_url) : 'Isi URL server live') : ''))

const handleStart = async () => {
  if (optionsProblem.value) {
    showConfig.value = true
    toast.add({ title: optionsProblem.value, icon: 'i-lucide-globe', color: 'warning' })
    return
  }
  const o = options.value
  await start({
    component_id: o.component_id > 0 ? o.component_id : undefined,
    base_url: useLive.value ? o.base_url.trim() : undefined,
    start_server: !useLive.value && o.start_server,
    keep_running: !useLive.value && o.start_server && o.keep_running,
    skip_burst: !o.burst,
    skip_attacks: !o.attacks,
    auth_header: o.attacks ? o.auth_header.trim() || undefined : undefined
  })
  if (isRunning.value) showConfig.value = false
}

// Log server untuk terminal (komponen terminal Uji Runtime dipakai bersama).
const terminalComponents = computed(() => (session.value?.server ? [session.value.server as unknown as RuntimeSimComponent] : []))
const finished = computed(() => Boolean(session.value && session.value.status !== 'running'))

const filter = reactive<SecurityFilter>({ tab: 'all', status: 'all', source: 'all', q: '' })
const checks = computed(() => report.value?.checks || [])
const visible = computed(() => filterSecurityChecks(checks.value, filter))
const tabCounts = computed(() => countSecurityByTab(filterSecurityChecks(checks.value, { ...filter, tab: 'all' })))
const statusOptions = computed(() => [
  { label: `Semua status (${checks.value.length})`, value: 'all' },
  ...SEC_STATUS_ORDER.map(s => ({ label: `${SEC_STATUS_META[s].label} (${report.value?.summary[s] || 0})`, value: s }))
])
const sourceOptions = computed(() => [
  { label: `Kode & server (${checks.value.length})`, value: 'all' },
  { label: `Kode (${checks.value.filter(c => c.source === 'static').length})`, value: 'static' },
  { label: `Server (${checks.value.filter(c => c.source === 'live').length})`, value: 'live' }
])
const toggleStatus = (s: SecurityCheckStatus) => {
  filter.status = filter.status === s ? 'all' : s
}

// Rekap temuan → todo: pemeriksaan gagal & lemah (tanpa LLM; isi todo dari bukti & saran pemeriksaan itu).
const todoCount = computed(() => (report.value?.summary.fail || 0) + (report.value?.summary.warn || 0))
const todoNote = computed(() => {
  const s = report.value?.summary
  if (!s) return ''
  return todoCount.value
    ? `${s.fail} gagal dan ${s.warn} lemah perlu diperbaiki; ${s.pass} sudah terbukti lulus. Klik angka untuk menyaring daftar.`
    : `Tidak ada pemeriksaan yang gagal atau lemah; ${s.pass} terbukti lulus.`
})
const todoBusy = ref(false)
const makeAllTodos = async () => {
  if (!activeProjectId.value) return
  todoBusy.value = true
  try {
    const res = await createTodos(activeProjectId.value)
    toast.add({
      title: res.created ? `${res.created} todo keamanan dibuat` : 'Tidak ada todo baru',
      description: res.existed ? `${res.existed} pemeriksaan sudah punya todo terbuka.` : undefined,
      color: res.created ? 'success' : 'neutral',
      icon: 'i-lucide-list-checks',
      actions: [{ label: 'Buka Todo', color: 'neutral', variant: 'outline', onClick: () => { navigateTo('/todos') } }]
    })
  } catch (err: any) {
    toast.add({ title: 'Gagal menjadikan todo', description: err?.data?.error || err?.message || '', color: 'error' })
  } finally {
    todoBusy.value = false
  }
}
const criticalFails = computed(() => checks.value.filter(c => c.status === 'fail' && c.severity === 'CRITICAL').length)
const fmtTime = (s?: string) => (s ? new Date(s).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' }) : '-')
const serverStateLabel: Record<string, { label: string, dot: string }> = {
  starting: { label: 'Menyala…', dot: 'bg-sky-500 animate-pulse' },
  ready: { label: 'Berjalan', dot: 'bg-emerald-500' },
  stopped: { label: 'Dimatikan', dot: 'bg-neutral-400' },
  failed: { label: 'Gagal', dot: 'bg-rose-500' },
  external: { label: 'Server live', dot: 'bg-sky-500' }
}
const sourceLabel: Record<string, string> = { launched: 'dinyalakan uji ini', reused: 'sudah berjalan sebelumnya', external: 'URL live' }
</script>

<template>
  <div class="space-y-5 w-full min-w-0 px-4 sm:px-6 lg:px-8 pt-6 pb-16">
    <AppPageHeader
      title="Uji Keamanan"
      description="Server backend dinyalakan sungguhan lalu diserang seperti peretas: SQL/NoSQL injection, XSS, path traversal, command injection, JWT palsu, akses tanpa login, berkas bocor, dan rate limit — dibuktikan lewat respons HTTP nyata, bukan hanya dari kode."
    >
      <template v-if="session || report" #meta>
        <span class="inline-flex items-center gap-1.5">
          <span class="size-1.5 rounded-full" :class="isRunning ? 'bg-sky-500 animate-pulse' : 'bg-neutral-400'" />
          {{ isRunning ? 'Sedang berjalan' : 'Uji terakhir' }} {{ fmtTime(session?.started_at || report?.started_at) }}
        </span>
        <span v-if="report && !isRunning">Skor keamanan <b class="tabular-nums" :class="securityGradeColor(report.score)">{{ report.score }}/100</b></span>
        <span v-if="report && !isRunning" class="tabular-nums">{{ report.files_read }} berkas · {{ report.routes_seen }} rute</span>
        <span v-if="report?.summary.fail && !isRunning" class="tabular-nums text-rose-600 dark:text-rose-400">{{ report.summary.fail }} gagal</span>
      </template>
      <template #actions>
        <UButton v-if="isRunning" size="sm" color="error" variant="soft" icon="i-lucide-square" :loading="cancelling" @click="cancel">
          Batalkan
        </UButton>
        <AppActionButton
          v-else
          kind="scan"
          primary
          :label="session || report ? 'Jalankan ulang' : 'Mulai uji keamanan'"
          desc="Nyalakan server bila mati, baca router & kode, serang semua endpoint lewat request nyata, lalu matikan lagi"
          :loading="starting"
          :disabled="!activeProjectId"
          @click="handleStart"
        />
      </template>
    </AppPageHeader>

    <UAlert v-if="!activeProjectId" color="neutral" variant="subtle" icon="i-lucide-folder-search" title="Pilih proyek aktif" description="Pilih proyek lewat pemilih proyek di navbar atas untuk menguji keamanannya." />

    <template v-else>
      <!-- Checklist -->
      <div class="space-y-2">
        <button type="button" class="flex items-center gap-1.5 text-xs font-bold text-neutral-700 dark:text-neutral-200 cursor-pointer" @click="showConfig = !showConfig">
          <UIcon :name="showConfig ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="w-4 h-4" />
          Checklist uji
          <span v-if="useLive" class="font-normal text-neutral-500">· server live {{ options.base_url || '(belum diisi)' }}</span>
          <span v-else-if="options.start_server" class="font-normal text-neutral-500">· server dinyalakan otomatis bila mati</span>
          <span v-else class="font-normal text-amber-600 dark:text-amber-400">· hanya server yang sudah berjalan</span>
          <span v-if="options.attacks" class="font-normal text-rose-600 dark:text-rose-400">· serang semua endpoint{{ options.auth_header ? ' (dengan login)' : '' }}</span>
          <span v-if="!options.burst" class="font-normal text-neutral-500">· tanpa burst</span>
        </button>
        <SecurityTestConfig v-if="showConfig" v-model="options" :components="components" :disabled="isRunning" />
      </div>

      <UAlert v-if="error" color="error" variant="subtle" icon="i-lucide-alert-circle" title="Uji keamanan tidak bisa dijalankan" :description="error" />

      <div v-if="loading && !session && !report" class="py-16 flex flex-col items-center gap-2 text-neutral-500 text-sm">
        <UIcon name="i-lucide-loader-circle" class="size-8 animate-spin" />
        Memuat hasil uji terakhir…
      </div>

      <div v-else-if="!session && !report" class="p-12 text-center rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f]">
        <div class="mx-auto mb-3 size-12 rounded-full bg-sky-50 dark:bg-sky-500/10 text-sky-600 dark:text-sky-400 flex items-center justify-center">
          <UIcon name="i-lucide-shield" class="size-6" />
        </div>
        <h3 class="text-base font-bold text-neutral-900 dark:text-white">
          Belum ada hasil uji
        </h3>
        <p class="mt-1 text-xs text-neutral-500 dark:text-slate-400 max-w-md mx-auto">
          Klik Mulai uji keamanan. Server backend dinyalakan lewat dev command-nya, setiap kontrol dibuktikan lewat respons HTTP nyata, lalu server dimatikan lagi.
        </p>
      </div>

      <div v-else class="grid grid-cols-1 gap-4 items-start" :class="session ? 'lg:grid-cols-[minmax(0,1fr)_320px] 2xl:grid-cols-[minmax(0,1fr)_360px]' : ''">
        <!-- Sidebar kanan: langkah & aktivitas tetap terlihat saat hasil di-scroll -->
        <div v-if="session" class="lg:col-start-2 lg:row-start-1 lg:sticky lg:top-4">
          <SecurityTestRail :session="session" :running="isRunning" />
        </div>

        <div class="lg:col-start-1 lg:row-start-1 min-w-0 space-y-4">
          <!-- Server yang diuji + log -->
          <section v-if="session?.server" class="px-4 py-3.5 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] space-y-3">
            <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
              <h3 class="text-sm font-semibold text-neutral-900 dark:text-white flex items-center gap-2">
                <UIcon name="i-lucide-server" class="size-4 text-sky-500" /> {{ session.server.name }}
              </h3>
              <span class="inline-flex items-center gap-1.5 text-xs text-neutral-700 dark:text-slate-300">
                <span class="size-2 rounded-full" :class="serverStateLabel[session.server.state]?.dot || 'bg-neutral-400'" />
                {{ serverStateLabel[session.server.state]?.label || session.server.state }}
              </span>
              <span v-if="session.server.url" class="font-mono text-xs text-neutral-600 dark:text-slate-400 truncate">{{ session.server.url }}</span>
              <span v-if="session.server.source" class="text-xs text-neutral-500">{{ sourceLabel[session.server.source] }}</span>
              <span v-if="session.server.boot_ms" class="text-xs tabular-nums text-neutral-500">siap {{ (session.server.boot_ms / 1000).toFixed(1) }} dtk</span>
            </div>
            <p v-if="session.server.command" class="font-mono text-xs text-neutral-500 truncate">
              $ {{ session.server.command }}
            </p>
            <p v-if="session.server.note" class="text-xs" :class="session.server.state === 'failed' ? 'text-rose-600 dark:text-rose-400' : 'text-neutral-500'">
              {{ session.server.note }}
            </p>
          </section>
          <DiagnosticsRuntimeSimTerminal v-if="session" :events="session.events || []" :components="terminalComponents" :running="isRunning" />

          <template v-if="report && (finished || !session)">
            <UAlert
              v-if="criticalFails"
              color="warning"
              variant="subtle"
              icon="i-lucide-triangle-alert"
              :title="`${criticalFails} kontrol kritis gagal`"
              description="Tutup temuan bertanda kritis lebih dulu: rute login tanpa rate limit, query SQL dari penggabungan string, CORS dengan credentials, atau server non-lokal tanpa HTTPS."
            />

            <FindingsTodoBar
              :project-id="activeProjectId"
              :stats="[]"
              :count="todoCount"
              :note="todoNote"
              todo-desc="Setiap pemeriksaan gagal atau lemah dijadikan satu todo keamanan berisi bukti dan langkahnya — tanpa LLM"
              :busy="todoBusy"
              @todo-all="makeAllTodos"
            >
              <!-- Skor + rekap angka: satu bingkai bersambung, 6 sel (skor + 5 status) → 2 / 3 / 6 kolom -->
              <div class="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-6 gap-px overflow-hidden rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-neutral-200 dark:bg-white/[0.08]">
                <div class="bg-white dark:bg-[#04111f] px-4 py-3.5 sm:px-5 sm:py-4">
                  <p class="text-xs font-medium text-neutral-500 dark:text-slate-400">
                    Skor · nilai {{ report.grade }}
                  </p>
                  <p class="text-2xl font-semibold tracking-tight tabular-nums" :class="securityGradeColor(report.score)">
                    {{ report.score }}<span class="text-sm font-medium text-neutral-400">/100</span>
                  </p>
                  <p class="text-xs text-neutral-400 truncate" :title="securityGradeLabel(report.grade)">
                    {{ securityGradeLabel(report.grade) }}
                  </p>
                </div>
                <button
                  v-for="s in SEC_STATUS_ORDER"
                  :key="s"
                  type="button"
                  class="text-left px-4 py-3.5 sm:px-5 sm:py-4 transition-colors cursor-pointer"
                  :class="filter.status === s ? 'bg-sky-50/60 dark:bg-sky-500/10' : 'bg-white dark:bg-[#04111f] hover:bg-neutral-50 dark:hover:bg-white/[0.03]'"
                  :title="`${SEC_STATUS_META[s].hint} — klik untuk memfilter`"
                  @click="toggleStatus(s)"
                >
                  <p class="text-xs font-medium text-neutral-500 dark:text-slate-400">
                    {{ SEC_STATUS_META[s].label }}
                  </p>
                  <p class="text-2xl font-semibold tracking-tight tabular-nums" :class="report.summary[s] ? SEC_STATUS_META[s].text : 'text-neutral-300 dark:text-slate-600'">
                    {{ report.summary[s] }}
                  </p>
                  <p class="text-xs text-neutral-400 truncate">
                    {{ SEC_STATUS_META[s].hint }}
                  </p>
                </button>
              </div>
            </FindingsTodoBar>

            <!-- Bukti dari server -->
            <section v-if="report.live" class="px-4 py-3.5 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] space-y-3">
              <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
                <h3 class="text-sm font-semibold text-neutral-900 dark:text-white flex items-center gap-2">
                  <UIcon name="i-lucide-radio-tower" class="size-4 text-sky-500" /> Respons server
                </h3>
                <span class="font-mono text-xs text-neutral-700 dark:text-slate-300 truncate">{{ report.live.base_url }}</span>
                <span class="inline-flex items-center gap-1.5 text-xs" :class="report.live.reachable ? 'text-emerald-600' : 'text-rose-600'">
                  <span class="size-2 rounded-full" :class="report.live.reachable ? 'bg-emerald-500' : 'bg-rose-500'" />
                  {{ report.live.reachable ? `HTTP ${report.live.status_code} · ${report.live.latency_ms} ms` : 'Tidak terjangkau' }}
                </span>
                <span class="text-xs tabular-nums text-neutral-500">{{ report.live.requests_run }} request</span>
              </div>
              <p v-if="report.live.paths?.length" class="flex flex-wrap items-center gap-1.5 text-xs text-neutral-500 dark:text-slate-400">
                Rute diuji ({{ report.live.paths.length }}):
                <code v-for="p in report.live.paths" :key="p" class="px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-white/[0.06] font-mono text-neutral-700 dark:text-slate-300">{{ p }}</code>
              </p>
              <p v-if="report.live.error" class="text-xs text-rose-600 dark:text-rose-400 font-mono">
                {{ report.live.error }}
              </p>
              <dl v-if="report.live.headers && Object.keys(report.live.headers).length" class="grid gap-x-4 gap-y-1 sm:grid-cols-[auto_minmax(0,1fr)] text-xs">
                <template v-for="(v, k) in report.live.headers" :key="k">
                  <dt class="font-mono text-neutral-500 dark:text-slate-400">
                    {{ k }}
                  </dt>
                  <dd class="font-mono text-neutral-800 dark:text-slate-200 truncate">
                    {{ v }}
                  </dd>
                </template>
              </dl>
            </section>
            <SecurityAttackTable v-if="report.live?.attacks?.length" :attacks="report.live.attacks" />
            <UAlert v-for="(n, i) in report.notes" :key="i" color="neutral" variant="subtle" icon="i-lucide-info" :description="n" />

            <!-- Filter -->
            <div class="flex flex-col 2xl:flex-row justify-between gap-3 pb-3 border-b border-neutral-200 dark:border-white/[0.07]">
              <div class="flex gap-1.5 overflow-x-auto [scrollbar-width:none] py-1">
                <button
                  v-for="t in SEC_TAB_META"
                  :key="t.id"
                  type="button"
                  class="shrink-0 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium whitespace-nowrap transition-colors cursor-pointer"
                  :class="filter.tab === t.id ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900' : 'text-neutral-600 dark:text-slate-400 hover:bg-neutral-100 dark:hover:bg-white/[0.05]'"
                  @click="filter.tab = t.id"
                >
                  <UIcon :name="t.icon" class="size-3.5" /> {{ t.label }} <span class="opacity-60 tabular-nums">{{ tabCounts[t.id] }}</span>
                </button>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <USelect v-model="filter.status" :items="statusOptions" size="sm" class="w-40" />
                <USelect v-model="filter.source" :items="sourceOptions" size="sm" class="w-40" />
                <UInput v-model="filter.q" size="sm" icon="i-lucide-search" placeholder="Cari ID, judul, berkas…" class="w-48 sm:w-64" />
              </div>
            </div>

            <div v-if="visible.length" class="space-y-3">
              <SecurityCheckCard v-for="c in visible" :key="c.id + c.source" :check="c" />
            </div>
            <div v-else class="p-12 text-center rounded-xl border border-neutral-200 dark:border-white/[0.08] text-xs text-neutral-500">
              Tidak ada pemeriksaan yang cocok dengan filter.
            </div>
          </template>
        </div>
      </div>
    </template>
  </div>
</template>
