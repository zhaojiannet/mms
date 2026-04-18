<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <p class="text-sm text-stone-500">共 {{ items.length }} 种方式</p>
      <UButton icon="i-lucide-plus" @click="openCreate">新建支付方式</UButton>
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-x-auto">
      <table class="w-full min-w-[520px] text-base">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-3 font-medium">名称</th>
            <th class="text-center px-4 py-3 font-medium">排序</th>
            <th class="text-center px-4 py-3 font-medium">启用</th>
            <th class="text-right px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
          <tr v-for="m in items" :key="m.id" class="hover:bg-stone-50/60 dark:hover:bg-stone-800/40">
            <td class="px-4 py-3 font-medium">{{ m.name }}</td>
            <td class="px-4 py-3 text-center text-stone-500 tabular-nums">{{ m.sort_order }}</td>
            <td class="px-4 py-3 text-center">
              <UBadge :label="m.is_active ? '启用' : '停用'" :color="m.is_active ? 'success' : 'neutral'" variant="soft" size="sm" />
            </td>
            <td class="px-4 py-3 text-right">
              <div class="inline-flex items-center gap-1.5">
                <UButton
                  size="xs" variant="soft" color="primary"
                  icon="i-lucide-pencil"
                  class="active:scale-95 transition-transform"
                  @click="openEdit(m)"
                >编辑</UButton>
                <UButton
                  size="xs" variant="soft" color="error"
                  icon="i-lucide-trash-2"
                  class="active:scale-95 transition-transform"
                  @click="confirmDelete(m)"
                >删除</UButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <EmptyState v-else-if="!loading" icon="i-lucide-wallet" text="暂无支付方式" hint="点击右上角新建" />
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 5" :key="i" class="h-14 rounded-2xl" /></div>

    <UModal v-model:open="formOpen" :title="editingId ? '编辑方式' : '新增方式'" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <UForm :state="form" class="space-y-4" @submit="onSubmit">
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-3">
            <UFormField label="名称" required>
              <UInput v-model="form.name" placeholder="如：微信、信用卡" size="md" class="w-full" />
            </UFormField>
            <UFormField label="排序" help="数字越小越靠前">
              <UInput v-model="form.sort_order" type="number" size="md" class="w-40" />
            </UFormField>
          </div>

          <div v-if="editingId" class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900">
            <div class="flex items-center gap-2 mb-3">
              <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
              <h3 class="text-base font-medium">启用状态</h3>
            </div>
            <URadioGroup v-model="form.is_active" :items="[{label:'启用', value:true},{label:'停用', value:false}]" orientation="horizontal" />
          </div>

          <UAlert v-if="formError" :description="formError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton type="button" variant="ghost" color="neutral" @click="formOpen = false">取消</UButton>
            <UButton type="submit" :loading="submitting">保存</UButton>
          </div>
        </UForm>
      </template>
    </UModal>

    <UModal v-model:open="deleteOpen" title="删除支付方式" :ui="{ content: 'sm:max-w-md' }">
      <template #body><p>确认删除「<strong>{{ deleting?.name }}</strong>」？已被交易引用的方式不能删除，可改为停用。</p></template>
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
interface PaymentMethod { id: string; name: string; sort_order: number; is_active: boolean }

const api = useApi()
const items = ref<PaymentMethod[]>([])
const loading = ref(true)

const formOpen = ref(false)
const submitting = ref(false)
const formError = ref('')
const editingId = ref<string | null>(null)
const form = reactive({ name: '', sort_order: 99, is_active: true })

const deleteOpen = ref(false)
const deleting = ref<PaymentMethod | null>(null)
const deletingLoading = ref(false)

async function fetchList() {
  loading.value = true
  try {
    const data = await api<{ items: PaymentMethod[] }>('/api/payment-methods')
    items.value = data.items
  } finally { loading.value = false }
}

function resetForm() { form.name = ''; form.sort_order = 99; form.is_active = true; formError.value = '' }
function openCreate() { editingId.value = null; resetForm(); formOpen.value = true }
function openEdit(m: PaymentMethod) {
  editingId.value = m.id
  form.name = m.name; form.sort_order = m.sort_order; form.is_active = m.is_active
  formError.value = ''; formOpen.value = true
}

async function onSubmit() {
  submitting.value = true; formError.value = ''
  try {
    const body: any = { name: form.name, sort_order: Number(form.sort_order) || 99 }
    if (editingId.value) {
      body.is_active = form.is_active
      await api(`/api/payment-methods/${editingId.value}`, { method: 'PUT', body })
    } else {
      await api('/api/payment-methods', { method: 'POST', body })
    }
    formOpen.value = false
    await fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || e?.message || '保存失败'
  } finally { submitting.value = false }
}

function confirmDelete(m: PaymentMethod) { deleting.value = m; deleteOpen.value = true }
async function onDelete() {
  if (!deleting.value) return
  deletingLoading.value = true
  try {
    await api(`/api/payment-methods/${deleting.value.id}`, { method: 'DELETE' })
    deleteOpen.value = false; deleting.value = null
    await fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || '删除失败'
  } finally { deletingLoading.value = false }
}

onMounted(fetchList)
</script>
