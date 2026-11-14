import { describe, expect, it } from 'bun:test'
import type { OwnerRecap } from '../app/types'
import { nextImprovementStep, STALE_AUDIT_DAYS } from '../app/utils/improvementFlow'
import { actionTitle } from '../app/utils/actionKinds'

const NOW = Date.parse('2026-11-14T10:00:00Z')
const daysAgo = (d: number) => new Date(NOW - d * 86_400_000).toISOString()

const recap = (p: { audit?: string, findings?: number, critical?: number, todos?: number, high?: number }) => ({
  findings: { open: p.findings ?? 0, open_critical: p.critical ?? 0, last_audit_at: p.audit },
  todos: { open: p.todos ?? 0, open_high_priority: p.high ?? 0 }
}) as unknown as OwnerRecap

describe('langkah berikutnya', () => {
  it('belum pernah diperiksa → periksa', () => {
    expect(nextImprovementStep(recap({}), NOW)).toMatchObject({ step: 'check', action: 'scan' })
  })
  it('pemeriksaan basi → periksa ulang walau ada todo', () => {
    expect(nextImprovementStep(recap({ audit: daysAgo(STALE_AUDIT_DAYS), todos: 3 }), NOW)).toMatchObject({ step: 'check', action: 'scan' })
  })
  it('temuan belum jadi todo → tinjau temuan, sebut yang kritis', () => {
    const s = nextImprovementStep(recap({ audit: daysAgo(1), findings: 5, critical: 2, todos: 4 }), NOW)
    expect(s).toMatchObject({ step: 'check', action: 'process' })
    expect(s.reason).toContain('2 di antaranya kritis')
  })
  it('semua temuan diproses, todo terbuka → kerjakan todo', () => {
    expect(nextImprovementStep(recap({ audit: daysAgo(1), todos: 4, high: 1 }), NOW)).toMatchObject({ step: 'work' })
  })
  it('tidak ada yang tersisa → lihat hasil', () => {
    expect(nextImprovementStep(recap({ audit: daysAgo(0) }), NOW)).toMatchObject({ step: 'result' })
  })
})

describe('tooltip jenis tombol', () => {
  it('aksi proses yang memanggil LLM menyebut kuota', () => {
    expect(actionTitle('process', true, 'Selesaikan semua')).toContain('LLM')
    expect(actionTitle('scan', false)).toContain('tanpa LLM')
  })
})
