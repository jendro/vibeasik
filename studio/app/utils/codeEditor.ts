// Tampilan kode bergaya editor untuk bukti biaya proses: pewarnaan sintaks ringan (tanpa pustaka) dan jendela
// baris di sekitar baris bukti, dengan celah panjang dilipat seperti editor.
import type { CostEvidence } from '~/types'

export type TokenKind = 'plain' | 'comment' | 'string' | 'keyword' | 'number' | 'func' | 'var'

export interface CodeToken {
  text: string
  kind: TokenKind
}

const KEYWORDS = new Set([
  // Go
  'func', 'return', 'if', 'else', 'for', 'range', 'switch', 'case', 'default', 'break', 'continue', 'go', 'defer', 'select',
  'var', 'const', 'type', 'struct', 'interface', 'map', 'chan', 'package', 'import', 'nil', 'true', 'false', 'err',
  // PHP / JS / TS
  'function', 'foreach', 'as', 'while', 'do', 'new', 'class', 'public', 'private', 'protected', 'static', 'use', 'namespace',
  'null', 'echo', 'try', 'catch', 'finally', 'throw', 'let', 'await', 'async', 'export', 'from', 'this', 'self', 'array', 'fn'
])

const TOKEN_RE = /(\/\/.*$|#(?!\[).*$|\/\*.*?(?:\*\/|$))|("(?:\\.|[^"\\])*"?|'(?:\\.|[^'\\])*'?|`[^`]*`?)|(\$[A-Za-z_]\w*)|(\b\d[\d_.]*\b)|([A-Za-z_]\w*)(?=\s*\()|([A-Za-z_]\w*)/g

/** Memecah satu baris kode menjadi token berwarna. Cukup untuk Go, PHP, JS/TS; komentar blok lintas baris tidak dilacak. */
export const tokenizeLine = (line: string): CodeToken[] => {
  const out: CodeToken[] = []
  let last = 0
  const push = (text: string, kind: TokenKind) => {
    if (!text) return
    const prev = out[out.length - 1]
    if (prev && prev.kind === kind) prev.text += text
    else out.push({ text, kind })
  }
  for (const m of line.matchAll(TOKEN_RE)) {
    const at = m.index ?? 0
    push(line.slice(last, at), 'plain')
    const [all, comment, str, phpVar, num, call, word] = m
    if (comment) push(comment, 'comment')
    else if (str) push(str, 'string')
    else if (phpVar) push(phpVar, 'var')
    else if (num) push(num, 'number')
    else if (call) push(call, KEYWORDS.has(call) ? 'keyword' : 'func')
    else if (word) push(word, KEYWORDS.has(word) ? 'keyword' : 'plain')
    last = at + all.length
  }
  push(line.slice(last), 'plain')
  return out
}

/** Bagian jendela editor: rentang baris yang tampil, dipisah lipatan. */
export interface EditorSegment {
  from: number
  to: number
}

export const EDITOR_CONTEXT = 3
const MAX_GAP = 6

/** Rentang baris yang ditampilkan: tiap baris bukti ± konteks; celah ≤ 6 baris digabung, selebihnya dilipat. */
export const editorSegments = (lines: number[], context = EDITOR_CONTEXT, total = Infinity): EditorSegment[] => {
  const sorted = [...new Set(lines.filter(n => n > 0))].sort((a, b) => a - b)
  const segs: EditorSegment[] = []
  for (const n of sorted) {
    const from = Math.max(1, n - context)
    const to = Math.min(total, n + context)
    const last = segs[segs.length - 1]
    if (last && from - last.to <= MAX_GAP) last.to = Math.max(last.to, to)
    else segs.push({ from, to })
  }
  return segs
}

/** Query yang biayanya ikut tumbuh bersama data: di dalam loop (N+1) atau Q(n) berpangkat n lewat fungsi lain. */
export const isHeavyQuery = (e: CostEvidence): boolean => {
  if (e.kind !== 'query') return false
  if (e.depth > 0) return true
  const q = e.note.match(/Q = ([^,]+)/)?.[1] || ''
  return /n/.test(q)
}

/** Ringkas penanda baris di gutter editor. */
export const evidenceMark = (e: CostEvidence): { label: string, tone: 'heavy' | 'query' | 'loop' | 'call' } => {
  if (isHeavyQuery(e)) return { label: e.depth > 0 ? 'N+1' : 'Q(n)', tone: 'heavy' }
  if (e.kind === 'query') return { label: 'query', tone: 'query' }
  if (e.kind === 'loop') return { label: e.depth > 1 ? `loop ×${e.depth}` : 'loop', tone: 'loop' }
  return { label: 'panggil', tone: 'call' }
}
