<template>
  <div class="space-y-4">
    <!-- 开关面板 -->
    <div class="p-6 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
      <div class="flex items-start justify-between gap-4 mb-2">
        <div>
          <h2 class="text-base font-medium">交易撤销</h2>
          <p class="text-sm text-stone-500 mt-1">为防止误操作，撤销功能默认关闭。开启后 10 分钟自动关闭。</p>
        </div>
        <USwitch
          :model-value="enabled"
          size="lg"
          :loading="loading"
          @update:model-value="onToggle"
        />
      </div>

      <!-- 超级密码管理（店内共享） -->
      <div class="mt-4 p-3 rounded-xl bg-stone-50 dark:bg-stone-950/40 ring-1 ring-stone-200/60 dark:ring-stone-800 flex items-center gap-3">
        <UIcon name="i-lucide-shield-check" class="size-5 text-stone-500" />
        <div class="flex-1 text-sm">
          <span class="font-medium">超级密码</span>
          <span class="text-stone-500 ml-2">店内共享，用于开启撤销功能</span>
        </div>
        <UBadge :label="superSet ? '已设置' : '未设置'" :color="superSet ? 'success' : 'warning'" variant="soft" size="sm" />
        <UButton
          size="xs" variant="soft" color="primary"
          icon="i-lucide-key-round"
          class="active:scale-95 transition-transform"
          @click="spOpen = true"
        >{{ superSet ? '修改' : '设置' }}</UButton>
      </div>

      <!-- 已开启状态 -->
      <div v-if="enabled" class="mt-4 p-3 rounded-xl bg-success-50 dark:bg-success-950/30 ring-1 ring-success-500/20 flex items-center gap-3">
        <UIcon name="i-lucide-check-circle" class="size-5 text-success-600" />
        <div class="flex-1 text-sm">
          <span class="font-medium text-success-700 dark:text-success-300">已开启</span>
          <span class="text-stone-600 dark:text-stone-400 ml-2">将在 <span class="tabular-nums font-medium">{{ formatRemaining(remaining) }}</span> 后自动关闭</span>
        </div>
      </div>

      <!-- 未开启 → 需要超级密码 -->
      <div v-else-if="pendingEnable" class="mt-4 p-4 rounded-xl bg-stone-50 dark:bg-stone-950/40 ring-1 ring-stone-200/60 dark:ring-stone-800">
        <UAlert v-if="!superSet" color="warning" variant="soft" icon="i-lucide-alert-triangle"
          title="尚未设置超级密码"
          description="请先在上方点击「设置」按钮配置超级密码，然后再开启此功能。"
          class="mb-3" />
        <UFormField v-if="superSet" label="超级密码">
          <UInput v-model="superPwd" type="password" autocomplete="off" class="w-full sm:w-72" autofocus @keydown.enter="enable" />
        </UFormField>
        <UAlert v-if="err" :description="err" color="error" variant="soft" icon="i-lucide-alert-circle" class="mt-3" />
        <div class="mt-3 flex items-center gap-2">
          <UButton v-if="superSet" :loading="loading" :disabled="!superPwd" @click="enable">确认开启</UButton>
          <UButton variant="ghost" color="neutral" @click="cancelPending">取消</UButton>
        </div>
      </div>
    </div>

    <!-- 超级密码编辑弹窗 -->
    <UModal v-model:open="spOpen" :title="superSet ? '修改超级密码' : '设置超级密码'" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <div class="space-y-4">
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-3">
            <p class="text-xs text-stone-500">超级密码用于开启交易撤销等敏感操作，全店共享。改密需输入当前登录用户的登录密码确认。</p>
            <UFormField label="当前登录密码" required>
              <UInput v-model="sp.login" type="password" size="md" autocomplete="current-password" class="w-full" />
            </UFormField>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <UFormField label="新的超级密码（≥8 位）" required>
                <UInput v-model="sp.next" type="password" size="md" autocomplete="new-password" class="w-full" />
              </UFormField>
              <UFormField
                label="确认超级密码"
                required
                :error="sp.next && sp.confirm && sp.next !== sp.confirm ? '两次输入不一致' : undefined"
              >
                <UInput v-model="sp.confirm" type="password" size="md" autocomplete="new-password" class="w-full" />
              </UFormField>
            </div>
          </div>
          <div class="flex justify-end gap-2 pt-2">
            <UButton type="button" variant="ghost" color="neutral" @click="spOpen = false">取消</UButton>
            <UButton :loading="sp.saving" :disabled="!canSaveSuper" @click="saveSuperPwd">保存</UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- 撤销记录 -->
    <div class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-hidden">
      <div class="flex items-center justify-between px-5 py-3 border-b border-stone-200/60 dark:border-stone-800">
        <h3 class="text-sm font-medium">撤销记录</h3>
        <div class="flex items-center gap-1">
          <UButton
            v-for="r in rangeOptions" :key="r.key"
            size="xs"
            :variant="range === r.key ? 'soft' : 'ghost'"
            :color="range === r.key ? 'primary' : 'neutral'"
            class="active:scale-95 transition-transform"
            @click="switchRange(r.key)"
          >{{ r.label }}</UButton>
        </div>
      </div>
      <div v-if="!voidLoading && voidItems.length === 0" class="py-12 text-center text-sm text-stone-400">暂无撤销记录</div>
      <div v-else-if="voidItems.length > 0" class="overflow-x-auto">
      <table class="w-full min-w-[720px] text-sm">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-44">撤销时间</th>
            <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-20">交易类型</th>
            <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-32">会员</th>
            <th class="text-right px-4 py-2.5 font-medium whitespace-nowrap w-24">金额</th>
            <th class="text-left px-4 py-2.5 font-medium">原因</th>
            <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-32">操作人</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="v in voidItems" :key="v.id" class="border-t border-stone-100 dark:border-stone-800/60 even:bg-stone-50/40 dark:even:bg-stone-950/30 hover:bg-primary-50/30 dark:hover:bg-primary-950/10 transition-colors">
            <td class="px-4 py-2 text-stone-500 tabular-nums whitespace-nowrap">{{ formatTime(v.voided_at || v.transaction_time) }}</td>
            <td class="px-4 py-2"><UBadge :label="kindLabel(v.kind)" size="md" variant="soft" color="neutral" /></td>
            <td class="px-4 py-2">{{ v.member_name || '—' }}</td>
            <td class="px-4 py-2 text-right tabular-nums font-medium">¥{{ v.actual_paid_amount }}</td>
            <td class="px-4 py-2 text-stone-500">
              <UTooltip v-if="v.void_reason" :text="v.void_reason" :delay-duration="200" :ui="{ content: 'max-w-xs whitespace-normal' }">
                <span class="block truncate max-w-64">{{ v.void_reason }}</span>
              </UTooltip>
              <span v-else>—</span>
            </td>
            <td class="px-4 py-2 text-stone-500">{{ v.voided_by_name || '—' }}</td>
          </tr>
        </tbody>
      </table>
      </div>
      <div v-if="voidLoading" class="py-8 text-center text-sm text-stone-400">加载中…</div>
      <div v-if="voidHasMore" class="px-4 py-2.5 border-t border-stone-200/60 dark:border-stone-800 text-center">
        <UButton variant="ghost" color="neutral" size="xs" @click="loadMore">
          加载更多（剩余 {{ voidTotal - voidItems.length }} 条）
        </UButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ middleware: 'super-admin' })

