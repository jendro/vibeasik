<script setup lang="ts">
import type { AgentRun, Todo } from '~/types'

defineProps<{
  autopilotEnabled: boolean
  autopilotAgent: string
  activeRunningTodo?: Todo
  agentRunsByTodo: Record<number, AgentRun>
  aiNotice: { type: 'success' | 'error' | 'info', message: string } | null
}>()
const emit = defineEmits<{
  toggleAutopilot: [value: boolean]
  dismissAiNotice: []
}>()
const handleToggleAutopilot = (value: boolean) => emit('toggleAutopilot', value)
const dismissAiNotice = () => emit('dismissAiNotice')
</script>

<template>
  <!-- Autopilot Live Progress Banner -->
  <div
    v-if="autopilotEnabled"
    class="p-3.5 rounded-xl border border-emerald-300 dark:border-emerald-800/80 bg-linear-to-r from-emerald-50/90 via-teal-50/70 to-emerald-50/90 dark:from-emerald-950/40 dark:via-teal-950/30 dark:to-emerald-950/40 text-xs transition-all"
  >
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <div class="relative size-8 rounded-lg bg-emerald-600 text-white flex items-center justify-center shrink-0">
          <UIcon name="i-lucide-bot" class="w-4 h-4" />
          <span class="absolute -top-1 -right-1 flex h-2.5 w-2.5">
            <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
            <span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-600 border border-white dark:border-neutral-900" />
          </span>
        </div>
        <div>
          <div class="flex items-center gap-2 flex-wrap">
            <span class="font-bold text-emerald-950 dark:text-emerald-100 text-sm">
              Pengerjaan Otomatis Sedang Berjalan (Autopilot)
            </span>
            <span class="px-2 py-0.5 rounded-full bg-emerald-100 dark:bg-emerald-900/60 text-emerald-800 dark:text-emerald-300 text-xs font-semibold border border-emerald-300 dark:border-emerald-800">
              1 Per 1 Berurutan
            </span>
            <span class="px-2 py-0.5 rounded-full bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 text-xs font-mono border border-neutral-300 dark:border-neutral-700">
              CLI: {{ autopilotAgent === 'opencode' ? 'OpenCode (Provider Kustom)' : autopilotAgent === 'agy' ? 'Antigravity (7 Akun)' : 'Router Otomatis' }}
            </span>
          </div>
          <p class="text-emerald-800 dark:text-emerald-300 text-xs mt-0.5">
            <span v-if="activeRunningTodo">
              Sedang mengeksekusi: <strong>#{{ activeRunningTodo.id }} - {{ activeRunningTodo.title }}</strong>
              <span v-if="agentRunsByTodo[activeRunningTodo.id]?.profile_name" class="font-mono ml-1 opacity-90">
                [{{ agentRunsByTodo[activeRunningTodo.id]?.profile_name }}]
              </span>
            </span>
            <span v-else>
              Menyiapkan agenda berikutnya untuk dikerjakan...
            </span>
          </p>
        </div>
      </div>

      <div class="flex items-center gap-2 shrink-0">
        <UButton
          size="xs"
          color="error"
          variant="solid"
          icon="i-lucide-square"
          label="Hentikan Autopilot"
          @click="handleToggleAutopilot(false)"
        />
      </div>
    </div>
  </div>

  <!-- AI Suggestion Result Alert Banner -->
  <div
    v-if="aiNotice"
    class="p-3.5 rounded-xl border text-xs flex items-center justify-between gap-3 transition-all"
    :class="{
      'border-emerald-300 dark:border-emerald-800/80 bg-emerald-50/90 dark:bg-emerald-950/30 text-emerald-900 dark:text-emerald-200': aiNotice.type === 'success',
      'border-blue-300 dark:border-blue-800/80 bg-blue-50/90 dark:bg-blue-950/30 text-blue-900 dark:text-blue-200': aiNotice.type === 'info',
      'border-rose-300 dark:border-rose-800/80 bg-rose-50/90 dark:bg-rose-950/30 text-rose-900 dark:text-rose-200': aiNotice.type === 'error'
    }"
  >
    <div class="flex items-center gap-2.5">
      <UIcon
        :name="aiNotice.type === 'success' ? 'i-lucide-check-circle' : aiNotice.type === 'info' ? 'i-lucide-info' : 'i-lucide-alert-circle'"
        class="w-4 h-4 shrink-0"
        :class="{
          'text-emerald-600 dark:text-emerald-400': aiNotice.type === 'success',
          'text-blue-600 dark:text-blue-400': aiNotice.type === 'info',
          'text-rose-600 dark:text-rose-400': aiNotice.type === 'error'
        }"
      />
      <span class="font-medium">{{ aiNotice.message }}</span>
    </div>
    <button
      type="button"
      class="opacity-70 hover:opacity-100 font-bold px-1.5 py-0.5 rounded cursor-pointer"
      @click="dismissAiNotice"
    >
      ✕
    </button>
  </div>
</template>
