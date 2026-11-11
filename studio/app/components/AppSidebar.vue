<script setup lang="ts">
// Sidebar Studio: merek + edisi, navigasi berkelompok (AppSidebarNav), status engine yang sebenarnya.
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
  (e: 'toggleCollapse'): void
  (e: 'closeMobile'): void
}>()

const { fetchProjects } = useActiveProject()
const { load: loadEdition, info: editionInfo } = useEdition()
const { account, load: loadAccount } = useAuth()

// Lencana paket: "Pro" bila akun berpaket berbayar atau engine ini membuka semua fitur Pro (build pemilik),
// "Free" bila masih ada fitur yang terkunci.
const planLabel = computed(() => {
  if (account.value?.logged_in && account.value.plan !== 'free') return 'Pro'
  const feats = editionInfo.value?.features
  if (feats && Object.keys(feats).length > 0 && Object.values(feats).every(Boolean)) return 'Pro'
  return 'Free'
})

// Alamat engine yang sebenarnya (bukan ditulis tetap :8765).
const engineHost = useEngineHost()

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.mobileOpen) emit('closeMobile')
}

onMounted(() => {
  loadEdition()
  loadAccount()
  fetchProjects()
  window.addEventListener('keydown', onKeydown)
})
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <!-- ==================== DESKTOP SIDEBAR ==================== -->
  <aside
    class="hidden md:flex flex-col sticky top-0 h-screen border-r border-neutral-200 dark:border-white/[0.07] bg-neutral-50/60 dark:bg-[#020a14] transition-[width] duration-200 shrink-0 select-none z-20"
    :class="collapsed ? 'w-[72px]' : 'w-64'"
  >
    <!-- Merek -->
    <div class="h-16 shrink-0 flex items-center gap-3 border-b border-neutral-200 dark:border-white/[0.07]" :class="collapsed ? 'justify-center px-2' : 'px-4'">
      <NuxtLink to="/" class="flex items-center gap-2.5 min-w-0" title="Beranda vibeasik">
        <AppLogo class="size-8 shrink-0" />
        <div v-if="!collapsed" class="min-w-0 leading-tight">
          <div class="flex items-center gap-1.5">
            <span class="font-semibold text-[15px] tracking-tight text-neutral-950 dark:text-white">vibeasik</span>
            <span
              class="rounded-md px-1.5 py-px text-xs font-medium"
              :class="planLabel === 'Pro' ? 'bg-amber-500/10 text-amber-700 dark:text-amber-400' : 'bg-neutral-900/5 text-neutral-500 dark:bg-white/5 dark:text-slate-400'"
            >{{ planLabel }}</span>
          </div>
          <AppVersionBadge />
        </div>
      </NuxtLink>
    </div>

    <div class="flex-1 overflow-y-auto overflow-x-hidden py-4" :class="collapsed ? 'px-2' : 'px-3'">
      <AppSidebarNav :collapsed="collapsed" />
    </div>

    <!-- Status engine + lipat -->
    <div class="shrink-0 border-t border-neutral-200 dark:border-white/[0.07] p-2 space-y-1">
      <div
        class="flex items-center gap-2 rounded-lg text-xs"
        :class="collapsed ? 'justify-center h-8' : 'px-2.5 h-8'"
        :title="`Engine ${apiConnected ? 'terhubung' : 'tidak terjangkau'} di ${engineHost}`"
      >
        <span class="relative flex size-2 shrink-0">
          <span v-if="apiConnected" class="absolute inline-flex size-full rounded-full bg-emerald-400 opacity-60 animate-ping" />
          <span class="relative inline-flex size-2 rounded-full" :class="apiConnected ? 'bg-emerald-500' : 'bg-rose-500'" />
        </span>
        <template v-if="!collapsed">
          <span class="text-neutral-600 dark:text-slate-300 shrink-0">{{ apiConnected ? 'Engine' : 'Engine offline' }}</span>
          <span class="ml-auto font-mono text-neutral-400 dark:text-slate-500 truncate">{{ engineHost }}</span>
        </template>
      </div>
      <button
        type="button"
        class="w-full flex items-center gap-2 h-8 rounded-lg text-xs text-neutral-500 dark:text-slate-400 hover:bg-neutral-900/[0.04] dark:hover:bg-white/[0.04] hover:text-neutral-800 dark:hover:text-slate-200 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
        :class="collapsed ? 'justify-center' : 'px-2.5'"
        :title="collapsed ? 'Lebarkan sidebar' : 'Ciutkan sidebar'"
        :aria-label="collapsed ? 'Lebarkan sidebar' : 'Ciutkan sidebar'"
        @click="emit('toggleCollapse')"
      >
        <UIcon :name="collapsed ? 'i-lucide-panel-left-open' : 'i-lucide-panel-left-close'" class="size-4 shrink-0" />
        <span v-if="!collapsed">Ciutkan</span>
      </button>
    </div>
  </aside>

  <!-- ==================== LACI MOBILE ==================== -->
  <Teleport to="body">
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
        class="fixed inset-0 z-50 bg-neutral-950/60 backdrop-blur-xs md:hidden"
        aria-hidden="true"
        @click="emit('closeMobile')"
      />
    </Transition>
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
        class="fixed inset-y-0 left-0 z-50 w-72 bg-white dark:bg-[#020a14] border-r border-neutral-200 dark:border-white/[0.07] shadow-2xl flex flex-col md:hidden"
        role="dialog"
        aria-modal="true"
        aria-label="Menu navigasi"
      >
        <div class="h-16 px-4 flex items-center justify-between border-b border-neutral-200 dark:border-white/[0.07]">
          <NuxtLink to="/" class="flex items-center gap-2.5" @click="emit('closeMobile')">
            <AppLogo class="size-8" />
            <div class="leading-tight">
              <div class="flex items-center gap-1.5">
                <span class="font-semibold text-[15px] tracking-tight text-neutral-950 dark:text-white">vibeasik</span>
                <span class="rounded-md px-1.5 py-px text-xs font-medium bg-neutral-900/5 text-neutral-500 dark:bg-white/5 dark:text-slate-400">{{ planLabel }}</span>
              </div>
              <AppVersionBadge />
            </div>
          </NuxtLink>
          <button
            type="button"
            class="size-11 -mr-2 flex items-center justify-center text-neutral-500 hover:text-neutral-800 dark:text-slate-400 dark:hover:text-white rounded-lg focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
            aria-label="Tutup menu"
            @click="emit('closeMobile')"
          >
            <UIcon name="i-lucide-x" class="size-5" />
          </button>
        </div>
        <div class="flex-1 overflow-y-auto px-3 py-4">
          <AppSidebarNav mobile @navigate="emit('closeMobile')" />
        </div>
        <div class="px-4 py-3 border-t border-neutral-200 dark:border-white/[0.07] flex items-center gap-2 text-xs">
          <span class="size-2 rounded-full" :class="apiConnected ? 'bg-emerald-500' : 'bg-rose-500'" />
          <span class="text-neutral-600 dark:text-slate-300">{{ apiConnected ? 'Engine terhubung' : 'Engine offline' }}</span>
          <span class="ml-auto font-mono text-neutral-400 dark:text-slate-500">{{ engineHost }}</span>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
