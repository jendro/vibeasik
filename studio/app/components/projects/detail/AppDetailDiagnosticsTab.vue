<script setup lang="ts">
import type { Diagnostic, Finding, ProjectComponent } from '~/types'

const props = defineProps<{
  comp: ProjectComponent
  loading: boolean
  diagnosing: boolean
  componentDiagnostic: Diagnostic | null
  parsedFindings: Finding[]
}>()
const emit = defineEmits<{ run: [] }>()
const handleRunComponentDiagnostics = () => emit('run')

// Satu kartu temuan ≈ 30 simpul DOM → 25 per halaman (anggaran 800 simpul, utils/listPaging).
const findingPaging = useListPaging(() => props.parsedFindings, { nodesPerRow: 30, queryKey: 'fpg' })
</script>

<template>
  <div class="space-y-6">
    <!-- Action Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4.5 rounded-xl bg-white dark:bg-[#04111f] border border-neutral-200 dark:border-white/[0.08]">
      <div>
        <h4 class="font-bold text-neutral-900 dark:text-white text-sm flex items-center gap-2">
          <UIcon name="i-lucide-shield-alert" class="w-4 h-4 text-sky-400" />
          <span>Audit Kesehatan Kode Aplikasi</span>
        </h4>
        <p class="text-xs text-neutral-500 mt-0.5">
          Audit statis integritas path, konfigurasi .env, test suites, dan deteksi bentrok port untuk {{ comp.name }}.
        </p>
      </div>

      <UButton
        icon="i-lucide-activity"
        color="primary"
        size="xs"
        :loading="diagnosing"
        title="Jalankan audit kesehatan kode aplikasi ini sekarang"
        @click="handleRunComponentDiagnostics"
      >
        {{ componentDiagnostic ? 'Audit Ulang Aplikasi' : 'Jalankan Diagnostik' }}
      </UButton>
    </div>

    <!-- State Belum Diaudit -->
    <div v-if="!componentDiagnostic && !diagnosing" class="p-8 text-center rounded-xl border border-dashed border-neutral-200 dark:border-white/[0.08] bg-white/50 dark:bg-neutral-900/50 space-y-3">
      <div class="w-10 h-10 mx-auto rounded-full bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center text-neutral-400">
        <UIcon name="i-lucide-stethoscope" class="w-5 h-5" />
      </div>
      <div class="space-y-1">
        <h4 class="text-sm font-semibold text-neutral-900 dark:text-white">
          Aplikasi ini belum diaudit secara terpisah
        </h4>
        <p class="text-xs text-neutral-500 max-w-md mx-auto">
          Klik tombol di atas untuk menjalankan diagnosa integritas kode, git status, dan deteksi port untuk aplikasi ini.
        </p>
      </div>
      <UButton size="xs" color="primary" @click="handleRunComponentDiagnostics">
        Jalankan Diagnostik Sekarang
      </UButton>
    </div>

    <!-- Hasil Diagnostik -->
    <div v-else-if="componentDiagnostic" class="space-y-4">
      <!-- Score Banner -->
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] flex items-center gap-4">
        <div
          class="w-14 h-14 rounded-xl flex flex-col items-center justify-center font-mono font-bold border shrink-0"
          :class="getScoreColor(componentDiagnostic.score)"
        >
          <span class="text-xl">{{ componentDiagnostic.score }}</span>
          <span class="text-xs font-normal">Score</span>
        </div>
        <div class="space-y-0.5 min-w-0">
          <div class="font-bold text-sm text-neutral-900 dark:text-white">
            {{ componentDiagnostic.score >= 90 ? 'Kondisi Kode Optimal' : componentDiagnostic.score >= 70 ? 'Perlu Optimasi' : 'Peringatan Kesehatan' }}
          </div>
          <p class="text-xs text-neutral-500 line-clamp-2">
            {{ componentDiagnostic.summary }}
          </p>
        </div>
      </div>

      <!-- Findings List -->
      <div id="app-finding-list" class="space-y-3 scroll-mt-32">
        <h5 class="text-xs font-semibold text-neutral-400">
          Temuan Audit ({{ parsedFindings.length }})
        </h5>

        <div v-if="parsedFindings.length === 0" class="p-6 rounded-xl border border-dashed border-neutral-200 dark:border-white/[0.08] text-center text-xs text-neutral-500 bg-white/50 dark:bg-neutral-900/50">
          Tidak ada masalah terdeteksi. Konfigurasi dan port aplikasi aman.
        </div>

        <div
          v-for="(f, idx) in findingPaging.pagedItems.value"
          :key="idx"
          class="p-4 rounded-xl border space-y-2 text-xs"
          :class="f.severity === 'CRITICAL'
            ? 'border-rose-300 dark:border-rose-900/60 bg-rose-50/40 dark:bg-rose-950/20'
            : f.severity === 'WARNING'
              ? 'border-amber-300 dark:border-amber-900/60 bg-amber-50/40 dark:bg-amber-950/20'
              : 'border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f]'"
        >
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <UIcon
                :name="f.severity === 'CRITICAL' ? 'i-lucide-alert-triangle' : f.severity === 'WARNING' ? 'i-lucide-alert-circle' : 'i-lucide-info'"
                class="w-4 h-4"
                :class="f.severity === 'CRITICAL' ? 'text-rose-500' : f.severity === 'WARNING' ? 'text-amber-500' : 'text-blue-500'"
              />
              <span class="font-bold text-neutral-900 dark:text-neutral-100">{{ f.title }}</span>
              <span class="text-xs font-mono px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 text-neutral-500">
                {{ f.category }}
              </span>
            </div>

            <UBadge size="xs" :color="f.severity === 'CRITICAL' ? 'error' : f.severity === 'WARNING' ? 'warning' : 'neutral'">
              {{ f.severity }}
            </UBadge>
          </div>

          <p class="text-neutral-600 dark:text-neutral-300 leading-relaxed">
            {{ f.description }}
          </p>

          <div class="flex items-center gap-1.5 font-mono text-sky-700 dark:text-sky-400 pt-1">
            <UIcon name="i-lucide-check-circle" class="w-3.5 h-3.5 shrink-0" />
            <span>Saran: {{ f.suggestion }}</span>
          </div>
        </div>
        <AppListPager
          v-model:page="findingPaging.page.value"
          v-model:per-page="findingPaging.perPage.value"
          :total="findingPaging.total.value"
          :per-page-options="findingPaging.perPageOptions"
          :range-start="findingPaging.rangeStart.value"
          :range-end="findingPaging.rangeEnd.value"
          noun="temuan"
          scroll-target="app-finding-list"
        />
      </div>
    </div>
  </div>
</template>
