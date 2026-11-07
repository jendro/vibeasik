<script setup lang="ts">
const apiConnected = ref(true)
const checking = ref(false)
const { apiBase } = useApi()

const checkApiStatus = async () => {
  checking.value = true
  try {
    const res = await $fetch<{ status: string }>(`${apiBase}/health`, { timeout: 2000 })
    apiConnected.value = res.status === 'ok'
  } catch {
    apiConnected.value = false
  } finally {
    checking.value = false
  }
}

// Sidebar responsive states
const sidebarCollapsed = ref(false)
const mobileSidebarOpen = ref(false)

const toggleDesktopSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

const toggleMobileSidebar = () => {
  mobileSidebarOpen.value = !mobileSidebarOpen.value
}

const closeMobileSidebar = () => {
  mobileSidebarOpen.value = false
}

// Route detection: Public website vs Admin Cockpit Workspace
const route = useRoute()
const publicRoutes = ['/setup']
const isPublicRoute = computed(() => publicRoutes.includes(route.path) || Boolean(route.meta.public))

const isFluid = computed(() => !isPublicRoute.value && (Boolean(route.meta.fluid) || route.path.startsWith('/projects/') || route.path.startsWith('/todos')))
// Halaman layar penuh (mis. Ruang Klien): tinggi = layar dikurangi navbar (h-16), sisakan tempat footer antrean (h-8).
const isFullHeight = computed(() => !isPublicRoute.value && Boolean(route.meta.fullHeight))

watch(() => route.path, () => {
  mobileSidebarOpen.value = false
})

onMounted(() => {
  checkApiStatus()
  const interval = setInterval(checkApiStatus, 15000)
  onUnmounted(() => clearInterval(interval))
})

useHead({
  title: 'vibeasik.com - Vibecoder Delivery, Domain, Hosting & AI Agent Hub',
  meta: [
    { name: 'description', content: 'Platform all-in-one untuk vibecoder memastikan app buatan aman, bebas port bentrok, dan mudah dimaintenance.' }
  ]
})
</script>

<template>
  <UApp :toaster="{ position: 'top-right' }">
    <!-- 1. PENYIAPAN AWAL (tanpa sidebar; wajib selesai sebelum Cockpit) -->
    <template v-if="isPublicRoute">
      <div class="min-h-screen flex flex-col font-sans bg-neutral-50 dark:bg-[#020c18] text-neutral-900 dark:text-neutral-100">
        <header class="h-16 px-4 sm:px-6 flex items-center justify-between border-b border-neutral-200 dark:border-neutral-800">
          <span class="flex items-center gap-2 font-bold text-lg tracking-tight">
            <svg viewBox="0 0 64 64" class="w-7 h-7" aria-hidden="true">
              <rect width="64" height="64" rx="14" fill="#0b1020" />
              <path d="M14 20l18 28 18-28" fill="none" stroke="#7c9cff" stroke-width="7" stroke-linecap="round" stroke-linejoin="round" />
              <circle cx="32" cy="48" r="4" fill="#5eead4" />
            </svg>
            vibeasik
          </span>
          <UButton to="https://vibeasik.com" target="_blank" icon="i-lucide-globe" color="neutral" variant="ghost" size="sm">
            vibeasik.com
          </UButton>
        </header>
        <main class="flex-1 w-full">
          <slot>
            <NuxtPage />
          </slot>
        </main>
      </div>
    </template>

    <!-- 2. ADMIN & COCKPIT WORKSPACE LAYOUT (Requires Authentication) -->
    <template v-else>
      <div class="min-h-screen bg-slate-50 dark:bg-[#020c18] text-slate-900 dark:text-slate-100 flex font-sans transition-colors duration-200">
        <!-- Left Sidebar (Desktop persistent + Mobile drawer) -->
        <AppSidebar
          :collapsed="sidebarCollapsed"
          :mobile-open="mobileSidebarOpen"
          :api-connected="apiConnected"
          @toggle-collapse="toggleDesktopSidebar"
          @close-mobile="closeMobileSidebar"
        />

        <!-- Main Shell Area (Navbar + Page Content + Footer) -->
        <div class="flex-1 flex flex-col min-w-0 min-h-screen">
          <!-- Top Navbar -->
          <AppNavbar
            :api-connected="apiConnected"
            :checking="checking"
            @check-api="checkApiStatus"
            @toggle-mobile-sidebar="toggleMobileSidebar"
            @toggle-desktop-sidebar="toggleDesktopSidebar"
          />

          <!-- Page Content -->
          <main :class="isFullHeight ? 'w-full h-[calc(100dvh-4rem)] pb-8 flex flex-col overflow-hidden' : isFluid ? 'flex-1 w-full p-0 flex flex-col pb-16 sm:pb-24' : 'flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 pt-6 sm:pt-8 pb-16 sm:pb-24'">
            <slot>
              <NuxtPage />
            </slot>
          </main>
        </div>
      </div>

      <!-- Mini Popup Ping LLM Notification (Pojok Kiri Bawah) -->
      <LLMPingMiniPopup />

      <!-- Fixed Footer Background Tasks & LLM Worker Monitor -->
      <QueueFooterBar :sidebar-collapsed="sidebarCollapsed" />
    </template>
  </UApp>
</template>
