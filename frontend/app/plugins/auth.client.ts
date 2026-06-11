import { useAuthStore } from '~/stores/auth'

// 持久登录恢复必须放在 plugin：Nuxt plugins 先于路由中间件执行，
// 而 app.vue 的 setup 晚于初始导航——放那里会让 auth.global.ts
// 在 hydrate 前读到空 token，硬刷新受保护页直接被踢回 /login
export default defineNuxtPlugin(() => {
  const auth = useAuthStore()
  auth.hydrate()
  // 续签 fire-and-forget：不能 await，否则阻塞首屏导航；失败由 store 内静默处理
  void auth.maybeRefresh()
})
