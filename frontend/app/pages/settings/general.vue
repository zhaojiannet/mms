<template>
  <div class="space-y-4">
    <div class="p-6 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
      <h2 class="text-base font-medium mb-4">登录</h2>
      <div class="flex items-center justify-between gap-4">
        <div>
          <div class="text-base">登录验证码</div>
          <div class="text-sm text-stone-500">开启后登录页会显示图形验证码</div>
        </div>
        <USwitch v-model="captcha" :disabled="saving" @update:model-value="toggleCaptcha" />
      </div>
    </div>
    <UAlert v-if="msg" :description="msg" color="success" variant="soft" icon="i-lucide-check" />
  </div>
</template>

<script setup lang="ts">
const api = useApi()
const captcha = ref(false)
const saving = ref(false)
const msg = ref('')

async function fetchSettings() {
  try {
    const row = await api<{ key: string; value: any }>('/api/tenant-settings/enable_login_captcha')
    captcha.value = row.value === true
  } catch {}
}

async function toggleCaptcha(v: boolean) {
  saving.value = true
  try {
    await api('/api/tenant-settings/enable_login_captcha', { method: 'PUT', body: { value: v } })
    msg.value = v ? '已开启登录验证码' : '已关闭登录验证码'
    setTimeout(() => msg.value = '', 2000)
  } catch (e: any) {
    captcha.value = !v
    msg.value = e?.data?.message || '保存失败'
  } finally { saving.value = false }
}

onMounted(fetchSettings)
</script>
