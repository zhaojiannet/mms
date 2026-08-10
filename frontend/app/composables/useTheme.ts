export interface ThemeOption {
  key: string     // Tailwind/Nuxt UI primary 色名（全部内置色盘，零自定义）
  name: string    // 一字雅名
  tone: string    // 调性描述（去掉"行业绑定"，参考 GlossGenius 让商户按个人品牌气质选）
  hex: string     // 色盘 500 号代表色
}

// 色盘按商户品牌气质自选（参考 GlossGenius，不绑行业）。
// 原 10 色实测后撤掉 3 个功能性退化的：
//   - 金 amber：与 warning 语义色同源同色，「挂账」「补录」等警示与品牌色无差，
//     且 solid 按钮白字对比 ~2.2:1 远低于可读线
//   - 摩卡 stone / 玄 zinc：与中性色盘同源，侧栏激活、KPI 数字、结算按钮
//     全部失去强调身份，暗色下 solid 按钮与禁用态无法区分
// init 对已存被撤色的商户自动回退默认青。再增色先过一遍
// 明/暗 × warning 撞色 × solid 对比的组合检查
export const THEMES: ThemeOption[] = [
  { key: 'teal',    name: '青',    tone: '沉静自然',  hex: '#0d9488' },
  { key: 'rose',    name: '胭',    tone: '温柔生机',  hex: '#e11d48' },
  { key: 'pink',    name: '桃',    tone: '甜美柔软',  hex: '#ec4899' },
  { key: 'indigo',  name: '靛',    tone: '静谧深思',  hex: '#4f46e5' },
  { key: 'emerald', name: '竹',    tone: '清新治愈',  hex: '#10b981' },
  { key: 'blue',    name: '海',    tone: '专注信赖',  hex: '#2563eb' },
  { key: 'orange',  name: '橘',    tone: '活力亲和',  hex: '#f97316' },
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

// 导航激活态「白胶囊」——商户侧栏与运营后台顶栏共用，改样式两行同步改。
// 两份必须都是完整类名字面量（Tailwind 官方规则：类名只从源码文本提取，
// 运行时拼接的类不会生成 CSS）；侧栏经 NuxtLink active-class 注入需 ! 提权
export const navActiveClass = 'bg-white dark:bg-stone-800 shadow-xs dark:ring-1 dark:ring-stone-700 text-primary-700 dark:text-primary-300 font-medium'
export const navActiveClassImportant = 'bg-white! dark:bg-stone-800! shadow-xs! dark:ring-1! dark:ring-stone-700! text-primary-700! dark:text-primary-300! font-medium'
