// vibeasik Traffic Analyzer — plugin Nitro (Nuxt server / Nitro standalone). Simpan di server/plugins/.
// Env sama dengan SDK lain: VIBEASIK_TRAFFIC_URL, VIBEASIK_TRAFFIC_KEY, VIBEASIK_TRAFFIC_FLUSH_SEC, VIBEASIK_TRAFFIC_INSTANCE,
// VIBEASIK_TRAFFIC_HASH_IP=1, VIBEASIK_TRAFFIC_SKIP (default /health,/_nuxt). Query string, header lain & body tidak dikirim.
import { createHash } from 'node:crypto'
import { hostname } from 'node:os'
import { getRequestHeader, getRequestIP } from 'h3'

const ENDPOINT = process.env.VIBEASIK_TRAFFIC_URL || ''
const KEY = process.env.VIBEASIK_TRAFFIC_KEY || ''
const FLUSH_MS = Math.max(1, Number(process.env.VIBEASIK_TRAFFIC_FLUSH_SEC) || 10) * 1000
const INSTANCE = process.env.VIBEASIK_TRAFFIC_INSTANCE || hostname()
const HASH_IP = process.env.VIBEASIK_TRAFFIC_HASH_IP === '1'
const SKIP = (process.env.VIBEASIK_TRAFFIC_SKIP || '/health,/_nuxt').split(',').map(s => s.trim()).filter(Boolean)
const MAX_BUFFER = 20000

let buffer: Record<string, unknown>[] = []

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
  setInterval(flush, FLUSH_MS).unref?.()
  nitroApp.hooks.hook('close', flush)

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
    if (buffer.length >= MAX_BUFFER) return
    buffer.push({
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
