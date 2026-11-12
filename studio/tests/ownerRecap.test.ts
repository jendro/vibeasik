import { describe, expect, test } from 'bun:test'
import type { OwnerRecap } from '../app/types'
import { recapDayLabels, recapHeadline } from '../app/utils/ownerRecap'
import { costClassInfo, costStackLabel, fmtTicks, shortFormula } from '../app/utils/processCost'

const recap = (over: Partial<OwnerRecap['todos']> = {}, bugs = 0, algo = 0, findings = 0): OwnerRecap => ({
  project_id: 1, days: 30, generated_at: '', health_score: 80, timeline: [], categories: [], health: [], highlights: [],
  todos: { total: 10, done: 6, open: 4, in_progress: 1, suggested: 0, percent: 60, done_in_period: 3, created_in_period: 2, avg_score: 0, scored_count: 0, open_high_priority: 0, ...over },
  bugs: { done: bugs, open: 0, done_in_period: bugs, latest: [] },
  algorithm: { done: algo, open: 0, done_in_period: algo, latest: [] },
  findings: { open: 0, resolved: findings, ignored: 0, open_critical: 0, resolved_in_period: findings, open_by_severity: {} }
})

describe('recapHeadline', () => {
  test('hanya menyebut pencapaian yang angkanya ada', () => {
    expect(recapHeadline(recap())).toBe('Dalam 30 hari terakhir: 3 pekerjaan selesai. 4 masih terbuka.')
    expect(recapHeadline(recap({ open_high_priority: 2 }, 2, 1, 5))).toBe(
      'Dalam 30 hari terakhir: 3 pekerjaan selesai, 2 bug ditutup, 1 perbaikan algoritma, 5 temuan audit dibereskan. 4 masih terbuka, 2 di antaranya prioritas tinggi.')
  })
  test('tanpa pekerjaan terbuka dan proyek kosong', () => {
    expect(recapHeadline(recap({ open: 0 }))).toContain('Tidak ada pekerjaan yang tertunda.')
    expect(recapHeadline(recap({ total: 0 }))).toContain('Belum ada pekerjaan tercatat')
  })
})

test('recapDayLabels memendekkan tanggal', () => {
  expect(recapDayLabels(['2026-03-09', '2026-11-30'])).toEqual(['9/3', '30/11'])
})

describe('kelas biaya proses', () => {
  test('kelas tak dikenal jatuh ke O(1) dan tiap kelas punya label', () => {
    expect(costClassInfo('quadratic').bigO).toBe('O(n²)')
    expect(costClassInfo('entah').key).toBe('constant')
  })
  test('fmtTicks meringkas angka besar', () => {
    expect(fmtTicks(16)).toBe('16')
    expect(fmtTicks(30205)).toBe('30.205')
    expect(fmtTicks(300000000)).toMatch(/300/)
  })
  test('shortFormula menyisakan suku tertinggi', () => {
    expect(shortFormula('3n² + 2n + 5')).toBe('3n² + 2n + 5')
    expect(shortFormula('9n⁴ + 8n³ + 7n² + 6n + 5')).toBe('9n⁴ + 8n³ + 7n² + …')
  })
  test('costStackLabel memberi label Go / framework PHP', () => {
    expect(costStackLabel({ lang: 'go', framework: 'fiber' })).toBe('Go')
    expect(costStackLabel({ lang: 'php', framework: 'Laravel' })).toBe('Laravel')
    expect(costStackLabel({ lang: 'php' })).toBe('PHP')
  })
})
