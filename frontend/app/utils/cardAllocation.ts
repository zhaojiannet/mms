import { compareCardUseOrder } from './cards'

export interface AllocCard {
  id: string
  balance: string
  final_discount_rate: string
  issued_at: string
  expires_at: string | null
}
export interface CardAllocation { card_id: string; deduct: string }
// 扣款方案与「卡最多能抵的标价」必须同源：余额不足横幅拿 coverable、结算拿 plan，
// 分头计算就会出现「余额比应付多却说余额不足」
export interface CardCoverage { plan: CardAllocation[]; coverable: number }

// 与后端 CreateRequest 的 10 张卡上限一致
export const MAX_CARDS_PER_TX = 10

// 自动分配：按折扣率从低到高用卡，同折扣率按 compareCardUseOrder；
// 不打折的项按面额先扣，打折的项按各卡折扣率折算
export function autoAllocate(cards: AllocCard[], discountable: number, noDiscount: number): CardCoverage {
  const sorted = cards
    .filter(c => parseFloat(c.balance) > 0)
    .sort((a, b) => parseFloat(a.final_discount_rate) - parseFloat(b.final_discount_rate) || compareCardUseOrder(a, b))
  const result: CardAllocation[] = []
  let remainNoDiscount = noDiscount
  let remainDiscountable = discountable
  for (const c of sorted) {
    if (remainNoDiscount <= 0 && remainDiscountable <= 0) break
    let bal = parseFloat(c.balance)
    const rate = parseFloat(c.final_discount_rate)
    let deduct = 0
    if (remainNoDiscount > 0) {
      const take = Math.min(remainNoDiscount, bal)
      deduct += take; bal -= take; remainNoDiscount -= take
    }
    if (remainDiscountable > 0 && bal > 0) {
      const maxCoverable = bal / rate
      const take = Math.min(remainDiscountable, maxCoverable)
      deduct += take * rate; remainDiscountable -= take
    }
    // 半分以下的 deduct 是上一张卡的浮点残渣，push 会产生 ¥0.00 的幽灵分配行
    if (deduct >= 0.005) result.push({ card_id: c.id, deduct: deduct.toFixed(2) })
  }
  const uncovered = Math.max(0, remainNoDiscount) + Math.max(0, remainDiscountable)
  const coverable = Math.max(0, noDiscount + discountable - uncovered)
  // 半分以下的剩余是浮点残渣，不是真缺口：按此判「不足」会把余额恰好够扣的
  // 单（如 33.33×0.85 对 28.33 余额）错推进挂账
  if (uncovered > 0.005) return { plan: [], coverable }
  return { plan: result, coverable }
}

// 改价单：一单一价，改价即本单最终应收，卡只是余额钱包，不按折扣率放大。
// 全程用「分」做整数运算，保证扣卡合计与改价金额分毫不差（后端强校验相等）
export function allocateManualPrice(cards: AllocCard[], manualPrice: number, onlyCardId?: string): CardCoverage {
  let remainC = Math.round(manualPrice * 100)
  const usable = onlyCardId
    ? cards.filter(c => c.id === onlyCardId)
    : [...cards].sort(compareCardUseOrder)
  const plan: CardAllocation[] = []
  let balSumC = 0
  for (const c of usable) {
    const balC = Math.round(parseFloat(c.balance) * 100)
    if (balC <= 0) continue
    balSumC += balC
    const take = Math.min(remainC, balC)
    if (take > 0) { plan.push({ card_id: c.id, deduct: (take / 100).toFixed(2) }); remainC -= take }
  }
  if (remainC > 0) return { plan: [], coverable: Math.min(balSumC, Math.round(manualPrice * 100)) / 100 }
  return { plan, coverable: manualPrice }
}

// 收银员手选一张卡：只用这张卡，付不动时给出它最多能抵多少标价
export function allocateSingleCard(card: AllocCard, discountable: number, noDiscount: number, total: number): CardCoverage {
  const rate = parseFloat(card.final_discount_rate)
  const bal = parseFloat(card.balance)
  // 按分舍入后再与余额比较：未舍入浮点会在半分边界把够扣判成不足
  const need = Math.round((noDiscount + discountable * rate) * 100) / 100
  if (bal >= need) return { plan: [{ card_id: card.id, deduct: need.toFixed(2) }], coverable: total }
  // 先抵不打折的项，剩下的按折扣率放大
  const forNoDisc = Math.min(noDiscount, bal)
  const rest = bal - forNoDisc
  return { plan: [], coverable: Math.min(total, forNoDisc + (rate > 0 ? rest / rate : 0)) }
}

// 挂账「利用余额」：把卡里的钱全部扣掉。超过张数上限时按余额降序取，
// 小额卡多的会员不至于整条路被后端 400
export function drainBalancesPlan(cards: AllocCard[]): CardAllocation[] {
  return [...cards]
    .sort((a, b) => parseFloat(b.balance) - parseFloat(a.balance))
    .slice(0, MAX_CARDS_PER_TX)
    .map(c => ({ card_id: c.id, deduct: parseFloat(c.balance).toFixed(2) }))
}
