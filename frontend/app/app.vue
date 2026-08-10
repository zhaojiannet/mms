<template>
  <UApp :locale="zh_cn">
    <NuxtLoadingIndicator color="var(--ui-primary)" :height="2" />
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </UApp>
</template>

<script setup lang="ts">
import { zh_cn } from '@nuxt/ui/locale'

useTheme().init()
useFontSize().init()

// 全局默认 title（各页面 useHead 覆盖）
useHead({
  // t 等于产品名本身时不拼接：那是 nuxt.config 里 SPA 壳的兜底 title 传进来的，
  // 拼了会显示「通用会员管理系统 · 通用会员管理系统」
  titleTemplate: (t) => (t && t !== '通用会员管理系统' ? `${t} · 通用会员管理系统` : '通用会员管理系统'),
  meta: [
    { name: 'viewport', content: 'width=device-width, initial-scale=1' },
    { name: 'format-detection', content: 'telephone=no' },
  ],
})
</script>
