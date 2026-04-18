<template>
  <div class="p-6 lg:p-8 max-w-7xl mx-auto space-y-5">
    <!-- Head -->
    <header class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-4">
      <div>
        <h1 class="text-2xl sm:text-3xl font-semibold tracking-tight">预约</h1>
        <p class="mt-1 text-sm text-stone-500">
          今日 {{ todayCount }} 条待处理 · 共 {{ items.length }} 条
        </p>
      </div>
      <div class="flex items-center gap-2 w-full sm:w-auto">
        <UPopover :ui="{ content: 'p-2 w-auto' }" class="flex-1 sm:flex-none">
          <UInputDate
            :model-value="rangeCalValue"
            @update:model-value="onRangeUpdate"
            range
            size="sm"
            icon="i-lucide-calendar"
            class="!h-8 !py-0 !text-sm w-full"
            :ui="{
              leadingIcon: 'size-4 text-stone-400',
            }"
          />
          <template #content>
            <UCalendar :model-value="rangeCalValue" @update:model-value="onRangeUpdate" class="p-2" range />
          </template>
        </UPopover>
        <UButton icon="i-lucide-plus" class="shrink-0 whitespace-nowrap" :aria-label="'新建预约'" @click="openCreate">
          <span class="hidden sm:inline">新建预约</span>
        </UButton>
      </div>
    </header>

    <!-- 快捷日期范围 -->
    <div class="flex items-center gap-2 flex-wrap">
      <UBadge
        v-for="p in datePresets" :key="p.key"
        :color="datePreset === p.key ? 'primary' : 'neutral'"
        :variant="datePreset === p.key ? 'solid' : 'soft'"
        class="cursor-pointer active:scale-95 transition-transform"
        @click="applyDatePreset(p.key)"
      >{{ p.label }}</UBadge>
    </div>

    <!-- 状态筛选 -->
    <div class="flex items-center gap-2 flex-wrap">
      <UBadge
        v-for="f in statusFilters" :key="f.key"
        :color="status === f.key ? 'primary' : 'neutral'"
        :variant="status === f.key ? 'solid' : 'soft'"
        class="cursor-pointer active:scale-95 transition-transform"
        @click="status = f.key; fetchList()"
      >{{ f.label }}</UBadge>
    </div>

    <!-- Desktop: UTable -->
    <div
      v-if="!loading && items.length > 0"
      class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-x-auto"
    >
      <UTable
        :data="items"
        :columns="columns"
        :ui="{
          thead: 'bg-stone-50/60 dark:bg-stone-950/40',
          th: 'px-3.5 py-2.5 text-xs text-stone-500 font-medium tracking-wide whitespace-nowrap',
          td: 'px-3.5 py-3 text-sm whitespace-nowrap',
          tr: 'even:bg-stone-50 dark:even:bg-stone-800/30 hover:bg-stone-100/60 dark:hover:bg-stone-800/40 transition-colors',
        }"
      >
        <template #appointment_time-cell="{ row }">
          <button type="button" class="text-left text-stone-900 dark:text-stone-100 font-medium tabular-nums hover:text-primary-600 transition-colors cursor-pointer" @click="openDetail(row.original)">
            {{ formatDateTime(row.original.appointment_time) }}
          </button>
        </template>

        <template #customer-cell="{ row }">
          <div class="min-w-0">
            <div class="font-medium text-stone-900 dark:text-stone-100 truncate">{{ row.original.customer_name || '—' }}</div>
            <div class="text-xs text-stone-500 tabular-nums mt-0.5">{{ row.original.customer_phone || '未留手机' }}</div>
          </div>
        </template>

        <template #services-cell="{ row }">
          <div class="text-stone-600 dark:text-stone-400 truncate max-w-56" :title="serviceNames(row.original.service_ids).join('、')">
            {{ serviceNames(row.original.service_ids).join('、') || '—' }}
          </div>
        </template>

        <template #staff-cell="{ row }">
          <span class="text-stone-600 dark:text-stone-400">{{ staffName(row.original.assigned_staff_id) || '—' }}</span>
        </template>

        <template #source-cell="{ row }">
          <span class="text-stone-500">{{ row.original.source === 'booking' ? 'C 端' : '管理' }}</span>
        </template>

        <template #status-cell="{ row }">
          <span class="inline-flex items-center gap-1.5">
            <span class="size-1.5 rounded-full" :class="statusDotClass(row.original.status)" />
            <span class="text-stone-700 dark:text-stone-300">{{ statusLabel(row.original.status) }}</span>
          </span>
        </template>

        <template #actions-header>
          <div class="text-center">操作</div>
        </template>
        <template #actions-cell="{ row }">
          <div class="text-center">
            <UButton
              size="xs" variant="soft" color="primary"
              icon="i-lucide-eye"
              class="active:scale-95 transition-transform"
              @click="openDetail(row.original)"
            >详情</UButton>
          </div>
        </template>
      </UTable>
    </div>

    <!-- Empty state -->
    <EmptyState v-else-if="!loading" icon="i-lucide-calendar-x" text="暂无预约" hint="该时段还没有新的预约" />

    <!-- Loading -->
    <div v-if="loading" class="space-y-2">
      <USkeleton v-for="i in 4" :key="i" class="h-14 rounded-2xl" />
    </div>

    <!-- ===== 详情 Modal ===== -->
    <UModal v-model:open="detailOpen" :title="detailItem?.customer_name || '预约详情'" :ui="{ content: 'sm:max-w-lg' }">
      <template #body>
        <div v-if="detailItem" class="space-y-4">
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800">
            <div class="flex items-center gap-2 flex-wrap">
              <h2 class="text-xl font-semibold text-stone-900 dark:text-stone-100">{{ detailItem.customer_name || '未填写' }}</h2>
              <span class="inline-flex items-center gap-1.5">
                <span class="size-1.5 rounded-full" :class="statusDotClass(detailItem.status)" />
                <span class="text-sm text-stone-600 dark:text-stone-400">{{ statusLabel(detailItem.status) }}</span>
              </span>
              <span class="text-stone-300 dark:text-stone-600">·</span>
              <span class="text-sm text-stone-500">{{ detailItem.source === 'booking' ? 'C 端' : '管理' }}</span>
            </div>
            <div class="mt-1 text-sm text-stone-600 dark:text-stone-400 tabular-nums">{{ detailItem.customer_phone || '未留手机' }}</div>
            <div class="mt-3 pt-3 border-t border-stone-200/60 dark:border-stone-800 grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-1.5 text-sm">
              <div class="flex gap-2"><span class="text-stone-500 w-16 shrink-0">预约时间</span><span class="text-stone-900 dark:text-stone-100 tabular-nums">{{ formatDateTime(detailItem.appointment_time) }}</span></div>
              <div class="flex gap-2"><span class="text-stone-500 w-16 shrink-0">服务员工</span><span class="text-stone-900 dark:text-stone-100">{{ staffName(detailItem.assigned_staff_id) || '不指定' }}</span></div>
              <div class="flex gap-2 col-span-2"><span class="text-stone-500 w-16 shrink-0">预约项目</span><span class="text-stone-900 dark:text-stone-100">{{ serviceNames(detailItem.service_ids).join('、') || '—' }}</span></div>
              <div v-if="detailItem.notes" class="flex gap-2 col-span-2"><span class="text-stone-500 w-16 shrink-0">备注</span><span class="text-stone-600 dark:text-stone-400">{{ detailItem.notes }}</span></div>
            </div>
          </div>

          <div v-if="canChangeStatus(detailItem.status)" class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900">
            <div class="flex items-center gap-2 mb-3">
              <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
              <h3 class="text-base font-medium">更改状态</h3>
            </div>
            <div class="flex flex-wrap gap-2">
              <UButton v-if="detailItem.status === 'pending'" size="sm" icon="i-lucide-check" @click="changeStatus(detailItem, 'confirmed')">标记已确认</UButton>
              <UButton v-if="detailItem.status === 'confirmed'" size="sm" icon="i-lucide-check-check" @click="changeStatus(detailItem, 'completed')">标记已完成</UButton>
              <UButton size="sm" variant="ghost" color="neutral" icon="i-lucide-user-x" @click="changeStatus(detailItem, 'no_show')">未到店</UButton>
              <UButton size="sm" variant="ghost" color="neutral" icon="i-lucide-x" @click="changeStatus(detailItem, 'cancelled')">取消预约</UButton>
            </div>
          </div>
        </div>
      </template>
    </UModal>

    <!-- ===== 新建预约 Modal ===== -->
    <UModal v-model:open="formOpen" title="新建预约" :ui="{ content: 'sm:max-w-lg' }">
      <template #body>
        <UForm :state="form" :validate="validateForm" class="space-y-4" @submit="onSubmit">
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-3">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <UFormField label="顾客姓名" name="customer_name" required>
                <UInput v-model="form.customer_name" placeholder="王小花" size="md" class="w-full" />
              </UFormField>
              <UFormField label="联系电话" name="customer_phone" required>
                <UInput v-model="form.customer_phone" placeholder="13800138000" size="md" class="w-full" />
              </UFormField>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <UFormField label="预约时间" name="appointment_time" required>
                <UPopover :ui="{ content: 'p-2 w-auto' }">
                  <UInputDate :model-value="apptDateTimeVal" @update:model-value="onApptDateTimeUpdate" granularity="minute" size="md" class="w-full" icon="i-lucide-calendar-clock" />
                  <template #content>
                    <UCalendar :model-value="apptCalDate" @update:model-value="onApptCalDateSelect" class="p-2" />
                  </template>
                </UPopover>
              </UFormField>
              <UFormField label="服务员工">
                <USelect v-model="form.assigned_staff_id" :items="staffOptions" placeholder="不指定" class="w-full" />
              </UFormField>
            </div>
          </div>

          <div class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900">
            <div class="flex items-center gap-2 mb-3">
              <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
              <h3 class="text-base font-medium">预约项目</h3>
              <span class="text-xs text-stone-400 ml-auto">{{ form.service_ids.length > 0 ? `已选 ${form.service_ids.length} 项` : '' }}</span>
            </div>
            <UInput
              v-if="services.length > 6"
              v-model="svcSearch"
              icon="i-lucide-search"
              placeholder="搜索服务名称或金额"
              size="sm"
              class="w-full mb-2"
              :ui="{ trailing: 'pe-1' }"
            >
              <template v-if="svcSearch?.length" #trailing>
                <UButton
                  color="neutral"
                  variant="link"
                  size="sm"
                  icon="i-lucide-circle-x"
                  aria-label="清空搜索"
                  @click="svcSearch = ''"
                />
              </template>
            </UInput>
            <div class="space-y-1 max-h-48 overflow-y-auto rounded-lg ring-1 ring-stone-200 dark:ring-stone-800 p-2">
              <label v-for="s in filteredServices" :key="s.id" class="flex items-center gap-2 px-2 py-1.5 hover:bg-stone-50 dark:hover:bg-stone-800/40 rounded cursor-pointer">
                <UCheckbox :model-value="form.service_ids.includes(s.id)" @update:model-value="v => toggleSvc(s.id, v)" />
                <span class="text-sm flex-1">{{ s.name }}</span>
                <span class="text-xs text-stone-400 tabular-nums">¥{{ s.price }}</span>
              </label>
              <div v-if="filteredServices.length === 0" class="py-3 text-center text-sm text-stone-400">没有匹配的服务</div>
            </div>
          </div>

          <UFormField label="备注">
            <UTextarea v-model="form.notes" placeholder="特殊要求 / 提醒事项" :rows="2" class="w-full" />
          </UFormField>
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
import type { TableColumn } from '@nuxt/ui'
import { CalendarDate, CalendarDateTime } from '@internationalized/date'

