import type { Todo } from '~/types'
import { impactPerEffort } from '../composables/useTodoImpact'
import { parseTodoTags } from './todoTags'
import {
  isExecutiveOrigin,
  isAuditOrigin,
  isPmOrigin,
  isAiSuggestionOrigin,
  isAgentOrigin
} from './todoCardMeta'

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
export const priorityWeights: Record<string, number> = {
  critical: 4,
  high: 3,
  medium: 2,
  low: 1
}

export const statusWeights: Record<string, number> = {
  in_progress: 1,
  pending: 2,
  suggested: 3,
  completed: 4,
  rejected: 5
}

export const matchesLevelFilter = (t: Todo, level: string): boolean => {
  if (!level || level === 'all') return true
  return t.level === level
}

export const matchesCategoryFilter = (t: Todo, category: string): boolean => {
  if (!category || category === 'all') return true
  const itemCat = t.category || 'general'
  return itemCat === category
}

export const matchesPriorityFilter = (t: Todo, priority: string): boolean => {
  if (!priority || priority === 'all') return true
  return t.priority === priority
}

export const matchesTagFilter = (t: Todo, tag: string): boolean => {
  if (!tag || tag === 'all') return true
  const tags = parseTodoTags(t.tags)
  if (tag === 'untagged') {
    return tags.length === 0
  }
  return tags.includes(tag)
}

export const isAgentCreator = (t: Todo): boolean => {
  if (isExecutiveOrigin(t) || isAuditOrigin(t) || isPmOrigin(t) || isAiSuggestionOrigin(t)) {
    return false
  }
  return isAgentOrigin(t)
}

export const isHumanOnlyCreator = (t: Todo): boolean => {
  const src = (t.source || '').toLowerCase()
  const cType = (t.creator_type || '').toLowerCase()
  const isHuman = cType === 'human' || src === 'manual'
  if (!isHuman) return false
  return !isExecutiveOrigin(t) && !isAuditOrigin(t) && !isPmOrigin(t) && !isAiSuggestionOrigin(t) && !isAgentOrigin(t)
}

export const creatorMatchers: Record<string, (t: Todo) => boolean> = {
  executive: isExecutiveOrigin,
  audit: isAuditOrigin,
  pm: isPmOrigin,
  ai_suggestion: isAiSuggestionOrigin,
  agent: isAgentCreator,
  human: isHumanOnlyCreator,
  ai: (t: Todo) => Boolean(t.is_ai_suggestion || t.creator_type === 'ai_agent')
}

export const creatorPredicates = creatorMatchers

export const matchesCreatorFilter = (t: Todo, creator: string): boolean => {
  if (!creator || creator === 'all') return true
  const predicate = creatorMatchers[creator]
  return predicate ? predicate(t) : true
}

export const matchesEmployeeFilter = (t: Todo, employee: string): boolean => {
  if (!employee || employee === 'all') return true
  const target = employee.toLowerCase()
  if (employee === 'unassigned') {
    return !t.employee_name && !t.employee_id
  }
  const nameMatch = t.employee_name ? t.employee_name.toLowerCase().includes(target) : false
  const idMatch = t.employee_id ? String(t.employee_id) === target : false
  return nameMatch || idMatch
}

export const matchesStatusFilter = (t: Todo, status: string): boolean => {
  if (!status || status === 'all') return true
  if (status === 'uncompleted') {
    return t.status !== 'completed' && t.status !== 'rejected'
  }
  return t.status === status
}

export const matchesSearchFilter = (t: Todo, search: string): boolean => {
  const q = (search || '').trim().toLowerCase()
  if (!q) return true

  const fields = [
    t.title,
    t.description,
    t.guideline_ref,
    t.category,
    t.target_apps
  ]
  return fields.some(field => Boolean(field && field.toLowerCase().includes(q)))
}

export const matchesTodoFilter = (t: Todo, f: TodoFilterState): boolean => {
  const predicates: Array<(t: Todo) => boolean> = [
    t => matchesLevelFilter(t, f.level),
    t => matchesCategoryFilter(t, f.category),
    t => matchesTagFilter(t, f.tag),
    t => matchesCreatorFilter(t, f.creator),
    t => matchesStatusFilter(t, f.status),
    t => matchesSearchFilter(t, f.search)
  ]
  if (f.employee) {
    predicates.push(t => matchesEmployeeFilter(t, f.employee!))
  }
  return predicates.every(p => p(t))
}

export const comparePriorityDesc = (a: Todo, b: Todo): number => {
  const wA = priorityWeights[a.priority] || 2
  const wB = priorityWeights[b.priority] || 2
  if (wB !== wA) return wB - wA
  return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
}

export const comparePriorityAsc = (a: Todo, b: Todo): number => {
  const wA = priorityWeights[a.priority] || 2
  const wB = priorityWeights[b.priority] || 2
  if (wA !== wB) return wA - wB
  return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
}

export const compareImpactDesc = (a: Todo, b: Todo): number => {
  return (b.impact_score || 0) - (a.impact_score || 0)
}

export const compareValueDesc = (a: Todo, b: Todo): number => {
  return impactPerEffort(b) - impactPerEffort(a)
}

export const compareCreatedDesc = (a: Todo, b: Todo): number => {
  return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
}

export const compareCreatedAsc = (a: Todo, b: Todo): number => {
  return new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
}

export const compareTitleAsc = (a: Todo, b: Todo): number => {
  return (a.title || '').localeCompare(b.title || '')
}

export const compareStatus = (a: Todo, b: Todo): number => {
  const sA = statusWeights[a.status] || 9
  const sB = statusWeights[b.status] || 9
  return sA - sB
}

export type TodoComparator = (a: Todo, b: Todo) => number

export const orderComparators: Record<string, TodoComparator> = {
  priority_desc: comparePriorityDesc,
  priority_asc: comparePriorityAsc,
  impact_desc: compareImpactDesc,
  value_desc: compareValueDesc,
  created_desc: compareCreatedDesc,
  created_asc: compareCreatedAsc,
  title_asc: compareTitleAsc,
  status: compareStatus
}

export const todoComparators = orderComparators

export const getTodoComparator = (sortBy?: string, sortOrder?: string): TodoComparator => {
  if (!sortBy) return () => 0
  if (orderComparators[sortBy]) {
    return orderComparators[sortBy]
  }
  const composite = sortOrder ? `${sortBy}_${sortOrder}` : sortBy
  if (orderComparators[composite]) {
    return orderComparators[composite]
  }
  return () => 0
}

// Saring lalu urutkan todo sesuai filter halaman Todo.
export const filterAndSortTodos = (todos: Todo[], f: TodoFilterState): Todo[] => {
  const predicates: Array<(t: Todo) => boolean> = [
    t => matchesLevelFilter(t, f.level),
    t => matchesCategoryFilter(t, f.category),
    t => matchesTagFilter(t, f.tag),
    t => matchesCreatorFilter(t, f.creator),
    t => matchesStatusFilter(t, f.status),
    t => matchesSearchFilter(t, f.search)
  ]
  if (f.employee) {
    predicates.push(t => matchesEmployeeFilter(t, f.employee!))
  }
  const comparator = getTodoComparator(f.orderBy)
  return [...todos]
    .filter(t => predicates.every(p => p(t)))
    .sort(comparator)
}
