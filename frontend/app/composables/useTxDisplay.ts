// 交易流水的折扣 / 加价 / 多卡展示逻辑，POS 今日记录与报表流水共用一份，
// 保证同一笔交易两处显示一致

// 展示函数只读这些字段；两处列表的行类型都是它的超集
export interface TxCardSnapshot {
  card_id: string
  card_type_name: string
  balance_before: string
  balance_after: string
  delta: string
  change_type: string
}
export interface TxDisplayRow {
  kind: string
  total_amount: string
  actual_paid_amount: string
  discount_amount: string
  credit_amount: string
  items_total: string
  card_type_name: string | null
  card_snapshots: TxCardSnapshot[] | null
}

// 挂账登记交易：0 实收 + 关联着一笔挂账
export function isCreditTx(t: TxDisplayRow) { return parseFloat(t.credit_amount || '0') > 0 }

// 加价额：消费单实付高于应付的部分；挂账实付为 0、充值差价是卖卡折扣，均不算
export function surcharge(t: TxDisplayRow): number {
  if (t.kind !== 'sale' || isCreditTx(t)) return 0
  // 原应收优先取明细标价合计：老迁移数据把加价单 total 抬平到实收，原标价
  // 只在明细里；新系统原生加价行 total 即标价，与明细合计一致，两者皆对
  const base = parseFloat(t.items_total) > 0 ? parseFloat(t.items_total) : parseFloat(t.total_amount)
  const d = parseFloat(t.actual_paid_amount) - base
  return d > 0.001 ? d : 0
}

// 划线展示的原价；返回空串表示没有值得划的数字（0 元单加价、抬平后无明细），
// 模板据此整个不渲染划线
export function strikePrice(t: TxDisplayRow): string {
  if (parseFloat(t.discount_amount) > 0) return t.total_amount
  if (surcharge(t) > 0 && parseFloat(t.items_total) > 0) return t.items_total
  return ''
}

// 多卡支付：本笔扣了 ≥2 张卡（按余额流水的 consume 行判断）
export function isMultiCard(t: TxDisplayRow) {
  return (t.card_snapshots || []).filter(s => s.change_type === 'consume').length >= 2
}

// 多卡分卡明细：直接从结构化余额流水组装（老系统靠 notes 文本，这里比它可靠）
export function multiCardText(t: TxDisplayRow): string {
  const parts = (t.card_snapshots || []).filter(s => s.change_type === 'consume')
  if (parts.length < 2) return ''
  return '多卡：' + parts
    .map(s => `${s.card_type_name}¥${(parseFloat(s.balance_before) - parseFloat(s.balance_after)).toFixed(2)}`)
    .join(' + ')
}

// 标称折扣率：从卡显示名（"500元储值卡 7折"）提取；多卡取第一张扣款卡
export function nominalRate(t: TxDisplayRow): number | null {
  const name = t.card_type_name
    || (t.card_snapshots || []).find(s => s.change_type === 'consume')?.card_type_name
  const m = (name || '').match(/([0-9]+(?:\.[0-9])?)\s*折/)
  return m ? parseFloat(m[1]!) / 10 : null
}

// 折数：实付/应收 × 10，整数不带小数（8折），否则一位小数（8.5折）。
// 反推率与卡标称折扣吻合才写「N折 省」，否则只写「减」——反推的伪折扣率不展示
export function discountLabel(t: TxDisplayRow): string {
  const total = parseFloat(t.total_amount)
  const paid = parseFloat(t.actual_paid_amount)
  if (!(total > 0) || paid >= total) return ''
  const rate = paid / total
  const nominal = nominalRate(t)
  if (nominal != null && Math.abs(rate - nominal) < 0.005) {
    const zhe = Math.round(rate * 100) / 10
    return `${Number.isInteger(zhe) ? zhe : zhe.toFixed(1)}折 省`
  }
  return '减'
}
