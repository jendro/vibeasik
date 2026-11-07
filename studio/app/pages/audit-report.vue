<script setup lang="ts">
import type { AuditVerdict, Finding, DiagnosticReportResponse, ResolutionResponse, DeepCheckLatestResponse } from '~/types'
import type { FindingGroup } from '~/utils/findingGroups'

definePageMeta({
  fluid: true
})

const route = useRoute()
const toast = useToast()
const {
  projects,
  activeProjectId,
  activeProject,
  fetchProjects,
  selectActiveProject
} = useActiveProject()

const {
  getLatestProjectDiagnostic,
  runDiagnostics,
  runDeepCheck,
  getLatestDeepCheck,
  getDeepCheckDownloadUrl,
  resolveDiagnosticIssue,
  resolveDiagnosticGroup,
  deleteDiagnosticFinding,
  ignoreDiagnosticFinding,
  recordFindingFeedback,
  groupTodos
} = useApi()

const reportData = ref<DiagnosticReportResponse | null>(null)
const loading = ref(true)
const diagnosing = ref(false)
const deepChecking = ref(false)
const latestDeepCheck = ref<DeepCheckLatestResponse | null>(null)
const resolvingAll = ref(false)
const resolvingFindingKey = ref<string | null>(null)

// Filters
const activeCategory = ref('all')
const activeSeverity = ref('all')
const activeStatus = ref('active') // Default: sembunyikan temuan selesai & diabaikan
const searchQuery = ref('')

// Resolution & Criteria Modal State
const criteriaDrawerOpen = ref(false)
const simulationDrawerOpen = ref(false)
const resolutionModalOpen = ref(false)
const batchModalOpen = ref(false)
const activeResolution = ref<ResolutionResponse | null>(null)
const modalLoading = ref(false)
const resolvedFindingKeys = ref<Set<string>>(new Set())

const currentScore = computed(() => reportData.value?.score ?? activeProject.value?.health_score ?? 100)

const getFindingKey = (f: Finding): string => {
  if (f.id) return f.id
  const rule = f.rule_id || f.title || 'issue'
  const file = f.target_file || 'global'
  const line = f.line || 0
  return `${rule}_${file}_${line}`
}

const areFindingsMatching = (a: Finding, b: Finding): boolean => {
  if (a.id && b.id && a.id === b.id) return true
  const keyA = getFindingKey(a)
  const keyB = getFindingKey(b)
  if (keyA === keyB) return true

  // Match by RuleID + TargetFile + Line
  if (a.rule_id && b.rule_id && a.rule_id === b.rule_id
    && (a.target_file || '') === (b.target_file || '')
    && (a.line || 0) === (b.line || 0)) {
    return true
  }

  // Fallback for title matching if rule_id is empty
  if (a.title && b.title && a.title === b.title
    && (a.target_file || '') === (b.target_file || '')
    && (a.line || 0) === (b.line || 0)) {
    return true
  }

  return false
}

const deduplicateFindings = (list: Finding[]): Finding[] => {
  const unique: Finding[] = []
  for (const f of list) {
    const existingIndex = unique.findIndex(u => areFindingsMatching(u, f))
    if (existingIndex >= 0) {
      const match = unique[existingIndex]
      if (match && match.status === 'open' && f.status && f.status !== 'open') {
        match.status = f.status
      }
      continue
    }
    unique.push(f)
  }
  return unique
}

const mergeFindingsLists = (existing: Finding[], incoming: Finding[]): Finding[] => {
  // Start with a clone of existing findings so none of them are deleted!
  const result: Finding[] = [...existing]

  for (const inc of incoming) {
    const existingIdx = result.findIndex(curr => areFindingsMatching(curr, inc))
    if (existingIdx >= 0) {
      // Already present in the table: do NOT add again!
      const current = result[existingIdx]
      if (current) {
        if (current.status === 'open' && inc.status && inc.status !== 'open') {
          current.status = inc.status
        }
        if (!current.theory_basis && inc.theory_basis) {
          current.theory_basis = inc.theory_basis
        }
        if (!current.suggested_fix && inc.suggested_fix) {
          current.suggested_fix = inc.suggested_fix
        }
        if (!current.snippet && inc.snippet) {
          current.snippet = inc.snippet
        }
        if (inc.is_deep) {
          current.is_deep = true
        }
        if (inc.is_llm) {
          current.is_llm = true
        }
      }
    } else {
      // Genuinely new finding: add it to the table!
      result.push(inc)
    }
  }

  return deduplicateFindings(result)
}

