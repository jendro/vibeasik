import type { Todo, TrafficMeta, TrafficRecentEvent, TrafficReport, TrafficSDKInfo, TrafficSource, TrafficSourceInput, TrafficWindow } from '~/types'

/** Traffic Analyzer: sumber traffic aplikasi live, laporan per jendela waktu, ekor live, SDK & saran → todo. */
export const useTraffic = () => {
  const { apiBase } = useApi()
  const base = `${apiBase}/api/traffic`
  const src = (id: number) => `${base}/sources/${id}`

  return {
    getMeta: () => $fetch<TrafficMeta>(`${base}/meta`),
    listSources: (projectId?: number | null) => $fetch<TrafficSource[]>(`${base}/sources`, { query: { project_id: projectId || undefined } }),
    createSource: (body: TrafficSourceInput) => $fetch<TrafficSource>(`${base}/sources`, { method: 'POST', body }),
    updateSource: (id: number, body: TrafficSourceInput) => $fetch<TrafficSource>(src(id), { method: 'PUT', body }),
    deleteSource: (id: number) => $fetch<{ deleted: boolean, warning: string }>(src(id), { method: 'DELETE' }),
    syncSource: (id: number) => $fetch<{ pulled: number, source: TrafficSource }>(`${src(id)}/sync`, { method: 'POST' }),
    rotateKey: (id: number) => $fetch<{ rotated: boolean }>(`${src(id)}/rotate-key`, { method: 'POST' }),
    getReport: (id: number, window: TrafficWindow) => $fetch<TrafficReport>(`${src(id)}/report`, { query: { window } }),
    getRecent: (id: number, limit = 100) => $fetch<{ events: TrafficRecentEvent[], source: TrafficSource }>(`${src(id)}/recent`, { query: { limit } }),
    getSDK: (id: number) => $fetch<TrafficSDKInfo>(`${src(id)}/sdk`),
    installSDK: (id: number, kit: string, overwrite = false) =>
      $fetch<{ written: string[], skipped: string[], dir: string, usage: string }>(`${src(id)}/sdk/install`, { method: 'POST', body: { kit, overwrite } }),
    recommendationToTodo: (id: number, recId: string, window: TrafficWindow) =>
      $fetch<{ todo: Todo, already_exists: boolean }>(`${src(id)}/todo`, { method: 'POST', body: { rec_id: recId, window } })
  }
}

export const trafficIntervalLabel = (sec: number) => {
  if (sec <= 5) return 'Realtime (5 dtk)'
  if (sec < 60) return `${sec} detik`
  if (sec < 3600) return `${sec / 60} menit`
  return `${sec / 3600} jam`
}

export const trafficWindowLabel: Record<TrafficWindow, string> = {
  '15m': '15 menit', '1h': '1 jam', '6h': '6 jam', '24h': '24 jam', '7d': '7 hari', '30d': '30 hari'
}

export const trafficCategoryMeta: Record<string, { label: string, icon: string }> = {
  rate_limit: { label: 'Rate limit', icon: 'i-lucide-gauge' },
  cache: { label: 'Cache', icon: 'i-lucide-database-zap' },
  load_balancer: { label: 'Load balancer', icon: 'i-lucide-split' },
  split_service: { label: 'Pecah service', icon: 'i-lucide-boxes' },
  performance: { label: 'Performa', icon: 'i-lucide-timer' },
  errors: { label: 'Error', icon: 'i-lucide-circle-x' },
  security: { label: 'Keamanan', icon: 'i-lucide-shield-alert' },
  payload: { label: 'Payload', icon: 'i-lucide-package' },
  capacity: { label: 'Kapasitas', icon: 'i-lucide-trending-up' },
  info: { label: 'Info', icon: 'i-lucide-info' }
}

export const fmtMs = (v: number) => v >= 1000 ? `${(v / 1000).toFixed(2)} dtk` : `${Math.round(v)} ms`
export const fmtPct = (v: number) => `${(v * 100).toFixed(v > 0 && v < 0.01 ? 2 : 1)}%`
export const fmtNum = (v: number) => new Intl.NumberFormat('id-ID', { maximumFractionDigits: 1 }).format(v)
export const fmtBytes = (v: number) => v >= 1 << 20 ? `${(v / (1 << 20)).toFixed(1)} MB` : v >= 1024 ? `${Math.round(v / 1024)} KB` : `${Math.round(v)} B`
export const fmtAgo = (iso: string | null) => {
  if (!iso) return 'belum pernah'
  const s = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 1000))
  if (s < 60) return `${s} dtk lalu`
  if (s < 3600) return `${Math.round(s / 60)} mnt lalu`
  if (s < 86400) return `${Math.round(s / 3600)} jam lalu`
  return `${Math.round(s / 86400)} hari lalu`
}
