<script setup lang="ts">
import { useActiveProject } from '~/composables/useActiveProject'

const props = withDefaults(
  defineProps<{
    collapsed?: boolean
    mobileOpen?: boolean
    apiConnected?: boolean
  }>(),
  {
    collapsed: false,
    mobileOpen: false,
    apiConnected: true
  }
)

const emit = defineEmits<{
  (e: 'update:collapsed', val: boolean): void
  (e: 'update:mobileOpen', val: boolean): void
  (e: 'toggleCollapse'): void
  (e: 'closeMobile'): void
}>()

const route = useRoute()
const {
  activeProjectId,
  activeProject,
  fetchProjects
} = useActiveProject()

// Gembok untuk menu fitur berbayar yang belum terbuka (lihat utils/proFeatures.ts).
const { load: loadEdition, isLocked } = useEdition()
onMounted(() => loadEdition())
const lockedItem = (item: { to: string }) => isLocked(proFeatureForPath(item.to))

interface NavItem {
  id: string
  to: string
  label: string
  desc: string
  icon: string
  exact?: boolean
  isDetail?: boolean
  badge?: string
}

const projectNavItems = computed<NavItem[]>(() => {
  const detailTo = activeProjectId.value ? `/projects/${activeProjectId.value}` : '/'
  const detailDesc = activeProject.value ? activeProject.value.name : 'Pilih proyek aktif'
  const detailBadge = activeProject.value ? 'Aktif' : undefined

  return [
    {
      id: 'project-detail',
      to: detailTo,
      label: 'Detail Proyek',
      desc: detailDesc,
      icon: 'i-lucide-folder-git-2',
      exact: false,
      isDetail: true,
      badge: detailBadge
    },
    {
      // Satu-satunya ruang obrolan pengguna (klien) dengan dewan eksekutif proyek aktif.
      id: 'client-room',
      to: '/client-room',
      label: 'Ruang Klien',
      desc: 'Lapor bug, tanya & minta fitur',
      icon: 'i-lucide-handshake',
      exact: false
    },
    {
      id: 'todos',
      to: '/todos',
      label: 'Perencanaan & Todo',
      desc: 'Roadmap & task bertingkat',
      icon: 'i-lucide-clipboard-check',
      exact: false
    },
    {
      id: 'code-map',
      to: '/code-map',
      label: 'Peta Kode',
      desc: 'Indeks RAG kode untuk AI agent',
      icon: 'i-lucide-map',
      exact: false
    },
    {
      id: 'audit-report',
      to: '/audit-report',
      label: 'Audit & Diagnostik',
      desc: 'Query, router & smelly code',
      icon: 'i-lucide-file-text',
      exact: false
    },
    {
      id: 'runtime-test',
      to: '/runtime-test',
      label: 'Uji Runtime',
      desc: 'API, login, kecepatan & responsif',
      icon: 'i-lucide-laptop',
      exact: false
    },
    {
      // Statistik request aplikasi live (lewat vibeasik live) → saran rate limit, cache, LB & pecah service.
      id: 'traffic',
      to: '/traffic',
      label: 'Traffic Analyzer',
      desc: 'Traffic live, rate limit & skala',
      icon: 'i-lucide-activity',
      badge: 'Live',
      exact: false
    },
    {
      id: 'audit-learning',
      to: '/audit-learning',
      label: 'Pembelajaran Audit',
      desc: 'Vonis audit keliru & benar',
      icon: 'i-lucide-graduation-cap',
      exact: false
    },
    {
      id: 'screenshots',
      to: '/screenshots',
      label: 'Screenshot & Visual',
      desc: 'Galeri snapshot UI proyek',
      icon: 'i-lucide-camera',
      exact: false
    },
    {
      id: 'scale-up',
      to: '/scale-up',
      label: 'Scale Up',
      desc: 'Best practice BE, Mobile & FE',
      icon: 'i-lucide-trending-up',
      exact: false
    },
    {
      id: 'queue',
      to: '/queue',
      label: 'Antrean Proses',
      desc: 'Monitor antrean & pipeline',
      icon: 'i-lucide-list-todo',
      exact: false
    },
    {
      id: 'ports',
      to: '/ports',
      label: 'Monitor Port',
      desc: 'Pantau port bentrok live',
      icon: 'i-lucide-network',
      exact: false
    },
    {
      id: 'delivery',
      to: '/delivery',
      label: 'Delivery & Domain',
      desc: 'Beli domain & deploy hosting',
      icon: 'i-lucide-rocket',
      exact: false
    }
  ]
})

