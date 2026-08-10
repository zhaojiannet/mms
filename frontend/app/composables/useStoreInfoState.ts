import { reactive } from 'vue'
import { useLocalStorage } from '@vueuse/core'

// 店铺信息共享状态，独立成叶子模块（只依赖 vue/@vueuse）：与请求逻辑同文件
// 会构成 auth → storeInfo → api → auth 循环 import，chunk 重排时 TDZ 整站 500

export interface StoreInfo {
  name: string
  slug: string
  login_bg: string
  logo_url: string
}

// login_bg 缓存上次的值做登录页首帧，避免默认图闪换；键按 hostname 区分，
// 多租户共用 origin 时不串背景。写缓存收敛在这一个响应式 ref 上
const bgCache = useLocalStorage(
  `mms.login_bg.${typeof location !== 'undefined' ? location.hostname : ''}`, '')

export const globalStoreInfo = reactive<StoreInfo>({
  name: '',
  slug: '',
  login_bg: bgCache.value,
  logo_url: '',
})

// 登出时由 auth store 调用；login_bg 不清——背景与账号无关，且内存值比缓存新
export function resetStoreInfo() {
  Object.assign(globalStoreInfo, { name: '', slug: '', logo_url: '' })
}

export function rememberBg(bg: string) {
  // 只记真值：空值强转具体主题写入缓存，等于断言商户做过没做过的选择
  if (bg) {
    globalStoreInfo.login_bg = bg
    bgCache.value = bg
  }
}
