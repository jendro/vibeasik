<script setup lang="ts">
import type { FastScanResult } from '~/types'

defineProps<{ scanResult: FastScanResult }>()

const { copyToClipboard: copyText } = useCopyToClipboard()
const copyToClipboard = (text: string, label = 'Copied!') => copyText(text, label)
</script>

<template>
  <div class="space-y-3">
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-2">
      <div class="text-xs font-bold text-neutral-900 dark:text-white">
        Deskripsi Struktur & Arsitektur Modul:
      </div>
      <p class="text-xs leading-relaxed text-neutral-600 dark:text-neutral-400 whitespace-pre-line">
        {{ scanResult.folder_structure_summary }}
      </p>
    </div>

    <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-900 text-neutral-200 space-y-2">
      <div class="flex items-center justify-between text-xs font-mono text-neutral-400">
        <span>Struktur Direktori Komponen</span>
        <button
          class="text-sky-400 hover:underline cursor-pointer"
          @click="copyToClipboard(scanResult.directory_tree, 'Tree copied!')"
        >
          copy tree
        </button>
      </div>
      <pre class="font-mono text-xs max-h-72 overflow-y-auto leading-relaxed text-sky-300/90 whitespace-pre">{{ scanResult.directory_tree }}</pre>
    </div>
  </div>
</template>
