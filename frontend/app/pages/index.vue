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

      <!-- 右：KPI 紧贴通知按钮 -->
      <div class="flex items-center gap-4 shrink-0">
        <div class="hidden lg:flex items-center gap-3 text-sm tabular-nums">
          <div class="flex items-baseline gap-1.5">
            <span class="text-xs text-stone-500">今日</span>
            <span class="font-semibold text-stone-900 dark:text-stone-100">¥{{ kpi.todayRevenue }}</span>
            <span class="text-xs text-stone-400">/ {{ kpi.todayCount }} 笔</span>
          </div>
          <span class="text-stone-300 dark:text-stone-700">·</span>
          <div class="flex items-baseline gap-1.5">
            <span class="text-xs text-stone-500">本月</span>
            <span class="font-semibold text-stone-900 dark:text-stone-100">¥{{ kpi.monthRevenue }}</span>
          </div>
          <span class="text-stone-300 dark:text-stone-700">·</span>
          <div class="flex items-baseline gap-1.5">
            <span class="text-xs text-stone-500">卡池</span>
            <span class="font-semibold text-primary-600 dark:text-primary-400">¥{{ kpi.cardPool }}</span>
          </div>
          <template v-if="parseFloat(kpi.pending) > 0">
            <span class="text-stone-300 dark:text-stone-700">·</span>
            <div class="flex items-baseline gap-1.5">
              <span class="text-xs text-stone-500">挂账</span>
              <span class="font-semibold text-error-600">¥{{ kpi.pending }}</span>
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
              class="!bg-stone-100 dark:!bg-stone-800 hover:!bg-stone-200/70 dark:hover:!bg-stone-700 transition-colors duration-150"
            />
            <span
              v-if="hasUnread"
              class="absolute top-1 right-1 size-2 rounded-full bg-red-500 ring-2 ring-stone-50 dark:ring-stone-950 pointer-events-none"
            />
          </div>
          <template #content>
            <div class="w-96 max-h-[28rem] flex flex-col">
              <div class="flex items-center justify-between px-4 py-3 border-b border-stone-200/70 dark:border-stone-800">
                <div class="flex items-center gap-2">
                  <span class="text-sm font-semibold">通知</span>
                  <UBadge v-if="hasUnread" :label="String(unreadCount)" color="error" variant="soft" size="sm" />
                </div>
                <UButton v-if="hasUnread" label="全部标记为已读" variant="ghost" color="neutral" size="xs" @click="markAllRead" />
              </div>
              <div class="flex-1 overflow-y-auto divide-y divide-stone-200/60 dark:divide-stone-800">
                <div v-if="!notifications.length" class="px-6 py-16 text-center">
                  <UIcon name="i-lucide-bell-off" class="size-8 text-stone-300 dark:text-stone-600 mx-auto" />
                  <p class="mt-3 text-sm text-stone-500 dark:text-stone-400">暂无通知</p>
                </div>
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
const api = useApi()

const today = new Date().toLocaleDateString('zh-CN', { month: 'long', day: 'numeric', weekday: 'long' })

// KPI：商户最关注的 4 项数据
//   1. 今日营收 + 笔数（最高频，开机第一眼）
//   2. 本月累计营收（看月度业绩）
//   3. 会员卡池余额（潜在未来收入 = 已收的钱还能消多少）
//   4. 未结挂账（仅 > 0 时显示，红色提醒）
const kpi = reactive({
  todayRevenue: '0',
  todayCount: 0,
  monthRevenue: '0',
  cardPool: '0',
  pending: '0',
})

async function loadKpi() {
  try {
    const now = new Date()
    const todayStr = now.toISOString().slice(0, 10)
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1).toISOString().slice(0, 10)

    const [today, month, pending, members] = await Promise.all([
      api<any>(`/api/reports/business?start_date=${todayStr}&end_date=${todayStr}`).catch(() => null),
      api<any>(`/api/reports/business?start_date=${monthStart}&end_date=${todayStr}`).catch(() => null),
      api<any>('/api/reports/pending-stats?limit=1').catch(() => null),
      api<{ items: { total_balance: string }[] }>('/api/members?limit=200').catch(() => null),
    ])

    if (today) {
      kpi.todayRevenue = today.total_revenue || '0'
      kpi.todayCount = today.customer_count || 0
    }
    if (month) {
      kpi.monthRevenue = month.total_revenue || '0'
    }
    if (pending) kpi.pending = pending.summary?.total_amount || '0'

    // 卡池余额：本应有专属 endpoint；先用会员列表 sum total_balance（暂时近似）
    if (members?.items) {
      const sum = members.items.reduce((s, m) => s + parseFloat(m.total_balance || '0'), 0)
      kpi.cardPool = sum.toFixed(2)
    }
  } catch {}
}
onMounted(loadKpi)

// 祝福语：从 useGreeting composable 取（可被 PosWorkbench 结算成功时刷新）
const { greeting, refresh: refreshGreeting } = useGreeting()
let _greetingTimer: any
onMounted(() => {
  // 每 30 分钟自动换一次（一个客人服务周期，避免久看同一句腻）
  _greetingTimer = setInterval(refreshGreeting, 30 * 60 * 1000)
})
onBeforeUnmount(() => clearInterval(_greetingTimer))

// 通知（mock）
const notifications = ref<any[]>([])
const unreadCount = computed(() => notifications.value.filter(n => !n.read).length)
const hasUnread = computed(() => unreadCount.value > 0)
function markAllRead() { notifications.value.forEach(n => { n.read = true }) }
</script>
