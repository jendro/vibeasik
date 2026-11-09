import { describe, expect, test } from 'bun:test'
import {
  filterAndSortTodos,
  matchesLevelFilter,
  matchesCategoryFilter,
  matchesPriorityFilter,
  matchesTagFilter,
  matchesCreatorFilter,
  matchesEmployeeFilter,
  matchesStatusFilter,
  matchesSearchFilter,
  matchesTodoFilter,
  isAgentCreator,
  isHumanOnlyCreator,
  creatorMatchers,
  creatorPredicates,
  comparePriorityDesc,
  comparePriorityAsc,
  compareImpactDesc,
  compareValueDesc,
  compareCreatedDesc,
  compareCreatedAsc,
  compareTitleAsc,
  compareStatus,
  orderComparators,
  todoComparators,
  getTodoComparator,
  priorityWeights,
  statusWeights,
  type TodoFilterState,
  type TodoOrderBy
} from '../app/utils/todoFilter'
import type { Todo } from '../app/types'

const createMockTodo = (overrides?: Partial<Todo>): Todo => ({
  id: 1,
  project_id: 1,
  level: 'project',
  title: 'Audit arsitektur Nuxt',
  description: 'Refactor modularisasi fungsi kompleks',
  status: 'in_progress',
  priority: 'high',
  category: 'architecture',
  creator_type: 'human',
  creator_name: 'Dewi Kusuma',
  is_ai_suggestion: false,
  source: 'manual',
  tags: 'arsitektur,refactor',
  target_apps: 'studio,api',
  impact_score: 85,
  effort_estimate: 3,
  created_at: '2026-11-01T10:00:00Z',
  ...overrides
})

describe('todoFilter - Ekspor Helper Bernama & Tabel Lookup (K3475, K5273)', () => {
  test('semua helper predikat dan tabel comparator diekspor secara bernama', () => {
    expect(typeof matchesLevelFilter).toBe('function')
    expect(typeof matchesCategoryFilter).toBe('function')
    expect(typeof matchesPriorityFilter).toBe('function')
    expect(typeof matchesTagFilter).toBe('function')
    expect(typeof matchesCreatorFilter).toBe('function')
    expect(typeof matchesEmployeeFilter).toBe('function')
    expect(typeof matchesStatusFilter).toBe('function')
    expect(typeof matchesSearchFilter).toBe('function')
    expect(typeof matchesTodoFilter).toBe('function')
    expect(typeof filterAndSortTodos).toBe('function')
    expect(typeof getTodoComparator).toBe('function')

    expect(typeof creatorMatchers).toBe('object')
    expect(creatorPredicates).toBe(creatorMatchers)

    expect(typeof orderComparators).toBe('object')
    expect(todoComparators).toBe(orderComparators)
  })

  test('tabel orderComparators memuat tepat 8 comparator kanonikal', () => {
    const expectedKeys: TodoOrderBy[] = [
      'priority_desc',
      'priority_asc',
      'impact_desc',
      'value_desc',
      'created_desc',
      'created_asc',
      'title_asc',
      'status'
    ]
    for (const key of expectedKeys) {
      expect(typeof orderComparators[key]).toBe('function')
    }
  })

  test('tabel creatorMatchers memuat seluruh entitas pencipta yang didukung', () => {
    const keys = ['executive', 'audit', 'pm', 'ai_suggestion', 'agent', 'human', 'ai']
    for (const k of keys) {
      expect(typeof creatorMatchers[k]).toBe('function')
    }
  })
})

