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
  <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
    <!-- 1. Hero Card: Sisa Tugas Harus Dikerjakan -->
    <div
      class="p-3.5 rounded-xl border border-amber-300 dark:border-amber-700/80 bg-amber-50/70 dark:bg-amber-950/30 cursor-pointer hover:border-amber-400 dark:hover:border-amber-600 transition-colors"
      title="Klik untuk memfilter semua tugas yang belum selesai"
      @click="emit('filterStatus', 'uncompleted')"
    >
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-1.5 text-xs font-bold text-amber-900 dark:text-amber-200 uppercase tracking-wider">
          <UIcon name="i-lucide-alert-circle" class="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" />
          <span>Sisa Tugas</span>
        </div>
        <span class="text-xs font-semibold px-1.5 py-0.5 rounded-full bg-amber-200/80 dark:bg-amber-900/60 text-amber-900 dark:text-amber-200">
          Perlu Dikerjakan
        </span>
      </div>
      <div class="text-2xl font-bold font-mono text-amber-950 dark:text-amber-100 mt-1">
        {{ stats.remainingCount }}
      </div>
      <div class="text-xs text-amber-800/90 dark:text-amber-300 mt-0.5 font-medium truncate">
        {{ stats.inProgressCount }} jalan • {{ stats.pendingCount }} pending
      </div>
    </div>

    <!-- 2. Sedang Dikerjakan (In Progress) -->
    <div
      class="p-3.5 rounded-xl border border-sky-200/80 dark:border-sky-900/40 bg-sky-50/50 dark:bg-sky-950/20 cursor-pointer hover:border-sky-300 dark:hover:border-sky-800 transition-colors"
      title="Klik untuk melihat tugas yang sedang berjalan"
      @click="emit('filterStatus', 'in_progress')"
    >
      <div class="flex items-center gap-1.5 text-xs font-semibold text-sky-700 dark:text-sky-300">
        <UIcon name="i-lucide-play-circle" class="w-3.5 h-3.5" />
        <span>Sedang Jalan</span>
      </div>
      <div class="text-2xl font-bold font-mono text-sky-900 dark:text-sky-200 mt-1">
        {{ stats.inProgressCount }}
      </div>
      <div class="text-xs text-sky-600/90 dark:text-sky-400 mt-0.5 truncate">
        Dalam proses pengerjaan
      </div>
    </div>

    <!-- 3. Menunggu (Pending) -->
    <div
      class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] cursor-pointer hover:border-neutral-300 dark:hover:border-neutral-700 transition-colors"
      title="Klik untuk melihat tugas pending"
      @click="emit('filterStatus', 'pending')"
    >
      <div class="flex items-center gap-1.5 text-xs font-semibold text-neutral-700 dark:text-neutral-300">
        <UIcon name="i-lucide-clock" class="w-3.5 h-3.5 text-neutral-500" />
        <span>Pending</span>
      </div>
      <div class="text-2xl font-bold font-mono text-neutral-900 dark:text-white mt-1">
        {{ stats.pendingCount }}
      </div>
      <div class="text-xs text-neutral-400 mt-0.5 truncate">
        Antrean siap dieksekusi
      </div>
    </div>

    <!-- 4. Saran AI (Menunggu Approval) -->
    <div
      class="p-3.5 rounded-xl border border-violet-200/60 dark:border-violet-900/40 bg-violet-50/50 dark:bg-violet-950/20 cursor-pointer hover:border-violet-300 dark:hover:border-violet-800 transition-colors"
      title="Klik untuk melihat rekomendasi AI yang menunggu persetujuan"
      @click="emit('filterStatus', 'suggested')"
    >
      <div class="flex items-center gap-1.5 text-xs font-semibold text-violet-700 dark:text-violet-300">
        <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5" />
        <span>Saran AI</span>
      </div>
      <div class="text-2xl font-bold font-mono text-violet-900 dark:text-violet-200 mt-1">
        {{ stats.suggestedCount }}
      </div>
      <div class="text-xs text-violet-600/80 dark:text-violet-400 mt-0.5 truncate">
        Perlu persetujuan
      </div>
    </div>

    <!-- 5. Telah Selesai -->
    <div
      class="p-3.5 rounded-xl border border-sky-200/70 dark:border-sky-900/40 bg-sky-50/40 dark:bg-sky-950/20 cursor-pointer hover:border-sky-300 dark:hover:border-sky-800 transition-colors"
      title="Klik untuk melihat tugas yang sudah diselesaikan"
      @click="emit('filterStatus', 'completed')"
    >
      <div class="flex items-center gap-1.5 text-xs font-semibold text-sky-700 dark:text-sky-300">
        <UIcon name="i-lucide-check-circle-2" class="w-3.5 h-3.5 text-sky-600 dark:text-sky-400" />
        <span>Selesai</span>
      </div>
      <div class="text-2xl font-bold font-mono text-sky-900 dark:text-sky-200 mt-1">
        {{ stats.completedCount }}
      </div>
      <div class="text-xs text-sky-600/90 dark:text-sky-400 mt-0.5 truncate">
        Tervalidasi tuntas
      </div>
    </div>

    <!-- 6. Total & Persentase Progres Proyek -->
    <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f]">
      <div class="flex items-center gap-1.5 text-xs font-semibold text-neutral-600 dark:text-neutral-400">
        <UIcon name="i-lucide-pie-chart" class="w-3.5 h-3.5 text-blue-500" />
        <span>Capaian</span>
      </div>
      <div class="text-2xl font-bold font-mono text-neutral-900 dark:text-white mt-1">
        {{ stats.completionPercent }}%
      </div>
      <div class="text-xs text-neutral-400 mt-0.5 truncate">
        Dari {{ stats.total }} total agenda
      </div>
    </div>
  </div>
</template>
