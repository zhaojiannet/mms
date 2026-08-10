import { reactive } from 'vue'

export interface StoreInfo {
  name: string
  slug: string
  login_bg: string
  logo_url: string
}

// login_bg 记住上次的值：登录页背景来自 /api/store/info，异步返回前若先渲染
// 默认图，商户自选背景会在首帧后闪换一次。缓存上次的选择做首帧，日常访问零闪烁；
// 首次访问（无缓存）留空，页面渲染纯色底、接口返回后随过渡渐入，不闪默认图。
const BG_CACHE_KEY = 'mms.login_bg'

function cachedBg(): string {
  try {
    return window.localStorage.getItem(BG_CACHE_KEY) || ''
  } catch {
    return ''
  }
}

const globalInfo = reactive<StoreInfo>({
  name: '',
  slug: '',
  login_bg: cachedBg(),
  logo_url: '',
})

// 模块级缓存跨账号常驻，登出时由 auth store 调用清空（login_bg 保留：
// 同一门店的登录页背景与账号无关，清了反而让下次登录又闪一次）
export function resetStoreInfo() {
  Object.assign(globalInfo, { name: '', slug: '', login_bg: cachedBg(), logo_url: '' })
}

export function useStoreInfo() {
  const api = useApi()

  async function refresh() {
    try {
      const r = await api<StoreInfo>('/api/store/info')
      globalInfo.name     = r.name     || ''
      globalInfo.slug     = r.slug     || ''
      globalInfo.login_bg = r.login_bg || 'beauty'
      globalInfo.logo_url = r.logo_url || ''
      try {
        window.localStorage.setItem(BG_CACHE_KEY, globalInfo.login_bg)
      } catch { /* 隐私模式等存不了就算了，只是回到会闪一次的行为 */ }
    } catch (e) {
      console.warn('useStoreInfo.refresh failed', e)
    }
  }

  function set(partial: Partial<StoreInfo>) {
    Object.assign(globalInfo, partial)
  }

  return { info: globalInfo, refresh, set }
}
