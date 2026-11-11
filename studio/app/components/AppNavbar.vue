<script setup lang="ts">
// Navbar atas: breadcrumb (proyek aktif yang bisa dicari → halaman), model LLM aktif + ping,
// status engine, tema, dan menu akun. Judul halaman diambil dari menu sidebar agar selalu sama.
import { useLLMState } from '~/composables/useLLMState'
import { pageTitleFor } from '~/utils/sidebarMenuItems'

const props = defineProps<{
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
const { user, isAuthenticated, logout, load: loadAccount } = useAuth()
const { projects, activeProjectId, fetchProjects, selectActiveProject } = useActiveProject()
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
  loadAccount()
})

const page = computed(() => pageTitleFor(route.path))
const isProjectPage = computed(() => page.value.section === 'Proyek aktif')

// Pilihan proyek: 0 = semua proyek. Proyek non-aktif diberi keterangan status.
const STATUS_LABEL: Record<string, string> = { hold: 'ditahan', done: 'selesai', deprecated: 'usang' }
const projectItems = computed(() => [
  { label: 'Semua proyek', value: 0, icon: 'i-lucide-layout-grid' },
  ...[...projects.value]
    .sort((a, b) => Number(a.status && a.status !== 'active') - Number(b.status && b.status !== 'active') || a.name.localeCompare(b.name))
    .map(p => ({
      label: p.name,
      value: p.id,
      icon: 'i-lucide-folder-git-2',
      suffix: STATUS_LABEL[p.status || ''] || ''
    }))
])
const selectedProject = computed({
  get: () => activeProjectId.value ?? 0,
  set: (v: number) => {
    const id = v ? Number(v) : null
    selectActiveProject(id)
    if (route.path.startsWith('/projects/') && id) router.push(`/projects/${id}`)
  }
})

const modelItems = computed(() => availableModels.value.map(m => ({ label: modelLabel(m), value: m })))
const modelValue = computed({
  get: () => selectedModel.value,
  set: (v: string) => {
    if (v && v !== selectedModel.value) handleSelectModel(v)
  }
})

// Alamat engine yang sebenarnya (bukan ditulis tetap :8765).
const engineHost = useEngineHost()

const initials = computed(() => (user.value?.name || 'V').split(/\s+/).map(s => s[0]).slice(0, 2).join('').toUpperCase())
const accountMenu = computed(() => [
  [{ label: user.value?.name || 'Akun', type: 'label' as const }],
  [
    { label: 'Akun & paket', icon: 'i-lucide-user-round', to: '/account' },
    { label: 'Pengaturan', icon: 'i-lucide-sliders-horizontal', to: '/settings' },
    { label: 'vibeasik.com', icon: 'i-lucide-globe', to: 'https://vibeasik.com', target: '_blank' }
  ],
  [{ label: 'Keluar', icon: 'i-lucide-log-out', color: 'error' as const, onSelect: () => logout() }]
])
</script>

