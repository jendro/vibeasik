import type { Finding } from '~/types'

// Satukan Serupa (konsep Aturan 43 untuk temuan audit): temuan dengan rule sama dan ekosistem berkas sama
// (Go tidak disatukan dengan kode web) menjadi satu temuan utama; perbedaannya (judul sisa, berkas, baris)
// tampil sebagai sub temuan.
export interface FindingGroup {
  key: string
  findings: Finding[]
  /** Bagian judul yang sama di semua anggota */
  title: string
  severity: string
  fileCount: number
}

const ecoByExt: Record<string, string> = {
  go: 'go', php: 'php', py: 'python', rb: 'ruby', rs: 'rust', java: 'jvm', kt: 'jvm', dart: 'dart', swift: 'swift',
  vue: 'web', ts: 'web', tsx: 'web', js: 'web', jsx: 'web', mjs: 'web', css: 'web', scss: 'web', html: 'web', svelte: 'web'
}

const ecosystemOf = (file?: string) => {
  const ext = file?.match(/\.([a-z0-9]+)$/i)?.[1]?.toLowerCase()
  return (ext && ecoByExt[ext]) || ''
}

const severityRank: Record<string, number> = { CRITICAL: 3, WARNING: 2, INFO: 1 }

const unbalanced = (s: string) => (s.match(/\(/g)?.length || 0) > (s.match(/\)/g)?.length || 0)

// Awalan kata yang sama di semua judul, tanpa kurung yang terbuka setengah (mis. "Rute (GET").
const commonTitle = (titles: string[]) => {
  const words = titles.map(t => t.split(/\s+/))
  const first = words[0] || []
  let n = 0
  while (n < first.length && words.every(w => w[n] === first[n])) n++
  while (n > 0 && unbalanced(first.slice(0, n).join(' '))) n--
  const prefix = first.slice(0, n).join(' ').replace(/[\s:\-–—]+$/, '')
  return prefix.length >= 8 ? prefix : (titles[0] || '')
}

/** Sisa judul anggota setelah bagian yang sama — kosong bila judul identik. */
export const findingGroupRemainder = (group: FindingGroup, f: Finding) => {
  const t = f.title || ''
  if (!t.startsWith(group.title)) return t
  const rest = t.slice(group.title.length).replace(/^[\s:\-–—]+/, '').trim()
  // "(549 Baris)" → "549 Baris"
  return /^\([^()]*\)$/.test(rest) ? rest.slice(1, -1) : rest
}

export const groupSimilarFindings = (list: Finding[]): FindingGroup[] => {
  const byKey = new Map<string, Finding[]>()
  for (const f of list) {
    const rule = f.rule_id || f.title || 'issue'
    const key = `${rule}|${f.category || ''}|${ecosystemOf(f.target_file)}`
    const arr = byKey.get(key)
    if (arr) arr.push(f)
    else byKey.set(key, [f])
  }
  return [...byKey.entries()].map(([key, findings]) => ({
    key,
    findings,
    title: findings.length > 1 ? commonTitle(findings.map(f => f.title || '')) : (findings[0]?.title || ''),
    severity: findings.reduce((best, f) => {
      const s = f.severity?.toUpperCase() || ''
      return (severityRank[s] || 0) > (severityRank[best] || 0) ? s : best
    }, ''),
    fileCount: new Set(findings.map(f => f.target_file || '')).size
  }))
}
