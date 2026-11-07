<script setup lang="ts">
import type { Project, ProjectComponent, ComponentRoute } from '~/types'

const props = defineProps<{
  comp: ProjectComponent
  project: Project
}>()

const emit = defineEmits<{
  'routes-count-changed': [count: number]
}>()

const {
  getComponentRoutes,
  syncComponentRoutes,
  updateComponentRoute,
  deleteComponentRoute
} = useApi()

const toast = useToast()

// State
const routes = ref<ComponentRoute[]>([])
const loading = ref(false)
const syncing = ref(false)

// "Kirim ke LLM" is queued on the engine: the page stays usable while it waits and runs.
const enrichJob = useJobTarget('enrich_routes', () => ({ componentId: props.comp?.id }), async () => {
  await loadRoutes()
  feedbackSuccess.value = true
  const count = routes.value.length
  feedbackMessage.value = isFrontend.value
    ? `Berhasil! ${count} halaman antarmuka telah dianalisis & dikelompokkan ke modul fitur serta dilengkapi keterangan oleh LLM.`
    : `Berhasil! ${count} endpoint telah dianalisis & dikelompokkan ke domain bisnis serta dilengkapi keterangan oleh LLM.`
  setTimeout(() => {
    feedbackMessage.value = null
  }, 9000)
})
// The pipeline's route detection also rewrites routes: reload when it finishes.
const detectJob = useJobTarget('detect_routes', () => ({ componentId: props.comp?.id }), () => loadRoutes())
const enriching = computed(() => enrichJob.busy.value || detectJob.busy.value)
const enrichLabel = computed(() => enrichJob.label.value || detectJob.label.value)
const feedbackMessage = ref<string | null>(null)
const feedbackSuccess = ref(true)

const isFrontend = computed(() => {
  const r = props.comp?.role || ''
  return r.includes('frontend') || r === 'frontend_client' || r === 'frontend_admin' || r === 'frontend_mobile' || routes.value.some(x => (x.method || '').startsWith('PAGE'))
})

const parseTargetRequests = (targetReqs?: string): string[] => {
  if (!targetReqs) return []
  try {
    const parsed = JSON.parse(targetReqs)
    if (Array.isArray(parsed)) return parsed
  } catch {
    return targetReqs.split(/[,\n]/).map(s => s.trim()).filter(Boolean)
  }
  return []
}

const totalTargetRequestsCount = computed(() => {
  let count = 0
  routes.value.forEach((r) => {
    count += parseTargetRequests(r.target_requests).length
  })
  return count
})

const expandedRequests = ref<Record<number, boolean>>({})
const toggleExpandRequests = (routeId: number) => {
  expandedRequests.value[routeId] = !expandedRequests.value[routeId]
}

const getRequestMethodColor = (req: string): { bg: string, text: string, border: string } => {
  const m = req.trim().split(' ')[0]?.toUpperCase() || 'GET'
  if (m === 'GET') {
    return {
      bg: 'bg-sky-50 dark:bg-sky-950/60',
      text: 'text-sky-700 dark:text-sky-300',
      border: 'border-sky-200 dark:border-sky-800'
    }
  }
  if (m === 'POST') {
    return {
      bg: 'bg-blue-50 dark:bg-blue-950/60',
      text: 'text-blue-700 dark:text-blue-300',
      border: 'border-blue-200 dark:border-blue-800'
    }
  }
  if (m === 'PUT' || m === 'PATCH') {
    return {
      bg: 'bg-amber-50 dark:bg-amber-950/60',
      text: 'text-amber-700 dark:text-amber-300',
      border: 'border-amber-200 dark:border-amber-800'
    }
  }
  if (m === 'DELETE') {
    return {
      bg: 'bg-rose-50 dark:bg-rose-950/60',
      text: 'text-rose-700 dark:text-rose-300',
      border: 'border-rose-200 dark:border-rose-800'
    }
  }
  return {
    bg: 'bg-neutral-100 dark:bg-neutral-800',
    text: 'text-neutral-700 dark:text-neutral-300',
    border: 'border-neutral-200 dark:border-neutral-700'
  }
}

// Filter & View State
const searchQuery = ref('')
const selectedDomain = ref<string>('ALL')
const selectedMethod = ref<string>('ALL')
const viewMode = ref<'grouped' | 'table'>('grouped')
const copiedPath = ref<string | null>(null)

// Edit Modal State
const editModalOpen = ref(false)
const editingRoute = ref<ComponentRoute | null>(null)
const editForm = ref({
  domain: '',
  function_group: '',
  description: '',
  tags: '',
  target_requests: ''
})
const savingEdit = ref(false)

// Method Badge Colors
const { getMethodBadgeColor } = useFormatters()

// Domain Statistics
const domainList = computed(() => {
  const counts: Record<string, number> = {}
  routes.value.forEach((r) => {
    const d = r.domain || 'Umum'
    counts[d] = (counts[d] || 0) + 1
  })
  return Object.entries(counts)
    .map(([domain, count]) => ({ domain, count }))
    .sort((a, b) => b.count - a.count)
})

