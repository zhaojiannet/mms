<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <p class="text-sm text-stone-500">共 {{ items.length }} 项</p>
      <UButton icon="i-lucide-plus" @click="onOpenCreate">新建项目</UButton>
    </div>

    <div
      v-if="!loading && items.length > 0"
      class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-x-auto"
    >
      <table class="w-full min-w-[640px] text-sm">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-3 font-medium">名称</th>
            <th class="text-right px-4 py-3 font-medium">价格</th>
            <th class="text-center px-4 py-3 font-medium">折扣</th>
            <th class="text-center px-4 py-3 font-medium">排序</th>
            <th class="text-center px-4 py-3 font-medium">状态</th>
            <th class="text-right px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
          <tr v-for="s in items" :key="s.id" class="hover:bg-stone-50/60 dark:hover:bg-stone-800/40">
            <td class="px-4 py-3 font-medium">{{ s.name }}</td>
            <td class="px-4 py-3 text-right tabular-nums">¥{{ s.price }}</td>
            <td class="px-4 py-3 text-center">
              <UBadge v-if="s.no_discount" label="不打折" color="warning" variant="soft" size="md" />
              <span v-else class="text-stone-400">—</span>
            </td>
            <td class="px-4 py-3 text-center text-stone-500 tabular-nums text-sm">{{ s.sort_order }}</td>
            <td class="px-4 py-3 text-center">
              <UBadge :label="s.status === 'active' ? '上架' : '下架'" :color="s.status === 'active' ? 'success' : 'neutral'" variant="soft" size="md" />
            </td>
            <td class="px-4 py-3 text-right">
              <RowActions @edit="openEdit(s)" @delete="confirmDelete(s)" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <EmptyState v-else-if="!loading" icon="i-lucide-scissors" text="暂无项目" hint="点击右上角新建" />
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 5" :key="i" class="h-14 rounded-2xl" /></div>

    <UModal v-model:open="formOpen" :title="editingId ? '编辑项目' : '新建项目'" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <UForm :state="form" class="space-y-4" @submit="onSubmit">
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-3">
            <UFormField label="名称" required>
              <UInput v-model="form.name" placeholder="剪发 / 染发 / 护理" size="md" class="w-full" />
            </UFormField>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <UFormField label="价格（元）" required>
                <UInput v-model="form.price" type="number" step="0.01" placeholder="58.00" size="md" class="w-full" />
              </UFormField>
              <UFormField label="排序" help="数字越小越靠前">
                <UInput v-model="form.sort_order" type="number" size="md" class="w-full" />
              </UFormField>
            </div>
            <UCheckbox v-model="form.no_discount" label="此项目不参与会员卡折扣" />
          </div>

          <div v-if="editingId" class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900">
            <SectionTitle class="mb-3">状态</SectionTitle>
            <URadioGroup v-model="form.status" :items="[{label:'上架', value:'active'},{label:'下架', value:'inactive'}]" orientation="horizontal" />
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
      title="删除项目"
      :target="deleting?.name"
      hint="有交易引用的项目不能删除。"
      :loading="deletingLoading"
      @confirm="onDelete"
    />
  </div>
</template>

<script setup lang="ts">
interface Service {
  id: string; name: string; price: string
  category: string | null; description: string | null
  no_discount: boolean; sort_order: number; status: 'active' | 'inactive'
}

const {
  items, loading,
  formOpen, editingId, submitting, formError,
  deleteOpen, deleting, deletingLoading,
  fetchList, openCreate, openEdit: baseOpenEdit, submit, confirmDelete, onDelete,
} = useResourceCrud<Service>('/api/services')

const form = reactive({
  name: '', price: '',
  no_discount: false, sort_order: 99,
  status: 'active' as 'active' | 'inactive',
})

function resetForm() {
  form.name = ''; form.price = ''
  form.no_discount = false; form.sort_order = 99
  form.status = 'active'
}
const onOpenCreate = () => { resetForm(); openCreate() }
function openEdit(s: Service) {
  form.name = s.name; form.price = s.price
  form.no_discount = s.no_discount
  form.sort_order = s.sort_order
  form.status = s.status
  baseOpenEdit(s)
}

async function onSubmit() {
  const body: Record<string, unknown> = {
    name: form.name,
    price: form.price,
    no_discount: form.no_discount,
    sort_order: Number(form.sort_order) || 99,
  }
  if (editingId.value) body.status = form.status
  await submit(body)
}

onMounted(fetchList)
</script>
