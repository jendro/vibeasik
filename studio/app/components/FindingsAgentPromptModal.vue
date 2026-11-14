<script setup lang="ts">
import type { FindingScope, FindingsTodoPrompt } from '~/types'

/**
 * Prompt siap salin: agent coding pengguna menjadikan temuan todo lewat MCP vibeasik (triase sendiri, tanpa LLM luar).
 * Teksnya dari engine — sama dengan yang dijalankan tool MCP vibeasik_findings_todo_prompt saat pengguna cukup
 * berkata "jadikan todo temuan audit <proyek>" ke agentnya.
 */
const props = defineProps<{ projectId: number, scope: FindingScope['key'] }>()
const open = defineModel<boolean>('open', { default: false })

const { getFindingsTodoPrompt } = useApi()
const toast = useToast()
const data = ref<FindingsTodoPrompt | null>(null)
const loading = ref(false)
const failed = ref('')
const copied = ref(false)

const load = async () => {
  loading.value = true
  failed.value = ''
  try {
    data.value = await getFindingsTodoPrompt(props.projectId, props.scope)
  } catch (err: any) {
    failed.value = err?.data?.error || err?.message || 'Gagal menyusun prompt'
  } finally {
    loading.value = false
  }
}
// Dimuat saat dibuka agar jumlah temuan di prompt selalu terbaru.
watch(open, (v) => { if (v) { copied.value = false; load() } })

const shortAsk = computed(() => data.value
  ? `jadikan todo temuan ${data.value.scope.label} ${data.value.project_name}`
  : '')

const copy = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1800)
  } catch {
    toast.add({ title: 'Gagal menyalin', description: 'Blok teks prompt lalu salin manual (Ctrl+C).', icon: 'i-lucide-alert-circle', color: 'error' })
  }
}
</script>

<template>
  <UModal v-model:open="open" :ui="{ content: 'max-w-2xl' }">
    <template #content>
      <div class="p-6 space-y-4 max-h-[85vh] overflow-y-auto">
        <div>
          <h3 class="text-base font-semibold text-neutral-900 dark:text-white flex items-center gap-2">
            <UIcon name="i-lucide-clipboard-copy" class="size-5 text-neutral-500" />
            Prompt untuk agent coding
          </h3>
          <p class="mt-1 text-sm text-neutral-500">
            Tempel di chat agent coding Anda (Claude Code, Cursor, Cline, Antigravity …) yang sudah tersambung ke MCP <code>vibeasik</code>.
            Agent memeriksa setiap temuan pada kodenya lalu membuat todo — memakai kuota agent Anda, bukan LLM lewat API.
          </p>
        </div>

        <div v-if="loading && !data" class="h-48 rounded-xl bg-neutral-100 dark:bg-white/[0.04] animate-pulse" />
        <UAlert v-else-if="failed" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="failed" />

        <template v-else-if="data">
          <p class="text-sm text-neutral-700 dark:text-neutral-200">
            <span class="font-semibold tabular-nums">{{ data.open }}</span> temuan {{ data.scope.label }} terbuka di {{ data.project_name }}<span v-if="data.critical" class="text-rose-600 dark:text-rose-400"> · {{ data.critical }} kritis</span><span v-if="data.advisory" class="text-neutral-500"> · {{ data.advisory }} di antaranya perkiraan, tidak ikut triase</span>.
          </p>
          <pre class="p-4 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-neutral-50 dark:bg-[#04111f] text-xs leading-relaxed font-mono whitespace-pre-wrap break-words text-neutral-800 dark:text-neutral-200 select-all">{{ data.prompt }}</pre>
          <div class="flex items-start gap-2 rounded-lg bg-neutral-50 dark:bg-white/[0.03] p-3 text-xs text-neutral-600 dark:text-neutral-300">
            <UIcon name="i-lucide-message-square-text" class="size-4 shrink-0 mt-px" />
            <p>
              Lebih singkat: cukup minta agent Anda
              <button type="button" class="font-mono text-neutral-900 dark:text-white underline decoration-dotted underline-offset-2" title="Salin kalimat ini" @click="copy(shortAsk)">
                "{{ shortAsk }}"
              </button>.
              Tool MCP <code>vibeasik_findings_todo_prompt</code> memberi agent langkah yang sama dengan prompt di atas.
            </p>
          </div>
        </template>

        <div class="flex justify-end gap-2 pt-3 border-t border-neutral-100 dark:border-white/[0.08]">
          <UButton color="neutral" variant="ghost" label="Tutup" @click="open = false" />
          <UButton
            color="neutral"
            :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'"
            :label="copied ? 'Tersalin' : 'Salin prompt'"
            :disabled="!data?.prompt"
            @click="copy(data!.prompt)"
          />
        </div>
      </div>
    </template>
  </UModal>
</template>
