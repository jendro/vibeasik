// Uji keamanan proyek (GET/POST /api/projects/:id/security-test, MCP vibeasik_security_test).

export type SecurityCheckStatus = 'pass' | 'declared' | 'warn' | 'fail' | 'skipped'
export type SecurityCheckCategory = 'rate_limit' | 'headers' | 'cors' | 'injection' | 'input' | 'auth' | 'transport' | 'error_leaks'

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

export interface SecurityLiveResult {
  base_url: string
  reachable: boolean
  error?: string
  status_code?: number
  latency_ms?: number
  headers?: Record<string, string>
  paths?: string[]
  requests_run: number
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
}
