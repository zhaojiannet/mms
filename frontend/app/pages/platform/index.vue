<template>
  <div class="space-y-5">
    <div class="flex items-end justify-between gap-2 flex-wrap">
      <div>
        <h1 class="text-2xl sm:text-3xl font-semibold tracking-tight">商户</h1>
        <p class="text-sm text-stone-500 mt-1">共 {{ items.length }} 个商户 · 到期与状态一览</p>
      </div>
      <UButton icon="i-lucide-plus" class="active:scale-95 transition-transform" @click="openCreate">新建商户</UButton>
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-x-auto">
      <table class="w-full min-w-[760px] text-sm">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-3 font-medium">商户</th>
            <th class="text-left px-4 py-3 font-medium">子域</th>
            <th class="text-center px-4 py-3 font-medium">状态</th>
            <th class="text-center px-4 py-3 font-medium">套餐</th>
            <th class="text-left px-4 py-3 font-medium">到期日</th>
            <th class="text-right px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
          <tr v-for="t in items" :key="t.id" class="hover:bg-primary-50/30 dark:hover:bg-primary-950/10">
            <td class="px-4 py-3 font-medium">{{ t.name }}</td>
            <td class="px-4 py-3 text-stone-500 font-mono text-xs">{{ t.slug }}</td>
            <td class="px-4 py-3 text-center">
              <UBadge :label="statusLabel(t.status)" :color="t.status === 'active' ? 'success' : 'neutral'" variant="soft" size="md" />
            </td>
            <td class="px-4 py-3 text-center">
              <UBadge v-if="t.edition" :label="t.edition_name || t.edition" color="primary" variant="soft" size="md" />
              <span v-else class="text-stone-400 text-xs">未配置</span>
            </td>
            <td class="px-4 py-3 tabular-nums" :class="expiryClass(t)">{{ fmtDate(t.current_period_end) }}</td>
            <td class="px-4 py-3 text-right">
              <UDropdownMenu :items="rowMenu(t)">
                <UButton icon="i-lucide-ellipsis" variant="ghost" color="neutral" size="sm" />
              </UDropdownMenu>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <EmptyState v-else-if="!loading" icon="i-lucide-store" text="暂无商户" hint="点击右上角新建，或等待申请审批" />
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 4" :key="i" class="h-14 rounded-2xl" /></div>

    <!-- 新建商户 -->
    <UModal v-model:open="createOpen" title="新建商户" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <UForm :state="createForm" class="space-y-4" @submit="onCreate">
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-3">
            <UFormField label="店铺名" required>
              <UInput v-model="createForm.name" size="md" class="w-full" />
            </UFormField>
            <UFormField label="子域（slug）" required help="小写字母开头，仅限字母数字连字符，3-30 位">
              <UInput v-model="createForm.slug" size="md" class="w-full font-mono" placeholder="mystore" />
            </UFormField>
            <UFormField label="管理员邮箱" required>
              <UInput v-model="createForm.admin_email" type="email" size="md" class="w-full" />
            </UFormField>
            <UFormField label="管理员姓名">
              <UInput v-model="createForm.admin_name" size="md" class="w-full" placeholder="管理员" />
            </UFormField>
          </div>
          <SubscriptionFields :form="createForm" :editions="editions" />
          <UAlert v-if="createError" :description="createError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton type="button" variant="ghost" color="neutral" @click="createOpen = false">取消</UButton>
            <UButton type="submit" :loading="creating">开通</UButton>
          </div>
        </UForm>
      </template>
    </UModal>

    <!-- 续期 / 改套餐 -->
    <UModal v-model:open="renewOpen" :title="`续期 / 改套餐 · ${renewTarget?.name ?? ''}`" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <UForm :state="renewForm" class="space-y-4" @submit="onRenew">
          <SubscriptionFields :form="renewForm" :editions="editions" />
          <p class="text-xs text-stone-500">续期从当前到期日与今天的较晚者起算；改套餐即刻生效。</p>
          <UAlert v-if="renewError" :description="renewError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton type="button" variant="ghost" color="neutral" @click="renewOpen = false">取消</UButton>
            <UButton type="submit" :loading="renewing">保存</UButton>
          </div>
        </UForm>
      </template>
    </UModal>

    <!-- 重置管理员密码 -->
    <UModal v-model:open="resetOpen" :title="`重置管理员密码 · ${resetTarget?.name ?? ''}`" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <UForm :state="resetForm" class="space-y-4" @submit="onReset">
          <UFormField label="管理员邮箱" required help="必须是该商户的超级管理员邮箱">
            <UInput v-model="resetForm.email" type="email" size="md" class="w-full" />
          </UFormField>
          <UAlert v-if="resetError" :description="resetError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton type="button" variant="ghost" color="neutral" @click="resetOpen = false">取消</UButton>
            <UButton type="submit" color="warning" :loading="resetting">重置</UButton>
          </div>
        </UForm>
      </template>
    </UModal>

    <CredentialModal v-model:open="credOpen" v-bind="cred" />
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'platform' })
useHead({ title: '商户管理' })

interface TenantRow {
  id: string
  slug: string
  name: string
  status: string
  edition: string | null
  edition_name: string | null
  current_period_end: string | null
}
interface Edition { code: string; name: string }

const api = usePlatformApi()
const toast = useToast()

