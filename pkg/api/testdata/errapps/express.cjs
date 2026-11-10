// Aplikasi contoh yang sengaja error (Express-style + SDK traffic). Env: VIBEASIK_SDK_DIR.
const path = require('node:path')
const { vibeasikTraffic, vibeasikErrors, flush } = require(path.join(process.env.VIBEASIK_SDK_DIR, 'node/vibeasik-traffic.cjs'))
const express = require('./fake-express.cjs')

class DbError extends Error {}

function loadUser(id, token) {
  void token // nilai argumen tidak boleh muncul di stack
  throw new DbError('db down id=' + id)
}

const app = express()
let n = 0
app.use(vibeasikTraffic())
app.get('/boom/:id', (req, res) => { loadUser(++n, req.headers.authorization) })
app.get('/ok', (req, res) => { res.end('ok') })
app.use(vibeasikErrors())

const server = app.listen(0, async () => {
  const base = 'http://127.0.0.1:' + server.address().port
  for (const p of ['/boom/11', '/boom/22', '/ok', '/boom/33', '/ok']) {
    await fetch(base + p + '?token=SECRETQ', { headers: { Authorization: 'Bearer SECRETH', Cookie: 'sid=SECRETC' } }).then(r => r.text())
  }
  await flush()
  console.log('selesai')
  process.exit(0)
})
