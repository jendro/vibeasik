<script setup lang="ts">
import type { Finding, ResolutionResponse } from '~/types'
import type { FindingGroup } from '~/utils/findingGroups'

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  projectId: number
  findings: Finding[]
  /** Satukan Serupa aktif: tiap baris = temuan pertama kelompok; seluruh kelompok diselesaikan dengan satu panggilan LLM. */
  groups?: FindingGroup[]
}>()

const emit = defineEmits<{
  (e: 'findingResolved', payload: { finding: Finding, resolution: ResolutionResponse }): void
  (e: 'viewResolution', finding: Finding): void
  (e: 'completed'): void
}>()

const { resolveDiagnosticIssue, resolveDiagnosticGroup } = useApi()

// Batch concurrency setting (Default: 3, optimal sweet-spot)
const batchSize = ref(3)
const batchOptions = [
  { value: 1, label: '1 Temuan', note: 'Sekuensial / Sangat Aman' },
  { value: 3, label: '3 Temuan (Rekomendasi)', note: 'Optimal & Bebas Rate Limit' },
  { value: 5, label: '5 Temuan', note: 'Mode Cepat' }
]

// Execution state
const isExecuting = ref(false)
const isPaused = ref(false)
const isCancelled = ref(false)

interface ItemState {
  status: 'pending' | 'processing' | 'success' | 'error'
  error?: string
  todoId?: number
  outcome?: string
  resolution?: ResolutionResponse
}

const itemStates = ref<Record<string, ItemState>>({})

// Temuan serupa (rule & berkas sama) digabung ke satu todo audit; yang sudah terwakili diabaikan.
const outcomeLabel = (outcome?: string) =>
  outcome === 'merged' ? 'Digabung' : outcome === 'represented' ? 'Sudah terwakili' : outcome === 'rejected' ? 'Pernah ditolak' : 'Selesai'

const getFindingKey = (f: Finding) => f.id || `${f.rule_id || f.title}_${f.target_file || ''}_${f.line || 0}`

const groupOf = computed(() => {
  const m = new Map<string, FindingGroup>()
  for (const g of props.groups || []) {
    if (g.findings[0] && g.findings.length > 1) m.set(getFindingKey(g.findings[0]), g)
  }
  return m
})

// Initialize or sync item states when findings change or modal opens
watch(() => [props.findings, open.value], () => {
  if (!open.value) return
  const current = { ...itemStates.value }
  for (const f of props.findings) {
    const k = getFindingKey(f)
    if (!current[k]) {
      current[k] = {
        status: f.status === 'resolved' ? 'success' : 'pending',
        todoId: f.todo_id
      }
    }
  }
  itemStates.value = current
}, { immediate: true })

// Counters
const totalCount = computed(() => props.findings.length)

const successCount = computed(() => {
  return props.findings.filter(f => itemStates.value[getFindingKey(f)]?.status === 'success').length
})

const errorCount = computed(() => {
  return props.findings.filter(f => itemStates.value[getFindingKey(f)]?.status === 'error').length
})

const processingCount = computed(() => {
  return props.findings.filter(f => itemStates.value[getFindingKey(f)]?.status === 'processing').length
})

const pendingCount = computed(() => {
  return props.findings.filter((f) => {
    const s = itemStates.value[getFindingKey(f)]?.status
    return !s || s === 'pending'
  }).length
})

const progressPercent = computed(() => {
  if (totalCount.value === 0) return 0
  return Math.min(100, Math.round((successCount.value / totalCount.value) * 100))
})

const deepFindingsCount = computed(() => {
  return props.findings.filter(f => f.is_deep || f.rule_id?.startsWith('DEEP-') || f.title?.startsWith('[DEEP]')).length
})

const isAllDone = computed(() => {
  return totalCount.value > 0 && pendingCount.value === 0 && processingCount.value === 0
})

