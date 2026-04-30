// useEnums：业务状态字符串集中管理
//
// 设计原则：
//   - 后端 sqlc 把 PG 字段映射成 Go 的 string，本项目不引 PG enum 类型
//   - 前端用 `as const` + union type 替代散落字符串，编译期保证拼写
//   - 同时提供中文 label map / Badge color map，避免每个页面手写 record
//
// 当 DB 改值（极少发生）时，本文件是单一改点。

// ===== 会员状态 =====
export const MEMBER_STATUS = ['active', 'inactive'] as const
export type MemberStatus = typeof MEMBER_STATUS[number]
export const MEMBER_STATUS_LABEL: Record<MemberStatus, string> = {
  active: '正常',
  inactive: '已停用',
}

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
type BadgeColor = 'primary' | 'info' | 'warning' | 'error' | 'neutral' | 'success'
export const TX_KIND_COLOR: Record<TxKind, BadgeColor> = {
  sale:               'primary',
  recharge:           'info',
  credit_settlement:  'neutral',
}

// ===== 交易状态 =====
export const TX_STATUS = ['completed', 'voided'] as const
export type TxStatus = typeof TX_STATUS[number]
export const TX_STATUS_LABEL: Record<TxStatus, string> = {
  completed: '已完成',
  voided:    '已撤销',
}

// ===== 预约状态 =====
export const APPT_STATUS = ['pending', 'confirmed', 'completed', 'cancelled', 'no_show'] as const
export type ApptStatus = typeof APPT_STATUS[number]
export const APPT_STATUS_LABEL: Record<ApptStatus, string> = {
  pending:   '待确认',
  confirmed: '已确认',
  completed: '已完成',
  cancelled: '已取消',
  no_show:   '未到店',
}
export const APPT_STATUS_COLOR: Record<ApptStatus, BadgeColor> = {
  pending:   'warning',
  confirmed: 'primary',
  completed: 'success',
  cancelled: 'neutral',
  no_show:   'error',
}

// ===== 用户角色 =====
export const USER_ROLE = ['super_admin', 'admin', 'staff'] as const
export type UserRole = typeof USER_ROLE[number]
// 已有 stores/auth.ts 的 roleLabel/roleHint，不重复定义
