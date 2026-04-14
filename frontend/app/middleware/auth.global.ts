// 全局路由守卫：未登录访问非 /login 页面 → 跳 /login
// 登录页已登录 → 跳 /
export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()
  auth.hydrate()

  const isLoginPage = to.path === '/login'

  if (!auth.isAuthenticated && !isLoginPage) {
    return navigateTo('/login')
  }
  if (auth.isAuthenticated && isLoginPage) {
    return navigateTo('/')
  }
})
