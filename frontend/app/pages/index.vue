<template>
  <div>
    <!-- Navbar -->
    <header class="h-14 flex items-center justify-between gap-3 px-8 bg-stone-50/80 dark:bg-stone-950/80 backdrop-blur-sm border-b border-stone-200/60 dark:border-stone-800 sticky top-0 z-10">
      <div class="flex items-center gap-2.5 min-w-0">
        <UIcon name="i-lucide-layout-dashboard" class="size-4 text-stone-400" />
        <span class="text-base font-medium text-stone-700 dark:text-stone-300">工作台</span>
      </div>
      <div class="flex items-center gap-1.5">
        <UPopover :content="{ side: 'bottom', align: 'end', sideOffset: 8 }">
          <div class="relative">
            <UButton
              icon="i-lucide-bell"
              variant="ghost"
              color="neutral"
              size="md"
              square
              aria-label="通知"
              class="!bg-stone-100 dark:!bg-stone-800 hover:!bg-stone-200/70 dark:hover:!bg-stone-700
                     transition-colors duration-150"
            />
            <span
              v-if="hasUnread"
              class="absolute top-1 right-1 size-2 rounded-full bg-red-500
                     ring-2 ring-stone-50 dark:ring-stone-950 pointer-events-none"
            />
          </div>

          <template #content>
            <div class="w-96 max-h-[28rem] flex flex-col">
              <!-- 头部 -->
              <div class="flex items-center justify-between px-4 py-3 border-b border-stone-200/70 dark:border-stone-800">
                <div class="flex items-center gap-2">
                  <span class="text-sm font-semibold">通知</span>
                  <UBadge v-if="hasUnread" :label="String(unreadCount)" color="error" variant="soft" size="sm" />
                </div>
                <UButton
                  v-if="hasUnread"
                  label="全部标记为已读"
                  variant="ghost"
                  color="neutral"
                  size="xs"
                  @click="markAllRead"
                />
              </div>

              <!-- 列表 -->
              <div class="flex-1 overflow-y-auto divide-y divide-stone-200/60 dark:divide-stone-800">
                <div
                  v-for="n in notifications"
                  :key="n.id"
                  class="flex items-start gap-3 px-4 py-3 hover:bg-stone-50 dark:hover:bg-stone-800/40 transition-colors duration-150 cursor-pointer"
                >
                  <div
                    :class="[
                      'w-8 h-8 rounded-lg flex items-center justify-center shrink-0',
                      notifPalette[n.type].bg,
                    ]"
                  >
                    <UIcon :name="notifPalette[n.type].icon" :class="['size-4', notifPalette[n.type].text]" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="text-sm font-medium truncate">{{ n.title }}</span>
                      <span v-if="!n.read" class="size-1.5 rounded-full bg-primary-500 shrink-0" />
                    </div>
                    <p class="text-sm text-stone-500 dark:text-stone-400 mt-0.5 line-clamp-2">{{ n.body }}</p>
                    <div class="text-xs text-stone-400 dark:text-stone-500 mt-1">{{ n.time }}</div>
                  </div>
                </div>

                <!-- 空态 -->
                <div v-if="!notifications.length" class="px-6 py-16 text-center">
                  <UIcon name="i-lucide-bell-off" class="size-8 text-stone-300 dark:text-stone-600 mx-auto" />
                  <p class="mt-3 text-sm text-stone-500 dark:text-stone-400">暂无通知</p>
                </div>
              </div>

              <!-- 底部 -->
              <div
                v-if="notifications.length"
                class="px-3 py-2 border-t border-stone-200/70 dark:border-stone-800"
              >
                <UButton
                  label="查看全部通知"
                  variant="ghost"
                  color="neutral"
                  size="sm"
                  block
                  class="!justify-center"
                />
              </div>
            </div>
          </template>
        </UPopover>

      </div>
    </header>

    <div class="px-8 py-10 space-y-10 max-w-[1400px] mx-auto w-full">
      <!-- 问候 -->
      <div>
        <h1 class="text-3xl font-semibold tracking-tight text-stone-900 dark:text-stone-50">
          {{ greeting }}，{{ auth.user?.name || '老板' }}
        </h1>
        <p class="mt-1.5 text-sm text-stone-500 dark:text-stone-400">
          {{ today }}
        </p>
      </div>

      <!-- KPI grid -->
      <section class="grid grid-cols-2 lg:grid-cols-4 gap-5">
        <KpiCard
          icon="i-lucide-wallet"
          label="今日营收"
          :value="kpi.revenue"
          prefix="¥"
          :delta="kpi.revenueDelta"
        />
        <KpiCard
          icon="i-lucide-users"
          label="今日客流"
          :value="kpi.footfall"
          :delta="kpi.footfallDelta"
        />
        <KpiCard
          icon="i-lucide-receipt-text"
          label="未结挂账"
          :value="kpi.pending"
          prefix="¥"
          :delta="null"
        />
        <KpiCard
          icon="i-lucide-user-round-plus"
          label="会员总数"
          :value="kpi.members"
          :delta="kpi.membersDelta"
        />
      </section>

      <!-- Two columns -->
      <div class="grid lg:grid-cols-3 gap-8">
        <!-- Recent transactions -->
        <section class="lg:col-span-2">
          <div class="flex items-center justify-between mb-4">
            <h2 class="text-lg font-semibold">最近消费</h2>
            <UButton
              variant="ghost"
              color="neutral"
              size="xs"
              trailing-icon="i-lucide-arrow-right"
              to="/reports"
            >查看全部</UButton>
          </div>
          <div class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs divide-y divide-stone-200/60 dark:divide-stone-800 overflow-hidden">
            <div
              v-for="t in recentTx"
              :key="t.id"
              class="flex items-center gap-3 px-6 py-4 hover:bg-stone-50 dark:hover:bg-stone-800/40 transition-colors duration-150"
            >
              <UAvatar :alt="t.name" size="md" />
              <div class="min-w-0 flex-1">
                <div class="text-base font-medium truncate">{{ t.name }}</div>
                <div class="text-sm text-stone-500 dark:text-stone-400 truncate mt-0.5">
                  {{ t.service }} · {{ t.staff }} · {{ t.time }}
                </div>
              </div>
              <div class="text-right shrink-0">
                <div class="tabular-nums font-semibold text-lg">¥{{ t.amount }}</div>
                <div class="text-xs text-stone-500 dark:text-stone-400 mt-0.5">{{ t.pay }}</div>
              </div>
            </div>
            <div v-if="!recentTx.length" class="px-6 py-20 text-center">
              <UIcon name="i-lucide-coffee" class="size-10 text-stone-300 dark:text-stone-600 mx-auto" />
              <p class="mt-4 text-base text-stone-500 dark:text-stone-400">今日还没有消费记录</p>
            </div>
          </div>
        </section>

        <!-- Quick + Reminders -->
        <section class="space-y-8">
          <div>
            <h2 class="text-lg font-semibold mb-4">快捷入口</h2>
            <div class="grid grid-cols-2 gap-4">
              <QuickTile icon="i-lucide-credit-card" label="开单" sub="收银结算" to="/transactions" />
              <QuickTile icon="i-lucide-user-plus" label="新建会员" sub="录入信息" @click="goNewMember" />
              <QuickTile icon="i-lucide-calendar-plus" label="登记预约" sub="到店排期" to="/appointments" />
              <QuickTile icon="i-lucide-gift" label="办卡充值" sub="充值赠送" to="/members" />
            </div>
          </div>

          <div>
            <h2 class="text-lg font-semibold mb-4">提醒</h2>
            <div class="space-y-2">
              <ReminderItem icon="i-lucide-cake" text="本周 3 位会员生日" tone="warning" />
              <ReminderItem icon="i-lucide-bed-double" text="12 位 90 天未到店" tone="info" />
            </div>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()
