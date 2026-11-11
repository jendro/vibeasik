<script setup lang="ts">
import type { FastScanResult } from '~/types'

defineProps<{ scanResult: FastScanResult }>()

const { copyToClipboard } = useCopyToClipboard()
</script>

<template>
  <div class="space-y-4">
    <div v-if="scanResult.how_to_run.prerequisites && scanResult.how_to_run.prerequisites.length > 0" class="space-y-1.5">
      <label class="block text-xs font-semibold text-neutral-500 uppercase tracking-wider">Prasyarat Runtime:</label>
      <div class="flex flex-wrap gap-1.5">
        <span
          v-for="(p, pIdx) in scanResult.how_to_run.prerequisites"
          :key="pIdx"
          class="px-2 py-0.5 rounded text-xs font-mono bg-neutral-100 dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 border border-neutral-200 dark:border-neutral-700"
        >
          {{ p }}
        </span>
      </div>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs font-mono">
      <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] space-y-1.5">
        <div class="flex items-center justify-between text-neutral-500 text-xs uppercase font-bold">
          <span class="flex items-center gap-1 text-sky-600 dark:text-sky-400">
            <UIcon name="i-lucide-terminal" class="w-3.5 h-3.5" />
            Dev Command
          </span>
          <button
            class="text-sky-600 dark:text-sky-400 hover:underline cursor-pointer"
            @click="copyToClipboard(scanResult.how_to_run.dev_command, 'Dev command disalin!')"
          >
            salin
          </button>
        </div>
        <div class="p-2 rounded bg-neutral-50 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 select-all font-bold text-neutral-900 dark:text-white truncate">
          {{ scanResult.how_to_run.dev_command || 'Tidak terdeteksi' }}
        </div>
      </div>

      <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] space-y-1.5">
        <div class="flex items-center justify-between text-neutral-500 text-xs uppercase font-bold">
          <span class="flex items-center gap-1 text-blue-600 dark:text-blue-400">
            <UIcon name="i-lucide-download" class="w-3.5 h-3.5" />
            Install Dependencies
          </span>
          <button
            class="text-blue-600 dark:text-blue-400 hover:underline cursor-pointer"
            @click="copyToClipboard(scanResult.how_to_run.install_command, 'Install command disalin!')"
          >
            salin
          </button>
        </div>
        <div class="p-2 rounded bg-neutral-50 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 select-all text-neutral-800 dark:text-neutral-200 truncate">
          {{ scanResult.how_to_run.install_command || 'Tidak terdeteksi' }}
        </div>
      </div>

      <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] space-y-1.5">
        <div class="flex items-center justify-between text-neutral-500 text-xs uppercase font-bold">
          <span class="flex items-center gap-1 text-amber-600 dark:text-amber-400">
            <UIcon name="i-lucide-hammer" class="w-3.5 h-3.5" />
            Build Command
          </span>
          <button
            class="text-amber-600 dark:text-amber-400 hover:underline cursor-pointer"
            @click="copyToClipboard(scanResult.how_to_run.build_command, 'Build command disalin!')"
          >
            salin
          </button>
        </div>
        <div class="p-2 rounded bg-neutral-50 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 select-all text-neutral-800 dark:text-neutral-200 truncate">
          {{ scanResult.how_to_run.build_command || 'Tidak terdeteksi' }}
        </div>
      </div>

      <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] space-y-1.5">
        <div class="flex items-center justify-between text-neutral-500 text-xs uppercase font-bold">
          <span class="flex items-center gap-1 text-purple-600 dark:text-purple-400">
            <UIcon name="i-lucide-check-circle-2" class="w-3.5 h-3.5" />
            Test Command
          </span>
          <button
            class="text-purple-600 dark:text-purple-400 hover:underline cursor-pointer"
            @click="copyToClipboard(scanResult.how_to_run.test_command, 'Test command disalin!')"
          >
            salin
          </button>
        </div>
        <div class="p-2 rounded bg-neutral-50 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 select-all text-neutral-800 dark:text-neutral-200 truncate">
          {{ scanResult.how_to_run.test_command || 'Tidak terdeteksi' }}
        </div>
      </div>
    </div>

    <div v-if="scanResult.how_to_run.steps && scanResult.how_to_run.steps.length > 0" class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] space-y-2.5">
      <div class="text-xs font-semibold text-neutral-700 dark:text-neutral-300">
        Langkah Menjalankan (Step-by-Step):
      </div>
      <ul class="space-y-1.5 text-xs text-neutral-600 dark:text-neutral-400">
        <li v-for="(s, sIdx) in scanResult.how_to_run.steps" :key="sIdx" class="flex items-start gap-2">
          <span class="font-bold text-sky-600 dark:text-sky-400 mt-0.5">•</span>
          <span>{{ s }}</span>
        </li>
      </ul>
    </div>
  </div>
</template>
