// 运营后台的会话与请求封装：与商户侧 useAuthStore/useApi 完全隔离
// token 独立存储（mms-platform-auth），请求 401 → 跳平台登录页

const STORAGE_KEY = 'mms-platform-auth'

interface PlatformSession {
  token: string
  expires_at: string
  operator: { email: string; name: string }
}

// isPlatformHost 当前访问域是否运营后台（admin.<appDomain>；本地开发放行）
export function isPlatformHost(): boolean {
  if (import.meta.server) return false
  const host = window.location.hostname
  return host.startsWith('admin.') || host === 'localhost' || host === '127.0.0.1'
}

export function platformSession(): PlatformSession | null {
  if (import.meta.server) return null
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const s = JSON.parse(raw) as PlatformSession
    if (!s.token || new Date(s.expires_at).getTime() <= Date.now()) return null
    return s
  } catch {
    return null
  }
}

export function platformLoginSave(s: PlatformSession) {
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(s))
}

export function platformLogout() {
  window.localStorage.removeItem(STORAGE_KEY)
}

// usePlatformApi 平台版 $fetch：带 token，401 清会话回登录页
// baseURL 与商户侧 useApi 同源逻辑：生产同源路径分流留空，开发指到 localhost:8081
export const usePlatformApi = () => {
  const cfg = useRuntimeConfig().public
  const router = useRouter()
  const toast = useToast()
  const nuxtApp = useNuxtApp()

  return $fetch.create({
    baseURL: cfg.apiBase,
    onRequest({ options }) {
      const s = platformSession()
      if (s) {
        const headers = new Headers(options.headers as HeadersInit | undefined)
        headers.set('Authorization', `Bearer ${s.token}`)
        options.headers = headers
      }
    },
    onResponseError({ response }) {
      if (response.status === 401) {
        platformLogout()
        void nuxtApp.runWithContext(() => {
          if (router.currentRoute.value.path !== '/platform/login') {
            return navigateTo('/platform/login')
          }
        })
        return
      }
      if (response.status >= 500) {
        toast.add({ title: '服务暂时不可用', description: '请稍后再试', color: 'error', icon: 'i-lucide-alert-triangle' })
      }
    },
  })
}
