<script setup lang="ts">
import type { Todo, GuidelineCategory } from '~/types'

definePageMeta({
  fluid: true
})

const { getTodos, getGuidelines, lastTodoStatusCounts } = useApi()

const {
  activeProjectId,
  activeProject,
  projects,
  fetchProjects,
  selectActiveProject
} = useActiveProject()

const { initing: initingProject, initProject } = useInitVibeasik()

const todos = ref<Todo[]>([])
const guidelines = ref<GuidelineCategory[]>([])
const loading = ref(true)

// Todos yang murni milik Proyek Aktif saat ini
const projectTodos = computed(() => {
  if (!activeProjectId.value) return todos.value
  return todos.value.filter(t => t.project_id === activeProjectId.value)
})

// Todo gabungan (Aturan 43): sub todo tampil di dalam kartu induknya, bukan sebagai kartu sendiri.
const projectTodoIds = computed(() => new Set(projectTodos.value.map(t => t.id)))
const isSubTodo = (t: Todo) => Boolean(t.parent_id && projectTodoIds.value.has(t.parent_id))
const subTodosByParent = computed(() => {
  const map: Record<number, Todo[]> = {}
  for (const t of projectTodos.value) {
    if (isSubTodo(t)) (map[t.parent_id!] ??= []).push(t)
  }
  return map
})
// Satuan kerja yang dihitung di rekap: sub todo sudah terwakili todo gabungannya.
const workTodos = computed(() => projectTodos.value.filter(t => !isSubTodo(t)))
const mergeOpen = ref(false)
const promptsOpen = ref(false)

const pendingAISuggestionsCount = computed(() => {
  return projectTodos.value.filter(t => t.status === 'suggested').length
})

const remainingTasksCount = computed(() => {
  return workTodos.value.filter(t => t.status !== 'completed' && t.status !== 'rejected').length
})

// Modals
const guidelinesDrawerOpen = ref(false)
const openGuidelinesDrawer = () => {
  guidelinesDrawerOpen.value = true
}

let loadingDataInProgress = false
const loadData = async () => {
  if (loadingDataInProgress) return
  loadingDataInProgress = true
  loading.value = true
  try {
    await fetchProjects()

    // Jika belum ada activeProjectId terpilih, otomatis pilih proyek pertama yang tersedia
    if (!activeProjectId.value && projects.value.length > 0 && projects.value[0]) {
      selectActiveProject(projects.value[0].id)
    }

    const targetProjectId = activeProjectId.value || (projects.value.length > 0 ? projects.value[0]?.id : undefined)

    const [tList, gList] = await Promise.all([
      // Batas maksimum server (200); todo terbuka selalu didahulukan, sisanya todo selesai terbaru.
      getTodos(targetProjectId ? { project_id: targetProjectId, limit: 200 } : { limit: 200 }),
      guidelines.value.length === 0 ? getGuidelines() : Promise.resolve(guidelines.value)
    ])
    todos.value = tList || []
    if (gList) {
      guidelines.value = gList
    }
  } catch (err) {
    console.error('Failed to load todos data:', err)
  } finally {
    loading.value = false
    loadingDataInProgress = false
  }
}

// Filter, urutan & paginasi (state di composable)
const {
  selectedLevel, selectedCategory, selectedCreator, selectedTag, selectedStatus, selectedOrderBy, searchQuery,
  setStatus, filteredTodos, page, perPage, perPageOptions, totalPages, paginatedTodos
} = useTodoFilters(workTodos, activeProjectId)

// Orkestrator: kirim todo ke agent coding di laptop & tampilkan run yang sedang berjalan
const dispatchTodo = ref<Todo | null>(null)
const dispatchOpen = ref(false)
const handleDispatch = (todo: Todo) => {
  dispatchTodo.value = todo
  dispatchOpen.value = true
}

// Autopilot: kerjakan todo proyek aktif satu per satu secara otomatis
const { autopilotEnabled, autopilotAgent, activeRunningTodo, agentRunsByTodo, refreshAgentRuns, handleToggleAutopilot } = useTodoAutopilot({
  projectTodos,
  filteredTodos,
  activeProject,
  loadData
})

// "Suruh Agent Otomatis": semua todo pending yang tampil (sesuai filter) dan belum dipegang agent
const batchOpen = ref(false)
const batchTodos = computed(() => filteredTodos.value.filter(t => t.status === 'pending' && !agentRunsByTodo.value[t.id]))

