import type { SecurityAudit } from './ai'

export interface CensoredKeyword {
  id: number
  keyword: string
  category: string
  description: string
  is_regex: boolean
  is_active: boolean
  created_at?: string
  updated_at?: string
}

export interface CensoredKeywordsResponse {
  keywords: CensoredKeyword[]
  total: number
  active_count: number
  built_in_patterns: string[]
  double_security: boolean
}

export interface TestCensorResponse {
  original: string
  sanitized_text: string
  audit: SecurityAudit
}

/** Satu model dari katalog /models provider. */
export interface DiscoveredLLMModel {
  id: string
  display_name?: string
  owned_by?: string
}

export interface LLMConfig {
  id: number
  provider: 'system_one' | 'standard_llm'
  name: string
  url: string
  key?: string
  masked_key?: string
  has_key?: boolean
  models: string[]
  selected_model: string
  is_active: boolean
  /** Dukungan gambar efektif per model (override pengguna atau tebakan dari nama model). */
  vision?: Record<string, boolean>
  /** Model yang status vision-nya diatur manual oleh pengguna. */
  vision_manual?: string[]
  vision_tested?: string[]
  /** Alias tampilan per model (pemilih model & Kantor Agent); request tetap memakai nama model asli. */
  aliases?: Record<string, string>
  ping_status?: 'online' | 'offline' | 'pending'
  ping_latency_ms?: number
  created_at?: string
  updated_at?: string
}

export interface LLMSettings {
  id?: number
  active_provider: 'system_one' | 'standard_llm'
  system_one_url: string
  system_one_model: string
  system_one_key: string
  standard_llm_url: string
  standard_llm_key: string
  standard_llm_models: string[]
  standard_llm_selected: string
  updated_at?: string
}

export interface LLMTestResult {
  success: boolean
  latency_ms: number
  message: string
  sample_response?: string
}

/** Satu model di pool LLM (semua alamat & model di Pengaturan LLM) dengan tingkat kecerdasan & status provider. */
export interface LLMPoolModel {
  config_id: number
  config: string
  host: string
  model: string
  alias?: string
  score: number
  tier: 'frontier' | 'tinggi' | 'menengah' | 'ringan'
  vision: boolean
  active: boolean
  selected: boolean
  usable: boolean
  /** Alasan tidak dipakai: kuota habis / jeda rate limit / model tidak bisa */
  unusable?: string
  /** Kesehatan model dari log request: ok / slow / broken */
  health?: 'ok' | 'slow' | 'broken'
  avg_ms?: number
  requests?: number
  fails?: number
}

/** Status satu provider LLM (alamat + akun). Kuota habis = blocked sampai diaktifkan lagi. */
export interface LLMProviderState {
  id: number
  provider_key: string
  url: string
  config_id: number
  config_name: string
  blocked: boolean
  blocked_until?: string
  blocked_at?: string
  block_reason?: string
  cooldown_until?: string
  last_error?: string
  failures: number
  last_ok_at?: string
  updated_at: string
}

export interface LLMPoolResponse {
  models: LLMPoolModel[]
  providers: LLMProviderState[]
}

/** Satu request ke LLM luar (log per model). */
export interface LLMRequestLog {
  id: number
  created_at: string
  host: string
  config_name?: string
  model: string
  kind: 'chat' | 'vision' | 'vision_probe' | 'ping'
  purpose?: string
  ok: boolean
  err_kind?: 'quota' | 'rate_limit' | 'auth' | 'unavailable' | 'model' | 'other'
  error?: string
  latency_ms: number
  prompt_chars: number
  resp_chars: number
  images?: number
}

/** Ringkasan kesehatan satu model di satu provider. broken = tidak dipakai pool sampai broken_until. */
export interface LLMModelHealth {
  id: number
  model: string
  host: string
  config_id: number
  config_name: string
  requests: number
  oks: number
  fails: number
  consec_fails: number
  avg_ms: number
  last_ms: number
  status: 'ok' | 'slow' | 'broken'
  broken_until?: string
  last_err_kind?: string
  last_error?: string
  last_ok_at?: string
  last_fail_at?: string
  vision_checked_at?: string
  vision_note?: string
  updated_at: string
}

export type LLMFailedStatus = 'open' | 'running' | 'answered' | 'used' | 'resolved' | 'dismissed'

/** Request LLM yang gagal berulang dan menunggu dieksekusi dari lokal. */
export interface LLMFailedRequest {
  id: number
  created_at: string
  updated_at: string
  status: LLMFailedStatus
  notified: boolean
  kind: string
  purpose?: string
  task?: string
  project_name?: string
  attempts: number
  err_kind?: string
  last_error?: string
  system_prompt?: string
  user_prompt?: string
  prompt_chars: number
  answer?: string
  answered_by?: string
  answered_at?: string
  used_at?: string
  run_error?: string
  models_tried?: string[]
  prompt_preview?: string
}

export interface NotificationsResponse {
  failed: LLMFailedRequest[]
  counts: Record<string, number>
  broken_models: LLMModelHealth[] | null
}

export interface LLMFailedDetail {
  request: LLMFailedRequest
  models_tried: string[] | null
  local_prompt: string
}
