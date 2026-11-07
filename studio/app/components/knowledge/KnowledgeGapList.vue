<script setup lang="ts">
import type { KnowledgeGap } from '~/types'
import { GAP_STATUS } from '~/utils/knowledge'

// Celah pengetahuan: ekosistem/metode yang muncul di pekerjaan tetapi belum dikenal kantor. Celah terbuka
// otomatis menjadi agenda belajar pegawai yang paling cocok; hasilnya masuk pustaka bersama.
defineProps<{ gaps: KnowledgeGap[] }>()
const emit = defineEmits<{
  (e: 'report', body: { topic: string, domain?: string }): void
  (e: 'dismiss', g: KnowledgeGap): void
}>()

const form = reactive({ topic: '', domain: '' })
const submit = () => {
  if (!form.topic.trim()) return
  emit('report', { topic: form.topic.trim(), domain: form.domain.trim() || undefined })
  Object.assign(form, { topic: '', domain: '' })
}
const when = (iso?: string) => iso ? new Date(iso).toLocaleString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) : ''
</script>

<template>
  <div class="space-y-4">
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-2">
      <p class="text-sm font-semibold text-neutral-900 dark:text-white">
        Laporkan hal baru yang perlu dipelajari kantor
      </p>
      <p class="text-xs text-neutral-500">
        Ekosistem tanpa satu node pun terdeteksi otomatis saat agent bekerja. Pakai form ini untuk metode/fitur baru di ekosistem yang sudah dikenal.
      </p>
      <div class="flex flex-col sm:flex-row gap-2">
        <UInput v-model="form.topic" placeholder="mis. Go 1.26 range-over-func iterator" class="flex-1" />
        <UInput v-model="form.domain" placeholder="domain (opsional): go, web, gleam…" class="w-full sm:w-56" />
        <UButton icon="i-lucide-graduation-cap" @click="submit">
          Catat celah
        </UButton>
      </div>
    </div>

    <p v-if="!gaps.length" class="text-sm text-neutral-500">
      Belum ada celah pengetahuan.
    </p>
    <div v-for="g in gaps" :key="g.id" class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <div class="flex items-center gap-1.5 flex-wrap">
            <UBadge size="xs" :color="GAP_STATUS[g.status].color" variant="subtle">
              {{ GAP_STATUS[g.status].label }}
            </UBadge>
            <UBadge size="xs" color="neutral" variant="outline">
              {{ g.domain || 'umum' }}
            </UBadge>
            <UBadge v-if="g.auto" size="xs" color="secondary" variant="soft" icon="i-lucide-radar">
              Terdeteksi otomatis
            </UBadge>
            <span class="text-[11px] text-neutral-400">muncul {{ g.hits }}× · {{ when(g.updated_at) }}</span>
          </div>
          <p class="mt-1 text-sm font-semibold text-neutral-900 dark:text-white">
            {{ g.topic }}
          </p>
          <p v-if="g.employee_name" class="text-xs text-neutral-500">
            Dipelajari {{ g.employee_name }}<span v-if="g.agenda_id"> · agenda #{{ g.agenda_id }}</span><span v-if="g.attempts"> · percobaan ke-{{ g.attempts + 1 }}</span>
          </p>
          <p v-if="g.status === 'learned'" class="text-xs text-emerald-600 dark:text-emerald-400">
            {{ g.nodes_learned }} node baru masuk pustaka bersama
          </p>
          <p v-if="g.note" class="text-xs text-neutral-500 mt-0.5 line-clamp-2">
            {{ g.note }}
          </p>
        </div>
        <UButton v-if="g.status === 'open' || g.status === 'learning'" size="xs" color="neutral" variant="ghost" icon="i-lucide-x" @click="emit('dismiss', g)">
          Abaikan
        </UButton>
      </div>
    </div>
  </div>
</template>
