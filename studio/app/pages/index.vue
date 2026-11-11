<script setup lang="ts">
// Beranda Studio: dasbor kerja auditor — status engine & edisi yang sebenarnya, ringkasan audit proyek aktif,
// proyek yang perlu perhatian, lalu panduan singkat. Semua angka dibaca dari engine, tidak ditulis tetap.
import type { Project } from '~/types'

useHead({ title: 'Beranda' })

const { apiBase, getProjects } = useApi()
const { info: edition, load: loadEdition } = useEdition()
const { activeProject, activeProjectId, fetchProjects } = useActiveProject()
const toast = useToast()

interface LatestDiag {
  score: number
  active_findings_count: number
  resolved_findings_count: number
  ignored_findings_count: number
  total_findings: number
  severity_counts?: Record<string, number>
  diagnostic?: { created_at?: string }
}
interface GuidelineCategory { guidelines?: unknown[] }

const loaded = ref(false)
const engineOk = ref<boolean | null>(null)
const projects = ref<Project[] | null>(null)
const rules = ref<{ total: number, categories: number } | null>(null)
const latest = ref<LatestDiag | null>(null)
const latestState = ref<'idle' | 'loading' | 'none' | 'error'>('idle')

const loadLatest = async (id: number | null) => {
  latest.value = null
  if (!id) {
    latestState.value = 'idle'
    return
  }
  latestState.value = 'loading'
  try {
    latest.value = await $fetch<LatestDiag>(`${apiBase}/api/projects/${id}/diagnostics/latest`, { timeout: 5000 })
    latestState.value = 'idle'
  } catch (e: unknown) {
    latestState.value = (e as { statusCode?: number })?.statusCode === 404 ? 'none' : 'error'
  }
}

onMounted(async () => {
  loadEdition()
  fetchProjects()
  const [health, list, guides] = await Promise.allSettled([
    $fetch<{ status: string }>(`${apiBase}/health`, { timeout: 2000 }),
    getProjects(),
    $fetch<GuidelineCategory[]>(`${apiBase}/api/guidelines`, { timeout: 4000 })
  ])
  engineOk.value = health.status === 'fulfilled' && health.value.status === 'ok'
  projects.value = list.status === 'fulfilled' ? list.value : null
  if (guides.status === 'fulfilled' && Array.isArray(guides.value)) {
    rules.value = { total: guides.value.reduce((n, c) => n + (c.guidelines?.length || 0), 0), categories: guides.value.length }
  }
  loaded.value = true
})
watch(activeProjectId, id => loadLatest(id), { immediate: true })

const host = useEngineHost()
const isActive = (p: Project) => !p.status || !['hold', 'done', 'deprecated'].includes(p.status)
const activeCount = computed(() => projects.value?.filter(isActive).length ?? 0)

// Edisi: jelaskan keadaan fitur Pro apa adanya (bukan kalimat tetap).
const lockedCount = computed(() => {
  const f = edition.value?.features
  return f ? Object.values(f).filter(v => !v).length : 0
})
const editionHint = computed(() => {
  if (!edition.value) return 'Engine tidak melaporkan edisi'
  if (lockedCount.value === 0) return 'Semua fitur Pro aktif'
  return `${lockedCount.value} fitur Pro terkunci — masuk akun untuk membuka`
})

const kpis = computed(() => [
  {
    label: 'Engine lokal',
    value: engineOk.value === null ? '…' : engineOk.value ? 'Berjalan' : 'Offline',
    hint: host.value || 'engine lokal',
    dot: engineOk.value === null ? 'bg-neutral-400' : engineOk.value ? 'bg-emerald-500' : 'bg-rose-500'
  },
  {
    label: 'Proyek terdaftar',
    value: !loaded.value ? '…' : projects.value === null ? '—' : String(projects.value.length),
    hint: !loaded.value ? 'Memuat…' : projects.value === null ? 'Gagal memuat' : `${activeCount.value} aktif`,
    to: '/cockpit'
  },
  {
    label: 'Rule audit',
    value: !loaded.value ? '…' : rules.value ? String(rules.value.total) : '—',
    hint: !loaded.value ? 'Memuat…' : rules.value ? `${rules.value.categories} kategori` : 'Gagal memuat',
    to: '/audit-report'
  },
  {
    label: 'Edisi',
    value: edition.value?.edition || '—',
    hint: editionHint.value,
    to: lockedCount.value ? '/plans' : undefined
  }
])

