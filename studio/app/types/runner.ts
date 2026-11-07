// Dev runner & port registry: vibeasik mengalokasikan port unik per komponen lalu menjalankan dev command-nya.

export type RunState = 'idle' | 'starting' | 'running' | 'stopped' | 'exited'

export interface RunStatus {
  component_id: number
  project_id: number
  name: string
  state: RunState
  pid?: number
  port: number
  url?: string
  command?: string
  notes?: string[]
  warning?: string
  listening: boolean
  started_at?: string
  exited_at?: string
  exit_code?: number
  logs?: string[]
}

export interface PortRegistryEntry {
  component_id: number
  project_id: number
  project_name: string
  component_name: string
  role: string
  declared_port: number
  run_port: number
  state: RunState
  listening: boolean
  has_dev_command: boolean
}

export interface PortRegistryResponse {
  entries: PortRegistryEntry[]
  range: { start: number, end: number }
}

export interface ProjectRunItem {
  component_id: number
  name: string
  status?: RunStatus
  error?: string
  skipped?: string
}
