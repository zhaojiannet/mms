import { describe, expect, it } from 'vitest'
import {
  allocateManualPrice,
  allocateSingleCard,
  autoAllocate,
  drainBalancesPlan,
  MAX_CARDS_PER_TX,
  type AllocCard,
} from '~/utils/cardAllocation'

function card(id: string, balance: string, rate: string, issuedAt: string, expiresAt: string | null = null): AllocCard {
  return { id, balance, final_discount_rate: rate, issued_at: issuedAt, expires_at: expiresAt }
}

describe('autoAllocate', () => {
  it('同折扣率先用办得早的卡，一张够付就不动其他卡', () => {
    const cards = [
      card('new-a', '300.00', '0.800', '2026-11-13T14:05:00+08:00'),
      card('small', '2.00', '0.800', '2026-05-04T10:00:00+08:00'),
      card('old', '165.60', '0.800', '2026-01-07T10:00:00+08:00'),
      card('new-b', '125.60', '0.800', '2026-11-13T15:13:00+08:00'),
      card('empty', '0.00', '0.800', '2024-11-12T10:00:00+08:00'),
    ]
    expect(autoAllocate(cards, 108, 0)).toEqual({ plan: [{ card_id: 'old', deduct: '86.40' }], coverable: 108 })
  })

  it('折扣率低的卡优先于办得早的卡', () => {
    const cards = [
      card('early-90', '500.00', '0.900', '2025-01-01T10:00:00+08:00'),
      card('late-70', '500.00', '0.700', '2026-01-01T10:00:00+08:00'),
    ]
    expect(autoAllocate(cards, 100, 0).plan).toEqual([{ card_id: 'late-70', deduct: '70.00' }])
  })

  it('同折扣率时快到期的卡先用，不限期的排后', () => {
    const cards = [
      card('no-expiry', '500.00', '0.800', '2025-01-01T10:00:00+08:00'),
      card('expiring', '500.00', '0.800', '2026-06-01T10:00:00+08:00', '2027-01-01T00:00:00+08:00'),
    ]
    expect(autoAllocate(cards, 100, 0).plan).toEqual([{ card_id: 'expiring', deduct: '80.00' }])
  })

  it('一张卡不够时按顺序组合多张卡，扣卡合计等于折后应付', () => {
    const cards = [
      card('later', '300.00', '0.800', '2026-11-13T10:00:00+08:00'),
      card('earlier', '50.00', '0.800', '2026-01-01T10:00:00+08:00'),
    ]
    expect(autoAllocate(cards, 108, 0)).toEqual({
      plan: [{ card_id: 'earlier', deduct: '50.00' }, { card_id: 'later', deduct: '36.40' }],
      coverable: 108,
    })
  })

  it('不打折的项按面额先扣', () => {
    const cards = [card('a', '100.00', '0.800', '2026-01-01T10:00:00+08:00')]
    expect(autoAllocate(cards, 50, 30).plan).toEqual([{ card_id: 'a', deduct: '70.00' }])
  })

  it('余额不足时不出方案，并给出最多能抵的标价', () => {
    const cards = [card('a', '50.00', '0.800', '2026-01-01T10:00:00+08:00')]
    expect(autoAllocate(cards, 100, 0)).toEqual({ plan: [], coverable: 62.5 })
  })

  it('半分以下的浮点残渣不算余额不足', () => {
    const cards = [card('a', '28.33', '0.850', '2026-01-01T10:00:00+08:00')]
    expect(autoAllocate(cards, 33.33, 0).plan).toEqual([{ card_id: 'a', deduct: '28.33' }])
  })

  it('先旧后新会超过张数上限时，改用余额多的卡少用几张', () => {
    const smalls = Array.from({ length: 11 }, (_, i) =>
      card(`small-${i}`, '5.00', '0.800', `2025-01-${String(i + 1).padStart(2, '0')}T10:00:00+08:00`))
    const cards = [...smalls, card('big', '1000.00', '0.800', '2026-11-13T10:00:00+08:00')]
    expect(autoAllocate(cards, 75, 0)).toEqual({ plan: [{ card_id: 'big', deduct: '60.00' }], coverable: 75 })
  })

  it('张数上限内凑不够时不出方案，能抵金额只算上限内的卡', () => {
    const cards = Array.from({ length: 12 }, (_, i) =>
      card(`c${i}`, '8.00', '0.800', `2025-01-${String(i + 1).padStart(2, '0')}T10:00:00+08:00`))
    // 12 张合计 96 元够付 120×0.8=96，但一单最多 10 张，只能抵 80/0.8=100
    expect(autoAllocate(cards, 120, 0)).toEqual({ plan: [], coverable: 100 })
  })

  it('不改动传入数组的顺序', () => {
    const cards = [
      card('later', '300.00', '0.800', '2026-11-13T10:00:00+08:00'),
      card('earlier', '50.00', '0.800', '2026-01-01T10:00:00+08:00'),
    ]
    autoAllocate(cards, 108, 0)
    expect(cards.map(c => c.id)).toEqual(['later', 'earlier'])
  })
})

