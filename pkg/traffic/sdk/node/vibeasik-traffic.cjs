// vibeasik Traffic Analyzer — collector Node.js (Express/Connect/Nest-express). Berkas .cjs: bisa di-require (CommonJS)
// maupun di-import dari ESM: import { vibeasikTraffic } from './vibeasik-traffic.cjs'
// Mengirim rute, status, lama proses, IP, negara, UA & host perujuk. Query string, header lain & body tidak dikirim.
// Env: VIBEASIK_TRAFFIC_URL, VIBEASIK_TRAFFIC_KEY (wajib) · VIBEASIK_TRAFFIC_FLUSH_SEC (default 10; 1–2 = realtime)
//      VIBEASIK_TRAFFIC_INSTANCE · VIBEASIK_TRAFFIC_HASH_IP=1 · VIBEASIK_TRAFFIC_SKIP (default /health)
const crypto = require('node:crypto')
const os = require('node:os')

const ENDPOINT = process.env.VIBEASIK_TRAFFIC_URL || ''
const KEY = process.env.VIBEASIK_TRAFFIC_KEY || ''
const FLUSH_MS = Math.max(1, Number(process.env.VIBEASIK_TRAFFIC_FLUSH_SEC) || 10) * 1000
const INSTANCE = process.env.VIBEASIK_TRAFFIC_INSTANCE || os.hostname()
const HASH_IP = process.env.VIBEASIK_TRAFFIC_HASH_IP === '1'
const SKIP = (process.env.VIBEASIK_TRAFFIC_SKIP || '/health').split(',').map(s => s.trim()).filter(Boolean)
const MAX_BUFFER = 20000
const enabled = Boolean(ENDPOINT && KEY)

let buffer = []
let timer = null

function header(headers, name) {
  const v = headers[name]
  return Array.isArray(v) ? v[0] : (v || '')
}

function clientIp(req) {
  const h = req.headers
  const direct = header(h, 'cf-connecting-ip') || header(h, 'true-client-ip') || header(h, 'x-real-ip')
  if (direct) return direct.trim()
  const xff = header(h, 'x-forwarded-for')
  if (xff) return xff.split(',')[0].trim()
  return (req.socket && req.socket.remoteAddress) || ''
}

function refererHost(ref) {
  try { return ref ? new URL(ref).host : '' } catch { return '' }
}

function record(ev) {
  if (!enabled || SKIP.some(p => ev.path.startsWith(p))) return
  if (HASH_IP && ev.ip) ev.ip = 'h:' + crypto.createHash('sha256').update(ev.ip + KEY).digest('hex').slice(0, 16)
  if (buffer.length < MAX_BUFFER) buffer.push(ev)
  if (!timer) {
    timer = setInterval(flush, FLUSH_MS)
    timer.unref?.()
  }
}

async function flush() {
  if (!buffer.length) return
  let batch = buffer
  buffer = []
  while (batch.length) {
    const chunk = batch.slice(0, 5000)
    try {
      const res = await fetch(ENDPOINT, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Vibeasik-Key': KEY },
        body: JSON.stringify({ instance: INSTANCE, events: chunk }),
        signal: AbortSignal.timeout(10000)
      })
      if (res.status >= 500 || res.status === 429) throw new Error('retry')
    } catch {
      // vibeasik sedang tidak terjangkau: simpan untuk flush berikutnya (tetap dibatasi MAX_BUFFER)
      buffer = batch.concat(buffer).slice(-MAX_BUFFER)
      return
    }
    batch = batch.slice(5000)
  }
}

/** Middleware Express/Connect: app.use(vibeasikTraffic()) — pasang paling awal, sebelum router. */
function vibeasikTraffic() {
  return function (req, res, next) {
    if (!enabled) return next()
    const start = process.hrtime.bigint()
    const ts = Date.now()
    res.on('finish', () => {
      const route = req.route && req.route.path ? (req.baseUrl || '') + req.route.path : ''
      record({
        ts,
        method: req.method,
        route: typeof route === 'string' ? route : '',
        path: (req.originalUrl || req.url || '/').split('?')[0],
        status: res.statusCode,
        dur_ms: Number(process.hrtime.bigint() - start) / 1e6,
        bytes_in: Number(header(req.headers, 'content-length')) || 0,
        bytes_out: Number(res.getHeader('content-length')) || 0,
        ip: clientIp(req),
        country: header(req.headers, 'cf-ipcountry') || header(req.headers, 'x-vercel-ip-country') || header(req.headers, 'x-country-code'),
        ua: header(req.headers, 'user-agent'),
        referer: refererHost(header(req.headers, 'referer'))
      })
    })
    next()
  }
}

// Graceful shutdown aplikasi: panggil `await flush()` sebelum process.exit agar buffer terakhir terkirim.
process.once('beforeExit', flush)

module.exports = { vibeasikTraffic, record, flush, clientIp, refererHost }
