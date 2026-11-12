// Label, warna, dan penyaringan hasil uji keamanan proyek (logika murni, diuji di tests/securityTest.test.ts).
import type { SecurityCheck, SecurityCheckCategory, SecurityCheckStatus } from '~/types/security_test'

export const SEC_STATUS_META: Record<SecurityCheckStatus, { label: string, hint: string, icon: string, text: string, border: string, badge: 'success' | 'info' | 'warning' | 'error' | 'neutral' }> = {
  fail: { label: 'Gagal', hint: 'Kontrol tidak ada / terbukti tidak jalan', icon: 'i-lucide-shield-x', text: 'text-rose-600 dark:text-rose-400', border: 'border-rose-200 dark:border-rose-500/30', badge: 'error' },
  warn: { label: 'Lemah', hint: 'Ada, tetapi sebagian', icon: 'i-lucide-shield-alert', text: 'text-amber-600 dark:text-amber-400', border: 'border-amber-200 dark:border-amber-500/30', badge: 'warning' },
  declared: { label: 'Terlihat di kode', hint: 'Belum dibuktikan di server berjalan', icon: 'i-lucide-file-code-2', text: 'text-sky-600 dark:text-sky-400', border: 'border-neutral-200 dark:border-white/[0.08]', badge: 'info' },
  pass: { label: 'Lulus', hint: 'Terbukti aman', icon: 'i-lucide-shield-check', text: 'text-emerald-600 dark:text-emerald-400', border: 'border-neutral-200 dark:border-white/[0.08]', badge: 'success' },
  skipped: { label: 'Dilewati', hint: 'Tidak bisa diuji', icon: 'i-lucide-circle-slash', text: 'text-neutral-500 dark:text-slate-400', border: 'border-neutral-200 dark:border-white/[0.08]', badge: 'neutral' }
}

export const SEC_STATUS_ORDER: SecurityCheckStatus[] = ['fail', 'warn', 'declared', 'pass', 'skipped']

// Tab kategori: 5 utama + "Lainnya" (validasi input, HTTPS, bocoran galat).
export type SecurityTab = 'all' | 'rate_limit' | 'injection' | 'headers' | 'cors' | 'auth' | 'other'

export const SEC_TAB_META: { id: SecurityTab, label: string, icon: string }[] = [
  { id: 'all', label: 'Semua', icon: 'i-lucide-list' },
  { id: 'rate_limit', label: 'Rate limit & DDoS', icon: 'i-lucide-gauge' },
  { id: 'injection', label: 'Injection', icon: 'i-lucide-syringe' },
  { id: 'headers', label: 'Header', icon: 'i-lucide-heading' },
  { id: 'cors', label: 'CORS', icon: 'i-lucide-globe-lock' },
  { id: 'auth', label: 'Autentikasi', icon: 'i-lucide-key-round' },
  { id: 'other', label: 'Lainnya', icon: 'i-lucide-ellipsis' }
]

const OTHER: SecurityCheckCategory[] = ['input', 'transport', 'error_leaks']

export const SEC_CATEGORY_LABEL: Record<SecurityCheckCategory, string> = {
  rate_limit: 'Rate limit',
  headers: 'Header',
  cors: 'CORS',
  injection: 'Injection',
  input: 'Validasi input',
  auth: 'Autentikasi',
  transport: 'HTTPS',
  error_leaks: 'Bocoran galat'
}

export const securityTabOf = (c: SecurityCheck): SecurityTab => (OTHER.includes(c.category) ? 'other' : c.category as SecurityTab)

export interface SecurityFilter {
  tab: SecurityTab
  status: SecurityCheckStatus | 'all'
  source: 'all' | 'static' | 'live'
  q: string
}

export const filterSecurityChecks = (checks: SecurityCheck[], f: SecurityFilter): SecurityCheck[] => {
  const q = f.q.trim().toLowerCase()
  return checks.filter((c) => {
    if (f.tab !== 'all' && securityTabOf(c) !== f.tab) return false
    if (f.status !== 'all' && c.status !== f.status) return false
    if (f.source !== 'all' && c.source !== f.source) return false
    if (!q) return true
    const hay = [c.id, c.title, ...(c.evidence || []), ...(c.locations || []).map(l => l.file)].join(' ').toLowerCase()
    return hay.includes(q)
  })
}

export const countSecurityByTab = (checks: SecurityCheck[]): Record<SecurityTab, number> => {
  const out = { all: checks.length, rate_limit: 0, injection: 0, headers: 0, cors: 0, auth: 0, other: 0 } as Record<SecurityTab, number>
  for (const c of checks) out[securityTabOf(c)]++
  return out
}

export const securityGradeColor = (score: number): string => {
  if (score >= 90) return 'text-emerald-600 dark:text-emerald-400'
  if (score >= 75) return 'text-sky-600 dark:text-sky-400'
  if (score >= 60) return 'text-amber-600 dark:text-amber-400'
  return 'text-rose-600 dark:text-rose-400'
}

export const securityGradeLabel = (grade: string): string => ({
  A: 'Kontrol standar terpasang dan terbukti',
  B: 'Sebagian besar aman, ada yang perlu dibuktikan',
  C: 'Ada celah yang perlu ditutup',
  D: 'Banyak kontrol standar belum ada',
  E: 'Rawan: kontrol dasar belum terpasang'
} as Record<string, string>)[grade] || ''

// Validasi URL server sebelum dikirim ke engine (engine memvalidasi ulang).
export const securityBaseUrlProblem = (raw: string): string => {
  const v = raw.trim()
  if (!v) return ''
  const withScheme = v.includes('://') ? v : `http://${v}`
  try {
    const u = new URL(withScheme)
    if (u.protocol !== 'http:' && u.protocol !== 'https:') return 'URL harus http atau https'
    if (!u.hostname) return 'URL tanpa host'
    return ''
  } catch {
    return 'URL tidak valid'
  }
}
