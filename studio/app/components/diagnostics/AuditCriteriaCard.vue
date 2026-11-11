<script setup lang="ts">
import type { AuditCriterion } from '~/utils/auditCriteria'

defineProps<{ item: AuditCriterion }>()

const severityBadgeClass = (sev: string) => {
  switch (sev.toLowerCase()) {
    case 'critical':
      return 'text-rose-700 bg-rose-100 dark:bg-rose-950/60 dark:text-rose-300 border-rose-200 dark:border-rose-800'
    case 'high':
      return 'text-amber-700 bg-amber-100 dark:bg-amber-950/60 dark:text-amber-300 border-amber-200 dark:border-amber-800'
    case 'warning':
      return 'text-amber-700 bg-amber-100 dark:bg-amber-950/60 dark:text-amber-300 border-amber-200 dark:border-amber-800'
    case 'info':
      return 'text-blue-700 bg-blue-100 dark:bg-blue-950/60 dark:text-blue-300 border-blue-200 dark:border-blue-800'
    default:
      return 'text-neutral-600 bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-400 border-neutral-200 dark:border-neutral-700'
  }
}

const scopeBadgeClass = (scope: string) => {
  if (scope.includes('Backend')) {
    return 'bg-violet-100 dark:bg-violet-950/50 text-violet-700 dark:text-violet-300 border-violet-200 dark:border-violet-800'
  }
  if (scope.includes('Frontend')) {
    return 'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-300 border-emerald-200 dark:border-emerald-800'
  }
  if (scope.includes('Mobile')) {
    return 'bg-sky-100 dark:bg-sky-950/50 text-sky-700 dark:text-sky-300 border-sky-200 dark:border-sky-800'
  }
  if (scope.includes('Database')) {
    return 'bg-amber-100 dark:bg-amber-950/50 text-amber-700 dark:text-amber-300 border-amber-200 dark:border-amber-800'
  }
  return 'bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 border-neutral-200 dark:border-neutral-700'
}
</script>

<template>
  <div
    class="p-4 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-neutral-900/90 space-y-3 hover:border-neutral-300 dark:hover:border-neutral-700 transition-colors"
  >
    <!-- Top Row: ID, Title, Severity, Scope -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-neutral-100 dark:border-neutral-800/80 pb-2.5">
      <div class="flex items-center gap-2 flex-wrap min-w-0">
        <span class="px-2 py-0.5 rounded font-mono font-bold text-xs bg-neutral-100 dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 border border-neutral-200 dark:border-neutral-700 shrink-0">
          {{ item.id }}
        </span>
        <h3 class="font-bold text-sm text-neutral-900 dark:text-white">
          {{ item.title }}
        </h3>
      </div>

      <div class="flex items-center gap-1.5 shrink-0">
        <!-- Deep Check Badge -->
        <span
          v-if="item.is_deep"
          class="px-2 py-0.5 rounded text-xs font-mono font-semibold bg-primary-50 dark:bg-primary-950/60 text-primary-700 dark:text-primary-300 border border-primary-200 dark:border-primary-800 flex items-center gap-1 shrink-0"
          title="Kriteria ini hanya diuji saat Deep Check dijalankan"
        >
          <UIcon name="i-lucide-layers" class="w-3 h-3 text-primary-500" />
          <span>DEEP</span>
        </span>

        <!-- Scope Badge -->
        <span class="px-2 py-0.5 rounded text-xs font-semibold border" :class="scopeBadgeClass(item.scope)">
          {{ item.scope }}
        </span>
        <!-- Severity Badge -->
        <span class="px-2 py-0.5 rounded text-xs font-semibold border" :class="severityBadgeClass(item.severity)">
          {{ item.severity }}
        </span>
      </div>
    </div>

    <!-- Rule Description -->
    <div class="space-y-1">
      <div class="text-xs font-semibold text-neutral-500 dark:text-neutral-400 flex items-center gap-1">
        <UIcon name="i-lucide-check-circle" class="w-3.5 h-3.5 text-emerald-500" />
        <span>Standar / Aturan (Rule):</span>
      </div>
      <p class="text-xs text-neutral-800 dark:text-neutral-200 leading-relaxed font-sans">
        {{ item.rule }}
      </p>
    </div>

    <!-- Rationale / Dampak Risiko Sistem -->
    <div class="space-y-1 p-2.5 rounded-lg bg-amber-50/60 dark:bg-amber-950/20 border border-amber-200/60 dark:border-amber-900/40">
      <div class="text-xs font-semibold text-amber-800 dark:text-amber-400 flex items-center gap-1">
        <UIcon name="i-lucide-alert-triangle" class="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" />
        <span>Rasional & Dampak Kerusakan Sistem:</span>
      </div>
      <p class="text-xs text-amber-900 dark:text-amber-200 leading-relaxed font-sans">
        {{ item.rationale }}
      </p>
    </div>

    <!-- Inspection Check -->
    <div class="space-y-1 p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-200/60 dark:border-neutral-700/60">
      <div class="text-xs font-semibold text-neutral-600 dark:text-neutral-300 flex items-center gap-1">
        <UIcon name="i-lucide-search" class="w-3.5 h-3.5 text-neutral-500" />
        <span>Metode Pemeriksaan Otomatis (Inspection Check):</span>
      </div>
      <p class="text-xs font-mono text-neutral-600 dark:text-neutral-300">
        {{ item.inspection_check }}
      </p>
    </div>

    <!-- Suggested Action / Solusi -->
    <div class="space-y-1 p-2.5 rounded-lg bg-sky-50/60 dark:bg-sky-950/20 border border-sky-200/60 dark:border-sky-900/40">
      <div class="text-xs font-semibold text-sky-800 dark:text-sky-300 flex items-center gap-1">
        <UIcon name="i-lucide-wrench" class="w-3.5 h-3.5 text-sky-500" />
        <span>Saran Solusi & Remediasi:</span>
      </div>
      <p class="text-xs text-sky-900 dark:text-sky-200 font-sans leading-relaxed">
        {{ item.suggested_action }}
      </p>
    </div>

    <!-- Code Example (Before vs After) if present -->
    <div v-if="item.example_code" class="mt-2 space-y-1.5 pt-1">
      <div class="text-xs font-semibold text-neutral-400 dark:text-neutral-500">
        Contoh Implementasi:
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-2 text-xs font-mono">
        <div class="p-2 rounded bg-rose-50/70 dark:bg-rose-950/20 border border-rose-200/60 dark:border-rose-900/40 space-y-1 overflow-x-auto">
          <span class="text-xs font-semibold text-rose-700 dark:text-rose-400">❌ Sebelum (Smelly / Rentan):</span>
          <pre class="text-rose-900 dark:text-rose-300 whitespace-pre-wrap leading-tight">{{ item.example_code.bad }}</pre>
        </div>
        <div class="p-2 rounded bg-emerald-50/70 dark:bg-emerald-950/20 border border-emerald-200/60 dark:border-emerald-900/40 space-y-1 overflow-x-auto">
          <span class="text-xs font-semibold text-emerald-700 dark:text-emerald-400">✅ Sesudah (Solusi Bersih):</span>
          <pre class="text-emerald-900 dark:text-emerald-300 whitespace-pre-wrap leading-tight">{{ item.example_code.good }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>
