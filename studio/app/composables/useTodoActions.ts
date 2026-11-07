import type { TodoStatus } from '~/types'

// Aksi pada satu todo (setujui, tolak, pecah, ubah status, hapus); data dimuat ulang lewat reload.
export function useTodoActions(reload: () => Promise<void>) {
  const toast = useToast()
  const { approveTodo, rejectTodo, updateTodo, deleteTodo, splitTodoGroup } = useApi()

  const handleApprove = async (id: number) => {
    try {
      await approveTodo(id)
      await reload()
    } catch (err) {
      console.error('Failed to approve todo:', err)
    }
  }

  const handleReject = async (id: number) => {
    try {
      await rejectTodo(id)
      await reload()
    } catch (err) {
      console.error('Failed to reject todo:', err)
    }
  }

  const handleSplit = async (id: number) => {
    try {
      const res = await splitTodoGroup(id)
      toast.add({ title: 'Todo gabungan dipecah', description: `${res.todo_ids.length} sub todo kembali menjadi todo biasa.`, icon: 'i-lucide-split', color: 'success' })
      await reload()
    } catch (err: unknown) {
      const e = err as { data?: { error?: string }, message?: string }
      toast.add({ title: 'Gagal memecah todo gabungan', description: e?.data?.error || e?.message || 'Error', icon: 'i-lucide-alert-circle', color: 'error' })
    }
  }

  const handleUpdateStatus = async (id: number, status: string) => {
    try {
      await updateTodo(id, { status: status as TodoStatus })
      await reload()
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
      await reload()
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

  return { handleApprove, handleReject, handleSplit, handleUpdateStatus, handleDelete }
}
