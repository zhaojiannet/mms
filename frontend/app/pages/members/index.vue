<template>
  <div class="p-6 lg:p-8 max-w-7xl mx-auto space-y-6">
    <!-- Head -->
    <header class="flex items-center justify-between gap-4 flex-wrap">
      <div>
        <h1 class="text-3xl font-semibold tracking-tight">会员</h1>
        <p class="mt-1 text-sm text-stone-500">共 {{ total }} 位会员</p>
      </div>
      <div class="flex items-center gap-2">
        <UInput
          v-model="search"
          icon="i-lucide-search"
          placeholder="按姓名或手机搜索"
          size="md"
          class="w-56"
        />
        <UButton icon="i-lucide-user-plus" @click="openCreate">
          新建会员
        </UButton>
      </div>
    </header>

    <!-- Filters -->
    <div class="flex items-center gap-2 flex-wrap">
      <UBadge
        v-for="f in filters"
        :key="f.key"
        :color="filter === f.key ? 'primary' : 'neutral'"
        :variant="filter === f.key ? 'solid' : 'soft'"
        class="cursor-pointer active:scale-95 transition-transform"
        @click="filter = f.key"
      >
        {{ f.label }}
      </UBadge>
    </div>

    <!-- Desktop table -->
    <div
      v-if="!loading && filtered.length > 0"
      class="hidden md:block rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-hidden"
    >
      <table class="w-full text-base">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-3 font-medium">会员</th>
            <th class="text-left px-4 py-3 font-medium">手机</th>
            <th class="text-right px-4 py-3 font-medium">卡余额</th>
            <th class="text-right px-4 py-3 font-medium">挂账</th>
            <th class="text-left px-4 py-3 font-medium">状态</th>
            <th class="text-right px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
          <tr
            v-for="m in filtered"
            :key="m.id"
            class="hover:bg-stone-50/60 dark:hover:bg-stone-800/40 transition-colors"
          >
            <td class="px-4 py-3">
              <NuxtLink :to="`/members/${m.id}`" class="flex items-center gap-2.5 hover:text-primary-600 transition-colors">
                <UAvatar :alt="m.name" size="sm" />
                <span class="font-medium">{{ m.name }}</span>
              </NuxtLink>
            </td>
            <td class="px-4 py-3 text-stone-600 dark:text-stone-400 tabular-nums">
              {{ m.phone || '—' }}
            </td>
            <td class="px-4 py-3 text-right tabular-nums">
              <span v-if="parseFloat(m.total_balance) > 0" class="font-medium">¥{{ m.total_balance }}</span>
              <span v-else class="text-stone-400">—</span>
              <span v-if="m.card_count > 0" class="text-xs text-stone-400 ml-1">({{ m.card_count }})</span>
            </td>
            <td class="px-4 py-3 text-right tabular-nums">
              <span v-if="parseFloat(m.total_pending) > 0" class="text-warning-600 font-medium">¥{{ m.total_pending }}</span>
              <span v-else class="text-stone-400">—</span>
            </td>
            <td class="px-4 py-3">
              <UBadge
                :label="m.status === 'active' ? '正常' : '停用'"
                :color="m.status === 'active' ? 'success' : 'neutral'"
                variant="soft"
                size="sm"
              />
            </td>
            <td class="px-4 py-3 text-right">
              <div class="flex justify-end gap-1">
                <UButton size="xs" variant="ghost" color="neutral" @click="openEdit(m)">编辑</UButton>
                <UButton size="xs" variant="ghost" color="neutral" @click="toggleStatus(m)">
                  {{ m.status === 'active' ? '停用' : '启用' }}
                </UButton>
                <UButton size="xs" variant="ghost" color="error" @click="confirmDelete(m)">删除</UButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Mobile cards -->
    <div v-if="!loading && filtered.length > 0" class="md:hidden space-y-2.5">
      <div
        v-for="m in filtered"
        :key="m.id"
        class="p-4 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs"
      >
        <div class="flex items-center gap-3">
          <UAvatar :alt="m.name" />
          <div class="flex-1 min-w-0">
            <div class="font-medium">{{ m.name }}</div>
            <div class="text-sm text-stone-500">{{ m.phone || '未留手机' }}</div>
          </div>
          <UBadge
            :label="m.status === 'active' ? '正常' : '停用'"
            :color="m.status === 'active' ? 'success' : 'neutral'"
            variant="soft"
            size="sm"
          />
        </div>
        <div class="mt-3 flex items-center justify-end gap-1 pt-3 border-t border-stone-200/60 dark:border-stone-800">
          <UButton size="xs" variant="ghost" color="neutral" @click="openEdit(m)">编辑</UButton>
          <UButton size="xs" variant="ghost" color="neutral" @click="toggleStatus(m)">
            {{ m.status === 'active' ? '停用' : '启用' }}
          </UButton>
          <UButton size="xs" variant="ghost" color="error" @click="confirmDelete(m)">删除</UButton>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-if="!loading && filtered.length === 0"
      class="py-20 text-center rounded-2xl border border-dashed border-stone-300/70 dark:border-stone-700 bg-stone-100/40 dark:bg-stone-900/40"
    >
      <UIcon name="i-lucide-users-round" class="size-10 text-stone-400 mx-auto" />
      <p class="mt-3 text-base text-stone-500">
        {{ search ? '没有匹配的会员' : '还没有会员，点击右上角"新建会员"开始' }}
      </p>
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="space-y-2">
      <USkeleton v-for="i in 5" :key="i" class="h-14 w-full rounded-2xl" />
    </div>

    <!-- Form drawer -->
    <USlideover v-model:open="formOpen" :title="editingId ? '编辑会员' : '新建会员'" :ui="{ content: 'w-full sm:max-w-md' }">
      <template #body>
        <UForm :state="form" class="space-y-4" @submit="onSubmit">
          <UFormField label="姓名" name="name" required>
            <UInput v-model="form.name" placeholder="王小花" size="md" class="w-full" />
          </UFormField>
          <UFormField label="手机" name="phone">
            <UInput v-model="form.phone" placeholder="13800138000" size="md" class="w-full" />
          </UFormField>
          <UFormField label="性别" name="gender">
            <URadioGroup v-model="form.gender" :items="genderOptions" orientation="horizontal" />
          </UFormField>
          <UFormField label="生日" name="birthday">
            <UInput v-model="form.birthday" type="date" size="md" class="w-full" />
          </UFormField>
          <UFormField label="备注" name="notes">
            <UTextarea v-model="form.notes" placeholder="过敏 / 偏好 / 其他" :rows="3" class="w-full" />
          </UFormField>
          <UAlert v-if="formError" :description="formError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton variant="ghost" color="neutral" @click="formOpen = false">取消</UButton>
            <UButton type="submit" :loading="submitting">保存</UButton>
          </div>
        </UForm>
      </template>
    </USlideover>

    <!-- Delete confirm -->
    <UModal v-model:open="deleteOpen" title="删除会员" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <p class="text-base leading-relaxed">
          确认删除「<strong class="font-medium">{{ deleting?.name }}</strong>」？
          <br /><span class="text-stone-500 text-sm">此操作不可撤销，该会员的历史消费记录将保留。</span>
        </p>
      </template>
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
interface Member {
  id: string
  name: string
  phone: string | null
  gender: 'male' | 'female' | 'unknown'
  birthday: string | null
  notes: string | null
  status: 'active' | 'inactive'
  total_balance: string
  total_pending: string
  card_count: number
  created_at: string
  updated_at: string
}
interface ListResponse { items: Member[]; total: number }

