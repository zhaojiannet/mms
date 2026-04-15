<template>
  <div class="p-6 lg:p-8 max-w-7xl mx-auto space-y-6">
    <header class="flex items-center justify-between gap-4 flex-wrap">
      <div>
        <h1 class="text-3xl font-semibold tracking-tight">报表</h1>
        <p class="mt-1 text-sm text-stone-500">{{ startDate }} ~ {{ endDate }}</p>
      </div>
      <div class="flex items-center gap-2 flex-wrap">
        <UBadge
          v-for="p in presets" :key="p.label"
          variant="soft" color="neutral"
          class="cursor-pointer active:scale-95"
          @click="applyPreset(p)"
        >{{ p.label }}</UBadge>
        <UInput v-model="startDate" type="date" size="md" />
        <span class="text-stone-400">→</span>
        <UInput v-model="endDate" type="date" size="md" />
        <UButton color="neutral" variant="soft" @click="onQuery">查询</UButton>
      </div>
    </header>

    <!-- Tabs：流水 + 经营报表 -->
    <UTabs v-model="activeTab" :items="tabs" class="w-full" :ui="{ trigger: 'cursor-pointer' }">
      <template #流水>
        <div class="space-y-4 mt-4">
          <div class="flex items-center gap-2 flex-wrap">
            <UBadge
              v-for="f in txFilters" :key="f.key"
              :color="kind === f.key ? 'primary' : 'neutral'"
              :variant="kind === f.key ? 'solid' : 'soft'"
              class="cursor-pointer active:scale-95"
              @click="kind = f.key as any; fetchTx()"
            >{{ f.label }}</UBadge>
            <span class="ml-2 text-sm text-stone-500">共 {{ txTotal }} 笔</span>
          </div>

          <div v-if="!txLoading && txItems.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-hidden">
            <table class="w-full text-base">
              <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
                <tr>
                  <th class="text-left px-4 py-3 font-medium">时间</th>
                  <th class="text-left px-4 py-3 font-medium">类型</th>
                  <th class="text-left px-4 py-3 font-medium">摘要</th>
                  <th class="text-right px-4 py-3 font-medium">应收</th>
                  <th class="text-right px-4 py-3 font-medium">实收</th>
                  <th class="text-center px-4 py-3 font-medium">状态</th>
                  <th class="text-right px-4 py-3 font-medium">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
                <tr v-for="t in txItems" :key="t.id" class="hover:bg-stone-50/60 dark:hover:bg-stone-800/40">
                  <td class="px-4 py-3 text-stone-500 text-sm tabular-nums">{{ formatTime(t.transaction_time) }}</td>
                  <td class="px-4 py-3"><UBadge :label="kindLabel(t.kind)" :color="kindColor(t.kind)" variant="soft" size="sm" /></td>
                  <td class="px-4 py-3 max-w-sm truncate">{{ t.summary || '—' }}</td>
                  <td class="px-4 py-3 text-right tabular-nums">¥{{ t.total_amount }}</td>
                  <td class="px-4 py-3 text-right tabular-nums font-medium">¥{{ t.actual_paid_amount }}</td>
                  <td class="px-4 py-3 text-center">
                    <UBadge v-if="t.status === 'voided'" label="已撤销" color="error" variant="soft" size="sm" />
                    <UBadge v-else label="正常" color="success" variant="soft" size="sm" />
                  </td>
                  <td class="px-4 py-3 text-right">
                    <UButton size="xs" variant="ghost" color="neutral" :disabled="t.status === 'voided'" @click="openVoid(t)">撤销</UButton>
                  </td>
                </tr>
              </tbody>
            </table>
            <div class="flex items-center justify-between p-3 border-t border-stone-200/60 dark:border-stone-800 bg-stone-50/30">
              <div class="text-sm text-stone-500">第 {{ txPage }} 页 / 共 {{ Math.ceil(txTotal / txLimit) }} 页</div>
              <div class="flex gap-1">
                <UButton size="sm" variant="soft" :disabled="txPage <= 1" @click="txPage--; fetchTx()">上一页</UButton>
                <UButton size="sm" variant="soft" :disabled="txPage * txLimit >= txTotal" @click="txPage++; fetchTx()">下一页</UButton>
              </div>
            </div>
          </div>
          <div v-else-if="!txLoading" class="py-20 text-center text-stone-500">当前日期范围无交易</div>
          <div v-if="txLoading" class="space-y-2"><USkeleton v-for="i in 5" :key="i" class="h-14 rounded-2xl" /></div>
        </div>
      </template>

      <template #经营>
        <div class="space-y-4 mt-4">
          <!-- KPI -->
          <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <KpiBox label="总消费" :value="fmtMoney(biz?.total_revenue)" hint="含挂账" />
            <KpiBox label="会员卡消费" :value="fmtMoney(biz?.card_consumption)" hint="卡耗" />
            <KpiBox label="充值金额" :value="fmtMoney(biz?.recharge_amount)" hint="办卡" />
            <KpiBox label="客数" :value="biz?.customer_count ?? '0'" :hint="`客单价 ¥${biz?.average_order_value || 0}`" />
          </div>

          <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <section class="p-5 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
              <h2 class="text-base font-medium mb-3">项目销售排行（Top 10）</h2>
              <div v-if="svcRanking.length === 0" class="text-sm text-stone-400 text-center py-6">无数据</div>
              <table v-else class="w-full text-sm">
                <tr v-for="(s, i) in svcRanking" :key="s.service_id" class="border-b last:border-0 border-stone-200/60 dark:border-stone-800">
                  <td class="py-2 tabular-nums text-stone-500 w-8">{{ i + 1 }}</td>
                  <td class="py-2">{{ s.service_name }}</td>
                  <td class="py-2 text-right tabular-nums text-stone-500">{{ s.total_count }} 次</td>
                  <td class="py-2 text-right tabular-nums font-medium w-24">¥{{ s.total_sales }}</td>
                </tr>
              </table>
            </section>

            <section class="p-5 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
              <h2 class="text-base font-medium mb-3">支付方式构成</h2>
              <div v-if="paymentSum.length === 0" class="text-sm text-stone-400 text-center py-6">无数据</div>
              <div v-else class="space-y-3">
                <div v-for="p in paymentSum" :key="p.payment_method_id">
                  <div class="flex items-center justify-between text-sm mb-1">
                    <span>{{ p.payment_method_name }}</span>
                    <span class="tabular-nums">¥{{ p.total }} <span class="text-stone-400">· {{ p.tx_count }}笔</span></span>
                  </div>
                  <div class="h-1.5 rounded-full bg-stone-100 dark:bg-stone-800 overflow-hidden">
                    <div class="h-full bg-primary-500" :style="{ width: paymentPct(p) + '%' }" />
                  </div>
                </div>
              </div>
            </section>
          </div>

          <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <section class="p-5 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
              <h2 class="text-base font-medium mb-3">挂账统计（未清）</h2>
              <div v-if="pendingStats" class="grid grid-cols-3 gap-2 mb-3 text-center">
                <div><div class="text-xs text-stone-500">笔数</div><div class="text-xl font-semibold tabular-nums">{{ pendingStats.summary.record_count }}</div></div>
                <div><div class="text-xs text-stone-500">会员数</div><div class="text-xl font-semibold tabular-nums">{{ pendingStats.summary.member_count }}</div></div>
                <div><div class="text-xs text-stone-500">总额</div><div class="text-xl font-semibold tabular-nums text-warning-600">¥{{ pendingStats.summary.total_amount }}</div></div>
              </div>
              <table v-if="pendingStats && pendingStats.data.length > 0" class="w-full text-sm">
                <tr v-for="p in pendingStats.data.slice(0, 5)" :key="p.id" class="border-b last:border-0 border-stone-200/60 dark:border-stone-800">
                  <td class="py-2">{{ p.name }}</td>
                  <td class="py-2 text-right tabular-nums text-stone-500">{{ p.record_count }} 笔</td>
                  <td class="py-2 text-right tabular-nums font-medium w-20">¥{{ p.total_pending }}</td>
                </tr>
              </table>
            </section>

            <section class="p-5 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
              <h2 class="text-base font-medium mb-3">会员消费排行（Top 10）</h2>
              <div v-if="memberRanking.length === 0" class="text-sm text-stone-400 text-center py-6">无数据</div>
              <table v-else class="w-full text-sm">
                <tr v-for="(m, i) in memberRanking" :key="m.id" class="border-b last:border-0 border-stone-200/60 dark:border-stone-800">
                  <td class="py-2 tabular-nums text-stone-500 w-8">{{ i + 1 }}</td>
                  <td class="py-2">{{ m.name }}</td>
                  <td class="py-2 text-right tabular-nums font-medium w-24">¥{{ m.total_consumption }}</td>
                </tr>
              </table>
            </section>
          </div>

          <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <section class="p-5 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
              <h2 class="text-base font-medium mb-3">沉睡会员（90 天未到店）</h2>
              <div v-if="sleeping.length === 0" class="text-sm text-stone-400 text-center py-6">无</div>
              <div v-else class="space-y-1 text-sm max-h-60 overflow-y-auto">
                <div v-for="m in sleeping.slice(0, 10)" :key="m.id" class="flex justify-between py-1">
                  <span>{{ m.name }} · {{ m.phone || '—' }}</span>
                  <span class="text-stone-500 tabular-nums">{{ m.last_visit ? formatDate(m.last_visit) : '从未' }}</span>
                </div>
              </div>
            </section>

            <section class="p-5 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
              <h2 class="text-base font-medium mb-3">生日提醒（未来 15 天）</h2>
              <div v-if="birthdays.length === 0" class="text-sm text-stone-400 text-center py-6">无</div>
              <div v-else class="space-y-1 text-sm max-h-60 overflow-y-auto">
                <div v-for="m in birthdays" :key="m.id" class="flex justify-between py-1">
                  <span>🎂 {{ m.name }} · {{ m.phone || '—' }}</span>
                  <span class="text-stone-500 tabular-nums">{{ m.birthday }}</span>
                </div>
              </div>
            </section>
          </div>
        </div>
      </template>
    </UTabs>

    <!-- 撤销弹窗 -->
    <UModal v-model:open="voidOpen" title="撤销交易" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <div class="space-y-3">
          <p class="text-sm text-stone-600">将交易 <strong>{{ voiding?.id.slice(0, 8) }}</strong> 设为已撤销，关联的卡余额 / 挂账会自动恢复。</p>
          <UFormField label="撤销原因" required>
            <UTextarea v-model="voidReason" :rows="2" class="w-full" />
          </UFormField>
          <UAlert v-if="voidErr" :description="voidErr" color="error" variant="soft" icon="i-lucide-alert-circle" />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton variant="ghost" color="neutral" @click="voidOpen = false">取消</UButton>
          <UButton color="error" :loading="voidLoading" :disabled="!voidReason" @click="doVoid">确认撤销</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
