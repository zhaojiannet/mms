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
