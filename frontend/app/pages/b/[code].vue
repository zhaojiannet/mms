<template>
  <div class="min-h-screen bg-linear-to-b from-stone-50 to-white dark:from-stone-950 dark:to-stone-900">
    <div class="max-w-lg mx-auto p-6 pt-10">
      <div class="text-center mb-8">
        <h1 class="text-3xl font-semibold">在线预约</h1>
        <p class="text-sm text-stone-500 mt-1">填写资料即可预约到店时间</p>
      </div>

      <!-- 无效 code -->
      <div v-if="invalidCode" class="p-8 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 text-center">
        <UIcon name="i-lucide-link-off" class="size-10 mx-auto text-stone-400" />
        <p class="mt-3 text-base">预约链接无效或已过期</p>
        <p class="text-sm text-stone-500 mt-1">请联系商户获取新链接</p>
      </div>

      <!-- 成功 -->
      <div v-else-if="submitted" class="p-8 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 text-center">
        <UIcon name="i-lucide-check-circle" class="size-12 mx-auto text-primary-500" />
        <p class="mt-3 text-xl font-medium text-stone-900 dark:text-stone-100">预约成功</p>
        <p class="text-sm text-stone-500 mt-1">我们会尽快与您确认，到店见</p>
        <UButton class="mt-5" block variant="soft" @click="reset">再约一次</UButton>
      </div>

      <!-- 表单 -->
      <div v-else class="space-y-4">
        <div class="p-5 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 space-y-3">
          <UFormField label="姓名" required>
            <UInput v-model="form.customer_name" placeholder="如：张女士" class="w-full" size="lg" />
          </UFormField>
          <UFormField label="手机号" required>
            <UInput v-model="form.customer_phone" placeholder="11 位手机号" class="w-full" size="lg" />
          </UFormField>
          <div class="grid grid-cols-2 gap-4">
            <UFormField label="到店时间" required>
              <UPopover :ui="{ content: 'p-2 w-auto' }">
                <UInputDate :model-value="apptDateTimeVal" @update:model-value="onApptDateTimeUpdate" granularity="minute" size="lg" class="w-full" icon="i-lucide-calendar-clock" />
                <template #content>
                  <UCalendar :model-value="apptCalDate" @update:model-value="onApptCalDateSelect" class="p-2" />
                </template>
              </UPopover>
            </UFormField>
            <UFormField label="指定员工（可选）">
              <USelect v-model="form.assigned_staff_id" :items="staffOptions" placeholder="不指定" class="w-full" size="lg" />
            </UFormField>
          </div>
        </div>

        <div class="p-5 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800">
          <SectionTitle class="mb-3">
            预约项目
            <template #suffix>
              <span class="text-xs text-stone-400 ml-auto">{{ form.service_ids.length > 0 ? `已选 ${form.service_ids.length} 项` : '请至少选一项' }}</span>
            </template>
          </SectionTitle>
          <UInput
            v-if="services.length > 6"
            v-model="svcSearch"
            icon="i-lucide-search"
            placeholder="搜索服务名称或金额"
            size="md"
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
          <div class="grid grid-cols-2 gap-2 max-h-60 overflow-y-auto p-1">
            <button
              v-for="s in filteredServices" :key="s.id"
              type="button"
              :class="[
                'p-2.5 rounded-xl text-left ring-1 transition active:scale-95',
                form.service_ids.includes(s.id)
                  ? 'bg-primary-50 dark:bg-primary-950/40 ring-primary-500/40 text-primary-700 dark:text-primary-300'
                  : 'ring-stone-200 dark:ring-stone-700 hover:ring-primary-500/30'
              ]"
              @click="toggleSvc(s.id)"
            >
              <div class="text-sm truncate">{{ s.name }}</div>
              <div class="text-xs text-stone-500 tabular-nums">¥{{ s.price }}</div>
            </button>
            <div v-if="filteredServices.length === 0" class="col-span-2 py-4 text-center text-sm text-stone-400">没有匹配的服务</div>
          </div>
        </div>

        <div class="p-5 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 space-y-3">
          <UFormField label="备注（可选）">
            <UTextarea v-model="form.notes" :rows="2" placeholder="任何想告诉商户的事" class="w-full" />
          </UFormField>
          <UAlert v-if="err" :description="err" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <UButton block size="lg" :loading="submitting" @click="submit">提交预约</UButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
useHead({ title: '在线预约' })
import { CalendarDate, CalendarDateTime } from '@internationalized/date'

definePageMeta({ layout: 'public' })

const route = useRoute()
const api = useApi()
const code = ref(route.params.code as string || '')

interface Svc { id: string; name: string; price: string }
interface Stf { id: string; name: string }
const services = ref<Svc[]>([])
const staff = ref<Stf[]>([])
const invalidCode = ref(false)
const submitted = ref(false)
const submitting = ref(false)
const err = ref('')

const form = reactive({
  customer_name: '', customer_phone: '', appointment_time: '',
  assigned_staff_id: '', service_ids: [] as string[], notes: '',
})

const staffOptions = computed(() =>
  staff.value.map(s => ({ label: s.name, value: s.id }))
)

function toggleSvc(id: string) {
  const i = form.service_ids.indexOf(id)
  if (i >= 0) form.service_ids.splice(i, 1)
  else form.service_ids.push(id)
}

// ---- 日历选择器 ----
const apptDateTimeVal = computed(() => {
  if (!form.appointment_time) return undefined
  const [datePart, timePart] = form.appointment_time.split('T')
  if (!datePart) return undefined
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

const svcSearch = ref('')
const filteredServices = computed(() => {
  const q = svcSearch.value.trim().toLowerCase()
  if (!q) return services.value
  return services.value.filter(s => s.name.toLowerCase().includes(q) || s.price.includes(q))
})

function setDefaultTime() {
  const now = new Date()
  const hh = String((now.getHours() + 1) % 24).padStart(2, '0')
  form.appointment_time = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}T${hh}:00`
}

async function fetchOptions() {
  if (!code.value) { invalidCode.value = true; return }
  try {
    const d = await api<{ services: Svc[]; staff: Stf[] }>(`/api/booking/options?code=${encodeURIComponent(code.value)}`)
    services.value = d.services
    staff.value = d.staff
    setDefaultTime()
  } catch (e: any) {
    invalidCode.value = true
  }
}

async function submit() {
  err.value = ''
  if (!form.customer_name || !form.customer_phone || !form.appointment_time || form.service_ids.length === 0) {
    err.value = '请填写完整信息'; return
  }
  if (!/^1[3-9]\d{9}$/.test(form.customer_phone)) { err.value = '手机号格式不正确'; return }

  submitting.value = true
  try {
    const body: any = {
      customer_name: form.customer_name,
      customer_phone: form.customer_phone,
      appointment_time: new Date(form.appointment_time).toISOString(),
      service_ids: form.service_ids,
    }
    if (form.assigned_staff_id) body.assigned_staff_id = form.assigned_staff_id
    if (form.notes) body.notes = form.notes
    await api(`/api/booking?code=${encodeURIComponent(code.value)}`, { method: 'POST', body })
    submitted.value = true
  } catch (e: any) {
    err.value = e?.data?.error || e?.data?.message || '预约失败'
  } finally { submitting.value = false }
}

function reset() {
  submitted.value = false
  form.customer_name = ''; form.customer_phone = ''; form.appointment_time = ''
  form.assigned_staff_id = ''; form.service_ids = []; form.notes = ''
  svcSearch.value = ''
  setDefaultTime()
}

onMounted(fetchOptions)
</script>
