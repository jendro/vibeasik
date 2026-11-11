<script setup lang="ts">
import type { Project, ProjectChatMessageDTO, LLMPlannerTodoItem, LLMConfig, TodoPriority, TodoLevel, ProjectBaseKnowledge } from '~/types'

const { modelName } = useModelNames()

const props = defineProps<{
  project: Project
  cachedBaseKnowledge?: ProjectBaseKnowledge | null
}>()

const toast = useToast()
const {
  chatProject,
  getProjectChatHistory,
  clearProjectChatHistory,
  createTodo,
  getLLMConfigs
} = useApi()

// Chat states
const messages = ref<ProjectChatMessageDTO[]>([])
const inputPrompt = ref('')
const sending = ref(false)
const loadingHistory = ref(true)
const messagesContainer = ref<HTMLElement | null>(null)

// Tracking added todos: Map of messageId_todoIdx -> boolean
const addedTodos = ref<Record<string, boolean>>({})
const addingTodoKey = ref<string | null>(null)
const batchAddingMessageId = ref<number | null>(null)

// LLM Model Config
const llmConfigs = ref<LLMConfig[]>([])
const selectedModel = ref<string>('')

// Computed stats without making redundant heavy BE calls
const baseKnowledgeStats = computed(() => {
  const apps = props.cachedBaseKnowledge?.apps_count || props.project?.components?.length || 0
  const routes = props.cachedBaseKnowledge?.routers_scope?.total_routes || 0
  if (apps === 0) return null
  return { apps, routes }
})

const loadInitialData = async () => {
  loadingHistory.value = true
  try {
    const [history, configs] = await Promise.all([
      getProjectChatHistory(props.project.id).catch(() => []),
      getLLMConfigs('standard_llm').catch(() => [])
    ])

    messages.value = history || []
    llmConfigs.value = configs || []

    if (configs && configs.length > 0) {
      const active = configs.find(c => c.is_active) || configs[0]
      selectedModel.value = active?.selected_model || ''
    }

    scrollToBottom()
  } catch (err: unknown) {
    console.error('Failed to load chat data:', err)
  } finally {
    loadingHistory.value = false
  }
}

onMounted(() => {
  loadInitialData()
})

