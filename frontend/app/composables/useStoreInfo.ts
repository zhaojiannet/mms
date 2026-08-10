// 状态在 useStoreInfoState（叶子模块）；这里只挂请求逻辑，防经 useApi 绕回 auth 的循环 import
import { globalStoreInfo, rememberBg, type StoreInfo } from '~/composables/useStoreInfoState'

export type { StoreInfo }

export function useStoreInfo() {
  const api = useApi()

  async function refresh() {
    try {
      const r = await api<StoreInfo>('/api/store/info')
      globalStoreInfo.name     = r.name     || ''
      globalStoreInfo.slug     = r.slug     || ''
      globalStoreInfo.logo_url = r.logo_url || ''
      rememberBg(r.login_bg || '')
    } catch (e) {
      console.warn('useStoreInfo.refresh failed', e)
    }
  }

  function set(partial: Partial<StoreInfo>) {
    const { login_bg, ...rest } = partial
    Object.assign(globalStoreInfo, rest)
    // 设置页改背景走这里：缓存必须跟着写，否则登出后首帧闪回旧背景
    if (login_bg !== undefined) rememberBg(login_bg)
  }

  return { info: globalStoreInfo, refresh, set }
}
