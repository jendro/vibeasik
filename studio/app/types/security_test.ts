// Uji keamanan proyek (GET/POST /api/projects/:id/security-test, MCP vibeasik_security_test).

export type SecurityCheckStatus = 'pass' | 'declared' | 'warn' | 'fail' | 'skipped'
export type SecurityCheckCategory = 'rate_limit' | 'headers' | 'cors' | 'injection' | 'input' | 'auth' | 'transport' | 'error_leaks' | 'xss' | 'traversal'

export interface SecurityCheckLocation {
  file: string
  line: number
  snippet?: string
}

export interface SecurityCheck {
  id: string
  category: SecurityCheckCategory
  title: string
  status: SecurityCheckStatus
  severity: 'CRITICAL' | 'WARNING' | 'INFO'
  source: 'static' | 'live'
  evidence?: string[]
  suggestion?: string
  locations?: SecurityCheckLocation[]
  guideline_id?: string
}

export type SecurityAttackVerdict = 'vulnerable' | 'suspect' | 'safe' | 'blocked' | 'error'

/** Satu serangan yang dikirim ke server (SQLi, XSS, traversal, JWT palsu, …) beserta vonisnya. */
export interface SecurityAttack {
  category: string
  route: string
  method: string
  path: string
  payload: string
  status?: number
  latency_ms: number
  verdict: SecurityAttackVerdict
  note?: string
}

export interface SecurityLiveResult {
  base_url: string
  reachable: boolean
  error?: string
  status_code?: number
  latency_ms?: number
  headers?: Record<string, string>
  paths?: string[]
  requests_run: number
  attacks?: SecurityAttack[]
}

export interface SecurityTestReport {
  project_id: number
  component_id?: number
  target: string
  score: number
  grade: 'A' | 'B' | 'C' | 'D' | 'E'
  summary: Record<SecurityCheckStatus, number>
  checks: SecurityCheck[]
  live?: SecurityLiveResult
  files_read: number
  routes_seen: number
  started_at: string
  duration_ms: number
  notes?: string[]
}

export interface SecurityTestRequest {
  component_id?: number
  base_url?: string
  live?: boolean
  skip_burst?: boolean
  burst?: number
  /** nyalakan server komponen bila belum berjalan (lalu dimatikan lagi setelah uji) */
  start_server?: boolean
  keep_running?: boolean
  /** lewati serangan SQLi/XSS/traversal/… ke semua endpoint */
  skip_attacks?: boolean
  /** "Bearer …" atau "Cookie: …" akun uji agar endpoint berlogin ikut diserang (tidak disimpan) */
  auth_header?: string
}

// Sesi uji di latar (POST …/security-test/start, dipantau lewat GET …/security-test).
export type SecurityTestStepStatus = 'pending' | 'running' | 'done' | 'failed' | 'skipped'

export interface SecurityTestStep {
  key: 'server' | 'static' | 'live' | 'attack' | 'burst' | 'stop'
  title: string
  status: SecurityTestStepStatus
  detail?: string
  started_at?: string
  finished_at?: string
  duration_ms?: number
}

export interface SecurityTestEvent {
  at: string
  level: 'info' | 'ok' | 'warn' | 'error'
  message: string
}

/** Server yang diuji; bentuknya sejalan dengan komponen Uji Runtime agar terminal log bisa dipakai bersama. */
export interface SecurityTestServer {
  component_id: number
  name: string
  role: string
  state: 'starting' | 'ready' | 'stopped' | 'failed' | 'external' | string
  source: 'launched' | 'reused' | 'external' | ''
  command?: string
  port?: number
  url?: string
  pid?: number
  boot_ms?: number
  kept_running?: boolean
  error_lines: number
  routes_found: number
  note?: string
  logs?: string[]
}

export interface SecurityTestSession {
  id: string
  project_id: number
  project_name: string
  status: 'running' | 'completed' | 'failed' | 'cancelled'
  steps: SecurityTestStep[]
  events: SecurityTestEvent[]
  server?: SecurityTestServer
  report?: SecurityTestReport
  summary?: string
  error?: string
  started_at: string
  finished_at?: string
}
