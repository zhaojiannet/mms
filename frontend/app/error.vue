<template>
  <div class="min-h-screen flex items-center justify-center bg-stone-50 dark:bg-stone-950 p-6">
    <div class="max-w-md w-full text-center space-y-6">
      <div class="flex justify-center">
        <div
          class="w-16 h-16 rounded-2xl flex items-center justify-center bg-stone-100 dark:bg-stone-900 ring-1 ring-stone-200/60 dark:ring-stone-800"
        >
          <UIcon
            :name="status === 404 ? 'i-lucide-compass' : 'i-lucide-alert-triangle'"
            class="size-8 text-stone-500"
          />
        </div>
      </div>

      <div class="space-y-2">
        <div class="text-5xl font-semibold tracking-tight text-stone-900 dark:text-stone-100 tabular-nums">
          {{ status }}
        </div>
        <h1 class="text-base font-medium text-stone-700 dark:text-stone-300">
          {{ headline }}
        </h1>
        <p v-if="detail" class="text-sm text-stone-500 dark:text-stone-400">{{ detail }}</p>
      </div>

      <div class="flex items-center justify-center gap-2">
        <UButton variant="soft" color="neutral" @click="handleBack">返回</UButton>
        <UButton @click="handleRetry">
          {{ status === 404 ? '回到首页' : '重试' }}
        </UButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { NuxtError } from '#app'

const props = defineProps<{ error: NuxtError }>()

// Nuxt 上抛 error 不一定带 statusCode（如 throw 普通 Error）；500 兜底
const status = computed(() => props.error?.statusCode ?? 500)

useHead({ title: `${status.value} · 出错了` })

const headline = computed(() => {
  if (status.value === 404) return '页面不存在'
  if (status.value === 403) return '没有访问权限'
  if (status.value >= 500) return '服务暂时不可用'
  return '发生了一些问题'
})
const detail = computed(() => {
  if (status.value === 404) return '地址可能输错了，或该内容已被删除'
  if (status.value >= 500) return '已自动告知管理员，请稍后再试'
  return props.error.message || ''
})

function handleBack() {
  if (typeof window !== 'undefined' && window.history.length > 1) {
    window.history.back()
  } else {
    clearError({ redirect: '/' })
  }
}
function handleRetry() {
  if (status.value === 404) {
    clearError({ redirect: '/' })
  } else {
    clearError({ redirect: useRoute().fullPath })
  }
}
</script>
