<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <p class="text-sm text-stone-500">共 {{ items.length }} 条规则；优先级：员工×服务 → 员工通用 → staff.default → 0</p>
      <UButton icon="i-lucide-plus" @click="openCreate" :disabled="staff.length === 0">新增规则</UButton>
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-hidden">
      <table class="w-full text-base">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-3 font-medium">员工</th>
            <th class="text-left px-4 py-3 font-medium">服务</th>
            <th class="text-right px-4 py-3 font-medium">提成率</th>
            <th class="text-right px-4 py-3 font-medium">固定金额</th>
            <th class="text-left px-4 py-3 font-medium">备注</th>
            <th class="text-right px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
          <tr v-for="r in items" :key="r.id" class="hover:bg-stone-50/60 dark:hover:bg-stone-800/40">
            <td class="px-4 py-3 font-medium">{{ r.staff_name }}</td>
            <td class="px-4 py-3 text-stone-600">
              <span v-if="r.service_name">{{ r.service_name }}</span>
              <UBadge v-else label="员工通用" color="info" variant="soft" size="sm" />
            </td>
            <td class="px-4 py-3 text-right tabular-nums">
              <span v-if="r.rate">{{ (parseFloat(r.rate) * 100).toFixed(1) }}%</span>
              <span v-else class="text-stone-400">—</span>
            </td>
            <td class="px-4 py-3 text-right tabular-nums">
              <span v-if="r.fixed_amount">¥{{ r.fixed_amount }}</span>
              <span v-else class="text-stone-400">—</span>
            </td>
            <td class="px-4 py-3 text-sm text-stone-500">{{ r.note || '—' }}</td>
            <td class="px-4 py-3 text-right">
              <UButton size="xs" variant="ghost" color="neutral" @click="openEdit(r)">编辑</UButton>
              <UButton size="xs" variant="ghost" color="error" @click="onDelete(r)">删除</UButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else-if="!loading" class="py-16 text-center text-stone-500">暂无规则</div>
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 3" :key="i" class="h-14 rounded-2xl" /></div>

    <USlideover v-model:open="formOpen" :title="editingId ? '编辑规则' : '新建规则'" :ui="{ content: 'w-full sm:max-w-md' }">
      <template #body>
        <UForm :state="form" class="space-y-4" @submit="onSubmit">
          <UFormField label="员工" required>
            <USelect v-model="form.staff_id" :items="staffOptions" :disabled="!!editingId" class="w-full" />
          </UFormField>
          <UFormField label="服务（留空 = 该员工所有服务的通用规则）">
            <USelect v-model="form.service_id" :items="serviceOptions" :disabled="!!editingId" class="w-full" />
          </UFormField>
          <UFormField label="计算方式" required>
            <URadioGroup v-model="form.kind" :items="[{label:'按比例', value:'rate'},{label:'固定金额', value:'fixed'}]" orientation="horizontal" />
          </UFormField>
          <UFormField v-if="form.kind === 'rate'" label="提成率（0-1，如 0.2 = 20%）" required>
            <UInput v-model="form.rate" type="number" step="0.001" min="0" max="1" class="w-full" />
          </UFormField>
          <UFormField v-if="form.kind === 'fixed'" label="固定金额（元）" required>
            <UInput v-model="form.fixed_amount" type="number" step="0.01" min="0" class="w-full" />
          </UFormField>
          <UFormField label="备注"><UInput v-model="form.note" class="w-full" /></UFormField>
          <UAlert v-if="formError" :description="formError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton variant="ghost" color="neutral" @click="formOpen = false">取消</UButton>
            <UButton type="submit" :loading="submitting">保存</UButton>
          </div>
        </UForm>
      </template>
    </USlideover>
  </div>
</template>

<script setup lang="ts">
interface Rule {
  id: string; staff_id: string; staff_name: string
  service_id: string | null; service_name: string | null
  rate: string | null; fixed_amount: string | null; note: string | null
}
interface Staff { id: string; name: string; position: string }
interface Service { id: string; name: string }

const api = useApi()
const items = ref<Rule[]>([])
const staff = ref<Staff[]>([])
const services = ref<Service[]>([])
const loading = ref(true)

const formOpen = ref(false)
const submitting = ref(false)
const formError = ref('')
const editingId = ref<string | null>(null)
const form = reactive({
  staff_id: '', service_id: '',
  kind: 'rate' as 'rate' | 'fixed',
  rate: '', fixed_amount: '', note: '',
})

const staffOptions = computed(() => staff.value.map(s => ({ label: `${s.name}（${s.position}）`, value: s.id })))
const serviceOptions = computed(() => [{ label: '员工通用', value: '' }, ...services.value.map(s => ({ label: s.name, value: s.id }))])

async function fetchAll() {
  loading.value = true
  try {
    const [r, s, sv] = await Promise.all([
      api<{ items: Rule[] }>('/api/commission-rules'),
      api<{ items: Staff[] }>('/api/staff?status=active'),
      api<{ items: Service[] }>('/api/services?status=active'),
    ])
    items.value = r.items; staff.value = s.items; services.value = sv.items
  } finally { loading.value = false }
}

function resetForm() {
  form.staff_id = staff.value[0]?.id ?? ''
  form.service_id = ''; form.kind = 'rate'; form.rate = '0.2'; form.fixed_amount = ''; form.note = ''
  formError.value = ''
}
function openCreate() { editingId.value = null; resetForm(); formOpen.value = true }
function openEdit(r: Rule) {
  editingId.value = r.id
  form.staff_id = r.staff_id; form.service_id = r.service_id ?? ''
  if (r.rate) { form.kind = 'rate'; form.rate = r.rate; form.fixed_amount = '' }
  else { form.kind = 'fixed'; form.fixed_amount = r.fixed_amount ?? '0'; form.rate = '' }
  form.note = r.note ?? ''; formError.value = ''
  formOpen.value = true
}

async function onSubmit() {
  submitting.value = true; formError.value = ''
  try {
    const body: any = { note: form.note || null }
    if (form.kind === 'rate') body.rate = form.rate
    else body.fixed_amount = form.fixed_amount

    if (editingId.value) {
      await api(`/api/commission-rules/${editingId.value}`, { method: 'PUT', body })
    } else {
      body.staff_id = form.staff_id
      if (form.service_id) body.service_id = form.service_id
      await api('/api/commission-rules', { method: 'POST', body })
    }
    formOpen.value = false
    await fetchAll()
  } catch (e: any) {
    formError.value = e?.data?.message || '保存失败'
  } finally { submitting.value = false }
}

async function onDelete(r: Rule) {
  if (!confirm(`删除「${r.staff_name} · ${r.service_name || '通用'}」的规则？`)) return
  await api(`/api/commission-rules/${r.id}`, { method: 'DELETE' })
  await fetchAll()
}

onMounted(fetchAll)
</script>