const globalNavItems: NavItem[] = [
  {
    id: 'catalog',
    to: '/cockpit',
    label: 'Katalog Proyek',
    desc: 'Pantau bundle & topologi',
    icon: 'i-lucide-layout-grid',
    exact: true
  },
  {
    id: 'adapters',
    to: '/adapters',
    label: 'Katalog Adapter & Stack',
    desc: 'Nuxt, Go, Flutter & versi terbaru',
    icon: 'i-lucide-layers',
    exact: false
  },
  {
    id: 'organizer',
    to: '/organizer',
    label: 'Tata Folder',
    desc: 'Workspace product-centric',
    icon: 'i-lucide-folder-tree',
    exact: false
  },
  {
    id: 'scan',
    to: '/scan',
    label: 'Pindai Laptop',
    desc: 'Deteksi otomatis repo lokal',
    icon: 'i-lucide-folder-search',
    exact: false
  },
  {
    id: 'ai-hub',
    to: '/ai-hub',
    label: 'AI Agent Hub',
    desc: 'Tanamkan System One & MCP',
    icon: 'i-lucide-sparkles',
    exact: false
  },
  {
    id: 'mcp',
    to: '/mcp',
    label: 'AI & MCP Hub',
    desc: 'Protokol konteks AI agent',
    icon: 'i-lucide-bot',
    exact: false
  },
  {
    id: 'orchestrator',
    to: '/orchestrator',
    label: 'Orkestrator Agent',
    desc: 'Suruh agent di laptop bekerja',
    icon: 'i-lucide-network',
    exact: false
  },
  {
    id: 'agent-office',
    to: '/agent-office',
    label: 'Kantor Agent',
    desc: 'Agent AI bekerja secara live',
    icon: 'i-lucide-gamepad-2',
    badge: 'Live',
    exact: false
  },
  {
    id: 'staff',
    to: '/staff',
    label: 'Pegawai',
    desc: 'Agent berkepribadian & belajar',
    icon: 'i-lucide-id-card',
    exact: false
  },
  {
    // Terpisah dari Pegawai: dewan eksekutif terikat satu proyek, pegawai umum lintas proyek.
    id: 'executives',
    to: '/executives',
    label: 'Pegawai Eksekutif',
    desc: 'Dewan CEO, CTO, PO & CMO per proyek',
    icon: 'i-lucide-crown',
    exact: false
  },
  {
    // Bukan milik satu proyek: satu pintu patokan & pengetahuan umum seluruh pekerja AI.
    id: 'knowledge',
    to: '/knowledge',
    label: 'Pusat Pengetahuan',
    desc: 'Satu pintu patokan & belajar AI',
    icon: 'i-lucide-brain-circuit',
    exact: false
  },
  {
    id: 'knowledge-sync',
    to: '/knowledge-sync',
    label: 'Berbagi Pengetahuan',
    desc: 'Persetujuan, kirim otomatis & riwayat',
    icon: 'i-lucide-share-2',
    exact: false
  },
  {
    id: 'knowledge-graph',
    to: '/knowledge-graph',
    label: 'Jaring Pengetahuan',
    desc: 'Peta node & tautan pengetahuan',
    icon: 'i-lucide-waypoints',
    exact: false
  },
  {
    id: 'recap',
    to: '/recap',
    label: 'Rekap Progres',
    desc: 'Commit & todo harian',
    icon: 'i-lucide-trophy',
    exact: false
  },
  {
    id: 'academy',
    to: '/academy',
    label: 'Vibe Academy',
    desc: 'Tutorial dasar & arsitektur',
    icon: 'i-lucide-graduation-cap',
    exact: false
  },
  {
    // Bukan milik satu proyek: referensi desain dipakai bersama oleh semua proyek & agent AI.
    id: 'design-library',
    to: '/design-library',
    label: 'Galeri Desain',
    desc: 'Referensi UI modern untuk AI',
    icon: 'i-lucide-palette',
    exact: false
  },
  {
    id: 'design-lessons',
    to: '/design-lessons',
    label: 'Pelajaran Desain',
    desc: 'Before / after + prinsip desain',
    icon: 'i-lucide-git-compare-arrows',
    exact: false
  },
  {
    id: 'download',
    to: '/download',
    label: 'Pasang vibeasik',
    desc: 'Instal satu baris & MCP',
    icon: 'i-lucide-download',
    exact: false
  },
  {
    id: 'settings',
    to: '/settings',
    label: 'Pengaturan LLM',
    desc: 'System One & model AI',
    icon: 'i-lucide-sliders-horizontal',
    exact: false
  }
]

