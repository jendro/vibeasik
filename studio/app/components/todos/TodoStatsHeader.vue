<script setup lang="ts">
import type { Todo } from '~/types'

const props = defineProps<{
  todos: Todo[]
  projectName?: string
  // Jumlah per status dari engine (tanpa batas halaman). Bila ada, dipakai menggantikan hitungan baris termuat.
  counts?: Record<string, number> | null
}>()

const emit = defineEmits<{
  (e: 'filterStatus', status: string): void
}>()

const stats = computed(() => {
  const c = props.counts
  if (c) {
    const n = (k: string) => c[k] || 0
    const total = Object.values(c).reduce((a, b) => a + b, 0)
    const completedCount = n('completed')
    const rejectedCount = n('rejected')
    return {
      total,
      remainingCount: total - completedCount - rejectedCount,
      inProgressCount: n('in_progress'),
      pendingCount: n('pending'),
      suggestedCount: n('suggested'),
      completedCount,
      rejectedCount,
      completionPercent: total > 0 ? Math.round((completedCount / total) * 100) : 0
    }
  }
  const total = props.todos.length
  const completedCount = props.todos.filter(t => t.status === 'completed').length
  const rejectedCount = props.todos.filter(t => t.status === 'rejected').length
  // Sisa tugas yang harus dikerjakan: segala status yang belum selesai dan belum ditolak
  const remainingCount = props.todos.filter(t => t.status !== 'completed' && t.status !== 'rejected').length
  const inProgressCount = props.todos.filter(t => t.status === 'in_progress').length
  const pendingCount = props.todos.filter(t => t.status === 'pending').length
  const suggestedCount = props.todos.filter(t => t.status === 'suggested').length
  const completionPercent = total > 0 ? Math.round((completedCount / total) * 100) : 0

  return {
    total,
    remainingCount,
    inProgressCount,
    pendingCount,
    suggestedCount,
    completedCount,
    rejectedCount,
    completionPercent
  }
})
</script>

<template>
  <AppStatStrip
    :items="[
      { label: 'Sisa pekerjaan', value: stats.remainingCount, hint: `${stats.inProgressCount} jalan · ${stats.pendingCount} menunggu`, tone: stats.remainingCount ? 'text-amber-600 dark:text-amber-400' : undefined, action: () => emit('filterStatus', 'uncompleted') },
      { label: 'Sedang dikerjakan', value: stats.inProgressCount, hint: 'Sedang diproses agent', dot: 'bg-sky-500', action: () => emit('filterStatus', 'in_progress') },
      { label: 'Menunggu', value: stats.pendingCount, hint: 'Siap dikerjakan', dot: 'bg-neutral-400', action: () => emit('filterStatus', 'pending') },
      { label: 'Saran AI', value: stats.suggestedCount, hint: 'Perlu persetujuan', dot: 'bg-violet-500', action: () => emit('filterStatus', 'suggested') },
      { label: 'Selesai', value: stats.completedCount, hint: stats.rejectedCount ? `${stats.rejectedCount} ditolak` : 'Tervalidasi tuntas', dot: 'bg-emerald-500', action: () => emit('filterStatus', 'completed') },
      { label: 'Capaian', value: `${stats.completionPercent}%`, hint: `dari ${stats.total} todo` }
    ]"
  />
</template>
