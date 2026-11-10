// Menjalankan plugin Nitro asli (vibeasik-traffic.nitro.ts) dengan nitroApp tiruan (Nuxt/Nitro tidak dipasang).
// Env: VIBEASIK_PLUGIN (jalur berkas .ts hasil salinan, dengan stub h3 di sebelahnya).
const hooks = {}
const nitroApp = { hooks: { hook(name, fn) { (hooks[name] ||= []).push(fn) } } }
globalThis.defineNitroPlugin = (fn) => fn
const plugin = (await import(process.env.VIBEASIK_PLUGIN)).default
plugin(nitroApp)
const call = async (name, ...args) => { for (const fn of hooks[name] || []) await fn(...args) }

class DbError extends Error {}
function loadUser(id, token) { void token; throw new DbError('db down id=' + id) }

let n = 0
async function hit(path) {
  const event = {
    path: path + '?token=SECRETQ', method: 'GET', context: { matchedRoute: { path: path.startsWith('/boom') ? '/boom/:id' : path } },
    headers: { authorization: 'Bearer SECRETH', cookie: 'sid=SECRETC', 'user-agent': 'harness' },
    node: { res: { statusCode: 200, getHeader() { return undefined } } }
  }
  await call('request', event)
  if (path.startsWith('/boom')) {
    try { loadUser(++n, event.headers.authorization) } catch (e) {
      event.node.res.statusCode = 500
      // h3 membungkus error asli di `cause`
      await call('error', Object.assign(new Error(e.message), { statusCode: 500, cause: e }), { event })
    }
  }
  await call('afterResponse', event)
}
for (const p of ['/boom/11', '/boom/22', '/ok', '/boom/33', '/ok']) await hit(p)
await call('close')
console.log('selesai')
process.exit(0)