<template>
  <header class="sticky top-0 z-30 h-16 shrink-0 border-b border-neutral-200 dark:border-white/[0.07] bg-white/85 dark:bg-[#020c18]/80 backdrop-blur-md">
    <div class="h-full px-3 sm:px-6 flex items-center gap-3">
      <button
        type="button"
        class="md:hidden size-10 -ml-1 flex items-center justify-center text-neutral-600 dark:text-slate-300 rounded-lg hover:bg-neutral-100 dark:hover:bg-white/5 focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
        aria-label="Buka menu navigasi"
        @click="emit('toggleMobileSidebar')"
      >
        <UIcon name="i-lucide-menu" class="size-5" />
      </button>

      <!-- Breadcrumb: proyek aktif / halaman -->
      <nav class="flex-1 min-w-0 flex items-center gap-1.5 text-sm" aria-label="Lokasi">
        <USelectMenu
          v-if="isProjectPage"
          v-model="selectedProject"
          :items="projectItems"
          value-key="value"
          variant="ghost"
          color="neutral"
          size="sm"
          icon="i-lucide-folder-git-2"
          :search-input="{ placeholder: 'Cari proyek…' }"
          :content="{ align: 'start' }"
          :ui="{ base: 'font-medium max-w-[42vw] sm:max-w-[260px] text-neutral-700 dark:text-slate-200', content: 'min-w-72' }"
          aria-label="Pilih proyek aktif"
        >
          <template #item-trailing="{ item }">
            <span v-if="(item as any).suffix" class="text-xs text-neutral-400">{{ (item as any).suffix }}</span>
          </template>
        </USelectMenu>
        <span v-else class="hidden sm:inline text-neutral-500 dark:text-slate-400 px-1">{{ page.section }}</span>
        <UIcon name="i-lucide-chevron-right" class="size-4 shrink-0 text-neutral-300 dark:text-slate-600" :class="isProjectPage ? '' : 'hidden sm:inline'" />
        <h1 class="font-semibold text-neutral-950 dark:text-white truncate">
          {{ page.title }}
        </h1>
      </nav>

      <!-- Aksi kanan -->
      <div class="flex items-center gap-1.5 sm:gap-2 shrink-0">
        <div v-if="modelItems.length" class="hidden md:flex items-center rounded-lg border border-neutral-200 dark:border-white/10 bg-white dark:bg-white/[0.03]">
          <USelectMenu
            v-model="modelValue"
            :items="modelItems"
            value-key="value"
            variant="none"
            size="sm"
            :icon="isSwitchingModel ? 'i-lucide-loader-2' : 'i-lucide-cpu'"
            :disabled="isSwitchingModel"
            :search-input="{ placeholder: 'Cari model…' }"
            :content="{ align: 'end' }"
            :ui="{ base: 'max-w-[150px] xl:max-w-[200px] text-xs font-medium', content: 'min-w-64', leadingIcon: isSwitchingModel ? 'animate-spin' : 'text-sky-600 dark:text-sky-400' }"
            :title="`Model LLM aktif: ${selectedModel}`"
            aria-label="Pilih model LLM aktif"
          />
          <button
            type="button"
            class="size-8 flex items-center justify-center border-l border-neutral-200 dark:border-white/10 text-neutral-500 hover:text-sky-600 dark:hover:text-sky-400 disabled:opacity-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500 rounded-r-lg"
            :disabled="isPinging || !selectedModel"
            :title="isPinging ? 'Sedang ping…' : `Ping ${modelLabel(selectedModel || '')}`"
            aria-label="Ping model LLM aktif"
            @click="triggerPing(undefined, selectedModel)"
          >
            <UIcon :name="isPinging ? 'i-lucide-loader-2' : 'i-lucide-activity'" class="size-3.5" :class="{ 'animate-spin': isPinging }" />
          </button>
        </div>

        <button
          type="button"
          class="h-8 inline-flex items-center gap-2 rounded-lg px-2.5 text-xs border transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
          :class="props.apiConnected
            ? 'border-neutral-200 dark:border-white/10 text-neutral-600 dark:text-slate-300 hover:bg-neutral-50 dark:hover:bg-white/5'
            : 'border-rose-300 bg-rose-50 text-rose-700 dark:border-rose-800/60 dark:bg-rose-950/40 dark:text-rose-300'"
          :title="`Engine ${props.apiConnected ? 'terhubung' : 'tidak terjangkau'} di ${engineHost}. Klik untuk memeriksa ulang.`"
          :aria-label="`Status engine: ${props.apiConnected ? 'terhubung' : 'offline'}`"
          @click="emit('checkApi')"
        >
          <span class="size-2 rounded-full" :class="[props.apiConnected ? 'bg-emerald-500' : 'bg-rose-500', checking ? 'animate-pulse' : '']" />
          <span class="hidden lg:inline font-mono">{{ props.apiConnected ? engineHost : 'Engine offline' }}</span>
        </button>

        <NotificationBell />

        <UColorModeButton size="sm" class="hidden sm:inline-flex" />

        <UDropdownMenu v-if="isAuthenticated" :items="accountMenu" :content="{ align: 'end' }">
          <button
            type="button"
            class="size-8 rounded-full bg-neutral-900 text-white dark:bg-white dark:text-neutral-900 text-xs font-semibold flex items-center justify-center focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
            :title="`${user?.name || 'Akun'}${user?.role ? ` · ${user.role}` : ''}`"
            aria-label="Menu akun"
          >
            {{ initials }}
          </button>
        </UDropdownMenu>
        <UButton v-else to="/account" size="sm" color="neutral" variant="outline" icon="i-lucide-log-in" aria-label="Masuk akun vibeasik">
          <span class="hidden xl:inline">Masuk</span>
        </UButton>
      </div>
    </div>
  </header>
</template>
