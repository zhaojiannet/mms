<template>
  <div class="p-6 lg:p-8 max-w-7xl mx-auto space-y-5">
    <header class="flex items-center justify-between gap-4 flex-wrap">
      <div>
        <h1 class="text-3xl font-semibold tracking-tight">预约</h1>
        <p class="mt-1 text-sm text-stone-500">今日 {{ todayCount }} 条待处理 · 共 {{ items.length }} 条</p>
      </div>
      <div class="flex items-center gap-2">
        <UInput v-model="startDate" type="date" size="md" />
        <span class="text-stone-400">→</span>
        <UInput v-model="endDate" type="date" size="md" />
        <UButton color="neutral" variant="soft" @click="fetchList">查询</UButton>
        <UButton icon="i-lucide-plus" @click="openCreate">新建预约</UButton>
      </div>
    </header>

    <div class="flex items-center gap-2 flex-wrap">
      <UBadge
        v-for="f in filters" :key="f.key"
        :color="status === f.key ? 'primary' : 'neutral'"
        :variant="status === f.key ? 'solid' : 'soft'"
        class="cursor-pointer active:scale-95 transition-transform"
        @click="status = f.key; fetchList()"
      >{{ f.label }}</UBadge>
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-hidden">
      <table class="w-full text-base">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-3 font-medium">时间</th>
            <th class="text-left px-4 py-3 font-medium">客人</th>
            <th class="text-left px-4 py-3 font-medium">手机</th>
            <th class="text-left px-4 py-3 font-medium">员工</th>
            <th class="text-center px-4 py-3 font-medium">来源</th>
            <th class="text-center px-4 py-3 font-medium">状态</th>
            <th class="text-right px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
          <tr v-for="a in items" :key="a.id" class="hover:bg-stone-50/60 dark:hover:bg-stone-800/40">
            <td class="px-4 py-3 text-stone-700 tabular-nums">{{ formatTime(a.appointment_time) }}</td>
            <td class="px-4 py-3 font-medium">{{ a.customer_name }}</td>
            <td class="px-4 py-3 text-stone-500 tabular-nums">{{ a.customer_phone }}</td>
            <td class="px-4 py-3 text-stone-500 text-sm">{{ staffName(a.assigned_staff_id) || '—' }}</td>
            <td class="px-4 py-3 text-center">
              <UBadge :label="a.source === 'booking' ? 'C端' : '管理'" variant="soft" size="sm" color="neutral" />
            </td>
            <td class="px-4 py-3 text-center">
              <UBadge :label="statusLabel(a.status)" :color="statusColor(a.status)" variant="soft" size="sm" />
            </td>
            <td class="px-4 py-3 text-right">
              <UDropdown v-if="a.status === 'pending' || a.status === 'confirmed'" :items="statusMenu(a)">
                <UButton size="xs" variant="ghost" color="neutral" icon="i-lucide-more-horizontal" />
              </UDropdown>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else-if="!loading" class="py-16 text-center text-stone-500">暂无预约</div>
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 3" :key="i" class="h-14 rounded-2xl" /></div>

    <!-- Form Drawer -->
    <USlideover v-model:open="formOpen" title="新建预约" :ui="{ content: 'w-full sm:max-w-md' }">
      <template #body>
        <UForm :state="form" class="space-y-4" @submit="onSubmit">
          <UFormField label="客人姓名" required><UInput v-model="form.customer_name" class="w-full" /></UFormField>
          <UFormField label="手机号" required><UInput v-model="form.customer_phone" class="w-full" /></UFormField>
          <UFormField label="预约时间" required><UInput v-model="form.appointment_time" type="datetime-local" class="w-full" /></UFormField>
          <UFormField label="员工">
            <USelect v-model="form.assigned_staff_id" :items="staffOptions" placeholder="不指定" class="w-full" />
          </UFormField>
          <UFormField label="服务项目" required>
            <div class="space-y-1 max-h-48 overflow-y-auto border rounded-lg p-2">
              <label v-for="s in services" :key="s.id" class="flex items-center gap-2 p-1 hover:bg-stone-50 rounded">
                <UCheckbox :model-value="form.service_ids.includes(s.id)" @update:model-value="v => toggleSvc(s.id, v)" />
                <span class="text-sm">{{ s.name }} <span class="text-stone-400 tabular-nums">¥{{ s.price }}</span></span>
              </label>
            </div>
          </UFormField>
          <UFormField label="备注"><UTextarea v-model="form.notes" :rows="2" class="w-full" /></UFormField>
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
interface Appointment {
  id: string; member_id: string | null; customer_name: string; customer_phone: string
  appointment_time: string; assigned_staff_id: string | null; status: string
  source: string; notes: string | null
}
interface Staff { id: string; name: string }
interface Service { id: string; name: string; price: string }

