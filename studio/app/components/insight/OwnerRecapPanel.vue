<script setup lang="ts">
import type { Project } from '~/types'
import type { StatItem } from '~/components/AppStatStrip.vue'
import { CATEGORY_LABEL } from '~/utils/recap'

// Rekap untuk pemilik proyek. `compact` = versi ringkas di tab Ringkasan (kalimat, angka utama, satu grafik).
const props = defineProps<{ project: Project, compact?: boolean }>()
const emit = defineEmits<{ switchTab: [tab: string] }>()

const projectId = computed(() => props.project.id)
const days = ref(30)
const { recap, loading, failed, load } = useOwnerRecap(projectId, days)

const RANGES = [{ value: 7, label: '7 hari' }, { value: 30, label: '30 hari' }, { value: 90, label: '90 hari' }]

const labels = computed(() => recapDayLabels((recap.value?.timeline || []).map(p => p.date)))
const completedSeries = computed(() => (recap.value?.timeline || []).map(p => p.completed))
const openSeries = computed(() => (recap.value?.timeline || []).map(p => p.open))
const healthSeries = computed(() => (recap.value?.health || []).map(h => h.score))
const healthLabels = computed(() => recapDayLabels((recap.value?.health || []).map(h => h.date)))

const healthTone = (score: number) => score >= 80 ? 'text-emerald-600 dark:text-emerald-400' : score >= 60 ? 'text-amber-600 dark:text-amber-400' : 'text-rose-600 dark:text-rose-400'

const stats = computed<StatItem[]>(() => {
  const r = recap.value
  if (!r) return []
  return [
    { label: 'Progres', value: `${r.todos.percent}%`, hint: `${r.todos.done} dari ${r.todos.total} pekerjaan` },
    { label: `Selesai ${r.days} hari`, value: r.todos.done_in_period, hint: `${r.todos.created_in_period} pekerjaan baru masuk` },
    { label: 'Bug ditutup', value: r.bugs.done, hint: r.bugs.open ? `${r.bugs.open} masih terbuka` : 'tidak ada yang tertunda', dot: 'bg-emerald-500' },
    { label: 'Perbaikan algoritma', value: r.algorithm.done, hint: r.algorithm.open ? `${r.algorithm.open} dijadwalkan` : 'performa & query', dot: 'bg-sky-500' },
    {
      label: 'Temuan audit terbuka', value: r.findings.open,
      hint: r.findings.open_critical ? `${r.findings.open_critical} kritis · ${r.findings.resolved} sudah beres` : `${r.findings.resolved} sudah beres`,
      tone: r.findings.open_critical ? 'text-rose-600 dark:text-rose-400' : undefined
    },
    { label: 'Skor kesehatan', value: r.health_score || '—', hint: r.todos.scored_count ? `nilai QA rata-rata ${r.todos.avg_score}` : 'dari audit kode', tone: r.health_score ? healthTone(r.health_score) : undefined }
  ]
})

const categories = computed(() => (recap.value?.categories || []).map(c => ({
  ...c,
  label: CATEGORY_LABEL[c.category] || CATEGORY_LABEL.general!,
  percent: c.done + c.open > 0 ? Math.round(c.done / (c.done + c.open) * 100) : 0
})))

const SEVERITY_LABEL: Record<string, string> = { CRITICAL: 'Kritis', WARNING: 'Peringatan', INFO: 'Info', OPTIMIZATION: 'Optimasi' }
const severities = computed(() => Object.entries(recap.value?.findings.open_by_severity || {})
  .map(([k, n]) => ({ key: k, label: SEVERITY_LABEL[k] || k, n }))
  .sort((a, b) => b.n - a.n))

const fmtDate = (iso?: string) => iso ? new Date(iso).toLocaleDateString('id-ID', { day: 'numeric', month: 'short' }) : ''
const card = 'min-w-0 rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] p-5'
</script>

