import type { RoleBadgeColor } from '../types'

export type MethodBadgeColor = 'success' | 'info' | 'warning' | 'error' | 'neutral' | 'primary'

export interface RoleLabelOptions {
  /** Label untuk role `frontend_mobile` (default 'Mobile'). */
  mobile?: string
  /** Label untuk role `other` (default 'Lainnya'). */
  other?: string
  /** Nilai bila role kosong/tak dikenal selain `other`; default mengembalikan role apa adanya. */
  fallback?: string
}

// Pemformat & pemeta tampilan murni yang dipakai banyak komponen (satu sumber kebenaran).
export const useFormatters = () => {
  const formatDateTime = (dateStr?: string | null): string => {
    if (!dateStr) return '-'
    try {
      return new Date(dateStr).toLocaleString('id-ID', {
        day: 'numeric',
        month: 'short',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      })
    } catch {
      return dateStr
    }
  }

  const formatBytes = (bytes?: number | null): string => {
    if (!bytes) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
  }

  const getSeverityBadgeColor = (sev: string) => {
    switch (sev?.toUpperCase()) {
      case 'CRITICAL':
        return 'error'
      case 'WARNING':
        return 'warning'
      case 'INFO':
        return 'info'
      default:
        return 'neutral'
    }
  }

  const getCategoryIcon = (cat: string) => {
    switch (cat?.toLowerCase()) {
      case 'query':
      case 'database':
        return 'i-lucide-database'
      case 'router':
      case 'api':
        return 'i-lucide-route'
      case 'code_smell':
        return 'i-lucide-flame'
      case 'security':
        return 'i-lucide-shield-alert'
      case 'port':
        return 'i-lucide-network'
      case 'runtime':
        return 'i-lucide-play-circle'
      case 'frontend':
        return 'i-lucide-layout-template'
      case 'mobile':
        return 'i-lucide-smartphone'
      case 'deployment':
      case 'config':
        return 'i-lucide-hard-drive-upload'
      default:
        return 'i-lucide-alert-circle'
    }
  }

  const getRoleBadgeColor = (role?: string): RoleBadgeColor => {
    switch (role) {
      case 'backend': return 'info'
      case 'frontend_admin': return 'secondary'
      case 'frontend_client': return 'success'
      case 'frontend_mobile': return 'warning'
      default: return 'neutral'
    }
  }

  // Label ringkas role komponen. Teks tiap layar berbeda tipis, jadi diparameterkan.
  const getRoleLabel = (role?: string, opts: RoleLabelOptions = {}): string => {
    switch (role) {
      case 'backend': return 'BE'
      case 'frontend_admin': return 'FE Admin'
      case 'frontend_client': return 'FE Web'
      case 'frontend_mobile': return opts.mobile ?? 'Mobile'
      case 'docs': return 'Docs'
      case 'cli': return 'CLI'
      case 'worker': return 'Worker'
      default:
        if (role === 'other') return opts.other ?? 'Lainnya'
        return (opts.fallback !== undefined ? (role || opts.fallback) : role) as string
    }
  }

  const getMethodBadgeColor = (method: string): MethodBadgeColor => {
    const m = (method || '').toUpperCase()
    if (m === 'GET') return 'success'
    if (m === 'POST') return 'info'
    if (m === 'PUT' || m === 'PATCH') return 'warning'
    if (m === 'DELETE') return 'error'
    if (m.includes('PAGE') || m.includes('VIEW') || m.includes('SCREEN')) return 'primary'
    return 'neutral'
  }

  // Kelas Tailwind lencana metode HTTP; warna POST berbeda antar layar ('blue' atau 'sky').
  const getMethodColorClass = (method: string, postTone: 'blue' | 'sky' = 'blue'): string => {
    const m = method.toUpperCase()
    if (m.startsWith('GET')) return 'bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-400 border-emerald-200 dark:border-emerald-800'
    if (m.startsWith('POST')) {
      return postTone === 'sky'
        ? 'bg-sky-50 dark:bg-sky-950/60 text-sky-700 dark:text-sky-400 border-sky-200 dark:border-sky-800'
        : 'bg-blue-50 dark:bg-blue-950/60 text-blue-700 dark:text-blue-400 border-blue-200 dark:border-blue-800'
    }
    if (m.startsWith('PUT') || m.startsWith('PATCH')) return 'bg-amber-50 dark:bg-amber-950/60 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800'
    if (m.startsWith('DELETE')) return 'bg-rose-50 dark:bg-rose-950/60 text-rose-700 dark:text-rose-400 border-rose-200 dark:border-rose-800'
    return 'bg-purple-50 dark:bg-purple-950/60 text-purple-700 dark:text-purple-400 border-purple-200 dark:border-purple-800'
  }

  return {
    formatDateTime,
    formatBytes,
    getSeverityBadgeColor,
    getCategoryIcon,
    getRoleBadgeColor,
    getRoleLabel,
    getMethodBadgeColor,
    getMethodColorClass
  }
}