const api = useApi()
const route = useRoute()
const router = useRouter()

const items = ref<Member[]>([])
const total = ref(0)
const loading = ref(true)

const search = ref('')
const filter = ref<'all' | 'active' | 'inactive'>('all')
const filters = [
  { key: 'all',      label: '全部' },
  { key: 'active',   label: '正常' },
  { key: 'inactive', label: '停用' },
]

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return items.value.filter(m => {
    if (filter.value === 'active' && m.status !== 'active') return false
    if (filter.value === 'inactive' && m.status !== 'inactive') return false
    if (!q) return true
    return m.name.toLowerCase().includes(q) || (m.phone || '').includes(q)
  })
})

const formOpen = ref(false)
const submitting = ref(false)
const formError = ref('')
const editingId = ref<string | null>(null)
const form = reactive({
  name: '',
  phone: '',
  gender: 'unknown' as 'male' | 'female' | 'unknown',
  birthday: '',
  notes: '',
})
const genderOptions = [
  { label: '未知', value: 'unknown' },
  { label: '男', value: 'male' },
  { label: '女', value: 'female' },
]

const deleteOpen = ref(false)
const deleting = ref<Member | null>(null)
const deletingLoading = ref(false)

function genderLabel(g: string) {
  return { male: '男', female: '女', unknown: '—' }[g] ?? g
}
function formatDate(s: string) {
  if (!s) return ''
  return new Date(s).toLocaleString('zh-CN', { hour12: false })
}

async function fetchList() {
  loading.value = true
  try {
    const data = await api<ListResponse>('/api/members')
    items.value = data.items
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.name = ''
  form.phone = ''
  form.gender = 'unknown'
  form.birthday = ''
  form.notes = ''
  formError.value = ''
}

function openCreate() {
  editingId.value = null
  resetForm()
  formOpen.value = true
}

function openEdit(m: Member) {
  editingId.value = m.id
  form.name = m.name
  form.phone = m.phone ?? ''
  form.gender = m.gender
  form.birthday = m.birthday ?? ''
  form.notes = m.notes ?? ''
  formError.value = ''
  formOpen.value = true
}

async function onSubmit() {
  submitting.value = true
  formError.value = ''
  try {
    const body = {
      name: form.name,
      phone: form.phone || null,
      gender: form.gender,
      birthday: form.birthday || null,
      notes: form.notes || null,
    }
    if (editingId.value) {
      await api(`/api/members/${editingId.value}`, { method: 'PUT', body })
    } else {
      await api('/api/members', { method: 'POST', body })
    }
    formOpen.value = false
    await fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || e?.message || '保存失败'
  } finally {
    submitting.value = false
  }
}

async function toggleStatus(m: Member) {
  const next = m.status === 'active' ? 'inactive' : 'active'
  try {
    await api(`/api/members/${m.id}`, { method: 'PUT', body: { status: next } })
    await fetchList()
  } catch {}
}

function confirmDelete(m: Member) {
  deleting.value = m
  deleteOpen.value = true
}

async function onDelete() {
  if (!deleting.value) return
  deletingLoading.value = true
  try {
    await api(`/api/members/${deleting.value.id}`, { method: 'DELETE' })
    deleteOpen.value = false
    deleting.value = null
    await fetchList()
  } finally {
    deletingLoading.value = false
  }
}

onMounted(async () => {
  await fetchList()
  if (route.query.new === '1') {
    openCreate()
    router.replace({ query: {} })
  }
})
</script>
