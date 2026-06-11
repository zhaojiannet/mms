// 全局路由守卫：未登录访问非 /login 页面 → 跳 /login（带 redirect，登录后回跳）
// 登录页已登录 → 跳 /
// auth.hydrate() 由 plugins/auth.client.ts 在一切路由中间件之前完成，这里只读内存态
export default defineNuxtRouteMiddleware((to) => {
  // /b/<预约码> 是 C 端公开预约页：顾客无账号、扫商户二维码直达，必须放行
  if (to.path.startsWith('/b/')) return

  const auth = useAuthStore()
  const isLoginPage = to.path === '/login'

  if (!auth.isAuthenticated && !isLoginPage) {
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
  }
  if (auth.isAuthenticated && isLoginPage) {
    return navigateTo('/')
  }
})
