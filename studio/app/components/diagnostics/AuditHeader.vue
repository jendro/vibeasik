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

// Prompt agent: alternatif "Jadikan todo" tanpa kuota LLM API — agent coding pengguna yang memeriksa & membuat todo.
const promptOpen = ref(false)

const getScoreTextColor = healthTone
const getScoreLabel = healthLabel

const { formatDateTime } = useFormatters()

// Aksi jarang dipakai dikumpulkan di menu "Lainnya" agar baris aksi utama tetap tiga tombol.
const moreItems = computed(() => [
  [
    { label: props.llmReviewBusy && props.llmReviewLabel ? props.llmReviewLabel : 'Review kode oleh LLM', icon: 'i-lucide-sparkles', disabled: !props.project || props.llmReviewBusy, onSelect: () => emit('runLlmReview') },
    { label: props.sweepBusy && props.sweepLabel ? props.sweepLabel : 'Audit & jadikan todo semua (otomatis, LLM)', icon: 'i-lucide-sparkles', disabled: !props.project || props.sweepBusy, onSelect: () => emit('runSweep') }
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
          <!-- Jenis tombol seragam (AppActionButton): Periksa = tanpa LLM, Proses temuan, penanda LLM = pakai kuota API -->
          <AppActionButton
            kind="scan"
            label="Audit standar"
            desc="Audit statis cepat: port, konfigurasi, dan 85 rule standar"
            :loading="diagnosing"
            @click="emit('runAudit')"
          />
          <AppActionButton
            kind="scan"
            label="Deep Check"
            desc="Simulasi latensi rute API, validasi form, dan risiko kueri"
            :loading="deepChecking"
            @click="emit('runDeepCheck')"
          />
          <UButton
            size="sm"
            color="neutral"
            variant="outline"
            icon="i-lucide-clipboard-copy"
            label="Salin prompt agent"
            title="Tanpa kuota LLM API: salin prompt untuk agent coding Anda (Claude Code, Cursor, Cline …) — agent memeriksa temuan di kode lalu membuat todo lewat MCP vibeasik"
            :disabled="!project"
            @click="promptOpen = true"
          />
          <AppActionButton
            kind="process"
            llm
            primary
            :label="pendingCount ? `Jadikan todo (${pendingCount})` : 'Jadikan todo'"
            desc="Susun solusi tiap temuan terbuka lalu jadikan todo, bertahap per batch"
            :loading="resolvingAll"
            :disabled="!pendingCount"
            @click="emit('resolveAll')"
          />
          <UDropdownMenu :items="moreItems" :content="{ align: 'end' }">
            <UButton color="neutral" variant="ghost" size="sm" icon="i-lucide-ellipsis" aria-label="Aksi audit lainnya" />
          </UDropdownMenu>
          <FindingsAgentPromptModal v-if="project" v-model:open="promptOpen" :project-id="project.id" scope="audit" />
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
      <AppActionButton kind="scan" label="Deep Check ulang" desc="Periksa ulang rute & query yang berubah" :loading="deepChecking" class="shrink-0" @click="emit('runDeepCheck')" />
    </div>
  </div>
</template>
