// 全站字号三档：通过动态修改 html 根字号触发 Tailwind v4 rem-based utility 整体缩放
// 小 14px / 中 16px（Tailwind 默认） / 大 18px
export interface FontSizeOption {
  key: 'small' | 'medium' | 'large'
  label: string
  px: string
}

export const FONT_SIZE_OPTIONS: FontSizeOption[] = [
  { key: 'small',  label: '小', px: '14px' },
  { key: 'medium', label: '中', px: '16px' },
  { key: 'large',  label: '大', px: '18px' },
]

export type FontSizeKey = FontSizeOption['key']

const STORAGE_KEY = 'mms.font-size'

export const useFontSize = () => {
  const current = useState<FontSizeKey>('font-size', () => 'medium')

  const apply = (key: FontSizeKey) => {
    const opt = FONT_SIZE_OPTIONS.find(o => o.key === key)
    if (!opt) return
    current.value = key
    if (import.meta.client) {
      document.documentElement.style.fontSize = opt.px
      localStorage.setItem(STORAGE_KEY, key)
    }
  }

  const init = () => {
    if (import.meta.client) {
      const saved = localStorage.getItem(STORAGE_KEY) as FontSizeKey | null
      const opt = FONT_SIZE_OPTIONS.find(o => o.key === saved)
      if (opt) {
        current.value = opt.key
        document.documentElement.style.fontSize = opt.px
      }
    }
  }

  return { current, apply, init, options: FONT_SIZE_OPTIONS }
}