// Proyek aktif yang paling butuh perhatian (skor terendah dulu).
const attention = computed(() => (projects.value || [])
  .filter(isActive)
  .slice()
  .sort((a, b) => (a.health_score ?? 0) - (b.health_score ?? 0))
  .slice(0, 6))

const scoreTone = healthTone
const scoreBar = healthBar

const SEVERITIES = [
  { key: 'CRITICAL', label: 'Kritis', cls: 'bg-rose-500' },
  { key: 'HIGH', label: 'Tinggi', cls: 'bg-orange-500' },
  { key: 'MEDIUM', label: 'Sedang', cls: 'bg-amber-400' },
  { key: 'LOW', label: 'Rendah', cls: 'bg-sky-400' }
]
const severityRows = computed(() => {
  const c = latest.value?.severity_counts || {}
  const get = (k: string) => c[k] ?? c[k.toLowerCase()] ?? 0
  return SEVERITIES.map(s => ({ ...s, count: get(s.key) }))
})
const lastAudit = computed(() => {
  const d = latest.value?.diagnostic?.created_at
  if (!d) return ''
  return new Date(d).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' })
})

const shortcuts = computed(() => {
  const id = activeProjectId.value
  return [
    { label: 'Temuan audit', icon: 'i-lucide-shield-check', to: '/audit-report' },
    { label: 'Todo', icon: 'i-lucide-clipboard-check', to: '/todos' },
    { label: 'Screenshot', icon: 'i-lucide-camera', to: '/screenshots' },
    { label: 'Peta kode', icon: 'i-lucide-map', to: id ? '/code-map' : '/cockpit' }
  ]
})

const steps = [
  { title: 'Daftarkan proyek', text: 'Pilih folder proyek; vibeasik mengenali backend, frontend, dan stack-nya.', cmd: 'vibeasik scan ~/proyek-saya' },
  { title: 'Jalankan audit', text: 'Periksa temuan keamanan, arsitektur, dan API beserta cara memperbaikinya.', cmd: 'vibeasik diagnose ~/proyek-saya' },
  { title: 'Atur port', text: 'Tiap proyek mendapat blok port sendiri; jangan memilih port manual.', cmd: 'vibeasik ports' },
  { title: 'Sambungkan agent', text: 'Pasang vibeasik sebagai server MCP di Claude Code, Cursor, atau Windsurf.', cmd: 'vibeasik mcp install' },
  { title: 'Pusat Pengetahuan', text: 'Opsional: masuk akun untuk menarik paket panduan & checklist.', cmd: 'vibeasik login' }
]

async function copy(cmd: string) {
  try {
    await navigator.clipboard.writeText(cmd)
    toast.add({ title: 'Perintah disalin', description: cmd, color: 'success' })
  } catch {
    toast.add({ title: 'Tidak bisa menyalin', description: 'Salin manual dari kotak perintah.', color: 'warning' })
  }
}
</script>

