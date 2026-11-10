// Traffic Analyzer (Aturan 48): statistik request aplikasi live → saran rate limit, cache, load balancer, pecah service.
export type TrafficMode = 'cloud' | 'direct'
export type TrafficWindow = '15m' | '1h' | '6h' | '24h' | '7d' | '30d'
export type TrafficSeverity = 'critical' | 'warning' | 'info'

export interface TrafficSource {
  id: number
  project_id?: number
  component_id?: number
  project_name?: string
  component_name?: string
  name: string
  mode: TrafficMode
  cloud_url: string
  cloud_uid?: string
  sync_interval_sec: number
  enabled: boolean
  pending: number
  dropped: number
  total_events: number
  last_sync_at: string | null
  last_event_at: string | null
  last_ingest_at: string | null
  last_instance: string
  last_error: string
  created_at: string
}

export interface TrafficMeta {
  sync_intervals: number[]
  windows: TrafficWindow[]
  default_cloud_url: string
}

export interface TrafficTotals {
  requests: number
  err_4xx: number
  err_5xx: number
  err_rate: number
  client_err_rate: number
  rps_avg: number
  rpm_peak: number
  peak_at: number
  concurrency_avg: number
  concurrency_peak: number
  avg_ms: number
  p50: number
  p95: number
  p99: number
  max_ms: number
  avg_out: number
  avg_in: number
  unique_ips: number
  instances: number
  read_share: number
  span_minutes: number
}

export interface TrafficPoint {
  t: number
  requests: number
  err_4xx: number
  err_5xx: number
  avg_ms: number
  p95: number
}

export interface TrafficRouteRow {
  method: string
  route: string
  group: string
  count: number
  share: number
  time_share: number
  rpm: number
  err_4xx: number
  err_5xx: number
  err_rate: number
  avg_ms: number
  p50: number
  p95: number
  p99: number
  max_ms: number
  avg_out: number
  avg_in: number
  hot_share: number
}

export interface TrafficGroupRow {
  group: string
  routes: number
  count: number
  share: number
  time_share: number
  p95: number
  err_rate: number
  avg_ms: number
}

export interface TrafficKV {
  value: string
  count: number
  err: number
  share: number
}

export interface TrafficIPRow {
  ip: string
  count: number
  share: number
  peak_per_min: number
  err_rate: number
  auth_hits: number
  auth_peak_per_min: number
}

export interface TrafficRecommendation {
  id: string
  rule: string
  category: string
  severity: TrafficSeverity
  title: string
  detail: string
  action: string
  evidence: string[]
  routes?: string[]
}

export interface TrafficReport {
  source: TrafficSource
  window: TrafficWindow
  gran: 'm' | 'h'
  from: string
  to: string
  step_sec: number
  totals: TrafficTotals
  series: TrafficPoint[]
  routes: TrafficRouteRow[]
  groups: TrafficGroupRow[]
  status: TrafficKV[]
  countries: TrafficKV[]
  /** Kota, ASN, organisasi, jenis jaringan: hanya ada bila basis data geo terpasang dan ada event ber-IP publik. */
  cities?: TrafficKV[]
  asns?: TrafficKV[]
  orgs?: TrafficKV[]
  net_types?: TrafficKV[]
  referers: TrafficKV[]
  ua_classes: TrafficKV[]
  clients: TrafficKV[]
  instances: TrafficKV[]
  not_found: TrafficKV[]
  top_ips: TrafficIPRow[]
  recommendations: TrafficRecommendation[]
}

export interface TrafficRecentEvent {
  id: number
  at: string
  method: string
  route: string
  path: string
  status: number
  dur_ms: number
  bytes_out: number
  ip: string
  country: string
  city?: string
  asn?: number
  org?: string
  net_type?: string
  ua_class: string
  client: string
  referer: string
  instance: string
}

export interface TrafficSDKKit {
  id: string
  label: string
  language: string
  files: { path: string, content: string }[]
  usage: string
  notes: string
}

export interface TrafficSDKInfo {
  ingest_url: string
  ingest_key: string
  env: string
  env_file: string
  suggested: string
  kits: TrafficSDKKit[]
  component_dir: string
  component_name: string
  curl_test: string
}

export interface TrafficSourceInput {
  name?: string
  project_id?: number | null
  component_id?: number | null
  mode?: TrafficMode
  cloud_url?: string
  sync_interval_sec?: number
  enabled?: boolean
}

export interface TrafficGeoStatus {
  city_ready: boolean
  asn_ready: boolean
  city_built?: number
  asn_built?: number
  auto_update: boolean
  update_cmd: string
}
