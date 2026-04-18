<template>
  <UModal :open="open" :title="title" :ui="{ content: 'sm:max-w-lg' }" @update:open="(v) => emit('update:open', v)">
    <template #body>
      <UForm :state="form" class="space-y-4" @submit="onSubmit">
        <!-- 账号信息段 -->
        <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-3">
          <div class="grid grid-cols-2 gap-4">
            <UFormField label="姓名" required>
              <UInput v-model="form.name" placeholder="王小花" size="md" class="w-full" />
            </UFormField>
            <UFormField label="账号" required>
              <UInput v-model="form.email" :disabled="!isCreate" placeholder="admin@example.com" size="md" class="w-full" />
            </UFormField>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <UFormField label="手机">
              <UInput v-model="form.phone" placeholder="13800138000" size="md" class="w-full" />
            </UFormField>
            <UFormField v-if="!isCreate && !isSelf" label="状态">
              <URadioGroup
                v-model="form.status"
                :items="[{label:'启用', value:'active'},{label:'停用', value:'disabled'}]"
                orientation="horizontal"
              />
            </UFormField>
          </div>
          <UFormField v-if="!isSelf" label="角色" required>
            <URadioGroup v-model="form.role" :items="roleOptions" orientation="horizontal" />
            <div class="mt-1.5 flex items-start gap-1.5 text-xs text-stone-500 dark:text-stone-400">
              <UIcon name="i-lucide-info" class="size-3.5 mt-0.5 shrink-0" />
              <span>{{ roleHint }}</span>
            </div>
          </UFormField>
        </div>

        <!-- 密码段 -->
        <div class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900 space-y-3">
          <div class="flex items-center gap-2">
            <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
            <h3 class="text-base font-medium">{{ pwdSectionTitle }}</h3>
            <span v-if="!isCreate" class="text-xs text-stone-400 ml-auto">留空则不改</span>
          </div>
          <UFormField v-if="isSelf" label="当前密码">
            <UInput v-model="form.currentPwd" type="password" size="md" autocomplete="current-password" class="w-full" />
          </UFormField>
          <div class="grid grid-cols-2 gap-4">
            <UFormField :label="isCreate ? '初始密码(≥8 位)' : '新密码(≥8 位)'" :required="isCreate">
              <UInput v-model="form.newPwd" type="password" size="md" autocomplete="new-password" class="w-full" />
            </UFormField>
            <UFormField
              :label="isCreate ? '确认初始密码' : '确认新密码'"
              :required="isCreate || !!form.newPwd"
              :error="mismatch ? '两次输入不一致' : undefined"
            >
              <UInput v-model="form.confirmPwd" type="password" size="md" autocomplete="new-password" class="w-full" />
            </UFormField>
          </div>
        </div>

        <UAlert v-if="formMsg" :description="formMsg" :color="formErr ? 'error' : 'success'" variant="soft" icon="i-lucide-alert-circle" />
        <div class="flex justify-end gap-2 pt-2">
          <UButton type="button" variant="ghost" color="neutral" @click="emit('update:open', false)">取消</UButton>
          <UButton type="submit" :loading="saving" :disabled="!canSave">保存</UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useAuthStore, ROLE_OPTIONS, roleHint as authRoleHint, type Role } from '~/stores/auth'

interface UserLike {
  id: string
  email: string
  name: string
  phone?: string | null
  status?: 'active' | 'disabled'
  role?: Role
}

const roleOptions = ROLE_OPTIONS

const props = defineProps<{
  open: boolean
  target?: UserLike | null
}>()
const emit = defineEmits<{
  (e: 'update:open', v: boolean): void
  (e: 'saved'): void
}>()

const api = useApi()
const auth = useAuthStore()

const isCreate = computed(() => !props.target)
const isSelf = computed(() => !!props.target && props.target.id === auth.user?.id)

const title = computed(() => {
  if (isCreate.value) return '新建账号'
  if (isSelf.value)   return '个人设置'
  return '账号设置'
})
const pwdSectionTitle = computed(() => {
  if (isCreate.value) return '设置初始密码'
  if (isSelf.value)   return '修改登录密码'
  return '重置登录密码'
})

const form = reactive({
  name: '', email: '', phone: '',
  status: 'active' as 'active' | 'disabled',
  role: 'staff' as Role,
  currentPwd: '', newPwd: '', confirmPwd: '',
})
const saving = ref(false)
const formErr = ref(false)
const formMsg = ref('')

const mismatch = computed(() => !!form.confirmPwd && form.newPwd !== form.confirmPwd)
const roleHint = computed(() => authRoleHint(form.role))

const canSave = computed(() => {
  if (!form.name || !form.email) return false
  if (mismatch.value) return false
  if (isCreate.value) {
    if (form.newPwd.length < 8) return false
  } else if (form.newPwd) {
    if (form.newPwd.length < 8) return false
    if (isSelf.value && !form.currentPwd) return false
  }
  return true
})

function reset() {
  const t = props.target
  if (!t) {
    form.name = ''; form.email = ''; form.phone = ''
    form.status = 'active'; form.role = 'staff'
  } else {
    form.name = t.name; form.email = t.email
    form.phone = t.phone ?? ''
    form.status = t.status ?? 'active'
    form.role = (t.role ?? 'staff') as Role
  }
  form.currentPwd = ''; form.newPwd = ''; form.confirmPwd = ''
  formErr.value = false; formMsg.value = ''
}

async function onSubmit() {
  saving.value = true; formErr.value = false; formMsg.value = ''
  try {
    if (isCreate.value) {
      await api('/api/users', {
        method: 'POST',
        body: {
          name: form.name, email: form.email,
          phone: form.phone || null,
          role: form.role,
          password: form.newPwd,
        },
      })
      emit('saved')
      emit('update:open', false)
      return
    }

    const uid = props.target!.id
    const infoBody: any = { name: form.name, phone: form.phone || null }
    if (!isSelf.value) {
      infoBody.status = form.status
      infoBody.role = form.role
    }
    await api(`/api/users/${uid}`, { method: 'PUT', body: infoBody })

    if (form.newPwd) {
      if (isSelf.value) {
        await api('/api/auth/change-password', {
          method: 'POST',
          body: { old_password: form.currentPwd, new_password: form.newPwd },
        })
      } else {
        await api(`/api/users/${uid}/reset-password`, {
          method: 'POST',
          body: { new_password: form.newPwd },
        })
      }
    }

    emit('saved')
    emit('update:open', false)
  } catch (e: any) {
    formErr.value = true
    formMsg.value = e?.data?.message || e?.message || '保存失败'
  } finally { saving.value = false }
}

watch(() => props.open, (v) => { if (v) reset() })
</script>
