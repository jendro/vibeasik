import type { Finding } from '~/types'

// Kunci stabil, pencocokan & penggabungan temuan audit (pure).
export const getFindingKey = (f: Finding): string => {
  if (f.id) return f.id
  const rule = f.rule_id || f.title || 'issue'
  const file = f.target_file || 'global'
  const line = f.line || 0
  return `${rule}_${file}_${line}`
}

export const areFindingsMatching = (a: Finding, b: Finding): boolean => {
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

export const deduplicateFindings = (list: Finding[]): Finding[] => {
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

export const mergeFindingsLists = (existing: Finding[], incoming: Finding[]): Finding[] => {
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
