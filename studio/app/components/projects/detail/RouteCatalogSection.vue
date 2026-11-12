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

const totalTargetRequestsCount = computed(() => {
  let count = 0
  routes.value.forEach((r) => {
    count += parseTargetRequests(r.target_requests).length
  })
  return count
})

// Baris yang request-nya diperluas & path yang baru disalin (dipakai tampilan grup + tabel)
const expandedRequests = ref<Record<number, boolean>>({})
const toggleExpandRequests = (routeId: number) => {
  expandedRequests.value[routeId] = !expandedRequests.value[routeId]
}

// Filter & View State
const searchQuery = ref('')
const selectedDomain = ref<string>('ALL')
const selectedMethod = ref<string>('ALL')
const viewMode = ref<'grouped' | 'table'>('grouped')
const copiedPath = ref<string | null>(null)

const editModal = ref<{ open: (r: ComponentRoute) => void } | null>(null)
const openEditModal = (r: ComponentRoute) => editModal.value?.open(r)
const onRouteSaved = (updated: ComponentRoute) => {
  const idx = routes.value.findIndex(item => item.id === updated.id)
  if (idx !== -1) {
    routes.value[idx] = updated
  }
}

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

// Paging: satu baris rute (badge, path, request target, aksi) ≈ 40 simpul DOM → 20 per halaman
// (anggaran 800 simpul, utils/listPaging). Tampilan grup dipaging dari daftar yang sudah diurutkan per domain
// agar satu domain tidak tercecer di beberapa halaman secara acak.
const domainOf = (r: ComponentRoute) => r.domain || 'Umum'
const pagingSource = computed(() => viewMode.value === 'grouped'
  ? [...filteredRoutes.value].sort((a, b) => domainOf(a).localeCompare(domainOf(b)))
  : filteredRoutes.value)
const routePaging = useListPaging(pagingSource, {
  nodesPerRow: 40, queryKey: 'rpg', resetOn: [searchQuery, selectedDomain, selectedMethod, viewMode]
})

const domainTotals = computed(() => {
  const counts: Record<string, number> = {}
  for (const r of filteredRoutes.value) counts[domainOf(r)] = (counts[domainOf(r)] || 0) + 1
  return counts
})

// Grouped by Domain (dari halaman yang sedang tampil)
const groupedRoutes = computed<{ domain: string, routes: ComponentRoute[] }[]>(() => {
  const groups: Record<string, ComponentRoute[]> = {}
  routePaging.pagedItems.value.forEach((r) => {
    const d = domainOf(r)
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
    <ProjectsDetailRouteCatalogHeader
      v-model:search-query="searchQuery"
      v-model:selected-domain="selectedDomain"
      v-model:selected-method="selectedMethod"
      v-model:view-mode="viewMode"
      :routes-count="routes.length"
      :is-frontend="isFrontend"
      :loading="loading"
      :syncing="syncing"
      :enriching="enriching"
      :enrich-label="enrichLabel"
      :feedback-message="feedbackMessage"
      :feedback-success="feedbackSuccess"
      :enriched-count="enrichedCount"
      :total-target-requests-count="totalTargetRequestsCount"
      :domain-list="domainList"
      :method-list="methodList"
      @sync="handleSyncFromCode"
      @enrich="handleEnrichWithLLM"
    />

    <!-- Empty State: Belum Ada Data di Database -->
    <div
      v-if="routes.length === 0 && !loading"
      class="p-8 text-center rounded-xl border border-dashed border-neutral-200 dark:border-white/[0.08] bg-white/50 dark:bg-neutral-900/50 space-y-3"
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
      class="p-8 text-center rounded-xl border border-dashed border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] text-xs text-neutral-400 space-y-2"
    >
      <UIcon name="i-lucide-search-x" class="w-6 h-6 mx-auto text-neutral-400" />
      <div>
        {{ isFrontend ? 'Tidak ada halaman yang cocok dengan filter atau kata kunci pencarian.' : 'Tidak ada rute yang cocok dengan filter atau kata kunci pencarian.' }}
      </div>
      <UButton size="xs" variant="ghost" color="neutral" @click="searchQuery = ''; selectedDomain = 'ALL'; selectedMethod = 'ALL'">
        Reset Filter
      </UButton>
    </div>

    <ProjectsDetailRouteCatalogGroupedView
      v-else-if="viewMode === 'grouped'"
      id="route-catalog-list"
      class="scroll-mt-32"
      :grouped-routes="groupedRoutes"
      :domain-totals="domainTotals"
      :is-frontend="isFrontend"
      :project-id="project?.id"
      :component-id="comp?.id"
      :expanded-requests="expandedRequests"
      :copied-path="copiedPath"
      @toggle-expand="toggleExpandRequests"
      @copy-path="copyRoutePath"
      @edit="openEditModal"
      @delete="handleDelete"
    />

    <ProjectsDetailRouteCatalogTableView
      v-else
      id="route-catalog-list"
      class="scroll-mt-32"
      :filtered-routes="routePaging.pagedItems.value"
      :is-frontend="isFrontend"
      :project-id="project?.id"
      :component-id="comp?.id"
      :expanded-requests="expandedRequests"
      :copied-path="copiedPath"
      @toggle-expand="toggleExpandRequests"
      @copy-path="copyRoutePath"
      @edit="openEditModal"
      @delete="handleDelete"
    />

    <AppListPager
      v-if="filteredRoutes.length > 0"
      v-model:page="routePaging.page.value"
      v-model:per-page="routePaging.perPage.value"
      :total="routePaging.total.value"
      :per-page-options="routePaging.perPageOptions"
      :range-start="routePaging.rangeStart.value"
      :range-end="routePaging.rangeEnd.value"
      :noun="isFrontend ? 'halaman' : 'rute'"
      scroll-target="route-catalog-list"
    />

    <ProjectsDetailRouteCatalogEditModal
      ref="editModal"
      :component-id="comp?.id"
      :is-frontend="isFrontend"
      :loading="loading"
      @saved="onRouteSaved"
    />

    <!-- Modal konfirmasi hapus endpoint rute -->
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