interface Appointment {
  id: string
  member_id: string | null
  customer_name: string
  customer_phone: string
  appointment_time: string
  assigned_staff_id: string | null
  status: string
  source: string
  notes: string | null
  service_ids?: string[]
}
interface Staff { id: string; name: string }
interface Service { id: string; name: string; price: string }

const api = useApi()
const items = ref<Appointment[]>([])
const staff = ref<Staff[]>([])
const services = ref<Service[]>([])
const loading = ref(true)
const todayCount = ref(0)

// ---- 日期范围 ----
const today = new Date()
const startDate = ref(fmtDate(today))
const endDate = ref(fmtDate(new Date(today.getTime() + 7 * 86400000)))
const datePreset = ref<'today' | 'week' | 'month' | 'custom'>('week')

const datePresets = [
  { key: 'today' as const, label: '今日' },
  { key: 'week'  as const, label: '本周' },
  { key: 'month' as const, label: '本月' },
]

let suppressRangeUpdate = false
function applyDatePreset(key: 'today' | 'week' | 'month') {
  suppressRangeUpdate = true
  datePreset.value = key
  const now = new Date()
  if (key === 'today') {
    startDate.value = fmtDate(now)
    endDate.value = fmtDate(now)
  } else if (key === 'week') {
    const day = now.getDay() || 7
    const monday = new Date(now); monday.setDate(now.getDate() - day + 1)
    const sunday = new Date(monday); sunday.setDate(monday.getDate() + 6)
    startDate.value = fmtDate(monday)
    endDate.value = fmtDate(sunday)
  } else {
    startDate.value = fmtDate(new Date(now.getFullYear(), now.getMonth(), 1))
    endDate.value = fmtDate(new Date(now.getFullYear(), now.getMonth() + 1, 0))
  }
  nextTick(() => { suppressRangeUpdate = false })
  fetchList()
}

