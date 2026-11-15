import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useIssueDupGuard, type IssuedCard } from '~/composables/useIssueDupGuard'

const today = (h: number) => new Date(2026, 10, 15, h).toISOString()
const issued = (id: string, typeId: string): IssuedCard =>
  ({ id, card_type_id: typeId, status: 'active', issued_at: today(8), final_price: '300.00', balance: '300.00' })

describe('useIssueDupGuard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 10, 15, 10, 0))
  })
  afterEach(() => { vi.useRealTimers() })

  it('提交时现取会员的卡，当天办过同卡型就拦下一次，再次提交放行', async () => {
    const fetchCards = vi.fn(async () => ({ items: [issued('a', 'T300')] }))
    const guard = useIssueDupGuard(fetchCards)

    expect(await guard.shouldHold('T300')).toBe(true)
    expect(guard.dups.value.map(c => c.id)).toEqual(['a'])
    expect(guard.prompted.value).toBe(true)

    expect(await guard.shouldHold('T300')).toBe(false)
    expect(fetchCards).toHaveBeenCalledTimes(1)
  })

  it('换卡型 reset 后重新判断', async () => {
    const fetchCards = vi.fn(async () => ({ items: [issued('a', 'T300')] }))
    const guard = useIssueDupGuard(fetchCards)
    await guard.shouldHold('T300')
    guard.reset()
    expect(guard.prompted.value).toBe(false)
    expect(await guard.shouldHold('T300')).toBe(true)
  })

  it('没有同卡型的当天办卡记录或没选卡型时不拦', async () => {
    const fetchCards = vi.fn(async () => ({ items: [issued('a', 'T500')] }))
    const guard = useIssueDupGuard(fetchCards)
    expect(await guard.shouldHold('T300')).toBe(false)
    expect(await guard.shouldHold('')).toBe(false)
    expect(guard.prompted.value).toBe(false)
  })
})
