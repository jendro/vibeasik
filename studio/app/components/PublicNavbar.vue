<script setup lang="ts">
const { isAuthenticated, user, logout } = useAuth()
const route = useRoute()
const mobileMenuOpen = ref(false)

const isHomePage = computed(() => route.path === '/')
const isScrolled = ref(false)

const onScroll = () => {
  if (typeof window !== 'undefined') {
    isScrolled.value = window.scrollY > 20
  }
}

onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true })
  onScroll()
})

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('scroll', onScroll)
  }
})

const navLinks = [
  { label: 'Beranda', to: '/' },
  { label: 'Featured Apps', to: '/#featured-apps' },
  { label: 'Kenapa vibeasik?', to: '/#features' },
  { label: 'Arsitektur & Keamanan', to: '/#security' },
  { label: 'MCP Tools (20)', to: '/#mcp' },
  { label: 'Download App', to: '/download' }
]

watch(() => route.fullPath, () => {
  mobileMenuOpen.value = false
})
</script>

<template>
  <header
    class="w-full transition-all duration-300"
    :class="[
      isHomePage
        ? (isScrolled
          ? 'fixed top-0 left-0 z-50 bg-[#020c18]/85 backdrop-blur-md border-b border-white/10 text-white shadow-lg shadow-black/30'
          : 'fixed top-0 left-0 z-50 bg-transparent border-b border-white/10 text-white')
        : 'sticky top-0 z-40 border-b border-neutral-200/80 dark:border-neutral-800/80 bg-white/90 dark:bg-neutral-950/90 backdrop-blur-md text-neutral-900 dark:text-white'
    ]"
  >
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between gap-4">
      <!-- Left: Brand Logo -->
      <NuxtLink to="/" class="flex items-center gap-2.5 group">
        <div
          class="w-9 h-9 rounded-xl flex items-center justify-center transition-transform group-hover:scale-105"
          :class="isHomePage ? 'bg-sky-500/20 text-sky-300 border border-sky-400/30 shadow-xs' : 'bg-sky-600 text-white shadow-xs'"
        >
          <UIcon name="i-lucide-sparkles" class="w-5 h-5" />
        </div>
        <div class="flex flex-col">
          <div class="flex items-center gap-1.5">
            <span
              class="font-extrabold text-base tracking-tight"
              :class="isHomePage ? 'text-white' : 'text-neutral-900 dark:text-white'"
            >
              vibeasik
            </span>
            <span
              class="text-[10px] font-mono font-semibold px-1.5 py-0.5 rounded-md"
              :class="isHomePage ? 'bg-sky-950/60 text-sky-300 border border-sky-500/30' : 'bg-sky-50 dark:bg-sky-950/60 text-sky-600 dark:text-sky-400 border border-sky-200/60 dark:border-sky-800/60'"
            >
              v1.0
            </span>
          </div>
          <span
            class="text-[10px] -mt-0.5 tracking-wide"
            :class="isHomePage ? 'text-slate-300/80' : 'text-neutral-500 dark:text-neutral-400'"
          >
            AI Code & MCP Studio
          </span>
        </div>
      </NuxtLink>

      <!-- Center: Navigation Links (Desktop) -->
      <nav class="hidden md:flex items-center gap-6 text-sm font-medium">
        <NuxtLink
          v-for="link in navLinks"
          :key="link.to"
          :to="link.to"
          class="transition-colors"
          :class="[
            isHomePage
              ? ((route.hash && link.to.endsWith(route.hash)) || (route.path === link.to && !route.hash && link.to === '/')
                ? 'text-sky-300 font-semibold'
                : 'text-slate-200/90 hover:text-sky-200')
              : (route.path === link.to
                ? 'text-sky-600 dark:text-sky-400 font-semibold'
                : 'text-neutral-600 dark:text-neutral-300 hover:text-sky-600 dark:hover:text-sky-400')
          ]"
        >
          {{ link.label }}
        </NuxtLink>
      </nav>

      <!-- Right: Auth Action & Theme Switcher -->
      <div class="flex items-center gap-3">
        <UColorModeButton />

        <!-- If Authenticated -->
        <template v-if="isAuthenticated">
          <NuxtLink
            to="/cockpit"
            class="hidden sm:inline-flex items-center gap-2 px-4 py-2 rounded-lg text-xs font-semibold transition-all"
            :class="isHomePage ? 'bg-sky-600 hover:bg-sky-500 text-white shadow-md shadow-sky-950/40' : 'bg-primary-600 hover:bg-primary-500 text-white'"
          >
            <UIcon name="i-lucide-layout-grid" class="w-4 h-4" />
            <span>Buka Cockpit</span>
          </NuxtLink>

          <UButton
            icon="i-lucide-log-out"
            color="neutral"
            variant="ghost"
            size="sm"
            title="Keluar dari akun"
            @click="logout"
          />
        </template>

        <!-- If Public Guest -->
        <template v-else>
          <NuxtLink
            to="/login"
            class="px-3.5 py-1.5 rounded-lg text-xs font-medium transition-colors"
            :class="isHomePage ? 'text-white/90 hover:text-white hover:bg-white/10' : 'text-neutral-700 dark:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
          >
            Masuk
          </NuxtLink>

          <NuxtLink
            to="/register"
            class="hidden sm:inline-flex items-center justify-center px-4 py-2 rounded-lg text-xs font-semibold transition-all shadow-xs"
            :class="isHomePage ? 'bg-sky-600 hover:bg-sky-500 text-white shadow-md shadow-sky-950/40' : 'bg-neutral-900 hover:bg-neutral-800 text-white dark:bg-white dark:hover:bg-neutral-100 dark:text-neutral-900'"
          >
            Mulai Gratis
          </NuxtLink>
        </template>

        <!-- Mobile Menu Hamburger -->
        <button
          type="button"
          class="md:hidden p-2 rounded-lg focus:outline-none transition-colors"
          :class="isHomePage ? 'text-white/90 hover:text-white hover:bg-white/10' : 'text-neutral-600 dark:text-neutral-300 hover:text-neutral-900 dark:hover:text-white'"
          aria-label="Buka navigasi mobile"
          @click="mobileMenuOpen = !mobileMenuOpen"
        >
          <UIcon :name="mobileMenuOpen ? 'i-lucide-x' : 'i-lucide-menu'" class="w-6 h-6" />
        </button>
      </div>
    </div>

    <!-- Mobile Navigation Drawer -->
    <div
      v-if="mobileMenuOpen"
      class="md:hidden px-4 pt-2 pb-4 space-y-3"
      :class="isHomePage ? 'border-b border-white/10 bg-[#020c18]/95 backdrop-blur-xl text-white' : 'border-b border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-950 text-neutral-900 dark:text-white'"
    >
      <NuxtLink
        v-for="link in navLinks"
        :key="link.to"
        :to="link.to"
        class="block py-2 text-sm font-medium transition-colors"
        :class="isHomePage ? 'text-slate-200 hover:text-sky-300' : 'text-neutral-700 dark:text-neutral-200 hover:text-sky-600 dark:hover:text-sky-400'"
      >
        {{ link.label }}
      </NuxtLink>

      <div
        class="pt-2 border-t flex flex-col gap-2"
        :class="isHomePage ? 'border-white/10' : 'border-neutral-100 dark:border-neutral-800'"
      >
        <template v-if="isAuthenticated">
          <NuxtLink
            to="/cockpit"
            class="w-full inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg text-sm font-semibold bg-sky-600 text-white"
          >
            <UIcon name="i-lucide-layout-grid" class="w-4 h-4" />
            <span>Buka Cockpit Studio</span>
          </NuxtLink>
          <button
            type="button"
            class="w-full py-2 rounded-lg text-sm font-medium border border-white/20 text-white/90 hover:bg-white/10"
            @click="logout"
          >
            Keluar Akun ({{ user?.name }})
          </button>
        </template>
        <template v-else>
          <NuxtLink
            to="/register"
            class="w-full text-center py-2.5 rounded-lg text-sm font-semibold bg-sky-600 text-white"
          >
            Daftar Akun Baru
          </NuxtLink>
          <NuxtLink
            to="/login"
            class="w-full text-center py-2 rounded-lg text-sm font-medium border border-white/20 text-white hover:bg-white/10"
          >
            Masuk ke Studio
          </NuxtLink>
        </template>
      </div>
    </div>
  </header>
</template>
