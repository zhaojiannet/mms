// useVoidEnabled
// 全局共享的"交易撤销功能"状态：
//   - enabled: 设置 enable_transaction_void 为 true
//   - 且 void_enabled_at 距今 ≤ 10 分钟
// 任何 UI（撤销交易、撤消挂账等）都应该通过此 composable 判断按钮是否显示。
import { useState } from '#imports'

const VOID_WINDOW_SECONDS = 600

interface VoidState {
  fetched: boolean
  enabled: boolean
  enabledAt: number | null   // ms timestamp
}

export function useVoidEnabled() {
  const api = useApi()
  const state = useState<VoidState>('void-enabled', () => ({
    fetched: false,
    enabled: false,
    enabledAt: null,
  }))
  const _now = useState<number>('void-enabled-now', () => Date.now())

  // 全局只启动一个 ticker（用于撤销窗口剩余秒数倒计时显示）
  if (import.meta.client) {
    const started = useState<boolean>('void-enabled-ticker', () => false)
    if (!started.value) {
      started.value = true
      useIntervalFn(() => { _now.value = Date.now() }, 1000)
    }
  }

  const remainingSec = computed(() => {
    if (!state.value.enabledAt) return 0
    const elapsed = Math.floor((_now.value - state.value.enabledAt) / 1000)
    return Math.max(0, VOID_WINDOW_SECONDS - elapsed)
  })

  const canVoid = computed(() => state.value.enabled && remainingSec.value > 0)

  async function refresh() {
    try {
      const enabledRow = await api<{ value: any }>('/api/tenant-settings/enable_transaction_void').catch(() => null)
      state.value.enabled = enabledRow?.value === true
      if (state.value.enabled) {
        const tsRow = await api<{ value: any }>('/api/tenant-settings/void_enabled_at').catch(() => null)
        state.value.enabledAt = tsRow?.value ? new Date(tsRow.value).getTime() : null
      } else {
        state.value.enabledAt = null
      }
    } finally {
      state.value.fetched = true
    }
  }

  async function ensureFetched() {
    if (!state.value.fetched) await refresh()
  }

  return { canVoid, remainingSec, refresh, ensureFetched }
}