const pendingActionableFindings = computed(() => {
  if (!reportData.value?.findings) return []
  return reportData.value.findings.filter(f =>
    f.status !== 'resolved'
    && !resolvedFindingKeys.value.has(getFindingKey(f))
    && f.status !== 'ignored'
  )
})
const pendingActionableCount = computed(() => pendingActionableFindings.value.length)
const isFindingResolved = (f: Finding) => f.status === 'resolved' || (f.status !== 'ignored' && resolvedFindingKeys.value.has(getFindingKey(f)))
const isFindingIgnored = (f: Finding) => f.status === 'ignored'

const statusCounts = computed(() => {
  const list = reportData.value?.findings || []
  let resolved = 0
  let ignored = 0
  let active = 0

  for (const f of list) {
    if (isFindingIgnored(f)) {
      ignored++
    } else if (isFindingResolved(f)) {
      resolved++
    } else {
      active++
    }
  }

  return {
    active,
    resolved,
    ignored,
    all: list.length
  }
})

const categoryCounts = computed(() => {
  const counts: Record<string, number> = {
    deep: 0,
    llm: 0,
    query: 0,
    router: 0,
    code_smell: 0,
    frontend: 0,
    mobile: 0,
    security: 0,
    port: 0,
    architecture: 0,
    runtime: 0
  }
  const list = reportData.value?.findings || []
  for (const f of list) {
    const resolved = isFindingResolved(f)
    const ignored = isFindingIgnored(f)
    if (activeStatus.value === 'active' && (resolved || ignored)) continue
    if (activeStatus.value === 'resolved' && !resolved) continue
    if (activeStatus.value === 'ignored' && !ignored) continue

    if (f.is_deep || f.rule_id?.startsWith('DEEP-') || f.title?.startsWith('[DEEP]')) {
      counts.deep = (counts.deep || 0) + 1
    }
    if (f.is_llm || f.rule_id?.startsWith('LLM-REVIEW-')) {
      counts.llm = (counts.llm || 0) + 1
    }

    const cat = f.category?.toLowerCase() || 'other'
    if (counts[cat] !== undefined) {
      counts[cat] = (counts[cat] || 0) + 1
    }
  }
  return counts
})

const severityCounts = computed(() => {
  const counts: Record<string, number> = {
    CRITICAL: 0,
    WARNING: 0,
    INFO: 0
  }
  const list = reportData.value?.findings || []
  for (const f of list) {
    const resolved = isFindingResolved(f)
    const ignored = isFindingIgnored(f)
    if (activeStatus.value === 'active' && (resolved || ignored)) continue
    if (activeStatus.value === 'resolved' && !resolved) continue
    if (activeStatus.value === 'ignored' && !ignored) continue

    const sev = f.severity?.toUpperCase() || ''
    if (counts[sev] !== undefined) {
      counts[sev]++
    }
  }
  return counts
})

