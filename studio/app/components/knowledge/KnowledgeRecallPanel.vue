<script setup lang="ts">
import type { Employee, KnowledgeHit, KnowledgeRecallResult } from '~/types'

// Uji temu-kembali: tulis pekerjaan (judul todo + berkas target), lihat persis butir yang akan diterima agent
// beserta skornya. Tidak mencatat pemakaian.
const props = defineProps<{ employees: Employee[], recall: (text: string, employeeId?: number) => Promise<KnowledgeRecallResult | null> }>()
const emit = defineEmits<{
  (e: 'open', n: KnowledgeHit): void
  (e: 'feedback', n: KnowledgeHit, helpful: boolean): void
  (e: 'retire', n: KnowledgeHit): void
}>()

const text = ref('Perbaiki handler upload di internal/api/upload.go: goroutine tidak berhenti & error query diabaikan')
const employeeId = ref<number>(0)
const busy = ref(false)
const result = ref<KnowledgeRecallResult | null>(null)
const employeeItems = computed(() => [
  { label: 'Tanpa pegawai (pustaka bersama saja)', value: 0 },
  ...props.employees.filter(e => e.division !== 'executive').map(e => ({ label: `${e.name} — ${e.title}`, value: e.id }))
])

const run = async () => {
  if (!text.value.trim()) return
  busy.value = true
  result.value = await props.recall(text.value, employeeId.value)
  busy.value = false
}
</script>

<template>
  <div class="space-y-4">
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-3">
      <p class="text-xs text-neutral-500">
        Skor = 0.55 relevansi (vektor TF-IDF + bonus domain) + 0.20 utilitas (bukti ✓/✗) + 0.15 bobot + 0.10 kebaruan. Patokan ekosistem lain disaring; tetangga bertaut ikut dipertimbangkan.
      </p>
      <UTextarea v-model="text" :rows="3" autoresize class="w-full" placeholder="Judul & keterangan todo, berkas target, atau pertanyaan rapat…" />
      <div class="flex flex-col sm:flex-row gap-2">
        <USelect v-model="employeeId" :items="employeeItems" class="w-full sm:w-80" />
        <UButton icon="i-lucide-search" :loading="busy" @click="run">
          Uji temu-kembali
        </UButton>
      </div>
    </div>

    <div v-if="result" class="space-y-2">
      <div class="flex flex-wrap items-center gap-1.5 text-xs text-neutral-500">
        <span>Domain dikenali:</span>
        <UBadge v-for="d in result.domains || []" :key="d" size="xs" color="neutral" variant="outline">
          {{ d }}
        </UBadge>
        <span v-if="!(result.domains || []).length">—</span>
      </div>
      <p v-if="!(result.hits || []).length" class="text-sm text-neutral-500">
        Tidak ada butir yang cukup relevan.
      </p>
      <KnowledgeNodeCard
        v-for="h in result.hits || []"
        :key="h.id"
        :node="h"
        show-score
        @open="emit('open', $event)"
        @feedback="(n, ok) => emit('feedback', n, ok)"
        @retire="emit('retire', $event)"
      />
    </div>
  </div>
</template>
