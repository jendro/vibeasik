<script setup lang="ts">
import type { FastScanResult } from '~/types'

defineProps<{ scanResult: FastScanResult }>()
</script>

<template>
  <div class="space-y-4">
    <!-- Framework & Engine -->
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3 shadow-2xs">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-cpu" class="w-4 h-4 text-primary-600 dark:text-primary-400" />
          <span class="text-xs font-bold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">Framework & Engine HTTP</span>
        </div>
        <TechBadge
          :tech="scanResult.architecture?.framework || scanResult.detected_params.framework"
          size="sm"
        />
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs font-mono">
        <div class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200 dark:border-neutral-700/60">
          <div class="text-xs uppercase font-bold text-neutral-400">
            Framework Name
          </div>
          <div class="mt-1 flex items-center gap-1.5 font-semibold text-neutral-800 dark:text-neutral-100">
            <TechBadge :tech="scanResult.architecture?.framework || scanResult.detected_params.framework || 'Standard Go'" size="xs" />
          </div>
        </div>

        <div class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200 dark:border-neutral-700/60 sm:col-span-2">
          <div class="text-xs uppercase font-bold text-neutral-400">
            Engine / Architecture Type
          </div>
          <div class="mt-1 font-semibold text-neutral-800 dark:text-neutral-100 truncate" :title="scanResult.architecture?.framework_type">
            {{ scanResult.architecture?.framework_type || 'HTTP Server & RESTful API Engine' }}
          </div>
        </div>
      </div>
    </div>

    <!-- Design Pattern Card -->
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3 shadow-2xs">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-network" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
          <span class="text-xs font-bold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">Desain Pola Arsitektur (Design Pattern)</span>
        </div>
        <UBadge size="xs" color="success" variant="subtle">
          {{ scanResult.architecture?.design_pattern || 'MVC Pattern' }}
        </UBadge>
      </div>

      <div class="p-3.5 rounded-lg bg-sky-50/70 dark:bg-sky-950/40 border border-sky-200/70 dark:border-sky-900/40 space-y-1.5 text-xs">
        <div class="font-bold text-sky-900 dark:text-sky-200">
          {{ scanResult.architecture?.design_pattern || 'Model-Controller Architecture' }}
        </div>
        <p class="text-sky-800 dark:text-sky-300 text-xs leading-relaxed whitespace-pre-line">
          {{ scanResult.architecture?.pattern_details || scanResult.folder_structure_summary }}
        </p>
      </div>
    </div>

    <!-- Plugins Card -->
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3 shadow-2xs">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-blocks" class="w-4 h-4 text-purple-600 dark:text-purple-400" />
          <span class="text-xs font-bold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">Plugins, Middlewares & Libraries</span>
        </div>
        <span class="text-xs text-neutral-400 font-mono">
          {{ scanResult.architecture?.plugins?.length || 0 }} library terdeteksi
        </span>
      </div>

      <div v-if="scanResult.architecture?.plugins && scanResult.architecture.plugins.length > 0" class="flex flex-wrap gap-2">
        <TechBadge
          v-for="(plug, pIdx) in scanResult.architecture.plugins"
          :key="pIdx"
          :tech="plug"
          size="sm"
        />
      </div>
      <div v-else class="text-xs text-neutral-500 italic">
        Tidak ada plugin atau middleware pihak ketiga tambahan yang terdeteksi.
      </div>
    </div>
  </div>
</template>
