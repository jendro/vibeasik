<script setup lang="ts">
import type { ResolutionResponse } from '~/types'

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  resolution: ResolutionResponse | null
  loading: boolean
}>()

const router = useRouter()
const toast = useToast()
const copied = ref(false)

const copyCode = async () => {
  if (!props.resolution?.code_after) return
  try {
    await navigator.clipboard.writeText(props.resolution.code_after)
    copied.value = true
    toast.add({
      title: 'Kode Disalin',
      description: 'Kode perbaikan berhasil disalin ke clipboard.',
      icon: 'i-lucide-check',
      color: 'success'
    })
    setTimeout(() => { copied.value = false }, 2000)
  } catch (err) {
    console.error('Copy failed:', err)
  }
}

const goToTodos = () => {
  open.value = false
  router.push('/todos')
}
</script>

<template>
  <UModal
    v-model:open="open"
    title="Rekomendasi Solusi & Rencana Aksi AI"
    description="Analisis perbaikan kode berbasis teori & pembuatan Todo otomatis."
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #body>
      <!-- Loading State -->
      <div v-if="loading" class="py-12 flex flex-col items-center justify-center space-y-3">
        <UIcon name="i-lucide-sparkles" class="w-8 h-8 text-primary-500 animate-spin" />
        <p class="text-sm font-semibold text-neutral-800 dark:text-neutral-200">
          Menganalisis Kode & Menghubungi Mesin LLM External...
        </p>
        <p class="text-xs text-neutral-500 text-center max-w-sm">
          Mengekstrak konteks berkas, memvalidasi landasan teori, dan merumuskan Todo perbaikan...
        </p>
      </div>

      <!-- Result State -->
      <div v-else-if="resolution" class="space-y-4 text-xs">
        <!-- Header Info -->
        <div class="p-3.5 rounded-xl bg-violet-50/70 dark:bg-violet-950/30 border border-violet-200 dark:border-violet-800/60 space-y-2">
          <div class="flex items-center justify-between">
            <span class="px-2 py-0.5 rounded font-mono font-bold text-[11px] bg-violet-100 dark:bg-violet-900/60 text-violet-800 dark:text-violet-300">
              {{ resolution.rule_id }}
            </span>
            <div class="flex items-center gap-1.5 text-neutral-500 text-[11px]">
              <UIcon name="i-lucide-cpu" class="w-3.5 h-3.5" />
              <span>Model: <strong class="text-neutral-700 dark:text-neutral-300">{{ resolution.model_used }}</strong></span>
            </div>
          </div>

          <h3 class="text-sm font-bold text-neutral-900 dark:text-white">
            {{ resolution.title }}
          </h3>

          <p class="text-neutral-600 dark:text-neutral-400 leading-relaxed font-sans">
            {{ resolution.explanation }}
          </p>

          <!-- Theory Basis -->
          <div class="flex items-start gap-1.5 pt-1 text-[11px] text-violet-700 dark:text-violet-300 font-medium">
            <UIcon name="i-lucide-book-open" class="w-4 h-4 shrink-0 mt-0.5" />
            <span>Landasan Teori: {{ resolution.theory_basis }}</span>
          </div>
        </div>

        <!-- Target File & Line Alert -->
        <div class="p-2.5 rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-800/40 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-file-code" class="w-4 h-4 text-primary-500" />
            <span class="font-mono text-neutral-800 dark:text-neutral-200 font-semibold">{{ resolution.target_file }}</span>
          </div>
          <span class="px-2 py-0.5 rounded bg-primary-100 dark:bg-primary-950/50 text-primary-700 dark:text-primary-300 font-mono font-bold">
            Baris {{ resolution.target_lines }}
          </span>
        </div>

        <!-- Todo Created Confirmation -->
        <div v-if="resolution.created_todo" class="p-3 rounded-xl bg-sky-50 dark:bg-sky-950/30 border border-sky-200 dark:border-sky-800/60 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-check-circle-2" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
            <div>
              <span class="font-bold text-sky-900 dark:text-sky-300">Todo Otomatis Dibuat di Modul Perencanaan</span>
              <p class="text-[11px] text-sky-700 dark:text-sky-400 mt-0.5 line-clamp-1">
                "{{ resolution.created_todo.title }}"
              </p>
            </div>
          </div>
          <UButton size="xs" color="success" variant="outline" @click="goToTodos">
            Buka Todo
          </UButton>
        </div>

        <DiagnosticsPlacementPlanTable v-if="resolution.placement_plan?.length" :plan="resolution.placement_plan" />

        <!-- Code Replacement (Before vs After) -->
        <div v-if="resolution.code_after" class="space-y-2">
          <div class="flex items-center justify-between">
            <span class="font-semibold uppercase tracking-wider text-neutral-500 text-[11px]">
              Kode Rekomendasi Siap Pakai (Drop-in Replacement):
            </span>
            <UButton
              size="xs"
              color="neutral"
              variant="outline"
              :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'"
              @click="copyCode"
            >
              {{ copied ? 'Tersalin' : 'Salin Kode' }}
            </UButton>
          </div>

          <pre class="p-3 rounded-xl bg-[#010810] text-sky-300 font-mono text-xs overflow-x-auto border border-white/10 leading-relaxed max-h-56"><code>{{ resolution.code_after }}</code></pre>
        </div>

        <!-- Step-by-Step Action Plan -->
        <div v-if="resolution.action_steps && resolution.action_steps.length > 0" class="space-y-1.5">
          <span class="font-semibold uppercase tracking-wider text-neutral-500 text-[11px]">
            Langkah Implementasi:
          </span>
          <div class="space-y-1">
            <div
              v-for="(step, idx) in resolution.action_steps"
              :key="idx"
              class="flex items-start gap-2 p-2 rounded-lg bg-neutral-50 dark:bg-neutral-800/30 border border-neutral-100 dark:border-neutral-800"
            >
              <span class="w-4 h-4 rounded-full bg-neutral-200 dark:bg-neutral-700 text-neutral-700 dark:text-neutral-300 flex items-center justify-center font-bold text-[10px] shrink-0 mt-0.5">
                {{ idx + 1 }}
              </span>
              <span class="text-neutral-700 dark:text-neutral-300 leading-normal">{{ step }}</span>
            </div>
          </div>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-between w-full">
        <UButton color="neutral" variant="outline" size="sm" @click="open = false">
          Tutup
        </UButton>
        <div class="flex items-center gap-2">
          <UButton
            v-if="resolution?.created_todo"
            color="primary"
            variant="solid"
            size="sm"
            icon="i-lucide-clipboard-list"
            @click="goToTodos"
          >
            Lihat di Daftar Todo
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