const filteredFindings = computed(() => {
  if (!reportData.value?.findings) return []
  return reportData.value.findings.filter((f) => {
    // Status filter: by default 'active' hides resolved & ignored
    const resolved = isFindingResolved(f)
    const ignored = isFindingIgnored(f)

    if (activeStatus.value === 'active') {
      if (resolved || ignored) return false
    } else if (activeStatus.value === 'resolved') {
      if (!resolved) return false
    } else if (activeStatus.value === 'ignored') {
      if (!ignored) return false
    }

    // Category filter
    if (activeCategory.value !== 'all') {
      if (activeCategory.value === 'deep') {
        const isDeep = Boolean(f.is_deep || f.rule_id?.startsWith('DEEP-') || f.title?.startsWith('[DEEP]'))
        if (!isDeep) return false
      } else if (activeCategory.value === 'llm') {
        if (!f.is_llm && !f.rule_id?.startsWith('LLM-REVIEW-')) return false
      } else {
        const cat = f.category?.toLowerCase()
        if (activeCategory.value === 'code_smell') {
          if (cat !== 'code_smell' && cat !== 'architecture') return false
        } else if (activeCategory.value === 'security') {
          if (cat !== 'security' && cat !== 'port') return false
        } else if (activeCategory.value === 'ui') {
          if (cat !== 'frontend' && cat !== 'mobile') return false
        } else if (activeCategory.value === 'deployment') {
          if (cat !== 'deployment' && cat !== 'config') return false
        } else if (cat !== activeCategory.value) {
          return false
        }
      }
    }

    // Severity filter
    if (activeSeverity.value !== 'all') {
      if (f.severity?.toUpperCase() !== activeSeverity.value.toUpperCase()) return false
    }

    // Search query filter
    if (searchQuery.value.trim() !== '') {
      const q = searchQuery.value.toLowerCase()
      const matchTitle = f.title?.toLowerCase().includes(q)
      const matchFile = f.target_file?.toLowerCase().includes(q)
      const matchRule = f.rule_id?.toLowerCase().includes(q)
      const matchTheory = f.theory_basis?.toLowerCase().includes(q)
      if (!matchTitle && !matchFile && !matchRule && !matchTheory) return false
    }

    return true
  })
})

// Satukan Serupa: temuan rule + ekosistem sama tampil sebagai satu temuan utama dengan sub temuan.
const GROUP_PREF_KEY = 'vibeasik_audit_group_similar'
const groupSimilar = ref(false)
const findingGroups = computed(() => groupSimilarFindings(filteredFindings.value))
const mergeableGroupCount = computed(() => findingGroups.value.filter(g => g.findings.length > 1).length)

const toggleGroupSimilar = () => {
  groupSimilar.value = !groupSimilar.value
  try {
    localStorage.setItem(GROUP_PREF_KEY, groupSimilar.value ? '1' : '0')
  } catch {
    // penyimpanan browser tidak tersedia — preferensi tidak diingat
  }
}

const loadLatestDeepCheck = async (projectId: number) => {
  try {
    const res = await getLatestDeepCheck(projectId)
    latestDeepCheck.value = res
  } catch (err) {
    console.warn('Gagal memuat status latest deep check:', err)
  }
}

// Review audit LLM berjalan di antrean engine (1 slot LLM); halaman tetap bisa dipakai dan
// progres per berkas tampil di bar Antrean. Laporan dimuat ulang saat job selesai.
const llmReviewJob = useJobTarget('llm_audit', () => (activeProjectId.value ? { projectId: activeProjectId.value } : null), async () => {
  if (activeProjectId.value) {
    await loadAuditReport(activeProjectId.value, true)
  }
  toast.add({
    title: 'Review LLM Selesai',
    description: 'Temuan baru ditandai badge LLM. Gunakan filter "Review LLM" untuk melihatnya.',
    icon: 'i-lucide-sparkles',
    color: 'success'
  })
})

