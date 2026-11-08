<script setup lang="ts">
import type { Project, ProjectProfile } from '~/types'

const props = defineProps<{
  project: Project
  profile: ProjectProfile | null
}>()

const activeTab = defineModel<string>('activeTab', { default: 'overview' })

interface TabItem {
  id: string
  title: string
  desc: string
  icon: string
  badge?: number
}

interface TabGroup {
  label: string
  items: TabItem[]
}

const navGroups = computed<TabGroup[]>(() => [
  {
    label: 'Memahami Proyek',
    items: [
      { id: 'overview', title: 'Ringkasan', desc: 'Tujuan, fitur & arah pengembangan', icon: 'i-lucide-layout-dashboard', badge: props.profile?.features?.length || undefined },
      { id: 'apps', title: 'Aplikasi', desc: 'Modul & hubungan antar aplikasi', icon: 'i-lucide-boxes', badge: props.project?.components?.length },
      { id: 'technical', title: 'Rute, File & Env', desc: 'Katalog router, struktur & .env', icon: 'i-lucide-book-open-check' },
      { id: 'synergy', title: 'Alur Data', desc: 'Halaman ➔ API ➔ Database', icon: 'i-lucide-network' }
    ]
  },
  {
    label: 'Mengembangkan',
    items: [
      { id: 'ai_chat', title: 'Kaji Ide & TODO', desc: 'AI architect & planner TODO', icon: 'i-lucide-message-square-code' },
      { id: 'visual_audit', title: 'Live Visual', desc: 'Snapshot FE & audit multimodal', icon: 'i-lucide-camera' }
    ]
  }
])

const healthTone = computed(() => {
  const score = props.project.health_score
  if (score >= 90) return { cls: 'text-sky-600 dark:text-sky-400 border-sky-500/30 bg-sky-50 dark:bg-sky-950/30', label: 'Konfigurasi Optimal' }
  if (score >= 70) return { cls: 'text-amber-600 dark:text-amber-400 border-amber-500/30 bg-amber-50 dark:bg-amber-950/30', label: 'Perlu Optimasi' }
  return { cls: 'text-rose-600 dark:text-rose-400 border-rose-500/30 bg-rose-50 dark:bg-rose-950/30', label: 'Peringatan Kesehatan' }
})

const maturityLabel: Record<string, string> = {
  idea: 'Ide', prototype: 'Prototipe', mvp: 'MVP', beta: 'Beta', production: 'Produksi'
}
</script>

<template>
  <aside class="w-full lg:w-72 xl:w-80 shrink-0 border-t lg:border-t-0 lg:border-l border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 lg:fixed lg:right-0 lg:top-16 lg:bottom-0 lg:overflow-y-auto p-4 sm:p-5 pb-16 sm:pb-20 space-y-5 z-20">
    <!-- Health & maturity -->
    <div class="p-4 rounded-xl border border-neutral-200/80 dark:border-neutral-800 bg-neutral-50/70 dark:bg-neutral-800/40 shadow-2xs space-y-3">
      <div class="flex items-center gap-3">
        <div class="px-3 py-2 rounded-xl text-center border font-mono shrink-0" :class="healthTone.cls">
          <div class="text-xl font-bold leading-none">
            {{ project.health_score }}<span class="text-xs font-normal">/100</span>
          </div>
          <div class="text-xs uppercase font-semibold mt-0.5">
            Health
          </div>
        </div>
        <div class="text-xs min-w-0 space-y-0.5">
          <div class="font-bold text-neutral-900 dark:text-white truncate">
            {{ healthTone.label }}
          </div>
          <div v-if="profile" class="text-xs text-neutral-500 truncate">
            Tahap: <span class="font-semibold text-neutral-700 dark:text-neutral-300">{{ maturityLabel[profile.maturity.stage] || profile.maturity.stage }}</span>
            <span v-if="profile.source === 'heuristic'" class="text-amber-500"> (perkiraan)</span>
          </div>
          <div class="text-xs font-mono text-neutral-400">
            ID #{{ project.id }}
          </div>
        </div>
      </div>
    </div>

    <!-- Navigation -->
    <nav class="space-y-4 w-full" aria-label="Navigasi Tampilan Proyek">
      <div v-for="group in navGroups" :key="group.label" class="space-y-1.5">
        <div class="px-1 text-xs font-semibold text-neutral-400 uppercase tracking-wider">
          {{ group.label }}
        </div>
        <button
          v-for="t in group.items"
          :key="t.id"
          type="button"
          class="w-full group text-left px-2.5 py-2 rounded-lg transition-all duration-150 cursor-pointer flex items-center justify-between gap-2.5 border"
          :class="activeTab === t.id
            ? 'bg-sky-50 dark:bg-sky-950/40 border-sky-500/40 text-sky-950 dark:text-sky-100 shadow-2xs'
            : 'border-transparent text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100/80 dark:hover:bg-neutral-800/60'"
          :aria-current="activeTab === t.id ? 'page' : undefined"
          @click="activeTab = t.id"
        >
          <div class="flex items-center gap-2.5 min-w-0">
            <div
              class="p-1.5 rounded-md shrink-0 transition-colors"
              :class="activeTab === t.id
                ? 'bg-sky-500 text-white'
                : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-500 dark:text-neutral-400 group-hover:text-neutral-800 dark:group-hover:text-neutral-200'"
            >
              <UIcon :name="t.icon" class="w-4 h-4" />
            </div>
            <div class="min-w-0">
              <div class="text-xs font-bold leading-tight truncate" :class="activeTab === t.id ? 'text-sky-900 dark:text-sky-200' : 'text-neutral-800 dark:text-neutral-200'">
                {{ t.title }}
              </div>
              <div class="text-xs leading-tight truncate mt-0.5" :class="activeTab === t.id ? 'text-sky-700/90 dark:text-sky-300/80' : 'text-neutral-500 dark:text-neutral-400'">
                {{ t.desc }}
              </div>
            </div>
          </div>
          <UBadge
            v-if="t.badge !== undefined"
            size="xs"
            :color="activeTab === t.id ? 'primary' : 'neutral'"
            variant="subtle"
            class="font-mono text-xs shrink-0"
          >
            {{ t.badge }}
          </UBadge>
        </button>
      </div>
    </nav>
  </aside>
</template>
