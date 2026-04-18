<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <p class="text-sm text-stone-500">共 {{ items.length }} 种卡型</p>
      <UButton icon="i-lucide-plus" @click="openCreate">新建卡型</UButton>
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-x-auto">
      <table class="w-full min-w-[640px] text-base">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-3 font-medium">名称</th>
            <th class="text-right px-4 py-3 font-medium">售价</th>
            <th class="text-center px-4 py-3 font-medium">折扣</th>
            <th class="text-center px-4 py-3 font-medium">排序</th>
            <th class="text-center px-4 py-3 font-medium">状态</th>
            <th class="text-right px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
          <tr v-for="c in items" :key="c.id" class="hover:bg-stone-50/60 dark:hover:bg-stone-800/40">
            <td class="px-4 py-3 font-medium">{{ c.name }}</td>
            <td class="px-4 py-3 text-right tabular-nums">¥{{ c.price }}</td>
            <td class="px-4 py-3 text-center font-medium tabular-nums">{{ displayRate(c.discount_rate) }}</td>
            <td class="px-4 py-3 text-center text-stone-500 tabular-nums text-sm">{{ c.sort_order }}</td>
            <td class="px-4 py-3 text-center">
              <UBadge :label="c.status === 'active' ? '启用' : '停用'" :color="c.status === 'active' ? 'success' : 'neutral'" variant="soft" size="sm" />
            </td>
            <td class="px-4 py-3 text-right">
              <div class="inline-flex items-center gap-1.5">
                <UButton
                  size="xs" variant="soft" color="primary"
                  icon="i-lucide-pencil"
                  class="active:scale-95 transition-transform"
                  @click="openEdit(c)"
                >编辑</UButton>
                <UButton
                  size="xs" variant="soft" color="error"
                  icon="i-lucide-trash-2"
                  class="active:scale-95 transition-transform"
                  @click="confirmDelete(c)"
                >删除</UButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <EmptyState v-else-if="!loading" icon="i-lucide-credit-card" text="暂无会员卡类型" hint="点击右上角新建" />
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 3" :key="i" class="h-14 rounded-2xl" /></div>

    <UModal v-model:open="formOpen" :title="editingId ? '编辑卡型' : '新建卡型'" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <UForm :state="form" class="space-y-4" @submit="onSubmit">
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-3">
            <UFormField label="卡名" required>
              <UInput v-model="form.name" placeholder="如：500 元储值卡" size="md" class="w-full" />
            </UFormField>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <UFormField label="售价（元）" required>
                <UInput v-model="form.price" type="number" step="0.01" placeholder="500.00" size="md" class="w-full" />
              </UFormField>
              <UFormField label="排序" help="数字越小越靠前">
                <UInput v-model="form.sort_order" type="number" size="md" class="w-full" />
              </UFormField>
            </div>
            <UFormField label="折扣率" required>
              <UInput v-model="form.discount_rate" type="number" step="0.001" min="0.001" max="1" placeholder="0.7" size="md" class="w-full" />
              <template #help>
                <span class="text-xs text-stone-500">0-1 小数，如 0.7 表示七折 · 当前：<span class="font-medium">{{ displayRate(form.discount_rate) }}</span></span>
              </template>
            </UFormField>
          </div>

          <div v-if="editingId" class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900">
            <div class="flex items-center gap-2 mb-3">
              <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
              <h3 class="text-base font-medium">状态</h3>
            </div>
            <URadioGroup v-model="form.status" :items="[{label:'启用', value:'active'},{label:'停用', value:'inactive'}]" orientation="horizontal" />
          </div>

          <UAlert v-if="formError" :description="formError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton type="button" variant="ghost" color="neutral" @click="formOpen = false">取消</UButton>
            <UButton type="submit" :loading="submitting">保存</UButton>
          </div>
        </UForm>
      </template>
    </UModal>

    <UModal v-model:open="deleteOpen" title="删除卡型" :ui="{ content: 'sm:max-w-md' }">
      <template #body><p>确认删除「<strong>{{ deleting?.name }}</strong>」？已有会员办理过该卡型的不能删除。</p></template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton variant="ghost" color="neutral" @click="deleteOpen = false">取消</UButton>
          <UButton color="error" :loading="deletingLoading" @click="onDelete">确认删除</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
interface CardType {
  id: string; name: string; price: string
  discount_rate: string; sort_order: number
  status: 'active' | 'inactive'
}

const api = useApi()
const items = ref<CardType[]>([])
const loading = ref(true)

const formOpen = ref(false)
const submitting = ref(false)
const formError = ref('')
const editingId = ref<string | null>(null)
const form = reactive({
  name: '', price: '', discount_rate: '1',
  sort_order: 99, status: 'active' as 'active' | 'inactive',
})

const deleteOpen = ref(false)
const deleting = ref<CardType | null>(null)
const deletingLoading = ref(false)

function displayRate(r: string | number): string {
  const n = typeof r === 'string' ? parseFloat(r) : r
  if (isNaN(n) || n <= 0) return '—'
  const folds = n * 10
  return Math.round(folds * 10) / 10 + ' 折'
}

async function fetchList() {
  loading.value = true
  try {
    const data = await api<{ items: CardType[] }>('/api/card-types')
    items.value = data.items
  } finally { loading.value = false }
}

function resetForm() {
  form.name = ''; form.price = ''; form.discount_rate = '1'
  form.sort_order = 99; form.status = 'active'; formError.value = ''
}
function openCreate() { editingId.value = null; resetForm(); formOpen.value = true }
function openEdit(c: CardType) {
  editingId.value = c.id
  form.name = c.name; form.price = c.price
  form.discount_rate = c.discount_rate
  form.sort_order = c.sort_order; form.status = c.status
  formError.value = ''; formOpen.value = true
}

async function onSubmit() {
  submitting.value = true; formError.value = ''
  try {
    const body: any = {
      name: form.name,
      price: form.price,
      discount_rate: form.discount_rate,
      sort_order: Number(form.sort_order) || 99,
    }
    if (editingId.value) {
      body.status = form.status
      await api(`/api/card-types/${editingId.value}`, { method: 'PUT', body })
    } else {
      await api('/api/card-types', { method: 'POST', body })
    }
    formOpen.value = false
    await fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || e?.message || '保存失败'
  } finally { submitting.value = false }
}

function confirmDelete(c: CardType) { deleting.value = c; deleteOpen.value = true }
async function onDelete() {
  if (!deleting.value) return
  deletingLoading.value = true
  try {
    await api(`/api/card-types/${deleting.value.id}`, { method: 'DELETE' })
    deleteOpen.value = false; deleting.value = null
    await fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || '删除失败'
  } finally { deletingLoading.value = false }
}

onMounted(fetchList)
</script>
