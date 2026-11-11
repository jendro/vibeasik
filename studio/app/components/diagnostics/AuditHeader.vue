<script setup lang="ts">
import type { Project } from '~/types'

const props = defineProps<{
  project: Project | null
  projects?: Project[]
  score: number | null
  loading?: boolean
  resolvingAll?: boolean
  diagnosing?: boolean
  deepChecking?: boolean
  pendingCount?: number
  deepPendingCount?: number
  lastDeepCheckAt?: string | null
  lastDeepCheckScore?: number | null
  hasChangesSinceDeepCheck?: boolean
  changeSummary?: string
  hasLastDeepCheck?: boolean
  llmReviewBusy?: boolean
  llmReviewLabel?: string
  sweepBusy?: boolean
  sweepLabel?: string
}>()

const emit = defineEmits<{
  (e: 'switchProject', id: number): void
  (e: 'runAudit'): void
  (e: 'runDeepCheck'): void
  (e: 'downloadDeepCheck', format: 'markdown' | 'json'): void
  (e: 'openSimulations'): void
  (e: 'resolveAll'): void
  (e: 'openCriteria'): void
  (e: 'runLlmReview'): void
  (e: 'runSweep'): void
}>()

const getScoreTextColor = healthTone
const getScoreLabel = healthLabel

const { formatDateTime } = useFormatters()

