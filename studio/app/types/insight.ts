// Rekap pemilik & biaya proses (tick / Big-O) — cermin pkg/insight dan pkg/routemap (cost.go).

export interface RecapDayPoint {
  date: string
  created: number
  completed: number
  bugs_closed: number
  findings_resolved: number
  open: number
}

export interface RecapTodoBrief {
  id: number
  title: string
  category: string
  priority: string
  completed_at?: string
  score?: number
  /** commit hasil todo (GET /api/todos/:id/changes) */
  commit_sha?: string
}

export interface RecapWorkStat {
  done: number
  open: number
  done_in_period: number
  latest: RecapTodoBrief[]
}

export interface OwnerRecap {
  project_id: number
  days: number
  generated_at: string
  todos: {
    total: number
    done: number
    open: number
    in_progress: number
    suggested: number
    percent: number
    done_in_period: number
    created_in_period: number
    avg_score: number
    scored_count: number
    open_high_priority: number
    open_from_audit: number
  }
  timeline: RecapDayPoint[]
  categories: { category: string, done: number, open: number }[]
  bugs: RecapWorkStat
  algorithm: RecapWorkStat
  findings: {
    open: number
    resolved: number
    ignored: number
    open_critical: number
    resolved_in_period: number
    open_by_severity: Record<string, number>
    /** temuan terbuka per area perbaikan (speed, data, ui, security, code, ops) */
    open_by_area: Record<string, number>
    last_audit_at?: string
  }
  health: { date: string, score: number }[]
  health_score: number
  highlights: RecapTodoBrief[]
}

export type CostClass = 'constant' | 'log' | 'linear' | 'linearithmic' | 'quadratic' | 'cubic' | 'polynomial'

/** Satu baris kode yang menjadi dasar rumus biaya (bukti): loop atas data, query, atau pemanggilan berbiaya. */
export interface CostEvidence {
  kind: 'loop' | 'query' | 'call'
  line: number
  code: string
  /** jumlah loop atas data yang melingkupi baris ini (untuk loop: termasuk dirinya) */
  depth: number
  note: string
  /** fungsi lokal terpanggil yang membawa biaya baris ini (klik untuk membukanya di daftar Fungsi) */
  callee?: CostCallee
}

export interface CostCallee {
  name: string
  file: string
  line: number
}

/** Potongan berkas untuk tampilan editor bukti biaya. */
export interface CostSourceWindow {
  file: string
  from: number
  lines: string[]
  total: number
}

export interface CostTodoBatch {
  created: number
  merged: number
  files: number
  todos: { id: number, title: string, status: string }[]
}

export interface FindingScope {
  key: 'audit' | 'cost' | 'runtime'
  label: string
  page: string
  rule_prefixes?: string[]
}

/** Prompt untuk agent coding pengguna: menjadikan temuan todo lewat MCP vibeasik (sama dengan tool vibeasik_findings_todo_prompt). */
export interface FindingsTodoPrompt {
  project_id: number
  project_name: string
  scope: FindingScope
  open: number
  critical: number
  /** temuan perkiraan / keadaan sesaat yang tidak ikut triase agent */
  advisory: number
  prompt: string
}

export interface ProcessCost {
  formula: string
  big_o: string
  class: CostClass
  degree: number
  ticks_1: number
  query_formula: string
  query_big_o: string
  queries_1: number
  n_plus_one: boolean
  recursive?: boolean
  unbounded?: boolean
  loop_depth: number
  growth: { n: number, ticks: number, queries: number }[]
  /** baris kode yang menaikkan biaya, di berkas evidence_file; kosong untuk biaya tetap tanpa query */
  evidence?: CostEvidence[]
  evidence_file?: string
}

export type CostLang = 'go' | 'php'

/** Sumber biaya tinggi: fungsi/halaman tempat loop N+1 atau O(n²) ke atas ditulis (bisa dijadikan todo). */
export interface CostSource {
  kind: 'nplus1' | 'growth'
  rule: string
  func: string
  file: string
  line: number
  big_o: string
  query_formula?: string
  /** baris kode di fungsi sumber yang menaikkan biaya */
  evidence?: CostEvidence[]
}

/** todo = todo terbuka · audit = temuan audit terbuka · ignored = bukan masalah · recheck = sudah selesai tapi masih tinggi */
export type CostHandledWhere = 'todo' | 'audit' | 'ignored' | 'recheck'

export interface CostHandling {
  where: CostHandledWhere
  todo_id?: number
  todo_title?: string
  todo_status?: string
  rule: string
  source: CostSource
  /** langkah berikutnya, kalimat siap tampil */
  next: string
}

/** Penanda penanganan pada baris tabel biaya. */
export interface CostRowHandling {
  /** sumber biaya baris ini yang belum ditangani */
  source?: CostSource
  handled?: CostHandling
}

export interface CostHandlingSummary {
  sources: number
  in_todo: number
  todo_count: number
  in_audit: number
  ignored: number
  recheck: number
  open: number
  todos: { id: number, title: string, status: string }[]
  audit_at?: string
}

export interface RouteCost extends ProcessCost, CostRowHandling {
  module: string
  lang: CostLang
  /** fiber, gin, echo … (Go) atau Laravel, Lumen, CodeIgniter 4, Slim, PHP */
  framework?: string
  method: string
  path: string
  handler: string
  handler_decl?: string
  auth: boolean
  middlewares: number
}

export interface FuncCost extends ProcessCost, CostRowHandling {
  module: string
  lang: CostLang
  name: string
  file: string
  line: number
  /** baris penutup badan fungsi */
  end?: number
  routes: number
}

export interface ProcessCostSnapshot {
  snap_date: string
  routes: number
  funcs: number
  constant: number
  linear: number
  linearithmic: number
  quadratic: number
  worse: number
  n_plus_one: number
  ticks_1: number
}

export interface ProcessCostSummary {
  supported: boolean
  modules: string[]
  /** mis. ['Go (fiber)', 'Laravel'] */
  frameworks: string[]
  routes: number
  funcs: number
  route_classes: Partial<Record<CostClass, number>>
  func_classes: Partial<Record<CostClass, number>>
  n_plus_one_routes: number
  n_plus_one_funcs: number
  recursive: number
  unbounded: number
  route_ticks_1: number
  worst_route?: RouteCost
}

export interface ProcessCostPage {
  summary: ProcessCostSummary
  kind: 'routes' | 'funcs'
  total: number
  limit: number
  offset: number
  routes: RouteCost[]
  funcs: FuncCost[]
  history: ProcessCostSnapshot[]
  handling?: CostHandlingSummary
  /** kapan rumus dihitung dari kode di disk (setiap muat = hitung ulang) dan lamanya */
  computed_at?: string
  duration_ms?: number
}
