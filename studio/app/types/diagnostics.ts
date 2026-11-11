export interface Finding {
  id?: string
  category: string
  severity: 'CRITICAL' | 'WARNING' | 'INFO' | 'OPTIMIZATION' | string
  title: string
  description: string
  suggestion: string
  target_file?: string
  line?: number
  snippet?: string
  rule_id?: string
  theory_basis?: string
  suggested_fix?: string
  status?: 'open' | 'resolved' | 'ignored' | string
  is_deep?: boolean
  // true jika temuan berasal dari job antrean review audit LLM (rule_id LLM-REVIEW-*)
  is_llm?: boolean
  // Vonis pembelajaran audit: false_positive = audit keliru (kode sudah benar), confirmed = temuan benar
  verdict?: AuditVerdict | ''
  verdict_reason?: string
  // Saved LLM resolution & action steps
  resolution_title?: string
  explanation?: string
  action_steps?: string[]
  code_before?: string
  code_after?: string
  target_lines?: string
  // Rencana penempatan logika sesuai peta layer proyek (cegah fungsi kembar)
  placement_plan?: PlacementStep[]
  todo_id?: number
  model_used?: string
  resolved_at?: string
}

export interface PlacementStep {
  symbol: string
  action: 'pindah' | 'pakai_ulang' | 'satukan' | 'tetap' | 'baru' | string
  target: string
  layer: string
  reuse?: string
  reason?: string
  // hasil cek vibeasik terhadap peta kode (folder baru, simbol tidak ada di berkas)
  warning?: string
}

export interface Optimization {
  title: string
  description: string
  category: string
  impact: 'HIGH' | 'MEDIUM' | 'LOW' | string
  action_steps: string[]
  prompt_for_ai: string
}

export interface DiagnosticReportResponse {
  diagnostic: any
  findings: Finding[]
  optimizations: Optimization[]
  category_counts: Record<string, number>
  severity_counts: Record<string, number>
  total_findings: number
  score: number
  summary: string
}

export interface ResolutionResponse {
  success: boolean
  rule_id: string
  title: string
  theory_basis: string
  target_file: string
  target_lines: string
  code_before: string
  code_after: string
  explanation: string
  action_steps: string[]
  placement_plan?: PlacementStep[]
  created_todo?: any
  todo_outcome?: 'created' | 'merged' | 'represented' | 'rejected' | string // merged = digabung ke todo audit terbuka, represented = sudah terwakili, rejected = sejenis todo yang pernah ditolak
  findings?: number
  model_used: string
  generated_via: 'external_llm' | 'heuristic_engine' | string
  score?: number
}

export interface SecurityStatus {
  global_sanitizer_active: boolean
  double_security_active?: boolean
  censoring_mode: string
  intercepted_providers: string[]
  protected_patterns: string[]
  active_custom_keywords?: number
  leak_prevention: string
}

export interface RouteSimulation {
  method: string
  path: string
  handler: string
  component: string
  simulated_latency_ms: number
  real_latency_ms?: number
  latency_status: 'FAST' | 'NORMAL' | 'SLOW' | 'CRITICAL' | string
  has_form_validation: boolean
  has_rate_limiter: boolean
  query_vulnerability_risk: 'LOW' | 'MEDIUM' | 'HIGH' | string
  details?: string
}

export interface DeepCheckReport {
  id: number
  project_id: number
  score: number
  summary: string
  findings_json: string
  route_simulations_json: string
  git_commit_hash: string
  project_checksum: string
  total_routes_checked: number
  total_functions_checked: number
  total_findings: number
  created_at: string
}

export interface DeepCheckLatestResponse {
  has_report: boolean
  report?: DeepCheckReport
  findings?: Finding[]
  route_simulations?: RouteSimulation[]
  score?: number
  summary?: string
  last_deep_check_at?: string | null
  has_changes_since_last_deep_check: boolean
  change_summary?: string
  total_routes_checked?: number
  total_functions_checked?: number
  total_findings?: number
  message?: string
}

// Simulasi Runtime: aplikasi dijalankan sungguhan di laptop lalu diuji dengan request nyata
export type RuntimeSimStatus = 'running' | 'completed' | 'failed' | 'cancelled'
export type RuntimeSimStepStatus = 'pending' | 'running' | 'done' | 'failed' | 'skipped'
export type RuntimeSimComponentState = 'pending' | 'booting' | 'running' | 'crashed' | 'timeout' | 'failed' | 'skipped' | 'stopped'
export type RuntimeSimVerdict = 'OK' | 'SLOW' | 'REDIRECT' | 'PROTECTED' | 'NOT_FOUND' | 'CLIENT_ERROR' | 'VALIDATION' | 'STREAM' | 'RATE_LIMITED' | 'SERVER_ERROR' | 'FAILED'
export type RuntimeSimViewport = 'mobile' | 'tablet' | 'desktop'
export type RuntimeSimPageVerdict = 'OK' | 'SLOW' | 'REDIRECT_LOGIN' | 'NOT_FOUND' | 'BLANK' | 'ERROR' | 'FAILED'

export interface RuntimeSimAuth {
  identifier: string
  password?: string
  identifier_field?: string
  password_field?: string
  login_path?: string
  login_body?: string
  login_page_path?: string
  has_password?: boolean
}

