<script setup lang="ts">
import type { AIReviewResponse, ProjectComponent } from '~/types'

defineProps<{
  comp: ProjectComponent
  loading: boolean
  reviewing: boolean
  componentAiReview: AIReviewResponse | null
}>()
const customAiQuery = defineModel<string>('query', { default: '' })
const emit = defineEmits<{ run: [] }>()
const { copyToClipboard } = useCopyToClipboard()
const handleRunComponentAiReview = () => emit('run')
</script>

<template>
  <div class="space-y-6">
    <!-- Action Header & Custom Query -->
    <div class="p-5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs space-y-3.5">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h4 class="font-bold text-neutral-900 dark:text-white text-sm flex items-center gap-2">
            <UIcon name="i-lucide-bot" class="w-4 h-4 text-sky-400" />
            <span>AI Architecture & Code Advisor untuk {{ comp.name }}</span>
          </h4>
          <p class="text-xs text-neutral-500 mt-0.5">
            Menganalisis stack {{ comp.language }}/{{ comp.framework }} dan menghasilkan saran arsitektur serta instruksi prompt AI.
          </p>
        </div>

        <UButton
          icon="i-lucide-sparkles"
          color="primary"
          size="xs"
          :loading="reviewing"
          title="Jalankan review AI untuk aplikasi ini"
          @click="handleRunComponentAiReview"
        >
          {{ componentAiReview ? 'Review Ulang AI' : 'Jalankan Review AI' }}
        </UButton>
      </div>

      <!-- Custom query input -->
      <div class="flex items-center gap-2 pt-1 border-t border-neutral-100 dark:border-neutral-800">
        <UInput
          v-model="customAiQuery"
          placeholder="Ajukan pertanyaan khusus seputar aplikasi ini (opsional)..."
          size="xs"
          class="flex-1"
        />
      </div>
    </div>

    <!-- State Belum Ada Review -->
    <div v-if="!componentAiReview && !reviewing" class="p-8 text-center rounded-xl border border-dashed border-neutral-200 dark:border-neutral-800 bg-white/50 dark:bg-neutral-900/50 space-y-3">
      <div class="w-10 h-10 mx-auto rounded-full bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center text-neutral-400">
        <UIcon name="i-lucide-sparkles" class="w-5 h-5" />
      </div>
      <div class="space-y-1">
        <h4 class="text-sm font-semibold text-neutral-900 dark:text-white">
          Belum ada analisis AI untuk aplikasi ini
        </h4>
        <p class="text-xs text-neutral-500 max-w-md mx-auto">
          Klik Jalankan Review AI untuk mendapatkan analisis kekuatan, rencana perbaikan, dan instruksi prompt agent.
        </p>
      </div>
      <UButton size="xs" color="primary" @click="handleRunComponentAiReview">
        Mulai Review AI
      </UButton>
    </div>

    <!-- Hasil Review AI -->
    <div v-else-if="componentAiReview" class="space-y-4">
      <!-- Summary Box -->
      <div class="p-4 rounded-xl border border-sky-200 dark:border-sky-800/60 bg-sky-50/70 dark:bg-sky-950/40 text-xs space-y-1.5 shadow-2xs">
        <div class="font-bold text-sky-900 dark:text-sky-200 flex items-center gap-1.5">
          <UIcon name="i-lucide-check-circle" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
          <span>Ringkasan Evaluasi AI:</span>
        </div>
        <p class="text-neutral-700 dark:text-neutral-300 leading-relaxed">
          {{ componentAiReview.summary }}
        </p>
      </div>

      <!-- Action Plan Grid -->
      <div v-if="componentAiReview.action_plan?.length" class="space-y-2.5">
        <h5 class="text-xs font-semibold uppercase tracking-wider text-neutral-400">
          Rencana Aksi & Rekomendasi Fitur
        </h5>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
          <div
            v-for="(plan, idx) in componentAiReview.action_plan"
            :key="idx"
            class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-1.5 shadow-2xs"
          >
            <div class="flex items-center justify-between">
              <span class="font-bold text-neutral-900 dark:text-white">{{ plan.title }}</span>
              <UBadge size="xs" :color="plan.priority === 'P0' ? 'error' : 'warning'">
                {{ plan.priority }}
              </UBadge>
            </div>
            <p class="text-neutral-600 dark:text-neutral-400 text-xs leading-relaxed">
              {{ plan.description }}
            </p>
          </div>
        </div>
      </div>

      <!-- AI Agent Ready Prompt -->
      <div v-if="componentAiReview.ai_agent_prompt" class="p-4 rounded-xl bg-neutral-950 text-neutral-200 space-y-2 text-xs shadow-xs">
        <div class="flex items-center justify-between">
          <span class="font-mono text-sky-400 font-semibold text-xs">Prompt AI Agent (Antigravity / Claude Code)</span>
          <UButton
            icon="i-lucide-copy"
            color="neutral"
            variant="ghost"
            size="xs"
            @click="copyToClipboard(componentAiReview.ai_agent_prompt, 'Prompt AI berhasil disalin!')"
          >
            Salin Prompt
          </UButton>
        </div>
        <pre class="p-3 rounded-lg bg-black/40 font-mono text-xs text-neutral-300 overflow-x-auto whitespace-pre-wrap max-h-48 overflow-y-auto">{{ componentAiReview.ai_agent_prompt }}</pre>
      </div>
    </div>
  </div>
</template>
