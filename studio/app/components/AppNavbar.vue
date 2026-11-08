<script setup lang="ts">
import { useLLMState } from '~/composables/useLLMState'
import { useActiveProject } from '~/composables/useActiveProject'

defineProps<{
  apiConnected: boolean
  checking: boolean
}>()

const emit = defineEmits<{
  (e: 'checkApi'): void
  (e: 'toggleMobileSidebar'): void
  (e: 'toggleDesktopSidebar'): void
}>()

const route = useRoute()
const router = useRouter()
const { apiBase } = useApi()
const { user, isAuthenticated, logout } = useAuth()
const {
  projects,
  activeProjectId,
  activeProject,
  fetchProjects,
  selectActiveProject
} = useActiveProject()
const {
  availableModels,
  selectedModel,
  isSwitchingModel,
  isPinging,
  handleSelectModel,
  modelLabel,
  triggerPing,
  refreshLLMState
} = useLLMState()

onMounted(() => {
  refreshLLMState()
  fetchProjects()
})

const handleSelectActiveProject = (projectIdStr: string) => {
  const pId = projectIdStr ? Number(projectIdStr) : null
  selectActiveProject(pId)
  if (route.path.startsWith('/projects/') && pId) {
    router.push(`/projects/${pId}`)
  }
}

const pageTitle = computed(() => {
  if (route.path === '/cockpit' || route.path === '/') return 'Katalog Proyek & Cockpit'
  if (route.path === '/todos') return 'Perencanaan & Agenda Bertingkat'
  if (route.path === '/audit-report') return 'Laporan Audit Diagnostik & Code Smells'
  if (route.path === '/runtime-test') return 'Uji Runtime'
  if (route.path === '/audit-learning') return 'Pembelajaran Audit (Vonis Keliru & Benar)'
  if (route.path === '/delivery') return 'Delivery, Domain & Hosting'
  if (route.path === '/ai-hub') return 'AI Agent Hub & Templates'
  if (route.path === '/scan') return 'Pindai & Impor Repository'
  if (route.path === '/adapters') return 'Katalog Adapter Rute'
  if (route.path === '/mcp') return 'AI & MCP Protocol Hub'
  if (route.path === '/ports') return 'Monitor Port Lokal'
  if (route.path === '/queue') return 'Antrean Tugas & Pipeline'
  if (route.path === '/settings') return 'Pengaturan LLM & System One'
  if (route.path === '/design-library') return 'Galeri Desain (Referensi Tampilan untuk AI)'
  if (route.path === '/design-lessons') return 'Pelajaran Desain (Before / After)'
  if (route.path === '/screenshots') return 'Screenshot & Galeri Visual Proyek'
  if (route.path.startsWith('/projects/')) return 'Detail Proyek'
  return 'Workspace & Cockpit'
})
</script>