export interface RuntimeSimOptions {
  keep_running?: boolean
  // Komponen yang tidak dicentang; kosong = semua yang bisa dijalankan.
  skip_component_ids?: number[]
  // Override opsional (UI tidak mengisinya): kosong/0 = ditentukan otomatis oleh engine.
  max_routes?: number
  boot_timeout_sec?: number
  param_value?: string
  test_mutations?: boolean
  test_delete?: boolean
  skip_patterns?: string[]
  auth?: RuntimeSimAuth | null
  skip_pages?: boolean
  max_pages?: number
  viewports?: RuntimeSimViewport[]
}

export interface RuntimeSimPortConflict {
  port: number
  kind: 'busy' | 'duplicate' | 'reserved'
  message: string
  suggested_port?: number
}

export interface RuntimeSimPlanComponent {
  component_id: number
  name: string
  role: string
  dev_command?: string
  runnable: boolean
  reason?: string
  running: boolean
  port?: number
  endpoints: number
  pages: number
  dynamic: number
  needed_by?: string[]
  conflict?: RuntimeSimPortConflict
}

export interface RuntimeSimPlan {
  project_id: number
  components: RuntimeSimPlanComponent[]
  conflicts: number
  auto: string[]
}

export interface RuntimeTestProfileResponse {
  options: RuntimeSimOptions | null
  // Akun uji terakhir (tanpa password) — tetap tersimpan walau login dimatikan.
  saved_auth?: RuntimeSimAuth | null
  default_skip_patterns: string[]
}

export interface RuntimeSimLogin {
  kind: 'api' | 'browser'
  component_id: number
  component: string
  target: string
  ok: boolean
  status_code?: number
  method?: string
  message: string
  duration_ms?: number
  screenshot?: string
}

export interface RuntimeSimShot {
  viewport: RuntimeSimViewport
  width: number
  height: number
  file: string
  load_ms: number
  scroll_width: number
  page_height: number
  overflow_x: boolean
  offenders?: string[]
  small_tap_targets?: number
  small_text?: number
}

export interface RuntimeSimPage {
  component_id: number
  component: string
  path: string
  source_file?: string
  url: string
  auth: 'guest' | 'auth'
  final_url?: string
  title?: string
  status_code?: number
  ttfb_ms: number
  fcp_ms: number
  dom_ready_ms: number
  load_ms: number
  cold_load_ms: number
  requests: number
  transfer_kb: number
  failed_requests?: string[]
  console_errors?: string[]
  shots: RuntimeSimShot[]
  verdict: RuntimeSimPageVerdict
  note?: string
  error?: string
}

export interface RuntimeSimStep {
  key: string
  title: string
  status: RuntimeSimStepStatus
  detail?: string
  started_at?: string
  finished_at?: string
  duration_ms?: number
}

export interface RuntimeSimComponent {
  component_id: number
  name: string
  role: string
  state: RuntimeSimComponentState
  source?: 'launched' | 'reused' | 'external'
  command?: string
  port?: number
  url?: string
  pid?: number
  boot_ms?: number
  memory_mb?: number
  routes_found: number
  error_lines: number
  note?: string
  logs?: string[]
  screenshot?: string
}

export interface RuntimeSimProbe {
  component_id: number
  component: string
  method: string
  path: string
  route?: string
  handler?: string
  auth?: 'guest' | 'auth'
  url: string
  status_code?: number
  cold_ms: number
  warm_ms: number
  samples: number
  size_bytes?: number
  content_type?: string
  verdict: RuntimeSimVerdict
  error?: string
  at: string
}

export interface RuntimeSimEvent {
  at: string
  level: 'info' | 'ok' | 'warn' | 'error'
  message: string
}

export interface RuntimeSimSession {
  id: string
  project_id: number
  project_name: string
  status: RuntimeSimStatus
  options: RuntimeSimOptions
  started_at: string
  finished_at?: string
  duration_ms?: number
  score: number
  summary?: string
  steps: RuntimeSimStep[]
  components: RuntimeSimComponent[]
  probes: RuntimeSimProbe[]
  logins?: RuntimeSimLogin[]
  pages?: RuntimeSimPage[]
  events: RuntimeSimEvent[]
  findings: Finding[]
  report_id?: number
}

export interface RuntimeSimResponse {
  active: boolean
  session: RuntimeSimSession | null
}

export type AuditVerdict = 'false_positive' | 'confirmed'

// Satu vonis pembelajaran audit (tabel audit_feedbacks)
export interface AuditFeedback {
  id: number
  project_id: number
  finding_key: string
  rule_id: string
  category: string
  severity: string
  title: string
  description: string
  target_file: string
  line: number
  snippet: string
  is_llm: boolean
  verdict: AuditVerdict
  reason: string
  rule_advice?: string
  fix_code?: string
  source: 'manual' | 'llm' | 'agent' | string
  model?: string
  hit_count: number
  rule_updated: boolean
  rule_updated_at?: string
  created_at: string
  updated_at: string
}

export interface AuditRuleStat {
  rule_id: string
  is_llm: boolean
  false_positive: number
  confirmed: number
  pending_update: number
  suppressed: number
  precision: number
  needs_update: boolean
  reasons?: string[]
}

export interface AuditLearningResponse {
  feedback: AuditFeedback[]
  rules: AuditRuleStat[]
  totals: { false_positive: number, confirmed: number, suppressed: number, rules_need_update: number }
}

// Hasil "Cek ulang via LLM" — hanya saran, manusia yang memutuskan
export interface FindingVerification {
  verdict: AuditVerdict | 'uncertain'
  confidence: number
  reason: string
  rule_advice?: string
  model: string
}
