<script setup lang="ts">
import type { CodeMapOverview } from '~/types'
import { kindMeta } from '~/composables/useCodeMap'

/** Struktur peta: aplikasi → direktori → berkas (dengan ringkasan & simbol), sama seperti .vibeasik/CODEMAP.md. */
const props = defineProps<{ overview: CodeMapOverview }>()
const emit = defineEmits<{ (e: 'open', path: string): void }>()

const filter = ref('')
const openDirs = ref<Set<string>>(new Set())

const toggle = (key: string) => {
  const next = new Set(openDirs.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  openDirs.value = next
}

const components = computed(() => {
  const f = filter.value.trim().toLowerCase()
  return props.overview.components.map(c => ({
    ...c,
    dirs: c.dirs
      .map(d => ({ ...d, entries: (d.entries || []).filter(e => !f || e.path.toLowerCase().includes(f) || (e.summary || '').toLowerCase().includes(f)) }))
      .filter(d => !f || d.entries.length > 0)
  })).filter(c => c.dirs.length > 0)
})

const topKinds = (kinds: Record<string, number>) =>
  Object.entries(kinds).sort((a, b) => b[1] - a[1]).slice(0, 3).map(([k, n]) => `${n} ${kindMeta(k).label.toLowerCase()}`).join(' · ')
</script>

<template>
  <section class="p-4 sm:p-5 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 shadow-xs space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3 pb-2 border-b border-neutral-100 dark:border-neutral-800/80">
      <div>
        <h2 class="text-sm sm:text-base font-semibold text-neutral-800 dark:text-neutral-100">
          Struktur aplikasi
        </h2>
        <p class="text-[11px] text-neutral-400 mt-0.5">
          Eksplorasi direktori dan berkas kode proyek
        </p>
      </div>
      <UInput v-model="filter" size="sm" icon="i-lucide-filter" placeholder="Saring path / ringkasan" class="w-full sm:w-64" />
    </div>

    <div v-for="c in components" :key="c.id + c.name" class="space-y-2">
      <div class="flex flex-wrap items-baseline gap-2 pt-2">
        <UIcon name="i-lucide-layers" class="size-4 text-sky-600 self-center shrink-0" />
        <span class="text-sm font-semibold text-neutral-900 dark:text-white">{{ c.name }}</span>
        <span class="font-mono text-[11px] text-neutral-500">{{ c.path || '/' }}</span>
        <span class="text-[11px] text-neutral-400">{{ c.files }} berkas · {{ c.lines.toLocaleString('id-ID') }} baris · {{ topKinds(c.kinds) }}</span>
      </div>

      <div v-for="d in c.dirs" :key="c.id + d.path" class="ml-2 sm:ml-5 border-l-2 border-neutral-200/70 dark:border-neutral-800 pl-3">
        <button type="button" class="w-full flex items-center gap-2 py-1.5 text-left group hover:bg-neutral-50 dark:hover:bg-neutral-800/50 rounded-lg px-2 -mx-2 transition-colors cursor-pointer" @click="toggle(c.id + d.path)">
          <UIcon :name="openDirs.has(c.id + d.path) || filter ? 'i-lucide-folder-open' : 'i-lucide-folder'" class="size-4 text-amber-500 shrink-0" />
          <span class="font-mono text-xs font-medium text-neutral-800 dark:text-neutral-200 group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors">{{ d.path }}/</span>
          <span class="text-[10px] text-neutral-400 font-mono">{{ d.files }} berkas</span>
          <span v-if="d.key_files.length" class="text-[10px] text-neutral-400 truncate">· kunci: {{ d.key_files.join(', ') }}</span>
          <UIcon :name="openDirs.has(c.id + d.path) || filter ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="size-3.5 text-neutral-400 ml-auto" />
        </button>
        <ul v-if="openDirs.has(c.id + d.path) || filter" class="ml-4 sm:ml-5 my-1 space-y-1">
          <li v-for="e in d.entries" :key="e.path" class="flex items-start gap-2 text-xs py-1 px-2 -mx-2 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800/40 transition-colors">
            <UIcon :name="kindMeta(e.kind).icon" class="size-3.5 mt-0.5 text-neutral-400 shrink-0" :title="kindMeta(e.kind).label" />
            <button type="button" class="font-mono text-[11px] font-medium text-sky-700 dark:text-sky-300 hover:underline shrink-0 cursor-pointer" @click="emit('open', e.path)">
              {{ e.path.split('/').pop() }}
            </button>
            <span class="text-[10px] text-neutral-400 shrink-0 font-mono">{{ e.lines }}b<template v-if="e.used_by"> · ↩{{ e.used_by }}</template></span>
            <span class="text-[11px] text-neutral-500 dark:text-neutral-400 line-clamp-1">{{ e.summary || (e.symbols || []).join(', ') }}</span>
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>
