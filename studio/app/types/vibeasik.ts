export interface Tutorial {
  id: number
  slug: string
  title: string
  category: 'dasar-pemrograman' | 'database' | 'security' | 'deployment' | 'ai-agent' | string
  difficulty: 'pemula' | 'menengah' | 'lanjutan'
  estimated_minutes: number
  summary: string
  content_markdown: string
  code_snippets: string
  tags: string
  source_url?: string
  created_at: string
}

export interface DomainItem {
  domain: string
  tld: string
  available: boolean
  price_annual: number
  currency: string
  auto_dns_ready: boolean
}

export interface DomainSearchResponse {
  query: string
  suggestions: DomainItem[]
}

export interface DeployFile {
  filename: string
  content: string
  purpose: string
}

export interface DeployPlanRequest {
  project_name: string
  language: string
  framework: string
  port: number
  domain: string
  target_env: string
  use_database: boolean
  db_type: string
}

export interface DeployPlanResponse {
  target_platform: string
  architecture: string
  generated_files: DeployFile[]
  deploy_commands: string[]
  env_template: string
  notes: string
}

export interface SecurityIssue {
  rule_id: string
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW'
  category: string
  title: string
  file: string
  line: number
  description: string
  remediation: string
}

export interface CloudAuditResponse {
  safety_score: number
  status: 'PASSED' | 'WARNING' | 'FAILED'
  issues: SecurityIssue[]
  passed_rules: string[]
  summary: string
}

export interface AITemplate {
  id: string
  name: string
  type: string
  target_language: string
  target_framework: string
  description: string
  benefits: string[]
  files: DeployFile[]
  dependencies: string[]
  install_guide: string
}