interface VoidedTx {
  id: string
  kind: string
  actual_paid_amount: string
  member_id: string | null
  member_name: string | null
  transaction_time: string
  voided_at: string | null
  void_reason: string | null
  voided_by_name: string | null
}

const api = useApi()
const voidGlobal = useVoidEnabled()

// ---- 开关状态：单一数据源走 useVoidEnabled ----
// canVoid / remainingSec 均为全局 reactive state
const enabled = voidGlobal.canVoid
const remaining = voidGlobal.remainingSec
const loading = ref(false)
const pendingEnable = ref(false)
const superPwd = ref('')
const err = ref('')
const superSet = ref(false)

// ---- 超级密码编辑 ----
const toast = useToast()
const spOpen = ref(false)
const sp = reactive({ login: '', next: '', confirm: '', saving: false })
const canSaveSuper = computed(() => !!sp.login && sp.next.length >= 8 && sp.next === sp.confirm)

async function saveSuperPwd() {
  sp.saving = true
  try {
    await api('/api/tenant-settings/super-password', {
      method: 'POST',
      body: { current_password: sp.login, new_password: sp.next },
    })
    superSet.value = true
    sp.login = ''; sp.next = ''; sp.confirm = ''
    spOpen.value = false
    toast.add({ title: '超级密码已保存', color: 'success', icon: 'i-lucide-check' })
  } catch (e: any) {
    toast.add({ title: '设置失败', description: e?.data?.message, color: 'error', icon: 'i-lucide-alert-triangle' })
  } finally { sp.saving = false }
}

