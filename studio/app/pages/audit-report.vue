<script setup lang="ts">
definePageMeta({
  fluid: true
})

const route = useRoute()
const {
  projects,
  activeProjectId,
  activeProject,
  fetchProjects,
  selectActiveProject
} = useActiveProject()

// Laporan (memuat, audit, deep check) + filter temuan + aksi temuan: state & logika ada di composable.
const {
  reportData, loading, diagnosing, deepChecking, latestDeepCheck, resolvedFindingKeys, currentScore,
  llmReviewJob, auditSweepJob, loadAuditReport, handleRunAudit, handleRunDeepCheck, handleDownloadDeepCheck
} = useAuditReport()

const {
  activeCategory, activeSeverity, activeStatus, searchQuery,
  pendingActionableFindings, pendingActionableCount, isFindingResolved, isFindingIgnored,
  statusCounts, categoryCounts, severityCounts, filteredFindings,
  groupSimilar, findingGroups, mergeableGroupCount, toggleGroupSimilar, GROUP_PREF_KEY
} = useAuditFindingFilters(reportData, resolvedFindingKeys)

const {
  resolutionModalOpen, batchModalOpen, activeResolution, modalLoading, resolvingFindingKey,
  feedbackModalOpen, feedbackFinding, feedbackVerdict, feedbackAutoVerify, applyFeedbackResult,
  batchGroups, batchFindings,
  handleResolveFinding, handleViewResolution, handleIgnoreFinding, handleFindingFeedback, handleDeleteFinding,
  handleResolveAll, handleBatchItemResolved, handleResolveGroup, handleBatchCompleted
} = useAuditFindingActions({
  reportData, resolvedFindingKeys, pendingActionableFindings, pendingActionableCount,
  isFindingResolved, isFindingIgnored, groupSimilar, loadAuditReport
})

const resolvingAll = ref(false)

// Audit & Selesaikan Semua: dari tombol header, tombol di dialog batch, atau otomatis setelah batch selesai.
const AUTO_SWEEP_KEY = 'vibeasik.audit.autoSweep'
const autoSweep = ref(true)
watch(autoSweep, (v) => {
  try {
    localStorage.setItem(AUTO_SWEEP_KEY, v ? '1' : '0')
  } catch {
    // penyimpanan browser tidak tersedia — pilihan tidak diingat
  }
})
// Toast "Masuk antrean" & progres sudah diurus useBackgroundJobs.
const handleRunSweep = async () => {
  if (!auditSweepJob.busy.value) await auditSweepJob.start()
}
const handleBatchDone = async () => {
  await handleBatchCompleted()
  if (autoSweep.value) await handleRunSweep()
}
const criteriaDrawerOpen = ref(false)
const simulationDrawerOpen = ref(false)

// Uji Runtime kini halaman sendiri (/runtime-test); di sini cukup ringkasan hasil terakhir. Temuan
// RUNTIME-* sudah digabung engine ke diagnostic terbaru.
const { session: runtimeSession, load: loadRuntimeSimulation } = useRuntimeSimulation(async () => {
  if (activeProjectId.value) await loadAuditReport(activeProjectId.value, false)
})

const handleShowRuntimeFindings = () => {
  activeCategory.value = 'runtime'
  activeStatus.value = 'active'
}

const handleSwitchProject = (newId: number) => {
  // Pemuatan ulang diurus watch(activeProjectId) di bawah.
  selectActiveProject(newId)
}

// Proyek diganti dari navbar (atau halaman lain): muat laporan proyek itu, jangan tampilkan temuan proyek lama.
let mounted = false
watch(activeProjectId, (id, old) => {
  if (!mounted || !id || id === old) return
  loadAuditReport(id, false)
  loadRuntimeSimulation(id)
})

onMounted(async () => {
  try {
    groupSimilar.value = localStorage.getItem(GROUP_PREF_KEY) === '1'
    autoSweep.value = localStorage.getItem(AUTO_SWEEP_KEY) !== '0'
  } catch {
    // penyimpanan browser tidak tersedia — preferensi tidak diingat
  }
  await fetchProjects()

  // Parse project_id from query if available
  const queryPId = route.query.project_id ? Number(route.query.project_id) : null
  if (queryPId) {
    selectActiveProject(queryPId)
  } else if (!activeProjectId.value && projects.value.length > 0 && projects.value[0]) {
    selectActiveProject(projects.value[0].id)
  }

  if (activeProjectId.value) {
    await Promise.all([
      loadAuditReport(activeProjectId.value),
      loadRuntimeSimulation(activeProjectId.value)
    ])
  } else {
    loading.value = false
  }
  mounted = true
})
</script>

