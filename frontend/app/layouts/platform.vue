<template>
  <div class="min-h-screen bg-stone-50 dark:bg-stone-950 text-default">
    <header class="h-16 flex items-center gap-8 px-4 lg:px-8 bg-stone-100 dark:bg-stone-900 border-b border-stone-200/60 dark:border-stone-800">
      <div class="flex items-center gap-2.5 shrink-0">
        <div class="w-9 h-9 rounded-lg flex items-center justify-center bg-primary-500 text-white font-semibold">M</div>
        <div class="leading-tight">
          <div class="font-semibold">运营后台</div>
          <div class="text-xs text-stone-500">MMS 平台</div>
        </div>
      </div>
      <nav class="flex items-center gap-1 text-sm">
        <NuxtLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-2 px-3 py-2 rounded-lg
                 text-stone-700 dark:text-stone-300
                 hover:bg-stone-200/70 dark:hover:bg-stone-800/70
                 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40
                 transition-colors"
          :class="route.path === item.to
            ? navActiveClass
            : ''"
        >
          <UIcon :name="item.icon" class="size-4" />
          {{ item.label }}
        </NuxtLink>
      </nav>
      <div class="ml-auto flex items-center gap-3">
        <span class="text-sm text-stone-500 hidden sm:inline">{{ operatorName }}</span>
        <UTooltip text="修改密码">
          <UButton
            icon="i-lucide-key-round"
            variant="ghost"
            color="neutral"
            size="sm"
            class="active:scale-95 transition-transform"
            @click="openPassword"
          />
        </UTooltip>
        <UTooltip text="退出登录">
          <UButton
            icon="i-lucide-log-out"
            variant="ghost"
            color="neutral"
            size="sm"
            class="active:scale-95 transition-transform"
            @click="onLogout"
          />
        </UTooltip>
      </div>
    </header>
    <main class="p-6 lg:p-8 max-w-7xl mx-auto space-y-5">
      <slot />
    </main>

    <UModal v-model:open="pwdOpen" title="修改密码" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <UForm :state="pwdForm" class="space-y-4" @submit="onChangePassword">
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-3">
            <UFormField label="当前密码" required>
              <UInput v-model="pwdForm.current_password" type="password" autocomplete="current-password" size="md" class="w-full" />
            </UFormField>
            <UFormField label="新密码" required help="至少 8 位">
              <UInput v-model="pwdForm.new_password" type="password" autocomplete="new-password" size="md" class="w-full" />
            </UFormField>
            <UFormField label="确认新密码" required>
              <UInput v-model="pwdForm.confirm" type="password" autocomplete="new-password" size="md" class="w-full" />
            </UFormField>
          </div>
          <UAlert
            description="修改后所有已登录设备都会退出，需要用新密码重新登录。"
            color="warning"
            variant="soft"
            icon="i-lucide-info"
          />
          <UAlert v-if="pwdError" :description="pwdError" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <div class="flex justify-end gap-2 pt-2">
            <UButton type="button" variant="ghost" color="neutral" @click="pwdOpen = false">取消</UButton>
            <UButton type="submit" :loading="pwdSubmitting" class="active:scale-95 transition-transform">保存</UButton>
          </div>
        </UForm>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()

const navItems = [
  { to: '/platform', label: '商户', icon: 'i-lucide-store' },
  { to: '/platform/applications', label: '申请', icon: 'i-lucide-inbox' },
  { to: '/platform/editions', label: '套餐', icon: 'i-lucide-layers' },
]

const operatorName = computed(() => platformSession()?.operator?.name ?? '')

function onLogout() {
  platformLogout()
  navigateTo('/platform/login')
}

// ---- 改密 ----
const api = usePlatformApi()
const toast = useToast()
const pwdOpen = ref(false)
const pwdSubmitting = ref(false)
const pwdError = ref('')
const pwdForm = reactive({ current_password: '', new_password: '', confirm: '' })

function openPassword() {
  Object.assign(pwdForm, { current_password: '', new_password: '', confirm: '' })
  pwdError.value = ''
  pwdOpen.value = true
}

async function onChangePassword() {
  pwdError.value = ''
  if (pwdForm.new_password !== pwdForm.confirm) {
    pwdError.value = '两次输入的新密码不一致'
    return
  }
  pwdSubmitting.value = true
  try {
    await api('/api/platform/password', {
      method: 'POST',
      body: { current_password: pwdForm.current_password, new_password: pwdForm.new_password },
    })
    pwdOpen.value = false
    // 后端递增 token_version 吊销了本次会话，必须重新登录
    platformLogout()
    toast.add({ title: '密码已修改', description: '请用新密码重新登录', color: 'success', icon: 'i-lucide-check' })
    navigateTo('/platform/login')
  } catch (e: any) {
    pwdError.value = e?.data?.message || '修改失败'
  } finally {
    pwdSubmitting.value = false
  }
}
</script>