<template>
  <header class="sticky top-0 z-30 h-16 border-b border-neutral-200 dark:border-white/10 bg-white/90 dark:bg-[#020c18]/85 backdrop-blur-md transition-colors duration-200">
    <div class="h-full px-4 sm:px-6 flex items-center justify-between gap-4">
      <!-- Left side: Toggle & Brand / Page Title -->
      <div class="flex items-center gap-3 sm:gap-4 min-w-0">
        <!-- Mobile Sidebar Toggle -->
        <button
          type="button"
          class="md:hidden p-2 -ml-1 text-neutral-600 dark:text-neutral-300 hover:text-neutral-900 dark:hover:text-white rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
          aria-label="Toggle navigation menu"
          @click="emit('toggleMobileSidebar')"
        >
          <UIcon name="i-lucide-menu" class="w-5 h-5" />
        </button>

        <!-- Desktop Sidebar Toggle -->
        <button
          type="button"
          class="hidden md:flex p-1.5 text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-200 rounded-md hover:bg-neutral-100 dark:hover:bg-neutral-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500 transition-colors"
          title="Toggle sidebar collapse"
          aria-label="Toggle sidebar collapse"
          @click="emit('toggleDesktopSidebar')"
        >
          <UIcon name="i-lucide-panel-left" class="w-5 h-5" />
        </button>

        <!-- Brand / Current View Title: Project Name on Header -->
        <div class="flex items-center gap-2.5 min-w-0">
          <NuxtLink to="/cockpit" class="flex items-center gap-2 group md:hidden">
            <div class="w-8 h-8 rounded-lg bg-sky-600 flex items-center justify-center text-white shrink-0 shadow-xs">
              <UIcon name="i-lucide-layers" class="w-4 h-4" />
            </div>
          </NuxtLink>
          <div class="flex flex-col min-w-0">
            <div class="flex items-center gap-2 min-w-0">
              <h1 class="text-sm sm:text-base font-bold text-neutral-900 dark:text-neutral-100 truncate tracking-tight">
                {{ activeProject ? activeProject.name : 'Semua Proyek' }}
              </h1>
              <span
                v-if="activeProject"
                class="hidden sm:inline-flex items-center px-1.5 py-0.5 rounded text-xs font-semibold bg-sky-100 text-sky-800 dark:bg-sky-950/60 dark:text-sky-300 border border-sky-200 dark:border-sky-800/80 shrink-0 font-sans"
              >
                Aktif
              </span>
            </div>
            <span class="text-xs text-neutral-500 dark:text-neutral-400 hidden sm:inline font-sans truncate">
              {{ pageTitle }}
            </span>
          </div>
        </div>
      </div>

      <!-- Right side: Actions & Status -->
      <div class="flex items-center gap-2 sm:gap-2.5 shrink-0">
        <!-- Active Project Selector (Di sebelah kiri model AI selector) -->
        <div class="flex items-center gap-1.5">
          <div class="relative flex items-center">
            <span class="absolute left-2.5 pointer-events-none text-neutral-400 dark:text-neutral-500 flex items-center">
              <UIcon name="i-lucide-folder-git-2" class="w-3.5 h-3.5 text-blue-600 dark:text-blue-400" />
            </span>
            <select
              :value="activeProjectId ?? ''"
              class="h-7 appearance-none pl-8 pr-7 py-0 text-xs font-mono font-medium rounded-lg border border-neutral-200 dark:border-neutral-700 bg-neutral-50/80 dark:bg-neutral-800/80 text-neutral-800 dark:text-neutral-200 hover:border-neutral-300 dark:hover:border-neutral-600 focus:outline-none focus:ring-1 focus:ring-blue-500 cursor-pointer transition-colors max-w-[125px] sm:max-w-[180px] truncate"
              title="Proyek Aktif (Klik untuk memilih atau memfilter proyek)"
              aria-label="Pilih proyek aktif"
              @change="(e) => handleSelectActiveProject((e.target as HTMLSelectElement).value)"
            >
              <option value="">
                Semua Proyek
              </option>
              <option
                v-for="p in projects"
                :key="p.id"
                :value="p.id"
              >
                {{ p.name }}
              </option>
            </select>
            <span class="absolute right-2 pointer-events-none text-neutral-400 dark:text-neutral-500 flex items-center">
              <UIcon name="i-lucide-chevron-down" class="w-3.5 h-3.5" />
            </span>
          </div>

          <!-- Quick link to active project detail if selected -->
          <NuxtLink
            v-if="activeProjectId"
            :to="`/projects/${activeProjectId}`"
            class="h-7 w-7 flex items-center justify-center shrink-0 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-neutral-50/80 dark:bg-neutral-800/80 text-neutral-500 hover:text-blue-600 dark:hover:text-blue-400 hover:border-blue-300 dark:hover:border-blue-800 transition-colors"
            title="Buka detail proyek aktif"
            aria-label="Buka detail proyek aktif"
          >
            <UIcon name="i-lucide-external-link" class="w-3.5 h-3.5" />
          </NuxtLink>
        </div>

        <!-- Active LLM Model Selector Combobox (Disamping menu Go API) -->
        <div v-if="availableModels.length > 0" class="flex items-center gap-1.5">
          <div class="relative flex items-center">
            <span class="absolute left-2.5 pointer-events-none text-neutral-400 dark:text-neutral-500 flex items-center">
              <UIcon name="i-lucide-cpu" class="w-3.5 h-3.5 text-sky-600 dark:text-sky-400" />
            </span>
            <select
              :value="selectedModel"
              :disabled="isSwitchingModel"
              class="h-7 appearance-none pl-8 pr-7 py-0 text-xs font-mono font-medium rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50/80 dark:bg-slate-900/80 text-neutral-800 dark:text-neutral-200 hover:border-neutral-300 dark:hover:border-neutral-600 focus:outline-none focus:ring-1 focus:ring-sky-500 cursor-pointer disabled:opacity-60 transition-colors max-w-[130px] sm:max-w-[185px] truncate"
              :title="`Model LLM Biasa aktif: ${selectedModel} (klik untuk memilih model)`"
              aria-label="Pilih model LLM Biasa aktif"
              @change="(e) => handleSelectModel((e.target as HTMLSelectElement).value)"
            >
              <option
                v-for="model in availableModels"
                :key="model"
                :value="model"
                :title="model"
              >
                {{ modelLabel(model) }}
              </option>
            </select>
            <span class="absolute right-2 pointer-events-none text-neutral-400 dark:text-neutral-500 flex items-center">
              <UIcon
                :name="isSwitchingModel ? 'i-lucide-loader-2' : 'i-lucide-chevron-down'"
                class="w-3.5 h-3.5"
                :class="{ 'animate-spin': isSwitchingModel }"
              />
            </span>
          </div>

          <!-- Quick Ping Action Button (Sama tinggi dengan select model: h-7 w-7) -->
          <button
            type="button"
            class="h-7 w-7 flex items-center justify-center shrink-0 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50/80 dark:bg-slate-900/80 text-neutral-500 hover:text-sky-600 dark:hover:text-sky-400 hover:border-sky-300 dark:hover:border-sky-800 transition-colors cursor-pointer disabled:opacity-50"
            :disabled="isPinging || !selectedModel"
            :title="isPinging ? 'Sedang melakukan ping...' : `Ping model aktif (${modelLabel(selectedModel || '')})`"
            aria-label="Ping model LLM aktif"
            @click="triggerPing(undefined, selectedModel)"
          >
            <UIcon
              :name="isPinging ? 'i-lucide-loader-2' : 'i-lucide-activity'"
              class="w-3.5 h-3.5"
              :class="{ 'animate-spin text-sky-600': isPinging }"
            />
          </button>
        </div>
        <!-- API Connection Status Badge -->
        <button
          type="button"
          class="h-7 inline-flex items-center justify-center shrink-0 gap-1.5 px-2.5 rounded-full text-xs border transition-colors select-none focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500 min-w-[76px] sm:min-w-[140px]"
          :class="apiConnected
            ? 'bg-sky-50 text-sky-700 border-sky-200 dark:bg-sky-950/40 dark:text-sky-300 dark:border-sky-800/60'
            : 'bg-rose-50 text-rose-700 border-rose-200 dark:bg-rose-950/40 dark:text-rose-400 dark:border-rose-800/60'"
          :title="`Click to refresh. API ${apiConnected ? 'Active' : 'Offline'} on ${apiBase}`"
          @click="emit('checkApi')"
        >
          <span
            class="w-2 h-2 rounded-full shrink-0"
            :class="[
              apiConnected ? 'bg-sky-500' : 'bg-rose-500',
              checking ? 'animate-ping' : ''
            ]"
          />
          <span class="font-mono font-medium hidden sm:inline">
            {{ checking ? 'Checking...' : apiConnected ? 'Go API :8765' : 'Go API Offline' }}
          </span>
          <span class="font-mono font-medium sm:hidden">
            {{ checking ? '...' : apiConnected ? ':8765' : 'Offline' }}
          </span>
        </button>
        <UColorModeButton />

        <!-- Link to Public Website -->
        <UButton
          to="https://vibeasik.com"
          target="_blank"
          icon="i-lucide-globe"
          color="neutral"
          variant="ghost"
          size="sm"
          class="hidden lg:inline-flex"
          title="Buka vibeasik.com"
        >
          vibeasik.com
        </UButton>

        <!-- Quick Import Action -->
        <UButton
          to="/scan"
          icon="i-lucide-plus"
          color="primary"
          variant="solid"
          size="sm"
          class="hidden sm:inline-flex"
        >
          Import
        </UButton>

        <!-- User Profile & Logout -->
        <div v-if="isAuthenticated" class="flex items-center gap-2 pl-2 border-l border-neutral-200 dark:border-neutral-800">
          <div class="hidden xl:flex flex-col text-right">
            <NuxtLink to="/account" class="text-xs font-semibold text-neutral-800 dark:text-neutral-200 leading-tight truncate max-w-[120px] hover:underline">{{ user?.name || 'Vibecoder' }}</NuxtLink>
            <span class="text-xs text-neutral-400 font-mono leading-tight">{{ user?.role }}</span>
          </div>
          <button
            type="button"
            class="p-1.5 text-neutral-500 hover:text-rose-600 dark:text-neutral-400 dark:hover:text-rose-400 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-rose-500 cursor-pointer"
            title="Keluar dari akun vibeasik"
            aria-label="Logout"
            @click="logout"
          >
            <UIcon name="i-lucide-log-out" class="w-4 h-4" />
          </button>
        </div>
        <UButton v-else to="/account" size="xs" variant="soft" icon="i-lucide-log-in" class="ml-2">
          Masuk akun
        </UButton>
      </div>
    </div>
  </header>
</template>
