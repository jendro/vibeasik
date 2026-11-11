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
  /** Alasan tidak dipakai: kuota habis / jeda rate limit */
  unusable?: string
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
