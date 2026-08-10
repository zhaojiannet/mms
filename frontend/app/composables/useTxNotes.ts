// 交易备注的"人话"部分：迁移自老系统的 notes 混有内部技术标记
// （[手动设置时间]、多卡联合支付明细、清挂账结构化标记等），这些信息在新系统
// 都有结构化展示（余额快照 / 挂账关联），流水里只显示剩下的真实备注——
// 典型如「价格调整：烫发198+修面58」，这是加价/改价交易的金额解释，必须让商户看见。
export function humanTxNotes(notes: string | null | undefined): string {
  if (!notes) return ''
  return notes
    .split(' | ')
    .map(seg => seg
      .replace(/\[手动设置时间\]\s*/g, '')
      .replace(/多卡联合支付:.*$/, '')
      .replace(/(?:CARD_)?CLEAR(?:_ALL)?_PENDING:[^|]*/g, '')
      .replace(/ISSUE_CARD:\S*/g, '')
      .replace(/MIGRATED_FROM_PENDING_PAYMENT:?\S*/g, '')
      .trim())
    .filter(Boolean)
    .join(' · ')
}
