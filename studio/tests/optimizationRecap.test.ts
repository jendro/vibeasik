import { describe, expect, test } from 'bun:test'
import { buildOptimizationRecap } from '../app/utils/optimizationRecap'

const t = (id: number, category: string | undefined, status: string, completed_at?: string) =>
  ({ id, title: `todo ${id}`, category, status, completed_at })

describe('buildOptimizationRecap', () => {
  test('mengelompokkan todo selesai per kategori, terbanyak di atas', () => {
    const r = buildOptimizationRecap([
      t(1, 'security', 'completed'), t(2, 'performance', 'completed'), t(3, 'performance', 'completed'),
      t(4, 'performance', 'pending')
    ])
    expect(r.done).toBe(3)
    expect(r.open).toBe(1)
    expect(r.categories.map(c => c.category)).toEqual(['performance', 'security'])
    expect(r.categories[0]).toMatchObject({ label: 'Performa', done: 2, open: 1, percent: 67 })
  })

  test('kategori kosong/tak dikenal jatuh ke Umum; status rejected/merged diabaikan', () => {
    const r = buildOptimizationRecap([
      t(1, undefined, 'completed'), t(2, 'aneh', 'completed'), t(3, 'security', 'rejected'), t(4, 'security', 'merged')
    ])
    expect(r.categories).toHaveLength(1)
    expect(r.categories[0]).toMatchObject({ category: 'general', label: 'Umum', done: 2 })
    expect(r.total).toBe(2)
  })

  test('terbaru dulu dan dibatasi', () => {
    const r = buildOptimizationRecap([
      t(1, 'ui_ux', 'completed', '2026-11-01T10:00:00Z'),
      t(2, 'ui_ux', 'completed', '2026-11-03T10:00:00Z'),
      t(3, 'ui_ux', 'completed', '2026-11-02T10:00:00Z')
    ], 2)
    expect(r.categories[0]!.latest.map(x => x.id)).toEqual([2, 3])
  })

  test('tanpa todo: semua nol', () => {
    expect(buildOptimizationRecap([])).toEqual({ done: 0, open: 0, total: 0, percent: 0, categories: [] })
  })
})
