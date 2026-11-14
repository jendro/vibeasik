import type { Ref } from 'vue'
import type { DiagnosticReportResponse, Finding } from '~/types'

// Filter, hitungan per status/kategori/keparahan, dan pengelompokan temuan audit.
export function useAuditFindingFilters(reportData: Ref<DiagnosticReportResponse | null>, resolvedFindingKeys: Ref<Set<string>>) {
  // Filters
  const activeCategory = ref('all')
  // Area perbaikan (Finding.area dari engine: speed, data, ui, security, code, ops) — sama dengan ringkasan proyek.
  const activeArea = ref('all')
  const activeSeverity = ref('all')
  const activeStatus = ref('active') // Default: sembunyikan temuan selesai & diabaikan
  const searchQuery = ref('')

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
      runtime: 0,
      risk: 0
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

  const areaCounts = computed(() => {
    const counts: Record<string, number> = {}
    for (const f of reportData.value?.findings || []) {
      const resolved = isFindingResolved(f)
      const ignored = isFindingIgnored(f)
      if (activeStatus.value === 'active' && (resolved || ignored)) continue
      if (activeStatus.value === 'resolved' && !resolved) continue
      if (activeStatus.value === 'ignored' && !ignored) continue
      if (f.area) counts[f.area] = (counts[f.area] || 0) + 1
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

      if (activeArea.value !== 'all' && f.area !== activeArea.value) return false

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

  return {
    activeCategory, activeArea, activeSeverity, activeStatus, searchQuery,
    pendingActionableFindings, pendingActionableCount, isFindingResolved, isFindingIgnored,
    statusCounts, categoryCounts, areaCounts, severityCounts, filteredFindings,
    groupSimilar, findingGroups, mergeableGroupCount, toggleGroupSimilar, GROUP_PREF_KEY
  }
}
