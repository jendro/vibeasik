<script setup lang="ts">
import type { Project, RecapTodoBrief } from '~/types'
import { CATEGORY_LABEL } from '~/utils/recap'
import { FINDING_AREAS, nextImprovementStep, type ImprovementStep } from '~/utils/improvementFlow'

// Alur memperbaiki aplikasi di tab Ringkasan: 1 periksa menyeluruh → 2 kerjakan todo → 3 lihat hasil.
// Langkah yang perlu dikerjakan sekarang ditandai, supaya pengguna tahu harus mulai dari mana.
const props = defineProps<{ project: Project }>()
const emit = defineEmits<{ switchTab: [tab: string] }>()

const projectId = computed(() => props.project.id)
const days = ref(30)
const { recap, loading, failed, load } = useOwnerRecap(projectId, days)
const { runDiagnostics, runDeepCheck } = useApi()
const toast = useToast()

const now = ref(Date.now())
const next = computed(() => recap.value ? nextImprovementStep(recap.value, now.value) : null)

const lastAudit = computed(() => {
  const at = recap.value?.findings.last_audit_at
  if (!at) return 'Belum pernah diperiksa'
  const d = Math.floor((now.value - new Date(at).getTime()) / 86_400_000)
  const when = new Date(at).toLocaleString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
  return `Diperiksa ${d <= 0 ? 'hari ini' : `${d} hari lalu`} · ${when}`
})

const areas = computed(() => FINDING_AREAS.map(a => ({ ...a, n: recap.value?.findings.open_by_area?.[a.key] || 0 })))
const openCats = computed(() => (recap.value?.categories || [])
  .filter(c => c.open > 0)
  .sort((a, b) => b.open - a.open)
  .slice(0, 4)
  .map(c => ({ ...c, label: CATEGORY_LABEL[c.category] || CATEGORY_LABEL.general! })))
const latestDone = computed(() => (recap.value?.highlights || []).slice(0, 3))
const healthDelta = computed(() => {
  const h = recap.value?.health || []
  return h.length >= 2 ? h[h.length - 1]!.score - h[h.length - 2]!.score : 0
})

