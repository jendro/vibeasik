<script setup lang="ts">
import type { ProfileFeature, ProfileFeatureStatus } from '~/types'

const props = defineProps<{
  features: ProfileFeature[]
}>()

const statusFilter = ref<'all' | ProfileFeatureStatus>('all')
const appFilter = ref('all')
const expanded = ref<Record<string, boolean>>({})

const statusMeta: Record<ProfileFeatureStatus, { label: string, color: 'success' | 'warning' | 'neutral', icon: string }> = {
  done: { label: 'Selesai', color: 'success', icon: 'i-lucide-circle-check' },
  partial: { label: 'Sebagian', color: 'warning', icon: 'i-lucide-circle-dashed' },
  planned: { label: 'Rencana', color: 'neutral', icon: 'i-lucide-circle-dot' }
}

const counts = computed(() => {
  const c = { all: props.features.length, done: 0, partial: 0, planned: 0 }
  for (const f of props.features) c[f.status]++
  return c
})

const appOptions = computed(() => {
  const set = new Set<string>()
  for (const f of props.features) f.apps.forEach(a => set.add(a))
  return [{ label: 'Semua aplikasi', value: 'all' }, ...[...set].sort().map(a => ({ label: a, value: a }))]
})

const filtered = computed(() => props.features.filter(f =>
  (statusFilter.value === 'all' || f.status === statusFilter.value)
  && (appFilter.value === 'all' || f.apps.includes(appFilter.value))
))

const statusTabs = computed(() => [
  { value: 'all' as const, label: 'Semua', count: counts.value.all },
  { value: 'done' as const, label: 'Selesai', count: counts.value.done },
  { value: 'partial' as const, label: 'Sebagian', count: counts.value.partial },
  { value: 'planned' as const, label: 'Rencana', count: counts.value.planned }
])

const hasDetail = (f: ProfileFeature) => f.pages.length + f.endpoints.length + f.tables.length + f.evidence.length > 0
</script>

<template>
  <section class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 p-4 sm:p-5 shadow-xs space-y-4">
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-3">
      <div>
        <h2 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-puzzle" class="w-4 h-4 text-sky-500" />
          Fitur Proyek
        </h2>
        <p class="text-xs text-neutral-500">
          Dikelompokkan per kapabilitas — satu fitur bisa melibatkan beberapa aplikasi sekaligus.
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <div class="inline-flex rounded-lg border border-neutral-200 dark:border-neutral-800 p-0.5 bg-neutral-50 dark:bg-neutral-900">
          <button
            v-for="t in statusTabs"
            :key="t.value"
            type="button"
            class="px-2.5 py-1 rounded-md text-[11px] font-medium transition-colors cursor-pointer"
            :class="statusFilter === t.value
              ? 'bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white shadow-2xs'
              : 'text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-200'"
            @click="statusFilter = t.value"
          >
            {{ t.label }} <span class="font-mono text-neutral-400">{{ t.count }}</span>
          </button>
        </div>
        <USelect v-model="appFilter" :items="appOptions" size="xs" class="w-44" />
      </div>
    </div>

    <div v-if="filtered.length === 0" class="py-8 text-center text-xs text-neutral-400">
      Tidak ada fitur yang cocok dengan filter.
    </div>

    <div v-else class="grid grid-cols-1 xl:grid-cols-2 gap-3">
      <article
        v-for="f in filtered"
        :key="f.name"
        class="rounded-lg border border-neutral-200/80 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-900/40 p-3.5 space-y-2.5"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-neutral-900 dark:text-white leading-snug">
              {{ f.name }}
            </h3>
            <p v-if="f.domain && f.domain !== f.name" class="text-[10px] uppercase tracking-wider text-neutral-400 mt-0.5">
              {{ f.domain }}
            </p>
          </div>
          <UBadge :color="statusMeta[f.status].color" variant="subtle" size="sm" :icon="statusMeta[f.status].icon" class="shrink-0">
            {{ statusMeta[f.status].label }}
          </UBadge>
        </div>

        <p v-if="f.description" class="text-xs text-neutral-600 dark:text-neutral-300 leading-relaxed">
          {{ f.description }}
        </p>

        <div class="flex flex-wrap items-center gap-1.5">
          <span
            v-for="a in f.apps"
            :key="a"
            class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium bg-sky-50 dark:bg-sky-950/40 text-sky-700 dark:text-sky-300 border border-sky-200/70 dark:border-sky-800/60"
          >
            <UIcon name="i-lucide-box" class="w-3 h-3" />{{ a }}
          </span>
          <span class="text-[10px] font-mono text-neutral-400">
            {{ f.pages.length }} halaman · {{ f.endpoints.length }} endpoint<template v-if="f.tables.length"> · {{ f.tables.length }} tabel</template>
          </span>
        </div>

        <button
          v-if="hasDetail(f)"
          type="button"
          class="inline-flex items-center gap-1 text-[11px] text-sky-600 dark:text-sky-400 hover:underline cursor-pointer"
          @click="expanded[f.name] = !expanded[f.name]"
        >
          <UIcon :name="expanded[f.name] ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" class="w-3.5 h-3.5" />
          {{ expanded[f.name] ? 'Sembunyikan detail' : 'Lihat halaman, endpoint & bukti' }}
        </button>

        <div v-if="expanded[f.name]" class="space-y-2 pt-2 border-t border-neutral-200/70 dark:border-neutral-800 text-[11px]">
          <div v-if="f.pages.length">
            <div class="font-semibold text-neutral-500 mb-1">
              Halaman
            </div>
            <div class="flex flex-wrap gap-1">
              <code v-for="p in f.pages" :key="p" class="px-1.5 py-0.5 rounded bg-white dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700">{{ p }}</code>
            </div>
          </div>
          <div v-if="f.endpoints.length">
            <div class="font-semibold text-neutral-500 mb-1">
              Endpoint
            </div>
            <div class="flex flex-wrap gap-1">
              <code v-for="e in f.endpoints" :key="e" class="px-1.5 py-0.5 rounded bg-white dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700">{{ e }}</code>
            </div>
          </div>
          <div v-if="f.tables.length">
            <div class="font-semibold text-neutral-500 mb-1">
              Tabel
            </div>
            <div class="flex flex-wrap gap-1">
              <code v-for="t in f.tables" :key="t" class="px-1.5 py-0.5 rounded bg-amber-50 dark:bg-amber-950/30 border border-amber-200/70 dark:border-amber-800/50">{{ t }}</code>
            </div>
          </div>
          <div v-if="f.evidence.length">
            <div class="font-semibold text-neutral-500 mb-1">
              Bukti
            </div>
            <ul class="list-disc pl-4 space-y-0.5 text-neutral-600 dark:text-neutral-400">
              <li v-for="e in f.evidence" :key="e">
                {{ e }}
              </li>
            </ul>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>