const isItemActive = (item: NavItem) => {
  if (item.isDetail) {
    return route.path.startsWith('/projects/')
  }
  if (item.exact) {
    return route.path === item.to
  }
  return route.path === item.to || route.path.startsWith(item.to + '/')
}

// Close mobile sidebar on Escape key
const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.mobileOpen) {
    emit('closeMobile')
  }
}

onMounted(() => {
  fetchProjects()
  window.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <!-- ==================== DESKTOP SIDEBAR ==================== -->
  <aside
    class="hidden md:flex flex-col sticky top-0 h-screen border-r border-neutral-200 dark:border-white/10 bg-white dark:bg-[#020c18] transition-all duration-200 shrink-0 select-none z-20"
    :class="collapsed ? 'w-18' : 'w-64'"
  >
    <!-- Brand Header -->
    <div class="h-16 flex items-center px-4 border-b border-neutral-200 dark:border-white/10 justify-between gap-3">
      <NuxtLink to="/cockpit" class="flex items-center gap-3 overflow-hidden group">
        <div class="w-9 h-9 rounded-lg bg-sky-600 dark:bg-sky-500 flex items-center justify-center text-white shadow-xs shrink-0 group-hover:scale-105 transition-transform">
          <UIcon name="i-lucide-layers" class="w-5 h-5" />
        </div>
        <div v-if="!collapsed" class="flex flex-col min-w-0 transition-opacity duration-200">
          <div class="flex items-center gap-1.5">
            <span class="font-bold text-sm tracking-tight text-neutral-900 dark:text-white truncate">vibeasik.com</span>
            <UBadge size="xs" color="primary" variant="subtle">Vibecoder</UBadge>
          </div>
          <span class="text-[10px] text-neutral-500 dark:text-slate-400 font-mono truncate">Delivery & AI Agent Hub</span>
        </div>
      </NuxtLink>
    </div>

    <!-- Navigation Links -->
    <div class="flex-1 py-4 px-2 space-y-1 overflow-y-auto">
      <!-- Bagian Atas: Menu yang berhubungan dan butuh difilter khusus untuk project active -->
      <NuxtLink
        v-for="item in projectNavItems"
        :key="item.id"
        :to="item.to"
        class="flex items-center gap-3 rounded-lg transition-colors group relative"
        :class="[
          collapsed ? 'justify-center p-2.5' : 'px-3 py-2.5',
          isItemActive(item)
            ? 'bg-sky-50 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300 font-medium'
            : 'text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800/80 hover:text-neutral-900 dark:hover:text-white'
        ]"
        :title="collapsed ? (item.desc ? `${item.label} (${item.desc})` : item.label) : undefined"
      >
        <!-- Active bar indicator for collapsed mode -->
        <div
          v-if="collapsed && isItemActive(item)"
          class="absolute left-0 top-1.5 bottom-1.5 w-1 bg-sky-500 rounded-r-md"
        />

        <UIcon
          :name="item.icon"
          class="w-5 h-5 shrink-0 transition-transform group-hover:scale-110"
          :class="isItemActive(item) ? 'text-sky-600 dark:text-sky-400' : 'text-neutral-500 dark:text-neutral-400 group-hover:text-neutral-700 dark:group-hover:text-neutral-200'"
        />

        <div v-if="!collapsed" class="flex flex-col min-w-0">
          <div class="flex items-center gap-1.5 min-w-0">
            <span class="text-sm font-medium leading-tight truncate">{{ item.label }}</span>
            <UIcon v-if="lockedItem(item)" name="i-lucide-lock" class="w-3 h-3 shrink-0 text-amber-500" />
            <UBadge
              v-if="item.badge"
              size="xs"
              color="primary"
              variant="subtle"
              class="text-[9px] px-1 py-0 font-mono shrink-0"
            >
              {{ item.badge }}
            </UBadge>
          </div>
          <span class="text-[11px] text-neutral-500 dark:text-neutral-400 leading-tight truncate font-sans">{{ item.desc }}</span>
        </div>
      </NuxtLink>

      <!-- Pemisah Menu Proyek Aktif dan Menu Global -->
      <hr class="my-2.5 border-neutral-200 dark:border-white/10">

      <!-- Bagian Bawah: Menu Global -->
      <NuxtLink
        v-for="item in globalNavItems"
        :key="item.id"
        :to="item.to"
        class="flex items-center gap-3 rounded-lg transition-colors group relative"
        :class="[
          collapsed ? 'justify-center p-2.5' : 'px-3 py-2.5',
          isItemActive(item)
            ? 'bg-sky-50 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300 font-medium'
            : 'text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800/80 hover:text-neutral-900 dark:hover:text-white'
        ]"
        :title="collapsed ? (item.desc ? `${item.label} (${item.desc})` : item.label) : undefined"
      >
        <!-- Active bar indicator for collapsed mode -->
        <div
          v-if="collapsed && isItemActive(item)"
          class="absolute left-0 top-1.5 bottom-1.5 w-1 bg-sky-500 rounded-r-md"
        />

        <UIcon
          :name="item.icon"
          class="w-5 h-5 shrink-0 transition-transform group-hover:scale-110"
          :class="isItemActive(item) ? 'text-sky-600 dark:text-sky-400' : 'text-neutral-500 dark:text-neutral-400 group-hover:text-neutral-700 dark:group-hover:text-neutral-200'"
        />

        <div v-if="!collapsed" class="flex flex-col min-w-0">
          <span class="text-sm font-medium leading-tight truncate flex items-center gap-1.5">
            {{ item.label }}
            <UIcon v-if="lockedItem(item)" name="i-lucide-lock" class="w-3 h-3 shrink-0 text-amber-500" />
          </span>
          <span class="text-[11px] text-neutral-500 dark:text-neutral-400 leading-tight truncate font-sans">{{ item.desc }}</span>
        </div>
      </NuxtLink>
    </div>

    <!-- Environment & Status info card (expanded only) -->
    <div v-if="!collapsed" class="p-3 mx-2 mb-3 rounded-lg bg-neutral-50 dark:bg-slate-900/50 border border-neutral-200/80 dark:border-white/10 text-xs text-neutral-600 dark:text-neutral-400 space-y-2">
      <div class="flex items-center justify-between">
        <span class="font-medium text-neutral-700 dark:text-neutral-300 text-[11px]">System Status</span>
        <span
          class="inline-flex items-center gap-1 text-[11px] font-mono"
          :class="apiConnected ? 'text-sky-600 dark:text-sky-400' : 'text-rose-600 dark:text-rose-400'"
        >
          <span class="w-1.5 h-1.5 rounded-full" :class="apiConnected ? 'bg-sky-500' : 'bg-rose-500'" />
          {{ apiConnected ? 'Online' : 'Offline' }}
        </span>
      </div>
      <div class="text-[10px] space-y-0.5 font-mono text-neutral-500 dark:text-slate-400">
        <div>Backend: Go Fiber :8765</div>
        <div>Protocol: MCP 2024-11-05</div>
      </div>
    </div>

    <!-- Footer Collapse Button -->
    <div class="p-2 border-t border-neutral-200 dark:border-white/10 flex items-center" :class="collapsed ? 'justify-center' : 'justify-between'">
      <button
        type="button"
        class="flex items-center gap-2 p-2 rounded-lg text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 text-xs font-medium w-full transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
        :class="collapsed ? 'justify-center' : ''"
        :title="collapsed ? 'Expand sidebar' : 'Collapse sidebar'"
        aria-label="Toggle sidebar collapse"
        @click="emit('toggleCollapse')"
      >
        <UIcon
          :name="collapsed ? 'i-lucide-chevron-right' : 'i-lucide-chevron-left'"
          class="w-4 h-4 shrink-0"
        />
        <span v-if="!collapsed" class="truncate font-sans">Collapse Sidebar</span>
      </button>
    </div>
  </aside>

  <!-- ==================== MOBILE DRAWER & BACKDROP ==================== -->
  <Teleport to="body">
    <!-- Backdrop -->
    <Transition
      enter-active-class="transition-opacity ease-out duration-200"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity ease-in duration-150"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="mobileOpen"
        class="fixed inset-0 z-50 bg-neutral-900/60 backdrop-blur-xs md:hidden"
        aria-hidden="true"
        @click="emit('closeMobile')"
      />
    </Transition>

    <!-- Drawer Content -->
    <Transition
      enter-active-class="transition-transform ease-out duration-200"
      enter-from-class="-translate-x-full"
      enter-to-class="translate-x-0"
      leave-active-class="transition-transform ease-in duration-150"
      leave-from-class="translate-x-0"
      leave-to-class="-translate-x-full"
    >
      <div
        v-if="mobileOpen"
        class="fixed inset-y-0 left-0 z-50 w-72 bg-white dark:bg-[#020c18] border-r border-neutral-200 dark:border-white/10 shadow-2xl flex flex-col md:hidden"
        role="dialog"
        aria-modal="true"
        aria-label="Navigation drawer"
      >
        <!-- Drawer Header -->
        <div class="h-16 px-4 flex items-center justify-between border-b border-neutral-200 dark:border-white/10">
          <NuxtLink to="/cockpit" class="flex items-center gap-3" @click="emit('closeMobile')">
            <div class="w-9 h-9 rounded-lg bg-sky-600 flex items-center justify-center text-white shadow-xs">
              <UIcon name="i-lucide-layers" class="w-5 h-5" />
            </div>
            <div class="flex flex-col">
              <div class="flex items-center gap-1.5">
                <span class="font-bold text-sm tracking-tight text-neutral-900 dark:text-white">vibeasik.com</span>
                <UBadge size="xs" color="primary" variant="subtle">Vibecoder</UBadge>
              </div>
              <span class="text-[10px] text-neutral-500 dark:text-slate-400 font-mono">Delivery & AI Platform</span>
            </div>
          </NuxtLink>

          <button
            type="button"
            class="p-2 -mr-1 text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-200 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
            aria-label="Close navigation drawer"
            @click="emit('closeMobile')"
          >
            <UIcon name="i-lucide-x" class="w-5 h-5" />
          </button>
        </div>

        <!-- Drawer Navigation -->
        <div class="flex-1 py-4 px-3 space-y-1.5 overflow-y-auto">
          <!-- Project Specific Menu -->
          <NuxtLink
            v-for="item in projectNavItems"
            :key="item.id"
            :to="item.to"
            class="flex items-center gap-3 px-3 py-3 rounded-lg transition-colors group min-h-[44px]"
            :class="[
              isItemActive(item)
                ? 'bg-sky-50 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300 font-medium'
                : 'text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 hover:text-neutral-900 dark:hover:text-white'
            ]"
            @click="emit('closeMobile')"
          >
            <UIcon
              :name="item.icon"
              class="w-5 h-5 shrink-0"
              :class="isItemActive(item) ? 'text-sky-600 dark:text-sky-400' : 'text-neutral-500 dark:text-neutral-400'"
            />
            <div class="flex flex-col min-w-0">
              <div class="flex items-center gap-1.5 min-w-0">
                <span class="text-sm font-medium leading-tight truncate">{{ item.label }}</span>
                <UBadge
                  v-if="item.badge"
                  size="xs"
                  color="primary"
                  variant="subtle"
                  class="text-[9px] px-1 py-0 font-mono shrink-0"
                >
                  {{ item.badge }}
                </UBadge>
              </div>
              <span class="text-xs text-neutral-500 dark:text-neutral-400 leading-tight truncate">{{ item.desc }}</span>
            </div>
          </NuxtLink>

          <!-- Pemisah Menu Proyek Aktif dan Menu Global -->
          <hr class="my-2.5 border-neutral-200 dark:border-white/10">

          <!-- Global Menu -->
          <NuxtLink
            v-for="item in globalNavItems"
            :key="item.id"
            :to="item.to"
            class="flex items-center gap-3 px-3 py-3 rounded-lg transition-colors group min-h-[44px]"
            :class="[
              isItemActive(item)
                ? 'bg-sky-50 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300 font-medium'
                : 'text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 hover:text-neutral-900 dark:hover:text-white'
            ]"
            @click="emit('closeMobile')"
          >
            <UIcon
              :name="item.icon"
              class="w-5 h-5 shrink-0"
              :class="isItemActive(item) ? 'text-sky-600 dark:text-sky-400' : 'text-neutral-500 dark:text-neutral-400'"
            />
            <div class="flex flex-col min-w-0">
              <span class="text-sm font-medium leading-tight truncate">{{ item.label }}</span>
              <span class="text-xs text-neutral-500 dark:text-neutral-400 leading-tight truncate">{{ item.desc }}</span>
            </div>
          </NuxtLink>
        </div>

        <!-- Drawer Footer Info -->
        <div class="p-4 border-t border-neutral-200 dark:border-white/10 bg-neutral-50/50 dark:bg-slate-900/40 space-y-2">
          <div class="flex items-center justify-between text-xs">
            <span class="font-medium text-neutral-700 dark:text-neutral-300">Go Backend API</span>
            <span
              class="font-mono inline-flex items-center gap-1"
              :class="apiConnected ? 'text-sky-600 dark:text-sky-400' : 'text-rose-600 dark:text-rose-400'"
            >
              <span class="w-1.5 h-1.5 rounded-full" :class="apiConnected ? 'bg-sky-500' : 'bg-rose-500'" />
              {{ apiConnected ? ':8765 Active' : 'Offline' }}
            </span>
          </div>
          <div class="text-[11px] text-neutral-500 dark:text-slate-400 font-mono">
            MCP Protocol 2024-11-05
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
