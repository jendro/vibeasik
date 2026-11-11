<script setup lang="ts">
import type { AgentRun, Todo } from '~/types'

defineProps<{
  filteredTodos: Todo[]
  paginatedTodos: Todo[]
  workTodoCount: number
  agentRunsByTodo: Record<number, AgentRun>
  subTodosByParent: Record<number, Todo[]>
  perPageOptions: number[]
  totalPages: number
}>()
const page = defineModel<number>('page', { required: true })
const perPage = defineModel<number>('perPage', { required: true })
const emit = defineEmits<{
  approve: [id: number]
  reject: [id: number]
  updateStatus: [id: number, status: string]
  delete: [id: number]
  dispatch: [todo: Todo]
  split: [id: number]
}>()
const todosContainerRef = ref<HTMLElement | null>(null)
const setPage = (newPage: number) => {
  page.value = newPage
  if (import.meta.client && todosContainerRef.value) {
    todosContainerRef.value.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}
</script>

<template>
  <div id="todos-list-container" ref="todosContainerRef" class="space-y-4">
    <!-- Top Pagination Toolbar -->
    <div
      class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 px-3.5 py-2 rounded-xl border border-neutral-200/80 dark:border-white/10 bg-neutral-50/70 dark:bg-[#04111f] text-xs"
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
          <span v-if="filteredTodos.length !== workTodoCount" class="text-neutral-400">
            (difilter dari {{ workTodoCount }})
          </span>
        </span>
      </div>

      <div class="flex items-center gap-2.5 self-end sm:self-center flex-wrap">
        <!-- Per Page Selector -->
        <div class="flex items-center gap-1.5 text-neutral-500 dark:text-neutral-400 text-xs">
          <label for="todos-per-page" class="text-xs font-medium">Per hal:</label>
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
        @approve="emit('approve', $event)"
        @reject="emit('reject', $event)"
        @update-status="(id: number, status: string) => emit('updateStatus', id, status)"
        @delete="emit('delete', $event)"
        @dispatch="emit('dispatch', $event)"
        @split="emit('split', $event)"
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
</template>
