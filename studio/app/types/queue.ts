export type JobType
  = | 'fast_scan'
    | 'project_summary'
    | 'detect_routes'
    | 'diagnose'
    | 'project_profile'
    | 'enrich_routes'
    | 'synergy_map'
    | 'llm_audit'
    | 'audit_sweep'
    | 'code_map_enrich'
    | 'pm_plan'
    | 'design_analyze'
    | 'design_measure'
    | 'design_lesson'
    | 'staff_review'
    | 'staff_chat'
    | 'staff_agenda'
    | 'staff_social'
    | 'staff_hr'

/** Job types that can be started on demand via POST /api/queue/enqueue. */
export type EnqueueJobType = Exclude<JobType, 'diagnose' | 'staff_review' | 'staff_chat' | 'staff_agenda' | 'staff_social' | 'staff_hr'>

export interface EnqueueJobRequest {
  type: EnqueueJobType
  project_id?: number
  component_id?: number
  /** Target di luar proyek, mis. id referensi Galeri Desain (design_analyze). */
  target_id?: number
}

export interface EnqueueJobResponse {
  job: QueueJob
  already_queued: boolean
}

export type JobStatus = 'queued' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface QueueJob {
  id: string
  type: JobType
  title: string
  project_id: number
  component_id?: number
  target_id?: number
  project_name: string
  component_name?: string
  is_llm: boolean
  /** Jalur sumber daya; job dengan lane bentrok mengantre, lane lain berjalan paralel. */
  lane?: string
  /** Judul job yang menahan job antre ini, atau "slot antrean penuh". */
  waiting_for?: string
  status: JobStatus
  progress: number
  message: string
  error?: string
  created_at: string
  started_at?: string
  completed_at?: string
}

export interface QueueStats {
  running_count: number
  queued_count: number
  completed_count: number
  failed_count: number
  max_concurrent_llm: number
  max_concurrent_total: number
}

export interface QueueResponse {
  jobs: QueueJob[]
  stats: QueueStats
}
