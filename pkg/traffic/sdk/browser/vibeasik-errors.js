/* vibeasik Traffic Analyzer - snippet error browser (VIBEASIK_TRAFFIC_KEY sumber = atribut data-key).
 * Pasang:  <script src="/vibeasik-errors.js" data-url="https://.../api/traffic/ingest" data-key="KUNCI" data-release="v1.2.3" defer></script>
 * atau:    window.VIBEASIK_ERRORS = { url, key, release } sebelum skrip ini dimuat.
 * Menangkap window error & unhandledrejection, lalu mengirim batch kecil ke endpoint ingest yang sama dengan SDK server.
 * Kunci ikut di badan (bukan header) sehingga tanpa preflight CORS; endpoint hanya menerima event error browser dari kunci ini.
 * Yang dikirim: tipe, pesan (maks 500), frame stack (fungsi + berkas:baris, tanpa nilai variabel), location.pathname,
 * release. TIDAK dikirim: query, hash, cookie, header, isi form, body. Dibatasi: dedup per halaman + maksimum 20 event.
 */
(function () {
  'use strict'
  var script = document.currentScript
  var cfg = window.VIBEASIK_ERRORS || {}
  var URL_ = cfg.url || (script && script.getAttribute('data-url')) || ''
  var KEY = cfg.key || (script && script.getAttribute('data-key')) || ''
  var RELEASE = String(cfg.release || (script && script.getAttribute('data-release')) || '').slice(0, 64)
  if (!URL_ || !KEY || window.__vibeasikErrors) return
  window.__vibeasikErrors = true

  var MAX_PER_PAGE = 20
  var SEND_DELAY_MS = 2000
  var seen = {}
  var queue = []
  var sentTotal = 0
  var timer = null

  function frames(err) {
    var raw = err && typeof err.stack === 'string' ? err.stack : ''
    // Chrome: "    at fn (url:l:c)"; Firefox/Safari: "fn@url:l:c". Baris pesan dibuang.
    return raw.split('\n').map(function (l) { return l.trim() }).filter(function (l) {
      return l.indexOf('at ') === 0 || l.indexOf('@') > 0
    }).slice(0, 30).join('\n').replace(/[?#][^\s):]*/g, '').slice(0, 8000)
  }

  function typeOf(err) {
    var c = err && err.constructor && err.constructor.name
    if (c && c !== 'Object') return c
    return (err && err.name) || typeof err
  }

  function enqueue(type, msg, stack) {
    var topLine = stack.split('\n')[0] || ''
    var sig = type + '|' + topLine.replace(/[:\d]+\)?$/, '')
    if (seen[sig] || sentTotal + queue.length >= MAX_PER_PAGE) return
    seen[sig] = true
    queue.push({
      kind: 'error', side: 'browser', ts: Date.now(), method: 'GET', status: 0,
      route: location.pathname, path: location.pathname,
      err_type: String(type).slice(0, 128), err_msg: String(msg).slice(0, 500), stack: stack, release: RELEASE
    })
    if (!timer) timer = setTimeout(send, SEND_DELAY_MS)
  }

  function send() {
    timer = null
    if (!queue.length) return
    var batch = queue
    queue = []
    sentTotal += batch.length
    var body = JSON.stringify({ key: KEY, events: batch })
    try {
      // text/plain = "simple request": tanpa preflight CORS.
      if (navigator.sendBeacon && navigator.sendBeacon(URL_, new Blob([body], { type: 'text/plain' }))) return
      fetch(URL_, { method: 'POST', mode: 'no-cors', keepalive: true, headers: { 'Content-Type': 'text/plain' }, body: body })['catch'](function () {})
    } catch (e) { /* pelaporan tidak boleh mengganggu halaman */ }
  }

  window.addEventListener('error', function (e) {
    try {
      if (e.error) return enqueue(typeOf(e.error), e.message || e.error.message, frames(e.error))
      // Error sumber lain / skrip lintas origin: hanya pesan + lokasi berkas (tanpa query).
      var file = String(e.filename || '').split('?')[0].split('#')[0]
      enqueue('Error', e.message || 'Script error', file ? '@' + file + ':' + (e.lineno || 0) + ':' + (e.colno || 0) : '')
    } catch (x) { /* abaikan */ }
  })

  window.addEventListener('unhandledrejection', function (e) {
    try {
      var r = e.reason
      var msg = r && r.message !== undefined ? r.message : String(r)
      enqueue(r instanceof Error ? typeOf(r) : 'UnhandledRejection', msg, frames(r))
    } catch (x) { /* abaikan */ }
  })

  window.addEventListener('pagehide', send)
  document.addEventListener('visibilitychange', function () { if (document.visibilityState === 'hidden') send() })
})()