// Re-fetch saat activeProjectId berubah (misal dipilih dari Navbar atau dropdown proyek)
watch(activeProjectId, async () => {
  await loadData()
})

onMounted(() => {
  loadData()
})

const handleSwitchProject = (newIdStr: string) => {
  const pId = newIdStr ? Number(newIdStr) : null
  selectActiveProject(pId)
}

const aiNotice = ref<{ type: 'success' | 'error' | 'info', message: string } | null>(null)
const dismissAiNotice = () => {
  aiNotice.value = null
}

// Pemantauan perencanaan produk otomatis: saat selesai di antrean background, data dimuat ulang.
useJobCompleted(async (done) => {
  if (done.type === 'pm_plan' && activeProjectId.value && done.project_id === activeProjectId.value) {
    await loadData()
    if (selectedStatus.value !== 'all' && selectedStatus.value !== 'suggested' && selectedStatus.value !== 'uncompleted') {
      selectedStatus.value = 'uncompleted'
    }
    aiNotice.value = {
      type: 'success',
      message: `Pegawai perencana produk selesai merumuskan agenda untuk ${done.project_name}. ${done.message.replace(/^Selesai:\s*/, '')} — keputusan lengkap ada di Kantor Agent.`
    }
  }
})

const { handleApprove, handleReject, handleSplit, handleUpdateStatus, handleDelete } = useTodoActions(loadData)
</script>

