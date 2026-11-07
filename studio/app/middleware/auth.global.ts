export default defineNuxtRouteMiddleware((to) => {
  const { isAuthenticated } = useAuth()

  const publicRoutes = ['/', '/download', '/login', '/register']
  const isPublic = publicRoutes.includes(to.path) || Boolean(to.meta.public)

  // 1. If user is logged in and tries to access login or register, redirect to cockpit
  if (isAuthenticated.value && (to.path === '/login' || to.path === '/register')) {
    return navigateTo('/cockpit')
  }

  // 2. If trying to access admin/workspace without authentication, redirect to login
  if (!isPublic && !isAuthenticated.value) {
    return navigateTo({
      path: '/login',
      query: {
        redirect: to.fullPath
      }
    })
  }
})
