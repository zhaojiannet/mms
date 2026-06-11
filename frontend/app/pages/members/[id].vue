<template>
  <div class="p-6 lg:p-8 max-w-7xl mx-auto space-y-5">
    <UButton variant="ghost" color="neutral" icon="i-lucide-arrow-left" @click="router.push('/members')">返回会员列表</UButton>

    <div v-if="loading" class="space-y-3">
      <USkeleton class="h-24 rounded-2xl" />
      <USkeleton class="h-40 rounded-2xl" />
    </div>

    <template v-else-if="member">
      <!-- Header -->
      <header class="flex items-start justify-between gap-4 flex-wrap">
        <div class="flex items-center gap-4">
          <UAvatar :alt="member.name" size="lg" />
          <div>
            <h1 class="text-2xl sm:text-3xl font-semibold tracking-tight">{{ member.name }}</h1>
            <div class="mt-1 text-sm text-stone-500 space-x-3 tabular-nums">
              <span>{{ member.phone || '未留手机' }}</span>
              <span>{{ genderLabel(member.gender) }}</span>
              <span v-if="member.birthday">生日：{{ member.birthday }}</span>
            </div>
            <p v-if="member.notes" class="mt-2 text-sm text-stone-600 dark:text-stone-400 italic">{{ member.notes }}</p>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2 w-full sm:w-auto">
          <UButton color="neutral" variant="soft" icon="i-lucide-plus-circle" class="flex-1 sm:flex-none justify-center" @click="openIssueCard">办卡</UButton>
          <UButton color="neutral" variant="soft" icon="i-lucide-file-minus" class="flex-1 sm:flex-none justify-center" @click="openPending">挂账</UButton>
          <UButton icon="i-lucide-shopping-cart" class="flex-1 sm:flex-none justify-center" @click="goPos">开单</UButton>
        </div>
      </header>

      <!-- KPI: 卡总余额 / 挂账未清 / 累计消费 -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div class="p-4 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
          <div class="text-xs text-stone-500">卡余额合计</div>
          <div class="text-2xl font-semibold tabular-nums mt-1">¥{{ totalBalance }}</div>
        </div>
        <div class="p-4 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
          <div class="text-xs text-stone-500">未清挂账</div>
          <div class="text-2xl font-semibold tabular-nums mt-1 text-warning-600">¥{{ totalPending }}</div>
        </div>
        <div class="p-4 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
          <div class="text-xs text-stone-500">状态</div>
          <div class="text-lg font-medium mt-1">
            <UBadge :label="member.status === 'active' ? '正常' : '停用'" :color="member.status === 'active' ? 'success' : 'neutral'" variant="soft" />
          </div>
        </div>
      </div>

      <!-- 会员卡 -->
      <section>
        <div class="flex items-center justify-between mb-3">
          <SectionTitle as="h2">会员卡</SectionTitle>
          <span class="text-sm text-stone-500">{{ cards.length }} 张</span>
        </div>
        <div v-if="cards.length > 0" class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div
            v-for="c in cards" :key="c.id"
            class="p-4 rounded-2xl bg-linear-to-br from-primary-50/50 to-white dark:from-primary-950/30 dark:to-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800"
          >
            <div class="flex items-start justify-between">
              <div>
                <div class="font-medium">{{ c.card_type_name }}</div>
                <div class="text-xs text-stone-500 mt-1">
                  <span>面值 ¥{{ c.final_price }}</span>
                  <span class="mx-1">·</span>
                  <span>{{ displayRate(c.final_discount_rate) }}</span>
                  <UBadge v-if="c.is_custom" label="定制" color="info" variant="soft" size="sm" class="ml-2" />
                </div>
              </div>
              <UBadge
                :label="statusLabel(c.status)"
                :color="c.status === 'active' ? 'success' : 'neutral'"
                variant="soft" size="sm"
              />
            </div>
            <div class="mt-3 text-3xl font-semibold tabular-nums">¥{{ c.balance }}</div>
            <div class="mt-1 text-xs text-stone-500">开卡于 {{ formatDate(c.issued_at) }}</div>
          </div>
        </div>
        <EmptyState v-else icon="i-lucide-credit-card" text="暂无会员卡" hint="点击右上角「办卡」添加" />
      </section>

      <!-- 挂账 -->
      <section>
        <div class="flex items-center justify-between mb-3">
          <SectionTitle as="h2">未清挂账</SectionTitle>
          <UButton v-if="pendings.length > 1" size="sm" variant="soft" @click="openSettleAll">批量清账</UButton>
        </div>
        <div v-if="pendings.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 overflow-hidden">
          <table class="w-full text-sm">
            <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
              <tr>
                <th class="text-left px-3.5 py-2.5 font-medium">事由</th>
                <th class="text-right px-3.5 py-2.5 font-medium">金额</th>
                <th class="text-left px-3.5 py-2.5 font-medium">挂账日</th>
                <th class="text-right px-3.5 py-2.5 font-medium">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
              <tr v-for="p in pendings" :key="p.id" class="even:bg-stone-50 dark:even:bg-stone-800/30 hover:bg-stone-100/60 dark:hover:bg-stone-800/40 transition-colors">
                <td class="px-3.5 py-3">{{ p.summary || '—' }}</td>
                <td class="px-3.5 py-3 text-right tabular-nums">¥{{ p.amount }}</td>
                <td class="px-3.5 py-3 text-stone-500 tabular-nums">{{ formatDate(p.charged_at) }}</td>
                <td class="px-3.5 py-3 text-right">
                  <div class="inline-flex items-center gap-1.5">
                    <UButton
                      size="xs" variant="soft" color="primary"
                      icon="i-lucide-circle-check"
                      class="active:scale-95 transition-transform"
                      @click="openSettle(p)"
                    >清账</UButton>
                    <UButton
                      size="xs" variant="soft" color="error"
                      icon="i-lucide-rotate-ccw"
                      class="active:scale-95 transition-transform"
                      @click="askRemovePending(p)"
                    >撤消</UButton>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <EmptyState v-else icon="i-lucide-file-minus" text="无未清挂账" />
      </section>

      <!-- 最近交易 -->
      <section>
        <SectionTitle as="h2" class="mb-3">最近交易</SectionTitle>
        <div v-if="recentTx.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 overflow-hidden">
          <table class="w-full text-sm">
            <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
              <tr>
                <th class="text-left px-3.5 py-2.5 font-medium">时间</th>
                <th class="text-left px-3.5 py-2.5 font-medium">类型</th>
                <th class="text-left px-3.5 py-2.5 font-medium">摘要</th>
                <th class="text-right px-3.5 py-2.5 font-medium">金额</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
              <tr v-for="t in recentTx" :key="t.id" class="even:bg-stone-50 dark:even:bg-stone-800/30 hover:bg-stone-100/60 dark:hover:bg-stone-800/40 transition-colors">
                <td class="px-3.5 py-3 text-stone-500 tabular-nums">{{ formatTime(t.transaction_time) }}</td>
                <td class="px-3.5 py-3">
                  <UBadge :label="kindLabel(t.kind)" variant="soft" size="md" />
                </td>
                <td class="px-3.5 py-3 max-w-md truncate">{{ t.summary || '—' }}</td>
                <td class="px-3.5 py-3 text-right tabular-nums font-medium">¥{{ t.actual_paid_amount }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <EmptyState v-else icon="i-lucide-receipt" text="无交易记录" />
      </section>

      <!-- 挂账 / 清账 dialogs -->
      <UModal v-model:open="pendOpen" title="新增挂账" :ui="{ content: 'sm:max-w-md' }">
        <template #body>
          <div class="space-y-3">
            <UFormField label="金额（元）" required><UInput v-model="pendForm.amount" type="number" step="0.01" class="w-full" /></UFormField>
            <UFormField label="事由"><UInput v-model="pendForm.summary" placeholder="如：剪发+洗面 38 元" class="w-full" /></UFormField>
          </div>
        </template>
        <template #footer>
          <div class="flex justify-end gap-2 w-full">
            <UButton variant="ghost" color="neutral" @click="pendOpen = false">取消</UButton>
            <UButton :loading="pendSaving" @click="submitPending">保存</UButton>
          </div>
        </template>
      </UModal>

      <UModal v-model:open="settleOpen" title="清账" :ui="{ content: 'sm:max-w-md' }">
        <template #body>
          <div class="space-y-3">
            <p v-if="settleAll">将一次性结清 {{ pendings.length }} 笔挂账，总额 <strong>¥{{ totalPending }}</strong></p>
            <p v-else>事由：{{ settleTarget?.summary || '—' }}<br />金额：<strong>¥{{ settleTarget?.amount }}</strong></p>
            <UFormField label="收款方式" required>
              <USelect v-model="settleForm.paymentMethodId" :items="paymentOptions" class="w-full" />
            </UFormField>
            <UFormField v-if="isMemberCard" label="会员卡">
              <USelect v-model="settleForm.cardId" :items="cardOptions" placeholder="选择卡" class="w-full" />
            </UFormField>
            <UAlert v-if="settleErr" :description="settleErr" color="error" variant="soft" icon="i-lucide-alert-circle" />
          </div>
        </template>
        <template #footer>
          <div class="flex justify-end gap-2 w-full">
            <UButton variant="ghost" color="neutral" @click="settleOpen = false">取消</UButton>
            <UButton :loading="settleSaving" @click="submitSettle">确认清账</UButton>
          </div>
        </template>
      </UModal>

      <UModal v-model:open="issueOpen" title="办卡" :ui="{ content: 'sm:max-w-md' }">
        <template #body>
          <div class="space-y-3">
            <UFormField label="卡型" required>
              <USelect v-model="issueForm.cardTypeId" :items="cardTypeOptions" class="w-full" />
            </UFormField>
            <UFormField label="实际金额（默认 = 卡型售价，定制卡改这里）">
              <UInput v-model="issueForm.finalPrice" type="number" step="0.01" class="w-full" />
            </UFormField>
            <UFormField label="收款方式" required>
              <USelect v-model="issueForm.paymentMethodId" :items="paymentOptions" class="w-full" />
            </UFormField>
            <UAlert v-if="issueErr" :description="issueErr" color="error" variant="soft" icon="i-lucide-alert-circle" />
          </div>
        </template>
        <template #footer>
          <div class="flex justify-end gap-2 w-full">
            <UButton variant="ghost" color="neutral" @click="issueOpen = false">取消</UButton>
            <UButton :loading="issueSaving" @click="submitIssue">办理</UButton>
          </div>
        </template>
      </UModal>

      <DeleteConfirmModal
        v-model:open="removePendOpen"
        title="撤消挂账"
        confirm-text="确认撤消"
        :loading="removePendLoading"
        @confirm="confirmRemovePending"
      >
        <template #message>
          <p>确认撤消这笔 <strong>¥{{ removePendTarget?.amount }}</strong> 的未清挂账？</p>
        </template>
      </DeleteConfirmModal>
    </template>
  </div>
</template>

<script setup lang="ts">
const api = useApi()
const route = useRoute()
const router = useRouter()
const toast = useToast()
const id = route.params.id as string
const ops = useMemberOps(() => id)

interface Member { id: string; name: string; phone: string | null; gender: string; birthday: string | null; notes: string | null; status: string }
interface Card { id: string; card_type_name: string; card_type_id: string; final_price: string; final_discount_rate: string; balance: string; status: string; issued_at: string; is_custom: boolean }
interface Pending { id: string; amount: string; summary: string | null; charged_at: string }
interface Tx { id: string; kind: string; summary: string | null; actual_paid_amount: string; transaction_time: string }

const loading = ref(true)
const member = ref<Member | null>(null)
const cards = ref<Card[]>([])
const pendings = ref<Pending[]>([])
const recentTx = ref<Tx[]>([])
const totalPending = ref('0')
const paymentMethods = ref<{ id: string; name: string }[]>([])
const cardTypes = ref<{ id: string; name: string; price: string; discount_rate: string }[]>([])

const totalBalance = computed(() => {
  return cards.value
    .filter(c => c.status === 'active')
    .reduce((s, c) => s + parseFloat(c.balance), 0)
    .toFixed(2)
})

const paymentOptions = computed(() => paymentMethods.value.map(p => ({ label: p.name, value: p.id })))
const cardOptions = computed(() => cards.value.filter(c => c.status === 'active' && parseFloat(c.balance) > 0).map(c => ({ label: `${c.card_type_name}（余额 ¥${c.balance}）`, value: c.id })))
const cardTypeOptions = computed(() => cardTypes.value.map(c => ({ label: `${c.name}  ¥${c.price}`, value: c.id })))

async function fetchAll() {
  loading.value = true
  try {
    const [m, cs, ps, tx, pm, ct] = await Promise.all([
      api<Member>(`/api/members/${id}`),
      api<{ items: Card[] }>(`/api/members/${id}/cards`),
      api<{ items: Pending[]; total_amount: string }>(`/api/members/${id}/pending`),
      api<{ items: Tx[] }>(`/api/transactions?member_id=${id}&limit=10`),
      api<{ items: { id: string; name: string }[] }>('/api/payment-methods?active=1'),
      api<{ items: { id: string; name: string; price: string; discount_rate: string }[] }>('/api/card-types?status=active'),
    ])
    member.value = m
    cards.value = cs.items
    pendings.value = ps.items
    totalPending.value = ps.total_amount
    recentTx.value = tx.items
    paymentMethods.value = pm.items
    cardTypes.value = ct.items
  } finally { loading.value = false }
}

function genderLabel(g: string) { return { male: '男', female: '女', unknown: '—' }[g] ?? '—' }
function statusLabel(s: string) { return (CARD_STATUS_LABEL as Record<string, string>)[s] ?? s }
function kindLabel(k: string) { return (TX_KIND_LABEL as Record<string, string>)[k] ?? k }
function displayRate(r: string) { const n = parseFloat(r); return (Math.round(n * 100) / 10).toFixed(1) + '折' }

// ---- 挂账 ----
const pendOpen = ref(false)
const pendSaving = ref(false)
const pendForm = reactive({ amount: '', summary: '' })
function openPending() { pendForm.amount = ''; pendForm.summary = ''; pendOpen.value = true }
async function submitPending() {
  const amt = pendForm.amount.trim()
  // 后端金额列为 NUMERIC(10,2)，超过 2 位小数会被静默舍入，前端先拦
  if (!/^\d+(\.\d{1,2})?$/.test(amt) || parseFloat(amt) <= 0) {
    toast.add({ title: '金额无效', description: '请输入大于 0 的金额，最多 2 位小数', color: 'error', icon: 'i-lucide-alert-circle' })
    return
  }
  pendSaving.value = true
  try {
    await ops.addPending({ amount: amt, summary: pendForm.summary || null })
    pendOpen.value = false
  } catch (e: any) {
    toast.add({ title: '挂账失败', description: e?.data?.message || e?.message || '请稍后重试', color: 'error' })
    return
  } finally { pendSaving.value = false }
  await fetchAll()
}
const removePendOpen = ref(false)
const removePendTarget = ref<Pending | null>(null)
const removePendLoading = ref(false)
function askRemovePending(p: Pending) {
  removePendTarget.value = p
  removePendOpen.value = true
}
async function confirmRemovePending() {
  if (!removePendTarget.value) return
  removePendLoading.value = true
  try {
    await ops.removePending(removePendTarget.value.id)
    removePendOpen.value = false
  } catch (e: any) {
    toast.add({ title: '撤消挂账失败', description: e?.data?.message || e?.message || '请稍后重试', color: 'error' })
    return
  } finally { removePendLoading.value = false }
  await fetchAll()
}

// ---- 清账 ----
const settleOpen = ref(false)
const settleSaving = ref(false)
const settleErr = ref('')
const settleAll = ref(false)
const settleTarget = ref<Pending | null>(null)
const settleForm = reactive({ paymentMethodId: '', cardId: '' })
const isMemberCard = computed(() => {
  const pm = paymentMethods.value.find(p => p.id === settleForm.paymentMethodId)
  return pm?.name === '会员卡'
})
function openSettle(p: Pending) { settleAll.value = false; settleTarget.value = p; settleForm.paymentMethodId = ''; settleForm.cardId = ''; settleErr.value = ''; settleOpen.value = true }
function openSettleAll() { settleAll.value = true; settleTarget.value = null; settleForm.paymentMethodId = ''; settleForm.cardId = ''; settleErr.value = ''; settleOpen.value = true }
async function submitSettle() {
  if (!settleForm.paymentMethodId) { settleErr.value = '请选择收款方式'; return }
  settleSaving.value = true; settleErr.value = ''
  try {
    const body = {
      payment_method_id: settleForm.paymentMethodId,
      card_id: isMemberCard.value ? (settleForm.cardId || null) : undefined,
    }
    if (settleAll.value) await ops.settleAll(body)
    else await ops.settleOne(settleTarget.value!.id, body)
    settleOpen.value = false
  } catch (e: any) { settleErr.value = e?.data?.message || '清账失败'; return }
  finally { settleSaving.value = false }
  await fetchAll()
}

// ---- 办卡 ----
const issueOpen = ref(false)
const issueSaving = ref(false)
const issueErr = ref('')
const issueForm = reactive({ cardTypeId: '', finalPrice: '', paymentMethodId: '' })
function openIssueCard() { issueForm.cardTypeId = ''; issueForm.finalPrice = ''; issueForm.paymentMethodId = ''; issueErr.value = ''; issueOpen.value = true }
async function submitIssue() {
  if (!issueForm.cardTypeId || !issueForm.paymentMethodId) { issueErr.value = '请选卡型和收款方式'; return }
  issueSaving.value = true; issueErr.value = ''
  try {
    const body: Record<string, unknown> = {
      card_type_id: issueForm.cardTypeId,
      payment_method_id: issueForm.paymentMethodId,
    }
    if (issueForm.finalPrice) body.final_price = issueForm.finalPrice
    await ops.issueCard(body)
    issueOpen.value = false
  } catch (e: any) { issueErr.value = e?.data?.message || '办卡失败'; return }
  finally { issueSaving.value = false }
  await fetchAll()
}

function goPos() { router.push(`/pos?member=${id}`) }

onMounted(fetchAll)
</script>
