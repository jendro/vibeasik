import type { CodeMapBuildResult, CodeMapExportResult, CodeMapFileDetail, CodeMapOverview, CodeMapSearchResponse } from '~/types'

/** Akses peta kode proyek: overview, pencarian ala RAG, detail berkas, build & ekspor ke repositori. */
export const useCodeMap = () => {
  const { apiBase } = useApi()
  const base = (projectId: number) => `${apiBase}/api/projects/${projectId}/code-map`

  return {
    getOverview: (projectId: number, detail: 'dirs' | 'files' = 'files') =>
      $fetch<CodeMapOverview>(base(projectId), { query: { detail } }),
    search: (projectId: number, q: string, opts: { kind?: string, componentId?: number, limit?: number, snippet?: boolean } = {}) =>
      $fetch<CodeMapSearchResponse>(`${base(projectId)}/search`, {
        query: { q, kind: opts.kind || undefined, component_id: opts.componentId || undefined, limit: opts.limit, snippet: opts.snippet ? 'true' : undefined }
      }),
    getFile: (projectId: number, path: string) => $fetch<CodeMapFileDetail>(`${base(projectId)}/file`, { query: { path } }),
    build: (projectId: number, force = false) => $fetch<CodeMapBuildResult>(`${base(projectId)}/build`, { method: 'POST', body: { force } }),
    exportToRepo: (projectId: number) => $fetch<CodeMapExportResult>(`${base(projectId)}/export`, { method: 'POST' })
  }
}

export const codeMapKindMeta: Record<string, { label: string, icon: string }> = {
  page: { label: 'Halaman', icon: 'i-lucide-app-window' },
  component: { label: 'Komponen', icon: 'i-lucide-component' },
  composable: { label: 'Composable', icon: 'i-lucide-function-square' },
  store: { label: 'Store', icon: 'i-lucide-database-zap' },
  layout: { label: 'Layout', icon: 'i-lucide-layout-template' },
  handler: { label: 'Handler API', icon: 'i-lucide-server' },
  route: { label: 'Rute', icon: 'i-lucide-route' },
  service: { label: 'Service', icon: 'i-lucide-cog' },
  repository: { label: 'Repository', icon: 'i-lucide-database' },
  model: { label: 'Model', icon: 'i-lucide-box' },
  type: { label: 'Tipe', icon: 'i-lucide-braces' },
  middleware: { label: 'Middleware', icon: 'i-lucide-shield' },
  util: { label: 'Utilitas', icon: 'i-lucide-wrench' },
  config: { label: 'Konfigurasi', icon: 'i-lucide-settings-2' },
  entry: { label: 'Titik masuk', icon: 'i-lucide-log-in' },
  test: { label: 'Test', icon: 'i-lucide-flask-conical' },
  doc: { label: 'Dokumen', icon: 'i-lucide-book-open' },
  migration: { label: 'Migrasi', icon: 'i-lucide-database-backup' },
  style: { label: 'Style', icon: 'i-lucide-palette' },
  script: { label: 'Skrip', icon: 'i-lucide-terminal' },
  module: { label: 'Modul', icon: 'i-lucide-file-code-2' }
}

export const kindMeta = (kind?: string) => codeMapKindMeta[kind || 'module'] || codeMapKindMeta.module!
