<script setup lang="ts">
// Navigasi tampilan proyek: bilah tab horizontal yang menempel di bawah navbar (dulu panel kanan tetap).
// Nama komponen dipertahankan agar halaman & tautan lama tetap bekerja.
import type { Project, ProjectProfile } from '~/types'

const props = defineProps<{
  project: Project
  profile: ProjectProfile | null
}>()

const activeTab = defineModel<string>('activeTab', { default: 'overview' })

const tabs = computed(() => [
  { id: 'overview', title: 'Ringkasan', icon: 'i-lucide-layout-dashboard', desc: 'Tujuan, fitur & arah pengembangan' },
  { id: 'recap', title: 'Rekap Pemilik', icon: 'i-lucide-chart-no-axes-column', desc: 'Progres, bug ditutup, perbaikan algoritma & kesehatan kode' },
  { id: 'cost', title: 'Biaya Proses', icon: 'i-lucide-sigma', desc: 'Rumus tick T(n) & Big-O tiap rute dan fungsi' },
  { id: 'apps', title: 'Aplikasi', icon: 'i-lucide-boxes', desc: 'Modul & hubungan antar aplikasi', badge: props.project?.components?.length },
  { id: 'technical', title: 'Rute, File & Env', icon: 'i-lucide-book-open-check', desc: 'Katalog router, struktur & .env' },
  { id: 'synergy', title: 'Alur Data', icon: 'i-lucide-network', desc: 'Halaman → API → Database' },
  { id: 'visual_audit', title: 'Live Visual', icon: 'i-lucide-camera', desc: 'Snapshot FE & audit multimodal' }
])
</script>

<template>
  <nav
    class="sticky top-16 z-20 -mx-4 sm:-mx-6 lg:-mx-8 px-4 sm:px-6 lg:px-8 bg-white/90 dark:bg-[#020c18]/90 backdrop-blur-md border-b border-neutral-200 dark:border-white/[0.07]"
    aria-label="Tampilan proyek"
  >
    <div class="flex items-stretch gap-2">
      <div class="flex-1 min-w-0 flex gap-1 overflow-x-auto no-scrollbar" role="tablist">
        <button
          v-for="t in tabs"
          :key="t.id"
          type="button"
          role="tab"
          :aria-selected="activeTab === t.id"
          :title="t.desc"
          class="relative shrink-0 flex items-center gap-2 h-11 px-3 text-sm transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-sky-500 rounded-t-md"
          :class="activeTab === t.id
            ? 'text-neutral-950 dark:text-white font-medium'
            : 'text-neutral-500 dark:text-slate-400 hover:text-neutral-800 dark:hover:text-slate-200'"
          @click="activeTab = t.id"
        >
          <UIcon :name="t.icon" class="size-4 shrink-0" :class="activeTab === t.id ? 'text-sky-600 dark:text-sky-400' : ''" />
          {{ t.title }}
          <span v-if="t.badge !== undefined" class="rounded-md px-1.5 text-xs tabular-nums bg-neutral-100 dark:bg-white/[0.06] text-neutral-600 dark:text-slate-300">{{ t.badge }}</span>
          <span v-if="activeTab === t.id" class="absolute inset-x-2 -bottom-px h-0.5 rounded-full bg-sky-500" />
        </button>
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
