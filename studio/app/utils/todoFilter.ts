import type { Todo } from '~/types'
import { impactPerEffort } from '~/composables/useTodoImpact'
import { parseTodoTags } from '~/utils/todoTags'

export type TodoOrderBy = 'priority_desc' | 'priority_asc' | 'impact_desc' | 'value_desc' | 'created_desc' | 'created_asc' | 'title_asc' | 'status'

export interface TodoFilterState {
  level: string
  category: string
  creator: string
  tag: string
  status: string
  orderBy: TodoOrderBy
  search: string
}

// Bobot prioritas untuk ordering
const priorityWeights: Record<string, number> = {
  critical: 4,
  high: 3,
  medium: 2,
  low: 1
}

// Saring lalu urutkan todo sesuai filter halaman Todo.
export const filterAndSortTodos = (todos: Todo[], f: TodoFilterState): Todo[] => {
  const filtered = todos.filter((t) => {
    // Level filter (khusus proyek aktif: project bundle atau component)
    if (f.level !== 'all' && t.level !== f.level) {
      return false
    }

    // Category filter
    if (f.category !== 'all') {
      const itemCat = t.category || 'general'
      if (itemCat !== f.category) {
        return false
      }
    }

    // Tag fokus filter
    if (f.tag !== 'all') {
      const tags = parseTodoTags(t.tags)
      if (f.tag === 'untagged' ? tags.length > 0 : !tags.includes(f.tag)) {
        return false
      }
    }

    // Creator / Origin filter
    if (f.creator !== 'all') {
      const src = (t.source || '').toLowerCase()
      const creator = (t.creator_name || '').toLowerCase()
      const cType = (t.creator_type || '').toLowerCase()
      const isExec = src === 'autonomous_executive' || src === 'executive_chat' || creator.includes('eksekutif') || cType === 'executive'
      const isAudit = src === 'diagnostic_audit' || src === 'audit' || Boolean((t as any).audit_key) || Boolean(t.guideline_ref)
      const isPm = src === 'pm_agent' || src === 'planner' || creator.includes('project-manager') || creator.includes('pm')
      const isSugg = t.is_ai_suggestion || creator.includes('advisor') || src === 'ai_template' || src === 'template'
      const isAgent = (cType === 'ai_agent' || creator.includes('claude') || creator.includes('antigravity') || creator.includes('opencode')) && !isExec && !isAudit && !isPm && !isSugg
      const isHuman = cType === 'human' || src === 'manual'

      if (f.creator === 'executive' && !isExec) return false
      if (f.creator === 'audit' && !isAudit) return false
      if (f.creator === 'pm' && !isPm) return false
      if (f.creator === 'ai_suggestion' && !isSugg) return false
      if (f.creator === 'agent' && !isAgent) return false
      if (f.creator === 'human' && (!isHuman || isExec || isAudit || isPm || isSugg || isAgent)) return false

      // Backward compatibility with legacy 'ai'
      if (f.creator === 'ai' && !t.is_ai_suggestion && t.creator_type !== 'ai_agent') return false
    }

    // Status filter:
    // Jika 'uncompleted', hide semua yang statusnya 'completed' dan 'rejected' sehingga terlihat sisa tugas yang belum selesai
    if (f.status === 'uncompleted') {
      if (t.status === 'completed' || t.status === 'rejected') {
        return false
      }
    } else if (f.status !== 'all' && t.status !== f.status) {
      return false
    }

    // Search query
    if (f.search.trim()) {
      const q = f.search.toLowerCase()
      const matchTitle = t.title.toLowerCase().includes(q)
      const matchDesc = (t.description || '').toLowerCase().includes(q)
      const matchGuideline = (t.guideline_ref || '').toLowerCase().includes(q)
      const matchCat = (t.category || '').toLowerCase().includes(q)
      const matchApps = (t.target_apps || '').toLowerCase().includes(q)
      if (!matchTitle && !matchDesc && !matchGuideline && !matchCat && !matchApps) return false
    }

    return true
  })

  // Order By Logic
  return filtered.sort((a, b) => {
    if (f.orderBy === 'priority_desc') {
      const wA = priorityWeights[a.priority] || 2
      const wB = priorityWeights[b.priority] || 2
      if (wB !== wA) return wB - wA
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    }
    if (f.orderBy === 'priority_asc') {
      const wA = priorityWeights[a.priority] || 2
      const wB = priorityWeights[b.priority] || 2
      if (wA !== wB) return wA - wB
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    }
    if (f.orderBy === 'impact_desc') {
      return (b.impact_score || 0) - (a.impact_score || 0)
    }
    if (f.orderBy === 'value_desc') {
      return impactPerEffort(b) - impactPerEffort(a)
    }
    if (f.orderBy === 'created_desc') {
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    }
    if (f.orderBy === 'created_asc') {
      return new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
    }
    if (f.orderBy === 'title_asc') {
      return (a.title || '').localeCompare(b.title || '')
    }
    if (f.orderBy === 'status') {
      const statusWeights: Record<string, number> = {
        in_progress: 1,
        pending: 2,
        suggested: 3,
        completed: 4,
        rejected: 5
      }
      const sA = statusWeights[a.status] || 9
      const sB = statusWeights[b.status] || 9
      return sA - sB
    }
    return 0
  })
}
