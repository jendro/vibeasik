export interface ReaStatus {
  installed: boolean
  path: string
  version: string
  capabilities: string[]
  install_cmd: string
  setup_cmd: string
  description: string
  message?: string
}

export interface ReaAnalysisRequest {
  target_path: string
  analysis_type?: 'javascript' | 'native' | 'android' | 'sqlite' | 'website'
}

export interface ReaAnalysisResult {
  success: boolean
  output: string
  analysis_type: string
  target_path: string
  duration?: string
}
