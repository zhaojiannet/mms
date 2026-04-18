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

export function getThemeByKey(key: string | null | undefined): LoginBgTheme {
  return LOGIN_BG_THEMES.find(t => t.key === key) || LOGIN_BG_THEMES[0]!
}

// 把老配置 key（classic 等）或行业 key 都能兼容
export function bgStyle(theme: LoginBgTheme): string {
  return `url('${theme.image}') center/cover, #44403c`
}
