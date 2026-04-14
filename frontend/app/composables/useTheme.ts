export interface ThemeOption {
  key: string     // Tailwind/Nuxt UI primary 色名（全部内置色盘，零自定义）
  name: string    // 一字雅名
  tone: string    // 调性描述（去掉"行业绑定"，参考 GlossGenius 让商户按个人品牌气质选）
  hex: string     // 色盘 500 号代表色
}

// 10 色方案基于头部美业/零售 SaaS 真实数据调研修订：
//   - Fresha（美业预约#1）用 Prince 紫 + Limelight 荧光黄，不走传统"行业匹配"
//   - Mindbody 2024 用绿+深炭+亮粉三色
//   - GlossGenius 让商户自选 21 色，官方原话："品牌是商户的艺术 DNA"
//   - Pantone 2026 Color of the Year: Cloud Dancer 纯净白，趋势是极简+个性强调
// 设计取向：不绑行业，给纯粹好色 + 调性文案，让商户按品牌气质自选
export const THEMES: ThemeOption[] = [
  { key: 'teal',    name: '青',    tone: '沉静自然',  hex: '#0d9488' },
  { key: 'rose',    name: '胭',    tone: '温柔生机',  hex: '#e11d48' },
  { key: 'pink',    name: '桃',    tone: '甜美柔软',  hex: '#ec4899' },
  { key: 'indigo',  name: '靛',    tone: '静谧深思',  hex: '#4f46e5' },
  { key: 'emerald', name: '竹',    tone: '清新治愈',  hex: '#10b981' },
  { key: 'blue',    name: '海',    tone: '专注信赖',  hex: '#2563eb' },
  { key: 'orange',  name: '橘',    tone: '活力亲和',  hex: '#f97316' },
  { key: 'amber',   name: '金',    tone: '尊贵传统',  hex: '#f59e0b' },
  { key: 'stone',   name: '摩卡',  tone: '温润复古',  hex: '#78716c' },
  { key: 'zinc',    name: '玄',    tone: '高冷克制',  hex: '#27272a' },
]

const STORAGE_KEY = 'mms.theme'

export const useTheme = () => {
  const appConfig = useAppConfig()

  const current = computed<string>(() => (appConfig.ui.colors as any).primary || 'teal')

  const apply = (key: string) => {
    if (!THEMES.some(t => t.key === key)) return
    updateAppConfig({ ui: { colors: { primary: key } } })
    if (import.meta.client) {
      localStorage.setItem(STORAGE_KEY, key)
    }
  }

  const init = () => {
    if (import.meta.client) {
      const saved = localStorage.getItem(STORAGE_KEY)
      if (saved && THEMES.some(t => t.key === saved)) {
        updateAppConfig({ ui: { colors: { primary: saved } } })
      }
    }
  }

  const find = (key: string) => THEMES.find(t => t.key === key)

  return { current, apply, init, find, themes: THEMES }
}