describe('allocateManualPrice', () => {
  it('按用卡先后扣，扣卡合计与改价金额分毫不差', () => {
    const cards = [
      card('later', '100.00', '0.800', '2026-11-13T10:00:00+08:00'),
      card('earlier', '30.00', '0.800', '2026-01-01T10:00:00+08:00'),
      card('empty', '0.00', '0.800', '2025-01-01T10:00:00+08:00'),
    ]
    expect(allocateManualPrice(cards, 50.55)).toEqual({
      plan: [{ card_id: 'earlier', deduct: '30.00' }, { card_id: 'later', deduct: '20.55' }],
      coverable: 50.55,
    })
  })

  it('先旧后新会超过张数上限时，改用余额多的卡', () => {
    const smalls = Array.from({ length: 11 }, (_, i) =>
      card(`small-${i}`, '5.00', '0.800', `2025-01-${String(i + 1).padStart(2, '0')}T10:00:00+08:00`))
    const cards = [...smalls, card('big', '1000.00', '0.800', '2026-11-13T10:00:00+08:00')]
    expect(allocateManualPrice(cards, 60)).toEqual({ plan: [{ card_id: 'big', deduct: '60.00' }], coverable: 60 })
  })

  it('手选卡时只用这张，不够就不出方案', () => {
    const cards = [
      card('a', '500.00', '0.800', '2026-01-01T10:00:00+08:00'),
      card('b', '100.00', '0.800', '2026-11-13T10:00:00+08:00'),
    ]
    expect(allocateManualPrice(cards, 120, 'b')).toEqual({ plan: [], coverable: 100 })
  })
})

describe('allocateSingleCard', () => {
  it('付得动时按折扣率算出扣卡额', () => {
    const c = card('a', '100.00', '0.800', '2026-01-01T10:00:00+08:00')
    expect(allocateSingleCard(c, 100, 10, 110)).toEqual({ plan: [{ card_id: 'a', deduct: '90.00' }], coverable: 110 })
  })

  it('付不动时先抵不打折的项，剩余按折扣率放大', () => {
    const c = card('a', '50.00', '0.800', '2026-01-01T10:00:00+08:00')
    expect(allocateSingleCard(c, 100, 10, 110)).toEqual({ plan: [], coverable: 60 })
  })
})

describe('drainBalancesPlan', () => {
  it('超过张数上限时按余额从高到低取，每张扣光', () => {
    const cards = Array.from({ length: 12 }, (_, i) =>
      card(`c${i + 1}`, `${i + 1}.00`, '0.800', '2026-01-01T10:00:00+08:00'))
    const plan = drainBalancesPlan(cards)
    expect(plan).toHaveLength(MAX_CARDS_PER_TX)
    expect(plan[0]).toEqual({ card_id: 'c12', deduct: '12.00' })
    expect(plan.map(p => p.card_id)).not.toContain('c1')
    expect(plan.map(p => p.card_id)).not.toContain('c2')
  })
})
