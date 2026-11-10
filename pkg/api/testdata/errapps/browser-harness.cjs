// Menjalankan snippet browser asli (vibeasik-errors.js) di vm dengan window/document tiruan (tanpa browser sungguhan).
// Tiga "halaman" memicu error yang sama; satu halaman memicu duplikat dan 30 error berbeda untuk menguji dedup + batas.
const fs = require('node:fs')
const path = require('node:path')
const vm = require('node:vm')

const code = fs.readFileSync(path.join(process.env.VIBEASIK_SDK_DIR, 'browser/vibeasik-errors.js'), 'utf8')
const pending = []

function page(url, key, release) {
  const listeners = {}
  const add = (n, f) => { (listeners[n] ||= []).push(f) }
  const win = {
    addEventListener: add,
    location: { pathname: '/checkout', search: '?token=SECRETQ', hash: '#SECRETC' },
    VIBEASIK_ERRORS: { url, key, release }
  }
  win.window = win
  const ctx = vm.createContext({
    window: win, location: win.location, Blob, JSON, setTimeout, clearTimeout,
    document: { currentScript: null, addEventListener: add, visibilityState: 'visible' },
    navigator: { sendBeacon(u, blob) { pending.push(blob.text().then(b => fetch(u, { method: 'POST', body: b, headers: { 'Content-Type': 'text/plain' } }))); return true } }
  })
  vm.runInContext(code, ctx)
  const fire = (name, ev) => (listeners[name] || []).forEach(f => f(ev))
  return { fire, flush: () => fire('pagehide', {}) }
}

class CartError extends Error {}
function fakeErr(msg) {
  const e = new CartError(msg)
  e.stack = 'CartError: ' + msg + '\n    at renderCart (https://shop.example.com/_nuxt/entry.3f2a1b9c.js?v=SECRETQ:10:5)\n    at onClick (https://shop.example.com/_nuxt/entry.3f2a1b9c.js:99:1)'
  return e
}

for (let i = 1; i <= 3; i++) {
  const p = page(process.env.VIBEASIK_TRAFFIC_URL, process.env.VIBEASIK_TRAFFIC_KEY, process.env.VIBEASIK_RELEASE)
  const err = fakeErr('keranjang rusak ' + i)
  p.fire('error', { error: err, message: err.message })
  p.fire('error', { error: err, message: err.message }) // duplikat di halaman yang sama: harus diabaikan
  if (i === 3) {
    p.fire('unhandledrejection', { reason: Promise.reject.name && new TypeError('x') }) // tipe lain, ikut sebagai grup lain
    for (let k = 0; k < 30; k++) p.fire('error', { error: Object.assign(new RangeError('r' + k), { stack: 'RangeError\n    at f' + k + ' (https://shop.example.com/a.js:1:1)' }), message: 'r' + k })
  }
  p.flush()
}
Promise.all(pending).then(() => { console.log('selesai'); process.exit(0) })