watch(spOpen, (v) => {
  if (v) { sp.login = ''; sp.next = ''; sp.confirm = '' }
})

async function fetchState() {
  await voidGlobal.refresh()
  try {
    const s = await api<{ is_set: boolean }>('/api/tenant-settings/super-password/status')
    superSet.value = s.is_set
  } catch {}
}

function formatRemaining(sec: number) {
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

function onToggle(v: boolean) {
  if (v) {
    pendingEnable.value = true
    err.value = ''
  } else {
    disable()
  }
}
function cancelPending() {
  pendingEnable.value = false
  superPwd.value = ''
  err.value = ''
}

async function enable() {
  err.value = ''; loading.value = true
  try {
    await api<{ enabled: boolean; enabled_at: string }>('/api/tenant-settings/enable-void', {
      method: 'POST',
      body: { password: superPwd.value },
    })
    superPwd.value = ''
    pendingEnable.value = false
    await voidGlobal.refresh()
  } catch (e: any) {
    err.value = e?.data?.message || '验证失败'
  } finally { loading.value = false }
}

async function disable() {
  loading.value = true
  try {
    await api('/api/tenant-settings/disable-void', { method: 'POST' })
    await voidGlobal.refresh()
  } catch {} finally { loading.value = false }
}

// ---- 撤销记录列表 ----
const rangeOptions = [
  { key: '30d' as const, label: '近 30 天' },
  { key: 'all' as const, label: '全部' },
]
const range = ref<'30d' | 'all'>('30d')
const PAGE = 20
const voidItems = ref<VoidedTx[]>([])
const voidTotal = ref(0)
const voidLoading = ref(false)
const voidPage = ref(1)
const voidHasMore = computed(() => voidItems.value.length < voidTotal.value)

async function fetchVoided(reset = false) {
  if (reset) { voidPage.value = 1; voidItems.value = [] }
  voidLoading.value = true
  try {
    const q = new URLSearchParams({
      page: String(voidPage.value),
      limit: String(PAGE),
      status: 'voided',
    })
    if (range.value === '30d') {
      const start = new Date(); start.setDate(start.getDate() - 30)
      q.set('start_date', start.toISOString())
    }
    const data = await api<{ items: VoidedTx[]; total: number }>(`/api/transactions?${q}`)
    voidItems.value = reset ? data.items : [...voidItems.value, ...data.items]
    voidTotal.value = data.total
    voidPage.value++
  } finally { voidLoading.value = false }
}
function switchRange(key: '30d' | 'all') {
  range.value = key
  fetchVoided(true)
}
function loadMore() { fetchVoided() }

function kindLabel(k: string) { return (TX_KIND_LABEL as Record<string, string>)[k] ?? k }

onMounted(() => {
  fetchState()
  fetchVoided(true)
})
</script>
