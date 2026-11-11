<script setup lang="ts">
import type { FastScanResult } from '~/types'

defineProps<{ scanResult: FastScanResult }>()
</script>

<template>
  <div class="space-y-3">
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] space-y-3">
      <MarkdownViewer :content="scanResult.ai_analysis_markdown" />
    </div>
  </div>
</template>
