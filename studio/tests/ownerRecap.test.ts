import { describe, expect, test } from 'bun:test'
import type { OwnerRecap } from '../app/types'
import { recapDayLabels, recapHeadline } from '../app/utils/ownerRecap'
import { costClassInfo, costHandledBadge, costHandlingNote, costStackLabel, fmtTicks, shortFormula } from '../app/utils/processCost'

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

describe('costHandlingNote', () => {
  test('menyebut tempat penanganan dan langkah berikutnya', () => {
    const n = costHandlingNote({ sources: 5, in_todo: 2, in_audit: 1, ignored: 0, recheck: 0, open: 2, todos: [{ id: 12, title: 'a', status: 'pending' }, { id: 15, title: 'b', status: 'in_progress' }] })
    expect(n.title).toBe('Sudah ditangani 3 dari 5 sumber biaya tinggi')
    expect(n.where).toBe('2 di todo (#12, #15) · 1 di temuan Audit')
    expect(n.next).toContain('kerjakan todo-nya')
    expect(n.next).toContain('buat todo untuk 2 sumber')
  })
  test('belum ada yang ditangani', () => {
    const n = costHandlingNote({ sources: 3, in_todo: 0, in_audit: 0, ignored: 0, recheck: 0, open: 3, todos: [] })
    expect(n.title).toBe('3 sumber biaya tinggi belum ditangani')
    expect(n.where).toBe('')
  })
  test('label penanda baris', () => {
    const src = { kind: 'nplus1' as const, rule: 'R', func: 'f', file: 'a.go', line: 1, big_o: 'O(n)' }
    expect(costHandledBadge({ where: 'todo', todo_id: 7, todo_status: 'pending', rule: 'R', source: src, next: '' }).label).toBe('Todo #7 · menunggu')
    expect(costHandledBadge({ where: 'audit', rule: 'R', source: src, next: '' }).label).toBe('Di Audit')
  })
})
