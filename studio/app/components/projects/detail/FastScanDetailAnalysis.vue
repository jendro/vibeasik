<script setup lang="ts">
import type { FastScanResult } from '~/types'

defineProps<{ scanResult: FastScanResult }>()

const { copyToClipboard } = useCopyToClipboard()
</script>

<template>
  <div class="space-y-3">
    <div class="p-5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] space-y-3">
      <div class="flex items-center justify-between pb-2 border-b border-neutral-100 dark:border-white/[0.08]">
        <span class="text-xs font-bold text-neutral-800 dark:text-neutral-200 flex items-center gap-2">
          <UIcon name="i-lucide-bot" class="w-4 h-4 text-sky-400" />
          <span>Laporan Lengkap Hasil Analisis Fast Scan LLM</span>
        </span>
        <UButton
          icon="i-lucide-copy"
          size="xs"
          color="neutral"
          variant="ghost"
          @click="copyToClipboard(scanResult.ai_analysis_markdown, 'Laporan analisis disalin!')"
        >
          Salin Laporan
        </UButton>
      </div>
      <MarkdownViewer :content="scanResult.ai_analysis_markdown" />
    </div>
  </div>
</template>