describe('todoFilter - Table-Driven Predicate Boundary Tests (K64, K4698)', () => {
  describe('matchesLevelFilter', () => {
    const cases = [
      { filter: '', level: 'project', expected: true, desc: 'filter kosong lolos' },
      { filter: 'all', level: 'project', expected: true, desc: 'filter all lolos' },
      { filter: 'project', level: 'project', expected: true, desc: 'level cocok persis' },
      { filter: 'component', level: 'project', expected: false, desc: 'level tidak cocok' },
      { filter: 'project', level: undefined, expected: false, desc: 'level undefined tidak cocok' }
    ]
    for (const c of cases) {
      test(`${c.desc} (${c.filter} vs ${c.level})`, () => {
        const todo = createMockTodo({ level: c.level })
        expect(matchesLevelFilter(todo, c.filter)).toBe(c.expected)
      })
    }
  })

  describe('matchesCategoryFilter', () => {
    const cases = [
      { filter: '', category: 'security', expected: true, desc: 'filter kosong lolos' },
      { filter: 'all', category: 'security', expected: true, desc: 'filter all lolos' },
      { filter: 'security', category: 'security', expected: true, desc: 'kategori cocok' },
      { filter: 'backend', category: 'security', expected: false, desc: 'kategori beda' },
      { filter: 'general', category: undefined, expected: true, desc: 'category undefined fallback ke general' },
      { filter: 'general', category: '', expected: true, desc: 'category kosong fallback ke general' }
    ]
    for (const c of cases) {
      test(`${c.desc} (${c.filter} vs ${c.category})`, () => {
        const todo = createMockTodo({ category: c.category })
        expect(matchesCategoryFilter(todo, c.filter)).toBe(c.expected)
      })
    }
  })

  describe('matchesPriorityFilter', () => {
    const cases = [
      { filter: 'all', priority: 'high', expected: true, desc: 'filter all lolos' },
      { filter: 'critical', priority: 'critical', expected: true, desc: 'prioritas cocok' },
      { filter: 'critical', priority: 'low', expected: false, desc: 'prioritas beda' }
    ]
    for (const c of cases) {
      test(`${c.desc} (${c.filter} vs ${c.priority})`, () => {
        const todo = createMockTodo({ priority: c.priority })
        expect(matchesPriorityFilter(todo, c.filter)).toBe(c.expected)
      })
    }
  })

  describe('matchesTagFilter', () => {
    const cases = [
      { filter: 'all', tags: 'arsitektur,refactor', expected: true, desc: 'filter all lolos' },
      { filter: 'arsitektur', tags: 'arsitektur,refactor', expected: true, desc: 'tag ada di daftar' },
      { filter: 'keamanan', tags: 'arsitektur,refactor', expected: false, desc: 'tag tidak ada' },
      { filter: 'untagged', tags: '', expected: true, desc: 'untagged untuk tags kosong' },
      { filter: 'untagged', tags: undefined, expected: true, desc: 'untagged untuk tags undefined' },
      { filter: 'untagged', tags: 'arsitektur', expected: false, desc: 'untagged gagal jika ada tag' }
    ]
    for (const c of cases) {
      test(`${c.desc} (${c.filter} vs ${c.tags})`, () => {
        const todo = createMockTodo({ tags: c.tags })
        expect(matchesTagFilter(todo, c.filter)).toBe(c.expected)
      })
    }
  })

  describe('matchesStatusFilter (K4909 - Seluruh Kontrak Status)', () => {
    const statuses = ['in_progress', 'pending', 'suggested', 'completed', 'rejected']
    for (const s of statuses) {
      test(`filter status spesifik ${s} cocok hanya dengan status ${s}`, () => {
        const matchTodo = createMockTodo({ status: s })
        expect(matchesStatusFilter(matchTodo, s)).toBe(true)

        const otherStatus = s === 'in_progress' ? 'completed' : 'in_progress'
        const nonMatchTodo = createMockTodo({ status: otherStatus })
        expect(matchesStatusFilter(nonMatchTodo, s)).toBe(false)
      })
    }

    test('filter uncompleted menyaring status aktif dan menyembunyikan completed/rejected', () => {
      expect(matchesStatusFilter(createMockTodo({ status: 'in_progress' }), 'uncompleted')).toBe(true)
      expect(matchesStatusFilter(createMockTodo({ status: 'pending' }), 'uncompleted')).toBe(true)
      expect(matchesStatusFilter(createMockTodo({ status: 'suggested' }), 'uncompleted')).toBe(true)
      expect(matchesStatusFilter(createMockTodo({ status: 'completed' }), 'uncompleted')).toBe(false)
      expect(matchesStatusFilter(createMockTodo({ status: 'rejected' }), 'uncompleted')).toBe(false)
    })

    test('filter all atau kosong meloloskan semua status', () => {
      expect(matchesStatusFilter(createMockTodo({ status: 'completed' }), 'all')).toBe(true)
      expect(matchesStatusFilter(createMockTodo({ status: 'rejected' }), '')).toBe(true)
    })
  })

  describe('matchesCreatorFilter & Origin Classifiers', () => {
    test('isAgentCreator mendeteksi agent coding murni dan mengecualikan eksekutif/audit/pm/ai_sugg', () => {
      const pureAgent = createMockTodo({ creator_type: 'ai_agent', creator_name: 'Antigravity' })
      expect(isAgentCreator(pureAgent)).toBe(true)

      const execTodo = createMockTodo({ source: 'autonomous_executive', creator_type: 'ai_agent' })
      expect(isAgentCreator(execTodo)).toBe(false)

      const auditTodo = createMockTodo({ source: 'diagnostic_audit', creator_type: 'ai_agent' })
      expect(isAgentCreator(auditTodo)).toBe(false)
    })

    test('isHumanOnlyCreator mendeteksi manusia manual murni dan mengecualikan semua origin otomatis', () => {
      const human = createMockTodo({ creator_type: 'human', source: 'manual' })
      expect(isHumanOnlyCreator(human)).toBe(true)

      const nonHuman = createMockTodo({ creator_type: 'ai_agent', source: 'manual' })
      expect(isHumanOnlyCreator(nonHuman)).toBe(false)
    })

    const creatorCases = [
      { creator: 'executive', setup: { source: 'autonomous_executive' }, expected: true },
      { creator: 'audit', setup: { source: 'diagnostic_audit' }, expected: true },
      { creator: 'pm', setup: { source: 'pm_agent' }, expected: true },
      { creator: 'ai_suggestion', setup: { is_ai_suggestion: true }, expected: true },
      { creator: 'agent', setup: { creator_type: 'ai_agent', creator_name: 'Antigravity' }, expected: true },
      { creator: 'human', setup: { creator_type: 'human', source: 'manual' }, expected: true },
      { creator: 'ai', setup: { is_ai_suggestion: true }, expected: true },
      { creator: 'ai', setup: { creator_type: 'ai_agent' }, expected: true },
      { creator: 'unknown_custom', setup: { creator_type: 'human' }, expected: true, desc: 'unknown creator defaults to true' }
    ]
    for (const c of creatorCases) {
      test(`matchesCreatorFilter: ${c.creator}`, () => {
        const todo = createMockTodo(c.setup)
        expect(matchesCreatorFilter(todo, c.creator)).toBe(c.expected)
      })
    }
  })

  describe('matchesEmployeeFilter', () => {
    test('mencocokkan nama dan ID pegawai secara toleran', () => {
      const assigned = createMockTodo({ employee_id: 10, employee_name: 'Dewi Kusuma' })
      const unassigned = createMockTodo({ employee_id: undefined, employee_name: undefined })

      expect(matchesEmployeeFilter(assigned, 'all')).toBe(true)
      expect(matchesEmployeeFilter(assigned, '')).toBe(true)
      expect(matchesEmployeeFilter(assigned, 'dewi')).toBe(true)
      expect(matchesEmployeeFilter(assigned, '10')).toBe(true)
      expect(matchesEmployeeFilter(assigned, 'budi')).toBe(false)
      expect(matchesEmployeeFilter(unassigned, 'unassigned')).toBe(true)
      expect(matchesEmployeeFilter(assigned, 'unassigned')).toBe(false)
    })
  })

  describe('matchesSearchFilter', () => {
    const searchTarget = createMockTodo({
      title: 'Optimalisasi Rendering Canvas Tampilan',
      description: 'Pecah grafik besar ke chunk kecil',
      guideline_ref: 'PERF-009',
      category: 'performance',
      target_apps: 'analytics-dashboard'
    })

    const fieldsTested = [
      { q: 'rendering', field: 'title' },
      { q: 'grafik besar', field: 'description' },
      { q: 'perf-009', field: 'guideline_ref' },
      { q: 'performance', field: 'category' },
      { q: 'analytics-dashboard', field: 'target_apps' }
    ]
    for (const f of fieldsTested) {
      test(`mencocokkan query pencarian pada field ${f.field}`, () => {
        expect(matchesSearchFilter(searchTarget, f.q)).toBe(true)
        expect(matchesSearchFilter(searchTarget, f.q.toUpperCase())).toBe(true)
      })
    }

    test('query kosong atau spasi selalu lolos', () => {
      expect(matchesSearchFilter(searchTarget, '')).toBe(true)
      expect(matchesSearchFilter(searchTarget, '   ')).toBe(true)
    })

    test('kata kunci tidak cocok mengembalikan false', () => {
      expect(matchesSearchFilter(searchTarget, 'kata_kunci_mustahil_ada_xyz')).toBe(false)
    })
  })
})

