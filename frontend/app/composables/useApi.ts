import { useAuthStore } from '~/stores/auth'

// useApi 返回一个带 token + X-Tenant-Slug 自动装配的 $fetch 变种
//
// 错误处理：
//   401 → 自动 logout + 跳 /login（session 失效）
//   403 / 429 / 5xx → 统一 Toast 兜底；handler 仍能通过 try/catch 接住做更细致处理
//   404 / 400 → 不自动 Toast（多用于业务级失败，由调用方决定如何展示）
export const useApi = () => {
  const cfg = useRuntimeConfig().public
  const auth = useAuthStore()
  const router = useRouter()
  const toast = useToast()
  const nuxtApp = useNuxtApp()

  return $fetch.create({
    baseURL: cfg.apiBase,
    onRequest({ options }) {
      const headers = new Headers(options.headers as HeadersInit | undefined)
      headers.set('X-Tenant-Slug', cfg.tenantSlug)
      if (auth.token) {
        headers.set('Authorization', `Bearer ${auth.token}`)
      }
      options.headers = headers
    },
    onResponseError({ response }) {
      const status = response.status
      const msg = (response._data as any)?.message || ''
      if (status === 401) {
        // onResponseError 是 $fetch 异步回调，已脱离 Nuxt 上下文；
        // logout 内部要用 useState/useRuntimeConfig，必须 runWithContext 恢复，
        // 否则抛 "nuxt instance unavailable" 中断回调、跳转永远不执行
        void nuxtApp.runWithContext(() => {
          auth.logout()
          const current = router.currentRoute.value
          // /b/ 是 C 端公开页不踢商户登录页；已在 /login 时避免自跳产生 redirect=/login
          if (current.path !== '/login' && !current.path.startsWith('/b/')) {
            return navigateTo({ path: '/login', query: { redirect: current.fullPath } })
          }
        })
        return
      }
      if (status === 403) {
        toast.add({ title: '权限不足', description: msg || '当前账号无此操作权限', color: 'error', icon: 'i-lucide-shield-x' })
        return
      }
      if (status === 429) {
        toast.add({ title: '请求过于频繁', description: msg || '请稍后再试', color: 'warning', icon: 'i-lucide-hourglass' })
        return
      }
      if (status >= 500) {
        toast.add({ title: '服务暂时不可用', description: '请稍后再试，或联系管理员', color: 'error', icon: 'i-lucide-alert-triangle' })
        return
      }
      // 4xx 业务错误（400/404/409 等）：不全局 Toast，由调用方自己处理
    },
  })
}
