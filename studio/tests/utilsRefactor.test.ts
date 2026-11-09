import { describe, expect, test } from 'bun:test'
import {
  getFindingKey,
  areFindingsMatching,
  hasSameId,
  hasSameGeneratedKey,
  hasSameLocation,
  hasSameRuleAtLocation,
  hasSameTitleAtLocation,
  deduplicateFindings,
  mergeFindingsLists
} from '../app/utils/auditFindings'
import {
  filterSynergyPages,
  matchesMethod,
  matchesTableOrStatus,
  matchesMethodAndStatus,
  matchesMethodAndTable,
  matchesPageSearch,
  matchesRequestSearch,
  matchesSearchQuery,
  filterPageRequests,
  shouldKeepPage
} from '../app/utils/synergy'
import {
  todoOriginMeta,
  todoRecommendedModelMeta,
  isFlagshipTask,
  isFastTask,
  MODEL_SUGGESTIONS,
  isExecutiveOrigin,
  isAuditOrigin,
  isPmOrigin,
  isAiSuggestionOrigin,
  isAgentOrigin,
  createExecutiveOriginMeta,
  createAuditOriginMeta,
  createPmOriginMeta,
  createAiSuggestionOriginMeta,
  createAgentOriginMeta,
  createHumanOriginMeta
} from '../app/utils/todoCardMeta'
import {
  filterAndSortTodos,
  matchesLevelFilter,
  matchesCategoryFilter,
  matchesPriorityFilter,
  matchesTagFilter,
  isAgentCreator,
  isHumanOnlyCreator,
  matchesCreatorFilter,
  matchesEmployeeFilter,
  matchesStatusFilter,
  matchesSearchFilter,
  matchesTodoFilter,
  comparePriorityDesc,
  comparePriorityAsc,
  compareImpactDesc,
  compareValueDesc,
  compareCreatedDesc,
  compareCreatedAsc,
  compareTitleAsc,
  compareStatus,
  getTodoComparator
} from '../app/utils/todoFilter'
import type { Finding, ProjectSynergyMap, SynergyFrontendPage, SynergyRequestFlow, Todo } from '../app/types'

