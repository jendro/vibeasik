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
const publicRoutes = ['/download', '/login', '/register']
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
    <!-- 1. PUBLIC MARKETING & ONBOARDING LAYOUT (Home, Download, Login, Register) -->
    <template v-if="isPublicRoute">
      <div
        class="min-h-screen flex flex-col font-sans transition-colors duration-200"
        :class="route.path === '/' ? 'bg-[#020c18] text-slate-100' : 'bg-neutral-50 dark:bg-neutral-950 text-neutral-900 dark:text-neutral-100'"
      >
        <PublicNavbar />
        <main class="flex-1 w-full">
          <slot>
            <NuxtPage />
          </slot>
        </main>
        <PublicFooter />
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
