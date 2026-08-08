// 运营后台的会话与请求封装：与商户侧 useAuthStore/useApi 完全隔离
// token 独立存储（mms-platform-auth），请求 401 → 跳平台登录页

const STORAGE_KEY = 'mms-platform-auth'

interface PlatformSession {
  token: string
  expires_at: string
  operator: { email: string; name: string }
}

// isPlatformHost 当前访问域是否运营后台
// 与后端 RequirePlatformHost 保持同一判定来源：优先 NUXT_PUBLIC_PLATFORM_HOST
// 指定的完整主机名（逗号分隔），否则退回 admin. 前缀；本地开发放行
export function isPlatformHost(): boolean {
  if (import.meta.server) return false
  const host = window.location.hostname.toLowerCase()
  if (host === 'localhost' || host === '127.0.0.1') return true

  const configured = String(useRuntimeConfig().public.platformHost || '')
    .split(',')
    .map(h => h.trim().toLowerCase())
    .filter(Boolean)
  if (configured.length > 0) return configured.includes(host)

  return host.startsWith('admin.')
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
      const msg = (response._data as any)?.message || ''
      // 403 同样要清会话回登录页：操作员被停用时 RequireOperator 返回 403，
      // 只处理 401 会让页面停在"已登录"状态、空态里看不到任何原因
      if (response.status === 401 || response.status === 403) {
        platformLogout()
        void nuxtApp.runWithContext(() => {
          toast.add({
            title: response.status === 403 ? '账号不可用' : '会话已失效',
            description: msg || '请重新登录',
            color: 'error',
            icon: 'i-lucide-shield-x',
          })
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