const api = useApi()
const items = ref<Appointment[]>([])
const staff = ref<Staff[]>([])
const services = ref<Service[]>([])
const loading = ref(true)
const todayCount = ref(0)

const today = new Date()
const startDate = ref(fmtDate(today))
const endDate = ref(fmtDate(new Date(today.getTime() + 7 * 86400000)))
const status = ref<string>('')

const filters = [
  { key: '',          label: '全部' },
  { key: 'pending',   label: '待确认' },
  { key: 'confirmed', label: '已确认' },
  { key: 'completed', label: '已完成' },
  { key: 'cancelled', label: '已取消' },
  { key: 'no_show',   label: '未到店' },
]

const formOpen = ref(false)
const submitting = ref(false)
const formError = ref('')
const form = reactive({
  customer_name: '', customer_phone: '', appointment_time: '',
  assigned_staff_id: '', service_ids: [] as string[], notes: '',
})

const staffOptions = computed(() => [{ label: '不指定', value: '' }, ...staff.value.map(s => ({ label: s.name, value: s.id }))])

function fmtDate(d: Date) { return d.toISOString().slice(0, 10) }
function formatTime(s: string) { return new Date(s).toLocaleString('zh-CN', { hour12: false, month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) }
function statusLabel(s: string) { return ({ pending: '待确认', confirmed: '已确认', completed: '已完成', cancelled: '已取消', no_show: '未到店' } as any)[s] ?? s }
function statusColor(s: string): any { return ({ pending: 'warning', confirmed: 'info', completed: 'success', cancelled: 'neutral', no_show: 'error' } as any)[s] ?? 'neutral' }
function staffName(id: string | null) { return id ? staff.value.find(s => s.id === id)?.name : null }
function statusMenu(a: Appointment) {
  return [
    [
      { label: '标记已确认', icon: 'i-lucide-check', click: () => changeStatus(a, 'confirmed') },
      { label: '标记已完成', icon: 'i-lucide-check-check', click: () => changeStatus(a, 'completed') },
      { label: '标记未到店', icon: 'i-lucide-user-x', click: () => changeStatus(a, 'no_show') },
      { label: '取消预约', icon: 'i-lucide-x', click: () => changeStatus(a, 'cancelled') },
    ],
  ]
}

async function fetchList() {
  loading.value = true
  try {
    const q = new URLSearchParams({
      start_date: `${startDate.value}T00:00:00Z`,
      end_date: `${endDate.value}T23:59:59Z`,
    })
    if (status.value) q.set('status', status.value)
    const data = await api<{ items: Appointment[] }>(`/api/appointments?${q}`)
    items.value = data.items
    const td = await api<{ count: number }>('/api/appointments/count/today')
    todayCount.value = td.count
  } finally { loading.value = false }
}

async function fetchStaff() {
  const d = await api<{ items: Staff[] }>('/api/staff?status=active')
  staff.value = d.items
}
async function fetchServices() {
  const d = await api<{ items: Service[] }>('/api/services?status=active')
  services.value = d.items
}

function openCreate() {
  form.customer_name = ''; form.customer_phone = ''; form.appointment_time = ''
  form.assigned_staff_id = ''; form.service_ids = []; form.notes = ''
  formError.value = ''; formOpen.value = true
}

function toggleSvc(id: string, v: boolean | 'indeterminate') {
  const i = form.service_ids.indexOf(id)
  if (v && i < 0) form.service_ids.push(id)
  else if (!v && i >= 0) form.service_ids.splice(i, 1)
}

async function onSubmit() {
  submitting.value = true; formError.value = ''
  try {
    if (form.service_ids.length === 0) { formError.value = '至少选择一个服务'; return }
    if (!form.appointment_time) { formError.value = '请选择预约时间'; return }
    // datetime-local 格式 "2026-01-01T10:00" → 加 ":00Z"
    const apptTime = new Date(form.appointment_time).toISOString()
    const body: any = {
      customer_name: form.customer_name,
      customer_phone: form.customer_phone,
      appointment_time: apptTime,
      service_ids: form.service_ids,
    }
    if (form.assigned_staff_id) body.assigned_staff_id = form.assigned_staff_id
    if (form.notes) body.notes = form.notes
    await api('/api/appointments', { method: 'POST', body })
    formOpen.value = false
    await fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || '保存失败'
  } finally { submitting.value = false }
}

async function changeStatus(a: Appointment, status: string) {
  await api(`/api/appointments/${a.id}/status`, { method: 'PATCH', body: { status } })
  await fetchList()
}

onMounted(async () => {
  await Promise.all([fetchStaff(), fetchServices()])
  await fetchList()
})
</script>