const items = ref<TenantRow[]>([])
const editions = ref<Edition[]>([])
const loading = ref(true)

async function fetchList() {
  loading.value = true
  try {
    const r = await api<{ items: TenantRow[] }>('/api/platform/tenants')
    items.value = r.items
  } finally {
    loading.value = false
  }
}
async function fetchEditions() {
  const r = await api<{ items: Edition[] }>('/api/platform/editions')
  editions.value = r.items
}

const statusLabel = (s: string) => ({ active: '正常', suspended: '已停用', pending: '待开通', deleted: '已删除' }[s] ?? s)
const fmtDate = (d: string | null) => (d ? new Date(d).toLocaleDateString('zh-CN') : '—')
function expiryClass(t: TenantRow) {
  if (!t.current_period_end) return 'text-stone-400'
  const days = (new Date(t.current_period_end).getTime() - Date.now()) / 86400000
  if (days < 0) return 'text-red-600 dark:text-red-400 font-medium'
  if (days <= 14) return 'text-amber-600 dark:text-amber-400 font-medium'
  return 'text-stone-600 dark:text-stone-400'
}

// ---- 行操作 ----
function rowMenu(t: TenantRow) {
  return [[
    { label: '续期 / 改套餐', icon: 'i-lucide-calendar-plus', onSelect: () => openRenew(t) },
    { label: '重置管理员密码', icon: 'i-lucide-key-round', onSelect: () => openReset(t) },
    t.status === 'active'
      ? { label: '停用', icon: 'i-lucide-ban', color: 'error' as const, onSelect: () => onToggle(t, 'suspend') }
      : { label: '恢复', icon: 'i-lucide-play', onSelect: () => onToggle(t, 'resume') },
  ]]
}

async function onToggle(t: TenantRow, action: 'suspend' | 'resume') {
  await api(`/api/platform/tenants/${t.id}/${action}`, { method: 'POST' })
  toast.add({ title: action === 'suspend' ? `已停用 ${t.name}` : `已恢复 ${t.name}`, color: 'success' })
  await fetchList()
}

// ---- 新建 ----
const createOpen = ref(false)
const creating = ref(false)
const createError = ref('')
const createForm = reactive({ name: '', slug: '', admin_email: '', admin_name: '', edition: 'plus', months: 12, source: 'manual' })

function openCreate() {
  Object.assign(createForm, { name: '', slug: '', admin_email: '', admin_name: '', edition: 'plus', months: 12, source: 'manual' })
  createError.value = ''
  createOpen.value = true
}
async function onCreate() {
  creating.value = true
  createError.value = ''
  try {
    const r = await api<{ slug: string; admin_email: string; admin_password: string }>('/api/platform/tenants', {
      method: 'POST',
      body: { ...createForm, months: Number(createForm.months) },
    })
    createOpen.value = false
    showCred(r.slug, r.admin_email, r.admin_password)
    await fetchList()
  } catch (e: any) {
    createError.value = e?.data?.message || '开通失败'
  } finally {
    creating.value = false
  }
}

// ---- 续期 ----
const renewOpen = ref(false)
const renewing = ref(false)
const renewError = ref('')
const renewTarget = ref<TenantRow | null>(null)
const renewForm = reactive({ edition: 'plus', months: 12, source: 'manual' })

function openRenew(t: TenantRow) {
  renewTarget.value = t
  Object.assign(renewForm, { edition: t.edition ?? 'plus', months: 12, source: 'manual' })
  renewError.value = ''
  renewOpen.value = true
}
async function onRenew() {
  if (!renewTarget.value) return
  renewing.value = true
  renewError.value = ''
  try {
    await api(`/api/platform/tenants/${renewTarget.value.id}/subscription`, {
      method: 'POST',
      body: { ...renewForm, months: Number(renewForm.months) },
    })
    renewOpen.value = false
    toast.add({ title: '套餐已更新', color: 'success' })
    await fetchList()
  } catch (e: any) {
    renewError.value = e?.data?.message || '操作失败'
  } finally {
    renewing.value = false
  }
}

// ---- 重置密码 ----
const resetOpen = ref(false)
const resetting = ref(false)
const resetError = ref('')
const resetTarget = ref<TenantRow | null>(null)
const resetForm = reactive({ email: '' })

function openReset(t: TenantRow) {
  resetTarget.value = t
  resetForm.email = ''
  resetError.value = ''
  resetOpen.value = true
}
async function onReset() {
  if (!resetTarget.value) return
  resetting.value = true
  resetError.value = ''
  try {
    const r = await api<{ email: string; new_password: string }>(
      `/api/platform/tenants/${resetTarget.value.id}/reset-admin-password`,
      { method: 'POST', body: { email: resetForm.email } },
    )
    resetOpen.value = false
    showCred(resetTarget.value.slug, r.email, r.new_password)
  } catch (e: any) {
    resetError.value = e?.data?.message || '重置失败'
  } finally {
    resetting.value = false
  }
}

// ---- 一次性凭证展示 ----
const credOpen = ref(false)
const cred = reactive({ slug: '', email: '', password: '' })
function showCred(slug: string, email: string, password: string) {
  Object.assign(cred, { slug, email, password })
  credOpen.value = true
}

onMounted(() => Promise.all([fetchList(), fetchEditions()]))
</script>
