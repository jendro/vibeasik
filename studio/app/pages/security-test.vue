<script setup lang="ts">
import type { SecurityCheckStatus, SecurityTestReport } from '~/types'
import { SEC_STATUS_META, SEC_STATUS_ORDER, SEC_TAB_META, securityBaseUrlProblem, countSecurityByTab, filterSecurityChecks, securityGradeColor, securityGradeLabel, type SecurityFilter } from '~/utils/securityTest'

/**
 * Uji Keamanan: memeriksa router/kode backend proyek aktif untuk kontrol keamanan standar, lalu
 * membuktikannya lewat respons HTTP nyata dari server proyek sendiri (port lokal atau URL live).
 */
definePageMeta({ fluid: true })

const { activeProjectId, activeProject, fetchProjects } = useActiveProject()
const { getLast, run } = useSecurityTest()
const toast = useToast()

const report = ref<SecurityTestReport | null>(null)
const loading = ref(false)
const running = ref(false)
const error = ref('')

const componentId = ref<number>(0)
const baseUrl = ref('')
const withBurst = ref(true)
const filter = reactive<SecurityFilter>({ tab: 'all', status: 'all', source: 'all', q: '' })

const BURST_KEY = 'vibeasik_sectest_burst'
onMounted(async () => {
  try {
    withBurst.value = localStorage.getItem(BURST_KEY) !== '0'
  } catch { /* penyimpanan tidak tersedia */ }
  await fetchProjects()
  await load()
})
watch(withBurst, (v) => {
  try {
    localStorage.setItem(BURST_KEY, v ? '1' : '0')
  } catch { /* abaikan */ }
})
watch(activeProjectId, () => {
  report.value = null
  componentId.value = 0
  load()
})

const load = async () => {
  if (!activeProjectId.value) return
  loading.value = true
  error.value = ''
  try {
    report.value = (await getLast(activeProjectId.value)).report
  } catch (err: any) {
    error.value = err?.data?.error || err?.message || 'Gagal memuat hasil uji'
  } finally {
    loading.value = false
  }
}

const componentOptions = computed(() => [
  { label: 'Semua komponen backend', value: 0 },
  ...(activeProject.value?.components || []).map(c => ({ label: `${c.name} · ${c.role}${c.run_port || c.port ? ` · :${c.run_port || c.port}` : ''}`, value: c.id }))
])

const urlError = computed(() => securityBaseUrlProblem(baseUrl.value))

const start = async () => {
  if (!activeProjectId.value || urlError.value) return
  running.value = true
  error.value = ''
  try {
    report.value = await run(activeProjectId.value, {
      component_id: componentId.value || undefined,
      base_url: baseUrl.value.trim() || undefined,
      skip_burst: !withBurst.value
    })
    const s = report.value.summary
    toast.add({ title: `Uji keamanan selesai — skor ${report.value.score}`, description: `${s.fail} gagal, ${s.warn} lemah, ${s.pass} lulus (${report.value.duration_ms} ms)`, icon: 'i-lucide-shield', color: s.fail ? 'warning' : 'success' })
  } catch (err: any) {
    error.value = err?.data?.error || err?.message || 'Uji keamanan gagal dijalankan'
  } finally {
    running.value = false
  }
}

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
const criticalFails = computed(() => checks.value.filter(c => c.status === 'fail' && c.severity === 'CRITICAL').length)
const fmtTime = (s?: string) => (s ? new Date(s).toLocaleString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) : '-')
</script>

