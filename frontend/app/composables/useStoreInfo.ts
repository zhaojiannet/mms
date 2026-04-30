import { reactive } from 'vue'

export interface StoreInfo {
  name: string
  slug: string
  login_bg: string
  logo_url: string
}

const globalInfo = reactive<StoreInfo>({
  name: '',
  slug: '',
  login_bg: 'beauty',
  logo_url: '',
})

export function useStoreInfo() {
  const api = useApi()

  async function refresh() {
    try {
      const r = await api<StoreInfo>('/api/store/info')
      globalInfo.name     = r.name     || ''
      globalInfo.slug     = r.slug     || ''
      globalInfo.login_bg = r.login_bg || 'beauty'
      globalInfo.logo_url = r.logo_url || ''
    } catch (e) {
      console.warn('useStoreInfo.refresh failed', e)
    }
  }

  function set(partial: Partial<StoreInfo>) {
    Object.assign(globalInfo, partial)
  }

  return { info: globalInfo, refresh, set }
}
