// Domain API "todos" — dipecah dari useApi.ts tanpa mengubah kontrak HTTP.
import { useApiBase } from './base'

export const useTodosApi = () => {
  const apiBase = useApiBase()

  const getTodos = async (params?: { project_id?: number, component_id?: number, level?: string, category?: string, status?: string, source?: string, is_ai_suggestion?: boolean, limit?: number, offset?: number }): Promise<Todo[]> => {
    const query = new URLSearchParams()
    if (params?.project_id) query.append('project_id', String(params.project_id))
    if (params?.component_id) query.append('component_id', String(params.component_id))
    if (params?.level) query.append('level', params.level)
    if (params?.category) query.append('category', params.category)
    if (params?.status) query.append('status', params.status)
    if (params?.source) query.append('source', params.source)
    if (params?.is_ai_suggestion !== undefined) query.append('is_ai_suggestion', String(params.is_ai_suggestion))
    if (params?.limit) query.append('limit', String(params.limit))
    if (params?.offset !== undefined) query.append('offset', String(params.offset))
    const url = `${apiBase}/api/todos${query.toString() ? '?' + query.toString() : ''}`
    return await $fetch<Todo[]>(url)
  }

  const getTodo = async (id: number): Promise<Todo> => {
    return await $fetch<Todo>(`${apiBase}/api/todos/${id}`)
  }

  const createTodo = async (todo: Partial<Todo>): Promise<Todo> => {
    return await $fetch<Todo>(`${apiBase}/api/todos`, {
      method: 'POST',
      body: todo
    })
  }

  const updateTodo = async (id: number, todo: Partial<Todo>): Promise<Todo> => {
    return await $fetch<Todo>(`${apiBase}/api/todos/${id}`, {
      method: 'PUT',
      body: todo
    })
  }

  const deleteTodo = async (id: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/todos/${id}`, {
      method: 'DELETE'
    })
  }

  const suggestTodos = async (projectId?: number, componentId?: number, saveToDB = true): Promise<Todo[]> => {
    const res = await $fetch<{ count?: number, new_count?: number, saved_to_db?: boolean, suggestions?: Todo[] } | Todo[]>(`${apiBase}/api/todos/suggest`, {
      method: 'POST',
      body: {
        project_id: projectId,
        component_id: componentId,
        save_to_db: saveToDB
      }
    })
    if (res && 'suggestions' in res && Array.isArray(res.suggestions)) {
      return res.suggestions
    }
    if (Array.isArray(res)) {
      return res
    }
    return []
  }

  /** Saran AI: saran aturan instan + rapat Project Manager di antrean untuk proyek (bila PM aktif). */

  const requestTodoSuggestions = async (projectId: number): Promise<{ suggestions: Todo[], new_count?: number, pm_session?: PlanningSession | null, pm_job_id?: string, pm_error?: string }> => {
    const res = await $fetch<{ suggestions?: Todo[], new_count?: number, pm_session?: PlanningSession | null, pm_job_id?: string, pm_error?: string }>(`${apiBase}/api/todos/suggest`, {
      method: 'POST',
      body: { project_id: projectId, save_to_db: true }
    })
    return { ...res, suggestions: res.suggestions ?? [] }
  }

  /** Lebur todo serupa (Aturan 43): dry_run = pratinjau kelompok tanpa mengubah todo. */

  const mergeTodos = async (body: { project_id?: number, todo_ids?: number[], dry_run?: boolean }): Promise<TodoMergeResult> => {
    return await $fetch<TodoMergeResult>(`${apiBase}/api/todos/merge`, { method: 'POST', body })
  }

  /** Satukan Serupa laporan audit: semua todo satu kelompok temuan → satu todo gabungan (tanpa batas sub todo). */

  const groupTodos = async (todoIds: number[]): Promise<TodoMergeGroup> => {
    return await $fetch<TodoMergeGroup>(`${apiBase}/api/todos/group`, { method: 'POST', body: { todo_ids: todoIds } })
  }

  /** Pecah todo gabungan yang belum dikerjakan kembali menjadi todo biasa. */

  const splitTodoGroup = async (id: number): Promise<{ status: string, todo_ids: number[] }> => {
    return await $fetch<{ status: string, todo_ids: number[] }>(`${apiBase}/api/todos/${id}/split`, { method: 'POST' })
  }

  const approveTodo = async (id: number): Promise<Todo> => {
    return await $fetch<Todo>(`${apiBase}/api/todos/${id}/approve`, {
      method: 'POST'
    })
  }

  const rejectTodo = async (id: number): Promise<Todo> => {
    return await $fetch<Todo>(`${apiBase}/api/todos/${id}/reject`, {
      method: 'POST'
    })
  }

  return {
    getTodos,
    getTodo,
    createTodo,
    updateTodo,
    deleteTodo,
    suggestTodos,
    requestTodoSuggestions,
    mergeTodos,
    groupTodos,
    splitTodoGroup,
    approveTodo,
    rejectTodo
  }
}
