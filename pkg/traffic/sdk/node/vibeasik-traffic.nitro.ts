// vibeasik Traffic Analyzer — plugin Nitro (Nuxt server / Nitro standalone). Simpan di server/plugins/.
// Env sama dengan SDK lain: VIBEASIK_TRAFFIC_URL, VIBEASIK_TRAFFIC_KEY, VIBEASIK_TRAFFIC_FLUSH_SEC, VIBEASIK_TRAFFIC_INSTANCE,
// VIBEASIK_TRAFFIC_HASH_IP=1, VIBEASIK_TRAFFIC_SKIP (default /health,/_nuxt), VIBEASIK_RELEASE (versi/commit rilis).
// Query string, header lain & body tidak dikirim. Hook `error` Nitro dilaporkan sebagai event error (tipe, pesan, frame stack
// tanpa nilai variabel); heartbeat tiap 30 detik (rilis, memori, uptime).
import { createHash } from 'node:crypto'
import { hostname } from 'node:os'
import { getRequestHeader, getRequestIP } from 'h3'

const ENDPOINT = process.env.VIBEASIK_TRAFFIC_URL || ''
const KEY = process.env.VIBEASIK_TRAFFIC_KEY || ''
const FLUSH_MS = Math.max(1, Number(process.env.VIBEASIK_TRAFFIC_FLUSH_SEC) || 10) * 1000
const INSTANCE = process.env.VIBEASIK_TRAFFIC_INSTANCE || hostname()
const HASH_IP = process.env.VIBEASIK_TRAFFIC_HASH_IP === '1'
const SKIP = (process.env.VIBEASIK_TRAFFIC_SKIP || '/health,/_nuxt').split(',').map(s => s.trim()).filter(Boolean)
const RELEASE = (process.env.VIBEASIK_RELEASE || '').slice(0, 64)
const HEARTBEAT_MS = 30000
const MAX_BUFFER = 20000

let buffer: Record<string, unknown>[] = []

function push(ev: Record<string, unknown>) {
  if (RELEASE) ev.release = RELEASE
  if (buffer.length < MAX_BUFFER) buffer.push(ev)
}

function errorType(err: unknown): string {
  const c = (err as { constructor?: { name?: string } } | null)?.constructor?.name
  if (c && c !== 'Object') return c
  return (err as { name?: string } | null)?.name || typeof err
}

// Hanya baris "at fungsi (berkas:baris:kolom)"; baris pesan error dibuang.
function stackFrames(err: unknown): string {
  const raw = (err as { stack?: unknown } | null)?.stack
  if (typeof raw !== 'string') return ''
  return raw.split('\n').map(l => l.trim()).filter(l => l.startsWith('at ')).slice(0, 40).join('\n').slice(0, 8000)
}

// h3 membungkus error asli di `cause`; tipe & stack diambil dari yang asli.
function rootCause(err: unknown): unknown {
  const cause = (err as { cause?: unknown } | null)?.cause
  return cause instanceof Error ? cause : err
}

function captureError(err: unknown, event?: { method?: string, path?: string, context?: Record<string, unknown> }) {
  const status = Number((err as { statusCode?: number } | null)?.statusCode) || 500
  if (status < 500) return
  const root = rootCause(err)
  const message = (root as { message?: unknown } | null)?.message
  push({
    kind: 'error',
    ts: Date.now(),
    method: event?.method || '',
    route: (event?.context?.matchedRoute as { path?: string } | undefined)?.path || '',
    path: (event?.path || '').split('?')[0],
    status,
    err_type: errorType(root).slice(0, 128),
    err_msg: String(message !== undefined ? message : root).slice(0, 500),
    stack: stackFrames(root),
    side: 'server'
  })
}

function heartbeat() {
  push({ kind: 'heartbeat', ts: Date.now(), instance: INSTANCE, mem_bytes: process.memoryUsage().rss, uptime_s: Math.floor(process.uptime()) })
}

async function flush() {
  if (!buffer.length) return
  let batch = buffer
  buffer = []
  while (batch.length) {
    try {
      const res = await fetch(ENDPOINT, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Vibeasik-Key': KEY },
        body: JSON.stringify({ instance: INSTANCE, events: batch.slice(0, 5000) }),
        signal: AbortSignal.timeout(10000)
      })
      if (res.status >= 500 || res.status === 429) throw new Error('retry')
    } catch {
      buffer = batch.concat(buffer).slice(-MAX_BUFFER)
      return
    }
    batch = batch.slice(5000)
  }
}

export default defineNitroPlugin((nitroApp) => {
  if (!ENDPOINT || !KEY) return
  const timerId = setInterval(flush, FLUSH_MS)
  timerId.unref?.()
  heartbeat()
  const beatId = setInterval(heartbeat, HEARTBEAT_MS)
  beatId.unref?.()
  nitroApp.hooks.hook('close', () => {
    clearInterval(timerId)
    clearInterval(beatId)
    return flush()
  })

  nitroApp.hooks.hook('error', (error: unknown, ctx: { event?: { method?: string, path?: string, context?: Record<string, unknown> } }) => {
    try { captureError(error, ctx?.event) } catch { /* SDK tidak boleh menjatuhkan aplikasi inang */ }
  })

  nitroApp.hooks.hook('request', (event) => {
    event.context.__vibeasikStart = performance.now()
    event.context.__vibeasikTs = Date.now()
  })
  nitroApp.hooks.hook('afterResponse', (event) => {
    const start = event.context.__vibeasikStart as number | undefined
    const path = (event.path || '/').split('?')[0] || '/'
    if (start === undefined || SKIP.some(p => path.startsWith(p))) return
    let ip = getRequestHeader(event, 'cf-connecting-ip') || getRequestIP(event, { xForwardedFor: true }) || ''
    if (HASH_IP && ip) ip = 'h:' + createHash('sha256').update(ip + KEY).digest('hex').slice(0, 16)
    let referer = ''
    try { referer = new URL(getRequestHeader(event, 'referer') || '').host } catch { /* tanpa perujuk */ }
    push({
      ts: event.context.__vibeasikTs,
      method: event.method,
      route: (event.context.matchedRoute as { path?: string } | undefined)?.path || '',
      path,
      status: event.node.res.statusCode,
      dur_ms: performance.now() - start,
      bytes_in: Number(getRequestHeader(event, 'content-length')) || 0,
      bytes_out: Number(event.node.res.getHeader('content-length')) || 0,
      ip,
      country: getRequestHeader(event, 'cf-ipcountry') || getRequestHeader(event, 'x-vercel-ip-country') || '',
      ua: getRequestHeader(event, 'user-agent') || '',
      referer
    })
  })
})
