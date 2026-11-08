import { describe, expect, test } from 'bun:test'
import { recapHighlights } from '../app/utils/recap'
import type { AgentRecap } from '../app/types'

const day = (o: Record<string, number | string> = {}) => ({
  date: '2026-11-08', commits: 0, additions: 0, deletions: 0, todos_created: 0, todos_completed: 0,
  todos_completed_by_ai: 0, ai_commits: 0, runs_completed: 0, ...o
})

const recap = (o: Record<string, unknown> = {}) => ({
  from: '2026-11-02', to: '2026-11-08', days: [],
  today: day(), yesterday: day(), totals: day(),
  streak: { current: 0, longest: 0 }, repos: null, categories: null, priorities: null,
  ...o
}) as unknown as AgentRecap

describe('recapHighlights', () => {
  test('kosong tanpa pencapaian', () => {
    expect(recapHighlights(recap())).toEqual([])
  })

  test('semua kategori terisi, urutan tetap', () => {
    const r = recap({
      streak: { current: 5, longest: 5 },
      today: day({ commits: 4, additions: 1200, todos_completed: 3, todos_completed_by_ai: 2, ai_commits: 1, runs_completed: 2, todos_created: 6 }),
      yesterday: day({ commits: 2 }),
      totals: day({ todos_completed: 1500, todos_created: 1200 }),
      priorities: [{ key: 'critical', count: 2 }, { key: 'high', count: 1 }, { key: 'low', count: 9 }],
      categories: [{ key: 'security', count: 3 }],
      best_day: day({ commits: 4 }),
      repos: [{ today_commits: 1 }, { today_commits: 2 }, { today_commits: 0 }]
    })
    expect(recapHighlights(r)).toMatchSnapshot()
  })

  test('cabang alternatif', () => {
    const r = recap({
      streak: { current: 3, longest: 9 },
      today: day({ commits: 1 }),
      yesterday: day({ commits: 5 }),
      totals: day({ todos_completed: 2, todos_created: 5 }),
      best_day: day({ date: '2026-11-07', commits: 9 }),
      repos: [{ today_commits: 1 }]
    })
    expect(recapHighlights(r)).toMatchSnapshot()
    expect(recapHighlights(recap({ streak: { current: 1, longest: 1 }, best_day: day() }))).toEqual([])
    expect(recapHighlights(recap({ best_day: day({ todos_completed: 1 }) })).map(h => h.icon)).toEqual(['i-lucide-trophy'])
  })
})
