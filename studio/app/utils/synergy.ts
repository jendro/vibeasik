import type { ProjectSynergyMap, SynergyFrontendPage, SynergyRequestFlow } from '~/types'

export type SynergyInspectTab = 'payload' | 'validation' | 'handler' | 'database' | 'response'
export interface SynergyInspectPage { title: string, page_path: string, file_path: string }

// Semua tabel database yang muncul di peta (untuk filter tabel).
export const synergyTables = (map: ProjectSynergyMap | null): string[] => {
  if (!map) return []
  const set = new Set<string>()
  for (const t of map.database_map || []) {
    if (t.table_name) set.add(t.table_name)
  }
  for (const p of map.pages || []) {
    for (const r of p.outbound_requests) {
      for (const tbl of r.database_operation?.target_tables || []) {
        if (tbl) set.add(tbl)
      }
    }
  }
  return Array.from(set).sort()
}

export const matchesMethod = (req: SynergyRequestFlow, method: string): boolean => {
  const m = (method || 'ALL').toUpperCase()
  return m === 'ALL' || req.http_method.toUpperCase() === m
}

export const matchesTableOrStatus = (req: SynergyRequestFlow, filterVal: string): boolean => {
  const val = (filterVal || 'all').toLowerCase()
  if (val === 'all') return true
  const reqTables = req.database_operation?.target_tables || []
  const matchTable = reqTables.some(t => t.toLowerCase() === val)
  const matchStatus = req.response_spec?.status_code != null && String(req.response_spec.status_code) === val
  return matchTable || matchStatus
}

export const matchesMethodAndStatus = (req: SynergyRequestFlow, method: string, tableOrStatus = 'all'): boolean => {
  return matchesMethod(req, method) && matchesTableOrStatus(req, tableOrStatus)
}

export const matchesMethodAndTable = matchesMethodAndStatus

export const matchesPageSearch = (page: SynergyFrontendPage, query: string): boolean => {
  const q = (query || '').trim().toLowerCase()
  if (!q) return true
  const pathMatch = page.page_path ? page.page_path.toLowerCase().includes(q) : false
  const titleMatch = page.title ? page.title.toLowerCase().includes(q) : false
  return pathMatch || titleMatch
}

export const matchesRequestSearch = (req: SynergyRequestFlow, query: string): boolean => {
  const q = (query || '').trim().toLowerCase()
  if (!q) return true

  const fields = [
    req.action_title,
    req.route_path,
    req.handler_function,
    req.http_method,
    req.middleware,
    ...(req.database_operation?.target_tables || []),
    ...(req.database_operation?.stored_fields || []),
    ...(req.validation_rules || [])
  ]

  return fields.some(val => Boolean(val && val.toLowerCase().includes(q)))
}

export const matchesSearchQuery = (page: SynergyFrontendPage, req: SynergyRequestFlow, query: string): boolean => {
  const q = (query || '').trim().toLowerCase()
  if (!q) return true
  if (matchesPageSearch(page, q)) return true
  return matchesRequestSearch(req, q)
}

export const filterPageRequests = (page: SynergyFrontendPage, query: string, method: string, tableOrStatus = 'all'): SynergyRequestFlow[] => {
  const reqs = page.outbound_requests || []
  return reqs.filter(req => {
    if (!matchesMethodAndStatus(req, method, tableOrStatus)) {
      return false
    }
    return matchesSearchQuery(page, req, query)
  })
}

export const shouldKeepPage = (pageMatches: boolean, reqCount: number, hasFilter: boolean): boolean => {
  if (hasFilter) {
    return reqCount > 0
  }
  return pageMatches || reqCount > 0
}

// Halaman + request yang lolos pencarian & filter metode/tabel (untuk pipeline, matriks & topologi).
export const filterSynergyPages = (map: ProjectSynergyMap | null, search: string, method: string, table: string): SynergyFrontendPage[] => {
  if (!map?.pages) return []

  const q = (search || '').trim().toLowerCase()
  const m = (method || 'ALL').toUpperCase()
  const tbl = (table || 'all').toLowerCase()
  const hasFilter = m !== 'ALL' || tbl !== 'all'

  const filtered: SynergyFrontendPage[] = []

  for (const page of map.pages) {
    const matchingRequests = filterPageRequests(page, q, m, tbl)
    const pageMatches = matchesPageSearch(page, q)

    if (!shouldKeepPage(pageMatches, matchingRequests.length, hasFilter)) {
      continue
    }

    filtered.push({
      ...page,
      outbound_requests: matchingRequests
    })
  }

  return filtered
}

export interface SynergyMatrixRow {
  pagePath: string
  pageTitle: string
  filePath: string
  navigationTargets: string[]
  method: string
  routePath: string
  handler: string
  middleware: string
  payload: string
  validations: string[]
  businessLogic: string[]
  dbEngine: string
  dbTables: string[]
  storedFields: string[]
  statusCode: number
  responseSample: string
  uiImpact: string
  rawFlow?: SynergyRequestFlow
  rawPage: { title: string, page_path: string, file_path: string }
}