describe('auditFindings helper refactor (DEEP-FUNC-HIGH-COMPLEXITY CC reduction)', () => {
  const baseFinding = (overrides?: Partial<Finding>): Finding => ({
    id: '',
    rule_id: 'SEC-001',
    category: 'security',
    severity: 'WARNING',
    title: 'SQL Injection Vulnerability',
    description: 'Unescaped user input',
    target_file: 'src/db.ts',
    line: 42,
    snippet: 'db.query(...)',
    status: 'open',
    ...overrides
  })

  test('hasSameId: cocok jika ID kedua finding sama dan tidak kosong', () => {
    const f1 = baseFinding({ id: 'finding-1' })
    const f2 = baseFinding({ id: 'finding-1' })
    const f3 = baseFinding({ id: 'finding-2' })
    const fEmpty1 = baseFinding({ id: '' })
    const fEmpty2 = baseFinding({ id: '' })

    expect(hasSameId(f1, f2)).toBe(true)
    expect(hasSameId(f1, f3)).toBe(false)
    expect(hasSameId(fEmpty1, fEmpty2)).toBe(false)
  })

  test('hasSameGeneratedKey: cocok berdasarkan getFindingKey yang dihasilkan', () => {
    const f1 = baseFinding({ id: '', rule_id: 'SEC-001', target_file: 'src/a.ts', line: 10 })
    const f2 = baseFinding({ id: '', rule_id: 'SEC-001', target_file: 'src/a.ts', line: 10 })
    const f3 = baseFinding({ id: '', rule_id: 'SEC-002', target_file: 'src/a.ts', line: 10 })

    expect(hasSameGeneratedKey(f1, f2)).toBe(true)
    expect(hasSameGeneratedKey(f1, f3)).toBe(false)
  })

  test('hasSameLocation: membandingkan target_file dan line secara toleran terhadap null/undefined', () => {
    const f1 = baseFinding({ target_file: 'src/api.ts', line: 20 })
    const f2 = baseFinding({ target_file: 'src/api.ts', line: 20 })
    const fDiffLine = baseFinding({ target_file: 'src/api.ts', line: 25 })
    const fEmptyLoc1 = baseFinding({ target_file: undefined, line: undefined })
    const fEmptyLoc2 = baseFinding({ target_file: '', line: 0 })

    expect(hasSameLocation(f1, f2)).toBe(true)
    expect(hasSameLocation(f1, fDiffLine)).toBe(false)
    expect(hasSameLocation(fEmptyLoc1, fEmptyLoc2)).toBe(true)
  })

  test('hasSameRuleAtLocation: cocok jika rule_id dan lokasi sama', () => {
    const f1 = baseFinding({ rule_id: 'RULE-A', target_file: 'file.ts', line: 5 })
    const f2 = baseFinding({ rule_id: 'RULE-A', target_file: 'file.ts', line: 5 })
    const fDiffRule = baseFinding({ rule_id: 'RULE-B', target_file: 'file.ts', line: 5 })
    const fNoRule1 = baseFinding({ rule_id: '', target_file: 'file.ts', line: 5 })
    const fNoRule2 = baseFinding({ rule_id: '', target_file: 'file.ts', line: 5 })

    expect(hasSameRuleAtLocation(f1, f2)).toBe(true)
    expect(hasSameRuleAtLocation(f1, fDiffRule)).toBe(false)
    expect(hasSameRuleAtLocation(fNoRule1, fNoRule2)).toBe(false)
  })

  test('hasSameTitleAtLocation: cocok jika title dan lokasi sama sebagai fallback bila rule_id kosong', () => {
    const f1 = baseFinding({ rule_id: '', title: 'Sama', target_file: 'file.ts', line: 5 })
    const f2 = baseFinding({ rule_id: '', title: 'Sama', target_file: 'file.ts', line: 5 })
    const fDiffTitle = baseFinding({ rule_id: '', title: 'Beda', target_file: 'file.ts', line: 5 })

    expect(hasSameTitleAtLocation(f1, f2)).toBe(true)
    expect(hasSameTitleAtLocation(f1, fDiffTitle)).toBe(false)
  })

  test('areFindingsMatching: orkestrator guard clause mengenali pencocokan ID, key, rule, atau title', () => {
    const fIdA = baseFinding({ id: 'id-xyz' })
    const fIdB = baseFinding({ id: 'id-xyz' })
    expect(areFindingsMatching(fIdA, fIdB)).toBe(true)

    const fKeyA = baseFinding({ id: '', rule_id: 'R1', target_file: 'a.ts', line: 1 })
    const fKeyB = baseFinding({ id: '', rule_id: 'R1', target_file: 'a.ts', line: 1 })
    expect(areFindingsMatching(fKeyA, fKeyB)).toBe(true)

    const fRuleA = baseFinding({ id: 'id-1', rule_id: 'R2', target_file: 'b.ts', line: 2 })
    const fRuleB = baseFinding({ id: 'id-2', rule_id: 'R2', target_file: 'b.ts', line: 2 })
    expect(areFindingsMatching(fRuleA, fRuleB)).toBe(true)

    const fTitleA = baseFinding({ id: 'id-3', rule_id: '', title: 'Title Match', target_file: 'c.ts', line: 3 })
    const fTitleB = baseFinding({ id: 'id-4', rule_id: '', title: 'Title Match', target_file: 'c.ts', line: 3 })
    expect(areFindingsMatching(fTitleA, fTitleB)).toBe(true)

    const fMismatch = baseFinding({ id: 'id-5', rule_id: 'R5', target_file: 'd.ts', line: 4 })
    const fOther = baseFinding({ id: 'id-6', rule_id: 'R6', target_file: 'e.ts', line: 99 })
    expect(areFindingsMatching(fMismatch, fOther)).toBe(false)
  })

  test('deduplicateFindings: menghapus temuan duplikat dan mempertahankan status resolved/ignored', () => {
    const list: Finding[] = [
      baseFinding({ id: 'f-1', status: 'open', target_file: 'src/db.ts', line: 10 }),
      baseFinding({ id: 'f-1', status: 'resolved', target_file: 'src/db.ts', line: 10 }),
      baseFinding({ id: 'f-2', status: 'open', target_file: 'src/other.ts', line: 20 })
    ]
    const deduped = deduplicateFindings(list)
    expect(deduped).toHaveLength(2)
    expect(deduped[0].status).toBe('resolved')
  })

  test('mergeFindingsLists: menggabungkan daftar baru tanpa menghapus temuan lama', () => {
    const existing = [baseFinding({ id: 'old-1', status: 'open', target_file: 'src/old.ts', line: 1 })]
    const incoming = [
      baseFinding({ id: 'old-1', status: 'resolved', target_file: 'src/old.ts', line: 1, snippet: 'new code' }),
      baseFinding({ id: 'brand-new', status: 'open', target_file: 'src/new.ts', line: 2 })
    ]
    const merged = mergeFindingsLists(existing, incoming)
    expect(merged).toHaveLength(2)
    expect(merged.find(f => f.id === 'old-1')?.status).toBe('resolved')
    expect(merged.find(f => f.id === 'brand-new')).toBeDefined()
  })
})

