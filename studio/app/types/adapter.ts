export interface AdapterMetadata {
  id: string
  name: string
  type: 'backend' | 'frontend' | string
  language: string
  framework: string
  icon: string
  description: string
  detect_rules: string[]
  scan_methods: string[]
  features: string[]
  example_file: string
  status: string
  latest_version?: string
  status_badge?: string
}

export interface AdaptersCatalogSummary {
  total: number
  backend: number
  frontend: number
  languages: string[]
  adapters: AdapterMetadata[]
}
