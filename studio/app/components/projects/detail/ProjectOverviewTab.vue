<script setup lang="ts">
import type { Project, ProjectProfile, ProfileMaturityStage } from '~/types'

const props = defineProps<{
  project: Project
  profile: ProjectProfile | null
  loading?: boolean
  generating?: boolean
  generatingLabel?: string
  activePortsCount: number
}>()

const emit = defineEmits<{
  generate: []
  updated: [profile: ProjectProfile]
  switchTab: [tab: string]
}>()

const showFullDescription = ref(false)

const stages: { key: ProfileMaturityStage, label: string }[] = [
  { key: 'idea', label: 'Ide' },
  { key: 'prototype', label: 'Prototipe' },
  { key: 'mvp', label: 'MVP' },
  { key: 'beta', label: 'Beta' },
  { key: 'production', label: 'Produksi' }
]
const stageIndex = computed(() => stages.findIndex(s => s.key === props.profile?.maturity?.stage))

const featureCounts = computed(() => {
  const f = props.profile?.features || []
  return { total: f.length, done: f.filter(x => x.status === 'done').length }
})

// Synergy map counts tables when it exists; otherwise fall back to what the profile found
const tableCount = computed(() => {
  const set = new Set<string>()
  for (const f of props.profile?.features || []) f.tables.forEach(t => set.add(t))
  for (const d of props.profile?.data_entities || []) set.add(d.name)
  return Math.max(props.profile?.stats?.tables ?? 0, set.size)
})

const stats = computed(() => {
  const s = props.profile?.stats
  return [
    { label: 'Aplikasi', value: s?.apps ?? props.project.components?.length ?? 0, sub: `${props.activePortsCount} sedang jalan`, icon: 'i-lucide-boxes', tab: 'apps' },
    { label: 'Fitur', value: featureCounts.value.total, sub: `${featureCounts.value.done} selesai`, icon: 'i-lucide-puzzle', tab: '' },
    { label: 'Halaman', value: s?.pages ?? 0, sub: 'rute frontend', icon: 'i-lucide-panels-top-left', tab: 'technical' },
    { label: 'Endpoint API', value: s?.endpoints ?? 0, sub: `${s?.domains ?? 0} domain`, icon: 'i-lucide-route', tab: 'technical' },
    { label: 'Entitas Data', value: tableCount.value, sub: 'tabel / model', icon: 'i-lucide-database', tab: 'synergy' },
    { label: 'TODO Terbuka', value: s?.open_todos ?? 0, sub: `${s?.done_todos ?? 0} selesai`, icon: 'i-lucide-list-todo', tab: 'ai_chat' }
  ]
})
</script>