describe('synergy helper refactor (filterSynergyPages CC reduction)', () => {
  const createMockReq = (overrides?: Partial<SynergyRequestFlow>): SynergyRequestFlow => ({
    http_method: 'GET',
    route_path: '/api/users',
    action_title: 'Ambil daftar pengguna',
    trigger_event: 'klik tab pengguna',
    handler_function: 'getUsersHandler',
    middleware: 'authMiddleware',
    database_operation: {
      engine: 'PostgreSQL',
      target_tables: ['users', 'roles'],
      stored_fields: ['name', 'email']
    },
    response_spec: {
      status_code: 200,
      data_type: 'application/json'
    },
    ...overrides
  })

  const createMockPage = (overrides?: Partial<SynergyFrontendPage>): SynergyFrontendPage => ({
    title: 'Manajemen Pengguna',
    page_path: '/admin/users',
    file_path: 'pages/admin/users.vue',
    navigation_targets: [],
    outbound_requests: [createMockReq()],
    ...overrides
  })

  test('matchesMethod: menyaring metode HTTP secara case-insensitive atau lolos jika ALL', () => {
    const req = createMockReq({ http_method: 'POST' })
    expect(matchesMethod(req, 'ALL')).toBe(true)
    expect(matchesMethod(req, 'post')).toBe(true)
    expect(matchesMethod(req, 'GET')).toBe(false)
  })

  test('matchesTableOrStatus: menyaring tabel DB atau kode status response', () => {
    const req = createMockReq({
      database_operation: { target_tables: ['users', 'audit_logs'] },
      response_spec: { status_code: 201 }
    })
    expect(matchesTableOrStatus(req, 'all')).toBe(true)
    expect(matchesTableOrStatus(req, 'users')).toBe(true)
    expect(matchesTableOrStatus(req, 'audit_logs')).toBe(true)
    expect(matchesTableOrStatus(req, '201')).toBe(true)
    expect(matchesTableOrStatus(req, 'orders')).toBe(false)
    expect(matchesTableOrStatus(req, '500')).toBe(false)
  })

  test('matchesMethodAndStatus: memadukan filter metode dan tabel/status', () => {
    const req = createMockReq({ http_method: 'POST' })
    expect(matchesMethodAndStatus(req, 'POST', 'users')).toBe(true)
    expect(matchesMethodAndStatus(req, 'GET', 'users')).toBe(false)
    expect(matchesMethodAndTable(req, 'POST', 'users')).toBe(true)
  })

  test('matchesPageSearch: mencari kecocokan pada path dan title halaman', () => {
    const page = createMockPage({ title: 'Dashboard Analytics', page_path: '/dashboard/overview' })
    expect(matchesPageSearch(page, '')).toBe(true)
    expect(matchesPageSearch(page, 'analytics')).toBe(true)
    expect(matchesPageSearch(page, 'overview')).toBe(true)
    expect(matchesPageSearch(page, 'settings')).toBe(false)
  })

  test('matchesRequestSearch: mencari kecocokan pada atribut request (handler, route, tables, rules)', () => {
    const req = createMockReq({
      handler_function: 'exportReportHandler',
      validation_rules: ['date_from:required']
    })
    expect(matchesRequestSearch(req, '')).toBe(true)
    expect(matchesRequestSearch(req, 'report')).toBe(true)
    expect(matchesRequestSearch(req, 'required')).toBe(true)
    expect(matchesRequestSearch(req, 'unknown_term')).toBe(false)
  })

  test('matchesSearchQuery: memadukan pencarian halaman dan request', () => {
    const page = createMockPage({ title: 'Pengaturan', page_path: '/settings' })
    const req = createMockReq({ action_title: 'Simpan Profil' })
    expect(matchesSearchQuery(page, req, 'pengaturan')).toBe(true)
    expect(matchesSearchQuery(page, req, 'profil')).toBe(true)
    expect(matchesSearchQuery(page, req, 'laporan')).toBe(false)
  })

  test('filterPageRequests: menyaring requests di dalam halaman', () => {
    const page = createMockPage({
      outbound_requests: [
        createMockReq({ http_method: 'GET', route_path: '/api/v1/users' }),
        createMockReq({ http_method: 'POST', route_path: '/api/v1/users' })
      ]
    })
    const filtered = filterPageRequests(page, '', 'POST', 'all')
    expect(filtered).toHaveLength(1)
    expect(filtered[0].http_method).toBe('POST')
  })

  test('shouldKeepPage: logika retensi halaman berdasarkan kecocokan query & filter', () => {
    expect(shouldKeepPage(true, 1, false)).toBe(true)
    expect(shouldKeepPage(false, 1, false)).toBe(true)
    expect(shouldKeepPage(true, 0, false)).toBe(true)
    expect(shouldKeepPage(false, 0, false)).toBe(false)

    // Saat filter aktif: hanya simpan jika request cocok > 0
    expect(shouldKeepPage(true, 1, true)).toBe(true)
    expect(shouldKeepPage(true, 0, true)).toBe(false)
  })

  test('filterSynergyPages: memfilter seluruh peta sinergi dengan benar', () => {
    const map: ProjectSynergyMap = {
      project_name: 'Test App',
      summary: 'Testing synergy',
      total_pages: 2,
      total_flows: 2,
      total_tables: 1,
      pages: [
        createMockPage({ title: 'Halaman User', page_path: '/user/profile', outbound_requests: [createMockReq({ http_method: 'GET' })] }),
        createMockPage({ title: 'Halaman Admin', page_path: '/admin/dashboard', outbound_requests: [createMockReq({ http_method: 'DELETE' })] })
      ]
    }

    expect(filterSynergyPages(null, '', 'ALL', 'all')).toEqual([])
    const getPages = filterSynergyPages(map, '', 'GET', 'all')
    expect(getPages).toHaveLength(1)
    expect(getPages[0].title).toBe('Halaman User')

    const searchPages = filterSynergyPages(map, 'admin', 'ALL', 'all')
    expect(searchPages).toHaveLength(1)
    expect(searchPages[0].title).toBe('Halaman Admin')
  })
})