<template>
  <div class="space-y-4">
    <div v-if="loading && !recap" class="h-44 rounded-2xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] animate-pulse" />

    <div v-else-if="failed && !recap" class="rounded-2xl border border-rose-300 dark:border-rose-800 bg-rose-50 dark:bg-rose-950/30 p-5 text-sm text-rose-800 dark:text-rose-200 flex items-center justify-between gap-3">
      <span>{{ failed }}</span>
      <UButton size="xs" color="neutral" variant="outline" @click="load(true)">
        Coba lagi
      </UButton>
    </div>

    <template v-else-if="recap">
      <section :class="card" class="space-y-4">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0 space-y-1">
            <span class="font-mono text-xs uppercase tracking-wider text-sky-600 dark:text-sky-400">Rekap pemilik</span>
            <p class="text-base font-semibold leading-snug text-neutral-900 dark:text-white max-w-3xl">
              {{ recapHeadline(recap) }}
            </p>
          </div>
          <div class="flex items-center gap-2 shrink-0">
            <div class="inline-flex rounded-lg border border-neutral-200 dark:border-white/10 p-0.5" role="group" aria-label="Rentang rekap">
              <button
                v-for="r in RANGES"
                :key="r.value"
                type="button"
                class="h-7 px-2.5 rounded-md text-xs transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
                :class="days === r.value ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900 font-medium' : 'text-neutral-600 dark:text-slate-300 hover:bg-neutral-100 dark:hover:bg-white/[0.06]'"
                :aria-pressed="days === r.value"
                @click="days = r.value"
              >
                {{ r.label }}
              </button>
            </div>
            <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-refresh-cw" :loading="loading" aria-label="Segarkan rekap" @click="load(true)" />
          </div>
        </div>

        <div class="space-y-1.5">
          <div class="h-2 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden" role="progressbar" :aria-valuenow="recap.todos.percent" aria-valuemin="0" aria-valuemax="100" aria-label="Progres pekerjaan">
            <div class="h-full bg-sky-500 rounded-full" :style="{ width: `${recap.todos.percent}%` }" />
          </div>
          <div class="flex justify-between text-xs text-neutral-500">
            <span>{{ recap.todos.done }} selesai</span>
            <span>{{ recap.todos.in_progress }} sedang dikerjakan · {{ recap.todos.open - recap.todos.in_progress }} menunggu</span>
          </div>
        </div>
      </section>

      <AppStatStrip :items="stats" />

      <div class="grid grid-cols-1 gap-3" :class="compact ? '' : 'lg:grid-cols-2'">
        <section :class="card" class="space-y-2">
          <div class="flex items-baseline justify-between gap-2">
            <h3 class="text-sm font-semibold text-neutral-900 dark:text-white">
              Pekerjaan selesai per hari
            </h3>
            <span class="text-xs text-neutral-500">{{ recap.todos.done_in_period }} dalam {{ recap.days }} hari</span>
          </div>
          <InsightMiniSeriesChart :values="completedSeries" :labels="labels" mode="bar" tone="sky" unit="selesai" label="Jumlah pekerjaan selesai per hari" />
        </section>

        <section v-if="!compact" :class="card" class="space-y-2">
          <div class="flex items-baseline justify-between gap-2">
            <h3 class="text-sm font-semibold text-neutral-900 dark:text-white">
              Sisa pekerjaan terbuka
            </h3>
            <span class="text-xs text-neutral-500">turun = utang kerja berkurang</span>
          </div>
          <InsightMiniSeriesChart :values="openSeries" :labels="labels" mode="line" tone="amber" unit="terbuka" label="Jumlah pekerjaan terbuka di akhir tiap hari" />
        </section>
      </div>

      <div v-if="compact" class="flex justify-end">
        <UButton size="sm" color="neutral" variant="outline" trailing-icon="i-lucide-arrow-right" @click="emit('switchTab', 'recap')">
          Lihat rekap lengkap
        </UButton>
      </div>

      <template v-else>
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-3">
          <section :class="card" class="space-y-3">
            <h3 class="text-sm font-semibold text-neutral-900 dark:text-white">
              Progres per kategori
            </h3>
            <p v-if="!categories.length" class="text-sm text-neutral-500">
              Belum ada pekerjaan berkategori.
            </p>
            <div v-for="c in categories" :key="c.category" class="space-y-1">
              <div class="flex justify-between text-sm">
                <span class="text-neutral-700 dark:text-neutral-200">{{ c.label }}</span>
                <span class="tabular-nums text-neutral-500">{{ c.done }} / {{ c.done + c.open }}</span>
              </div>
              <div class="h-1.5 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
                <div class="h-full bg-sky-500 rounded-full" :style="{ width: `${c.percent}%` }" />
              </div>
            </div>
          </section>

          <InsightRecapWorkList
            title="Bug ditutup"
            icon="i-lucide-bug-off"
            icon-class="text-emerald-500"
            :stat="recap.bugs"
            empty="Belum ada bug yang ditutup."
            hint="Todo dari temuan audit/scanner atau berjudul perbaikan galat."
          />
          <InsightRecapWorkList
            title="Perbaikan algoritma"
            icon="i-lucide-zap"
            icon-class="text-sky-500"
            :stat="recap.algorithm"
            empty="Belum ada perbaikan performa yang selesai."
            hint="Kategori performa, optimasi query, N+1, indeks, cache."
            action-label="Lihat Big-O tiap rute"
            @action="emit('switchTab', 'cost')"
          />
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-3 gap-3">
          <section :class="card" class="lg:col-span-2 space-y-2">
            <div class="flex items-baseline justify-between gap-2">
              <h3 class="text-sm font-semibold text-neutral-900 dark:text-white">
                Skor kesehatan kode
              </h3>
              <span class="text-xs text-neutral-500">{{ recap.health.length }} audit terakhir · skala 0–100</span>
            </div>
            <InsightMiniSeriesChart
              v-if="healthSeries.length"
              :values="healthSeries"
              :labels="healthLabels"
              mode="line"
              tone="emerald"
              :max="100"
              label="Skor kesehatan kode tiap audit"
            />
            <p v-else class="text-sm text-neutral-500 py-6 text-center">
              Belum ada audit. Jalankan audit kode agar tren kesehatan muncul di sini.
            </p>
          </section>

          <section :class="card" class="space-y-3">
            <h3 class="text-sm font-semibold text-neutral-900 dark:text-white">
              Temuan audit
            </h3>
            <dl class="grid grid-cols-3 gap-2 text-center">
              <div>
                <dd class="text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">
                  {{ recap.findings.open }}
                </dd>
                <dt class="text-xs text-neutral-500">
                  Terbuka
                </dt>
              </div>
              <div>
                <dd class="text-xl font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">
                  {{ recap.findings.resolved }}
                </dd>
                <dt class="text-xs text-neutral-500">
                  Dibereskan
                </dt>
              </div>
              <div>
                <dd class="text-xl font-semibold tabular-nums text-neutral-500">
                  {{ recap.findings.ignored }}
                </dd>
                <dt class="text-xs text-neutral-500">
                  Diabaikan
                </dt>
              </div>
            </dl>
            <ul v-if="severities.length" class="space-y-1 border-t border-neutral-200 dark:border-white/[0.08] pt-3">
              <li v-for="s in severities" :key="s.key" class="flex justify-between text-sm">
                <span class="text-neutral-600 dark:text-neutral-300">{{ s.label }} terbuka</span>
                <span class="tabular-nums font-medium" :class="s.key === 'CRITICAL' ? 'text-rose-600 dark:text-rose-400' : 'text-neutral-900 dark:text-white'">{{ s.n }}</span>
              </li>
            </ul>
          </section>
        </div>

        <section v-if="recap.highlights.length" :class="card" class="space-y-3">
          <h3 class="text-sm font-semibold text-neutral-900 dark:text-white">
            Terakhir diselesaikan
          </h3>
          <ul class="divide-y divide-neutral-200 dark:divide-white/[0.06]">
            <li v-for="t in recap.highlights" :key="t.id" class="flex items-center gap-3 py-2 text-sm">
              <UIcon name="i-lucide-check" class="size-4 shrink-0 text-sky-500" />
              <span class="min-w-0 flex-1 truncate text-neutral-800 dark:text-neutral-100" :title="t.title">{{ t.title }}</span>
              <span v-if="t.score != null" class="shrink-0 text-xs tabular-nums text-neutral-500">QA {{ t.score }}</span>
              <span class="shrink-0 text-xs text-neutral-500 w-14 text-right">{{ fmtDate(t.completed_at) }}</span>
            </li>
          </ul>
        </section>
      </template>
    </template>
  </div>
</template>
