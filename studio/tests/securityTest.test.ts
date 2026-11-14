import { describe, expect, it } from 'bun:test'
import type { SecurityCheck, SecurityAttack } from '../app/types/security_test'
import { securityBaseUrlProblem, countSecurityByTab, filterAttacks, filterSecurityChecks, securityTabOf } from '../app/utils/securityTest'
import { pageTitleFor } from '../app/utils/sidebarMenuItems'

const mk = (id: string, category: SecurityCheck['category'], status: SecurityCheck['status'], source: SecurityCheck['source'] = 'static'): SecurityCheck =>
  ({ id, category, status, source, title: id, severity: 'INFO', evidence: [`bukti ${id}`], locations: [{ file: `api/${id}.go`, line: 1 }] })

const checks = [
  mk('RATE-LIMIT', 'rate_limit', 'fail'),
  mk('LIVE-RATE-LIMIT', 'rate_limit', 'pass', 'live'),
  mk('SQL-INJECTION', 'injection', 'fail'),
  mk('LIVE-TRANSPORT', 'transport', 'skipped', 'live'),
  mk('ERROR-LEAK', 'error_leaks', 'warn')
]
const all = { tab: 'all', status: 'all', source: 'all', q: '' } as const

describe('uji keamanan: filter & hitungan', () => {
  it('kategori sekunder masuk tab Lainnya', () => {
    expect(securityTabOf(checks[3]!)).toBe('other')
    expect(countSecurityByTab(checks)).toMatchObject({ all: 5, rate_limit: 2, injection: 1, other: 2, cors: 0 })
  })
  it('filter tab, status, sumber, dan cari berkas digabung', () => {
    expect(filterSecurityChecks(checks, { ...all, tab: 'rate_limit' }).map(c => c.id)).toEqual(['RATE-LIMIT', 'LIVE-RATE-LIMIT'])
    expect(filterSecurityChecks(checks, { ...all, status: 'fail', source: 'static' }).length).toBe(2)
    expect(filterSecurityChecks(checks, { ...all, source: 'live' }).map(c => c.id)).toEqual(['LIVE-RATE-LIMIT', 'LIVE-TRANSPORT'])
    expect(filterSecurityChecks(checks, { ...all, q: 'api/sql' }).map(c => c.id)).toEqual(['SQL-INJECTION'])
  })
  it('URL server hanya http/https', () => {
    expect(securityBaseUrlProblem('')).toBe('')
    expect(securityBaseUrlProblem('127.0.0.1:4101')).toBe('')
    expect(securityBaseUrlProblem('https://api.contoh.com')).toBe('')
    expect(securityBaseUrlProblem('ftp://x')).not.toBe('')
    expect(securityBaseUrlProblem('http://')).not.toBe('')
  })
  it('judul navbar halaman diambil dari menu', () => {
    expect(pageTitleFor('/security-test')).toEqual({ title: 'Uji Keamanan', section: 'Proyek aktif' })
  })
})

describe('uji keamanan: tabel serangan', () => {
  const atk = (category: string, verdict: SecurityAttack['verdict'], payload = '\''): SecurityAttack =>
    ({ category, verdict, payload, route: '/api/items/:id', method: 'GET', path: `/api/items/${payload}`, latency_ms: 1 })
  const list = [atk('sqli', 'safe'), atk('xss', 'suspect'), atk('sqli', 'vulnerable', '1 OR 1'), atk('trace', 'blocked')]
  it('bawaan: hanya tembus & mencurigakan, yang tembus di atas', () => {
    expect(filterAttacks(list, { verdict: 'problems', category: 'all', q: '' }).map(a => a.verdict)).toEqual(['vulnerable', 'suspect'])
  })
  it('filter jenis, ditahan, dan cari payload', () => {
    expect(filterAttacks(list, { verdict: 'all', category: 'sqli', q: '' }).length).toBe(2)
    expect(filterAttacks(list, { verdict: 'blocked', category: 'all', q: '' }).map(a => a.category)).toEqual(['trace'])
    expect(filterAttacks(list, { verdict: 'all', category: 'all', q: 'or 1' }).length).toBe(1)
  })
})
