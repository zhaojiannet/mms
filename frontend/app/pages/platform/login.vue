<template>
  <div class="min-h-screen flex items-center justify-center bg-stone-50 dark:bg-stone-950 p-4">
    <div class="w-full max-w-sm rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs p-6 space-y-5">
      <div class="text-center space-y-1">
        <div class="mx-auto w-10 h-10 rounded-lg flex items-center justify-center bg-primary-500 text-white font-semibold text-lg">M</div>
        <h1 class="font-semibold text-lg">运营后台</h1>
        <p class="text-xs text-stone-500">平台操作员登录</p>
      </div>

      <UForm :state="form" class="space-y-4" @submit="onSubmit">
        <UFormField label="邮箱" required>
          <UInput v-model="form.email" type="email" autocomplete="username" size="md" class="w-full" />
        </UFormField>
        <UFormField label="密码" required>
          <UInput v-model="form.password" type="password" autocomplete="current-password" size="md" class="w-full" />
        </UFormField>

        <UFormField v-if="captcha.image" label="图形验证码" required>
          <div class="flex items-center gap-2">
            <UInput v-model="form.captcha_answer" size="md" class="flex-1" />
            <img
              :src="captcha.image"
              alt="验证码"
              class="h-9 rounded cursor-pointer ring-1 ring-stone-200 dark:ring-stone-700"
              title="点击刷新"
              @click="loadCaptcha"
            />
          </div>
        </UFormField>

        <UAlert v-if="error" :description="error" color="error" variant="soft" icon="i-lucide-alert-circle" />
        <UButton type="submit" block :loading="submitting">登录</UButton>
      </UForm>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: false })
useHead({ title: '运营后台登录' })

const cfg = useRuntimeConfig().public
const form = reactive({ email: '', password: '', captcha_answer: '' })
const captcha = reactive({ id: '', image: '' })
const error = ref('')
const submitting = ref(false)

async function loadCaptcha() {
  try {
    const r = await $fetch<{ id: string; image: string }>(`${cfg.apiBase}/api/public/captcha`)
    captcha.id = r.id
    captcha.image = r.image
    form.captcha_answer = ''
  } catch {
    // 验证码加载失败不阻塞登录尝试
  }
}

async function onSubmit() {
  error.value = ''
  submitting.value = true
  try {
    const r = await $fetch<{ access_token: string; expires_at: string; operator: { email: string; name: string } }>(
      `${cfg.apiBase}/api/platform/login`,
      {
        method: 'POST',
        body: {
          email: form.email,
          password: form.password,
          captcha_id: captcha.id || undefined,
          captcha_answer: form.captcha_answer || undefined,
        },
      },
    )
    platformLoginSave({ token: r.access_token, expires_at: r.expires_at, operator: r.operator })
    navigateTo('/platform')
  } catch (e: any) {
    error.value = e?.data?.message || '登录失败'
    // 任何失败后展示验证码：后端失败 2 次即强制要求
    await loadCaptcha()
  } finally {
    submitting.value = false
  }
}
</script>
