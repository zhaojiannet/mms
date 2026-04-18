// 页面级守卫：仅超级管理员可访问
// 非超级管理员 → 跳转首页
export default defineNuxtRouteMiddleware(() => {
  // auth.global.ts 已 hydrate，无需重复
  const auth = useAuthStore()
  if (auth.user?.role !== 'super_admin') {
    return navigateTo('/')
  }
})