// Method Statistics
const methodList = computed(() => {
  const counts: Record<string, number> = {}
  routes.value.forEach((r) => {
    const m = (r.method || 'GET').toUpperCase()
    counts[m] = (counts[m] || 0) + 1
  })
  return Object.entries(counts)
    .map(([method, count]) => ({ method, count }))
    .sort((a, b) => b.count - a.count)
})

// Enriched Status Count
const enrichedCount = computed(() => {
  return routes.value.filter(r => r.domain && r.domain !== 'Umum' && r.description && r.description.length > 10).length
})

// Filtered Routes
const filteredRoutes = computed(() => {
  let list = routes.value

  // Domain Filter
  if (selectedDomain.value !== 'ALL') {
    list = list.filter(r => (r.domain || 'Umum') === selectedDomain.value)
  }

  // Method Filter
  if (selectedMethod.value !== 'ALL') {
    list = list.filter(r => (r.method || '').toUpperCase() === selectedMethod.value)
  }

  // Search Filter
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    list = list.filter(r =>
      r.path.toLowerCase().includes(q)
      || r.method.toLowerCase().includes(q)
      || (r.handler && r.handler.toLowerCase().includes(q))
      || (r.description && r.description.toLowerCase().includes(q))
      || (r.domain && r.domain.toLowerCase().includes(q))
      || (r.function_group && r.function_group.toLowerCase().includes(q))
      || (r.middleware && r.middleware.toLowerCase().includes(q))
      || (r.tags && r.tags.toLowerCase().includes(q))
      || (r.target_requests && r.target_requests.toLowerCase().includes(q))
    )
  }

  return list
})

// Grouped by Domain
const groupedRoutes = computed<{ domain: string, routes: ComponentRoute[] }[]>(() => {
  const groups: Record<string, ComponentRoute[]> = {}
  filteredRoutes.value.forEach((r) => {
    const d = r.domain || 'Umum'
    if (!groups[d]) groups[d] = []
    groups[d]!.push(r)
  })

  return Object.keys(groups)
    .sort()
    .map(domainName => ({
      domain: domainName,
      routes: groups[domainName] || []
    }))
})

// Load routes from database
const loadRoutes = async () => {
  if (!props.comp?.id) return
  loading.value = true
  try {
    const list = await getComponentRoutes(props.comp.id)
    routes.value = list || []
    emit('routes-count-changed', routes.value.length)
  } catch (err: any) {
    console.error('Failed to load routes from database:', err)
  } finally {
    loading.value = false
  }
}

// Auto Sync from Code
const handleSyncFromCode = async () => {
  if (!props.comp?.id) return
  syncing.value = true
  feedbackMessage.value = null
  try {
    const res = await syncComponentRoutes(props.comp.id)
    routes.value = res.routes || []
    feedbackSuccess.value = true
    feedbackMessage.value = isFrontend.value
      ? `Sinkronisasi Halaman Berhasil: +${res.added_count} halaman baru ditambahkan, ~${res.updated_count} diperbarui (termasuk request API), -${res.deleted_count} halaman usang dihapus. Total: ${res.total_count} halaman tersimpan di database.`
      : `Sinkronisasi Kode Berhasil: +${res.added_count} rute baru ditambahkan, ~${res.updated_count} diperbarui, -${res.deleted_count} rute usang dihapus. Total: ${res.total_count} rute tersimpan di database.`
    emit('routes-count-changed', res.total_count)
    setTimeout(() => {
      feedbackMessage.value = null
    }, 9000)
  } catch (err: any) {
    feedbackSuccess.value = false
    feedbackMessage.value = `Gagal menyinkronkan dari kode: ${err.message || err}`
  } finally {
    syncing.value = false
  }
}

// Auto Enrich & Group via LLM
const handleEnrichWithLLM = () => {
  if (!props.comp?.id || enriching.value) return
  feedbackMessage.value = null
  enrichJob.start()
}

// Copy endpoint path
const copyRoutePath = async (p: string) => {
  try {
    await navigator.clipboard.writeText(p)
    copiedPath.value = p
    setTimeout(() => {
      if (copiedPath.value === p) copiedPath.value = null
    }, 2000)
  } catch {
    // ignore
  }
}

// Edit route modal
const openEditModal = (r: ComponentRoute) => {
  editingRoute.value = r
  editForm.value = {
    domain: r.domain || 'Umum',
    function_group: r.function_group || 'Lainnya',
    description: r.description || '',
    tags: r.tags || '',
    target_requests: r.target_requests ? parseTargetRequests(r.target_requests).join('\n') : ''
  }
  editModalOpen.value = true
}

