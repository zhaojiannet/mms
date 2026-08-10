// 登录页背景主题（沿用老系统 QingSi 的 8 张行业摄影图，按行业标注）
export interface LoginBgTheme {
  key: string
  name: string
  hint: string
  image: string // 对应的 /images/login-bg/*.jpg 路径
}

export const LOGIN_BG_THEMES: LoginBgTheme[] = [
  { key: 'beauty',   name: '美业',   hint: '综合美业',     image: '/images/login-bg/login_bg01.jpg' },
  { key: 'facial',   name: '美容',   hint: '皮肤 · 面护',   image: '/images/login-bg/login_bg02.jpg' },
  { key: 'hair',     name: '美发',   hint: '理发 · 造型',   image: '/images/login-bg/login_bg03.jpg' },
  { key: 'nail',     name: '美甲',   hint: '美甲 · 手足护', image: '/images/login-bg/login_bg04.jpg' },
  { key: 'massage',  name: '按摩',   hint: '按摩 · 推拿',   image: '/images/login-bg/login_bg05.jpg' },
  { key: 'yoga',     name: '瑜伽',   hint: '瑜伽 · 健身',   image: '/images/login-bg/login_bg06.jpg' },
  { key: 'training', name: '培训',   hint: '培训 · 教育',   image: '/images/login-bg/login_bg07.jpg' },
  { key: 'pet',      name: '宠物',   hint: '宠物 · 美容',   image: '/images/login-bg/login_bg08.jpg' },
]

// 图片未加载/未选择时的垫底色，同时也是每张主题图 url() 后面的兜底色——
// 必须是同一个值，否则首帧纯色和图片垫底色会是两种颜色
const FALLBACK_BG = '#44403c'

export function getThemeByKey(key: string | null | undefined): LoginBgTheme {
  return LOGIN_BG_THEMES.find(t => t.key === key) || LOGIN_BG_THEMES[0]!
}

// 把老配置 key（classic 等）或行业 key 都能兼容
export function bgStyle(theme: LoginBgTheme): string {
  return `url('${theme.image}') center/cover, ${FALLBACK_BG}`
}

// login_bg 的展示入口：key 为空 = 商户选择未知（接口没回或没配置），
// 给纯色底而不是默认主题图——猜一张图再换会闪，纯色到正确图不会。
// 消费 login_bg 的地方一律走这里，别直接 getThemeByKey（它把空值解析成默认主题）。
export function loginBgStyle(key: string | null | undefined): string {
  return key ? bgStyle(getThemeByKey(key)) : FALLBACK_BG
}