describe('todoFilter - Table-Driven Order Comparators (K64)', () => {
  const tHigh = createMockTodo({ id: 1, priority: 'high', created_at: '2026-11-01T10:00:00Z', impact_score: 90, effort_estimate: 2, title: 'B Task', status: 'in_progress' })
  const tLow = createMockTodo({ id: 2, priority: 'low', created_at: '2026-11-02T10:00:00Z', impact_score: 40, effort_estimate: 5, title: 'A Task', status: 'completed' })

  test('priority_desc: prioritas tinggi di atas; jika sama, created_at terbaru di atas', () => {
    expect(comparePriorityDesc(tHigh, tLow)).toBeLessThan(0)
    expect(comparePriorityDesc(tLow, tHigh)).toBeGreaterThan(0)

    const tCritNew = createMockTodo({ priority: 'critical', created_at: '2026-11-05T00:00:00Z' })
    const tCritOld = createMockTodo({ priority: 'critical', created_at: '2026-11-01T00:00:00Z' })
    expect(comparePriorityDesc(tCritNew, tCritOld)).toBeLessThan(0)
  })

  test('priority_asc: prioritas rendah di atas; jika sama, created_at terbaru di atas', () => {
    expect(comparePriorityAsc(tLow, tHigh)).toBeLessThan(0)
    expect(comparePriorityAsc(tHigh, tLow)).toBeGreaterThan(0)
  })

  test('impact_desc: impact score lebih tinggi di atas', () => {
    expect(compareImpactDesc(tHigh, tLow)).toBeLessThan(0)
    expect(compareImpactDesc(tLow, tHigh)).toBeGreaterThan(0)
  })

  test('value_desc: value impact/effort lebih tinggi di atas', () => {
    expect(compareValueDesc(tHigh, tLow)).toBeLessThan(0)
    expect(compareValueDesc(tLow, tHigh)).toBeGreaterThan(0)
  })

  test('created_desc: created_at lebih baru di atas', () => {
    expect(compareCreatedDesc(tLow, tHigh)).toBeLessThan(0)
    expect(compareCreatedDesc(tHigh, tLow)).toBeGreaterThan(0)
  })

  test('created_asc: created_at lebih lama di atas', () => {
    expect(compareCreatedAsc(tHigh, tLow)).toBeLessThan(0)
    expect(compareCreatedAsc(tLow, tHigh)).toBeGreaterThan(0)
  })

  test('title_asc: urutan abjad judul A-Z', () => {
    expect(compareTitleAsc(tLow, tHigh)).toBeLessThan(0) // 'A Task' vs 'B Task'
    expect(compareTitleAsc(tHigh, tLow)).toBeGreaterThan(0)
  })

  test('status: status berurutan in_progress < pending < suggested < completed < rejected', () => {
    const tInProg = createMockTodo({ status: 'in_progress' })
    const tPend = createMockTodo({ status: 'pending' })
    const tSugg = createMockTodo({ status: 'suggested' })
    const tComp = createMockTodo({ status: 'completed' })
    const tRej = createMockTodo({ status: 'rejected' })

    expect(compareStatus(tInProg, tPend)).toBeLessThan(0)
    expect(compareStatus(tPend, tSugg)).toBeLessThan(0)
    expect(compareStatus(tSugg, tComp)).toBeLessThan(0)
    expect(compareStatus(tComp, tRej)).toBeLessThan(0)
  })

  test('getTodoComparator menyelesaikan nama langsung, komposit, dan fallback aman', () => {
    expect(getTodoComparator('impact_desc')).toBe(compareImpactDesc)
    expect(getTodoComparator('priority', 'asc')).toBe(comparePriorityAsc)
    expect(getTodoComparator('status')).toBe(compareStatus)
    expect(typeof getTodoComparator(undefined)).toBe('function')
    expect(getTodoComparator('unknown_order')(tHigh, tLow)).toBe(0)
  })
})

