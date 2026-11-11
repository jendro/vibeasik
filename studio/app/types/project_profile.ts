export type ProfileFeatureStatus = 'done' | 'partial' | 'planned'
export type ProfileHorizon = 'now' | 'next' | 'later'
export type ProfileSeverity = 'high' | 'medium' | 'low'
export type ProfileMaturityStage = 'idea' | 'prototype' | 'mvp' | 'beta' | 'production'

export interface ProfileRouteRef {
  method?: string
  path: string
  purpose: string
}

export interface ProfileApp {
  component_id: number
  component_name: string
  role: string
  summary: string
  responsibilities: string[]
  key_pages: ProfileRouteRef[]
  key_endpoints: ProfileRouteRef[]
  talks_to: string[]
}

export interface ProfileFeature {
  name: string
  domain: string
  description: string
  status: ProfileFeatureStatus
  apps: string[]
  pages: string[]
  endpoints: string[]
  tables: string[]
  evidence: string[]
}

export interface ProfileIntegration {
  from: string
  to: string
  via: string
  purpose: string
}

export interface ProfileRoadmapItem {
  title: string
  horizon: ProfileHorizon
  rationale: string
  related_features: string[]
}

export type ProfileRiskStatus = 'open' | 'planned' | 'in_progress' | 'handled' | 'accepted'

export interface ProfileRisk {
  title: string
  severity: ProfileSeverity
  detail: string
  // Penanganan: diisi pemilik atau mengikuti status todo yang ditautkan.
  status?: ProfileRiskStatus
  todo_id?: number
  note?: string
  handled_at?: string
}

export interface ProfileStats {
  apps: number
  live_apps: number
  pages: number
  endpoints: number
  tables: number
  domains: number
  open_todos: number
  done_todos: number
  total_files: number
}

export interface ProjectProfile {
  project_id: number
  short_description: string
  full_description: string
  purpose: {
    problem: string
    solution: string
    target_users: string[]
    value_proposition: string
  }
  maturity: { stage: ProfileMaturityStage, reason: string }
  apps: ProfileApp[]
  features: ProfileFeature[]
  integrations: ProfileIntegration[]
  data_entities: { name: string, description: string, used_by: string[] }[]
  external_services: { name: string, purpose: string }[]
  roadmap: ProfileRoadmapItem[]
  risks: ProfileRisk[]
  open_questions: string[]
  confidence: number
  stats: ProfileStats
  // llm = vibeasik's own LLM, agent = written by an external AI agent via MCP / pasted JSON
  source: 'llm' | 'heuristic' | 'agent'
  model?: string
  llm_error?: string
  generated_at: string
}

export interface ProjectProfilePrompt {
  system_prompt: string
  user_prompt: string
  agent_prompt: string
  evidence_size: number
}