const scrollToBottom = () => {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

const handleSendMessage = async (textToSend?: string) => {
  const content = (textToSend || inputPrompt.value).trim()
  if (!content || sending.value) return

  inputPrompt.value = ''
  sending.value = true

  // Optimistically push user message
  const tempUserMsg: ProjectChatMessageDTO = {
    id: Date.now(),
    project_id: props.project.id,
    role: 'user',
    content,
    created_at: new Date().toISOString()
  }
  messages.value.push(tempUserMsg)
  scrollToBottom()

  try {
    const res = await chatProject(props.project.id, {
      message: content,
      model: selectedModel.value || undefined
    })

    const assistantMsg: ProjectChatMessageDTO = {
      id: res.message_id || Date.now() + 1,
      project_id: props.project.id,
      role: 'assistant',
      content: res.analysis || res.reply,
      todos: res.todos || [],
      model_name: res.model,
      created_at: new Date().toISOString()
    }
    messages.value.push(assistantMsg)
    scrollToBottom()

    toast.add({
      title: 'Kajian AI Siap',
      description: `AI berhasil mengkaji request dan menghasilkan ${res.todos?.length || 0} rekomendasi TODO siap eksekusi.`,
      icon: 'i-lucide-sparkles',
      color: 'success'
    })
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Gagal menghubungi AI planner'
    toast.add({
      title: 'Gagal Memproses Permintaan',
      description: msg,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    sending.value = false
  }
}

const handleClearHistory = async () => {
  try {
    await clearProjectChatHistory(props.project.id)
    messages.value = []
    addedTodos.value = {}
    toast.add({
      title: 'Riwayat Percakapan Dibersihkan',
      description: 'Seluruh percakapan ide proyek berhasil dihapus.',
      icon: 'i-lucide-trash-2',
      color: 'neutral'
    })
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Gagal membersihkan riwayat'
    toast.add({
      title: 'Gagal Membersihkan Riwayat',
      description: msg,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  }
}

// Convert a single planner todo into the official Project Todo roadmap
const handleAddSingleTodo = async (todo: LLMPlannerTodoItem, messageId: number, idx: number) => {
  const key = `${messageId}_${idx}`
  addingTodoKey.value = key

  try {
    await createTodo({
      project_id: props.project.id,
      title: todo.title,
      description: `${todo.description}\n\nTarget Apps: ${todo.target_apps}\n\nAlasan & Analisis:\n${todo.rationale}`,
      priority: (todo.priority as TodoPriority) || 'medium',
      category: todo.category || 'feature',
      level: (todo.level as TodoLevel) || 'component',
      target_apps: todo.target_apps,
      rationale: todo.rationale,
      guideline_ref: todo.guideline_ref,
      source: 'user_idea',
      creator_type: 'ai_agent',
      creator_name: 'AI Project Architect',
      status: 'pending'
    })

    addedTodos.value[key] = true
    toast.add({
      title: 'TODO Berhasil Ditambahkan',
      description: `Task "${todo.title}" berhasil dimasukkan ke Roadmap Todo Proyek.`,
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Gagal membuat todo'
    toast.add({
      title: 'Gagal Menambahkan TODO',
      description: msg,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    addingTodoKey.value = null
  }
}

// Add all generated todos in a message with 1 click
const handleAddAllTodos = async (todos: LLMPlannerTodoItem[], messageId: number) => {
  batchAddingMessageId.value = messageId
  let successCount = 0

  try {
    for (let i = 0; i < todos.length; i++) {
      const todo = todos[i]
      const key = `${messageId}_${i}`
      if (addedTodos.value[key]) continue

      if (todo) {
        await createTodo({
          project_id: props.project.id,
          title: todo.title,
          description: `${todo.description}\n\nTarget Apps: ${todo.target_apps}\n\nAlasan & Analisis:\n${todo.rationale}`,
          priority: (todo.priority as TodoPriority) || 'medium',
          category: todo.category || 'feature',
          level: (todo.level as TodoLevel) || 'component',
          target_apps: todo.target_apps,
          rationale: todo.rationale,
          guideline_ref: todo.guideline_ref,
          source: 'user_idea',
          creator_type: 'ai_agent',
          creator_name: 'AI Project Architect',
          status: 'pending'
        })
        addedTodos.value[key] = true
        successCount++
      }
    }

    toast.add({
      title: 'Semua TODO Berhasil Ditambahkan',
      description: `${successCount} task baru berhasil ditambahkan ke roadmap proyek.`,
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Gagal menambahkan beberapa todo'
    toast.add({
      title: 'Proses TODO Selesai Sebagian',
      description: msg,
      icon: 'i-lucide-alert-circle',
      color: 'warning'
    })
  } finally {
    batchAddingMessageId.value = null
  }
}

// Copy ready-to-run prompt for lightweight worker agent (e.g. Antigravity, Claude Code, Aider)
const copyWorkerPrompt = async (todo: LLMPlannerTodoItem) => {
  const prompt = `[TASK UNTUK AGENT PEKERJA AI]
PROYEK: ${props.project.name}
TARGET APLIKASI: ${todo.target_apps}
JUDUL TASK: ${todo.title}
KATEGORI: ${todo.category} (Prioritas: ${todo.priority})
RUBRIK PATOKAN: ${todo.guideline_ref || 'BEST_PRACTICE'}

SPESIFIKASI TEKNIS & LANGKAH PENGERJAAN:
${todo.description}

ALASAN TEKNIS:
${todo.rationale}

ATURAN EKSEKUSI MANDIRI:
1. Analisis kode di repositori target secara mendalam sebelum mengedit.
2. Terapkan perubahan kode secara presisi dan drop-in tanpa merusak fitur yang sudah ada.
3. Jalankan build dan test lokal untuk memverifikasi fungsionalitas.
4. Lakukan git commit rapi setelah seluruh tugas selesai.`

  try {
    await navigator.clipboard.writeText(prompt)
    toast.add({
      title: 'Prompt Pekerja Disalin',
      description: 'Prompt instruksi lengkap siap di-paste ke Google Antigravity, Claude Code, atau Aider.',
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

const quickIdeaChips = [
  'Tambah Login Google OAuth & Session JWT',
  'Implementasi Endpoint Payment Gateway Midtrans',
  'Fitur Export Laporan ke Excel & PDF',
  'Integrasi Realtime WebSocket Notification',
  'Proteksi Rate Limiting & Anti-Brute Force'
]

const getPriorityBadgeClass = (priority: string) => {
  switch (priority.toLowerCase()) {
    case 'critical':
      return 'bg-rose-50 dark:bg-rose-950/60 text-rose-700 dark:text-rose-400 border-rose-200 dark:border-rose-800'
    case 'high':
      return 'bg-amber-50 dark:bg-amber-950/60 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800'
    case 'medium':
      return 'bg-blue-50 dark:bg-blue-950/60 text-blue-700 dark:text-blue-400 border-blue-200 dark:border-blue-800'
    default:
      return 'bg-slate-50 dark:bg-slate-800 text-slate-600 dark:text-slate-400 border-slate-200 dark:border-slate-700'
  }
}
</script>

<template>
  <div class="flex flex-col h-[calc(100vh-13rem)] min-h-[580px] rounded-xl border border-slate-200 dark:border-white/[0.08] bg-white dark:bg-slate-900/60 overflow-hidden">
    <!-- Chat Header Bar -->
    <div class="flex flex-wrap items-center justify-between gap-3 p-4 border-b border-slate-200 dark:border-white/[0.08] bg-slate-50/70 dark:bg-slate-900/80">
      <div class="flex items-center gap-2.5">
        <div class="p-1.5 rounded-lg bg-indigo-50 dark:bg-indigo-950/80 text-indigo-600 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-800">
          <UIcon name="i-lucide-bot" class="size-4.5" />
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h3 class="text-sm font-bold text-slate-900 dark:text-slate-100">
              AI Project Architect & Planner
            </h3>
            <span
              v-if="baseKnowledgeStats"
              class="hidden sm:inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-mono bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800"
            >
              <span class="size-1.5 rounded-full bg-emerald-500 animate-pulse" />
              Base Knowledge: {{ baseKnowledgeStats.apps }} Apps, {{ baseKnowledgeStats.routes }} Rute
            </span>
          </div>
          <p class="text-xs text-slate-500 dark:text-slate-400">
            Kaji ide/request user menggunakan Base Knowledge proyek, hasilkan TODO berbobot untuk agent pekerja.
          </p>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <UButton
          v-if="messages.length > 0"
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-lucide-trash-2"
          title="Bersihkan riwayat percakapan"
          @click="handleClearHistory"
        >
          Bersihkan Chat
        </UButton>
      </div>
    </div>

    <!-- Messages Container -->
    <div
      ref="messagesContainer"
      class="flex-1 p-4 sm:p-6 overflow-y-auto space-y-6"
    >
      <!-- Empty State -->
      <div
        v-if="!loadingHistory && messages.length === 0"
        class="h-full flex flex-col items-center justify-center text-center p-6 space-y-4 max-w-lg mx-auto"
      >
        <div class="p-3 rounded-xl bg-indigo-50 dark:bg-indigo-950/60 text-indigo-500 border border-indigo-200/60 dark:border-indigo-900">
          <UIcon name="i-lucide-sparkles" class="size-8" />
        </div>
        <div class="space-y-1.5">
          <h4 class="text-base font-bold text-slate-900 dark:text-slate-100">
            Kaji Request & Ide Proyek Anda
          </h4>
          <p class="text-xs text-slate-500 dark:text-slate-400 leading-relaxed">
            AI Architect membaca seluruh ekosistem aplikasi, file, router, dan database proyek ini. Sampaikan fitur atau perbaikan yang ingin Anda tambahkan, lalu jadikan responnya TODO lengkap untuk pekerja AI.
          </p>
        </div>

        <!-- Quick Idea Prompts -->
        <div class="w-full pt-2 space-y-2">
          <span class="text-xs font-semibold text-slate-400">Contoh Ide Permintaan:</span>
          <div class="flex flex-wrap justify-center gap-1.5">
            <button
              v-for="chip in quickIdeaChips"
              :key="chip"
              class="px-2.5 py-1.5 rounded-lg text-xs text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 hover:bg-indigo-50 dark:hover:bg-indigo-950/60 hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors border border-slate-200/80 dark:border-slate-700/80 text-left"
              @click="handleSendMessage(chip)"
            >
              {{ chip }}
            </button>
          </div>
        </div>
      </div>

      <!-- Messages Stream -->
      <div
        v-for="msg in messages"
        :key="msg.id"
        class="flex flex-col gap-2"
        :class="msg.role === 'user' ? 'items-end' : 'items-start'"
      >
        <!-- User Message -->
        <div
          v-if="msg.role === 'user'"
          class="max-w-[85%] sm:max-w-[75%] p-3.5 rounded-xl rounded-br-xs bg-indigo-600 text-white text-xs leading-relaxed"
        >
          <p class="whitespace-pre-wrap">
            {{ msg.content }}
          </p>
          <div class="text-xs text-indigo-200 mt-1.5 text-right">
            {{ new Date(msg.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }}
          </div>
        </div>

        <!-- Assistant Message -->
        <div
          v-else
          class="w-full max-w-[95%] sm:max-w-[90%] space-y-4"
        >
          <!-- Analysis Bubble -->
          <div class="p-4 sm:p-5 rounded-xl rounded-tl-xs bg-slate-50 dark:bg-[#04111f] border border-slate-200 dark:border-white/[0.08] text-slate-800 dark:text-slate-200 text-xs leading-relaxed space-y-3">
            <div class="flex items-center justify-between gap-2 pb-2 border-b border-slate-200/80 dark:border-white/[0.08]">
              <span class="font-bold flex items-center gap-1.5 text-indigo-600 dark:text-indigo-400 text-xs">
                <UIcon name="i-lucide-cpu" class="size-4" />
                Kajian & Analisis Dampak Arsitektur
              </span>
              <span v-if="msg.model_name" class="text-xs font-mono text-slate-400">
                model: {{ modelName(msg.model_name) }}
              </span>
            </div>

            <!-- Analysis Markdown Text -->
            <div class="whitespace-pre-wrap font-sans leading-relaxed text-slate-700 dark:text-slate-300">
              {{ msg.content }}
            </div>
          </div>

          <!-- Generated TODOs Cards -->
          <div v-if="msg.todos && msg.todos.length > 0" class="space-y-3 pl-2 sm:pl-4">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-list-todo" class="size-4 text-emerald-500" />
                <h4 class="text-xs font-semibold text-slate-900 dark:text-slate-100">
                  Rekomendasi TODO untuk Pekerja AI ({{ msg.todos.length }} Task)
                </h4>
              </div>

              <!-- Batch Add All Button -->
              <UButton
                size="xs"
                color="primary"
                variant="solid"
                icon="i-lucide-plus-circle"
                :loading="batchAddingMessageId === msg.id"
                @click="handleAddAllTodos(msg.todos, msg.id)"
              >
                Tambahkan Semua ke TODO Proyek
              </UButton>
            </div>

            <!-- Single Todo Card -->
            <div class="grid grid-cols-1 gap-3">
              <div
                v-for="(todo, tIdx) in msg.todos"
                :key="tIdx"
                class="p-4 rounded-xl border bg-white dark:bg-slate-950/70 space-y-3 transition-all"
                :class="addedTodos[`${msg.id}_${tIdx}`]
                  ? 'border-emerald-300 dark:border-emerald-800 bg-emerald-50/20 dark:bg-emerald-950/10'
                  : 'border-slate-200 dark:border-white/[0.08]'"
              >
                <!-- Title & Meta Badges -->
                <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-2">
                  <div class="space-y-1">
                    <div class="flex flex-wrap items-center gap-1.5">
                      <span
                        class="px-2 py-0.5 rounded text-xs font-semibold border"
                        :class="getPriorityBadgeClass(todo.priority)"
                      >
                        {{ todo.priority }}
                      </span>
                      <span class="px-2 py-0.5 rounded text-xs font-mono bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300">
                        {{ todo.category }}
                      </span>
                      <span class="px-2 py-0.5 rounded text-xs font-mono bg-blue-50 dark:bg-blue-950 text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-900">
                        {{ todo.target_apps }}
                      </span>
                      <span v-if="todo.guideline_ref" class="text-xs font-mono text-slate-400">
                        #{{ todo.guideline_ref }}
                      </span>
                    </div>

                    <h5 class="text-sm font-bold text-slate-900 dark:text-slate-100">
                      {{ todo.title }}
                    </h5>
                  </div>

                  <!-- Actions per item -->
                  <div class="flex items-center gap-2 shrink-0 pt-1 sm:pt-0">
                    <UButton
                      size="xs"
                      color="neutral"
                      variant="outline"
                      icon="i-lucide-clipboard"
                      title="Salin prompt untuk agent pekerja (Antigravity/Claude Code)"
                      @click="copyWorkerPrompt(todo)"
                    >
                      Salin Prompt Pekerja
                    </UButton>

                    <UButton
                      v-if="!addedTodos[`${msg.id}_${tIdx}`]"
                      size="xs"
                      color="primary"
                      variant="solid"
                      icon="i-lucide-plus"
                      :loading="addingTodoKey === `${msg.id}_${tIdx}`"
                      @click="handleAddSingleTodo(todo, msg.id, tIdx)"
                    >
                      Masuk ke TODO
                    </UButton>

                    <span
                      v-else
                      class="inline-flex items-center gap-1 px-2.5 py-1 rounded text-xs font-semibold bg-emerald-100 dark:bg-emerald-950 text-emerald-700 dark:text-emerald-400"
                    >
                      <UIcon name="i-lucide-check" class="size-3.5" />
                      Ditambahkan
                    </span>
                  </div>
                </div>

                <!-- Description / Step-by-step & Rationale -->
                <div class="text-xs text-slate-600 dark:text-slate-300 bg-slate-50/70 dark:bg-slate-900/60 p-3 rounded-lg font-sans whitespace-pre-wrap leading-relaxed border border-slate-100 dark:border-slate-800/80">
                  {{ todo.description }}
                </div>

                <div v-if="todo.rationale" class="text-xs text-slate-500 dark:text-slate-400 italic">
                  💡 <strong>Alasan Teknis:</strong> {{ todo.rationale }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Thinking Indicator -->
      <div v-if="sending" class="flex items-center gap-2 text-xs text-slate-400 pl-2">
        <UIcon name="i-lucide-loader-2" class="size-4 animate-spin text-indigo-500" />
        <span>AI sedang mengkaji request dan menyusun spesifikasi TODO teknis...</span>
      </div>
    </div>

    <!-- Input Footer -->
    <div class="p-3 sm:p-4 border-t border-slate-200 dark:border-white/[0.08] bg-slate-50/50 dark:bg-slate-900/60 space-y-2">
      <!-- Quick Chips if messages exist -->
      <div v-if="messages.length > 0" class="flex items-center gap-1.5 overflow-x-auto pb-1 text-xs">
        <span class="text-xs font-semibold text-slate-400 shrink-0">Ide Cepat:</span>
        <button
          v-for="chip in quickIdeaChips"
          :key="chip"
          class="px-2 py-0.5 rounded text-xs whitespace-nowrap bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-300 hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors"
          @click="handleSendMessage(chip)"
        >
          {{ chip }}
        </button>
      </div>

      <!-- Input Form -->
      <form class="flex items-end gap-2" @submit.prevent="handleSendMessage()">
        <div class="flex-1 min-w-0">
          <textarea
            v-model="inputPrompt"
            rows="2"
            placeholder="Tuliskan ide atau request fitur baru (misal: 'Saya mau nambah fitur login dengan Google OAuth dan sistem wallet')..."
            class="w-full text-xs p-3 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-[#04111f] text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-hidden focus:ring-1 focus:ring-indigo-500 resize-none"
            :disabled="sending"
            @keydown.enter.exact.prevent="handleSendMessage()"
          />
        </div>

        <UButton
          type="submit"
          color="primary"
          variant="solid"
          size="md"
          icon="i-lucide-send"
          :loading="sending"
          :disabled="!inputPrompt.trim()"
          class="h-10 px-4 shrink-0 rounded-xl"
        >
          Kaji & Rancang TODO
        </UButton>
      </form>
    </div>
  </div>
</template>