const api = useApi()

const today = new Date().toLocaleDateString('zh-CN', { month: 'long', day: 'numeric', weekday: 'long' })
const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 6) return '凌晨好'
  if (h < 11) return '早上好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
})

const kpi = reactive({
  revenue: '0',
  revenueDelta: null as number | null,
  footfall: 0,
  footfallDelta: null as number | null,
  pending: '0',
  members: 0,
  membersDelta: null as number | null,
})

async function loadKpi() {
  try {
    const data = await api<{ total: number }>('/api/members?limit=1')
    kpi.members = data.total
  } catch {}
}
onMounted(loadKpi)

const recentTx: Array<{id:string;name:string;service:string;staff:string;time:string;amount:string;pay:string}> = []

// --- 通知（mock，待接入真实接口） ---
type NotifType = 'appointment' | 'update' | 'maintenance' | 'info'

interface Notif {
  id: string
  type: NotifType
  title: string
  body: string
  time: string
  read: boolean
}

const notifPalette: Record<NotifType, { icon: string; bg: string; text: string }> = {
  appointment:  { icon: 'i-lucide-calendar-clock',   bg: 'bg-primary-50 dark:bg-primary-950/40', text: 'text-primary-600 dark:text-primary-400' },
  update:       { icon: 'i-lucide-sparkles',         bg: 'bg-blue-50 dark:bg-blue-950/40',       text: 'text-blue-600 dark:text-blue-400' },
  maintenance:  { icon: 'i-lucide-wrench',           bg: 'bg-amber-50 dark:bg-amber-950/40',     text: 'text-amber-600 dark:text-amber-400' },
  info:         { icon: 'i-lucide-info',             bg: 'bg-stone-100 dark:bg-stone-800',       text: 'text-stone-600 dark:text-stone-300' },
}

const notifications = ref<Notif[]>([])

const unreadCount = computed(() => notifications.value.filter(n => !n.read).length)
const hasUnread = computed(() => unreadCount.value > 0)

function markAllRead() {
  notifications.value.forEach(n => { n.read = true })
}
</script>