const handleSaveEdit = async () => {
  if (!editingRoute.value || !props.comp?.id) return
  savingEdit.value = true
  try {
    const reqsArray = editForm.value.target_requests
      .split('\n')
      .map(s => s.trim())
      .filter(Boolean)
    const payload = {
      domain: editForm.value.domain,
      function_group: editForm.value.function_group,
      description: editForm.value.description,
      tags: editForm.value.tags,
      target_requests: JSON.stringify(reqsArray)
    }
    const updated = await updateComponentRoute(props.comp.id, editingRoute.value.id, payload)
    const idx = routes.value.findIndex(item => item.id === updated.id)
    if (idx !== -1) {
      routes.value[idx] = updated
    }
    editModalOpen.value = false
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menyimpan Rute',
      description: `Gagal menyimpan perubahan: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    savingEdit.value = false
  }
}

// Delete route
const pendingDelete = ref<ComponentRoute | null>(null)
const deleting = ref(false)

const handleDelete = (r: ComponentRoute) => {
  pendingDelete.value = r
}

const cancelDelete = () => {
  if (!deleting.value) pendingDelete.value = null
}

const confirmDelete = async () => {
  const r = pendingDelete.value
  if (!r) return
  deleting.value = true
  try {
    await deleteComponentRoute(props.comp.id, r.id)
    pendingDelete.value = null
    routes.value = routes.value.filter(item => item.id !== r.id)
    emit('routes-count-changed', routes.value.length)
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menghapus Rute',
      description: `Gagal menghapus rute: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    deleting.value = false
  }
}

// Watch component ID changes
watch(() => props.comp?.id, () => {
  loadRoutes()
}, { immediate: true })

defineExpose({
  loadRoutes,
  handleSyncFromCode,
  handleEnrichWithLLM
})
</script>

