<script setup lang="ts">
import type { CodeMapOverview, CodeMapRankedFile } from '~/types'
import { kindMeta } from '~/composables/useCodeMap'

/** Ringkasan yang dibaca agent pertama kali: dokumen konteks, titik masuk, modul inti, dan hotspot berkas besar. */
const props = defineProps<{ overview: CodeMapOverview }>()
const emit = defineEmits<{ (e: 'open', path: string): void }>()

const cards = computed(() => [
  { key: 'docs', title: 'Dokumen konteks', hint: 'Aturan proyek yang wajib dibaca agent', icon: 'i-lucide-book-open', tone: 'text-sky-600', items: props.overview.docs, note: (r: CodeMapRankedFile) => r.note || '' },
  { key: 'entry', title: 'Titik masuk & rute', hint: 'Tempat aplikasi dimulai dan rute didaftarkan', icon: 'i-lucide-log-in', tone: 'text-emerald-600', items: props.overview.entry_points, note: (r: CodeMapRankedFile) => r.note || '' },
  { key: 'core', title: 'Modul inti', hint: 'Paling banyak dipakai — perubahan berdampak luas', icon: 'i-lucide-orbit', tone: 'text-violet-600', items: props.overview.core, note: (r: CodeMapRankedFile) => `dipakai ${r.used_by} berkas` },
  { key: 'hot', title: 'Hotspot berkas besar', hint: 'Mahal dibaca utuh oleh AI — kandidat dipecah', icon: 'i-lucide-flame', tone: 'text-rose-600', items: props.overview.hotspots, note: (r: CodeMapRankedFile) => `${r.lines} baris` }
])
</script>

<template>
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-4">
    <section v-for="c in cards" :key="c.key" class="p-4 sm:p-5 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 shadow-xs">
      <div class="flex items-center gap-2 mb-3">
        <UIcon :name="c.icon" class="size-4 shrink-0" :class="c.tone" />
        <h3 class="text-sm font-semibold text-neutral-800 dark:text-neutral-100">
          {{ c.title }}
        </h3>
        <span class="text-[11px] text-neutral-400 truncate">{{ c.hint }}</span>
      </div>
      <p v-if="!c.items.length" class="text-xs text-neutral-400 py-3 text-center">
        Tidak ada.
      </p>
      <ul v-else class="divide-y divide-neutral-100 dark:divide-neutral-800/80">
        <li v-for="r in c.items.slice(0, 8)" :key="r.path" class="py-2 flex items-start gap-2.5 text-xs hover:bg-neutral-50/60 dark:hover:bg-neutral-800/30 px-2 -mx-2 rounded-lg transition-colors">
          <UIcon :name="kindMeta(r.kind).icon" class="size-3.5 mt-0.5 text-neutral-400 shrink-0" />
          <button type="button" class="font-mono text-[11px] text-sky-700 dark:text-sky-300 hover:underline text-left break-all cursor-pointer" @click="emit('open', r.path)">
            {{ r.path }}
          </button>
          <span class="ml-auto pl-2 text-[10px] text-neutral-500 dark:text-neutral-400 text-right line-clamp-2 max-w-[45%]">{{ c.note(r) }}</span>
        </li>
      </ul>
    </section>
  </div>
</template>
