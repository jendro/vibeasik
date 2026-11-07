/** Peta kode (gaya RAG) per proyek — dibangun engine: local-engine/internal/codemap */

export type CodeMapKind
  = | 'page' | 'component' | 'composable' | 'store' | 'layout' | 'handler' | 'route' | 'service' | 'repository' | 'model'
    | 'type' | 'middleware' | 'util' | 'config' | 'entry' | 'test' | 'doc' | 'migration' | 'style' | 'script' | 'module'

export interface CodeSymbol {
  name: string
  kind: string
  line: number
  exported?: boolean
  doc?: string
}

export interface CodeMapSearchHit {
  path: string
  abs_path: string
  component?: string
  kind: CodeMapKind
  lang: string
  lines: number
  score: number
  summary?: string
  symbols?: CodeSymbol[]
  routes?: string[]
  imports: number
  used_by?: string[]
  used_by_count: number
  snippet?: string
  snippet_start?: number
}

export interface CodeMapSearchResponse {
  query: string
  hits: CodeMapSearchHit[]
}

export interface CodeMapOverviewFile {
  path: string
  kind: CodeMapKind
  lang: string
  lines: number
  summary?: string
  symbols?: string[]
  used_by: number
  routes?: number
}

export interface CodeMapOverviewDir {
  path: string
  files: number
  lines: number
  kinds: Record<string, number>
  key_files: string[]
  entries?: CodeMapOverviewFile[]
}

export interface CodeMapOverviewComponent {
  id: number
  name: string
  path: string
  files: number
  lines: number
  kinds: Record<string, number>
  dirs: CodeMapOverviewDir[]
}

export interface CodeMapRankedFile {
  path: string
  kind: CodeMapKind
  lines: number
  used_by: number
  note?: string
}

export interface CodeMapOverview {
  project_id: number
  project_name: string
  root_path: string
  built_at: string
  exported_at?: string
  file_count: number
  symbol_count: number
  enriched_count: number
  total_lines: number
  route_count: number
  languages: Record<string, number>
  kinds: Record<string, number>
  components: CodeMapOverviewComponent[]
  core: CodeMapRankedFile[]
  hotspots: CodeMapRankedFile[]
  entry_points: CodeMapRankedFile[]
  docs: CodeMapRankedFile[]
  guide: string
}

export interface CodeMapFileDetail {
  path: string
  abs_path: string
  component_name?: string
  lang: string
  kind: CodeMapKind
  lines: number
  summary?: string
  ai_summary?: string
  keywords?: string
  updated_at: string
  symbols: CodeSymbol[]
  imports: string[] | null
  used_by: string[] | null
  routes: string[] | null
}

export interface CodeMapBuildResult {
  file_count: number
  parsed: number
  unchanged: number
  removed: number
  duration_ms: number
}

export interface CodeMapExportResult {
  markdown_path: string
  jsonl_path: string
  files: number
}

/** POST /api/projects/:id/init-vibeasik — tautan AI (.vibeasik.json) + peta kode di repo */
export interface InitVibeasikResult {
  project_id: number
  root: string
  files_created: string[]
  code_map: { build: CodeMapBuildResult, exported: boolean }
  code_map_markdown: string
  gitignore_updated: boolean
  /** GEMINI.md/AGENTS.md/CLAUDE.md yang diberi blok "pengetahuan vibeasik dulu" */
  agent_rules?: string[]
  auto_sync_minutes: number
}
