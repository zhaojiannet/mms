<template>
  <div>
    <!-- Top bar -->
    <header class="h-14 flex items-center justify-between gap-4 px-6 bg-stone-50/80 dark:bg-stone-950/80 backdrop-blur-sm border-b border-stone-200/60 dark:border-stone-800 sticky top-0 z-10">

      <!-- 左：标题 + 日期 + 祝福语（间距更宽，祝福语淡入过渡） -->
      <div class="flex items-center gap-6 min-w-0 flex-1">
        <div class="flex items-center gap-2 shrink-0">
          <UIcon name="i-lucide-calculator" class="size-4 text-stone-400" />
          <span class="text-base font-medium text-stone-700 dark:text-stone-300">收银</span>
          <span class="text-sm text-stone-400">·</span>
          <span class="text-sm text-stone-500">{{ today }}</span>
        </div>
        <Transition
          enter-active-class="transition-all duration-500 ease-out"
          enter-from-class="opacity-0 translate-y-1"
          enter-to-class="opacity-100 translate-y-0"
          leave-active-class="transition-all duration-300 ease-in"
          leave-from-class="opacity-100"
          leave-to-class="opacity-0"
          mode="out-in"
        >
          <span :key="greeting" class="hidden md:inline-flex items-center text-sm text-primary-600/80 dark:text-primary-400/80 italic select-none">
            {{ greeting }}
          </span>
        </Transition>
      </div>

      <!-- 右：KPI 紧贴通知按钮；md 显示今日+挂账，lg 显示全部 4 项 -->
      <div class="flex items-center gap-4 shrink-0">
        <div class="hidden md:flex items-center gap-3 text-sm tabular-nums">
          <div class="flex items-baseline gap-1.5">
            <span class="text-xs text-stone-500">今日</span>
            <span class="font-semibold text-stone-900 dark:text-stone-100">¥{{ kpi.todaySaleRevenue }}</span>
            <span class="text-xs text-stone-400">/ {{ kpi.todayCount }} 笔</span>
            <span v-if="parseFloat(kpi.todayCreditCharged) > 0" class="text-xs text-warning-600">+挂{{ kpi.todayCreditCharged }}</span>
          </div>
          <span class="hidden lg:inline text-stone-300 dark:text-stone-700">·</span>
          <div class="hidden lg:flex items-baseline gap-1.5">
            <span class="text-xs text-stone-500">本月</span>
            <span class="font-semibold text-stone-900 dark:text-stone-100">¥{{ kpi.monthRevenue }}</span>
          </div>
          <span class="hidden lg:inline text-stone-300 dark:text-stone-700">·</span>
          <div class="hidden lg:flex items-baseline gap-1.5">
            <span class="text-xs text-stone-500">卡池</span>
            <span class="font-semibold text-primary-600 dark:text-primary-400">¥{{ kpi.cardPool }}</span>
          </div>
          <template v-if="parseFloat(kpi.pending) > 0">
            <span class="text-stone-300 dark:text-stone-700">·</span>
            <div class="flex items-baseline gap-1.5">
              <span class="text-xs text-stone-500">挂账</span>
              <span class="font-semibold text-warning-600 dark:text-warning-400">¥{{ kpi.pending }}</span>
            </div>
          </template>
        </div>

        <UPopover :content="{ side: 'bottom', align: 'end', sideOffset: 8 }">
          <div class="relative">
            <UButton
              icon="i-lucide-bell"
              variant="ghost"
              color="neutral"
              size="md"
              square
              aria-label="通知"
              class="bg-stone-100! dark:bg-stone-800! hover:bg-stone-200/70! dark:hover:bg-stone-700! transition-colors duration-150"
            />
            <span
              v-if="hasUnread"
              class="absolute top-1 right-1 size-2 rounded-full bg-error-500 ring-2 ring-stone-50 dark:ring-stone-950 pointer-events-none"
            />
          </div>
          <template #content>
            <div class="w-96 max-h-112 flex flex-col">
              <div class="flex items-center justify-between px-4 py-3 border-b border-stone-200/70 dark:border-stone-800">
                <div class="flex items-center gap-2">
                  <span class="text-sm font-semibold">通知</span>
                  <UBadge v-if="hasUnread" :label="unreadCount > 99 ? '99+' : String(unreadCount)" color="error" variant="soft" size="sm" />
                </div>
                <UButton v-if="hasUnread" label="全部已读" variant="ghost" color="neutral" size="xs" @click="markAllRead" />
              </div>
              <div class="flex-1 overflow-y-auto divide-y divide-stone-200/60 dark:divide-stone-800">
                <div v-if="!notifications.length" class="px-6 py-16 text-center">
                  <UIcon name="i-lucide-bell-off" class="size-8 text-stone-300 dark:text-stone-600 mx-auto" />
                  <p class="mt-3 text-sm text-stone-500 dark:text-stone-400">暂无通知</p>
                </div>
                <NuxtLink
                  v-for="n in notifications" :key="n.type + n.id"
                  :to="n.link"
                  class="flex items-start gap-3 px-4 py-3 hover:bg-stone-50/80 dark:hover:bg-stone-800/40 transition-colors cursor-pointer"
                  @click="onNotiClick(n)"
                >
                  <UIcon :name="iconFor(n.type)" :class="['size-4 shrink-0 mt-0.5', colorFor(n.type)]" />
                  <div class="flex-1 min-w-0">
                    <div class="text-sm font-medium text-stone-900 dark:text-stone-100 truncate">{{ n.title }}</div>
                    <div class="text-xs text-stone-500 dark:text-stone-400 mt-0.5 line-clamp-2">{{ n.summary }}</div>
                  </div>
                </NuxtLink>
              </div>
            </div>
          </template>
        </UPopover>
      </div>
    </header>

    <!-- 主体：POS 收银工作区 -->
    <div class="px-6 py-5 max-w-7xl mx-auto">
      <PosWorkbench />
    </div>
  </div>
