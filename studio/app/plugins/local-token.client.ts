// Token akses lokal engine.
// - Studio produksi disajikan engine sendiri: cookie HttpOnly dari /auth ikut terkirim otomatis.
// - Dev server Nuxt (origin lain): set NUXT_PUBLIC_API_TOKEN=$(vibeasik token) agar header dikirim.
// Bila engine menjawab 401 (token diganti/cookie habis), muat ulang agar engine menampilkan petunjuk masuk.
export default defineNuxtPlugin(() => {
  const token = useRuntimeConfig().public.apiToken as string
  let reloading = false
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
