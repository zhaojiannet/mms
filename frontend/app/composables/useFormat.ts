// 公共格式化函数：在多个页面 / 组件中复用，避免内联 toFixed / toLocaleString 漂移。
//
// 设计原则：
//   - 输入容错：null / undefined / 空串 → 空字符串（UI 显示空白）
//   - 显式时区：日期使用本地时区（zh-CN），不做 toISOString 偏移

const ZH = 'zh-CN'

/** ISO 字符串 → "YYYY/MM/DD HH:mm"（本地时区） */
export const formatTime = (s?: string | null): string => {
  if (!s) return ''
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleString(ZH, { hour12: false })
}

/** ISO 字符串 → "YYYY/MM/DD"（零填充，本地时区） */
export const formatDate = (s?: string | null): string => {
  if (!s) return ''
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return ''
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}/${m}/${day}`
}

/** Date → "YYYY-MM-DD"（本地时区，避免 toISOString 跨日漂移） */
export const formatDateOnly = (d: Date): string => {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

/** 金额数字 → "¥123.45"（保留两位小数；NaN 兜底为空串） */
export const formatYuan = (v: number | string | null | undefined): string => {
  if (v === null || v === undefined || v === '') return ''
  const n = typeof v === 'string' ? Number(v) : v
  if (Number.isNaN(n)) return ''
  return `¥${n.toFixed(2)}`
}

/** 折扣率 0.85 → "8.5 折"；1.0 → "无折扣"；NaN/空 → "-" */
export const formatDiscountRate = (r: number | string | null | undefined): string => {
  if (r === null || r === undefined || r === '') return '-'
  const n = typeof r === 'string' ? Number(r) : r
  if (Number.isNaN(n)) return '-'
  if (n >= 1) return '无折扣'
  return `${(n * 10).toFixed(1)} 折`
}

// 头像首字/尾字：Intl.Segmenter 按字素簇切分（旗帜、ZWJ 组合 emoji 不碎），
// trim 防前导空格渲染成空白头像。此前 4 处内联 slice(0,1) 各自漂移，收敛于此
const seg = new Intl.Segmenter('zh', { granularity: 'grapheme' })
export const firstGrapheme = (s?: string | null, fallback = 'S'): string => {
  const t = (s || '').trim()
  if (!t) return fallback
  for (const g of seg.segment(t)) return g.segment
  return fallback
}
export const lastGrapheme = (s?: string | null, fallback = '?'): string => {
  const t = (s || '').trim()
  if (!t) return fallback
  let last = fallback
  for (const g of seg.segment(t)) last = g.segment
  return last
}
