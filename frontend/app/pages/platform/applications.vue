<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <UTabs
        v-model="statusTab"
        :items="[
          { label: '待审核', value: 'pending' },
          { label: '已通过', value: 'approved' },
          { label: '已拒绝', value: 'rejected' },
        ]"
        size="sm"
        :content="false"
      />
      <UButton icon="i-lucide-refresh-cw" variant="ghost" color="neutral" size="sm" @click="fetchList" />
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-x-auto">
      <table class="w-full min-w-[720px] text-sm">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-3 font-medium">店铺</th>
            <th class="text-left px-4 py-3 font-medium">行业</th>
            <th class="text-left px-4 py-3 font-medium">联系人</th>
            <th class="text-left px-4 py-3 font-medium">期望子域</th>
            <th class="text-left px-4 py-3 font-medium">申请时间</th>
            <th class="text-right px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
          <tr v-for="a in items" :key="a.id" class="hover:bg-stone-50/60 dark:hover:bg-stone-800/40">
            <td class="px-4 py-3 font-medium">{{ a.store_name }}</td>
            <td class="px-4 py-3 text-stone-500">{{ a.industry }}</td>
            <td class="px-4 py-3">{{ a.contact_name }} <span class="text-stone-400 text-xs">{{ a.phone }}</span></td>
            <td class="px-4 py-3 font-mono text-xs">{{ a.desired_slug }}</td>
            <td class="px-4 py-3 text-stone-500 tabular-nums">{{ fmtDate(a.created_at) }}</td>
            <td class="px-4 py-3 text-right">
              <template v-if="a.status === 'pending'">
                <UButton size="xs" variant="soft" @click="openApprove(a)">通过</UButton>
                <UButton size="xs" variant="ghost" color="error" class="ml-1" @click="openReject(a)">拒绝</UButton>
              </template>
              <span v-else-if="a.status === 'rejected'" class="text-xs text-stone-400" :title="a.reject_reason ?? ''">
                已拒绝{{ a.reject_reason ? `：${a.reject_reason}` : '' }}
              </span>
              <span v-else class="text-xs text-stone-400">已开通</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <EmptyState v-else-if="!loading" icon="i-lucide-inbox" text="没有申请" hint="商户在主站 /apply 提交后会出现在这里" />
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 3" :key="i" class="h-14 rounded-2xl" /></div>

    <!-- 审批通过 -->
    <UModal v-model:open="approveOpen" :title="`开通商户 · ${approveTarget?.store_name ?? ''}`" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <UForm :state="approveForm" class="space-y-4" @submit="onApprove">
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-3">
            <div class="text-sm flex justify-between">
              <span class="text-stone-500">子域</span>
              <span class="font-mono text-xs">{{ approveTarget?.desired_slug }}.{{ cfg.appDomain }}</span>
            </div>
            <UFormField label="管理员邮箱" required help="商户老板的登录邮箱">
              <UInput v-model="approveForm.admin_email" type="email" size="md" class="w-full" />
            </UFormField>
            <UFormField label="管理员姓名">
              <UInput v-model="approveForm.admin_name" size="md" class="w-full" :placeholder="approveTarget?.contact_name" />
            </UFormField>
          </div>
          <SubscriptionFields :form="approveForm" :editions="editions" />
          <UAlert v-if="approveError" :description="approveError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton type="button" variant="ghost" color="neutral" @click="approveOpen = false">取消</UButton>
            <UButton type="submit" :loading="approving">开通</UButton>
          </div>
        </UForm>
      </template>
    </UModal>

    <!-- 拒绝 -->
    <UModal v-model:open="rejectOpen" :title="`拒绝申请 · ${rejectTarget?.store_name ?? ''}`" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <UForm :state="rejectForm" class="space-y-4" @submit="onReject">
          <UFormField label="拒绝原因" help="仅存档备查，不会自动发给申请人">
            <UTextarea v-model="rejectForm.reason" :rows="3" class="w-full" />
          </UFormField>
          <div class="flex justify-end gap-2 pt-2">
            <UButton type="button" variant="ghost" color="neutral" @click="rejectOpen = false">取消</UButton>
            <UButton type="submit" color="error" :loading="rejecting">拒绝</UButton>
          </div>
        </UForm>
      </template>
    </UModal>

    <CredentialModal v-model:open="credOpen" v-bind="cred" />
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'platform' })
useHead({ title: '开通申请' })

interface ApplicationRow {
  id: string
  store_name: string
  industry: string
  contact_name: string
  phone: string
  desired_slug: string
  status: string
  reject_reason: string | null
  created_at: string
}
interface Edition { code: string; name: string }

const api = usePlatformApi()
const cfg = useRuntimeConfig().public
const toast = useToast()

const statusTab = ref('pending')
const items = ref<ApplicationRow[]>([])
const editions = ref<Edition[]>([])
const loading = ref(true)

async function fetchList() {
  loading.value = true
  try {
    const r = await api<{ items: ApplicationRow[] }>('/api/platform/applications', {
      query: { status: statusTab.value },
    })
    items.value = r.items
  } finally {
    loading.value = false
  }
}
watch(statusTab, fetchList)

const fmtDate = (d: string) => new Date(d).toLocaleString('zh-CN', { dateStyle: 'short', timeStyle: 'short' })

// ---- 审批 ----
const approveOpen = ref(false)
const approving = ref(false)
const approveError = ref('')
const approveTarget = ref<ApplicationRow | null>(null)
const approveForm = reactive({ admin_email: '', admin_name: '', edition: 'free', months: 12, source: 'manual' })

function openApprove(a: ApplicationRow) {
  approveTarget.value = a
  Object.assign(approveForm, { admin_email: '', admin_name: a.contact_name, edition: 'free', months: 12, source: 'manual' })
  approveError.value = ''
  approveOpen.value = true
}
async function onApprove() {
  if (!approveTarget.value) return
  approving.value = true
  approveError.value = ''
  try {
    const r = await api<{ slug: string; admin_email: string; admin_password: string }>(
      `/api/platform/applications/${approveTarget.value.id}/approve`,
      { method: 'POST', body: { ...approveForm, months: Number(approveForm.months) } },
    )
    approveOpen.value = false
    Object.assign(cred, { slug: r.slug, email: r.admin_email, password: r.admin_password })
    credOpen.value = true
    await fetchList()
  } catch (e: any) {
    approveError.value = e?.data?.message || '开通失败'
  } finally {
    approving.value = false
  }
}

// ---- 拒绝 ----
const rejectOpen = ref(false)
const rejecting = ref(false)
const rejectTarget = ref<ApplicationRow | null>(null)
const rejectForm = reactive({ reason: '' })

function openReject(a: ApplicationRow) {
  rejectTarget.value = a
  rejectForm.reason = ''
  rejectOpen.value = true
}
async function onReject() {
  if (!rejectTarget.value) return
  rejecting.value = true
  try {
    await api(`/api/platform/applications/${rejectTarget.value.id}/reject`, {
      method: 'POST',
      body: { reason: rejectForm.reason },
    })
    rejectOpen.value = false
    toast.add({ title: '已拒绝', color: 'success' })
    await fetchList()
  } finally {
    rejecting.value = false
  }
}

// ---- 凭证展示 ----
const credOpen = ref(false)
const cred = reactive({ slug: '', email: '', password: '' })

onMounted(() => Promise.all([fetchList(), api<{ items: Edition[] }>('/api/platform/editions').then(r => (editions.value = r.items))]))
</script>
