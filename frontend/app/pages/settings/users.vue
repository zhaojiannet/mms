<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <p class="text-sm text-stone-500">共 {{ items.length }} 个账号</p>
      <UButton icon="i-lucide-user-plus" @click="openCreate">新增账号</UButton>
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-hidden">
      <table class="w-full text-base">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-3 font-medium">姓名</th>
            <th class="text-left px-4 py-3 font-medium">账号 (email)</th>
            <th class="text-center px-4 py-3 font-medium">角色</th>
            <th class="text-center px-4 py-3 font-medium">状态</th>
            <th class="text-left px-4 py-3 font-medium">最后登录</th>
            <th class="text-right px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
          <tr v-for="u in items" :key="u.id" class="hover:bg-stone-50/60 dark:hover:bg-stone-800/40">
            <td class="px-4 py-3 font-medium">{{ u.name }}</td>
            <td class="px-4 py-3 text-stone-600 tabular-nums">{{ u.email }}</td>
            <td class="px-4 py-3 text-center">
              <UBadge :label="roleLabel(u.role)" :color="roleColor(u.role)" variant="soft" size="sm" />
            </td>
            <td class="px-4 py-3 text-center">
              <UBadge :label="u.status === 'active' ? '启用' : '停用'" :color="u.status === 'active' ? 'success' : 'neutral'" variant="soft" size="sm" />
            </td>
            <td class="px-4 py-3 text-stone-500 text-sm tabular-nums">
              {{ u.last_login_at ? formatTime(u.last_login_at) : '—' }}
            </td>
            <td class="px-4 py-3 text-right">
              <UButton size="xs" variant="ghost" color="neutral" @click="openEdit(u)">编辑</UButton>
              <UButton size="xs" variant="ghost" color="neutral" @click="openResetPwd(u)">重置密码</UButton>
              <UButton size="xs" variant="ghost" color="error" @click="confirmDelete(u)">删除</UButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else-if="!loading" class="py-16 text-center text-stone-500">暂无账号</div>
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 3" :key="i" class="h-14 rounded-2xl" /></div>

    <!-- Create/Edit -->
    <USlideover v-model:open="formOpen" :title="editingId ? '编辑账号' : '新增账号'" :ui="{ content: 'w-full sm:max-w-md' }">
      <template #body>
        <UForm :state="form" class="space-y-4" @submit="onSubmit">
          <UFormField label="姓名" required><UInput v-model="form.name" class="w-full" /></UFormField>
          <UFormField label="账号 (email 或自定义字符串)" required>
            <UInput v-model="form.email" :disabled="!!editingId" placeholder="admin@example.com 或 demo@example.com" class="w-full" />
          </UFormField>
          <UFormField label="手机">
            <UInput v-model="form.phone" class="w-full" />
          </UFormField>
          <UFormField label="角色" required>
            <URadioGroup v-model="form.role" :items="roleOptions" orientation="horizontal" />
          </UFormField>
          <UFormField v-if="!editingId" label="初始密码（≥6 位）" required>
            <UInput v-model="form.password" type="password" class="w-full" />
          </UFormField>
          <UFormField v-if="editingId" label="状态">
            <URadioGroup v-model="form.status" :items="[{label:'启用', value:'active'},{label:'停用', value:'disabled'}]" orientation="horizontal" />
          </UFormField>
          <UAlert v-if="formError" :description="formError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton variant="ghost" color="neutral" @click="formOpen = false">取消</UButton>
            <UButton type="submit" :loading="submitting">保存</UButton>
          </div>
        </UForm>
      </template>
    </USlideover>

    <!-- Reset Password -->
    <UModal v-model:open="resetOpen" title="重置密码" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <div class="space-y-3">
          <p class="text-sm">为「<strong>{{ resetting?.name }}</strong>」设置新密码。旧密码将失效。</p>
          <UFormField label="新密码（≥6 位）" required>
            <UInput v-model="newPwd" type="password" class="w-full" />
          </UFormField>
          <UAlert v-if="resetErr" :description="resetErr" color="error" variant="soft" icon="i-lucide-alert-circle" />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton variant="ghost" color="neutral" @click="resetOpen = false">取消</UButton>
          <UButton :loading="resetLoading" @click="doResetPwd">确认</UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="deleteOpen" title="删除账号" :ui="{ content: 'sm:max-w-md' }">
      <template #body><p>确认删除「<strong>{{ deleting?.name }}</strong>」？此账号所有登录会话立即失效。</p></template>
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
interface User {
  id: string; email: string; phone: string | null; name: string
  role: 'admin' | 'manager' | 'staff'; status: 'active' | 'disabled'
  last_login_at: string | null; created_at: string
}