// Execute a specific batch slice
const processBatchItems = async (items: Finding[]) => {
  if (items.length === 0) return

  // Mark all items in slice as processing
  for (const item of items) {
    const k = getFindingKey(item)
    itemStates.value[k] = { status: 'processing' }
  }

  // Execute concurrently
  const promises = items.map(async (finding) => {
    const k = getFindingKey(finding)
    try {
      const group = groupOf.value.get(k)
      const res = group
        ? await resolveDiagnosticGroup(props.projectId, group.findings)
        : await resolveDiagnosticIssue(props.projectId, finding, true)
      itemStates.value[k] = {
        status: 'success',
        todoId: res.created_todo?.id,
        outcome: res.todo_outcome,
        resolution: res
      }
      for (const f of group?.findings || [finding]) {
        applyResolutionToFinding(f, res)
        emit('findingResolved', { finding: f, resolution: res })
      }
    } catch (err: any) {
      itemStates.value[k] = {
        status: 'error',
        error: err.message || 'Gagal menghubungi external LLM.'
      }
    }
  })

  await Promise.allSettled(promises)
}

// Semi-manual: Execute only 1 single batch
const handleRunSingleBatch = async () => {
  if (isExecuting.value || !props.projectId) return

  // Get next batch of pending or failed items
  const pendingItems = props.findings.filter((f) => {
    const st = itemStates.value[getFindingKey(f)]?.status
    return !st || st === 'pending' || st === 'error'
  })

  const batch = pendingItems.slice(0, batchSize.value)
  if (batch.length === 0) return

  isExecuting.value = true
  try {
    await processBatchItems(batch)
    if (pendingCount.value === 0 && errorCount.value === 0) {
      emit('completed')
    }
  } finally {
    isExecuting.value = false
  }
}

// Continuous Progressive Execution
const handleStartAuto = async () => {
  if (isExecuting.value || !props.projectId) return

  isExecuting.value = true
  isPaused.value = false
  isCancelled.value = false

  try {
    while (!isCancelled.value && !isPaused.value) {
      const pendingItems = props.findings.filter((f) => {
        const st = itemStates.value[getFindingKey(f)]?.status
        return !st || st === 'pending'
      })

      if (pendingItems.length === 0) break

      const batch = pendingItems.slice(0, batchSize.value)
      await processBatchItems(batch)

      // Short breathing pause between batches to prevent hammering LLM quota & UI freeze
      const remaining = props.findings.filter((f) => {
        const st = itemStates.value[getFindingKey(f)]?.status
        return !st || st === 'pending'
      })

      if (remaining.length > 0 && !isCancelled.value && !isPaused.value) {
        await new Promise(r => setTimeout(r, 600))
      }
    }

    if (!isCancelled.value && !isPaused.value && pendingCount.value === 0) {
      emit('completed')
    }
  } finally {
    isExecuting.value = false
  }
}

const handlePause = () => {
  isPaused.value = true
}

const handleResume = () => {
  isPaused.value = false
  handleStartAuto()
}

const handleStop = () => {
  isCancelled.value = true
  isExecuting.value = false
  isPaused.value = false
}

const handleRetrySingle = async (finding: Finding) => {
  await processBatchItems([finding])
}

const getSeverityBadgeColor = (sev: string) => {
  switch (sev?.toUpperCase()) {
    case 'CRITICAL':
      return 'error'
    case 'WARNING':
      return 'warning'
    case 'INFO':
      return 'info'
    default:
      return 'neutral'
  }
}
</script>