const api = useApi()
const route = useRoute()

const today = new Date()
const startDate = ref(fmtDate(new Date(today.getFullYear(), today.getMonth(), 1)))
const endDate = ref(fmtDate(today))

const tabs = [
  { label: '流水', slot: '流水' as const },
  { label: '经营', slot: '经营' as const },
]
// 仅识别 'tx' / 'biz'；其他任意参数均回退到第 0 个 tab
const activeTab = ref(route.query.tab === 'biz' ? '1' : '0')

// === 流水 ===
interface Tx { id: string; kind: 'sale'|'recharge'|'credit_settlement'; status: 'completed'|'voided'; total_amount: string; actual_paid_amount: string; transaction_time: string; summary: string|null }
const txItems = ref<Tx[]>([])
const txTotal = ref(0)
const txLoading = ref(false)
const txPage = ref(1)
const txLimit = ref(50)
const kind = ref<'' | 'sale' | 'recharge' | 'credit_settlement'>('')
const txFilters = [
  { key: '',                  label: '全部' },
  { key: 'sale',              label: '消费' },
  { key: 'recharge',          label: '办卡 / 充值' },
  { key: 'credit_settlement', label: '清账' },
]

const voidOpen = ref(false)
const voiding = ref<Tx | null>(null)
const voidReason = ref('')
const voidLoading = ref(false)
const voidErr = ref('')