<template>
  <div class="space-y-6 w-full max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 pt-6 pb-16">
    <TodosTodoPageHeader
      v-model:autopilot-agent="autopilotAgent"
      :active-project="activeProject"
      :active-project-id="activeProjectId"
      :loading="loading"
      :initing-project="initingProject"
      :autopilot-enabled="autopilotEnabled"
      :batch-count="batchTodos.length"
      @open-guidelines="openGuidelinesDrawer"
      @refresh="loadData(); refreshAgentRuns()"
      @init-project="activeProjectId && initProject(activeProjectId, activeProject?.name)"
      @toggle-autopilot="handleToggleAutopilot"
      @open-merge="mergeOpen = true"
      @open-batch="batchOpen = true"
      @open-prompts="promptsOpen = true"
    />


    <TodosTodoLiveBanners
      :autopilot-enabled="autopilotEnabled"
      :autopilot-agent="autopilotAgent"
      :active-running-todo="activeRunningTodo"
      :agent-runs-by-todo="agentRunsByTodo"
      :ai-notice="aiNotice"
      @toggle-autopilot="handleToggleAutopilot"
      @dismiss-ai-notice="dismissAiNotice"
    />

    <!-- Stats Header Cards (Rekap Atas: Sisa tugas belum selesai & status rincian proyek aktif) -->
    <TodosTodoStatsHeader
      :todos="workTodos"
      :counts="lastTodoStatusCounts"
      :project-name="activeProject?.name"
      @filter-status="setStatus"
    />

    <!-- AI Suggestion Notice Banner -->
    <div
      v-if="pendingAISuggestionsCount > 0"
      class="p-4 rounded-xl border border-amber-300 dark:border-amber-900/60 bg-amber-50/70 dark:bg-amber-950/20 text-xs flex flex-col sm:flex-row sm:items-center justify-between gap-3"
    >
      <div class="flex items-center gap-2.5">
        <UIcon name="i-lucide-bot" class="w-5 h-5 text-amber-600 dark:text-amber-400 shrink-0" />
        <div>
          <span class="font-bold text-amber-900 dark:text-amber-200">
            Ada {{ pendingAISuggestionsCount }} Rekomendasi Agenda dari AI Agent
          </span>
          <p class="text-amber-800 dark:text-amber-300 text-xs mt-0.5">
            Dihasilkan otomatis berdasarkan evaluasi rubrik khusus proyek {{ activeProject?.name || '' }}. Silakan setujui untuk dieksekusi atau tolak.
          </p>
        </div>
      </div>

      <UButton
        size="xs"
        color="warning"
        variant="outline"
        label="Lihat Semua Saran AI"
        @click="setStatus('suggested')"
      />
    </div>

    <TodosTodoFilterToolbar
      v-model:level="selectedLevel"
      v-model:category="selectedCategory"
      v-model:creator="selectedCreator"
      v-model:tag="selectedTag"
      v-model:status="selectedStatus"
      v-model:order-by="selectedOrderBy"
      v-model:search="searchQuery"
      :filtered-count="filteredTodos.length"
      :total-count="lastTodoStatusCounts ? Object.values(lastTodoStatusCounts).reduce((a, b) => a + b, 0) : workTodos.length"
    />

    <!-- Loading State Skeleton -->
    <div v-if="loading" class="space-y-3">
      <div
        v-for="i in 3"
        :key="i"
        class="p-4 rounded-xl border border-neutral-200/80 dark:border-white/10 bg-white/60 dark:bg-[#04111f] animate-pulse space-y-3"
      >
        <div class="flex items-center justify-between">
          <div class="h-4 w-48 bg-neutral-200 dark:bg-neutral-800 rounded" />
          <div class="h-4 w-20 bg-neutral-200 dark:bg-neutral-800 rounded-full" />
        </div>
        <div class="h-3 w-3/4 bg-neutral-100 dark:bg-neutral-800/60 rounded" />
        <div class="h-3 w-1/2 bg-neutral-100 dark:bg-neutral-800/60 rounded" />
      </div>
    </div>

    <!-- Todos Card List & Pagination -->
    <TodosTodoListSection
      v-else-if="filteredTodos.length > 0"
      v-model:page="page"
      v-model:per-page="perPage"
      :filtered-todos="filteredTodos"
      :paginated-todos="paginatedTodos"
      :work-todo-count="workTodos.length"
      :agent-runs-by-todo="agentRunsByTodo"
      :sub-todos-by-parent="subTodosByParent"
      :per-page-options="perPageOptions"
      :total-pages="totalPages"
      @approve="handleApprove"
      @reject="handleReject"
      @update-status="handleUpdateStatus"
      @delete="handleDelete"
      @dispatch="handleDispatch"
      @split="handleSplit"
    />

    <!-- Empty State -->
    <div
      v-else
      class="p-12 text-center rounded-xl border border-dashed border-neutral-300 dark:border-white/10 bg-white/40 dark:bg-[#04111f] space-y-3"
    >
      <UIcon name="i-lucide-clipboard-check" class="w-10 h-10 mx-auto text-sky-500" />
      <h3 class="text-sm font-bold text-neutral-800 dark:text-neutral-200">
        {{ selectedStatus === 'uncompleted' ? 'Hebat! Semua Agenda Belum Selesai Sudah Tuntas!' : 'Belum Ada Agenda yang Sesuai Filter' }}
      </h3>
      <p class="text-xs text-neutral-500 max-w-md mx-auto">
        <span v-if="selectedStatus === 'uncompleted'">
          Tidak ada tugas tersisa yang belum diselesaikan pada proyek <strong>{{ activeProject?.name || '' }}</strong>. Klik "Semua Status" untuk melihat riwayat atau tunggu pegawai perencana produk merumuskan agenda berikutnya.
        </span>
        <span v-else>
          Tidak ditemukan agenda kerja pada kriteria filter saat ini. Seluruh perencanaan dan perbaikan produk dipikirkan dan dikelola oleh pegawai AI perencana produk.
        </span>
      </p>
      <div v-if="selectedStatus === 'uncompleted'" class="flex justify-center gap-2 pt-2 flex-wrap">
        <UButton
          size="xs"
          color="neutral"
          variant="outline"
          label="Lihat Semua Status (Termasuk Selesai)"
          @click="setStatus('all')"
        />
      </div>
    </div>

    <!-- Modals -->
    <TodosGuidelinesDrawer
      v-model:open="guidelinesDrawerOpen"
      :categories="guidelines"
    />

    <OrchestratorBatchDispatchModal
      v-model:open="batchOpen"
      :todos="batchTodos"
      @dispatched="refreshAgentRuns(); loadData()"
    />

    <TodosTodoMergeModal
      v-model:open="mergeOpen"
      :project-id="activeProjectId ?? undefined"
      @merged="loadData()"
    />

    <TodosTodoAgentPromptModal
      v-model:open="promptsOpen"
      :project-id="activeProjectId ?? undefined"
      :project-name="activeProject?.name"
    />

    <OrchestratorDispatchModal
      v-model:open="dispatchOpen"
      :todo="dispatchTodo"
      @dispatched="refreshAgentRuns(); loadData()"
    />
  </div>
</template>