// ---- 状态筛选 ----
const status = ref<string>('')
const statusFilters = [
  { key: '',          label: '全部' },
  { key: 'pending',   label: '待确认' },
  { key: 'confirmed', label: '已确认' },
  { key: 'completed', label: '已完成' },
  { key: 'cancelled', label: '已取消' },
  { key: 'no_show',   label: '未到店' },
]

// ---- 表格列 ----
const columns: TableColumn<Appointment>[] = [
  { accessorKey: 'appointment_time', header: '时间' },
  { id: 'customer',                  header: '顾客' },
  { id: 'services',                  header: '预约项目' },
  { id: 'staff',                     header: '员工' },
  { id: 'source',                    header: '来源' },
  { accessorKey: 'status',           header: '状态' },
  { id: 'actions',                   header: '操作', size: 70, meta: { class: { td: 'text-center w-28', th: 'text-center w-28' } } },
]

const staffOptions = computed(() =>
  staff.value.map(s => ({ label: s.name, value: s.id }))
)

function fmtDate(d: Date) { return d.toISOString().slice(0, 10) }
function formatDateTime(s: string) {
  const d = new Date(s)
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  return `${m}/${day} ${hh}:${mm}`
}
function statusLabel(s: string) {
  return ({ pending: '待确认', confirmed: '已确认', completed: '已完成', cancelled: '已取消', no_show: '未到店' } as any)[s] ?? s
}
function statusDotClass(s: string): string {
  return (s === 'pending' || s === 'confirmed')
    ? 'bg-primary-500'
    : 'bg-stone-400 dark:bg-stone-500'
}
function staffName(id: string | null) {
  return id ? staff.value.find(s => s.id === id)?.name : null
}
function serviceNames(ids: string[] | undefined) {
  if (!ids) return []
  return ids.map(id => services.value.find(s => s.id === id)?.name).filter(Boolean) as string[]
}
function canChangeStatus(s: string) {
  return s === 'pending' || s === 'confirmed'
}

