<script setup lang="ts">
import type { RuntimeSimSession } from '~/types'

// Ringkasan Uji Runtime terakhir di Laporan Audit; pengujian lengkap ada di halaman /runtime-test.
const props = defineProps<{
  session: RuntimeSimSession | null
  projectId: number | null
}>()

const emit = defineEmits<{
  (e: 'showFindings'): void
}>()

const running = computed(() => props.session?.status === 'running')
const openFindings = computed(() => (props.session?.findings || []).filter(f => f.status !== 'resolved' && f.status !== 'ignored').length)
const target = computed(() => props.projectId ? `/runtime-test?project_id=${props.projectId}` : '/runtime-test')
const scoreColor = (score: number) => score >= 90 ? 'text-emerald-600 dark:text-emerald-400' : score >= 70 ? 'text-amber-600 dark:text-amber-400' : 'text-rose-600 dark:text-rose-400'
</script>

<template>
  <section class="p-4 bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 rounded-2xl shadow-xs flex flex-col md:flex-row md:items-center justify-between gap-3">
    <div class="flex items-start gap-3 min-w-0">
      <span class="p-2 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800 shrink-0">
        <UIcon name="i-lucide-laptop" class="w-5 h-5" />
      </span>
      <div class="min-w-0 space-y-0.5">
        <div class="flex items-center gap-2 flex-wrap text-sm font-bold text-neutral-900 dark:text-white">
          Uji Runtime di Laptop
          <span v-if="running" class="inline-flex items-center gap-1 text-xs font-bold text-primary-600 dark:text-primary-400"><UIcon name="i-lucide-loader-2" class="w-3 h-3 animate-spin" />sedang berjalan</span>
          <span v-else-if="session" class="text-xs font-normal text-neutral-500">skor <strong class="font-mono" :class="scoreColor(session.score)">{{ session.score }}/100</strong> · {{ new Date(session.started_at).toLocaleString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) }}</span>
        </div>
        <p class="text-xs text-neutral-500 dark:text-neutral-400 leading-relaxed">
          {{ session?.summary || 'Jalankan aplikasi sungguhan: uji semua endpoint API (tanpa & dengan login), ukur kecepatan halaman, dan screenshot di mobile/tablet/desktop.' }}
        </p>
      </div>
    </div>
    <div class="flex items-center gap-2 shrink-0">
      <UButton v-if="openFindings > 0" size="sm" color="warning" variant="soft" icon="i-lucide-list-filter" @click="emit('showFindings')">
        {{ openFindings }} temuan Runtime
      </UButton>
      <UButton size="sm" color="primary" icon="i-lucide-arrow-right" trailing :to="target">
        Buka Uji Runtime
      </UButton>
    </div>
  </section>
</template>