describe('todoCardMeta helper refactor (todoOriginMeta CC reduction)', () => {
  const baseTodo = (overrides?: Partial<Todo>): Todo => ({
    id: 1,
    level: 'project',
    title: 'Periksa audit',
    description: 'Deskripsi todo',
    status: 'pending',
    priority: 'medium',
    creator_type: 'human',
    creator_name: 'Budi',
    is_ai_suggestion: false,
    source: 'manual',
    ...overrides
  })

  test('isExecutiveOrigin: deteksi dewan eksekutif dengan benar', () => {
    expect(isExecutiveOrigin(baseTodo({ source: 'autonomous_executive' }))).toBe(true)
    expect(isExecutiveOrigin(baseTodo({ source: 'executive_chat' }))).toBe(true)
    expect(isExecutiveOrigin(baseTodo({ creator_name: 'Dewan Eksekutif Vibeasik' }))).toBe(true)
    expect(isExecutiveOrigin(baseTodo({ creator_type: 'executive' as any }))).toBe(true)
    expect(isExecutiveOrigin(baseTodo({ source: 'manual', creator_name: 'Joko' }))).toBe(false)
  })

  test('isAuditOrigin: deteksi temuan audit & diagnostic', () => {
    expect(isAuditOrigin(baseTodo({ source: 'diagnostic_audit' }))).toBe(true)
    expect(isAuditOrigin(baseTodo({ source: 'audit' }))).toBe(true)
    expect(isAuditOrigin(baseTodo({ audit_key: 'AUDIT_01' }))).toBe(true)
    expect(isAuditOrigin(baseTodo({ guideline_ref: 'SEC-001' }))).toBe(true)
    expect(isAuditOrigin(baseTodo({ source: 'manual', guideline_ref: undefined }))).toBe(false)
  })

  test('isPmOrigin: deteksi perencana & project manager', () => {
    expect(isPmOrigin(baseTodo({ source: 'pm_agent' }))).toBe(true)
    expect(isPmOrigin(baseTodo({ source: 'planner' }))).toBe(true)
    expect(isPmOrigin(baseTodo({ creator_name: 'AI PM Assistant' }))).toBe(true)
    expect(isPmOrigin(baseTodo({ creator_name: 'Developer' }))).toBe(false)
  })

  test('isAiSuggestionOrigin: deteksi rekomendasi dan template AI', () => {
    expect(isAiSuggestionOrigin(baseTodo({ is_ai_suggestion: true }))).toBe(true)
    expect(isAiSuggestionOrigin(baseTodo({ creator_name: 'Architecture Advisor' }))).toBe(true)
    expect(isAiSuggestionOrigin(baseTodo({ source: 'ai_template' }))).toBe(true)
    expect(isAiSuggestionOrigin(baseTodo({ is_ai_suggestion: false, source: 'manual' }))).toBe(false)
  })

  test('isAgentOrigin: deteksi AI Coding Agent', () => {
    expect(isAgentOrigin(baseTodo({ creator_type: 'ai_agent' }))).toBe(true)
    expect(isAgentOrigin(baseTodo({ creator_name: 'Claude Code Agent' }))).toBe(true)
    expect(isAgentOrigin(baseTodo({ creator_name: 'Antigravity' }))).toBe(true)
    expect(isAgentOrigin(baseTodo({ creator_name: 'OpenCode Assistant' }))).toBe(true)
    expect(isAgentOrigin(baseTodo({ creator_name: 'Nayla', creator_type: 'human' }))).toBe(false)
  })

  test('create*OriginMeta builders: mengembalikan objek metadata lengkap', () => {
    const t = baseTodo({ creator_name: 'Claude 3.7', guideline_ref: 'PERF-001' })
    expect(createExecutiveOriginMeta(t).type).toBe('executive')
    expect(createAuditOriginMeta(t).sublabel).toBe('PERF-001')
    expect(createPmOriginMeta(t).type).toBe('pm')
    expect(createAiSuggestionOriginMeta(t).type).toBe('ai_suggestion')
    expect(createAgentOriginMeta(t).type).toBe('agent')
    expect(createHumanOriginMeta(t).type).toBe('human')
  })

  test('todoOriginMeta: mempertahankan urutan prioritas klasifikasi 100%', () => {
    // 1. Eksekutif menang atas audit
    const execWithAudit = baseTodo({
      source: 'autonomous_executive',
      guideline_ref: 'SEC-001'
    })
    expect(todoOriginMeta(execWithAudit).type).toBe('executive')

    // 2. Audit menang atas PM
    const auditWithPm = baseTodo({
      source: 'diagnostic_audit',
      creator_name: 'pm-agent'
    })
    expect(todoOriginMeta(auditWithPm).type).toBe('audit')

    // 3. PM menang atas saran AI
    const pmWithSugg = baseTodo({
      source: 'pm_agent',
      is_ai_suggestion: true
    })
    expect(todoOriginMeta(pmWithSugg).type).toBe('pm')

    // 4. AI suggestion menang atas coding agent
    const suggWithAgent = baseTodo({
      is_ai_suggestion: true,
      creator_type: 'ai_agent',
      creator_name: 'claude'
    })
    expect(todoOriginMeta(suggWithAgent).type).toBe('ai_suggestion')

    // 5. Coding Agent
    const pureAgent = baseTodo({
      creator_type: 'ai_agent',
      creator_name: 'antigravity'
    })
    expect(todoOriginMeta(pureAgent).type).toBe('agent')

    // 6. Human fallback
    const humanTodo = baseTodo({
      creator_type: 'human',
      creator_name: 'Nayla Maharani'
    })
    expect(todoOriginMeta(humanTodo).type).toBe('human')
  })
})