// ---- 日历选择器 ----
function calToStr(d: any): string {
  if (typeof d.toString === 'function') return d.toString()
  return `${d.year}-${String(d.month).padStart(2, '0')}-${String(d.day).padStart(2, '0')}`
}
function strToCal(s: string): CalendarDate | undefined {
  if (!s) return undefined
  const [y, m, d] = s.split('-').map(Number)
  if (!y || !m || !d) return undefined
  return new CalendarDate(y, m, d)
}

const rangeCalValue = computed(() => {
  const s = strToCal(startDate.value)
  const e = strToCal(endDate.value)
  return (s && e) ? { start: s, end: e } : undefined
})
function onRangeUpdate(v: any) {
  if (suppressRangeUpdate) return
  if (!v?.start || !v?.end) return
  const newStart = calToStr(v.start)
  const newEnd = calToStr(v.end)
  if (newStart === startDate.value && newEnd === endDate.value) return
  startDate.value = newStart
  endDate.value = newEnd
  datePreset.value = 'custom'
  fetchList()
}

const apptDateTimeVal = computed(() => {
  if (!form.appointment_time) return undefined
  const [datePart, timePart] = form.appointment_time.split('T')
  const [y, m, d] = datePart.split('-').map(Number)
  const [hh, mm] = (timePart || '10:00').split(':').map(Number)
  if (!y || !m || !d) return undefined
  return new CalendarDateTime(y, m, d, hh || 0, mm || 0)
})
const apptCalDate = computed(() => {
  const v = apptDateTimeVal.value
  if (!v) return undefined
  return new CalendarDate(v.year, v.month, v.day)
})
function onApptDateTimeUpdate(v: any) {
  if (!v) return
  form.appointment_time = `${v.year}-${String(v.month).padStart(2, '0')}-${String(v.day).padStart(2, '0')}T${String(v.hour ?? 0).padStart(2, '0')}:${String(v.minute ?? 0).padStart(2, '0')}`
}
function onApptCalDateSelect(d: any) {
  if (!d) return
  const cur = apptDateTimeVal.value
  form.appointment_time = `${d.year}-${String(d.month).padStart(2, '0')}-${String(d.day).padStart(2, '0')}T${String(cur?.hour ?? 10).padStart(2, '0')}:${String(cur?.minute ?? 0).padStart(2, '0')}`
}

