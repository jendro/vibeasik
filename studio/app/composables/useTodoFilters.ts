import type { Ref } from 'vue'
import type { Todo } from '~/types'

// Filter, urutan & paginasi daftar todo proyek aktif.
export function useTodoFilters(workTodos: Ref<Todo[]>, activeProjectId: Ref<number | null>) {
  const selectedLevel = ref<string>('all')
  const selectedCategory = ref<string>('all')
  const selectedCreator = ref<string>('all')
  const selectedTag = ref<string>('all')
  // Default filter status: 'uncompleted' untuk langsung menghide yang selesai dan memperlihatkan sisa tugas
  const selectedStatus = ref<string>('uncompleted')
  const selectedOrderBy = ref<TodoOrderBy>('priority_desc')
  const searchQuery = ref('')

  const setStatus = (status: string) => {
    selectedStatus.value = status
  }

  const filteredTodos = computed(() => filterAndSortTodos(workTodos.value, {
    level: selectedLevel.value,
    category: selectedCategory.value,
    creator: selectedCreator.value,
    tag: selectedTag.value,
    status: selectedStatus.value,
    orderBy: selectedOrderBy.value,
    search: searchQuery.value
  }))

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

  return {
    selectedLevel, selectedCategory, selectedCreator, selectedTag, selectedStatus, selectedOrderBy, searchQuery,
    setStatus, filteredTodos, page, perPage, perPageOptions, totalPages, paginatedTodos
  }
}
