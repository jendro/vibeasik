// Akses Studio dijaga engine (token lokal). Halaman masuk/daftar lama kini halaman Akun vibeasik.
export default defineNuxtRouteMiddleware((to) => {
  if (to.path === '/login' || to.path === '/register') {
    return navigateTo('/account', { replace: true })
  }
})
