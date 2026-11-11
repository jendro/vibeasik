<script setup lang="ts">
import type { ProjectSynergyMap } from '~/types'

defineProps<{
  synergyMap: ProjectSynergyMap | null
  loading: boolean
  generatingLLM: boolean
  jobLabel: string
}>()
const emit = defineEmits<{
  generate: []
  refresh: [force: boolean]
  download: []
  copy: []
}>()
const handleGenerateWithLLM = () => emit('generate')
const loadSynergyData = (force: boolean) => emit('refresh', force)
const downloadSynergyJson = () => emit('download')
const copySummaryMarkdown = () => emit('copy')
</script>

<template>
  <div class="p-5 sm:p-6 rounded-xl border border-neutral-200/80 dark:border-white/[0.08] bg-white dark:bg-[#04111f] space-y-4">
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
                class="font-mono text-xs flex items-center gap-1"
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
          <span>{{ generatingLLM ? jobLabel : 'Rangkai AI' }}</span>
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
        <span class="text-xs text-neutral-400 uppercase font-semibold">Halaman Frontend</span>
        <div class="font-bold text-neutral-800 dark:text-neutral-200 font-mono text-base mt-0.5 tabular-nums">
          {{ synergyMap?.total_pages || 0 }} halaman
        </div>
      </div>
      <div class="p-3 rounded-xl bg-neutral-50/70 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800/60">
        <span class="text-xs text-neutral-400 uppercase font-semibold">Alur Request API</span>
        <div class="font-bold font-mono text-base mt-0.5 text-sky-600 dark:text-sky-400 tabular-nums">
          {{ synergyMap?.total_flows || 0 }} alur terhubung
        </div>
      </div>
      <div class="p-3 rounded-xl bg-neutral-50/70 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800/60">
        <span class="text-xs text-neutral-400 uppercase font-semibold">Node Backend</span>
        <div class="font-bold text-neutral-800 dark:text-neutral-200 font-mono text-base mt-0.5 tabular-nums">
          {{ synergyMap?.backend_nodes?.length || 0 }} servis API
        </div>
      </div>
      <div class="p-3 rounded-xl bg-neutral-50/70 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800/60">
        <span class="text-xs text-neutral-400 uppercase font-semibold">Tabel Database</span>
        <div class="font-bold text-neutral-800 dark:text-neutral-200 font-mono text-base mt-0.5 tabular-nums">
          {{ synergyMap?.total_tables || 0 }} tabel
        </div>
      </div>
    </div>
  </div>
</template>
