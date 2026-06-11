// 页面级守卫：管理员及以上可访问（super_admin + admin）
// staff → 跳转首页
export default defineNuxtRouteMiddleware(() => {
  // hydrate 由 plugins/auth.client.ts 在中间件之前完成，无需重复
  const auth = useAuthStore()
  const r = auth.user?.role
  if (r !== 'super_admin' && r !== 'admin') {
    return navigateTo('/')
  }
})
