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

const showRationale = ref(false)
const isExpanded = ref(false)
const isLongDescription = computed(() => (props.todo.description || '').length > 280)

const categoryMeta = computed(() => todoCategoryMeta(props.todo))
</script>

<template>
  <div
    class="p-4 rounded-xl border bg-white dark:bg-[#04111f] transition-all space-y-3"
    :class="[
      todo.status === 'suggested'
        ? 'border-amber-300/80 dark:border-amber-900/60 ring-1 ring-amber-400/20'
        : 'border-neutral-200 dark:border-white/10 hover:border-neutral-300 dark:hover:border-neutral-700'
    ]"
  >
    <!-- Top Metadata Bar -->
    <TodosTodoCardMetaBar :todo="todo" />

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
            class="text-xs font-medium text-sky-600 dark:text-sky-400 hover:underline flex items-center gap-1 cursor-pointer"
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
        class="inline-flex items-center gap-1 text-xs text-amber-700 dark:text-amber-400 hover:underline font-medium"
        @click="showRationale = !showRationale"
      >
        <UIcon :name="showRationale ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="w-3.5 h-3.5" />
        {{ showRationale ? 'Sembunyikan Analisis & Patokan AI' : 'Lihat Analisis & Patokan Terukur AI' }}
      </button>

      <div
        v-if="showRationale"
        class="mt-2 p-3 rounded-lg bg-amber-50/70 dark:bg-amber-950/20 border border-amber-200/60 dark:border-amber-900/40 text-neutral-700 dark:text-neutral-300 text-xs space-y-1.5"
      >
        <div class="font-medium text-amber-800 dark:text-amber-300 flex items-center justify-between gap-1.5 text-xs">
          <div class="flex items-center gap-1.5">
            <UIcon name="i-lucide-compass" class="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" />
            <span>Dasar Patokan (Rubrik): <strong>{{ todo.guideline_ref || 'STANDAR' }}</strong></span>
          </div>
          <span class="text-xs font-semibold px-2 py-0.5 rounded-full bg-amber-200/70 dark:bg-amber-900/60 text-amber-900 dark:text-amber-200">
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
    <TodosTodoCardCompletion v-if="todo.status === 'completed'" :todo="todo" />

    <!-- Actions Footer -->
    <TodosTodoCardActions
      :todo="todo"
      :agent-run="agentRun"
      @approve="emit('approve', $event)"
      @reject="emit('reject', $event)"
      @update-status="(id, status) => emit('updateStatus', id, status)"
      @delete="emit('delete', $event)"
      @dispatch="emit('dispatch', $event)"
      @split="emit('split', $event)"
    />
  </div>
</template>
