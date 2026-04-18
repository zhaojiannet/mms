<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <p class="text-sm text-stone-500">共 {{ total }} 项</p>
      <UButton icon="i-lucide-plus" @click="openCreate">新建项目</UButton>
    </div>

    <div
      v-if="!loading && items.length > 0"
      class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-x-auto"
    >
      <table class="w-full min-w-[640px] text-base">
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
              <UBadge v-if="s.no_discount" label="不打折" color="warning" variant="soft" size="sm" />
              <span v-else class="text-stone-400">—</span>
            </td>
            <td class="px-4 py-3 text-center text-stone-500 tabular-nums text-sm">{{ s.sort_order }}</td>
            <td class="px-4 py-3 text-center">
              <UBadge :label="s.status === 'active' ? '上架' : '下架'" :color="s.status === 'active' ? 'success' : 'neutral'" variant="soft" size="sm" />
            </td>
            <td class="px-4 py-3 text-right">
              <div class="inline-flex items-center gap-1.5">
                <UButton
                  size="xs" variant="soft" color="primary"
                  icon="i-lucide-pencil"
                  class="active:scale-95 transition-transform"
                  @click="openEdit(s)"
                >编辑</UButton>
                <UButton
                  size="xs" variant="soft" color="error"
                  icon="i-lucide-trash-2"
                  class="active:scale-95 transition-transform"
                  @click="confirmDelete(s)"
                >删除</UButton>
              </div>
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
            <div class="flex items-center gap-2 mb-3">
              <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
              <h3 class="text-base font-medium">状态</h3>
            </div>
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

    <UModal v-model:open="deleteOpen" title="删除项目" :ui="{ content: 'sm:max-w-md' }">
      <template #body><p>确认删除「<strong>{{ deleting?.name }}</strong>」？有交易引用的项目不能删除。</p></template>
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
interface Service {
  id: string; name: string; price: string
  category: string | null; description: string | null
  no_discount: boolean; sort_order: number; status: 'active' | 'inactive'
}

const api = useApi()
const items = ref<Service[]>([])
const total = ref(0)
const loading = ref(true)

const formOpen = ref(false)
const submitting = ref(false)
const formError = ref('')
const editingId = ref<string | null>(null)
const form = reactive({
  name: '', price: '',
  no_discount: false, sort_order: 99,
  status: 'active' as 'active' | 'inactive',
})

const deleteOpen = ref(false)
const deleting = ref<Service | null>(null)
const deletingLoading = ref(false)

async function fetchList() {
  loading.value = true
  try {
    const data = await api<{ items: Service[]; total: number }>('/api/services')
    items.value = data.items
    total.value = data.total
  } finally { loading.value = false }
}

function resetForm() {
  form.name = ''; form.price = ''
  form.no_discount = false; form.sort_order = 99
  form.status = 'active'; formError.value = ''
}
function openCreate() { editingId.value = null; resetForm(); formOpen.value = true }
function openEdit(s: Service) {
  editingId.value = s.id
  form.name = s.name; form.price = s.price
  form.no_discount = s.no_discount
  form.sort_order = s.sort_order
  form.status = s.status
  formError.value = ''
  formOpen.value = true
}

async function onSubmit() {
  submitting.value = true; formError.value = ''
  try {
    const body: any = {
      name: form.name,
      price: form.price,
      no_discount: form.no_discount,
      sort_order: Number(form.sort_order) || 99,
    }
    if (editingId.value) {
      body.status = form.status
      await api(`/api/services/${editingId.value}`, { method: 'PUT', body })
    } else {
      await api('/api/services', { method: 'POST', body })
    }
    formOpen.value = false
    await fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || e?.message || '保存失败'
  } finally { submitting.value = false }
}

function confirmDelete(s: Service) { deleting.value = s; deleteOpen.value = true }
async function onDelete() {
  if (!deleting.value) return
  deletingLoading.value = true
  try {
    await api(`/api/services/${deleting.value.id}`, { method: 'DELETE' })
    deleteOpen.value = false
    deleting.value = null
    await fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || '删除失败'
  } finally { deletingLoading.value = false }
}

onMounted(fetchList)
</script>
