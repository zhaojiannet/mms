<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <p class="text-sm text-stone-500">共 {{ items.length }} 人</p>
      <UButton icon="i-lucide-user-plus" @click="openCreate">新建员工</UButton>
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-x-auto">
      <table class="w-full min-w-[720px] text-base">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-3 font-medium">姓名</th>
            <th class="text-left px-4 py-3 font-medium">职位</th>
            <th class="text-left px-4 py-3 font-medium">手机</th>
            <th class="text-center px-4 py-3 font-medium">计提</th>
            <th class="text-center px-4 py-3 font-medium">提成率</th>
            <th class="text-center px-4 py-3 font-medium">排序</th>
            <th class="text-center px-4 py-3 font-medium">状态</th>
            <th class="text-right px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
          <tr v-for="s in items" :key="s.id" class="hover:bg-stone-50/60 dark:hover:bg-stone-800/40">
            <td class="px-4 py-3 font-medium">{{ s.name }}</td>
            <td class="px-4 py-3 text-stone-600 dark:text-stone-400">{{ s.position }}</td>
            <td class="px-4 py-3 text-stone-500 tabular-nums">{{ s.phone || '—' }}</td>
            <td class="px-4 py-3 text-center">
              <UBadge
                :label="s.counts_commission ? '计' : '不计'"
                :color="s.counts_commission ? 'success' : 'neutral'"
                variant="soft"
                size="sm"
              />
            </td>
            <td class="px-4 py-3 text-center text-sm tabular-nums">
              <span v-if="s.counts_commission">{{ (parseFloat(s.default_commission_rate) * 100).toFixed(1) }}%</span>
              <span v-else class="text-stone-400">—</span>
            </td>
            <td class="px-4 py-3 text-center text-stone-500 tabular-nums text-sm">{{ s.sort_order }}</td>
            <td class="px-4 py-3 text-center">
              <UBadge :label="s.status === 'active' ? '在职' : '离职'" :color="s.status === 'active' ? 'success' : 'neutral'" variant="soft" size="sm" />
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
    <EmptyState v-else-if="!loading" icon="i-lucide-user-round" text="暂无员工" hint="点击右上角新建" />
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 3" :key="i" class="h-14 rounded-2xl" /></div>

    <UModal v-model:open="formOpen" :title="editingId ? '编辑员工' : '新增员工'" :ui="{ content: 'sm:max-w-lg' }">
      <template #body>
        <UForm :state="form" class="space-y-4" @submit="onSubmit">
          <!-- 基本信息 -->
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-3">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <UFormField label="姓名" required>
                <UInput v-model="form.name" placeholder="王小花" size="md" class="w-full" />
              </UFormField>
              <UFormField label="职位" required>
                <UInput v-model="form.position" placeholder="发型师 / 美容师 / 前台" size="md" class="w-full" />
              </UFormField>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <UFormField label="手机">
                <UInput v-model="form.phone" placeholder="13800138000" size="md" class="w-full" />
              </UFormField>
              <UFormField label="排序" help="数字越小越靠前">
                <UInput v-model="form.sort_order" type="number" size="md" class="w-full" />
              </UFormField>
            </div>
          </div>

          <!-- 提成 + 状态：一行两列 -->
          <div class="grid gap-4" :class="editingId ? 'grid-cols-2' : 'grid-cols-1'">
            <!-- 提成 -->
            <div class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900 space-y-3">
              <div class="flex items-center gap-2">
                <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
                <h3 class="text-base font-medium">提成</h3>
              </div>
              <UCheckbox v-model="form.counts_commission" label="参与提成核算" />
              <UFormField v-if="form.counts_commission" label="默认提成率" help="小数表示，如 0.5 = 5 成 / 50%">
                <UInput v-model="form.default_commission_rate" type="number" step="0.01" min="0" max="1" placeholder="0.5" size="md" class="w-full" />
              </UFormField>
            </div>

            <!-- 状态（仅编辑） -->
            <div v-if="editingId" class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900 space-y-3">
              <div class="flex items-center gap-2">
                <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
                <h3 class="text-base font-medium">状态</h3>
              </div>
              <URadioGroup v-model="form.status" :items="[{label:'在职', value:'active'},{label:'离职', value:'inactive'}]" orientation="horizontal" />
            </div>
          </div>

          <UAlert v-if="formError" :description="formError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton type="button" variant="ghost" color="neutral" @click="formOpen = false">取消</UButton>
            <UButton type="submit" :loading="submitting">保存</UButton>
          </div>
        </UForm>
      </template>
    </UModal>

    <UModal v-model:open="deleteOpen" title="删除员工" :ui="{ content: 'sm:max-w-md' }">
      <template #body><p>确认删除「<strong>{{ deleting?.name }}</strong>」？必须先将员工状态设为"离职"。</p></template>
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
interface Staff {
  id: string; name: string; position: string; phone: string | null
  status: 'active' | 'inactive'
  counts_commission: boolean; default_commission_rate: string; sort_order: number
}

const api = useApi()
const items = ref<Staff[]>([])
const loading = ref(true)

const formOpen = ref(false)
const submitting = ref(false)
const formError = ref('')
const editingId = ref<string | null>(null)
const form = reactive({
  name: '', position: '', phone: '',
  counts_commission: true, default_commission_rate: '0',
  sort_order: 99, status: 'active' as 'active' | 'inactive',
})

const deleteOpen = ref(false)
const deleting = ref<Staff | null>(null)
const deletingLoading = ref(false)

async function fetchList() {
  loading.value = true
  try {
    const data = await api<{ items: Staff[] }>('/api/staff')
    items.value = data.items
  } finally { loading.value = false }
}

function resetForm() {
  form.name = ''; form.position = ''; form.phone = ''
  form.counts_commission = true; form.default_commission_rate = '0'
  form.sort_order = 99; form.status = 'active'; formError.value = ''
}
function openCreate() { editingId.value = null; resetForm(); formOpen.value = true }
function openEdit(s: Staff) {
  editingId.value = s.id
  form.name = s.name; form.position = s.position
  form.phone = s.phone ?? ''
  form.counts_commission = s.counts_commission
  form.default_commission_rate = s.default_commission_rate
  form.sort_order = s.sort_order; form.status = s.status
  formError.value = ''; formOpen.value = true
}

async function onSubmit() {
  submitting.value = true; formError.value = ''
  try {
    const body: any = {
      name: form.name, position: form.position,
      phone: form.phone || null,
      counts_commission: form.counts_commission,
      default_commission_rate: parseFloat(form.default_commission_rate) || 0,
      sort_order: Number(form.sort_order) || 99,
    }
    if (editingId.value) {
      body.status = form.status
      await api(`/api/staff/${editingId.value}`, { method: 'PUT', body })
    } else {
      await api('/api/staff', { method: 'POST', body })
    }
    formOpen.value = false
    await fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || e?.message || '保存失败'
  } finally { submitting.value = false }
}

function confirmDelete(s: Staff) { deleting.value = s; deleteOpen.value = true }
async function onDelete() {
  if (!deleting.value) return
  deletingLoading.value = true
  try {
    await api(`/api/staff/${deleting.value.id}`, { method: 'DELETE' })
    deleteOpen.value = false; deleting.value = null
    await fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || '删除失败'
  } finally { deletingLoading.value = false }
}

onMounted(fetchList)
</script>