<template>
  <div class="space-y-6 w-full max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 pt-6 pb-16">
    <!-- Header -->
    <DiagnosticsAuditHeader
      :project="activeProject"
      :projects="projects"
      :score="currentScore"
      :loading="loading"
      :diagnosing="diagnosing"
      :deep-checking="deepChecking"
      :resolving-all="resolvingAll"
      :pending-count="pendingActionableCount"
      :deep-pending-count="categoryCounts['deep'] || 0"
      :last-deep-check-at="latestDeepCheck?.last_deep_check_at || activeProject?.last_deep_check_at"
      :last-deep-check-score="latestDeepCheck?.score ?? activeProject?.last_deep_check_score"
      :has-changes-since-deep-check="latestDeepCheck?.has_changes_since_last_deep_check"
      :change-summary="latestDeepCheck?.change_summary"
      :has-last-deep-check="Boolean(latestDeepCheck?.has_report || activeProject?.last_deep_check_at)"
      :llm-review-busy="llmReviewJob.busy.value"
      :llm-review-label="llmReviewJob.label.value"
      :sweep-busy="auditSweepJob.busy.value"
      :sweep-label="auditSweepJob.label.value"
      @switch-project="handleSwitchProject"
      @run-audit="handleRunAudit"
      @run-deep-check="handleRunDeepCheck"
      @download-deep-check="handleDownloadDeepCheck"
      @open-simulations="simulationDrawerOpen = true"
      @resolve-all="handleResolveAll"
      @open-criteria="criteriaDrawerOpen = true"
      @run-llm-review="llmReviewJob.start()"
      @run-sweep="handleRunSweep"
    />

    <!-- Uji Runtime: ringkasan hasil terakhir, pengujian lengkap di halaman /runtime-test -->
    <DiagnosticsRuntimeSimLinkCard
      :session="runtimeSession"
      :project-id="activeProjectId"
      @show-findings="handleShowRuntimeFindings"
    />

    <!-- Metric Summary Cards -->
    <DiagnosticsAuditMetricCards
      :category-counts="categoryCounts"
      :severity-counts="severityCounts"
      :total-findings="reportData?.total_findings || 0"
      :active-category="activeCategory"
      @select-category="cat => activeCategory = cat"
    />

    <!-- Filter & Search Bar -->
    <DiagnosticsAuditFilterBar
      v-model:active-category="activeCategory"
      v-model:active-severity="activeSeverity"
      v-model:active-status="activeStatus"
      v-model:search-query="searchQuery"
      :total-filtered="filteredFindings.length"
      :status-counts="statusCounts"
      :severity-counts="severityCounts"
      :category-counts="categoryCounts"
      :group-similar="groupSimilar"
      :mergeable-groups="mergeableGroupCount"
      @toggle-group="toggleGroupSimilar"
    />

    <!-- Loading State -->
    <div v-if="loading" class="py-16 text-center space-y-3">
      <UIcon name="i-lucide-loader-2" class="w-8 h-8 text-primary-500 animate-spin mx-auto" />
      <p class="text-sm font-medium text-neutral-600 dark:text-neutral-400">
        Memuat data laporan audit diagnostik...
      </p>
    </div>

    <!-- Findings List -->
    <div v-else-if="filteredFindings.length > 0" class="space-y-3">
      <p v-if="groupSimilar" class="text-xs text-neutral-500">
        {{ filteredFindings.length }} temuan disatukan menjadi {{ findingGroups.length }} temuan utama
      </p>

      <template v-if="groupSimilar">
        <template v-for="group in findingGroups" :key="group.key">
          <DiagnosticsFindingGroupCard
            v-if="group.findings.length > 1"
            :group="group"
            :resolving-key="resolvingFindingKey"
            :resolved-keys="resolvedFindingKeys"
            :finding-key="getFindingKey"
            @resolve-group="handleResolveGroup"
            @resolve="handleResolveFinding"
            @re-resolve="handleResolveFinding"
            @view-resolution="handleViewResolution"
            @ignore="handleIgnoreFinding"
            @delete="handleDeleteFinding"
            @feedback="handleFindingFeedback"
          />
          <DiagnosticsFindingCard
            v-else-if="group.findings[0]"
            :finding="group.findings[0]"
            :resolving="resolvingFindingKey === getFindingKey(group.findings[0])"
            :has-created-todo="resolvedFindingKeys.has(getFindingKey(group.findings[0]))"
            @resolve="handleResolveFinding"
            @re-resolve="handleResolveFinding"
            @view-resolution="handleViewResolution"
            @ignore="handleIgnoreFinding"
            @delete="handleDeleteFinding"
            @feedback="handleFindingFeedback"
          />
        </template>
      </template>
      <template v-else>
        <DiagnosticsFindingCard
          v-for="(finding, idx) in filteredFindings"
          :key="getFindingKey(finding) || String(idx)"
          :finding="finding"
          :resolving="resolvingFindingKey === getFindingKey(finding)"
          :has-created-todo="resolvedFindingKeys.has(getFindingKey(finding))"
          @resolve="handleResolveFinding"
          @re-resolve="handleResolveFinding"
          @view-resolution="handleViewResolution"
          @ignore="handleIgnoreFinding"
          @delete="handleDeleteFinding"
          @feedback="handleFindingFeedback"
        />
      </template>
    </div>

    <!-- Empty State -->
    <div
      v-else
      class="p-12 text-center rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] space-y-3"
    >
      <div class="w-12 h-12 rounded-full bg-sky-50 dark:bg-sky-950/40 text-sky-600 dark:text-sky-400 flex items-center justify-center mx-auto">
        <UIcon name="i-lucide-shield-check" class="w-6 h-6" />
      </div>
      <h3 class="text-base font-bold text-neutral-900 dark:text-white">
        Tidak Ada Temuan Masalah pada Kategori Ini
      </h3>
      <p class="text-xs text-neutral-500 max-w-md mx-auto">
        Kode pada kategori yang dipilih memenuhi standar arsitektur dan bebas dari anti-pattern yang terdeteksi.
      </p>
    </div>

    <!-- Vonis Pembelajaran Audit -->
    <DiagnosticsFindingFeedbackModal
      v-model:open="feedbackModalOpen"
      :project-id="activeProjectId || 0"
      :finding="feedbackFinding"
      :initial-verdict="feedbackVerdict"
      :auto-verify="feedbackAutoVerify"
      @saved="(p) => applyFeedbackResult(p.findings, p.score)"
    />

    <!-- Resolution Modal (Single Item View) -->
    <DiagnosticsResolutionModal
      v-model:open="resolutionModalOpen"
      :resolution="activeResolution"
      :loading="modalLoading"
    />

    <!-- Batch Resolution Modal (Progressive FE Batch Execution) -->
    <DiagnosticsBatchResolveModal
      v-model:open="batchModalOpen"
      v-model:auto-sweep="autoSweep"
      :project-id="activeProjectId || 0"
      :findings="batchFindings"
      :groups="batchGroups"
      :sweep-busy="auditSweepJob.busy.value"
      :sweep-label="auditSweepJob.label.value"
      @finding-resolved="handleBatchItemResolved"
      @view-resolution="handleViewResolution"
      @completed="handleBatchDone"
      @sweep="handleRunSweep"
    />

    <!-- Audit Inspection Criteria & Smells Drawer (Sidebar Kanan) -->
    <DiagnosticsAuditCriteriaDrawer
      v-model:open="criteriaDrawerOpen"
    />

    <!-- Route Simulation & Execution Time Drawer -->
    <DiagnosticsRouteSimulationDrawer
      v-model:open="simulationDrawerOpen"
      :simulations="latestDeepCheck?.route_simulations || []"
      :findings="latestDeepCheck?.findings || []"
      :last-check-at="latestDeepCheck?.last_deep_check_at || activeProject?.last_deep_check_at"
      :score="latestDeepCheck?.score ?? activeProject?.last_deep_check_score"
      @resolve-all="handleResolveAll"
    />
  </div>
</template>
