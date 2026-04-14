<template>
  <div class="min-h-screen grid lg:grid-cols-[2fr_3fr] bg-stone-50 dark:bg-stone-950">
    <!-- Left · Brand + 背景图 -->
    <div
      class="hidden lg:flex relative flex-col justify-between p-12 text-white overflow-hidden bg-cover bg-center"
      style="background-image: url('/images/login-bg.jpg')"
    >
      <div class="absolute inset-0 bg-stone-900/45" />
      <div class="absolute inset-0 bg-gradient-to-b from-stone-900/10 via-transparent to-stone-900/40" />

      <div class="relative flex items-center gap-3">
        <div
          class="w-10 h-10 rounded-xl flex items-center justify-center
                 bg-white/15 ring-1 ring-white/20 backdrop-blur-sm
                 font-semibold text-lg"
        >D</div>
        <div class="leading-tight">
          <div class="font-semibold text-base">Demo Store</div>
          <div class="text-xs text-white/70 tabular-nums">
            {{ cfg.tenantSlug }}.{{ cfg.appDomain }}
          </div>
        </div>
      </div>

      <div class="relative space-y-6 max-w-md">
        <h1 class="text-6xl leading-tight font-semibold tracking-tight">
          简单高效<br />管理不用愁
        </h1>
        <p class="text-base text-white/80 leading-relaxed">
          面向美业 · 美容 · 美发 · 美甲 · 按摩 · 瑜伽 · 培训 · 宠物等小微商户的会员管理 SaaS
        </p>
      </div>

      <div class="relative text-xs text-white/60">
        © {{ new Date().getFullYear() }} 赵健 · AGPL-3.0 开源
      </div>
    </div>

    <!-- Right · Form -->
    <div class="flex items-center justify-center p-6 sm:p-12">
      <div class="w-full max-w-sm space-y-8">
        <div class="lg:hidden flex items-center gap-2.5">
          <div
            class="w-9 h-9 rounded-xl flex items-center justify-center
                   bg-primary-500 text-white font-semibold shadow-xs"
          >D</div>
          <span class="font-semibold text-base">Demo Store</span>
        </div>

        <div class="space-y-1.5">
          <h2 class="text-2xl font-semibold tracking-tight">欢迎回来</h2>
          <p class="text-sm text-stone-500 dark:text-stone-400">
            请使用账户登录
          </p>
        </div>

        <UForm :state="form" @submit="onSubmit" class="space-y-4">
          <UFormField label="邮箱" name="email" required>
            <UInput
              v-model="form.email"
              type="email"
              placeholder="admin@example.com"
              icon="i-lucide-mail"
              size="lg"
              autocomplete="email"
              class="w-full"
            />
          </UFormField>

          <UFormField label="密码" name="password" required>
            <UInput
              v-model="form.password"
              type="password"
              placeholder="••••••••"
              icon="i-lucide-lock"
              size="lg"
              autocomplete="current-password"
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

          <UButton
            type="submit"
            :loading="loading"
            block
            size="lg"
            class="!h-11"
          >
            登录
          </UButton>
        </UForm>

        <p class="text-sm text-stone-500 dark:text-stone-400 text-center">
          忘记密码？联系店长或系统管理员
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'public' })

const cfg = useRuntimeConfig().public
const auth = useAuthStore()
const router = useRouter()

const form = reactive({ email: '', password: '' })
const loading = ref(false)
const errorMsg = ref('')

async function onSubmit() {
  errorMsg.value = ''
  loading.value = true
  try {
    await auth.login(form.email, form.password)
    await router.push('/')
  } catch (e: any) {
    errorMsg.value = e?.data?.message || e?.message || '登录失败，请检查账号密码'
  } finally {
    loading.value = false
  }
}
</script>
