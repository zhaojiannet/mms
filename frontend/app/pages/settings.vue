<template>
  <div class="p-6 lg:p-8 max-w-7xl mx-auto">
    <header class="mb-6">
      <h1 class="text-3xl font-semibold tracking-tight">设置</h1>
      <p class="mt-1 text-sm text-stone-500">门店基础数据与系统偏好</p>
    </header>

    <!-- Top tabs -->
    <nav class="flex items-center gap-1 flex-wrap border-b border-stone-200/60 dark:border-stone-800 mb-6">
      <NuxtLink
        v-for="tab in visibleTabs"
        :key="tab.to"
        :to="tab.to"
        class="inline-flex items-center gap-2 px-3 py-2.5 text-sm whitespace-nowrap
               text-stone-600 dark:text-stone-400
               hover:text-stone-900 dark:hover:text-stone-100
               border-b-2 border-transparent -mb-px
               transition-colors duration-150
               focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-primary-500/40 focus-visible:rounded"
        active-class="text-primary-700! dark:text-primary-300! border-primary-500! font-medium"
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
useHead({ title: '设置' })
import { useAuthStore } from '~/stores/auth'

// 整个设置页：admin 及以上才能进入（staff 直接跳首页）
definePageMeta({ middleware: 'at-least-admin' })

const auth = useAuthStore()

// requires：不填=所有登录用户可见；'admin'=admin 及以上；'super'=仅超级管理员
type Requires = 'admin' | 'super' | undefined
interface Tab { to: string; label: string; icon: string; requires?: Requires }

const tabs: Tab[] = [
  { to: '/settings/store',           label: '店铺',       icon: 'i-lucide-store',          requires: 'admin' },
  { to: '/settings/services',        label: '服务项目',   icon: 'i-lucide-scissors',       requires: 'admin' },
  { to: '/settings/cards',           label: '会员卡类型', icon: 'i-lucide-credit-card',    requires: 'admin' },
  { to: '/settings/staff',           label: '员工',       icon: 'i-lucide-user-round',     requires: 'admin' },
  { to: '/settings/payment-methods', label: '支付方式',   icon: 'i-lucide-wallet',         requires: 'admin' },
  { to: '/settings/users',           label: '账号',       icon: 'i-lucide-shield',         requires: 'super' },
  { to: '/settings/void',            label: '交易撤销',   icon: 'i-lucide-undo-2',         requires: 'super' },
  { to: '/settings/booking',         label: '预约配置',   icon: 'i-lucide-calendar-check', requires: 'admin' },
  { to: '/settings/audit',           label: '操作日志',   icon: 'i-lucide-history',        requires: 'admin' },
]

const visibleTabs = computed(() => {
  const role = auth.user?.role
  const isSuper = role === 'super_admin'
  const isAdminUp = isSuper || role === 'admin'
  return tabs.filter(t => {
    if (t.requires === 'super') return isSuper
    if (t.requires === 'admin') return isAdminUp
    return true
  })
})
</script>
