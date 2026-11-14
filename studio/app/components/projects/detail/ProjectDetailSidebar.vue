<script setup lang="ts">
// Navigasi tampilan proyek: bilah tab horizontal yang menempel di bawah navbar (dulu panel kanan tetap).
// Isinya kelompok sidebar yang memuat tab aktif (mis. "Periksa & tingkatkan": Audit · Biaya Proses · Uji Runtime …):
// tab detail proyek berpindah di tempat, halaman lain sekelompok dibuka lewat tautan. Tidak ada menu tersembunyi.
// Nama komponen dipertahankan agar halaman & tautan lama tetap bekerja.
import type { Project, ProjectProfile } from '~/types'
import { buildProjectNavItems, projectTabFor } from '~/utils/sidebarMenuItems'

const props = defineProps<{
  project: Project
  profile: ProjectProfile | null
}>()

const activeTab = defineModel<string>('activeTab', { default: 'overview' })

const { isLocked } = useEdition()
const group = computed(() => projectTabFor(activeTab.value)?.group || '')
const items = computed(() => buildProjectNavItems(props.project.id, props.project)
  .filter(i => i.group === group.value && i.id !== 'home')
  .map(i => ({
    ...i,
    badge: i.tab === 'apps' ? props.project?.components?.length : undefined,
    locked: !i.tab && isLocked(proFeatureForPath(i.to))
  })))

// Bilah tab digulir horizontal di layar sempit: tab aktif selalu digeser ke dalam pandangan.
const bar = useTemplateRef<HTMLElement>('bar')
const revealActive = () => nextTick(() => {
  const el = bar.value?.querySelector<HTMLElement>('[aria-selected="true"]')
  if (el && bar.value) bar.value.scrollLeft = el.offsetLeft - 16 // bar relative: offsetLeft dihitung dari bar
})
onMounted(revealActive)
watch(activeTab, revealActive)
</script>

<template>
  <nav
    class="sticky top-16 z-20 -mx-4 sm:-mx-6 lg:-mx-8 px-4 sm:px-6 lg:px-8 bg-white/90 dark:bg-[#020c18]/90 backdrop-blur-md border-b border-neutral-200 dark:border-white/[0.07]"
    aria-label="Tampilan proyek"
  >
    <div class="flex items-stretch gap-2">
      <div ref="bar" class="relative flex-1 min-w-0 flex items-stretch gap-1 overflow-x-auto no-scrollbar" role="tablist" :aria-label="group">
        <span class="hidden sm:flex shrink-0 items-center pr-2 mr-1 border-r border-neutral-200 dark:border-white/[0.07] text-xs font-medium text-neutral-400 dark:text-slate-500">
          {{ group }}
        </span>
        <template v-for="t in items" :key="t.id">
          <button
            v-if="t.tab"
            type="button"
            role="tab"
            :aria-selected="activeTab === t.tab"
            :title="t.desc"
            class="relative shrink-0 flex items-center gap-2 h-11 px-3 text-sm transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-sky-500 rounded-t-md"
            :class="activeTab === t.tab
              ? 'text-neutral-950 dark:text-white font-medium'
              : 'text-neutral-500 dark:text-slate-400 hover:text-neutral-800 dark:hover:text-slate-200'"
            @click="activeTab = t.tab"
          >
            <UIcon :name="t.icon" class="size-4 shrink-0" :class="activeTab === t.tab ? 'text-sky-600 dark:text-sky-400' : ''" />
            {{ t.label }}
            <span v-if="t.badge !== undefined" class="rounded-md px-1.5 text-xs tabular-nums bg-neutral-100 dark:bg-white/[0.06] text-neutral-600 dark:text-slate-300">{{ t.badge }}</span>
            <span v-if="activeTab === t.tab" class="absolute inset-x-2 -bottom-px h-0.5 rounded-full bg-sky-500" />
          </button>
          <NuxtLink
            v-else
            :to="t.to"
            :title="`${t.desc} — buka halaman ${t.label}`"
            class="shrink-0 flex items-center gap-2 h-11 px-3 text-sm text-neutral-500 dark:text-slate-400 hover:text-neutral-800 dark:hover:text-slate-200 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-sky-500 rounded-t-md"
          >
            <UIcon :name="t.icon" class="size-4 shrink-0" />
            {{ t.label }}
            <UIcon v-if="t.locked" name="i-lucide-lock" class="size-3 shrink-0 text-amber-500" />
          </NuxtLink>
        </template>
      </div>
      <!-- Satu pintu permintaan: fitur baru, perubahan & rombak desain diajukan ke dewan proyek di Ruang Klien -->
      <NuxtLink
        to="/client-room"
        title="Minta fitur baru, perubahan, atau rombak desain ke dewan proyek ini"
        class="shrink-0 flex items-center gap-2 h-11 pl-3 border-l border-neutral-200 dark:border-white/[0.07] text-sm text-sky-700 dark:text-sky-300 hover:text-sky-900 dark:hover:text-sky-100 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-sky-500 rounded-t-md"
      >
        <UIcon name="i-lucide-handshake" class="size-4 shrink-0" />
        <span class="hidden md:inline">Minta fitur di Ruang Klien</span>
        <span class="md:hidden">Ruang Klien</span>
        <UIcon name="i-lucide-arrow-up-right" class="size-3.5 shrink-0 opacity-70" />
      </NuxtLink>
    </div>
  </nav>
</template>