<template>
  <div class="space-y-4">
    <!-- Top Action Card & Header -->
    <div class="p-5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs space-y-4">
      <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
        <div>
          <div class="flex flex-wrap items-center gap-2.5">
            <h3 class="font-bold text-neutral-900 dark:text-white text-sm uppercase tracking-wider flex items-center gap-2">
              <UIcon :name="isFrontend ? 'i-lucide-panels-top-left' : 'i-lucide-route'" class="w-4 h-4 text-sky-400" />
              <span>{{ isFrontend ? 'Daftar Halaman & Request API' : 'Daftar Router & API Endpoints Database' }}</span>
            </h3>
            <UBadge size="xs" color="success" variant="subtle" class="font-bold">
              {{ routes.length }} {{ isFrontend ? 'Halaman di DB' : 'Rute di DB' }}
            </UBadge>
            <UBadge size="xs" color="neutral" variant="subtle">
              {{ domainList.length }} {{ isFrontend ? 'Modul Fitur' : 'Domain Bisnis' }}
            </UBadge>
            <UBadge v-if="isFrontend && totalTargetRequestsCount > 0" size="xs" color="info" variant="subtle" class="font-semibold">
              {{ totalTargetRequestsCount }} Endpoint API Terpetakan
            </UBadge>
            <UBadge v-if="enrichedCount > 0" size="xs" color="primary" variant="subtle">
              {{ enrichedCount }} Terkategori LLM
            </UBadge>
          </div>
          <p class="text-xs text-neutral-500 mt-1">
            {{ isFrontend
              ? 'Katalog halaman antarmuka pengguna tersimpan permanen di database dan memetakan seluruh request endpoint API yang dipanggil di tiap halaman.'
              : 'Router tersimpan secara permanen di database. Sinkronkan dengan file kode sumber kapan saja dan kelompokkan fungsionalitasnya menggunakan AI / LLM.' }}
          </p>
        </div>

        <!-- 2 Action Buttons -->
        <div class="flex flex-wrap items-center gap-2 shrink-0">
          <!-- Button 1: Auto Sinkron Kode -->
          <UButton
            icon="i-lucide-refresh-cw"
            color="primary"
            variant="soft"
            size="xs"
            :loading="syncing"
            :title="isFrontend
              ? 'Pindai kode sumber aktual: sinkronkan daftar halaman dan deteksi request API terbaru dari komponen/halaman'
              : 'Pindai kode sumber aktual: tambah rute baru, perbarui perubahan, dan hapus rute yang hilang dari file kode'"
            @click="handleSyncFromCode"
          >
            {{ syncing ? 'Menyinkronkan...' : (isFrontend ? 'Auto Sinkron Halaman' : 'Auto Sinkron Kode') }}
          </UButton>

          <!-- Button 2: Auto Keterangan & Kelompokkan via LLM -->
          <UButton
            icon="i-lucide-sparkles"
            color="primary"
            variant="solid"
            size="xs"
            :loading="enriching"
            :title="isFrontend
              ? 'Kirim daftar halaman ke LLM untuk dikelompokkan ke modul fitur serta dilengkapi keterangan profesional'
              : 'Kirim daftar rute ke model LLM untuk dikelompokkan berdasarkan fungsi & domain serta dilengkapi keterangan profesional'"
            @click="handleEnrichWithLLM"
          >
            {{ enriching ? (enrichLabel || 'Menganalisis LLM...') : 'Kirim ke LLM & Kelompokkan' }}
          </UButton>
        </div>
      </div>

      <!-- Feedback / Sync Notification Banner -->
      <div
        v-if="feedbackMessage"
        class="p-3 rounded-lg border text-xs flex items-center justify-between gap-3 transition-all"
        :class="feedbackSuccess ? 'bg-sky-50/70 dark:bg-sky-950/40 border-sky-200 dark:border-sky-800 text-sky-800 dark:text-sky-200' : 'bg-rose-50/70 dark:bg-rose-950/40 border-rose-200 dark:border-rose-800 text-rose-800 dark:text-rose-200'"
      >
        <div class="flex items-center gap-2">
          <UIcon :name="feedbackSuccess ? 'i-lucide-check-circle' : 'i-lucide-alert-circle'" class="w-4 h-4 shrink-0" />
          <span>{{ feedbackMessage }}</span>
        </div>
        <button
          type="button"
          class="text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200 cursor-pointer"
          @click="feedbackMessage = null"
        >
          <UIcon name="i-lucide-x" class="w-3.5 h-3.5" />
        </button>
      </div>

      <!-- Toolbar: Search, Filters & View Toggle -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-2 border-t border-neutral-100 dark:border-neutral-800/80">
        <!-- Search Input -->
        <div class="w-full sm:w-80">
          <UInput
            v-model="searchQuery"
            icon="i-lucide-search"
            :placeholder="isFrontend ? 'Cari rute halaman, file vue, modul, atau request API...' : 'Cari path, method, handler, domain...'"
            size="xs"
          />
        </div>

        <!-- Right Controls: Method filter & View Mode -->
        <div class="flex flex-wrap items-center gap-2">
          <!-- Method Filter Dropdown / Select -->
          <div class="flex items-center gap-1 text-xs text-neutral-500">
            <span class="hidden md:inline text-[11px]">Method:</span>
            <select
              v-model="selectedMethod"
              class="px-2 py-1 rounded-md border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 text-xs font-mono font-bold text-neutral-700 dark:text-neutral-300"
            >
              <option value="ALL">
                Semua Method ({{ routes.length }})
              </option>
              <option v-for="m in methodList" :key="m.method" :value="m.method">
                {{ m.method }} ({{ m.count }})
              </option>
            </select>
          </div>

          <!-- View Mode Toggle -->
          <div class="flex items-center p-0.5 rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-100/80 dark:bg-neutral-800/60 text-xs">
            <button
              type="button"
              class="px-2.5 py-1 rounded-md flex items-center gap-1.5 transition-colors cursor-pointer text-[11px]"
              :class="viewMode === 'grouped' ? 'bg-white dark:bg-neutral-900 font-bold text-neutral-900 dark:text-white shadow-2xs' : 'text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-200'"
              @click="viewMode = 'grouped'"
            >
              <UIcon name="i-lucide-layers" class="w-3.5 h-3.5" />
              <span>{{ isFrontend ? 'Kelompok Modul' : 'Kelompok Domain' }}</span>
            </button>
            <button
              type="button"
              class="px-2.5 py-1 rounded-md flex items-center gap-1.5 transition-colors cursor-pointer text-[11px]"
              :class="viewMode === 'table' ? 'bg-white dark:bg-neutral-900 font-bold text-neutral-900 dark:text-white shadow-2xs' : 'text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-200'"
              @click="viewMode = 'table'"
            >
              <UIcon name="i-lucide-table" class="w-3.5 h-3.5" />
              <span>Tabel Lengkap</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Domain Selector Horizontal Pills -->
      <div v-if="domainList.length > 0" class="flex flex-wrap items-center gap-1.5 pt-1 text-xs">
        <span class="text-[11px] font-semibold text-neutral-400 mr-1">{{ isFrontend ? 'Modul:' : 'Domain:' }}</span>
        <button
          type="button"
          class="px-2.5 py-1 rounded-full text-[11px] font-medium transition-colors cursor-pointer"
          :class="selectedDomain === 'ALL' ? 'bg-primary-500 text-white font-bold' : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:bg-neutral-200 dark:hover:bg-neutral-700'"
          @click="selectedDomain = 'ALL'"
        >
          Semua ({{ routes.length }})
        </button>

        <button
          v-for="d in domainList"
          :key="d.domain"
          type="button"
          class="px-2.5 py-1 rounded-full text-[11px] font-medium transition-colors cursor-pointer flex items-center gap-1"
          :class="selectedDomain === d.domain ? 'bg-primary-500 text-white font-bold' : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:bg-neutral-200 dark:hover:bg-neutral-700'"
          @click="selectedDomain = d.domain"
        >
          <span>{{ d.domain }}</span>
          <span class="opacity-70 text-[10px]">({{ d.count }})</span>
        </button>
      </div>
    </div>

    <!-- Empty State: Belum Ada Data di Database -->
    <div
      v-if="routes.length === 0 && !loading"
      class="p-8 text-center rounded-xl border border-dashed border-neutral-200 dark:border-neutral-800 bg-white/50 dark:bg-neutral-900/50 space-y-3"
    >
      <div class="w-12 h-12 mx-auto rounded-full bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center text-neutral-400">
        <UIcon :name="isFrontend ? 'i-lucide-panels-top-left' : 'i-lucide-route-off'" class="w-6 h-6" />
      </div>
      <div class="space-y-1">
        <h4 class="text-sm font-semibold text-neutral-900 dark:text-white">
          {{ isFrontend ? 'Belum Ada Halaman Tersimpan di Database' : 'Belum Ada Router Tersimpan di Database' }}
        </h4>
        <p class="text-xs text-neutral-500 max-w-md mx-auto">
          {{ isFrontend
            ? 'Klik tombol di bawah untuk memindai file halaman (pages) dan pemetaan request API aplikasi ini secara otomatis. Data akan disimpan ke database agar siap digunakan.'
            : 'Klik tombol di bawah untuk memindai file kode sumber aplikasi ini secara otomatis. Rute akan disimpan ke database agar tidak perlu dipindai berulang-ulang.' }}
        </p>
      </div>
      <UButton
        icon="i-lucide-refresh-cw"
        color="primary"
        size="xs"
        :loading="syncing"
        @click="handleSyncFromCode"
      >
        {{ isFrontend ? 'Auto Sinkron Halaman Sekarang' : 'Auto Sinkron Kode Sekarang' }}
      </UButton>
    </div>

    <!-- Filter Zero State: Pencarian Tidak Cocok -->
    <div
      v-else-if="filteredRoutes.length === 0 && routes.length > 0"
      class="p-8 text-center rounded-xl border border-dashed border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 text-xs text-neutral-400 space-y-2"
    >
      <UIcon name="i-lucide-search-x" class="w-6 h-6 mx-auto text-neutral-400" />
      <div>
        {{ isFrontend ? 'Tidak ada halaman yang cocok dengan filter atau kata kunci pencarian.' : 'Tidak ada rute yang cocok dengan filter atau kata kunci pencarian.' }}
      </div>
      <UButton size="xs" variant="ghost" color="neutral" @click="searchQuery = ''; selectedDomain = 'ALL'; selectedMethod = 'ALL'">
        Reset Filter
      </UButton>
    </div>

    <!-- VIEW MODE 1: DOMAIN GROUPED VIEW -->
    <div v-else-if="viewMode === 'grouped'" class="space-y-4">
      <div
        v-for="group in groupedRoutes"
        :key="group.domain"
        class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs overflow-hidden"
      >
        <!-- Group Header -->
        <div class="p-3.5 bg-neutral-50/80 dark:bg-neutral-800 border-b border-neutral-200 dark:border-neutral-800 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <UIcon :name="isFrontend ? 'i-lucide-panels-top-left' : 'i-lucide-folder-git-2'" class="w-4 h-4 text-primary-500" />
            <span class="font-bold text-xs text-neutral-900 dark:text-white uppercase tracking-wider">
              {{ group.domain }}
            </span>
            <UBadge size="xs" color="neutral" variant="subtle" class="font-semibold">
              {{ group.routes.length }} {{ isFrontend ? 'halaman' : 'endpoint' }}
            </UBadge>
          </div>
        </div>

        <!-- Table within Group -->
        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs">
            <thead class="bg-neutral-50/40 dark:bg-neutral-800/30 text-neutral-400 uppercase text-[10px] tracking-wider border-b border-neutral-100 dark:border-neutral-800/80">
              <tr v-if="isFrontend">
                <th class="py-2.5 px-4 w-20">
                  Tipe
                </th>
                <th class="py-2.5 px-4">
                  Rute Halaman
                </th>
                <th class="py-2.5 px-4 min-w-[240px]">
                  Request API yang Dipanggil
                </th>
                <th class="py-2.5 px-4 w-36">
                  Sub-Fitur
                </th>
                <th class="py-2.5 px-4">
                  Keterangan Halaman
                </th>
                <th class="py-2.5 px-4 text-right w-44">
                  File Sumber
                </th>
                <th class="py-2.5 px-4 text-center w-20">
                  Aksi
                </th>
              </tr>
              <tr v-else>
                <th class="py-2.5 px-4 w-24">
                  Method
                </th>
                <th class="py-2.5 px-4">
                  Endpoint Path & Middleware
                </th>
                <th class="py-2.5 px-4 w-44">
                  Kelompok Fungsi
                </th>
                <th class="py-2.5 px-4">
                  Keterangan / Penjelasan
                </th>
                <th class="py-2.5 px-4 text-right w-44">
                  Handler
                </th>
                <th class="py-2.5 px-4 text-center w-20">
                  Aksi
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/80">
              <tr
                v-for="r in group.routes"
                :key="r.id"
                class="hover:bg-neutral-50/60 dark:hover:bg-neutral-800/40 transition-colors"
              >
                <!-- Method / Type Badge -->
                <td class="py-3 px-4 whitespace-nowrap align-top">
                  <UBadge
                    size="xs"
                    :color="getMethodBadgeColor(r.method)"
                    variant="solid"
                    class="font-mono font-bold shrink-0 tracking-wider"
                  >
                    {{ r.method }}
                  </UBadge>
                </td>

                <!-- Path & Middleware -->
                <td class="py-3 px-4 align-top">
                  <div class="flex items-center gap-2 flex-wrap">
                    <span class="font-mono font-semibold text-neutral-900 dark:text-neutral-100 select-all">
                      {{ r.path }}
                    </span>
                    <button
                      type="button"
                      class="text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 cursor-pointer transition-colors"
                      :title="isFrontend ? 'Salin rute halaman' : 'Salin path endpoint'"
                      @click="copyRoutePath(r.path)"
                    >
                      <UIcon
                        :name="copiedPath === r.path ? 'i-lucide-check' : 'i-lucide-copy'"
                        class="w-3.5 h-3.5"
                        :class="copiedPath === r.path ? 'text-sky-400' : ''"
                      />
                    </button>
                    <span
                      v-if="r.middleware"
                      class="text-[10px] px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 font-mono text-neutral-500 shrink-0"
                    >
                      {{ r.middleware }}
                    </span>
                  </div>
                  <div v-if="r.tags" class="text-[10px] text-neutral-400 mt-0.5">
                    Tags: {{ r.tags }}
                  </div>
                </td>

                <!-- Target Requests (Only Frontend) -->
                <td v-if="isFrontend" class="py-3 px-4 align-top">
                  <div v-if="parseTargetRequests(r.target_requests).length > 0" class="space-y-1.5 max-w-sm">
                    <div class="flex flex-wrap gap-1">
                      <span
                        v-for="(req, idx) in (expandedRequests[r.id] ? parseTargetRequests(r.target_requests) : parseTargetRequests(r.target_requests).slice(0, 3))"
                        :key="idx"
                        class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[10px] font-mono border"
                        :class="[getRequestMethodColor(req).bg, getRequestMethodColor(req).text, getRequestMethodColor(req).border]"
                      >
                        <span class="font-bold shrink-0">{{ req.split(' ')[0] }}</span>
                        <span class="truncate max-w-[200px]" :title="req">{{ req.substring(req.indexOf(' ') + 1) || req }}</span>
                      </span>
                    </div>
                    <button
                      v-if="parseTargetRequests(r.target_requests).length > 3"
                      type="button"
                      class="text-[10px] font-semibold text-primary-500 hover:text-primary-600 dark:hover:text-primary-400 cursor-pointer block mt-1"
                      @click="toggleExpandRequests(r.id)"
                    >
                      {{ expandedRequests[r.id] ? 'Tutup' : `+${parseTargetRequests(r.target_requests).length - 3} request lainnya` }}
                    </button>
                  </div>
                  <span v-else class="text-neutral-400 italic text-[11px]">
                    -
                  </span>
                </td>

                <!-- Kelompok Fungsi / Sub-Fitur -->
                <td class="py-3 px-4 align-top">
                  <UBadge size="xs" color="neutral" variant="outline" class="font-medium text-[11px]">
                    {{ r.function_group || (isFrontend ? 'Halaman' : 'Umum') }}
                  </UBadge>
                </td>

                <!-- Keterangan / Deskripsi -->
                <td class="py-3 px-4 align-top text-neutral-700 dark:text-neutral-300 leading-relaxed text-[11.5px]">
                  <p v-if="r.description">
                    {{ r.description }}
                  </p>
                  <p v-else class="text-neutral-400 italic text-[11px]">
                    {{ isFrontend ? 'Belum ada keterangan. Klik "Kirim ke LLM" untuk membuat keterangan halaman otomatis.' : 'Belum ada keterangan. Klik "Kirim ke LLM" untuk membuat keterangan otomatis.' }}
                  </p>
                </td>

                <!-- Handler / File Sumber -->
                <td class="py-3 px-4 text-right align-top font-mono text-[11px] text-neutral-500 truncate max-w-xs">
                  <span :title="r.handler">{{ r.handler || '-' }}</span>
                </td>

                <!-- Actions -->
                <td class="py-3 px-4 text-center align-top whitespace-nowrap">
                  <div class="flex items-center justify-center gap-1">
                    <button
                      type="button"
                      class="p-1 rounded text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 cursor-pointer"
                      :title="isFrontend ? 'Edit keterangan dan modul halaman' : 'Edit keterangan dan domain'"
                      @click="openEditModal(r)"
                    >
                      <UIcon name="i-lucide-pencil" class="w-3.5 h-3.5" />
                    </button>
                    <button
                      type="button"
                      class="p-1 rounded text-neutral-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/40 cursor-pointer"
                      :title="isFrontend ? 'Hapus halaman dari database' : 'Hapus rute dari database'"
                      @click="handleDelete(r)"
                    >
                      <UIcon name="i-lucide-trash-2" class="w-3.5 h-3.5" />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- VIEW MODE 2: FLAT DATA TABLE -->
    <div
      v-else
      class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs overflow-hidden"
    >
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-neutral-50 dark:bg-neutral-800 text-neutral-500 uppercase text-[10px] tracking-wider border-b border-neutral-200 dark:border-neutral-800">
            <tr v-if="isFrontend">
              <th class="py-3 px-4 w-20">
                Tipe
              </th>
              <th class="py-3 px-4">
                Rute Halaman
              </th>
              <th class="py-3 px-4 w-36">
                Modul Fitur
              </th>
              <th class="py-3 px-4 min-w-[220px]">
                Request API yang Dipanggil
              </th>
              <th class="py-3 px-4 w-36">
                Sub-Fitur
              </th>
              <th class="py-3 px-4">
                Keterangan Halaman
              </th>
              <th class="py-3 px-4 text-right w-40">
                File Sumber
              </th>
              <th class="py-3 px-4 text-center w-20">
                Aksi
              </th>
            </tr>
            <tr v-else>
              <th class="py-3 px-4 w-20">
                Method
              </th>
              <th class="py-3 px-4">
                Endpoint Path & Middleware
              </th>
              <th class="py-3 px-4 w-36">
                Domain
              </th>
              <th class="py-3 px-4 w-40">
                Kelompok Fungsi
              </th>
              <th class="py-3 px-4">
                Keterangan / Penjelasan
              </th>
              <th class="py-3 px-4 text-right w-40">
                Handler
              </th>
              <th class="py-3 px-4 text-center w-20">
                Aksi
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/80">
            <tr
              v-for="r in filteredRoutes"
              :key="r.id"
              class="hover:bg-neutral-50/60 dark:hover:bg-neutral-800/40 transition-colors"
            >
              <!-- Method / Type Badge -->
              <td class="py-3.5 px-4 whitespace-nowrap align-top">
                <UBadge
                  size="xs"
                  :color="getMethodBadgeColor(r.method)"
                  variant="solid"
                  class="font-mono font-bold shrink-0 tracking-wider"
                >
                  {{ r.method }}
                </UBadge>
              </td>

              <!-- Endpoint Path & Middleware -->
              <td class="py-3.5 px-4 align-top">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-mono font-semibold text-neutral-900 dark:text-neutral-100 select-all">
                    {{ r.path }}
                  </span>
                  <button
                    type="button"
                    class="text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 cursor-pointer transition-colors"
                    :title="isFrontend ? 'Salin rute halaman' : 'Salin path endpoint'"
                    @click="copyRoutePath(r.path)"
                  >
                    <UIcon
                      :name="copiedPath === r.path ? 'i-lucide-check' : 'i-lucide-copy'"
                      class="w-3.5 h-3.5"
                      :class="copiedPath === r.path ? 'text-sky-400' : ''"
                    />
                  </button>
                  <span
                    v-if="r.middleware"
                    class="text-[10px] px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 font-mono text-neutral-500 shrink-0"
                  >
                    {{ r.middleware }}
                  </span>
                </div>
              </td>

              <!-- Domain / Modul Fitur -->
              <td class="py-3.5 px-4 align-top">
                <UBadge size="xs" color="primary" variant="subtle" class="font-semibold text-[11px]">
                  {{ r.domain || 'Umum' }}
                </UBadge>
              </td>

              <!-- Target Requests (Only Frontend) -->
              <td v-if="isFrontend" class="py-3.5 px-4 align-top">
                <div v-if="parseTargetRequests(r.target_requests).length > 0" class="space-y-1.5 max-w-sm">
                  <div class="flex flex-wrap gap-1">
                    <span
                      v-for="(req, idx) in (expandedRequests[r.id] ? parseTargetRequests(r.target_requests) : parseTargetRequests(r.target_requests).slice(0, 3))"
                      :key="idx"
                      class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[10px] font-mono border"
                      :class="[getRequestMethodColor(req).bg, getRequestMethodColor(req).text, getRequestMethodColor(req).border]"
                    >
                      <span class="font-bold shrink-0">{{ req.split(' ')[0] }}</span>
                      <span class="truncate max-w-[180px]" :title="req">{{ req.substring(req.indexOf(' ') + 1) || req }}</span>
                    </span>
                  </div>
                  <button
                    v-if="parseTargetRequests(r.target_requests).length > 3"
                    type="button"
                    class="text-[10px] font-semibold text-primary-500 hover:text-primary-600 dark:hover:text-primary-400 cursor-pointer block mt-1"
                    @click="toggleExpandRequests(r.id)"
                  >
                    {{ expandedRequests[r.id] ? 'Tutup' : `+${parseTargetRequests(r.target_requests).length - 3} request lainnya` }}
                  </button>
                </div>
                <span v-else class="text-neutral-400 italic text-[11px]">
                  -
                </span>
              </td>

              <!-- Kelompok Fungsi / Sub-Fitur -->
              <td class="py-3.5 px-4 align-top">
                <span class="text-neutral-700 dark:text-neutral-300 font-medium text-[11px]">
                  {{ r.function_group || (isFrontend ? 'Halaman' : 'General') }}
                </span>
              </td>

              <!-- Keterangan / Deskripsi -->
              <td class="py-3.5 px-4 align-top text-neutral-700 dark:text-neutral-300 leading-relaxed text-[11.5px]">
                <p v-if="r.description">
                  {{ r.description }}
                </p>
                <p v-else class="text-neutral-400 italic text-[11px]">
                  {{ isFrontend ? 'Belum ada keterangan. Klik "Kirim ke LLM" untuk membuat keterangan halaman otomatis.' : 'Belum ada keterangan. Klik "Kirim ke LLM" untuk membuat keterangan otomatis.' }}
                </p>
              </td>

              <!-- Handler / File Sumber -->
              <td class="py-3.5 px-4 text-right align-top font-mono text-[11px] text-neutral-500 truncate max-w-xs">
                <span :title="r.handler">{{ r.handler || '-' }}</span>
              </td>

              <!-- Aksi -->
              <td class="py-3.5 px-4 text-center align-top whitespace-nowrap">
                <div class="flex items-center justify-center gap-1">
                  <button
                    type="button"
                    class="p-1 rounded text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 cursor-pointer"
                    :title="isFrontend ? 'Edit keterangan dan modul halaman' : 'Edit keterangan dan domain'"
                    @click="openEditModal(r)"
                  >
                    <UIcon name="i-lucide-pencil" class="w-3.5 h-3.5" />
                  </button>
                  <button
                    type="button"
                    class="p-1 rounded text-neutral-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/40 cursor-pointer"
                    :title="isFrontend ? 'Hapus halaman dari database' : 'Hapus rute dari database'"
                    @click="handleDelete(r)"
                  >
                    <UIcon name="i-lucide-trash-2" class="w-3.5 h-3.5" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Edit Route Modal -->
    <UModal
      v-model:open="editModalOpen"
      :title="isFrontend ? 'Edit Keterangan Halaman & Request API' : 'Edit Keterangan & Kelompok Router'"
      :description="isFrontend ? 'Sesuaikan modul fitur, sub-fitur, request API, atau keterangan untuk halaman ini.' : 'Sesuaikan domain bisnis, kelompok fungsi, atau keterangan untuk endpoint ini.'"
      :ui="{ content: 'sm:max-w-lg' }"
    >
      <template #body>
        <div v-if="editingRoute" class="space-y-4 text-xs">
          <!-- Endpoint Identifier Badge -->
          <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 flex items-center gap-2">
            <UBadge size="xs" :color="getMethodBadgeColor(editingRoute.method)" variant="solid" class="font-mono font-bold">
              {{ editingRoute.method }}
            </UBadge>
            <span class="font-mono font-semibold text-neutral-900 dark:text-neutral-100">{{ editingRoute.path }}</span>
          </div>

          <!-- Domain / Modul Fitur -->
          <div>
            <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">
              {{ isFrontend ? 'Modul Fitur' : 'Domain Bisnis' }} <span class="text-rose-500">*</span>
            </label>
            <UInput
              v-model="editForm.domain"
              :placeholder="isFrontend ? 'misal: Dashboard, Manajemen Proyek, Pengaturan' : 'misal: Autentikasi & Keamanan, Manajemen Proyek, Aset'"
            />
          </div>

          <!-- Kelompok Fungsi / Sub-Fitur -->
          <div>
            <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">
              {{ isFrontend ? 'Sub-Fitur / Jenis Halaman' : 'Kelompok Fungsi' }} <span class="text-rose-500">*</span>
            </label>
            <UInput
              v-model="editForm.function_group"
              :placeholder="isFrontend ? 'misal: Detail Aplikasi, Form Buat, Ringkasan' : 'misal: Login & Sesi, Pengambilan Data (Read), Mutasi Data'"
            />
          </div>

          <!-- Target Requests (Only Frontend) -->
          <div v-if="isFrontend">
            <div class="flex items-center justify-between mb-1">
              <label class="block font-medium text-neutral-700 dark:text-neutral-300">
                Request API yang Dipanggil (1 baris per request)
              </label>
              <span class="text-[10px] text-neutral-400">Contoh: GET /api/v1/projects</span>
            </div>
            <UTextarea
              v-model="editForm.target_requests"
              :rows="4"
              placeholder="GET /api/v1/projects&#10;POST /api/v1/projects&#10;GET /api/v1/auth/me"
              class="font-mono text-xs"
            />
            <p class="text-[10.5px] text-neutral-400 mt-1">
              Daftar endpoint backend yang dipanggil oleh halaman ini.
            </p>
          </div>

          <!-- Keterangan / Deskripsi -->
          <div>
            <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">
              {{ isFrontend ? 'Keterangan / Tujuan Halaman' : 'Keterangan / Penjelasan Fungsi' }} <span class="text-rose-500">*</span>
            </label>
            <UTextarea
              v-model="editForm.description"
              :rows="3"
              :placeholder="isFrontend ? 'Jelaskan tujuan antarmuka halaman ini, interaksi yang disediakan, dan data yang dimuat...' : 'Jelaskan peran endpoint ini, parameter yang diterima, dan respon yang dihasilkan...'"
            />
          </div>

          <!-- Tags -->
          <div>
            <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">Kata Kunci (Tags)</label>
            <UInput v-model="editForm.tags" placeholder="misal: auth, login, jwt, session (pisahkan dengan koma)" />
          </div>
        </div>
      </template>

      <template #footer>
        <div class="flex items-center justify-end gap-2">
          <UButton color="neutral" variant="ghost" size="xs" @click="editModalOpen = false">
            Batal
          </UButton>
          <UButton color="primary" size="xs" :loading="savingEdit" @click="handleSaveEdit">
            Simpan Perubahan
          </UButton>
        </div>
      </template>
    </UModal>
    <!-- Konfirmasi (pengganti confirm() bawaan browser) -->
    <UModal
      :open="!!pendingDelete"
      title="Hapus rute?"
      :description="pendingDelete ? `Hapus endpoint ${pendingDelete.method} ${pendingDelete.path} dari database?` : undefined"
      @update:open="(v: boolean) => { if (!v) cancelDelete() }"
    >
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton color="neutral" variant="ghost" label="Batal" :disabled="deleting" @click="cancelDelete" />
          <UButton color="error" icon="i-lucide-trash-2" label="Hapus" :loading="deleting" @click="confirmDelete" />
        </div>
      </template>
    </UModal>
  </div>
</template>
