<script setup lang="ts">
import type { CodeMapSearchHit } from '~/types'
import { codeMapKindMeta, kindMeta } from '~/composables/useCodeMap'

/** Pencarian peta kode ala RAG — hasil yang sama dengan MCP vibeasik_code_map_search yang dipakai agent. */
const props = defineProps<{ projectId: number }>()
const emit = defineEmits<{ (e: 'open', path: string): void }>()

const { search } = useCodeMap()
const query = ref('')
const kind = ref('all')
const withSnippet = ref(true)
const hits = ref<CodeMapSearchHit[]>([])
const loading = ref(false)
const error = ref('')
const searched = ref('')

const kindItems = [{ label: 'Semua peran', value: 'all' }, ...Object.entries(codeMapKindMeta).map(([value, m]) => ({ label: m.label, value }))]
const examples = ['hapus todo', 'exec guard', 'antrean job', 'halaman pengaturan', '/api/projects']

let timer: ReturnType<typeof setTimeout> | undefined
const run = async () => {
  const q = query.value.trim()
  if (!q) {
    hits.value = []
    searched.value = ''
    return
  }
  loading.value = true
  error.value = ''
  try {
    const res = await search(props.projectId, q, { kind: kind.value === 'all' ? undefined : kind.value, snippet: withSnippet.value, limit: 12 })
    hits.value = res.hits
    searched.value = q
  } catch (err: any) {
    error.value = err?.data?.error || err?.message || 'Pencarian gagal'
  } finally {
    loading.value = false
  }
}
watch(query, () => {
  clearTimeout(timer)
  timer = setTimeout(run, 300)
})
watch([kind, withSnippet], run)
watch(() => props.projectId, run)
onBeforeUnmount(() => clearTimeout(timer))

const useExample = (q: string) => {
  query.value = q
}
</script>

<template>
  <section class="p-4 sm:p-5 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 shadow-xs space-y-3.5">
    <div class="flex flex-col md:flex-row gap-2.5">
      <UInput
        v-model="query"
        icon="i-lucide-search"
        placeholder="Cari seperti agent: 'hapus todo', 'useBackgroundJobs', '/api/queue'…"
        class="flex-1"
        :loading="loading"
      />
      <USelect v-model="kind" :items="kindItems" class="md:w-44" />
      <USwitch v-model="withSnippet" label="Cuplikan kode" class="shrink-0 self-start sm:self-center" />
    </div>

    <div v-if="!searched && !loading" class="flex flex-wrap items-center gap-1.5 text-[11px] text-neutral-500">
      <span>Contoh:</span>
      <UButton v-for="ex in examples" :key="ex" size="xs" color="neutral" variant="soft" :label="ex" @click="useExample(ex)" />
    </div>

    <UAlert v-if="error" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="error" />

    <p v-if="searched && !hits.length && !loading" class="text-xs text-neutral-500 py-4 text-center">
      Tidak ada berkas yang cocok dengan "{{ searched }}".
    </p>

    <ul v-if="hits.length" class="space-y-2.5">
      <li
        v-for="h in hits"
        :key="h.path"
        class="p-3.5 sm:p-4 rounded-xl border border-neutral-200/80 dark:border-neutral-800/80 bg-neutral-50/40 dark:bg-neutral-900/40 hover:border-sky-300 dark:hover:border-sky-800 transition-colors"
      >
        <div class="flex flex-wrap items-center gap-2">
          <button type="button" class="font-mono text-xs font-semibold text-sky-700 dark:text-sky-300 hover:underline text-left break-all cursor-pointer" @click="emit('open', h.path)">
            {{ h.path }}
          </button>
          <UBadge size="sm" color="neutral" variant="subtle" :icon="kindMeta(h.kind).icon" :label="kindMeta(h.kind).label" />
          <span class="text-[10px] text-neutral-400 font-mono">{{ h.lines }} baris · skor {{ h.score }}</span>
          <span v-if="h.used_by_count" class="text-[10px] text-violet-600 dark:text-violet-400 font-medium" :title="(h.used_by || []).join('\n')">
            dipakai {{ h.used_by_count }} berkas
          </span>
        </div>
        <p v-if="h.summary" class="mt-1.5 text-xs text-neutral-600 dark:text-neutral-300 leading-relaxed">
          {{ h.summary }}
        </p>
        <div v-if="h.symbols?.length || h.routes?.length" class="mt-2 flex flex-wrap gap-1.5">
          <span v-for="s in h.symbols" :key="s.name + s.line" class="px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 font-mono text-[10px] text-neutral-700 dark:text-neutral-300 border border-neutral-200/50 dark:border-neutral-700/50" :title="s.doc || s.kind">
            {{ s.name }}<span class="text-neutral-400">:{{ s.line }}</span>
          </span>
          <span v-for="r in (h.routes || []).slice(0, 4)" :key="r" class="px-1.5 py-0.5 rounded bg-emerald-50 dark:bg-emerald-950/40 font-mono text-[10px] text-emerald-700 dark:text-emerald-300 border border-emerald-200/40 dark:border-emerald-800/40">
            {{ r }}
          </span>
        </div>
        <pre v-if="h.snippet" class="mt-2.5 max-h-56 overflow-auto rounded-lg bg-neutral-950 text-neutral-200 text-[10.5px] leading-relaxed p-3 font-mono border border-neutral-800">{{ h.snippet }}</pre>
      </li>
    </ul>
  </section>
</template>