// Aksi jarang dipakai dikumpulkan di menu "Lainnya" agar baris aksi utama tetap tiga tombol.
const moreItems = computed(() => [
  [
    { label: props.llmReviewBusy && props.llmReviewLabel ? props.llmReviewLabel : 'Review kode oleh LLM', icon: 'i-lucide-sparkles', disabled: !props.project || props.llmReviewBusy, onSelect: () => emit('runLlmReview') },
    { label: props.sweepBusy && props.sweepLabel ? props.sweepLabel : 'Audit & selesaikan semua (otomatis)', icon: 'i-lucide-repeat', disabled: !props.project || props.sweepBusy, onSelect: () => emit('runSweep') }
  ],
  [
    ...(props.hasLastDeepCheck
      ? [
          { label: 'Simulasi router & latensi', icon: 'i-lucide-activity', onSelect: () => emit('openSimulations') },
          { label: 'Unduh laporan Deep Check (.md)', icon: 'i-lucide-download', onSelect: () => emit('downloadDeepCheck', 'markdown' as const) }
        ]
      : []),
    { label: 'Daftar cek audit (kriteria)', icon: 'i-lucide-list-checks', onSelect: () => emit('openCriteria') }
  ]
])
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-col lg:flex-row lg:items-end justify-between gap-5 pb-5 border-b border-neutral-200 dark:border-white/[0.07]">
      <div class="min-w-0 space-y-1.5">
        <h2 class="text-xl sm:text-2xl font-semibold tracking-tight text-neutral-950 dark:text-white">
          Audit &amp; Diagnostik
        </h2>
        <p class="text-sm text-neutral-600 dark:text-slate-400 max-w-2xl">
          Temuan keamanan, arsitektur, API, dan kebersihan kode dari 85 rule, Deep Check (simulasi rute & kueri), dan review LLM.
        </p>
        <div class="flex flex-wrap items-center gap-x-4 gap-y-1 pt-1 text-xs text-neutral-500 dark:text-slate-400">
          <button
            v-if="hasLastDeepCheck"
            type="button"
            class="inline-flex items-center gap-1.5 hover:text-neutral-900 dark:hover:text-white"
            :title="'Buka simulasi rute API & latensi dari Deep Check terakhir'"
            @click="emit('openSimulations')"
          >
            <UIcon name="i-lucide-layers" class="size-3.5" />
            Deep Check terakhir {{ formatDateTime(lastDeepCheckAt) }}
            <span v-if="lastDeepCheckScore != null" class="tabular-nums" title="Skor pada saat Deep Check itu dijalankan; skor kesehatan sekarang ada di kanan">· skor saat itu <b class="font-medium text-neutral-800 dark:text-slate-200">{{ lastDeepCheckScore }}/100</b></span>
            <span v-if="deepPendingCount" class="rounded-md bg-rose-500/10 px-1.5 text-rose-700 dark:text-rose-300 tabular-nums">{{ deepPendingCount }} isu</span>
          </button>
          <span v-else class="inline-flex items-center gap-1.5 text-amber-600 dark:text-amber-400">
            <UIcon name="i-lucide-info" class="size-3.5" />Belum pernah Deep Check
          </span>
        </div>
      </div>

      <div class="flex flex-col sm:flex-row lg:flex-col items-start sm:items-center lg:items-end gap-4 shrink-0">
        <div v-if="project" class="flex items-center gap-4 rounded-xl border border-neutral-200 dark:border-white/[0.08] px-4 py-3">
          <div class="text-right">
            <div class="text-3xl font-semibold tracking-tight tabular-nums leading-none" :class="getScoreTextColor(score)">
              {{ loading ? '…' : score ?? '—' }}
            </div>
            <div class="mt-1 text-xs text-neutral-500">
              skor kesehatan
            </div>
          </div>
          <div class="w-px self-stretch bg-neutral-200 dark:bg-white/[0.08]" />
          <div class="text-xs space-y-1">
            <div class="font-medium" :class="getScoreTextColor(score)">
              {{ getScoreLabel(score) }}
            </div>
            <div class="text-neutral-500 tabular-nums">
              {{ pendingCount || 0 }} temuan bisa diselesaikan
            </div>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <UButton
            color="neutral"
            variant="outline"
            icon="i-lucide-refresh-cw"
            size="sm"
            :loading="diagnosing"
            title="Audit statis cepat: port, konfigurasi, dan 85 rule standar"
            @click="emit('runAudit')"
          >
            Audit standar
          </UButton>
          <UButton
            color="neutral"
            variant="outline"
            icon="i-lucide-scan-search"
            size="sm"
            :loading="deepChecking"
            title="Deep Check: simulasi latensi rute API, validasi form, dan risiko kueri"
            @click="emit('runDeepCheck')"
          >
            Deep Check
          </UButton>
          <UButton
            color="neutral"
            icon="i-lucide-wand-sparkles"
            size="sm"
            :loading="resolvingAll"
            :disabled="!pendingCount"
            title="Selesaikan temuan terbuka bertahap per batch lewat LLM"
            @click="emit('resolveAll')"
          >
            Selesaikan via LLM
            <span v-if="pendingCount" class="rounded bg-white/15 dark:bg-neutral-900/15 px-1.5 tabular-nums">{{ pendingCount }}</span>
          </UButton>
          <UDropdownMenu :items="moreItems" :content="{ align: 'end' }">
            <UButton color="neutral" variant="ghost" size="sm" icon="i-lucide-ellipsis" aria-label="Aksi audit lainnya" />
          </UDropdownMenu>
        </div>
      </div>
    </div>

    <!-- Kode berubah sejak Deep Check terakhir -->
    <div
      v-if="hasChangesSinceDeepCheck"
      class="px-4 py-3 rounded-xl border border-amber-300/60 dark:border-amber-800/50 bg-amber-50 dark:bg-amber-950/20 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-sm"
    >
      <div class="flex items-start sm:items-center gap-2.5 min-w-0 text-neutral-800 dark:text-slate-200">
        <UIcon name="i-lucide-git-compare" class="size-4 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5 sm:mt-0" />
        <span><span class="font-medium">Kode berubah sejak Deep Check terakhir.</span> {{ changeSummary || 'Ada berkas yang diubah.' }}</span>
      </div>
      <UButton color="warning" variant="soft" size="sm" icon="i-lucide-scan-search" :loading="deepChecking" class="shrink-0" @click="emit('runDeepCheck')">
        Deep Check ulang
      </UButton>
    </div>
  </div>
</template>