describe('todoFilter helper refactor (filterAndSortTodos CC reduction)', () => {
  const mockTodo = (overrides?: Partial<Todo>): Todo => ({
    id: 1,
    level: 'project',
    title: 'Migrasi database',
    description: 'Deskripsi refactor',
    status: 'in_progress',
    priority: 'high',
    category: 'backend',
    creator_type: 'human',
    creator_name: 'Ahmad',
    is_ai_suggestion: false,
    source: 'manual',
    tags: 'fitur,keamanan-dasar',
    target_apps: 'api,admin',
    impact_score: 80,
    effort_estimate: 2,
    created_at: '2026-11-01T10:00:00Z',
    ...overrides
  })

  test('matchesLevelFilter: memfilter berdasarkan level atau all', () => {
    const t = mockTodo({ level: 'project' })
    expect(matchesLevelFilter(t, 'all')).toBe(true)
    expect(matchesLevelFilter(t, 'project')).toBe(true)
    expect(matchesLevelFilter(t, 'component')).toBe(false)
  })

  test('matchesCategoryFilter: memfilter berdasarkan kategori dengan fallback general', () => {
    const t = mockTodo({ category: 'security' })
    const tEmpty = mockTodo({ category: undefined })
    expect(matchesCategoryFilter(t, 'all')).toBe(true)
    expect(matchesCategoryFilter(t, 'security')).toBe(true)
    expect(matchesCategoryFilter(t, 'ui_ux')).toBe(false)
    expect(matchesCategoryFilter(tEmpty, 'general')).toBe(true)
  })

  test('matchesPriorityFilter: memfilter prioritas', () => {
    const t = mockTodo({ priority: 'critical' })
    expect(matchesPriorityFilter(t, 'all')).toBe(true)
    expect(matchesPriorityFilter(t, 'critical')).toBe(true)
    expect(matchesPriorityFilter(t, 'low')).toBe(false)
  })

  test('matchesTagFilter: memfilter tag fokus, untagged, atau all', () => {
    const tTagged = mockTodo({ tags: 'belajar,fitur' })
    const tUntagged = mockTodo({ tags: '' })
    expect(matchesTagFilter(tTagged, 'all')).toBe(true)
    expect(matchesTagFilter(tTagged, 'belajar')).toBe(true)
    expect(matchesTagFilter(tTagged, 'optimasi-query')).toBe(false)
    expect(matchesTagFilter(tUntagged, 'untagged')).toBe(true)
    expect(matchesTagFilter(tTagged, 'untagged')).toBe(false)
  })

  test('isAgentCreator & isHumanOnlyCreator: eksklusi pembuat', () => {
    const execTodo = mockTodo({ source: 'autonomous_executive', creator_name: 'claude' })
    expect(isAgentCreator(execTodo)).toBe(false)

    const agentTodo = mockTodo({ creator_type: 'ai_agent', creator_name: 'antigravity' })
    expect(isAgentCreator(agentTodo)).toBe(true)
    expect(isHumanOnlyCreator(agentTodo)).toBe(false)

    const humanTodo = mockTodo({ creator_type: 'human', source: 'manual', creator_name: 'Joko' })
    expect(isHumanOnlyCreator(humanTodo)).toBe(true)
  })

  test('matchesCreatorFilter: memfilter berdasarkan tipe pembuat', () => {
    const execTodo = mockTodo({ source: 'autonomous_executive' })
    const auditTodo = mockTodo({ source: 'diagnostic_audit' })
    const pmTodo = mockTodo({ source: 'pm_agent' })
    const suggTodo = mockTodo({ is_ai_suggestion: true })
    const agentTodo = mockTodo({ creator_type: 'ai_agent', creator_name: 'claude' })
    const humanTodo = mockTodo({ creator_type: 'human', source: 'manual' })

    expect(matchesCreatorFilter(execTodo, 'executive')).toBe(true)
    expect(matchesCreatorFilter(auditTodo, 'audit')).toBe(true)
    expect(matchesCreatorFilter(pmTodo, 'pm')).toBe(true)
    expect(matchesCreatorFilter(suggTodo, 'ai_suggestion')).toBe(true)
    expect(matchesCreatorFilter(agentTodo, 'agent')).toBe(true)
    expect(matchesCreatorFilter(humanTodo, 'human')).toBe(true)
    expect(matchesCreatorFilter(agentTodo, 'ai')).toBe(true)
    expect(matchesCreatorFilter(humanTodo, 'ai')).toBe(false)
  })

  test('matchesEmployeeFilter: memfilter berdasarkan nama atau ID pegawai', () => {
    const tAssigned = mockTodo({ employee_id: 10, employee_name: 'Nayla Maharani' })
    const tUnassigned = mockTodo({ employee_id: undefined, employee_name: undefined })

    expect(matchesEmployeeFilter(tAssigned, 'all')).toBe(true)
    expect(matchesEmployeeFilter(tAssigned, 'Nayla')).toBe(true)
    expect(matchesEmployeeFilter(tAssigned, '10')).toBe(true)
    expect(matchesEmployeeFilter(tAssigned, 'Hendra')).toBe(false)
    expect(matchesEmployeeFilter(tUnassigned, 'unassigned')).toBe(true)
    expect(matchesEmployeeFilter(tAssigned, 'unassigned')).toBe(false)
  })

  test('matchesStatusFilter: mendukung status spesifik dan uncompleted', () => {
    const tProgress = mockTodo({ status: 'in_progress' })
    const tDone = mockTodo({ status: 'completed' })
    const tRejected = mockTodo({ status: 'rejected' })

    expect(matchesStatusFilter(tProgress, 'all')).toBe(true)
    expect(matchesStatusFilter(tProgress, 'uncompleted')).toBe(true)
    expect(matchesStatusFilter(tDone, 'uncompleted')).toBe(false)
    expect(matchesStatusFilter(tRejected, 'uncompleted')).toBe(false)
    expect(matchesStatusFilter(tProgress, 'in_progress')).toBe(true)
    expect(matchesStatusFilter(tProgress, 'completed')).toBe(false)
  })

  test('matchesSearchFilter: pencarian multi-field case-insensitive', () => {
    const t = mockTodo({
      title: 'Optimalisasi Algoritma',
      description: 'Menurunkan latensi response',
      guideline_ref: 'PERF-002',
      category: 'performance',
      target_apps: 'api-gateway'
    })

    expect(matchesSearchFilter(t, '')).toBe(true)
    expect(matchesSearchFilter(t, 'algoritma')).toBe(true)
    expect(matchesSearchFilter(t, 'latensi')).toBe(true)
    expect(matchesSearchFilter(t, 'perf-002')).toBe(true)
    expect(matchesSearchFilter(t, 'performance')).toBe(true)
    expect(matchesSearchFilter(t, 'gateway')).toBe(true)
    expect(matchesSearchFilter(t, 'tidak ada')).toBe(false)
  })

  test('matchesTodoFilter: menggabungkan semua kriteria filter', () => {
    const t = mockTodo({
      level: 'project',
      category: 'backend',
      status: 'in_progress',
      title: 'Audit Logging'
    })
    const filterState = {
      level: 'project',
      category: 'backend',
      creator: 'all',
      tag: 'all',
      status: 'in_progress',
      orderBy: 'priority_desc' as const,
      search: 'audit'
    }
    expect(matchesTodoFilter(t, filterState)).toBe(true)

    const nonMatchFilter = { ...filterState, category: 'frontend' }
    expect(matchesTodoFilter(t, nonMatchFilter)).toBe(false)
  })

  test('Todo Comparators: menguji perbandingan prioritas, tanggal, skor dampak, judul, dan status', () => {
    const tCrit = mockTodo({ priority: 'critical', created_at: '2026-11-01T10:00:00Z' })
    const tLow = mockTodo({ priority: 'low', created_at: '2026-11-02T10:00:00Z' })
    expect(comparePriorityDesc(tCrit, tLow)).toBeLessThan(0)
    expect(comparePriorityAsc(tCrit, tLow)).toBeGreaterThan(0)

    const tHighImpact = mockTodo({ impact_score: 95 })
    const tLowImpact = mockTodo({ impact_score: 20 })
    expect(compareImpactDesc(tHighImpact, tLowImpact)).toBeLessThan(0)

    const tOld = mockTodo({ created_at: '2026-10-01T00:00:00Z' })
    const tNew = mockTodo({ created_at: '2026-11-01T00:00:00Z' })
    expect(compareCreatedDesc(tOld, tNew)).toBeGreaterThan(0)
    expect(compareCreatedAsc(tOld, tNew)).toBeLessThan(0)

    const tAlpha = mockTodo({ title: 'Aplikasi' })
    const tBeta = mockTodo({ title: 'Backend' })
    expect(compareTitleAsc(tAlpha, tBeta)).toBeLessThan(0)

    const tInProg = mockTodo({ status: 'in_progress' })
    const tComp = mockTodo({ status: 'completed' })
    expect(compareStatus(tInProg, tComp)).toBeLessThan(0)
  })

  test('getTodoComparator: resolusi kunci composite dan fallback aman', () => {
    expect(typeof getTodoComparator('priority_desc')).toBe('function')
    expect(typeof getTodoComparator('priority', 'asc')).toBe('function')
    expect(typeof getTodoComparator('status')).toBe('function')
    expect(getTodoComparator('non_existent')(mockTodo(), mockTodo())).toBe(0)
  })

  test('filterAndSortTodos: pipeline filter dan sort bekerja secara terpadu', () => {
    const todos: Todo[] = [
      mockTodo({ id: 1, priority: 'low', status: 'completed', title: 'Task 1' }),
      mockTodo({ id: 2, priority: 'critical', status: 'in_progress', title: 'Task 2' }),
      mockTodo({ id: 3, priority: 'high', status: 'pending', title: 'Task 3' })
    ]

    const filterState = {
      level: 'all',
      category: 'all',
      creator: 'all',
      tag: 'all',
      status: 'uncompleted',
      orderBy: 'priority_desc' as const,
      search: ''
    }

    const result = filterAndSortTodos(todos, filterState)
    expect(result).toHaveLength(2)
    expect(result[0].id).toBe(2) // critical di atas
    expect(result[1].id).toBe(3) // high di bawah critical
  })
})

