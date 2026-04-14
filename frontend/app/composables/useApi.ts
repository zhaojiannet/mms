import { useAuthStore } from '~/stores/auth'

// useApi 返回一个带 token + X-Tenant-Slug 自动装配的 $fetch 变种
// 401 自动 logout 并跳登录
export const useApi = () => {
  const cfg = useRuntimeConfig().public
  const auth = useAuthStore()
  const router = useRouter()

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
      if (response.status === 401) {
        auth.logout()
        if (typeof window !== 'undefined') {
          router.push('/login')
        }
      }
    },
  })
}
