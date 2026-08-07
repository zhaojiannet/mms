<template>
  <div class="min-h-screen flex items-center justify-center bg-stone-50 dark:bg-stone-950 p-4">
    <div class="w-full max-w-md rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs p-6 space-y-5">
      <template v-if="!submitted">
        <div class="text-center space-y-2">
          <h1 class="text-2xl font-semibold tracking-tight">申请开通</h1>
          <p class="text-sm text-stone-500">填写店铺信息，我们审核后会尽快联系你</p>
        </div>

        <UForm :state="form" class="space-y-4" @submit="onSubmit">
          <UFormField label="店铺名" required>
            <UInput v-model="form.store_name" size="md" class="w-full" placeholder="填写你的店铺名称" />
          </UFormField>
          <UFormField label="行业" required>
            <USelect
              v-model="form.industry"
              :items="['美容美发', '美甲美睫', '培训教育', '宠物服务', '养生保健', '其他']"
              size="md"
              class="w-full"
              placeholder="选择行业"
            />
          </UFormField>
          <div class="flex gap-3">
            <UFormField label="联系人" required class="flex-1">
              <UInput v-model="form.contact_name" size="md" class="w-full" />
            </UFormField>
            <UFormField label="手机号" required class="flex-1">
              <UInput v-model="form.phone" type="tel" size="md" class="w-full" />
            </UFormField>
          </div>
          <UFormField label="期望网址" required help="小写字母开头，仅限字母数字连字符，3-30 位">
            <UInput v-model="form.desired_slug" size="md" class="w-full font-mono" placeholder="mystore">
              <template #trailing>
                <span class="text-xs text-stone-400">.{{ cfg.appDomain }}</span>
              </template>
            </UInput>
          </UFormField>
          <UFormField label="图形验证码" required>
            <div class="flex items-center gap-2">
              <UInput v-model="form.captcha_answer" size="md" class="flex-1" />
              <img
                v-if="captcha.image"
                :src="captcha.image"
                alt="验证码"
                class="h-9 rounded cursor-pointer ring-1 ring-stone-200 dark:ring-stone-700"
                title="点击刷新"
                @click="loadCaptcha"
              />
            </div>
          </UFormField>

          <UAlert v-if="error" :description="error" color="error" variant="soft" icon="i-lucide-alert-circle" />
          <UButton type="submit" block :loading="submitting" class="active:scale-95 transition-transform">提交申请</UButton>
        </UForm>
      </template>

      <template v-else>
        <div class="text-center space-y-3 py-6">
          <UIcon name="i-lucide-check-circle" class="text-4xl text-primary-500" />
          <h1 class="text-2xl font-semibold tracking-tight">申请已提交</h1>
          <p class="text-sm text-stone-500">我们会在 1-2 个工作日内联系你完成开通</p>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: false })
useHead({ title: '申请开通' })

const cfg = useRuntimeConfig().public

const form = reactive({
  store_name: '',
  industry: '',
  contact_name: '',
  phone: '',
  desired_slug: '',
  captcha_answer: '',
})
const captcha = reactive({ id: '', image: '' })
const error = ref('')
const submitting = ref(false)
const submitted = ref(false)

async function loadCaptcha() {
  try {
    const r = await $fetch<{ id: string; image: string }>(`${cfg.apiBase}/api/public/captcha`)
    captcha.id = r.id
    captcha.image = r.image
    form.captcha_answer = ''
  } catch {
    // 加载失败时提交会因验证码校验被后端拒绝，用户可点击图片重试
  }
}

async function onSubmit() {
  error.value = ''
  submitting.value = true
  try {
    await $fetch(`${cfg.apiBase}/api/signup/applications`, {
      method: 'POST',
      body: {
        ...form,
        desired_slug: form.desired_slug.toLowerCase().trim(),
        captcha_id: captcha.id,
      },
    })
    submitted.value = true
  } catch (e: any) {
    error.value = e?.data?.message || '提交失败，请稍后再试'
    await loadCaptcha()
  } finally {
    submitting.value = false
  }
}

onMounted(loadCaptcha)
</script>
