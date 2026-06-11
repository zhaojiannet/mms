<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <p class="text-sm text-stone-500">共 {{ items.length }} 种方式</p>
      <UButton icon="i-lucide-plus" @click="onOpenCreate">新建支付方式</UButton>
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-x-auto">
      <table class="w-full min-w-[520px] text-sm">
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
              <UBadge :label="m.is_active ? '启用' : '停用'" :color="m.is_active ? 'success' : 'neutral'" variant="soft" size="md" />
            </td>
            <td class="px-4 py-3 text-right">
              <RowActions @edit="openEdit(m)" @delete="confirmDelete(m)" />
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
            <SectionTitle class="mb-3">启用状态</SectionTitle>
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

    <DeleteConfirmModal
      v-model:open="deleteOpen"
      title="删除支付方式"
      :target="deleting?.name"
      hint="已被交易引用的方式不能删除，可改为停用。"
      :loading="deletingLoading"
      @confirm="onDelete"
    />
  </div>
</template>

<script setup lang="ts">
interface PaymentMethod { id: string; name: string; sort_order: number; is_active: boolean }

const {
  items, loading,
  formOpen, editingId, submitting, formError,
  deleteOpen, deleting, deletingLoading,
  fetchList, openCreate, openEdit: baseOpenEdit, submit, confirmDelete, onDelete,
} = useResourceCrud<PaymentMethod>('/api/payment-methods')

const form = reactive({ name: '', sort_order: 99, is_active: true })

function resetForm() { form.name = ''; form.sort_order = 99; form.is_active = true }
const onOpenCreate = () => { resetForm(); openCreate() }
function openEdit(m: PaymentMethod) {
  form.name = m.name; form.sort_order = m.sort_order; form.is_active = m.is_active
  baseOpenEdit(m)
}

async function onSubmit() {
  const body: Record<string, unknown> = { name: form.name, sort_order: Number(form.sort_order) || 99 }
  if (editingId.value) body.is_active = form.is_active
  await submit(body)
}

onMounted(fetchList)
</script>
