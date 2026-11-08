<script setup lang="ts">
import { useActiveProject } from '~/composables/useActiveProject'
import { GLOBAL_NAV_ITEMS, buildProjectNavItems, type NavItem } from '~/utils/sidebarMenuItems'

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
// Lencana paket di header sidebar: "Pro" bila akun berpaket berbayar, selain itu "Free".
const { account, load: loadAccount } = useAuth()
const planLabel = computed(() => (account.value?.logged_in && account.value.plan !== 'free' ? 'Pro' : 'Free'))
onMounted(() => {
  loadEdition()
  loadAccount()
})
const lockedItem = (item: { to: string }) => isLocked(proFeatureForPath(item.to))

const projectNavItems = computed<NavItem[]>(() => buildProjectNavItems(activeProjectId.value, activeProject.value))
const globalNavItems = GLOBAL_NAV_ITEMS

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
            <UBadge size="xs" :color="planLabel === 'Pro' ? 'warning' : 'neutral'" variant="subtle">{{ planLabel }}</UBadge>
          </div>
          <span class="text-xs text-neutral-500 dark:text-slate-400 font-mono truncate">Delivery & AI Agent Hub</span>
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
              class="text-xs px-1 py-0 font-mono shrink-0"
            >
              {{ item.badge }}
            </UBadge>
          </div>
          <span class="text-xs text-neutral-500 dark:text-neutral-400 leading-tight truncate font-sans">{{ item.desc }}</span>
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
          <span class="text-xs text-neutral-500 dark:text-neutral-400 leading-tight truncate font-sans">{{ item.desc }}</span>
        </div>
      </NuxtLink>
    </div>

    <!-- Environment & Status info card (expanded only) -->
    <div v-if="!collapsed" class="p-3 mx-2 mb-3 rounded-lg bg-neutral-50 dark:bg-slate-900/50 border border-neutral-200/80 dark:border-white/10 text-xs text-neutral-600 dark:text-neutral-400 space-y-2">
      <div class="flex items-center justify-between">
        <span class="font-medium text-neutral-700 dark:text-neutral-300 text-xs">System Status</span>
        <span
          class="inline-flex items-center gap-1 text-xs font-mono"
          :class="apiConnected ? 'text-sky-600 dark:text-sky-400' : 'text-rose-600 dark:text-rose-400'"
        >
          <span class="w-1.5 h-1.5 rounded-full" :class="apiConnected ? 'bg-sky-500' : 'bg-rose-500'" />
          {{ apiConnected ? 'Online' : 'Offline' }}
        </span>
      </div>
      <div class="text-xs space-y-0.5 font-mono text-neutral-500 dark:text-slate-400">
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
                <UBadge size="xs" :color="planLabel === 'Pro' ? 'warning' : 'neutral'" variant="subtle">{{ planLabel }}</UBadge>
              </div>
              <span class="text-xs text-neutral-500 dark:text-slate-400 font-mono">Delivery & AI Platform</span>
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
                  class="text-xs px-1 py-0 font-mono shrink-0"
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
          <div class="text-xs text-neutral-500 dark:text-slate-400 font-mono">
            MCP Protocol 2024-11-05
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
