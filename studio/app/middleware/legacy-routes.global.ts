// Halaman obrolan/rapat lama dilebur ke Ruang Klien; tautan lama diarahkan ke sana.
const MERGED: Record<string, string> = {
  '/meetings': '/client-room',
  '/planner': '/client-room'
}

export default defineNuxtRouteMiddleware((to) => {
  const target = MERGED[to.path]
  if (target) return navigateTo(target, { replace: true })
})
