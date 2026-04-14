<template>
  <div class="p-6 lg:p-8 max-w-7xl mx-auto">
    <header class="mb-6">
      <h1 class="text-3xl font-semibold tracking-tight">设置</h1>
      <p class="mt-1 text-sm text-stone-500">门店基础数据与系统偏好</p>
    </header>

    <!-- Top tabs -->
    <nav class="flex items-center gap-1 flex-wrap border-b border-stone-200/60 dark:border-stone-800 mb-6">
      <NuxtLink
        v-for="tab in tabs"
        :key="tab.to"
        :to="tab.to"
        class="inline-flex items-center gap-2 px-3 py-2.5 text-sm whitespace-nowrap
               text-stone-600 dark:text-stone-400
               hover:text-stone-900 dark:hover:text-stone-100
               border-b-2 border-transparent -mb-px
               transition-colors duration-150
               focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 focus-visible:rounded"
        active-class="!text-primary-700 dark:!text-primary-300 !border-primary-500 font-medium"
      >
        <UIcon :name="tab.icon" class="size-4 shrink-0" />
        {{ tab.label }}
      </NuxtLink>
    </nav>

    <!-- Content -->
    <main>
      <NuxtPage />
    </main>
  </div>
</template>

<script setup lang="ts">
const tabs = [
  { to: '/settings/services', label: '服务项目',   icon: 'i-lucide-scissors' },
  { to: '/settings/cards',    label: '会员卡类型', icon: 'i-lucide-credit-card' },
  { to: '/settings/staff',    label: '员工',       icon: 'i-lucide-user-round' },
  { to: '/settings/account',  label: '账户',       icon: 'i-lucide-key-round' },
  { to: '/settings/general',  label: '通用',       icon: 'i-lucide-sliders-horizontal' },
  { to: '/settings/void',     label: '交易撤销',   icon: 'i-lucide-undo-2' },
  { to: '/settings/booking',  label: '预约配置',   icon: 'i-lucide-calendar-check' },
]
</script>
