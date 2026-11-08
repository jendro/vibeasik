<script setup lang="ts">
import type { Project, ProjectBaseKnowledge } from '~/types'

const props = defineProps<{
  project: Project
  cachedData?: ProjectBaseKnowledge | null
  loadingData?: boolean
}>()

const emit = defineEmits<{
  switchTab: [tab: string]
  load: []
  refresh: []
}>()

const toast = useToast()
const { getProjectBaseKnowledge, refreshProjectBaseKnowledge } = useApi()

const internalLoading = ref(false)
const internalRefreshing = ref(false)
const internalBaseKnowledge = ref<ProjectBaseKnowledge | null>(null)

const baseKnowledge = computed(() => props.cachedData ?? internalBaseKnowledge.value)
const loading = computed(() => props.loadingData !== undefined ? props.loadingData : internalLoading.value)
const refreshing = computed(() => internalRefreshing.value)

const searchRouteQuery = ref('')
const selectedDomain = ref<string>('all')
const isTreeOpen = ref(false)
const isEnvKeysOpen = ref(false)

const handleRefresh = async () => {
  if (props.cachedData !== undefined) {
    emit('refresh')
    return
  }
  await loadBaseKnowledge(true)
}

const loadBaseKnowledge = async (forceRefresh = false) => {
  if (forceRefresh) {
    internalRefreshing.value = true
  } else {
    internalLoading.value = true
  }

  try {
    if (forceRefresh) {
      internalBaseKnowledge.value = await refreshProjectBaseKnowledge(props.project.id)
      toast.add({
        title: 'Base Knowledge Diperbarui',
        description: 'Informasi aplikasi, file, dan rute proyek berhasil disinkronkan ulang.',
        icon: 'i-lucide-check-circle',
        color: 'success'
      })
    } else {
      internalBaseKnowledge.value = await getProjectBaseKnowledge(props.project.id)
    }
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Gagal memuat base knowledge proyek'
    toast.add({
      title: 'Gagal Memuat Base Knowledge',
      description: msg,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    internalLoading.value = false
    internalRefreshing.value = false
  }
}

onMounted(() => {
  // If parent already has cached data in variable, do not fetch again!
  if (!props.cachedData && !internalBaseKnowledge.value) {
    if (props.loadingData === undefined) {
      loadBaseKnowledge(false)
    } else {
      emit('load')
    }
  }
})

const filteredRoutes = computed(() => {
  if (!baseKnowledge.value?.routers_scope?.routes) return []
  return baseKnowledge.value.routers_scope.routes.filter((r) => {
    const matchesDomain = selectedDomain.value === 'all' || r.domain === selectedDomain.value
    const q = searchRouteQuery.value.trim().toLowerCase()
    if (!q) return matchesDomain

    const matchesQuery
      = r.path.toLowerCase().includes(q)
        || r.method.toLowerCase().includes(q)
        || (r.handler && r.handler.toLowerCase().includes(q))
        || (r.description && r.description.toLowerCase().includes(q))
        || (r.component_name && r.component_name.toLowerCase().includes(q))

    return matchesDomain && matchesQuery
  })
})

const availableDomains = computed(() => {
  if (!baseKnowledge.value?.routers_scope?.domain_groups) return []
  return Object.keys(baseKnowledge.value.routers_scope.domain_groups)
})

const copyMarkdown = async () => {
  if (!baseKnowledge.value?.summary_markdown) return
  try {
    await navigator.clipboard.writeText(baseKnowledge.value.summary_markdown)
    toast.add({
      title: 'Base Knowledge Disalin',
      description: 'Format Markdown lengkap berhasil disalin ke clipboard.',
      icon: 'i-lucide-clipboard-check',
      color: 'success'
    })
  } catch {
    toast.add({
      title: 'Gagal Menyalin',
      description: 'Silakan salin teks secara manual.',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  }
}

const { getMethodColorClass } = useFormatters()
</script>

<template>
  <div class="space-y-6">
    <!-- Top Action Bar -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-5 rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/60 shadow-xs">
      <div class="space-y-1">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-book-open-check" class="size-5 text-indigo-500" />
          <h2 class="text-base font-bold text-slate-900 dark:text-slate-100">
            Rute, File & Lingkungan
          </h2>
          <span
            v-if="baseKnowledge"
            class="text-xs px-2 py-0.5 rounded-full font-mono font-medium bg-indigo-50 dark:bg-indigo-950 text-indigo-600 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-800"
          >
            {{ baseKnowledge.routers_scope?.backend_routes_count || 0 }} API • {{ baseKnowledge.routers_scope?.frontend_pages_count || 0 }} Halaman • ~{{ baseKnowledge.files_scope?.total_files || 0 }} File
          </span>
        </div>
        <p class="text-xs text-slate-500 dark:text-slate-400">
          Base knowledge teknis: katalog rute seluruh aplikasi, struktur file, dan variabel lingkungan. Data ini juga menjadi bukti untuk analisis AI & planner TODO.
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <UButton
          size="xs"
          color="neutral"
          variant="outline"
          icon="i-lucide-clipboard"
          :disabled="!baseKnowledge"
          @click="copyMarkdown"
        >
          Salin Markdown
        </UButton>
        <UButton
          size="xs"
          color="neutral"
          variant="outline"
          icon="i-lucide-refresh-cw"
          :loading="refreshing"
          @click="handleRefresh"
        >
          Segarkan Data
        </UButton>
        <UButton
          size="xs"
          color="primary"
          variant="solid"
          icon="i-lucide-message-square-code"
          @click="emit('switchTab', 'ai_chat')"
        >
          Diskusi Ide & Chat TODO
        </UButton>
      </div>
    </div>

    <!-- Loading Skeleton -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <div v-for="i in 4" :key="i" class="h-32 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-100 dark:bg-slate-800 animate-pulse" />
    </div>

    <div v-else-if="baseKnowledge" class="space-y-6">
      <!-- Detail Section 2: Katalog Rute & Fungsi Router (Searchable) -->
      <div class="p-5 rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/70 shadow-xs space-y-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-route" class="size-4.5 text-indigo-500" />
            <h3 class="text-sm font-bold text-slate-900 dark:text-slate-100">
              Katalog Router & Fungsi Endpoint ({{ filteredRoutes.length }} dari {{ baseKnowledge.routers_scope?.total_routes || 0 }})
            </h3>
          </div>

          <!-- Route Filters -->
          <div class="flex flex-wrap items-center gap-2">
            <UInput
              v-model="searchRouteQuery"
              icon="i-lucide-search"
              placeholder="Cari rute, path, handler..."
              size="xs"
              class="w-48 sm:w-60"
            />
            <select
              v-model="selectedDomain"
              class="text-xs rounded-lg px-2.5 py-1.5 border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-200 focus:outline-hidden"
            >
              <option value="all">
                Semua Domain
              </option>
              <option v-for="dom in availableDomains" :key="dom" :value="dom">
                {{ dom }}
              </option>
            </select>
          </div>
        </div>

        <!-- Table of routes with progressive disclosure -->
        <div class="overflow-x-auto rounded-xl border border-slate-200 dark:border-slate-800">
          <table class="w-full text-left text-xs">
            <thead class="bg-slate-50 dark:bg-slate-800/80 text-slate-500 font-semibold border-b border-slate-200 dark:border-slate-800">
              <tr>
                <th class="py-2.5 px-3">
                  Metode
                </th>
                <th class="py-2.5 px-3">
                  Path / Rute
                </th>
                <th class="py-2.5 px-3">
                  Komponen
                </th>
                <th class="py-2.5 px-3">
                  Domain
                </th>
                <th class="py-2.5 px-3">
                  Fungsi / Deskripsi
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100 dark:divide-slate-800/80 font-mono text-xs">
              <tr
                v-for="(r, idx) in filteredRoutes.slice(0, 30)"
                :key="idx"
                class="hover:bg-slate-50/60 dark:hover:bg-slate-800/40 transition-colors"
              >
                <td class="py-2 px-3 whitespace-nowrap">
                  <span
                    class="px-2 py-0.5 rounded text-xs font-bold border"
                    :class="getMethodColorClass(r.method)"
                  >
                    {{ r.method }}
                  </span>
                </td>
                <td class="py-2 px-3 font-semibold text-slate-900 dark:text-slate-100 whitespace-nowrap">
                  {{ r.path }}
                </td>
                <td class="py-2 px-3 text-slate-500 font-sans whitespace-nowrap">
                  {{ r.component_name }}
                </td>
                <td class="py-2 px-3 whitespace-nowrap font-sans">
                  <span class="px-1.5 py-0.5 rounded bg-slate-100 dark:bg-slate-800 text-xs text-slate-600 dark:text-slate-300">
                    {{ r.domain || 'Umum' }}
                  </span>
                </td>
                <td class="py-2 px-3 font-sans text-slate-600 dark:text-slate-400 max-w-xs truncate">
                  {{ r.description || '-' }}
                </td>
              </tr>
              <tr v-if="filteredRoutes.length > 30">
                <td colspan="5" class="py-2.5 px-3 text-center text-slate-400 font-sans text-xs bg-slate-50/50 dark:bg-slate-900/40">
                  Menampilkan 30 rute teratas. Gunakan pencarian untuk menyaring {{ filteredRoutes.length - 30 }} rute lainnya.
                </td>
              </tr>
              <tr v-if="filteredRoutes.length === 0">
                <td colspan="5" class="py-6 px-3 text-center text-slate-400 font-sans text-xs">
                  Tidak ada rute yang cocok dengan filter.
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Detail Section 3: File Scope & Struktur Direktori (Progressive Disclosure) -->
      <div class="p-5 rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/70 shadow-xs space-y-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-folder-git-2" class="size-4.5 text-emerald-500" />
            <h3 class="text-sm font-bold text-slate-900 dark:text-slate-100">
              File Scope & Pohon Direktori
            </h3>
          </div>
          <UButton
            size="xs"
            color="neutral"
            variant="ghost"
            :icon="isTreeOpen ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
            @click="isTreeOpen = !isTreeOpen"
          >
            {{ isTreeOpen ? 'Sembunyikan Pohon' : 'Tampilkan Pohon Direktori' }}
          </UButton>
        </div>

        <!-- Key Config Files Chips -->
        <div v-if="baseKnowledge.files_scope?.config_files?.length" class="space-y-1.5">
          <span class="text-xs font-semibold text-slate-500 uppercase tracking-wider">
            File Konfigurasi Utama yang Terdeteksi:
          </span>
          <div class="flex flex-wrap gap-1.5">
            <span
              v-for="cfg in baseKnowledge.files_scope.config_files"
              :key="cfg"
              class="px-2 py-0.5 rounded text-xs font-mono bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-200/80 dark:border-slate-700"
            >
              {{ cfg }}
            </span>
          </div>
        </div>

        <!-- Collapsible Tree -->
        <div v-if="isTreeOpen" class="p-3 rounded-xl bg-slate-950 text-slate-200 font-mono text-xs overflow-x-auto leading-relaxed border border-slate-800">
          <pre>{{ baseKnowledge.files_scope.directory_tree }}</pre>
        </div>
      </div>

      <!-- Detail Section 4: Environment & Sanitized Keys (Progressive Disclosure) -->
      <div class="p-5 rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/70 shadow-xs space-y-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-key-round" class="size-4.5 text-amber-500" />
            <h3 class="text-sm font-bold text-slate-900 dark:text-slate-100">
              Lingkungan & Keamanan Variabel (.env)
            </h3>
          </div>
          <UButton
            v-if="baseKnowledge.environment_scope?.detected_env_keys?.length"
            size="xs"
            color="neutral"
            variant="ghost"
            :icon="isEnvKeysOpen ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
            @click="isEnvKeysOpen = !isEnvKeysOpen"
          >
            {{ isEnvKeysOpen ? 'Sembunyikan Kunci' : `Lihat ${baseKnowledge.environment_scope.detected_env_keys.length} Kunci Terdeteksi` }}
          </UButton>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-3 text-xs">
          <div class="p-3 rounded-lg border border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-950/30">
            <span class="text-xs uppercase font-semibold text-slate-400">Database Engine</span>
            <p class="font-bold text-slate-800 dark:text-slate-200 mt-0.5">
              {{ baseKnowledge.environment_scope?.database_type || 'SQLite' }}
            </p>
          </div>
          <div class="p-3 rounded-lg border border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-950/30">
            <span class="text-xs uppercase font-semibold text-slate-400">Metode Autentikasi</span>
            <p class="font-bold text-slate-800 dark:text-slate-200 mt-0.5">
              {{ baseKnowledge.environment_scope?.auth_methods?.join(', ') || 'Session / API Key' }}
            </p>
          </div>
          <div class="p-3 rounded-lg border border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-950/30">
            <span class="text-xs uppercase font-semibold text-slate-400">Layanan Pihak Ke-3</span>
            <p class="font-bold text-slate-800 dark:text-slate-200 mt-0.5 truncate">
              {{ baseKnowledge.environment_scope?.third_party_services?.join(', ') || 'None' }}
            </p>
          </div>
        </div>

        <!-- Collapsible Env Keys -->
        <div v-if="isEnvKeysOpen && baseKnowledge.environment_scope?.detected_env_keys?.length" class="space-y-2 pt-2 border-t border-slate-100 dark:border-slate-800">
          <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
            <div
              v-for="k in baseKnowledge.environment_scope.detected_env_keys"
              :key="k.key"
              class="p-2.5 rounded-lg border border-slate-200/80 dark:border-slate-800 bg-white dark:bg-slate-900 text-xs"
            >
              <div class="flex items-center justify-between gap-1">
                <span class="font-mono font-bold text-slate-800 dark:text-slate-200">{{ k.key }}</span>
                <span v-if="k.is_secret" class="text-xs px-1 py-0.5 rounded font-mono bg-rose-50 text-rose-600 dark:bg-rose-950 dark:text-rose-400">
                  Secret
                </span>
              </div>
              <p class="text-xs text-slate-500 mt-1">
                {{ k.purpose || 'Variabel lingkungan aplikasi' }}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
