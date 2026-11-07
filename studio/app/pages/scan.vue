<script setup lang="ts">
import type { ScanResult } from '~/types'

const { scanDirectory, browseDirectories } = useApi()
const {
  loadingAi,
  llmUsed,
  aiSummary,
  manualGroups,
  existingProjects,
  importing,
  loadExistingProjects,
  runAiGrouping,
  handleAddGroup,
  handleRemoveGroup,
  handleMoveComponent,
  handleCreateNewGroupWithComponent,
  handleToggleSelectAll,
  allSelected,
  totalComponents,
  totalSelected,
  validationError,
  handleConfirmImport
} = useScanGrouping()

const targetPath = ref('')
const scanning = ref(false)
const scanResult = ref<ScanResult | null>(null)
const errorMsg = ref('')
const quickPaths = ref<{ label: string, path: string }[]>([])

onMounted(async () => {
  await loadExistingProjects()
  try {
    const res = await browseDirectories()
    if (res?.current_path) {
      targetPath.value = res.current_path
    }
    if (res?.quick_paths && res.quick_paths.length > 0) {
      quickPaths.value = res.quick_paths.map(qp => ({
        label: qp.name,
        path: qp.path
      }))
    }
  } catch {
    targetPath.value = '.'
  }
})

const handleScan = async () => {
  if (!targetPath.value.trim()) return
  scanning.value = true
  errorMsg.value = ''
  scanResult.value = null
  manualGroups.value = []

  try {
    const res = await scanDirectory(targetPath.value.trim())
    scanResult.value = res
    await loadExistingProjects()
    await runAiGrouping(res)
  } catch (err: any) {
    errorMsg.value = err.data?.error || err.message || 'Scan failed'
    scanResult.value = null
  } finally {
    scanning.value = false
  }
}
</script>

<template>
  <div class="space-y-6 max-w-4xl mx-auto pb-16">
    <!-- Header -->
    <div class="pb-2 border-b border-neutral-200 dark:border-white/10">
      <h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white flex items-center gap-2">
        <UIcon name="i-lucide-folder-search" class="w-6 h-6 text-sky-500" />
        Pindai & Daftarkan ke Pemantauan
      </h1>
      <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1">
        Pindai folder laptop Anda. LLM akan mengenali dan mengelompokkan komponen ke dalam project. Tentukan project serta nama komponen secara manual sebelum didaftarkan ke pemantauan.
      </p>
    </div>

    <!-- Scanner Input Card -->
    <ScanInputCard
      v-model:path="targetPath"
      :scanning="scanning"
      :error-msg="errorMsg"
      :quick-paths="quickPaths"
      @scan="handleScan"
      @select-quick-path="(p) => { targetPath = p; handleScan(); }"
    />

    <!-- Scan Results & Project Grouping Section -->
    <div v-if="scanResult" class="space-y-6">
      <!-- AI Assistant Banner -->
      <ScanAiGroupingBanner
        :loading-ai="loadingAi"
        :llm-used="llmUsed"
        :ai-summary="aiSummary"
        :total-components="totalComponents"
        :all-selected="allSelected"
        @regroup="scanResult ? runAiGrouping(scanResult) : null"
        @add-group="handleAddGroup"
        @toggle-select-all="handleToggleSelectAll"
      />

      <!-- List of Project Groups -->
      <div class="space-y-5">
        <ScanProjectGroupCard
          v-for="(grp, idx) in manualGroups"
          :key="grp.id"
          :group="grp"
          :group-index="idx"
          :existing-projects="existingProjects"
          :all-groups="manualGroups"
          @remove-group="handleRemoveGroup"
          @move-component="handleMoveComponent"
          @create-new-group-with-component="handleCreateNewGroupWithComponent"
        />
      </div>

      <!-- Directory Tree Inspection Preview -->
      <div class="p-5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xs font-semibold uppercase tracking-wider text-neutral-500">
            Struktur Pohon Direktori (Tree)
          </span>
          <span class="text-xs font-mono text-neutral-400">{{ scanResult.root_path }}</span>
        </div>
        <pre class="p-4 rounded-lg bg-[#010810] border border-white/10 text-neutral-200 font-mono text-xs overflow-x-auto max-h-48">{{ scanResult.directory_tree }}</pre>
      </div>

      <!-- Sticky Confirmation Footer -->
      <ScanSummaryFooter
        :total-selected="totalSelected"
        :total-components="totalComponents"
        :total-projects="manualGroups.filter(g => g.components.some(c => c.selected)).length"
        :importing="importing"
        :validation-error="validationError"
        @confirm="scanResult ? handleConfirmImport(scanResult) : null"
        @reset="scanResult = null; manualGroups = []"
      />
    </div>
  </div>
</template>
