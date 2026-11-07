export interface AgentStatusDTO {
  key: string
  name: string
  command: string
  detected: boolean
  path: string
  version: string
  description: string
  capabilities: string[]
  is_builtin: boolean
}

export interface AgentExecutionRequest {
  agent_key: string
  prompt: string
  project_path: string
  timeout_sec?: number
}

export interface AgentExecutionResult {
  agent_key: string
  success: boolean
  output: string
  exit_code: number
  duration: string
  message: string
}

export interface CustomAgentForm {
  key: string
  name: string
  command: string
  args_template: string
  description: string
  icon?: string
}