<template>
  <div v-if="loading && !profile" class="space-y-4">
    <div class="h-40 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 animate-pulse" />
    <div class="h-64 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 animate-pulse" />
  </div>

  <div v-else-if="profile" class="space-y-5">
    <ProjectsDetailProfileFillPanel
      :project="project"
      :profile="profile"
      :generating="generating"
      :generating-label="generatingLabel"
      @generate="emit('generate')"
      @updated="emit('updated', $event)"
    />

    <!-- Stats strip -->
    <div class="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-6 gap-3">
      <component
        :is="s.tab ? 'button' : 'div'"
        v-for="s in stats"
        :key="s.label"
        type="button"
        class="text-left rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 p-3.5 shadow-2xs transition-colors"
        :class="s.tab ? 'hover:border-sky-400/60 cursor-pointer' : ''"
        @click="s.tab && emit('switchTab', s.tab)"
      >
        <div class="flex items-center justify-between text-neutral-400">
          <span class="text-[10px] uppercase font-semibold tracking-wider">{{ s.label }}</span>
          <UIcon :name="s.icon" class="w-3.5 h-3.5" />
        </div>
        <div class="text-2xl font-bold text-neutral-900 dark:text-white mt-1 font-mono">
          {{ s.value }}
        </div>
        <div class="text-[11px] text-neutral-500">
          {{ s.sub }}
        </div>
      </component>
    </div>

    <!-- Purpose & maturity -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
      <section class="lg:col-span-2 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 p-4 sm:p-5 shadow-xs space-y-4">
        <h2 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-target" class="w-4 h-4 text-sky-500" />
          Tujuan Proyek
        </h2>
        <p v-if="profile.purpose.value_proposition" class="text-base font-semibold text-neutral-800 dark:text-neutral-100 leading-snug">
          {{ profile.purpose.value_proposition }}
        </p>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
          <div class="rounded-lg bg-neutral-50 dark:bg-neutral-900/60 border border-neutral-200/70 dark:border-neutral-800 p-3">
            <div class="text-[10px] uppercase font-bold text-rose-500 mb-1">
              Masalah
            </div>
            <p class="text-neutral-700 dark:text-neutral-300 leading-relaxed">
              {{ profile.purpose.problem || 'Belum diketahui — jalankan analisis AI.' }}
            </p>
          </div>
          <div class="rounded-lg bg-neutral-50 dark:bg-neutral-900/60 border border-neutral-200/70 dark:border-neutral-800 p-3">
            <div class="text-[10px] uppercase font-bold text-emerald-500 mb-1">
              Solusi
            </div>
            <p class="text-neutral-700 dark:text-neutral-300 leading-relaxed">
              {{ profile.purpose.solution || '-' }}
            </p>
          </div>
        </div>
        <div v-if="profile.purpose.target_users?.length" class="space-y-1.5">
          <div class="text-[10px] uppercase font-bold text-neutral-400">
            Target Pengguna
          </div>
          <div class="flex flex-wrap gap-1.5">
            <span v-for="u in profile.purpose.target_users" :key="u" class="inline-flex items-center gap-1 px-2 py-1 rounded-md text-[11px] bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-200">
              <UIcon name="i-lucide-user" class="w-3 h-3 text-neutral-400" />{{ u }}
            </span>
          </div>
        </div>
      </section>

      <section class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 p-4 sm:p-5 shadow-xs space-y-3">
        <h2 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-gauge" class="w-4 h-4 text-sky-500" />
          Tingkat Kematangan
        </h2>
        <div class="flex items-center gap-1">
          <div
            v-for="(st, i) in stages"
            :key="st.key"
            class="flex-1 h-1.5 rounded-full"
            :class="i <= stageIndex ? 'bg-sky-500' : 'bg-neutral-200 dark:bg-neutral-800'"
          />
        </div>
        <div class="flex justify-between text-[10px] text-neutral-400">
          <span v-for="(st, i) in stages" :key="st.key" :class="i === stageIndex ? 'text-sky-600 dark:text-sky-400 font-bold' : ''">{{ st.label }}</span>
        </div>
        <p class="text-xs text-neutral-600 dark:text-neutral-300 leading-relaxed">
          {{ profile.maturity.reason }}
        </p>
      </section>
    </div>

    <!-- Narrative -->
    <section v-if="profile.full_description" class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 p-4 sm:p-5 shadow-xs space-y-3">
      <div class="flex items-center justify-between">
        <h2 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-file-text" class="w-4 h-4 text-sky-500" />
          Keterangan Arsitektur
        </h2>
        <UButton size="xs" color="neutral" variant="ghost" :icon="showFullDescription ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" @click="showFullDescription = !showFullDescription">
          {{ showFullDescription ? 'Ringkas' : 'Baca lengkap' }}
        </UButton>
      </div>
      <div :class="showFullDescription ? '' : 'max-h-28 overflow-hidden [mask-image:linear-gradient(to_bottom,black_55%,transparent)]'">
        <MarkdownViewer :content="profile.full_description" />
      </div>
    </section>

    <ProjectsDetailProfileFeatureList :features="profile.features" />

    <ProjectsDetailProfileDirection
      :roadmap="profile.roadmap"
      :risks="profile.risks"
      :open-questions="profile.open_questions"
    />
  </div>
</template>
