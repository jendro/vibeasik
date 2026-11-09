<script setup lang="ts">
import { agentPromptTargets, buildAgentPrompt } from '~/utils/agentPrompts'

/** Prompt siap copas untuk menyuruh agent coding (Claude Code, Antigravity, Cline, …) mengerjakan todo lewat MCP vibeasik. */
const props = defineProps<{
  open: boolean
  projectId?: number
  projectName?: string
}>()
const emit = defineEmits<{
  (e: 'update:open', v: boolean): void
}>()

const toast = useToast()
const selectedKey = ref(agentPromptTargets[0]!.key)
const copied = ref(false)

const target = computed(() => agentPromptTargets.find(t => t.key === selectedKey.value) ?? agentPromptTargets[0]!)
const prompt = computed(() => props.projectId
  ? buildAgentPrompt(target.value, { id: props.projectId, name: props.projectName })
  : '')

watch(selectedKey, () => {
  copied.value = false
})

const copy = async () => {
  try {
    await navigator.clipboard.writeText(prompt.value)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 1800)
  } catch {
    toast.add({ title: 'Gagal menyalin', description: 'Blok teks prompt lalu salin manual (Ctrl+C).', icon: 'i-lucide-alert-circle', color: 'error' })
  }
}
</script>

<template>
  <UModal :open="open" :ui="{ content: 'max-w-2xl' }" @update:open="emit('update:open', $event)">
    <template #content>
      <div class="p-6 space-y-4 max-h-[85vh] overflow-y-auto">
        <div>
          <h3 class="text-base font-semibold text-neutral-900 dark:text-white flex items-center gap-2">
            <UIcon name="i-lucide-clipboard-copy" class="size-5 text-violet-500" />
            Prompt untuk Agent Coding
          </h3>
          <p class="mt-1 text-sm text-neutral-500">
            Salin lalu tempel di chat agent coding yang sudah tersambung ke MCP <code>vibeasik</code>. Prompt menyebut project ID, tool pertama, dan larangan scan agar agent langsung mengerjakan todo.
          </p>
        </div>

        <UAlert
          v-if="!projectId"
          color="warning"
          variant="subtle"
          icon="i-lucide-info"
          description="Pilih proyek aktif dulu agar project ID terisi di prompt."
        />

        <template v-else>
          <div class="flex flex-wrap gap-2">
            <UButton
              v-for="t in agentPromptTargets"
              :key="t.key"
              :icon="t.icon"
              :label="t.label"
              size="sm"
              :color="t.key === selectedKey ? 'primary' : 'neutral'"
              :variant="t.key === selectedKey ? 'solid' : 'outline'"
              @click="selectedKey = t.key"
            />
          </div>

          <p class="text-sm text-neutral-600 dark:text-neutral-300">
            {{ target.hint }}
          </p>

          <pre class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-900 text-xs leading-relaxed font-mono whitespace-pre-wrap break-words text-neutral-800 dark:text-neutral-200 select-all">{{ prompt }}</pre>

          <p v-if="!target.subagents" class="text-xs text-neutral-500">
            Mode worker butuh engine <code>vibeasik serve</code> menyala dan minimal satu profil agent aktif di Orkestrator; tanpa itu agent akan berhenti dan melapor.
          </p>
        </template>

        <div class="flex justify-end gap-2 pt-3 border-t border-neutral-100 dark:border-neutral-800">
          <UButton color="neutral" variant="ghost" label="Tutup" @click="emit('update:open', false)" />
          <UButton
            color="primary"
            :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'"
            :label="copied ? 'Tersalin' : 'Salin prompt'"
            :disabled="!prompt"
            @click="copy"
          />
        </div>
      </div>
    </template>
  </UModal>
</template>
