<script setup lang="ts">
import type { AgentRun, Todo } from '~/types'

const props = defineProps<{
  todo: Todo
  agentRun?: AgentRun
  /** Sub todo bila todo ini todo gabungan (Aturan 43) */
  subTodos?: Todo[]
}>()

const emit = defineEmits<{
  (e: 'approve', id: number): void
  (e: 'reject', id: number): void
  (e: 'updateStatus', id: number, status: string): void
  (e: 'delete', id: number): void
  (e: 'dispatch', todo: Todo): void
  (e: 'split', id: number): void
}>()

const isGroup = computed(() => props.todo.source === 'todo_group')
const canSplit = computed(() => isGroup.value && !props.agentRun && (props.todo.status === 'pending' || props.todo.status === 'suggested'))

const showRationale = ref(false)
const isExpanded = ref(false)
const isLongDescription = computed(() => (props.todo.description || '').length > 280)

const formatDateTime = (dateStr?: string) => {
  if (!dateStr) return '-'
  try {
    return new Date(dateStr).toLocaleDateString('id-ID', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  } catch {
    return dateStr
  }
}

const levelMeta = computed(() => {
  switch (props.todo.level) {
    case 'global':
      return { label: 'Global (Platform)', color: 'neutral', icon: 'i-lucide-globe', badgeClass: 'bg-violet-100 text-violet-800 dark:bg-violet-900/40 dark:text-violet-300 border-violet-200 dark:border-violet-800' }
    case 'project':
      return { label: 'Proyek (Bundle)', color: 'info', icon: 'i-lucide-folder-kanban', badgeClass: 'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300 border-blue-200 dark:border-blue-800' }
    case 'component':
      return { label: 'Komponen (Module)', color: 'success', icon: 'i-lucide-layers', badgeClass: 'bg-sky-100 text-sky-800 dark:bg-sky-900/40 dark:text-sky-300 border-sky-200 dark:border-sky-800' }
    default:
      return { label: props.todo.level, color: 'neutral', icon: 'i-lucide-check-square', badgeClass: 'bg-neutral-100 text-neutral-800 dark:bg-neutral-800 dark:text-neutral-300 border-neutral-200 dark:border-neutral-700' }
  }
})

const priorityMeta = computed(() => {
  switch (props.todo.priority) {
    case 'critical':
      return { label: 'Kritis', class: 'text-red-700 bg-red-50 dark:bg-red-950/40 dark:text-red-300 border-red-200 dark:border-red-800' }
    case 'high':
      return { label: 'Tinggi', class: 'text-amber-700 bg-amber-50 dark:bg-amber-950/40 dark:text-amber-300 border-amber-200 dark:border-amber-800' }
    case 'medium':
      return { label: 'Sedang', class: 'text-blue-700 bg-blue-50 dark:bg-blue-950/40 dark:text-blue-300 border-blue-200 dark:border-blue-800' }
    default:
      return { label: 'Rendah', class: 'text-neutral-600 bg-neutral-50 dark:bg-neutral-800/40 dark:text-neutral-400 border-neutral-200 dark:border-neutral-700' }
  }
})

const statusMeta = computed(() => {
  switch (props.todo.status) {
    case 'suggested':
      return { label: 'Saran AI Menunggu Approval', class: 'text-amber-700 bg-amber-50 dark:bg-amber-950/30 dark:text-amber-300 border-amber-300 dark:border-amber-700' }
    case 'in_progress':
      return { label: 'Sedang Dikerjakan', class: 'text-sky-700 bg-sky-50 dark:bg-sky-950/30 dark:text-sky-300 border-sky-300 dark:border-sky-700' }
    case 'completed':
      return { label: 'Selesai', class: 'text-emerald-700 bg-emerald-50 dark:bg-emerald-950/30 dark:text-emerald-300 border-emerald-300 dark:border-emerald-700' }
    case 'rejected':
      return { label: 'Ditolak', class: 'text-neutral-500 bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-400 border-neutral-200 dark:border-neutral-700' }
    case 'merged':
      return { label: 'Dilebur ke Todo Gabungan', class: 'text-violet-700 bg-violet-50 dark:bg-violet-950/30 dark:text-violet-300 border-violet-300 dark:border-violet-700' }
    case 'split':
      return { label: 'Dipecah per Aplikasi', class: 'text-indigo-700 bg-indigo-50 dark:bg-indigo-950/30 dark:text-indigo-300 border-indigo-300 dark:border-indigo-700' }
    default:
      return { label: 'Menunggu (Pending)', class: 'text-neutral-700 bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-300 border-neutral-200 dark:border-neutral-700' }
  }
})

const categoryMeta = computed(() => {
  const cat = props.todo.category || 'general'
  switch (cat) {
    case 'security':
      return { label: 'Keamanan (Security)', icon: 'i-lucide-shield-check', class: 'bg-rose-100 text-rose-800 dark:bg-rose-950/40 dark:text-rose-300 border-rose-200 dark:border-rose-800' }
    case 'ui_ux':
      return { label: 'UI / UX', icon: 'i-lucide-palette', class: 'bg-fuchsia-100 text-fuchsia-800 dark:bg-fuchsia-950/40 dark:text-fuchsia-300 border-fuchsia-200 dark:border-fuchsia-800' }
    case 'feature':
      return { label: 'Fitur', icon: 'i-lucide-sparkles', class: 'bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300 border-amber-200 dark:border-amber-800' }
    case 'architecture':
      return { label: 'Arsitektur', icon: 'i-lucide-network', class: 'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300 border-blue-200 dark:border-blue-800' }
    case 'performance':
      return { label: 'Performa & Port', icon: 'i-lucide-zap', class: 'bg-sky-100 text-sky-800 dark:bg-sky-950/40 dark:text-sky-300 border-sky-200 dark:border-sky-800' }
    case 'backend':
      return { label: 'Backend & API', icon: 'i-lucide-server', class: 'bg-sky-100 text-sky-800 dark:bg-sky-950/40 dark:text-sky-300 border-sky-200 dark:border-sky-800' }
    case 'devops':
      return { label: 'DevOps / CI-CD', icon: 'i-lucide-git-branch', class: 'bg-violet-100 text-violet-800 dark:bg-violet-950/40 dark:text-violet-300 border-violet-200 dark:border-violet-800' }
    default:
      return { label: 'Umum', icon: 'i-lucide-tag', class: 'bg-neutral-100 text-neutral-800 dark:bg-neutral-800 dark:text-neutral-300 border-neutral-200 dark:border-neutral-700' }
  }
})

const targetAppsDisplay = computed(() => {
  if (props.todo.target_apps) return props.todo.target_apps
  if (props.todo.component_name) return `${props.todo.component_name} [1 App]`
  if (props.todo.project_name) return `Seluruh Aplikasi (${props.todo.project_name})`
  return 'Global Platform'
})

const recommendedModelMeta = computed(() => {
  if (props.todo.status === 'completed' || props.todo.status === 'rejected') return null

  const comp = props.todo.complexity_level
  const cat = props.todo.category
  const prio = props.todo.priority

  if (comp === 'complex' || prio === 'critical' || cat === 'architecture' || cat === 'security') {
    return {
      label: 'Saran: Claude Opus 5',
      badgeClass: 'bg-violet-100/80 text-violet-800 dark:bg-violet-950/40 dark:text-violet-300 border-violet-200 dark:border-violet-800',
      icon: 'i-lucide-brain-circuit',
      title: 'Tugas rumit / arsitektur / keamanan kritis: Disarankan model flagship (Claude Opus 5 via OpenCode)'
    }
  }
  if (comp === 'quick' || cat === 'ui_ux' || cat === 'general') {
    return {
      label: 'Saran: GLM 5.2 / Flash',
      badgeClass: 'bg-emerald-100/80 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300 border-emerald-200 dark:border-emerald-800',
      icon: 'i-lucide-zap',
      title: 'Tugas ringan / UI polish: Cukup model cepat & hemat (GLM 5.2 OpenCode atau Gemini Flash Antigravity)'
    }
  }
  return {
    label: 'Saran: Claude Sonnet 5',
    badgeClass: 'bg-blue-100/80 text-blue-800 dark:bg-blue-950/40 dark:text-blue-300 border-blue-200 dark:border-blue-800',
    icon: 'i-lucide-sparkles',
    title: 'Tugas fitur / backend standar: Disarankan model cerdas & seimbang (Claude Sonnet 5 / DeepSeek V4 Pro via OpenCode)'
  }
})

const originMeta = computed(() => {
  const src = (props.todo.source || '').toLowerCase()
  const creator = (props.todo.creator_name || '').toLowerCase()
  const cType = (props.todo.creator_type || '').toLowerCase()
  const auditKey = (props.todo as any).audit_key || ''
  const guideline = props.todo.guideline_ref || ''

  // 1. Dewan Eksekutif (Autonomus atau Chat Eksekutif)
  if (src === 'autonomous_executive' || src === 'executive_chat' || creator.includes('eksekutif') || cType === 'executive') {
    return {
      type: 'executive',
      label: '🏛️ Dewan Eksekutif',
      sublabel: props.todo.creator_name || 'Dewan Eksekutif',
      badgeClass: 'bg-amber-100 text-amber-900 dark:bg-amber-950/60 dark:text-amber-300 border-amber-300 dark:border-amber-800 font-semibold',
      icon: 'i-lucide-crown',
      title: 'Dirumuskan melalui musyawarah Dewan Eksekutif (CEO, CTO, Product Owner, CMO)'
    }
  }

  // 2. Optimasi Audit (Diagnostic Audit / Deep Check / Guideline Findings)
  if (src === 'diagnostic_audit' || src === 'audit' || auditKey !== '' || guideline !== '') {
    const modelOrEngine = props.todo.creator_name || 'Engine'
    return {
      type: 'audit',
      label: '🔍 Optimasi Audit',
      sublabel: guideline ? `${guideline}` : modelOrEngine,
      badgeClass: 'bg-sky-100 text-sky-900 dark:bg-sky-950/60 dark:text-sky-300 border-sky-300 dark:border-sky-800 font-medium',
      icon: 'i-lucide-shield-alert',
      title: `Temuan dari hasil audit kualitas/arsitektur: ${guideline || modelOrEngine}`
    }
  }

  // 3. Project Manager / Planner
  if (src === 'pm_agent' || src === 'planner' || creator.includes('project-manager') || creator.includes('pm')) {
    return {
      type: 'pm',
      label: '📋 Project Manager',
      sublabel: 'PM Planner',
      badgeClass: 'bg-indigo-100 text-indigo-900 dark:bg-indigo-950/60 dark:text-indigo-300 border-indigo-300 dark:border-indigo-800 font-medium',
      icon: 'i-lucide-clipboard-list',
      title: 'Dirumuskan oleh AI Project Manager / Product Planner'
    }
  }

  // 4. Rekomendasi AI (AI Advisor / Template)
  if (props.todo.is_ai_suggestion || creator.includes('advisor') || src === 'ai_template' || src === 'template') {
    return {
      type: 'ai_suggestion',
      label: '💡 Rekomendasi AI',
      sublabel: props.todo.creator_name || 'AI Advisor',
      badgeClass: 'bg-fuchsia-100 text-fuchsia-900 dark:bg-fuchsia-950/60 dark:text-fuchsia-300 border-fuchsia-300 dark:border-fuchsia-800 font-medium',
      icon: 'i-lucide-sparkles',
      title: 'Saran perbaikan/fitur dari Vibeasik AI Advisor'
    }
  }

  // 5. Agent Coding (Claude Code, Antigravity, OpenCode, Aider, dll)
  if (cType === 'ai_agent' || creator.includes('claude') || creator.includes('antigravity') || creator.includes('opencode') || creator.includes('agent')) {
    return {
      type: 'agent',
      label: '💻 Agent Coding',
      sublabel: props.todo.creator_name || 'AI Agent',
      badgeClass: 'bg-emerald-100 text-emerald-900 dark:bg-emerald-950/60 dark:text-emerald-300 border-emerald-300 dark:border-emerald-800 font-medium',
      icon: 'i-lucide-bot',
      title: `Dibuat oleh AI Coding Agent: ${props.todo.creator_name}`
    }
  }

  // 6. Developer (Manual / Human)
  return {
    type: 'human',
    label: '👤 Developer (Manual)',
    sublabel: props.todo.creator_name || 'Developer',
    badgeClass: 'bg-neutral-100 text-neutral-800 dark:bg-neutral-800 dark:text-neutral-300 border-neutral-300 dark:border-neutral-700 font-medium',
    icon: 'i-lucide-user',
    title: `Dibuat manual oleh developer: ${props.todo.creator_name || 'developer'}`
  }
})
</script>

<template>
  <div
    class="p-4 rounded-xl border bg-white dark:bg-slate-900/40 transition-all shadow-2xs space-y-3"
    :class="[
      todo.status === 'suggested'
        ? 'border-amber-300/80 dark:border-amber-900/60 ring-1 ring-amber-400/20'
        : 'border-neutral-200 dark:border-white/10 hover:border-neutral-300 dark:hover:border-neutral-700'
    ]"
  >
    <!-- Top Metadata Bar -->
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex flex-wrap items-center gap-1.5 text-xs">
        <!-- Level Badge -->
        <span
          class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-medium text-[11px] border"
          :class="levelMeta.badgeClass"
        >
          <UIcon :name="levelMeta.icon" class="w-3.5 h-3.5" />
          {{ levelMeta.label }}
        </span>

        <!-- Context Pill (Project / Component) -->
        <span
          v-if="todo.project_name"
          class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-mono bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 border border-neutral-200 dark:border-neutral-700"
        >
          {{ todo.project_name }}
          <template v-if="todo.component_name">
            <span class="text-neutral-400">/</span>
            <span class="font-semibold">{{ todo.component_name }}</span>
          </template>
        </span>

        <!-- Category Badge -->
        <span
          class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-medium text-[11px] border"
          :class="categoryMeta.class"
        >
          <UIcon :name="categoryMeta.icon" class="w-3.5 h-3.5" />
          {{ categoryMeta.label }}
        </span>

        <TodosTodoTagChips :todo="todo" />
        <TodosComplexityBadge :todo="todo" />
        <TodosImpactBadge :todo="todo" />

        <!-- Recommended Model Badge -->
        <span
          v-if="recommendedModelMeta"
          class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-medium text-[11px] border"
          :class="recommendedModelMeta.badgeClass"
          :title="recommendedModelMeta.title"
        >
          <UIcon :name="recommendedModelMeta.icon" class="w-3.5 h-3.5" />
          <span>{{ recommendedModelMeta.label }}</span>
        </span>
        <span
          v-if="todo.split_from_id"
          class="inline-flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-md border border-indigo-200 dark:border-indigo-800 text-indigo-700 dark:text-indigo-300"
          title="Sub todo pecahan per aplikasi: hanya aplikasi ini, dikerjakan pegawai dengan keahliannya; induk selesai saat semua bagian selesai"
        >
          <UIcon name="i-lucide-split" class="w-3 h-3" />
          Bagian dari #{{ todo.split_from_id }}
        </span>
        <NuxtLink
          v-if="todo.employee_id && todo.employee_name"
          :to="`/staff?id=${todo.employee_id}`"
          class="inline-flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-md border border-sky-200 dark:border-sky-800 text-sky-700 dark:text-sky-300 hover:bg-sky-50 dark:hover:bg-sky-950/40"
          :title="todo.employee_reason || 'Pegawai vibeasik yang ditugaskan: agent coding mengerjakan todo ini dengan persona & pengetahuannya'"
        >
          <UIcon name="i-lucide-id-card" class="w-3 h-3" />
          {{ todo.employee_name }}
        </NuxtLink>
        <span
          v-if="todo.rework_count"
          class="inline-flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-md border"
          :class="(todo.employee_reason?.includes('di-hold') || todo.rework_note?.includes('di-hold'))
            ? 'bg-rose-50 dark:bg-rose-950/40 text-rose-700 dark:text-rose-300 border-rose-300 dark:border-rose-800 font-semibold'
            : 'border-orange-200 dark:border-orange-800 text-orange-700 dark:text-orange-300'"
          :title="todo.employee_reason?.includes('di-hold') ? todo.employee_reason : (todo.rework_note || 'Dikerjakan ulang karena nilai QA rendah')"
        >
          <UIcon :name="(todo.employee_reason?.includes('di-hold') || todo.rework_note?.includes('di-hold')) ? 'i-lucide-alert-octagon' : 'i-lucide-rotate-ccw'" class="w-3 h-3" />
          {{ (todo.employee_reason?.includes('di-hold') || todo.rework_note?.includes('di-hold')) ? `Ulang ×${todo.rework_count} · Di-hold` : `Ulang ×${todo.rework_count}` }}
        </span>

        <!-- Target Application Badge -->
        <span
          class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-medium bg-neutral-100 dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 border border-neutral-200 dark:border-neutral-700"
          :title="'Aplikasi target pengerjaan: ' + targetAppsDisplay"
        >
          <UIcon name="i-lucide-layout-grid" class="w-3.5 h-3.5 text-neutral-500" />
          <span>{{ targetAppsDisplay }}</span>
        </span>

        <!-- Origin / Source Tag -->
        <span
          class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[11px] border"
          :class="originMeta.badgeClass"
          :title="originMeta.title"
        >
          <UIcon :name="originMeta.icon" class="w-3.5 h-3.5" />
          <span>{{ originMeta.label }}</span>
          <span v-if="originMeta.sublabel && originMeta.sublabel !== originMeta.label" class="opacity-75 text-[10px]">
            ({{ originMeta.sublabel }})
          </span>
        </span>
      </div>

      <div class="flex items-center gap-1.5 text-xs">
        <!-- Priority Badge -->
        <span
          class="px-2 py-0.5 rounded-md text-[11px] font-medium border"
          :class="priorityMeta.class"
        >
          Prioritas: {{ priorityMeta.label }}
        </span>

        <!-- Status Badge -->
        <span
          class="px-2 py-0.5 rounded-md text-[11px] font-medium border"
          :class="statusMeta.class"
        >
          {{ statusMeta.label }}
        </span>
      </div>
    </div>

    <!-- Title & Description -->
    <div>
      <h3
        class="text-sm font-semibold text-neutral-900 dark:text-white flex items-center gap-2"
        :class="{ 'line-through text-neutral-400 dark:text-neutral-500': todo.status === 'completed' }"
      >
        <span v-if="todo.guideline_ref" class="font-mono text-xs text-violet-600 dark:text-violet-400 font-bold">
          [{{ todo.guideline_ref }}]
        </span>
        {{ todo.title }}
      </h3>

      <!-- Markdown Viewer Keterangan -->
      <div
        v-if="todo.description"
        class="mt-2 rounded-lg bg-neutral-50/70 dark:bg-neutral-800/40 p-2.5 border border-neutral-100 dark:border-white/10 transition-colors"
        :class="{ 'opacity-60': todo.status === 'completed' }"
      >
        <MarkdownViewer
          :content="todo.description"
          compact
          :max-lines="isExpanded ? undefined : (isLongDescription ? 4 : undefined)"
          empty-text="Belum ada keterangan tugas."
        />

        <div v-if="isLongDescription" class="mt-1.5 pt-1 border-t border-neutral-200/50 dark:border-neutral-700/50 flex justify-end">
          <button
            type="button"
            class="text-[10px] font-medium text-sky-600 dark:text-sky-400 hover:underline flex items-center gap-1 cursor-pointer"
            @click="isExpanded = !isExpanded"
          >
            <UIcon :name="isExpanded ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" class="w-3 h-3" />
            <span>{{ isExpanded ? 'Ciutkan Keterangan' : 'Lihat Selengkapnya' }}</span>
          </button>
        </div>
      </div>

      <TodosTodoSubTodoList v-if="subTodos?.length" :sub-todos="subTodos" />
    </div>

    <!-- Rationale Accordion (if AI recommendation) -->
    <div v-if="todo.rationale" class="text-xs pt-1">
      <button
        type="button"
        class="inline-flex items-center gap-1 text-[11px] text-amber-700 dark:text-amber-400 hover:underline font-medium"
        @click="showRationale = !showRationale"
      >
        <UIcon :name="showRationale ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="w-3.5 h-3.5" />
        {{ showRationale ? 'Sembunyikan Analisis & Patokan AI' : 'Lihat Analisis & Patokan Terukur AI' }}
      </button>

      <div
        v-if="showRationale"
        class="mt-2 p-3 rounded-lg bg-amber-50/70 dark:bg-amber-950/20 border border-amber-200/60 dark:border-amber-900/40 text-neutral-700 dark:text-neutral-300 text-xs space-y-1.5"
      >
        <div class="font-medium text-amber-800 dark:text-amber-300 flex items-center justify-between gap-1.5 text-[11px]">
          <div class="flex items-center gap-1.5">
            <UIcon name="i-lucide-compass" class="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" />
            <span>Dasar Patokan (Rubrik): <strong>{{ todo.guideline_ref || 'STANDAR' }}</strong></span>
          </div>
          <span class="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-amber-200/70 dark:bg-amber-900/60 text-amber-900 dark:text-amber-200">
            {{ categoryMeta.label }}
          </span>
        </div>
        <div class="pt-1 border-t border-amber-200/40 dark:border-amber-900/40">
          <MarkdownViewer
            :content="todo.rationale"
            compact
            empty-text="Belum ada catatan analisis."
          />
        </div>
      </div>
    </div>

    <!-- Completion Details Bar (when completed) -->
    <div
      v-if="todo.status === 'completed'"
      class="space-y-2 p-2.5 rounded-lg bg-emerald-50/70 dark:bg-emerald-950/20 border border-emerald-200/60 dark:border-emerald-800/40 text-xs"
    >
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex items-center gap-1.5 font-medium text-emerald-800 dark:text-emerald-300">
          <UIcon name="i-lucide-check-circle-2" class="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
          <span>Selesai:</span>
          <span class="font-normal text-neutral-700 dark:text-neutral-300">
            {{ todo.completed_at ? formatDateTime(todo.completed_at) : 'Selesai' }}
          </span>
        </div>

        <div
          v-if="todo.completed_by_agent"
          class="flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-md bg-white dark:bg-slate-900/40 border border-emerald-200 dark:border-emerald-800 text-emerald-800 dark:text-emerald-300"
        >
          <UIcon name="i-lucide-bot" class="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
          <span>Agent:</span>
          <span class="font-semibold">{{ todo.completed_by_agent }}</span>
        </div>

        <div
          v-if="todo.completed_by_model"
          class="flex items-center gap-1 font-mono text-[10px] px-2 py-0.5 rounded-md bg-white dark:bg-slate-900/40 border border-emerald-200 dark:border-emerald-800 text-emerald-700 dark:text-emerald-400"
        >
          <UIcon name="i-lucide-cpu" class="w-3 h-3 text-sky-500" />
          <span>Model:</span>
          <span class="font-semibold">{{ todo.completed_by_model }}</span>
        </div>

        <!-- QA Score Badge -->
        <span
          v-if="todo.score !== undefined && todo.score !== null"
          class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-semibold text-[11px] border ml-auto"
          :class="todo.score >= 80
            ? 'bg-amber-100 text-amber-900 border-amber-300 dark:bg-amber-950/60 dark:text-amber-200 dark:border-amber-800'
            : todo.score >= 60
              ? 'bg-blue-100 text-blue-900 border-blue-300 dark:bg-blue-950/60 dark:text-blue-200 dark:border-blue-800'
              : 'bg-red-100 text-red-900 border-red-300 dark:bg-red-950/60 dark:text-red-200 dark:border-red-800'"
        >
          <UIcon :name="todo.score >= 80 ? 'i-lucide-star' : 'i-lucide-award'" class="w-3 h-3 text-amber-500" />
          QA Score: {{ todo.score }}
        </span>

        <span
          v-if="todo.inherited_from"
          class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] bg-purple-100 text-purple-800 dark:bg-purple-950/50 dark:text-purple-300 border border-purple-200 dark:border-purple-800"
        >
          <UIcon name="i-lucide-scroll" class="w-3 h-3" />
          Warisan: {{ todo.inherited_from }}
        </span>
      </div>

      <!-- High Score Skills & Success Factors (Penyebab Nilai Tinggi) -->
      <div
        v-if="(todo.high_score_skills && todo.high_score_skills.length) || todo.high_score_notes"
        class="mt-1.5 p-2 rounded-md bg-white/80 dark:bg-neutral-900/60 border border-emerald-200 dark:border-emerald-800/60 space-y-1.5"
      >
        <div class="flex items-center justify-between text-[11px] font-medium text-emerald-900 dark:text-emerald-200">
          <span class="flex items-center gap-1">
            <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5 text-amber-500" />
            Penyebab Nilai Tinggi / Warisan Pengetahuan:
          </span>
          <span v-if="todo.high_score_skills && todo.high_score_skills.length" class="text-[10px] text-neutral-400">
            {{ todo.high_score_skills.length }} skill kunci
          </span>
        </div>

        <div v-if="todo.high_score_skills && todo.high_score_skills.length" class="flex flex-wrap gap-1">
          <span
            v-for="sk in todo.high_score_skills"
            :key="sk"
            class="px-1.5 py-0.5 rounded bg-amber-50 dark:bg-amber-950/40 text-amber-800 dark:text-amber-300 border border-amber-200 dark:border-amber-900 text-[10px] font-mono flex items-center gap-1"
          >
            <UIcon name="i-lucide-zap" class="w-2.5 h-2.5 text-amber-500" />
            {{ sk }}
          </span>
        </div>

        <p v-if="todo.high_score_notes" class="text-[11px] text-neutral-700 dark:text-neutral-300 leading-relaxed">
          {{ todo.high_score_notes }}
        </p>

        <ul v-if="todo.success_factors && todo.success_factors.length" class="list-disc list-inside text-[11px] text-neutral-600 dark:text-neutral-400 space-y-0.5">
          <li v-for="(factor, idx) in todo.success_factors" :key="idx">
            {{ factor }}
          </li>
        </ul>
      </div>
    </div>

    <!-- Actions Footer -->
    <div class="pt-2 border-t border-neutral-100 dark:border-white/10 flex flex-wrap items-center justify-between gap-2">
      <div class="text-[10px] text-neutral-400 font-mono">
        ID #{{ todo.id }} • {{ formatDateTime(todo.created_at) }}
      </div>

      <div class="flex items-center gap-1.5">
        <!-- Approval buttons for suggested AI items -->
        <template v-if="todo.status === 'suggested'">
          <UButton
            size="xs"
            color="success"
            variant="solid"
            icon="i-lucide-check"
            title="Setujui rekomendasi AI ini dan mulai masukkan ke rencana kerja"
            @click="emit('approve', todo.id)"
          >
            Setujui (Approve)
          </UButton>
          <UButton
            size="xs"
            color="neutral"
            variant="ghost"
            icon="i-lucide-x"
            title="Tolak saran AI"
            @click="emit('reject', todo.id)"
          >
            Tolak
          </UButton>
        </template>

        <!-- Standard workflow status buttons -->
        <template v-else>
          <UButton
            v-if="todo.status !== 'completed' && todo.status !== 'rejected'"
            size="xs"
            color="neutral"
            variant="outline"
            icon="i-lucide-check-circle"
            title="Tandai selesai"
            @click="emit('updateStatus', todo.id, 'completed')"
          >
            Selesai
          </UButton>

          <UButton
            v-if="agentRun"
            size="xs"
            color="info"
            variant="soft"
            icon="i-lucide-loader-circle"
            :ui="{ leadingIcon: agentRun.status === 'running' ? 'animate-spin' : '' }"
            :title="agentRun.status === 'running' ? agentRun.route_reason : (agentRun.wait_reason || 'Menunggu worker yang tersedia')"
            to="/queue"
          >
            <template v-if="agentRun.kind === 'merge'">
              {{ agentRun.status === 'running' ? (agentRun.profile_name || 'Agent') + ' me-merge' : 'Antre merge' }}
            </template>
            <template v-else>
              {{ agentRun.status === 'running' ? (agentRun.profile_name || 'Agent') + ' bekerja' : 'Antre di agent' }}
            </template>
          </UButton>
          <UButton
            v-else-if="todo.status !== 'completed' && todo.status !== 'rejected'"
            size="xs"
            color="primary"
            variant="soft"
            icon="i-lucide-bot"
            title="Kirim todo ini ke agent coding di laptop (Claude Code, Antigravity, ...)"
            @click="emit('dispatch', todo)"
          >
            Suruh Agent
          </UButton>

          <UButton
            v-if="todo.status === 'pending'"
            size="xs"
            color="neutral"
            variant="ghost"
            icon="i-lucide-play"
            title="Mulai kerjakan"
            @click="emit('updateStatus', todo.id, 'in_progress')"
          >
            Kerjakan
          </UButton>

          <UButton
            v-if="todo.status === 'completed' || todo.status === 'rejected'"
            size="xs"
            color="neutral"
            variant="ghost"
            icon="i-lucide-rotate-ccw"
            :title="todo.status === 'rejected' ? 'Pulihkan agenda yang ditolak' : 'Buka kembali agenda'"
            @click="emit('updateStatus', todo.id, 'pending')"
          >
            {{ todo.status === 'rejected' ? 'Pulihkan' : 'Buka Lagi' }}
          </UButton>
        </template>

        <UButton
          v-if="canSplit"
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-lucide-split"
          title="Pecah todo gabungan ini kembali menjadi todo biasa"
          @click="emit('split', todo.id)"
        >
          Pecah
        </UButton>

        <!-- Popover Verifikasi Hapus Agenda -->
        <UPopover :content="{ align: 'end', side: 'top' }">
          <UButton
            size="xs"
            color="neutral"
            variant="ghost"
            icon="i-lucide-trash-2"
            class="text-neutral-400 hover:text-red-500 transition-colors"
            title="Hapus agenda"
          />

          <template #content="{ close }">
            <div class="p-3 w-64 space-y-2.5">
              <div class="flex items-start gap-2.5">
                <div class="p-1.5 rounded-lg bg-red-50 dark:bg-red-950/50 text-red-600 dark:text-red-400 shrink-0">
                  <UIcon name="i-lucide-alert-triangle" class="w-4 h-4" />
                </div>
                <div class="space-y-0.5">
                  <p class="text-xs font-semibold text-neutral-900 dark:text-neutral-100">
                    Hapus Agenda?
                  </p>
                  <p class="text-[11px] text-neutral-500 dark:text-neutral-400 leading-snug">
                    {{ isGroup ? 'Todo gabungan dihapus; sub todo yang belum selesai kembali menjadi todo biasa.' : 'Agenda ini akan dihapus permanen dari daftar tugas.' }}
                  </p>
                </div>
              </div>
              <div class="flex items-center justify-end gap-1.5 pt-2 border-t border-neutral-100 dark:border-white/10">
                <UButton
                  size="xs"
                  color="neutral"
                  variant="ghost"
                  label="Batal"
                  @click="close"
                />
                <UButton
                  size="xs"
                  color="error"
                  variant="solid"
                  icon="i-lucide-trash-2"
                  label="Ya, Hapus"
                  @click="() => { close?.(); emit('delete', todo.id); }"
                />
              </div>
            </div>
          </template>
        </UPopover>
      </div>
    </div>
  </div>
</template>
