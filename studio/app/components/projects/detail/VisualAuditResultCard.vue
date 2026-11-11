<script setup lang="ts">
import type { VisualAuditResponse } from '~/types'

const props = defineProps<{
  audit: VisualAuditResponse
  projectId: number
}>()

const toast = useToast()
const { createTodo } = useApi()
const creatingTodo = ref(false)

const copyAgentPrompt = async () => {
  try {
    await navigator.clipboard.writeText(props.audit.ai_agent_prompt)
    toast.add({
      title: 'Prompt AI Disalin',
      description: 'Prompt perbaikan siap di-paste ke Google Antigravity atau Claude Code.',
      icon: 'i-lucide-clipboard-check',
      color: 'success'
    })
  } catch {
    toast.add({
      title: 'Gagal Menyalin',
      description: 'Silakan salin teks prompt secara manual.',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  }
}

const handleConvertToTodo = async () => {
  creatingTodo.value = true
  try {
    await createTodo({
      project_id: props.projectId,
      title: `[Visual Audit] Perbaikan UI & Data Rute ${props.audit.route}`,
      description: `${props.audit.summary}\n\nRekomendasi Perbaikan:\n- ` + props.audit.suggested_fixes.join('\n- '),
      priority: 'high',
      status: 'pending',
      category: 'frontend'
    })

    toast.add({
      title: 'Todo Berhasil Dibuat',
      description: `Task perbaikan untuk rute ${props.audit.route} berhasil ditambahkan ke roadmap proyek.`,
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Gagal menambahkan todo'
    toast.add({
      title: 'Gagal Membuat Todo',
      description: msg,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    creatingTodo.value = false
  }
}
</script>

<template>
  <div class="space-y-5 p-5 sm:p-6 rounded-xl border border-slate-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] shadow-sm">
    <!-- Header with 2 Metric Scores -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 pb-4 border-b border-slate-100 dark:border-white/[0.08]">
      <div>
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-sparkles" class="size-5 text-amber-500" />
          <h3 class="text-base font-bold text-slate-900 dark:text-slate-100">
            Hasil Audit Multimodal AI
          </h3>
          <span class="text-xs px-2 py-0.5 rounded-full font-mono bg-blue-50 dark:bg-blue-950 text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-800">
            {{ audit.route }}
          </span>
        </div>
        <p class="text-xs text-slate-500 dark:text-slate-400 mt-1">
          {{ audit.summary }}
        </p>
      </div>

      <!-- Score Badges -->
      <div class="flex items-center gap-3 shrink-0">
        <!-- Visual Score -->
        <div class="flex flex-col items-center px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700/80 min-w-20">
          <span class="text-xs font-semibold text-slate-500">Kualitas UI</span>
          <span class="text-lg font-black tracking-tight" :class="audit.visual_score >= 80 ? 'text-emerald-500' : 'text-amber-500'">
            {{ audit.visual_score }}<span class="text-xs font-normal text-slate-400">/100</span>
          </span>
        </div>

        <!-- Data Hygiene Score -->
        <div class="flex flex-col items-center px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700/80 min-w-20">
          <span class="text-xs font-semibold text-slate-500">Integritas Data</span>
          <span class="text-lg font-black tracking-tight" :class="audit.data_hygiene_score >= 80 ? 'text-blue-500' : 'text-rose-500'">
            {{ audit.data_hygiene_score }}<span class="text-xs font-normal text-slate-400">/100</span>
          </span>
        </div>
      </div>
    </div>

    <!-- 2 Column Breakdown: Data Hygiene & Visual UI/UX -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <!-- Data Issues Card -->
      <div class="p-4 rounded-xl border border-slate-200/80 dark:border-white/[0.08] bg-slate-50/50 dark:bg-slate-950/40 space-y-3">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-database-zap" class="size-4 text-blue-500" />
          <h4 class="text-xs font-semibold text-slate-700 dark:text-slate-300">
            Pemeriksaan Integritas & Anomali Data
          </h4>
        </div>

        <div v-if="audit.data_issues?.length" class="space-y-2">
          <div
            v-for="(item, idx) in audit.data_issues"
            :key="idx"
            class="p-2.5 rounded-lg border text-xs"
            :class="item.severity === 'error'
              ? 'bg-rose-50 dark:bg-rose-950/40 border-rose-200 dark:border-rose-900 text-rose-800 dark:text-rose-200'
              : 'bg-white dark:bg-[#04111f] border-slate-200 dark:border-white/[0.08] text-slate-700 dark:text-slate-300'"
          >
            <div class="flex items-center justify-between gap-1 mb-1">
              <span class="font-semibold">{{ item.title }}</span>
              <span v-if="item.target_area" class="text-xs px-1.5 py-0.5 rounded font-mono bg-slate-100 dark:bg-slate-800">
                {{ item.target_area }}
              </span>
            </div>
            <p class="text-xs opacity-90 leading-relaxed">
              {{ item.description }}
            </p>
          </div>
        </div>
        <div v-else class="text-xs text-slate-400 py-2">
          Tidak ada anomali atau kebocoran placeholder data terdeteksi.
        </div>
      </div>

      <!-- Visual Issues Card -->
      <div class="p-4 rounded-xl border border-slate-200/80 dark:border-white/[0.08] bg-slate-50/50 dark:bg-slate-950/40 space-y-3">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-layout" class="size-4 text-amber-500" />
          <h4 class="text-xs font-semibold text-slate-700 dark:text-slate-300">
            Kualitas Visual & Kebersihan Desain
          </h4>
        </div>

        <div v-if="audit.visual_issues?.length" class="space-y-2">
          <div
            v-for="(item, idx) in audit.visual_issues"
            :key="idx"
            class="p-2.5 rounded-lg border text-xs bg-white dark:bg-[#04111f] border-slate-200 dark:border-white/[0.08] text-slate-700 dark:text-slate-300"
          >
            <div class="flex items-center justify-between gap-1 mb-1">
              <span class="font-semibold">{{ item.title }}</span>
              <span v-if="item.target_area" class="text-xs px-1.5 py-0.5 rounded font-mono bg-slate-100 dark:bg-slate-800">
                {{ item.target_area }}
              </span>
            </div>
            <p class="text-xs opacity-90 leading-relaxed">
              {{ item.description }}
            </p>
          </div>
        </div>
        <div v-else class="text-xs text-slate-400 py-2">
          Tata letak rapi, proporsi seimbang dan kontras memenuhi standar.
        </div>
      </div>
    </div>

    <!-- Concrete Code Suggestions -->
    <div v-if="audit.suggested_fixes?.length" class="p-4 rounded-xl border border-emerald-200/80 dark:border-emerald-900/60 bg-emerald-50/30 dark:bg-emerald-950/20 space-y-2.5">
      <div class="flex items-center gap-2">
        <UIcon name="i-lucide-check-circle" class="size-4 text-emerald-600 dark:text-emerald-400" />
        <h4 class="text-xs font-semibold text-emerald-800 dark:text-emerald-300">
          Rekomendasi Perbaikan Kode Frontend
        </h4>
      </div>
      <ul class="space-y-1.5 text-xs text-emerald-900 dark:text-emerald-200 pl-4 list-disc">
        <li v-for="(fix, idx) in audit.suggested_fixes" :key="idx" class="leading-relaxed">
          {{ fix }}
        </li>
      </ul>
    </div>

    <!-- Action Bar: Copy Prompt & Convert to Todo -->
    <div class="flex flex-wrap items-center justify-between gap-3 pt-3 border-t border-slate-100 dark:border-white/[0.08]">
      <span class="text-xs text-slate-400">
        Diaudit pada: {{ new Date(audit.audited_at).toLocaleTimeString() }}
      </span>

      <div class="flex items-center gap-2">
        <UButton
          size="xs"
          color="neutral"
          variant="outline"
          icon="i-lucide-clipboard"
          @click="copyAgentPrompt"
        >
          Salin Prompt untuk AI Agent
        </UButton>

        <UButton
          size="xs"
          color="primary"
          variant="solid"
          icon="i-lucide-plus"
          :loading="creatingTodo"
          @click="handleConvertToTodo"
        >
          Jadikan Todo Proyek
        </UButton>
      </div>
    </div>
  </div>
</template>
