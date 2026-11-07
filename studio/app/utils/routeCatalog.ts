// Pembantu murni katalog rute (parsing target request & warna method).
export const parseTargetRequests = (targetReqs?: string): string[] => {
  if (!targetReqs) return []
  try {
    const parsed = JSON.parse(targetReqs)
    if (Array.isArray(parsed)) return parsed
  } catch {
    return targetReqs.split(/[,\n]/).map(s => s.trim()).filter(Boolean)
  }
  return []
}

export const getRequestMethodColor = (req: string): { bg: string, text: string, border: string } => {
  const m = req.trim().split(' ')[0]?.toUpperCase() || 'GET'
  if (m === 'GET') {
    return {
      bg: 'bg-sky-50 dark:bg-sky-950/60',
      text: 'text-sky-700 dark:text-sky-300',
      border: 'border-sky-200 dark:border-sky-800'
    }
  }
  if (m === 'POST') {
    return {
      bg: 'bg-blue-50 dark:bg-blue-950/60',
      text: 'text-blue-700 dark:text-blue-300',
      border: 'border-blue-200 dark:border-blue-800'
    }
  }
  if (m === 'PUT' || m === 'PATCH') {
    return {
      bg: 'bg-amber-50 dark:bg-amber-950/60',
      text: 'text-amber-700 dark:text-amber-300',
      border: 'border-amber-200 dark:border-amber-800'
    }
  }
  if (m === 'DELETE') {
    return {
      bg: 'bg-rose-50 dark:bg-rose-950/60',
      text: 'text-rose-700 dark:text-rose-300',
      border: 'border-rose-200 dark:border-rose-800'
    }
  }
  return {
    bg: 'bg-neutral-100 dark:bg-neutral-800',
    text: 'text-neutral-700 dark:text-neutral-300',
    border: 'border-neutral-200 dark:border-neutral-700'
  }
}
