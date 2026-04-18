<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <p class="text-sm text-stone-500">共 {{ items.length }} 个账号</p>
      <UButton icon="i-lucide-user-plus" @click="openCreate">新建账号</UButton>
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-x-auto">
      <table class="w-full min-w-[640px] text-base">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-3 font-medium">姓名</th>
            <th class="text-left px-4 py-3 font-medium">账号</th>
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
              <div class="inline-flex items-center gap-1.5">
                <UButton
                  size="xs" variant="soft" color="primary"
                  icon="i-lucide-user-cog"
                  class="active:scale-95 transition-transform"
                  @click="openEdit(u)"
                >设置</UButton>
                <UButton
                  v-if="u.id !== currentUserId"
                  size="xs" variant="soft" color="error"
                  icon="i-lucide-trash-2"
                  class="active:scale-95 transition-transform"
                  @click="confirmDelete(u)"
                >删除</UButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <EmptyState v-else-if="!loading" icon="i-lucide-shield" text="暂无账号" hint="点击右上角新建" />
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 3" :key="i" class="h-14 rounded-2xl" /></div>

    <!-- 统一的账号 Dialog：创建 / 编辑 / 重置密码（与"个人设置"共享同一个组件、同一套行为） -->
    <UserAccountDialog
      v-model:open="dialogOpen"
      :target="editing"
      @saved="fetchList"
    />

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
import { useAuthStore, roleLabel as roleLabelFn } from '~/stores/auth'

definePageMeta({ middleware: 'super-admin' })

interface User {
  id: string; email: string; phone: string | null; name: string
  role: 'super_admin' | 'admin' | 'staff'; status: 'active' | 'disabled'
  last_login_at: string | null; created_at: string
}

const api = useApi()
const auth = useAuthStore()
const currentUserId = computed(() => auth.user?.id ?? '')
const items = ref<User[]>([])
const loading = ref(true)

const dialogOpen = ref(false)
const editing = ref<User | null>(null)

const deleteOpen = ref(false)
const deleting = ref<User | null>(null)
const deletingLoading = ref(false)

function roleLabel(r: string) { return roleLabelFn(r) }
function roleColor(r: string): any { return ({ super_admin: 'primary', admin: 'warning', staff: 'neutral' } as any)[r] ?? 'neutral' }
function formatTime(s: string) { return new Date(s).toLocaleString('zh-CN', { hour12: false }) }

async function fetchList() {
  loading.value = true
  try {
    const data = await api<{ items: User[] }>('/api/users')
    items.value = data.items
  } finally { loading.value = false }
}

function openCreate() {
  editing.value = null
  dialogOpen.value = true
}
function openEdit(u: User) {
  editing.value = u
  dialogOpen.value = true
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