const api = useApi()
const items = ref<User[]>([])
const loading = ref(true)

const formOpen = ref(false)
const submitting = ref(false)
const formError = ref('')
const editingId = ref<string | null>(null)
const form = reactive({
  name: '', email: '', phone: '', role: 'staff' as any,
  password: '', status: 'active' as 'active' | 'disabled',
})

const resetOpen = ref(false)
const resetting = ref<User | null>(null)
const newPwd = ref('')
const resetLoading = ref(false)
const resetErr = ref('')

const deleteOpen = ref(false)
const deleting = ref<User | null>(null)
const deletingLoading = ref(false)

const roleOptions = [
  { label: '管理员', value: 'admin' },
  { label: '经理',   value: 'manager' },
  { label: '员工',   value: 'staff' },
]
function roleLabel(r: string) { return ({ admin: '管理员', manager: '经理', staff: '员工' } as any)[r] ?? r }
function roleColor(r: string): any { return ({ admin: 'error', manager: 'warning', staff: 'neutral' } as any)[r] ?? 'neutral' }
function formatTime(s: string) { return new Date(s).toLocaleString('zh-CN', { hour12: false }) }

async function fetchList() {
  loading.value = true
  try {
    const data = await api<{ items: User[] }>('/api/users')
    items.value = data.items
  } finally { loading.value = false }
}

function resetForm() {
  form.name = ''; form.email = ''; form.phone = ''
  form.role = 'staff'; form.password = ''; form.status = 'active'
  formError.value = ''
}
function openCreate() { editingId.value = null; resetForm(); formOpen.value = true }
function openEdit(u: User) {
  editingId.value = u.id
  form.name = u.name; form.email = u.email
  form.phone = u.phone ?? ''; form.role = u.role
  form.password = ''; form.status = u.status
  formError.value = ''; formOpen.value = true
}

async function onSubmit() {
  submitting.value = true; formError.value = ''
  try {
    if (editingId.value) {
      const body: any = {
        name: form.name, phone: form.phone || null,
        role: form.role, status: form.status,
      }
      await api(`/api/users/${editingId.value}`, { method: 'PUT', body })
    } else {
      const body: any = {
        name: form.name, email: form.email, phone: form.phone || null,
        role: form.role, password: form.password,
      }
      await api('/api/users', { method: 'POST', body })
    }
    formOpen.value = false
    await fetchList()
  } catch (e: any) {
    formError.value = e?.data?.message || '保存失败'
  } finally { submitting.value = false }
}

function openResetPwd(u: User) { resetting.value = u; newPwd.value = ''; resetErr.value = ''; resetOpen.value = true }
async function doResetPwd() {
  if (!resetting.value) return
  if (newPwd.value.length < 6) { resetErr.value = '密码至少 6 位'; return }
  resetLoading.value = true
  try {
    await api(`/api/users/${resetting.value.id}/reset-password`, { method: 'POST', body: { new_password: newPwd.value } })
    resetOpen.value = false
  } catch (e: any) {
    resetErr.value = e?.data?.message || '重置失败'
  } finally { resetLoading.value = false }
}

function confirmDelete(u: User) { deleting.value = u; deleteOpen.value = true }
async function onDelete() {
  if (!deleting.value) return
  deletingLoading.value = true
  try {
    await api(`/api/users/${deleting.value.id}`, { method: 'DELETE' })
    deleteOpen.value = false; deleting.value = null
    await fetchList()
  } catch (e: any) {
    alert(e?.data?.message || '删除失败')
  } finally { deletingLoading.value = false }
}

onMounted(fetchList)
</script>
