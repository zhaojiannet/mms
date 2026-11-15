import { formatDate } from '~/composables/useFormat'

// 过期判定的唯一前端实现。边界对齐后端 cardExpired 的 Before（严格小于）。
// 注意用的是本机时钟：工作站时钟偏差会让判定与后端不一致（快则有效卡从
// 列表消失，慢则过期卡可选、提交时被后端 400 拒），根治要后端直接下发标志
export function isCardExpired(c: { expires_at: string | null }): boolean {
  return !!c.expires_at && new Date(c.expires_at).getTime() < Date.now()
}

// 可用卡判据的唯一前端实现：收银分配 / 清账选卡共用。
// 状态口径与后端定价、扣卡、清账一致——改这里必须同步后端语义
export function isUsableCard(c: { status: string; balance: string; expires_at: string | null }): boolean {
  return c.status === 'active' && parseFloat(c.balance) > 0 && !isCardExpired(c)
}

// 同等条件下的用卡先后：先到期的先用，不限期的排最后，再按办卡先后。
// 余额多的先用会让零头卡一直剩着，还会让刚办的卡被下一单动用，
// 办重了就撤不掉（撤销办卡要求余额等于面值）
export function compareCardUseOrder(
  a: { expires_at: string | null; issued_at: string },
  b: { expires_at: string | null; issued_at: string },
): number {
  const exp = (c: { expires_at: string | null }) => c.expires_at ? new Date(c.expires_at).getTime() : Infinity
  const ea = exp(a), eb = exp(b)
  if (ea !== eb) return ea < eb ? -1 : 1
  return new Date(a.issued_at).getTime() - new Date(b.issued_at).getTime()
}

// 会员今天办过的同卡型的卡，办卡前拿来提醒是否重复办理。
// 撤销过的办卡会冻结卡，不算在内
export function cardsIssuedTodayOfType<T extends { card_type_id: string; status: string; issued_at: string }>(
  cards: T[], cardTypeId: string,
): T[] {
  const today = formatDate(new Date().toISOString())
  return cards.filter(c => c.card_type_id === cardTypeId && c.status === 'active' && formatDate(c.issued_at) === today)
}