function formatTime(s: string) { return new Date(s).toLocaleString('zh-CN', { hour12: false }) }
function kindLabel(k: string) { return ({ sale: '消费', recharge: '办卡', credit_settlement: '清账' } as Record<string,string>)[k] ?? k }
function kindColor(k: string): any { return ({ sale: 'primary', recharge: 'info', credit_settlement: 'neutral' } as Record<string,string>)[k] ?? 'neutral' }

async function fetchTx() {
  txLoading.value = true
  try {
    const query = new URLSearchParams({
      start_date: `${startDate.value}T00:00:00Z`,
      end_date:   `${endDate.value}T23:59:59Z`,
      page:       String(txPage.value),
      limit:      String(txLimit.value),
      include_voided: '1',
    })
    if (kind.value) query.set('kind', kind.value)
    const data = await api<{ items: Tx[]; total: number }>(`/api/transactions?${query}`)
    txItems.value = data.items
    txTotal.value = data.total
  } finally { txLoading.value = false }
}

function openVoid(t: Tx) { voiding.value = t; voidReason.value = ''; voidErr.value = ''; voidOpen.value = true }
async function doVoid() {
  if (!voiding.value) return
  voidLoading.value = true; voidErr.value = ''
  try {
    await api(`/api/transactions/${voiding.value.id}/void`, { method: 'POST', body: { reason: voidReason.value } })
    voidOpen.value = false
    await fetchTx()
  } catch (e: any) {
    voidErr.value = e?.data?.message || '撤销失败（请确认在"设置 → 交易撤销"里开启）'
  } finally { voidLoading.value = false }
}