// Pemeriksaan menyeluruh = audit kode (langsung) + Deep Check (antrean engine, selesai belakangan).
const scanning = ref(false)
const deepRunning = ref(false)
const runFullCheck = async () => {
  scanning.value = true
  try {
    await runDiagnostics(projectId.value)
    await load(true)
    now.value = Date.now()
    deepRunning.value = true
    toast.add({ title: 'Audit kode selesai', description: 'Deep Check (rute, query, aset) berjalan di antrean; angka diperbarui saat selesai.', icon: 'i-lucide-scan-search', color: 'success' })
    runDeepCheck(projectId.value)
      .then(() => load(true))
      .catch((err: any) => toast.add({ title: 'Deep Check gagal', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' }))
      .finally(() => { deepRunning.value = false })
  } catch (err: any) {
    toast.add({ title: 'Pemeriksaan gagal', description: err?.data?.error || err?.message || 'Terjadi kesalahan', icon: 'i-lucide-alert-circle', color: 'error' })
  } finally {
    scanning.value = false
  }
}

const changesFor = ref<RecapTodoBrief | null>(null)
const changesOpen = computed({
  get: () => changesFor.value !== null,
  set: (v: boolean) => { if (!v) changesFor.value = null }
})

const stepClass = (s: ImprovementStep) => [
  'min-w-0 rounded-xl border bg-white dark:bg-[#04111f] p-4 sm:p-5 flex flex-col gap-4',
  next.value?.step === s ? 'border-sky-400 dark:border-sky-500/60 ring-1 ring-sky-400/30' : 'border-neutral-200 dark:border-white/[0.08]'
]
const fmtDate = (iso?: string) => iso ? new Date(iso).toLocaleDateString('id-ID', { day: 'numeric', month: 'short' }) : ''
</script>

<template>
  <section class="space-y-3" aria-labelledby="improvement-flow-title">
    <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-2">
      <div class="space-y-0.5">
        <h2 id="improvement-flow-title" class="text-base font-semibold text-neutral-900 dark:text-white">
          Tingkatkan aplikasi
        </h2>
        <p class="text-xs text-neutral-500 dark:text-slate-400">
          Periksa menyeluruh, kerjakan todo hasil temuan, lalu cek perubahan kodenya.
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-neutral-500 dark:text-slate-400" aria-label="Jenis tombol">
        <span class="inline-flex items-center gap-1"><UIcon name="i-lucide-scan-search" class="size-3.5" />Periksa: tanpa LLM</span>
        <span class="inline-flex items-center gap-1"><UIcon name="i-lucide-list-checks" class="size-3.5" />Proses temuan</span>
        <span class="inline-flex items-center gap-1"><UIcon name="i-lucide-sparkles" class="size-3.5 text-violet-500" />LLM: pakai kuota API</span>
      </div>
    </div>

    <div v-if="loading && !recap" class="grid grid-cols-1 lg:grid-cols-3 gap-3">
      <div v-for="i in 3" :key="i" class="h-64 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] animate-pulse" />
    </div>

    <UAlert v-else-if="failed && !recap" color="error" variant="subtle" icon="i-lucide-alert-circle" :title="failed">
      <template #actions>
        <UButton size="xs" color="neutral" variant="outline" @click="load(true)">
          Coba lagi
        </UButton>
      </template>
    </UAlert>

    <template v-else-if="recap && next">
      <!-- Langkah berikutnya -->
      <div class="rounded-xl border border-sky-200 dark:border-sky-500/30 bg-sky-50/70 dark:bg-sky-950/20 px-4 py-3 text-sm flex items-start gap-3">
        <UIcon :name="next.icon" class="size-4 mt-0.5 shrink-0 text-sky-600 dark:text-sky-400" />
        <p class="text-neutral-800 dark:text-slate-200">
          <span class="font-semibold">Langkah berikutnya: {{ next.title }}.</span>
          {{ next.reason }}
        </p>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-3">
        <!-- 1. Periksa menyeluruh -->
        <article :class="stepClass('check')">
          <header class="space-y-1">
            <p class="text-xs font-medium text-neutral-500 dark:text-slate-400">
              1 · Periksa menyeluruh
            </p>
            <div class="flex items-baseline gap-2">
              <span class="text-3xl font-semibold tabular-nums leading-none" :class="recap.findings.open_critical ? 'text-rose-600 dark:text-rose-400' : recap.findings.open ? 'text-neutral-900 dark:text-white' : 'text-neutral-300 dark:text-slate-600'">{{ recap.findings.open }}</span>
              <span class="text-sm text-neutral-600 dark:text-slate-300">temuan belum jadi todo</span>
            </div>
            <p class="text-xs text-neutral-500 dark:text-slate-400">
              {{ lastAudit }}<template v-if="recap.findings.open_critical">
                · <span class="text-rose-600 dark:text-rose-400">{{ recap.findings.open_critical }} kritis</span>
              </template>
            </p>
          </header>
          <ul class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-1 2xl:grid-cols-2 gap-x-4 gap-y-1.5 text-xs">
            <li v-for="a in areas" :key="a.key" class="min-w-0">
              <NuxtLink
                :to="`/audit-report?area=${a.key}`"
                class="-mx-1.5 px-1.5 py-0.5 rounded-md flex items-center justify-between gap-2 min-w-0 hover:bg-neutral-100 dark:hover:bg-white/[0.05] focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
                :title="`${a.desc} — buka temuan area ini di Audit & Diagnostik`"
              >
                <span class="flex items-center gap-1.5 min-w-0 text-neutral-600 dark:text-slate-300">
                  <UIcon :name="a.icon" class="size-3.5 shrink-0 text-neutral-400" />
                  <span class="truncate">{{ a.label }}</span>
                </span>
                <span class="tabular-nums font-medium" :class="a.n ? 'text-neutral-900 dark:text-white' : 'text-neutral-300 dark:text-slate-600'">{{ a.n }}</span>
              </NuxtLink>
            </li>
          </ul>
          <div class="mt-auto flex flex-wrap gap-2">
            <AppActionButton
              kind="scan"
              :label="deepRunning ? 'Deep Check berjalan…' : 'Periksa menyeluruh'"
              desc="Audit kode lalu Deep Check (rute, query, aset) di antrean engine"
              :primary="next.step === 'check' && next.action === 'scan'"
              :loading="scanning || deepRunning"
              @click="runFullCheck"
            />
            <AppActionButton
              kind="process"
              label="Tinjau temuan"
              desc="Buka Audit & Diagnostik untuk menjadikan temuan todo"
              :primary="next.step === 'check' && next.action === 'process'"
              to="/audit-report"
            />
          </div>
        </article>

        <!-- 2. Kerjakan todo -->
        <article :class="stepClass('work')">
          <header class="space-y-1">
            <p class="text-xs font-medium text-neutral-500 dark:text-slate-400">
              2 · Kerjakan todo
            </p>
            <div class="flex items-baseline gap-2">
              <span class="text-3xl font-semibold tabular-nums leading-none" :class="recap.todos.open ? 'text-neutral-900 dark:text-white' : 'text-neutral-300 dark:text-slate-600'">{{ recap.todos.open }}</span>
              <span class="text-sm text-neutral-600 dark:text-slate-300">todo terbuka</span>
            </div>
            <p class="text-xs text-neutral-500 dark:text-slate-400">
              {{ recap.todos.open_from_audit }} dari audit · {{ recap.todos.in_progress }} sedang dikerjakan<template v-if="recap.todos.open_high_priority">
                · <span class="text-amber-600 dark:text-amber-400">{{ recap.todos.open_high_priority }} prioritas tinggi</span>
              </template>
            </p>
          </header>
          <ul v-if="openCats.length" class="space-y-1.5 text-xs">
            <li v-for="c in openCats" :key="c.category" class="flex items-center justify-between gap-2">
              <span class="text-neutral-600 dark:text-slate-300">{{ c.label }}</span>
              <span class="tabular-nums font-medium text-neutral-900 dark:text-white">{{ c.open }}</span>
            </li>
          </ul>
          <p v-else class="text-xs text-neutral-500 dark:text-slate-400">
            Tidak ada todo terbuka.
          </p>
          <p v-if="recap.todos.suggested" class="text-xs text-neutral-500 dark:text-slate-400">
            {{ recap.todos.suggested }} saran todo menunggu persetujuan.
          </p>
          <div class="mt-auto flex flex-wrap gap-2">
            <AppActionButton
              kind="process"
              label="Buka todo"
              desc="Daftar todo proyek ini; kirim ke agent dari sana"
              :primary="next.step === 'work'"
              to="/todos"
            />
            <UButton size="sm" color="neutral" variant="ghost" icon="i-lucide-list-todo" to="/queue" title="Todo & pipeline yang sedang berjalan">
              Antrean
            </UButton>
          </div>
        </article>

        <!-- 3. Hasil -->
        <article :class="stepClass('result')">
          <header class="space-y-1">
            <p class="text-xs font-medium text-neutral-500 dark:text-slate-400">
              3 · Hasil {{ recap.days }} hari
            </p>
            <div class="flex items-baseline gap-2">
              <span class="text-3xl font-semibold tabular-nums leading-none" :class="recap.todos.done_in_period ? 'text-emerald-600 dark:text-emerald-400' : 'text-neutral-300 dark:text-slate-600'">{{ recap.todos.done_in_period }}</span>
              <span class="text-sm text-neutral-600 dark:text-slate-300">todo selesai</span>
            </div>
            <p class="text-xs text-neutral-500 dark:text-slate-400">
              {{ recap.findings.resolved_in_period }} temuan beres<template v-if="recap.health_score">
                · skor kesehatan {{ recap.health_score }}<span v-if="healthDelta" :class="healthDelta > 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'"> ({{ healthDelta > 0 ? '+' : '' }}{{ healthDelta }})</span>
              </template>
            </p>
          </header>
          <ul v-if="latestDone.length" class="space-y-2">
            <li v-for="t in latestDone" :key="t.id" class="text-xs min-w-0">
              <p class="truncate text-neutral-800 dark:text-slate-200" :title="t.title">
                {{ t.title }}
              </p>
              <div class="flex items-center gap-2 text-neutral-500 dark:text-slate-400">
                <span>{{ fmtDate(t.completed_at) }}</span>
                <!-- Commit belum tercatat (todo lama) dicari engine lewat trailer branch saat perubahannya dibuka -->
                <button
                  type="button"
                  class="inline-flex items-center gap-1 text-sky-700 dark:text-sky-300 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500 rounded"
                  :class="t.commit_sha ? 'font-mono' : ''"
                  :title="t.commit_sha ? `Lihat kode yang berubah di commit ${t.commit_sha}` : 'Cari commit todo ini dan lihat kode yang berubah'"
                  @click="changesFor = t"
                >
                  <UIcon name="i-lucide-git-commit-horizontal" class="size-3.5" />{{ t.commit_sha ? t.commit_sha.slice(0, 7) : 'Lihat perubahan' }}
                </button>
              </div>
            </li>
          </ul>
          <p v-else class="text-xs text-neutral-500 dark:text-slate-400">
            Belum ada todo yang selesai.
          </p>
          <div class="mt-auto flex flex-wrap gap-2">
            <UButton size="sm" color="neutral" :variant="next.step === 'result' ? 'solid' : 'outline'" trailing-icon="i-lucide-arrow-right" @click="emit('switchTab', 'recap')">
              Rekap lengkap
            </UButton>
          </div>
        </article>
      </div>
    </template>

    <TodosTodoChangesModal v-if="changesFor" v-model:open="changesOpen" :todo-id="changesFor.id" :todo-title="changesFor.title" />
  </section>
</template>
