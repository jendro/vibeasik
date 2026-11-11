// Token akses lokal engine.
// - Studio produksi disajikan engine sendiri: cookie HttpOnly dari /auth ikut terkirim otomatis.
// - Dev server Nuxt (origin lain): set NUXT_PUBLIC_API_TOKEN=$(vibeasik token) agar header dikirim.
// Bila engine menjawab 401 (token diganti/cookie habis), muat ulang agar engine menampilkan petunjuk masuk.
export default defineNuxtPlugin(() => {
  const cfg = useRuntimeConfig().public
  const token = cfg.apiToken as string
  let reloading = false
  // $fetch yang diimpor otomatis bisa terikat ke ofetch sebelum globalThis.$fetch diganti di bawah;
  // karena itu header token juga dipasang di fetch bawaan untuk permintaan ke engine (hanya bila token di-set).
  if (token) {
    const apiBase = String(cfg.apiBase || '')
    const nativeFetch = window.fetch.bind(window)
    window.fetch = (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      if (apiBase && url.startsWith(apiBase)) {
        const headers = new Headers(init?.headers || (input instanceof Request ? input.headers : undefined))
        if (!headers.has('X-Vibeasik-Token')) headers.set('X-Vibeasik-Token', token)
        return nativeFetch(input, { ...init, headers })
      }
      return nativeFetch(input, init)
    }
  }
  globalThis.$fetch = $fetch.create({
    onRequest({ options }) {
      if (!token) return
      const headers = new Headers(options.headers as HeadersInit | undefined)
      if (!headers.has('X-Vibeasik-Token')) headers.set('X-Vibeasik-Token', token)
      options.headers = headers
    },
    onResponseError({ response }) {
      if (response.status === 401 && !import.meta.dev && !reloading) {
        reloading = true
        window.location.reload()
      }
    }
  }) as typeof $fetch
})