</template>

<script setup lang="ts">
useHead({ title: '收银' })

const api = useApi()

const today = new Date().toLocaleDateString('zh-CN', { month: 'long', day: 'numeric', weekday: 'long' })

// KPI：商户最关注的 4 项数据
//   1. 今日营收 + 笔数（最高频，开机第一眼）
//   2. 本月累计营收（看月度业绩）
//   3. 会员卡池余额（潜在未来收入 = 已收的钱还能消多少）
//   4. 未结挂账（仅 > 0 时显示，红色提醒）
const kpi = reactive({
  todayRevenue: '0',
  todaySaleRevenue: '0',
  todayCreditCharged: '0',
  todayCount: 0,
  monthRevenue: '0',
  cardPool: '0',
  pending: '0',
})

async function loadKpi() {
  try {
    // 本地日期（非 toISOString 的 UTC 日期）：凌晨 0-8 点 UTC 日期是昨天，会查错业务日
    const now = new Date()
    const todayStr = formatDateOnly(now)
    const monthStart = formatDateOnly(new Date(now.getFullYear(), now.getMonth(), 1))

    const [today, month, pending, cardPool] = await Promise.all([
      api<any>(`/api/reports/business?start_date=${todayStr}&end_date=${todayStr}`).catch(() => null),
      api<any>(`/api/reports/business?start_date=${monthStart}&end_date=${todayStr}`).catch(() => null),
      api<any>('/api/reports/pending-stats?limit=1').catch(() => null),
      api<{ total: string }>('/api/reports/card-pool').catch(() => null),
    ])

    if (today) {
      kpi.todayRevenue = today.total_revenue || '0'
      kpi.todaySaleRevenue = today.sale_revenue || '0'
      kpi.todayCreditCharged = today.credit_charged || '0'
      kpi.todayCount = today.customer_count || 0
    }
    if (month) {
      kpi.monthRevenue = month.total_revenue || '0'
    }
    if (pending) kpi.pending = pending.summary?.total_amount || '0'
    if (cardPool) kpi.cardPool = cardPool.total || '0'
  } catch (e) {
    console.warn('loadKpi failed', e)
  }
}
onMounted(loadKpi)

// 祝福语：从 useGreeting composable 取（可被 PosWorkbench 结算成功时刷新）
const { greeting, refresh: refreshGreeting } = useGreeting()
// 每 30 分钟自动换一次（一个客人服务周期，避免久看同一句腻）
useIntervalFn(refreshGreeting, 30 * 60 * 1000)

// 通知：3 类合并（生日 / 预约 / 系统公告）
interface NotiItem { id: string; type: 'birthday' | 'appointment' | 'announcement'; title: string; summary: string; link: string; occurred_at: string; read: boolean }
const notifications = ref<NotiItem[]>([])
const unreadCount = ref(0)
const hasUnread = computed(() => unreadCount.value > 0)

async function loadNotifications() {
  try {
    const r = await api<{ items: NotiItem[]; unread_count: number }>('/api/notifications')
    notifications.value = r.items || []
    unreadCount.value = r.unread_count || 0
  } catch (e) {
    console.warn('loadNotifications failed', e)
  }
}
async function markAllRead() {
  try {
    await api('/api/notifications/read-all', { method: 'POST' })
    await loadNotifications()
  } catch (e) {
    console.warn('markAllRead failed', e)
  }
}
async function onNotiClick(n: NotiItem) {
  if (n.type === 'announcement') {
    try { await api(`/api/notifications/${n.id}/read`, { method: 'POST' }) } catch {}
    await loadNotifications()
  }
}
function iconFor(t: string) {
  return ({ birthday: 'i-lucide-cake', appointment: 'i-lucide-calendar-clock', announcement: 'i-lucide-megaphone' } as Record<string, string>)[t] || 'i-lucide-bell'
}
function colorFor(t: string) {
  return ({ birthday: 'text-pink-500', appointment: 'text-primary-500', announcement: 'text-warning-500' } as Record<string, string>)[t] || 'text-stone-400'
}
onMounted(loadNotifications)
// 60 秒静默轮询
useIntervalFn(loadNotifications, 60 * 1000)
</script>
