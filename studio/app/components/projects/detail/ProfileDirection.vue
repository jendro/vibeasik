<script setup lang="ts">
import type { ProfileHorizon, ProfileRisk, ProfileRoadmapItem, ProjectProfile } from '~/types'

const props = defineProps<{
  projectId: number
  roadmap: ProfileRoadmapItem[]
  risks: ProfileRisk[]
  openQuestions: string[]
}>()
const emit = defineEmits<{ updated: [profile: ProjectProfile] }>()

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
</script>

<template>
  <section class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] p-4 sm:p-5 space-y-5">
    <div>
      <h2 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
        <UIcon name="i-lucide-compass" class="w-4 h-4 text-sky-500" />
        Arah Pengembangan
      </h2>
      <p class="text-xs text-neutral-500">
        Langkah berikutnya, disusun dari todo terbuka, fitur yang belum lengkap, risiko, dan commit terbaru.
      </p>
    </div>

    <div v-if="roadmap.length" class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <div
        v-for="h in horizons"
        :key="h.key"
        class="rounded-lg border border-neutral-200/80 dark:border-white/[0.08] bg-neutral-50/50 dark:bg-[#04111f] p-3 space-y-2"
      >
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="w-2 h-2 rounded-full" :class="h.dot" />
            <span class="text-xs font-bold text-neutral-800 dark:text-neutral-100">{{ h.label }}</span>
          </div>
          <span class="text-xs font-mono text-neutral-400">{{ byHorizon[h.key].length }}</span>
        </div>
        <p class="text-xs text-neutral-400">
          {{ h.hint }}
        </p>
        <div v-if="byHorizon[h.key].length === 0" class="text-xs text-neutral-400 italic py-2">
          Belum ada.
        </div>
        <div
          v-for="r in byHorizon[h.key]"
          :key="r.title"
          class="rounded-md bg-white dark:bg-[#04111f] border border-neutral-200/80 dark:border-white/[0.08] p-2.5 space-y-1"
        >
          <div class="text-xs font-semibold text-neutral-900 dark:text-white leading-snug">
            {{ r.title }}
          </div>
          <p v-if="r.rationale" class="text-xs text-neutral-500 leading-relaxed">
            {{ r.rationale }}
          </p>
          <div v-if="r.related_features.length" class="flex flex-wrap gap-1 pt-0.5">
            <span v-for="f in r.related_features" :key="f" class="px-1.5 py-0.5 rounded text-xs bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-300">{{ f }}</span>
          </div>
        </div>
      </div>
    </div>
    <p v-else class="text-xs text-neutral-400">
      Belum ada roadmap. Jalankan analisis AI atau tambahkan TODO proyek.
    </p>

    <div v-if="openQuestions.length" class="space-y-2">
      <h3 class="text-sm font-semibold text-neutral-800 dark:text-neutral-100 flex items-center gap-1.5">
        <UIcon name="i-lucide-circle-help" class="size-4 text-sky-500" />
        Pertanyaan terbuka ({{ openQuestions.length }})
      </h3>
      <ul class="grid grid-cols-1 md:grid-cols-2 gap-2">
        <li
          v-for="q in openQuestions"
          :key="q"
          class="rounded-xl border border-neutral-200/80 dark:border-white/[0.08] px-3 py-2 text-sm text-neutral-600 dark:text-neutral-300"
        >
          {{ q }}
        </li>
      </ul>
    </div>
  </section>

  <ProjectsDetailProfileRisks :project-id="projectId" :risks="risks" @updated="emit('updated', $event)" />
</template>
