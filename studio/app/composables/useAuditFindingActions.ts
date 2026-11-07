import type { Ref, ComputedRef } from 'vue'
import type { AuditVerdict, DiagnosticReportResponse, Finding, ResolutionResponse } from '~/types'
import type { FindingGroup } from '~/utils/findingGroups'

// Aksi pada temuan audit: solusi LLM, abaikan, vonis, hapus, selesaikan semua/grup, batch bertahap.
export function useAuditFindingActions(ctx: {
  reportData: Ref<DiagnosticReportResponse | null>
  resolvedFindingKeys: Ref<Set<string>>
  pendingActionableFindings: ComputedRef<Finding[]>
  pendingActionableCount: ComputedRef<number>
  isFindingResolved: (f: Finding) => boolean
  isFindingIgnored: (f: Finding) => boolean
  groupSimilar: Ref<boolean>
  loadAuditReport: (projectId: number, preserveExisting?: boolean) => Promise<void>
}) {
  const { reportData, resolvedFindingKeys, pendingActionableFindings, pendingActionableCount, isFindingResolved, isFindingIgnored, groupSimilar, loadAuditReport } = ctx
  const toast = useToast()
  const { activeProjectId, fetchProjects } = useActiveProject()
  const {
    resolveDiagnosticIssue,
    resolveDiagnosticGroup,
    deleteDiagnosticFinding,
    ignoreDiagnosticFinding,
    recordFindingFeedback,
    groupTodos
  } = useApi()

  // Resolution Modal State
  const resolutionModalOpen = ref(false)
  const batchModalOpen = ref(false)
  const activeResolution = ref<ResolutionResponse | null>(null)
  const modalLoading = ref(false)
  const resolvingFindingKey = ref<string | null>(null)

  const handleResolveFinding = async (finding: Finding) => {
    if (!activeProjectId.value) return
    const key = getFindingKey(finding)
    resolvingFindingKey.value = key
    modalLoading.value = true
    resolutionModalOpen.value = true
    activeResolution.value = null

    try {
      const res = await resolveDiagnosticIssue(activeProjectId.value, finding, true)
      activeResolution.value = res
      resolvedFindingKeys.value.add(key)
      applyResolutionToFinding(finding, res)
      if (res.score !== undefined && reportData.value) {
        reportData.value.score = res.score
      }
      await fetchProjects(true)
      const merged = res.todo_outcome === 'merged' || res.todo_outcome === 'represented'
      if (res.todo_outcome === 'rejected') {
        toast.add({
          title: `Pernah Ditolak di TODO #${res.created_todo?.id}`,
          description: res.explanation || `Temuan '${finding.title}' sejenis dengan todo yang pernah ditolak, jadi tidak dibuat todo baru.`,
          icon: 'i-lucide-ban',
          color: 'neutral'
        })
        return
      }
      toast.add({
        title: merged
          ? (res.todo_outcome === 'merged' ? `Digabung ke TODO #${res.created_todo?.id}` : `Sudah Terwakili TODO #${res.created_todo?.id}`)
          : 'Solusi AI Dihasilkan & Tersimpan di TODO',
        description: merged
          ? `Temuan '${finding.title}' memiliki rule & berkas yang sama dengan todo audit yang masih terbuka, jadi tidak dibuat todo baru.`
          : `Rencana aksi & langkah perbaikan '${finding.title}' tersimpan lengkap di daftar TODO untuk dieksekusi agent.`,
        icon: merged ? 'i-lucide-merge' : 'i-lucide-sparkles',
        color: 'success'
      })
    } catch (err: any) {
      toast.add({
        title: 'Gagal Menghasilkan Solusi',
        description: err.message || 'Gagal menghubungi external LLM.',
        icon: 'i-lucide-alert-circle',
        color: 'error'
      })
      resolutionModalOpen.value = false
    } finally {
      resolvingFindingKey.value = null
      modalLoading.value = false
    }
  }

  const handleViewResolution = (finding: Finding) => {
    activeResolution.value = {
      success: true,
      rule_id: finding.rule_id || '',
      title: finding.resolution_title || finding.title,
      theory_basis: finding.theory_basis || 'Standar Rekayasa Perangkat Lunak',
      target_file: finding.target_file || '',
      target_lines: finding.target_lines || (finding.line ? String(finding.line) : '1'),
      code_before: finding.code_before || finding.snippet || '',
      code_after: finding.code_after || finding.suggested_fix || finding.suggestion,
      explanation: finding.explanation || finding.description,
      action_steps: finding.action_steps || [],
      placement_plan: finding.placement_plan,
      created_todo: finding.todo_id ? { id: finding.todo_id, title: finding.resolution_title || finding.title } : undefined,
      model_used: finding.model_used || 'vibeasik-engine',
      generated_via: 'external_llm'
    }
    resolutionModalOpen.value = true
  }

  const handleIgnoreFinding = async (finding: Finding) => {
    if (!activeProjectId.value) return
    const key = getFindingKey(finding)
    try {
      const res = await ignoreDiagnosticFinding(activeProjectId.value, finding)
      const newStatus = res.status || (finding.status === 'ignored' ? 'open' : 'ignored')
      finding.status = newStatus

      if (newStatus === 'ignored') {
        resolvedFindingKeys.value.delete(key)
      }

      if (res.findings && Array.isArray(res.findings)) {
        if (reportData.value) {
          reportData.value.findings = deduplicateFindings(res.findings)
          reportData.value.total_findings = reportData.value.findings.length
        }
      }

      if (res.score !== undefined && reportData.value) {
        reportData.value.score = res.score
        await fetchProjects(true)
      }
      const isNowIgnored = newStatus === 'ignored'
      toast.add({
        title: isNowIgnored ? 'Temuan Diabaikan' : 'Pengabaian Dibatalkan',
        description: isNowIgnored
          ? `'${finding.title}' telah ditandai sebagai diabaikan dan dilewati dari proses otomatis.`
          : `'${finding.title}' kini aktif kembali dalam daftar audit.`,
        icon: isNowIgnored ? 'i-lucide-eye-off' : 'i-lucide-eye',
        color: isNowIgnored ? 'neutral' : 'info'
      })
    } catch (err: any) {
      toast.add({
        title: 'Gagal Mengubah Status',
        description: err.message || 'Gagal mengubah status pengabaian temuan.',
        icon: 'i-lucide-alert-circle',
        color: 'error'
      })
    }
  }

  // Vonis pembelajaran audit (audit keliru / temuan benar)
  const feedbackModalOpen = ref(false)
  const feedbackFinding = ref<Finding | null>(null)
  const feedbackVerdict = ref<AuditVerdict | undefined>(undefined)
  const feedbackAutoVerify = ref(false)

  const applyFeedbackResult = async (findings: Finding[], score: number) => {
    if (reportData.value && Array.isArray(findings)) {
      reportData.value.findings = deduplicateFindings(findings)
      reportData.value.total_findings = reportData.value.findings.length
      reportData.value.score = score
    }
    await fetchProjects(true)
  }

  const handleFindingFeedback = async (finding: Finding, mode: 'verify' | 'false_positive' | 'confirmed' | 'clear') => {
    if (!activeProjectId.value) return
    if (mode === 'clear') {
      try {
        const res = await recordFindingFeedback(activeProjectId.value, finding, { verdict: '' })
        await applyFeedbackResult(res.findings, res.score)
        toast.add({ title: 'Vonis dicabut', description: `'${finding.title}' tidak lagi menjadi bahan pembelajaran audit.`, icon: 'i-lucide-undo-2', color: 'neutral' })
      } catch (err: any) {
        toast.add({ title: 'Gagal mencabut vonis', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' })
      }
      return
    }
    feedbackFinding.value = finding
    feedbackVerdict.value = mode === 'verify' ? undefined : mode
    feedbackAutoVerify.value = mode === 'verify'
    feedbackModalOpen.value = true
  }

  const handleDeleteFinding = async (finding: Finding) => {
    if (!activeProjectId.value) return
    const targetKey = getFindingKey(finding)
    try {
      const res = await deleteDiagnosticFinding(activeProjectId.value, finding)
      resolvedFindingKeys.value.delete(targetKey)
      if (res.findings && Array.isArray(res.findings)) {
        if (reportData.value) {
          reportData.value.findings = deduplicateFindings(res.findings)
          reportData.value.total_findings = reportData.value.findings.length
        }
      } else if (reportData.value) {
        reportData.value.findings = reportData.value.findings.filter(f => getFindingKey(f) !== targetKey)
        reportData.value.total_findings = reportData.value.findings.length
      }
      if (res.score !== undefined && reportData.value) {
        reportData.value.score = res.score
        await fetchProjects(true)
      }
      toast.add({
        title: 'Temuan Dihapus',
        description: `'${finding.title}' telah dihapus dari laporan audit diagnostik.`,
        icon: 'i-lucide-trash-2',
        color: 'success'
      })
    } catch (err: any) {
      toast.add({
        title: 'Gagal Menghapus Temuan',
        description: err.message || 'Terjadi kesalahan saat menghapus temuan audit.',
        icon: 'i-lucide-alert-circle',
        color: 'error'
      })
    }
  }

  const handleResolveAll = () => {
    if (!activeProjectId.value || !reportData.value?.findings) return

    if (pendingActionableCount.value === 0) {
      toast.add({
        title: 'Tidak Ada Masalah Tertunda',
        description: 'Semua temuan audit sudah diselesaikan via LLM atau diabaikan.',
        icon: 'i-lucide-check-circle',
        color: 'info'
      })
      return
    }

    // Buka dialog batch resolution progressive dari Frontend
    batchModalOpen.value = true
  }

  const handleBatchItemResolved = ({ finding, resolution }: { finding: Finding, resolution: ResolutionResponse }) => {
    const key = getFindingKey(finding)
    resolvedFindingKeys.value.add(key)
    applyResolutionToFinding(finding, resolution)
    if (resolution.score !== undefined && reportData.value) {
      reportData.value.score = resolution.score
    }
  }

  // Selesaikan Grup: batch hanya sub temuan grup, lalu todo hasilnya dilebur jadi satu todo gabungan (Aturan 43).
  // Satukan Serupa aktif: Selesaikan Bertahap memproses per kelompok (satu panggilan LLM, satu todo per kelompok).
  const batchGroups = computed(() => (groupSimilar.value ? groupSimilarFindings(pendingActionableFindings.value) : []))
  const batchFindings = computed(() => (groupSimilar.value
    ? batchGroups.value.map(g => g.findings[0]).filter((f): f is Finding => !!f)
    : pendingActionableFindings.value))
  // Kunci temuan yang diproses batch ini: kelompok yang tersentuh disatukan dengan todo lamanya setelah batch selesai.
  const batchKeys = ref<Set<string>>(new Set())

  watch(batchModalOpen, (isOpen) => {
    if (isOpen) batchKeys.value = new Set(batchGroups.value.flatMap(g => g.findings.map(getFindingKey)))
  })

  // Selesaikan Grup: semua sub temuan terbuka → SATU panggilan LLM (rangkuman singkat mencakup semua lokasi) → satu todo.
  const handleResolveGroup = async (group: FindingGroup) => {
    if (!activeProjectId.value) return
    const open = group.findings.filter(f => !isFindingResolved(f) && !isFindingIgnored(f))
    if (open.length === 0) return
    const first = open[0]!
    resolvingFindingKey.value = getFindingKey(first)
    modalLoading.value = true
    resolutionModalOpen.value = true
    activeResolution.value = null
    try {
      const res = await resolveDiagnosticGroup(activeProjectId.value, open)
      activeResolution.value = res
      for (const f of open) {
        resolvedFindingKeys.value.add(getFindingKey(f))
        applyResolutionToFinding(f, res)
      }
      if (res.score !== undefined && reportData.value) {
        reportData.value.score = res.score
      }
      // Sub temuan yang dulu sudah punya todo sendiri ikut disatukan ke todo grup ini.
      await groupFindingTodos([group])
      await fetchProjects(true)
      const merged = res.todo_outcome === 'merged' || res.todo_outcome === 'represented'
      toast.add({
        title: merged ? `Digabung ke TODO #${res.created_todo?.id}` : `Grup Diselesaikan → TODO #${res.created_todo?.id}`,
        description: `${open.length} temuan '${group.title}' (${group.fileCount} berkas) dicakup satu todo dari satu panggilan LLM.`,
        icon: 'i-lucide-merge',
        color: 'success'
      })
    } catch (err: any) {
      toast.add({ title: 'Gagal Menyelesaikan Grup', description: err?.data?.error || err.message, icon: 'i-lucide-alert-circle', color: 'error' })
      resolutionModalOpen.value = false
    } finally {
      resolvingFindingKey.value = null
      modalLoading.value = false
    }
  }

  // Todo semua temuan satu kelompok (termasuk yang sudah diselesaikan sebelumnya) → satu todo gabungan.
  const groupFindingTodos = async (groups: FindingGroup[]) => {
    const parents: number[] = []
    let failed = 0
    for (const group of groups) {
      const ids = [...new Set(group.findings.filter(f => f.status !== 'ignored').map(f => f.todo_id).filter((id): id is number => !!id))]
      if (ids.length < 2) continue
      try {
        const res = await groupTodos(ids)
        if (res.parent_id) parents.push(res.parent_id)
      } catch {
        failed++
      }
    }
    if (parents.length > 0) {
      toast.add({
        title: parents.length === 1 ? `Disatukan ke Todo Gabungan #${parents[0]}` : `${parents.length} Todo Gabungan Dibuat`,
        description: `Todo tiap kelompok temuan serupa kini menjadi sub todo dari satu todo gabungan${parents.length > 1 ? ` (#${parents.join(', #')})` : ''}.`,
        icon: 'i-lucide-merge',
        color: 'success'
      })
    }
    if (failed > 0) {
      toast.add({ title: 'Sebagian Todo Gagal Disatukan', description: `${failed} kelompok tetap berupa todo terpisah.`, icon: 'i-lucide-alert-circle', color: 'warning' })
    }
  }

  const handleBatchCompleted = async () => {
    // Kelompok yang punya todo lama (temuan yang diselesaikan sebelumnya) disatukan dengan todo kelompoknya.
    if (groupSimilar.value && reportData.value) {
      const touched = groupSimilarFindings(reportData.value.findings.filter(f => f.status !== 'ignored'))
        .filter(g => g.findings.length > 1 && g.findings.some(f => batchKeys.value.has(getFindingKey(f))))
      await groupFindingTodos(touched)
    }
    if (activeProjectId.value) {
      await loadAuditReport(activeProjectId.value, true)
      await fetchProjects(true)
    }
    toast.add({
      title: 'Eksekusi Bertahap Selesai',
      description: 'Semua batch temuan audit berhasil diproses & disimpan ke daftar TODO.',
      icon: 'i-lucide-check-circle-2',
      color: 'success'
    })
  }

  return {
    resolutionModalOpen, batchModalOpen, activeResolution, modalLoading, resolvingFindingKey,
    feedbackModalOpen, feedbackFinding, feedbackVerdict, feedbackAutoVerify, applyFeedbackResult,
    batchGroups, batchFindings, batchKeys,
    handleResolveFinding, handleViewResolution, handleIgnoreFinding, handleFindingFeedback, handleDeleteFinding,
    handleResolveAll, handleBatchItemResolved, handleResolveGroup, handleBatchCompleted
  }
}
