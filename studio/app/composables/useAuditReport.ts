import type { DiagnosticReportResponse, DeepCheckLatestResponse } from '~/types'

// Memuat laporan audit proyek aktif (digabung dengan Deep Check) + menjalankan audit/deep check/unduhan.
export function useAuditReport() {
  const toast = useToast()
  const { activeProjectId, activeProject, fetchProjects } = useActiveProject()
  const {
    getLatestProjectDiagnostic,
    runDiagnostics,
    runDeepCheck,
    getLatestDeepCheck,
    getDeepCheckDownloadUrl
  } = useApi()

  const reportData = ref<DiagnosticReportResponse | null>(null)
  const loading = ref(true)
  const diagnosing = ref(false)
  const deepChecking = ref(false)
  const latestDeepCheck = ref<DeepCheckLatestResponse | null>(null)
  const resolvedFindingKeys = ref<Set<string>>(new Set())

  // null bila proyek belum punya laporan audit (bukan 100 — skor sempurna yang menyesatkan).
  const currentScore = computed<number | null>(() => reportData.value ? reportData.value.score : null)

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

  // Audit & Selesaikan Semua (antrean engine): audit standar + Deep Check lalu temuan terbuka langsung
  // diselesaikan via LLM, diulang sampai audit ulang tidak memunculkan temuan baru (maks 3 putaran).
  const auditSweepJob = useJobTarget('audit_sweep', () => (activeProjectId.value ? { projectId: activeProjectId.value } : null), async (job) => {
    if (activeProjectId.value) {
      await loadAuditReport(activeProjectId.value, true)
      await fetchProjects(true)
    }
    toast.add({
      title: 'Audit & Penyelesaian Selesai',
      description: (job.message || '').replace(/^Selesai:\s*/, '') || 'Temuan baru dari audit ulang sudah dimasukkan ke daftar TODO.',
      icon: 'i-lucide-repeat',
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

  return {
    reportData, loading, diagnosing, deepChecking, latestDeepCheck, resolvedFindingKeys, currentScore,
    llmReviewJob, auditSweepJob, loadAuditReport, handleRunAudit, handleRunDeepCheck, handleDownloadDeepCheck
  }
}
