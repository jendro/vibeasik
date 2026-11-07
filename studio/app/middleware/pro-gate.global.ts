// Halaman fitur berbayar yang belum terbuka (modul tidak ada di build ini, atau paket belum aktif)
// diarahkan ke /upgrade yang menjelaskan fiturnya.
export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server) return
  const feature = proFeatureForPath(to.path)
  if (!feature) return
  const { load, isLocked } = useEdition()
  await load()
  if (isLocked(feature)) {
    return navigateTo({ path: '/upgrade', query: { feature, from: to.fullPath } }, { replace: true })
  }
})