const loadAuditReport = async (projectId: number, preserveExisting = false) => {
  if (!preserveExisting) {
    loading.value = true
  }
  try {
    const [data] = await Promise.all([
      getLatestProjectDiagnostic(projectId),
      loadLatestDeepCheck(projectId)
    ])

    const deepFindings = (latestDeepCheck.value?.findings && Array.isArray(latestDeepCheck.value.findings))
      ? latestDeepCheck.value.findings
      : []

    let targetReport = data
    if (!targetReport) {
      targetReport = {
        diagnostic: null,
        findings: [],
        optimizations: [],
        category_counts: {},
        severity_counts: {},
        total_findings: 0,
        score: latestDeepCheck.value?.score ?? 100,
        summary: latestDeepCheck.value?.summary ?? 'Laporan Diagnostik Proyek'
      }
    }

    const currentList = (preserveExisting && reportData.value?.findings) ? reportData.value.findings : []
    let merged = mergeFindingsLists(currentList, targetReport.findings || [])
    if (deepFindings.length > 0) {
      merged = mergeFindingsLists(merged, deepFindings)
    }

    targetReport.findings = deduplicateFindings(merged)
    targetReport.total_findings = targetReport.findings.length

    // Ensure each finding has a guaranteed unique and stable client key/ID
    targetReport.findings.forEach((f) => {
      if (!f.id) {
        f.id = getFindingKey(f)
      }
    })

    resolvedFindingKeys.value.clear()
    for (const f of targetReport.findings) {
      if (f.status === 'resolved') {
        resolvedFindingKeys.value.add(getFindingKey(f))
      }
    }

    reportData.value = targetReport
  } catch (err: any) {
    toast.add({
      title: 'Gagal Memuat Laporan Audit',
      description: err.message || 'Terjadi kesalahan saat memuat audit diagnostik.',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}

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
  selectActiveProject(newId)
  loadAuditReport(newId, false)
  loadRuntimeSimulation(newId)
}

const handleRunAudit = async () => {
  if (!activeProjectId.value) return
  diagnosing.value = true
  try {
    const report = await runDiagnostics(activeProjectId.value)
    if (report?.score !== undefined && reportData.value) {
      reportData.value.score = report.score
    }
    await loadAuditReport(activeProjectId.value, true)
    await fetchProjects(true)
    toast.add({
      title: 'Audit Selesai',
      description: 'Pemeriksaan statis kode, database, dan router berhasil diperbarui.',
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
  } catch (err: any) {
    toast.add({
      title: 'Audit Gagal',
      description: err.message || 'Gagal menjalankan audit diagnostik.',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    diagnosing.value = false
  }
}

const handleRunDeepCheck = async () => {
  if (!activeProjectId.value) return
  deepChecking.value = true
  try {
    const res = await runDeepCheck(activeProjectId.value)
    latestDeepCheck.value = res
    if (res?.score !== undefined && reportData.value) {
      reportData.value.score = res.score
    }
    if (res?.findings && Array.isArray(res.findings)) {
      const currentList = reportData.value?.findings || []
      const merged = mergeFindingsLists(currentList, res.findings)
      if (reportData.value) {
        reportData.value.findings = merged
        reportData.value.total_findings = merged.length
      }
    }
    await loadAuditReport(activeProjectId.value, true)
    await fetchProjects(true)
    toast.add({
      title: 'Deep Check Selesai',
      description: res.summary || `Simulasi real-time rute API dan analisis mendalam berhasil dijalankan. Teridentifikasi ${res.total_findings || 0} temuan.`,
      icon: 'i-lucide-layers',
      color: 'success'
    })
  } catch (err: any) {
    toast.add({
      title: 'Deep Check Gagal',
      description: err.message || 'Gagal menjalankan Deep Check diagnostik.',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    deepChecking.value = false
  }
}

const handleDownloadDeepCheck = (format: 'markdown' | 'json' = 'markdown') => {
  if (!activeProjectId.value) return
  const url = getDeepCheckDownloadUrl(activeProjectId.value, format)
  const link = document.createElement('a')
  link.href = url
  link.setAttribute('download', `deep-check-report-${activeProject.value?.slug || activeProjectId.value}.${format === 'json' ? 'json' : 'md'}`)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  toast.add({
    title: 'Download Dimulai',
    description: `Laporan Deep Check (${format.toUpperCase()}) berhasil diunduh tanpa perlu simulasi ulang.`,
    icon: 'i-lucide-download',
    color: 'success'
  })
}

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

onMounted(async () => {
  try {
    groupSimilar.value = localStorage.getItem(GROUP_PREF_KEY) === '1'
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
})
</script>

<template>
  <div class="space-y-6 w-full px-4 sm:px-6 lg:px-8 pt-6 pb-16">
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
      @switch-project="handleSwitchProject"
      @run-audit="handleRunAudit"
      @run-deep-check="handleRunDeepCheck"
      @download-deep-check="handleDownloadDeepCheck"
      @open-simulations="simulationDrawerOpen = true"
      @resolve-all="handleResolveAll"
      @open-criteria="criteriaDrawerOpen = true"
      @run-llm-review="llmReviewJob.start()"
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
      class="p-12 text-center rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3"
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
      :project-id="activeProjectId || 0"
      :findings="batchFindings"
      :groups="batchGroups"
      @finding-resolved="handleBatchItemResolved"
      @view-resolution="handleViewResolution"
      @completed="handleBatchCompleted"
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
