<script setup lang="ts">
import type { CodeMapFileDetail } from '~/types'
import { kindMeta } from '~/composables/useCodeMap'

/** Detail satu berkas peta: simbol, impor, dipakai oleh (radius dampak perubahan), rute, ringkasan. */
const props = defineProps<{ projectId: number, path: string | null }>()
const emit = defineEmits<{ (e: 'update:path', v: string | null): void }>()

const { getFile } = useCodeMap()
const detail = ref<CodeMapFileDetail | null>(null)
const loading = ref(false)
const error = ref('')

const open = computed({
  get: () => props.path !== null,
  set: (v) => { if (!v) emit('update:path', null) }
})

watch(() => props.path, async (p) => {
  detail.value = null
  error.value = ''
  if (!p) return
  loading.value = true
  try {
    detail.value = await getFile(props.projectId, p)
  } catch (err: any) {
    error.value = err?.data?.error || err?.message || 'Gagal memuat berkas'
  } finally {
    loading.value = false
  }
}, { immediate: true })

const go = (p: string) => emit('update:path', p)
</script>

<template>
  <USlideover v-model:open="open" :title="path?.split('/').pop() || 'Berkas'" :description="path || ''" :ui="{ content: 'max-w-xl' }">
    <template #body>
      <div v-if="loading" class="py-10 flex justify-center">
        <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin text-neutral-400" />
      </div>
      <UAlert v-else-if="error" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="error" />
      <div v-else-if="detail" class="space-y-4 text-xs">
        <div class="flex flex-wrap gap-1.5">
          <UBadge color="neutral" variant="subtle" :icon="kindMeta(detail.kind).icon" :label="kindMeta(detail.kind).label" />
          <UBadge color="neutral" variant="outline" :label="detail.lang" />
          <UBadge color="neutral" variant="outline" :label="`${detail.lines} baris`" />
          <UBadge v-if="detail.component_name" color="info" variant="subtle" :label="detail.component_name" />
        </div>

        <div v-if="detail.ai_summary || detail.summary" class="space-y-1">
          <p v-if="detail.ai_summary" class="text-neutral-800 dark:text-neutral-100">
            <UIcon name="i-lucide-sparkles" class="size-3.5 text-violet-500 inline -mt-0.5" /> {{ detail.ai_summary }}
          </p>
          <p v-if="detail.summary" class="text-neutral-500">
            {{ detail.summary }}
          </p>
          <p v-if="detail.keywords" class="text-xs text-neutral-400">
            Kata kunci: {{ detail.keywords }}
          </p>
        </div>

        <section>
          <h4 class="font-semibold text-neutral-700 dark:text-neutral-200 mb-1">
            Dipakai oleh ({{ detail.used_by?.length || 0 }})
            <span class="font-normal text-neutral-400">— berkas yang ikut terdampak bila ini diubah</span>
          </h4>
          <p v-if="!detail.used_by?.length" class="text-neutral-400">
            Tidak ada pemakai lokal yang terdeteksi.
          </p>
          <ul class="space-y-0.5">
            <li v-for="u in detail.used_by" :key="u">
              <button type="button" class="font-mono text-xs text-sky-700 dark:text-sky-300 hover:underline break-all text-left" @click="go(u)">
                {{ u }}
              </button>
            </li>
          </ul>
        </section>

        <section>
          <h4 class="font-semibold text-neutral-700 dark:text-neutral-200 mb-1">
            Memakai ({{ detail.imports?.length || 0 }})
          </h4>
          <ul class="space-y-0.5">
            <li v-for="i in detail.imports" :key="i">
              <span v-if="i.endsWith('/')" class="font-mono text-xs text-neutral-500" title="Paket/direktori">{{ i }}</span>
              <button v-else type="button" class="font-mono text-xs text-sky-700 dark:text-sky-300 hover:underline break-all text-left" @click="go(i)">
                {{ i }}
              </button>
            </li>
          </ul>
        </section>

        <section v-if="detail.routes?.length">
          <h4 class="font-semibold text-neutral-700 dark:text-neutral-200 mb-1">
            Rute & panggilan API
          </h4>
          <div class="flex flex-wrap gap-1">
            <span v-for="r in detail.routes" :key="r" class="px-1.5 py-0.5 rounded bg-emerald-50 dark:bg-emerald-950/40 font-mono text-xs text-emerald-700 dark:text-emerald-300">{{ r }}</span>
          </div>
        </section>

        <section>
          <h4 class="font-semibold text-neutral-700 dark:text-neutral-200 mb-1">
            Simbol ({{ detail.symbols.length }})
          </h4>
          <ul class="divide-y divide-neutral-100 dark:divide-neutral-800">
            <li v-for="s in detail.symbols" :key="s.name + s.line" class="py-1 flex gap-2">
              <span class="font-mono text-xs text-neutral-400 w-10 shrink-0 text-right">{{ s.line }}</span>
              <span class="font-mono text-xs" :class="s.exported ? 'text-neutral-900 dark:text-white font-semibold' : 'text-neutral-600 dark:text-neutral-400'">{{ s.name }}</span>
              <span class="text-xs text-neutral-400">{{ s.kind }}</span>
              <span v-if="s.doc" class="text-xs text-neutral-500 line-clamp-1">{{ s.doc }}</span>
            </li>
          </ul>
        </section>

        <p class="font-mono text-xs text-neutral-400 break-all">
          {{ detail.abs_path }}
        </p>
      </div>
    </template>
  </USlideover>
</template>
