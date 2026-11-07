export interface VisualCaptureRequest {
  project_id: number
  component_id?: number
  port?: number
  route?: string
  width?: number
  height?: number
  custom_url?: string
}

export interface VisualCaptureResult {
  project_id: number
  component_id: number
  component_name: string
  target_url: string
  port: number
  route: string
  captured_at: string
  image_path: string
  base64_image?: string
  width: number
  height: number
  size_kb: number
}

export interface VisualIssue {
  category: 'data' | 'visual' | 'layout' | 'accessibility'
  severity: 'error' | 'warning' | 'info'
  title: string
  description: string
  target_area?: string
}

export interface VisualAuditRequest {
  project_id: number
  component_id?: number
  port?: number
  route?: string
  custom_url?: string
  base64_image?: string
  prompt?: string
}

export interface VisualAuditResponse {
  project_id: number
  component_id: number
  component_name: string
  route: string
  target_url: string
  visual_score: number
  data_hygiene_score: number
  summary: string
  data_issues: VisualIssue[]
  visual_issues: VisualIssue[]
  suggested_fixes: string[]
  ai_agent_prompt: string
  screenshot_path: string
  audited_at: string
}

export interface ProjectSnapshotFile {
  name: string
  route: string
  url: string
  size_kb: number
  modified_at: string
}
