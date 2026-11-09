<script setup lang="ts">
import type { Project } from '~/types'

defineProps<{
  project: Project | null
  projects?: Project[]
  score: number
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

const getScoreTextColor = (score: number) => {
  if (score >= 90) return 'text-sky-600 dark:text-sky-400'
  if (score >= 70) return 'text-amber-600 dark:text-amber-400'
  return 'text-rose-600 dark:text-rose-400'
}

const getScoreLabel = (score: number) => {
  if (score >= 90) return 'Kondisi Prima & Siap Produksi'
  if (score >= 70) return 'Perlu Refactoring Ringan'
  return 'Kritis: Ditemukan Anti-Pattern / Smells'
}

const { formatDateTime } = useFormatters()
</script>

<template>
  <div class="p-6 bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 rounded-2xl shadow-xs space-y-5">
    <!-- Top Row: Title on the left, Health Score on the right -->
    <div class="flex items-start justify-between gap-4">
      <div class="flex items-start gap-3.5 min-w-0">
        <span class="p-2.5 rounded-xl bg-violet-50 dark:bg-violet-950/40 text-violet-600 dark:text-violet-400 border border-violet-200 dark:border-violet-800 shrink-0 mt-0.5">
          <UIcon name="i-lucide-file-search" class="w-5 h-5" />
        </span>
        <div class="space-y-1.5 min-w-0">
          <div class="flex items-center gap-2.5 flex-wrap">
            <h1 class="text-xl font-bold text-neutral-900 dark:text-white">
              Laporan Audit Diagnostik & Code Hygiene
            </h1>

            <!-- Last Deep Check Pill -->
            <button
              v-if="hasLastDeepCheck"
              type="button"
              class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-md text-xs font-medium bg-neutral-100 dark:bg-neutral-800/80 hover:bg-neutral-200/80 dark:hover:bg-neutral-700/80 text-neutral-700 dark:text-neutral-300 border border-neutral-200 dark:border-neutral-700 transition-colors cursor-pointer"
              :title="`Klik untuk melihat simulasi rute API & latensi (Last Deep Check: ${formatDateTime(lastDeepCheckAt)})`"
              @click="emit('openSimulations')"
            >
              <UIcon name="i-lucide-layers" class="w-3.5 h-3.5 text-primary-500" />
              <span>Last Deep Check: <strong>{{ formatDateTime(lastDeepCheckAt) }}</strong></span>
              <span v-if="lastDeepCheckScore !== undefined && lastDeepCheckScore !== null" class="text-neutral-400">·</span>
              <span v-if="lastDeepCheckScore !== undefined && lastDeepCheckScore !== null" class="font-mono font-bold text-primary-600 dark:text-primary-400">
                {{ lastDeepCheckScore }}/100
              </span>
              <span
                v-if="deepPendingCount && deepPendingCount > 0"
                class="ml-1 px-1.5 py-0.5 rounded text-xs font-mono font-bold bg-rose-100 dark:bg-rose-950/60 text-rose-700 dark:text-rose-300 border border-rose-200 dark:border-rose-800"
                title="Jumlah temuan Deep Check yang belum diselesaikan"
              >
                {{ deepPendingCount }} isu
              </span>
              <UIcon name="i-lucide-chevron-right" class="w-3 h-3 text-neutral-400 ml-0.5" />
            </button>

            <span
              v-else
              class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-xs font-medium bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300 border border-amber-200 dark:border-amber-800"
            >
              <UIcon name="i-lucide-info" class="w-3.5 h-3.5 text-amber-500" />
              <span>Belum pernah Deep Check</span>
            </span>
          </div>

          <p class="text-xs text-neutral-500 dark:text-neutral-400 max-w-2xl leading-relaxed">
            Pemeriksaan statis mendalam dan simulasi real-time untuk waktu eksekusi router API, validasi form/payload, dan kerentanan kueri database.
          </p>
        </div>
      </div>

      <!-- Health Score -->
      <div v-if="project" class="text-right shrink-0">
        <div class="text-4xl sm:text-5xl font-black font-mono tracking-tight leading-none" :class="getScoreTextColor(score)">
          {{ score }}
        </div>
        <div class="mt-1 text-right space-y-0.5">
          <div class="text-xs font-bold uppercase tracking-wider text-neutral-400 dark:text-neutral-500">
            Health Score
          </div>
          <div class="text-xs font-semibold" :class="getScoreTextColor(score)">
            {{ getScoreLabel(score) }}
          </div>
        </div>
      </div>
    </div>

    <!-- Alert Banner: Code Changes Detected Since Last Deep Check -->
    <div
      v-if="hasChangesSinceDeepCheck"
      class="p-3.5 rounded-xl border border-amber-200 dark:border-amber-900/60 bg-amber-50/70 dark:bg-amber-950/20 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs"
    >
      <div class="flex items-start sm:items-center gap-2.5 min-w-0">
        <UIcon name="i-lucide-alert-triangle" class="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5 sm:mt-0" />
        <div class="text-neutral-800 dark:text-neutral-200">
          <span class="font-bold text-amber-900 dark:text-amber-300">Perubahan Kode Terdeteksi: </span>
          <span>{{ changeSummary || 'Terdapat berkas termodifikasi sejak Deep Check terakhir.' }}</span>
          <span class="text-neutral-500 dark:text-neutral-400 ml-1">Disarankan menjalankan Deep Check kembali.</span>
        </div>
      </div>

      <UButton
        color="warning"
        variant="solid"
        size="xs"
        icon="i-lucide-layers"
        :loading="deepChecking"
        class="shrink-0 self-end sm:self-auto font-medium"
        @click="emit('runDeepCheck')"
      >
        Jalankan Deep Check Ulang
      </UButton>
    </div>

    <!-- Action Buttons Row: Pepet Pojok Kanan -->
    <div class="flex items-center justify-end gap-2.5 pt-4 border-t border-neutral-100 dark:border-neutral-800/80 flex-wrap">
      <!-- Daftar Cek Audit Criteria Drawer -->
      <UButton
        color="neutral"
        variant="outline"
        icon="i-lucide-list-checks"
        size="sm"
        title="Lihat daftar lengkap kriteria, celah keamanan, dan smelly code/design yang dicek sistem"
        @click="emit('openCriteria')"
      >
        Daftar Cek Audit
      </UButton>

      <!-- Download Laporan Last Deep Check (Tanpa Perlu Jalankan Ulang) -->
      <UButton
        v-if="hasLastDeepCheck"
        color="neutral"
        variant="outline"
        icon="i-lucide-download"
        size="sm"
        title="Download laporan Deep Check terakhir dalam format Markdown tanpa perlu menjalankan deep check ulang"
        @click="emit('downloadDeepCheck', 'markdown')"
      >
        Download Laporan Deep
      </UButton>

      <!-- Simulasi Router & Latensi Modal/Drawer Trigger -->
      <UButton
        v-if="hasLastDeepCheck"
        color="neutral"
        variant="outline"
        icon="i-lucide-activity"
        size="sm"
        title="Buka hasil simulasi real-time latensi rute API, validasi form, dan analisis risiko kueri"
        @click="emit('openSimulations')"
      >
        Simulasi Router
      </UButton>

      <!-- Audit Standar -->
      <UButton
        color="neutral"
        variant="outline"
        icon="i-lucide-refresh-cw"
        size="sm"
        :loading="diagnosing"
        title="Jalankan audit statis cepat (port, config, smells standar)"
        @click="emit('runAudit')"
      >
        Audit Standar
      </UButton>

      <!-- Deep Check Button -->
      <UButton
        color="primary"
        variant="outline"
        icon="i-lucide-scan-search"
        size="sm"
        :loading="deepChecking"
        title="Jalankan Deep Check: simulasi real-time latensi rute API, validasi form, dan kerentanan kueri"
        @click="emit('runDeepCheck')"
      >
        <span class="font-bold">Deep Check</span>
      </UButton>

      <!-- Review Audit LLM: berjalan di antrean engine (non-blocking), progres tampil di bar Antrean -->
      <UButton
        color="neutral"
        variant="outline"
        icon="i-lucide-sparkles"
        size="sm"
        :loading="llmReviewBusy"
        :disabled="!project || llmReviewBusy"
        title="Masukkan review kode oleh LLM ke antrean: berkas prioritas direview dengan checklist sesuai bahasanya (Go, Node, Python, PHP, JVM, Vue/React, Android, Flutter)"
        @click="emit('runLlmReview')"
      >
        {{ llmReviewBusy && llmReviewLabel ? llmReviewLabel : 'Review LLM' }}
      </UButton>

      <!-- Audit & Selesaikan Semua: putaran audit ulang + penyelesaian di antrean engine -->
      <UButton
        color="neutral"
        variant="outline"
        icon="i-lucide-repeat"
        size="sm"
        :loading="sweepBusy"
        :disabled="!project || sweepBusy"
        title="Audit Standar + Deep Check, lalu temuan terbuka langsung diselesaikan via LLM. Diulang otomatis sampai audit ulang tidak memunculkan temuan baru (maks 3 putaran). Berjalan di antrean, progres tampil di bar Antrean."
        @click="emit('runSweep')"
      >
        {{ sweepBusy && sweepLabel ? sweepLabel : 'Audit & Selesaikan Semua' }}
      </UButton>

      <!-- Selesaikan Bertahap via LLM -->
      <UButton
        color="primary"
        variant="solid"
        icon="i-lucide-sparkles"
        size="sm"
        :loading="resolvingAll"
        title="Buka panel eksekusi solusi bertahap per batch dari frontend"
        @click="emit('resolveAll')"
      >
        <span>Selesaikan Bertahap via LLM</span>
        <UBadge
          v-if="pendingCount && pendingCount > 0"
          size="xs"
          color="neutral"
          variant="subtle"
          class="ml-1 px-1.5 py-0 font-mono text-xs"
        >
          {{ pendingCount }}
        </UBadge>
      </UButton>
    </div>
  </div>
</template>