// Baris matriks audit tabular dari halaman terfilter.
export const buildSynergyMatrix = (pages: SynergyFrontendPage[]): SynergyMatrixRow[] => {
  const rows: SynergyMatrixRow[] = []
  for (const p of pages) {
    const navTextList = p.navigation_targets.map(n => `${n.target_path} (${n.target_label})`)
    const pageObj = { title: p.title, page_path: p.page_path, file_path: p.file_path }

    if (p.outbound_requests.length === 0) {
      rows.push({
        pagePath: p.page_path,
        pageTitle: p.title,
        filePath: p.file_path,
        navigationTargets: navTextList,
        method: 'NONE',
        routePath: 'Tanpa request API langsung',
        handler: '-',
        middleware: '-',
        payload: '-',
        validations: ['-'],
        businessLogic: ['Halaman statis / presentasi UI'],
        dbEngine: '-',
        dbTables: [],
        storedFields: [],
        statusCode: 200,
        responseSample: '-',
        uiImpact: 'Render antarmuka pengguna',
        rawPage: pageObj
      })
    } else {
      for (const req of p.outbound_requests) {
        rows.push({
          pagePath: p.page_path,
          pageTitle: p.title,
          filePath: p.file_path,
          navigationTargets: navTextList,
          method: req.http_method,
          routePath: req.route_path,
          handler: req.handler_function,
          middleware: req.middleware,
          payload: [req.request_payload?.params, req.request_payload?.body].filter(Boolean).join(' | ') || '-',
          validations: req.validation_rules || [],
          businessLogic: req.business_process || [],
          dbEngine: req.database_operation?.engine || 'SQLite',
          dbTables: req.database_operation?.target_tables || [],
          storedFields: req.database_operation?.stored_fields || [],
          statusCode: req.response_spec?.status_code || 200,
          responseSample: req.response_spec?.body_sample || '-',
          uiImpact: req.ui_impact || 'Perbarui state reaktif UI',
          rawFlow: req,
          rawPage: pageObj
        })
      }
    }
  }
  return rows
}

// Ringkasan peta sinergi sebagai Markdown.
export const buildSynergyMarkdown = (map: ProjectSynergyMap): string => {
  let md = `# Peta Sinergi Proyek: ${map.project_name}\n\n`
  md += `**Ringkasan**: ${map.summary}\n`
  md += `**Total Halaman**: ${map.total_pages} | **Total Alur**: ${map.total_flows} | **Tabel DB**: ${map.total_tables}\n\n`

  for (const p of map.pages) {
    md += `## 📱 Halaman: ${p.title} (\`${p.page_path}\`)\n`
    md += `- **File**: \`${p.file_path || 'Unknown'}\`\n`
    if (p.navigation_targets.length > 0) {
      md += `- **Navigasi Tujuan**:\n`
      for (const n of p.navigation_targets) {
        md += `  - ➔ \`${n.target_path}\`: ${n.target_label} (${n.trigger})\n`
      }
    }
    if (p.outbound_requests.length > 0) {
      md += `\n### Alur Request API:\n`
      for (const r of p.outbound_requests) {
        md += `#### [${r.http_method}] \`${r.route_path}\` - ${r.action_title}\n`
        md += `- **Pemicu**: ${r.trigger_event}\n`
        md += `- **Handler BE**: \`${r.handler_function}\` (${r.middleware})\n`
        md += `- **Data Dikirim**: ${r.request_payload?.params || ''} ${r.request_payload?.body || ''}\n`
        md += `- **Validasi**: ${r.validation_rules?.join('; ') || '-'}\n`
        md += `- **Proses**: ${r.business_process?.join(' ➔ ') || '-'}\n`
        md += `- **Database**: Tabel \`${r.database_operation?.target_tables?.join(', ')}\` | Kolom tersimpan: \`${r.database_operation?.stored_fields?.join(', ')}\`\n`
        md += `- **Response**: HTTP ${r.response_spec?.status_code} (${r.response_spec?.data_type})\n`
        md += `- **UI Impact**: ${r.ui_impact}\n\n`
      }
    }
    md += `---\n\n`
  }
  return md
}

// Perintah cURL untuk satu alur request pada port backend proyek.
export const buildCurlCommand = (req: SynergyRequestFlow, port: number): string => {
  const method = req.http_method.toUpperCase()
  let cmd = `curl -X ${method} "http://localhost:${port}${req.route_path}"`
  if (req.request_payload?.headers) {
    for (const h of req.request_payload.headers.split(',')) {
      if (h.trim()) cmd += ` \\\n  -H "${h.trim()}"`
    }
  } else {
    cmd += ` \\\n  -H "Content-Type: application/json"`
  }
  if (req.request_payload?.body && method !== 'GET') {
    cmd += ` \\\n  -d '${req.request_payload.body}'`
  }
  return cmd
}