// === 经营报表 ===
const biz = ref<any>(null)
const svcRanking = ref<any[]>([])
const paymentSum = ref<any[]>([])
const pendingStats = ref<any>(null)
const memberRanking = ref<any[]>([])
const sleeping = ref<any[]>([])
const birthdays = ref<any[]>([])

const presets = [
  { label: '今日',     days: 0 },
  { label: '本月',     days: -1 },
  { label: '近 7 天',  days: 7 },
  { label: '近 30 天', days: 30 },
]

function fmtDate(d: Date) { return d.toISOString().slice(0, 10) }
function formatDate(s: string) { return s ? s.slice(0, 10) : '' }
function fmtMoney(n: string | undefined | null) { return '¥' + (n || '0') }
function paymentPct(p: any) {
  const maxT = Math.max(...paymentSum.value.map(x => parseFloat(x.total)))
  return maxT > 0 ? (parseFloat(p.total) / maxT * 100) : 0
}
function applyPreset(p: typeof presets[number]) {
  const now = new Date()
  endDate.value = fmtDate(now)
  if (p.days === 0) startDate.value = fmtDate(now)
  else if (p.days === -1) startDate.value = fmtDate(new Date(now.getFullYear(), now.getMonth(), 1))
  else { const s = new Date(now); s.setDate(s.getDate() - p.days + 1); startDate.value = fmtDate(s) }
  onQuery()
}

async function fetchBiz() {
  const q = `start_date=${startDate.value}&end_date=${endDate.value}`
  const [b, sr, ps, pst, mr, sl, bd] = await Promise.all([
    api(`/api/reports/business?${q}`),
    api<{ data: any[] }>(`/api/reports/service-ranking?${q}&limit=10`),
    api<{ data: any[] }>(`/api/reports/payment-summary?${q}`),
    api<any>(`/api/reports/pending-stats?limit=10`),
    api<{ data: any[] }>(`/api/reports/member-ranking?${q}&limit=10`),
    api<{ data: any[] }>(`/api/reports/sleeping-members?limit=10`),
    api<{ data: any[] }>(`/api/reports/birthday-reminders`),
  ])
  biz.value = b
  svcRanking.value = sr.data || []
  paymentSum.value = ps.data || []
  pendingStats.value = pst
  memberRanking.value = mr.data || []
  sleeping.value = sl.data || []
  birthdays.value = bd.data || []
}

function onQuery() {
  txPage.value = 1
  fetchTx()
  fetchBiz()
}

onMounted(onQuery)
</script>

<script lang="ts">
export default {
  components: {
    KpiBox: {
      template: `<div class="p-4 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs"><div class="text-xs text-stone-500">{{ label }}</div><div class="text-2xl font-semibold tabular-nums mt-1">{{ value }}</div><div v-if="hint" class="text-xs text-stone-500 mt-1">{{ hint }}</div></div>`,
      props: ['label','value','hint'],
    },
  },
}
</script>
