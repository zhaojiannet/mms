<script setup lang="ts">
// 访问 /settings 根路径自动跳转到第一个有实际内容的 tab（账户）
definePageMeta({ middleware: () => navigateTo('/settings/store', { replace: true }) })
</script>

<template><div /></template>
