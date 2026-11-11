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
  }
  health: { date: string, score: number }[]
  health_score: number
  highlights: RecapTodoBrief[]
}

export type CostClass = 'constant' | 'log' | 'linear' | 'linearithmic' | 'quadratic' | 'cubic' | 'polynomial'

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
}

export interface RouteCost extends ProcessCost {
  module: string
  method: string
  path: string
  handler: string
  handler_decl?: string
  auth: boolean
  middlewares: number
}

export interface FuncCost extends ProcessCost {
  module: string
  name: string
  file: string
  line: number
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
}
