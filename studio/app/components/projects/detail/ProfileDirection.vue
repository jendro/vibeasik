<script setup lang="ts">
import type { ProfileHorizon, ProfileRisk, ProfileRoadmapItem } from '~/types'

const props = defineProps<{
  roadmap: ProfileRoadmapItem[]
  risks: ProfileRisk[]
  openQuestions: string[]
}>()

const horizons: { key: ProfileHorizon, label: string, hint: string, dot: string }[] = [
  { key: 'now', label: 'Sekarang', hint: 'Dikerjakan / harus segera', dot: 'bg-sky-500' },
  { key: 'next', label: 'Berikutnya', hint: 'Setelah prioritas saat ini', dot: 'bg-amber-500' },
  { key: 'later', label: 'Nanti', hint: 'Arah jangka panjang', dot: 'bg-neutral-400' }
]

const byHorizon = computed(() => {
  const map: Record<ProfileHorizon, ProfileRoadmapItem[]> = { now: [], next: [], later: [] }
  for (const r of props.roadmap) map[r.horizon]?.push(r)
  return map
})

const severityOrder = { high: 0, medium: 1, low: 2 }
const sortedRisks = computed(() => [...props.risks].sort((a, b) => severityOrder[a.severity] - severityOrder[b.severity]))

const severityClass: Record<ProfileRisk['severity'], string> = {
  high: 'bg-rose-50 dark:bg-rose-950/40 text-rose-700 dark:text-rose-300 border-rose-200 dark:border-rose-800/60',
  medium: 'bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300 border-amber-200 dark:border-amber-800/60',
  low: 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-300 border-neutral-200 dark:border-neutral-700'
}
</script>

<template>
  <section class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 p-4 sm:p-5 shadow-xs space-y-5">
    <div>
      <h2 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
        <UIcon name="i-lucide-compass" class="w-4 h-4 text-sky-500" />
        Arah Pengembangan
      </h2>
      <p class="text-xs text-neutral-500">
        Roadmap disusun dari TODO terbuka, fitur yang belum lengkap, risiko, dan commit terbaru.
      </p>
    </div>

    <div v-if="roadmap.length" class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <div
        v-for="h in horizons"
        :key="h.key"
        class="rounded-lg border border-neutral-200/80 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-900/40 p-3 space-y-2"
      >
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="w-2 h-2 rounded-full" :class="h.dot" />
            <span class="text-xs font-bold text-neutral-800 dark:text-neutral-100">{{ h.label }}</span>
          </div>
          <span class="text-[10px] font-mono text-neutral-400">{{ byHorizon[h.key].length }}</span>
        </div>
        <p class="text-[10px] text-neutral-400">
          {{ h.hint }}
        </p>
        <div v-if="byHorizon[h.key].length === 0" class="text-[11px] text-neutral-400 italic py-2">
          Belum ada.
        </div>
        <div
          v-for="r in byHorizon[h.key]"
          :key="r.title"
          class="rounded-md bg-white dark:bg-neutral-900 border border-neutral-200/80 dark:border-neutral-800 p-2.5 space-y-1"
        >
          <div class="text-xs font-semibold text-neutral-900 dark:text-white leading-snug">
            {{ r.title }}
          </div>
          <p v-if="r.rationale" class="text-[11px] text-neutral-500 leading-relaxed">
            {{ r.rationale }}
          </p>
          <div v-if="r.related_features.length" class="flex flex-wrap gap-1 pt-0.5">
            <span v-for="f in r.related_features" :key="f" class="px-1.5 py-0.5 rounded text-[10px] bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-300">{{ f }}</span>
          </div>
        </div>
      </div>
    </div>
    <p v-else class="text-xs text-neutral-400">
      Belum ada roadmap. Jalankan analisis AI atau tambahkan TODO proyek.
    </p>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <div class="space-y-2">
        <h3 class="text-xs font-bold text-neutral-700 dark:text-neutral-200 flex items-center gap-1.5">
          <UIcon name="i-lucide-triangle-alert" class="w-3.5 h-3.5 text-amber-500" />
          Risiko ({{ risks.length }})
        </h3>
        <p v-if="!risks.length" class="text-[11px] text-neutral-400">
          Tidak ada risiko yang tercatat.
        </p>
        <div
          v-for="r in sortedRisks"
          :key="r.title"
          class="rounded-md border px-3 py-2 text-xs"
          :class="severityClass[r.severity]"
        >
          <div class="flex items-center justify-between gap-2">
            <span class="font-semibold">{{ r.title }}</span>
            <span class="text-[10px] uppercase font-bold tracking-wider shrink-0">{{ r.severity }}</span>
          </div>
          <p v-if="r.detail" class="text-[11px] mt-0.5 opacity-90 leading-relaxed">
            {{ r.detail }}
          </p>
        </div>
      </div>

      <div class="space-y-2">
        <h3 class="text-xs font-bold text-neutral-700 dark:text-neutral-200 flex items-center gap-1.5">
          <UIcon name="i-lucide-circle-help" class="w-3.5 h-3.5 text-sky-500" />
          Pertanyaan Terbuka ({{ openQuestions.length }})
        </h3>
        <p v-if="!openQuestions.length" class="text-[11px] text-neutral-400">
          Tidak ada pertanyaan terbuka.
        </p>
        <ul v-else class="space-y-1.5">
          <li
            v-for="q in openQuestions"
            :key="q"
            class="rounded-md border border-neutral-200/80 dark:border-neutral-800 px-3 py-2 text-xs text-neutral-600 dark:text-neutral-300"
          >
            {{ q }}
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>
