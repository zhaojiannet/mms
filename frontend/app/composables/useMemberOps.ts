// useMemberOps：以 memberId 为参数的会员操作 API 客户端
//
// 抽离的是 endpoint 拼接 + body 类型定义；不抽 form 状态 / Modal UI
// （members/index 与 members/[id] 的 issueCard 表单结构差异大，强行合并易过度设计）
//
// 调用：
//   const ops = useMemberOps(() => memberId)
//   await ops.addPending({ amount: '38', summary: '剪发' })

interface PendingBody {
  amount: string
  summary?: string | null
}

interface SettleBody {
  payment_method_id: string
  card_id?: string | null
}

/**
 * memberIdRef 传函数而非字符串：兼容 route.params 动态变化（如 [id].vue）
 * 与 reactive memberId（如 detailMember.value.id）
 */
export function useMemberOps(memberIdRef: () => string) {
  const api = useApi()
  const base = () => `/api/members/${memberIdRef()}`

  return {
    addPending: (body: PendingBody) =>
      api(`${base()}/pending`, { method: 'POST', body }),

    removePending: (pendingId: string) =>
      api(`${base()}/pending/${pendingId}`, { method: 'DELETE' }),

    settleOne: (pendingId: string, body: SettleBody) =>
      api(`${base()}/pending/${pendingId}/settle`, { method: 'POST', body }),

    settleAll: (body: SettleBody) =>
      api(`${base()}/pending/settle-all`, { method: 'POST', body }),

    /** body 类型留 Record：两份 issueForm schema 不一致（standard/custom mode），调用方负责构造 */
    issueCard: (body: Record<string, unknown>) =>
      api(`${base()}/cards/with-transaction`, { method: 'POST', body }),

    fetchCards: <T = unknown>() =>
      api<{ items: T[] }>(`${base()}/cards`),

    fetchPendings: <T = unknown>() =>
      api<{ items: T[]; total_amount?: string }>(`${base()}/pending`),
  }
}
