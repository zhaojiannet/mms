import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cardsIssuedTodayOfType, compareCardUseOrder } from '~/utils/cards'

describe('compareCardUseOrder', () => {
  it('有到期日的排在不限期的前面', () => {
    const expiring = { expires_at: '2027-01-01T00:00:00+08:00', issued_at: '2026-06-01T10:00:00+08:00' }
    const noExpiry = { expires_at: null, issued_at: '2025-01-01T10:00:00+08:00' }
    expect(compareCardUseOrder(expiring, noExpiry)).toBeLessThan(0)
    expect(compareCardUseOrder(noExpiry, expiring)).toBeGreaterThan(0)
  })

  it('到期日相同按办卡先后', () => {
    const early = { expires_at: null, issued_at: '2026-01-07T10:00:00+08:00' }
    const late = { expires_at: null, issued_at: '2026-11-13T10:00:00+08:00' }
    expect(compareCardUseOrder(early, late)).toBeLessThan(0)
  })
})

describe('cardsIssuedTodayOfType', () => {
  // 用本地时间构造，结果不受运行环境时区影响
  const at = (d: number, h: number, m = 0) => new Date(2026, 10, d, h, m).toISOString()

  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 10, 15, 10, 0))
  })
  afterEach(() => { vi.useRealTimers() })

  it('只算今天办的、同卡型、未撤销的卡', () => {
    const cards = [
      { id: 'today', card_type_id: 'T300', status: 'active', issued_at: at(15, 8) },
      { id: 'today-voided', card_type_id: 'T300', status: 'frozen', issued_at: at(15, 9) },
      { id: 'yesterday', card_type_id: 'T300', status: 'active', issued_at: at(14, 23, 59) },
      { id: 'other-type', card_type_id: 'T500', status: 'active', issued_at: at(15, 8) },
    ]
    expect(cardsIssuedTodayOfType(cards, 'T300').map(c => c.id)).toEqual(['today'])
  })
})
