<template>
  <div class="min-h-screen grid lg:grid-cols-[2fr_3fr] bg-stone-50 dark:bg-stone-950">
    <!-- Left · Brand + 背景图 -->
    <div
      class="hidden lg:flex relative flex-col justify-between p-12 text-white overflow-hidden transition-[background] duration-500 ease-out motion-reduce:transition-none"
      :style="{ background: leftBg }"
    >
      <div class="absolute inset-0 bg-stone-900/45" />
      <div class="absolute inset-0 bg-linear-to-b from-stone-900/10 via-transparent to-stone-900/40" />

      <div class="relative leading-tight">
        <div class="font-semibold text-base">{{ storeInfo.name || 'Demo Store' }}</div>
        <div class="text-xs text-white/70 tabular-nums">
          {{ storeInfo.slug || cfg.tenantSlug }}.{{ cfg.appDomain }}
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
      <div class="w-full max-w-sm space-y-8 -translate-y-10 lg:translate-y-0">
        <div v-if="!storeInfo.logo_url" class="lg:hidden flex items-center gap-2.5">
          <div
            class="w-9 h-9 rounded-xl overflow-hidden flex items-center justify-center
                   bg-primary-500 text-white font-semibold shadow-xs"
          >
            <span>{{ (storeInfo.name || 'S').slice(0, 1) }}</span>
          </div>
          <span class="font-semibold text-base">{{ storeInfo.name || 'Demo Store' }}</span>
        </div>

        <div class="space-y-4">
          <div v-if="storeInfo.logo_url" class="flex justify-center pt-2">
            <img
              :key="storeInfo.logo_url"
              :src="cfg.apiBase + safeAssetUrl(storeInfo.logo_url)"
              alt="Logo"
              class="h-28 sm:h-32 md:h-40 w-auto max-w-full object-contain"
            />
          </div>
          <div class="space-y-1.5 text-center">
            <h2 class="text-2xl font-semibold tracking-tight">欢迎回来</h2>
            <p class="text-sm text-stone-500 dark:text-stone-400">
              {{ storeInfo.name ? `登录 ${storeInfo.name}` : '请使用账户登录' }}
            </p>
          </div>
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

          <UFormField v-if="captchaNeeded" label="验证码" name="captcha" required>
            <div class="flex items-stretch gap-2">
              <UInput
                v-model="form.captchaAnswer"
                placeholder="请输入图中数字"
                icon="i-lucide-shield-check"
                size="lg"
                autocomplete="off"
                class="flex-1"
              />
              <button
                type="button"
                class="shrink-0 rounded-md ring-1 ring-stone-200 dark:ring-stone-700 overflow-hidden hover:ring-primary-400 transition-[--tw-ring-color] cursor-pointer"
                title="点击刷新"
                @click="loadCaptcha"
              >
                <img v-if="captchaImage" :src="captchaImage" alt="验证码" class="h-11 w-auto block" />
                <div v-else class="h-11 w-28 flex items-center justify-center text-xs text-stone-400">加载中…</div>
              </button>
            </div>
          </UFormField>

          <label class="flex items-center gap-2 cursor-pointer select-none">
            <UCheckbox v-model="form.remember" />
            <span class="text-sm text-stone-600 dark:text-stone-300">信任此设备（7 天内免登录）</span>
          </label>

          <UAlert
            v-if="errorMsg"
            :description="errorMsg"
            color="error"
            variant="soft"
            icon="i-lucide-alert-circle"
          />

          <p v-if="captchaNeeded" class="text-xs text-stone-500 dark:text-stone-400 text-center">
            多次登录失败，请输入验证码继续
          </p>

          <UButton
            type="submit"
            :loading="loading"
            block
            size="lg"
            class="h-11!"
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
import { getThemeByKey, bgStyle } from '~/composables/useLoginBgThemes'

definePageMeta({ layout: 'public' })

const cfg = useRuntimeConfig().public
const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const { info: storeInfo, refresh } = useStoreInfo()

const form = reactive({ email: '', password: '', remember: false, captchaAnswer: '' })
const loading = ref(false)
const errorMsg = ref('')

// 验证码：首次失败触发；保证一次一换（captcha 后端一次性校验后销毁）
const CAPTCHA_TRIGGER_AFTER = 2 // 连续失败 N 次后要求验证码
const failCount = ref(0)
const captchaNeeded = ref(false)
const captchaId = ref('')
const captchaImage = ref('')

const leftBg = computed(() => bgStyle(getThemeByKey(storeInfo.login_bg)))

onMounted(refresh)

async function loadCaptcha() {
  try {
    const r = await $fetch<{ id: string; image: string }>(`${cfg.apiBase}/api/captcha`, {
      headers: { 'X-Tenant-Slug': cfg.tenantSlug },
    })
    captchaId.value = r.id
    captchaImage.value = r.image
    form.captchaAnswer = ''
  } catch (e) {
    errorMsg.value = '验证码加载失败，请稍后再试'
  }
}

async function onSubmit() {
  errorMsg.value = ''
  loading.value = true
  try {
    const captcha = captchaNeeded.value && captchaId.value
      ? { id: captchaId.value, answer: form.captchaAnswer }
      : undefined
    await auth.login(form.email, form.password, form.remember, captcha)
    // 回跳被守卫拦截前的目标页；只接受站内路径（// 开头是协议相对外链，防开放重定向）
    const redirect = route.query.redirect
    const target =
      typeof redirect === 'string' && redirect.startsWith('/') && !redirect.startsWith('//')
        ? redirect
        : '/'
    await router.push(target)
  } catch (e: any) {
    failCount.value++
    errorMsg.value = e?.data?.message || e?.message || '登录失败，请检查账号密码'
    // 失败后策略：触发或刷新验证码（captcha 本身一次性，必须换新的）
    if (captchaNeeded.value || failCount.value >= CAPTCHA_TRIGGER_AFTER) {
      captchaNeeded.value = true
      await loadCaptcha()
    }
  } finally {
    loading.value = false
  }
}
</script>
