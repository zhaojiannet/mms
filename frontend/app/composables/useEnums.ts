// useEnums：业务状态字符串集中管理
//
// 设计原则：
//   - 后端 sqlc 把 PG 字段映射成 Go 的 string，本项目不引 PG enum 类型
//   - 前端用 `as const` + union type 替代散落字符串，编译期保证拼写
//   - 提供中文 label map / Badge color map，避免页面手写 record
//
// 只保留有真实调用方的枚举；新枚举等页面要用时再加，
// 避免"看似单一改点、实际无人使用"的假集中管理。

type BadgeColor = 'primary' | 'info' | 'warning' | 'error' | 'neutral' | 'success'

// ===== 会员卡状态 =====
export const CARD_STATUS = ['active', 'frozen', 'expired', 'depleted'] as const
export type CardStatus = typeof CARD_STATUS[number]
export const CARD_STATUS_LABEL: Record<CardStatus, string> = {
  active:   '正常',
  frozen:   '冻结',
  expired:  '过期',
  depleted: '用尽',
}

// ===== 交易种类 =====
export const TX_KIND = ['sale', 'recharge', 'credit_settlement'] as const
export type TxKind = typeof TX_KIND[number]
export const TX_KIND_LABEL: Record<TxKind, string> = {
  sale:               '消费',
  recharge:           '办卡',
  credit_settlement:  '清账',
}
export const TX_KIND_COLOR: Record<TxKind, BadgeColor> = {
  sale:               'primary',
  recharge:           'info',
  credit_settlement:  'neutral',
}