<template>
  <UModal
    v-model:open="open"
    title="Eksekusi Solusi AI Bertahap (Progressive Batch Resolver)"
    description="Selesaikan semua temuan audit arsitektur, kueri database, keamanan, dan Deep Check simulasi rute secara bertahap dan terkontrol via LLM."
    :ui="{ content: 'sm:max-w-3xl' }"
  >
    <template #body>
      <div class="space-y-5 text-xs">
        <!-- Estimation & Capacity Recommendation Banner -->
        <div class="p-3.5 rounded-xl bg-sky-50/70 dark:bg-sky-950/30 border border-sky-200 dark:border-sky-800/60 space-y-2">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 font-bold text-sm text-neutral-900 dark:text-white">
              <UIcon name="i-lucide-cpu" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
              <span>Estimasi Kapasitas Eksekusi Bersamaan</span>
            </div>
            <UBadge size="xs" color="primary" variant="subtle">
              Kapasitas Optimal: 3 Temuan Sekaligus
            </UBadge>
          </div>

          <p class="text-neutral-600 dark:text-neutral-400 leading-relaxed font-sans">
            Mengeksekusi <strong>3 temuan per batch</strong> adalah perkiraan kapasitas paling ideal: cepat, aman dari batasan laju panggilan (RPM/TPM) provider LLM eksternal, mencegah timeout koneksi jaringan, dan mengeliminasi risiko kunci database (DB lock).
          </p>

          <!-- Concurrency Selector -->
          <div class="pt-2 flex flex-wrap items-center gap-2 border-t border-sky-200/60 dark:border-sky-800/40">
            <span class="text-neutral-600 dark:text-neutral-400 font-semibold">Ukuran Batch:</span>
            <div class="flex items-center gap-1.5">
              <button
                v-for="opt in batchOptions"
                :key="opt.value"
                type="button"
                :disabled="isExecuting"
                class="px-2.5 py-1 rounded-lg text-[11px] font-semibold transition-all border"
                :class="[
                  batchSize === opt.value
                    ? 'bg-primary-600 text-white border-primary-600 shadow-xs'
                    : 'bg-white dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 border-neutral-200 dark:border-neutral-700 hover:border-neutral-300'
                ]"
                @click="batchSize = opt.value"
              >
                {{ opt.label }}
              </button>
            </div>
          </div>
        </div>

        <!-- Progress Overview & Metrics -->
        <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-3">
          <div class="flex items-center justify-between">
            <div class="space-y-0.5">
              <div class="text-[11px] font-bold uppercase tracking-wider text-neutral-400 dark:text-neutral-500">
                Progress Penyelesaian Temuan
              </div>
              <div class="text-sm font-bold text-neutral-900 dark:text-white">
                {{ successCount }} dari {{ totalCount }} temuan selesai diproses ({{ progressPercent }}%)
              </div>
            </div>

            <!-- Mini State Badges -->
            <div class="flex items-center gap-2 font-mono text-[11px] flex-wrap">
              <span class="px-2 py-0.5 rounded bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800">
                Berhasil: {{ successCount }}
              </span>
              <span v-if="deepFindingsCount > 0" class="px-2 py-0.5 rounded bg-sky-50 dark:bg-sky-950/40 text-sky-700 dark:text-sky-300 border border-sky-200 dark:border-sky-800">
                Deep Check: {{ deepFindingsCount }}
              </span>
              <span v-if="errorCount > 0" class="px-2 py-0.5 rounded bg-rose-50 dark:bg-rose-950/40 text-rose-700 dark:text-rose-300 border border-rose-200 dark:border-rose-800">
                Gagal: {{ errorCount }}
              </span>
              <span class="px-2 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 border border-neutral-200 dark:border-neutral-700">
                Tersisa: {{ pendingCount }}
              </span>
            </div>
          </div>

          <!-- Progress Bar -->
          <div class="h-2.5 w-full bg-neutral-100 dark:bg-neutral-800 rounded-full overflow-hidden border border-neutral-200 dark:border-neutral-700">
            <div
              class="h-full bg-primary-600 transition-all duration-300 rounded-full"
              :style="{ width: `${progressPercent}%` }"
            />
          </div>

          <!-- Action Buttons / Semi-Manual Controls -->
          <div class="flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-neutral-100 dark:border-neutral-800">
            <div class="flex items-center gap-2">
              <!-- Mode 1: Step-by-Step Single Batch -->
              <UButton
                v-if="!isExecuting && pendingCount > 0"
                color="neutral"
                variant="outline"
                size="xs"
                icon="i-lucide-play"
                title="Eksekusi hanya 1 batch (semi-manual terkontrol)"
                @click="handleRunSingleBatch"
              >
                Proses 1 Batch ({{ Math.min(batchSize, pendingCount) }} Temuan)
              </UButton>

              <!-- Mode 2: Full Auto Progressive Execution -->
              <UButton
                v-if="!isExecuting && pendingCount > 0"
                color="primary"
                variant="solid"
                size="xs"
                icon="i-lucide-fast-forward"
                title="Jalankan otomatis bertahap sampai selesai dengan jeda antar-batch"
                @click="handleStartAuto"
              >
                Mulai Otomatis Bertahap
              </UButton>

              <!-- Pause Button when running auto -->
              <UButton
                v-if="isExecuting && !isPaused"
                color="warning"
                variant="solid"
                size="xs"
                icon="i-lucide-pause"
                @click="handlePause"
              >
                Jeda (Pause)
              </UButton>

              <!-- Resume Button when paused -->
              <UButton
                v-if="isExecuting && isPaused"
                color="primary"
                variant="solid"
                size="xs"
                icon="i-lucide-play"
                @click="handleResume"
              >
                Lanjutkan (Resume)
              </UButton>

              <!-- Stop Button -->
              <UButton
                v-if="isExecuting"
                color="neutral"
                variant="outline"
                size="xs"
                icon="i-lucide-square"
                @click="handleStop"
              >
                Hentikan
              </UButton>
            </div>

            <div v-if="isExecuting" class="flex items-center gap-1.5 text-primary-600 dark:text-primary-400 font-semibold animate-pulse">
              <UIcon name="i-lucide-loader-2" class="w-3.5 h-3.5 animate-spin" />
              <span>Memproses batch {{ batchSize }} temuan via LLM...</span>
            </div>
            <div v-else-if="isAllDone" class="flex items-center gap-1 text-emerald-600 dark:text-emerald-400 font-semibold">
              <UIcon name="i-lucide-check-circle" class="w-4 h-4" />
              <span>Semua temuan berhasil diproses!</span>
            </div>
          </div>
        </div>

        <!-- Findings Queue List -->
        <div class="space-y-2">
          <div class="flex items-center justify-between text-neutral-500 font-semibold text-[11px] uppercase tracking-wider">
            <span>Daftar Antrean Temuan ({{ totalCount }})</span>
            <span>Status Eksekusi</span>
          </div>

          <div class="max-h-64 overflow-y-auto space-y-2 pr-1 rounded-xl border border-neutral-200 dark:border-neutral-800 p-2 bg-neutral-50/50 dark:bg-neutral-900/50">
            <div
              v-for="finding in findings"
              :key="getFindingKey(finding)"
              class="p-2.5 rounded-lg border bg-white dark:bg-neutral-900 flex items-center justify-between gap-3 transition-colors"
              :class="[
                itemStates[getFindingKey(finding)]?.status === 'processing'
                  ? 'border-primary-400 dark:border-primary-600 ring-1 ring-primary-400/30'
                  : itemStates[getFindingKey(finding)]?.status === 'success'
                    ? 'border-emerald-200 dark:border-emerald-900/40 bg-emerald-50/20'
                    : itemStates[getFindingKey(finding)]?.status === 'error'
                      ? 'border-rose-200 dark:border-rose-900/40 bg-rose-50/20'
                      : 'border-neutral-200 dark:border-neutral-800'
              ]"
            >
              <!-- Info Title & Target -->
              <div class="min-w-0 flex-1 space-y-1">
                <div class="flex items-center gap-2">
                  <UBadge size="xs" :color="getSeverityBadgeColor(finding.severity)" variant="subtle">
                    {{ finding.severity }}
                  </UBadge>
                  <span
                    v-if="finding.rule_id"
                    class="px-1.5 py-0.5 rounded text-[10px] font-mono font-bold bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300"
                  >
                    {{ finding.rule_id }}
                  </span>
                  <span
                    v-if="finding.is_deep || finding.rule_id?.startsWith('DEEP-') || finding.title?.startsWith('[DEEP]')"
                    class="px-1.5 py-0.5 rounded text-[10px] font-bold bg-sky-100 dark:bg-sky-950/60 text-sky-700 dark:text-sky-300 border border-sky-200 dark:border-sky-800 shrink-0 uppercase"
                  >
                    Deep Check
                  </span>
                  <span class="font-bold text-neutral-900 dark:text-white truncate">
                    {{ groupOf.get(getFindingKey(finding))?.title || finding.title }}
                  </span>
                </div>

                <div v-if="groupOf.get(getFindingKey(finding))" class="flex items-center gap-1.5 text-[10px] text-primary-600 dark:text-primary-400 font-semibold">
                  <UIcon name="i-lucide-list-tree" class="w-3 h-3" />
                  <span>{{ groupOf.get(getFindingKey(finding))?.findings.length }} temuan serupa · {{ groupOf.get(getFindingKey(finding))?.fileCount }} berkas — satu panggilan LLM, satu todo</span>
                </div>
                <div v-else-if="finding.target_file" class="flex items-center gap-1.5 text-[10px] text-neutral-500 font-mono">
                  <UIcon name="i-lucide-file-code" class="w-3 h-3 text-neutral-400" />
                  <span class="truncate">{{ finding.target_file }}</span>
                  <span v-if="finding.line">L{{ finding.line }}</span>
                </div>
              </div>

              <!-- Status Action Pill -->
              <div class="shrink-0 flex items-center gap-2">
                <!-- Status: Success -->
                <template v-if="itemStates[getFindingKey(finding)]?.status === 'success'">
                  <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-semibold bg-emerald-100 dark:bg-emerald-950/60 text-emerald-800 dark:text-emerald-300 border border-emerald-300 dark:border-emerald-800">
                    <UIcon name="i-lucide-check" class="w-3 h-3" />
                    {{ outcomeLabel(itemStates[getFindingKey(finding)]?.outcome) }}
                    <span v-if="itemStates[getFindingKey(finding)]?.todoId" class="font-mono">
                      (TODO #{{ itemStates[getFindingKey(finding)]?.todoId }})
                    </span>
                  </span>
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="ghost"
                    icon="i-lucide-eye"
                    title="Lihat Solusi"
                    @click="emit('viewResolution', finding)"
                  />
                </template>

                <!-- Status: Processing -->
                <template v-else-if="itemStates[getFindingKey(finding)]?.status === 'processing'">
                  <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[10px] font-semibold bg-primary-50 dark:bg-primary-950/60 text-primary-700 dark:text-primary-300 border border-primary-300 dark:border-primary-800">
                    <UIcon name="i-lucide-loader-2" class="w-3 h-3 animate-spin" />
                    Memproses...
                  </span>
                </template>

                <!-- Status: Error -->
                <template v-else-if="itemStates[getFindingKey(finding)]?.status === 'error'">
                  <span
                    class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-semibold bg-rose-100 dark:bg-rose-950/60 text-rose-800 dark:text-rose-300 border border-rose-300 dark:border-rose-800"
                    :title="itemStates[getFindingKey(finding)]?.error"
                  >
                    <UIcon name="i-lucide-alert-circle" class="w-3 h-3" />
                    Gagal
                  </span>
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="ghost"
                    icon="i-lucide-refresh-cw"
                    title="Coba Lagi"
                    :disabled="isExecuting"
                    @click="handleRetrySingle(finding)"
                  />
                </template>

                <!-- Status: Pending -->
                <template v-else>
                  <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-semibold bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 border border-neutral-200 dark:border-neutral-700">
                    <UIcon name="i-lucide-clock" class="w-3 h-3" />
                    Menunggu
                  </span>
                </template>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-between w-full">
        <UButton color="neutral" variant="outline" size="sm" @click="open = false">
          {{ isAllDone ? 'Tutup & Kembali ke Laporan' : 'Tutup' }}
        </UButton>

        <div class="flex items-center gap-2">
          <UButton
            v-if="!isExecuting && pendingCount > 0"
            color="primary"
            variant="solid"
            size="sm"
            icon="i-lucide-fast-forward"
            @click="handleStartAuto"
          >
            Mulai Eksekusi Bertahap
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