describe('todoFilter - Characterization & Immutability Tests (K5302, K1971)', () => {
  const sampleList: Todo[] = [
    createMockTodo({ id: 101, title: 'Bug Login', priority: 'critical', status: 'in_progress', category: 'auth', level: 'project', created_at: '2026-11-01T00:00:00Z' }),
    createMockTodo({ id: 102, title: 'Audit Cache', priority: 'high', status: 'pending', category: 'backend', level: 'project', created_at: '2026-11-02T00:00:00Z' }),
    createMockTodo({ id: 103, title: 'Fix Layout Mobile', priority: 'medium', status: 'completed', category: 'ui_ux', level: 'component', created_at: '2026-11-03T00:00:00Z' }),
    createMockTodo({ id: 104, title: 'Hapus dead code', priority: 'low', status: 'rejected', category: 'general', level: 'project', created_at: '2026-11-04T00:00:00Z' })
  ]

  const baseFilter: TodoFilterState = {
    level: 'all',
    category: 'all',
    creator: 'all',
    tag: 'all',
    status: 'uncompleted',
    orderBy: 'priority_desc',
    search: ''
  }

  test('filterAndSortTodos: tidak memutasi array input asli (K1971 Immutability)', () => {
    const originalCopy = [...sampleList]
    const result = filterAndSortTodos(sampleList, baseFilter)

    expect(result).not.toBe(sampleList)
    expect(sampleList.length).toBe(originalCopy.length)
    expect(sampleList[0].id).toBe(originalCopy[0].id)
  })

  test('filterAndSortTodos: pipeline orkestrator menyaring status uncompleted dan mengurutkan priority_desc', () => {
    const result = filterAndSortTodos(sampleList, baseFilter)

    // uncompleted harus menyisakan bug login (critical) dan audit cache (high)
    expect(result).toHaveLength(2)
    expect(result[0].id).toBe(101) // critical
    expect(result[1].id).toBe(102) // high
  })

  test('filterAndSortTodos: mendukung filter level spesifik dan query teks terpadu', () => {
    const filter: TodoFilterState = {
      ...baseFilter,
      level: 'component',
      status: 'all',
      search: 'layout'
    }
    const result = filterAndSortTodos(sampleList, filter)
    expect(result).toHaveLength(1)
    expect(result[0].id).toBe(103)
  })

  test('filterAndSortTodos: menangani array kosong tanpa error', () => {
    const result = filterAndSortTodos([], baseFilter)
    expect(result).toEqual([])
  })

  test('filterAndSortTodos: mendukung f.employee bila dispesifikasikan', () => {
    const listWithEmployee: Todo[] = [
      createMockTodo({ id: 201, employee_name: 'Dewi Kusuma', status: 'in_progress' }),
      createMockTodo({ id: 202, employee_name: 'Nayla Maharani', status: 'in_progress' })
    ]
    const filter: TodoFilterState = {
      ...baseFilter,
      employee: 'Dewi'
    }
    const result = filterAndSortTodos(listWithEmployee, filter)
    expect(result).toHaveLength(1)
    expect(result[0].id).toBe(201)
  })
})
