<script setup lang="ts">
// Rekap temuan satu halaman pemeriksaan + "Jadikan semua todo". Sama di semua halaman Periksa & tingkatkan:
// deret angka rekap, satu aksi utama (solid), dan — bila temuannya bisa ditriase agent lewat MCP — tombol
// "Salin prompt agent" agar pengguna bisa memakai agent coding sendiri tanpa kuota LLM luar.
import type { StatItem } from '~/components/AppStatStrip.vue'
import type { FindingScope } from '~/types'

const props = withDefaults(defineProps<{
  projectId: number | null
  stats: StatItem[]
  /** jumlah temuan yang akan dijadikan todo oleh tombol utama */
  count: number
  title?: string
  /** kalimat di bawah judul: status penanganan / langkah berikutnya */
  note?: string
  /** penjelasan tombol utama (tooltip) */
  todoDesc?: string
  /** tombol utama memanggil LLM lewat API */
  llm?: boolean
  busy?: boolean
  /** sumber temuan untuk prompt agent; kosong = tanpa tombol prompt, untuk aksi tanpa LLM */
  promptScope?: FindingScope['key']
}>(), { title: 'Rekap temuan', note: '', todoDesc: '', llm: false, busy: false, promptScope: undefined })

const emit = defineEmits<{ todoAll: [] }>()
const promptOpen = ref(false)
const label = computed(() => props.count ? `Jadikan semua todo (${props.count})` : 'Semua sudah jadi todo')
</script>

<template>
  <section class="rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] p-4 space-y-3" aria-label="Rekap temuan">
    <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-3">
      <div class="min-w-0 space-y-0.5">
        <h2 class="text-sm font-semibold text-neutral-900 dark:text-white">
          {{ title }}
        </h2>
        <p v-if="note" class="text-xs text-neutral-600 dark:text-neutral-300">
          {{ note }}
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2 shrink-0">
        <UButton
          v-if="promptScope"
          size="sm"
          color="neutral"
          variant="outline"
          icon="i-lucide-clipboard-copy"
          label="Salin prompt agent"
          title="Prompt untuk agent coding Anda (Claude Code, Cursor, Cline …) yang tersambung ke MCP vibeasik: agent memeriksa kode lalu membuat todo — tanpa kuota LLM luar"
          :disabled="!projectId"
          @click="promptOpen = true"
        />
        <AppActionButton
          kind="process"
          primary
          :llm="llm"
          :label="label"
          :desc="todoDesc"
          :loading="busy"
          :disabled="!count || busy || !projectId"
          @click="emit('todoAll')"
        />
      </div>
    </div>
    <AppStatStrip v-if="stats.length" :items="stats" />
    <slot />
    <FindingsAgentPromptModal v-if="promptScope && projectId" v-model:open="promptOpen" :project-id="projectId" :scope="promptScope" />
  </section>
</template>
