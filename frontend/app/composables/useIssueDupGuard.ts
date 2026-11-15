import { ref } from 'vue'
import { cardsIssuedTodayOfType } from '~/utils/cards'

export interface IssuedCard {
  id: string
  card_type_id: string
  status: string
  issued_at: string
  final_price: string
  balance: string
}

// 办卡前的当天重复提醒：第一次提交若查到当天办过同卡型的卡就拦下并列出来，
// 再次提交放行，换卡型后调用 reset 重新判断。
// 提交时现取会员的卡：页面上的卡列表可能还没加载完，从会员列表行菜单进来时也没加载
export function useIssueDupGuard(fetchCards: () => Promise<{ items: IssuedCard[] }>) {
  const dups = ref<IssuedCard[]>([])
  const prompted = ref(false)

  function reset() {
    dups.value = []
    prompted.value = false
  }

  async function shouldHold(cardTypeId: string): Promise<boolean> {
    if (!cardTypeId || prompted.value) return false
    const { items } = await fetchCards()
    dups.value = cardsIssuedTodayOfType(items, cardTypeId)
    prompted.value = dups.value.length > 0
    return prompted.value
  }

  return { dups, prompted, reset, shouldHold }
}
