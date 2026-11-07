<script setup lang="ts">
import type { Todo, GuidelineCategory, AgentDispatchRequest, TodoStatus } from '~/types'
import { impactPerEffort } from '~/composables/useTodoImpact'
import { parseTodoTags } from '~/utils/todoTags'

definePageMeta({
  fluid: true
})

const {
  getTodos,
  getGuidelines,
  approveTodo,
  rejectTodo,
  updateTodo,
  deleteTodo,
  splitTodoGroup,
  dispatchAgentRun
} = useApi()

const toast = useToast()

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

// Filters & Ordering
const selectedLevel = ref<string>('all')
const selectedCategory = ref<string>('all')
const selectedCreator = ref<string>('all')
const selectedTag = ref<string>('all')
// Default filter status: 'uncompleted' untuk langsung menghide yang selesai dan memperlihatkan sisa tugas
const selectedStatus = ref<string>('uncompleted')
const selectedOrderBy = ref<'priority_desc' | 'priority_asc' | 'impact_desc' | 'value_desc' | 'created_desc' | 'created_asc' | 'title_asc' | 'status'>('priority_desc')
const searchQuery = ref('')

// Setter eksplisit agar tidak melakukan mutasi assignment langsung di dalam template inline event handler
const setLevel = (level: string) => {
  selectedLevel.value = level
}

const setCategory = (category: string) => {
  selectedCategory.value = category
}

const setStatus = (status: string) => {
  selectedStatus.value = status
}

const toggleUncompletedStatus = () => {
  selectedStatus.value = selectedStatus.value === 'uncompleted' ? 'all' : 'uncompleted'
}

// Bobot prioritas untuk ordering
const priorityWeights: Record<string, number> = {
  critical: 4,
  high: 3,
  medium: 2,
  low: 1
}

// Filtered and Sorted list
const filteredTodos = computed(() => {
  const filtered = workTodos.value.filter((t) => {
    // Level filter (khusus proyek aktif: project bundle atau component)
    if (selectedLevel.value !== 'all' && t.level !== selectedLevel.value) {
      return false
    }

    // Category filter
    if (selectedCategory.value !== 'all') {
      const itemCat = t.category || 'general'
      if (itemCat !== selectedCategory.value) {
        return false
      }
    }

    // Tag fokus filter
    if (selectedTag.value !== 'all') {
      const tags = parseTodoTags(t.tags)
      if (selectedTag.value === 'untagged' ? tags.length > 0 : !tags.includes(selectedTag.value)) {
        return false
      }
    }

    // Creator / Origin filter
    if (selectedCreator.value !== 'all') {
      const src = (t.source || '').toLowerCase()
      const creator = (t.creator_name || '').toLowerCase()
      const cType = (t.creator_type || '').toLowerCase()
      const isExec = src === 'autonomous_executive' || src === 'executive_chat' || creator.includes('eksekutif') || cType === 'executive'
      const isAudit = src === 'diagnostic_audit' || src === 'audit' || Boolean((t as any).audit_key) || Boolean(t.guideline_ref)
      const isPm = src === 'pm_agent' || src === 'planner' || creator.includes('project-manager') || creator.includes('pm')
      const isSugg = t.is_ai_suggestion || creator.includes('advisor') || src === 'ai_template' || src === 'template'
      const isAgent = (cType === 'ai_agent' || creator.includes('claude') || creator.includes('antigravity') || creator.includes('opencode')) && !isExec && !isAudit && !isPm && !isSugg
      const isHuman = cType === 'human' || src === 'manual'

      if (selectedCreator.value === 'executive' && !isExec) return false
      if (selectedCreator.value === 'audit' && !isAudit) return false
      if (selectedCreator.value === 'pm' && !isPm) return false
      if (selectedCreator.value === 'ai_suggestion' && !isSugg) return false
      if (selectedCreator.value === 'agent' && !isAgent) return false
      if (selectedCreator.value === 'human' && (!isHuman || isExec || isAudit || isPm || isSugg || isAgent)) return false

      // Backward compatibility with legacy 'ai'
      if (selectedCreator.value === 'ai' && !t.is_ai_suggestion && t.creator_type !== 'ai_agent') return false
    }

    // Status filter:
    // Jika 'uncompleted', hide semua yang statusnya 'completed' dan 'rejected' sehingga terlihat sisa tugas yang belum selesai
    if (selectedStatus.value === 'uncompleted') {
      if (t.status === 'completed' || t.status === 'rejected') {
        return false
      }
    } else if (selectedStatus.value !== 'all' && t.status !== selectedStatus.value) {
      return false
    }

    // Search query
    if (searchQuery.value.trim()) {
      const q = searchQuery.value.toLowerCase()
      const matchTitle = t.title.toLowerCase().includes(q)
      const matchDesc = (t.description || '').toLowerCase().includes(q)
      const matchGuideline = (t.guideline_ref || '').toLowerCase().includes(q)
      const matchCat = (t.category || '').toLowerCase().includes(q)
      const matchApps = (t.target_apps || '').toLowerCase().includes(q)
      if (!matchTitle && !matchDesc && !matchGuideline && !matchCat && !matchApps) return false
    }

    return true
  })

  // Order By Logic
  return filtered.sort((a, b) => {
    if (selectedOrderBy.value === 'priority_desc') {
      const wA = priorityWeights[a.priority] || 2
      const wB = priorityWeights[b.priority] || 2
      if (wB !== wA) return wB - wA
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    }
    if (selectedOrderBy.value === 'priority_asc') {
      const wA = priorityWeights[a.priority] || 2
      const wB = priorityWeights[b.priority] || 2
      if (wA !== wB) return wA - wB
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    }
    if (selectedOrderBy.value === 'impact_desc') {
      return (b.impact_score || 0) - (a.impact_score || 0)
    }
    if (selectedOrderBy.value === 'value_desc') {
      return impactPerEffort(b) - impactPerEffort(a)
    }
    if (selectedOrderBy.value === 'created_desc') {
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    }
    if (selectedOrderBy.value === 'created_asc') {
      return new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
    }
    if (selectedOrderBy.value === 'title_asc') {
      return (a.title || '').localeCompare(b.title || '')
    }
    if (selectedOrderBy.value === 'status') {
      const statusWeights: Record<string, number> = {
        in_progress: 1,
        pending: 2,
        suggested: 3,
        completed: 4,
        rejected: 5
      }
      const sA = statusWeights[a.status] || 9
      const sB = statusWeights[b.status] || 9
      return sA - sB
    }
    return 0
  })
})

