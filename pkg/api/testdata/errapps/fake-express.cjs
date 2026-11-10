// Pengganti Express minimal untuk uji (express tidak dipasang di repo): rantai middleware, rute dengan req.route,
// dan middleware error 4 argumen, dengan semantik yang sama dengan Express untuk yang dipakai SDK.
const http = require('node:http')

module.exports = function fakeExpress() {
  const stack = []
  const app = {
    use(fn) { stack.push({ fn }); return app },
    get(path, fn) { stack.push({ fn, path, method: 'GET' }); return app },
    listen(port, cb) { return http.createServer((req, res) => run(req, res)).listen(port, '127.0.0.1', cb) }
  }
  function run(req, res) {
    req.originalUrl = req.url
    req.baseUrl = ''
    let i = 0
    const final = (err) => {
      if (res.writableEnded) return
      res.statusCode = err ? (err.status || 500) : 404
      res.end(err ? 'error' : 'not found')
    }
    const next = (err) => {
      const layer = stack[i++]
      if (!layer) return final(err)
      const isErrMw = layer.fn.length === 4
      if (err && !isErrMw) return next(err)
      if (!err && isErrMw) return next()
      if (layer.path && (layer.path !== req.url.split('?')[0].replace(/\/\d+$/, '/:id') || layer.method !== req.method)) return next(err)
      if (layer.path) req.route = { path: layer.path }
      try { isErrMw ? layer.fn(err, req, res, next) : layer.fn(req, res, next) } catch (e) { next(e) }
    }
    next()
  }
  return app
}
