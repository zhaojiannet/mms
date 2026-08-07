<template>
  <div class="space-y-4">
    <p class="text-sm text-stone-500">五档套餐的价格与限额；限额改动即刻作用于该档全部商户的后续新增，已有数据不追溯</p>

    <div v-if="!loading" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-x-auto">
      <table class="w-full min-w-[720px] text-sm">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-3 font-medium">套餐</th>
            <th class="text-right px-4 py-3 font-medium">月付</th>
            <th class="text-right px-4 py-3 font-medium">年付</th>
            <th class="text-right px-4 py-3 font-medium">会员上限</th>
            <th class="text-right px-4 py-3 font-medium">员工上限</th>
            <th class="text-right px-4 py-3 font-medium">门店上限</th>
            <th class="text-right px-4 py-3 font-medium">推送/月</th>
            <th class="text-right px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
          <tr v-for="e in items" :key="e.code" class="hover:bg-stone-50/60 dark:hover:bg-stone-800/40">
            <td class="px-4 py-3">
              <span class="font-medium">{{ e.name }}</span>
              <span class="ml-2 text-xs text-stone-400 font-mono">{{ e.code }}</span>
            </td>
            <td class="px-4 py-3 text-right tabular-nums">{{ fmtPrice(e.price_monthly) }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ fmtPrice(e.price_yearly) }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ fmtQuota(e.quotas.max_members) }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ fmtQuota(e.quotas.max_staff) }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ fmtQuota(e.quotas.max_branches) }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ fmtQuota(e.quotas.push_monthly) }}</td>
            <td class="px-4 py-3 text-right">
              <UButton size="xs" variant="soft" icon="i-lucide-pencil" @click="openEdit(e)">编辑</UButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="space-y-2"><USkeleton v-for="i in 5" :key="i" class="h-14 rounded-2xl" /></div>

    <UModal v-model:open="formOpen" :title="`编辑套餐 · ${editing?.name ?? ''}`" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <UForm :state="form" class="space-y-4" @submit="onSubmit">
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-3">
            <SectionTitle>价格（元）</SectionTitle>
            <div class="flex gap-3">
              <UFormField label="月付" class="flex-1">
                <UInput v-model="form.price_monthly" type="number" step="0.1" min="0" size="md" class="w-full" />
              </UFormField>
              <UFormField label="年付" class="flex-1">
                <UInput v-model="form.price_yearly" type="number" step="0.1" min="0" size="md" class="w-full" />
              </UFormField>
            </div>
          </div>
          <div class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900 space-y-3">
            <SectionTitle>限额（0 = 不限）</SectionTitle>
            <div class="grid grid-cols-2 gap-3">
              <UFormField label="会员上限">
                <UInput v-model="form.max_members" type="number" min="0" size="md" class="w-full" />
              </UFormField>
              <UFormField label="员工上限">
                <UInput v-model="form.max_staff" type="number" min="0" size="md" class="w-full" />
              </UFormField>
              <UFormField label="门店上限">
                <UInput v-model="form.max_branches" type="number" min="0" size="md" class="w-full" />
              </UFormField>
              <UFormField label="推送/月">
                <UInput v-model="form.push_monthly" type="number" min="0" size="md" class="w-full" />
              </UFormField>
            </div>
            <p class="text-xs text-stone-500">会员与员工上限即刻执行；门店与推送待对应功能上线后生效。</p>
          </div>
          <UAlert v-if="formError" :description="formError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton type="button" variant="ghost" color="neutral" @click="formOpen = false">取消</UButton>
            <UButton type="submit" :loading="submitting">保存</UButton>
          </div>
        </UForm>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'platform' })
useHead({ title: '套餐管理' })

interface Edition {
  code: string
  name: string
  price_monthly: string | null
  price_yearly: string | null
  quotas: Record<string, number>
}

const api = usePlatformApi()
const toast = useToast()

const items = ref<Edition[]>([])
const loading = ref(true)

async function fetchList() {
  loading.value = true
  try {
    const r = await api<{ items: Edition[] }>('/api/platform/editions')
    items.value = r.items
  } finally {
    loading.value = false
  }
}

const fmtPrice = (p: string | null) => (p == null ? '—' : `¥${p}`)
const fmtQuota = (n: number | undefined) => (!n || n <= 0 ? '不限' : n.toLocaleString())

const formOpen = ref(false)
const submitting = ref(false)
const formError = ref('')
const editing = ref<Edition | null>(null)
const form = reactive({ price_monthly: '', price_yearly: '', max_members: 0, max_staff: 0, max_branches: 0, push_monthly: 0 })

function openEdit(e: Edition) {
  editing.value = e
  form.price_monthly = e.price_monthly ?? ''
  form.price_yearly = e.price_yearly ?? ''
  // 旧种子里 -1 也表示"不限"，统一归到 0（后端只收非负数）
  const clamp = (n: number | undefined) => (!n || n < 0 ? 0 : n)
  form.max_members = clamp(e.quotas.max_members)
  form.max_staff = clamp(e.quotas.max_staff)
  form.max_branches = clamp(e.quotas.max_branches)
  form.push_monthly = clamp(e.quotas.push_monthly)
  formError.value = ''
  formOpen.value = true
}

async function onSubmit() {
  if (!editing.value) return
  submitting.value = true
  formError.value = ''
  try {
    await api(`/api/platform/editions/${editing.value.code}`, {
      method: 'PUT',
      body: {
        price_monthly: form.price_monthly === '' ? null : String(form.price_monthly),
        price_yearly: form.price_yearly === '' ? null : String(form.price_yearly),
        quotas: {
          max_members: Number(form.max_members) || 0,
          max_staff: Number(form.max_staff) || 0,
          max_branches: Number(form.max_branches) || 0,
          push_monthly: Number(form.push_monthly) || 0,
        },
      },
    })
    formOpen.value = false
    toast.add({ title: '套餐已更新', color: 'success' })
    await fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || '保存失败'
  } finally {
    submitting.value = false
  }
}

onMounted(fetchList)
</script>
