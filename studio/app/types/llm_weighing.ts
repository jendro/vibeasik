/** Penimbang permintaan LLM oleh System One / Jev TypeSafe (GET /api/settings/llm/weighing). */
export interface LLMWeighing {
  id: number
  at: string
  purpose: string
  requested: string
  jev_tier: string
  tier: string
  probs?: Record<string, number>
  confidence: number
  risk: number
  reason: string
  source: string
  jev_model?: string
  jev_tokens: number
  ms: number
  prompt_chars: number
  model?: string
  error?: string
}

export interface LLMWeighingReport {
  enabled: boolean
  total: number
  by_tier: Record<string, number>
  by_source: Record<string, number>
  fallback: number
  avg_ms: number
  jev_tokens: number
  by_purpose: Record<string, number>
  recent: LLMWeighing[] | null
  system_one: { name: string, url: string, model: string, active: boolean, paused: boolean }[] | null
}
