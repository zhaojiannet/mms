<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <p class="text-sm text-stone-500">共 {{ items.length }} 种方式</p>
      <UButton icon="i-lucide-plus" @click="openCreate">新增方式</UButton>
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-hidden">
      <table class="w-full text-base">
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
              <UButton size="xs" variant="ghost" color="neutral" @click="openEdit(m)">编辑</UButton>
              <UButton size="xs" variant="ghost" color="error" @click="confirmDelete(m)">删除</UButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else-if="!loading" class="py-16 text-center text-stone-500">暂无支付方式</div>
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 5" :key="i" class="h-14 rounded-2xl" /></div>

    <USlideover v-model:open="formOpen" :title="editingId ? '编辑方式' : '新增方式'" :ui="{ content: 'w-full sm:max-w-md' }">
      <template #body>
        <UForm :state="form" class="space-y-4" @submit="onSubmit">
          <UFormField label="名称" required>
            <UInput v-model="form.name" placeholder="如：微信、老婆微信、POS" class="w-full" />
          </UFormField>
          <UFormField label="排序（数字越小越靠前）">
            <UInput v-model="form.sort_order" type="number" class="w-full" />
          </UFormField>
          <UFormField v-if="editingId" label="启用状态">
            <URadioGroup v-model="form.is_active" :items="[{label:'启用', value:true},{label:'停用', value:false}]" orientation="horizontal" />
          </UFormField>
          <UAlert v-if="formError" :description="formError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton variant="ghost" color="neutral" @click="formOpen = false">取消</UButton>
            <UButton type="submit" :loading="submitting">保存</UButton>
          </div>
        </UForm>
      </template>
    </USlideover>

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
