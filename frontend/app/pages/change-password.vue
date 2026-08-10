<template>
  <div class="min-h-screen flex items-center justify-center p-6 sm:p-12 bg-stone-50 dark:bg-stone-950">
    <div class="w-full max-w-sm space-y-8">
      <div class="space-y-1.5 text-center">
        <h2 class="text-2xl font-semibold tracking-tight">请先设置新密码</h2>
        <p class="text-sm text-stone-500 dark:text-stone-400">
          当前密码由他人代设，改掉之后才能使用系统
        </p>
      </div>

      <UForm :state="form" @submit="onSubmit" class="space-y-4">
        <UFormField label="当前密码" name="currentPwd" required>
          <UInput
            v-model="form.currentPwd"
            type="password"
            placeholder="••••••••"
            icon="i-lucide-lock"
            size="lg"
            autocomplete="current-password"
            class="w-full"
          />
        </UFormField>

        <UFormField label="新密码" name="newPwd" required hint="至少 8 位">
          <UInput
            v-model="form.newPwd"
            type="password"
            placeholder="••••••••"
            icon="i-lucide-key-round"
            size="lg"
            autocomplete="new-password"
            class="w-full"
          />
        </UFormField>

        <UFormField label="确认新密码" name="confirmPwd" required>
          <UInput
            v-model="form.confirmPwd"
            type="password"
            placeholder="••••••••"
            icon="i-lucide-key-round"
            size="lg"
            autocomplete="new-password"
            class="w-full"
          />
        </UFormField>

        <UAlert
          v-if="errorMsg"
          :description="errorMsg"
          color="error"
          variant="soft"
          icon="i-lucide-alert-circle"
        />

        <UButton type="submit" :loading="loading" block size="lg" class="h-11!">
          设置新密码
        </UButton>
      </UForm>

      <p class="text-sm text-stone-500 dark:text-stone-400 text-center">
        改密后需要用新密码重新登录
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'public' })

const cfg = useRuntimeConfig().public
const auth = useAuthStore()

const form = reactive({ currentPwd: '', newPwd: '', confirmPwd: '' })
const loading = ref(false)
const errorMsg = ref('')

async function onSubmit() {
  errorMsg.value = ''
  if (form.newPwd.length < 8) {
    errorMsg.value = '新密码至少 8 位'
    return
  }
  if (form.newPwd !== form.confirmPwd) {
    errorMsg.value = '两次输入的新密码不一致'
    return
  }
  if (form.newPwd === form.currentPwd) {
    errorMsg.value = '新密码不能与当前密码相同'
    return
  }

  loading.value = true
  try {
    // 用裸 $fetch 而非 useApi：useApi 的 401 处理会把人踢回登录页，
    // 而改密本就会让当前 token 失效，交给下面的 logout 收尾更干净
    await $fetch(`${cfg.apiBase}/api/auth/change-password`, {
      method: 'POST',
      headers: { 'X-Tenant-Slug': cfg.tenantSlug, Authorization: `Bearer ${auth.token}` },
      body: { old_password: form.currentPwd, new_password: form.newPwd },
    })
    // 后端改密会递增 token_version，手上这个 token 已经作废，只能重新登录
    auth.logout()
    await navigateTo('/login')
  } catch (e: any) {
    errorMsg.value = e?.data?.message || e?.message || '设置失败，请重试'
  } finally {
    loading.value = false
  }
}
</script>
