// 全局路由守卫：未登录访问非 /login 页面 → 跳 /login
// 登录页已登录 → 跳 /
// auth.hydrate() 已在 app.vue 启动时调用一次，Pinia 状态后续路由切换都保留，无需重复读 localStorage
export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()
  const isLoginPage = to.path === '/login'

  if (!auth.isAuthenticated && !isLoginPage) {
    return navigateTo('/login')
  }
  if (auth.isAuthenticated && isLoginPage) {
    return navigateTo('/')
  }
})