<template>
  <div class="space-y-8 pb-10">
    <!-- Kepala halaman -->
    <header class="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
      <div class="space-y-1.5">
        <p class="text-xs font-medium text-neutral-500 dark:text-slate-400 flex items-center gap-1.5">
          <UIcon name="i-lucide-lock-keyhole" class="size-3.5" /> Berjalan lokal · kode tidak pernah diunggah
        </p>
        <h2 class="text-2xl sm:text-3xl font-semibold tracking-tight text-neutral-950 dark:text-white">
          Audit kode buatan AI, dari satu tempat
        </h2>
        <p class="text-sm text-neutral-600 dark:text-slate-400 max-w-2xl">
          Pantau kesehatan proyek, tindak lanjuti temuan audit, dan beri agent coding konteks yang benar lewat MCP.
        </p>
      </div>
      <div class="flex gap-2 shrink-0">
        <UButton to="/scan" icon="i-lucide-plus" color="neutral" variant="outline">
          Daftarkan proyek
        </UButton>
        <UButton to="/cockpit" icon="i-lucide-layout-grid" color="neutral">
          Katalog proyek
        </UButton>
      </div>
    </header>

    <!-- KPI -->
    <section class="grid grid-cols-2 lg:grid-cols-4 gap-px overflow-hidden rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-neutral-200 dark:bg-white/[0.08]" aria-label="Status">
      <component
        :is="k.to ? 'NuxtLink' : 'div'"
        v-for="k in kpis"
        :key="k.label"
        :to="k.to"
        class="bg-white dark:bg-[#04111f] p-5 space-y-1 transition-colors"
        :class="k.to ? 'hover:bg-neutral-50 dark:hover:bg-white/[0.03]' : ''"
      >
        <div class="text-xs font-medium text-neutral-500 dark:text-slate-400 flex items-center gap-1.5">
          <span v-if="k.dot" class="size-2 rounded-full" :class="k.dot" />{{ k.label }}
        </div>
        <div class="text-2xl font-semibold tracking-tight tabular-nums text-neutral-950 dark:text-white">
          {{ k.value }}
        </div>
        <div class="text-xs text-neutral-500 dark:text-slate-500 truncate" :title="k.hint">
          {{ k.hint }}
        </div>
      </component>
    </section>

    <div class="grid grid-cols-1 lg:grid-cols-5 gap-6">
      <!-- Proyek aktif -->
      <section class="lg:col-span-3 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] p-6 space-y-5" aria-label="Proyek aktif">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="text-xs font-medium text-neutral-500 dark:text-slate-400">
              Proyek aktif
            </p>
            <h3 class="text-lg font-semibold text-neutral-950 dark:text-white truncate">
              {{ activeProject?.name || 'Belum dipilih' }}
            </h3>
          </div>
          <UButton v-if="activeProject" :to="`/projects/${activeProject.id}`" size="sm" color="neutral" variant="ghost" trailing-icon="i-lucide-arrow-right">
            Buka
          </UButton>
        </div>

        <div v-if="!activeProject" class="rounded-lg border border-dashed border-neutral-300 dark:border-white/10 p-6 text-center space-y-3">
          <UIcon name="i-lucide-folder-search" class="size-8 text-neutral-400 mx-auto" />
          <p class="text-sm text-neutral-600 dark:text-slate-400">
            Pilih proyek di navbar (atau di katalog) untuk melihat skor audit dan temuan terbukanya di sini.
          </p>
          <UButton to="/cockpit" size="sm" color="neutral" variant="outline">
            Pilih dari katalog
          </UButton>
        </div>

        <template v-else>
          <div v-if="latestState === 'loading'" class="h-28 rounded-lg bg-neutral-100 dark:bg-white/5 animate-pulse" />
          <div v-else-if="latestState === 'none'" class="rounded-lg border border-dashed border-neutral-300 dark:border-white/10 p-5 text-sm text-neutral-600 dark:text-slate-400 flex items-center justify-between gap-3">
            Proyek ini belum pernah diaudit.
            <UButton to="/audit-report" size="sm" icon="i-lucide-play">
              Jalankan audit
            </UButton>
          </div>
          <div v-else-if="latestState === 'error'" class="rounded-lg bg-rose-50 dark:bg-rose-950/30 p-4 text-sm text-rose-700 dark:text-rose-300">
            Gagal memuat hasil audit terakhir.
          </div>
          <div v-else-if="latest" class="grid grid-cols-1 sm:grid-cols-[auto_1fr] gap-6 items-center">
            <div class="flex sm:flex-col items-baseline sm:items-start gap-2">
              <span class="text-5xl font-semibold tracking-tight tabular-nums" :class="scoreTone(latest.score)">{{ latest.score }}</span>
              <span class="text-sm text-neutral-500">skor kesehatan / 100</span>
            </div>
            <div class="space-y-3">
              <div class="flex items-baseline justify-between text-sm">
                <span class="text-neutral-700 dark:text-slate-300"><b class="tabular-nums">{{ latest.active_findings_count }}</b> temuan terbuka</span>
                <span class="text-xs text-neutral-500 tabular-nums">{{ latest.resolved_findings_count }} selesai · {{ latest.ignored_findings_count }} diabaikan</span>
              </div>
              <div class="grid grid-cols-4 gap-2">
                <div v-for="s in severityRows" :key="s.key" class="rounded-lg bg-neutral-50 dark:bg-white/[0.03] px-3 py-2">
                  <div class="flex items-center gap-1.5 text-xs text-neutral-500">
                    <span class="size-1.5 rounded-full" :class="s.cls" />{{ s.label }}
                  </div>
                  <div class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">
                    {{ s.count }}
                  </div>
                </div>
              </div>
              <p v-if="lastAudit" class="text-xs text-neutral-500">
                Audit terakhir {{ lastAudit }}
              </p>
            </div>
          </div>

          <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-1">
            <NuxtLink
              v-for="s in shortcuts"
              :key="s.label"
              :to="s.to"
              class="flex items-center gap-2 rounded-lg border border-neutral-200 dark:border-white/[0.08] px-3 h-10 text-sm text-neutral-700 dark:text-slate-300 hover:border-sky-500/50 hover:text-sky-700 dark:hover:text-sky-300 transition-colors"
            >
              <UIcon :name="s.icon" class="size-4 shrink-0 text-neutral-400" />
              <span class="truncate">{{ s.label }}</span>
            </NuxtLink>
          </div>
        </template>
      </section>

      <!-- Perlu perhatian -->
      <section class="lg:col-span-2 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] p-6 space-y-4" aria-label="Proyek perlu perhatian">
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-semibold text-neutral-950 dark:text-white">
            Skor terendah
          </h3>
          <NuxtLink to="/cockpit" class="text-xs text-neutral-500 hover:text-sky-600">Semua proyek</NuxtLink>
        </div>
        <div v-if="!loaded" class="space-y-2">
          <div v-for="i in 4" :key="i" class="h-9 rounded-lg bg-neutral-100 dark:bg-white/5 animate-pulse" />
        </div>
        <p v-else-if="projects === null" class="text-sm text-rose-600">
          Gagal memuat daftar proyek.
        </p>
        <p v-else-if="!attention.length" class="text-sm text-neutral-500">
          Belum ada proyek aktif.
        </p>
        <ul v-else class="divide-y divide-neutral-100 dark:divide-white/[0.06]">
          <li v-for="p in attention" :key="p.id">
            <NuxtLink :to="`/projects/${p.id}`" class="flex items-center gap-3 py-2.5 group">
              <span class="flex-1 min-w-0 text-sm text-neutral-700 dark:text-slate-300 truncate group-hover:text-sky-700 dark:group-hover:text-sky-300">{{ p.name }}</span>
              <span class="w-20 h-1.5 rounded-full bg-neutral-100 dark:bg-white/10 overflow-hidden">
                <span class="block h-full rounded-full" :class="scoreBar(p.health_score ?? 0)" :style="{ width: `${p.health_score ?? 0}%` }" />
              </span>
              <span class="w-8 text-right text-sm font-semibold tabular-nums" :class="scoreTone(p.health_score ?? 0)">{{ p.health_score ?? 0 }}</span>
            </NuxtLink>
          </li>
        </ul>
      </section>
    </div>

    <!-- Panduan singkat -->
    <section id="tutorial" class="scroll-mt-20 space-y-4">
      <div class="flex items-end justify-between gap-3">
        <div>
          <h3 class="text-sm font-semibold text-neutral-950 dark:text-white">
            Mulai dalam lima langkah
          </h3>
          <p class="text-sm text-neutral-500 dark:text-slate-400">
            Perintah terminal; semuanya juga bisa dilakukan dari Studio ini.
          </p>
        </div>
      </div>
      <ol class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
        <li
          v-for="(s, i) in steps"
          :key="s.title"
          class="rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] p-4 flex flex-col gap-2"
        >
          <span class="text-xs font-mono text-neutral-400">0{{ i + 1 }}</span>
          <h4 class="text-sm font-semibold text-neutral-900 dark:text-white">
            {{ s.title }}
          </h4>
          <p class="text-xs text-neutral-600 dark:text-slate-400 flex-1">
            {{ s.text }}
          </p>
          <button
            type="button"
            class="w-full text-left flex items-center justify-between gap-2 rounded-md bg-neutral-950 text-neutral-100 px-2.5 py-1.5 font-mono text-xs hover:bg-neutral-800"
            :aria-label="`Salin perintah ${s.cmd}`"
            @click="copy(s.cmd)"
          >
            <span class="truncate">{{ s.cmd }}</span>
            <UIcon name="i-lucide-copy" class="size-3.5 shrink-0 text-neutral-400" />
          </button>
        </li>
      </ol>
      <p class="text-xs text-neutral-500 dark:text-slate-500 flex flex-wrap items-center gap-x-3 gap-y-1">
        <span class="inline-flex items-center gap-1"><UIcon name="i-lucide-shield-check" class="size-3.5" /> Engine hanya mendengarkan 127.0.0.1 dan wajib token lokal.</span>
        <NuxtLink to="/settings?tab=knowledge_sync" class="underline-offset-2 hover:underline">Berbagi pengetahuan</NuxtLink>
        <a href="https://vibeasik.com/terms" target="_blank" rel="noopener" class="underline-offset-2 hover:underline">Syarat &amp; ketentuan</a>
        <a href="https://github.com/vibeasik/vibeasik" target="_blank" rel="noopener" class="underline-offset-2 hover:underline">GitHub</a>
      </p>
    </section>
  </div>
</template>
