<script setup lang="ts">
import type { ArchitectureEvaluation, Project } from '~/types'

const props = defineProps<{
  activeProject: Project | null
  evaluations: ArchitectureEvaluation[]
  selectedComponentId: number | null
  evaluating: boolean
}>()

const emit = defineEmits<{
  (e: 'selectComponent', id: number): void
  (e: 'refresh'): void
}>()

const currentEval = computed(() => {
  if (!props.evaluations.length) return null
  if (props.selectedComponentId) {
    return props.evaluations.find(e => e.component_id === props.selectedComponentId) || props.evaluations[0]
  }
  return props.evaluations[0]
})

const getScoreColor = (score: number) => {
  if (score >= 80) return 'text-sky-600 dark:text-sky-400'
  if (score >= 60) return 'text-blue-600 dark:text-blue-400'
  if (score >= 40) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

const getScoreBg = (score: number) => {
  if (score >= 80) return 'bg-sky-50 dark:bg-sky-950/40 border-sky-500/20'
  if (score >= 60) return 'bg-blue-50 dark:bg-blue-950/40 border-blue-500/20'
  if (score >= 40) return 'bg-amber-50 dark:bg-amber-950/40 border-amber-500/20'
  return 'bg-red-50 dark:bg-red-950/40 border-red-500/20'
}

const getPriorityBadge = (p: string) => {
  if (p === 'urgent') return 'bg-red-100 text-red-800 dark:bg-red-900/60 dark:text-red-300'
  if (p === 'high') return 'bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-300'
  if (p === 'medium') return 'bg-blue-100 text-blue-800 dark:bg-blue-900/60 dark:text-blue-300'
  return 'bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300'
}
</script>

<template>
  <div class="space-y-6">
    <!-- Component Selector if multiple components -->
    <div v-if="evaluations.length > 1" class="flex items-center gap-2 overflow-x-auto pb-1 no-scrollbar">
      <span class="text-xs font-medium text-neutral-500 shrink-0">Pilih Komponen:</span>
      <button
        v-for="ev in evaluations"
        :key="ev.component_id"
        type="button"
        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer shrink-0"
        :class="currentEval?.component_id === ev.component_id
          ? 'bg-sky-600 text-white shadow-xs'
          : 'bg-white dark:bg-neutral-900 text-neutral-600 dark:text-neutral-400 border border-neutral-200 dark:border-neutral-800'"
        @click="emit('selectComponent', ev.component_id)"
      >
        <UIcon name="i-lucide-server" class="w-3.5 h-3.5" />
        <span>{{ ev.component_name }}</span>
        <span class="text-[10px] opacity-80">({{ ev.maturity_score }} pts)</span>
      </button>
    </div>

    <!-- Empty State -->
    <div v-if="!currentEval" class="text-center py-12 p-6 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
      <UIcon name="i-lucide-layers" class="w-12 h-12 text-neutral-400 mx-auto mb-3" />
      <h3 class="text-sm font-bold text-neutral-800 dark:text-neutral-200">
        Belum Ada Evaluasi Komponen Backend
      </h3>
      <p class="text-xs text-neutral-500 mt-1 max-w-md mx-auto">
        Pilih proyek aktif yang memiliki komponen backend di dropdown navbar, lalu klik tombol 'Audit Arsitektur'.
      </p>
      <button
        type="button"
        class="mt-4 px-4 py-2 text-xs font-semibold rounded-lg bg-sky-600 text-white hover:bg-sky-500 cursor-pointer"
        @click="emit('refresh')"
      >
        Jalankan Audit Sekarang
      </button>
    </div>

    <!-- Evaluation Results -->
    <div v-else class="space-y-6">
      <!-- Top Card: Score & Pattern -->
      <div class="p-5 rounded-xl border transition-all duration-200 shadow-2xs" :class="getScoreBg(currentEval.maturity_score)">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <div class="flex items-center gap-2">
              <span class="text-xs font-bold uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
                Komponen Backend: {{ currentEval.component_name }}
              </span>
              <span class="text-[11px] font-mono px-2 py-0.5 rounded bg-neutral-200/80 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300">
                {{ currentEval.language }} • {{ currentEval.framework || 'Custom' }}
              </span>
            </div>
            <h2 class="text-lg sm:text-xl font-bold text-neutral-900 dark:text-white mt-1">
              Pola Terdeteksi: {{ currentEval.detected_pattern_name }}
            </h2>
            <p class="text-xs text-neutral-600 dark:text-neutral-400 mt-0.5 font-mono truncate max-w-xl">
              {{ currentEval.absolute_path }}
            </p>
          </div>

          <!-- Big Score Meter -->
          <div class="flex items-center gap-3 self-end sm:self-center">
            <div class="text-right">
              <span class="text-[10px] text-neutral-500 uppercase tracking-wider block">Skor Kematangan</span>
              <span class="text-2xl sm:text-3xl font-extrabold tracking-tight" :class="getScoreColor(currentEval.maturity_score)">
                {{ currentEval.maturity_score }}<span class="text-sm font-normal text-neutral-400">/100</span>
              </span>
            </div>
            <div class="w-12 h-12 rounded-xl flex items-center justify-center border" :class="getScoreBg(currentEval.maturity_score)">
              <UIcon name="i-lucide-award" class="w-6 h-6" :class="getScoreColor(currentEval.maturity_score)" />
            </div>
          </div>
        </div>
      </div>

      <!-- 4 Score Cards -->
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
          <span class="text-[11px] font-medium text-neutral-500 block mb-1">Router Separation</span>
          <div class="flex items-baseline justify-between">
            <span class="text-xl font-bold text-neutral-900 dark:text-white">{{ currentEval.metrics.router_score }} / 25</span>
            <UIcon name="i-lucide-network" class="w-4 h-4 text-sky-400" />
          </div>
        </div>

        <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
          <span class="text-[11px] font-medium text-neutral-500 block mb-1">Thin Controller</span>
          <div class="flex items-baseline justify-between">
            <span class="text-xl font-bold text-neutral-900 dark:text-white">{{ currentEval.metrics.controller_score }} / 25</span>
            <UIcon name="i-lucide-cpu" class="w-4 h-4 text-blue-500" />
          </div>
        </div>

        <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
          <span class="text-[11px] font-medium text-neutral-500 block mb-1">Service & Repo Layer</span>
          <div class="flex items-baseline justify-between">
            <span class="text-xl font-bold text-neutral-900 dark:text-white">{{ currentEval.metrics.service_score }} / 25</span>
            <UIcon name="i-lucide-database" class="w-4 h-4 text-purple-500" />
          </div>
        </div>

        <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
          <span class="text-[11px] font-medium text-neutral-500 block mb-1">MCP & AI Readiness</span>
          <div class="flex items-baseline justify-between">
            <span class="text-xl font-bold text-neutral-900 dark:text-white">{{ currentEval.metrics.ai_mcp_score }} / 25</span>
            <UIcon name="i-lucide-bot" class="w-4 h-4 text-amber-500" />
          </div>
        </div>
      </div>

      <!-- Traits Checklist & Findings Grid -->
      <div class="grid grid-cols-1 lg:grid-cols-12 gap-6">
        <!-- Traits (5 cols) -->
        <div class="lg:col-span-5 p-5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-3">
          <h3 class="text-xs font-bold uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
            Karakteristik Arsitektur Terdeteksi
          </h3>
          <ul class="space-y-2.5 text-xs">
            <li class="flex items-center justify-between">
              <span class="text-neutral-700 dark:text-neutral-300">Router Terpisah & Modular</span>
              <UIcon :name="currentEval.detected_traits.has_modular_routing ? 'i-lucide-check-circle-2' : 'i-lucide-x-circle'" class="w-4 h-4" :class="currentEval.detected_traits.has_modular_routing ? 'text-sky-400' : 'text-neutral-400'" />
            </li>
            <li class="flex items-center justify-between">
              <span class="text-neutral-700 dark:text-neutral-300">Thin Controllers (Kecil & Ramping)</span>
              <UIcon :name="currentEval.detected_traits.has_thin_controllers ? 'i-lucide-check-circle-2' : 'i-lucide-x-circle'" class="w-4 h-4" :class="currentEval.detected_traits.has_thin_controllers ? 'text-sky-400' : 'text-neutral-400'" />
            </li>
            <li class="flex items-center justify-between">
              <span class="text-neutral-700 dark:text-neutral-300">Service Layer (Logika Bisnis Mandiri)</span>
              <UIcon :name="currentEval.detected_traits.has_service_layer ? 'i-lucide-check-circle-2' : 'i-lucide-x-circle'" class="w-4 h-4" :class="currentEval.detected_traits.has_service_layer ? 'text-sky-400' : 'text-neutral-400'" />
            </li>
            <li class="flex items-center justify-between">
              <span class="text-neutral-700 dark:text-neutral-300">Repository Layer (Abstraksi Database)</span>
              <UIcon :name="currentEval.detected_traits.has_repository_layer ? 'i-lucide-check-circle-2' : 'i-lucide-x-circle'" class="w-4 h-4" :class="currentEval.detected_traits.has_repository_layer ? 'text-sky-400' : 'text-neutral-400'" />
            </li>
            <li class="flex items-center justify-between">
              <span class="text-neutral-700 dark:text-neutral-300">Model Context Protocol (MCP) Siap</span>
              <UIcon :name="currentEval.detected_traits.has_mcp_support ? 'i-lucide-check-circle-2' : 'i-lucide-x-circle'" class="w-4 h-4" :class="currentEval.detected_traits.has_mcp_support ? 'text-sky-400' : 'text-neutral-400'" />
            </li>
            <li class="flex items-center justify-between">
              <span class="text-neutral-700 dark:text-neutral-300">Planner AI Layer (MVPP)</span>
              <UIcon :name="currentEval.detected_traits.has_planner_layer ? 'i-lucide-check-circle-2' : 'i-lucide-x-circle'" class="w-4 h-4" :class="currentEval.detected_traits.has_planner_layer ? 'text-sky-400' : 'text-neutral-400'" />
            </li>
          </ul>
        </div>

        <!-- Scale-Up Action Recommendations (7 cols) -->
        <div class="lg:col-span-7 p-5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-4">
          <div class="flex items-center justify-between">
            <h3 class="text-xs font-bold uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
              Rekomendasi Langkah Scale-Up
            </h3>
            <span class="text-xs font-semibold text-sky-600 dark:text-sky-400">
              Target Selanjutnya: {{ currentEval.recommended_next_pattern }}
            </span>
          </div>

          <div class="space-y-3">
            <div
              v-for="rec in currentEval.scale_up_recommendations"
              :key="rec.title"
              class="p-3.5 rounded-lg border border-neutral-100 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-800/40 space-y-1.5"
            >
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-2">
                  <span class="text-xs font-bold text-neutral-900 dark:text-white">
                    {{ rec.phase }}: {{ rec.title }}
                  </span>
                </div>
                <span class="text-[10px] font-semibold uppercase px-2 py-0.5 rounded-full" :class="getPriorityBadge(rec.priority)">
                  {{ rec.priority }}
                </span>
              </div>
              <p class="text-xs text-neutral-600 dark:text-neutral-300">
                {{ rec.description }}
              </p>
              <div class="flex items-center gap-1.5 text-[11px] font-mono text-neutral-500 pt-0.5">
                <UIcon name="i-lucide-folder" class="w-3.5 h-3.5 text-neutral-400" />
                <span>Target: {{ rec.target_dir }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