<template>
  <div class="w-full max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 pt-6 pb-16 space-y-6">
    <AppPageHeader
      title="Uji Keamanan"
      description="Periksa router backend proyek untuk kontrol keamanan standar — rate limit (DDoS & brute-force), query berparameter (SQL injection), header, CORS, validasi input, autentikasi — lalu buktikan di server yang berjalan."
    >
      <template v-if="report" #meta>
        <span class="inline-flex items-center gap-1"><UIcon name="i-lucide-clock" class="size-3.5" /> Terakhir diuji {{ fmtTime(report.started_at) }}</span>
        <span class="inline-flex items-center gap-1"><UIcon name="i-lucide-files" class="size-3.5" /> {{ report.files_read }} berkas server · {{ report.routes_seen }} rute</span>
        <span v-if="report.live" class="inline-flex items-center gap-1"><UIcon name="i-lucide-radio-tower" class="size-3.5" /> {{ report.live.requests_run }} request ke server</span>
        <span v-if="report.summary.fail" class="rounded-md bg-rose-500/10 px-1.5 text-rose-700 dark:text-rose-300 font-medium">{{ report.summary.fail }} gagal</span>
      </template>
    </AppPageHeader>

    <UAlert v-if="!activeProjectId" color="neutral" variant="subtle" icon="i-lucide-folder-search" title="Pilih proyek aktif" description="Pilih proyek lewat pemilih proyek di navbar atas untuk menguji keamanannya." />

    <template v-else>
      <!-- Sasaran uji -->
      <section class="p-4 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] space-y-3">
        <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-[minmax(0,16rem)_minmax(0,1fr)_auto] items-start">
          <UFormField label="Komponen" size="sm">
            <USelect v-model="componentId" :items="componentOptions" size="sm" class="w-full" />
          </UFormField>
          <UFormField label="URL server live (opsional)" size="sm" :error="urlError || undefined" help="Kosong = port lokal komponen yang sedang berjalan">
            <UInput v-model="baseUrl" size="sm" icon="i-lucide-globe" placeholder="https://api.contoh.com atau 127.0.0.1:4101" class="w-full" @keyup.enter="start" />
          </UFormField>
          <div class="flex flex-wrap items-center gap-3 md:col-span-2 xl:col-span-1 xl:pt-6">
            <USwitch v-model="withBurst" size="sm" label="Uji burst rate limit" title="Kirim 25 request cepat untuk membuktikan 429; matikan untuk server produksi yang rapuh" />
            <AppActionButton kind="scan" primary label="Jalankan uji" desc="Uji router & server live: rate limit, SQL injection, header, CORS, auth" :loading="running" :disabled="!!urlError" @click="start" />
          </div>
        </div>
        <p class="flex gap-2 text-xs text-neutral-500 dark:text-slate-400">
          <UIcon name="i-lucide-info" class="size-3.5 mt-0.5 shrink-0" />
          Probe hanya mengirim request biasa ke server milik proyek ini (GET, OPTIONS, POST body kosong ke rute login) — tidak ada data yang diubah. Gunakan hanya pada server yang Anda kelola.
        </p>
      </section>

      <UAlert v-if="error" color="error" variant="subtle" icon="i-lucide-alert-circle" title="Uji keamanan tidak bisa dijalankan" :description="error" />

      <div v-if="running && !report" class="py-16 flex flex-col items-center gap-2 text-neutral-500 text-sm">
        <UIcon name="i-lucide-loader-circle" class="size-8 animate-spin" />
        Memeriksa router dan menguji server…
      </div>
      <div v-else-if="loading && !report" class="py-16 flex flex-col items-center gap-2 text-neutral-500 text-sm">
        <UIcon name="i-lucide-loader-circle" class="size-8 animate-spin" />
        Memuat hasil uji terakhir…
      </div>

      <div v-else-if="!report" class="p-12 text-center rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f]">
        <div class="mx-auto mb-3 size-12 rounded-full bg-sky-50 dark:bg-sky-500/10 text-sky-600 dark:text-sky-400 flex items-center justify-center">
          <UIcon name="i-lucide-shield" class="size-6" />
        </div>
        <h3 class="text-base font-bold text-neutral-900 dark:text-white">
          Belum ada hasil uji
        </h3>
        <p class="mt-1 text-xs text-neutral-500 dark:text-slate-400 max-w-md mx-auto">
          Klik Jalankan uji. Router dibaca dari kode komponen backend; bila komponen sedang berjalan atau URL live diisi, setiap kontrol dibuktikan lewat respons HTTP nyata.
        </p>
      </div>

      <template v-if="report">
        <UAlert
          v-if="criticalFails"
          color="warning"
          variant="subtle"
          icon="i-lucide-triangle-alert"
          :title="`${criticalFails} kontrol kritis gagal`"
          description="Tutup temuan bertanda kritis lebih dulu: rute login tanpa rate limit, query SQL dari penggabungan string, CORS dengan credentials, atau server non-lokal tanpa HTTPS."
        />

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
            class="text-left px-4 py-3.5 sm:px-5 sm:py-4 transition-colors"
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

        <!-- Server berjalan -->
        <section v-if="report.live" class="p-4 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] space-y-3">
          <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
            <h3 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
              <UIcon name="i-lucide-radio-tower" class="size-4 text-sky-500" /> Server yang diuji
            </h3>
            <span class="font-mono text-xs text-neutral-700 dark:text-slate-300 truncate">{{ report.live.base_url }}</span>
            <span class="inline-flex items-center gap-1.5 text-xs" :class="report.live.reachable ? 'text-emerald-600' : 'text-rose-600'">
              <span class="size-2 rounded-full" :class="report.live.reachable ? 'bg-emerald-500' : 'bg-rose-500'" />
              {{ report.live.reachable ? `HTTP ${report.live.status_code} · ${report.live.latency_ms} ms` : 'Tidak terjangkau' }}
            </span>
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
        <UAlert v-for="(n, i) in report.notes" :key="i" color="neutral" variant="subtle" icon="i-lucide-info" :description="n" />

        <!-- Filter -->
        <div class="flex flex-col 2xl:flex-row justify-between gap-3 pb-3 border-b border-neutral-200 dark:border-white/[0.07]">
          <div class="flex gap-1 overflow-x-auto [scrollbar-width:none]">
            <button
              v-for="t in SEC_TAB_META"
              :key="t.id"
              type="button"
              class="shrink-0 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium"
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
    </template>
  </div>
</template>
