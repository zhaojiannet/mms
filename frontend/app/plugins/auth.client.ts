import { useAuthStore } from '~/stores/auth'

// 持久登录恢复必须放在 plugin：Nuxt plugins 先于路由中间件执行，
// 而 app.vue 的 setup 晚于初始导航——放那里会让 auth.global.ts
// 在 hydrate 前读到空 token，硬刷新受保护页直接被踢回 /login
export default defineNuxtPlugin(async () => {
  const auth = useAuthStore()
  auth.hydrate()
  // 续签 fire-and-forget：不能 await，否则阻塞首屏导航；失败由 store 内静默处理
  void auth.maybeRefresh()

  // 当前域名是不是运营后台，问后端要答案：/api/platform/* 挂了 RequirePlatformHost，
  // 商户域名一律被拦成 404，平台域名才走到鉴权层回 401。
  // 这个必须 await——路由守卫要靠它决定去哪，晚一步首屏就跳错地方。
  //
  // 判据只认 401：探测不带 token，后端不可能回 2xx，回了说明这个 /api 根本
  // 没转到后端（反代漏配时 Nuxt 自己接走返回首页 HTML），此时按商户域处理，
  // 至少商户界面还是对的。后端不可达、5xx 同理，免得后端一挂商户站集体跳运营登录页。
  const isPlatform = useState<boolean>(PLATFORM_HOST_STATE, () => false)
  try {
    await $fetch('/api/platform/me', { baseURL: useRuntimeConfig().public.apiBase })
    isPlatform.value = false
  } catch (e) {
    isPlatform.value = (e as { response?: { status?: number } })?.response?.status === 401
  }
})
