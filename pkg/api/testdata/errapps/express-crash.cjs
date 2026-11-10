// Proses yang mati karena uncaughtException async: SDK harus sempat mengirim event error sebelum mati.
const path = require('node:path')
const { vibeasikTraffic } = require(path.join(process.env.VIBEASIK_SDK_DIR, 'node/vibeasik-traffic.cjs'))
const express = require('./fake-express.cjs')

class CrashError extends Error {}
function crashNow(secret) { void secret; throw new CrashError('crash terjadwal') }

const app = express()
app.use(vibeasikTraffic())
app.get('/ok', (req, res) => { res.end('ok'); setTimeout(() => crashNow('SECRETA'), 20) })
const server = app.listen(0, async () => {
  await fetch('http://127.0.0.1:' + server.address().port + '/ok?token=SECRETQ').then(r => r.text())
})