describe('todoRecommendedModelMeta helper refactor (CC reduction via lookup table)', () => {
  const mockTask = (overrides?: Partial<Todo>): Todo => ({
    id: 1,
    project_id: 1,
    title: 'Task Contoh',
    category: 'backend',
    priority: 'medium',
    status: 'pending',
    complexity_level: 'normal',
    created_at: new Date().toISOString(),
    ...overrides
  })

  test('mengembalikan null jika task completed atau rejected', () => {
    expect(todoRecommendedModelMeta(mockTask({ status: 'completed' }))).toBeNull()
    expect(todoRecommendedModelMeta(mockTask({ status: 'rejected' }))).toBeNull()
  })

  test('isFlagshipTask: mengenali task kompleks, arsitektur, security, atau critical', () => {
    expect(isFlagshipTask(mockTask({ complexity_level: 'complex' }))).toBe(true)
    expect(isFlagshipTask(mockTask({ priority: 'critical' }))).toBe(true)
    expect(isFlagshipTask(mockTask({ category: 'architecture' }))).toBe(true)
    expect(isFlagshipTask(mockTask({ category: 'security' }))).toBe(true)
    expect(isFlagshipTask(mockTask({ category: 'backend', priority: 'medium', complexity_level: 'normal' }))).toBe(false)
  })

  test('isFastTask: mengenali task quick, ui_ux, atau general', () => {
    expect(isFastTask(mockTask({ complexity_level: 'quick' }))).toBe(true)
    expect(isFastTask(mockTask({ category: 'ui_ux' }))).toBe(true)
    expect(isFastTask(mockTask({ category: 'general' }))).toBe(true)
    expect(isFastTask(mockTask({ category: 'backend', complexity_level: 'normal' }))).toBe(false)
  })

  test('todoRecommendedModelMeta: memilih saran flagship, fast, atau balanced secara deterministik', () => {
    const flagshipTodo = mockTask({ complexity_level: 'complex' })
    expect(todoRecommendedModelMeta(flagshipTodo)).toEqual(MODEL_SUGGESTIONS.flagship)

    const fastTodo = mockTask({ category: 'ui_ux' })
    expect(todoRecommendedModelMeta(fastTodo)).toEqual(MODEL_SUGGESTIONS.fast)

    const standardTodo = mockTask({ category: 'backend', complexity_level: 'normal', priority: 'medium' })
    expect(todoRecommendedModelMeta(standardTodo)).toEqual(MODEL_SUGGESTIONS.balanced)
  })
})