// ---- 服务搜索 ----
const svcSearch = ref('')
const filteredServices = computed(() => {
  const q = svcSearch.value.trim().toLowerCase()
  if (!q) return services.value
  return services.value.filter(s => s.name.toLowerCase().includes(q) || s.price.includes(q))
})

// ---- 数据 ----
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

// ---- 详情 Modal ----
const detailOpen = ref(false)
const detailItem = ref<Appointment | null>(null)

function openDetail(a: Appointment) {
  detailItem.value = a
  detailOpen.value = true
}

async function changeStatus(a: Appointment, st: string) {
  await api(`/api/appointments/${a.id}/status`, { method: 'PATCH', body: { status: st } })
  detailOpen.value = false
  await fetchList()
}

// ---- 新建 Modal ----
const formOpen = ref(false)
const submitting = ref(false)
const formError = ref('')
const form = reactive({
  customer_name: '',
  customer_phone: '',
  appointment_time: '',
  assigned_staff_id: '',
  service_ids: [] as string[],
  notes: '',
})

function openCreate() {
  const now = new Date()
  const hh = String((now.getHours() + 1) % 24).padStart(2, '0')
  form.customer_name = ''
  form.customer_phone = ''
  form.appointment_time = `${fmtDate(now)}T${hh}:00`
  form.assigned_staff_id = ''
  form.service_ids = []
  form.notes = ''
  formError.value = ''
  svcSearch.value = ''
  formOpen.value = true
}

function toggleSvc(id: string, v: boolean | 'indeterminate') {
  const i = form.service_ids.indexOf(id)
  if (v && i < 0) form.service_ids.push(id)
  else if (!v && i >= 0) form.service_ids.splice(i, 1)
}

function validateForm(state: typeof form) {
  const errors: { name: string; message: string }[] = []
  if (!state.customer_name?.trim()) errors.push({ name: 'customer_name', message: '请填写姓名' })
  const phone = (state.customer_phone || '').trim()
  if (!phone) errors.push({ name: 'customer_phone', message: '请填写手机号' })
  else if (phone !== '00000000000' && !/^1[3-9]\d{9}$/.test(phone)) {
    errors.push({ name: 'customer_phone', message: '手机号格式不正确' })
  }
  if (!state.appointment_time) errors.push({ name: 'appointment_time', message: '请选择预约时间' })
  if (state.service_ids.length === 0) errors.push({ name: 'services', message: '至少选择一个服务' })
  return errors
}

async function onSubmit() {
  submitting.value = true
  formError.value = ''
  try {
    const apptTime = new Date(form.appointment_time).toISOString()
    const body: any = {
      customer_name: form.customer_name,
      customer_phone: form.customer_phone,
      appointment_time: apptTime,
      service_ids: form.service_ids,
      status: 'pending',
    }
    if (form.assigned_staff_id) body.assigned_staff_id = form.assigned_staff_id
    if (form.notes) body.notes = form.notes
    await api('/api/appointments', { method: 'POST', body })
    submitting.value = false
    formOpen.value = false
    fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || e?.message || '保存失败'
    submitting.value = false
  }
}

onMounted(async () => {
  await Promise.all([fetchStaff(), fetchServices()])
  applyDatePreset('week')  // 默认本周
})
</script>
