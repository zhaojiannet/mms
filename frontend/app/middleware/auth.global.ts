// 全局路由守卫：未登录访问非 /login 页面 → 跳 /login（带 redirect，登录后回跳）
// 登录页已登录 → 跳 /
// auth.hydrate() 由 plugins/auth.client.ts 在一切路由中间件之前完成，这里只读内存态
export default defineNuxtRouteMiddleware((to) => {
  // /b/<预约码> 是 C 端公开预约页：顾客无账号、扫商户二维码直达，必须放行
  if (to.path.startsWith('/b/')) return
  // /apply 商户开通申请：主站公开表单，无账号
  if (to.path === '/apply') return

  // 运营后台自成体系：平台 token 守卫，与商户会话互不相干
  if (to.path.startsWith('/platform')) {
    const loggedIn = platformSession() !== null
    if (!loggedIn && to.path !== '/platform/login') {
      return navigateTo('/platform/login')
    }
    if (loggedIn && to.path === '/platform/login') {
      return navigateTo('/platform')
    }
    return
  }
  // admin 子域直接进运营后台，不展示商户界面
  if (isPlatformHost() && window.location.hostname.startsWith('admin.')) {
    return navigateTo('/platform')
  }

  const auth = useAuthStore()
  const isLoginPage = to.path === '/login'

  if (!auth.isAuthenticated && !isLoginPage) {
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
  }
  if (auth.isAuthenticated && isLoginPage) {
    return navigateTo('/')
  }
})
