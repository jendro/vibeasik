<script setup lang="ts">
import type { FastScanResult } from '~/types'

defineProps<{ scanResult: FastScanResult }>()

const { copyToClipboard } = useCopyToClipboard()
</script>

<template>
  <div class="space-y-3">
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-neutral-900 text-neutral-200">
      <div class="flex items-center justify-between text-xs font-mono text-neutral-400 mb-2">
        <span>Directory Tree Snapshot</span>
        <button
          class="text-sky-400 hover:underline cursor-pointer"
          @click="copyToClipboard(scanResult.directory_tree, 'Tree disalin!')"
        >
          salin tree
        </button>
      </div>
      <pre class="font-mono text-xs max-h-96 overflow-y-auto leading-relaxed text-neutral-300 whitespace-pre">{{ scanResult.directory_tree }}</pre>
    </div>
  </div>
</template>
