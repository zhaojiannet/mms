import { reactive } from 'vue'
import { useLocalStorage } from '@vueuse/core'

// 店铺信息的共享状态。单独成叶子模块（只依赖 vue / @vueuse）：
// auth store 登出要调 resetStoreInfo，而 useStoreInfo 依赖 useApi、useApi 又
// 依赖 auth store——状态若和请求逻辑同住一个文件，就形成 auth → storeInfo →
// api → auth 的循环 import，打包 chunk 重排时会在特定加载顺序下抛 TDZ
// （生产 settings 区直接加载曾因此整站 500）。

export interface StoreInfo {
  name: string
  slug: string
  login_bg: string
  logo_url: string
}

// login_bg 记住上次的值：登录页背景来自 /api/store/info，异步返回前若先渲染
// 默认图，商户自选背景会在首帧后闪换一次。缓存上次的选择做首帧；首次访问
// （无缓存）渲染纯色底，图片就绪后再显示。
//
// 键按 hostname 区分：生产每个商户独立子域天然隔离；dev / self-hosted 多租户
// 共用一个 origin 时，不区分会让 B 租户首帧画出 A 租户选的背景。
// useLocalStorage 是响应式 ref，自带 SSR 守卫与隐私模式容错——写缓存的动作
// 收敛在这一个 ref 上，改背景（set）、拉取（refresh）都不会再漏同步。
const bgCache = useLocalStorage(
  `mms.login_bg.${typeof location !== 'undefined' ? location.hostname : ''}`, '')

export const globalStoreInfo = reactive<StoreInfo>({
  name: '',
  slug: '',
  login_bg: bgCache.value,
  logo_url: '',
})

// 模块级缓存跨账号常驻，登出时由 auth store 调用清空。
// login_bg 不清：同一门店的登录页背景与账号无关，且本会话拉到的内存值
// 比缓存新（缓存可能因隐私模式没写进去），保留内存值而不是重读存储。
export function resetStoreInfo() {
  Object.assign(globalStoreInfo, { name: '', slug: '', logo_url: '' })
}

export function rememberBg(bg: string) {
  // 只记真值：后端返回空（未配置的租户）不能被强转成某个具体主题写进缓存，
  // 否则缓存断言了一个商户从未做过的选择
  if (bg) {
    globalStoreInfo.login_bg = bg
    bgCache.value = bg
  }
}
