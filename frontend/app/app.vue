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
  titleTemplate: (t) => (t ? `${t} · 通用会员管理系统` : '通用会员管理系统'),
  meta: [
    { name: 'viewport', content: 'width=device-width, initial-scale=1' },
    { name: 'format-detection', content: 'telephone=no' },
  ],
})
</script>