// Pagination
const page = ref(1)
const perPage = ref(20)
const perPageOptions = [10, 20, 50, 100]

const totalPages = computed(() => {
  return Math.ceil(filteredTodos.value.length / perPage.value) || 1
})

const paginatedTodos = computed(() => {
  const start = (page.value - 1) * perPage.value
  return filteredTodos.value.slice(start, start + perPage.value)
})

const todosContainerRef = ref<HTMLElement | null>(null)
const setPage = (newPage: number) => {
  page.value = newPage
  if (import.meta.client && todosContainerRef.value) {
    todosContainerRef.value.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}

// Reset page ke 1 saat filter atau parameter berganti
watch(
  [selectedLevel, selectedCategory, selectedCreator, selectedStatus, selectedOrderBy, searchQuery, activeProjectId, perPage],
  () => {
    page.value = 1
  }
)

watch(() => filteredTodos.value.length, () => {
  if (page.value > totalPages.value) {
    page.value = totalPages.value
  }
})

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
      getTodos(targetProjectId ? { project_id: targetProjectId } : undefined),
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

// Orkestrator: kirim todo ke agent coding di laptop & tampilkan run yang sedang berjalan
const dispatchTodo = ref<Todo | null>(null)
const dispatchOpen = ref(false)
const { byTodo: agentRunsByTodo, refresh: refreshAgentRuns } = useTodoAgentRuns(async () => {
  await loadData()
  if (autopilotEnabled.value) {
    await runNextAutopilot()
  }
})
const handleDispatch = (todo: Todo) => {
  dispatchTodo.value = todo
  dispatchOpen.value = true
}
// "Suruh Agent Otomatis": semua todo pending yang tampil (sesuai filter) dan belum dipegang agent
const batchOpen = ref(false)
const openBatchModal = () => {
  batchOpen.value = true
}
const batchTodos = computed(() => filteredTodos.value.filter(t => t.status === 'pending' && !agentRunsByTodo.value[t.id]))

// Autopilot: Kerjakan todo proyek aktif satu per satu secara otomatis
const autopilotEnabled = ref(false)
const autopilotAgent = ref<'agy' | 'cline' | 'opencode' | 'auto'>('agy') // default: agy (Google Antigravity CLI dengan 7 akun aktif & terbukti mengubah kode)
const autopilotRunning = ref(false)
const autopilotCurrentTodoId = ref<number | null>(null)
const autopilotProcessedCount = ref(0)

const activeRunningTodo = computed(() => {
  return projectTodos.value.find((t) => {
    const run = agentRunsByTodo.value[t.id]
    return run && (run.status === 'running' || run.status === 'queued')
  })
})

const runNextAutopilot = async () => {
  if (!autopilotEnabled.value || autopilotRunning.value) return

  // Tunggu jika sedang ada tugas yang berjalan
  if (activeRunningTodo.value) {
    autopilotCurrentTodoId.value = activeRunningTodo.value.id
    return
  }

  // Muat data todo terbaru agar status yang baru selesai tidak terbaca pending
  await loadData()

  const currentlyRunning = activeRunningTodo.value as Todo | undefined
  if (currentlyRunning) {
    autopilotCurrentTodoId.value = currentlyRunning.id
    return
  }

  // Ambil todo pending berikutnya (prioritas tertinggi dulu)
  const candidate = filteredTodos.value.find(t => t.status === 'pending' && !agentRunsByTodo.value[t.id])
    || projectTodos.value.find(t => t.status === 'pending' && !agentRunsByTodo.value[t.id])

  if (!candidate) {
    autopilotEnabled.value = false
    autopilotCurrentTodoId.value = null
    toast.add({
      title: 'Autopilot Selesai',
      description: `Seluruh agenda pada proyek ${activeProject.value?.name || ''} telah selesai dikerjakan!`,
      icon: 'i-lucide-check-circle-2',
      color: 'success'
    })
    return
  }

  autopilotRunning.value = true
  autopilotCurrentTodoId.value = candidate.id

  try {
    const payload: AgentDispatchRequest = {
      todo_id: candidate.id,
      router: 'rules'
    }

    if (autopilotAgent.value === 'agy') {
      payload.pool_agent_key = 'agy'
    } else if (autopilotAgent.value === 'cline') {
      payload.pool_agent_key = 'cline'
    } else if (autopilotAgent.value === 'opencode') {
      payload.pool_agent_key = 'opencode'
    }

    const res = await dispatchAgentRun(payload)
    autopilotProcessedCount.value++
    toast.add({
      title: `Autopilot: Menjalankan Agenda #${candidate.id}`,
      description: `"${candidate.title.slice(0, 45)}..." dikerjakan oleh ${res?.run?.profile_name || autopilotAgent.value}`,
      icon: 'i-lucide-bot',
      color: 'info'
    })
    await refreshAgentRuns()
  } catch (err: unknown) {
    const e = err as { data?: { error?: string }, message?: string }
    console.error('Gagal menjalankan autopilot todo:', err)
    toast.add({
      title: 'Autopilot Dijeda',
      description: `Gagal mengirim todo #${candidate.id}: ${e?.data?.error || e?.message || 'Error'}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
    autopilotEnabled.value = false
    autopilotCurrentTodoId.value = null
  } finally {
    autopilotRunning.value = false
  }
}

// Watcher ketika tugas selesai dijalankan
watch(activeRunningTodo, (cur, prev) => {
  if (autopilotEnabled.value && !cur && prev) {
    setTimeout(async () => {
      if (autopilotEnabled.value) {
        await runNextAutopilot()
      }
    }, 1500)
  }
})

const handleToggleAutopilot = async (val: boolean) => {
  autopilotEnabled.value = val
  if (val) {
    toast.add({
      title: 'Autopilot Aktif',
      description: `Agenda proyek aktif akan dikerjakan satu per satu via ${autopilotAgent.value === 'opencode' ? 'OpenCode' : autopilotAgent.value === 'agy' ? 'Antigravity' : 'Auto'}.`,
      icon: 'i-lucide-play',
      color: 'success'
    })
    await runNextAutopilot()
  } else {
    autopilotCurrentTodoId.value = null
    toast.add({
      title: 'Autopilot Dimatikan',
      description: 'Pengerjaan otomatis dihentikan. Tugas yang sedang aktif berjalan tetap diselesaikan.',
      icon: 'i-lucide-square',
      color: 'neutral'
    })
  }
}

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

const handleApprove = async (id: number) => {
  try {
    await approveTodo(id)
    await loadData()
  } catch (err) {
    console.error('Failed to approve todo:', err)
  }
}

const handleReject = async (id: number) => {
  try {
    await rejectTodo(id)
    await loadData()
  } catch (err) {
    console.error('Failed to reject todo:', err)
  }
}

const handleSplit = async (id: number) => {
  try {
    const res = await splitTodoGroup(id)
    toast.add({ title: 'Todo gabungan dipecah', description: `${res.todo_ids.length} sub todo kembali menjadi todo biasa.`, icon: 'i-lucide-split', color: 'success' })
    await loadData()
  } catch (err: unknown) {
    const e = err as { data?: { error?: string }, message?: string }
    toast.add({ title: 'Gagal memecah todo gabungan', description: e?.data?.error || e?.message || 'Error', icon: 'i-lucide-alert-circle', color: 'error' })
  }
}

const handleUpdateStatus = async (id: number, status: string) => {
  try {
    await updateTodo(id, { status: status as TodoStatus })
    await loadData()
  } catch (err) {
    console.error('Failed to update status:', err)
  }
}

const handleDelete = async (id: number) => {
  try {
    await deleteTodo(id)
    toast.add({
      title: 'Agenda Dihapus',
      description: 'Agenda berhasil dihapus dari daftar.',
      icon: 'i-lucide-trash-2',
      color: 'success'
    })
    await loadData()
  } catch (err: unknown) {
    const e = err as { data?: { error?: string }, message?: string }
    console.error('Failed to delete todo:', err)
    toast.add({
      title: 'Gagal Menghapus Agenda',
      description: e?.data?.error || e?.message || 'Terjadi kesalahan sistem saat menghapus agenda',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  }
}
</script>

<template>
  <div class="space-y-6 w-full px-4 sm:px-6 lg:px-8 pt-6 pb-16">
    <!-- Page Header -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-3 border-b border-neutral-200 dark:border-white/10">
      <div class="min-w-0">
        <div class="flex items-center gap-2.5 flex-wrap">
          <h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white flex items-center gap-2">
            <UIcon name="i-lucide-clipboard-check" class="w-6 h-6 text-violet-500 shrink-0" />
            <span>Perencanaan & Agenda Bertingkat</span>
          </h1>
          <UBadge v-if="activeProject" size="sm" color="primary" variant="subtle">
            {{ activeProject.name }}
          </UBadge>
        </div>
        <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1">
          Fokus perencanaan teknis proyek aktif (Bundle & Komponen/Module) dengan rubrik terukur, pelacak tugas sisa, dan saran AI Agent.
        </p>
      </div>

      <!-- Action Buttons Row: Rata Kanan -->
      <div class="flex items-center justify-end flex-wrap gap-2.5 shrink-0 self-end md:self-center ml-auto">
        <UButton
          icon="i-lucide-book-open"
          color="neutral"
          variant="outline"
          size="sm"
          title="Buka daftar buku panduan dan rubrik penilaian AI"
          @click="openGuidelinesDrawer"
        >
          Lihat Patokan
        </UButton>

        <UButton
          icon="i-lucide-refresh-cw"
          color="neutral"
          variant="outline"
          size="sm"
          :loading="loading"
          title="Muat ulang data todo proyek aktif"
          @click="loadData(); refreshAgentRuns()"
        >
          Refresh
        </UButton>

        <UButton
          v-if="activeProjectId"
          icon="i-lucide-sparkles"
          color="info"
          variant="outline"
          size="sm"
          :loading="initingProject"
          title="Siapkan proyek untuk AI agent: tulis .vibeasik.json (project ID & TODO) dan peta kode .vibeasik/CODEMAP.md yang diperbarui otomatis saat kode berubah."
          @click="initProject(activeProjectId, activeProject?.name)"
        >
          Init vibeasik
        </UButton>

        <!-- Toggle Kerjakan Otomatis (Autopilot) -->
        <div class="flex items-center gap-1 p-0.5 rounded-xl bg-neutral-100 dark:bg-neutral-800/80 border border-neutral-200 dark:border-white/10 shadow-2xs">
          <div class="flex items-center gap-2 px-2.5 py-1">
            <span class="relative flex h-2 w-2">
              <span v-if="autopilotEnabled" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
              <span class="relative inline-flex rounded-full h-2 w-2" :class="autopilotEnabled ? 'bg-emerald-500' : 'bg-neutral-400'" />
            </span>
            <span
              class="text-xs font-semibold text-neutral-800 dark:text-neutral-200 select-none cursor-pointer"
              @click="handleToggleAutopilot(!autopilotEnabled)"
            >
              Kerjakan Otomatis
            </span>
            <USwitch
              :model-value="autopilotEnabled"
              color="success"
              size="sm"
              title="Aktifkan untuk mengerjakan todo proyek aktif satu per satu secara otomatis"
              @update:model-value="handleToggleAutopilot"
            />
          </div>

          <!-- Quick Config Popover for Autopilot -->
          <UPopover mode="click" :content="{ side: 'bottom', align: 'end' }">
            <UButton
              icon="i-lucide-sliders-horizontal"
              color="neutral"
              variant="ghost"
              size="xs"
              title="Pengaturan CLI & Model Pengerjaan Otomatis"
            />
            <template #content>
              <div class="p-3.5 w-72 space-y-3 text-xs">
                <div class="flex items-center justify-between pb-1.5 border-b border-neutral-100 dark:border-neutral-800">
                  <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1.5">
                    <UIcon name="i-lucide-bot" class="w-4 h-4 text-sky-500" />
                    <span>Konfigurasi Autopilot</span>
                  </span>
                  <UBadge size="xs" :color="autopilotEnabled ? 'success' : 'neutral'" variant="subtle">
                    {{ autopilotEnabled ? 'Aktif' : 'Standby' }}
                  </UBadge>
                </div>

                <div class="space-y-1">
                  <label class="text-[11px] font-semibold text-neutral-600 dark:text-neutral-400">Pilihan CLI Coding Agent:</label>
                  <select
                    v-model="autopilotAgent"
                    class="w-full px-2 py-1.5 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 text-xs focus:ring-1 focus:ring-sky-500 cursor-pointer"
                  >
                    <option value="agy">
                      ⭐ Antigravity CLI (7 Akun Aktif, Terbukti Mengubah Kode) [Rekomendasi]
                    </option>
                    <option value="cline">
                      🚀 Cline CLI (Otonom, gunakan provider di cline auth)
                    </option>
                    <option value="opencode">
                      ⚡ OpenCode (Provider Kustom / API Key)
                    </option>
                    <option value="auto">
                      🌐 Otomatis (Router Terbaik)
                    </option>
                  </select>
                </div>

                <div class="p-2 rounded-lg bg-sky-50/80 dark:bg-sky-950/30 border border-sky-200/80 dark:border-sky-900/40 text-[11px] text-sky-900 dark:text-sky-200 space-y-1">
                  <p class="font-semibold flex items-center gap-1">
                    <UIcon name="i-lucide-check-circle" class="w-3.5 h-3.5 text-sky-600 dark:text-sky-400" />
                    <span>Status Eksekusi Coding:</span>
                  </p>
                  <p class="text-neutral-600 dark:text-neutral-400 leading-tight">
                    • <strong>Antigravity CLI:</strong> Mengubah kode langsung via Gemini Flash, rotasi 7 akun tanpa limit.<br>
                    • <strong>Cline CLI:</strong> Tool calling otonom via CLI Cline.<br>
                    • <strong>OpenCode:</strong> Memerlukan provider API valid (bukan mock/troll).
                  </p>
                </div>
              </div>
            </template>
          </UPopover>
        </div>

        <UButton
          v-if="activeProjectId"
          icon="i-lucide-layers"
          color="neutral"
          variant="outline"
          size="sm"
          title="Lebur todo serupa (rule/berkas sama atau isi mirip) menjadi satu todo gabungan yang dikerjakan satu agent sekaligus"
          @click="mergeOpen = true"
        >
          Lebur Serupa
        </UButton>

        <UButton
          icon="i-lucide-bot"
          color="neutral"
          variant="outline"
          size="sm"
          :disabled="!batchTodos.length"
          title="Kirim semua todo pending yang tampil ke agent coding; agent & effort dipilih sesuai kerumitan tiap todo"
          @click="openBatchModal"
        >
          Batch ({{ batchTodos.length }})
        </UButton>
      </div>
    </div>

    <!-- Active Project Banner & Selector -->
    <div class="p-3.5 rounded-xl border border-blue-200 dark:border-blue-900/60 bg-blue-50/60 dark:bg-blue-950/20 text-xs flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-2xs">
      <div class="flex items-center gap-2.5 min-w-0">
        <UIcon name="i-lucide-folder-git-2" class="w-5 h-5 text-blue-600 dark:text-blue-400 shrink-0" />
        <div class="min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <span class="text-neutral-500 dark:text-neutral-400 font-medium">Proyek Aktif:</span>
            <span class="font-bold text-sm text-blue-950 dark:text-blue-100 truncate">
              {{ activeProject ? activeProject.name : 'Belum Ada Proyek Terpilih' }}
            </span>
            <span
              v-if="activeProject"
              class="px-2 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900/50 text-blue-800 dark:text-blue-300 text-[10px] font-semibold border border-blue-200 dark:border-blue-800"
            >
              {{ activeProject.components?.length || 0 }} Komponen
            </span>
            <span
              v-if="remainingTasksCount > 0"
              class="px-2 py-0.5 rounded-full bg-amber-100 dark:bg-amber-900/50 text-amber-800 dark:text-amber-300 text-[10px] font-semibold border border-amber-200 dark:border-amber-800"
            >
              {{ remainingTasksCount }} tugas perlu diselesaikan
            </span>
          </div>
          <p v-if="activeProject && activeProject.components && activeProject.components.length > 0" class="text-[11px] text-neutral-500 dark:text-neutral-400 mt-0.5 truncate">
            Aplikasi: {{ activeProject.components.map(c => c.name).join(', ') }}
          </p>
        </div>
      </div>

      <!-- Quick Switch Active Project on this page -->
      <div class="flex items-center gap-2 shrink-0">
        <label for="project-switcher" class="text-[11px] text-neutral-500 dark:text-neutral-400 font-medium hidden sm:inline">
          Ganti:
        </label>
        <select
          id="project-switcher"
          :value="activeProjectId ?? ''"
          class="h-8 pl-2.5 pr-7 py-0 text-xs font-mono font-medium rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 hover:border-neutral-300 dark:hover:border-neutral-600 focus:outline-none focus:ring-1 focus:ring-blue-500 cursor-pointer max-w-[200px] truncate"
          title="Pilih proyek aktif untuk perencanaan agenda"
          @change="(e) => handleSwitchProject((e.target as HTMLSelectElement).value)"
        >
          <option value="" disabled>
            -- Pilih Proyek Aktif --
          </option>
          <option v-for="p in projects" :key="p.id" :value="p.id">
            {{ p.name }} ({{ p.components?.length || 0 }} apps)
          </option>
        </select>
      </div>
    </div>

    <!-- Autopilot Live Progress Banner -->
    <div
      v-if="autopilotEnabled"
      class="p-3.5 rounded-xl border border-emerald-300 dark:border-emerald-800/80 bg-linear-to-r from-emerald-50/90 via-teal-50/70 to-emerald-50/90 dark:from-emerald-950/40 dark:via-teal-950/30 dark:to-emerald-950/40 text-xs shadow-xs transition-all"
    >
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="flex items-center gap-3">
          <div class="relative size-8 rounded-lg bg-emerald-600 text-white flex items-center justify-center shrink-0 shadow-xs">
            <UIcon name="i-lucide-bot" class="w-4 h-4" />
            <span class="absolute -top-1 -right-1 flex h-2.5 w-2.5">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
              <span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-600 border border-white dark:border-neutral-900" />
            </span>
          </div>
          <div>
            <div class="flex items-center gap-2 flex-wrap">
              <span class="font-bold text-emerald-950 dark:text-emerald-100 text-sm">
                Pengerjaan Otomatis Sedang Berjalan (Autopilot)
              </span>
              <span class="px-2 py-0.5 rounded-full bg-emerald-100 dark:bg-emerald-900/60 text-emerald-800 dark:text-emerald-300 text-[10px] font-semibold border border-emerald-300 dark:border-emerald-800">
                1 Per 1 Berurutan
              </span>
              <span class="px-2 py-0.5 rounded-full bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 text-[10px] font-mono border border-neutral-300 dark:border-neutral-700">
                CLI: {{ autopilotAgent === 'opencode' ? 'OpenCode (Provider Kustom)' : autopilotAgent === 'agy' ? 'Antigravity (7 Akun)' : 'Router Otomatis' }}
              </span>
            </div>
            <p class="text-emerald-800 dark:text-emerald-300 text-[11px] mt-0.5">
              <span v-if="activeRunningTodo">
                Sedang mengeksekusi: <strong>#{{ activeRunningTodo.id }} - {{ activeRunningTodo.title }}</strong>
                <span v-if="agentRunsByTodo[activeRunningTodo.id]?.profile_name" class="font-mono ml-1 opacity-90">
                  [{{ agentRunsByTodo[activeRunningTodo.id]?.profile_name }}]
                </span>
              </span>
              <span v-else>
                Menyiapkan agenda berikutnya untuk dikerjakan...
              </span>
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2 shrink-0">
          <UButton
            size="xs"
            color="error"
            variant="solid"
            icon="i-lucide-square"
            label="Hentikan Autopilot"
            @click="handleToggleAutopilot(false)"
          />
        </div>
      </div>
    </div>

    <!-- AI Suggestion Result Alert Banner -->
    <div
      v-if="aiNotice"
      class="p-3.5 rounded-xl border text-xs flex items-center justify-between gap-3 shadow-2xs transition-all"
      :class="{
        'border-emerald-300 dark:border-emerald-800/80 bg-emerald-50/90 dark:bg-emerald-950/30 text-emerald-900 dark:text-emerald-200': aiNotice.type === 'success',
        'border-blue-300 dark:border-blue-800/80 bg-blue-50/90 dark:bg-blue-950/30 text-blue-900 dark:text-blue-200': aiNotice.type === 'info',
        'border-rose-300 dark:border-rose-800/80 bg-rose-50/90 dark:bg-rose-950/30 text-rose-900 dark:text-rose-200': aiNotice.type === 'error'
      }"
    >
      <div class="flex items-center gap-2.5">
        <UIcon
          :name="aiNotice.type === 'success' ? 'i-lucide-check-circle' : aiNotice.type === 'info' ? 'i-lucide-info' : 'i-lucide-alert-circle'"
          class="w-4 h-4 shrink-0"
          :class="{
            'text-emerald-600 dark:text-emerald-400': aiNotice.type === 'success',
            'text-blue-600 dark:text-blue-400': aiNotice.type === 'info',
            'text-rose-600 dark:text-rose-400': aiNotice.type === 'error'
          }"
        />
        <span class="font-medium">{{ aiNotice.message }}</span>
      </div>
      <button
        type="button"
        class="opacity-70 hover:opacity-100 font-bold px-1.5 py-0.5 rounded cursor-pointer"
        @click="dismissAiNotice"
      >
        ✕
      </button>
    </div>

    <!-- Stats Header Cards (Rekap Atas: Sisa tugas belum selesai & status rincian proyek aktif) -->
    <TodosTodoStatsHeader
      :todos="workTodos"
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
          <p class="text-amber-800 dark:text-amber-300 text-[11px] mt-0.5">
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

    <!-- Filters & Search Toolbar -->
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-2xs space-y-3">
      <!-- Row 1: Level Tabs (Dikhususkan untuk Proyek Aktif) -->
      <div class="flex flex-wrap items-center justify-between gap-2 border-b border-neutral-100 dark:border-white/10 pb-3">
        <div class="flex flex-wrap gap-1 text-xs">
          <button
            type="button"
            class="px-3 py-1.5 rounded-lg font-medium transition-colors cursor-pointer"
            :class="selectedLevel === 'all' ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900 shadow-xs' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="setLevel('all')"
          >
            Semua Tingkat Proyek
          </button>
          <button
            type="button"
            class="px-3 py-1.5 rounded-lg font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
            :class="selectedLevel === 'project' ? 'bg-blue-600 text-white shadow-xs' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="setLevel('project')"
          >
            <UIcon name="i-lucide-folder-kanban" class="w-3.5 h-3.5" />
            <span>Proyek (Bundle)</span>
          </button>
          <button
            type="button"
            class="px-3 py-1.5 rounded-lg font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
            :class="selectedLevel === 'component' ? 'bg-sky-600 text-white shadow-xs' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="setLevel('component')"
          >
            <UIcon name="i-lucide-layers" class="w-3.5 h-3.5" />
            <span>Komponen (Module)</span>
          </button>
        </div>

        <!-- Quick Toggle Sembunyikan Selesai & Counter -->
        <div class="flex items-center gap-3">
          <button
            type="button"
            class="px-2.5 py-1 rounded-lg text-xs font-medium transition-all flex items-center gap-1.5 border cursor-pointer"
            :class="selectedStatus === 'uncompleted'
              ? 'bg-amber-100 text-amber-900 border-amber-300 dark:bg-amber-950/60 dark:text-amber-200 dark:border-amber-800 shadow-2xs font-semibold'
              : 'border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800/80 text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-700'"
            title="Klik untuk menyembunyikan atau menampilkan tugas yang sudah selesai"
            @click="toggleUncompletedStatus"
          >
            <UIcon :name="selectedStatus === 'uncompleted' ? 'i-lucide-eye-off' : 'i-lucide-check-circle'" class="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" />
            <span>{{ selectedStatus === 'uncompleted' ? 'Selesai Tersembunyi' : 'Sembunyikan Selesai' }}</span>
          </button>

          <div class="text-xs font-mono text-neutral-400">
            Menampilkan {{ filteredTodos.length }} dari {{ workTodos.length }} agenda
          </div>
        </div>
      </div>

      <!-- Row 2: Category Filter Pills -->
      <div class="flex flex-wrap items-center gap-1.5 text-xs pb-1">
        <span class="text-neutral-400 text-[11px] font-semibold uppercase tracking-wider mr-1">Kategori:</span>
        <button
          type="button"
          class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors cursor-pointer"
          :class="selectedCategory === 'all' ? 'bg-neutral-800 text-white dark:bg-neutral-100 dark:text-neutral-900' : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:bg-neutral-200 dark:hover:bg-neutral-700'"
          @click="setCategory('all')"
        >
          Semua Kategori
        </button>
        <button
          type="button"
          class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
          :class="selectedCategory === 'security' ? 'bg-rose-600 text-white' : 'bg-rose-50 dark:bg-rose-950/40 text-rose-700 dark:text-rose-300 hover:bg-rose-100 dark:hover:bg-rose-900/50 border border-rose-200 dark:border-rose-900/60'"
          @click="setCategory('security')"
        >
          <UIcon name="i-lucide-shield-check" class="w-3 h-3" />
          <span>Keamanan</span>
        </button>
        <button
          type="button"
          class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
          :class="selectedCategory === 'ui_ux' ? 'bg-fuchsia-600 text-white' : 'bg-fuchsia-50 dark:bg-fuchsia-950/40 text-fuchsia-700 dark:text-fuchsia-300 hover:bg-fuchsia-100 dark:hover:bg-fuchsia-900/50 border border-fuchsia-200 dark:border-fuchsia-900/60'"
          @click="setCategory('ui_ux')"
        >
          <UIcon name="i-lucide-palette" class="w-3 h-3" />
          <span>UI / UX</span>
        </button>
        <button
          type="button"
          class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
          :class="selectedCategory === 'feature' ? 'bg-amber-600 text-white' : 'bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300 hover:bg-amber-100 dark:hover:bg-amber-900/50 border border-amber-200 dark:border-amber-900/60'"
          @click="setCategory('feature')"
        >
          <UIcon name="i-lucide-sparkles" class="w-3 h-3" />
          <span>Fitur</span>
        </button>
        <button
          type="button"
          class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
          :class="selectedCategory === 'architecture' ? 'bg-blue-600 text-white' : 'bg-blue-50 dark:bg-blue-950/40 text-blue-700 dark:text-blue-300 hover:bg-blue-100 dark:hover:bg-blue-900/50 border border-blue-200 dark:border-blue-900/60'"
          @click="setCategory('architecture')"
        >
          <UIcon name="i-lucide-network" class="w-3 h-3" />
          <span>Arsitektur</span>
        </button>
        <button
          type="button"
          class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
          :class="selectedCategory === 'performance' ? 'bg-sky-600 text-white' : 'bg-sky-50 dark:bg-sky-950/40 text-sky-700 dark:text-sky-300 hover:bg-sky-100 dark:hover:bg-sky-900/50 border border-sky-200 dark:border-sky-900/60'"
          @click="setCategory('performance')"
        >
          <UIcon name="i-lucide-zap" class="w-3 h-3" />
          <span>Performa</span>
        </button>
        <button
          type="button"
          class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
          :class="selectedCategory === 'backend' ? 'bg-sky-600 text-white' : 'bg-sky-50 dark:bg-sky-950/40 text-sky-700 dark:text-sky-300 hover:bg-sky-100 dark:hover:bg-sky-900/50 border border-sky-200 dark:border-sky-900/60'"
          @click="setCategory('backend')"
        >
          <UIcon name="i-lucide-server" class="w-3 h-3" />
          <span>Backend</span>
        </button>
        <button
          type="button"
          class="px-2.5 py-1 rounded-md font-medium text-xs transition-colors flex items-center gap-1 cursor-pointer"
          :class="selectedCategory === 'devops' ? 'bg-violet-600 text-white' : 'bg-violet-50 dark:bg-violet-950/40 text-violet-700 dark:text-violet-300 hover:bg-violet-100 dark:hover:bg-violet-900/50 border border-violet-200 dark:border-violet-900/60'"
          @click="setCategory('devops')"
        >
          <UIcon name="i-lucide-git-branch" class="w-3 h-3" />
          <span>DevOps</span>
        </button>
      </div>

      <TodosTodoTagFilter v-model="selectedTag" />

      <!-- Row 3: Secondary Filters & Order By (Prioritas, dll.) -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2.5 text-xs">
        <!-- Search -->
        <div class="relative">
          <UIcon name="i-lucide-search" class="w-4 h-4 absolute left-2.5 top-2.5 text-neutral-400" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Cari agenda, kode [SEC-001], dll..."
            class="w-full pl-8 pr-3 py-1.5 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800 text-neutral-900 dark:text-white"
          >
        </div>

        <!-- Order By Selector (Prioritas, Tanggal, dll.) -->
        <div class="relative flex items-center">
          <span class="absolute left-2.5 pointer-events-none text-neutral-400">
            <UIcon name="i-lucide-arrow-down-up" class="w-3.5 h-3.5 text-violet-500" />
          </span>
          <select
            v-model="selectedOrderBy"
            class="w-full pl-8 pr-7 py-1.5 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800 text-neutral-900 dark:text-white appearance-none cursor-pointer"
            title="Urutkan agenda berdasarkan"
          >
            <option value="priority_desc">
              ⚡ Urut: Prioritas (Tertinggi)
            </option>
            <option value="priority_asc">
              ⚡ Urut: Prioritas (Terendah)
            </option>
            <option value="impact_desc">
              🚀 Urut: Dampak (Terbesar)
            </option>
            <option value="value_desc">
              🎯 Urut: Dampak per Usaha
            </option>
            <option value="created_desc">
              🕒 Urut: Terbaru Dibuat
            </option>
            <option value="created_asc">
              🕒 Urut: Terlama Dibuat
            </option>
            <option value="title_asc">
              🔤 Urut: Judul (A - Z)
            </option>
            <option value="status">
              📊 Urut: Status Tugas
            </option>
          </select>
          <span class="absolute right-2 pointer-events-none text-neutral-400">
            <UIcon name="i-lucide-chevron-down" class="w-3.5 h-3.5" />
          </span>
        </div>

        <!-- Status Filter (Termasuk Opsi Belum Selesai) -->
        <div class="relative flex items-center">
          <span class="absolute left-2.5 pointer-events-none text-neutral-400">
            <UIcon name="i-lucide-filter" class="w-3.5 h-3.5 text-amber-500" />
          </span>
          <select
            v-model="selectedStatus"
            class="w-full pl-8 pr-7 py-1.5 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800 text-neutral-900 dark:text-white appearance-none cursor-pointer"
            title="Filter berdasarkan status pengerjaan"
          >
            <option value="uncompleted">
              ⏳ Belum Selesai (Sembunyikan Selesai)
            </option>
            <option value="all">
              Semua Status (Termasuk Selesai)
            </option>
            <option value="in_progress">
              Sedang Dikerjakan
            </option>
            <option value="pending">
              Menunggu (Pending)
            </option>
            <option value="suggested">
              Saran AI (Menunggu Approval)
            </option>
            <option value="completed">
              Selesai
            </option>
            <option value="rejected">
              Ditolak
            </option>
          </select>
          <span class="absolute right-2 pointer-events-none text-neutral-400">
            <UIcon name="i-lucide-chevron-down" class="w-3.5 h-3.5" />
          </span>
        </div>

        <!-- Origin / Asal Todo Filter -->
        <div class="relative flex items-center">
          <span class="absolute left-2.5 pointer-events-none text-neutral-400">
            <UIcon name="i-lucide-layers" class="w-3.5 h-3.5 text-blue-500" />
          </span>
          <select
            v-model="selectedCreator"
            class="w-full pl-8 pr-7 py-1.5 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800 text-neutral-900 dark:text-white appearance-none cursor-pointer"
          >
            <option value="all">
              🌐 Semua Asal Todo
            </option>
            <option value="executive">
              🏛️ Dewan Eksekutif
            </option>
            <option value="audit">
              🔍 Optimasi Audit
            </option>
            <option value="pm">
              📋 Project Manager
            </option>
            <option value="ai_suggestion">
              💡 Rekomendasi AI
            </option>
            <option value="agent">
              💻 Agent Coding
            </option>
            <option value="human">
              👤 Developer (Manual)
            </option>
          </select>
          <span class="absolute right-2 pointer-events-none text-neutral-400">
            <UIcon name="i-lucide-chevron-down" class="w-3.5 h-3.5" />
          </span>
        </div>
      </div>
    </div>

    <!-- Loading State Skeleton -->
    <div v-if="loading" class="space-y-3">
      <div
        v-for="i in 3"
        :key="i"
        class="p-4 rounded-xl border border-neutral-200/80 dark:border-white/10 bg-white/60 dark:bg-slate-900/40 animate-pulse space-y-3"
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
    <div v-else-if="filteredTodos.length > 0" id="todos-list-container" ref="todosContainerRef" class="space-y-4">
      <!-- Top Pagination Toolbar -->
      <div
        class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 px-3.5 py-2 rounded-xl border border-neutral-200/80 dark:border-white/10 bg-neutral-50/70 dark:bg-neutral-900/40 text-xs shadow-2xs"
      >
        <div class="flex items-center gap-2 text-neutral-600 dark:text-neutral-400 font-medium">
          <UIcon name="i-lucide-list-todo" class="w-4 h-4 text-sky-500" />
          <span>
            Menampilkan
            <strong class="font-mono text-neutral-900 dark:text-white">
              {{ (page - 1) * perPage + 1 }}–{{ Math.min(page * perPage, filteredTodos.length) }}
            </strong>
            dari
            <strong class="font-mono text-neutral-900 dark:text-white">{{ filteredTodos.length }}</strong>
            agenda
            <span v-if="filteredTodos.length !== workTodos.length" class="text-neutral-400">
              (difilter dari {{ workTodos.length }})
            </span>
          </span>
        </div>

        <div class="flex items-center gap-2.5 self-end sm:self-center flex-wrap">
          <!-- Per Page Selector -->
          <div class="flex items-center gap-1.5 text-neutral-500 dark:text-neutral-400 text-xs">
            <label for="todos-per-page" class="text-[11px] font-medium">Per hal:</label>
            <select
              id="todos-per-page"
              v-model.number="perPage"
              class="h-7 pl-2 pr-6 py-0 text-xs font-mono font-medium rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 focus:outline-none focus:ring-1 focus:ring-sky-500 cursor-pointer"
            >
              <option v-for="opt in perPageOptions" :key="opt" :value="opt">
                {{ opt }}
              </option>
            </select>
          </div>

          <!-- Top UPagination -->
          <UPagination
            v-if="filteredTodos.length > perPage"
            :page="page"
            :total="filteredTodos.length"
            :items-per-page="perPage"
            size="xs"
            @update:page="setPage"
          />
        </div>
      </div>

      <!-- Paginated Todos Cards -->
      <div class="space-y-3">
        <TodosTodoCard
          v-for="todo in paginatedTodos"
          :key="todo.id"
          :todo="todo"
          :agent-run="agentRunsByTodo[todo.id]"
          :sub-todos="subTodosByParent[todo.id]"
          @approve="handleApprove"
          @reject="handleReject"
          @update-status="handleUpdateStatus"
          @delete="handleDelete"
          @dispatch="handleDispatch"
          @split="handleSplit"
        />
      </div>

      <!-- Bottom Pagination Bar -->
      <div
        v-if="filteredTodos.length > perPage"
        class="pt-3 pb-1 flex flex-col sm:flex-row items-center justify-between gap-3 border-t border-neutral-200/80 dark:border-white/10"
      >
        <span class="text-xs text-neutral-500 font-mono">
          Halaman {{ page }} dari {{ totalPages }} ({{ filteredTodos.length }} total agenda)
        </span>

        <UPagination
          :page="page"
          :total="filteredTodos.length"
          :items-per-page="perPage"
          size="sm"
          @update:page="setPage"
        />
      </div>
    </div>

    <!-- Empty State -->
    <div
      v-else
      class="p-12 text-center rounded-2xl border border-dashed border-neutral-300 dark:border-white/10 bg-white/40 dark:bg-slate-900/40 space-y-3"
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

    <OrchestratorDispatchModal
      v-model:open="dispatchOpen"
      :todo="dispatchTodo"
      @dispatched="refreshAgentRuns(); loadData()"
    />
  </div>
</template>
